package keys

import (
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys/evm"
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys/p2p"
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	keysCmd := &cobra.Command{
		Use:          "keys",
		Short:        "QGB keys manager",
		SilenceUsage: true,
	}

	keysCmd.AddCommand(
		evm.Root(),
		p2p.Root(),
	)

	keysCmd.SetHelpCommand(&cobra.Command{})

	return keysCmd
}
