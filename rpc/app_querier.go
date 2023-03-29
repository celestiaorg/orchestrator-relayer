package rpc

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/celestiaorg/orchestrator-relayer/types"

	"github.com/celestiaorg/celestia-app/app/encoding"
	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	cdctypes "github.com/cosmos/cosmos-sdk/codec/types"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

// AppQuerier queries the application for attestations and unbonding periods.
type AppQuerier struct {
	qgbRPC     string
	clientConn *grpc.ClientConn
	Logger     tmlog.Logger
	EncCfg     encoding.Config
}

func NewAppQuerier(logger tmlog.Logger, qgbRPC string, encCft encoding.Config) *AppQuerier {
	return &AppQuerier{Logger: logger, qgbRPC: qgbRPC, EncCfg: encCft}
}

func (aq *AppQuerier) Start() error {
	// creating a grpc connection to Celestia-app
	qgbGRPC, err := grpc.Dial(aq.qgbRPC, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	aq.clientConn = qgbGRPC
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

// QueryDataCommitmentByNonce query a data commitment by its nonce.
func (aq *AppQuerier) QueryDataCommitmentByNonce(ctx context.Context, nonce uint64) (*celestiatypes.DataCommitment, error) {
	attestation, err := aq.QueryAttestationByNonce(ctx, nonce)
	if err != nil {
		return nil, err
	}
	if attestation == nil {
		return nil, types.ErrAttestationNotFound
	}

	if attestation.Type() != celestiatypes.DataCommitmentRequestType {
		return nil, types.ErrAttestationNotDataCommitmentRequest
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
	dcc, err := aq.QueryDataCommitmentByNonce(ctx, resp.Nonce)
	if err != nil {
		return nil, err
	}
	return dcc, nil
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

	if attestation.Type() != celestiatypes.ValsetRequestType {
		return nil, types.ErrAttestationNotValsetRequest
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

// QueryLastValsetBeforeNonce returns the last valset before nonce.
// This will be needed when signing to know the validator set at that particular nonce.
// the provided `nonce` can be a valset, but this will return the valset before it.
// If nonce is 1, it will return an error. Because, there is no valset before nonce 1.
func (aq *AppQuerier) QueryLastValsetBeforeNonce(ctx context.Context, nonce uint64) (*celestiatypes.Valset, error) {
	queryClient := celestiatypes.NewQueryClient(aq.clientConn)
	resp, err := queryClient.LastValsetRequestBeforeNonce(
		ctx,
		&celestiatypes.QueryLastValsetRequestBeforeNonceRequest{Nonce: nonce},
	)
	if err != nil {
		return nil, err
	}

	return resp.Valset, nil
}

// QueryLastUnbondingHeight query the last unbonding height from state machine.
func (aq *AppQuerier) QueryLastUnbondingHeight(ctx context.Context) (int64, error) {
	queryClient := celestiatypes.NewQueryClient(aq.clientConn)
	resp, err := queryClient.LastUnbondingHeight(ctx, &celestiatypes.QueryLastUnbondingHeightRequest{})
	if err != nil {
		return 0, err
	}

	return int64(resp.Height), nil
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
