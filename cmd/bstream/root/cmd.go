package root

import (
	"github.com/celestiaorg/orchestrator-relayer/cmd/bstream/bootstrapper"
	"github.com/celestiaorg/orchestrator-relayer/cmd/bstream/generate"
	"github.com/celestiaorg/orchestrator-relayer/cmd/bstream/query"

	"github.com/celestiaorg/celestia-app/x/blobstream/client"
	"github.com/celestiaorg/orchestrator-relayer/cmd/bstream/deploy"
	"github.com/celestiaorg/orchestrator-relayer/cmd/bstream/orchestrator"
	"github.com/celestiaorg/orchestrator-relayer/cmd/bstream/relayer"

	"github.com/spf13/cobra"
)

// Cmd creates a new root command for the BlobStream CLI. It is called once in the
// main function.
func Cmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:          "bstream",
		Short:        "The BlobStream CLI",
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
