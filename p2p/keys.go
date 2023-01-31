package p2p

import "strconv"

// GetDataCommitmentConfirmKey creates a data commitment confirm in the
// format: "/<DataCommitmentConfirmNamespace>/<nonce>:<orchestrator_address>":
// - nonce: in hex format
// - orchestrator address: the `celes1` account address
// Expects the orchestrator address to be a correct address.
func GetDataCommitmentConfirmKey(nonce uint64, orchestratorAddress string) string {
	return "/" + DataCommitmentConfirmNamespace + "/" +
		strconv.FormatUint(nonce, 16) + ":" + orchestratorAddress
}

// GetValsetConfirmKey creates a valset confirm in the
// format: "/<ValsetNamespace>/<nonce>:<orchestrator_address>":
// - nonce: in hex format
// - orchestrator address: the `celes1` account address
// Expects the orchestrator address to be a correct address.
func GetValsetConfirmKey(nonce uint64, orchestratorAddress string) string {
	return "/" + ValsetConfirmNamespace + "/" +
		strconv.FormatUint(nonce, 16) + ":" + orchestratorAddress
}
