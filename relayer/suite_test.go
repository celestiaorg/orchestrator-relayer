package relayer_test

import (
	"context"
	"testing"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/orchestrator"

	"github.com/celestiaorg/orchestrator-relayer/relayer"
	blobstreamtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type RelayerTestSuite struct {
	suite.Suite
	Node         *blobstreamtesting.TestNode
	Orchestrator *orchestrator.Orchestrator
	Relayer      *relayer.Relayer
}

func (s *RelayerTestSuite) SetupSuite() {
	t := s.T()
	if testing.Short() {
		t.Skip("skipping relayer tests in short mode.")
	}
	ctx := context.Background()
	s.Node = blobstreamtesting.NewTestNode(ctx, t)
	_, err := s.Node.CelestiaNetwork.WaitForHeight(2)
	require.NoError(t, err)
	s.Orchestrator = blobstreamtesting.NewOrchestrator(t, s.Node)
	s.Relayer = blobstreamtesting.NewRelayer(t, s.Node)
	go s.Node.EVMChain.PeriodicCommit(ctx, time.Millisecond)
	initVs, err := s.Relayer.AppQuerier.QueryLatestValset(s.Node.Context)
	require.NoError(t, err)
	_, _, _, err = s.Relayer.EVMClient.DeployBlobstreamContract(s.Node.EVMChain.Auth, s.Node.EVMChain.Backend, *initVs, initVs.Nonce, true)
	require.NoError(t, err)
}

func (s *RelayerTestSuite) TearDownSuite() {
	s.Node.Close()
}

func TestRelayer(t *testing.T) {
	suite.Run(t, new(RelayerTestSuite))
}
