package orchestrator

import (
	"encoding/hex"
	"errors"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/base"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/cobra"
)

const (
	celestiaChainIDFlag  = "celes-chain-id"
	celestiaGRPCFlag     = "celes-grpc"
	evmAccAddressFlag    = "evm-address"
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
		evmAccAddressFlag,
		"d",
		"",
		"Specify the EVM account address to use for signing (Note: the private key should be in the keystore)",
	)
	cmd.Flags().StringP(bootstrappersFlag, "b", "", "Comma-separated multiaddresses of p2p peers to connect to")
	cmd.Flags().StringP(p2pIdentityFlag, "p", "", "Ed25519 private key in hex format (without 0x) for the p2p peer identity. Use the generate command to generate a new one")
	cmd.Flags().StringP(p2pListenAddressFlag, "q", "/ip4/0.0.0.0/tcp/30000", "MultiAddr for the p2p peer to listen on")
	cmd.Flags().String(base.FlagHome, "", "The qgb orchestrator home directory")
	cmd.Flags().String(base.FlagEVMPassphrase, "", "the evm account passphrase (if not specified as a flag, it will be asked interactively)")

	return cmd
}

type StartConfig struct {
	*base.Config
	celestiaChainID, celesGRPC, tendermintRPC string
	evmAccAddress                             string
	bootstrappers, p2pListenAddr              string
	p2pIdentity                               crypto.PrivKey
}

func parseOrchestratorFlags(cmd *cobra.Command) (StartConfig, error) {
	evmAccAddr, err := cmd.Flags().GetString(evmAccAddressFlag)
	if err != nil {
		return StartConfig{}, err
	}
	if evmAccAddr == "" {
		return StartConfig{}, errors.New("the evm account address should be specified")
	}
	chainID, err := cmd.Flags().GetString(celestiaChainIDFlag)
	if err != nil {
		return StartConfig{}, err
	}
	tendermintRPC, err := cmd.Flags().GetString(tendermintRPCFlag)
	if err != nil {
		return StartConfig{}, err
	}
	celesGRPC, err := cmd.Flags().GetString(celestiaGRPCFlag)
	if err != nil {
		return StartConfig{}, err
	}
	bootstrappers, err := cmd.Flags().GetString(bootstrappersFlag)
	if err != nil {
		return StartConfig{}, err
	}
	p2pListenAddress, err := cmd.Flags().GetString(p2pListenAddressFlag)
	if err != nil {
		return StartConfig{}, err
	}
	hexIdentity, err := cmd.Flags().GetString(p2pIdentityFlag)
	if err != nil {
		return StartConfig{}, err
	}
	bIdentity, err := hex.DecodeString(hexIdentity)
	if err != nil {
		return StartConfig{}, err
	}
	identity, err := crypto.UnmarshalEd25519PrivateKey(bIdentity)
	if err != nil {
		return StartConfig{}, err
	}
	homeDir, err := cmd.Flags().GetString(base.FlagHome)
	if err != nil {
		return StartConfig{}, err
	}
	if homeDir == "" {
		var err error
		homeDir, err = base.DefaultServicePath("orchestrator")
		if err != nil {
			return StartConfig{}, err
		}
	}
	passphrase, err := cmd.Flags().GetString(base.FlagEVMPassphrase)
	if err != nil {
		return StartConfig{}, err
	}

	return StartConfig{
		evmAccAddress:   evmAccAddr,
		celestiaChainID: chainID,
		celesGRPC:       celesGRPC,
		tendermintRPC:   tendermintRPC,
		bootstrappers:   bootstrappers,
		p2pIdentity:     identity,
		p2pListenAddr:   p2pListenAddress,
		Config: &base.Config{
			Home:          homeDir,
			EVMPassphrase: passphrase,
		},
	}, nil
}

func addInitFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(base.FlagHome, "", "The qgb orchestrator home directory")
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
		homeDir, err = base.DefaultServicePath("orchestrator")
		if err != nil {
			return InitConfig{}, err
		}
	}

	return InitConfig{
		home: homeDir,
	}, nil
}
