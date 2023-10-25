package types

import (
	"encoding/json"

	"github.com/celestiaorg/celestia-app/x/qgb/types"
)

// MarshalValset Encodes a valset to Json bytes.
func MarshalValset(lv types.Valset) ([]byte, error) {
	encoded, err := json.Marshal(lv)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

// UnmarshalValset Decodes a valset from Json bytes.
func UnmarshalValset(encoded []byte) (types.Valset, error) {
	var valset types.Valset
	err := json.Unmarshal(encoded, &valset)
	if err != nil {
		return types.Valset{}, err
	}
	return valset, nil
}

// IsEmptyValset takes a msg valset and checks if it is empty.
func IsEmptyValset(valset types.Valset) bool {
	emptyVs := types.Valset{}
	return valset.Time.Equal(emptyVs.Time) &&
		valset.Nonce == emptyVs.Nonce &&
		valset.Height == emptyVs.Height &&
		len(valset.Members) == 0
}
