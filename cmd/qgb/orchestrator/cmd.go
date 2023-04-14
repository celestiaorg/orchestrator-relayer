package orchestrator

import (
	"context"
	"os"
	"time"

	cmdcommon "github.com/celestiaorg/orchestrator-relayer/cmd/qgb/common"

	evm2 "github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys/evm"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys"
	"github.com/celestiaorg/orchestrator-relayer/store"

	"github.com/celestiaorg/orchestrator-relayer/helpers"
	"github.com/celestiaorg/orchestrator-relayer/orchestrator"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	dssync "github.com/ipfs/go-datastore/sync"
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
				NeedP2PKeyStore: true,
			})
			if !isInit {
				logger.Info("please initialize the orchestrator using `qgb orchestrator init` command")
				return store.ErrNotInited
			}

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			stopFuncs := make([]func() error, 0)
			defer func() {
				for _, f := range stopFuncs {
					err := f()
					if err != nil {
						logger.Error(err.Error())
					}
				}
			}()
			tmQuerier, appQuerier, stops, err := cmdcommon.NewTmAndAppQuerier(logger, config.tendermintRPC, config.celesGRPC)
			if err != nil {
				return err
			}
			stopFuncs = append(stopFuncs, stops...)

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
			stopFuncs = append(stopFuncs, func() error { return s.Close(logger, openOptions) })
			dataStore := dssync.MutexWrap(s.DataStore)

			dht, err := cmdcommon.CreateDHTAndWaitForPeers(ctx, logger, s.P2PKeyStore, config.p2pNickname, config.p2pListenAddr, config.bootstrappers, dataStore)
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

			logger.Info("loading EVM account", "address", config.evmAccAddress)

			acc, err := evm2.GetAccountFromStoreAndUnlockIt(s.EVMKeyStore, config.evmAccAddress, config.EVMPassphrase)
			if err != nil {
				return err
			}

			stopFuncs = append(stopFuncs, func() error { return s.EVMKeyStore.Lock(acc.Address) })

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
				return err
			}

			logger.Debug("starting orchestrator")

			// Listen for and trap any OS signal to graceful shutdown and exit
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
