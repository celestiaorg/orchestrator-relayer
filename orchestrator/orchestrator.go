package orchestrator

import (
	"context"
	goerrors "errors"
	"fmt"
	"math/big"
	"strconv"
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

// RequeueWindow the number of nonces that we want to re-enqueue if we can't process them even after retry.
// After this window is elapsed, the nonce is discarded.
const RequeueWindow = 50

// The queue channel's size
const queueSize = 1000

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
	noncesQueue := make(chan uint64, queueSize)
	defer close(noncesQueue)
	// contains the failed nonces to be re-processed.
	failedNoncesQueue := make(chan uint64, queueSize)
	defer close(failedNoncesQueue)

	// used to send a signal when the nonces processor wants to notify the nonces enqueuing services to stop.
	signalChan := make(chan struct{})

	wg := &sync.WaitGroup{}

	// go routine to listen for new attestation nonces
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := orch.StartNewEventsListener(ctx, noncesQueue, signalChan)
		if err != nil {
			orch.Logger.Error("error listening to new attestations", "err", err)
			return
		}
		orch.Logger.Info("stopping listening to new attestations")
	}()

	// go routine for processing nonces
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := orch.ProcessNonces(ctx, noncesQueue, failedNoncesQueue, signalChan)
		if err != nil {
			orch.Logger.Error("error processing attestations", "err", err)
			return
		}
		orch.Logger.Info("stopping processing attestations")
	}()

	// go routine for handling the previous attestation nonces
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := orch.EnqueueMissingEvents(ctx, noncesQueue, signalChan)
		if err != nil {
			orch.Logger.Error("error enqueuing missing attestations", "err", err)
			return
		}
	}()

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
			return nil
		case <-ctx.Done():
			if goerrors.Is(ctx.Err(), context.Canceled) {
				return nil
			}
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
					orch.Logger.Debug("recovered connection")
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
			attestationEvents := mustGetEvent(result, attestationEventName)
			for _, attEvent := range attestationEvents {
				nonce, err := strconv.Atoi(attEvent)
				if err != nil {
					return err
				}
				orch.Logger.Debug("enqueueing new attestation nonce", "nonce", nonce)
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

	earliestAttestationNonce, err := orch.AppQuerier.QueryEarliestAttestationNonce(ctx)
	if err != nil {
		return err
	}

	orch.Logger.Info("syncing missing nonces", "latest_nonce", latestNonce, "first_nonce", earliestAttestationNonce)

	// To accommodate the delay that might happen between starting the two go routines above.
	// Probably, it would be a good idea to further refactor the orchestrator to the relayer style
	// as it is entirely synchronous. Probably, enqueuing separately old nonces and new ones, is not
	// the best design.
	// TODO decide on this later
	for i := uint64(0); i < latestNonce-uint64(earliestAttestationNonce)+1; i++ {
		select {
		case <-signalChan:
			return nil
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
	orch.Logger.Info("finished syncing missing nonces", "latest_nonce", latestNonce, "first_nonce", earliestAttestationNonce)
	return nil
}

func (orch Orchestrator) ProcessNonces(
	ctx context.Context,
	noncesQueue chan uint64,
	requeueQueue chan uint64,
	signalChan chan<- struct{},
) error {
	ticker := time.NewTicker(time.Hour)
	for {
		select {
		case <-ctx.Done():
			close(signalChan)
			return ErrSignalChanNotif
		case <-ticker.C:
			if len(requeueQueue) > 0 && len(noncesQueue) < queueSize {
				// The use of the go routine is to avoid blocking
				go func() {
					nonce := <-requeueQueue
					noncesQueue <- nonce
					orch.Logger.Debug("failed nonce added to the nonces queue to be processed", "nonce", nonce)
				}()
			}
		case nonce := <-noncesQueue:
			orch.Logger.Info("processing nonce", "nonce", nonce)
			if err := orch.Process(ctx, nonce); err != nil {
				orch.Logger.Error("failed to process nonce, retrying", "nonce", nonce, "err", err)
				if err := orch.Retrier.Retry(ctx, func() error {
					return orch.Process(ctx, nonce)
				}); err != nil {
					orch.Logger.Error("error processing nonce even after retrying", "err", err.Error())
					go orch.MaybeRequeue(ctx, requeueQueue, nonce)
				}
			}
		}
	}
}

// MaybeRequeue requeue the nonce to be re-processed subsequently if it's recent.
func (orch Orchestrator) MaybeRequeue(ctx context.Context, requeueQueue chan<- uint64, nonce uint64) {
	latestNonce, err := orch.AppQuerier.QueryLatestAttestationNonce(ctx)
	if err != nil {
		orch.Logger.Debug("error requeuing nonce", "nonce", nonce, "err", err.Error())
		return
	}
	if latestNonce <= RequeueWindow || nonce >= latestNonce-RequeueWindow {
		orch.Logger.Debug("adding failed nonce to requeue queue", "nonce", nonce)
		requeueQueue <- nonce
	} else {
		orch.Logger.Debug("nonce is too old, will not retry it in the future", "nonce", nonce)
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

	if nonce == 1 {
		// there is no need to sign nonce 1 as the Blobstream contract will trust it when deploying.
		return nil
	}
	// check if we need to sign or not
	if nonce != 1 {
		previousValset, err := orch.AppQuerier.QueryLastValsetBeforeNonce(ctx, att.GetNonce())
		if err != nil {
			orch.Logger.Debug("failed to query last valset before nonce (most likely pruned). signing anyway", "err", err.Error())
		} else if !ValidatorPartOfValset(previousValset.Members, orch.EvmAccount.Address.Hex()) {
			// no need to sign if the orchestrator is not part of the validator set that needs to sign the attestation
			orch.Logger.Info("validator not part of valset. won't sign", "nonce", nonce)
			return nil
		}

		if err == nil && previousValset != nil {
			// add the valset to the p2p network
			// it's alright if this fails, we can expect other nodes to do it successfully
			orch.Logger.Debug("providing previous valset to P2P network", "nonce", previousValset.Nonce)
			_ = orch.Broadcaster.ProvideLatestValset(ctx, *types.ToLatestValset(*previousValset))
		}
	}

	switch castedAtt := att.(type) {
	case *celestiatypes.Valset:
		orch.Logger.Debug("creating valset sign bytes", "nonce", castedAtt.Nonce)
		signBytes, err := castedAtt.SignBytes()
		if err != nil {
			return err
		}
		orch.Logger.Debug("checking if a signature has already been provided to the P2P network", "nonce", castedAtt.Nonce)
		resp, err := orch.P2PQuerier.QueryValsetConfirmByEVMAddress(ctx, nonce, orch.EvmAccount.Address.Hex(), signBytes.Hex())
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("valset %d", nonce))
		}
		if resp != nil {
			orch.Logger.Debug("already signed valset", "nonce", nonce, "signature", resp.Signature)
			return nil
		}
		err = orch.ProcessValsetEvent(ctx, *castedAtt)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("valset %d", nonce))
		}
		return nil

	case *celestiatypes.DataCommitment:
		orch.Logger.Debug("querying data commitment from core", "nonce", castedAtt.Nonce, "begin_block", castedAtt.BeginBlock, "end_block", castedAtt.EndBlock)
		commitment, err := orch.TmQuerier.QueryCommitment(
			ctx,
			castedAtt.BeginBlock,
			castedAtt.EndBlock,
		)
		if err != nil {
			return err
		}
		orch.Logger.Debug("creating data commitment sign bytes", "nonce", castedAtt.Nonce)
		dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(castedAtt.Nonce)), commitment)
		orch.Logger.Debug("checking if a signature has already been provided to the P2P network", "nonce", nonce)
		resp, err := orch.P2PQuerier.QueryDataCommitmentConfirmByEVMAddress(
			ctx,
			castedAtt.Nonce,
			orch.EvmAccount.Address.Hex(),
			dataRootHash.Hex(),
		)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("data commitment %d", nonce))
		}
		if resp != nil {
			orch.Logger.Debug("already signed data commitment", "nonce", nonce, "begin_block", castedAtt.BeginBlock, "end_block", castedAtt.EndBlock, "data_root_tuple_root", dataRootHash.Hex(), "signature", resp.Signature)
			return nil
		}
		err = orch.ProcessDataCommitmentEvent(ctx, *castedAtt, dataRootHash)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("data commitment %d", nonce))
		}
		return nil

	default:
		return errors.Wrap(types.ErrUnknownAttestationType, strconv.FormatUint(nonce, 10))
	}
}

func (orch Orchestrator) ProcessValsetEvent(ctx context.Context, valset celestiatypes.Valset) error {
	// add the valset to the p2p network
	// it's alright if this fails, we can expect other nodes to do it successfully
	orch.Logger.Debug("providing the latest valset to P2P network", "nonce", valset.Nonce)
	_ = orch.Broadcaster.ProvideLatestValset(ctx, *types.ToLatestValset(valset))

	signBytes, err := valset.SignBytes()
	if err != nil {
		return err
	}
	orch.Logger.Debug("signing valset", "nonce", valset.Nonce)
	signature, err := evm.NewEthereumSignature(signBytes.Bytes(), orch.EvmKeyStore, *orch.EvmAccount)
	if err != nil {
		return err
	}

	// create and send the valset hash
	msg := types.NewValsetConfirm(
		orch.EvmAccount.Address,
		ethcmn.Bytes2Hex(signature),
	)
	orch.Logger.Debug("providing the valset confirm to P2P network", "nonce", valset.Nonce)
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
	orch.Logger.Debug("signing data commitment", "nonce", dc.Nonce)
	dcSig, err := evm.NewEthereumSignature(dataRootTupleRoot.Bytes(), orch.EvmKeyStore, *orch.EvmAccount)
	if err != nil {
		return err
	}
	msg := types.NewDataCommitmentConfirm(ethcmn.Bytes2Hex(dcSig), orch.EvmAccount.Address)
	orch.Logger.Debug("providing the data commitment confirm to P2P network", "nonce", dc.Nonce)
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
