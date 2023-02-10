package types

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/celestiaorg/celestia-app/x/qgb/types"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// DataCommitmentConfirm describes a data commitment for a set of blocks.
type DataCommitmentConfirm struct {
	// Signature over the commitment, the range of blocks, the validator address
	// and the Ethereum address.
	Signature string
	// Hex `0x` encoded Ethereum public key that will be used by this validator on
	// Ethereum.
	EthAddress string
	// Merkle root over a merkle tree containing the data roots of a set of
	// blocks.
	Commitment string
}

// NewDataCommitmentConfirm creates a new NewDataCommitmentConfirm.
func NewDataCommitmentConfirm(
	commitment string,
	signature string,
	ethAddress ethcmn.Address,
) *DataCommitmentConfirm {
	return &DataCommitmentConfirm{
		Commitment: commitment,
		Signature:  signature,
		EthAddress: ethAddress.Hex(),
	}
}

// MarshalDataCommitmentConfirm Encodes a data commitment confirm to Json bytes.
func MarshalDataCommitmentConfirm(dcc DataCommitmentConfirm) ([]byte, error) {
	encoded, err := json.Marshal(dcc)
	if err != nil {
		return nil, err
	}
	return encoded, nil
}

// UnmarshalDataCommitmentConfirm Decodes a data commitment confirm from Json bytes.
func UnmarshalDataCommitmentConfirm(encoded []byte) (DataCommitmentConfirm, error) {
	var dataCommitmentConfirm DataCommitmentConfirm
	err := json.Unmarshal(encoded, &dataCommitmentConfirm)
	if err != nil {
		return DataCommitmentConfirm{}, err
	}
	return dataCommitmentConfirm, nil
}

func IsEmptyMsgDataCommitmentConfirm(dcc DataCommitmentConfirm) bool {
	emptyDcc := DataCommitmentConfirm{}
	return dcc.EthAddress == emptyDcc.EthAddress &&
		dcc.Commitment == emptyDcc.Commitment &&
		dcc.Signature == emptyDcc.Signature
}

// DataCommitmentTupleRootSignBytes EncodeDomainSeparatedDataCommitment takes the required input data and
// produces the required signature to confirm a validator set update on the QGB Ethereum contract.
// This value will then be signed before being submitted to Cosmos, verified, and then relayed to Ethereum.
func DataCommitmentTupleRootSignBytes(nonce *big.Int, commitment []byte) ethcmn.Hash {
	var dataCommitment [32]uint8
	copy(dataCommitment[:], commitment)

	// the word 'transactionBatch' needs to be the same as the 'name' above in the DataCommitmentConfirmABIJSON
	// but other than that it's a constant that has no impact on the output. This is because
	// it gets encoded as a function name which we must then discard.
	bytes, err := types.InternalQGBabi.Pack(
		"domainSeparateDataRootTupleRoot",
		types.DcDomainSeparator,
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
