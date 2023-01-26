package types_test

import (
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/stretchr/testify/assert"
)

func TestMarshalDataCommitmentConfirm(t *testing.T) {
	dataCommitmentConfirm := types.DataCommitmentConfirm{
		Signature:  "signature",
		EthAddress: "eth_address",
		Commitment: "commitment",
	}

	jsonData, err := types.MarshalDataCommitmentConfirm(dataCommitmentConfirm)
	assert.NoError(t, err)
	expectedJSON := `{"Signature":"signature","EthAddress":"eth_address","Commitment":"commitment"}`
	assert.Equal(t, expectedJSON, string(jsonData))
}

func TestUnmarshalDataCommitmentConfirm(t *testing.T) {
	jsonData := []byte(`{"Signature":"signature","EthAddress":"eth_address","Commitment":"commitment"}`)
	expectedDataCommitmentConfirm := types.DataCommitmentConfirm{
		Signature:  "signature",
		EthAddress: "eth_address",
		Commitment: "commitment",
	}

	dataCommitmentConfirm, err := types.UnmarshalDataCommitmentConfirm(jsonData)
	assert.NoError(t, err)
	assert.Equal(t, dataCommitmentConfirm, expectedDataCommitmentConfirm)
}
