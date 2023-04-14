package common

import (
	"context"
	"strings"
	"time"

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
func NewTmAndAppQuerier(logger tmlog.Logger, tendermintRPC string, celesGRPC string, stopFuncs []func() error) (*rpc.TmQuerier, *rpc.AppQuerier, error) {
	// load app encoding configuration
	encCfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)

	// creating tendermint querier
	tmQuerier := rpc.NewTmQuerier(tendermintRPC, logger)
	err := tmQuerier.Start()
	if err != nil {
		return nil, nil, err
	}
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
		return nil, nil, err
	}
	stopFuncs = append(stopFuncs, func() error {
		err = appQuerier.Stop()
		if err != nil {
			return err
		}
		return nil
	})

	return tmQuerier, appQuerier, nil
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
	logger.Info(
		"created host",
		"ID",
		h.ID().String(),
		"Addresses",
		h.Addrs(),
	)

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
