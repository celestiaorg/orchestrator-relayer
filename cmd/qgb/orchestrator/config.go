package orchestrator

import (
	"crypto/ecdsa"
	"errors"
	"fmt"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/spf13/cobra"
)

const (
	celestiaChainIDFlag = "celes-chain-id"
	celestiaGRPCFlag    = "celes-grpc"
	evmPrivateKeyFlag   = "evm-priv-key"
	tendermintRPCFlag   = "celes-http-rpc"
)

func addOrchestratorFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(celestiaChainIDFlag, "x", "user", "Specify the celestia chain id")
	cmd.Flags().StringP(tendermintRPCFlag, "t", "http://localhost:26657", "Specify the rest rpc address")
	cmd.Flags().StringP(celestiaGRPCFlag, "c", "localhost:9090", "Specify the grpc address")
	cmd.Flags().StringP(
		evmPrivateKeyFlag,
		"d",
		"",
		"Specify the ECDSA private key used to sign orchestrator commitments in hex",
	)
	return cmd
}

type Config struct {
	celestiaChainID, celesGRPC, tendermintRPC string
	privateKey                                *ecdsa.PrivateKey
}

func parseOrchestratorFlags(cmd *cobra.Command) (Config, error) {
	rawPrivateKey, err := cmd.Flags().GetString(evmPrivateKeyFlag)
	if err != nil {
		return Config{}, err
	}
	if rawPrivateKey == "" {
		return Config{}, errors.New("private key flag required")
	}
	evmPrivKey, err := ethcrypto.HexToECDSA(rawPrivateKey)
	if err != nil {
		return Config{}, fmt.Errorf("failed to hex-decode EVM ECDSA Private Key: %w", err)
	}
	chainID, err := cmd.Flags().GetString(celestiaChainIDFlag)
	if err != nil {
		return Config{}, err
	}
	tendermintRPC, err := cmd.Flags().GetString(tendermintRPCFlag)
	if err != nil {
		return Config{}, err
	}
	celesGRPC, err := cmd.Flags().GetString(celestiaGRPCFlag)
	if err != nil {
		return Config{}, err
	}

	return Config{
		privateKey:      evmPrivKey,
		celestiaChainID: chainID,
		celesGRPC:       celesGRPC,
		tendermintRPC:   tendermintRPC,
	}, nil
}
