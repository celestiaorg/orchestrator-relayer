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
		EthAddress: "celes1qktu8009djs6uym9uwj84ead24exkezsaqrmn5",
		Commitment: "test commitment",
		Signature:  "test signature",
	}

	// generate a test key for the DataCommitmentConfirm
	testKey := p2p.GetDataCommitmentConfirmKey(10, "celes1qktu8009djs6uym9uwj84ead24exkezsaqrmn5")

	// Broadcast the confirm
	broadcaster := orchestrator.NewBroadcaster(network.DHTs[1])
	err := broadcaster.BroadcastDataCommitmentConfirm(context.Background(), 10, expectedConfirm)
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
		EthAddress: "celes1qktu8009djs6uym9uwj84ead24exkezsaqrmn5",
		Signature:  "test signature",
	}

	// generate a test key for the ValsetConfirm
	testKey := p2p.GetValsetConfirmKey(10, "celes1qktu8009djs6uym9uwj84ead24exkezsaqrmn5")

	// Broadcast the confirm
	broadcaster := orchestrator.NewBroadcaster(network.DHTs[1])
	err := broadcaster.BroadcastValsetConfirm(context.Background(), 10, expectedConfirm)
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
		EthAddress: "celes1qktu8009djs6uym9uwj84ead24exkezsaqrmn5",
		Commitment: "test commitment",
		Signature:  "test signature",
	}

	// Broadcast the confirm
	broadcaster := orchestrator.NewBroadcaster(dht)
	err := broadcaster.BroadcastDataCommitmentConfirm(context.Background(), 10, dcConfirm)

	// check if the correct error is returned
	assert.Error(t, err)
	assert.Equal(t, orchestrator.ErrEmptyPeersTable, err)

	// try with a valset confirm
	vsConfirm := types.ValsetConfirm{
		EthAddress: "celes1qktu8009djs6uym9uwj84ead24exkezsaqrmn5",
		Signature:  "test signature",
	}

	// Broadcast the confirm
	err = broadcaster.BroadcastValsetConfirm(context.Background(), 10, vsConfirm)

	// check if the correct error is returned
	assert.Error(t, err)
	assert.Equal(t, orchestrator.ErrEmptyPeersTable, err)
}
