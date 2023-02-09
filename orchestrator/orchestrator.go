package orchestrator

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"strconv"
	"sync"

	blobtypes "github.com/celestiaorg/celestia-app/x/blob/types"
	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	"github.com/celestiaorg/orchestrator-relayer/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	ethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/pkg/errors"
	tmlog "github.com/tendermint/tendermint/libs/log"
	corerpctypes "github.com/tendermint/tendermint/rpc/core/types"
	coretypes "github.com/tendermint/tendermint/types"
)

var _ I = &Orchestrator{}

type I interface {
	Start(ctx context.Context)
	StartNewEventsListener(ctx context.Context, queue chan<- uint64, signalChan <-chan struct{}) error
	EnqueueMissingEvents(ctx context.Context, queue chan<- uint64, signalChan <-chan struct{}) error
	ProcessNonces(ctx context.Context, noncesQueue <-chan uint64, signalChan chan<- struct{}) error
	Process(ctx context.Context, nonce uint64) error
	ProcessValsetEvent(ctx context.Context, valset celestiatypes.Valset) error
	ProcessDataCommitmentEvent(ctx context.Context, dc celestiatypes.DataCommitment) error
}

type Orchestrator struct {
	Logger tmlog.Logger // maybe use a more general interface

	EvmPrivateKey  ecdsa.PrivateKey
	Signer         *blobtypes.KeyringSigner
	OrchEVMAddress ethcmn.Address
	OrchAccAddress sdk.AccAddress

	AppQuerier  rpc.AppQuerierI
	TmQuerier   rpc.TmQuerierI
	P2PQuerier  *p2p.Querier
	Broadcaster BroadcasterI
	Retrier     RetrierI
}

func New(
	logger tmlog.Logger,
	appQuerier rpc.AppQuerierI,
	tmQuerier rpc.TmQuerierI,
	p2pQuerier *p2p.Querier,
	broadcaster BroadcasterI,
	retrier RetrierI,
	signer *blobtypes.KeyringSigner,
	evmPrivateKey ecdsa.PrivateKey,
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
		AppQuerier:     appQuerier,
		TmQuerier:      tmQuerier,
		P2PQuerier:     p2pQuerier,
		Broadcaster:    broadcaster,
		Retrier:        retrier,
		OrchAccAddress: orchAccAddr,
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
	results, err := orch.TmQuerier.SubscribeEvents(
		ctx,
		"attestation-changes",
		fmt.Sprintf("%s.%s='%s'", celestiatypes.EventTypeAttestationRequest, sdk.AttributeKeyModule, celestiatypes.ModuleName),
	)
	if err != nil {
		return err
	}
	attestationEventName := fmt.Sprintf("%s.%s", celestiatypes.EventTypeAttestationRequest, celestiatypes.AttributeKeyNonce)
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
	latestNonce, err := orch.AppQuerier.QueryLatestAttestationNonce(ctx)
	if err != nil {
		return err
	}

	lastUnbondingHeight, err := orch.AppQuerier.QueryLastUnbondingHeight(ctx)
	if err != nil {
		return err
	}

	orch.Logger.Info("syncing missing nonces", "latest_nonce", latestNonce, "last_unbonding_height", lastUnbondingHeight)

	// To accommodate the delay that might happen between starting the two go routines above.
	// Probably, it would be a good idea to further refactor the orchestrator to the relayer style
	// as it is entirely synchronous. Probably, enqueing separatly old nonces and new ones, is not
	// the best design.
	// TODO decide on this later
	for i := uint64(lastUnbondingHeight); i < latestNonce; i++ {
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
	if !ValidatorPartOfValset(previousValset.Members, orch.OrchEVMAddress.Hex()) {
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
		resp, err := orch.P2PQuerier.QueryValsetConfirmByEVMAddress(ctx, nonce, orch.OrchEVMAddress.Hex())
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
		resp, err := orch.P2PQuerier.QueryDataCommitmentConfirmByEVMAddress(
			ctx,
			dc.Nonce,
			orch.OrchEVMAddress.Hex(),
		)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("data commitment %d", nonce))
		}
		if resp != nil {
			orch.Logger.Debug("already signed data commitment", "nonce", nonce, "begin_block", dc.BeginBlock, "end_block", dc.EndBlock, "commitment", resp.Commitment, "signature", resp.Signature)
			return nil
		}
		err = orch.ProcessDataCommitmentEvent(ctx, *dc)
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
	signature, err := evm.NewEthereumSignature(signBytes.Bytes(), &orch.EvmPrivateKey)
	if err != nil {
		return err
	}

	// create and send the valset hash
	msg := types.NewMsgValsetConfirm(
		orch.OrchEVMAddress,
		ethcmn.Bytes2Hex(signature),
	)
	hash, err := orch.Broadcaster.BroadcastConfirm(ctx, msg)
	if err != nil {
		return err
	}
	orch.Logger.Info("signed Valset", "nonce", valset.Nonce, "tx_hash", hash)
	return nil
}

func (orch Orchestrator) ProcessDataCommitmentEvent(
	ctx context.Context,
	dc celestiatypes.DataCommitment,
) error {
	commitment, err := orch.TmQuerier.QueryCommitment(
		ctx,
		dc.BeginBlock,
		dc.EndBlock,
	)
	if err != nil {
		return err
	}
	dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(dc.Nonce)), commitment)
	dcSig, err := evm.NewEthereumSignature(dataRootHash.Bytes(), &orch.EvmPrivateKey)
	if err != nil {
		return err
	}

	msg := types.NewMsgDataCommitmentConfirm(commitment.String(), ethcmn.Bytes2Hex(dcSig), orch.OrchEVMAddress)
	hash, err := orch.Broadcaster.BroadcastConfirm(ctx, msg)
	if err != nil {
		return err
	}
	orch.Logger.Info("signed commitment", "nonce", dc.Nonce, "begin_block", dc.BeginBlock, "end_block", dc.EndBlock, "commitment", commitment, "tx_hash", hash)
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
