package rpc_test

import (
	"context"
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
	ctx := context.Background()
	s.Network = qgbtesting.NewCelestiaNetwork(ctx, t, time.Millisecond)
	_, err := s.Network.WaitForHeightWithTimeout(400, 30*time.Second)
	s.EncConf = encoding.MakeConfig(app.ModuleEncodingRegisters...)
	s.Logger = tmlog.NewNopLogger()
	require.NoError(t, err)
}

func TestQueriers(t *testing.T) {
	suite.Run(t, new(QuerierTestSuite))
}
