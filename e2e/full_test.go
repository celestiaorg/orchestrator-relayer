package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFullLongBehaviour mainly lets a multiple validator network run for 200 blocks, then checks if
// the valsets and data commitments are relayed correctly.
func TestFullLongBehaviour(t *testing.T) {
	if os.Getenv("QGB_INTEGRATION_TEST") != TRUE {
		t.Skip("Skipping QGB integration tests")
	}

	network, err := NewQGBNetwork()
	HandleNetworkError(t, network, err, false)

	// to release resources after tests
	defer network.DeleteAll() //nolint:errcheck

	// start full network with four validators
	err = network.StartAll()
	HandleNetworkError(t, network, err, false)

	ctx := context.Background()

	err = network.WaitForBlockWithCustomTimeout(ctx, 200, 8*time.Minute)
	HandleNetworkError(t, network, err, false)

	// check whether the four validators are up and running
	qgbGRPC, err := grpc.Dial(network.CelestiaGRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	HandleNetworkError(t, network, err, false)
	defer qgbGRPC.Close()
	appQuerier := rpc.NewAppQuerier(network.Logger, qgbGRPC, network.EncCfg)
	HandleNetworkError(t, network, err, false)

	// check whether all the validators are up and running
	latestValset, err := appQuerier.QueryLatestValset(ctx)
	assert.NoError(t, err)
	require.NotNil(t, latestValset)
	assert.Equal(t, 4, len(latestValset.Members))

	// check whether the QGB contract was deployed
	bridge, err := network.GetLatestDeployedQGBContract(ctx)
	HandleNetworkError(t, network, err, false)

	evmClient := evm.NewClient(nil, bridge, nil, network.EVMRPC, evm.DEFAULTEVMGASLIMIT)

	// check whether the relayer relayed all attestations
	eventNonce, err := evmClient.StateLastEventNonce(&bind.CallOpts{Context: ctx})
	assert.NoError(t, err)

	// attestations are either data commitments or valsets
	latestNonce, err := appQuerier.QueryLatestAttestationNonce(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, eventNonce, latestNonce-1)
}
