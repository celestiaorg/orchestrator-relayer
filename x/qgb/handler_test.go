package qgb_test

import (
	"crypto/ecdsa"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"github.com/celestiaorg/celestia-app/testutil"
	"github.com/celestiaorg/celestia-app/x/qgb"
	"github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/cosmos-sdk/x/staking"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO add test for all the possible scenarios defined in msg_server.go.
var (
	blockTime = time.Date(2020, 9, 14, 15, 20, 10, 0, time.UTC)

	validator1AccPrivateKey = secp256k1.GenPrivKey()
	validator1AccPublicKey  = validator1AccPrivateKey.PubKey()
	validator1AccAddress    = sdk.AccAddress(validator1AccPublicKey.Address())
	validator1ValAddress    = sdk.ValAddress(validator1AccPublicKey.Address())

	validator2AccPrivateKey = secp256k1.GenPrivKey()
	validator2AccPublicKey  = validator2AccPrivateKey.PubKey()
	validator2AccAddress    = sdk.AccAddress(validator2AccPublicKey.Address())
	validator2ValAddress    = sdk.ValAddress(validator2AccPublicKey.Address())

	orch1EVMPrivateKey, _ = crypto.GenerateKey()
	orch1EVMPublicKey     = orch1EVMPrivateKey.Public().(*ecdsa.PublicKey)
	orch1EVMAddress       = crypto.PubkeyToAddress(*orch1EVMPublicKey)

	orch1PrivateKey = secp256k1.GenPrivKey()
	orch1PublicKey  = orch1PrivateKey.PubKey()
	orch1Address    = sdk.AccAddress(orch1PublicKey.Address())

	orch2EVMPrivateKey, _ = crypto.GenerateKey()
	orch2EVMPublicKey     = orch2EVMPrivateKey.Public().(*ecdsa.PublicKey)
	orch2EVMAddress       = crypto.PubkeyToAddress(*orch2EVMPublicKey)
	orch2HexEVMAddress    = orch2EVMAddress.Hex()

	orch2PrivateKey = secp256k1.GenPrivKey()
	orch2PublicKey  = orch2PrivateKey.PubKey()
	orch2Address    = sdk.AccAddress(orch2PublicKey.Address())

	// If adding more orchestrators, validators, or eth address, please create struct containing this information.
	// Then, create a function that iterates and creates enough of them.
)

// TestMsgValsetConfirm ensures that the valset confirm message sets a validator set confirm
// in the store.
func TestMsgValsetConfirm(t *testing.T) {
	blockHeight := int64(200)

	input, ctx := testutil.SetupFiveValChain(t)
	k := input.QgbKeeper
	h := qgb.NewHandler(*input.QgbKeeper)

	// create new validator
	err := createNewValidator(
		input,
		validator1AccAddress,
		validator1AccPublicKey,
		uint64(120),
		0,
		orch1Address,
		orch1EVMAddress,
		validator1ValAddress,
		validator1AccPublicKey,
	)
	require.NoError(t, err)

	// set a validator set in the store
	vs, err := k.GetCurrentValset(ctx)
	require.NoError(t, err)
	vs.Height = uint64(1)
	vs.Nonce = uint64(1)

	err = k.SetAttestationRequest(ctx, &vs)
	require.Nil(t, err)

	signBytes, err := vs.SignBytes(types.BridgeID)
	require.NoError(t, err)
	signatureBytes, err := types.NewEthereumSignature(signBytes.Bytes(), orch1EVMPrivateKey)
	signature := hex.EncodeToString(signatureBytes)
	require.NoError(t, err)

	// try wrong EVM address
	msg := types.NewMsgValsetConfirm(
		1,
		testutil.EVMAddrs[1], // wrong because validator 0 should have EVMAddrs[0]
		testutil.OrchAddrs[0],
		signature,
	)
	ctx = ctx.WithBlockTime(blockTime).WithBlockHeight(blockHeight)
	_, err = h(ctx, msg)
	require.Error(t, err)

	// try a nonexisting valset
	msg = types.NewMsgValsetConfirm(
		10,
		testutil.EVMAddrs[0],
		testutil.OrchAddrs[0],
		signature,
	)
	ctx = ctx.WithBlockTime(blockTime).WithBlockHeight(blockHeight)
	_, err = h(ctx, msg)
	require.Error(t, err)

	msg = types.NewMsgValsetConfirm(
		1,
		orch1EVMAddress,
		orch1Address,
		signature,
	)
	ctx = ctx.WithBlockTime(blockTime).WithBlockHeight(blockHeight)
	_, err = h(ctx, msg)
	require.NoError(t, err)
}

// TestMsgDataCommitmentConfirm ensures that the data commitment confirm message sets a commitment in the store.
func TestMsgDataCommitmentConfirm(t *testing.T) {
	// Init chain
	input, ctx := testutil.SetupFiveValChain(t)
	k := input.QgbKeeper

	err := createNewValidator(
		input,
		validator1AccAddress,
		validator1AccPublicKey,
		uint64(120),
		0,
		orch1Address,
		orch1EVMAddress,
		validator1ValAddress,
		validator1AccPublicKey,
	)
	require.NoError(t, err)

	// set a validator set in the store
	vs, err := k.GetCurrentValset(ctx)
	require.NoError(t, err)
	vs.Height = uint64(1)
	vs.Nonce = uint64(1)

	err = k.SetAttestationRequest(ctx, &vs)
	require.Nil(t, err)

	h := qgb.NewHandler(*input.QgbKeeper)

	commitment := "102030"
	bytesCommitment, err := hex.DecodeString(commitment)
	require.NoError(t, err)
	dataHash := types.DataCommitmentTupleRootSignBytes(
		types.BridgeID,
		big.NewInt(2),
		bytesCommitment,
	)

	dataCommitment := types.NewDataCommitment(2, 1, 100)
	err = k.SetAttestationRequest(ctx, dataCommitment)
	require.Nil(t, err)

	// Signs the commitment using the orch EVM private key
	signature, err := types.NewEthereumSignature(dataHash.Bytes(), orch1EVMPrivateKey)
	require.NoError(t, err)

	// Sending a data commitment confirm with a nonce referring to a valset nonce
	wrongDCConfirmConfirm := types.NewMsgDataCommitmentConfirm(
		commitment,
		hex.EncodeToString(signature),
		orch1Address,
		orch1EVMAddress,
		1,
		100,
		1,
	)
	_, err = h(ctx, wrongDCConfirmConfirm)
	require.Error(t, err)

	// Sending a data commitment confirm with a wrong begin block
	wrongDCConfirmConfirm = types.NewMsgDataCommitmentConfirm(
		commitment,
		hex.EncodeToString(signature),
		orch1Address,
		orch1EVMAddress,
		2,
		100,
		2,
	)
	_, err = h(ctx, wrongDCConfirmConfirm)
	require.Error(t, err)

	// Sending a data commitment confirm with a wrong end block
	wrongDCConfirmConfirm = types.NewMsgDataCommitmentConfirm(
		commitment,
		hex.EncodeToString(signature),
		orch1Address,
		orch1EVMAddress,
		1,
		101,
		2,
	)
	_, err = h(ctx, wrongDCConfirmConfirm)
	require.Error(t, err)

	// Sending a data commitment confirm with a wrong begin and end block
	wrongDCConfirmConfirm = types.NewMsgDataCommitmentConfirm(
		commitment,
		hex.EncodeToString(signature),
		orch1Address,
		orch1EVMAddress,
		2,
		101,
		2,
	)
	_, err = h(ctx, wrongDCConfirmConfirm)
	require.Error(t, err)

	// Sending a correct data commitment confirm
	setDCCMsg := types.NewMsgDataCommitmentConfirm(
		commitment,
		hex.EncodeToString(signature),
		orch1Address,
		orch1EVMAddress,
		1,
		100,
		2,
	)
	result, err := h(ctx, setDCCMsg)
	require.NoError(t, err)

	// Checking if it was correctly submitted
	actualCommitment, found, err := k.GetDataCommitmentConfirm(ctx, 100, 1, orch1Address)
	require.True(t, found)
	require.Nil(t, err)
	assert.Equal(t, setDCCMsg, actualCommitment)

	// Checking if the event was successfully sent
	actualEvent := result.Events[0]
	assert.Equal(t, sdk.EventTypeMessage, actualEvent.Type)
	assert.Equal(t, sdk.AttributeKeyModule, string(actualEvent.Attributes[0].Key))
	assert.Equal(t, setDCCMsg.Type(), string(actualEvent.Attributes[0].Value))
	assert.Equal(t, types.AttributeKeyDataCommitmentConfirmKey, string(actualEvent.Attributes[1].Key))
	assert.Equal(t, setDCCMsg.String(), string(actualEvent.Attributes[1].Value))
}

// TestMsgDataCommimtentConfirmWithValidatorNotPartOfValset ensures that the data commitment
// confirm message is not accepted if the validator signing it is not part of the valset.
func TestMsgDataCommimtentConfirmWithValidatorNotPartOfValset(t *testing.T) {
	input, ctx := testutil.SetupFiveValChain(t)
	k := input.QgbKeeper
	h := qgb.NewHandler(*input.QgbKeeper)

	// create new validator
	err := createNewValidator(
		input,
		validator1AccAddress,
		validator1AccPublicKey,
		uint64(120),
		0,
		orch1Address,
		orch1EVMAddress,
		validator1ValAddress,
		validator1AccPublicKey,
	)
	require.NoError(t, err)

	// set a validator set in the store
	vs, err := k.GetCurrentValset(ctx)
	require.NoError(t, err)
	vs.Height = uint64(1)
	vs.Nonce = uint64(1)

	err = k.SetAttestationRequest(ctx, &vs)
	require.Nil(t, err)

	// create another validator
	err = createNewValidator(
		input,
		validator2AccAddress,
		validator2AccPublicKey,
		uint64(121),
		0,
		orch2Address,
		orch2EVMAddress,
		validator2ValAddress,
		validator2AccPublicKey,
	)
	require.NoError(t, err)

	// find the new validator
	newVs, err := k.GetCurrentValset(ctx)
	require.NoError(t, err)
	newVs.Height = uint64(10)
	newVs.Nonce = uint64(2)
	bridgeVal := types.BridgeValidator{
		Power:      uint64(613566756),
		EvmAddress: orch2HexEVMAddress,
	}
	require.Contains(t, newVs.Members, bridgeVal)

	// Set a new data commitment
	dc := types.NewDataCommitment(2, 0, 10)
	err = k.SetAttestationRequest(ctx, dc)
	require.Nil(t, err)

	commitment := "102030"
	bytesCommitment, err := hex.DecodeString(commitment)
	require.NoError(t, err)
	dataHash := types.DataCommitmentTupleRootSignBytes(
		types.BridgeID,
		big.NewInt(2),
		bytesCommitment,
	)

	// Signs the commitment using the second validator
	signature, err := types.NewEthereumSignature(dataHash.Bytes(), orch2EVMPrivateKey)
	require.NoError(t, err)

	// Sending a data commitment confirm
	setDCCMsg := types.NewMsgDataCommitmentConfirm(
		commitment,
		hex.EncodeToString(signature),
		orch2Address,
		orch2EVMAddress,
		0,
		100,
		2,
	)
	_, err = h(ctx, setDCCMsg)
	require.Error(t, err)
}

func createNewValidator(
	input testutil.TestInput,
	addr sdk.AccAddress,
	pubKey cryptotypes.PubKey,
	accountNumber uint64,
	sequence uint64,
	orchAddr sdk.AccAddress,
	orchEVMAddr common.Address,
	valAddress sdk.ValAddress,
	valAccPublicKey cryptotypes.PubKey,
) error {
	// Create a new validator
	acc := input.AccountKeeper.NewAccount(
		input.Context,
		authtypes.NewBaseAccount(addr, pubKey, accountNumber, sequence),
	)
	err := input.BankKeeper.MintCoins(input.Context, types.ModuleName, testutil.InitCoins)
	if err != nil {
		return err
	}
	err = input.BankKeeper.SendCoinsFromModuleToAccount(
		input.Context,
		types.ModuleName,
		acc.GetAddress(),
		testutil.InitCoins,
	)
	if err != nil {
		return err
	}
	input.AccountKeeper.SetAccount(input.Context, acc)

	msgServer := stakingkeeper.NewMsgServerImpl(input.StakingKeeper)
	_, err = msgServer.CreateValidator(
		input.Context,
		testutil.NewTestMsgCreateValidator(
			valAddress,
			valAccPublicKey,
			testutil.StakingAmount,
			orchAddr,
			orchEVMAddr,
		),
	)
	if err != nil {
		return err
	}
	staking.EndBlocker(input.Context, input.StakingKeeper)
	return nil
}
