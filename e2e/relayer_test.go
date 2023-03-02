package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/helpers"
	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	qgbtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/stretchr/testify/assert"
)

func TestRelayerWithOneValidator(t *testing.T) {
	if os.Getenv("QGB_INTEGRATION_TEST") != TRUE {
		t.Skip("Skipping QGB integration tests")
	}

	network, err := NewQGBNetwork()
	HandleNetworkError(t, network, err, false)

	// to release resources after tests
	defer network.DeleteAll() //nolint:errcheck

	err = network.StartMinimal()
	HandleNetworkError(t, network, err, false)

	ctx := context.Background()
	err = network.WaitForBlock(ctx, int64(network.DataCommitmentWindow+50))
	HandleNetworkError(t, network, err, false)

	// create dht for querying
	bootstrapper, err := helpers.ParseAddrInfos(network.Logger, BOOTSTRAPPERS)
	HandleNetworkError(t, network, err, false)
	_, _, dht := qgbtesting.NewTestDHT(ctx, bootstrapper)
	defer dht.Close()

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE0ACCOUNTADDRESS)
	HandleNetworkError(t, network, err, false)

	bridge, err := network.GetLatestDeployedQGBContract(ctx)
	HandleNetworkError(t, network, err, false)

	err = network.WaitForRelayerToStart(ctx, bridge)
	HandleNetworkError(t, network, err, false)

	evmClient := evm.NewClient(nil, bridge, nil, network.EVMRPC, evm.DEFAULTEVMGASLIMIT)

	vsNonce, err := evmClient.StateLastEventNonce(&bind.CallOpts{Context: ctx})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, vsNonce, uint64(2))
}

func TestRelayerWithTwoValidators(t *testing.T) {
	if os.Getenv("QGB_INTEGRATION_TEST") != TRUE {
		t.Skip("Skipping QGB integration tests")
	}

	network, err := NewQGBNetwork()
	HandleNetworkError(t, network, err, false)

	// to release resources after tests
	defer network.DeleteAll() //nolint:errcheck

	// start minimal network with one validator
	err = network.StartMinimal()
	HandleNetworkError(t, network, err, false)

	// add second validator
	err = network.Start(Core1)
	HandleNetworkError(t, network, err, false)

	// add second orchestrator
	err = network.Start(Core1Orch)
	HandleNetworkError(t, network, err, false)

	ctx := context.Background()

	err = network.WaitForBlock(ctx, int64(network.DataCommitmentWindow+50))
	HandleNetworkError(t, network, err, false)

	// create dht for querying
	bootstrapper, err := helpers.ParseAddrInfos(network.Logger, BOOTSTRAPPERS)
	HandleNetworkError(t, network, err, false)
	_, _, dht := qgbtesting.NewTestDHT(ctx, bootstrapper)
	defer dht.Close()

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE0ACCOUNTADDRESS)
	HandleNetworkError(t, network, err, false)

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE1ACCOUNTADDRESS)
	HandleNetworkError(t, network, err, false)

	// give the orchestrators some time to catchup
	time.Sleep(30 * time.Second)

	bridge, err := network.GetLatestDeployedQGBContract(ctx)
	HandleNetworkError(t, network, err, false)

	err = network.WaitForRelayerToStart(ctx, bridge)
	HandleNetworkError(t, network, err, false)

	evmClient := evm.NewClient(nil, bridge, nil, network.EVMRPC, evm.DEFAULTEVMGASLIMIT)

	dcNonce, err := evmClient.StateLastEventNonce(&bind.CallOpts{Context: ctx})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, dcNonce, uint64(2))
}

func TestRelayerWithMultipleValidators(t *testing.T) {
	if os.Getenv("QGB_INTEGRATION_TEST") != TRUE {
		t.Skip("Skipping QGB integration tests")
	}

	network, err := NewQGBNetwork()
	HandleNetworkError(t, network, err, false)

	// to release resources after tests
	defer network.DeleteAll() //nolint:errcheck

	// start full network with four validatorS
	err = network.StartAll()
	HandleNetworkError(t, network, err, false)

	ctx := context.Background()
	err = network.WaitForBlock(ctx, int64(2*network.DataCommitmentWindow))
	HandleNetworkError(t, network, err, false)

	// create dht for querying
	bootstrapper, err := helpers.ParseAddrInfos(network.Logger, BOOTSTRAPPERS)
	HandleNetworkError(t, network, err, false)
	_, _, dht := qgbtesting.NewTestDHT(ctx, bootstrapper)
	defer dht.Close()

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE0ACCOUNTADDRESS)
	HandleNetworkError(t, network, err, false)

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE1ACCOUNTADDRESS)
	HandleNetworkError(t, network, err, false)

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE2ACCOUNTADDRESS)
	HandleNetworkError(t, network, err, false)

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE3ACCOUNTADDRESS)
	HandleNetworkError(t, network, err, false)

	// give the orchestrators some time to catchup
	time.Sleep(30 * time.Second)

	// check whether the four validators are up and running
	qgbGRPC, err := grpc.Dial(network.CelestiaGRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	HandleNetworkError(t, network, err, false)
	defer qgbGRPC.Close()
	appQuerier := rpc.NewAppQuerier(network.Logger, qgbGRPC, network.EncCfg)
	HandleNetworkError(t, network, err, false)

	latestValset, err := appQuerier.QueryLatestValset(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(latestValset.Members))

	bridge, err := network.GetLatestDeployedQGBContract(ctx)
	HandleNetworkError(t, network, err, false)

	err = network.WaitForRelayerToStart(ctx, bridge)
	HandleNetworkError(t, network, err, false)

	evmClient := evm.NewClient(nil, bridge, nil, network.EVMRPC, evm.DEFAULTEVMGASLIMIT)

	dcNonce, err := evmClient.StateLastEventNonce(&bind.CallOpts{Context: ctx})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, dcNonce, uint64(2))
}
