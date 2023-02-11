package testing

import (
	"crypto/ecdsa"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func NewEVMClient(key *ecdsa.PrivateKey) *evm.Client {
	logger := tmlog.NewNopLogger()
	// specifying an empty RPC endpoint as we will not be testing the methods that require it.
	// the simulated backend doesn't provide an RPC endpoint.
	return evm.NewClient(logger, nil, key, "", 100000000)
}
