package p2p_test

import (
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/stretchr/testify/assert"
)

func TestGetValsetConfirmKey(t *testing.T) {
	nonce := uint64(10)
	evmAddr := "0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b"

	expectedKey := "/vc/a:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b"
	actualKey := p2p.GetValsetConfirmKey(nonce, evmAddr)

	assert.Equal(t, expectedKey, actualKey)
}

func TestGetDataCommitmentConfirmKey(t *testing.T) {
	nonce := uint64(10)
	evmAddr := "0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b"

	expectedKey := "/dcc/a:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b"
	actualKey := p2p.GetDataCommitmentConfirmKey(nonce, evmAddr)

	assert.Equal(t, expectedKey, actualKey)
}
