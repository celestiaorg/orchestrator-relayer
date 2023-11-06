package relayer_test

import (
	"context"
	"testing"
	"time"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/celestia-app/test/util/testnode"
	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/rpc"

	"github.com/celestiaorg/orchestrator-relayer/orchestrator"

	"github.com/celestiaorg/orchestrator-relayer/relayer"
	blobstreamtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HistoricalRelayerTestSuite struct {
	suite.Suite
	Node         *blobstreamtesting.TestNode
	Orchestrator *orchestrator.Orchestrator
	Relayer      *relayer.Relayer
}

func (s *HistoricalRelayerTestSuite) SetupSuite() {
	t := s.T()
	if testing.Short() {
		t.Skip("skipping relayer tests in short mode.")
	}
	ctx := context.Background()
	s.Node = blobstreamtesting.NewTestNode(
		ctx,
		t,
		blobstreamtesting.CelestiaNetworkParams{
			GenesisOpts: []testnode.GenesisOption{blobstreamtesting.SetDataCommitmentWindowParams(
				encoding.MakeConfig(app.ModuleEncodingRegisters...).Codec,
				types.Params{DataCommitmentWindow: 101},
			)},
			TimeIotaMs:    3048000,   // so that old attestations are deleted as soon as a new one appears
			Pruning:       "nothing", // make the node an archive one
			TimeoutCommit: 20 * time.Millisecond,
		},
	)
	_, err := s.Node.CelestiaNetwork.WaitForHeight(2)
	require.NoError(t, err)
	s.Orchestrator = blobstreamtesting.NewOrchestrator(t, s.Node)
	s.Relayer = blobstreamtesting.NewRelayer(t, s.Node)
	go s.Node.EVMChain.PeriodicCommit(ctx, time.Millisecond)
	initVs, err := s.Relayer.AppQuerier.QueryLatestValset(s.Node.Context)
	require.NoError(t, err)
	_, _, _, err = s.Relayer.EVMClient.DeployBlobstreamContract(s.Node.EVMChain.Auth, s.Node.EVMChain.Backend, *initVs, initVs.Nonce, true)
	require.NoError(t, err)
	rpc.BlocksIn20DaysPeriod = 50
}

func (s *HistoricalRelayerTestSuite) TearDownSuite() {
	s.Node.Close()
}

func TestHistoricRelayer(t *testing.T) {
	suite.Run(t, new(HistoricalRelayerTestSuite))
}
