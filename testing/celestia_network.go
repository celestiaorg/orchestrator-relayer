package testing

import (
	"testing"
	"time"

	celestiatestnode "github.com/celestiaorg/celestia-app/testutil/testnode"
)

// CelestiaNetwork is a Celestia-app validator running in-process.
type CelestiaNetwork struct {
	celestiatestnode.Context
	Accounts  []string
	BlockTime time.Duration
}

// NewCelestiaNetwork creates a new CelestiaNetwork.
// Uses `testing.T` to fail if an error happens.
// Only supports the creation of a single validator currently.
func NewCelestiaNetwork(t *testing.T, blockTime time.Duration) *CelestiaNetwork {
	accounts, clientContext := celestiatestnode.DefaultNetwork(t, blockTime)
	return &CelestiaNetwork{
		Context:   clientContext,
		Accounts:  accounts,
		BlockTime: blockTime,
	}
}
