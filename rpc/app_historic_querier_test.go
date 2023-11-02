package rpc_test

import (
	"context"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/orchestrator-relayer/rpc"
)

func (s *HistoricQuerierTestSuite) TestQueryHistoricAttestationByNonce() {
	appQuerier := rpc.NewAppQuerier(
		s.Logger,
		s.Network.GRPCAddr,
		s.EncConf,
	)
	require.NoError(s.T(), appQuerier.Start())
	defer appQuerier.Stop() //nolint:errcheck

	// this one should fail because the attestation is deleted from the state
	_, err := appQuerier.QueryAttestationByNonce(context.Background(), 1)
	s.Error(err)

	att, err := appQuerier.QueryHistoricalAttestationByNonce(context.Background(), 1, 10)
	s.NoError(err)
	s.NotNil(att)
	s.Equal(uint64(1), att.GetNonce())
}

func (s *HistoricQuerierTestSuite) TestQueryRecursiveHistoricAttestationByNonce() {
	appQuerier := rpc.NewAppQuerier(
		s.Logger,
		s.Network.GRPCAddr,
		s.EncConf,
	)
	require.NoError(s.T(), appQuerier.Start())
	defer appQuerier.Stop() //nolint:errcheck

	// this one should fail because the attestation is deleted from the state
	_, err := appQuerier.QueryAttestationByNonce(context.Background(), 1)
	s.Error(err)

	height, err := s.Network.LatestHeight()
	s.Require().NoError(err)
	att, err := appQuerier.QueryRecursiveHistoricalAttestationByNonce(context.Background(), 1, uint64(height))
	s.Require().NoError(err)
	s.NotNil(att)
	s.Equal(uint64(1), att.GetNonce())
}

func (s *HistoricQuerierTestSuite) TestQueryHistoricalLatestAttestationNonce() {
	appQuerier := rpc.NewAppQuerier(
		s.Logger,
		s.Network.GRPCAddr,
		s.EncConf,
	)
	require.NoError(s.T(), appQuerier.Start())
	defer appQuerier.Stop() //nolint:errcheck

	nonce, err := appQuerier.QueryHistoricalLatestAttestationNonce(context.Background(), 2)
	s.Require().NoError(err)
	s.Equal(uint64(1), nonce)
}

func (s *HistoricQuerierTestSuite) TestQueryHistoricalValsetByNonce() {
	appQuerier := rpc.NewAppQuerier(
		s.Logger,
		s.Network.GRPCAddr,
		s.EncConf,
	)
	require.NoError(s.T(), appQuerier.Start())
	defer appQuerier.Stop() //nolint:errcheck

	// this one should fail because the attestation is deleted from the state
	_, err := appQuerier.QueryValsetByNonce(context.Background(), 1)
	s.Error(err)

	att, err := appQuerier.QueryHistoricalValsetByNonce(context.Background(), 1, 10)
	s.Require().NoError(err)
	s.NotNil(att)
	s.Equal(uint64(1), att.GetNonce())
}

func (s *HistoricQuerierTestSuite) TestQueryHistoricalLastValsetBeforeNonce() {
	appQuerier := rpc.NewAppQuerier(
		s.Logger,
		s.Network.GRPCAddr,
		s.EncConf,
	)
	require.NoError(s.T(), appQuerier.Start())
	defer appQuerier.Stop() //nolint:errcheck

	// this one should fail because the attestation is deleted from the state
	_, err := appQuerier.QueryLastValsetBeforeNonce(context.Background(), 2)
	s.Error(err)

	att, err := appQuerier.QueryHistoricalLastValsetBeforeNonce(context.Background(), 2, 102)
	s.Require().NoError(err)
	s.NotNil(att)
	s.Equal(uint64(1), att.GetNonce())
}

func (s *HistoricQuerierTestSuite) TestQueryRecursiveHistoricalLastValsetBeforeNonce() {
	appQuerier := rpc.NewAppQuerier(
		s.Logger,
		s.Network.GRPCAddr,
		s.EncConf,
	)
	require.NoError(s.T(), appQuerier.Start())
	defer appQuerier.Stop() //nolint:errcheck

	// this one should fail because the attestation is deleted from the state
	_, err := appQuerier.QueryLastValsetBeforeNonce(context.Background(), 2)
	s.Error(err)

	att, err := appQuerier.QueryRecursiveHistoricalLastValsetBeforeNonce(context.Background(), 2, 201)
	s.Require().NoError(err)
	s.NotNil(att)
	s.Equal(uint64(1), att.GetNonce())
}
