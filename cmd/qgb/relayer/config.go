package relayer

import (
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/spf13/cobra"

	ethcmn "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

const (
	//nolint:unused
	celestiaChainIDFlag = "celes-chain-id"
	privateKeyFlag      = "eth-priv-key"
	evmChainIDFlag      = "evm-chain-id"
	celesGRPCFlag       = "celes-grpc"
	tendermintRPCFlag   = "celes-http-rpc"
	evmRPCFlag          = "evm-rpc"
	contractAddressFlag = "contract-address"
	//nolint:unused
	startingNonceFlag = "starting-nonce"
	evmGasLimitFlag   = "evm-gas-limit"
)

func addRelayerFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(privateKeyFlag, "d", "", "Provide the private key used to sign relayed evm transactions")
	cmd.Flags().Uint64P(evmChainIDFlag, "z", 5, "Specify the evm chain id")
	cmd.Flags().StringP(celesGRPCFlag, "c", "localhost:9090", "Specify the grpc address")
	cmd.Flags().StringP(tendermintRPCFlag, "t", "http://localhost:26657", "Specify the rest rpc address")
	cmd.Flags().StringP(evmRPCFlag, "e", "http://localhost:8545", "Specify the ethereum rpc address")
	cmd.Flags().StringP(contractAddressFlag, "a", "", "Specify the contract at which the qgb is deployed")
	cmd.Flags().Uint64P(evmGasLimitFlag, "l", evm.DEFAULTEVMGASLIMIT, "Specify the evm gas limit")

	return cmd
}

type relayerConfig struct {
	evmChainID                       uint64
	evmRPC, celesGRPC, tendermintRPC string
	privateKey                       *ecdsa.PrivateKey
	contractAddr                     ethcmn.Address
	evmGasLimit                      uint64
}

func parseRelayerFlags(cmd *cobra.Command) (relayerConfig, error) {
	rawPrivateKey, err := cmd.Flags().GetString(privateKeyFlag)
	if err != nil {
		return relayerConfig{}, err
	}
	if rawPrivateKey == "" {
		return relayerConfig{}, errors.New("private key flag required")
	}
	ethPrivKey, err := ethcrypto.HexToECDSA(rawPrivateKey)
	if err != nil {
		return relayerConfig{}, fmt.Errorf("failed to hex-decode Ethereum ECDSA Private Key: %w", err)
	}
	evmChainID, err := cmd.Flags().GetUint64(evmChainIDFlag)
	if err != nil {
		return relayerConfig{}, err
	}
	tendermintRPC, err := cmd.Flags().GetString(tendermintRPCFlag)
	if err != nil {
		return relayerConfig{}, err
	}
	celesGRPC, err := cmd.Flags().GetString(celesGRPCFlag)
	if err != nil {
		return relayerConfig{}, err
	}
	contractAddr, err := cmd.Flags().GetString(contractAddressFlag)
	if err != nil {
		return relayerConfig{}, err
	}
	if contractAddr == "" {
		return relayerConfig{}, fmt.Errorf("contract address flag is required: %s", contractAddressFlag)
	}
	if !ethcmn.IsHexAddress(contractAddr) {
		return relayerConfig{}, fmt.Errorf("valid contract address flag is required: %s", contractAddressFlag)
	}
	address := ethcmn.HexToAddress(contractAddr)
	ethRPC, err := cmd.Flags().GetString(evmRPCFlag)
	if err != nil {
		return relayerConfig{}, err
	}
	evmGasLimit, err := cmd.Flags().GetUint64(evmGasLimitFlag)
	if err != nil {
		return relayerConfig{}, err
	}

	return relayerConfig{
		privateKey:    ethPrivKey,
		evmChainID:    evmChainID,
		celesGRPC:     celesGRPC,
		tendermintRPC: tendermintRPC,
		contractAddr:  address,
		evmRPC:        ethRPC,
		evmGasLimit:   evmGasLimit,
	}, nil
}
