package relayer_test

import (
	"context"
	"testing"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/orchestrator"

	"github.com/celestiaorg/orchestrator-relayer/relayer"
	qgbtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RelayerTestSuite struct {
	suite.Suite
	Node         *qgbtesting.TestNode
	Orchestrator *orchestrator.Orchestrator
	Relayer      *relayer.Relayer
}

func (s *RelayerTestSuite) SetupSuite() {
	t := s.T()
	if testing.Short() {
		t.Skip("skipping relayer tests in short mode.")
	}
	ctx := context.Background()
	s.Node = qgbtesting.NewTestNode(ctx, t)
	_, err := s.Node.CelestiaNetwork.WaitForHeight(2)
	require.NoError(t, err)
	s.Orchestrator = qgbtesting.NewOrchestrator(t, s.Node)
	s.Relayer = qgbtesting.NewRelayer(t, s.Node)
	go s.Node.EVMChain.PeriodicCommit(ctx, time.Millisecond)
	initVs, err := s.Relayer.AppQuerier.QueryValsetByNonce(s.Node.Context, 1)
	require.NoError(t, err)
	_, _, _, err = s.Relayer.EVMClient.DeployQGBContract(s.Node.EVMChain.Auth, s.Node.EVMChain.Backend, *initVs, 1, true)
	require.NoError(t, err)
}

func (s *RelayerTestSuite) TearDownSuite() {
	s.Node.Close()
}

func TestRelayer(t *testing.T) {
	suite.Run(t, new(RelayerTestSuite))
}
