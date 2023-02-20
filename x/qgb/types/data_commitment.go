package types

import appTypes "github.com/celestiaorg/celestia-app/x/qgb/types"

var _ appTypes.AttestationRequestI = &DataCommitment{}

// NewDataCommitment creates a new DataCommitment.
func NewDataCommitment(
	nonce uint64,
	beginBlock uint64,
	endBlock uint64,
) *DataCommitment {
	return &DataCommitment{
		Nonce:      nonce,
		BeginBlock: beginBlock,
		EndBlock:   endBlock,
	}
}

func (m *DataCommitment) Type() appTypes.AttestationType {
	return appTypes.DataCommitmentRequestType
}
