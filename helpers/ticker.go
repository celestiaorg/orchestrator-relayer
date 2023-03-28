package helpers

import (
	"context"
	"time"
)

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
