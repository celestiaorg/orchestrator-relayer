package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/host"

	evm2 "github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys/evm"
	"github.com/celestiaorg/orchestrator-relayer/store"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"
	dssync "github.com/ipfs/go-datastore/sync"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	common2 "github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys/p2p"
	"github.com/celestiaorg/orchestrator-relayer/helpers"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	keystore2 "github.com/ipfs/boxo/keystore"
	ds "github.com/ipfs/go-datastore"
	"github.com/libp2p/go-libp2p/core/peer"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

// NewTmAndAppQuerier helper function that creates a new TmQuerier and AppQuerier and registers their stop functions in the
// stopFuncs slice.
func NewTmAndAppQuerier(logger tmlog.Logger, tendermintRPC string, celesGRPC string) (*rpc.TmQuerier, *rpc.AppQuerier, []func() error, error) {
	// load app encoding configuration
	encCfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)

	// creating tendermint querier
	tmQuerier := rpc.NewTmQuerier(tendermintRPC, logger)
	err := tmQuerier.Start()
	if err != nil {
		return nil, nil, nil, err
	}
	stopFuncs := make([]func() error, 0)
	stopFuncs = append(stopFuncs, func() error {
		err := tmQuerier.Stop()
		if err != nil {
			return err
		}
		return nil
	})

	// creating the application querier
	appQuerier := rpc.NewAppQuerier(logger, celesGRPC, encCfg)
	err = appQuerier.Start()
	if err != nil {
		return nil, nil, stopFuncs, err
	}
	stopFuncs = append(stopFuncs, func() error {
		err = appQuerier.Stop()
		if err != nil {
			return err
		}
		return nil
	})

	return tmQuerier, appQuerier, stopFuncs, nil
}

// CreateDHTAndWaitForPeers helper function that creates a new QGB DHT and waits for some peers to connect to it.
func CreateDHTAndWaitForPeers(
	ctx context.Context,
	logger tmlog.Logger,
	p2pKeyStore *keystore2.FSKeystore,
	p2pNickname string,
	p2pListenAddr string,
	bootstrappers string,
	dataStore ds.Batching,
) (*p2p.QgbDHT, error) {
	// get the p2p private key or generate a new one
	privKey, err := common2.GetP2PKeyOrGenerateNewOne(p2pKeyStore, p2pNickname)
	if err != nil {
		return nil, err
	}

	// creating the host
	h, err := p2p.CreateHost(p2pListenAddr, privKey)
	if err != nil {
		return nil, err
	}
	logger.Info("created host")

	prettyPrintHost(h)

	// get the bootstrappers
	var aIBootstrappers []peer.AddrInfo
	if bootstrappers == "" {
		aIBootstrappers = nil
	} else {
		bs := strings.Split(bootstrappers, ",")
		aIBootstrappers, err = helpers.ParseAddrInfos(logger, bs)
		if err != nil {
			return nil, err
		}
	}

	// creating the dht
	dht, err := p2p.NewQgbDHT(ctx, h, dataStore, aIBootstrappers, logger)
	if err != nil {
		return nil, err
	}

	// wait for the dht to have some peers
	err = dht.WaitForPeers(ctx, 5*time.Minute, 10*time.Second, 1)
	if err != nil {
		return nil, err
	}
	return dht, nil
}

func prettyPrintHost(h host.Host) {
	fmt.Printf("ID: %s\n", h.ID().String())
	fmt.Println("Listen addresses:")
	for _, addr := range h.Addrs() {
		fmt.Printf("\t%s\n", addr.String())
	}
}

// InitBase initializes the base components for the orchestrator and relayer.
func InitBase(
	ctx context.Context,
	logger tmlog.Logger,
	tendermintRPC, celesGRPC, home, evmAccAddress, evmPassphrase, p2pNickname, p2pListenAddr, bootstrappers string,
) (*rpc.TmQuerier, *rpc.AppQuerier, *p2p.Querier, *helpers.Retrier, *keystore.KeyStore, *accounts.Account, []func() error, error) {
	stopFuncs := make([]func() error, 0)

	tmQuerier, appQuerier, stops, err := NewTmAndAppQuerier(logger, tendermintRPC, celesGRPC)
	stopFuncs = append(stopFuncs, stops...)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, stopFuncs, err
	}

	// checking if the provided home is already initiated
	isInit := store.IsInit(logger, home, store.InitOptions{NeedDataStore: true, NeedEVMKeyStore: true, NeedP2PKeyStore: true})
	if !isInit {
		return nil, nil, nil, nil, nil, nil, stopFuncs, store.ErrNotInited
	}

	// creating the data store
	openOptions := store.OpenOptions{
		HasDataStore:   true,
		BadgerOptions:  store.DefaultBadgerOptions(home),
		HasEVMKeyStore: true,
		HasP2PKeyStore: true,
	}
	s, err := store.OpenStore(logger, home, openOptions)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, stopFuncs, err
	}
	stopFuncs = append(stopFuncs, func() error { return s.Close(logger, openOptions) })

	logger.Info("loading EVM account", "address", evmAccAddress)

	acc, err := evm2.GetAccountFromStoreAndUnlockIt(s.EVMKeyStore, evmAccAddress, evmPassphrase)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, stopFuncs, err
	}
	stopFuncs = append(stopFuncs, func() error { return s.EVMKeyStore.Lock(acc.Address) })

	// creating the data store
	dataStore := dssync.MutexWrap(s.DataStore)

	dht, err := CreateDHTAndWaitForPeers(ctx, logger, s.P2PKeyStore, p2pNickname, p2pListenAddr, bootstrappers, dataStore)
	if err != nil {
		return nil, nil, nil, nil, nil, nil, stopFuncs, err
	}
	stopFuncs = append(stopFuncs, func() error { return dht.Close() })

	// creating the p2p querier
	p2pQuerier := p2p.NewQuerier(dht, logger)
	retrier := helpers.NewRetrier(logger, 6, time.Minute)

	return tmQuerier, appQuerier, p2pQuerier, retrier, s.EVMKeyStore, &acc, stopFuncs, nil
}
