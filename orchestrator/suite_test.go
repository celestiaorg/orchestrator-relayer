package orchestrator_test

import (
	"context"
	"testing"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/celestia-app/test/util/testnode"
	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/orchestrator"
	blobstreamtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	"github.com/stretchr/testify/suite"
)

type OrchestratorTestSuite struct {
	suite.Suite
	Node         *blobstreamtesting.TestNode
	Orchestrator *orchestrator.Orchestrator
}

func (s *OrchestratorTestSuite) SetupSuite() {
	t := s.T()
	ctx := context.Background()
	codec := encoding.MakeConfig(app.ModuleEncodingRegisters...).Codec
	s.Node = blobstreamtesting.NewTestNode(
		ctx,
		t,
		blobstreamtesting.CelestiaNetworkParams{
			GenesisOpts: []testnode.GenesisOption{
				testnode.ImmediateProposals(codec),
				blobstreamtesting.SetDataCommitmentWindowParams(codec, types.Params{DataCommitmentWindow: 101}),
			},
			TimeIotaMs: 1,
		},
	)
	s.Orchestrator = blobstreamtesting.NewOrchestrator(t, s.Node)
}

func (s *OrchestratorTestSuite) TearDownSuite() {
	s.Node.Close()
}

func TestOrchestrator(t *testing.T) {
	suite.Run(t, new(OrchestratorTestSuite))
}
