package helpers

import (
	"context"
	"time"
)

// ImmediateTicker is a wrapper around time.Ticker that ticks an extra time during creation before
// starting to tick every `duration`.
// The reason for adding it is for multiple services that use the ticker, to be able to run their logic
// a single time before waiting for the `duration` to elapse.
// This allows for faster execution.
func ImmediateTicker(ctx context.Context, duration time.Duration, f func() error) error {
	ticker := time.NewTicker(duration)
	if err := f(); err != nil {
		return err
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			err := f()
			if err != nil {
				return err
			}
		}
	}
}
