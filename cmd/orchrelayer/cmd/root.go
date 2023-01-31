package cmd

import (
	"github.com/celestiaorg/orchestrator-relayer/x/qgb/orchestrator"
	"github.com/spf13/cobra"
)

const EnvPrefix = "CELESTIA"

// NewRootCmd creates a new root command for celestia-appd. It is called once in the
// main function.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "celestia-appd",
		Short: "Start celestia app",
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			return nil
		},
		SilenceUsage: true,
	}

	initRootCmd(rootCmd)

	return rootCmd
}

func initRootCmd(rootCmd *cobra.Command) {
	// add qgb related commands
	rootCmd.AddCommand(
		orchestrator.DeployCmd(),
		orchestrator.OrchRelayerCmd(),
	)
}
