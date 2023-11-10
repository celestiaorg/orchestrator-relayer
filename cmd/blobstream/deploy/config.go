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

func parseDeployFlags(cmd *cobra.Command) (deployConfig, error) {
	evmAccAddr, _, err := base.GetEVMAccAddressFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}
	if evmAccAddr == "" {
		return deployConfig{}, errors.New("the evm account address should be specified")
	}

	evmChainID, _, err := base.GetEVMChainIDFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}

	coreRPC, _, err := base.GetCoreRPCFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}
	if !strings.HasPrefix(coreRPC, "tcp://") {
		coreRPC = fmt.Sprintf("tcp://%s", coreRPC)
	}

	coreGRPC, _, err := base.GetCoreGRPCFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}

	evmRPC, _, err := base.GetEVMRPCFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}

	startingNonce, _, err := base.GetStartingNonceFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}

	evmGasLimit, _, err := base.GetEVMGasLimitFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}

	homeDir, _, err := base.GetHomeFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}

	passphrase, _, err := base.GetEVMPassphraseFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}

	grpcInsecure, _, err := base.GetGRPCInsecureFlag(cmd)
	if err != nil {
		return deployConfig{}, err
	}

	return deployConfig{
		Config: base.Config{
			Home:          homeDir,
			EVMPassphrase: passphrase,
		},
		evmRPC:        evmRPC,
		coreRPC:       coreRPC,
		coreGRPC:      coreGRPC,
		evmChainID:    evmChainID,
		evmAccAddress: evmAccAddr,
		startingNonce: startingNonce,
		evmGasLimit:   evmGasLimit,
		grpcInsecure:  grpcInsecure,
	}, nil
}
