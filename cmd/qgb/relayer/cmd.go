package relayer

import (
	"context"
	"os"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/common"
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys"
	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/helpers"
	"github.com/celestiaorg/orchestrator-relayer/store"

	"github.com/celestiaorg/orchestrator-relayer/relayer"
	wrapper "github.com/celestiaorg/quantum-gravity-bridge/wrappers/QuantumGravityBridge.sol"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func Command() *cobra.Command {
	relCmd := &cobra.Command{
		Use:          "relayer",
		Aliases:      []string{"rel"},
		Short:        "QGB relayer that relays signatures to the target EVM chain",
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

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			tmQuerier, appQuerier, p2pQuerier, retrier, evmKeystore, acc, stopFuncs, err := common.InitBase(
				ctx,
				logger,
				config.tendermintRPC,
				config.celesGRPC,
				config.Home,
				config.evmAccAddress,
				config.EVMPassphrase,
				config.p2pNickname,
				config.p2pListenAddr,
				config.bootstrappers,
			)
			defer func() {
				for _, f := range stopFuncs {
					err := f()
					if err != nil {
						logger.Error(err.Error())
					}
				}
			}()
			if err != nil {
				return err
			}

			// connecting to a QGB contract
			ethClient, err := ethclient.Dial(config.evmRPC)
			if err != nil {
				return err
			}
			defer ethClient.Close()
			qgbWrapper, err := wrapper.NewQuantumGravityBridge(config.contractAddr, ethClient)
			if err != nil {
				return err
			}

			evmClient := evm.NewClient(
				logger,
				qgbWrapper,
				evmKeystore,
				acc,
				config.evmRPC,
				config.evmGasLimit,
			)

			relay := relayer.NewRelayer(
				tmQuerier,
				appQuerier,
				p2pQuerier,
				evmClient,
				logger,
				retrier,
			)

			// Listen for and trap any OS signal to graceful shutdown and exit
			go helpers.TrapSignal(logger, cancel)

			logger.Debug("starting relayer")
			err = relay.Start(ctx)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return addRelayerStartFlags(command)
}
