package evm_test

import (
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/stretchr/testify/assert"
)

func TestSigToVRS(t *testing.T) {
	tests := []struct {
		name      string
		signature string
		expectedR string
		expectedS string
		expectedV uint8
		expectErr bool
	}{
		{
			name:      "valid signature with vParam=0",
			signature: "0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
			expectedR: "0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e1",
			expectedS: "0x37103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a",
			expectedV: uint8(27),
			expectErr: false,
		},
		{
			name:      "valid signature with vParam=27",
			signature: "0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a1b",
			expectedR: "0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e1",
			expectedS: "0x37103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a",
			expectedV: uint8(27),
			expectErr: false,
		},
		{
			name:      "short signature",
			signature: "0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a",
			expectErr: true,
		},
		{
			name:      "long signature",
			signature: "0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a001b1",
			expectErr: true,
		},
		{
			name:      "valid signature with invalid vParam=10",
			signature: "0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a0a",
			expectErr: true,
		},
		{
			name:      "invalid zero sParam",
			signature: "0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e1000000000000000000000000000000000000000000000000000000000000000000",
			expectErr: true,
		},
		{
			name:      "sParam higher than malleability threshold",
			signature: "0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e17fffffffffffffffffffffffffffffff5d576e7357a4501ddfe92f46681b20a000",
			expectErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, r, s, err := evm.SigToVRS(tt.signature)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedV, v)
				assert.Equal(t, tt.expectedR, r.Hex())
				assert.Equal(t, tt.expectedS, s.Hex())
			}
		})
	}
}
