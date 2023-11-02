package relayer_test

import (
	"context"
	"math/big"
	"time"

	blobstreamtypes "github.com/celestiaorg/orchestrator-relayer/types"

	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/stretchr/testify/require"
)

func (s *HistoricalRelayerTestSuite) TestProcessHistoricAttestation() {
	t := s.T()
	_, err := s.Node.CelestiaNetwork.WaitForHeightWithTimeout(400, 30*time.Second)
	require.NoError(t, err)

	ctx := context.Background()
	valset, err := s.Orchestrator.AppQuerier.QueryLatestValset(ctx)
	require.NoError(t, err)

	for {
		_, err = s.Orchestrator.AppQuerier.QueryAttestationByNonce(ctx, valset.Nonce)
		if err != nil {
			break
		}
	}

	att := types.NewDataCommitment(valset.Nonce+1, 10, 100, time.Now())
	commitment, err := s.Orchestrator.TmQuerier.QueryCommitment(ctx, att.BeginBlock, att.EndBlock)
	require.NoError(t, err)
	dataRootTupleRoot := blobstreamtypes.DataCommitmentTupleRootSignBytes(big.NewInt(int64(att.Nonce)), commitment)
	err = s.Orchestrator.ProcessDataCommitmentEvent(ctx, *att, dataRootTupleRoot)
	require.NoError(t, err)

	_, err = s.Relayer.ProcessAttestation(ctx, s.Node.EVMChain.Auth, att)
	require.NoError(t, err)
}
