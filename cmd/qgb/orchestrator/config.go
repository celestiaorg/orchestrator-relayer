package orchestrator

import (
	"errors"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/base"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

const (
	FlagCelestiaGRPC     = "celes-grpc"
	FlagEVMAccAddress    = "evm-address"
	FlagTendermintRPC    = "celes-rpc"
	FlagBootstrappers    = "p2p-bootstrappers"
	FlagP2PListenAddress = "p2p-listen-addr"
	FlagP2PNickname      = "p2p-nickname"
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
	cmd.Flags().StringP(FlagBootstrappers, "b", "", "Comma-separated multiaddresses of p2p peers to connect to")
	cmd.Flags().StringP(FlagP2PNickname, "p", "", "Nickname of the p2p private key to use (if not provided, an existing one from the p2p store or a newly generated one will be used)")
	cmd.Flags().StringP(FlagP2PListenAddress, "q", "/ip4/0.0.0.0/tcp/30000", "MultiAddr for the p2p peer to listen on")
	homeDir, err := base.DefaultServicePath("orchestrator")
	if err != nil {
		panic(err)
	}
	cmd.Flags().String(base.FlagHome, homeDir, "The qgb orchestrator home directory")
	cmd.Flags().String(base.FlagEVMPassphrase, "", "the evm account passphrase (if not specified as a flag, it will be asked interactively)")

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
	bootstrappers, err := cmd.Flags().GetString(FlagBootstrappers)
	if err != nil {
		return StartConfig{}, err
	}
	p2pListenAddress, err := cmd.Flags().GetString(FlagP2PListenAddress)
	if err != nil {
		return StartConfig{}, err
	}
	p2pNickname, err := cmd.Flags().GetString(FlagP2PNickname)
	if err != nil {
		return StartConfig{}, err
	}
	homeDir, err := cmd.Flags().GetString(base.FlagHome)
	if err != nil {
		return StartConfig{}, err
	}
	if homeDir == "" {
		var err error
		homeDir, err = base.DefaultServicePath("orchestrator")
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
	homeDir, err := base.DefaultServicePath("orchestrator")
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
		homeDir, err = base.DefaultServicePath("orchestrator")
		if err != nil {
			return InitConfig{}, err
		}
	}

	return InitConfig{
		home: homeDir,
	}, nil
}
