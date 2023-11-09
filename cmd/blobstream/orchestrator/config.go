package orchestrator

import (
	"errors"
	"fmt"
	"strings"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/base"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

const (
	ServiceNameOrchestrator = "orchestrator"
)

func addOrchestratorFlags(cmd *cobra.Command) *cobra.Command {
	base.AddCoreRPCFlag(cmd)
	base.AddCoreGRPCFlag(cmd)
	base.AddEVMAccAddressFlag(cmd)
	base.AddEVMPassphraseFlag(cmd)
	homeDir, err := base.DefaultServicePath(ServiceNameOrchestrator)
	if err != nil {
		panic(err)
	}
	base.AddHomeFlag(cmd, ServiceNameOrchestrator, homeDir)
	base.AddP2PNicknameFlag(cmd)
	base.AddP2PListenAddressFlag(cmd)
	base.AddBootstrappersFlag(cmd)
	base.AddGRPCInsecureFlag(cmd)
	return cmd
}

type StartConfig struct {
	base.Config
	coreGRPC, coreRPC            string
	evmAccAddress                string
	bootstrappers, p2pListenAddr string
	p2pNickname                  string
	grpcInsecure                 bool
}

func parseOrchestratorFlags(cmd *cobra.Command, fileConfig *StartConfig) (StartConfig, error) {
	evmAccAddr, changed, err := base.GetEVMAccAddressFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		if evmAccAddr == "" && fileConfig.evmAccAddress == "" {
			return StartConfig{}, errors.New("the evm account address should be specified")
		}
		fileConfig.evmAccAddress = evmAccAddr
	}

	coreRPC, changed, err := base.GetCoreRPCFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		if !strings.HasPrefix(coreRPC, "tcp://") {
			coreRPC = fmt.Sprintf("tcp://%s", coreRPC)
		}
		fileConfig.coreRPC = coreRPC
	}

	coreGRPC, changed, err := base.GetCoreGRPCFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.coreGRPC = coreGRPC
	}

	bootstrappers, changed, err := base.GetBootstrappersFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.bootstrappers = bootstrappers
	}

	p2pListenAddress, changed, err := base.GetP2PListenAddressFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.p2pListenAddr = p2pListenAddress
	}

	p2pNickname, changed, err := base.GetP2PNicknameFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.p2pNickname = p2pNickname
	}

	homeDir, changed, err := base.GetHomeFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.Home = homeDir
	}

	passphrase, changed, err := base.GetEVMPassphraseFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.EVMPassphrase = passphrase
	}

	grpcInsecure, changed, err := base.GetGRPCInsecureFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.grpcInsecure = grpcInsecure
	}

	return *fileConfig, nil
}

func addInitFlags(cmd *cobra.Command) *cobra.Command {
	homeDir, err := base.DefaultServicePath(ServiceNameOrchestrator)
	if err != nil {
		panic(err)
	}
	base.AddHomeFlag(cmd, ServiceNameOrchestrator, homeDir)
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
