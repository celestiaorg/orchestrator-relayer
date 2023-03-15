package types_test

import (
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/stretchr/testify/assert"
)

func TestMarshalValsetConfirm(t *testing.T) {
	valsetConfirm := types.ValsetConfirm{
		EthAddress: "eth_address",
		Signature:  "signature",
		SignBytes:  "bytes",
	}

	jsonData, err := types.MarshalValsetConfirm(valsetConfirm)
	assert.NoError(t, err)
	expectedJSON := `{"EthAddress":"eth_address","Signature":"signature","SignBytes":"bytes"}`
	assert.Equal(t, string(jsonData), expectedJSON)
}

func TestUnmarshalValsetConfirm(t *testing.T) {
	jsonData := []byte(`{"EthAddress":"eth_address","Signature":"signature","SignBytes":"bytes"}`)
	expectedValsetConfirm := types.ValsetConfirm{
		EthAddress: "eth_address",
		Signature:  "signature",
		SignBytes:  "bytes",
	}

	valsetConfirm, err := types.UnmarshalValsetConfirm(jsonData)
	assert.NoError(t, err)
	assert.Equal(t, valsetConfirm, expectedValsetConfirm)
}
