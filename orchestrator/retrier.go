package orchestrator

import (
	"context"
	"time"

	tmlog "github.com/tendermint/tendermint/libs/log"
)

// Retrier handles orchestrator retries of failed nonces.
type Retrier struct {
	logger        tmlog.Logger
	retriesNumber int
	delay         time.Duration
}

// DefaultRetrierDelay default retrier delay
const DefaultRetrierDelay = 10 * time.Second

func NewRetrier(logger tmlog.Logger, retriesNumber int, delay time.Duration) *Retrier {
	return &Retrier{
		logger:        logger,
		retriesNumber: retriesNumber,
		delay:         delay,
	}
}

func (r Retrier) Retry(ctx context.Context, nonce uint64, retryMethod func(context.Context, uint64) error) error {
	var err error
	for i := 0; i < r.retriesNumber; i++ {
		// We can implement some exponential backoff in here
		select {
		case <-ctx.Done():
			return nil
		default:
			time.Sleep(r.delay)
			r.logger.Info("retrying", "nonce", nonce, "retry_number", i, "retries_left", r.retriesNumber-i)
			err = retryMethod(ctx, nonce)
			if err == nil {
				r.logger.Info("nonce processing succeeded", "nonce", nonce, "retries_number", i)
				return nil
			}
			r.logger.Error("failed to process nonce", "nonce", nonce, "retry", i, "err", err)
		}
	}
	return err
}

func (r Retrier) RetryThenFail(ctx context.Context, nonce uint64, retryMethod func(context.Context, uint64) error) {
	err := r.Retry(ctx, nonce, retryMethod)
	if err != nil {
		panic(err)
	}
}
