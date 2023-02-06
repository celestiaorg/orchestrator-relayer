package p2p

import "strconv"

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
