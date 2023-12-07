package relayer

import (
	"bytes"
	"context"
	"encoding/hex"
	stderrors "errors"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/ethclient"

	"github.com/ipfs/go-datastore"
	badger "github.com/ipfs/go-ds-badger2"

	"github.com/pkg/errors"

	"github.com/celestiaorg/orchestrator-relayer/helpers"

	coregethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"

	wrapper "github.com/celestiaorg/blobstream-contracts/v4/wrappers/Blobstream.sol"
	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/evm"
	"github.com/celestiaorg/orchestrator-relayer/p2p"
	"github.com/celestiaorg/orchestrator-relayer/rpc"
	"github.com/celestiaorg/orchestrator-relayer/types"
	ethcmn "github.com/ethereum/go-ethereum/common"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

type Relayer struct {
	TmQuerier             *rpc.TmQuerier
	AppQuerier            *rpc.AppQuerier
	P2PQuerier            *p2p.Querier
	EVMClient             *evm.Client
	logger                tmlog.Logger
	Retrier               *helpers.Retrier
	SignatureStore        *badger.Datastore
	RetryTimeout          time.Duration
	IsBackupRelayer       bool
	BackupRelayerWaitTime time.Duration
}

func NewRelayer(
	tmQuerier *rpc.TmQuerier,
	appQuerier *rpc.AppQuerier,
	p2pQuerier *p2p.Querier,
	evmClient *evm.Client,
	logger tmlog.Logger,
	retrier *helpers.Retrier,
	sigStore *badger.Datastore,
	retryTimeout time.Duration,
	isBackupRelayer bool,
	backupRelayerWaitTime time.Duration,
) *Relayer {
	return &Relayer{
		TmQuerier:             tmQuerier,
		AppQuerier:            appQuerier,
		P2PQuerier:            p2pQuerier,
		EVMClient:             evmClient,
		logger:                logger,
		Retrier:               retrier,
		SignatureStore:        sigStore,
		RetryTimeout:          retryTimeout,
		IsBackupRelayer:       isBackupRelayer,
		BackupRelayerWaitTime: backupRelayerWaitTime,
	}
}

func (r *Relayer) Start(ctx context.Context) error {
	ethClient, err := r.EVMClient.NewEthClient()
	if err != nil {
		r.logger.Error(err.Error())
		return err
	}
	defer ethClient.Close()

	backupRelayerShouldRelay := false
	processFunc := func() error {
		// this function will relay attestations as long as there are confirms. And, after the contract is
		// up-to-date with the chain, it will stop.
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				lastContractNonce, err := r.EVMClient.StateLastEventNonce(&bind.CallOpts{})
				if err != nil {
					return err
				}

				latestNonce, err := r.AppQuerier.QueryLatestAttestationNonce(ctx)
				if err != nil {
					return err
				}

				// If the contract has already the last version, no need to relay anything
				if lastContractNonce >= latestNonce {
					r.logger.Debug("waiting for new nonce", "current_contract_nonce", lastContractNonce)
					return nil
				}

				if r.IsBackupRelayer {
					if !backupRelayerShouldRelay {
						// if the relayer is a backup relayer, sleep for the wait time before checking
						// if the signatures haven't been relayed to relay them.
						r.logger.Debug("waiting for the backup relayer wait time to elapse before trying to relay attestation", "nonce", lastContractNonce+1)
						time.Sleep(r.BackupRelayerWaitTime)
						backupRelayerShouldRelay = true
						continue
					}
				}

				att, err := r.AppQuerier.QueryAttestationByNonce(ctx, lastContractNonce+1)
				if err != nil {
					return err
				}
				if att == nil {
					return ErrAttestationNotFound
				}

				opts, err := r.EVMClient.NewTransactionOpts(ctx)
				if err != nil {
					return err
				}

				tx, err := r.ProcessAttestation(ctx, opts, att)
				if err != nil {
					return err
				}

				err = r.waitForTransactionAndRetryIfNeeded(ctx, ethClient, opts, tx)
				if err != nil {
					return err
				}

				if r.IsBackupRelayer {
					// if the transaction was mined correctly, the relayer gets back to the pending
					// state waiting for the next nonce + the backup relayer wait time.
					backupRelayerShouldRelay = false
				}
			}
		}
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			// using an immediate ticker not to wait the initial wait period before starting to relay
			err := helpers.ImmediateTicker(
				ctx,
				100*time.Second,
				processFunc,
			)
			if err != nil {
				// if an error occurs, retry a few times before exiting
				r.logger.Error(err.Error())
				err = r.Retrier.Retry(ctx, processFunc)
				if err != nil {
					return err
				}
			}
		}
	}
}

func (r *Relayer) ProcessAttestation(ctx context.Context, opts *bind.TransactOpts, attI celestiatypes.AttestationRequestI) (*coregethtypes.Transaction, error) {
	previousValset, err := r.AppQuerier.QueryLastValsetBeforeNonce(ctx, attI.GetNonce())
	if err != nil {
		r.logger.Debug("failed to query the last valset before nonce (probably pruned). recovering via falling back to the P2P network", "err", err.Error())
		previousValset, err = r.QueryValsetFromP2PNetworkAndValidateIt(ctx)
		if err != nil {
			r.logger.Debug("failed to query the last valset before nonce from p2p network. attempting via using an archive node (might take some time)", "err", err.Error())
			currentHeight, err := r.TmQuerier.QueryHeight(ctx)
			if err != nil {
				return nil, err
			}
			previousValset, err = r.AppQuerier.QueryRecursiveHistoricalLastValsetBeforeNonce(ctx, attI.GetNonce(), uint64(currentHeight))
			if err != nil {
				return nil, err
			}
		}
		r.logger.Debug("found the needed valset")
	}
	switch att := attI.(type) {
	case *celestiatypes.Valset:
		signBytes, err := att.SignBytes()
		if err != nil {
			return nil, err
		}
		confirms, err := r.P2PQuerier.QueryTwoThirdsValsetConfirms(ctx, 30*time.Minute, 10*time.Second, att.Nonce, *previousValset, signBytes.Hex())
		if err != nil {
			return nil, err
		}
		err = r.SaveValsetSignaturesToStore(ctx, *att, confirms)
		if err != nil {
			return nil, err
		}
		tx, err := r.UpdateValidatorSet(ctx, opts, *att, att.TwoThirdsThreshold(), confirms)
		if err != nil {
			return nil, err
		}
		return tx, nil
	case *celestiatypes.DataCommitment:
		commitment, err := r.TmQuerier.QueryCommitment(ctx, att.BeginBlock, att.EndBlock)
		if err != nil {
			return nil, err
		}
		dataRootHash := types.DataCommitmentTupleRootSignBytes(big.NewInt(int64(att.Nonce)), commitment)
		confirms, err := r.P2PQuerier.QueryTwoThirdsDataCommitmentConfirms(ctx, 30*time.Minute, 10*time.Second, *previousValset, att.Nonce, dataRootHash.Hex())
		if err != nil {
			return nil, err
		}
		err = r.SaveDataCommitmentSignaturesToStore(ctx, *att, dataRootHash.String(), confirms)
		if err != nil {
			return nil, err
		}
		tx, err := r.SubmitDataRootTupleRoot(opts, *att, *previousValset, commitment.String(), confirms)
		if err != nil {
			return nil, err
		}
		return tx, nil
	default:
		return nil, errors.Wrap(types.ErrUnknownAttestationType, strconv.FormatUint(attI.GetNonce(), 10))
	}
}

// QueryValsetFromP2PNetworkAndValidateIt Queries the latest valset from the P2P network
// and validates it against the validator set hash used in the contract.
func (r *Relayer) QueryValsetFromP2PNetworkAndValidateIt(ctx context.Context) (*celestiatypes.Valset, error) {
	latestValset, err := r.P2PQuerier.QueryLatestValset(ctx)
	if err != nil {
		return nil, err
	}
	vs := latestValset.ToValset()
	vsHash, err := vs.SignBytes()
	if err != nil {
		return nil, err
	}
	r.logger.Debug("found the latest valset in P2P network. Authenticating it against the contract to verify it's valid", "nonce", vs.Nonce, "hash", vsHash.Hex())

	contractHash, err := r.EVMClient.StateLastValidatorSetCheckpoint(&bind.CallOpts{Context: ctx})
	if err != nil {
		return nil, err
	}

	bzVSHash, err := hex.DecodeString(vsHash.Hex()[2:])
	if err != nil {
		return nil, err
	}

	if !bytes.Equal(bzVSHash, contractHash[:]) {
		r.logger.Error("valset hash from contract mismatches that of P2P one, halting. try running the relayer with an archive node (if that's not the case) to continue relaying", "contract_vs_hash", ethcmn.Bytes2Hex(contractHash[:]), "p2p_vs_hash", vsHash.Hex())
		return nil, ErrValidatorSetMismatch
	}

	r.logger.Info("valset is valid. continuing relaying using the latest valset from P2P network", "nonce", vs.Nonce)
	return vs, nil
}

func (r *Relayer) UpdateValidatorSet(
	ctx context.Context,
	opts *bind.TransactOpts,
	valset celestiatypes.Valset,
	newThreshold uint64,
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
	// to fetch the signatures easily by eth address
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
		newThreshold,
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
	// to fetch the signatures easily by eth address
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

func (r *Relayer) SaveValsetSignaturesToStore(ctx context.Context, att celestiatypes.Valset, confirms []types.ValsetConfirm) error {
	batch, err := r.SignatureStore.Batch(ctx)
	if err != nil {
		return err
	}
	signBytes, err := att.SignBytes()
	if err != nil {
		return err
	}
	for _, confirm := range confirms {
		key := datastore.NewKey(p2p.GetValsetConfirmKey(att.Nonce, confirm.EthAddress, signBytes.Hex()))
		value, err := types.MarshalValsetConfirm(confirm)
		if err != nil {
			return err
		}
		has, err := r.SignatureStore.Has(ctx, key)
		if err != nil {
			return err
		}
		if !has {
			err := batch.Put(ctx, key, value)
			if err != nil {
				return err
			}
		}
	}
	return batch.Commit(ctx)
}

func (r *Relayer) SaveDataCommitmentSignaturesToStore(ctx context.Context, att celestiatypes.DataCommitment, dataRootTupleRoot string, confirms []types.DataCommitmentConfirm) error {
	batch, err := r.SignatureStore.Batch(ctx)
	if err != nil {
		return err
	}
	for _, confirm := range confirms {
		key := datastore.NewKey(p2p.GetDataCommitmentConfirmKey(att.Nonce, confirm.EthAddress, dataRootTupleRoot))
		value, err := types.MarshalDataCommitmentConfirm(confirm)
		if err != nil {
			return err
		}
		has, err := r.SignatureStore.Has(ctx, key)
		if err != nil {
			return err
		}
		if !has {
			err := batch.Put(ctx, key, value)
			if err != nil {
				return err
			}
		}
	}
	return batch.Commit(ctx)
}

// waitForTransactionAndRetryIfNeeded waits for transaction to be mined. If it's not mined in the provided timeout, it will
// attempt to speed it up via updating the gas price.
func (r *Relayer) waitForTransactionAndRetryIfNeeded(ctx context.Context, ethClient *ethclient.Client, opts *bind.TransactOpts, tx *coregethtypes.Transaction) error {
	r.logger.Debug("submitted transaction", "hash", tx.Hash().Hex(), "gas_price", tx.GasPrice().Uint64())
	newTx := tx
	for i := 0; i < 10; i++ {
		_, err := r.EVMClient.WaitForTransaction(ctx, ethClient, newTx, r.RetryTimeout)
		if err != nil {
			if stderrors.Is(err, context.DeadlineExceeded) {
				var rawTx *coregethtypes.Transaction
				if tx.GasPrice() != nil {
					rawTx, err = createSpeededUpLegacyTransaction(ctx, ethClient, newTx)
					if err != nil {
						return err
					}
					if rawTx.GasPrice().Cmp(newTx.GasPrice()) <= 0 {
						// no need to resend the transaction if the suggested gas price is lower than the original one
						continue
					}
				} else if tx.GasTipCap() != nil && tx.GasFeeCap() != nil {
					rawTx, err = createSpeededUpDynamicTransaction(ctx, ethClient, newTx)
					if err != nil {
						return err
					}
					if rawTx.GasFeeCap().Cmp(newTx.GasFeeCap()) <= 0 {
						// no need to resend the transaction if the suggested gas price is lower than the original one
						continue
					}
				} else {
					// Only query for basefee if gasPrice not specified
					if head, errHead := ethClient.HeaderByNumber(ctx, nil); errHead != nil {
						return errHead
					} else if head.BaseFee != nil {
						rawTx, err = createSpeededUpDynamicTransaction(ctx, ethClient, newTx)
						if err != nil {
							return err
						}
						if rawTx.GasFeeCap().Cmp(newTx.GasFeeCap()) <= 0 {
							// no need to resend the transaction if the suggested gas price is lower than the original one
							continue
						}
					} else {
						// Chain is not London ready -> use legacy transaction
						rawTx, err = createSpeededUpLegacyTransaction(ctx, ethClient, newTx)
						if err != nil {
							return err
						}
						if rawTx.GasPrice().Cmp(newTx.GasPrice()) <= 0 {
							// no need to resend the transaction if the suggested gas price is lower than the original one
							continue
						}
					}
				}
				r.logger.Debug("transaction still not included. updating the gas price", "retry_number", i)
				signedTx, err := opts.Signer(opts.From, rawTx)
				if err != nil {
					return err
				}
				err = ethClient.SendTransaction(ctx, signedTx)
				r.logger.Info("submitted speed up transaction", "hash", signedTx.Hash().Hex(), "new_gas_price", signedTx.GasPrice().Uint64())
				if err != nil {
					r.logger.Debug("response of sending speed up transaction", "resp", err.Error())
				}
			} else {
				return err
			}
		} else {
			return nil
		}
	}
	return ErrTransactionStillPending
}

// createSpeededUpDynamicTransaction update the EIP1559 dynamic transaction with the current gas price.
func createSpeededUpDynamicTransaction(ctx context.Context, ethClient *ethclient.Client, newTx *coregethtypes.Transaction) (*coregethtypes.Transaction, error) {
	// Estimate TipCap
	gasTipCap, err := ethClient.SuggestGasTipCap(ctx)
	if err != nil {
		return nil, err
	}
	lastKnownHeader, err := ethClient.HeaderByNumber(ctx, nil)
	if err != nil {
		return nil, err
	}
	// Estimate FeeCap
	gasFeeCap := new(big.Int).Add(
		gasTipCap,
		// the DefaultElasticityMultiplier is used to define the wiggle room for the gas
		// in EIP1559
		new(big.Int).Mul(lastKnownHeader.BaseFee, big.NewInt(params.DefaultElasticityMultiplier)),
	)
	if gasFeeCap.Cmp(gasTipCap) < 0 {
		return nil, fmt.Errorf("maxFeePerGas (%v) < maxPriorityFeePerGas (%v)", gasFeeCap, gasTipCap)
	}

	dynamicTransaction := toDynamicTransaction(newTx)
	dynamicTransaction.GasTipCap = gasTipCap
	dynamicTransaction.GasFeeCap = gasFeeCap
	return coregethtypes.NewTx(dynamicTransaction), nil
}

// createSpeededUpLegacyTransaction update the legacy transaction with the new gas price.
func createSpeededUpLegacyTransaction(ctx context.Context, ethClient *ethclient.Client, newTx *coregethtypes.Transaction) (tx *coregethtypes.Transaction, err error) {
	newGasPrice, err := ethClient.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}
	legacyTx := toLegacyTransaction(newTx)
	legacyTx.GasPrice = newGasPrice
	return coregethtypes.NewTx(legacyTx), nil
}

func toLegacyTransaction(tx *coregethtypes.Transaction) *coregethtypes.LegacyTx {
	v, r, s := tx.RawSignatureValues()
	return &coregethtypes.LegacyTx{
		Nonce:    tx.Nonce(),
		GasPrice: tx.GasPrice(),
		Gas:      tx.Gas(),
		To:       tx.To(),
		Value:    tx.Value(),
		Data:     tx.Data(),
		V:        v,
		R:        r,
		S:        s,
	}
}

func toDynamicTransaction(tx *coregethtypes.Transaction) *coregethtypes.DynamicFeeTx {
	v, r, s := tx.RawSignatureValues()
	return &coregethtypes.DynamicFeeTx{
		ChainID:    tx.ChainId(),
		Nonce:      tx.Nonce(),
		GasTipCap:  tx.GasTipCap(),
		GasFeeCap:  tx.GasFeeCap(),
		Gas:        tx.Gas(),
		To:         tx.To(),
		Value:      tx.Value(),
		Data:       tx.Data(),
		AccessList: tx.AccessList(),
		V:          v,
		R:          r,
		S:          s,
	}
}

// matchAttestationConfirmSigs matches and sorts the confirm signatures with the valset
// members as expected by the Blobstream contract.
// Also, it leaves the non provided signatures as nil in the `sigs` slice:
// https://github.com/celestiaorg/celestia-app/issues/628
func matchAttestationConfirmSigs(
	signatures map[string]string,
	currentValset celestiatypes.Valset,
) ([]wrapper.Signature, error) {
	sigs := make([]wrapper.Signature, len(currentValset.Members))
	// the Blobstream contract expects the signatures to be ordered by validators in valset
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
