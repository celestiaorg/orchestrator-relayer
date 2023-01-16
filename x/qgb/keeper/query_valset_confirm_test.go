package keeper_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/celestiaorg/celestia-app/testutil"
	"github.com/celestiaorg/celestia-app/x/qgb/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryValsetConfirm(t *testing.T) {
	var (
		addrStr                     = "cosmos1v4s3yfg8rujaz56yt5a3xznqjqgyeff4552l40"
		nonce                       = uint64(1)
		myValidatorCosmosAddr, err1 = sdk.AccAddressFromBech32(addrStr)
		myValidatorEVMAddr          = gethcommon.HexToAddress("0x3232323232323232323232323232323232323232")
	)
	require.NoError(t, err1)
	input := testutil.CreateTestEnv(t)
	sdkCtx := input.Context
	ctx := sdk.WrapSDKContext(input.Context)
	k := input.QgbKeeper
	_, _ = input.QgbKeeper.SetValsetConfirm(sdkCtx, *types.NewMsgValsetConfirm(
		nonce,
		myValidatorEVMAddr,
		myValidatorCosmosAddr,
		"alksdjhflkasjdfoiasjdfiasjdfoiasdj",
	))

	specs := map[string]struct {
		src     types.QueryValsetConfirmRequest
		expErr  bool
		expResp types.QueryValsetConfirmResponse
	}{
		"all good": {
			src: types.QueryValsetConfirmRequest{Nonce: 1, Address: myValidatorCosmosAddr.String()},
			expResp: types.QueryValsetConfirmResponse{
				Confirm: types.NewMsgValsetConfirm(
					1,
					myValidatorEVMAddr,
					myValidatorCosmosAddr,
					"alksdjhflkasjdfoiasjdfiasjdfoiasdj",
				),
			},
			expErr: false,
		},
		"unknown nonce": {
			src:     types.QueryValsetConfirmRequest{Nonce: 999999, Address: myValidatorCosmosAddr.String()},
			expResp: types.QueryValsetConfirmResponse{Confirm: nil},
		},
		"invalid address": {
			src:    types.QueryValsetConfirmRequest{Nonce: 1, Address: "not a valid addr"},
			expErr: true,
		},
	}

	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			got, err := k.ValsetConfirm(ctx, &spec.src)
			if spec.expErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			if spec.expResp == (types.QueryValsetConfirmResponse{}) {
				assert.True(t, got == nil || got.Confirm == nil)
				return
			}
			assert.Equal(t, &spec.expResp, got)
		})
	}
}

func TestAllValsetConfirmsByNonce(t *testing.T) {
	addrs := []string{
		"cosmos1v4s3yfg8rujaz56yt5a3xznqjqgyeff4552l40",
		"cosmos1dz6pu605p5x79dh5pz4dardhuzws6c0qqr0l6e",
		"cosmos1er9mgk7x30aspqd2zwn970ywfls36ktdmgyzry",
	}
	var (
		nonce                     = uint64(1)
		myValidatorCosmosAddr1, _ = sdk.AccAddressFromBech32(addrs[0])
		myValidatorCosmosAddr2, _ = sdk.AccAddressFromBech32(addrs[1])
		myValidatorCosmosAddr3, _ = sdk.AccAddressFromBech32(addrs[2])
		myValidatorEVMAddr1       = gethcommon.HexToAddress("0x0101010101010101010101010101010101010101")
		myValidatorEVMAddr2       = gethcommon.HexToAddress("0x0202020202020202020202020202020202020202")
		myValidatorEVMAddr3       = gethcommon.HexToAddress("0x0303030303030303030303030303030303030303")
	)

	input := testutil.CreateTestEnv(t)
	sdkCtx := input.Context
	ctx := sdk.WrapSDKContext(input.Context)
	k := input.QgbKeeper

	// seed confirmations
	for i := 0; i < 3; i++ {
		addr, _ := sdk.AccAddressFromBech32(addrs[i])
		msg := types.MsgValsetConfirm{}
		msg.EvmAddress = gethcommon.BytesToAddress(bytes.Repeat([]byte{byte(i + 1)}, 20)).String()
		msg.Nonce = uint64(1)
		msg.Orchestrator = addr.String()
		msg.Signature = fmt.Sprintf("signature %d", i+1)
		_, _ = input.QgbKeeper.SetValsetConfirm(sdkCtx, msg)
	}

	specs := map[string]struct {
		src     types.QueryValsetConfirmsByNonceRequest
		expErr  bool
		expResp types.QueryValsetConfirmsByNonceResponse
	}{
		"all good": {
			src: types.QueryValsetConfirmsByNonceRequest{Nonce: 1},
			expResp: types.QueryValsetConfirmsByNonceResponse{Confirms: []types.MsgValsetConfirm{
				*types.NewMsgValsetConfirm(nonce, myValidatorEVMAddr1, myValidatorCosmosAddr1, "signature 1"),
				*types.NewMsgValsetConfirm(nonce, myValidatorEVMAddr2, myValidatorCosmosAddr2, "signature 2"),
				*types.NewMsgValsetConfirm(nonce, myValidatorEVMAddr3, myValidatorCosmosAddr3, "signature 3"),
			}},
		},
		"unknown nonce": {
			src:     types.QueryValsetConfirmsByNonceRequest{Nonce: 999999},
			expResp: types.QueryValsetConfirmsByNonceResponse{},
		},
	}
	for msg, spec := range specs {
		t.Run(msg, func(t *testing.T) {
			got, err := k.ValsetConfirmsByNonce(ctx, &types.QueryValsetConfirmsByNonceRequest{Nonce: spec.src.Nonce})
			if spec.expErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			var gotArray []types.MsgValsetConfirm
			if len(spec.expResp.Confirms) != 0 {
				gotArray = make([]types.MsgValsetConfirm, len(got.Confirms))
				copy(gotArray, got.Confirms)
			}
			assert.Equal(t, spec.expResp.Confirms, gotArray)
		})
	}
}

func TestQueryCurrentValset(t *testing.T) {
	expectedValset := types.Valset{
		Nonce:  1,
		Height: 1234567,
		Members: []types.BridgeValidator{
			{
				Power:      858993459,
				EvmAddress: testutil.EVMAddrs[0].Hex(),
			},
			{
				Power:      858993459,
				EvmAddress: testutil.EVMAddrs[1].Hex(),
			},
			{
				Power:      858993459,
				EvmAddress: testutil.EVMAddrs[2].Hex(),
			},
			{
				Power:      858993459,
				EvmAddress: testutil.EVMAddrs[3].Hex(),
			},
			{
				Power:      858993459,
				EvmAddress: testutil.EVMAddrs[4].Hex(),
			},
		},
	}
	input, _ := testutil.SetupFiveValChain(t)
	sdkCtx := input.Context

	currentValset, err := input.QgbKeeper.GetCurrentValset(sdkCtx)
	require.NoError(t, err)

	assert.Equal(t, expectedValset, currentValset)
}
