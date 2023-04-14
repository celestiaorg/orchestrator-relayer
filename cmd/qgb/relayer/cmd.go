package relayer

import (
	"context"
	"os"
	"time"

	cmdcommon "github.com/celestiaorg/orchestrator-relayer/cmd/qgb/common"

	evm2 "github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys/evm"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys"
	"github.com/celestiaorg/orchestrator-relayer/helpers"
	"github.com/celestiaorg/orchestrator-relayer/store"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/relayer"
	wrapper "github.com/celestiaorg/quantum-gravity-bridge/wrappers/QuantumGravityBridge.sol"
	"github.com/ethereum/go-ethereum/ethclient"
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
			stopFuncs = append(stopFuncs, stops...)
			if err != nil {
				return err
			}

			// checking if the provided home is already initiated
			isInit := store.IsInit(logger, config.Home, store.InitOptions{NeedDataStore: true, NeedEVMKeyStore: true, NeedP2PKeyStore: true})
			if !isInit {
				// TODO we don't need to manually initialize the p2p keystore
				logger.Info("please initialize the relayer using `qgb relayer init` command")
				return store.ErrNotInited
			}

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

			logger.Info("loading EVM account", "address", config.evmAccAddress)

			acc, err := evm2.GetAccountFromStoreAndUnlockIt(s.EVMKeyStore, config.evmAccAddress, config.EVMPassphrase)
			if err != nil {
				return err
			}
			stopFuncs = append(stopFuncs, func() error { return s.EVMKeyStore.Lock(acc.Address) })

			// creating the data store
			dataStore := dssync.MutexWrap(s.DataStore)

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			dht, err := cmdcommon.CreateDHTAndWaitForPeers(ctx, logger, s.P2PKeyStore, config.p2pNickname, config.p2pListenAddr, config.bootstrappers, dataStore)
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

			// Listen for and trap any OS signal to graceful shutdown and exit
			go helpers.TrapSignal(logger, cancel)

			err = relay.Start(ctx)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return addRelayerStartFlags(command)
}
