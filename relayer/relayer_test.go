package relayer_test

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/celestia-app/test/util/testnode"
	qgbtesting "github.com/celestiaorg/orchestrator-relayer/testing"

	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/ipfs/go-datastore"

	blobstreamtypes "github.com/celestiaorg/orchestrator-relayer/types"

	"github.com/stretchr/testify/assert"

	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/stretchr/testify/require"
)

func (s *RelayerTestSuite) TestProcessAttestation() {
	t := s.T()
	_, err := s.Node.CelestiaNetwork.WaitForHeightWithTimeout(400, 30*time.Second)
	require.NoError(t, err)

	ctx := context.Background()
	latestValset, err := s.Orchestrator.AppQuerier.QueryLatestValset(ctx)
	require.NoError(t, err)
	att := types.NewDataCommitment(latestValset.Nonce+1, 10, 100, time.Now())
	commitment, err := s.Orchestrator.TmQuerier.QueryCommitment(ctx, att.BeginBlock, att.EndBlock)
	require.NoError(t, err)
	dataRootTupleRoot := blobstreamtypes.DataCommitmentTupleRootSignBytes(big.NewInt(int64(att.Nonce)), commitment)
	err = s.Orchestrator.ProcessDataCommitmentEvent(ctx, *att, dataRootTupleRoot)
	require.NoError(t, err)

	tx, err := s.Relayer.ProcessAttestation(ctx, s.Node.EVMChain.Auth, att)
	require.NoError(t, err)
	receipt, err := s.Relayer.EVMClient.WaitForTransaction(ctx, s.Node.EVMChain.Backend, tx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), receipt.Status)

	lastNonce, err := s.Relayer.EVMClient.StateLastEventNonce(nil)
	require.NoError(t, err)
	assert.Equal(t, att.Nonce, lastNonce)

	// check if the relayed data commitment confirm is saved to relayer store
	key := datastore.NewKey(p2p.GetDataCommitmentConfirmKey(att.Nonce, s.Orchestrator.EvmAccount.Address.Hex(), dataRootTupleRoot.Hex()))
	has, err := s.Relayer.SignatureStore.Has(ctx, key)
	require.NoError(t, err)
	assert.True(t, has)
}

func TestUseValsetFromP2P(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	codec := encoding.MakeConfig(app.ModuleEncodingRegisters...).Codec
	node := qgbtesting.NewTestNode(
		ctx,
		t,
		qgbtesting.CelestiaNetworkParams{
			GenesisOpts: []testnode.GenesisOption{
				testnode.ImmediateProposals(codec),
				qgbtesting.SetDataCommitmentWindowParams(codec, types.Params{DataCommitmentWindow: 101}),
			},
			TimeIotaMs: 2000000, // so attestations are pruned after they're queried
		},
	)

	// process valset nonce so that it is added to the DHT
	orch := qgbtesting.NewOrchestrator(t, node)
	vs, err := orch.AppQuerier.QueryLatestValset(ctx)
	require.NoError(t, err)
	err = orch.ProcessValsetEvent(ctx, *vs)
	require.NoError(t, err)

	_, err = node.CelestiaNetwork.WaitForHeight(400)
	require.NoError(t, err)

	for {
		time.Sleep(time.Second)
		// Wait until the valset is pruned
		_, err = orch.AppQuerier.QueryLatestValset(ctx)
		if err != nil {
			break
		}
	}

	// the valset should be in the DHT
	latestValset, err := orch.P2PQuerier.QueryLatestValset(ctx)
	require.NoError(t, err)

	att := types.NewDataCommitment(latestValset.Nonce+1, 10, 100, time.Now())
	commitment, err := orch.TmQuerier.QueryCommitment(ctx, att.BeginBlock, att.EndBlock)
	require.NoError(t, err)
	dataRootTupleRoot := blobstreamtypes.DataCommitmentTupleRootSignBytes(big.NewInt(int64(att.Nonce)), commitment)
	err = orch.ProcessDataCommitmentEvent(ctx, *att, dataRootTupleRoot)
	require.NoError(t, err)

	relayer := qgbtesting.NewRelayer(t, node)
	go node.EVMChain.PeriodicCommit(ctx, time.Millisecond)
	_, _, _, err = relayer.EVMClient.DeployBlobstreamContract(node.EVMChain.Auth, node.EVMChain.Backend, *latestValset.ToValset(), latestValset.Nonce, true)
	require.NoError(t, err)

	// make sure the relayer is able to relay the signature using the pruned valset
	tx, err := relayer.ProcessAttestation(ctx, node.EVMChain.Auth, att)
	require.NoError(t, err)

	receipt, err := relayer.EVMClient.WaitForTransaction(ctx, node.EVMChain.Backend, tx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), receipt.Status)

	lastNonce, err := relayer.EVMClient.StateLastEventNonce(nil)
	require.NoError(t, err)
	assert.Equal(t, att.Nonce, lastNonce)
}
