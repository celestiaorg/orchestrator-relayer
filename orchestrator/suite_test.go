package orchestrator_test

import (
	"context"
	"testing"

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
	s.Node = qgbtesting.NewTestNode(ctx, t)
	s.Orchestrator = qgbtesting.NewOrchestrator(s.Node)
}

func (s *OrchestratorTestSuite) TearDownSuite() {
	s.Node.Close()
}

func TestOrchestrator(t *testing.T) {
	suite.Run(t, new(OrchestratorTestSuite))
}
