package orchestrator

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/celestiaorg/celestia-app/app/encoding"
	"github.com/celestiaorg/orchestrator-relayer/x/qgb/types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/libs/bytes"
	tmlog "github.com/tendermint/tendermint/libs/log"
	"github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ Querier = &querier{}

type Querier interface {
	// attestation queries

	// QueryAttestationByNonce Queries the attestation with nonce `nonce.
	// Returns nil if not found.
	QueryAttestationByNonce(ctx context.Context, nonce uint64) (types.AttestationRequestI, error)
	QueryLatestAttestationNonce(ctx context.Context) (uint64, error)

	// data commitment queries

	QueryDataCommitmentByNonce(ctx context.Context, nonce uint64) (*types.DataCommitment, error)

	// data commitment confirm queries

	QueryDataCommitmentConfirm(
		ctx context.Context,
		endBlock uint64,
		beginBlock uint64,
		address string,
	) (*types.MsgDataCommitmentConfirm, error)
	QueryDataCommitmentConfirmsByExactRange(
		ctx context.Context,
		start uint64,
		end uint64,
	) ([]types.MsgDataCommitmentConfirm, error)
	QueryTwoThirdsDataCommitmentConfirms(
		ctx context.Context,
		timeout time.Duration,
		dc types.DataCommitment,
	) ([]types.MsgDataCommitmentConfirm, error)

	// valset queries

	QueryValsetByNonce(ctx context.Context, nonce uint64) (*types.Valset, error)
	QueryLatestValset(ctx context.Context) (*types.Valset, error)
	QueryLastValsetBeforeNonce(
		ctx context.Context,
		nonce uint64,
	) (*types.Valset, error)

	// valset confirm queries

	QueryTwoThirdsValsetConfirms(
		ctx context.Context,
		timeout time.Duration,
		valset types.Valset,
	) ([]types.MsgValsetConfirm, error)
	QueryValsetConfirm(ctx context.Context, nonce uint64, address string) (*types.MsgValsetConfirm, error)

	// misc queries

	QueryHeight(ctx context.Context) (uint64, error)
	QueryLastUnbondingHeight(ctx context.Context) (uint64, error)

	// tendermint

	QueryCommitment(ctx context.Context, beginBlock uint64, endBlock uint64) (bytes.HexBytes, error)
	SubscribeEvents(ctx context.Context, subscriptionName string, eventName string) (<-chan coretypes.ResultEvent, error)
}

type querier struct {
	qgbRPC        *grpc.ClientConn
	logger        tmlog.Logger
	tendermintRPC *http.HTTP
	encCfg        encoding.Config
}

func NewQuerier(
	qgbRPCAddr, tendermintRPC string,
	logger tmlog.Logger,
	encCft encoding.Config,
) (*querier, error) {
	qgbGRPC, err := grpc.Dial(qgbRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	trpc, err := http.New(tendermintRPC, "/websocket")
	if err != nil {
		return nil, err
	}
	err = trpc.Start()
	if err != nil {
		return nil, err
	}

	return &querier{
		qgbRPC:        qgbGRPC,
		logger:        logger,
		tendermintRPC: trpc,
		encCfg:        encCft,
	}, nil
}

// TODO add the other stop methods for other clients.
func (q querier) Stop() {
	err := q.qgbRPC.Close()
	if err != nil {
		q.logger.Error(err.Error())
	}
	err = q.tendermintRPC.Stop()
	if err != nil {
		q.logger.Error(err.Error())
	}
}

func (q *querier) QueryTwoThirdsDataCommitmentConfirms(
	ctx context.Context,
	timeout time.Duration,
	dc types.DataCommitment,
) ([]types.MsgDataCommitmentConfirm, error) {
	valset, err := q.QueryLastValsetBeforeNonce(ctx, dc.Nonce)
	if err != nil {
		return nil, err
	}

	// create a map to easily search for power
	vals := make(map[string]types.BridgeValidator)
	for _, val := range valset.Members {
		vals[val.GetEvmAddress()] = val
	}

	majThreshHold := valset.TwoThirdsThreshold()

	for {
		select {
		case <-ctx.Done():
			return nil, nil //nolint:nilnil
		case <-time.After(timeout):
			return nil, errors.Wrap(
				ErrNotEnoughDataCommitmentConfirms,
				fmt.Sprintf("failure to query for majority validator set confirms: timout %s", timeout),
			)
		default:
			currThreshold := uint64(0)
			confirms, err := q.QueryDataCommitmentConfirmsByExactRange(ctx, dc.BeginBlock, dc.EndBlock)
			if err != nil {
				return nil, err
			}

			// used to be tested against when checking if all confirms have the correct commitment.
			// this can be extended to slashing the validators who submitted commitments that
			// have wrong commitments or signatures.
			// https://github.com/celestiaorg/celestia-app/pull/613/files#r947992851
			commitment, err := q.QueryCommitment(ctx, dc.BeginBlock, dc.EndBlock)
			if err != nil {
				return nil, err
			}

			correctConfirms := make([]types.MsgDataCommitmentConfirm, 0)
			for _, dataCommitmentConfirm := range confirms {
				val, has := vals[dataCommitmentConfirm.EvmAddress]
				if !has {
					q.logger.Debug(fmt.Sprintf(
						"dataCommitmentConfirm signer not found in stored validator set: address %s nonce %d",
						val.EvmAddress,
						valset.Nonce,
					))
					continue
				}
				if err := validateDCConfirm(commitment.String(), dataCommitmentConfirm); err != nil {
					q.logger.Error("found an invalid data commitment confirm",
						"nonce",
						dataCommitmentConfirm.Nonce,
						"signer_evm_address",
						dataCommitmentConfirm.EvmAddress,
						"err",
						err.Error(),
					)
					continue
				}
				currThreshold += val.Power
				correctConfirms = append(correctConfirms, dataCommitmentConfirm)
			}

			if currThreshold >= majThreshHold {
				q.logger.Debug("found enough data commitment confirms to be relayed",
					"majThreshHold",
					majThreshHold,
					"currThreshold",
					currThreshold,
				)
				return correctConfirms, nil
			}
			q.logger.Debug(
				"found DataCommitmentConfirms",
				"begin_block",
				dc.BeginBlock,
				"end_block",
				dc.EndBlock,
				"total_power",
				currThreshold,
				"number_of_confirms",
				len(confirms),
				"missing_confirms",
				len(valset.Members)-len(confirms),
			)
		}
		// TODO: make the timeout configurable
		time.Sleep(10 * time.Second)
	}
}

// validateDCConfirm runs validation on the data commitment confirm to make sure it was well created.
// it tests if the commitment it carries is the correct commitment. Then, checks whether the signature
// is valid.
func validateDCConfirm(commitment string, confirm types.MsgDataCommitmentConfirm) error {
	if confirm.Commitment != commitment {
		return ErrInvalidCommitmentInConfirm
	}
	bCommitment := common.Hex2Bytes(commitment)
	dataRootHash := types.DataCommitmentTupleRootSignBytes(types.BridgeID, big.NewInt(int64(confirm.Nonce)), bCommitment)
	err := types.ValidateEthereumSignature(dataRootHash.Bytes(), common.Hex2Bytes(confirm.Signature), common.HexToAddress(confirm.EvmAddress))
	if err != nil {
		return err
	}
	return nil
}

func (q querier) QueryTwoThirdsValsetConfirms(
	ctx context.Context,
	timeout time.Duration,
	valset types.Valset,
) ([]types.MsgValsetConfirm, error) {
	var currentValset types.Valset
	if valset.Nonce == 1 {
		// In fact, the first nonce should never be signed. Because, the first attestation, in the case
		// where the `earliest` flag is specified when deploying the contract, will be relayed as part of
		// the deployment of the QGB contract.
		// It will be signed temporarily for now.
		currentValset = valset
	} else {
		vs, err := q.QueryLastValsetBeforeNonce(ctx, valset.Nonce)
		if err != nil {
			return nil, err
		}
		currentValset = *vs
	}
	// create a map to easily search for power
	vals := make(map[string]types.BridgeValidator)
	for _, val := range currentValset.Members {
		vals[val.GetEvmAddress()] = val
	}

	majThreshHold := valset.TwoThirdsThreshold()

	for {
		select {
		case <-ctx.Done():
			return nil, nil //nolint:nilnil
		// TODO: remove this extra case, and we can instead rely on the caller to pass a context with a timeout
		case <-time.After(timeout):
			return nil, errors.Wrap(
				ErrNotEnoughValsetConfirms,
				fmt.Sprintf("failure to query for majority validator set confirms: timout %s", timeout),
			)
		default:
			currThreshold := uint64(0)
			queryClient := types.NewQueryClient(q.qgbRPC)
			confirmsResp, err := queryClient.ValsetConfirmsByNonce(ctx, &types.QueryValsetConfirmsByNonceRequest{
				Nonce: valset.Nonce,
			})
			if err != nil {
				return nil, err
			}

			confirms := make([]types.MsgValsetConfirm, 0)
			for _, valsetConfirm := range confirmsResp.Confirms {
				val, has := vals[valsetConfirm.EvmAddress]
				if !has {
					q.logger.Debug(
						fmt.Sprintf(
							"valSetConfirm signer not found in stored validator set: address %s nonce %d",
							val.EvmAddress,
							valset.Nonce,
						))
					continue
				}
				if err := validateValsetConfirm(valset, valsetConfirm); err != nil {
					q.logger.Error("found an invalid valset confirm",
						"nonce",
						valsetConfirm.Nonce,
						"signer_evm_address",
						valsetConfirm.EvmAddress,
						"err",
						err.Error(),
					)
					continue
				}
				currThreshold += val.Power
				confirms = append(confirms, valsetConfirm)
			}

			if currThreshold >= majThreshHold {
				q.logger.Debug("found enough valset confirms to be relayed",
					"majThreshHold",
					majThreshHold,
					"currThreshold",
					currThreshold,
				)
				return confirms, nil
			}
			q.logger.Debug(
				"found ValsetConfirms",
				"nonce",
				valset.Nonce,
				"total_power",
				currThreshold,
				"number_of_confirms",
				len(confirmsResp.Confirms),
				"missing_confirms",
				len(currentValset.Members)-len(confirmsResp.Confirms),
			)
		}
		// TODO: make the timeout configurable
		time.Sleep(10 * time.Second)
	}
}

// validateValsetConfirm runs validation on the valset confirm to make sure it was well created.
// For now, it only checks if the signature is correct. Can be improved afterwards.
func validateValsetConfirm(vs types.Valset, confirm types.MsgValsetConfirm) error {
	signBytes, err := vs.SignBytes(types.BridgeID)
	if err != nil {
		return err
	}
	err = types.ValidateEthereumSignature(signBytes.Bytes(), common.Hex2Bytes(confirm.Signature), common.HexToAddress(confirm.EvmAddress))
	if err != nil {
		return err
	}
	return nil
}

// QueryLastValsetBeforeNonce returns the last valset before nonce.
// the provided `nonce` can be a valset, but this will return the valset before it.
// If nonce is 1, it will return an error. Because, there is no valset before nonce 1.
func (q querier) QueryLastValsetBeforeNonce(ctx context.Context, nonce uint64) (*types.Valset, error) {
	queryClient := types.NewQueryClient(q.qgbRPC)
	resp, err := queryClient.LastValsetRequestBeforeNonce(
		ctx,
		&types.QueryLastValsetRequestBeforeNonceRequest{Nonce: nonce},
	)
	if err != nil {
		resp, err2 := q.QueryHeight(ctx)
		if err2 != nil {
			return nil, err
		}
		return &types.Valset{
			Nonce:   nonce,
			Members: nil,
			Height:  resp,
		}, nil
	}

	return resp.Valset, nil
}

func (q querier) QueryValsetConfirm(
	ctx context.Context,
	nonce uint64,
	address string,
) (*types.MsgValsetConfirm, error) {
	queryClient := types.NewQueryClient(q.qgbRPC)
	// FIXME this is not always a valset confirm (the nonce can be of a data commitment)
	// and might return an empty list. Should we worry?
	resp, err := queryClient.ValsetConfirm(ctx, &types.QueryValsetConfirmRequest{Nonce: nonce, Address: address})
	if err != nil {
		return nil, err
	}

	return resp.Confirm, nil
}

func (q querier) QueryHeight(ctx context.Context) (uint64, error) {
	resp, err := q.tendermintRPC.Status(ctx)
	if err != nil {
		return 0, err
	}

	return uint64(resp.SyncInfo.LatestBlockHeight), nil
}

func (q querier) QueryLastUnbondingHeight(ctx context.Context) (uint64, error) {
	queryClient := types.NewQueryClient(q.qgbRPC)
	resp, err := queryClient.LastUnbondingHeight(ctx, &types.QueryLastUnbondingHeightRequest{})
	if err != nil {
		return 0, err
	}

	return resp.Height, nil
}

func (q querier) QueryDataCommitmentConfirm(
	ctx context.Context,
	endBlock uint64,
	beginBlock uint64,
	address string,
) (*types.MsgDataCommitmentConfirm, error) {
	queryClient := types.NewQueryClient(q.qgbRPC)

	confirmsResp, err := queryClient.DataCommitmentConfirm(
		ctx,
		&types.QueryDataCommitmentConfirmRequest{
			EndBlock:   endBlock,
			BeginBlock: beginBlock,
			Address:    address,
		},
	)
	if err != nil {
		return nil, err
	}

	return confirmsResp.Confirm, nil
}

func (q querier) QueryDataCommitmentConfirmsByExactRange(
	ctx context.Context,
	start uint64,
	end uint64,
) ([]types.MsgDataCommitmentConfirm, error) {
	queryClient := types.NewQueryClient(q.qgbRPC)
	confirmsResp, err := queryClient.DataCommitmentConfirmsByExactRange(
		ctx,
		&types.QueryDataCommitmentConfirmsByExactRangeRequest{
			BeginBlock: start,
			EndBlock:   end,
		},
	)
	if err != nil {
		return nil, err
	}
	return confirmsResp.Confirms, nil
}

func (q querier) QueryDataCommitmentByNonce(ctx context.Context, nonce uint64) (*types.DataCommitment, error) {
	attestation, err := q.QueryAttestationByNonce(ctx, nonce)
	if err != nil {
		return nil, err
	}
	if attestation == nil {
		return nil, types.ErrAttestationNotFound
	}

	if attestation.Type() != types.DataCommitmentRequestType {
		return nil, types.ErrAttestationNotDataCommitmentRequest
	}

	dcc, ok := attestation.(*types.DataCommitment)
	if !ok {
		return nil, types.ErrAttestationNotDataCommitmentRequest
	}

	return dcc, nil
}

func (q querier) QueryAttestationByNonce(
	ctx context.Context,
	nonce uint64,
) (types.AttestationRequestI, error) { // FIXME is it alright to return interface?
	queryClient := types.NewQueryClient(q.qgbRPC)
	atResp, err := queryClient.AttestationRequestByNonce(
		ctx,
		&types.QueryAttestationRequestByNonceRequest{Nonce: nonce},
	)
	if err != nil {
		return nil, err
	}
	if atResp.Attestation == nil {
		return nil, nil
	}

	unmarshalledAttestation, err := q.unmarshallAttestation(atResp.Attestation)
	if err != nil {
		return nil, err
	}

	return unmarshalledAttestation, nil
}

func (q querier) QueryValsetByNonce(ctx context.Context, nonce uint64) (*types.Valset, error) {
	attestation, err := q.QueryAttestationByNonce(ctx, nonce)
	if err != nil {
		return nil, err
	}
	if attestation == nil {
		return nil, types.ErrAttestationNotFound
	}

	if attestation.Type() != types.ValsetRequestType {
		return nil, types.ErrAttestationNotValsetRequest
	}

	value, ok := attestation.(*types.Valset)
	if !ok {
		return nil, ErrUnmarshallValset
	}

	return value, nil
}

func (q querier) QueryLatestValset(ctx context.Context) (*types.Valset, error) {
	latestNonce, err := q.QueryLatestAttestationNonce(ctx)
	if err != nil {
		return nil, err
	}

	var latestValset *types.Valset
	if vs, err := q.QueryValsetByNonce(ctx, latestNonce); err == nil {
		latestValset = vs
	} else {
		latestValset, err = q.QueryLastValsetBeforeNonce(ctx, latestNonce)
		if err != nil {
			return nil, err
		}
	}
	return latestValset, nil
}

func (q querier) QueryLatestAttestationNonce(ctx context.Context) (uint64, error) {
	queryClient := types.NewQueryClient(q.qgbRPC)

	resp, err := queryClient.LatestAttestationNonce(
		ctx,
		&types.QueryLatestAttestationNonceRequest{},
	)
	if err != nil {
		return 0, err
	}

	return resp.Nonce, nil
}

// QueryCommitment queries the commitment over a set of blocks defined in the query.
func (q querier) QueryCommitment(ctx context.Context, beginBlock uint64, endBlock uint64) (bytes.HexBytes, error) {
	dcResp, err := q.tendermintRPC.DataCommitment(ctx, beginBlock, endBlock)
	if err != nil {
		return nil, err
	}
	return dcResp.DataCommitment, nil
}

func (q querier) SubscribeEvents(ctx context.Context, subscriptionName string, eventName string) (<-chan coretypes.ResultEvent, error) {
	// This doesn't seem to complain when the node is down
	results, err := q.tendermintRPC.Subscribe(
		ctx,
		"attestation-changes",
		fmt.Sprintf("%s.%s='%s'", types.EventTypeAttestationRequest, sdk.AttributeKeyModule, types.ModuleName),
	)
	if err != nil {
		return nil, err
	}
	return results, err
}

func (q querier) unmarshallAttestation(attestation *cdctypes.Any) (types.AttestationRequestI, error) {
	var unmarshalledAttestation types.AttestationRequestI
	err := q.encCfg.InterfaceRegistry.UnpackAny(attestation, &unmarshalledAttestation)
	if err != nil {
		return nil, err
	}
	return unmarshalledAttestation, nil
}
