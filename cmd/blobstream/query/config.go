package query

import (
	"fmt"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/relayer"
	"github.com/spf13/cobra"
)

const (
	FlagP2PNode    = "p2p-node"
	FlagOutputFile = "output-file"
)

func addFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().String(relayer.FlagCoreGRPCHost, "localhost", "Specify the grpc address host")
	cmd.Flags().Uint(relayer.FlagCoreGRPCPort, 9090, "Specify the grpc address port")
	cmd.Flags().String(relayer.FlagCoreRPCHost, "localhost", "Specify the rest rpc address host")
	cmd.Flags().Uint(relayer.FlagCoreRPCPort, 26657, "Specify the rest rpc address")
	cmd.Flags().String(FlagP2PNode, "", "P2P target node multiaddress (eg. /ip4/127.0.0.1/tcp/30000/p2p/12D3KooWBSMasWzRSRKXREhediFUwABNZwzJbkZcYz5rYr9Zdmfn)")
	cmd.Flags().String(FlagOutputFile, "", "Path to an output file path if the results need to be written to a json file. Leaving it as empty will result in printing the result to stdout")

	return cmd
}

type Config struct {
	coreGRPC, coreRPC string
	targetNode        string
	outputFile        string
}

func parseFlags(cmd *cobra.Command) (Config, error) {
	coreRPCHost, err := cmd.Flags().GetString(relayer.FlagCoreRPCHost)
	if err != nil {
		return Config{}, err
	}
	coreRPCPort, err := cmd.Flags().GetUint(relayer.FlagCoreRPCPort)
	if err != nil {
		return Config{}, err
	}
	coreGRPCHost, err := cmd.Flags().GetString(relayer.FlagCoreGRPCHost)
	if err != nil {
		return Config{}, err
	}
	coreGRPCPort, err := cmd.Flags().GetUint(relayer.FlagCoreGRPCPort)
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
		coreGRPC:   fmt.Sprintf("%s:%d", coreGRPCHost, coreGRPCPort),
		coreRPC:    fmt.Sprintf("tcp://%s:%d", coreRPCHost, coreRPCPort),
		targetNode: targetNode,
		outputFile: outputFile,
	}, nil
}
