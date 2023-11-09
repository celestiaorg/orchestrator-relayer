package deploy

import (
	"errors"
	"fmt"
	"strings"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/base"

	"github.com/spf13/cobra"
)

const (
	ServiceNameDeployer = "deployer"
)

func addDeployFlags(cmd *cobra.Command) *cobra.Command {
	base.AddEVMAccAddressFlag(cmd)
	base.AddEVMChainIDFlag(cmd)
	base.AddCoreGRPCFlag(cmd)
	base.AddCoreRPCFlag(cmd)
	base.AddEVMRPCFlag(cmd)
	base.AddStartingNonceFlag(cmd)
	base.AddEVMGasLimitFlag(cmd)
	base.AddEVMPassphraseFlag(cmd)
	homeDir, err := base.DefaultServicePath(ServiceNameDeployer)
	if err != nil {
		panic(err)
	}
	base.AddHomeFlag(cmd, ServiceNameDeployer, homeDir)
	base.AddGRPCInsecureFlag(cmd)
	return cmd
}

type deployConfig struct {
	base.Config
	evmRPC            string
	coreRPC, coreGRPC string
	evmChainID        uint64
	evmAccAddress     string
	startingNonce     string
	evmGasLimit       uint64
	grpcInsecure      bool
}

func parseDeployFlags(cmd *cobra.Command, fileConfig *deployConfig) (deployConfig, error) {
	evmAccAddr, changed, err := base.GetEVMAccAddressFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}
	if changed {
		if evmAccAddr == "" && fileConfig.evmAccAddress == "" {
			return deployConfig{}, errors.New("the evm account address should be specified")
		}
		fileConfig.evmAccAddress = evmAccAddr
	}

	evmChainID, changed, err := base.GetEVMChainIDFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}
	if changed {
		fileConfig.evmChainID = evmChainID
	}

	coreRPC, changed, err := base.GetCoreRPCFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}
	if changed {
		if !strings.HasPrefix(coreRPC, "tcp://") {
			coreRPC = fmt.Sprintf("tcp://%s", coreRPC)
		}
		fileConfig.coreRPC = coreRPC
	}

	coreGRPC, changed, err := base.GetCoreGRPCFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}
	if changed {
		fileConfig.coreGRPC = coreGRPC
	}

	evmRPC, changed, err := base.GetEVMRPCFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}
	if changed {
		fileConfig.evmRPC = evmRPC
	}

	startingNonce, changed, err := base.GetStartingNonceFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}
	if changed {
		fileConfig.startingNonce = startingNonce
	}

	evmGasLimit, changed, err := base.GetEVMGasLimitFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}
	if changed {
		fileConfig.evmGasLimit = evmGasLimit
	}

	homeDir, changed, err := base.GetHomeFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}
	if changed {
		fileConfig.Home = homeDir
	}

	passphrase, changed, err := base.GetEVMPassphraseFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}
	if changed {
		fileConfig.EVMPassphrase = passphrase
	}

	grpcInsecure, changed, err := base.GetGRPCInsecureFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}
	if changed {
		fileConfig.grpcInsecure = grpcInsecure
	}

	return *fileConfig, nil
}
