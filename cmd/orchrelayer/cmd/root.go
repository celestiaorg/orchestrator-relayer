package cmd

import (
	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/orchestrator-relayer/x/qgb/orchestrator"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
)

const EnvPrefix = "CELESTIA"

// NewRootCmd creates a new root command for celestia-appd. It is called once in the
// main function.
func NewRootCmd() *cobra.Command {
	cfg := sdk.GetConfig()
	cfg.SetBech32PrefixForAccount(app.Bech32PrefixAccAddr, app.Bech32PrefixAccPub)
	cfg.SetBech32PrefixForValidator(app.Bech32PrefixValAddr, app.Bech32PrefixValPub)
	cfg.SetBech32PrefixForConsensusNode(app.Bech32PrefixConsAddr, app.Bech32PrefixConsPub)
	cfg.Seal()

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
