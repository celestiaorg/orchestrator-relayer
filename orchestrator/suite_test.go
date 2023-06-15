package orchestrator_test

import (
	"context"
	"testing"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/celestia-app/test/util/testnode"
	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/orchestrator"
	qgbtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	"github.com/stretchr/testify/suite"
)

type OrchestratorTestSuite struct {
	suite.Suite
	Node         *qgbtesting.TestNode
	Orchestrator *orchestrator.Orchestrator
}

func (s *OrchestratorTestSuite) SetupSuite() {
	t := s.T()
	ctx := context.Background()
	codec := encoding.MakeConfig(app.ModuleEncodingRegisters...).Codec
	s.Node = qgbtesting.NewTestNode(
		ctx,
		t,
		testnode.ImmediateProposals(codec),
		qgbtesting.SetDataCommitmentWindowParams(codec, types.Params{DataCommitmentWindow: 101}),
		// qgbtesting.SetVotingParams(codec, v1beta1.VotingParams{VotingPeriod: 100 * time.Hour}),
	)
	s.Orchestrator = qgbtesting.NewOrchestrator(t, s.Node)
}

func (s *OrchestratorTestSuite) TearDownSuite() {
	s.Node.Close()
}

func TestOrchestrator(t *testing.T) {
	suite.Run(t, new(OrchestratorTestSuite))
}
