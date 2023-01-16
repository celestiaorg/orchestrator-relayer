package p2p

import (
	"context"
	"time"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/types"
)

// QuerierI queries the P2P network for confirms.
// Note: for now, we will be querying the P2P network directly without storing anything in the store.
// this would make it easier to have a simple working version. Then, we can iterate from there on.
type QuerierI interface {
	// QueryTwoThirdsDataCommitmentConfirms queries the p2p network for data commitments signed by more
	// than 2/3s of the last validator set.
	// It takes a timeout to wait for validators to sign and broadcast their signatures.
	// Should return an empty slice and no error if it couldn't find enough signatures.
	QueryTwoThirdsDataCommitmentConfirms(
		ctx context.Context,
		timeout time.Duration,
		dc celestiatypes.DataCommitment,
	) ([]types.DataCommitmentConfirm, error)

	// QueryTwoThirdsValsetConfirms queries the p2p network for valsets signed by more
	// than 2/3s of the last validator set.
	// It takes a timeout to wait for validators to sign and broadcast their signatures.
	// Should return an empty slice and no error if it couldn't find enough signatures.
	QueryTwoThirdsValsetConfirms(
		ctx context.Context,
		timeout time.Duration,
		valset celestiatypes.Valset,
	) ([]types.ValsetConfirm, error)

	// QueryValsetConfirmByOrchestratorAddress get the valset confirm having nonce `nonce`
	// and signed by the orchestrator whose address is `address`.
	QueryValsetConfirmByOrchestratorAddress(
		ctx context.Context,
		nonce uint64,
		address string,
	) (*types.ValsetConfirm, error)

	// QueryDataCommitmentConfirmByOrchestratorAddress get the data commitment confirm having nonce `nonce`
	// and signed by the orchestrator whose address is `address`.
	QueryDataCommitmentConfirmByOrchestratorAddress(
		ctx context.Context,
		nonce uint64,
		address string,
	) (*types.DataCommitmentConfirm, error)

	// QueryDataCommitmentConfirms get all the data commitment confirms in store for a certain nonce.
	QueryDataCommitmentConfirms(
		ctx context.Context,
		nonce uint64,
	) ([]types.DataCommitmentConfirm, error)

	// QueryValsetConfirms get all the valset confirms in store for a certain nonce.
	QueryValsetConfirms(
		ctx context.Context,
		nonce uint64,
	) ([]types.ValsetConfirm, error)
}
