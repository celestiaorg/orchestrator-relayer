package orchestrator

import (
	"context"
	wrapper "github.com/celestiaorg/quantum-gravity-bridge/wrappers/QuantumGravityBridge.sol"
	"github.com/ethereum/go-ethereum/ethclient"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	blobtypes "github.com/celestiaorg/celestia-app/x/blob/types"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func OrchRelayerCmd() *cobra.Command {
	command := &cobra.Command{
		Use:     "orchestrator-relayer <flags>",
		Aliases: []string{"orch-rel"},
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseOrchestratorRelayerFlags(cmd)
			if err != nil {
				return err
			}

			relayerConfig := *config.relayerConfig
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)

			ethClient, err := ethclient.Dial(relayerConfig.evmRPC)
			if err != nil {
				return err
			}
			qgbWrapper, err := wrapper.NewQuantumGravityBridge(relayerConfig.contractAddr, ethClient)
			if err != nil {
				return err
			}

			encCfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)

			querier, err := NewQuerier(config.celesGRPC, config.tendermintRPC, logger, encCfg)
			if err != nil {
				return err
			}

			relay, err := NewRelayer(
				querier,
				NewEvmClient(
					tmlog.NewTMLogger(os.Stdout),
					qgbWrapper,
					relayerConfig.privateKey,
					relayerConfig.evmRPC,
					relayerConfig.evmGasLimit,
				),
				logger,
			)

			logger.Debug("initializing orchestrator and relayer")

			ctx, cancel := context.WithCancel(cmd.Context())

			// creates the Signer
			// TODO: optionally ask for input for a password
			ring, err := keyring.New("orchestrator", config.keyringBackend, config.keyringPath, strings.NewReader(""), encCfg.Codec)
			if err != nil {
				panic(err)
			}
			signer := blobtypes.NewKeyringSigner(
				ring,
				config.keyringAccount,
				config.celestiaChainID,
			)

			broadcaster, err := NewBroadcaster(config.celesGRPC, signer, config.celestiaGasLimit, config.celestiaTxFee)
			if err != nil {
				panic(err)
			}

			retrier := NewRetrier(logger, 5)
			orch, err := NewOrchestrator(
				logger,
				querier,
				broadcaster,
				retrier,
				signer,
				*config.privateKey,
				relay,
			)
			if err != nil {
				panic(err)
			}

			logger.Debug("starting orchestrator")

			// Listen for and trap any OS signal to gracefully shutdown and exit
			go trapSignal(logger, cancel)

			orch.Start(ctx)

			return nil
		},
	}
	return addOrchestratorRelayerFlags(command)
}

// trapSignal will listen for any OS signal and gracefully exit.
func trapSignal(logger tmlog.Logger, cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)

	signal.Notify(sigCh, syscall.SIGTERM)
	signal.Notify(sigCh, syscall.SIGINT)

	sig := <-sigCh
	logger.Info("caught signal; shutting down...", "signal", sig.String())
	cancel()
}
