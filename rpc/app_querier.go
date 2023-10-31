package rpc

import (
	"context"
	"strconv"

	"github.com/celestiaorg/celestia-app/pkg/appconsts"
	cosmosgrpc "github.com/cosmos/cosmos-sdk/types/grpc"
	"google.golang.org/grpc/metadata"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/celestiaorg/orchestrator-relayer/types"

	"github.com/celestiaorg/celestia-app/app/encoding"
	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

var BlocksIn20DaysPeriod = 20 * 24 * 60 * 60 / appconsts.TimeoutCommit.Seconds()

// AppQuerier queries the application for attestations and unbonding periods.
type AppQuerier struct {
	blobStreamRPC string
	clientConn    *grpc.ClientConn
	Logger        tmlog.Logger
	EncCfg        encoding.Config
}

func NewAppQuerier(logger tmlog.Logger, blobStreamRPC string, encCft encoding.Config) *AppQuerier {
	return &AppQuerier{Logger: logger, blobStreamRPC: blobStreamRPC, EncCfg: encCft}
}

func (aq *AppQuerier) Start() error {
	// creating a grpc connection to Celestia-app
	blobStreamGRPC, err := grpc.Dial(aq.blobStreamRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	aq.clientConn = blobStreamGRPC
	return nil
}

func (aq *AppQuerier) Stop() error {
	return aq.clientConn.Close()
}

// QueryAttestationByNonce query an attestation by nonce from the state machine.
func (aq *AppQuerier) QueryAttestationByNonce(ctx context.Context, nonce uint64) (celestiatypes.AttestationRequestI, error) {
	queryClient := celestiatypes.NewQueryClient(aq.clientConn)

	atResp, err := queryClient.AttestationRequestByNonce(
		ctx,
		&celestiatypes.QueryAttestationRequestByNonceRequest{Nonce: nonce},
	)
	if err != nil {
		return nil, err
	}
	if atResp.Attestation == nil {
		return nil, nil
	}

	unmarshalledAttestation, err := aq.unmarshallAttestation(atResp.Attestation)
	if err != nil {
		return nil, err
	}

	return unmarshalledAttestation, nil
}

// QueryHistoricalAttestationByNonce query an attestation by nonce from the state machine at a certain height.
func (aq *AppQuerier) QueryHistoricalAttestationByNonce(ctx context.Context, nonce uint64, height uint64) (celestiatypes.AttestationRequestI, error) {
	queryClient := celestiatypes.NewQueryClient(aq.clientConn)

	var header metadata.MD
	atResp, err := queryClient.AttestationRequestByNonce(
		metadata.AppendToOutgoingContext(ctx, cosmosgrpc.GRPCBlockHeightHeader, strconv.FormatUint(height, 10)), // Add metadata to request
		&celestiatypes.QueryAttestationRequestByNonceRequest{Nonce: nonce},
		grpc.Header(&header), // Retrieve header from response
	)
	if err != nil {
		return nil, err
	}
	if atResp.Attestation == nil {
		return nil, nil
	}

	unmarshalledAttestation, err := aq.unmarshallAttestation(atResp.Attestation)
	if err != nil {
		return nil, err
	}

	return unmarshalledAttestation, nil
}

// QueryRecursiveHistoricalAttestationByNonce query an attestation by nonce from the state machine
// via going over the history step by step starting from height.
func (aq *AppQuerier) QueryRecursiveHistoricalAttestationByNonce(ctx context.Context, nonce uint64, height uint64) (celestiatypes.AttestationRequestI, error) {
	queryClient := celestiatypes.NewQueryClient(aq.clientConn)

	currentHeight := height
	for currentHeight >= 1 {
		var header metadata.MD
		atResp, err := queryClient.AttestationRequestByNonce(
			metadata.AppendToOutgoingContext(ctx, cosmosgrpc.GRPCBlockHeightHeader, strconv.FormatUint(currentHeight, 10)), // Add metadata to request
			&celestiatypes.QueryAttestationRequestByNonceRequest{Nonce: nonce},
			grpc.Header(&header), // Retrieve header from response
		)
		if err == nil {
			unmarshalledAttestation, err := aq.unmarshallAttestation(atResp.Attestation)
			if err != nil {
				return nil, err
			}
			return unmarshalledAttestation, nil
		}
		aq.Logger.Debug("keeping looking for attestation in archival state", "err", err.Error())
		currentHeight -= uint64(BlocksIn20DaysPeriod)
	}
	return nil, ErrNotFound
}

// QueryLatestAttestationNonce query the latest attestation nonce from the state machine.
func (aq *AppQuerier) QueryLatestAttestationNonce(ctx context.Context) (uint64, error) {
	queryClient := celestiatypes.NewQueryClient(aq.clientConn)

	resp, err := queryClient.LatestAttestationNonce(
		ctx,
		&celestiatypes.QueryLatestAttestationNonceRequest{},
	)
	if err != nil {
		return 0, err
	}

	return resp.Nonce, nil
}

// QueryHistoricalLatestAttestationNonce query the historical latest attestation nonce from the state machine at a certain nonce.
func (aq *AppQuerier) QueryHistoricalLatestAttestationNonce(ctx context.Context, height uint64) (uint64, error) {
	queryClient := celestiatypes.NewQueryClient(aq.clientConn)

	var header metadata.MD
	resp, err := queryClient.LatestAttestationNonce(
		metadata.AppendToOutgoingContext(ctx, cosmosgrpc.GRPCBlockHeightHeader, strconv.FormatUint(height, 10)),
		&celestiatypes.QueryLatestAttestationNonceRequest{},
		grpc.Header(&header),
	)
	if err != nil {
		return 0, err
	}

	return resp.Nonce, nil
}

// QueryDataCommitmentByNonce query a data commitment by its nonce.
func (aq *AppQuerier) QueryDataCommitmentByNonce(ctx context.Context, nonce uint64) (*celestiatypes.DataCommitment, error) {
	attestation, err := aq.QueryAttestationByNonce(ctx, nonce)
	if err != nil {
		return nil, err
	}
	if attestation == nil {
		return nil, types.ErrAttestationNotFound
	}

	dcc, ok := attestation.(*celestiatypes.DataCommitment)
	if !ok {
		return nil, types.ErrAttestationNotDataCommitmentRequest
	}

	return dcc, nil
}

// QueryDataCommitmentForHeight query a data commitment by one of the heights that it commits to.
func (aq *AppQuerier) QueryDataCommitmentForHeight(ctx context.Context, height uint64) (*celestiatypes.DataCommitment, error) {
	queryClient := celestiatypes.NewQueryClient(aq.clientConn)
	resp, err := queryClient.DataCommitmentRangeForHeight(ctx, &celestiatypes.QueryDataCommitmentRangeForHeightRequest{Height: height})
	if err != nil {
		return nil, err
	}
	return resp.DataCommitment, nil
}

// QueryLatestDataCommitment query the latest data commitment in Blobstream state machine.
func (aq *AppQuerier) QueryLatestDataCommitment(ctx context.Context) (*celestiatypes.DataCommitment, error) {
	queryClient := celestiatypes.NewQueryClient(aq.clientConn)
	resp, err := queryClient.LatestDataCommitment(ctx, &celestiatypes.QueryLatestDataCommitmentRequest{})
	if err != nil {
		return nil, err
	}
	return resp.DataCommitment, nil
}

// QueryValsetByNonce query a valset by nonce.
func (aq *AppQuerier) QueryValsetByNonce(ctx context.Context, nonce uint64) (*celestiatypes.Valset, error) {
	attestation, err := aq.QueryAttestationByNonce(ctx, nonce)
	if err != nil {
		return nil, err
	}
	if attestation == nil {
		return nil, types.ErrAttestationNotFound
	}

	value, ok := attestation.(*celestiatypes.Valset)
	if !ok {
		return nil, types.ErrUnmarshalValset
	}

	return value, nil
}

// QueryHistoricalValsetByNonce query a historical valset by nonce.
func (aq *AppQuerier) QueryHistoricalValsetByNonce(ctx context.Context, nonce uint64, height uint64) (*celestiatypes.Valset, error) {
	attestation, err := aq.QueryHistoricalAttestationByNonce(ctx, nonce, height)
	if err != nil {
		return nil, err
	}
	if attestation == nil {
		return nil, types.ErrAttestationNotFound
	}

	value, ok := attestation.(*celestiatypes.Valset)
	if !ok {
		return nil, types.ErrUnmarshalValset
	}

	return value, nil
}

// QueryLatestValset query the latest recorded valset in the state machine.
func (aq *AppQuerier) QueryLatestValset(ctx context.Context) (*celestiatypes.Valset, error) {
	latestNonce, err := aq.QueryLatestAttestationNonce(ctx)
	if err != nil {
		return nil, err
	}

	var latestValset *celestiatypes.Valset
	if vs, err := aq.QueryValsetByNonce(ctx, latestNonce); err == nil {
		latestValset = vs
	} else {
		latestValset, err = aq.QueryLastValsetBeforeNonce(ctx, latestNonce)
		if err != nil {
			return nil, err
		}
	}
	return latestValset, nil
}

// QueryRecursiveLatestValset query the latest recorded valset in the state machine in history.
func (aq *AppQuerier) QueryRecursiveLatestValset(ctx context.Context, height uint64) (*celestiatypes.Valset, error) {
	currentHeight := height
	for currentHeight >= 1 {
		latestNonce, err := aq.QueryHistoricalLatestAttestationNonce(ctx, currentHeight)
		if err != nil {
			return nil, err
		}

		var latestValset *celestiatypes.Valset
		if vs, err := aq.QueryHistoricalValsetByNonce(ctx, latestNonce, currentHeight); err == nil {
			latestValset = vs
		} else {
			latestValset, err = aq.QueryHistoricalLastValsetBeforeNonce(ctx, latestNonce, currentHeight)
			if err == nil {
				return latestValset, nil
			}
		}
		aq.Logger.Debug("keeping looking for attestation in archival state", "err", err.Error())
		currentHeight -= uint64(BlocksIn20DaysPeriod)
	}
	return nil, ErrNotFound
}

// QueryLastValsetBeforeNonce returns the last valset before nonce.
// This will be needed when signing to know the validator set at that particular nonce.
// the provided `nonce` can be a valset, but this will return the valset before it.
// If nonce is 1, it will return an error. Because, there is no valset before nonce 1.
func (aq *AppQuerier) QueryLastValsetBeforeNonce(ctx context.Context, nonce uint64) (*celestiatypes.Valset, error) {
	queryClient := celestiatypes.NewQueryClient(aq.clientConn)
	resp, err := queryClient.LatestValsetRequestBeforeNonce(
		ctx,
		&celestiatypes.QueryLatestValsetRequestBeforeNonceRequest{Nonce: nonce},
	)
	if err != nil {
		return nil, err
	}

	return resp.Valset, nil
}

// QueryHistoricalLastValsetBeforeNonce returns the last historical valset before nonce for a certain height.
func (aq *AppQuerier) QueryHistoricalLastValsetBeforeNonce(ctx context.Context, nonce uint64, height uint64) (*celestiatypes.Valset, error) {
	queryClient := celestiatypes.NewQueryClient(aq.clientConn)
	var header metadata.MD
	resp, err := queryClient.LatestValsetRequestBeforeNonce(
		metadata.AppendToOutgoingContext(ctx, cosmosgrpc.GRPCBlockHeightHeader, strconv.FormatUint(height, 10)),
		&celestiatypes.QueryLatestValsetRequestBeforeNonceRequest{Nonce: nonce},
		grpc.Header(&header),
	)
	if err != nil {
		return nil, err
	}

	return resp.Valset, nil
}

// QueryRecursiveHistoricalLastValsetBeforeNonce recursively looks for the last historical valset before nonce for a certain height until genesis.
func (aq *AppQuerier) QueryRecursiveHistoricalLastValsetBeforeNonce(ctx context.Context, nonce uint64, height uint64) (*celestiatypes.Valset, error) {
	queryClient := celestiatypes.NewQueryClient(aq.clientConn)

	currentHeight := height
	for currentHeight >= 1 {
		var header metadata.MD
		resp, err := queryClient.LatestValsetRequestBeforeNonce(
			metadata.AppendToOutgoingContext(ctx, cosmosgrpc.GRPCBlockHeightHeader, strconv.FormatUint(height, 10)),
			&celestiatypes.QueryLatestValsetRequestBeforeNonceRequest{Nonce: nonce},
			grpc.Header(&header),
		)
		if err == nil {
			return resp.Valset, err
		}
		aq.Logger.Debug("keeping looking for attestation in archival state", "err", err.Error())
		currentHeight -= uint64(BlocksIn20DaysPeriod)
	}
	return nil, ErrNotFound
}

// QueryLastUnbondingHeight query the last unbonding height from state machine.
func (aq *AppQuerier) QueryLastUnbondingHeight(ctx context.Context) (int64, error) {
	queryClient := celestiatypes.NewQueryClient(aq.clientConn)
	resp, err := queryClient.LatestUnbondingHeight(ctx, &celestiatypes.QueryLatestUnbondingHeightRequest{})
	if err != nil {
		return 0, err
	}

	return int64(resp.Height), nil
}

// QueryEarliestAttestationNonce query the earliest attestation nonce from state machine.
func (aq *AppQuerier) QueryEarliestAttestationNonce(ctx context.Context) (int64, error) {
	queryClient := celestiatypes.NewQueryClient(aq.clientConn)
	resp, err := queryClient.EarliestAttestationNonce(ctx, &celestiatypes.QueryEarliestAttestationNonceRequest{})
	if err != nil {
		return 0, err
	}

	return int64(resp.Nonce), nil
}

// unmarshallAttestation unmarshal a wrapper protobuf `Any` type to an `AttestationRequestI`.
func (aq *AppQuerier) unmarshallAttestation(attestation *cdctypes.Any) (celestiatypes.AttestationRequestI, error) {
	var unmarshalledAttestation celestiatypes.AttestationRequestI
	err := aq.EncCfg.InterfaceRegistry.UnpackAny(attestation, &unmarshalledAttestation)
	if err != nil {
		return nil, err
	}
	return unmarshalledAttestation, nil
}
