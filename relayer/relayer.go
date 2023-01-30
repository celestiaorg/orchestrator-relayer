package relayer

import (
	"context"
	"fmt"
	"time"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	"github.com/celestiaorg/orchestrator-relayer/types"
	wrapper "github.com/celestiaorg/quantum-gravity-bridge/wrappers/QuantumGravityBridge.sol"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	ethcmn "github.com/ethereum/go-ethereum/common"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

type Relayer struct {
	TmQuerier  rpc.TmQuerierI
	AppQuerier rpc.AppQuerierI
	P2PQuerier p2p.QuerierI
	EVMClient  evm.ClientI
	logger     tmlog.Logger
}

func NewRelayer(
	tmQuerier rpc.TmQuerierI,
	appQuerier rpc.AppQuerierI,
	p2pQuerier p2p.QuerierI,
	evmClient evm.ClientI,
	logger tmlog.Logger,
) (*Relayer, error) {
	return &Relayer{
		TmQuerier:  tmQuerier,
		AppQuerier: appQuerier,
		P2PQuerier: p2pQuerier,
		EVMClient:  evmClient,
		logger:     logger,
	}, nil
}

func (r *Relayer) ProcessEvents(ctx context.Context) error {
	for {
		lastContractNonce, err := r.EVMClient.StateLastEventNonce(&bind.CallOpts{})
		if err != nil {
			r.logger.Error(err.Error())
			continue
		}

		latestNonce, err := r.AppQuerier.QueryLatestAttestationNonce(ctx)
		if err != nil {
			r.logger.Error(err.Error())
			continue
		}

		// If the contract has already the last version, no need to relay anything
		if lastContractNonce >= latestNonce {
			time.Sleep(10 * time.Second) // TODO sleep and at the same time listen for interruptions
			continue
		}

		att, err := r.AppQuerier.QueryAttestationByNonce(ctx, lastContractNonce+1)
		if err != nil {
			r.logger.Error(err.Error())
			continue
		}
		if att == nil {
			r.logger.Error(ErrAttestationNotFound.Error())
			continue
		}
		if att.Type() == celestiatypes.ValsetRequestType {
			vs, ok := att.(*celestiatypes.Valset)
			if !ok {
				return ErrAttestationNotValsetRequest
			}
			confirms, err := r.P2PQuerier.QueryTwoThirdsValsetConfirms(ctx, time.Minute*30, *vs)
			if err != nil {
				return err
			}

			// FIXME: arguments to be verified
			err = r.UpdateValidatorSet(ctx, *vs, vs.TwoThirdsThreshold(), confirms)
			if err != nil {
				return err
			}
		} else {
			dc, ok := att.(*celestiatypes.DataCommitment)
			if !ok {
				return ErrAttestationNotDataCommitmentRequest
			}
			// todo: make times configurable
			confirms, err := r.P2PQuerier.QueryTwoThirdsDataCommitmentConfirms(ctx, time.Minute*30, *dc)
			if err != nil {
				return err
			}

			valset, err := r.AppQuerier.QueryLastValsetBeforeNonce(ctx, dc.Nonce)
			if err != nil {
				return err
			}

			err = r.SubmitDataRootTupleRoot(ctx, *valset, confirms[0].Commitment, confirms)
			if err != nil {
				return err
			}
		}
	}
}

func (r *Relayer) UpdateValidatorSet(
	ctx context.Context,
	valset celestiatypes.Valset,
	newThreshhold uint64,
	confirms []types.ValsetConfirm,
) error {
	var currentValset celestiatypes.Valset
	if valset.Nonce == 1 {
		currentValset = valset
	} else {
		vs, err := r.AppQuerier.QueryLastValsetBeforeNonce(ctx, valset.Nonce)
		if err != nil {
			return err
		}
		currentValset = *vs
	}

	sigsMap := make(map[string]string)
	// to fetch the signatures easilly by eth address
	for _, c := range confirms {
		sigsMap[c.EthAddress] = c.Signature
	}

	sigs, err := matchAttestationConfirmSigs(sigsMap, currentValset)
	if err != nil {
		return err
	}

	err = r.EVMClient.UpdateValidatorSet(
		ctx,
		valset.Nonce,
		newThreshhold,
		currentValset,
		valset,
		sigs,
		true,
	)
	if err != nil {
		return err
	}
	return nil
}

func (r *Relayer) SubmitDataRootTupleRoot(
	ctx context.Context,
	currentValset celestiatypes.Valset,
	commitment string,
	confirms []types.DataCommitmentConfirm,
) error {
	sigsMap := make(map[string]string)
	// to fetch the signatures easilly by eth address
	for _, c := range confirms {
		sigsMap[c.EthAddress] = c.Signature
	}

	sigs, err := matchAttestationConfirmSigs(sigsMap, currentValset)
	if err != nil {
		return err
	}

	// the confirm carries the correct nonce to be submitted
	newDataCommitmentNonce := confirms[0].Nonce

	r.logger.Info(fmt.Sprintf(
		"relaying data commitment %d-%d...",
		confirms[0].BeginBlock,
		confirms[0].EndBlock,
	))

	err = r.EVMClient.SubmitDataRootTupleRoot(
		ctx,
		ethcmn.HexToHash(commitment),
		newDataCommitmentNonce,
		currentValset,
		sigs,
		true,
	)
	if err != nil {
		return err
	}
	return nil
}

// matchAttestationConfirmSigs matches and sorts the confirm signatures with the valset
// members as expected by the QGB contract.
// Also, it leaves the non provided signatures as nil in the `sigs` slice:
// https://github.com/celestiaorg/celestia-app/issues/628
func matchAttestationConfirmSigs(
	signatures map[string]string,
	currentValset celestiatypes.Valset,
) ([]wrapper.Signature, error) {
	sigs := make([]wrapper.Signature, len(currentValset.Members))
	// the QGB contract expects the signatures to be ordered by validators in valset
	for i, val := range currentValset.Members {
		sig, has := signatures[val.EvmAddress]
		if !has {
			continue
		}
		v, r, s := evm.SigToVRS(sig)

		sigs[i] = wrapper.Signature{
			V: v,
			R: r,
			S: s,
		}
	}

	return sigs, nil
}
