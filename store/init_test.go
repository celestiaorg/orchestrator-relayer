package store_test

import (
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/store"
	"github.com/stretchr/testify/assert"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func TestInit(t *testing.T) {
	logger := tmlog.NewNopLogger()
	tmp := t.TempDir()

	options := store.InitOptions{
		NeedDataStore:   true,
		NeedEVMKeyStore: true,
		NeedP2PKeyStore: true,
	}

	err := store.Init(logger, tmp, options)
	assert.NoError(t, err)

	isInit := store.IsInit(logger, tmp, options)
	assert.True(t, isInit)
}
