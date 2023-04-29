package query

import (
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/relayer"
	"github.com/spf13/cobra"
)

const (
	FlagP2PNode    = "p2p-node"
	FlagOutputFile = "output-file"
)

func addFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(relayer.FlagCelesGRPC, "c", "localhost:9090", "Specify the grpc address")
	cmd.Flags().StringP(relayer.FlagTendermintRPC, "t", "http://localhost:26657", "Specify the rest rpc address")
	cmd.Flags().StringP(FlagP2PNode, "n", "", "P2P target node multiaddress (eg. /ip4/127.0.0.1/tcp/30000/p2p/12D3KooWBSMasWzRSRKXREhediFUwABNZwzJbkZcYz5rYr9Zdmfn)")
	cmd.Flags().StringP(FlagOutputFile, "o", "", "Path to an output file path if the results need to be written to a json file. Leaving it as empty will result in printing the result to stdout")

	return cmd
}

type Config struct {
	celesGRPC, tendermintRPC string
	targetNode               string
	outputFile               string
}

func parseFlags(cmd *cobra.Command) (Config, error) {
	tendermintRPC, err := cmd.Flags().GetString(relayer.FlagTendermintRPC)
	if err != nil {
		return Config{}, err
	}
	celesGRPC, err := cmd.Flags().GetString(relayer.FlagCelesGRPC)
	if err != nil {
		return Config{}, err
	}
	targetNode, err := cmd.Flags().GetString(FlagP2PNode)
	if err != nil {
		return Config{}, err
	}
	outputFile, err := cmd.Flags().GetString(FlagOutputFile)
	if err != nil {
		return Config{}, err
	}

	return Config{
		celesGRPC:     celesGRPC,
		tendermintRPC: tendermintRPC,
		targetNode:    targetNode,
		outputFile:    outputFile,
	}, nil
}
