package orchestrator

import (
	"context"

	"github.com/celestiaorg/orchestrator-relayer/p2p"

	"github.com/celestiaorg/orchestrator-relayer/types"
)

type Broadcaster struct {
	BlobstreamDHT *p2p.BlobstreamDHT
}

func NewBroadcaster(blobStreamDHT *p2p.BlobstreamDHT) *Broadcaster {
	return &Broadcaster{BlobstreamDHT: blobStreamDHT}
}

func (b Broadcaster) ProvideDataCommitmentConfirm(ctx context.Context, nonce uint64, confirm types.DataCommitmentConfirm, dataRootTupleRoot string) error {
	if len(b.BlobstreamDHT.RoutingTable().ListPeers()) == 0 {
		return ErrEmptyPeersTable
	}
	return b.BlobstreamDHT.PutDataCommitmentConfirm(ctx, p2p.GetDataCommitmentConfirmKey(nonce, confirm.EthAddress, dataRootTupleRoot), confirm)
}

func (b Broadcaster) ProvideValsetConfirm(ctx context.Context, nonce uint64, confirm types.ValsetConfirm, signBytes string) error {
	if len(b.BlobstreamDHT.RoutingTable().ListPeers()) == 0 {
		return ErrEmptyPeersTable
	}
	return b.BlobstreamDHT.PutValsetConfirm(ctx, p2p.GetValsetConfirmKey(nonce, confirm.EthAddress, signBytes), confirm)
}

func (b Broadcaster) ProvideLatestValset(ctx context.Context, latestValset types.LatestValset) error {
	if len(b.BlobstreamDHT.RoutingTable().ListPeers()) == 0 {
		return ErrEmptyPeersTable
	}
	return b.BlobstreamDHT.PutLatestValset(ctx, latestValset)
}
