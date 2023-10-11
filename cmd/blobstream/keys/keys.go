package keys

import (
	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/keys/evm"
	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/keys/p2p"
	"github.com/spf13/cobra"
)

func Command(serviceName string) *cobra.Command {
	keysCmd := &cobra.Command{
		Use:          "keys",
		Short:        "Blobstream keys manager",
		SilenceUsage: true,
	}

	keysCmd.AddCommand(
		evm.Root(serviceName),
		p2p.Root(serviceName),
	)

	keysCmd.SetHelpCommand(&cobra.Command{})

	return keysCmd
}
