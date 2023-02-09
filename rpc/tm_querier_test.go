package rpc_test

import (
	"context"
	"fmt"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/celestiaorg/orchestrator-relayer/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func (s *QuerierTestSuite) TestQueryCommitment() {
	t := s.T()
	_, err := s.Network.WaitForHeight(101)
	require.NoError(t, err)

	tmQuerier := rpc.NewTmQuerier(
		s.Network.Client,
		tmlog.NewNopLogger(),
	)

	expectedCommitment, err := s.Network.Client.DataCommitment(context.Background(), 1, 100)
	require.NoError(t, err)
	actualCommitment, err := tmQuerier.QueryCommitment(context.Background(), 1, 100)
	require.NoError(t, err)

	assert.Equal(t, expectedCommitment.DataCommitment, actualCommitment)
}

func (s *QuerierTestSuite) TestSubscribeEvents() {
	t := s.T()
	_, err := s.Network.WaitForHeight(101)
	require.NoError(t, err)

	tmQuerier := rpc.NewTmQuerier(
		s.Network.Client,
		tmlog.NewNopLogger(),
	)

	eventsChan, err := tmQuerier.SubscribeEvents(
		context.Background(),
		"test-subscription",
		fmt.Sprintf("%s.%s='%s'", celestiatypes.EventTypeAttestationRequest, sdk.AttributeKeyModule, celestiatypes.ModuleName),
	)
	require.NoError(t, err)
	event := <-eventsChan
	assert.NotNil(t, event.Events["AttestationRequest.nonce"])
}
