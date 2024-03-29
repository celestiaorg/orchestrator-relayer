package helpers_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/celestiaorg/orchestrator-relayer/helpers"

	"github.com/stretchr/testify/assert"
	tmlog "github.com/tendermint/tendermint/libs/log"
)

func TestRetry(t *testing.T) {
	ret := helpers.NewRetrier(tmlog.NewNopLogger(), 10, time.Millisecond)
	var count int
	tests := []struct {
		name          string
		f             func() error
		expectedCount int
		wantErr       bool
	}{
		{
			name: "always error",
			f: func() error {
				count++
				return errors.New("test error")
			},
			expectedCount: 10,
			wantErr:       true,
		},
		{
			name: "never error",
			f: func() error {
				count++
				return nil
			},
			expectedCount: 1,
			wantErr:       false,
		},
		{
			name: "error in the middle",
			f: func() error {
				count++
				if count == 5 {
					return nil
				}
				return errors.New("test error")
			},
			expectedCount: 5,
			wantErr:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count = 0
			err := ret.Retry(context.Background(), func() error {
				return tt.f()
			})
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedCount, count)
		})
	}
}

func TestRetryThenFail(t *testing.T) {
	ret := helpers.NewRetrier(tmlog.NewNopLogger(), 10, time.Millisecond)
	var count int
	tests := []struct {
		name          string
		f             func() error
		expectedCount int
		wantPanic     bool
	}{
		{
			name: "panic at the end",
			f: func() error {
				count++
				return errors.New("test error")
			},
			expectedCount: 10,
			wantPanic:     true,
		},
		{
			name: "never panic",
			f: func() error {
				count++
				return nil
			},
			expectedCount: 1,
			wantPanic:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count = 0
			if tt.wantPanic {
				assert.Panics(t, func() {
					ret.RetryThenFail(context.Background(), func() error {
						return tt.f()
					})
				})
			} else {
				assert.NotPanics(t, func() {
					ret.RetryThenFail(context.Background(), func() error {
						return tt.f()
					})
				})
			}
			assert.Equal(t, tt.expectedCount, count)
		})
	}
}
