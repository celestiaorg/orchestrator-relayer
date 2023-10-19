package root

import (
	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/bootstrapper"
	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/generate"
	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/query"

	"github.com/celestiaorg/celestia-app/x/qgb/client"
	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/deploy"
	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/orchestrator"
	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/relayer"

	"github.com/spf13/cobra"
)

// Cmd creates a new root command for the Blobstream CLI. It is called once in the
// main function.
func Cmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "blobstream",
		Short:        "The Blobstream CLI",
		SilenceUsage: true,
	}

	rootCmd.AddCommand(
		orchestrator.Command(),
		relayer.Command(),
		deploy.Command(),
		client.VerifyCmd(),
		generate.Command(),
		query.Command(),
		bootstrapper.Command(),
	)

	rootCmd.SetHelpCommand(&cobra.Command{})

	return rootCmd
}
