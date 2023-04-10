package orchestrator

import (
	"context"
	"os"
	"strings"
	"time"

	common2 "github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys/p2p"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"
	"golang.org/x/term"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys"

	"github.com/celestiaorg/orchestrator-relayer/store"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/orchestrator-relayer/helpers"
	"github.com/celestiaorg/orchestrator-relayer/orchestrator"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func Command() *cobra.Command {
	orchCmd := &cobra.Command{
		Use:          "orchestrator",
		Aliases:      []string{"orch"},
		Short:        "QGB orchestrator that signs attestations",
		SilenceUsage: true,
	}

	orchCmd.AddCommand(
		Start(),
		Init(),
		keys.Command(),
	)

	orchCmd.SetHelpCommand(&cobra.Command{})

	return orchCmd
}

// Start starts the orchestrator to listen on new attestations, sign them and broadcast them.
func Start() *cobra.Command {
	command := &cobra.Command{
		Use:   "start <flags>",
		Short: "Starts the QGB orchestrator to sign attestations",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseOrchestratorFlags(cmd)
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)
			logger.Debug("initializing orchestrator")

			// checking if the provided home is already initiated
			isInit := store.IsInit(logger, config.Home, store.InitOptions{
				NeedDataStore:   true,
				NeedEVMKeyStore: true,
				NeedP2PKeyStore: false,
			})
			if !isInit {
				return store.ErrNotInited
			}

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			// load app encoding configuration
			encCfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)

			// creating tendermint querier
			tmQuerier := rpc.NewTmQuerier(config.tendermintRPC, logger)
			if err != nil {
				return err
			}
			err = tmQuerier.Start()
			if err != nil {
				return err
			}
			defer func() {
				err := tmQuerier.Stop()
				if err != nil {
					logger.Error(err.Error())
				}
			}()

			// creating the application querier
			appQuerier := rpc.NewAppQuerier(logger, config.celesGRPC, encCfg)
			err = appQuerier.Start()
			if err != nil {
				return err
			}
			defer func() {
				err := appQuerier.Stop()
				if err != nil {
					logger.Error(err.Error())
				}
			}()

			// creating the data store
			openOptions := store.OpenOptions{
				HasDataStore:   true,
				BadgerOptions:  store.DefaultBadgerOptions(config.Home),
				HasEVMKeyStore: true,
				HasP2PKeyStore: true,
			}
			s, err := store.OpenStore(logger, config.Home, openOptions)
			if err != nil {
				return err
			}
			defer func(s *store.Store, log tmlog.Logger) {
				err := s.Close(log, openOptions)
				if err != nil {
					logger.Error(err.Error())
				}
			}(s, logger)
			dataStore := dssync.MutexWrap(s.DataStore)

			// get the p2p private key or generate a new one
			privKey, err := common2.GetP2PKeyOrGenerateNewOne(s.P2PKeyStore, config.p2pNickname)
			if err != nil {
				return err
			}

			// creating the host
			h, err := p2p.CreateHost(config.p2pListenAddr, privKey)
			if err != nil {
				return err
			}
			logger.Info(
				"created host",
				"ID",
				h.ID().String(),
				"Addresses",
				h.Addrs(),
			)

			// get the bootstrappers
			var bootstrappers []peer.AddrInfo
			if config.bootstrappers == "" {
				bootstrappers = nil
			} else {
				bs := strings.Split(config.bootstrappers, ",")
				bootstrappers, err = helpers.ParseAddrInfos(logger, bs)
				if err != nil {
					return err
				}
			}

			// creating the dht
			dht, err := p2p.NewQgbDHT(cmd.Context(), h, dataStore, bootstrappers, logger)
			if err != nil {
				return err
			}

			// wait for the dht to have some peers
			err = dht.WaitForPeers(cmd.Context(), 5*time.Minute, 10*time.Second, 1)
			if err != nil {
				return err
			}

			// creating the p2p querier
			p2pQuerier := p2p.NewQuerier(dht, logger)

			// creating the broadcasted
			broadcaster := orchestrator.NewBroadcaster(dht)
			if err != nil {
				return err
			}

			// creating the retrier
			retrier := helpers.NewRetrier(logger, 5, 15*time.Second)

			if !common.IsHexAddress(config.evmAccAddress) {
				logger.Error("provided address is not a correct EVM address", "address", config.evmAccAddress)
				return nil // should we return errors in these cases?
			}

			addr := common.HexToAddress(config.evmAccAddress)
			if !s.EVMKeyStore.HasAddress(addr) {
				logger.Info("account not found in keystore", "address", config.evmAccAddress)
				return nil
			}

			logger.Info("loading EVM account", "address", addr.String())

			var acc accounts.Account
			for _, storeAcc := range s.EVMKeyStore.Accounts() {
				if storeAcc.Address.String() == addr.String() {
					acc = storeAcc
				}
			}

			passphrase := config.Passphrase
			// if the passphrase is not specified as a flag, ask for it.
			if passphrase == "" {
				logger.Info("please provide the account passphrase")
				bzPassphrase, err := term.ReadPassword(int(os.Stdin.Fd()))
				if err != nil {
					return err
				}
				passphrase = string(bzPassphrase)
			}

			err = s.EVMKeyStore.Unlock(acc, passphrase)
			if err != nil {
				logger.Error("unable to load the EVM private key")
				return err
			}
			defer func(EVMKeyStore *keystore.KeyStore, addr common.Address) {
				err := EVMKeyStore.Lock(addr)
				if err != nil {
					panic(err)
				}
			}(s.EVMKeyStore, acc.Address)

			// creating the orchestrator
			orch := orchestrator.New(
				logger,
				appQuerier,
				tmQuerier,
				p2pQuerier,
				broadcaster,
				retrier,
				s.EVMKeyStore,
				&acc,
			)
			if err != nil {
				panic(err)
			}

			logger.Debug("starting orchestrator")

			// Listen for and trap any OS signal to gracefully shutdown and exit
			go helpers.TrapSignal(logger, cancel)

			// starting the orchestrator
			orch.Start(ctx)

			return nil
		},
	}
	return addOrchestratorFlags(command)
}

// Init initializes the orchestrator store and creates necessary files.
func Init() *cobra.Command {
	cmd := cobra.Command{
		Use:   "init",
		Short: "Initialize the QGB orchestrator store. Passed flags have persisted effect.",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseInitFlags(cmd)
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)

			initOptions := store.InitOptions{
				NeedDataStore:   true,
				NeedEVMKeyStore: true,
				NeedP2PKeyStore: false,
			}
			isInit := store.IsInit(logger, config.home, initOptions)
			if isInit {
				logger.Info("provided path is already initiated", "path", config.home)
				return nil
			}

			err = store.Init(logger, config.home, initOptions)
			if err != nil {
				return err
			}

			return nil
		},
	}
	return addInitFlags(&cmd)
}
