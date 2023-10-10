package bootstrapper

import (
	"github.com/celestiaorg/orchestrator-relayer/cmd/bstream/base"
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
	cmd.Flags().String(base.FlagHome, homeDir, "The BlobStream bootstrappers home directory")
	base.AddP2PNicknameFlag(cmd)
	base.AddP2PListenAddressFlag(cmd)
	base.AddBootstrappersFlag(cmd)
	return cmd
}

type StartConfig struct {
	home                       string
	p2pListenAddr, p2pNickname string
	bootstrappers              string
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

	return StartConfig{
		p2pNickname:   p2pNickname,
		p2pListenAddr: p2pListenAddress,
		home:          homeDir,
		bootstrappers: bootstrappers,
	}, nil
}

func addInitFlags(cmd *cobra.Command) *cobra.Command {
	homeDir, err := base.DefaultServicePath(ServiceNameBootstrapper)
	if err != nil {
		panic(err)
	}
	cmd.Flags().String(base.FlagHome, homeDir, "The BlobStream bootstrappers home directory")
	return cmd
}

type InitConfig struct {
	home string
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

	return InitConfig{
		home: homeDir,
	}, nil
}
