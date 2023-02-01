package p2p

import (
	"context"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/types"

	ds "github.com/ipfs/go-datastore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
)

const (
	ProtocolPrefix                 = "/qgb/0.1.0" // TODO "/qgb/<version>" ?
	DataCommitmentConfirmNamespace = "dcc"
	ValsetConfirmNamespace         = "vc"
)

// QgbDHT wrapper around the `IpfsDHT` implementation.
// Used to add helper methods to easily handle the DHT.
type QgbDHT struct {
	*dht.IpfsDHT
}

// NewQgbDHT create a new IPFS DHT using a suitable configuration for the QGB.
func NewQgbDHT(ctx context.Context, h host.Host, store ds.Batching) (*QgbDHT, error) {
	router, err := dht.New(
		ctx,
		h,
		dht.Datastore(store),
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix(ProtocolPrefix),
		dht.RoutingTableRefreshPeriod(time.Nanosecond), // TODO investigate which values to use
		dht.NamespacedValidator(DataCommitmentConfirmNamespace, DataCommitmentConfirmValidator{}),
		dht.NamespacedValidator(ValsetConfirmNamespace, ValsetConfirmValidator{}),
	)
	if err != nil {
		return nil, err
	}

	return &QgbDHT{router}, nil
}

// Note: The Get and Put methods do not run any validations on the data commitment confirms
// and valset confirms. The checks are supposed to be handled by the validators under `p2p/validators.go`.
// Same goes for the Marshal and Unmarshal methods (as long as they're using simple Json encoding).

// PutDataCommitmentConfirm encodes a data commitment confirm then puts its value to the DHT.
// The key can be generated using the `GetDataCommitmentConfirmKey` method.
// Returns an error if it fails to do so.
func (q QgbDHT) PutDataCommitmentConfirm(ctx context.Context, key string, dcc types.DataCommitmentConfirm) error {
	encodedData, err := types.MarshalDataCommitmentConfirm(dcc)
	if err != nil {
		return err
	}
	err = q.PutValue(ctx, key, encodedData)
	if err != nil {
		return err
	}
	return nil
}

// GetDataCommitmentConfirm looks for a data commitment confirm referenced by its key in the DHT.
// The key can be generated using the `GetDataCommitmentConfirmKey` method.
// Returns an error if it fails to get the confirm.
func (q QgbDHT) GetDataCommitmentConfirm(ctx context.Context, key string) (types.DataCommitmentConfirm, error) {
	encodedConfirm, err := q.GetValue(ctx, key) // this is a blocking call, we should probably use timeout and channel
	if err != nil {
		return types.DataCommitmentConfirm{}, err
	}
	confirm, err := types.UnmarshalDataCommitmentConfirm(encodedConfirm)
	if err != nil {
		return types.DataCommitmentConfirm{}, err
	}
	return confirm, nil
}

// PutValsetConfirm encodes a valset confirm then puts its value to the DHT.
// The key can be generated using the `GetValsetConfirmKey` method.
// Returns an error if it fails to do so.
func (q QgbDHT) PutValsetConfirm(ctx context.Context, key string, vc types.ValsetConfirm) error {
	encodedData, err := types.MarshalValsetConfirm(vc)
	if err != nil {
		return err
	}
	err = q.PutValue(ctx, key, encodedData)
	if err != nil {
		return err
	}
	return nil
}

// GetValsetConfirm looks for a valset confirm referenced by its key in the DHT.
// The key can be generated using the `GetValsetConfirmKey` method.
// Returns an error if it fails to get the confirm.
func (q QgbDHT) GetValsetConfirm(ctx context.Context, key string) (types.ValsetConfirm, error) {
	encodedConfirm, err := q.GetValue(ctx, key) // this is a blocking call, we should probably use timeout and channel
	if err != nil {
		return types.ValsetConfirm{}, err
	}
	confirm, err := types.UnmarshalValsetConfirm(encodedConfirm)
	if err != nil {
		return types.ValsetConfirm{}, err
	}
	return confirm, nil
}
