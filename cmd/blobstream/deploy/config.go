package deploy

import (
	"errors"
	"fmt"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/base"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/spf13/cobra"
)

const (
	FlagEVMAccAddress   = "evm.account"
	FlagEVMChainID      = "evm.chain-id"
	FlagEVMRPC          = "evm.rpc"
	FlagEVMGasLimit     = "evm.gas-limit"
	FlagCoreGRPCHost    = "core.grpc.host"
	FlagCoreGRPCPort    = "core.grpc.port"
	FlagCoreRPCHost     = "core.rpc.host"
	FlagCoreRPCPort     = "core.rpc.port"
	FlagStartingNonce   = "starting-nonce"
	ServiceNameDeployer = "deployer"
)

func addDeployFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(FlagEVMAccAddress, "", "Specify the EVM account address to use for signing (Note: the private key should be in the keystore)")
	cmd.Flags().Uint64(FlagEVMChainID, 5, "Specify the evm chain id")
	cmd.Flags().String(FlagCoreGRPCHost, "localhost", "Specify the grpc address host")
	cmd.Flags().Uint(FlagCoreGRPCPort, 9090, "Specify the grpc address port")
	cmd.Flags().String(FlagCoreRPCHost, "localhost", "Specify the rpc address host")
	cmd.Flags().Uint(FlagCoreRPCPort, 26657, "Specify the rpc address port")
	cmd.Flags().String(FlagEVMRPC, "http://localhost:8545", "Specify the ethereum rpc address")
	cmd.Flags().String(
		FlagStartingNonce,
		"latest",
		"Specify the nonce to start the Blobstream contract from. "+
			"\"earliest\": for genesis, "+
			"\"latest\": for latest valset nonce, "+
			"\"nonce\": for the latest valset before the provided nonce, provided nonce included.",
	)
	cmd.Flags().Uint64(FlagEVMGasLimit, evm.DefaultEVMGasLimit, "Specify the evm gas limit")
	homeDir, err := base.DefaultServicePath(ServiceNameDeployer)
	if err != nil {
		panic(err)
	}
	cmd.Flags().String(base.FlagHome, homeDir, "The Blobstream deployer home directory")
	cmd.Flags().String(base.FlagEVMPassphrase, "", "the evm account passphrase (if not specified as a flag, it will be asked interactively)")
	base.AddGRPCInsecureFlag(cmd)
	return cmd
}

type deployConfig struct {
	*base.Config
	evmRPC            string
	coreRPC, coreGRPC string
	evmChainID        uint64
	evmAccAddress     string
	startingNonce     string
	evmGasLimit       uint64
	grpcInsecure      bool
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
	coreGRPCHost, err := cmd.Flags().GetString(FlagCoreGRPCHost)
	if err != nil {
		return deployConfig{}, err
	}
	coreGRPCPort, err := cmd.Flags().GetUint(FlagCoreGRPCPort)
	if err != nil {
		return deployConfig{}, err
	}
	coreRPCHost, err := cmd.Flags().GetString(FlagCoreRPCHost)
	if err != nil {
		return deployConfig{}, err
	}
	coreRPCPort, err := cmd.Flags().GetUint(FlagCoreRPCPort)
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
	grpcInsecure, err := cmd.Flags().GetBool(base.FlagGRPCInsecure)
	if err != nil {
		return deployConfig{}, err
	}

	return deployConfig{
		evmAccAddress: evmAccAddr,
		evmChainID:    evmChainID,
		coreGRPC:      fmt.Sprintf("%s:%d", coreGRPCHost, coreGRPCPort),
		coreRPC:       fmt.Sprintf("tcp://%s:%d", coreRPCHost, coreRPCPort),
		evmRPC:        evmRPC,
		startingNonce: startingNonce,
		evmGasLimit:   evmGasLimit,
		Config: &base.Config{
			Home:          homeDir,
			EVMPassphrase: passphrase,
		},
		grpcInsecure: grpcInsecure,
	}, nil
}
