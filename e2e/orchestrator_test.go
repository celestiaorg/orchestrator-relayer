package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/helpers"

	qgbtesting "github.com/celestiaorg/orchestrator-relayer/testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrchestratorWithOneValidator(t *testing.T) {
	if os.Getenv("QGB_INTEGRATION_TEST") != TRUE {
		t.Skip("Skipping QGB integration tests")
	}

	network, err := NewQGBNetwork()
	HandleNetworkError(t, network, err, false)

	// to release resources after tests
	defer network.DeleteAll() //nolint:errcheck

	// start 1 validator
	err = network.StartBase()
	HandleNetworkError(t, network, err, false)

	// add orchestrator
	err = network.Start(Core0Orch)
	HandleNetworkError(t, network, err, false)

	ctx := context.Background()
	err = network.WaitForBlock(ctx, int64(network.DataCommitmentWindow+50))
	HandleNetworkError(t, network, err, false)

	// create dht for querying
	bootstrapper, err := helpers.ParseAddrInfos(network.Logger, BOOTSTRAPPERS)
	HandleNetworkError(t, network, err, false)
	_, _, dht := qgbtesting.NewTestDHT(ctx, bootstrapper)
	defer dht.Close()

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE0EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	// give the orchestrators some time to catchup
	time.Sleep(30 * time.Second)

	vsConfirm, err := network.GetValsetConfirm(ctx, dht, 1, CORE0EVMADDRESS)
	// assert the confirm exist
	assert.NoError(t, err)
	require.NotNil(t, vsConfirm)
	// assert that it carries the right evm address
	assert.Equal(t, CORE0EVMADDRESS, vsConfirm.EthAddress)

	dcConfirm, err := network.GetDataCommitmentConfirmByHeight(ctx, dht, network.DataCommitmentWindow-2, CORE0EVMADDRESS)
	// assert the confirm exist
	require.NoError(t, err)
	require.NotNil(t, dcConfirm)
	// assert that it carries the right evm address
	assert.Equal(t, CORE0EVMADDRESS, dcConfirm.EthAddress)
}

func TestOrchestratorWithTwoValidators(t *testing.T) {
	if os.Getenv("QGB_INTEGRATION_TEST") != TRUE {
		t.Skip("Skipping QGB integration tests")
	}

	network, err := NewQGBNetwork()
	HandleNetworkError(t, network, err, false)

	// to release resources after tests
	defer network.DeleteAll() //nolint:errcheck

	// start minimal network with one validator
	// start 1 validator
	err = network.StartBase()
	HandleNetworkError(t, network, err, false)

	// add core 0 orchestrator
	err = network.Start(Core0Orch)
	HandleNetworkError(t, network, err, false)

	// add core1 validator
	err = network.Start(Core1)
	HandleNetworkError(t, network, err, false)

	// add core1 orchestrator
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

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE0EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE1EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	// give the orchestrators some time to catchup
	time.Sleep(30 * time.Second)

	// check core0 submitted the valset confirm
	core0ValsetConfirm, err := network.GetValsetConfirm(ctx, dht, 1, CORE0EVMADDRESS)
	// assert the confirm exist
	assert.NoError(t, err)
	assert.NotNil(t, core0ValsetConfirm)
	// assert that it carries the right evm address
	assert.Equal(t, CORE0EVMADDRESS, core0ValsetConfirm.EthAddress)

	// check core0 submitted the data commitment confirm
	core0DataCommitmentConfirm, err := network.GetDataCommitmentConfirmByHeight(
		ctx,
		dht,
		network.DataCommitmentWindow-1,
		CORE0EVMADDRESS,
	)
	// assert the confirm exist
	require.NoError(t, err)
	require.NotNil(t, core0DataCommitmentConfirm)
	// assert that it carries the right evm address
	assert.Equal(t, CORE0EVMADDRESS, core0DataCommitmentConfirm.EthAddress)

	// get the last valset where all validators were created
	vs, err := network.GetValsetContainingVals(ctx, 2)
	require.NoError(t, err)
	require.NotNil(t, vs)

	// check core1 submitted the data commitment confirm
	core1Confirm, err := network.GetDataCommitmentConfirm(ctx, dht, vs.Nonce+1, CORE1EVMADDRESS)
	require.NoError(t, err)
	require.NotNil(t, core1Confirm)
}

func TestOrchestratorWithMultipleValidators(t *testing.T) {
	if os.Getenv("QGB_INTEGRATION_TEST") != TRUE {
		t.Skip("Skipping QGB integration tests")
	}

	network, err := NewQGBNetwork()
	assert.NoError(t, err)

	// to release resources after tests
	defer network.DeleteAll() //nolint:errcheck

	// start full network with four validatorS
	err = network.StartAll()
	HandleNetworkError(t, network, err, false)

	ctx := context.Background()

	err = network.WaitForBlock(ctx, int64(network.DataCommitmentWindow+50))
	HandleNetworkError(t, network, err, false)

	// create dht for querying
	bootstrapper, err := helpers.ParseAddrInfos(network.Logger, BOOTSTRAPPERS)
	HandleNetworkError(t, network, err, false)
	_, _, dht := qgbtesting.NewTestDHT(ctx, bootstrapper)
	defer dht.Close()

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE0EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE1EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE2EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE3EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	// give the orchestrators some time to catchup
	time.Sleep(30 * time.Second)

	// check core0 submitted the valset confirm
	core0ValsetConfirm, err := network.GetValsetConfirm(ctx, dht, 1, CORE0EVMADDRESS)
	// check the confirm exist
	require.NoError(t, err)
	require.NotNil(t, core0ValsetConfirm)
	// assert that it carries the right evm address
	assert.Equal(t, CORE0EVMADDRESS, core0ValsetConfirm.EthAddress)

	// check core0 submitted the data commitment confirm
	core0DataCommitmentConfirm, err := network.GetDataCommitmentConfirmByHeight(
		ctx,
		dht,
		network.DataCommitmentWindow-2,
		CORE0EVMADDRESS,
	)
	// check the confirm exist
	require.NoError(t, err)
	require.NotNil(t, core0DataCommitmentConfirm)
	// assert that it carries the right evm address
	assert.Equal(t, CORE0EVMADDRESS, core0DataCommitmentConfirm.EthAddress)

	// get the last valset where all validators were created
	vs, err := network.GetValsetContainingVals(ctx, 4)
	require.NoError(t, err)
	require.NotNil(t, vs)

	// check core1 submitted the data commitment confirm
	core1Confirm, err := network.GetDataCommitmentConfirm(ctx, dht, vs.Nonce+1, CORE1EVMADDRESS)
	require.NoError(t, err)
	require.NotNil(t, core1Confirm)

	// check core2 submitted the data commitment confirm
	core2Confirm, err := network.GetDataCommitmentConfirm(ctx, dht, vs.Nonce+1, CORE2EVMADDRESS)
	require.NoError(t, err)
	require.NotNil(t, core2Confirm)

	// check core3 submitted the data commitment confirm
	core3Confirm, err := network.GetDataCommitmentConfirm(ctx, dht, vs.Nonce+1, CORE3EVMADDRESS)
	require.NoError(t, err)
	require.NotNil(t, core3Confirm)
}

func TestOrchestratorReplayOld(t *testing.T) {
	if os.Getenv("QGB_INTEGRATION_TEST") != TRUE {
		t.Skip("Skipping QGB integration tests")
	}

	network, err := NewQGBNetwork()
	HandleNetworkError(t, network, err, false)

	// to release resources after tests
	defer network.DeleteAll() //nolint:errcheck

	// start 1 validator
	err = network.StartBase()
	HandleNetworkError(t, network, err, false)

	// add core1 validator
	err = network.Start(Core1)
	HandleNetworkError(t, network, err, false)

	ctx := context.Background()

	err = network.WaitForBlock(ctx, int64(2*network.DataCommitmentWindow))
	HandleNetworkError(t, network, err, false)

	// add core0 orchestrator
	err = network.Start(Core0Orch)
	HandleNetworkError(t, network, err, false)

	// add core1 orchestrator
	err = network.Start(Core1Orch)
	HandleNetworkError(t, network, err, false)

	// give time for the orchestrators to submit confirms
	err = network.WaitForBlock(ctx, int64(2*network.DataCommitmentWindow+50))
	HandleNetworkError(t, network, err, false)

	// create dht for querying
	bootstrapper, err := helpers.ParseAddrInfos(network.Logger, BOOTSTRAPPERS)
	HandleNetworkError(t, network, err, false)
	_, _, dht := qgbtesting.NewTestDHT(ctx, bootstrapper)
	defer dht.Close()

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE0EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	err = network.WaitForOrchestratorToStart(ctx, dht, CORE1EVMADDRESS)
	HandleNetworkError(t, network, err, false)

	// give the orchestrators some time to catchup
	time.Sleep(30 * time.Second)

	// check core0 submitted valset 1 confirm
	vs1Core0Confirm, err := network.GetValsetConfirm(ctx, dht, 1, CORE0EVMADDRESS)
	// assert the confirm exist
	require.NoError(t, err)
	require.NotNil(t, vs1Core0Confirm)
	// assert that it carries the right evm address
	assert.Equal(t, CORE0EVMADDRESS, vs1Core0Confirm.EthAddress)

	// get the last valset where all validators were created
	vs, err := network.GetValsetContainingVals(ctx, 2)
	require.NoError(t, err)
	require.NotNil(t, vs)

	latestNonce, err := network.GetLatestAttestationNonce(ctx)
	require.NoError(t, err)

	// checks that all nonces where all validators were part of the valset were signed
	for i := vs.Nonce + 1; i <= latestNonce; i++ {
		// check core1 submitted the attestation confirm
		wasSigned, err := network.WasAttestationSigned(ctx, dht, i, CORE1EVMADDRESS)
		require.NoError(t, err)
		require.True(t, wasSigned)
	}
}
