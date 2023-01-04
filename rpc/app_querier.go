package rpc

import (
	"context"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
)

// AppQuerierI queries the application for attestations and unbonding periods.
type AppQuerierI interface {
	// QueryAttestationByNonce query an attestation by nonce from the state machine.
	QueryAttestationByNonce(ctx context.Context, nonce uint64) (celestiatypes.AttestationRequestI, error)

	// QueryLatestAttestationNonce query the latest attestation nonce from the state machine.
	QueryLatestAttestationNonce(ctx context.Context) (uint64, error)

	// QueryDataCommitmentByNonce query a data commitment by its nonce.
	QueryDataCommitmentByNonce(ctx context.Context, nonce uint64) (*celestiatypes.DataCommitment, error)

	// QueryValsetByNonce query a valset by nonce.
	QueryValsetByNonce(ctx context.Context, nonce uint64) (*celestiatypes.Valset, error)

	// QueryLatestValset query the latest recorded valset in the state machine.
	QueryLatestValset(ctx context.Context) (*celestiatypes.Valset, error)

	// QueryLastValsetBeforeNonce query the last valset before a certain nonce from the state machine.
	// this will be needed when signing to know the validator set at that particular nonce.
	QueryLastValsetBeforeNonce(ctx context.Context, nonce uint64) (*celestiatypes.Valset, error)

	// QueryLastUnbondingHeight query the last unbonding height from state machine.
	QueryLastUnbondingHeight(ctx context.Context) (int64, error)

	// QueryLastUnbondingAttestationNonce query the attestation nonce that corresponds to the
	// latest attestation nonce.
	QueryLastUnbondingAttestationNonce(ctx context.Context) (uint64, error)
}
