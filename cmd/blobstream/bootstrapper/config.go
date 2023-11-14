package bootstrapper

import (
	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/base"
	"github.com/spf13/cobra"
)

const (
	ServiceNameBootstrapper = "bootstrapper"
)

func addStartFlags(cmd *cobra.Command) *cobra.Command {
	homeDir, err := base.DefaultServicePath(ServiceNameBootstrapper)
	if err != nil {
		panic(err)
	}
	cmd.Flags().String(base.FlagHome, homeDir, "The Blobstream bootstrappers home directory")
	base.AddP2PNicknameFlag(cmd)
	base.AddP2PListenAddressFlag(cmd)
	base.AddBootstrappersFlag(cmd)
	base.AddLogLevelFlag(cmd)
	base.AddLogFormatFlag(cmd)
	return cmd
}

type StartConfig struct {
	home                       string
	p2pListenAddr, p2pNickname string
	bootstrappers              string
	logLevel                   string
	logFormat                  string
}

func parseStartFlags(cmd *cobra.Command) (StartConfig, error) {
	p2pListenAddress, err := cmd.Flags().GetString(base.FlagP2PListenAddress)
	if err != nil {
		return StartConfig{}, err
	}
	p2pNickname, err := cmd.Flags().GetString(base.FlagP2PNickname)
	if err != nil {
		return StartConfig{}, err
	}
	homeDir, err := cmd.Flags().GetString(base.FlagHome)
	if err != nil {
		return StartConfig{}, err
	}
	if homeDir == "" {
		var err error
		homeDir, err = base.DefaultServicePath(ServiceNameBootstrapper)
		if err != nil {
			return StartConfig{}, err
		}
	}
	bootstrappers, err := cmd.Flags().GetString(base.FlagBootstrappers)
	if err != nil {
		return StartConfig{}, err
	}

	logLevel, _, err := base.GetLogLevelFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}

	logFormat, _, err := base.GetLogFormatFlag(cmd)
	if err != nil {
		return StartConfig{}, err
	}

	return StartConfig{
		p2pNickname:   p2pNickname,
		p2pListenAddr: p2pListenAddress,
		home:          homeDir,
		bootstrappers: bootstrappers,
		logFormat:     logFormat,
		logLevel:      logLevel,
	}, nil
}

func addInitFlags(cmd *cobra.Command) *cobra.Command {
	homeDir, err := base.DefaultServicePath(ServiceNameBootstrapper)
	if err != nil {
		panic(err)
	}
	cmd.Flags().String(base.FlagHome, homeDir, "The Blobstream bootstrappers home directory")
	base.AddLogLevelFlag(cmd)
	base.AddLogFormatFlag(cmd)
	return cmd
}

type InitConfig struct {
	home      string
	logLevel  string
	logFormat string
}

func parseInitFlags(cmd *cobra.Command) (InitConfig, error) {
	homeDir, err := cmd.Flags().GetString(base.FlagHome)
	if err != nil {
		return InitConfig{}, err
	}
	if homeDir == "" {
		var err error
		homeDir, err = base.DefaultServicePath(ServiceNameBootstrapper)
		if err != nil {
			return InitConfig{}, err
		}
	}
	logLevel, _, err := base.GetLogLevelFlag(cmd)
	if err != nil {
		return InitConfig{}, err
	}

	logFormat, _, err := base.GetLogFormatFlag(cmd)
	if err != nil {
		return InitConfig{}, err
	}
	return InitConfig{
		home:      homeDir,
		logFormat: logFormat,
		logLevel:  logLevel,
	}, nil
}
