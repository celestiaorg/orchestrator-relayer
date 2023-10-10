package p2p

import (
	"github.com/celestiaorg/orchestrator-relayer/cmd/bstream/base"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

func keysConfigFlags(cmd *cobra.Command, service string) *cobra.Command {
	homeDir, err := base.DefaultServicePath(service)
	if err != nil {
		panic(err)
	}
	cmd.Flags().String(base.FlagHome, homeDir, "The BlobStream p2p keys home directory")
	return cmd
}

type KeysConfig struct {
	home string
}

func parseKeysConfigFlags(cmd *cobra.Command, serviceName string) (KeysConfig, error) {
	homeDir, err := cmd.Flags().GetString(flags.FlagHome)
	if err != nil {
		return KeysConfig{}, err
	}
	if homeDir == "" {
		var err error
		homeDir, err = base.DefaultServicePath(serviceName)
		if err != nil {
			return KeysConfig{}, err
		}
	}
	return KeysConfig{
		home: homeDir,
	}, nil
}
