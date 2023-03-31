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

	err := store.Init(logger, tmp)
	assert.NoError(t, err)

	isInit := store.IsInit(logger, tmp)
	assert.True(t, isInit)
}
