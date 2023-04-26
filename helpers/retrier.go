package helpers

import (
	"context"
	"time"

	tmlog "github.com/tendermint/tendermint/libs/log"
)

// Retrier handles retries of failed services.
type Retrier struct {
	logger        tmlog.Logger
	retriesNumber int
	baseDelay     time.Duration
}

// DefaultRetrierDelay default retrier baseDelay
const DefaultRetrierDelay = 10 * time.Second

func NewRetrier(logger tmlog.Logger, retriesNumber int, baseDelay time.Duration) *Retrier {
	return &Retrier{
		logger:        logger,
		retriesNumber: retriesNumber,
		baseDelay:     baseDelay,
	}
}

// Retry retries the `retryMethod` for `r.retriesNumber` times, separated by an exponential delay
// calculated using the `NextTick(retryCount)` method.
// Returns the final execution error if all retries failed.
func (r Retrier) Retry(ctx context.Context, retryMethod func() error) error {
	r.logger.Info("trying to recover from error...")
	var err error
	for i := 0; i < r.retriesNumber; i++ {
		nextTick := time.NewTimer(r.NextTick(i))
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-nextTick.C:
			r.logger.Info("retrying", "retry_number", i, "retries_left", r.retriesNumber-i)
			err = retryMethod()
			if err == nil {
				r.logger.Info("succeeded", "retries_number", i)
				return nil
			}
			r.logger.Error("failed attempt", "retry", i, "err", err)
		}
	}
	return err
}

// RetryThenFail similar to `Retry` but panics upon failure.
func (r Retrier) RetryThenFail(ctx context.Context, retryMethod func() error) {
	err := r.Retry(ctx, retryMethod)
	if err != nil {
		panic(err)
	}
}

// NextTick calculates the next exponential tick based on the provided retry count
// and the initialized base delay.
func (r Retrier) NextTick(retryCount int) time.Duration {
	return 1 << retryCount * r.baseDelay
}
