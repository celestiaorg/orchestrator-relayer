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

func TestParseKey(t *testing.T) {
	tests := []struct {
		name            string
		key             string
		expectedNs      string
		expectedNonce   uint64
		expectedEVMAddr string
		wantErr         bool
	}{
		{
			name:            "valid valset confirm key",
			key:             "/vc/b:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b",
			expectedNs:      p2p.ValsetConfirmNamespace,
			expectedNonce:   11,
			expectedEVMAddr: "0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b",
			wantErr:         false,
		},
		{
			name:            "valid data commitment confirm key",
			key:             "/dcc/a:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b",
			expectedNs:      p2p.DataCommitmentConfirmNamespace,
			expectedNonce:   10,
			expectedEVMAddr: "0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b",
			wantErr:         false,
		},
		{
			name:    "missing namespace",
			key:     "/10:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b",
			wantErr: true,
		},
		{
			name:    "empty namespace",
			key:     "//10:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b",
			wantErr: true,
		},
		{
			name:    "missing nonce",
			key:     "/inv/0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b",
			wantErr: true,
		},
		{
			name:    "empty nonce",
			key:     "/inv/:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b",
			wantErr: true,
		},
		{
			name:    "invalid nonce",
			key:     "/inv/abjj:0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b",
			wantErr: true,
		},
		{
			name:    "missing evm address",
			key:     "/inv/123",
			wantErr: true,
		},
		{
			name:    "empty evm address",
			key:     "/inv/123:",
			wantErr: true,
		},
		{
			name:    "more /",
			key:     "/inv/123/123",
			wantErr: true,
		},
		{
			name:    "more :",
			key:     "/inv/123:123:123",
			wantErr: true,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			namespace, nonce, evmAddr, err := p2p.ParseKey(tt.key)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedNs, namespace)
				assert.Equal(t, tt.expectedNonce, nonce)
				assert.Equal(t, tt.expectedEVMAddr, evmAddr)
			}
		})
	}
}
