package orchestrator

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/cobra"
)

const (
	celestiaChainIDFlag  = "celes-chain-id"
	celestiaGRPCFlag     = "celes-grpc"
	evmPrivateKeyFlag    = "evm-priv-key"
	tendermintRPCFlag    = "celes-http-rpc"
	bootstrappersFlag    = "bootstrappers"
	p2pListenAddressFlag = "p2p-listen-addr"
	p2pIdentityFlag      = "p2p-priv-key"
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
	cmd.Flags().StringP(bootstrappersFlag, "b", "", "Comma-separated multiaddresses of p2p peers to connect to")
	cmd.Flags().StringP(p2pIdentityFlag, "p", "", "Ed25519 private key in hex format (without 0x) for the p2p peer identity. Will create a new one if none is specified")
	cmd.Flags().StringP(p2pListenAddressFlag, "q", "/ip4/0.0.0.0/tcp/30000", "MultiAddr for the p2p peer to listen on")
	return cmd
}

type Config struct {
	celestiaChainID, celesGRPC, tendermintRPC string
	evmPrivateKey                             *ecdsa.PrivateKey
	bootstrappers, p2pListenAddr              string
	p2pIdentity                               crypto.PrivKey
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
	bootstrappers, err := cmd.Flags().GetString(bootstrappersFlag)
	if err != nil {
		return Config{}, err
	}
	p2pListenAddress, err := cmd.Flags().GetString(p2pListenAddressFlag)
	if err != nil {
		return Config{}, err
	}
	hexIdentity, err := cmd.Flags().GetString(p2pIdentityFlag)
	if err != nil {
		return Config{}, err
	}
	bIdentity, err := hex.DecodeString(hexIdentity)
	if err != nil {
		return Config{}, err
	}
	identity, err := crypto.UnmarshalEd25519PrivateKey(bIdentity)
	if err != nil {
		return Config{}, err
	}

	return Config{
		evmPrivateKey:   evmPrivKey,
		celestiaChainID: chainID,
		celesGRPC:       celesGRPC,
		tendermintRPC:   tendermintRPC,
		bootstrappers:   bootstrappers,
		p2pIdentity:     identity,
		p2pListenAddr:   p2pListenAddress,
	}, nil
}
