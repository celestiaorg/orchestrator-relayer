package p2p

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p/core/routing"
	pkgerrors "github.com/pkg/errors"
	tmlog "github.com/tendermint/tendermint/libs/log"

	celestiatypes "github.com/celestiaorg/celestia-app/x/qgb/types"
	"github.com/celestiaorg/orchestrator-relayer/types"
)

// Querier used to query the DHT for confirms.
type Querier struct {
	BlobstreamDHT *BlobstreamDHT
	logger        tmlog.Logger
}

func NewQuerier(blobStreamDht *BlobstreamDHT, logger tmlog.Logger) *Querier {
	return &Querier{
		BlobstreamDHT: blobStreamDht,
		logger:        logger,
	}
}

// QueryTwoThirdsDataCommitmentConfirms queries two thirds or more of data commitment confirms from the
// P2P network. The method will not return unless it finds more than two thirds, or it times out.
// No validation is required to be done at this level because the P2P validators defined at
// `p2p/validators.go` will make sure that the values queried from the DHT are valid.
// The `timeout` parameter represents the amount of time to wait for confirms before returning, if their number
// is less than 2/3rds.
// The `rate` parameter represents the rate at which the requests for confirms are sent to the P2P network.
// Should return an empty slice and no error if it couldn't find enough signatures.
func (q Querier) QueryTwoThirdsDataCommitmentConfirms(
	ctx context.Context,
	timeout time.Duration,
	rate time.Duration,
	previousValset celestiatypes.Valset,
	nonce uint64,
	dataRootTupleRoot string,
) ([]types.DataCommitmentConfirm, error) {
	// create a map to easily search for power
	vals := make(map[string]celestiatypes.BridgeValidator)
	for _, val := range previousValset.Members {
		vals[val.GetEvmAddress()] = val
	}

	majThreshHold := previousValset.TwoThirdsThreshold()

	var validConfirms []types.DataCommitmentConfirm
	queryFunc := func() error {
		confirms, err := q.QueryDataCommitmentConfirms(ctx, previousValset, nonce, dataRootTupleRoot)
		if err != nil {
			return err
		}

		currThreshold := uint64(0)
		for _, dataCommitmentConfirm := range confirms {
			val, has := vals[dataCommitmentConfirm.EthAddress]
			if !has {
				q.logger.Debug(fmt.Sprintf(
					"dataCommitmentConfirm signer not found in stored validator set: address %s nonce %d",
					val.EvmAddress,
					previousValset.Nonce,
				))
				continue
			}
			currThreshold += val.Power
		}

		if currThreshold >= majThreshHold {
			q.logger.Info("found enough data commitment confirms to be relayed",
				"majThreshHold",
				majThreshHold,
				"currThreshold",
				currThreshold,
			)
			validConfirms = confirms
			return nil
		}
		q.logger.Debug(
			"found DataCommitmentConfirms",
			"total_power",
			currThreshold,
			"number_of_confirms",
			len(confirms),
			"missing_confirms",
			len(previousValset.Members)-len(confirms),
		)
		return nil
	}

	// because the ticker waits for the period to pass to return for the first time, we will execute
	// the query func here to get the confirms if they're already ready instead of waiting for the first
	// duration to elapse.
	err := queryFunc()
	if err != nil {
		return nil, err
	}
	if len(validConfirms) != 0 {
		return validConfirms, nil
	}

	t := time.After(timeout)
	ticker := time.NewTicker(rate)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-t:
			return nil, pkgerrors.Wrap(
				ErrNotEnoughDataCommitmentConfirms,
				fmt.Sprintf("failure to query for majority validator set confirms: timout %s", timeout),
			)
		case <-ticker.C:
			err := queryFunc()
			if err != nil {
				return nil, err
			}
			if len(validConfirms) != 0 {
				return validConfirms, nil
			}
		}
	}
}

// QueryTwoThirdsValsetConfirms queries two thirds or more of valset confirms from the
// P2P network. The method will not return unless it finds more than two thirds, or it times out.
// No validation is required to be done at this level because the P2P validators defined at
// `p2p/validators.go` will make sure that the values queried from the DHT are valid.
// The `timeout` parameter represents the amount of time to wait for confirms before returning, if their number
// is less than 2/3rds.
// The `rate` parameter represents the rate at which the requests for confirms are sent to the P2P network.
// Should return an empty slice and no error if it couldn't find enough signatures.
// For valsets, generally the first valset, whose nonce is 1, is never signed by the network.
// Thus, the call to this method for the genesis valset will only time out.
func (q Querier) QueryTwoThirdsValsetConfirms(
	ctx context.Context,
	timeout time.Duration,
	rate time.Duration,
	valsetNonce uint64,
	previousValset celestiatypes.Valset,
	signBytes string,
) ([]types.ValsetConfirm, error) {
	// create a map to easily search for power
	vals := make(map[string]celestiatypes.BridgeValidator)
	for _, val := range previousValset.Members {
		vals[val.GetEvmAddress()] = val
	}

	majThreshHold := previousValset.TwoThirdsThreshold()

	var validConfirms []types.ValsetConfirm
	queryFunc := func() error {
		confirms, err := q.QueryValsetConfirms(ctx, valsetNonce, previousValset, signBytes)
		if err != nil {
			return err
		}

		currThreshold := uint64(0)
		for _, valsetConfirm := range confirms {
			val, has := vals[valsetConfirm.EthAddress]
			if !has {
				q.logger.Debug(
					fmt.Sprintf(
						"valSetConfirm signer not found in stored validator set: address %s nonce %d",
						val.EvmAddress,
						previousValset.Nonce,
					))
				continue
			}
			currThreshold += val.Power
		}

		if currThreshold >= majThreshHold {
			q.logger.Info("found enough valset confirms to be relayed",
				"majThreshHold",
				majThreshHold,
				"currThreshold",
				currThreshold,
			)
			validConfirms = confirms
			return nil
		}
		q.logger.Debug(
			"found ValsetConfirms",
			"nonce",
			valsetNonce,
			"total_power",
			currThreshold,
			"number_of_confirms",
			len(confirms),
			"missing_confirms",
			len(previousValset.Members)-len(confirms),
		)
		return nil
	}

	// because the ticker waits for the period to pass to return for the first time, we will execute
	// the query func here to get the confirms if they're already ready instead of waiting for the first
	// duration to elapse.
	err := queryFunc()
	if err != nil {
		return nil, err
	}
	if len(validConfirms) != 0 {
		return validConfirms, nil
	}

	t := time.After(timeout)
	ticker := time.NewTicker(rate)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		// TODO: remove this extra case, and we can instead rely on the caller to pass a context with a timeout
		case <-t:
			return nil, pkgerrors.Wrap(
				ErrNotEnoughValsetConfirms,
				fmt.Sprintf("failure to query for majority validator set confirms: timout %s", timeout),
			)
		case <-ticker.C:
			err := queryFunc()
			if err != nil {
				return nil, err
			}
			if len(validConfirms) != 0 {
				return validConfirms, nil
			}
		}
	}
}

// QueryValsetConfirmByEVMAddress get the valset confirm having nonce `nonce`
// and signed by the orchestrator whose EVM address is `address`.
// Returns (nil, nil) if the confirm is not found.
func (q Querier) QueryValsetConfirmByEVMAddress(
	ctx context.Context,
	nonce uint64,
	address string,
	signBytes string,
) (*types.ValsetConfirm, error) {
	confirm, err := q.BlobstreamDHT.GetValsetConfirm(
		ctx,
		GetValsetConfirmKey(nonce, address, signBytes),
	)
	if err != nil {
		if errors.Is(err, routing.ErrNotFound) {
			return nil, nil
		}
		return nil, err

	}
	return &confirm, nil
}

// QueryDataCommitmentConfirmByEVMAddress get the data commitment confirm having nonce `nonce`
// and signed by the orchestrator whose EVM address is `address`.
// Returns (nil, nil) if the confirm is not found
func (q Querier) QueryDataCommitmentConfirmByEVMAddress(ctx context.Context, nonce uint64, address string, dataRootTupleRoot string) (*types.DataCommitmentConfirm, error) {
	confirm, err := q.BlobstreamDHT.GetDataCommitmentConfirm(
		ctx,
		GetDataCommitmentConfirmKey(nonce, address, dataRootTupleRoot),
	)
	if err != nil {
		if errors.Is(err, routing.ErrNotFound) {
			return nil, nil
		}
		return nil, err

	}
	return &confirm, nil
}

// QueryDataCommitmentConfirms get all the data commitment confirms in store for a certain nonce.
// It goes over the valset members and looks if they submitted any confirms.
func (q Querier) QueryDataCommitmentConfirms(ctx context.Context, valset celestiatypes.Valset, nonce uint64, dataRootTupleRoot string) ([]types.DataCommitmentConfirm, error) {
	confirms := make([]types.DataCommitmentConfirm, 0)
	for _, member := range valset.Members {
		confirm, err := q.BlobstreamDHT.GetDataCommitmentConfirm(
			ctx,
			GetDataCommitmentConfirmKey(nonce, member.EvmAddress, dataRootTupleRoot),
		)
		if err == nil {
			confirms = append(confirms, confirm)
		} else if errors.Is(err, routing.ErrNotFound) {
			continue
		} else {
			return nil, err
		}
	}
	return confirms, nil
}

// QueryValsetConfirms get all the valset confirms in store for a certain nonce.
// It goes over the specified valset members and looks if they submitted any confirms
// for the provided nonce.
func (q Querier) QueryValsetConfirms(ctx context.Context, nonce uint64, valset celestiatypes.Valset, signBytes string) ([]types.ValsetConfirm, error) {
	confirms := make([]types.ValsetConfirm, 0)
	for _, member := range valset.Members {
		confirm, err := q.BlobstreamDHT.GetValsetConfirm(
			ctx,
			GetValsetConfirmKey(nonce, member.EvmAddress, signBytes),
		)
		if err == nil {
			confirms = append(confirms, confirm)
		} else if errors.Is(err, routing.ErrNotFound) {
			continue
		} else {
			return nil, err
		}
	}
	return confirms, nil
}

// QueryLatestValset get the latest valset from the p2p network.
func (q Querier) QueryLatestValset(
	ctx context.Context,
) (*types.LatestValset, error) {
	latestValset, err := q.BlobstreamDHT.GetLatestValset(ctx)
	if err != nil {
		return nil, err
	}
	return &latestValset, nil
}
