package p2p_test

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/ethereum/go-ethereum/common"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/celestiaorg/orchestrator-relayer/p2p"
	blobstreamtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	"github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	evmAddress    = "0x966e6f22781EF6a6A82BBB4DB3df8E225DfD9488"
	privateKey, _ = ethcrypto.HexToECDSA("da6ed55cb2894ac2c9c10209c09de8e8b9d109b910338d5bf3d747a7e1fc9eb9")
)

func TestDHTBootstrappers(t *testing.T) {
	ctx := context.Background()
	// create first dht
	h1, _, dht1 := blobstreamtesting.NewTestDHT(ctx, nil)
	defer dht1.Close()

	// create second dht with dht1 being a bootstrapper
	h2, _, dht2 := blobstreamtesting.NewTestDHT(
		ctx,
		[]peer.AddrInfo{{
			ID:    h1.ID(),
			Addrs: h1.Addrs(),
		}},
	)

	// give some time for the routing table to be updated
	err := dht1.WaitForPeers(ctx, 5*time.Second, time.Millisecond, 1)
	require.NoError(t, err)
	err = dht2.WaitForPeers(ctx, 5*time.Second, time.Millisecond, 1)
	require.NoError(t, err)

	// check if connected
	require.NotEmpty(t, dht1.RoutingTable().ListPeers())
	require.NotEmpty(t, dht2.RoutingTable().ListPeers())
	assert.Equal(t, dht2.RoutingTable().ListPeers()[0].String(), h1.ID().String())
	assert.NotEmpty(t, dht2.RoutingTable().ListPeers())
	assert.Equal(t, dht1.RoutingTable().ListPeers()[0].String(), h2.ID().String())
	assert.NotEmpty(t, dht1.RoutingTable().ListPeers())
}

func TestPutDataCommitmentConfirm(t *testing.T) {
	network := blobstreamtesting.NewDHTNetwork(context.Background(), 2)
	defer network.Stop()

	ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.ImportECDSA(privateKey, "123")
	require.NoError(t, err)
	err = ks.Unlock(acc, "123")
	require.NoError(t, err)

	nonce := uint64(10)
	commitment := "1234"
	bCommitment, _ := hex.DecodeString(commitment)
	dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(nonce)), bCommitment)
	signature, err := evm.NewEthereumSignature(dataRootHash.Bytes(), ks, acc)
	require.NoError(t, err)

	// create a test DataCommitmentConfirm
	expectedConfirm := types.DataCommitmentConfirm{
		EthAddress: evmAddress,
		Signature:  hex.EncodeToString(signature),
	}

	// generate a test key for the DataCommitmentConfirm
	testKey := p2p.GetDataCommitmentConfirmKey(nonce, evmAddress, dataRootHash.Hex())

	// put the test DataCommitmentConfirm in the DHT
	err = network.DHTs[0].PutDataCommitmentConfirm(context.Background(), testKey, expectedConfirm)
	assert.NoError(t, err)

	// try to get the confirm from the same peer
	actualConfirm, err := network.DHTs[0].GetDataCommitmentConfirm(context.Background(), testKey)
	assert.NoError(t, err)
	assert.NotNil(t, actualConfirm)

	assert.Equal(t, expectedConfirm, actualConfirm)
}

func TestNetworkPutDataCommitmentConfirm(t *testing.T) {
	network := blobstreamtesting.NewDHTNetwork(context.Background(), 10)
	defer network.Stop()

	ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.ImportECDSA(privateKey, "123")
	require.NoError(t, err)
	err = ks.Unlock(acc, "123")
	require.NoError(t, err)

	nonce := uint64(10)
	commitment := "1234"
	bCommitment, _ := hex.DecodeString(commitment)
	dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(nonce)), bCommitment)
	signature, err := evm.NewEthereumSignature(dataRootHash.Bytes(), ks, acc)
	require.NoError(t, err)

	// create a test DataCommitmentConfirm
	expectedConfirm := types.DataCommitmentConfirm{
		EthAddress: evmAddress,
		Signature:  hex.EncodeToString(signature),
	}

	// generate a test key for the DataCommitmentConfirm
	testKey := p2p.GetDataCommitmentConfirmKey(nonce, evmAddress, dataRootHash.Hex())

	// put the test DataCommitmentConfirm in the DHT
	err = network.DHTs[2].PutDataCommitmentConfirm(context.Background(), testKey, expectedConfirm)
	assert.NoError(t, err)

	// try to get the DataCommitmentConfirm from another peer
	actualConfirm, err := network.DHTs[8].GetDataCommitmentConfirm(context.Background(), testKey)
	assert.NoError(t, err)
	assert.NotNil(t, actualConfirm)

	assert.Equal(t, expectedConfirm, actualConfirm)
}

func TestNetworkGetNonExistentDataCommitmentConfirm(t *testing.T) {
	network := blobstreamtesting.NewDHTNetwork(context.Background(), 10)
	defer network.Stop()

	// generate a test key for the DataCommitmentConfirm
	testKey := p2p.GetDataCommitmentConfirmKey(10, evmAddress, "0x1234")

	// try to get the non-existent DataCommitmentConfirm
	actualConfirm, err := network.DHTs[8].GetDataCommitmentConfirm(context.Background(), testKey)
	assert.Error(t, err)
	assert.True(t, types.IsEmptyMsgDataCommitmentConfirm(actualConfirm))
}

func TestPutValsetConfirm(t *testing.T) {
	network := blobstreamtesting.NewDHTNetwork(context.Background(), 2)
	defer network.Stop()

	ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.ImportECDSA(privateKey, "123")
	require.NoError(t, err)
	err = ks.Unlock(acc, "123")
	require.NoError(t, err)

	signBytes := common.HexToHash("1234")
	signature, err := evm.NewEthereumSignature(signBytes.Bytes(), ks, acc)
	require.NoError(t, err)

	// create a test ValsetConfirm
	expectedConfirm := types.NewValsetConfirm(
		common.HexToAddress(evmAddress),
		hex.EncodeToString(signature),
	)

	// generate a test key for the ValsetConfirm
	testKey := p2p.GetValsetConfirmKey(10, evmAddress, signBytes.Hex())

	// put the test ValsetConfirm in the DHT
	err = network.DHTs[0].PutValsetConfirm(context.Background(), testKey, *expectedConfirm)
	assert.NoError(t, err)

	// try to get the ValsetConfirm from the same peer
	actualConfirm, err := network.DHTs[0].GetValsetConfirm(context.Background(), testKey)
	assert.NoError(t, err)
	assert.NotNil(t, actualConfirm)

	assert.Equal(t, *expectedConfirm, actualConfirm)
}

func TestNetworkPutValsetConfirm(t *testing.T) {
	network := blobstreamtesting.NewDHTNetwork(context.Background(), 10)
	defer network.Stop()

	ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.ImportECDSA(privateKey, "123")
	require.NoError(t, err)
	err = ks.Unlock(acc, "123")
	require.NoError(t, err)

	signBytes := common.HexToHash("1234")
	signature, err := evm.NewEthereumSignature(signBytes.Bytes(), ks, acc)
	require.NoError(t, err)

	// create a test ValsetConfirm
	expectedConfirm := types.NewValsetConfirm(
		common.HexToAddress(evmAddress),
		hex.EncodeToString(signature),
	)

	// generate a test key for the DataCommitmentConfirm
	testKey := p2p.GetValsetConfirmKey(10, evmAddress, signBytes.Hex())

	// put the test DataCommitmentConfirm in the DHT
	err = network.DHTs[2].PutValsetConfirm(context.Background(), testKey, *expectedConfirm)
	assert.NoError(t, err)

	// try to get the DataCommitmentConfirm from another peer
	actualConfirm, err := network.DHTs[8].GetValsetConfirm(context.Background(), testKey)
	assert.NoError(t, err)
	assert.NotNil(t, actualConfirm)

	assert.Equal(t, *expectedConfirm, actualConfirm)
}

func TestNetworkGetNonExistentValsetConfirm(t *testing.T) {
	network := blobstreamtesting.NewDHTNetwork(context.Background(), 10)
	defer network.Stop()

	// generate a test key for the ValsetConfirm
	testKey := p2p.GetDataCommitmentConfirmKey(10, evmAddress, "0x1234")

	// try to get the non-existent ValsetConfirm
	actualConfirm, err := network.DHTs[8].GetValsetConfirm(context.Background(), testKey)
	assert.Error(t, err)
	assert.True(t, types.IsEmptyValsetConfirm(actualConfirm))
}

func TestWaitForPeers(t *testing.T) {
	ctx := context.Background()
	// create first dht
	h1, _, dht1 := blobstreamtesting.NewTestDHT(ctx, nil)
	defer dht1.Close()

	// wait for peers
	err := dht1.WaitForPeers(ctx, 10*time.Millisecond, time.Millisecond, 1)
	// should error because no peer is connected to this dht
	assert.Error(t, err)

	// create second dht
	h2, _, dht2 := blobstreamtesting.NewTestDHT(ctx, nil)
	defer dht2.Close()
	// connect to first dht
	err = h2.Connect(ctx, peer.AddrInfo{
		ID:    h1.ID(),
		Addrs: h1.Addrs(),
	})
	require.NoError(t, err)

	// wait for peers
	err = dht1.WaitForPeers(ctx, 100*time.Millisecond, time.Millisecond, 1)
	assert.NoError(t, err)
}
