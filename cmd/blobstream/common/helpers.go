package common

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/celestiaorg/orchestrator-relayer/store"

	"github.com/libp2p/go-libp2p/core/host"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	common2 "github.com/celestiaorg/orchestrator-relayer/cmd/blobstream/keys/p2p"
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
func NewTmAndAppQuerier(logger tmlog.Logger, tendermintRPC string, celesGRPC string, grpcInsecure bool) (*rpc.TmQuerier, *rpc.AppQuerier, []func() error, error) {
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
	err = appQuerier.Start(grpcInsecure)
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

// CreateDHTAndWaitForPeers helper function that creates a new Blobstream DHT and waits for some peers to connect to it.
func CreateDHTAndWaitForPeers(
	ctx context.Context,
	logger tmlog.Logger,
	p2pKeyStore *keystore2.FSKeystore,
	p2pNickname string,
	p2pListenAddr string,
	bootstrappers string,
	dataStore ds.Batching,
	registerer prometheus.Registerer,
) (*p2p.BlobstreamDHT, error) {
	// get the p2p private key or generate a new one
	privKey, err := common2.GetP2PKeyOrGenerateNewOne(p2pKeyStore, p2pNickname)
	if err != nil {
		return nil, err
	}

	// creating the host
	h, err := p2p.CreateHost(p2pListenAddr, privKey, registerer)
	if err != nil {
		return nil, err
	}
	logger.Info("created P2P host")

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
	dht, err := p2p.NewBlobstreamDHT(ctx, h, dataStore, aIBootstrappers, logger)
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

func OpenStore(logger tmlog.Logger, home string, openOptions store.OpenOptions) (*store.Store, []func() error, error) {
	stopFuncs := make([]func() error, 0)

	// checking if the provided home is already initiated
	isInit := store.IsInit(logger, home, store.InitOptions{
		NeedDataStore:      openOptions.HasDataStore,
		NeedEVMKeyStore:    openOptions.HasEVMKeyStore,
		NeedP2PKeyStore:    openOptions.HasP2PKeyStore,
		NeedSignatureStore: openOptions.HasSignatureStore,
	})
	if !isInit {
		return nil, stopFuncs, store.ErrNotInited
	}

	// creating the data store
	s, err := store.OpenStore(logger, home, openOptions)
	if err != nil {
		return nil, stopFuncs, err
	}
	stopFuncs = append(stopFuncs, func() error { return s.Close(logger, openOptions) })

	return s, stopFuncs, nil
}

func prettyPrintHost(h host.Host) {
	fmt.Printf("ID: %s\n", h.ID().String())
	fmt.Println("Listen addresses:")
	for _, addr := range h.Addrs() {
		fmt.Printf("\t%s\n", addr.String())
	}
}
