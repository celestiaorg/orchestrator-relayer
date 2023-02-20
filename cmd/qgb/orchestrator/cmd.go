package orchestrator

import (
	"context"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/helpers"
	"github.com/celestiaorg/orchestrator-relayer/orchestrator"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/rpc/client/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:     "orchestrator <flags>",
		Aliases: []string{"orch"},
		Short:   "Runs the QGB orchestrator to sign attestations",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseOrchestratorFlags(cmd)
			if err != nil {
				return err
			}
			logger := tmlog.NewTMLogger(os.Stdout)

			logger.Debug("initializing orchestrator")

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			encCfg := encoding.MakeConfig(app.ModuleEncodingRegisters...)

			// creating an RPC connection to tendermint
			trpc, err := http.New(config.tendermintRPC, "/websocket")
			if err != nil {
				return err
			}
			err = trpc.Start()
			if err != nil {
				return err
			}
			defer func(trpc *http.HTTP) {
				err := trpc.Stop()
				if err != nil {
					logger.Error(err.Error())
				}
			}(trpc)

			// creating tendermint querier
			tmQuerier := rpc.NewTmQuerier(trpc, logger)
			if err != nil {
				return err
			}

			// creating a grpc connection to Celestia-app
			qgbGRPC, err := grpc.Dial(config.celesGRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return err
			}
			defer func(qgbGRPC *grpc.ClientConn) {
				err := qgbGRPC.Close()
				if err != nil {
					logger.Error(err.Error())
				}
			}(qgbGRPC)

			// creating the application querier
			appQuerier := rpc.NewAppQuerier(logger, qgbGRPC, encCfg)

			// creating the host
			h, err := libp2p.New()
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

			// get the bootstrappers
			var bootstrappers []peer.AddrInfo
			if config.bootstrappers == "" {
				bootstrappers = nil
			} else {
				bs := strings.Split(config.bootstrappers, ",")
				bootstrappers, err = helpers.ParseAddrInfos(logger, bs)
				if err != nil {
					return err
				}
			}

			// creating the dht
			dht, err := p2p.NewQgbDHT(cmd.Context(), h, dataStore, bootstrappers, logger)
			if err != nil {
				return err
			}

			// wait for the dht to have some peers
			err = dht.WaitForPeers(cmd.Context(), time.Hour, 10*time.Second, 1)
			if err != nil {
				return err
			}

			// creating thee p2p querier
			p2pQuerier := p2p.NewQuerier(dht, logger)

			broadcaster := orchestrator.NewBroadcaster(dht)
			if err != nil {
				return err
			}

			retrier := orchestrator.NewRetrier(logger, 5, 15*time.Second)
			orch, err := orchestrator.New(
				logger,
				appQuerier,
				tmQuerier,
				p2pQuerier,
				broadcaster,
				retrier,
				*config.privateKey,
			)
			if err != nil {
				panic(err)
			}

			logger.Debug("starting orchestrator")

			// Listen for and trap any OS signal to gracefully shutdown and exit
			go trapSignal(logger, cancel)

			orch.Start(ctx)

			return nil
		},
	}
	return addOrchestratorFlags(command)
}

// trapSignal will listen for any OS signal and gracefully exit.
func trapSignal(logger tmlog.Logger, cancel context.CancelFunc) {
	sigCh := make(chan os.Signal, 1)

	signal.Notify(sigCh, syscall.SIGTERM)
	signal.Notify(sigCh, syscall.SIGINT)

	sig := <-sigCh
	logger.Info("caught signal; shutting down...", "signal", sig.String())
	cancel()
}
