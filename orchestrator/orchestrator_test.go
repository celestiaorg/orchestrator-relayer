package orchestrator_test

import (
	"math/big"
	"testing"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/ethereum/go-ethereum/common/hexutil"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/orchestrator"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (s *OrchestratorTestSuite) TestProcessDataCommitmentEvent() {
	t := s.T()
	_, err := s.Node.CelestiaNetwork.WaitForHeight(50)
	require.NoError(t, err)

	dc := celestiatypes.NewDataCommitment(2, 10, 20, time.Now())
	commitment, err := hexutil.Decode("0x1234")
	require.NoError(t, err)
	dataRootTupleRoot := types.DataCommitmentTupleRootSignBytes(big.NewInt(2), commitment)

	// signing and submitting the signature
	err = s.Orchestrator.ProcessDataCommitmentEvent(s.Node.Context, *dc, dataRootTupleRoot)
	require.NoError(t, err)

	// retrieving the signature
	confirm, err := s.Node.DHTNetwork.DHTs[0].GetDataCommitmentConfirm(
		s.Node.Context,
		p2p.GetDataCommitmentConfirmKey(2, s.Orchestrator.EvmAccount.Address.Hex(), dataRootTupleRoot.Hex()),
	)
	require.NoError(t, err)
	assert.Equal(t, s.Orchestrator.EvmAccount.Address.Hex(), confirm.EthAddress)
}

func (s *OrchestratorTestSuite) TestProcessValsetEvent() {
	t := s.T()
	_, err := s.Node.CelestiaNetwork.WaitForHeight(50)
	require.NoError(t, err)

	vs, err := celestiatypes.NewValset(
		2,
		10,
		[]*celestiatypes.InternalBridgeValidator{{
			Power:      10,
			EVMAddress: s.Orchestrator.EvmAccount.Address,
		}},
		time.Now(),
	)
	require.NoError(t, err)

	signBytes, err := vs.SignBytes()
	require.NoError(t, err)

	// signing and submitting the signature
	err = s.Orchestrator.ProcessValsetEvent(s.Node.Context, *vs)
	require.NoError(t, err)

	// retrieving the signature
	confirm, err := s.Node.DHTNetwork.DHTs[0].GetValsetConfirm(
		s.Node.Context,
		p2p.GetValsetConfirmKey(2, s.Orchestrator.EvmAccount.Address.Hex(), signBytes.Hex()),
	)
	require.NoError(t, err)
	assert.Equal(t, s.Orchestrator.EvmAccount.Address.Hex(), confirm.EthAddress)
}

func TestValidatorPartOfValset(t *testing.T) {
	tests := []struct {
		name           string
		members        []celestiatypes.BridgeValidator
		evmAddr        string
		expectedResult bool
	}{
		{
			name: "validator found",
			members: []celestiatypes.BridgeValidator{
				{EvmAddress: "0x123"},
				{EvmAddress: "0x456"},
				{EvmAddress: "0x789"},
			},
			evmAddr:        "0x456",
			expectedResult: true,
		},
		{
			name: "validator not found",
			members: []celestiatypes.BridgeValidator{
				{EvmAddress: "0x123"},
				{EvmAddress: "0x456"},
				{EvmAddress: "0x789"},
			},
			evmAddr:        "0x999",
			expectedResult: false,
		},
		{
			name:           "empty members",
			members:        []celestiatypes.BridgeValidator{},
			evmAddr:        "0x999",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := orchestrator.ValidatorPartOfValset(tt.members, tt.evmAddr)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}
