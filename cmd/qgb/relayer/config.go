package relayer

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/libp2p/go-libp2p/core/crypto"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/spf13/cobra"

	ethcmn "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
)

const (
	privateKeyFlag       = "eth-priv-key"
	evmChainIDFlag       = "evm-chain-id"
	celesGRPCFlag        = "celes-grpc"
	tendermintRPCFlag    = "celes-http-rpc"
	evmRPCFlag           = "evm-rpc"
	contractAddressFlag  = "contract-address"
	evmGasLimitFlag      = "evm-gas-limit"
	bootstrappersFlag    = "bootstrappers"
	p2pListenAddressFlag = "p2p-listen-addr"
	p2pIdentityFlag      = "p2p-priv-key"
)

func addRelayerFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(privateKeyFlag, "d", "", "Provide the private key used to sign relayed evm transactions")
	cmd.Flags().Uint64P(evmChainIDFlag, "z", 5, "Specify the evm chain id")
	cmd.Flags().StringP(celesGRPCFlag, "c", "localhost:9090", "Specify the grpc address")
	cmd.Flags().StringP(tendermintRPCFlag, "t", "http://localhost:26657", "Specify the rest rpc address")
	cmd.Flags().StringP(evmRPCFlag, "e", "http://localhost:8545", "Specify the ethereum rpc address")
	cmd.Flags().StringP(contractAddressFlag, "a", "", "Specify the contract at which the qgb is deployed")
	cmd.Flags().Uint64P(evmGasLimitFlag, "l", evm.DefaultEVMGasLimit, "Specify the evm gas limit")
	cmd.Flags().StringP(bootstrappersFlag, "b", "", "Comma-separated multiaddresses of p2p peers to connect to")
	cmd.Flags().StringP(p2pIdentityFlag, "p", "", "Ed25519 private key in hex format (without 0x) for the p2p peer identity. Use the generate command to generate a new one")
	cmd.Flags().StringP(p2pListenAddressFlag, "q", "/ip4/127.0.0.1/tcp/30000", "MultiAddr for the p2p peer to listen on")

	return cmd
}

type Config struct {
	evmChainID                       uint64
	evmRPC, celesGRPC, tendermintRPC string
	evmPrivateKey                    *ecdsa.PrivateKey
	contractAddr                     ethcmn.Address
	evmGasLimit                      uint64
	bootstrappers, p2pListenAddr     string
	p2pIdentity                      crypto.PrivKey
}

func parseRelayerFlags(cmd *cobra.Command) (Config, error) {
	rawPrivateKey, err := cmd.Flags().GetString(privateKeyFlag)
	if err != nil {
		return Config{}, err
	}
	if rawPrivateKey == "" {
		return Config{}, errors.New("private key flag required")
	}
	ethPrivKey, err := ethcrypto.HexToECDSA(rawPrivateKey)
	if err != nil {
		return Config{}, fmt.Errorf("failed to hex-decode Ethereum ECDSA Private Key: %w", err)
	}
	evmChainID, err := cmd.Flags().GetUint64(evmChainIDFlag)
	if err != nil {
		return Config{}, err
	}
	tendermintRPC, err := cmd.Flags().GetString(tendermintRPCFlag)
	if err != nil {
		return Config{}, err
	}
	celesGRPC, err := cmd.Flags().GetString(celesGRPCFlag)
	if err != nil {
		return Config{}, err
	}
	contractAddr, err := cmd.Flags().GetString(contractAddressFlag)
	if err != nil {
		return Config{}, err
	}
	if contractAddr == "" {
		return Config{}, fmt.Errorf("contract address flag is required: %s", contractAddressFlag)
	}
	if !ethcmn.IsHexAddress(contractAddr) {
		return Config{}, fmt.Errorf("valid contract address flag is required: %s", contractAddressFlag)
	}
	address := ethcmn.HexToAddress(contractAddr)
	ethRPC, err := cmd.Flags().GetString(evmRPCFlag)
	if err != nil {
		return Config{}, err
	}
	evmGasLimit, err := cmd.Flags().GetUint64(evmGasLimitFlag)
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
		evmPrivateKey: ethPrivKey,
		evmChainID:    evmChainID,
		celesGRPC:     celesGRPC,
		tendermintRPC: tendermintRPC,
		contractAddr:  address,
		evmRPC:        ethRPC,
		evmGasLimit:   evmGasLimit,
		bootstrappers: bootstrappers,
		p2pListenAddr: p2pListenAddress,
		p2pIdentity:   identity,
	}, nil
}
