package rpc_test

import (
	"testing"
	"time"

	qgbtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	"github.com/stretchr/testify/suite"
)

type QuerierTestSuite struct {
	suite.Suite
	Network *qgbtesting.CelestiaNetwork
}

func (s *QuerierTestSuite) SetupSuite() {
	t := s.T()
	s.Network = qgbtesting.NewCelestiaNetwork(t, time.Millisecond)
}

func (s *QuerierTestSuite) TearDownSuite() {
	s.Network.Stop()
}

func TestQueriers(t *testing.T) {
	suite.Run(t, new(QuerierTestSuite))
}
