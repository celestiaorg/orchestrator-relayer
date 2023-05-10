package orchestrator

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/ethereum/go-ethereum/accounts"
	"github.com/ethereum/go-ethereum/accounts/keystore"

	"github.com/celestiaorg/orchestrator-relayer/helpers"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	"github.com/celestiaorg/orchestrator-relayer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
	tmlog "github.com/tendermint/tendermint/libs/log"
	corerpctypes "github.com/tendermint/tendermint/rpc/core/types"
	coretypes "github.com/tendermint/tendermint/types"
)

type Orchestrator struct {
	Logger tmlog.Logger // maybe use a more general interface

	EvmKeyStore *keystore.KeyStore
	EvmAccount  *accounts.Account

	AppQuerier  *rpc.AppQuerier
	TmQuerier   *rpc.TmQuerier
	P2PQuerier  *p2p.Querier
	Broadcaster *Broadcaster
	Retrier     *helpers.Retrier
}

func New(
	logger tmlog.Logger,
	appQuerier *rpc.AppQuerier,
	tmQuerier *rpc.TmQuerier,
	p2pQuerier *p2p.Querier,
	broadcaster *Broadcaster,
	retrier *helpers.Retrier,
	evmKeyStore *keystore.KeyStore,
	evmAccount *accounts.Account,
) *Orchestrator {
	return &Orchestrator{
		Logger:      logger,
		EvmKeyStore: evmKeyStore,
		EvmAccount:  evmAccount,
		AppQuerier:  appQuerier,
		TmQuerier:   tmQuerier,
		P2PQuerier:  p2pQuerier,
		Broadcaster: broadcaster,
		Retrier:     retrier,
	}
}

func (orch Orchestrator) Start(ctx context.Context) {
	// contains the nonces that will be signed by the orchestrator.
	noncesQueue := make(chan uint64, 100)
	defer close(noncesQueue)

	// used to send a signal when the nonces processor wants to notify the nonces enqueuing services to stop.
	signalChan := make(chan struct{})

	withCancel, cancel := context.WithCancel(ctx)

	wg := &sync.WaitGroup{}

	// go routine to listen for new attestation nonces
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := orch.StartNewEventsListener(withCancel, noncesQueue, signalChan)
		if err != nil {
			orch.Logger.Error("error listening to new attestations", "err", err)
			cancel()
		}
		orch.Logger.Error("stopping listening to new attestations")
	}()

	// go routine for processing nonces
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := orch.ProcessNonces(withCancel, noncesQueue, signalChan)
		if err != nil {
			orch.Logger.Error("error processing attestations", "err", err)
			cancel()
		}
		orch.Logger.Error("stopping processing attestations")
	}()

	// go routine for handling the previous attestation nonces
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := orch.EnqueueMissingEvents(withCancel, noncesQueue, signalChan)
		if err != nil {
			orch.Logger.Error("error enqueuing missing attestations", "err", err)
			cancel()
		}
		orch.Logger.Error("stopping enqueuing missing attestations")
	}()

	// FIXME should we add  another go routine that keep checking if all the attestations
	// were signed every 10min for example?

	wg.Wait()
}

func (orch Orchestrator) StartNewEventsListener(
	ctx context.Context,
	queue chan<- uint64,
	signalChan <-chan struct{},
) error {
	subscriptionName := "attestation-changes"
	query := fmt.Sprintf("%s.%s='%s'", celestiatypes.EventTypeAttestationRequest, sdk.AttributeKeyModule, celestiatypes.ModuleName)
	results, err := orch.TmQuerier.SubscribeEvents(ctx, subscriptionName, query)
	if err != nil {
		return err
	}
	defer func() {
		err := orch.TmQuerier.UnsubscribeEvents(ctx, subscriptionName, query)
		if err != nil {
			orch.Logger.Error(err.Error())
		}
	}()
	attestationEventName := fmt.Sprintf("%s.%s", celestiatypes.EventTypeAttestationRequest, celestiatypes.AttributeKeyNonce)
	orch.Logger.Info("listening for new block events...")
	// ticker for keeping an eye on the health of the tendermint RPC
	// this is because the ws connection doesn't complain when the node is down
	// which leaves the orchestrator in a hanging state
	ticker := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-signalChan:
			return ErrSignalChanNotif
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			running := orch.TmQuerier.IsRunning(ctx)
			// if the connection is lost, retry connecting a few times
			if !running {
				orch.Logger.Error("tendermint RPC down. Retrying to connect")
				err := orch.Retrier.Retry(ctx, func() error {
					err := orch.TmQuerier.Reconnect()
					if err != nil {
						return err
					}
					results, err = orch.TmQuerier.SubscribeEvents(ctx, subscriptionName, query)
					if err != nil {
						return err
					}
					return nil
				})
				if err != nil {
					orch.Logger.Error(err.Error())
					return err
				}
			}
		case result := <-results:
			blockEvent := mustGetEvent(result, coretypes.EventTypeKey)
			isBlock := blockEvent[0] == coretypes.EventNewBlock
			if !isBlock {
				// we only want to handle the attestation when the block is committed
				continue
			}
			attestationEvent := mustGetEvent(result, attestationEventName)
			nonce, err := strconv.Atoi(attestationEvent[0])
			if err != nil {
				return err
			}
			orch.Logger.Info("enqueueing new attestation nonce", "nonce", nonce)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-signalChan:
				return ErrSignalChanNotif
			case queue <- uint64(nonce):
			}
		}
	}
}

func (orch Orchestrator) EnqueueMissingEvents(
	ctx context.Context,
	queue chan<- uint64,
	signalChan <-chan struct{},
) error {
	err := orch.TmQuerier.WaitForHeight(ctx, 1)
	if err != nil {
		return err
	}

	latestNonce, err := orch.AppQuerier.QueryLatestAttestationNonce(ctx)
	if err != nil {
		return err
	}

	lastUnbondingHeight, err := orch.AppQuerier.QueryLastUnbondingHeight(ctx)
	if err != nil {
		return err
	}
	var startingNonce uint64
	if lastUnbondingHeight == 0 {
		startingNonce = 1
	} else {
		dc, err := orch.AppQuerier.QueryDataCommitmentForHeight(ctx, uint64(lastUnbondingHeight))
		if err != nil {
			if strings.Contains(err.Error(), "no data commitment has been generated for the provided height") {
				orch.Logger.Info("finished syncing missing nonces", "latest_nonce", latestNonce, "first_nonce", latestNonce)
				return nil
			}
			return err
		}
		startingValset, err := orch.AppQuerier.QueryLastValsetBeforeNonce(ctx, dc.Nonce)
		if err != nil {
			return err
		}
		startingNonce = startingValset.Nonce
	}

	orch.Logger.Info("syncing missing nonces", "latest_nonce", latestNonce, "first_nonce", startingNonce)

	// To accommodate the delay that might happen between starting the two go routines above.
	// Probably, it would be a good idea to further refactor the orchestrator to the relayer style
	// as it is entirely synchronous. Probably, enqueuing separately old nonces and new ones, is not
	// the best design.
	// TODO decide on this later
	for i := uint64(0); i < latestNonce-startingNonce+1; i++ {
		select {
		case <-signalChan:
			return ErrSignalChanNotif
		case <-ctx.Done():
			return ctx.Err()
		default:
			orch.Logger.Debug("enqueueing missing attestation nonce", "nonce", latestNonce-i)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-signalChan:
				return ErrSignalChanNotif
			case queue <- latestNonce - i:
			}
		}
	}
	orch.Logger.Info("finished syncing missing nonces", "latest_nonce", latestNonce, "first_nonce", startingNonce)
	return nil
}

func (orch Orchestrator) ProcessNonces(
	ctx context.Context,
	noncesQueue <-chan uint64,
	signalChan chan<- struct{},
) error {
	for {
		select {
		case <-ctx.Done():
			close(signalChan)
			return ErrSignalChanNotif
		case nonce := <-noncesQueue:
			orch.Logger.Info("processing nonce", "nonce", nonce)
			if err := orch.Process(ctx, nonce); err != nil {
				orch.Logger.Error("failed to process nonce, retrying", "nonce", nonce, "err", err)
				if err := orch.Retrier.Retry(ctx, func() error {
					return orch.Process(ctx, nonce)
				}); err != nil {
					close(signalChan)
					return err
				}
			}
		}
	}
}

func (orch Orchestrator) Process(ctx context.Context, nonce uint64) error {
	att, err := orch.AppQuerier.QueryAttestationByNonce(ctx, nonce)
	if err != nil {
		return err
	}
	if att == nil {
		return celestiatypes.ErrAttestationNotFound
	}
	// check if the validator is part of the needed valset
	var previousValset *celestiatypes.Valset
	if att.GetNonce() == 1 {
		// if nonce == 1, then, the current valset should sign the confirm.
		// In fact, the first nonce should never be signed. Because, the first attestation, in the case
		// where the `earliest` flag is specified when deploying the contract, will be relayed as part of
		// the deployment of the QGB contract.
		// It will be signed temporarily for now.
		previousValset, err = orch.AppQuerier.QueryValsetByNonce(ctx, att.GetNonce())
		if err != nil {
			return err
		}
	} else {
		previousValset, err = orch.AppQuerier.QueryLastValsetBeforeNonce(ctx, att.GetNonce())
		if err != nil {
			return err
		}
	}
	if !ValidatorPartOfValset(previousValset.Members, orch.EvmAccount.Address.Hex()) {
		// no need to sign if the orchestrator is not part of the validator set that needs to sign the attestation
		orch.Logger.Debug("validator not part of valset. won't sign", "nonce", nonce)
		return nil
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
		resp, err := orch.P2PQuerier.QueryValsetConfirmByEVMAddress(ctx, nonce, orch.EvmAccount.Address.Hex(), signBytes.Hex())
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("valset %d", nonce))
		}
		if resp != nil {
			orch.Logger.Debug("already signed valset", "nonce", nonce, "signature", resp.Signature)
			return nil
		}
		err = orch.ProcessValsetEvent(ctx, *vs)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("valset %d", nonce))
		}
		return nil

	case celestiatypes.DataCommitmentRequestType:
		dc, ok := att.(*celestiatypes.DataCommitment)
		if !ok {
			return errors.Wrap(types.ErrAttestationNotDataCommitmentRequest, strconv.FormatUint(nonce, 10))
		}
		if dc.BeginBlock == 0 {
			dc.BeginBlock = 1
		}
		commitment, err := orch.TmQuerier.QueryCommitment(
			ctx,
			dc.BeginBlock,
			dc.EndBlock,
		)
		if err != nil {
			return err
		}
		dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(dc.Nonce)), commitment)
		resp, err := orch.P2PQuerier.QueryDataCommitmentConfirmByEVMAddress(
			ctx,
			dc.Nonce,
			orch.EvmAccount.Address.Hex(),
			dataRootHash.Hex(),
		)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("data commitment %d", nonce))
		}
		if resp != nil {
			orch.Logger.Debug("already signed data commitment", "nonce", nonce, "begin_block", dc.BeginBlock, "end_block", dc.EndBlock, "data_root_tuple_root", dataRootHash.Hex(), "signature", resp.Signature)
			return nil
		}
		err = orch.ProcessDataCommitmentEvent(ctx, *dc, dataRootHash)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("data commitment %d", nonce))
		}
		return nil

	default:
		return errors.Wrap(types.ErrUnknownAttestationType, strconv.FormatUint(nonce, 10))
	}
}

func (orch Orchestrator) ProcessValsetEvent(ctx context.Context, valset celestiatypes.Valset) error {
	signBytes, err := valset.SignBytes()
	if err != nil {
		return err
	}
	signature, err := evm.NewEthereumSignature(signBytes.Bytes(), orch.EvmKeyStore, *orch.EvmAccount)
	if err != nil {
		return err
	}

	// create and send the valset hash
	msg := types.NewValsetConfirm(
		orch.EvmAccount.Address,
		ethcmn.Bytes2Hex(signature),
	)
	err = orch.Broadcaster.ProvideValsetConfirm(ctx, valset.Nonce, *msg, signBytes.Hex())
	if err != nil {
		return err
	}
	orch.Logger.Info("signed Valset", "nonce", valset.Nonce)
	return nil
}

func (orch Orchestrator) ProcessDataCommitmentEvent(
	ctx context.Context,
	dc celestiatypes.DataCommitment,
	dataRootTupleRoot ethcmn.Hash,
) error {
	dcSig, err := evm.NewEthereumSignature(dataRootTupleRoot.Bytes(), orch.EvmKeyStore, *orch.EvmAccount)
	if err != nil {
		return err
	}
	msg := types.NewDataCommitmentConfirm(ethcmn.Bytes2Hex(dcSig), orch.EvmAccount.Address)
	err = orch.Broadcaster.ProvideDataCommitmentConfirm(ctx, dc.Nonce, *msg, dataRootTupleRoot.Hex())
	if err != nil {
		return err
	}
	orch.Logger.Info("signed commitment", "nonce", dc.Nonce, "begin_block", dc.BeginBlock, "end_block", dc.EndBlock, "data_root_tuple_root", dataRootTupleRoot.Hex())
	return nil
}

// mustGetEvent takes a corerpctypes.ResultEvent and checks whether it has
// the provided eventName. If not, it panics.
func mustGetEvent(result corerpctypes.ResultEvent, eventName string) []string {
	ev := result.Events[eventName]
	if len(ev) == 0 {
		panic(errors.Wrap(
			celestiatypes.ErrEmpty,
			fmt.Sprintf(
				"%s not found in event %s",
				coretypes.EventTypeKey,
				result.Events,
			),
		))
	}
	return ev
}

func ValidatorPartOfValset(members []celestiatypes.BridgeValidator, evmAddr string) bool {
	for _, val := range members {
		if val.EvmAddress == evmAddr {
			return true
		}
	}
	return false
}
