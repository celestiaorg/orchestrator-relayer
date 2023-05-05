package orchestrator

import (
	"errors"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/base"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

const (
	FlagCelestiaGRPC        = "celes-grpc"
	FlagEVMAccAddress       = "evm-address"
	FlagTendermintRPC       = "celes-rpc"
	ServiceNameOrchestrator = "orchestrator"
)

func addOrchestratorFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(FlagTendermintRPC, "t", "tcp://localhost:26657", "Specify the rest rpc address")
	cmd.Flags().StringP(FlagCelestiaGRPC, "c", "localhost:9090", "Specify the grpc address (without the protocol prefix)")
	cmd.Flags().StringP(
		FlagEVMAccAddress,
		"d",
		"",
		"Specify the EVM account address to use for signing (Note: the private key should be in the keystore)",
	)
	homeDir, err := base.DefaultServicePath(ServiceNameOrchestrator)
	if err != nil {
		panic(err)
	}
	cmd.Flags().String(base.FlagHome, homeDir, "The qgb orchestrator home directory")
	cmd.Flags().String(base.FlagEVMPassphrase, "", "the evm account passphrase (if not specified as a flag, it will be asked interactively)")
	base.AddP2PNicknameFlag(cmd)
	base.AddP2PListenAddressFlag(cmd)
	base.AddBootstrappersFlag(cmd)
	return cmd
}

type StartConfig struct {
	*base.Config
	celesGRPC, tendermintRPC     string
	evmAccAddress                string
	bootstrappers, p2pListenAddr string
	p2pNickname                  string
}

func parseOrchestratorFlags(cmd *cobra.Command) (StartConfig, error) {
	evmAccAddr, err := cmd.Flags().GetString(FlagEVMAccAddress)
	if err != nil {
		return StartConfig{}, err
	}
	if evmAccAddr == "" {
		return StartConfig{}, errors.New("the evm account address should be specified")
	}
	tendermintRPC, err := cmd.Flags().GetString(FlagTendermintRPC)
	if err != nil {
		return StartConfig{}, err
	}
	celesGRPC, err := cmd.Flags().GetString(FlagCelestiaGRPC)
	if err != nil {
		return StartConfig{}, err
	}
	bootstrappers, err := cmd.Flags().GetString(base.FlagBootstrappers)
	if err != nil {
		return StartConfig{}, err
	}
	p2pListenAddress, err := cmd.Flags().GetString(base.FlagP2PListenAddress)
	if err != nil {
		return StartConfig{}, err
	}
	p2pNickname, err := cmd.Flags().GetString(base.FlagP2PNickname)
	if err != nil {
		return StartConfig{}, err
	}
	homeDir, err := cmd.Flags().GetString(base.FlagHome)
	if err != nil {
		return StartConfig{}, err
	}
	if homeDir == "" {
		var err error
		homeDir, err = base.DefaultServicePath(ServiceNameOrchestrator)
		if err != nil {
			return StartConfig{}, err
		}
	}
	passphrase, err := cmd.Flags().GetString(base.FlagEVMPassphrase)
	if err != nil {
		return StartConfig{}, err
	}

	return StartConfig{
		evmAccAddress: evmAccAddr,
		celesGRPC:     celesGRPC,
		tendermintRPC: tendermintRPC,
		bootstrappers: bootstrappers,
		p2pNickname:   p2pNickname,
		p2pListenAddr: p2pListenAddress,
		Config: &base.Config{
			Home:          homeDir,
			EVMPassphrase: passphrase,
		},
	}, nil
}

func addInitFlags(cmd *cobra.Command) *cobra.Command {
	homeDir, err := base.DefaultServicePath(ServiceNameOrchestrator)
	if err != nil {
		panic(err)
	}
	cmd.Flags().String(base.FlagHome, homeDir, "The qgb orchestrator home directory")
	return cmd
}

type InitConfig struct {
	home string
}

func parseInitFlags(cmd *cobra.Command) (InitConfig, error) {
	homeDir, err := cmd.Flags().GetString(flags.FlagHome)
	if err != nil {
		return InitConfig{}, err
	}
	if homeDir == "" {
		var err error
		homeDir, err = base.DefaultServicePath(ServiceNameOrchestrator)
		if err != nil {
			return InitConfig{}, err
		}
	}

	return InitConfig{
		home: homeDir,
	}, nil
}
