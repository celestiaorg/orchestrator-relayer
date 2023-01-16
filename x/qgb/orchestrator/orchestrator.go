package orchestrator

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strconv"
	"sync"
	"time"

	"github.com/celestiaorg/celestia-app/app"

	blobtypes "github.com/celestiaorg/celestia-app/x/blob/types"
	"github.com/celestiaorg/celestia-app/x/qgb/keeper"
	"github.com/celestiaorg/celestia-app/x/qgb/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdktypestx "github.com/cosmos/cosmos-sdk/types/tx"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	tmlog "github.com/tendermint/tendermint/libs/log"
	corerpctypes "github.com/tendermint/tendermint/rpc/core/types"
	coretypes "github.com/tendermint/tendermint/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	MockedSignerPower = 100
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
	Signer         *blobtypes.KeyringSigner
	OrchEVMAddress ethcmn.Address
	OrchAccAddress sdk.AccAddress

	Querier     Querier
	Broadcaster BroadcasterI
	Retrier     RetrierI

	Relayer *Relayer
}

func NewOrchestrator(
	logger tmlog.Logger,
	querier Querier,
	broadcaster BroadcasterI,
	retrier RetrierI,
	signer *blobtypes.KeyringSigner,
	evmPrivateKey ecdsa.PrivateKey,
	relayer *Relayer,
) (*Orchestrator, error) {
	orchEVMAddr := crypto.PubkeyToAddress(evmPrivateKey.PublicKey)

	orchAccAddr, err := signer.GetSignerInfo().GetAddress()
	if err != nil {
		return nil, err
	}

	return &Orchestrator{
		Logger:         logger,
		Signer:         signer,
		EvmPrivateKey:  evmPrivateKey,
		OrchEVMAddress: orchEVMAddr,
		Querier:        querier,
		Broadcaster:    broadcaster,
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
	if !keeper.ValidatorPartOfValset(previousValset.Members, orch.OrchEVMAddress.Hex()) {
		// no need to sign if the orchestrator is not part of the validator set that needs to sign the attestation
		orch.Logger.Debug("validator not part of valset. won't sign", "nonce", nonce)
		return nil
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

var _ BroadcasterI = &Broadcaster{}

type BroadcasterI interface {
	BroadcastTx(ctx context.Context, msg sdk.Msg) (string, error)
}

type Broadcaster struct {
	mutex            *sync.Mutex
	signer           *blobtypes.KeyringSigner
	qgbGrpc          *grpc.ClientConn
	celestiaGasLimit uint64
	fee              int64
}

func NewBroadcaster(
	qgbGrpcAddr string,
	signer *blobtypes.KeyringSigner,
	celestiaGasLimit uint64,
	fee int64,
) (*Broadcaster, error) {
	qgbGrpc, err := grpc.Dial(qgbGrpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &Broadcaster{
		mutex:            &sync.Mutex{}, // investigate if this is needed
		signer:           signer,
		qgbGrpc:          qgbGrpc,
		celestiaGasLimit: celestiaGasLimit,
		fee:              fee,
	}, nil
}

func (bc *Broadcaster) BroadcastTx(ctx context.Context, msg sdk.Msg) (string, error) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()
	err := bc.signer.QueryAccountNumber(ctx, bc.qgbGrpc)
	if err != nil {
		return "", err
	}

	builder := bc.signer.NewTxBuilder()
	builder.SetGasLimit(bc.celestiaGasLimit)
	builder.SetFeeAmount(sdk.NewCoins(sdk.NewCoin(app.BondDenom, sdk.NewInt(bc.fee))))

	// TODO: update this api
	// via https://github.com/celestiaorg/celestia-app/pull/187/commits/37f96d9af30011736a3e6048bbb35bad6f5b795c
	tx, err := bc.signer.BuildSignedTx(builder, msg)
	if err != nil {
		return "", err
	}

	rawTx, err := bc.signer.EncodeTx(tx)
	if err != nil {
		return "", err
	}

	// FIXME sdktypestx.BroadcastMode_BROADCAST_MODE_BLOCK waits for a block to be minted containing
	// the transaction to continue. This makes the orchestrator slow to catchup.
	// It would be better to just send the transaction. Then, another job would keep an eye
	// if the transaction was included. If not, retry it. But this would mean we should increment ourselves
	// the sequence number after each broadcasted transaction.
	// We can also use BroadcastMode_BROADCAST_MODE_SYNC but it will also fail due to a non incremented
	// sequence number.

	resp, err := blobtypes.BroadcastTx(ctx, bc.qgbGrpc, sdktypestx.BroadcastMode_BROADCAST_MODE_BLOCK, rawTx)
	if err != nil {
		return "", err
	}

	if resp.TxResponse.Code != 0 {
		return "", errors.Wrap(ErrFailedBroadcast, resp.TxResponse.RawLog)
	}

	return resp.TxResponse.TxHash, nil
}

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
