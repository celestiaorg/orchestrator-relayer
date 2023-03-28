package types

import (
	"encoding/json"

	"github.com/ethereum/go-ethereum/common"
)

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
	// Ethereum address, associated to the orchestrator, used to sign the `ValSet`
	// message.
	EthAddress string
	// The `ValSet` message signature.
	Signature string
}

// NewValsetConfirm returns a new msgValSetConfirm.
func NewValsetConfirm(
	ethAddress common.Address,
	signature string,
) *ValsetConfirm {
	return &ValsetConfirm{
		EthAddress: ethAddress.Hex(),
		Signature:  signature,
	}
}

// MarshalValsetConfirm Encodes a valset confirm to Json bytes.
func MarshalValsetConfirm(vs ValsetConfirm) ([]byte, error) {
	encoded, err := json.Marshal(vs)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

// UnmarshalValsetConfirm Decodes a valset confirm from Json bytes.
func UnmarshalValsetConfirm(encoded []byte) (ValsetConfirm, error) {
	var valsetConfirm ValsetConfirm
	err := json.Unmarshal(encoded, &valsetConfirm)
	if err != nil {
		return ValsetConfirm{}, err
	}
	return valsetConfirm, nil
}

// IsEmptyValsetConfirm takes a msg valset confirm and checks if it is an empty one.
func IsEmptyValsetConfirm(vs ValsetConfirm) bool {
	emptyVsConfirm := ValsetConfirm{}
	return vs.EthAddress == emptyVsConfirm.EthAddress &&
		vs.Signature == emptyVsConfirm.Signature
}
