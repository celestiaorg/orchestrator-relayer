package orchestrator_test

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/ethereum/go-ethereum/common"

	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/orchestrator-relayer/orchestrator"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	qgbtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	"github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/stretchr/testify/assert"
)

var (
	evmAddress    = "0x966e6f22781EF6a6A82BBB4DB3df8E225DfD9488"
	privateKey, _ = ethcrypto.HexToECDSA("da6ed55cb2894ac2c9c10209c09de8e8b9d109b910338d5bf3d747a7e1fc9eb9")
)

func TestBroadcastDataCommitmentConfirm(t *testing.T) {
	network := qgbtesting.NewDHTNetwork(context.Background(), 4)
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
	expectedConfirm := types.NewDataCommitmentConfirm(hex.EncodeToString(signature), common.HexToAddress(evmAddress))

	// generate a test key for the DataCommitmentConfirm
	testKey := p2p.GetDataCommitmentConfirmKey(nonce, evmAddress, dataRootHash.Hex())

	// Broadcast the confirm
	broadcaster := orchestrator.NewBroadcaster(network.DHTs[1])
	err = broadcaster.ProvideDataCommitmentConfirm(context.Background(), nonce, *expectedConfirm, dataRootHash.Hex())
	assert.NoError(t, err)

	// try to get the confirm from another peer
	actualConfirm, err := network.DHTs[3].GetDataCommitmentConfirm(context.Background(), testKey)
	assert.NoError(t, err)
	assert.NotNil(t, actualConfirm)

	assert.Equal(t, *expectedConfirm, actualConfirm)
}

func TestBroadcastValsetConfirm(t *testing.T) {
	network := qgbtesting.NewDHTNetwork(context.Background(), 4)
	defer network.Stop()

	ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.ImportECDSA(privateKey, "123")
	require.NoError(t, err)
	err = ks.Unlock(acc, "123")
	require.NoError(t, err)

	nonce := uint64(10)
	signBytes := common.HexToHash("1234")
	signature, err := evm.NewEthereumSignature(signBytes.Bytes(), ks, acc)
	require.NoError(t, err)

	// create a test DataCommitmentConfirm
	expectedConfirm := types.NewValsetConfirm(common.HexToAddress(evmAddress), hex.EncodeToString(signature))

	// generate a test key for the ValsetConfirm
	testKey := p2p.GetValsetConfirmKey(nonce, evmAddress, signBytes.Hex())

	// Broadcast the confirm
	broadcaster := orchestrator.NewBroadcaster(network.DHTs[1])
	err = broadcaster.ProvideValsetConfirm(context.Background(), nonce, *expectedConfirm, signBytes.Hex())
	assert.NoError(t, err)

	// try to get the confirm from another peer
	actualConfirm, err := network.DHTs[3].GetValsetConfirm(context.Background(), testKey)
	assert.NoError(t, err)
	assert.NotNil(t, actualConfirm)

	assert.Equal(t, *expectedConfirm, actualConfirm)
}

// TestEmptyPeersTable tests that values are not broadcasted if the DHT peers
// table is empty.
func TestEmptyPeersTable(t *testing.T) {
	_, _, dht := qgbtesting.NewTestDHT(context.Background(), nil)
	defer func(dht *p2p.QgbDHT) {
		err := dht.Close()
		if err != nil {
			require.NoError(t, err)
		}
	}(dht)

	// create a test DataCommitmentConfirm
	dcConfirm := types.DataCommitmentConfirm{
		EthAddress: evmAddress,
		Signature:  "test signature",
	}

	// Broadcast the confirm
	broadcaster := orchestrator.NewBroadcaster(dht)
	err := broadcaster.ProvideDataCommitmentConfirm(context.Background(), 10, dcConfirm, "test root")

	// check if the correct error is returned
	assert.Error(t, err)
	assert.Equal(t, orchestrator.ErrEmptyPeersTable, err)

	// try with a valset confirm
	vsConfirm := types.ValsetConfirm{
		EthAddress: evmAddress,
		Signature:  "test signature",
	}

	// Broadcast the confirm
	err = broadcaster.ProvideValsetConfirm(context.Background(), 10, vsConfirm, "test root")

	// check if the correct error is returned
	assert.Error(t, err)
	assert.Equal(t, orchestrator.ErrEmptyPeersTable, err)
}
