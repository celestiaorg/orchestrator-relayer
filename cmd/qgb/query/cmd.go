package query

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"strconv"
	"time"

	common2 "github.com/ethereum/go-ethereum/common"

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
		Signature(),
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

			stopFuncs := make([]func() error, 0, 1)
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

			err = getSignaturesAndPrintThem(ctx, logger, appQuerier, tmQuerier, p2pQuerier, nonce, config.outputFile)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return addFlags(command)
}

type signature struct {
	EvmAddress string `json:"evmAddress"`
	Signature  string `json:"signature"`
	Signed     bool   `json:"signed"`
}
type queryOutput struct {
	Signatures        []signature `json:"signatures"`
	Nonce             uint64      `json:"nonce"`
	MajorityThreshold uint64      `json:"majority_threshold"`
	CurrentThreshold  uint64      `json:"current_threshold"`
	CanRelay          bool        `json:"can_relay"`
}

func getSignaturesAndPrintThem(
	ctx context.Context,
	logger tmlog.Logger,
	appQuerier *rpc.AppQuerier,
	tmQuerier *rpc.TmQuerier,
	p2pQuerier *p2p.Querier,
	nonce uint64,
	outputFile string,
) error {
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
		qOutput := toQueryOutput(toValsetConfirmsMap(confirms), nonce, *lastValset)
		if outputFile == "" {
			printConfirms(logger, qOutput)
		} else {
			err := writeConfirmsToJSONFile(logger, qOutput, outputFile)
			if err != nil {
				return err
			}
		}
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
		qOutput := toQueryOutput(toDataCommitmentConfirmsMap(confirms), nonce, *lastValset)
		if outputFile == "" {
			printConfirms(logger, qOutput)
		} else {
			err := writeConfirmsToJSONFile(logger, qOutput, outputFile)
			if err != nil {
				return err
			}
		}
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

func toValsetConfirmsMap(confirms []types.ValsetConfirm) map[string]string {
	// create a map for the signatures to get them easily
	confirmsMap := make(map[string]string)
	for _, confirm := range confirms {
		confirmsMap[confirm.EthAddress] = confirm.Signature
	}
	return confirmsMap
}

func toDataCommitmentConfirmsMap(confirms []types.DataCommitmentConfirm) map[string]string {
	// create a map for the signatures to get them easily
	confirmsMap := make(map[string]string)
	for _, confirm := range confirms {
		confirmsMap[confirm.EthAddress] = confirm.Signature
	}
	return confirmsMap
}

func toQueryOutput(confirmsMap map[string]string, nonce uint64, lastValset celestiatypes.Valset) queryOutput {
	currThreshold := uint64(0)
	signatures := make([]signature, len(lastValset.Members))
	// create the signature slice to be used for outputting the data
	for key, val := range lastValset.Members {
		sig, found := confirmsMap[val.EvmAddress]
		if found {
			signatures[key] = signature{
				EvmAddress: val.EvmAddress,
				Signature:  sig,
				Signed:     true,
			}
			currThreshold += val.Power
		} else {
			signatures[key] = signature{
				EvmAddress: val.EvmAddress,
				Signature:  "",
				Signed:     false,
			}
		}
	}
	return queryOutput{
		Signatures:        signatures,
		Nonce:             nonce,
		MajorityThreshold: lastValset.TwoThirdsThreshold(),
		CurrentThreshold:  currThreshold,
		CanRelay:          lastValset.TwoThirdsThreshold() <= currThreshold,
	}
}

func printConfirms(logger tmlog.Logger, qOutput queryOutput) {
	logger.Info(
		"query output",
		"nonce",
		qOutput.Nonce,
		"majority_threshold",
		qOutput.MajorityThreshold,
		"current_threshold",
		qOutput.CurrentThreshold,
		"can_relay",
		qOutput.CanRelay,
	)
	logger.Info("orchestrators that signed the attestation")
	for _, sig := range qOutput.Signatures {
		if sig.Signed {
			logger.Info(sig.EvmAddress, "signed", sig.Signed, "signature", sig.Signature)
		}
	}
	logger.Info("orchestrators that missed signing the attestation")
	for _, sig := range qOutput.Signatures {
		if !sig.Signed {
			logger.Info(sig.EvmAddress, "signed", sig.Signed)
		}
	}
	logger.Info("done")
}

func writeConfirmsToJSONFile(logger tmlog.Logger, qOutput queryOutput, outputFile string) error {
	logger.Info("writing confirms json file", "path", outputFile)

	file, err := os.OpenFile(outputFile, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			logger.Error("failed to close file", "err", err.Error())
		}
	}(file)

	encoder := json.NewEncoder(file)
	err = encoder.Encode(qOutput)
	if err != nil {
		return err
	}

	logger.Info("output written to file successfully", "path", outputFile)
	return nil
}

func Signature() *cobra.Command {
	command := &cobra.Command{
		Use:   "signature <nonce> <evm_address>",
		Args:  cobra.ExactArgs(2),
		Short: "Queries a specific signature referenced by an EVM address and a nonce",
		Long: "Queries a specific signature referenced by an EVM address and a nonce. The nonce is the attestation" +
			" nonce that the command will query signatures for. The EVM address is the address registered by the validator " +
			"in the staking module.",
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

			stopFuncs := make([]func() error, 0, 1)
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

			if !common2.IsHexAddress(args[1]) {
				return fmt.Errorf("invalid EVM address provided")
			}

			err = getSignatureAndPrintIt(ctx, logger, appQuerier, tmQuerier, p2pQuerier, args[1], nonce)
			if err != nil {
				return err
			}
			return nil
		},
	}
	return addFlags(command)
}

func getSignatureAndPrintIt(
	ctx context.Context,
	logger tmlog.Logger,
	appQuerier *rpc.AppQuerier,
	tmQuerier *rpc.TmQuerier,
	p2pQuerier *p2p.Querier,
	evmAddress string,
	nonce uint64,
) error {
	logger.Info("getting signature for address and nonce", "nonce", nonce, "evm_address", evmAddress)

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
		confirm, err := p2pQuerier.QueryValsetConfirmByEVMAddress(ctx, nonce, evmAddress, signBytes.Hex())
		if err != nil {
			return err
		}
		if confirm == nil {
			logger.Info("couldn't find orchestrator signature", "nonce", nonce, "evm_address", evmAddress)
		} else {
			logger.Info("found orchestrator signature", "nonce", nonce, "evm_address", evmAddress, "signature", confirm.Signature)
		}
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
		confirm, err := p2pQuerier.QueryDataCommitmentConfirmByEVMAddress(ctx, nonce, evmAddress, dataRootHash.Hex())
		if err != nil {
			return err
		}
		if confirm == nil {
			logger.Info("couldn't find orchestrator signature", "nonce", nonce, "evm_address", evmAddress)
		} else {
			logger.Info("found orchestrator signature", "nonce", nonce, "evm_address", evmAddress, "signature", confirm.Signature)
		}
	default:
		return errors.Wrap(types.ErrUnknownAttestationType, strconv.FormatUint(nonce, 10))
	}
	return nil
}
