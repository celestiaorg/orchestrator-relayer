package helpers

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

func (r Retrier) Retry(ctx context.Context, retryMethod func() error) error {
	var err error
	for i := 0; i < r.retriesNumber; i++ {
		// We can implement some exponential backoff in here
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			time.Sleep(r.delay)
			r.logger.Info("retrying", "retry_number", i, "retries_left", r.retriesNumber-i)
			err = retryMethod()
			if err == nil {
				r.logger.Info("succeeded", "retries_number", i)
				return nil
			}
			r.logger.Error("failed to process", "retry", i, "err", err)
		}
	}
	return err
}

func (r Retrier) RetryThenFail(ctx context.Context, retryMethod func() error) {
	err := r.Retry(ctx, retryMethod)
	if err != nil {
		panic(err)
	}
}
