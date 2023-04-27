package query

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/cmd/qgb/common"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	"github.com/celestiaorg/orchestrator-relayer/types"
	ds "github.com/ipfs/go-datastore"
	dssync "github.com/ipfs/go-datastore/sync"
	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func Command() *cobra.Command {
	queryCmd := &cobra.Command{
		Use:          "query",
		Aliases:      []string{"q"},
		Short:        "Query relevant information from a running QGB",
		SilenceUsage: true,
	}

	queryCmd.AddCommand(
		Signers(),
	)

	queryCmd.SetHelpCommand(&cobra.Command{})

	return queryCmd
}

func Signers() *cobra.Command {
	command := &cobra.Command{
		Use:   "signers <nonce>",
		Args:  cobra.ExactArgs(1),
		Short: "Queries the QGB for attestations signers",
		Long: "Queries the QGB for attestations signers. The nonce is the attestation nonce that the command" +
			" will query signatures for. It should be either a specific nonce starting from 2 and on." +
			" Or, use 'latest' as argument to check the latest attestation nonce",
		RunE: func(cmd *cobra.Command, args []string) error {
			config, err := parseFlags(cmd)
			if err != nil {
				return err
			}

			// creating the logger
			logger := tmlog.NewTMLogger(os.Stdout)
			logger.Debug("initializing queriers")

			ctx, cancel := context.WithCancel(cmd.Context())
			defer cancel()

			stopFuncs := make([]func() error, 1)
			defer func() {
				for _, f := range stopFuncs {
					err := f()
					if err != nil {
						logger.Error(err.Error())
					}
				}
			}()

			// create tm querier and app querier
			tmQuerier, appQuerier, stops, err := common.NewTmAndAppQuerier(logger, config.tendermintRPC, config.celesGRPC)
			stopFuncs = append(stopFuncs, stops...)
			if err != nil {
				return err
			}

			// creating the host
			h, err := libp2p.New()
			if err != nil {
				return err
			}
			addrInfo, err := peer.AddrInfoFromString(config.targetNode)
			if err != nil {
				return err
			}
			for i := 0; i < 5; i++ {
				logger.Debug("connecting to target node...")
				err := h.Connect(ctx, *addrInfo)
				if err != nil {
					logger.Error("couldn't connect to target node", "err", err.Error())
				}
				if err == nil {
					logger.Debug("connected to target node")
					break
				}
				time.Sleep(5 * time.Second)
			}

			// creating the data store
			dataStore := dssync.MutexWrap(ds.NewMapDatastore())

			// creating the dht
			dht, err := p2p.NewQgbDHT(cmd.Context(), h, dataStore, []peer.AddrInfo{}, logger)
			if err != nil {
				return err
			}

			// creating the p2p querier
			p2pQuerier := p2p.NewQuerier(dht, logger)

			nonce, err := parseNonce(ctx, appQuerier, args[0])
			if err != nil {
				return err
			}
			if nonce == 1 {
				return fmt.Errorf("nonce 1 doesn't need to be signed. signatures start from nonce 2")
			}

			err = getSignaturesAndPrintThem(ctx, logger, appQuerier, tmQuerier, p2pQuerier, nonce)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return addFlags(command)
}

func getSignaturesAndPrintThem(ctx context.Context, logger tmlog.Logger, appQuerier *rpc.AppQuerier, tmQuerier *rpc.TmQuerier, p2pQuerier *p2p.Querier, nonce uint64) error {
	logger.Info("getting signatures for nonce", "nonce", nonce)

	lastValset, err := appQuerier.QueryLastValsetBeforeNonce(ctx, nonce)
	if err != nil {
		return err
	}

	att, err := appQuerier.QueryAttestationByNonce(ctx, nonce)
	if err != nil {
		return err
	}
	if att == nil {
		return celestiatypes.ErrAttestationNotFound
	}

	switch att.Type() {
	case celestiatypes.ValsetRequestType:
		vs, ok := att.(*celestiatypes.Valset)
		if !ok {
			return errors.Wrap(celestiatypes.ErrAttestationNotValsetRequest, strconv.FormatUint(nonce, 10))
		}
		signBytes, err := vs.SignBytes()
		if err != nil {
			return err
		}
		confirms, err := p2pQuerier.QueryValsetConfirms(ctx, nonce, *lastValset, signBytes.Hex())
		if err != nil {
			return err
		}
		confirmsMap := make(map[string]string)
		for _, confirm := range confirms {
			confirmsMap[confirm.EthAddress] = confirm.Signature
		}
		printConfirms(logger, confirmsMap, lastValset)
	case celestiatypes.DataCommitmentRequestType:
		dc, ok := att.(*celestiatypes.DataCommitment)
		if !ok {
			return errors.Wrap(types.ErrAttestationNotDataCommitmentRequest, strconv.FormatUint(nonce, 10))
		}
		commitment, err := tmQuerier.QueryCommitment(
			ctx,
			dc.BeginBlock,
			dc.EndBlock,
		)
		if err != nil {
			return err
		}
		dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(dc.Nonce)), commitment)
		confirms, err := p2pQuerier.QueryDataCommitmentConfirms(ctx, *lastValset, nonce, dataRootHash.Hex())
		if err != nil {
			return err
		}
		confirmsMap := make(map[string]string)
		for _, confirm := range confirms {
			confirmsMap[confirm.EthAddress] = confirm.Signature
		}
		printConfirms(logger, confirmsMap, lastValset)
	default:
		return errors.Wrap(types.ErrUnknownAttestationType, strconv.FormatUint(nonce, 10))
	}
	return nil
}

func parseNonce(ctx context.Context, querier *rpc.AppQuerier, nonce string) (uint64, error) {
	switch nonce {
	case "latest":
		return querier.QueryLatestAttestationNonce(ctx)
	default:
		return strconv.ParseUint(nonce, 10, 0)
	}
}

func printConfirms(logger tmlog.Logger, confirmsMap map[string]string, valset *celestiatypes.Valset) {
	signers := make(map[string]string)
	missingSigners := make([]string, 0)

	for _, validator := range valset.Members {
		val, ok := confirmsMap[validator.EvmAddress]
		if ok {
			signers[validator.EvmAddress] = val
			continue
		}
		missingSigners = append(missingSigners, validator.EvmAddress)
	}

	logger.Info("orchestrators that signed the attestation", "count", len(signers))
	i := 0
	for addr, sig := range signers {
		logger.Info(addr, "number", i, "signature", sig)
		i++
	}

	logger.Info("orchestrators that missed signing the attestation", "count", len(missingSigners))
	for i, addr := range missingSigners {
		logger.Info(addr, "number", i)
	}
}
