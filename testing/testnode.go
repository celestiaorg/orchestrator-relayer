package testing

import (
	"context"
	"testing"
	"time"

	celestiatestnode "github.com/celestiaorg/celestia-app/testutil/testnode"
)

// TestNode contains a DHTNetwork along with a test Celestia network and a simulated EVM chain.
type TestNode struct {
	Context         context.Context
	DHTNetwork      *DHTNetwork
	CelestiaNetwork *CelestiaNetwork
	EVMChain        *EVMChain
}

func NewTestNode(ctx context.Context, t *testing.T) *TestNode {
	celestiaNetwork := NewCelestiaNetwork(t, time.Millisecond)
	dhtNetwork := NewDHTNetwork(ctx, 1)

	evmChain := NewEVMChain(celestiatestnode.NodeEVMPrivateKey)

	return &TestNode{
		Context:         ctx,
		DHTNetwork:      dhtNetwork,
		CelestiaNetwork: celestiaNetwork,
		EVMChain:        evmChain,
	}
}

func (tn TestNode) Close() {
	tn.DHTNetwork.Stop()
	tn.EVMChain.Close()
}
