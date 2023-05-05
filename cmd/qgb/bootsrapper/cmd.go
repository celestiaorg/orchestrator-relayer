package bootsrapper

import (
	"context"
	"os"
	"time"

	p2pcmd "github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys/p2p"
	"github.com/celestiaorg/orchestrator-relayer/helpers"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/store"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func Command() *cobra.Command {
	orchCmd := &cobra.Command{
		Use:          "bootstrapper",
		Aliases:      []string{"bs"},
		Short:        "QGB P2P network bootstrapper command",
		SilenceUsage: true,
	}

	orchCmd.AddCommand(
		Start(),
		Init(),
		p2pcmd.Root(ServiceNameBootstrapper),
	)

	orchCmd.SetHelpCommand(&cobra.Command{})

	return orchCmd
}

func Start() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "start",
		Short: "Starts the bootstrapper node using the provided home",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseStartFlags(cmd)
			if err != nil {
				return err
			}

			// creating the logger
			logger := tmlog.NewTMLogger(os.Stdout)
			logger.Debug("starting bootstrapper node")

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			// checking if the provided home is already initiated
			isInit := store.IsInit(logger, config.home, store.InitOptions{
				NeedDataStore:   false,
				NeedEVMKeyStore: false,
				NeedP2PKeyStore: true,
			})
			if !isInit {
				return store.ErrNotInited
			}

			// creating the data store
			openOptions := store.OpenOptions{
				HasDataStore:   false,
				HasEVMKeyStore: false,
				HasP2PKeyStore: true,
			}
			s, err := store.OpenStore(logger, config.home, openOptions)
			if err != nil {
				return err
			}

			// get the p2p private key or generate a new one
			privKey, err := p2pcmd.GetP2PKeyOrGenerateNewOne(s.P2PKeyStore, config.p2pNickname)
			if err != nil {
				return err
			}

			// creating the host
			h, err := p2p.CreateHost(config.p2pListenAddr, privKey)
			if err != nil {
				return err
			}
			logger.Info(
				"created host",
				"ID",
				h.ID().String(),
				"Addresses",
				h.Addrs(),
			)

			// creating the data store
			dataStore := dssync.MutexWrap(ds.NewMapDatastore())

			// creating the dht
			dht, err := p2p.NewQgbDHT(ctx, h, dataStore, []peer.AddrInfo{}, logger)
			if err != nil {
				return err
			}

			// Listen for and trap any OS signal to graceful shutdown and exit
			go helpers.TrapSignal(logger, cancel)

			logger.Info("starting bootstrapper")

			ticker := time.NewTicker(time.Minute)
			for {
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
					logger.Info("listening in bootstrapping mode", "peers_connected", dht.RoutingTable().Size())
				}
			}
		},
	}
	return addStartFlags(cmd)
}

func Init() *cobra.Command {
	cmd := cobra.Command{
		Use:   "init",
		Short: "Initialize the QGB bootstrapper store. Passed flags have persisted effect.",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseInitFlags(cmd)
			if err != nil {
				return err
			}

			logger := tmlog.NewTMLogger(os.Stdout)

			initOptions := store.InitOptions{
				NeedDataStore:   false,
				NeedEVMKeyStore: false,
				NeedP2PKeyStore: true,
			}
			isInit := store.IsInit(logger, config.home, initOptions)
			if isInit {
				logger.Info("provided path is already initiated", "path", config.home)
				return nil
			}

			err = store.Init(logger, config.home, initOptions)
			if err != nil {
				return err
			}

			return nil
		},
	}
	return addInitFlags(&cmd)
}
