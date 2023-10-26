package types_test

import (
	"testing"
	"time"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"

	"github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/stretchr/testify/assert"
)

func TestMarshalValset(t *testing.T) {
	valset := types.LatestValset{
		Nonce:  10,
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

	jsonData, err := types.MarshalLatestValset(valset)
	assert.NoError(t, err)
	expectedJSON := `{"nonce":10,"members":[{"power":100,"evm_address":"evm_addr1"},{"power":200,"evm_address":"evm_addr2"}],"height":5}`
	assert.Equal(t, expectedJSON, string(jsonData))
}

func TestUnmarshalValset(t *testing.T) {
	jsonData := []byte(`{"nonce":10,"members":[{"power":100,"evm_address":"evm_addr1"},{"power":200,"evm_address":"evm_addr2"}],"height":5}`)
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

	valset, err := types.UnmarshalLatestValset(jsonData)
	assert.NoError(t, err)
	assert.True(t, types.IsValsetEqualToLatestValset(expectedValset, valset))
}
