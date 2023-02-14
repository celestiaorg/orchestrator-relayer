package testing

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
)

// EVMTestNetworkChainID the test EVM network chain ID.
const EVMTestNetworkChainID = 1337

// EVMChain is a wrapped Geth simulated backend which will be used to simulate an EVM chain.
// The resulting test chain has always 1337 as a chain ID.
type EVMChain struct {
	Auth         *bind.TransactOpts
	GenesisAlloc core.GenesisAlloc
	Backend      *backends.SimulatedBackend
	Key          *ecdsa.PrivateKey
	ChainID      uint64
}

func NewEVMChain(key *ecdsa.PrivateKey) *EVMChain {
	auth, err := bind.NewKeyedTransactorWithChainID(key, big.NewInt(EVMTestNetworkChainID))
	if err != nil {
		panic(err)
	}
	auth.GasLimit = 10000000000000
	auth.GasPrice = big.NewInt(8750000000)

	genBal := &big.Int{}
	genBal.SetString("999999999999999999999999999999999999999999", 20)
	gAlloc := map[ethcmn.Address]core.GenesisAccount{
		auth.From: {Balance: genBal},
	}

	backend := backends.NewSimulatedBackend(gAlloc, 100000000000000)

	return &EVMChain{
		Auth:         auth,
		GenesisAlloc: gAlloc,
		Backend:      backend,
		Key:          key,
		ChainID:      EVMTestNetworkChainID,
	}
}

// DefaultPeriodicCommitDelay the default delay to running the commit function on the
// simulated network.
const DefaultPeriodicCommitDelay = time.Millisecond

// PeriodicCommit periodically run `commit()` on the simulated network to mine
// the hanging blocks.
// If there are no hanging transactions, the chain will not advance.
func (e *EVMChain) PeriodicCommit(ctx context.Context, delay time.Duration) {
	defer func() {
		if r := recover(); r != nil {
			// we want to exit when the simulated blockchain has stopped instead of panic.
			err, ok := r.(error)
			if !ok || err.Error() != "blockchain is stopped" {
				panic(r)
			}
		}
	}()
	ticker := time.NewTicker(delay)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			e.Backend.Commit()
		}
	}
}

// Close stops the EVM chain backend.
func (e *EVMChain) Close() {
	err := e.Backend.Close()
	if err != nil {
		panic(err)
	}
}
