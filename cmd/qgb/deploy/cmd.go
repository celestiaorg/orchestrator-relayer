package deploy

import (
	"context"
	"strconv"

	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	"github.com/spf13/cobra"
)

func DeployCmd() *cobra.Command {
	command := &cobra.Command{
		Use:   "deploy <flags>",
		Short: "Deploys the QGB contract and initializes it using the provided Celestia chain",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := parseDeployFlags(cmd)
			if err != nil {
				return err
			}

			// TODO add implementation

			return nil
		},
	}
	return addDeployFlags(command)
}

// nolint
func getStartingValset(ctx context.Context, querier rpc.AppQuerierI, startingNonce string) (*types.Valset, error) {
	switch startingNonce {
	case "latest":
		return querier.QueryLatestValset(ctx)
	case "earliest":
		// TODO make the first nonce 1 a const
		att, err := querier.QueryAttestationByNonce(ctx, 1)
		if err != nil {
			return nil, err
		}
		vs, ok := att.(*types.Valset)
		if !ok {
			return nil, ErrUnmarshallValset
		}
		return vs, nil
	default:
		nonce, err := strconv.ParseUint(startingNonce, 10, 0)
		if err != nil {
			return nil, err
		}
		attestation, err := querier.QueryAttestationByNonce(ctx, nonce)
		if err != nil {
			return nil, err
		}
		if attestation == nil {
			return nil, types.ErrNilAttestation
		}
		if attestation.Type() == types.ValsetRequestType {
			value, ok := attestation.(*types.Valset)
			if !ok {
				return nil, ErrUnmarshallValset
			}
			return value, nil
		}
		return querier.QueryLastValsetBeforeNonce(ctx, nonce)
	}
}
