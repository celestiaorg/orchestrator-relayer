package query

import (
	"fmt"
	"strings"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/base"

	"github.com/spf13/cobra"
)

const (
	FlagP2PNode    = "p2p-node"
	FlagOutputFile = "output-file"
)

func addFlags(cmd *cobra.Command) *cobra.Command {
	base.AddCoreGRPCFlag(cmd)
	base.AddCoreRPCFlag(cmd)
	cmd.Flags().String(FlagP2PNode, "", "P2P target node multiaddress (eg. /ip4/127.0.0.1/tcp/30000/p2p/12D3KooWBSMasWzRSRKXREhediFUwABNZwzJbkZcYz5rYr9Zdmfn)")
	cmd.Flags().String(FlagOutputFile, "", "Path to an output file path if the results need to be written to a json file. Leaving it as empty will result in printing the result to stdout")
	base.AddGRPCInsecureFlag(cmd)

	return cmd
}

type Config struct {
	coreGRPC, coreRPC string
	targetNode        string
	outputFile        string
	grpcInsecure      bool
}

func parseFlags(cmd *cobra.Command) (Config, error) {
	coreRPC, err := cmd.Flags().GetString(base.FlagCoreRPC)
	if err != nil {
		return Config{}, err
	}
	if !strings.HasPrefix(coreRPC, "tcp://") {
		coreRPC = fmt.Sprintf("tcp://%s", coreRPC)
	}
	coreGRPC, err := cmd.Flags().GetString(base.FlagCoreGRPC)
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
	grpcInsecure, err := cmd.Flags().GetBool(base.FlagGRPCInsecure)
	if err != nil {
		return Config{}, err
	}
	return Config{
		coreGRPC:     coreGRPC,
		coreRPC:      coreRPC,
		targetNode:   targetNode,
		outputFile:   outputFile,
		grpcInsecure: grpcInsecure,
	}, nil
}
