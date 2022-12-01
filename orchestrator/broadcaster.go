package orchestrator

import (
	"context"

	"github.com/celestiaorg/orchestrator-relayer/types"
)

type BroadcasterI interface {
	// BroadcastConfirm broadcasts an attestation confirm to the P2P network.
	BroadcastConfirm(ctx context.Context, confirm types.AttestationConfirm) (string, error)
}

// Note: broadcaster implementation will be done after defining the P2P interfaces.
