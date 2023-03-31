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

	// open non initiated store
	_, err := store.OpenStore(logger, path, store.DefaultBadgerOptions(path))
	assert.Error(t, err)

	// init directory
	err = store.Init(logger, path)
	assert.NoError(t, err)

	// open the store again
	s, err := store.OpenStore(logger, path, store.DefaultBadgerOptions(path))
	assert.NoError(t, err)
	assert.NotNil(t, s.DataStore)

	err = s.Close(logger)
	assert.NoError(t, err)
}
