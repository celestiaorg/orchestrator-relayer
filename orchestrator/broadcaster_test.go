package orchestrator_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/orchestrator-relayer/orchestrator"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	qgbtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	"github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/stretchr/testify/assert"
)

func TestBroadcastDataCommitmentConfirm(t *testing.T) {
	network := qgbtesting.NewDHTNetwork(context.Background(), 4)
	defer network.Stop()

	// create a test DataCommitmentConfirm
	expectedConfirm := types.DataCommitmentConfirm{
		EthAddress: "0x7c2B12b5a07FC6D719Ed7646e5041A7E85758329",
		Commitment: "test commitment",
		Signature:  "test signature",
	}

	// generate a test key for the DataCommitmentConfirm
	testKey := p2p.GetDataCommitmentConfirmKey(10, "0x7c2B12b5a07FC6D719Ed7646e5041A7E85758329")

	// Broadcast the confirm
	broadcaster := orchestrator.NewBroadcaster(network.DHTs[1])
	err := broadcaster.ProvideDataCommitmentConfirm(context.Background(), 10, expectedConfirm)
	assert.NoError(t, err)

	// try to get the confirm from another peer
	actualConfirm, err := network.DHTs[3].GetDataCommitmentConfirm(context.Background(), testKey)
	assert.NoError(t, err)
	assert.NotNil(t, actualConfirm)

	assert.Equal(t, expectedConfirm, actualConfirm)
}

func TestBroadcastValsetConfirm(t *testing.T) {
	network := qgbtesting.NewDHTNetwork(context.Background(), 4)
	defer network.Stop()

	// create a test DataCommitmentConfirm
	expectedConfirm := types.ValsetConfirm{
		EthAddress: "0x7c2B12b5a07FC6D719Ed7646e5041A7E85758329",
		Signature:  "test signature",
	}

	// generate a test key for the ValsetConfirm
	testKey := p2p.GetValsetConfirmKey(10, "0x7c2B12b5a07FC6D719Ed7646e5041A7E85758329")

	// Broadcast the confirm
	broadcaster := orchestrator.NewBroadcaster(network.DHTs[1])
	err := broadcaster.ProvideValsetConfirm(context.Background(), 10, expectedConfirm)
	assert.NoError(t, err)

	// try to get the confirm from another peer
	actualConfirm, err := network.DHTs[3].GetValsetConfirm(context.Background(), testKey)
	assert.NoError(t, err)
	assert.NotNil(t, actualConfirm)

	assert.Equal(t, expectedConfirm, actualConfirm)
}

// TestEmptyPeersTable tests that values are not broadcasted if the DHT peers
// table is empty.
func TestEmptyPeersTable(t *testing.T) {
	_, _, dht := qgbtesting.NewTestDHT(context.Background())
	defer func(dht *p2p.QgbDHT) {
		err := dht.Close()
		if err != nil {
			require.NoError(t, err)
		}
	}(dht)

	// create a test DataCommitmentConfirm
	dcConfirm := types.DataCommitmentConfirm{
		EthAddress: "0x7c2B12b5a07FC6D719Ed7646e5041A7E85758329",
		Commitment: "test commitment",
		Signature:  "test signature",
	}

	// Broadcast the confirm
	broadcaster := orchestrator.NewBroadcaster(dht)
	err := broadcaster.ProvideDataCommitmentConfirm(context.Background(), 10, dcConfirm)

	// check if the correct error is returned
	assert.Error(t, err)
	assert.Equal(t, orchestrator.ErrEmptyPeersTable, err)

	// try with a valset confirm
	vsConfirm := types.ValsetConfirm{
		EthAddress: "0x7c2B12b5a07FC6D719Ed7646e5041A7E85758329",
		Signature:  "test signature",
	}

	// Broadcast the confirm
	err = broadcaster.ProvideValsetConfirm(context.Background(), 10, vsConfirm)

	// check if the correct error is returned
	assert.Error(t, err)
	assert.Equal(t, orchestrator.ErrEmptyPeersTable, err)
}
