package testing

import (
	"crypto/ecdsa"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/relayer"
	"github.com/celestiaorg/orchestrator-relayer/rpc"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func NewRelayer(
	node *TestNode,
) *relayer.Relayer {
	logger := tmlog.NewNopLogger()
	appQuerier := rpc.NewAppQuerier(logger, node.CelestiaNetwork.GRPCClient, encoding.MakeConfig(app.ModuleEncodingRegisters...))
	tmQuerier := rpc.NewTmQuerier(node.CelestiaNetwork.Client, logger)
	p2pQuerier := p2p.NewQuerier(node.DHTNetwork.DHTs[0], logger)
	evmClient := NewEVMClient(node.EVMChain.Key)
	r, err := relayer.NewRelayer(tmQuerier, appQuerier, p2pQuerier, evmClient, logger)
	if err != nil {
		panic(err)
	}
	return r
}

func NewEVMClient(key *ecdsa.PrivateKey) *evm.Client {
	logger := tmlog.NewNopLogger()
	// specifying an empty RPC endpoint as we will not be testing the methods that require it.
	// the simulated backend doesn't provide an RPC endpoint.
	return evm.NewClient(logger, nil, key, "", 100000000)
}
