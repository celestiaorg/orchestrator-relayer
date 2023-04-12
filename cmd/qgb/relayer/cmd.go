package relayer

import (
	"context"
	"os"
	"strings"
	"time"

	evm2 "github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys/evm"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys"
	common2 "github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys/p2p"
	"github.com/celestiaorg/orchestrator-relayer/store"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	"github.com/ethereum/go-ethereum/common"

	"github.com/celestiaorg/orchestrator-relayer/helpers"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/relayer"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	wrapper "github.com/celestiaorg/quantum-gravity-bridge/wrappers/QuantumGravityBridge.sol"
	"github.com/ethereum/go-ethereum/ethclient"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func Command() *cobra.Command {
	orchCmd := &cobra.Command{
		Use:          "relayer",
		Aliases:      []string{"rel"},
		Short:        "QGB relayer that relays signatures to the target EVM chain",
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

// Init initializes the orchestrator store and creates necessary files.
func Init() *cobra.Command {
	cmd := cobra.Command{
		Use:   "init",
		Short: "Initialize the QGB relayer store. Passed flags have persisted effect.",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseInitFlags(cmd)
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)

			initOptions := store.InitOptions{
				NeedDataStore:   false,
				NeedEVMKeyStore: true,
				NeedP2PKeyStore: true,
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

func Start() *cobra.Command {
	command := &cobra.Command{
		Use:   "start <flags>",
		Short: "Runs the QGB relayer to submit attestations to the target EVM chain",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseRelayerStartFlags(cmd)
			if err != nil {
				return err
			}

			// creating the logger
			logger := tmlog.NewTMLogger(os.Stdout)
			logger.Debug("initializing relayer")

			// connecting to a QGB contract
			ethClient, err := ethclient.Dial(config.evmRPC)
			if err != nil {
				return err
			}
			qgbWrapper, err := wrapper.NewQuantumGravityBridge(config.contractAddr, ethClient)
			if err != nil {
				return err
			}

			// creating Celestia-app configuration
			encCfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)

			// creating tendermint querier
			tmQuerier := rpc.NewTmQuerier(config.tendermintRPC, logger)
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

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

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

			// checking if the provided home is already initiated
			isInit := store.IsInit(logger, config.Home, store.InitOptions{NeedEVMKeyStore: true, NeedP2PKeyStore: true})
			if !isInit {
				// TODO we don't need to manually initialize the p2p keystore
				logger.Info("please initialize the relayer using `qgb relayer init` command")
				return store.ErrNotInited
			}

			// creating the data store
			openOptions := store.OpenOptions{HasEVMKeyStore: true, HasP2PKeyStore: true}
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

			logger.Info("loading EVM account", "address", config.evmAccAddress)

			acc, err := evm2.GetAccountFromStore(s.EVMKeyStore, config.evmAccAddress)
			if err != nil {
				return err
			}

			passphrase := config.EVMPassphrase
			// if the passphrase is not specified as a flag, ask for it.
			if passphrase == "" {
				passphrase, err = evm2.GetPassphrase()
				if err != nil {
					return err
				}
			}

			err = s.EVMKeyStore.Unlock(acc, passphrase)
			if err != nil {
				logger.Error("unable to unlock the EVM private key")
				return err
			}
			defer func(EVMKeyStore *keystore.KeyStore, addr common.Address) {
				err := EVMKeyStore.Lock(addr)
				if err != nil {
					logger.Error(err.Error())
				}
			}(s.EVMKeyStore, acc.Address)

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
			logger.Info("created host", "ID", h.ID().String(), "Addresses", h.Addrs())
			// creating the data store
			dataStore := dssync.MutexWrap(ds.NewMapDatastore())

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
			dht, err := p2p.NewQgbDHT(ctx, h, dataStore, bootstrappers, logger)
			if err != nil {
				return err
			}

			// wait for the dht to have some peers
			err = dht.WaitForPeers(ctx, 2*time.Minute, 10*time.Second, 1)
			if err != nil {
				return err
			}

			// creating the p2p querier
			p2pQuerier := p2p.NewQuerier(dht, logger)
			retrier := helpers.NewRetrier(logger, 5, 15*time.Second)

			relay := relayer.NewRelayer(
				tmQuerier,
				appQuerier,
				p2pQuerier,
				evm.NewClient(
					logger,
					qgbWrapper,
					s.EVMKeyStore,
					&acc,
					config.evmRPC,
					config.evmGasLimit,
				),
				logger,
				retrier,
			)

			// Listen for and trap any OS signal to gracefully shutdown and exit
			go helpers.TrapSignal(logger, cancel)

			err = relay.Start(ctx)
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			return nil
		},
	}
	return addRelayerStartFlags(command)
}
