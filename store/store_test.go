package store_test

import (
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/store"
	"github.com/stretchr/testify/assert"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func TestStore(t *testing.T) {
	logger := tmlog.NewNopLogger()
	path := t.TempDir()

	options := store.OpenOptions{
		HasDataStore:   true,
		BadgerOptions:  store.DefaultBadgerOptions(path),
		HasEVMKeyStore: false,
		HasP2PKeyStore: false,
	}
	// open non initiated store
	_, err := store.OpenStore(logger, path, options)
	assert.Error(t, err)

	// init directory
	err = store.Init(logger, path, store.InitOptions{
		NeedDataStore:   true,
		NeedEVMKeyStore: false,
		NeedP2PKeyStore: false,
	})
	assert.NoError(t, err)

	// open the store again
	s, err := store.OpenStore(logger, path, options)
	assert.NoError(t, err)
	assert.NotNil(t, s.DataStore)

	err = s.Close(logger, options)
	assert.NoError(t, err)
}
