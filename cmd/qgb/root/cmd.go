package root

import (
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/generate"

	"github.com/celestiaorg/celestia-app/x/qgb/client"
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/deploy"
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/orchestrator"
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/relayer"

	"github.com/spf13/cobra"
)

// Cmd creates a new root command for the QGB CLI. It is called once in the
// main function.
func Cmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "qgb",
		Short:        "The Quantum-Gravity-Bridge CLI",
		SilenceUsage: true,
	}

	rootCmd.AddCommand(
		orchestrator.Command(),
		relayer.Command(),
		deploy.Command(),
		client.VerifyCmd(),
		generate.Command(),
	)

	rootCmd.SetHelpCommand(&cobra.Command{})

	return rootCmd
}
