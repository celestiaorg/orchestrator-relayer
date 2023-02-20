package relayer

import (
	"os"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/helpers"

	"github.com/celestiaorg/celestia-app/app"
	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/relayer"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	wrapper "github.com/celestiaorg/quantum-gravity-bridge/wrappers/QuantumGravityBridge.sol"
	"github.com/ethereum/go-ethereum/ethclient"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/rpc/client/http"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func Command() *cobra.Command {
	command := &cobra.Command{
		Use:   "relayer <flags>",
		Short: "Runs the QGB relayer to submit attestations to the target EVM chain",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseRelayerFlags(cmd)
			if err != nil {
				return err
			}

			// creating the logger
			logger := tmlog.NewTMLogger(os.Stdout)
			logger.Debug("initializing relayer")

			// connecting to a QGB contract
			ethClient, err := ethclient.Dial(config.evmRPC)
			if err != nil {
				return err
			}
			qgbWrapper, err := wrapper.NewQuantumGravityBridge(config.contractAddr, ethClient)
			if err != nil {
				return err
			}

			// creating Celestia-app configuration
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
			h, err := p2p.CreateHost(config.p2pListenAddr, config.p2pIdentity)
			if err != nil {
				return err
			}
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

			// creating the p2p querier
			p2pQuerier := p2p.NewQuerier(dht, logger)

			relay, err := relayer.NewRelayer(
				tmQuerier,
				appQuerier,
				p2pQuerier,
				evm.NewClient(
					logger,
					qgbWrapper,
					config.evmPrivateKey,
					config.evmRPC,
					config.evmGasLimit,
				),
				logger,
			)
			if err != nil {
				return err
			}

			err = relay.Start(cmd.Context())
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			return nil
		},
	}
	return addRelayerFlags(command)
}
