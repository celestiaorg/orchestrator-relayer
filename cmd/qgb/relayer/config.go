package relayer

import (
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/client/flags"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/base"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/spf13/cobra"

	ethcmn "github.com/ethereum/go-ethereum/common"
)

const (
	FlagEVMAccAddress   = "evm.account"
	FlagEVMChainID      = "evm.chain-id"
	FlagCoreGRPCHost    = "core.grpc.host"
	FlagCoreGRPCPort    = "core.grpc.port"
	FlagCoreRPCHost     = "core.rpc.host"
	FlagCoreRPCPort     = "core.rpc.port"
	FlagEVMRPC          = "evm.rpc"
	FlagContractAddress = "evm.contract-address"
	FlagEVMGasLimit     = "evm.gas-limit"
	ServiceNameRelayer  = "relayer"
)

func addRelayerStartFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(FlagEVMAccAddress, "", "Specify the EVM account address to use for signing (Note: the private key should be in the keystore)")
	cmd.Flags().Uint64(FlagEVMChainID, 5, "Specify the evm chain id")
	cmd.Flags().String(FlagCoreGRPCHost, "localhost", "Specify the grpc address host")
	cmd.Flags().Uint(FlagCoreGRPCPort, 9090, "Specify the grpc address port")
	cmd.Flags().String(FlagCoreRPCHost, "localhost", "Specify the rest rpc address host")
	cmd.Flags().Uint(FlagCoreRPCPort, 26657, "Specify the rest rpc address port")
	cmd.Flags().String(FlagEVMRPC, "http://localhost:8545", "Specify the ethereum rpc address")
	cmd.Flags().String(FlagContractAddress, "", "Specify the contract at which the qgb is deployed")
	cmd.Flags().Uint64(FlagEVMGasLimit, evm.DefaultEVMGasLimit, "Specify the evm gas limit")
	homeDir, err := base.DefaultServicePath(ServiceNameRelayer)
	if err != nil {
		panic(err)
	}
	cmd.Flags().String(base.FlagHome, homeDir, "The qgb relayer home directory")
	cmd.Flags().String(base.FlagEVMPassphrase, "", "the evm account passphrase (if not specified as a flag, it will be asked interactively)")
	base.AddP2PNicknameFlag(cmd)
	base.AddP2PListenAddressFlag(cmd)
	base.AddBootstrappersFlag(cmd)

	return cmd
}

type StartConfig struct {
	*base.Config
	evmChainID                   uint64
	evmRPC, coreGRPC, coreRPC    string
	evmAccAddress                string
	contractAddr                 ethcmn.Address
	evmGasLimit                  uint64
	bootstrappers, p2pListenAddr string
	p2pNickname                  string
}

func parseRelayerStartFlags(cmd *cobra.Command) (StartConfig, error) {
	evmAccAddr, err := cmd.Flags().GetString(FlagEVMAccAddress)
	if err != nil {
		return StartConfig{}, err
	}
	if evmAccAddr == "" {
		return StartConfig{}, errors.New("the evm account address should be specified")
	}
	evmChainID, err := cmd.Flags().GetUint64(FlagEVMChainID)
	if err != nil {
		return StartConfig{}, err
	}
	coreRPCHost, err := cmd.Flags().GetString(FlagCoreRPCHost)
	if err != nil {
		return StartConfig{}, err
	}
	coreRPCPort, err := cmd.Flags().GetUint(FlagCoreRPCPort)
	if err != nil {
		return StartConfig{}, err
	}
	coreGRPCHost, err := cmd.Flags().GetString(FlagCoreGRPCHost)
	if err != nil {
		return StartConfig{}, err
	}
	coreGRPCPort, err := cmd.Flags().GetUint(FlagCoreGRPCPort)
	if err != nil {
		return StartConfig{}, err
	}
	contractAddr, err := cmd.Flags().GetString(FlagContractAddress)
	if err != nil {
		return StartConfig{}, err
	}
	if contractAddr == "" {
		return StartConfig{}, fmt.Errorf("contract address flag is required: %s", FlagContractAddress)
	}
	if !ethcmn.IsHexAddress(contractAddr) {
		return StartConfig{}, fmt.Errorf("valid contract address flag is required: %s", FlagContractAddress)
	}
	address := ethcmn.HexToAddress(contractAddr)
	evmRPC, err := cmd.Flags().GetString(FlagEVMRPC)
	if err != nil {
		return StartConfig{}, err
	}
	evmGasLimit, err := cmd.Flags().GetUint64(FlagEVMGasLimit)
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
		homeDir, err = base.DefaultServicePath(ServiceNameRelayer)
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
		evmChainID:    evmChainID,
		coreGRPC:      fmt.Sprintf("%s:%d", coreGRPCHost, coreGRPCPort),
		coreRPC:       fmt.Sprintf("tcp://%s:%d", coreRPCHost, coreRPCPort),
		contractAddr:  address,
		evmRPC:        evmRPC,
		evmGasLimit:   evmGasLimit,
		bootstrappers: bootstrappers,
		p2pListenAddr: p2pListenAddress,
		p2pNickname:   p2pNickname,
		Config: &base.Config{
			Home:          homeDir,
			EVMPassphrase: passphrase,
		},
	}, nil
}

func addInitFlags(cmd *cobra.Command) *cobra.Command {
	homeDir, err := base.DefaultServicePath(ServiceNameRelayer)
	if err != nil {
		panic(err)
	}
	cmd.Flags().String(base.FlagHome, homeDir, "The qgb relayer home directory")
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
