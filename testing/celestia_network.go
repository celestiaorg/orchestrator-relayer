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
	Cleanup   func() error
	BlockTime time.Duration
}

// NewCelestiaNetwork creates a new CelestiaNetwork.
// Uses `testing.T` to fail if an error happens.
// Only supports the creation of a single validator currently.
func NewCelestiaNetwork(t *testing.T, blockTime time.Duration) *CelestiaNetwork {
	cleanup, accounts, clientContext := celestiatestnode.DefaultNetwork(t, blockTime)
	return &CelestiaNetwork{
		Context:   clientContext,
		Accounts:  accounts,
		Cleanup:   cleanup,
		BlockTime: blockTime,
	}
}

// Stop tears down the Celestia network and panics in case of error.
func (cn CelestiaNetwork) Stop() {
	err := cn.Cleanup()
	if err != nil {
		panic(err)
	}
}
