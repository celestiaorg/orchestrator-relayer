package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/orchestrator-relayer/helpers"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	qgbtesting "github.com/celestiaorg/orchestrator-relayer/testing"
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
	window, err := network.GetCurrentDataCommitmentWindow(ctx)
	require.NoError(t, err)
	err = network.WaitForBlock(ctx, int64(window+50))
	HandleNetworkError(t, network, err, false)

	// create dht for querying
	bootstrapper, err := helpers.ParseAddrInfos(network.Logger, BOOTSTRAPPERS)
	HandleNetworkError(t, network, err, false)
	host, _, dht := qgbtesting.NewTestDHT(ctx, bootstrapper)
	defer dht.Close()

	// force the connection to the DHT to start the orchestrator
	err = ConnectToDHT(ctx, host, dht, bootstrapper[0])
	HandleNetworkError(t, network, err, false)

	_, _, err = network.WaitForOrchestratorToStart(ctx, dht, CORE0EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	bridge, err := network.GetLatestDeployedQGBContract(ctx)
	HandleNetworkError(t, network, err, false)

	latestNonce, err := network.GetLatestAttestationNonce(ctx)
	require.NoError(t, err)

	err = network.WaitForRelayerToStart(ctx, bridge)
	HandleNetworkError(t, network, err, false)

	evmClient := evm.NewClient(nil, bridge, nil, nil, network.EVMRPC, evm.DefaultEVMGasLimit)

	err = network.WaitForEventNonce(ctx, bridge, latestNonce)
	HandleNetworkError(t, network, err, false)
	eventNonce, err := evmClient.StateLastEventNonce(&bind.CallOpts{Context: ctx})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, eventNonce, latestNonce)
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
	window, err := network.GetCurrentDataCommitmentWindow(ctx)
	require.NoError(t, err)
	err = network.WaitForBlock(ctx, int64(window+50))
	HandleNetworkError(t, network, err, false)

	// create dht for querying
	bootstrapper, err := helpers.ParseAddrInfos(network.Logger, BOOTSTRAPPERS)
	HandleNetworkError(t, network, err, false)
	host, _, dht := qgbtesting.NewTestDHT(ctx, bootstrapper)
	defer dht.Close()

	// force the connection to the DHT to start the orchestrator
	err = ConnectToDHT(ctx, host, dht, bootstrapper[0])
	HandleNetworkError(t, network, err, false)

	_, _, err = network.WaitForOrchestratorToStart(ctx, dht, CORE0EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	_, _, err = network.WaitForOrchestratorToStart(ctx, dht, CORE1EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	// give the orchestrators some time to catchup
	time.Sleep(time.Second)

	bridge, err := network.GetLatestDeployedQGBContract(ctx)
	HandleNetworkError(t, network, err, false)

	err = network.WaitForRelayerToStart(ctx, bridge)
	HandleNetworkError(t, network, err, false)

	evmClient := evm.NewClient(nil, bridge, nil, nil, network.EVMRPC, evm.DefaultEVMGasLimit)

	latestNonce, err := network.GetLatestAttestationNonce(ctx)
	require.NoError(t, err)

	err = network.WaitForEventNonce(ctx, bridge, latestNonce)
	HandleNetworkError(t, network, err, false)
	dcNonce, err := evmClient.StateLastEventNonce(&bind.CallOpts{Context: ctx})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, dcNonce, latestNonce)
}

func TestRelayerWithMultipleValidators(t *testing.T) {
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
	window, err := network.GetCurrentDataCommitmentWindow(ctx)
	require.NoError(t, err)
	err = network.WaitForBlock(ctx, int64(2*window))
	HandleNetworkError(t, network, err, false)

	// create dht for querying
	bootstrapper, err := helpers.ParseAddrInfos(network.Logger, BOOTSTRAPPERS)
	HandleNetworkError(t, network, err, false)
	host, _, dht := qgbtesting.NewTestDHT(ctx, bootstrapper)
	defer dht.Close()

	// force the connection to the DHT to start the orchestrator
	err = ConnectToDHT(ctx, host, dht, bootstrapper[0])
	HandleNetworkError(t, network, err, false)

	_, _, err = network.WaitForOrchestratorToStart(ctx, dht, CORE0EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	_, _, err = network.WaitForOrchestratorToStart(ctx, dht, CORE1EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	_, _, err = network.WaitForOrchestratorToStart(ctx, dht, CORE2EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	_, _, err = network.WaitForOrchestratorToStart(ctx, dht, CORE3EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	// give the orchestrators some time to catchup
	time.Sleep(time.Second)

	// check whether the four validators are up and running
	appQuerier := rpc.NewAppQuerier(network.Logger, network.CelestiaGRPC, network.EncCfg)
	HandleNetworkError(t, network, err, false)
	err = appQuerier.Start()
	HandleNetworkError(t, network, err, false)
	defer appQuerier.Stop() //nolint:errcheck

	latestValset, err := appQuerier.QueryLatestValset(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(latestValset.Members))

	bridge, err := network.GetLatestDeployedQGBContract(ctx)
	HandleNetworkError(t, network, err, false)

	err = network.WaitForRelayerToStart(ctx, bridge)
	HandleNetworkError(t, network, err, false)

	evmClient := evm.NewClient(nil, bridge, nil, nil, network.EVMRPC, evm.DefaultEVMGasLimit)

	err = network.WaitForEventNonce(ctx, bridge, latestValset.Nonce)
	HandleNetworkError(t, network, err, false)
	dcNonce, err := evmClient.StateLastEventNonce(&bind.CallOpts{Context: ctx})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, dcNonce, latestValset.Nonce)
}

func TestUpdatingTheDataCommitmentWindow(t *testing.T) {
	if os.Getenv("BLOBSTREAM_DATA_COMMITMENT_UPDATE_TEST") != TRUE {
		t.Skip("Skipping Blobstream data commitment update integration test")
	}

	network, err := NewQGBNetwork()
	HandleNetworkError(t, network, err, false)

	// to release resources after tests
	defer network.DeleteAll() //nolint:errcheck

	// start full network with four validators
	err = network.StartAll()
	HandleNetworkError(t, network, err, false)

	ctx := context.Background()
	window, err := network.GetCurrentDataCommitmentWindow(ctx)
	require.NoError(t, err)
	err = network.WaitForBlock(ctx, int64(window))
	HandleNetworkError(t, network, err, false)

	// update the data commitment window to 200
	err = network.UpdateDataCommitmentWindow(ctx, 200)
	require.NoError(t, err)
	err = network.WaitForBlock(ctx, int64(window+200+100))
	require.NoError(t, err)

	// shrink the data commitment window to 150
	err = network.UpdateDataCommitmentWindow(ctx, 150)
	require.NoError(t, err)
	err = network.WaitForBlock(ctx, int64(window+200+150+150+50))
	require.NoError(t, err)

	nonceAfterTheWindowChanges, err := network.GetLatestAttestationNonce(ctx)
	require.NoError(t, err)

	// create dht for querying
	bootstrapper, err := helpers.ParseAddrInfos(network.Logger, BOOTSTRAPPERS)
	HandleNetworkError(t, network, err, false)
	host, _, dht := qgbtesting.NewTestDHT(ctx, bootstrapper)
	defer dht.Close()

	// force the connection to the DHT to start the orchestrator
	err = ConnectToDHT(ctx, host, dht, bootstrapper[0])
	HandleNetworkError(t, network, err, false)

	_, _, err = network.WaitForOrchestratorToStart(ctx, dht, CORE0EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	_, _, err = network.WaitForOrchestratorToStart(ctx, dht, CORE1EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	_, _, err = network.WaitForOrchestratorToStart(ctx, dht, CORE2EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	_, _, err = network.WaitForOrchestratorToStart(ctx, dht, CORE3EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	// give the orchestrators some time to catchup
	time.Sleep(time.Second)

	// check whether the four validators are up and running
	appQuerier := rpc.NewAppQuerier(network.Logger, network.CelestiaGRPC, network.EncCfg)
	HandleNetworkError(t, network, err, false)
	err = appQuerier.Start()
	HandleNetworkError(t, network, err, false)
	defer appQuerier.Stop() //nolint:errcheck

	latestValset, err := appQuerier.QueryLatestValset(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 4, len(latestValset.Members))

	bridge, err := network.GetLatestDeployedQGBContract(ctx)
	HandleNetworkError(t, network, err, false)

	err = network.WaitForRelayerToStart(ctx, bridge)
	HandleNetworkError(t, network, err, false)

	evmClient := evm.NewClient(nil, bridge, nil, nil, network.EVMRPC, evm.DefaultEVMGasLimit)

	err = network.WaitForEventNonce(ctx, bridge, nonceAfterTheWindowChanges)
	HandleNetworkError(t, network, err, false)
	dcNonce, err := evmClient.StateLastEventNonce(&bind.CallOpts{Context: ctx})
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, dcNonce, nonceAfterTheWindowChanges)
}
