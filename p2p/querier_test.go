package p2p_test

import (
	"context"
	"testing"
	"time"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	qgbtesting "github.com/celestiaorg/orchestrator-relayer/testing"
	"github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func TestQueryTwoThirdsDataCommitmentConfirms(t *testing.T) {
	ctx := context.Background()
	network := qgbtesting.NewDHTNetwork(ctx, 2)
	defer network.Stop()

	vsNonce := uint64(2)
	ethAddr1 := common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b")
	ethAddr2 := common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622c")
	ethAddr3 := common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622d")

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

	// put a single confirm
	dc1 := types.NewDataCommitmentConfirm(
		"commitment",
		"signature",
		ethAddr1,
	)
	err := network.DHTs[0].PutDataCommitmentConfirm(
		ctx,
		p2p.GetDataCommitmentConfirmKey(dcNonce, ethAddr1.String()),
		*dc1,
	)
	require.NoError(t, err)

	querier := p2p.NewQuerier(network.DHTs[0], tmlog.NewNopLogger())

	// query two thirds of confirms. should time-out.
	confirms, err := querier.QueryTwoThirdsDataCommitmentConfirms(ctx, 10*time.Second, previousValset, dcNonce)
	require.Error(t, err)
	require.Nil(t, confirms)

	// put the second confirm.
	dc2 := types.NewDataCommitmentConfirm(
		"commitment",
		"signature",
		ethAddr2,
	)
	err = network.DHTs[0].PutDataCommitmentConfirm(
		ctx,
		p2p.GetDataCommitmentConfirmKey(dcNonce, ethAddr2.String()),
		*dc2,
	)
	require.NoError(t, err)

	// query two thirds of confirms. should return 2 confirms.
	confirms, err = querier.QueryTwoThirdsDataCommitmentConfirms(ctx, 20*time.Second, previousValset, dcNonce)
	require.NoError(t, err)
	assert.Contains(t, confirms, *dc1)
	assert.Contains(t, confirms, *dc2)

	// put the third confirm.
	dc3 := types.NewDataCommitmentConfirm(
		"commitment",
		"signature",
		ethAddr3,
	)
	err = network.DHTs[0].PutDataCommitmentConfirm(
		ctx,
		p2p.GetDataCommitmentConfirmKey(dcNonce, ethAddr3.String()),
		*dc3,
	)
	require.NoError(t, err)

	// query two thirds of confirms. should return 2 confirms.
	confirms, err = querier.QueryTwoThirdsDataCommitmentConfirms(ctx, 20*time.Second, previousValset, dcNonce)
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
	ethAddr1 := common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b")
	ethAddr2 := common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622c")
	ethAddr3 := common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622d")

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

	// put a single confirm
	vs1 := types.NewValsetConfirm(
		ethAddr1,
		"signature",
	)
	err := network.DHTs[0].PutValsetConfirm(
		ctx,
		p2p.GetValsetConfirmKey(vsNonce, ethAddr1.String()),
		*vs1,
	)
	require.NoError(t, err)

	querier := p2p.NewQuerier(network.DHTs[0], tmlog.NewNopLogger())

	// query two thirds of confirms. should time-out.
	confirms, err := querier.QueryTwoThirdsValsetConfirms(ctx, 10*time.Second, valset)
	require.Error(t, err)
	require.Nil(t, confirms)

	// put the second confirm.
	vs2 := types.NewValsetConfirm(
		ethAddr2,
		"signature",
	)
	err = network.DHTs[0].PutValsetConfirm(
		ctx,
		p2p.GetValsetConfirmKey(vsNonce, ethAddr2.String()),
		*vs2,
	)
	require.NoError(t, err)

	// query two thirds of confirms. should return 2 confirms.
	confirms, err = querier.QueryTwoThirdsValsetConfirms(ctx, 20*time.Second, valset)
	require.NoError(t, err)
	assert.Contains(t, confirms, *vs1)
	assert.Contains(t, confirms, *vs2)

	// put the third confirm.
	vs3 := types.NewValsetConfirm(
		ethAddr3,
		"signature",
	)
	err = network.DHTs[0].PutValsetConfirm(
		ctx,
		p2p.GetValsetConfirmKey(vsNonce, ethAddr3.String()),
		*vs3,
	)
	require.NoError(t, err)

	// query two thirds of confirms. should return 2 confirms.
	confirms, err = querier.QueryTwoThirdsValsetConfirms(ctx, 20*time.Second, valset)
	require.NoError(t, err)
	assert.Contains(t, confirms, *vs1)
	assert.Contains(t, confirms, *vs2)
	assert.Contains(t, confirms, *vs3)
}

func TestQueryValsetConfirmByEVMAddress(t *testing.T) {
	ctx := context.Background()
	network := qgbtesting.NewDHTNetwork(ctx, 2)
	defer network.Stop()

	ethAddr := common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b")
	vsNonce := uint64(10)

	querier := p2p.NewQuerier(network.DHTs[0], tmlog.NewNopLogger())

	// query the valset confirm: should return nil
	confirm, err := querier.QueryValsetConfirmByEVMAddress(ctx, vsNonce, ethAddr.String())
	require.NoError(t, err)
	assert.Nil(t, confirm)

	// put a single confirm
	vs := types.NewValsetConfirm(
		ethAddr,
		"signature",
	)
	err = network.DHTs[0].PutValsetConfirm(
		ctx,
		p2p.GetValsetConfirmKey(vsNonce, ethAddr.String()),
		*vs,
	)
	require.NoError(t, err)

	// query the valset confirm
	confirm, err = querier.QueryValsetConfirmByEVMAddress(ctx, vsNonce, ethAddr.String())
	require.NoError(t, err)
	require.NotNil(t, confirm)
	assert.Equal(t, vs, confirm)
}

func TestQueryDataCommitmentConfirmByEVMAddress(t *testing.T) {
	ctx := context.Background()
	network := qgbtesting.NewDHTNetwork(ctx, 2)
	defer network.Stop()

	ethAddr := common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b")
	dcNonce := uint64(10)

	querier := p2p.NewQuerier(network.DHTs[0], tmlog.NewNopLogger())

	// query the data commitment confirm: should return nil
	confirm, err := querier.QueryDataCommitmentConfirmByEVMAddress(ctx, dcNonce, ethAddr.String())
	require.NoError(t, err)
	assert.Nil(t, confirm)

	// put a single confirm
	dc := types.NewDataCommitmentConfirm(
		"commitment",
		"signature",
		ethAddr,
	)
	err = network.DHTs[0].PutDataCommitmentConfirm(
		ctx,
		p2p.GetDataCommitmentConfirmKey(dcNonce, ethAddr.String()),
		*dc,
	)
	require.NoError(t, err)

	// query the data commitment confirm
	confirm, err = querier.QueryDataCommitmentConfirmByEVMAddress(ctx, dcNonce, ethAddr.String())
	require.NoError(t, err)
	require.NotNil(t, confirm)
	assert.Equal(t, dc, confirm)
}

func TestQueryValsetConfirms(t *testing.T) {
	ctx := context.Background()
	network := qgbtesting.NewDHTNetwork(ctx, 2)
	defer network.Stop()

	vsNonce := uint64(2)
	ethAddr1 := common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b")
	ethAddr2 := common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622c")
	ethAddr3 := common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622d")

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

	// put the confirms
	vs1 := types.NewValsetConfirm(
		ethAddr1,
		"signature",
	)
	err := network.DHTs[0].PutValsetConfirm(
		ctx,
		p2p.GetValsetConfirmKey(vsNonce, ethAddr1.String()),
		*vs1,
	)
	require.NoError(t, err)
	vs2 := types.NewValsetConfirm(
		ethAddr2,
		"signature",
	)
	err = network.DHTs[0].PutValsetConfirm(
		ctx,
		p2p.GetValsetConfirmKey(vsNonce, ethAddr2.String()),
		*vs2,
	)
	require.NoError(t, err)
	vs3 := types.NewValsetConfirm(
		ethAddr3,
		"signature",
	)
	err = network.DHTs[0].PutValsetConfirm(
		ctx,
		p2p.GetValsetConfirmKey(vsNonce, ethAddr3.String()),
		*vs3,
	)
	require.NoError(t, err)

	querier := p2p.NewQuerier(network.DHTs[0], tmlog.NewNopLogger())

	// query the confirms
	confirms, err := querier.QueryValsetConfirms(ctx, valset)
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
	ethAddr1 := common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622b")
	ethAddr2 := common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622c")
	ethAddr3 := common.HexToAddress("0xfA906e15C9Eaf338c4110f0E21983c6b3b2d622d")

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

	// put the confirms
	dc1 := types.NewDataCommitmentConfirm(
		"commitment",
		"signature",
		ethAddr1,
	)
	err := network.DHTs[0].PutDataCommitmentConfirm(
		ctx,
		p2p.GetDataCommitmentConfirmKey(dcNonce, ethAddr1.String()),
		*dc1,
	)
	require.NoError(t, err)
	dc2 := types.NewDataCommitmentConfirm(
		"commitment",
		"signature",
		ethAddr2,
	)
	err = network.DHTs[0].PutDataCommitmentConfirm(
		ctx,
		p2p.GetDataCommitmentConfirmKey(dcNonce, ethAddr2.String()),
		*dc2,
	)
	require.NoError(t, err)
	dc3 := types.NewDataCommitmentConfirm(
		"commitment",
		"signature",
		ethAddr3,
	)
	err = network.DHTs[0].PutDataCommitmentConfirm(
		ctx,
		p2p.GetDataCommitmentConfirmKey(dcNonce, ethAddr3.String()),
		*dc3,
	)
	require.NoError(t, err)

	querier := p2p.NewQuerier(network.DHTs[0], tmlog.NewNopLogger())

	// query the confirms
	confirms, err := querier.QueryDataCommitmentConfirms(ctx, valset, dcNonce)
	require.NoError(t, err)
	require.NotNil(t, confirms)
	assert.Contains(t, confirms, *dc1)
	assert.Contains(t, confirms, *dc2)
	assert.Contains(t, confirms, *dc3)
}
