package orchestrator_test

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/celestiaorg/orchestrator-relayer/evm"
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

	nonce := uint64(10)
	commitment := "1234"
	bCommitment, _ := hex.DecodeString(commitment)
	dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(nonce)), bCommitment)
	signature, err := evm.NewEthereumSignature(dataRootHash.Bytes(), privateKey)
	require.NoError(t, err)

	// create a test DataCommitmentConfirm
	expectedConfirm := types.DataCommitmentConfirm{
		EthAddress: evmAddress,
		Commitment: commitment,
		Signature:  hex.EncodeToString(signature),
	}

	// generate a test key for the DataCommitmentConfirm
	testKey := p2p.GetDataCommitmentConfirmKey(nonce, evmAddress)

	// Broadcast the confirm
	broadcaster := orchestrator.NewBroadcaster(network.DHTs[1])
	err = broadcaster.ProvideDataCommitmentConfirm(context.Background(), nonce, expectedConfirm)
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
		EthAddress: evmAddress,
		Signature:  "0xca2aa01f5b32722238e8f45356878e2cfbdc7c3335fbbf4e1dc3dfc53465e3e137103769d6956414014ae340cc4cb97384b2980eea47942f135931865471031a00",
	}

	// generate a test key for the ValsetConfirm
	testKey := p2p.GetValsetConfirmKey(10, evmAddress)

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
		EthAddress: evmAddress,
		Signature:  "test signature",
	}

	// Broadcast the confirm
	err = broadcaster.ProvideValsetConfirm(context.Background(), 10, vsConfirm)

	// check if the correct error is returned
	assert.Error(t, err)
	assert.Equal(t, orchestrator.ErrEmptyPeersTable, err)
}
