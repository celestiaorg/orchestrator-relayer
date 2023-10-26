package types_test

import (
	"testing"
	"time"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"

	"github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/stretchr/testify/assert"
)

func TestMarshalValset(t *testing.T) {
	valset := celestiatypes.Valset{
		Nonce:  10,
		Time:   time.UnixMicro(10),
		Height: 5,
		Members: []celestiatypes.BridgeValidator{
			{
				Power:      100,
				EvmAddress: "evm_addr1",
			},
			{
				Power:      200,
				EvmAddress: "evm_addr2",
			},
		},
	}

	jsonData, err := types.MarshalValset(valset)
	assert.NoError(t, err)
	expectedJSON := `{"nonce":10,"members":[{"power":100,"evm_address":"evm_addr1"},{"power":200,"evm_address":"evm_addr2"}],"height":5,"time":"1970-01-01T01:00:00.00001+01:00"}`
	assert.Equal(t, string(jsonData), expectedJSON)
}

func TestUnmarshalValset(t *testing.T) {
	jsonData := []byte(`{"nonce":10,"members":[{"power":100,"evm_address":"evm_addr1"},{"power":200,"evm_address":"evm_addr2"}],"height":5,"time":"1970-01-01T01:00:00.00001+01:00"}`)
	expectedValset := celestiatypes.Valset{
		Nonce:  10,
		Time:   time.UnixMicro(10),
		Height: 5,
		Members: []celestiatypes.BridgeValidator{
			{
				Power:      100,
				EvmAddress: "evm_addr1",
			},
			{
				Power:      200,
				EvmAddress: "evm_addr2",
			},
		},
	}

	valset, err := types.UnmarshalValset(jsonData)
	assert.NoError(t, err)
	assert.Equal(t, valset, expectedValset)
}
