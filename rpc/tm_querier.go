package rpc

import (
	"context"

	"github.com/tendermint/tendermint/libs/bytes"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

// TmQuerierI queries tendermint for commitments and events.
type TmQuerierI interface {
	Stop() error
	// QueryCommitment queries tendermint for a commitment for the set of blocks
	// defined by `beginBlock` and `endBlock`.
	QueryCommitment(ctx context.Context, beginBlock uint64, endBlock uint64) (bytes.HexBytes, error)

	// SubscribeEvents subscribe to the events named `eventName`.
	SubscribeEvents(ctx context.Context, subscriptionName string, eventName string) (<-chan coretypes.ResultEvent, error)
}
