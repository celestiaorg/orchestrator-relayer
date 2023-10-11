package orchestrator_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/celestiaorg/celestia-app/test/util/testnode"
	qgbtesting "github.com/celestiaorg/orchestrator-relayer/testing"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	tmlog "github.com/tendermint/tendermint/libs/log"

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

func (s *OrchestratorTestSuite) TestEnqueuingAttestationNonces() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t := s.T()
	_, err := s.Node.CelestiaNetwork.WaitForHeight(10)
	require.NoError(t, err)

	// nonces queue will be closed below
	noncesQueue := make(chan uint64, 100)
	signalChan := make(chan struct{})
	defer close(signalChan)

	go func() {
		_ = s.Orchestrator.StartNewEventsListener(ctx, noncesQueue, signalChan)
	}()
	go func() {
		_ = s.Orchestrator.EnqueueMissingEvents(ctx, noncesQueue, signalChan)
	}()

	// set the data commitment window to a high value
	s.Node.CelestiaNetwork.SetDataCommitmentWindow(t, 1000)
	_, err = s.Node.CelestiaNetwork.WaitForHeightWithTimeout(1500, time.Minute)
	assert.NoError(t, err)

	// set the data commitment window to a low value
	s.Node.CelestiaNetwork.SetDataCommitmentWindow(t, 100)

	ecfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)
	appQuerier := rpc.NewAppQuerier(
		tmlog.NewNopLogger(),
		s.Node.CelestiaNetwork.GRPCAddr,
		ecfg,
	)
	require.NoError(s.T(), appQuerier.Start())
	defer appQuerier.Stop() //nolint:errcheck

	latestNonce, err := appQuerier.QueryLatestAttestationNonce(ctx)
	s.NoError(err)

	cancel()
	close(noncesQueue)
	assert.GreaterOrEqual(t, len(noncesQueue), int(latestNonce))
}

func TestProcessWithoutValsetInStore(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	codec := encoding.MakeConfig(app.ModuleEncodingRegisters...).Codec
	node := qgbtesting.NewTestNode(
		ctx,
		t,
		qgbtesting.CelestiaNetworkParams{
			GenesisOpts: []testnode.GenesisOption{
				testnode.ImmediateProposals(codec),
				qgbtesting.SetDataCommitmentWindowParams(codec, celestiatypes.Params{DataCommitmentWindow: 101}),
			},
			TimeIotaMs: 6048000, // to have enough time to sign attestations after they're pruned
		},
	)
	_, err := node.CelestiaNetwork.WaitForHeight(400)
	require.NoError(t, err)

	orch := qgbtesting.NewOrchestrator(t, node)

	latestNonce, err := orch.AppQuerier.QueryLatestAttestationNonce(ctx)
	require.NoError(t, err)
	assert.NoError(t, orch.Process(ctx, latestNonce))
}
