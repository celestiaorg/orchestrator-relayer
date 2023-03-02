package rpc_test

import (
	"context"

	"github.com/celestiaorg/orchestrator-relayer/rpc"
)

func (s *QuerierTestSuite) TestQueryAttestationByNonce() {
	appQuerier := rpc.NewAppQuerier(
		s.Logger,
		s.Network.GRPCClient,
		s.EncConf,
	)

	att, err := appQuerier.QueryAttestationByNonce(context.Background(), 1)
	s.NoError(err)
	s.Equal(uint64(1), att.GetNonce())
}

func (s *QuerierTestSuite) TestQueryLatestAttestationNonce() {
	appQuerier := rpc.NewAppQuerier(
		s.Logger,
		s.Network.GRPCClient,
		s.EncConf,
	)

	nonce, err := appQuerier.QueryLatestAttestationNonce(context.Background())
	s.NoError(err)
	s.Greater(nonce, uint64(1))
}

func (s *QuerierTestSuite) TestQueryDataCommitmentByNonce() {
	appQuerier := rpc.NewAppQuerier(
		s.Logger,
		s.Network.GRPCClient,
		s.EncConf,
	)

	dc, err := appQuerier.QueryDataCommitmentByNonce(context.Background(), 2)
	s.NoError(err)
	s.Equal(uint64(2), dc.Nonce)
}

func (s *QuerierTestSuite) TestQueryDataCommitmentForHeight() {
	appQuerier := rpc.NewAppQuerier(
		s.Logger,
		s.Network.GRPCClient,
		s.EncConf,
	)

	dc, err := appQuerier.QueryDataCommitmentForHeight(context.Background(), 10)
	s.NoError(err)
	s.Equal(uint64(2), dc.Nonce)
}

func (s *QuerierTestSuite) TestQueryValsetByNonce() {
	appQuerier := rpc.NewAppQuerier(
		s.Logger,
		s.Network.GRPCClient,
		s.EncConf,
	)

	vs, err := appQuerier.QueryValsetByNonce(context.Background(), 1)
	s.NoError(err)
	s.Equal(uint64(1), vs.Nonce)
}

func (s *QuerierTestSuite) TestQueryLatestValset() {
	appQuerier := rpc.NewAppQuerier(
		s.Logger,
		s.Network.GRPCClient,
		s.EncConf,
	)

	vs, err := appQuerier.QueryLatestValset(context.Background())
	s.NoError(err)
	s.Equal(uint64(1), vs.Nonce)
}

func (s *QuerierTestSuite) TestQueryLastValsetBeforeNonce() {
	appQuerier := rpc.NewAppQuerier(
		s.Logger,
		s.Network.GRPCClient,
		s.EncConf,
	)

	vs, err := appQuerier.QueryLastValsetBeforeNonce(context.Background(), 2)
	s.NoError(err)
	s.Equal(uint64(1), vs.Nonce)
}

func (s *QuerierTestSuite) TestQueryLastUnbondingHeight() {
	appQuerier := rpc.NewAppQuerier(
		s.Logger,
		s.Network.GRPCClient,
		s.EncConf,
	)

	unbondingHeight, err := appQuerier.QueryLastUnbondingHeight(context.Background())
	s.NoError(err)
	s.Equal(int64(0), unbondingHeight)
}
