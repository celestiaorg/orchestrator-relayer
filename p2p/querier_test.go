package p2p_test

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/ethereum/go-ethereum/crypto"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	qgbtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	"github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

var (
	ethAddr1       = common.HexToAddress("0x966e6f22781EF6a6A82BBB4DB3df8E225DfD9488")
	privateKey1, _ = crypto.HexToECDSA("da6ed55cb2894ac2c9c10209c09de8e8b9d109b910338d5bf3d747a7e1fc9eb9")
	ethAddr2       = common.HexToAddress("0x91DEd26b5f38B065FC0204c7929Da1b2A21877Ad")
	privateKey2, _ = crypto.HexToECDSA("002ad18ca3def673345897b063bfa98d829a4d812dbd07f1938676828a82c4f9")
	ethAddr3       = common.HexToAddress("0x3d22f0C38251ebdBE92e14BBF1bd2067F1C3b7D7")
	privateKey3, _ = crypto.HexToECDSA("6adac8b5de0ba702ec8feab6d386a0c7334c6720b9174c02333700d431057af8")
)

func TestQueryTwoThirdsDataCommitmentConfirms(t *testing.T) {
	ctx := context.Background()
	network := qgbtesting.NewDHTNetwork(ctx, 2)
	defer network.Stop()

	vsNonce := uint64(2)

	previousValset := celestiatypes.Valset{
		Nonce: vsNonce,
		Members: []celestiatypes.BridgeValidator{
			{
				Power:      10,
				EvmAddress: ethAddr1.String(),
			},
			{
				Power:      15,
				EvmAddress: ethAddr2.String(),
			},
			{
				Power:      10,
				EvmAddress: ethAddr3.String(),
			},
		},
		Height: 10,
	}
	dcNonce := uint64(4)
	commitment := "1234"
	bCommitment, _ := hex.DecodeString(commitment)
	dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(dcNonce)), bCommitment)

	signature1, err := evm.NewEthereumSignature(dataRootHash.Bytes(), privateKey1)
	require.NoError(t, err)
	// put a single confirm
	dc1 := types.NewDataCommitmentConfirm(hex.EncodeToString(signature1), ethAddr1)
	err = network.DHTs[0].PutDataCommitmentConfirm(
		ctx,
		p2p.GetDataCommitmentConfirmKey(dcNonce, ethAddr1.String(), dataRootHash.Hex()),
		*dc1,
	)
	require.NoError(t, err)

	querier := p2p.NewQuerier(network.DHTs[0], tmlog.NewNopLogger())

	// query two thirds of confirms. should time-out.
	confirms, err := querier.QueryTwoThirdsDataCommitmentConfirms(
		ctx,
		time.Second,
		time.Millisecond,
		previousValset,
		dcNonce,
		dataRootHash.Hex(),
	)
	require.Error(t, err)
	require.Nil(t, confirms)

	signature2, err := evm.NewEthereumSignature(dataRootHash.Bytes(), privateKey2)
	require.NoError(t, err)
	// put the second confirm.
	dc2 := types.NewDataCommitmentConfirm(hex.EncodeToString(signature2), ethAddr2)
	err = network.DHTs[0].PutDataCommitmentConfirm(
		ctx,
		p2p.GetDataCommitmentConfirmKey(dcNonce, ethAddr2.String(), dataRootHash.Hex()),
		*dc2,
	)
	require.NoError(t, err)

	// query two thirds of confirms. should return 2 confirms.
	confirms, err = querier.QueryTwoThirdsDataCommitmentConfirms(
		ctx,
		20*time.Second,
		time.Millisecond,
		previousValset,
		dcNonce,
		dataRootHash.Hex(),
	)
	require.NoError(t, err)
	assert.Contains(t, confirms, *dc1)
	assert.Contains(t, confirms, *dc2)

	signature3, err := evm.NewEthereumSignature(dataRootHash.Bytes(), privateKey3)
	require.NoError(t, err)
	// put the third confirm.
	dc3 := types.NewDataCommitmentConfirm(hex.EncodeToString(signature3), ethAddr3)
	err = network.DHTs[0].PutDataCommitmentConfirm(
		ctx,
		p2p.GetDataCommitmentConfirmKey(dcNonce, ethAddr3.String(), dataRootHash.Hex()),
		*dc3,
	)
	require.NoError(t, err)

	// query two thirds of confirms. should return 2 confirms.
	confirms, err = querier.QueryTwoThirdsDataCommitmentConfirms(
		ctx,
		20*time.Second,
		time.Millisecond,
		previousValset,
		dcNonce,
		dataRootHash.Hex(),
	)
	require.NoError(t, err)
	assert.Contains(t, confirms, *dc1)
	assert.Contains(t, confirms, *dc2)
	assert.Contains(t, confirms, *dc3)
}

func TestQueryTwoThirdsValsetConfirms(t *testing.T) {
	ctx := context.Background()
	network := qgbtesting.NewDHTNetwork(ctx, 2)
	defer network.Stop()

	vsNonce := uint64(2)

	previousValset := celestiatypes.Valset{
		Nonce: vsNonce - 1,
		Members: []celestiatypes.BridgeValidator{
			{
				Power:      10,
				EvmAddress: ethAddr1.String(),
			},
			{
				Power:      15,
				EvmAddress: ethAddr2.String(),
			},
			{
				Power:      10,
				EvmAddress: ethAddr3.String(),
			},
		},
		Height: 10,
	}

	signBytes, err := previousValset.SignBytes()
	require.NoError(t, err)

	signature1, err := evm.NewEthereumSignature(signBytes.Bytes(), privateKey1)
	require.NoError(t, err)

	// put a single confirm
	vs1 := types.NewValsetConfirm(
		ethAddr1,
		hex.EncodeToString(signature1),
	)
	err = network.DHTs[0].PutValsetConfirm(
		ctx,
		p2p.GetValsetConfirmKey(vsNonce, ethAddr1.String(), signBytes.Hex()),
		*vs1,
	)
	require.NoError(t, err)

	querier := p2p.NewQuerier(network.DHTs[0], tmlog.NewNopLogger())

	// query two thirds of confirms. should time-out.
	confirms, err := querier.QueryTwoThirdsValsetConfirms(
		ctx,
		time.Second,
		time.Millisecond,
		vsNonce,
		previousValset,
		signBytes.Hex(),
	)
	require.Error(t, err)
	require.Nil(t, confirms)

	signature2, err := evm.NewEthereumSignature(signBytes.Bytes(), privateKey2)
	require.NoError(t, err)

	// put the second confirm.
	vs2 := types.NewValsetConfirm(
		ethAddr2,
		hex.EncodeToString(signature2),
	)
	err = network.DHTs[0].PutValsetConfirm(
		ctx,
		p2p.GetValsetConfirmKey(vsNonce, ethAddr2.String(), signBytes.Hex()),
		*vs2,
	)
	require.NoError(t, err)

	// query two thirds of confirms. should return 2 confirms.
	confirms, err = querier.QueryTwoThirdsValsetConfirms(
		ctx,
		20*time.Second,
		time.Millisecond,
		vsNonce,
		previousValset,
		signBytes.Hex(),
	)
	require.NoError(t, err)
	assert.Contains(t, confirms, *vs1)
	assert.Contains(t, confirms, *vs2)

	signature3, err := evm.NewEthereumSignature(signBytes.Bytes(), privateKey3)
	require.NoError(t, err)

	// put the third confirm.
	vs3 := types.NewValsetConfirm(ethAddr3, hex.EncodeToString(signature3))
	err = network.DHTs[0].PutValsetConfirm(
		ctx,
		p2p.GetValsetConfirmKey(vsNonce, ethAddr3.String(), signBytes.Hex()),
		*vs3,
	)
	require.NoError(t, err)

	// query two thirds of confirms. should return 2 confirms.
	confirms, err = querier.QueryTwoThirdsValsetConfirms(
		ctx,
		20*time.Second,
		time.Millisecond,
		vsNonce,
		previousValset,
		signBytes.Hex(),
	)
	require.NoError(t, err)
	assert.Contains(t, confirms, *vs1)
	assert.Contains(t, confirms, *vs2)
	assert.Contains(t, confirms, *vs3)
}

func TestQueryValsetConfirmByEVMAddress(t *testing.T) {
	ctx := context.Background()
	network := qgbtesting.NewDHTNetwork(ctx, 2)
	defer network.Stop()

	vsNonce := uint64(10)

	querier := p2p.NewQuerier(network.DHTs[0], tmlog.NewNopLogger())

	signBytes := common.HexToHash("1234")

	// query the valset confirm: should return nil
	confirm, err := querier.QueryValsetConfirmByEVMAddress(ctx, vsNonce, ethAddr1.String(), signBytes.Hex())
	require.NoError(t, err)
	assert.Nil(t, confirm)

	signature1, err := evm.NewEthereumSignature(signBytes.Bytes(), privateKey1)
	require.NoError(t, err)

	// put a single confirm
	vs := types.NewValsetConfirm(ethAddr1, hex.EncodeToString(signature1))
	err = network.DHTs[0].PutValsetConfirm(
		ctx,
		p2p.GetValsetConfirmKey(vsNonce, ethAddr1.String(), signBytes.Hex()),
		*vs,
	)
	require.NoError(t, err)

	// query the valset confirm
	confirm, err = querier.QueryValsetConfirmByEVMAddress(ctx, vsNonce, ethAddr1.String(), signBytes.Hex())
	require.NoError(t, err)
	require.NotNil(t, confirm)
	assert.Equal(t, vs, confirm)
}

func TestQueryDataCommitmentConfirmByEVMAddress(t *testing.T) {
	ctx := context.Background()
	network := qgbtesting.NewDHTNetwork(ctx, 2)
	defer network.Stop()

	dcNonce := uint64(10)

	querier := p2p.NewQuerier(network.DHTs[0], tmlog.NewNopLogger())

	// query the data commitment confirm: should return nil
	commitment := "1234"
	bCommitment, _ := hex.DecodeString(commitment)
	dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(dcNonce)), bCommitment)

	confirm, err := querier.QueryDataCommitmentConfirmByEVMAddress(ctx, dcNonce, ethAddr1.String(), dataRootHash.Hex())
	require.NoError(t, err)
	assert.Nil(t, confirm)

	signature, err := evm.NewEthereumSignature(dataRootHash.Bytes(), privateKey1)
	require.NoError(t, err)
	// put a single confirm
	dc := types.NewDataCommitmentConfirm(hex.EncodeToString(signature), ethAddr1)
	err = network.DHTs[0].PutDataCommitmentConfirm(
		ctx,
		p2p.GetDataCommitmentConfirmKey(dcNonce, ethAddr1.String(), dataRootHash.Hex()),
		*dc,
	)
	require.NoError(t, err)

	// query the data commitment confirm
	confirm, err = querier.QueryDataCommitmentConfirmByEVMAddress(ctx, dcNonce, ethAddr1.String(), dataRootHash.Hex())
	require.NoError(t, err)
	require.NotNil(t, confirm)
	assert.Equal(t, dc, confirm)
}

func TestQueryValsetConfirms(t *testing.T) {
	ctx := context.Background()
	network := qgbtesting.NewDHTNetwork(ctx, 2)
	defer network.Stop()

	vsNonce := uint64(2)

	valset := celestiatypes.Valset{
		Nonce: vsNonce,
		Members: []celestiatypes.BridgeValidator{
			{
				Power:      10,
				EvmAddress: ethAddr1.String(),
			},
			{
				Power:      15,
				EvmAddress: ethAddr2.String(),
			},
			{
				Power:      10,
				EvmAddress: ethAddr3.String(),
			},
		},
		Height: 10,
	}

	signBytes, _ := valset.SignBytes()
	signature1, err := evm.NewEthereumSignature(signBytes.Bytes(), privateKey1)
	require.NoError(t, err)

	// put the confirms
	vs1 := types.NewValsetConfirm(ethAddr1, hex.EncodeToString(signature1))
	err = network.DHTs[0].PutValsetConfirm(
		ctx,
		p2p.GetValsetConfirmKey(vsNonce, ethAddr1.String(), signBytes.Hex()),
		*vs1,
	)
	require.NoError(t, err)
	signature2, err := evm.NewEthereumSignature(signBytes.Bytes(), privateKey2)
	require.NoError(t, err)
	vs2 := types.NewValsetConfirm(ethAddr2, hex.EncodeToString(signature2))
	err = network.DHTs[0].PutValsetConfirm(
		ctx,
		p2p.GetValsetConfirmKey(vsNonce, ethAddr2.String(), signBytes.Hex()),
		*vs2,
	)
	require.NoError(t, err)
	signature3, err := evm.NewEthereumSignature(signBytes.Bytes(), privateKey3)
	require.NoError(t, err)
	vs3 := types.NewValsetConfirm(ethAddr3, hex.EncodeToString(signature3))
	err = network.DHTs[0].PutValsetConfirm(
		ctx,
		p2p.GetValsetConfirmKey(vsNonce, ethAddr3.String(), signBytes.Hex()),
		*vs3,
	)
	require.NoError(t, err)

	querier := p2p.NewQuerier(network.DHTs[0], tmlog.NewNopLogger())

	// query the confirms
	confirms, err := querier.QueryValsetConfirms(ctx, valset.Nonce, valset, signBytes.Hex())
	require.NoError(t, err)
	require.NotNil(t, confirms)
	assert.Contains(t, confirms, *vs1)
	assert.Contains(t, confirms, *vs2)
	assert.Contains(t, confirms, *vs3)
}

func TestQueryDataCommitmentConfirms(t *testing.T) {
	ctx := context.Background()
	network := qgbtesting.NewDHTNetwork(ctx, 2)
	defer network.Stop()

	dcNonce := uint64(2)

	valset := celestiatypes.Valset{
		Nonce: 10,
		Members: []celestiatypes.BridgeValidator{
			{
				Power:      10,
				EvmAddress: ethAddr1.String(),
			},
			{
				Power:      15,
				EvmAddress: ethAddr2.String(),
			},
			{
				Power:      10,
				EvmAddress: ethAddr3.String(),
			},
		},
		Height: 10,
	}
	commitment := "1234"
	bCommitment, _ := hex.DecodeString(commitment)
	dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(dcNonce)), bCommitment)

	signature1, err := evm.NewEthereumSignature(dataRootHash.Bytes(), privateKey1)
	require.NoError(t, err)
	// put the confirms
	dc1 := types.NewDataCommitmentConfirm(hex.EncodeToString(signature1), ethAddr1)
	err = network.DHTs[0].PutDataCommitmentConfirm(
		ctx,
		p2p.GetDataCommitmentConfirmKey(dcNonce, ethAddr1.String(), dataRootHash.Hex()),
		*dc1,
	)
	require.NoError(t, err)
	signature2, err := evm.NewEthereumSignature(dataRootHash.Bytes(), privateKey2)
	require.NoError(t, err)
	dc2 := types.NewDataCommitmentConfirm(hex.EncodeToString(signature2), ethAddr2)
	err = network.DHTs[0].PutDataCommitmentConfirm(
		ctx,
		p2p.GetDataCommitmentConfirmKey(dcNonce, ethAddr2.String(), dataRootHash.Hex()),
		*dc2,
	)
	require.NoError(t, err)
	signature3, err := evm.NewEthereumSignature(dataRootHash.Bytes(), privateKey3)
	require.NoError(t, err)
	dc3 := types.NewDataCommitmentConfirm(hex.EncodeToString(signature3), ethAddr3)
	err = network.DHTs[0].PutDataCommitmentConfirm(
		ctx,
		p2p.GetDataCommitmentConfirmKey(dcNonce, ethAddr3.String(), dataRootHash.Hex()),
		*dc3,
	)
	require.NoError(t, err)

	querier := p2p.NewQuerier(network.DHTs[0], tmlog.NewNopLogger())

	// query the confirms
	confirms, err := querier.QueryDataCommitmentConfirms(ctx, valset, dcNonce, dataRootHash.Hex())
	require.NoError(t, err)
	require.NotNil(t, confirms)
	assert.Contains(t, confirms, *dc1)
	assert.Contains(t, confirms, *dc2)
	assert.Contains(t, confirms, *dc3)
}
