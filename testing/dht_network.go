package testing

import (
	"context"
	"time"

	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/celestiaorg/orchestrator-relayer/p2p"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// DHTNetwork is a test DHT network that can be used for tests.
type DHTNetwork struct {
	Context context.Context
	Hosts   []host.Host
	Stores  []ds.Batching
	DHTs    []*p2p.QgbDHT
}

// NewDHTNetwork creates a new DHT test network running in-memory.
// The stores are in-memory stores.
// The hosts listen on real ports.
// The nodes are all connected to `hosts[0]` node.
// The `count` parameter specifies the number of nodes that the network will run.
// This function doesn't return any errors, and panics in case any unexpected happened.
func NewDHTNetwork(ctx context.Context, count int) *DHTNetwork {
	if count <= 1 {
		panic("can't create a test network with a negative nodes count or only 1 DHT node")
	}
	hosts := make([]host.Host, count)
	stores := make([]ds.Batching, count)
	dhts := make([]*p2p.QgbDHT, count)
	for i := 0; i < count; i++ {
		h, store, dht := NewTestDHT(ctx)
		hosts[i] = h
		stores[i] = store
		dhts[i] = dht
		if i != 0 {
			err := h.Connect(ctx, peer.AddrInfo{
				ID:    hosts[0].ID(),
				Addrs: hosts[0].Addrs(),
			})
			if err != nil {
				panic(err)
			}
		}
	}
	// to give time for the DHT to update its peer table
	err := WaitForPeerTableToUpdate(ctx, dhts, time.Minute)
	if err != nil {
		panic(err)
	}
	return &DHTNetwork{
		Context: ctx,
		Hosts:   hosts,
		Stores:  stores,
		DHTs:    dhts,
	}
}

// NewTestDHT creates a test DHT not connected to any peers.
func NewTestDHT(ctx context.Context) (host.Host, ds.Batching, *p2p.QgbDHT) {
	h, err := libp2p.New()
	if err != nil {
		panic(err)
	}
	dataStore := dssync.MutexWrap(ds.NewMapDatastore())
	dht, err := p2p.NewQgbDHT(ctx, h, dataStore, tmlog.NewNopLogger())
	if err != nil {
		panic(err)
	}
	return h, dataStore, dht
}

// WaitForPeerTableToUpdate waits for nodes to have updated their peers list
func WaitForPeerTableToUpdate(ctx context.Context, dhts []*p2p.QgbDHT, timeout time.Duration) error {
	withTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	ticker := time.NewTicker(time.Millisecond)
	for {
		select {
		case <-withTimeout.Done():
			return ErrTimeout
		case <-ticker.C:
			allPeersConnected := func() bool {
				for _, dht := range dhts {
					if len(dht.RoutingTable().ListPeers()) == 0 {
						return false
					}
				}
				return true
			}
			if allPeersConnected() {
				return nil
			}
		}
	}
}

// Stop tears down the test network and stops all the services.
// Panics if an error occurs.
func (tn DHTNetwork) Stop() {
	for i := range tn.DHTs {
		err := tn.DHTs[i].Close()
		if err != nil {
			panic(err)
		}
		err = tn.Stores[i].Close()
		if err != nil {
			panic(err)
		}
		err = tn.Hosts[i].Close()
		if err != nil {
			panic(err)
		}
	}
}
