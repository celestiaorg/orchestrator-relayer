package main

import (
	"context"
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/deploy"
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/orchestrator"
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/relayer"
	"os"

	"github.com/spf13/cobra"
)

func main() {
	rootCmd := NewRootCmd()
	if err := rootCmd.ExecuteContext(context.Background()); err != nil {
		os.Exit(1)
	}
}

// NewRootCmd creates a new root command for the QGB CLI. It is called once in the
// main function.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "qgb",
		Short:        "The Quantum-Gravity-Bridge CLI",
		SilenceUsage: true,
	}

	rootCmd.AddCommand(
		orchestrator.Command(),
		relayer.Command(),
		deploy.Command(),
	)

	rootCmd.SetHelpCommand(&cobra.Command{})

	return rootCmd
}
