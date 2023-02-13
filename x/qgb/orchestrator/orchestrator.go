package orchestrator

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/x/qgb/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	tmlog "github.com/tendermint/tendermint/libs/log"
	corerpctypes "github.com/tendermint/tendermint/rpc/core/types"
	coretypes "github.com/tendermint/tendermint/types"
)

const (
	MockedSignerPower = 100
	OrchAddress       = "celestia1hu6qt83qczjvq2wd2t0qg82jlrstv3s0jcmycz"
	EVMAddress        = "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
)

var _ I = &Orchestrator{}

type I interface {
	Start(ctx context.Context)
	StartNewEventsListener(ctx context.Context, queue chan<- uint64, signalChan <-chan struct{}) error
	EnqueueMissingEvents(ctx context.Context, queue chan<- uint64, signalChan <-chan struct{}) error
	ProcessNonces(ctx context.Context, noncesQueue <-chan uint64, signalChan chan<- struct{}) error
	Process(ctx context.Context, nonce uint64) error
	ProcessValsetEvent(ctx context.Context, valset types.Valset) error
	ProcessDataCommitmentEvent(ctx context.Context, dc types.DataCommitment, valset types.Valset) error
}

type Orchestrator struct {
	Logger tmlog.Logger // maybe use a more general interface

	EvmPrivateKey  ecdsa.PrivateKey
	OrchEVMAddress ethcmn.Address
	OrchAccAddress sdk.AccAddress

	Querier Querier
	Retrier RetrierI

	Relayer *Relayer
}

func NewOrchestrator(
	logger tmlog.Logger,
	querier Querier,
	retrier RetrierI,
	evmPrivateKey ecdsa.PrivateKey,
	relayer *Relayer,
) (*Orchestrator, error) {
	orchEVMAddr := crypto.PubkeyToAddress(evmPrivateKey.PublicKey)

	orchAccAddr, err := sdk.AccAddressFromBech32(OrchAddress)
	if err != nil {
		panic(fmt.Errorf("orchestrator address generation should not fail: %s", err))
	}

	return &Orchestrator{
		Logger:         logger,
		EvmPrivateKey:  evmPrivateKey,
		OrchEVMAddress: orchEVMAddr,
		Querier:        querier,
		Retrier:        retrier,
		OrchAccAddress: orchAccAddr,
		Relayer:        relayer,
	}, nil
}

func (orch Orchestrator) Start(ctx context.Context) {
	// contains the nonces that will be signed by the orchestrator.
	noncesQueue := make(chan uint64, 100)
	defer close(noncesQueue)

	// used to send a signal when the nonces processor wants to notify the nonces enqueuing services to stop.
	signalChan := make(chan struct{})

	withCancel, cancel := context.WithCancel(ctx)

	wg := &sync.WaitGroup{}

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

	wg.Add(1)
	go func() {
		defer wg.Done()
		err := orch.EnqueueMissingEvents(withCancel, noncesQueue, signalChan)
		if err != nil {
			orch.Logger.Error("error enqueing missing attestations", "err", err)
			cancel()
		}
		orch.Logger.Error("stopping enqueing missing attestations")
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
	results, err := orch.Querier.SubscribeEvents(
		ctx,
		"attestation-changes",
		fmt.Sprintf("%s.%s='%s'", types.EventTypeAttestationRequest, sdk.AttributeKeyModule, types.ModuleName),
	)
	if err != nil {
		return err
	}
	attestationEventName := fmt.Sprintf("%s.%s", types.EventTypeAttestationRequest, types.AttributeKeyNonce)
	orch.Logger.Info("listening for new block events...")
	for {
		select {
		case <-signalChan:
			return nil
		case <-ctx.Done():
			return nil
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
			orch.Logger.Debug("enqueueing new attestation nonce", "nonce", nonce)
			select {
			case <-signalChan:
				return nil
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
	latestNonce, err := orch.Querier.QueryLatestAttestationNonce(ctx)
	if err != nil {
		return err
	}

	lastUnbondingHeight, err := orch.Querier.QueryLastUnbondingHeight(ctx)
	if err != nil {
		return err
	}

	orch.Logger.Info("syncing missing nonces", "latest_nonce", latestNonce, "last_unbonding_height", lastUnbondingHeight)

	// To accommodate the delay that might happen between starting the two go routines above.
	// Probably, it would be a good idea to further refactor the orchestrator to the relayer style
	// as it is entirely synchronous. Probably, enqueing separatly old nonces and new ones, is not
	// the best design.
	// TODO decide on this later
	for i := lastUnbondingHeight; i < latestNonce; i++ {
		select {
		case <-signalChan:
			return nil
		case <-ctx.Done():
			return nil
		default:
			orch.Logger.Debug("enqueueing missing attestation nonce", "nonce", latestNonce-i)
			select {
			case <-signalChan:
				return nil
			case queue <- latestNonce - i:
			}
		}
	}
	orch.Logger.Info("finished syncing missing nonces", "latest_nonce", latestNonce, "last_unbonding_height", lastUnbondingHeight)
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
			return nil
		case nonce := <-noncesQueue:
			orch.Logger.Debug("processing nonce", "nonce", nonce)
			if err := orch.Process(ctx, nonce); err != nil {
				orch.Logger.Error("failed to process nonce, retrying", "nonce", nonce, "err", err)
				if err := orch.Retrier.Retry(ctx, nonce, orch.Process); err != nil {
					close(signalChan)
					return err
				}
			}
		}
	}
}

func (orch Orchestrator) Process(ctx context.Context, nonce uint64) error {
	att, err := orch.Querier.QueryAttestationByNonce(ctx, nonce)
	if err != nil {
		return err
	}
	if att == nil {
		return types.ErrAttestationNotFound
	}
	// check if the validator is part of the needed valset
	var previousValset *types.Valset
	if att.GetNonce() == 1 {
		// if nonce == 1, then, the current valset should sign the confirm.
		// In fact, the first nonce should never be signed. Because, the first attestation, in the case
		// where the `earliest` flag is specified when deploying the contract, will be relayed as part of
		// the deployment of the QGB contract.
		// It will be signed temporarily for now.
		previousValset, err = orch.Querier.QueryValsetByNonce(ctx, att.GetNonce())
		if err != nil {
			return err
		}
	} else {
		previousValset, err = orch.Querier.QueryLastValsetBeforeNonce(ctx, att.GetNonce())
		if err != nil {
			return err
		}
	}
	previousValset.Members = []types.BridgeValidator{
		{
			Power:      MockedSignerPower,
			EvmAddress: orch.OrchEVMAddress.String(),
		},
	}
	switch att.Type() {
	case types.ValsetRequestType:
		vs, ok := att.(*types.Valset)
		if !ok {
			return errors.Wrap(types.ErrAttestationNotValsetRequest, strconv.FormatUint(nonce, 10))
		}
		vs.Members = previousValset.Members
		err = orch.ProcessValsetEvent(ctx, *vs)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("valset %d", nonce))
		}
		return nil

	case types.DataCommitmentRequestType:
		dc, ok := att.(*types.DataCommitment)
		if !ok {
			return errors.Wrap(types.ErrAttestationNotDataCommitmentRequest, strconv.FormatUint(nonce, 10))
		}
		resp, err := orch.Querier.QueryDataCommitmentConfirm(
			ctx,
			dc.EndBlock,
			dc.BeginBlock,
			orch.OrchAccAddress.String(),
		)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("data commitment %d", nonce))
		}
		if resp != nil {
			orch.Logger.Debug("already signed data commitment", "nonce", nonce, "begin_block", resp.BeginBlock, "end_block", resp.EndBlock, "commitment", resp.Commitment, "signature", resp.Signature)
			return nil
		}
		err = orch.ProcessDataCommitmentEvent(ctx, *dc, *previousValset)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("data commitment %d", nonce))
		}
		return nil

	default:
		return errors.Wrap(ErrUnknownAttestationType, strconv.FormatUint(nonce, 10))
	}
}

func (orch Orchestrator) ProcessValsetEvent(ctx context.Context, valset types.Valset) error {
	signBytes, err := valset.SignBytes(types.BridgeID)
	if err != nil {
		return err
	}
	signature, err := types.NewEthereumSignature(signBytes.Bytes(), &orch.EvmPrivateKey)
	if err != nil {
		return err
	}

	// create and send the valset hash
	msg := types.NewMsgValsetConfirm(
		valset.Nonce,
		orch.OrchEVMAddress,
		orch.OrchAccAddress,
		ethcmn.Bytes2Hex(signature),
	)
	return orch.Relayer.updateValidatorSet(ctx, valset, valset.TwoThirdsThreshold(), []types.MsgValsetConfirm{*msg})
}

func (orch Orchestrator) ProcessDataCommitmentEvent(
	ctx context.Context,
	dc types.DataCommitment,
	valset types.Valset,
) error {
	commitment, err := orch.Querier.QueryCommitment(
		ctx,
		dc.BeginBlock,
		dc.EndBlock,
	)
	if err != nil {
		return err
	}
	dataRootHash := types.DataCommitmentTupleRootSignBytes(types.BridgeID, big.NewInt(int64(dc.Nonce)), commitment)
	dcSig, err := types.NewEthereumSignature(dataRootHash.Bytes(), &orch.EvmPrivateKey)
	if err != nil {
		return err
	}

	msg := types.NewMsgDataCommitmentConfirm(
		commitment.String(),
		ethcmn.Bytes2Hex(dcSig),
		orch.OrchAccAddress,
		orch.OrchEVMAddress,
		dc.BeginBlock,
		dc.EndBlock,
		dc.Nonce,
	)

	return orch.Relayer.submitDataRootTupleRoot(ctx, valset, commitment.String(), []types.MsgDataCommitmentConfirm{*msg})
}

const (
	DEFAULTCELESTIAGASLIMIT = 100000
	DEFAULTCELESTIATXFEE    = 100
)

var _ RetrierI = &Retrier{}

type Retrier struct {
	logger        tmlog.Logger
	retriesNumber int
}

func NewRetrier(logger tmlog.Logger, retriesNumber int) *Retrier {
	return &Retrier{
		logger:        logger,
		retriesNumber: retriesNumber,
	}
}

type RetrierI interface {
	Retry(ctx context.Context, nonce uint64, retryMethod func(context.Context, uint64) error) error
	RetryThenFail(ctx context.Context, nonce uint64, retryMethod func(context.Context, uint64) error)
}

func (r Retrier) Retry(ctx context.Context, nonce uint64, retryMethod func(context.Context, uint64) error) error {
	var err error
	for i := 0; i <= r.retriesNumber; i++ {
		// We can implement some exponential backoff in here
		select {
		case <-ctx.Done():
			return nil
		default:
			time.Sleep(10 * time.Second)
			r.logger.Info("retrying", "nonce", nonce, "retry_number", i, "retries_left", r.retriesNumber-i)
			err = retryMethod(ctx, nonce)
			if err == nil {
				r.logger.Info("nonce processing succeeded", "nonce", nonce, "retries_number", i)
				return nil
			}
			r.logger.Error("failed to process nonce", "nonce", nonce, "retry", i, "err", err)
		}
	}
	return err
}

func (r Retrier) RetryThenFail(ctx context.Context, nonce uint64, retryMethod func(context.Context, uint64) error) {
	err := r.Retry(ctx, nonce, retryMethod)
	if err != nil {
		panic(err)
	}
}

// mustGetEvent takes a corerpctypes.ResultEvent and checks whether it has
// the provided eventName. If not, it panics.
func mustGetEvent(result corerpctypes.ResultEvent, eventName string) []string {
	ev := result.Events[eventName]
	if len(ev) == 0 {
		panic(errors.Wrap(
			types.ErrEmpty,
			fmt.Sprintf(
				"%s not found in event %s",
				coretypes.EventTypeKey,
				result.Events,
			),
		))
	}
	return ev
}
