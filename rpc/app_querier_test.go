package rpc_test

import (
	"context"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func (s *QuerierTestSuite) TestQueryAttestationByNonce() {
	t := s.T()
	_, err := s.Network.WaitForHeight(2)
	require.NoError(t, err)

	appQuerier := rpc.NewAppQuerier(
		tmlog.NewNopLogger(),
		s.Network.GRPCClient,
		encoding.MakeConfig(app.ModuleEncodingRegisters...),
	)

	att, err := appQuerier.QueryAttestationByNonce(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), att.GetNonce())
}

func (s *QuerierTestSuite) TestQueryLatestAttestationNonce() {
	t := s.T()
	_, err := s.Network.WaitForHeight(2)
	require.NoError(t, err)

	appQuerier := rpc.NewAppQuerier(
		tmlog.NewNopLogger(),
		s.Network.GRPCClient,
		encoding.MakeConfig(app.ModuleEncodingRegisters...),
	)

	nonce, err := appQuerier.QueryLatestAttestationNonce(context.Background())
	require.NoError(t, err)
	assert.Greater(t, nonce, uint64(1))
}

func (s *QuerierTestSuite) TestQueryDataCommitmentByNonce() {
	t := s.T()
	_, err := s.Network.WaitForHeight(500)
	require.NoError(t, err)

	appQuerier := rpc.NewAppQuerier(
		tmlog.NewNopLogger(),
		s.Network.GRPCClient,
		encoding.MakeConfig(app.ModuleEncodingRegisters...),
	)

	dc, err := appQuerier.QueryDataCommitmentByNonce(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), dc.Nonce)
}

func (s *QuerierTestSuite) TestQueryValsetByNonce() {
	t := s.T()
	_, err := s.Network.WaitForHeight(2)
	require.NoError(t, err)

	appQuerier := rpc.NewAppQuerier(
		tmlog.NewNopLogger(),
		s.Network.GRPCClient,
		encoding.MakeConfig(app.ModuleEncodingRegisters...),
	)

	vs, err := appQuerier.QueryValsetByNonce(context.Background(), 1)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), vs.Nonce)
}

func (s *QuerierTestSuite) TestQueryLatestValset() {
	t := s.T()
	_, err := s.Network.WaitForHeight(2)
	require.NoError(t, err)

	appQuerier := rpc.NewAppQuerier(
		tmlog.NewNopLogger(),
		s.Network.GRPCClient,
		encoding.MakeConfig(app.ModuleEncodingRegisters...),
	)

	vs, err := appQuerier.QueryLatestValset(context.Background())
	require.NoError(t, err)
	assert.Equal(t, uint64(1), vs.Nonce)
}

func (s *QuerierTestSuite) TestQueryLastValsetBeforeNonce() {
	t := s.T()
	_, err := s.Network.WaitForHeight(500)
	require.NoError(t, err)

	appQuerier := rpc.NewAppQuerier(
		tmlog.NewNopLogger(),
		s.Network.GRPCClient,
		encoding.MakeConfig(app.ModuleEncodingRegisters...),
	)

	vs, err := appQuerier.QueryLastValsetBeforeNonce(context.Background(), 2)
	require.NoError(t, err)
	assert.Equal(t, uint64(1), vs.Nonce)
}

func (s *QuerierTestSuite) TestQueryLastUnbondingHeight() {
	t := s.T()
	_, err := s.Network.WaitForHeight(2)
	require.NoError(t, err)

	appQuerier := rpc.NewAppQuerier(
		tmlog.NewNopLogger(),
		s.Network.GRPCClient,
		encoding.MakeConfig(app.ModuleEncodingRegisters...),
	)

	unbondingHeight, err := appQuerier.QueryLastUnbondingHeight(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(0), unbondingHeight)
}
