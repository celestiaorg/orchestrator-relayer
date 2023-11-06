package evm_test

import (
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/keys/evm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
)

// TestMnemonicToPrivateKey tests the generation of private keys using mnemonics.
// The test vectors were generated and verified using a Ledger Nano X with Ethereum accounts.
func TestMnemonicToPrivateKey(t *testing.T) {
	tests := []struct {
		name            string
		mnemonic        string
		passphrase      string
		expectedError   bool
		expectedResult  string
		expectedAddress string
	}{
		{
			name:            "Valid Mnemonic with passphrase",
			mnemonic:        "eight moment square film same crystal trophy diagram awkward defense crazy garlic exile rabbit coast truck foam broken shed attract bamboo drum dry cage",
			passphrase:      "abcd",
			expectedError:   false,
			expectedResult:  "5dfb97434a8a31cca1d1c2c6b6b9cf09b4946823331ec434894f204acf79d850",
			expectedAddress: "0x6Ca3653B3B50892e051Da60b1E14540f2f7EBdBF",
		},
		{
			name:            "Valid Mnemonic without passphrase",
			mnemonic:        "eight moment square film same crystal trophy diagram awkward defense crazy garlic exile rabbit coast truck foam broken shed attract bamboo drum dry cage",
			passphrase:      "",
			expectedError:   false,
			expectedResult:  "4252916c6e7f80dc96928c66a885be5a362790ad2fb3552ab781cd9112aef3a2",
			expectedAddress: "0x33bb23EB923C284fC76D93C26aFd1FdCAf770Ea2",
		},
		{
			name:          "Invalid Mnemonic",
			mnemonic:      "wrong mnemonic beginning poverty injury cradle wrong smoke sphere trap tumble girl monkey sibling festival mask second agent slice gadget census glare swear recycle",
			expectedError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			privateKey, err := evm.MnemonicToPrivateKey(test.mnemonic, test.passphrase)

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
