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
	cmd.Flags().String(base.FlagHome, "", "The Blobstream orchestrator|relayer home directory. If this flag is not set, it will try the orchestrator's default home directory, then the relayer's default home directory to get the necessary configuration")
	return cmd
}

type Config struct {
	coreGRPC, coreRPC string
	targetNode        string
	outputFile        string
	grpcInsecure      bool
}

func NewPartialConfig(coreGRPC, coreRPC, targetNode string, grpcInsecure bool) *Config {
	return &Config{
		coreGRPC:     coreGRPC,
		coreRPC:      coreRPC,
		targetNode:   targetNode,
		grpcInsecure: grpcInsecure,
	}
}

func DefaultConfig() *Config {
	return &Config{
		coreGRPC:     "localhost:9090",
		coreRPC:      "tcp://localhost:26657",
		targetNode:   "",
		outputFile:   "",
		grpcInsecure: true,
	}
}

func parseFlags(cmd *cobra.Command, startConf *Config) (Config, error) {
	coreRPC, changed, err := base.GetCoreRPCFlag(cmd)
	if err != nil {
		return Config{}, err
	}
	if changed {
		if !strings.HasPrefix(coreRPC, "tcp://") {
			coreRPC = fmt.Sprintf("tcp://%s", coreRPC)
		}
		startConf.coreRPC = coreRPC
	}

	coreGRPC, changed, err := base.GetCoreGRPCFlag(cmd)
	if err != nil {
		return Config{}, err
	}
	if changed {
		startConf.coreGRPC = coreGRPC
	}

	targetNode, changed, err := getP2PNodeFlag(cmd)
	if err != nil {
		return Config{}, err
	}
	if changed {
		startConf.targetNode = targetNode
	}

	outputFile, err := cmd.Flags().GetString(FlagOutputFile)
	if err != nil {
		return Config{}, err
	}
	startConf.outputFile = outputFile

	grpcInsecure, changed, err := base.GetGRPCInsecureFlag(cmd)
	if err != nil {
		return Config{}, err
	}
	if changed {
		startConf.grpcInsecure = grpcInsecure
	}

	return *startConf, nil
}

func getP2PNodeFlag(cmd *cobra.Command) (string, bool, error) {
	changed := cmd.Flags().Changed(FlagP2PNode)
	val, err := cmd.Flags().GetString(FlagP2PNode)
	if err != nil {
		return "", changed, err
	}
	return val, changed, nil
}
