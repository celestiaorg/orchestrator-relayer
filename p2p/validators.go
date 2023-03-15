package p2p

import (
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/types"
	"github.com/ethereum/go-ethereum/common"
)

// ValsetConfirmValidator runs stateless checks on valset confirms when submitting them to the DHT.
type ValsetConfirmValidator struct{}

// Validate runs stateless checks on the provided confirm key and value.
func (vcv ValsetConfirmValidator) Validate(key string, value []byte) error {
	namespace, _, evmAddr, err := ParseKey(key)
	if err != nil {
		return err
	}

	// check if namespace is of valset confirms
	if namespace != ValsetConfirmNamespace {
		return ErrInvalidConfirmNamespace
	}

	// check if the evm address is a valid eth address
	if !common.IsHexAddress(evmAddr) {
		return ErrInvalidEVMAddress
	}

	vsc, err := types.UnmarshalValsetConfirm(value)
	if err != nil {
		return err
	}

	// check if the evm address in the key is the same as the one in the confirm
	if !strings.EqualFold(vsc.EthAddress, evmAddr) {
		return ErrNotTheSameEVMAddress
	}

	// strip the 0x from the signBytes, if exists, to create its corresponding byte slice
	signBytes := vsc.SignBytes
	// we want to make sure that len(signBytes) > 2 to avoid slice bounds out of range
	// however, we don't care at this level if the signBytes is invalid as it will be checked below.
	if len(signBytes) > 2 && signBytes[:2] == "0x" {
		signBytes = signBytes[2:]
	}
	bSignBytes, err := hex.DecodeString(signBytes)
	if err != nil {
		return err
	}

	// strip the 0x from the signature, if exists, to create its corresponding byte slice
	signature := vsc.Signature
	// we want to make sure that len(signature) > 2 to avoid slice bounds out of range
	// however, we don't care at this level if the signature is invalid as it will be checked below.
	if len(signature) > 2 && signature[:2] == "0x" {
		signature = signature[2:]
	}
	bSignature, err := hex.DecodeString(signature)
	if err != nil {
		return err
	}

	// check that the provided signature was created by the provided evm address
	err = evm.ValidateEthereumSignature(bSignBytes, bSignature, common.HexToAddress(evmAddr))
	if err != nil {
		return err
	}

	return nil
}

// Select selects a valid dht confirm value from multiple ones.
// returns an error of no valid value is found.
func (vcv ValsetConfirmValidator) Select(key string, values [][]byte) (int, error) {
	if len(values) == 0 {
		return 0, ErrNoValues
	}
	for index, value := range values {
		// choose the first correct value
		if err := vcv.Validate(key, value); err == nil {
			return index, nil
		}
	}
	return 0, ErrNoValidValueFound
}

// DataCommitmentConfirmValidator runs stateless checks on data commitment confirms when submitting to the DHT.
type DataCommitmentConfirmValidator struct{}

// Validate runs stateless checks on the provided confirm key and value.
func (dcv DataCommitmentConfirmValidator) Validate(key string, value []byte) error {
	namespace, nonce, evmAddr, err := ParseKey(key)
	if err != nil {
		return err
	}

	// check if namespace is of valset confirms
	if namespace != DataCommitmentConfirmNamespace {
		return ErrInvalidConfirmNamespace
	}

	// check if the evm address is a valid eth address
	if !common.IsHexAddress(evmAddr) {
		return ErrInvalidEVMAddress
	}

	dcc, err := types.UnmarshalDataCommitmentConfirm(value)
	if err != nil {
		return err
	}

	// check if the evm address in the key is the same as the one in the confirm
	if !strings.EqualFold(dcc.EthAddress, evmAddr) {
		return ErrNotTheSameEVMAddress
	}

	// strip the 0x from the commitment, if exists, to create its corresponding byte slice
	// we want to make sure that len(commitment) > 2 to avoid slice bounds out of range
	// however, we don't care at this level if the commitment is invalid as it will be checked below.
	commitment := dcc.Commitment
	if len(commitment) > 2 && commitment[:2] == "0x" {
		commitment = commitment[2:]
	}
	bCommitment, err := hex.DecodeString(commitment)
	if err != nil {
		return err
	}

	// strip the 0x from the signature, if exists, to create its corresponding byte slice
	signature := dcc.Signature
	// we want to make sure that len(signature) > 2 to avoid slice bounds out of range
	// however, we don't care at this level if the signature is invalid as it will be checked below.
	if len(signature) > 2 && signature[:2] == "0x" {
		signature = signature[2:]
	}
	bSignature, err := hex.DecodeString(signature)
	if err != nil {
		return err
	}

	// check that the provided signature was created by the provided evm address
	dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(nonce)), bCommitment)
	err = evm.ValidateEthereumSignature(dataRootHash.Bytes(), bSignature, common.HexToAddress(evmAddr))
	if err != nil {
		return err
	}

	return nil
}

// Select selects a valid dht confirm value from multiple ones.
// returns an error of no valid value is found.
func (dcv DataCommitmentConfirmValidator) Select(key string, values [][]byte) (int, error) {
	if len(values) == 0 {
		return 0, ErrNoValues
	}
	for index, value := range values {
		// choose the first correct value
		if err := dcv.Validate(key, value); err == nil {
			return index, nil
		}
	}
	return 0, ErrNoValidValueFound
}
