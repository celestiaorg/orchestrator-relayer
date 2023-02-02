package testing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	mathrand "math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
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

	// generate a random EVM private key
	bytes := make([]byte, 32)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	privateKey, err := crypto.HexToECDSA(hex.EncodeToString(bytes))
	if err != nil {
		panic(err)
	}

	evmChain := NewEVMChain(privateKey, mathrand.Uint64())

	return &TestNode{
		Context:         ctx,
		DHTNetwork:      dhtNetwork,
		CelestiaNetwork: celestiaNetwork,
		EVMChain:        evmChain,
	}
}

func (tn TestNode) Close() {
	tn.DHTNetwork.Stop()
	tn.CelestiaNetwork.Stop()
	tn.EVMChain.Close()
}
