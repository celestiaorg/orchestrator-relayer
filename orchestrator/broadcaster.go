package orchestrator

import (
	"context"

	"github.com/celestiaorg/orchestrator-relayer/p2p"

	"github.com/celestiaorg/orchestrator-relayer/types"
)

type Broadcaster struct {
	QgbDHT *p2p.QgbDHT
}

func NewBroadcaster(qgbDHT *p2p.QgbDHT) *Broadcaster {
	return &Broadcaster{QgbDHT: qgbDHT}
}

func (b Broadcaster) BroadcastDataCommitmentConfirm(ctx context.Context, nonce uint64, confirm types.DataCommitmentConfirm) error {
	if len(b.QgbDHT.RoutingTable().ListPeers()) == 0 {
		return ErrEmptyPeersTable
	}
	return b.QgbDHT.PutDataCommitmentConfirm(ctx, p2p.GetDataCommitmentConfirmKey(nonce, confirm.EthAddress), confirm)
}

func (b Broadcaster) BroadcastValsetConfirm(ctx context.Context, nonce uint64, confirm types.ValsetConfirm) error {
	if len(b.QgbDHT.RoutingTable().ListPeers()) == 0 {
		return ErrEmptyPeersTable
	}
	return b.QgbDHT.PutValsetConfirm(ctx, p2p.GetValsetConfirmKey(nonce, confirm.EthAddress), confirm)
}
