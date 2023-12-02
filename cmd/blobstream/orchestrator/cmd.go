package orchestrator

import (
	"context"
	"path/filepath"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/version"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/base"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/common"
	evm2 "github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/keys/evm"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	dssync "github.com/ipfs/go-datastore/sync"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/keys"
	"github.com/celestiaorg/orchestrator-relayer/store"

	"github.com/celestiaorg/orchestrator-relayer/helpers"
	"github.com/celestiaorg/orchestrator-relayer/orchestrator"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	orchCmd := &cobra.Command{
		Use:          "orchestrator",
		Aliases:      []string{"orch"},
		Short:        "Blobstream orchestrator that signs attestations",
		SilenceUsage: true,
	}

	orchCmd.AddCommand(
		Start(),
		Init(),
		keys.Command(ServiceNameOrchestrator),
	)

	orchCmd.SetHelpCommand(&cobra.Command{})

	return orchCmd
}

// Start starts the orchestrator to listen on new attestations, sign them and broadcast them.
func Start() *cobra.Command {
	command := &cobra.Command{
		Use:   "start <flags>",
		Short: "Starts the Blobstream orchestrator to sign attestations",
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := base.GetHomeDirectory(cmd, ServiceNameOrchestrator)
			if err != nil {
				return err
			}
			fileConfig, err := LoadFileConfiguration(homeDir)
			if err != nil {
				return err
			}
			config, err := parseOrchestratorFlags(cmd, fileConfig)
			if err != nil {
				return err
			}
			if err := config.ValidateBasics(); err != nil {
				return err
			}

			logger, err := base.GetLogger(config.LogLevel, config.LogFormat)
			if err != nil {
				return err
			}

			buildInfo := version.GetBuildInfo()
			logger.Info("initializing orchestrator", "home", homeDir, "version", buildInfo.SemanticVersion, "build_date", buildInfo.BuildTime)

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

			tmQuerier, appQuerier, storeStops, err := common.NewTmAndAppQuerier(logger, config.CoreRPC, config.CoreGRPC, config.GRPCInsecure)
			stopFuncs = append(stopFuncs, storeStops...)
			if err != nil {
				return err
			}

			s, storeStops, err := common.OpenStore(logger, config.Home, store.OpenOptions{
				HasDataStore:      true,
				BadgerOptions:     store.DefaultBadgerOptions(config.Home),
				HasSignatureStore: false,
				HasEVMKeyStore:    true,
				HasP2PKeyStore:    true,
			})
			if err != nil {
				stopFuncs = append(stopFuncs, storeStops...)
				return err
			}

			logger.Info("loading EVM account", "address", config.EvmAccAddress)

			acc, err := evm2.GetAccountFromStoreAndUnlockIt(s.EVMKeyStore, config.EvmAccAddress, config.EVMPassphrase)
			stopFuncs = append(stopFuncs, func() error { return s.EVMKeyStore.Lock(acc.Address) })
			if err != nil {
				return err
			}

			// creating the data store
			dataStore := dssync.MutexWrap(s.DataStore)

			dht, err := common.CreateDHTAndWaitForPeers(ctx, logger, s.P2PKeyStore, config.P2pNickname, config.P2PListenAddr, config.Bootstrappers, dataStore)
			if err != nil {
				return err
			}
			stopFuncs = append(stopFuncs, func() error { return dht.Close() })
			stopFuncs = append(stopFuncs, storeStops...)

			// creating the p2p querier
			p2pQuerier := p2p.NewQuerier(dht, logger)
			retrier := helpers.NewRetrier(logger, 6, time.Minute)

			// creating the broadcaster
			broadcaster := orchestrator.NewBroadcaster(p2pQuerier.BlobstreamDHT)
			if err != nil {
				return err
			}

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

			logger.Info("starting orchestrator")

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
		Short: "Initialize the Blobstream orchestrator store. Passed flags have persisted effect.",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseInitFlags(cmd)
			if err != nil {
				return err
			}

			logger, err := base.GetLogger(config.logLevel, config.logFormat)
			if err != nil {
				return err
			}

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

			configPath := filepath.Join(config.home, "config")
			configFilePath := filepath.Join(configPath, "config.toml")
			conf := DefaultStartConfig()
			err = initializeConfigFile(configFilePath, configPath, conf)
			if err != nil {
				return err
			}

			return nil
		},
	}
	return addInitFlags(&cmd)
}
