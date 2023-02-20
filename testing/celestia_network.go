package testing

import (
	"testing"
	"time"

	celestiatestnode "github.com/celestiaorg/celestia-app/testutil/testnode"
)

// CelestiaNetwork is a Celestia-app validator running in-process.
// The EVM key that was used to create this network's single validator can
// be retrieved using: `celestiatestnode.NodeEVMPrivateKey`
type CelestiaNetwork struct {
	celestiatestnode.Context
	Accounts  []string
	BlockTime time.Duration
}

// NewCelestiaNetwork creates a new CelestiaNetwork.
// Uses `testing.T` to fail if an error happens.
// Only supports the creation of a single validator currently.
func NewCelestiaNetwork(t *testing.T, blockTime time.Duration) *CelestiaNetwork {
	if testing.Short() {
		// The main reason for skipping these tests in short mode is to avoid detecting unrelated
		// race conditions.
		// In fact, this test suite uses an existing Celestia-app node to simulate a real environment
		// to execute tests against. However, this leads to data races in multiple areas.
		// Thus, we can skip them as the races detected are not related to this repo.
		t.Skip("skipping tests in short mode.")
	}
	accounts, clientContext := celestiatestnode.DefaultNetwork(t, blockTime)
	return &CelestiaNetwork{
		Context:   clientContext,
		Accounts:  accounts,
		BlockTime: blockTime,
	}
}
