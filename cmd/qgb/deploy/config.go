package deploy

import (
	"crypto/ecdsa"
	"errors"
	"fmt"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

const (
	privateKeyFlag    = "evm-priv-key"
	evmChainIDFlag    = "evm-chain-id"
	celesGRPCFlag     = "celes-grpc"
	evmRPCFlag        = "evm-rpc"
	startingNonceFlag = "starting-nonce"
	evmGasLimitFlag   = "evm-gas-limit"
)

func addDeployFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(privateKeyFlag, "d", "", "Provide the private key used to sign the deploy transaction")
	cmd.Flags().Uint64P(evmChainIDFlag, "z", 5, "Specify the evm chain id")
	cmd.Flags().StringP(celesGRPCFlag, "c", "localhost:9090", "Specify the grpc address")
	cmd.Flags().StringP(evmRPCFlag, "e", "http://localhost:8545", "Specify the ethereum rpc address")
	cmd.Flags().StringP(
		startingNonceFlag,
		"n",
		"latest",
		"Specify the nonce to start the QGB contract from. "+
			"\"earliest\": for genesis, "+
			"\"latest\": for latest valset nonce, "+
			"\"nonce\": for the latest valset before the provided nonce, provided nonce included.",
	)
	cmd.Flags().Uint64P(evmGasLimitFlag, "l", evm.DefaultEVMGasLimit, "Specify the evm gas limit")

	return cmd
}

type deployConfig struct {
	evmRPC, celesGRPC string
	evmChainID        uint64
	privateKey        *ecdsa.PrivateKey
	startingNonce     string
	evmGasLimit       uint64
}

func parseDeployFlags(cmd *cobra.Command) (deployConfig, error) {
	rawPrivateKey, err := cmd.Flags().GetString(privateKeyFlag)
	if err != nil {
		return deployConfig{}, err
	}
	if rawPrivateKey == "" {
		return deployConfig{}, errors.New("private key flag required")
	}
	ethPrivKey, err := ethcrypto.HexToECDSA(rawPrivateKey)
	if err != nil {
		return deployConfig{}, fmt.Errorf("failed to hex-decode Ethereum ECDSA Private Key: %w", err)
	}
	evmChainID, err := cmd.Flags().GetUint64(evmChainIDFlag)
	if err != nil {
		return deployConfig{}, err
	}
	celesGRPC, err := cmd.Flags().GetString(celesGRPCFlag)
	if err != nil {
		return deployConfig{}, err
	}
	evmRPC, err := cmd.Flags().GetString(evmRPCFlag)
	if err != nil {
		return deployConfig{}, err
	}
	startingNonce, err := cmd.Flags().GetString(startingNonceFlag)
	if err != nil {
		return deployConfig{}, err
	}
	evmGasLimit, err := cmd.Flags().GetUint64(evmGasLimitFlag)
	if err != nil {
		return deployConfig{}, err
	}

	return deployConfig{
		privateKey:    ethPrivKey,
		evmChainID:    evmChainID,
		celesGRPC:     celesGRPC,
		evmRPC:        evmRPC,
		startingNonce: startingNonce,
		evmGasLimit:   evmGasLimit,
	}, nil
}
