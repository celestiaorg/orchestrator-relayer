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
	QgbDHT *QgbDHT
	logger tmlog.Logger
}

func NewQuerier(qgbDht *QgbDHT, logger tmlog.Logger) *Querier {
	return &Querier{
		QgbDHT: qgbDht,
		logger: logger,
	}
}

// QueryTwoThirdsDataCommitmentConfirms queries two thirds or more of data commitment confirms from the
// P2P network. The method will not return unless it finds more than two thirds, or it times out.
// No validation is required to be done at this level because the P2P validators defined at
// `p2p/validators.go` will make sure that the values queried from the DHT are valid.
// Should return an empty slice and no error if it couldn't find enough signatures.
func (q Querier) QueryTwoThirdsDataCommitmentConfirms(
	ctx context.Context,
	timeout time.Duration,
	previousValset celestiatypes.Valset,
	nonce uint64,
) ([]types.DataCommitmentConfirm, error) {
	// create a map to easily search for power
	vals := make(map[string]celestiatypes.BridgeValidator)
	for _, val := range previousValset.Members {
		vals[val.GetEvmAddress()] = val
	}

	majThreshHold := previousValset.TwoThirdsThreshold()

	t := time.After(timeout)
	for {
		select {
		case <-ctx.Done():
			return nil, nil //nolint:nilnil
		case <-t:
			return nil, pkgerrors.Wrap(
				ErrNotEnoughDataCommitmentConfirms,
				fmt.Sprintf("failure to query for majority validator set confirms: timout %s", timeout),
			)
		default:
			confirms, err := q.QueryDataCommitmentConfirms(ctx, previousValset, nonce)
			if err != nil {
				return nil, err
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
				q.logger.Debug("found enough data commitment confirms to be relayed",
					"majThreshHold",
					majThreshHold,
					"currThreshold",
					currThreshold,
				)
				return confirms, nil
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
		}
		// TODO: make the sleep configurable
		// TODO make it as a parameter and use ticker
		time.Sleep(10 * time.Second)
	}
}

// QueryTwoThirdsValsetConfirms queries two thirds or more of valset confirms from the
// P2P network. The method will not return unless it finds more than two thirds, or it times out.
// No validation is required to be done at this level because the P2P validators defined at
// `p2p/validators.go` will make sure that the values queried from the DHT are valid.
// Should return an empty slice and no error if it couldn't find enough signatures.
// For valsets, generally the first valset, whose nonce is 1, is never signed by the network.
// Thus, the call to this method for the genesis valset will only timeout.
func (q Querier) QueryTwoThirdsValsetConfirms(
	ctx context.Context,
	timeout time.Duration,
	valset celestiatypes.Valset,
) ([]types.ValsetConfirm, error) {
	// create a map to easily search for power
	vals := make(map[string]celestiatypes.BridgeValidator)
	for _, val := range valset.Members {
		vals[val.GetEvmAddress()] = val
	}

	majThreshHold := valset.TwoThirdsThreshold()
	t := time.After(timeout)
	for {
		select {
		case <-ctx.Done():
			return nil, nil //nolint:nilnil
		// TODO: remove this extra case, and we can instead rely on the caller to pass a context with a timeout
		case <-t:
			return nil, pkgerrors.Wrap(
				ErrNotEnoughValsetConfirms,
				fmt.Sprintf("failure to query for majority validator set confirms: timout %s", timeout),
			)
		default:
			confirms, err := q.QueryValsetConfirms(ctx, valset)
			if err != nil {
				return nil, err
			}

			currThreshold := uint64(0)
			for _, valsetConfirm := range confirms {
				val, has := vals[valsetConfirm.EthAddress]
				if !has {
					q.logger.Debug(
						fmt.Sprintf(
							"valSetConfirm signer not found in stored validator set: address %s nonce %d",
							val.EvmAddress,
							valset.Nonce,
						))
					continue
				}
				currThreshold += val.Power
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
				len(confirms),
				"missing_confirms",
				len(valset.Members)-len(confirms),
			)
		}
		// TODO: make the timeout configurable
		time.Sleep(10 * time.Second)
	}
}

// QueryValsetConfirmByEVMAddress get the valset confirm having nonce `nonce`
// and signed by the orchestrator whose EVM address is `address`.
// Returns (nil, nil) if the confirm is not found.
func (q Querier) QueryValsetConfirmByEVMAddress(
	ctx context.Context,
	nonce uint64,
	address string,
) (*types.ValsetConfirm, error) {
	confirm, err := q.QgbDHT.GetValsetConfirm(
		ctx,
		GetValsetConfirmKey(nonce, address),
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
func (q Querier) QueryDataCommitmentConfirmByEVMAddress(ctx context.Context, nonce uint64, address string) (*types.DataCommitmentConfirm, error) {
	confirm, err := q.QgbDHT.GetDataCommitmentConfirm(
		ctx,
		GetDataCommitmentConfirmKey(nonce, address),
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
func (q Querier) QueryDataCommitmentConfirms(ctx context.Context, valset celestiatypes.Valset, nonce uint64) ([]types.DataCommitmentConfirm, error) {
	confirms := make([]types.DataCommitmentConfirm, 0)
	for _, member := range valset.Members {
		confirm, err := q.QgbDHT.GetDataCommitmentConfirm(
			ctx,
			GetDataCommitmentConfirmKey(nonce, member.EvmAddress),
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
// It goes over the valset members and looks if they submitted any confirms.
func (q Querier) QueryValsetConfirms(ctx context.Context, valset celestiatypes.Valset) ([]types.ValsetConfirm, error) {
	confirms := make([]types.ValsetConfirm, 0)
	for _, member := range valset.Members {
		confirm, err := q.QgbDHT.GetValsetConfirm(
			ctx,
			GetValsetConfirmKey(valset.Nonce, member.EvmAddress),
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
