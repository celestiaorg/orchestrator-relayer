package relayer

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/base"

	"github.com/spf13/cobra"

	ethcmn "github.com/ethereum/go-ethereum/common"
)

const (
	ServiceNameRelayer = "relayer"
)

func addRelayerStartFlags(cmd *cobra.Command) *cobra.Command {
	homeDir, err := base.DefaultServicePath(ServiceNameRelayer)
	if err != nil {
		panic(err)
	}
	base.AddHomeFlag(cmd, ServiceNameRelayer, homeDir)
	base.AddEVMAccAddressFlag(cmd)
	base.AddEVMChainIDFlag(cmd)
	base.AddCoreGRPCFlag(cmd)
	base.AddCoreRPCFlag(cmd)
	base.AddEVMRPCFlag(cmd)
	base.AddEVMContractAddressFlag(cmd)
	base.AddEVMGasLimitFlag(cmd)
	base.AddEVMPassphraseFlag(cmd)
	base.AddP2PNicknameFlag(cmd)
	base.AddP2PListenAddressFlag(cmd)
	base.AddBootstrappersFlag(cmd)
	base.AddGRPCInsecureFlag(cmd)

	return cmd
}

type StartConfig struct {
	base.Config
	evmChainID                   uint64
	evmRPC, coreGRPC, coreRPC    string
	evmAccAddress                string
	contractAddr                 ethcmn.Address
	evmGasLimit                  uint64
	bootstrappers, p2pListenAddr string
	p2pNickname                  string
	grpcInsecure                 bool
}

// TODO add a validate basics

func parseRelayerStartFlags(cmd *cobra.Command, fileConfig *StartConfig) (StartConfig, error) {
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

	evmChainID, changed, err := base.GetEVMChainIDFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.evmChainID = evmChainID
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

	contractAddr, changed, err := base.GetEVMContractAddressFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		if contractAddr == "" {
			// TODO fix
			return StartConfig{}, fmt.Errorf("contract address flag is required: %s", base.FlagEVMContractAddress)
		}
		if !ethcmn.IsHexAddress(contractAddr) {
			// TODO probably not the right place
			return StartConfig{}, fmt.Errorf("valid contract address flag is required: %s", base.FlagEVMContractAddress)
		}
		address := ethcmn.HexToAddress(contractAddr)
		fileConfig.contractAddr = address
	}

	evmRPC, changed, err := base.GetEVMRPCFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.evmRPC = evmRPC
	}

	evmGasLimit, changed, err := base.GetEVMGasLimitFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}
	if changed {
		fileConfig.evmGasLimit = evmGasLimit
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
	homeDir, err := base.DefaultServicePath(ServiceNameRelayer)
	if err != nil {
		panic(err)
	}
	base.AddHomeFlag(cmd, ServiceNameRelayer, homeDir)
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
		homeDir, err = base.DefaultServicePath(ServiceNameRelayer)
		if err != nil {
			return InitConfig{}, err
		}
	}

	return InitConfig{
		home: homeDir,
	}, nil
}
