package relayer

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/keys"

	"github.com/celestiaorg/orchestrator-relayer/helpers"

	"github.com/libp2p/go-libp2p/core/peer"

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

			// creating tendermint querier
			tmQuerier := rpc.NewTmQuerier(config.tendermintRPC, logger)
			err = tmQuerier.Start()
			if err != nil {
				return err
			}
			defer func() {
				err := tmQuerier.Stop()
				if err != nil {
					logger.Error(err.Error())
				}
			}()

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			// creating the application querier
			appQuerier := rpc.NewAppQuerier(logger, config.celesGRPC, encCfg)
			err = appQuerier.Start()
			if err != nil {
				return err
			}
			defer func() {
				err := appQuerier.Stop()
				if err != nil {
					logger.Error(err.Error())
				}
			}()

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
			dht, err := p2p.NewQgbDHT(ctx, h, dataStore, bootstrappers, logger)
			if err != nil {
				return err
			}

			// wait for the dht to have some peers
			err = dht.WaitForPeers(ctx, 2*time.Minute, 10*time.Second, 1)
			if err != nil {
				return err
			}

			// creating the p2p querier
			p2pQuerier := p2p.NewQuerier(dht, logger)
			retrier := helpers.NewRetrier(logger, 5, 15*time.Second)
			relay := relayer.NewRelayer(
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
				retrier,
			)

			// Listen for and trap any OS signal to gracefully shutdown and exit
			go helpers.TrapSignal(logger, cancel)

			err = relay.Start(ctx)
			if err != nil {
				logger.Error(err.Error())
				return err
			}
			return nil
		},
	}
	command.AddCommand(keys.Command())
	return addRelayerFlags(command)
}
