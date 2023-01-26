package p2p_test

import (
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/stretchr/testify/assert"
)

func TestGetValsetConfirmKey(t *testing.T) {
	nonce := uint64(10)
	orchAddr := "celes1qktu8009djs6uym9uwj84ead24exkezsaqrmn5"

	expectedKey := "/vc/a:celes1qktu8009djs6uym9uwj84ead24exkezsaqrmn5"
	actualKey := p2p.GetValsetConfirmKey(nonce, orchAddr)

	assert.Equal(t, expectedKey, actualKey)
}

func TestGetDataCommitmentConfirmKey(t *testing.T) {
	nonce := uint64(10)
	orchAddr := "celes1qktu8009djs6uym9uwj84ead24exkezsaqrmn5"

	expectedKey := "/dcc/a:celes1qktu8009djs6uym9uwj84ead24exkezsaqrmn5"
	actualKey := p2p.GetDataCommitmentConfirmKey(nonce, orchAddr)

	assert.Equal(t, expectedKey, actualKey)
}
