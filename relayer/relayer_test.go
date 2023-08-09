package relayer_test

import (
	"context"
	"math/big"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/ipfs/go-datastore"

	qgbtypes "github.com/celestiaorg/orchestrator-relayer/types"

	"github.com/stretchr/testify/assert"

	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/stretchr/testify/require"
)

func (s *RelayerTestSuite) TestProcessAttestation() {
	t := s.T()
	_, err := s.Node.CelestiaNetwork.WaitForHeightWithTimeout(400, 30*time.Second)
	require.NoError(t, err)

	att := types.NewDataCommitment(2, 10, 100, time.Now())
	ctx := context.Background()
	commitment, err := s.Orchestrator.TmQuerier.QueryCommitment(ctx, att.BeginBlock, att.EndBlock)
	require.NoError(t, err)
	dataRootTupleRoot := qgbtypes.DataCommitmentTupleRootSignBytes(big.NewInt(int64(att.Nonce)), commitment)
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
