package testing

import (
	"context"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/p2p"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
)

// TestDHTNetwork is a test DHT network that can be used for tests.
type TestDHTNetwork struct {
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
func NewDHTNetwork(ctx context.Context, count int) *TestDHTNetwork {
	if count <= 0 {
		panic("can't create a test network with a negative nodes count")
	}
	hosts := make([]host.Host, count)
	stores := make([]ds.Batching, count)
	dhts := make([]*p2p.QgbDHT, count)
	for i := 0; i < count; i++ {
		h, err := libp2p.New()
		if err != nil {
			panic(err)
		}
		hosts[i] = h
		store := dssync.MutexWrap(ds.NewMapDatastore())
		stores[i] = store
		dht, err := p2p.NewQgbDHT(ctx, h, store)
		if err != nil {
			panic(err)
		}
		dhts[i] = dht
		if i != 0 {
			err = h.Connect(ctx, peer.AddrInfo{
				ID:    hosts[0].ID(),
				Addrs: hosts[0].Addrs(),
			})
			if err != nil {
				panic(err)
			}
		}
	}
	// to give time for the DHT to update its peer table
	time.Sleep(time.Millisecond)
	return &TestDHTNetwork{
		Context: ctx,
		Hosts:   hosts,
		Stores:  stores,
		DHTs:    dhts,
	}
}

// Stop tears down the test network and stops all the services.
// Panics if an error occurs.
func (tn TestDHTNetwork) Stop() {
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
