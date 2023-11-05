package evm_test

import (
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/keys/evm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

func TestMnemonicToPrivateKey(t *testing.T) {
	tests := []struct {
		name            string
		mnemonic        string
		expectedError   bool
		expectedResult  string
		expectedAddress string
	}{
		{
			name:            "Valid Mnemonic and Passphrase",
			mnemonic:        "rescue any open drink foster thing scale country embark stable segment stem portion ostrich spoon hat debate diesel morning galaxy weird firm capital census",
			expectedError:   false,
			expectedResult:  "cb4851012ea2e0421fee67c496b1ae43f0f863903f4e2b57459d3f49f365e926",
			expectedAddress: "0x082d835d29b0519e55401084Ef60fC3D720b62b6",
		},
		{
			name:          "Invalid Mnemonic",
			mnemonic:      "wrong mnemonic beginning poverty injury cradle wrong smoke sphere trap tumble girl monkey sibling festival mask second agent slice gadget census glare swear recycle",
			expectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			privateKey, err := evm.MnemonicToPrivateKey(test.mnemonic, "1234")

			if test.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				addr := crypto.PubkeyToAddress(privateKey.PublicKey)
				assert.Equal(t, test.expectedAddress, addr.Hex())

				expectedPrivateKey, err := crypto.HexToECDSA(test.expectedResult)
				assert.NoError(t, err)
				assert.Equal(t, expectedPrivateKey.D.Bytes(), privateKey.D.Bytes())
			}
		})
	}
}
