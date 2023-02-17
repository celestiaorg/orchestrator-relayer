package relayer_test

import (
	"context"

	"github.com/stretchr/testify/assert"

	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/stretchr/testify/require"
)

func (s *RelayerTestSuite) TestProcessAttestation() {
	t := s.T()
	_, err := s.Node.CelestiaNetwork.WaitForHeight(500)
	require.NoError(t, err)

	att := types.NewDataCommitment(2, 10, 100)
	ctx := context.Background()
	err = s.Orchestrator.ProcessDataCommitmentEvent(ctx, *att)
	require.NoError(t, err)

	tx, err := s.Relayer.ProcessAttestation(ctx, s.Node.EVMChain.Auth, att)
	require.NoError(t, err)
	receipt, err := s.Relayer.EVMClient.WaitForTransaction(ctx, s.Node.EVMChain.Backend, tx)
	assert.NoError(t, err)
	assert.Equal(t, uint64(1), receipt.Status)

	lastNonce, err := s.Relayer.EVMClient.StateLastEventNonce(nil)
	require.NoError(t, err)
	assert.Equal(t, att.Nonce, lastNonce)
}
