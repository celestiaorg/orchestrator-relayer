package evm_test

import (
	"crypto/ecdsa"
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The signatures in these tests are generated using the foundry setup in the quantum-gravity-bridge repository.

func TestNewEthereumSignature(t *testing.T) {
	digest, err := hexutil.Decode("0x078c42ff72a01b355f9d76bfeecd2132a0d3f1aad9380870026c56e23e6d00e5")
	require.NoError(t, err)
	testPrivateKey, err := crypto.HexToECDSA("64a1d6f0e760a8d62b4afdde4096f16f51b401eaaecc915740f71770ea76a8ad")
	require.NoError(t, err)
	tests := []struct {
		name              string
		privKey           *ecdsa.PrivateKey
		expectedSignature string
		expectErr         bool
	}{
		{
			name:              "valid signature",
			privKey:           testPrivateKey,
			expectedSignature: "ca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
			expectErr:         false,
		},
		{
			name:      "nil private key",
			privKey:   nil,
			expectErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evm.NewEthereumSignature(digest, tt.privKey)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSignature, ethcmn.Bytes2Hex(got))
			}
		})
	}
}

func TestEthAddressFromSignature(t *testing.T) {
	digest, err := hexutil.Decode("0x078c42ff72a01b355f9d76bfeecd2132a0d3f1aad9380870026c56e23e6d00e5")
	require.NoError(t, err)
	signature, err := hexutil.Decode("0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00")
	require.NoError(t, err)
	address := ethcmn.HexToAddress("0x9c2B12b5a07FC6D719Ed7646e5041A7E85758329")
	tests := []struct {
		name            string
		signature       []byte
		expectedAddress ethcmn.Address
		expectErr       bool
	}{
		{
			name:            "valid signature and hash",
			signature:       signature,
			expectedAddress: address,
			expectErr:       false,
		},
		{
			name: "short signature",
			signature: func() []byte {
				wrongSig, _ := hexutil.Decode("0x12345")
				return wrongSig
			}(),
			expectErr: true,
		},
		{
			name: "invalid signature",
			signature: func() []byte {
				wrongSig := make([]byte, len(signature))
				copy(wrongSig, signature)
				wrongSig[10] = 10 // changing a single byte to make the signature invalid
				return wrongSig
			}(),
			expectErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := evm.EthAddressFromSignature(digest, tt.signature)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedAddress, got)
			}
		})
	}
}

func TestValidateEthereumSignature(t *testing.T) {
	digest, err := hexutil.Decode("0x078c42ff72a01b355f9d76bfeecd2132a0d3f1aad9380870026c56e23e6d00e5")
	require.NoError(t, err)
	signature, err := hexutil.Decode("0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00")
	require.NoError(t, err)
	address := ethcmn.HexToAddress("0x9c2B12b5a07FC6D719Ed7646e5041A7E85758329")
	tests := []struct {
		name      string
		address   ethcmn.Address
		digest    []byte
		signature []byte
		expectErr bool
	}{
		{
			name:      "valid digest, signature and hash",
			digest:    digest,
			signature: signature,
			address:   address,
			expectErr: false,
		},
		{
			name:      "different address",
			digest:    digest,
			signature: signature,
			address:   ethcmn.HexToAddress("0x7c2B12b5a07FC6D719Ed7646e5041A7E85758329"),
			expectErr: true,
		},
		{
			name:      "different digest",
			digest:    []byte("12345"),
			signature: signature,
			address:   ethcmn.HexToAddress("0x7c2B12b5a07FC6D719Ed7646e5041A7E85758329"),
			expectErr: true,
		},
		{
			name:   "different signature",
			digest: digest,
			signature: func() []byte {
				wrongSig := make([]byte, len(signature))
				copy(wrongSig, signature)
				wrongSig[10] = 10 // changing a single byte to make the signature different but still valid
				return wrongSig
			}(),
			address:   address,
			expectErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := evm.ValidateEthereumSignature(digest, tt.signature, tt.address)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
