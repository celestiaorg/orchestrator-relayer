package p2p_test

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/orchestrator-relayer/p2p"
	qgbtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	"github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/stretchr/testify/assert"
)

func TestPutDataCommitmentConfirm(t *testing.T) {
	network := qgbtesting.NewDHTNetwork(context.Background(), 2)
	defer network.Stop()

	// create a test DataCommitmentConfirm
	expectedConfirm := types.DataCommitmentConfirm{
		EthAddress: "test address",
		Commitment: "test commitment",
		Signature:  "test signature",
	}

	// generate a test key for the DataCommitmentConfirm
	testKey := p2p.GetDataCommitmentConfirmKey(10, "0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b")

	// put the test DataCommitmentConfirm in the DHT
	err := network.DHTs[0].PutDataCommitmentConfirm(context.Background(), testKey, expectedConfirm)
	assert.NoError(t, err)

	// try to get the confirm from the same peer
	actualConfirm, err := network.DHTs[0].GetDataCommitmentConfirm(context.Background(), testKey)
	assert.NoError(t, err)
	assert.NotNil(t, actualConfirm)

	assert.Equal(t, expectedConfirm, actualConfirm)
}

func TestNetworkPutDataCommitmentConfirm(t *testing.T) {
	network := qgbtesting.NewDHTNetwork(context.Background(), 10)
	defer network.Stop()

	// create a test DataCommitmentConfirm
	expectedConfirm := types.DataCommitmentConfirm{
		EthAddress: "test address",
		Commitment: "test commitment",
		Signature:  "test signature",
	}

	// generate a test key for the DataCommitmentConfirm
	testKey := p2p.GetDataCommitmentConfirmKey(10, "0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b")

	// put the test DataCommitmentConfirm in the DHT
	err := network.DHTs[2].PutDataCommitmentConfirm(context.Background(), testKey, expectedConfirm)
	assert.NoError(t, err)

	// try to get the DataCommitmentConfirm from another peer
	actualConfirm, err := network.DHTs[8].GetDataCommitmentConfirm(context.Background(), testKey)
	assert.NoError(t, err)
	assert.NotNil(t, actualConfirm)

	assert.Equal(t, expectedConfirm, actualConfirm)
}

func TestNetworkGetNonExistentDataCommitmentConfirm(t *testing.T) {
	network := qgbtesting.NewDHTNetwork(context.Background(), 10)
	defer network.Stop()

	// generate a test key for the DataCommitmentConfirm
	testKey := p2p.GetDataCommitmentConfirmKey(10, "0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b")

	// try to get the non-existent DataCommitmentConfirm
	actualConfirm, err := network.DHTs[8].GetDataCommitmentConfirm(context.Background(), testKey)
	assert.Error(t, err)
	assert.True(t, types.IsEmptyMsgDataCommitmentConfirm(actualConfirm))
}

func TestPutValsetConfirm(t *testing.T) {
	network := qgbtesting.NewDHTNetwork(context.Background(), 2)
	defer network.Stop()

	// create a test ValsetConfirm
	expectedConfirm := types.ValsetConfirm{
		EthAddress: "test address",
		Signature:  "test signature",
	}

	// generate a test key for the ValsetConfirm
	testKey := p2p.GetValsetConfirmKey(10, "0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b")

	// put the test ValsetConfirm in the DHT
	err := network.DHTs[0].PutValsetConfirm(context.Background(), testKey, expectedConfirm)
	assert.NoError(t, err)

	// try to get the ValsetConfirm from the same peer
	actualConfirm, err := network.DHTs[0].GetValsetConfirm(context.Background(), testKey)
	assert.NoError(t, err)
	assert.NotNil(t, actualConfirm)

	assert.Equal(t, expectedConfirm, actualConfirm)
}

func TestNetworkPutValsetConfirm(t *testing.T) {
	network := qgbtesting.NewDHTNetwork(context.Background(), 10)
	defer network.Stop()

	// create a test ValsetConfirm
	expectedConfirm := types.ValsetConfirm{
		EthAddress: "test address",
		Signature:  "test signature",
	}

	// generate a test key for the DataCommitmentConfirm
	testKey := p2p.GetValsetConfirmKey(10, "0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b")

	// put the test DataCommitmentConfirm in the DHT
	err := network.DHTs[2].PutValsetConfirm(context.Background(), testKey, expectedConfirm)
	assert.NoError(t, err)

	// try to get the DataCommitmentConfirm from another peer
	actualConfirm, err := network.DHTs[8].GetValsetConfirm(context.Background(), testKey)
	assert.NoError(t, err)
	assert.NotNil(t, actualConfirm)

	assert.Equal(t, expectedConfirm, actualConfirm)
}

func TestNetworkGetNonExistentValsetConfirm(t *testing.T) {
	network := qgbtesting.NewDHTNetwork(context.Background(), 10)
	defer network.Stop()

	// generate a test key for the ValsetConfirm
	testKey := p2p.GetDataCommitmentConfirmKey(10, "0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b")

	// try to get the non-existent ValsetConfirm
	actualConfirm, err := network.DHTs[8].GetValsetConfirm(context.Background(), testKey)
	assert.Error(t, err)
	assert.True(t, types.IsEmptyValsetConfirm(actualConfirm))
}

func TestWaitForPeers(t *testing.T) {
	ctx := context.Background()
	// create first dht
	h1, _, dht1 := qgbtesting.NewTestDHT(ctx)
	defer dht1.Close()

	// wait for peers
	err := dht1.WaitForPeers(ctx, 10*time.Millisecond, time.Millisecond, 1)
	// should error because no peer is connected to this dht
	assert.Error(t, err)

	// create second dht
	h2, _, dht2 := qgbtesting.NewTestDHT(ctx)
	defer dht2.Close()
	// connect to first dht
	err = h2.Connect(ctx, peer.AddrInfo{
		ID:    h1.ID(),
		Addrs: h1.Addrs(),
	})
	require.NoError(t, err)

	// wait for peers
	err = dht1.WaitForPeers(ctx, 10*time.Millisecond, time.Millisecond, 1)
	assert.NoError(t, err)
}
