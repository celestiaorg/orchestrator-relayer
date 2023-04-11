package p2p_test

import (
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys/p2p"
	"github.com/ipfs/boxo/keystore"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetP2PKeyOrGenerateNewOne(t *testing.T) {
	tempDir := t.TempDir()
	ks, err := keystore.NewFSKeystore(tempDir)
	require.NoError(t, err)

	nickname := "test"
	// test non-existing nickname
	priv, err := p2p.GetP2PKeyOrGenerateNewOne(ks, nickname)
	// because the key is still not added
	assert.Error(t, err)

	// test empty nickname
	priv, err = p2p.GetP2PKeyOrGenerateNewOne(ks, "")
	// should create a new key with nickname 0
	assert.NoError(t, err)
	assert.NotNil(t, priv)

	// get the key with nickname 0
	priv2, err := p2p.GetP2PKeyOrGenerateNewOne(ks, "0")
	assert.NoError(t, err)
	assert.NotNil(t, priv2)
	assert.Equal(t, priv, priv2)

	// put a new key
	priv3, err := p2p.GenerateNewEd25519()
	require.NoError(t, err)
	err = ks.Put(nickname, priv3)
	require.NoError(t, err)
	priv4, err := p2p.GetP2PKeyOrGenerateNewOne(ks, nickname)
	assert.NoError(t, err)
	assert.NotNil(t, priv4)
	assert.Equal(t, priv3, priv4)
}

func TestGenerateNewEd25519(t *testing.T) {
	priv, err := p2p.GenerateNewEd25519()
	assert.NoError(t, err)
	assert.NotNil(t, priv)
}
