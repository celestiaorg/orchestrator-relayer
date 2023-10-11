package p2p

import (
	"fmt"
	"strconv"
	"strings"
)

// GetDataCommitmentConfirmKey creates a data commitment confirm in the
// format: "/<DataCommitmentConfirmNamespace>/<nonce>:<evm_account>:<data_root_tuple_root>":
// - nonce: in hex format
// - evm address: the 0x prefixed orchestrator EVM address in hex format
// - data root tuple root: is the digest, in a 0x prefixed hex format, that is signed over for a
// data commitment and whose signature is relayed to the Blobstream smart contract.
// Expects the EVM address to be a correct address.
func GetDataCommitmentConfirmKey(nonce uint64, evmAddr string, dataRootTupleRoot string) string {
	return "/" + DataCommitmentConfirmNamespace + "/" +
		strconv.FormatUint(nonce, 16) + ":" +
		evmAddr + ":" + dataRootTupleRoot
}

// GetValsetConfirmKey creates a valset confirm in the
// format: "/<ValsetNamespace>/<nonce>:<evm_account>:<sign_bytes>":
// - nonce: in hex format
// - evm address: the orchestrator EVM address in hex format
// - sign bytes: is the digest, in a 0x prefixed hex format, that is signed over for a valset and
// whose signature is relayed to the Blobstream smart contract.
// Expects the EVM address to be a correct address.
func GetValsetConfirmKey(nonce uint64, evmAddr string, signBytes string) string {
	return "/" + ValsetConfirmNamespace + "/" +
		strconv.FormatUint(nonce, 16) + ":" +
		evmAddr + ":" + signBytes
}

// ParseKey parses a key and returns its fields.
// Will return an error if the key is missing some fields, some fields are empty, or otherwise invalid.
func ParseKey(key string) (namespace string, nonce uint64, evmAddr string, digest string, err error) {
	parts := strings.Split(key, "/")
	if len(parts) != 3 {
		return "", 0, "", "", ErrInvalidConfirmKey
	}
	namespace = parts[1]
	if namespace == "" {
		return "", 0, "", "", ErrEmptyNamespace
	}
	values := strings.Split(parts[2], ":")
	if len(values) != 3 {
		return "", 0, "", "", ErrInvalidConfirmKey
	}
	nonce, err = strconv.ParseUint(values[0], 16, 64)
	if err != nil {
		return "", 0, "", "", fmt.Errorf("failed to parse nonce: %s", err.Error())
	}
	evmAddr = values[1]
	if evmAddr == "" {
		return "", 0, "", "", ErrEmptyEVMAddr
	}
	digest = values[2]
	if digest == "" {
		return "", 0, "", "", ErrEmptyDigest
	}
	return
}
