package orchestrator

import (
	"context"
	"fmt"
	tmlog "github.com/tendermint/tendermint/libs/log"

	"github.com/celestiaorg/orchestrator-relayer/x/qgb/types"
	wrapper "github.com/celestiaorg/quantum-gravity-bridge/wrappers/QuantumGravityBridge.sol"
	ethcmn "github.com/ethereum/go-ethereum/common"
)

type Relayer struct {
	querier   Querier
	evmClient EVMClient
	bridgeID  ethcmn.Hash
	logger    tmlog.Logger
}

func NewRelayer(querier Querier, evmClient EVMClient, logger tmlog.Logger) (*Relayer, error) {
	return &Relayer{
		querier:   querier,
		bridgeID:  types.BridgeID,
		evmClient: evmClient,
		logger:    logger,
	}, nil
}

func (r *Relayer) updateValidatorSet(
	ctx context.Context,
	valset types.Valset,
	newThreshhold uint64,
	confirms []types.MsgValsetConfirm,
) error {
	var currentValset types.Valset
	if valset.Nonce == 1 {
		currentValset = valset
	} else {
		vs, err := r.querier.QueryLastValsetBeforeNonce(ctx, valset.Nonce)
		if err != nil {
			return err
		}
		currentValset = *vs
	}

	sigsMap := make(map[string]string)
	// to fetch the signatures easilly by EVM address
	for _, c := range confirms {
		sigsMap[c.EvmAddress] = c.Signature
	}

	sigs, err := matchAttestationConfirmSigs(sigsMap, currentValset)
	if err != nil {
		return err
	}

	err = r.evmClient.UpdateValidatorSet(
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

func (r *Relayer) submitDataRootTupleRoot(
	ctx context.Context,
	currentValset types.Valset,
	commitment string,
	confirms []types.MsgDataCommitmentConfirm,
) error {
	sigsMap := make(map[string]string)
	// to fetch the signatures easilly by EVM address
	for _, c := range confirms {
		sigsMap[c.EvmAddress] = c.Signature
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

	err = r.evmClient.SubmitDataRootTupleRoot(
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
	currentValset types.Valset,
) ([]wrapper.Signature, error) {
	sigs := make([]wrapper.Signature, len(currentValset.Members))
	// the QGB contract expects the signatures to be ordered by validators in valset
	for i, val := range currentValset.Members {
		sig, has := signatures[val.EvmAddress]
		if !has {
			continue
		}
		v, r, s := SigToVRS(sig)

		sigs[i] = wrapper.Signature{
			V: v,
			R: r,
			S: s,
		}
	}

	return sigs, nil
}
