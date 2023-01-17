package types

import (
	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/evm"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

var _ AttestationConfirm = &ValsetConfirm{}

// ValsetConfirm
// this is the message sent by the validators when they wish to submit their
// signatures over the validator set at a given block height. A validators sign the validator set,
// powers, and Ethereum addresses of the entire validator set at the height of a
// ValsetRequest and submit that signature with this message.
//
// If a sufficient number of validators (66% of voting power) submit ValsetConfirm
// messages with their signatures, it is then possible for anyone to query them from
// the QGB P2P network and submit them to Ethereum to update the validator set.
type ValsetConfirm struct {
	// Universal nonce referencing the `ValSet`.
	Nonce uint64
	// Orchestrator `celes1` account address.
	Orchestrator string
	// Ethereum address, associated to the orchestrator, used to sign the `ValSet`
	// message.
	EthAddress string
	// The `ValSet` message signature.
	Signature string
}

// NewMsgValsetConfirm returns a new msgValSetConfirm.
func NewMsgValsetConfirm(
	nonce uint64,
	ethAddress common.Address,
	validator sdk.AccAddress,
	signature string,
) *ValsetConfirm {
	return &ValsetConfirm{
		Nonce:        nonce,
		Orchestrator: validator.String(),
		EthAddress:   ethAddress.Hex(),
		Signature:    signature,
	}
}

// IsEmptyMsgValsetConfirm takes a msg valset confirm and checks if it is an empty one.
func IsEmptyMsgValsetConfirm(vs ValsetConfirm) bool {
	emptyVsConfirm := ValsetConfirm{}
	return vs.Nonce == emptyVsConfirm.Nonce &&
		vs.EthAddress == emptyVsConfirm.EthAddress &&
		vs.Orchestrator == emptyVsConfirm.Orchestrator &&
		vs.Signature == emptyVsConfirm.Signature
}

// Validate runs validation on the valset confirm to make sure it was well created.
// For now, it only checks if the signature is correct. Can be improved afterwards.
func (msg *ValsetConfirm) Validate(vs celestiatypes.Valset) error {
	if _, err := sdk.AccAddressFromBech32(msg.Orchestrator); err != nil {
		return errors.Wrap(sdkerrors.ErrInvalidAddress, msg.Orchestrator)
	}
	if !common.IsHexAddress(msg.EthAddress) {
		return errors.Wrap(stakingtypes.ErrEVMAddressNotHex, "ethereum address")
	}
	signBytes, err := vs.SignBytes()
	if err != nil {
		return err
	}
	err = evm.ValidateEthereumSignature(signBytes.Bytes(), common.Hex2Bytes(msg.Signature), common.HexToAddress(msg.EthAddress))
	if err != nil {
		return err
	}
	return nil
}

// GetSigners defines whose signature is required.
func (msg *ValsetConfirm) GetSigners() []sdk.AccAddress {
	acc, err := sdk.AccAddressFromBech32(msg.Orchestrator)
	if err != nil {
		panic(err)
	}

	return []sdk.AccAddress{acc}
}
