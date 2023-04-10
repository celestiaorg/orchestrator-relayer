package relayer

import (
	"errors"
	"fmt"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/base"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/spf13/cobra"

	ethcmn "github.com/ethereum/go-ethereum/common"
)

const (
	evmAccAddressFlag    = "evm-address"
	evmChainIDFlag       = "evm-chain-id"
	celesGRPCFlag        = "celes-grpc"
	tendermintRPCFlag    = "celes-http-rpc"
	evmRPCFlag           = "evm-rpc"
	contractAddressFlag  = "contract-address"
	evmGasLimitFlag      = "evm-gas-limit"
	bootstrappersFlag    = "p2p-bootstrappers"
	p2pListenAddressFlag = "p2p-listen-addr"
	p2pNicknameFlag      = "p2p-nickname"
)

func addRelayerFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(evmAccAddressFlag, "d", "", "Specify the EVM account address to use for signing (Note: the private key should be in the keystore)")
	cmd.Flags().Uint64P(evmChainIDFlag, "z", 5, "Specify the evm chain id")
	cmd.Flags().StringP(celesGRPCFlag, "c", "localhost:9090", "Specify the grpc address")
	cmd.Flags().StringP(tendermintRPCFlag, "t", "http://localhost:26657", "Specify the rest rpc address")
	cmd.Flags().StringP(evmRPCFlag, "e", "http://localhost:8545", "Specify the ethereum rpc address")
	cmd.Flags().StringP(contractAddressFlag, "a", "", "Specify the contract at which the qgb is deployed")
	cmd.Flags().Uint64P(evmGasLimitFlag, "l", evm.DefaultEVMGasLimit, "Specify the evm gas limit")
	cmd.Flags().StringP(bootstrappersFlag, "b", "", "Comma-separated multiaddresses of p2p peers to connect to")
	cmd.Flags().StringP(p2pNicknameFlag, "p", "", "Nickname of the p2p private key to use (if not provided, an existing one from the p2p store or a newly generated one will be used)")
	cmd.Flags().StringP(p2pListenAddressFlag, "q", "/ip4/127.0.0.1/tcp/30000", "MultiAddr for the p2p peer to listen on")
	cmd.Flags().String(base.FlagHome, "", "The qgb relayer home directory")
	cmd.Flags().String(base.FlagPassphrase, "", "the account passphrase (if not specified as a flag, it will be asked interactively)")

	return cmd
}

type Config struct {
	*base.Config
	evmChainID                       uint64
	evmRPC, celesGRPC, tendermintRPC string
	evmAccAddress                    string
	contractAddr                     ethcmn.Address
	evmGasLimit                      uint64
	bootstrappers, p2pListenAddr     string
	p2pNickname                      string
}

func parseRelayerFlags(cmd *cobra.Command) (Config, error) {
	evmAccAddr, err := cmd.Flags().GetString(evmAccAddressFlag)
	if err != nil {
		return Config{}, err
	}
	if evmAccAddr == "" {
		return Config{}, errors.New("the evm account address should be specified")
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
	p2pNickname, err := cmd.Flags().GetString(p2pNicknameFlag)
	if err != nil {
		return Config{}, err
	}
	homeDir, err := cmd.Flags().GetString(base.FlagHome)
	if err != nil {
		return Config{}, err
	}
	if homeDir == "" {
		var err error
		homeDir, err = base.DefaultServicePath("relayer")
		if err != nil {
			return Config{}, err
		}
	}
	passphrase, err := cmd.Flags().GetString(base.FlagPassphrase)
	if err != nil {
		return Config{}, err
	}

	return Config{
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
			Home:       homeDir,
			Passphrase: passphrase,
		},
	}, nil
}
