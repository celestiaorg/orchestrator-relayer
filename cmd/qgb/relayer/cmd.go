package relayer

import (
	"github.com/spf13/cobra"
)

func RelayerCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "relayer <flags>",
		Short: "Runs the QGB relayer to submit attestations to the target EVM chain",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := parseRelayerFlags(cmd)
			if err != nil {
				return err
			}

			// TODO add implementation

			return nil
		},
	}
	return addRelayerFlags(command)
}
