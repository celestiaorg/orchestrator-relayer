package e2e

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/stretchr/testify/assert"
)

const TRUE = "true"

func HandleNetworkError(t *testing.T, network *BlobStreamNetwork, err error, expectError bool) {
	if expectError && err == nil {
		network.PrintLogs()
		assert.Error(t, err)
		t.FailNow()
	} else if !expectError && err != nil {
		network.PrintLogs()
		assert.NoError(t, err)
		if errors.Is(err, ErrNetworkStopped) {
			// if some other error occurred, we notify.
			network.toStopChan <- struct{}{}
		}
		t.FailNow()
	}
}

func ConnectToDHT(ctx context.Context, h host.Host, dht *p2p.BlobStreamDHT, target peer.AddrInfo) error {
	timeout := time.NewTimer(time.Minute)
	for {
		select {
		case <-timeout.C:
			return errors.New("couldn't connect to dht")
		default:
			if len(dht.RoutingTable().ListPeers()) == 0 {
				if h.Connect(ctx, target) == nil {
					return nil
				}
				time.Sleep(time.Second)
			} else {
				return nil
			}
		}
	}
}
