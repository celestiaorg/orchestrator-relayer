package types_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"math/big"
	"strconv"
	"testing"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDataCommitmentTupleRootSignBytes(t *testing.T) {
	nonce := int64(1)
	commitment := bytes.Repeat([]byte{2}, 32)

	hexRepresentation := strconv.FormatInt(nonce, 16)
	// Make sure hex representation has even length
	if len(hexRepresentation)%2 == 1 {
		hexRepresentation = "0" + hexRepresentation
	}
	hexBytes, err := hex.DecodeString(hexRepresentation)
	require.NoError(t, err)
	paddedNonce, err := padBytes(hexBytes, 32)
	require.NoError(t, err)

	expectedHash := crypto.Keccak256Hash(append(
		celestiatypes.DcDomainSeparator[:],
		append(
			paddedNonce,
			commitment...,
		)...,
	))

	result := types.DataCommitmentTupleRootSignBytes(big.NewInt(nonce), commitment)

	assert.Equal(t, expectedHash, result)
}

func TestMarshalDataCommitmentConfirm(t *testing.T) {
	dataCommitmentConfirm := types.DataCommitmentConfirm{
		Signature:  "signature",
		EthAddress: "eth_address",
	}

	jsonData, err := types.MarshalDataCommitmentConfirm(dataCommitmentConfirm)
	assert.NoError(t, err)
	expectedJSON := `{"Signature":"signature","EthAddress":"eth_address"}`
	assert.Equal(t, expectedJSON, string(jsonData))
}

func TestUnmarshalDataCommitmentConfirm(t *testing.T) {
	jsonData := []byte(`{"Signature":"signature","EthAddress":"eth_address"}`)
	expectedDataCommitmentConfirm := types.DataCommitmentConfirm{
		Signature:  "signature",
		EthAddress: "eth_address",
	}

	dataCommitmentConfirm, err := types.UnmarshalDataCommitmentConfirm(jsonData)
	assert.NoError(t, err)
	assert.Equal(t, dataCommitmentConfirm, expectedDataCommitmentConfirm)
}

// padBytes Pad bytes to a given length
func padBytes(buf []byte, length int) ([]byte, error) {
	l := len(buf)
	if l > length {
		return nil, fmt.Errorf(
			"cannot pad bytes because length of bytes array: %d is greater than given length: %d",
			l,
			length,
		)
	}
	if l == length {
		return buf, nil
	}
	tmp := make([]byte, length)
	copy(tmp[length-l:], buf)
	return tmp, nil
}
