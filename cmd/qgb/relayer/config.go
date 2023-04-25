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
	FlagEVMAccAddress    = "evm-address"
	FlagEVMChainID       = "evm-chain-id"
	FlagCelesGRPC        = "celes-grpc"
	FlagTendermintRPC    = "celes-http-rpc"
	FlagEVMRPC           = "evm-rpc"
	FlagContractAddress  = "contract-address"
	FlagEVMGasLimit      = "evm-gas-limit"
	FlagBootstrappers    = "p2p-bootstrappers"
	FlagP2PListenAddress = "p2p-listen-addr"
	FlagP2PNickname      = "p2p-nickname"
	FlagP2PNode          = "p2p-node"
)

func addRelayerStartFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(FlagEVMAccAddress, "d", "", "Specify the EVM account address to use for signing (Note: the private key should be in the keystore)")
	cmd.Flags().Uint64P(FlagEVMChainID, "z", 5, "Specify the evm chain id")
	cmd.Flags().StringP(FlagCelesGRPC, "c", "localhost:9090", "Specify the grpc address")
	cmd.Flags().StringP(FlagTendermintRPC, "t", "http://localhost:26657", "Specify the rest rpc address")
	cmd.Flags().StringP(FlagEVMRPC, "e", "http://localhost:8545", "Specify the ethereum rpc address")
	cmd.Flags().StringP(FlagContractAddress, "a", "", "Specify the contract at which the qgb is deployed")
	cmd.Flags().Uint64P(FlagEVMGasLimit, "l", evm.DefaultEVMGasLimit, "Specify the evm gas limit")
	cmd.Flags().StringP(FlagBootstrappers, "b", "", "Comma-separated multiaddresses of p2p peers to connect to")
	cmd.Flags().StringP(FlagP2PNickname, "p", "", "Nickname of the p2p private key to use (if not provided, an existing one from the p2p store or a newly generated one will be used)")
	cmd.Flags().StringP(FlagP2PListenAddress, "q", "/ip4/127.0.0.1/tcp/30000", "MultiAddr for the p2p peer to listen on")
	cmd.Flags().String(base.FlagHome, "", "The qgb relayer home directory")
	cmd.Flags().String(base.FlagEVMPassphrase, "", "the evm account passphrase (if not specified as a flag, it will be asked interactively)")

	return cmd
}

type StartConfig struct {
	*base.Config
	evmChainID                       uint64
	evmRPC, celesGRPC, tendermintRPC string
	evmAccAddress                    string
	contractAddr                     ethcmn.Address
	evmGasLimit                      uint64
	bootstrappers, p2pListenAddr     string
	p2pNickname                      string
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
	tendermintRPC, err := cmd.Flags().GetString(FlagTendermintRPC)
	if err != nil {
		return StartConfig{}, err
	}
	celesGRPC, err := cmd.Flags().GetString(FlagCelesGRPC)
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
	ethRPC, err := cmd.Flags().GetString(FlagEVMRPC)
	if err != nil {
		return StartConfig{}, err
	}
	evmGasLimit, err := cmd.Flags().GetUint64(FlagEVMGasLimit)
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
		homeDir, err = base.DefaultServicePath("relayer")
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
		celesGRPC:     celesGRPC,
		tendermintRPC: tendermintRPC,
		contractAddr:  address,
		evmRPC:        ethRPC,
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

func addRelayerQueryFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(FlagCelesGRPC, "c", "localhost:9090", "Specify the grpc address")
	cmd.Flags().StringP(FlagTendermintRPC, "t", "http://localhost:26657", "Specify the rest rpc address")
	cmd.Flags().StringP(FlagP2PNode, "n", "", "P2P target node multiaddress (eg. /ip4/127.0.0.1/tcp/30000/p2p/12D3KooWBSMasWzRSRKXREhediFUwABNZwzJbkZcYz5rYr9Zdmfn)")
	cmd.Flags().String(base.FlagHome, "", "The qgb relayer home directory")

	return cmd
}

type QueryConfig struct {
	home                     string
	celesGRPC, tendermintRPC string
	targetNode               string
}

func parseRelayerQueryFlags(cmd *cobra.Command) (QueryConfig, error) {
	tendermintRPC, err := cmd.Flags().GetString(FlagTendermintRPC)
	if err != nil {
		return QueryConfig{}, err
	}
	celesGRPC, err := cmd.Flags().GetString(FlagCelesGRPC)
	if err != nil {
		return QueryConfig{}, err
	}
	targetNode, err := cmd.Flags().GetString(FlagP2PNode)
	if err != nil {
		return QueryConfig{}, err
	}
	homeDir, err := cmd.Flags().GetString(base.FlagHome)
	if err != nil {
		return QueryConfig{}, err
	}
	if homeDir == "" {
		var err error
		homeDir, err = base.DefaultServicePath("relayer")
		if err != nil {
			return QueryConfig{}, err
		}
	}

	return QueryConfig{
		celesGRPC:     celesGRPC,
		tendermintRPC: tendermintRPC,
		targetNode:    targetNode,
		home:          homeDir,
	}, nil
}

func addInitFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(base.FlagHome, "", "The qgb relayer home directory")
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
		homeDir, err = base.DefaultServicePath("relayer")
		if err != nil {
			return InitConfig{}, err
		}
	}

	return InitConfig{
		home: homeDir,
	}, nil
}
