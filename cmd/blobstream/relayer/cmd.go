package relayer

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/telemetry"
	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/version"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/base"

	ethcmn "github.com/ethereum/go-ethereum/common"

	blobstreamwrapper "github.com/celestiaorg/blobstream-contracts/v4/wrappers/Blobstream.sol"

	evm2 "github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/keys/evm"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	dssync "github.com/ipfs/go-datastore/sync"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/common"
	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/keys"
	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/helpers"
	"github.com/celestiaorg/orchestrator-relayer/store"

	"github.com/celestiaorg/orchestrator-relayer/relayer"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	relCmd := &cobra.Command{
		Use:          "relayer",
		Aliases:      []string{"rel"},
		Short:        "Blobstream relayer that relays signatures to the target EVM chain",
		SilenceUsage: true,
	}

	relCmd.AddCommand(
		Start(),
		Init(),
		keys.Command(ServiceNameRelayer),
	)

	relCmd.SetHelpCommand(&cobra.Command{})

	return relCmd
}

// Init initializes the orchestrator store and creates necessary files.
func Init() *cobra.Command {
	cmd := cobra.Command{
		Use:   "init",
		Short: "Initialize the Blobstream relayer store. Passed flags have persisted effect.",
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
				NeedDataStore:      true,
				NeedEVMKeyStore:    true,
				NeedP2PKeyStore:    true,
				NeedSignatureStore: true,
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

func Start() *cobra.Command {
	command := &cobra.Command{
		Use:   "start <flags>",
		Short: "Runs the Blobstream relayer to submit attestations to the target EVM chain",
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := base.GetHomeDirectory(cmd, ServiceNameRelayer)
			if err != nil {
				return err
			}

			fileConfig, err := LoadFileConfiguration(homeDir)
			if err != nil {
				return err
			}
			config, err := parseRelayerStartFlags(cmd, fileConfig)
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
			logger.Info("initializing relayer", "home", homeDir, "version", buildInfo.SemanticVersion, "build_date", buildInfo.BuildTime)

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

			tmQuerier, appQuerier, storeStops, err := common.NewTmAndAppQuerier(logger, config.CoreRPC, config.CoreGRPC, config.GrpcInsecure)
			stopFuncs = append(stopFuncs, storeStops...)
			if err != nil {
				return err
			}

			s, storeStops, err := common.OpenStore(logger, config.Home, store.OpenOptions{
				HasDataStore:      true,
				BadgerOptions:     store.DefaultBadgerOptions(config.Home),
				HasSignatureStore: true,
				HasEVMKeyStore:    true,
				HasP2PKeyStore:    true,
			})
			if err != nil {
				return err
			}

			logger.Info("loading EVM account", "address", config.evmAccAddress)

			acc, err := evm2.GetAccountFromStoreAndUnlockIt(s.EVMKeyStore, config.evmAccAddress, config.EVMPassphrase)
			stopFuncs = append(stopFuncs, func() error { return s.EVMKeyStore.Lock(acc.Address) })
			if err != nil {
				return err
			}

			// creating the data store
			dataStore := dssync.MutexWrap(s.DataStore)

			relayerMeters, err := telemetry.InitRelayerMeters()
			if err != nil {
				return err
			}

			var registerer prometheus.Registerer
			if config.MetricsConfig.Metrics {
				opts := []otlpmetrichttp.Option{
					otlpmetrichttp.WithEndpoint(config.MetricsConfig.Endpoint),
					otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression),
				}
				if !config.MetricsConfig.TLS {
					opts = append(opts, otlpmetrichttp.WithInsecure())
				}
				var shutdown func() error
				registerer, shutdown, err = telemetry.Start(
					ctx,
					logger,
					fmt.Sprintf("%s:%s", ServiceNameRelayer, config.ContractAddr),
					acc.Address.Hex(),
					opts,
				)
				if shutdown != nil {
					stopFuncs = append(stopFuncs, shutdown)
				}
				if err != nil {
					return err
				}
				shutdown, err := telemetry.PrometheusMetrics(ctx, logger, registerer, config.MetricsConfig.P2PEndpoint)
				if shutdown != nil {
					stopFuncs = append(stopFuncs, shutdown)
				}
				if err != nil {
					return err
				}
			}

			dht, err := common.CreateDHTAndWaitForPeers(ctx, logger, s.P2PKeyStore, config.p2pNickname, config.P2PListenAddr, config.Bootstrappers, dataStore, registerer)
			if err != nil {
				return err
			}
			stopFuncs = append(stopFuncs, func() error { return dht.Close() })
			stopFuncs = append(stopFuncs, storeStops...)

			// creating the p2p querier
			p2pQuerier := p2p.NewQuerier(dht, logger)
			retrier := helpers.NewRetrier(logger, 6, time.Minute)

			// connecting to a Blobstream contract
			ethClient, err := ethclient.Dial(config.EvmRPC)
			if err != nil {
				return err
			}
			defer ethClient.Close()
			blobstreamWrapper, err := blobstreamwrapper.NewWrappers(ethcmn.HexToAddress(config.ContractAddr), ethClient)
			if err != nil {
				return err
			}

			evmClient := evm.NewClient(
				logger,
				blobstreamWrapper,
				s.EVMKeyStore,
				&acc,
				config.EvmRPC,
				config.EvmGasLimit,
			)

			relay := relayer.NewRelayer(
				tmQuerier,
				appQuerier,
				p2pQuerier,
				evmClient,
				logger,
				retrier,
				s.SignatureStore,
				time.Duration(config.EVMRetryTimeout)*time.Minute,
				config.isBackupRelayer,
				time.Duration(config.backupRelayerWaitTime)*time.Minute,
				relayerMeters,
			)

			// Listen for and trap any OS signal to graceful shutdown and exit
			go helpers.TrapSignal(logger, cancel)

			logger.Info("starting relayer")
			err = relay.Start(ctx)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return addRelayerStartFlags(command)
}
