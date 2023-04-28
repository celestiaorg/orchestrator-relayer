package deploy

import (
	"errors"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/base"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/spf13/cobra"
)

const (
	FlagEVMAccAddress   = "evm-address"
	FlagEVMChainID      = "evm-chain-id"
	FlagCelesGRPC       = "celes-grpc"
	FlagEVMRPC          = "evm-rpc"
	FlagStartingNonce   = "starting-nonce"
	FlagEVMGasLimit     = "evm-gas-limit"
	ServiceNameDeployer = "deployer"
)

func addDeployFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(FlagEVMAccAddress, "d", "", "Specify the EVM account address to use for signing (Note: the private key should be in the keystore)")
	cmd.Flags().Uint64P(FlagEVMChainID, "z", 5, "Specify the evm chain id")
	cmd.Flags().StringP(FlagCelesGRPC, "c", "localhost:9090", "Specify the grpc address")
	cmd.Flags().StringP(FlagEVMRPC, "e", "http://localhost:8545", "Specify the ethereum rpc address")
	cmd.Flags().StringP(
		FlagStartingNonce,
		"n",
		"latest",
		"Specify the nonce to start the QGB contract from. "+
			"\"earliest\": for genesis, "+
			"\"latest\": for latest valset nonce, "+
			"\"nonce\": for the latest valset before the provided nonce, provided nonce included.",
	)
	cmd.Flags().Uint64P(FlagEVMGasLimit, "l", evm.DefaultEVMGasLimit, "Specify the evm gas limit")
	homeDir, err := base.DefaultServicePath(ServiceNameDeployer)
	if err != nil {
		panic(err)
	}
	cmd.Flags().String(base.FlagHome, homeDir, "The qgb deployer home directory")
	cmd.Flags().String(base.FlagEVMPassphrase, "", "the evm account passphrase (if not specified as a flag, it will be asked interactively)")

	return cmd
}

type deployConfig struct {
	*base.Config
	evmRPC, celesGRPC string
	evmChainID        uint64
	evmAccAddress     string
	startingNonce     string
	evmGasLimit       uint64
}

func parseDeployFlags(cmd *cobra.Command) (deployConfig, error) {
	evmAccAddr, err := cmd.Flags().GetString(FlagEVMAccAddress)
	if err != nil {
		return deployConfig{}, err
	}
	if evmAccAddr == "" {
		return deployConfig{}, errors.New("the evm account address should be specified")
	}
	evmChainID, err := cmd.Flags().GetUint64(FlagEVMChainID)
	if err != nil {
		return deployConfig{}, err
	}
	celesGRPC, err := cmd.Flags().GetString(FlagCelesGRPC)
	if err != nil {
		return deployConfig{}, err
	}
	evmRPC, err := cmd.Flags().GetString(FlagEVMRPC)
	if err != nil {
		return deployConfig{}, err
	}
	startingNonce, err := cmd.Flags().GetString(FlagStartingNonce)
	if err != nil {
		return deployConfig{}, err
	}
	evmGasLimit, err := cmd.Flags().GetUint64(FlagEVMGasLimit)
	if err != nil {
		return deployConfig{}, err
	}
	homeDir, err := cmd.Flags().GetString(base.FlagHome)
	if err != nil {
		return deployConfig{}, err
	}
	if homeDir == "" {
		var err error
		homeDir, err = base.DefaultServicePath(ServiceNameDeployer)
		if err != nil {
			return deployConfig{}, err
		}
	}
	passphrase, err := cmd.Flags().GetString(base.FlagEVMPassphrase)
	if err != nil {
		return deployConfig{}, err
	}

	return deployConfig{
		evmAccAddress: evmAccAddr,
		evmChainID:    evmChainID,
		celesGRPC:     celesGRPC,
		evmRPC:        evmRPC,
		startingNonce: startingNonce,
		evmGasLimit:   evmGasLimit,
		Config: &base.Config{
			Home:          homeDir,
			EVMPassphrase: passphrase,
		},
	}, nil
}
