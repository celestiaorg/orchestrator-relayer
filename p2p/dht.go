package p2p

import (
	"context"
	"time"

	types2 "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/libp2p/go-libp2p-kad-dht/providers"

	"github.com/celestiaorg/orchestrator-relayer/types"
	ds "github.com/ipfs/go-datastore"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

const (
	ProtocolPrefix                 = "/blobstream/0.1.0" // TODO "/blobstream/<version>" ?
	DataCommitmentConfirmNamespace = "dcc"
	ValsetConfirmNamespace         = "vc"
	LatestValsetNamespace          = "lv"
)

// BlobstreamDHT wrapper around the `IpfsDHT` implementation.
// Used to add helper methods to easily handle the DHT.
type BlobstreamDHT struct {
	*dht.IpfsDHT
	logger tmlog.Logger
}

// NewBlobstreamDHT create a new IPFS DHT using a suitable configuration for the Blobstream.
// If nil is passed for bootstrappers, the DHT will not try to connect to any existing peer.
func NewBlobstreamDHT(ctx context.Context, h host.Host, store ds.Batching, bootstrappers []peer.AddrInfo, logger tmlog.Logger) (*BlobstreamDHT, error) {
	// this values is set to a year, so that even in super-stable networks, we have at least
	// one valset in store for a year.
	providers.ProvideValidity = time.Hour * 24 * 365

	router, err := dht.New(
		ctx,
		h,
		dht.Datastore(store),
		dht.Mode(dht.ModeServer),
		dht.ProtocolPrefix(ProtocolPrefix),
		dht.NamespacedValidator(DataCommitmentConfirmNamespace, DataCommitmentConfirmValidator{}),
		dht.NamespacedValidator(ValsetConfirmNamespace, ValsetConfirmValidator{}),
		dht.NamespacedValidator(LatestValsetNamespace, LatestValsetValidator{}),
		dht.BootstrapPeers(bootstrappers...),
		dht.DisableProviders(),
	)
	if err != nil {
		return nil, err
	}

	return &BlobstreamDHT{
		IpfsDHT: router,
		logger:  logger,
	}, nil
}

// WaitForPeers waits for peers to be connected to the DHT.
// Returns nil if the context is done or the peers list has more peers than the specified peersThreshold.
// Returns error if it times out.
func (q BlobstreamDHT) WaitForPeers(ctx context.Context, timeout time.Duration, rate time.Duration, peersThreshold int) error {
	if peersThreshold < 1 {
		return ErrPeersThresholdCannotBeNegative
	}

	// checking before entering the for loop to avoid waiting for the initial ticker duration.
	peersLen := len(q.RoutingTable().ListPeers())
	if peersLen >= peersThreshold {
		q.logger.Info("found peers", "peers count", peersLen)
		return nil
	}

	t := time.After(timeout)
	ticker := time.NewTicker(rate)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t:
			return ErrPeersTimeout
		case <-ticker.C:
			peersLen := len(q.RoutingTable().ListPeers())
			if peersLen >= peersThreshold {
				q.logger.Info("found peers", "peers count", peersLen)
				return nil
			}
			q.logger.Info(
				"waiting for routing table to populate",
				"target number of peers",
				peersThreshold,
				"current count",
				peersLen,
			)
		}
	}
}

// Note: The Get and Put methods do not run any validations on the data commitment confirms
// and valset confirms. The checks are supposed to be handled by the validators under `p2p/validators.go`.
// Same goes for the Marshal and Unmarshal methods (as long as they're using simple Json encoding).

// PutDataCommitmentConfirm encodes a data commitment confirm then puts its value to the DHT.
// The key can be generated using the `GetDataCommitmentConfirmKey` method.
// Returns an error if it fails to do so.
func (q BlobstreamDHT) PutDataCommitmentConfirm(ctx context.Context, key string, dcc types.DataCommitmentConfirm) error {
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
func (q BlobstreamDHT) GetDataCommitmentConfirm(ctx context.Context, key string) (types.DataCommitmentConfirm, error) {
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
func (q BlobstreamDHT) PutValsetConfirm(ctx context.Context, key string, vc types.ValsetConfirm) error {
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
func (q BlobstreamDHT) GetValsetConfirm(ctx context.Context, key string) (types.ValsetConfirm, error) {
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

// PutLatestValset encodes a valset then puts its value to the DHT.
// The key will be returned by the `GetValsetKey` method.
// If the valset is not the latest, it will fail.
// Returns an error if it fails.
func (q BlobstreamDHT) PutLatestValset(ctx context.Context, v types2.Valset) error {
	encodedData, err := types.MarshalValset(v)
	if err != nil {
		return err
	}
	err = q.PutValue(ctx, GetLatestValsetKey(), encodedData)
	if err != nil {
		return err
	}
	return nil
}

// GetLatestValset looks for a valset referenced by its key in the DHT.
// The key will be returned by the `GetValsetKey` method.
// Returns an error if it fails to get the valset.
func (q BlobstreamDHT) GetLatestValset(ctx context.Context) (types2.Valset, error) {
	encoded, err := q.GetValue(ctx, GetLatestValsetKey()) // this is a blocking call, we should probably use timeout and channel
	if err != nil {
		return types2.Valset{}, err
	}
	valset, err := types.UnmarshalValset(encoded)
	if err != nil {
		return types2.Valset{}, err
	}
	return valset, nil
}
