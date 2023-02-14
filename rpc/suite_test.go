package rpc_test

import (
	"testing"
	"time"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/stretchr/testify/require"

	qgbtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	"github.com/stretchr/testify/suite"
)

type QuerierTestSuite struct {
	suite.Suite
	Network *qgbtesting.CelestiaNetwork
	EncConf encoding.Config
	Logger  tmlog.Logger
}

func (s *QuerierTestSuite) SetupSuite() {
	t := s.T()
	if testing.Short() {
		// The main reason for skipping these tests in short mode is to avoid detecting unrelated
		// race conditions.
		// In fact, this test suite uses an existing Celestia-app node to simulate a real environment
		// to execute tests against. However, this leads to data races in multiple areas.
		// Thus, we can skip them as the races detected are not related to this repo.
		t.Skip("skipping queriers tests in short mode.")
	}
	s.Network = qgbtesting.NewCelestiaNetwork(t, time.Millisecond)
	_, err := s.Network.WaitForHeightWithTimeout(400, 30*time.Second)
	s.EncConf = encoding.MakeConfig(app.ModuleEncodingRegisters...)
	s.Logger = tmlog.NewNopLogger()
	require.NoError(t, err)
}

func TestQueriers(t *testing.T) {
	suite.Run(t, new(QuerierTestSuite))
}
