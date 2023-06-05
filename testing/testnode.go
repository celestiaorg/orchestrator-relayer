package testing

import (
	"context"
	"testing"

	celestiatestnode "github.com/celestiaorg/celestia-app/test/util/testnode"
)

// TestNode contains a DHTNetwork along with a test Celestia network and a simulated EVM chain.
type TestNode struct {
	Context         context.Context
	DHTNetwork      *DHTNetwork
	CelestiaNetwork *CelestiaNetwork
	EVMChain        *EVMChain
}

func NewTestNode(ctx context.Context, t *testing.T) *TestNode {
	celestiaNetwork := NewCelestiaNetwork(ctx, t)
	dhtNetwork := NewDHTNetwork(ctx, 2)

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
