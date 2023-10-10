package orchestrator

import (
	"context"

	"github.com/celestiaorg/orchestrator-relayer/p2p"

	"github.com/celestiaorg/orchestrator-relayer/types"
)

type Broadcaster struct {
	BlobStreamDHT *p2p.BlobStreamDHT
}

func NewBroadcaster(blobStreamDHT *p2p.BlobStreamDHT) *Broadcaster {
	return &Broadcaster{BlobStreamDHT: blobStreamDHT}
}

func (b Broadcaster) ProvideDataCommitmentConfirm(ctx context.Context, nonce uint64, confirm types.DataCommitmentConfirm, dataRootTupleRoot string) error {
	if len(b.BlobStreamDHT.RoutingTable().ListPeers()) == 0 {
		return ErrEmptyPeersTable
	}
	return b.BlobStreamDHT.PutDataCommitmentConfirm(ctx, p2p.GetDataCommitmentConfirmKey(nonce, confirm.EthAddress, dataRootTupleRoot), confirm)
}

func (b Broadcaster) ProvideValsetConfirm(ctx context.Context, nonce uint64, confirm types.ValsetConfirm, signBytes string) error {
	if len(b.BlobStreamDHT.RoutingTable().ListPeers()) == 0 {
		return ErrEmptyPeersTable
	}
	return b.BlobStreamDHT.PutValsetConfirm(ctx, p2p.GetValsetConfirmKey(nonce, confirm.EthAddress, signBytes), confirm)
}
