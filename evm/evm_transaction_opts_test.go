package evm_test

import (
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/stretchr/testify/assert"
)

func TestSigToVRS(t *testing.T) {
	sig := "0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00"
	v, r, s := evm.SigToVRS(sig)

	assert.Equal(t, uint8(27), v)
	assert.Equal(t, "0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e1", r.Hex())
	assert.Equal(t, "0x37103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a", s.Hex())
}
