package relayer

import (
	"context"
	"fmt"
	"math/big"
	"time"

	coregethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	"github.com/celestiaorg/orchestrator-relayer/types"
	wrapper "github.com/celestiaorg/quantum-gravity-bridge/wrappers/QuantumGravityBridge.sol"
	ethcmn "github.com/ethereum/go-ethereum/common"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

type Relayer struct {
	TmQuerier  *rpc.TmQuerier
	AppQuerier *rpc.AppQuerier
	P2PQuerier *p2p.Querier
	EVMClient  *evm.Client
	logger     tmlog.Logger
}

func NewRelayer(
	tmQuerier *rpc.TmQuerier,
	appQuerier *rpc.AppQuerier,
	p2pQuerier *p2p.Querier,
	evmClient *evm.Client,
	logger tmlog.Logger,
) *Relayer {
	return &Relayer{
		TmQuerier:  tmQuerier,
		AppQuerier: appQuerier,
		P2PQuerier: p2pQuerier,
		EVMClient:  evmClient,
		logger:     logger,
	}
}

func (r *Relayer) Start(ctx context.Context) error {
	ethClient, err := r.EVMClient.NewEthClient()
	if err != nil {
		r.logger.Error(err.Error())
		return err
	}
	defer ethClient.Close()
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

		opts, err := r.EVMClient.NewTransactionOpts(ctx)
		if err != nil {
			r.logger.Error(ErrAttestationNotFound.Error())
			time.Sleep(10 * time.Second)
			// will keep retrying indefinitely
			continue
		}

		tx, err := r.ProcessAttestation(ctx, opts, att)
		if err != nil {
			r.logger.Error(ErrAttestationNotFound.Error())
			time.Sleep(10 * time.Second)
			// will keep retrying indefinitely
			continue
		}

		// wait for transaction to be mined
		_, err = r.EVMClient.WaitForTransaction(ctx, ethClient, tx)
		if err != nil {
			r.logger.Error(err.Error())
			time.Sleep(2 * time.Second)
			continue
		}
	}
}

func (r *Relayer) Stop() error {
	err := r.TmQuerier.Stop()
	if err != nil {
		return err
	}
	return nil
}

func (r *Relayer) ProcessAttestation(ctx context.Context, opts *bind.TransactOpts, att celestiatypes.AttestationRequestI) (*coregethtypes.Transaction, error) {
	var tx *coregethtypes.Transaction
	if att.Type() == celestiatypes.ValsetRequestType {
		vs, ok := att.(*celestiatypes.Valset)
		if !ok {
			return nil, ErrAttestationNotValsetRequest
		}
		previousValset, err := r.AppQuerier.QueryLastValsetBeforeNonce(ctx, vs.Nonce)
		if err != nil {
			return nil, err
		}
		signBytes, err := vs.SignBytes()
		if err != nil {
			return nil, err
		}
		confirms, err := r.P2PQuerier.QueryTwoThirdsValsetConfirms(ctx, 30*time.Minute, 10*time.Second, vs.Nonce, *previousValset, signBytes.Hex())
		if err != nil {
			return nil, err
		}

		// FIXME: arguments to be verified
		tx, err = r.UpdateValidatorSet(ctx, opts, *vs, vs.TwoThirdsThreshold(), confirms)
		if err != nil {
			return nil, err
		}
	} else {
		dc, ok := att.(*celestiatypes.DataCommitment)
		if !ok {
			return nil, ErrAttestationNotDataCommitmentRequest
		}
		valset, err := r.AppQuerier.QueryLastValsetBeforeNonce(ctx, dc.Nonce)
		if err != nil {
			return nil, err
		}
		commitment, err := r.TmQuerier.QueryCommitment(ctx, dc.BeginBlock, dc.EndBlock)
		if err != nil {
			return nil, err
		}
		dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(dc.Nonce)), commitment)
		confirms, err := r.P2PQuerier.QueryTwoThirdsDataCommitmentConfirms(ctx, 30*time.Minute, 10*time.Second, *valset, dc.Nonce, dataRootHash.Hex())
		if err != nil {
			return nil, err
		}

		tx, err = r.SubmitDataRootTupleRoot(opts, *dc, *valset, commitment.String(), confirms)
		if err != nil {
			return nil, err
		}
	}
	return tx, nil
}

func (r *Relayer) UpdateValidatorSet(
	ctx context.Context,
	opts *bind.TransactOpts,
	valset celestiatypes.Valset,
	newThreshhold uint64,
	confirms []types.ValsetConfirm,
) (*coregethtypes.Transaction, error) {
	var currentValset celestiatypes.Valset
	if valset.Nonce == 1 {
		currentValset = valset
	} else {
		vs, err := r.AppQuerier.QueryLastValsetBeforeNonce(ctx, valset.Nonce)
		if err != nil {
			return nil, err
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
		return nil, err
	}

	tx, err := r.EVMClient.UpdateValidatorSet(
		opts,
		valset.Nonce,
		newThreshhold,
		currentValset,
		valset,
		sigs,
	)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func (r *Relayer) SubmitDataRootTupleRoot(
	opts *bind.TransactOpts,
	dataCommitment celestiatypes.DataCommitment,
	currentValset celestiatypes.Valset,
	commitment string,
	confirms []types.DataCommitmentConfirm,
) (*coregethtypes.Transaction, error) {
	sigsMap := make(map[string]string)
	// to fetch the signatures easilly by eth address
	for _, c := range confirms {
		sigsMap[c.EthAddress] = c.Signature
	}

	sigs, err := matchAttestationConfirmSigs(sigsMap, currentValset)
	if err != nil {
		return nil, err
	}

	r.logger.Info(fmt.Sprintf(
		"relaying data commitment %d-%d...",
		dataCommitment.BeginBlock,
		dataCommitment.EndBlock,
	))

	tx, err := r.EVMClient.SubmitDataRootTupleRoot(
		opts,
		ethcmn.HexToHash(commitment),
		dataCommitment.Nonce,
		currentValset,
		sigs,
	)
	if err != nil {
		return nil, err
	}
	return tx, nil
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
		v, r, s, err := evm.SigToVRS(sig)
		if err != nil {
			return nil, err
		}

		sigs[i] = wrapper.Signature{
			V: v,
			R: r,
			S: s,
		}
	}

	return sigs, nil
}
