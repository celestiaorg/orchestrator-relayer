package types

import (
	appTypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/cosmos/cosmos-sdk/codec"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

func RegisterCodec(cdc *codec.LegacyAmino) {
	cdc.RegisterConcrete(&MsgDataCommitmentConfirm{}, "qgb/DataCommitmentConfirm", nil)
	cdc.RegisterConcrete(&MsgValsetConfirm{}, "qgb/MsgValSetConfirm", nil)
}

func RegisterInterfaces(registry cdctypes.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgDataCommitmentConfirm{},
		&MsgValsetConfirm{},
	)

	registry.RegisterInterface(
		"AttestationRequestI",
		(*appTypes.AttestationRequestI)(nil),
		&DataCommitment{},
		&Valset{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}
