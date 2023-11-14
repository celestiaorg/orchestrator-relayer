package p2p

import (
	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/base"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/cobra"
)

func keysConfigFlags(cmd *cobra.Command, service string) *cobra.Command {
	homeDir, err := base.DefaultServicePath(service)
	if err != nil {
		panic(err)
	}
	cmd.Flags().String(base.FlagHome, homeDir, "The Blobstream p2p keys home directory")
	base.AddLogLevelFlag(cmd)
	base.AddLogFormatFlag(cmd)
	return cmd
}

type KeysConfig struct {
	home      string
	logLevel  string
	logFormat string
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
	logLevel, _, err := base.GetLogLevelFlag(cmd)
	if err != nil {
		return KeysConfig{}, err
	}

	logFormat, _, err := base.GetLogFormatFlag(cmd)
	if err != nil {
		return KeysConfig{}, err
	}
	return KeysConfig{
		home:      homeDir,
		logFormat: logFormat,
		logLevel:  logLevel,
	}, nil
}
