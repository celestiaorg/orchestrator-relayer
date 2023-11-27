package testing

import (
	"testing"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/store"
	badger "github.com/ipfs/go-ds-badger2"

	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/stretchr/testify/require"

	"github.com/celestiaorg/orchestrator-relayer/helpers"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/orchestrator-relayer/orchestrator"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/relayer"
	"github.com/celestiaorg/orchestrator-relayer/rpc"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func NewRelayer(
	t *testing.T,
	node *TestNode,
) *relayer.Relayer {
	logger := tmlog.NewNopLogger()
	node.CelestiaNetwork.GRPCClient.Close()
	appQuerier := rpc.NewAppQuerier(logger, node.CelestiaNetwork.GRPCAddr, encoding.MakeConfig(app.ModuleEncodingRegisters...))
	require.NoError(t, appQuerier.Start(true))
	t.Cleanup(func() {
		_ = appQuerier.Stop()
	})
	tmQuerier := rpc.NewTmQuerier(node.CelestiaNetwork.RPCAddr, logger)
	tmQuerier.WithClientConn(node.CelestiaNetwork.Client)
	p2pQuerier := p2p.NewQuerier(node.DHTNetwork.DHTs[0], logger)
	ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.ImportECDSA(NodeEVMPrivateKey, "123")
	require.NoError(t, err)
	err = ks.Unlock(acc, "123")
	require.NoError(t, err)
	evmClient := NewEVMClient(ks, &acc)
	retrier := helpers.NewRetrier(logger, 3, 500*time.Millisecond)
	tempDir := t.TempDir()
	sigStore, err := badger.NewDatastore(tempDir, store.DefaultBadgerOptions(tempDir))
	require.NoError(t, err)
	r := relayer.NewRelayer(tmQuerier, appQuerier, p2pQuerier, evmClient, logger, retrier, sigStore, 30*time.Second)
	return r
}

func NewEVMClient(ks *keystore.KeyStore, acc *accounts.Account) *evm.Client {
	logger := tmlog.NewNopLogger()
	// specifying an empty RPC endpoint as we will not be testing the methods that require it.
	// the simulated backend doesn't provide an RPC endpoint.
	return evm.NewClient(logger, nil, ks, acc, "", 100000000)
}

func NewOrchestrator(
	t *testing.T,
	node *TestNode,
) *orchestrator.Orchestrator {
	logger := tmlog.NewNopLogger()
	appQuerier := rpc.NewAppQuerier(logger, node.CelestiaNetwork.GRPCAddr, encoding.MakeConfig(app.ModuleEncodingRegisters...))
	require.NoError(t, appQuerier.Start(true))
	t.Cleanup(func() {
		_ = appQuerier.Stop()
	})
	tmQuerier := rpc.NewTmQuerier(node.CelestiaNetwork.RPCAddr, logger)
	tmQuerier.WithClientConn(node.CelestiaNetwork.Client)
	p2pQuerier := p2p.NewQuerier(node.DHTNetwork.DHTs[0], logger)
	broadcaster := orchestrator.NewBroadcaster(node.DHTNetwork.DHTs[0])
	retrier := helpers.NewRetrier(logger, 3, 500*time.Millisecond)
	ks := keystore.NewKeyStore(t.TempDir(), keystore.LightScryptN, keystore.LightScryptP)
	acc, err := ks.ImportECDSA(NodeEVMPrivateKey, "123")
	require.NoError(t, err)
	err = ks.Unlock(acc, "123")
	require.NoError(t, err)
	orch := orchestrator.New(logger, appQuerier, tmQuerier, p2pQuerier, broadcaster, retrier, ks, &acc)
	return orch
}
