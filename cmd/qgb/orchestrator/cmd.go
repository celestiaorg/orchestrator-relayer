package orchestrator

import (
	"github.com/spf13/cobra"
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:     "orchestrator <flags>",
		Aliases: []string{"orch"},
		Short:   "Runs the QGB orchestrator to sign attestations",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := parseOrchestratorFlags(cmd)
			if err != nil {
				return err
			}

			// TODO add implementation

			return nil
		},
	}
	return addOrchestratorFlags(command)
}
