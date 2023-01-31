package types

import (
	"fmt"
	"math/big"

	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// NewMsgDataCommitmentConfirm creates a new NewMsgDataCommitmentConfirm.
func NewMsgDataCommitmentConfirm(
	commitment string,
	signature string,
	validatorAddress sdk.AccAddress,
	evmAddr ethcmn.Address,
	beginBlock uint64,
	endBlock uint64,
	nonce uint64,
) *MsgDataCommitmentConfirm {
	return &MsgDataCommitmentConfirm{
		Commitment:       commitment,
		Signature:        signature,
		ValidatorAddress: validatorAddress.String(),
		EvmAddress:       evmAddr.Hex(),
		BeginBlock:       beginBlock,
		EndBlock:         endBlock,
		Nonce:            nonce,
	}
}

// GetSigners defines whose signature is required.
func (msg *MsgDataCommitmentConfirm) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.ValidatorAddress)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{acc}
}

// ValidateBasic performs stateless checks.
func (msg *MsgDataCommitmentConfirm) ValidateBasic() (err error) {
	if _, err = sdk.AccAddressFromBech32(msg.ValidatorAddress); err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidAddress, msg.ValidatorAddress)
	}
	if msg.BeginBlock > msg.EndBlock {
		return sdkerrors.Wrap(ErrInvalid, "begin block should be less than end block")
	}
	if !ethcmn.IsHexAddress(msg.EvmAddress) {
		return sdkerrors.Wrap(stakingtypes.ErrEVMAddressNotHex, "evm address")
	}
	return nil
}

// Type should return the action.
func (msg *MsgDataCommitmentConfirm) Type() string { return "data_commitment_confirm" }

// DataCommitmentTupleRootSignBytes EncodeDomainSeparatedDataCommitment takes the required input data and
// produces the required signature to confirm a validator set update on the QGB EVM contract.
// This value will then be signed before being submitted to Cosmos, verified, and then relayed to the
// target EVM chain.
func DataCommitmentTupleRootSignBytes(bridgeID ethcmn.Hash, nonce *big.Int, commitment []byte) ethcmn.Hash {
	var dataCommitment [32]uint8
	copy(dataCommitment[:], commitment)

	// the word 'transactionBatch' needs to be the same as the 'name' above in the DataCommitmentConfirmABIJSON
	// but other than that it's a constant that has no impact on the output. This is because
	// it gets encoded as a function name which we must then discard.
	bytes, err := InternalQGBabi.Pack(
		"domainSeparateDataRootTupleRoot",
		bridgeID,
		DcDomainSeparator,
		nonce,
		dataCommitment,
	)
	// this should never happen outside of test since any case that could crash on encoding
	// should be filtered above.
	if err != nil {
		panic(fmt.Sprintf("Error packing checkpoint! %s/n", err))
	}

	// we hash the resulting encoded bytes discarding the first 4 bytes these 4 bytes are the constant
	// method name 'checkpoint'. If you where to replace the checkpoint constant in this code you would
	// then need to adjust how many bytes you truncate off the front to get the output of abi.encode()
	hash := crypto.Keccak256Hash(bytes[4:])
	return hash
}
