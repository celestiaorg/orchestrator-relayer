package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/base"
	"github.com/spf13/viper"
	"os"
	"path/filepath"
	"text/template"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/common"
	evm2 "github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/keys/evm"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	dssync "github.com/ipfs/go-datastore/sync"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/keys"
	"github.com/celestiaorg/orchestrator-relayer/store"

	"github.com/celestiaorg/orchestrator-relayer/helpers"
	"github.com/celestiaorg/orchestrator-relayer/orchestrator"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
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
			logger := tmlog.NewTMLogger(os.Stdout)

			homeDir, err := base.GetHomeDirectory(cmd, ServiceNameOrchestrator)
			if err != nil {
				return err
			}
			logger.Debug("initializing orchestrator", "home", homeDir)

			fileConfig, err := LoadFileConfiguration(homeDir)
			if err != nil {
				return err
			}
			config, err := parseOrchestratorFlags(cmd, fileConfig)
			if err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			stopFuncs := make([]func() error, 0)

			tmQuerier, appQuerier, stops, err := common.NewTmAndAppQuerier(logger, config.CoreRPC, config.CoreGRPC, config.GRPCInsecure)
			stopFuncs = append(stopFuncs, stops...)
			if err != nil {
				return err
			}

			s, stops, err := common.OpenStore(logger, config.Home, store.OpenOptions{
				HasDataStore:      true,
				BadgerOptions:     store.DefaultBadgerOptions(config.Home),
				HasSignatureStore: false,
				HasEVMKeyStore:    true,
				HasP2PKeyStore:    true,
			})
			stopFuncs = append(stopFuncs, stops...)
			if err != nil {
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

			// creating the p2p querier
			p2pQuerier := p2p.NewQuerier(dht, logger)
			retrier := helpers.NewRetrier(logger, 6, time.Minute)

			defer func() {
				for _, f := range stopFuncs {
					err := f()
					if err != nil {
						logger.Error(err.Error())
					}
				}
			}()

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
		Short: "Initialize the Blobstream orchestrator store. Passed flags have persisted effect.",
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

func LoadFileConfiguration(homeDir string) (*StartConfig, error) {
	v := viper.New()
	v.SetEnvPrefix("")
	v.AutomaticEnv()
	configPath := filepath.Join(homeDir, "config")
	configFilePath := filepath.Join(configPath, "config.toml")
	conf := DefaultStartConfig()

	// if config.toml file does not exist, we create it and write default ClientConfig values into it.
	if _, err := os.Stat(configFilePath); os.IsNotExist(err) {
		if err := initializeConfigFile(configFilePath, configPath, conf); err != nil {
			return nil, err
		}
	}

	conf, err := getStartConfig(v, configPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't get client config: %v", err)
	}
	return conf, nil
}

func initializeConfigFile(configFilePath string, configPath string, conf *StartConfig) error {
	if err := ensureConfigPath(configPath); err != nil {
		return fmt.Errorf("couldn't make orchestrator config: %v", err)
	}

	if err := writeConfigToFile(configFilePath, conf); err != nil {
		return fmt.Errorf("could not write client config to the file: %v", err)
	}
	return nil
}

// ensureConfigPath creates a directory configPath if it does not exist
func ensureConfigPath(configPath string) error {
	return os.MkdirAll(configPath, os.ModePerm)
}

// writeConfigToFile parses DefaultConfigTemplate, renders config using the template and writes it to
// configFilePath.
func writeConfigToFile(configFilePath string, config *StartConfig) error {
	var buffer bytes.Buffer

	tmpl := template.New("orchestratorConfigFileTemplate")
	configTemplate, err := tmpl.Parse(DefaultConfigTemplate)
	if err != nil {
		return err
	}

	if err := configTemplate.Execute(&buffer, config); err != nil {
		return err
	}

	return os.WriteFile(configFilePath, buffer.Bytes(), 0o600)
}

// getStartConfig reads values from config.toml file and unmarshalls them into StartConfig
func getStartConfig(v *viper.Viper, configPath string) (*StartConfig, error) {
	v.AddConfigPath(configPath)
	v.SetConfigName("config")
	v.SetConfigType("toml")

	if err := v.ReadInConfig(); err != nil {
		return nil, err
	}

	conf := new(StartConfig)
	if err := v.Unmarshal(conf); err != nil {
		return nil, err
	}

	return conf, nil
}
