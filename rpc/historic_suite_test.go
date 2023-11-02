package rpc_test

import (
	"context"
	"testing"
	"time"

	"github.com/celestiaorg/celestia-app/test/util/testnode"
	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/rpc"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/stretchr/testify/require"

	blobstreamtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	"github.com/stretchr/testify/suite"
)

type HistoricQuerierTestSuite struct {
	suite.Suite
	Network *blobstreamtesting.CelestiaNetwork
	EncConf encoding.Config
	Logger  tmlog.Logger
}

func (s *HistoricQuerierTestSuite) SetupSuite() {
	t := s.T()
	ctx := context.Background()
	s.EncConf = encoding.MakeConfig(app.ModuleEncodingRegisters...)
	s.Network = blobstreamtesting.NewCelestiaNetwork(
		ctx,
		t,
		blobstreamtesting.CelestiaNetworkParams{
			GenesisOpts:   []testnode.GenesisOption{blobstreamtesting.SetDataCommitmentWindowParams(s.EncConf.Codec, types.Params{DataCommitmentWindow: 101})},
			TimeIotaMs:    6048000,   // so that old attestations are deleted as soon as a new one appears
			Pruning:       "nothing", // make the node an archive one
			TimeoutCommit: 20 * time.Millisecond,
		},
	)
	_, err := s.Network.WaitForHeightWithTimeout(401, 30*time.Second)
	require.NoError(t, err)
	s.Logger = tmlog.NewNopLogger()
	rpc.BlocksIn20DaysPeriod = 100
}

func TestHistoricQueriers(t *testing.T) {
	suite.Run(t, new(HistoricQuerierTestSuite))
}

func (s *HistoricQuerierTestSuite) setupAppQuerier() *rpc.AppQuerier {
	appQuerier := rpc.NewAppQuerier(
		s.Logger,
		s.Network.GRPCAddr,
		s.EncConf,
	)
	require.NoError(s.T(), appQuerier.Start())
	s.T().Cleanup(func() {
		appQuerier.Stop() //nolint:errcheck
	})
	return appQuerier
}
