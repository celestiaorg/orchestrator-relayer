package types

import (
	"encoding/json"
	"time"

	"github.com/celestiaorg/celestia-app/x/qgb/types"
)

// LatestValset a replica of the types.Valset to omit marshalling `time` as it bears different results on different machines.
type LatestValset struct {
	// Universal nonce defined under:
	// https://github.com/celestiaorg/celestia-app/pull/464
	Nonce uint64 `json:"nonce,omitempty"`
	// List of BridgeValidator containing the current validator set.
	Members []types.BridgeValidator `json:"members"`
	// Current chain height
	Height uint64 `json:"height,omitempty"`
}

func (v LatestValset) ToValset() *types.Valset {
	return &types.Valset{
		Nonce:   v.Nonce,
		Members: v.Members,
		Height:  v.Height,
		Time:    time.UnixMicro(1), // it's alright to put an arbitrary value in here since the time is not used in hash creation nor the threshold.
	}
}

func ToLatestValset(vs types.Valset) *LatestValset {
	return &LatestValset{
		Nonce:   vs.Nonce,
		Members: vs.Members,
		Height:  vs.Height,
	}
}

// MarshalLatestValset Encodes a valset to Json bytes.
func MarshalLatestValset(lv LatestValset) ([]byte, error) {
	encoded, err := json.Marshal(lv)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

// UnmarshalLatestValset Decodes a valset from Json bytes.
func UnmarshalLatestValset(encoded []byte) (LatestValset, error) {
	var valset LatestValset
	err := json.Unmarshal(encoded, &valset)
	if err != nil {
		return LatestValset{}, err
	}
	return valset, nil
}

// IsEmptyLatestValset takes a valset and checks if it is empty.
func IsEmptyLatestValset(latestValset LatestValset) bool {
	emptyVs := types.Valset{}
	return latestValset.Nonce == emptyVs.Nonce &&
		latestValset.Height == emptyVs.Height &&
		len(latestValset.Members) == 0
}

func IsValsetEqualToLatestValset(vs types.Valset, lvs LatestValset) bool {
	for index, value := range vs.Members {
		if value.EvmAddress != lvs.Members[index].EvmAddress ||
			value.Power != lvs.Members[index].Power {
			return false
		}
	}
	return vs.Nonce == lvs.Nonce && vs.Height == lvs.Height
}
