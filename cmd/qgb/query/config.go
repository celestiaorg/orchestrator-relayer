package query

import (
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/relayer"
	"github.com/spf13/cobra"
)

func addQueryFlags(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().StringP(relayer.FlagCelesGRPC, "c", "localhost:9090", "Specify the grpc address")
	cmd.Flags().StringP(relayer.FlagTendermintRPC, "t", "http://localhost:26657", "Specify the rest rpc address")
	cmd.Flags().StringP(relayer.FlagP2PNode, "n", "", "P2P target node multiaddress (eg. /ip4/127.0.0.1/tcp/30000/p2p/12D3KooWBSMasWzRSRKXREhediFUwABNZwzJbkZcYz5rYr9Zdmfn)")

	return cmd
}

type QueryConfig struct {
	celesGRPC, tendermintRPC string
	targetNode               string
}

func parseQueryFlags(cmd *cobra.Command) (QueryConfig, error) {
	tendermintRPC, err := cmd.Flags().GetString(relayer.FlagTendermintRPC)
	if err != nil {
		return QueryConfig{}, err
	}
	celesGRPC, err := cmd.Flags().GetString(relayer.FlagCelesGRPC)
	if err != nil {
		return QueryConfig{}, err
	}
	targetNode, err := cmd.Flags().GetString(relayer.FlagP2PNode)
	if err != nil {
		return QueryConfig{}, err
	}

	return QueryConfig{
		celesGRPC:     celesGRPC,
		tendermintRPC: tendermintRPC,
		targetNode:    targetNode,
	}, nil
}
