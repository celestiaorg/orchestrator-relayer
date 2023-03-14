package p2p

import (
	"fmt"
	"strconv"
	"strings"
)

// GetDataCommitmentConfirmKey creates a data commitment confirm in the
// format: "/<DataCommitmentConfirmNamespace>/<nonce>:<orchestrator_address>":
// - nonce: in hex format
// - evm address: the 0x prefixed orchestrator EVM address in hex format
// Expects the EVM address to be a correct address.
func GetDataCommitmentConfirmKey(nonce uint64, evmAddr string) string {
	return "/" + DataCommitmentConfirmNamespace + "/" +
		strconv.FormatUint(nonce, 16) + ":" + evmAddr
}

// GetValsetConfirmKey creates a valset confirm in the
// format: "/<ValsetNamespace>/<nonce>:<orchestrator_address>":
// - nonce: in hex format
// - evm address: the orchestrator EVM address in hex format
// Expects the EVM address to be a correct address.
func GetValsetConfirmKey(nonce uint64, evmAddr string) string {
	return "/" + ValsetConfirmNamespace + "/" +
		strconv.FormatUint(nonce, 16) + ":" + evmAddr
}

// ParseKey parses a key and returns its fields.
// Will return an error if the key is invalid, is missing some fields, or some fields are empty.
func ParseKey(key string) (namespace string, nonce uint64, evmAddr string, err error) {
	parts := strings.Split(key, "/")
	if len(parts) != 3 {
		return "", 0, "", ErrInvalidConfirmKey
	}
	namespace = parts[1]
	if namespace == "" {
		return "", 0, "", ErrEmptyNamespace
	}
	values := strings.Split(parts[2], ":")
	if len(values) != 2 {
		return "", 0, "", ErrInvalidConfirmKey
	}
	nonce, err = strconv.ParseUint(values[0], 16, 64)
	if err != nil {
		return "", 0, "", fmt.Errorf("failed to parse nonce: %s", err.Error())
	}
	evmAddr = values[1]
	if evmAddr == "" {
		return "", 0, "", ErrEmptyEVMAddr
	}
	return
}
