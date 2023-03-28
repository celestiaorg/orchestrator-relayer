package helpers

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestImmediateTicker(t *testing.T) {
	count := 0
	tests := []struct {
		name             string
		f                func() error
		expectedErrCount int
	}{
		{
			name: "error at the first execution",
			f: func() error {
				count++
				return errors.New("test error")
			},
			expectedErrCount: 1,
		},
		{
			name: "error at the second execution",
			f: func() error {
				count++
				if count == 2 {
					return errors.New("test error")
				}
				return nil
			},
			expectedErrCount: 2,
		},
		{
			name: "error at the third execution",
			f: func() error {
				count++
				if count == 3 {
					return errors.New("test error")
				}
				return nil
			},
			expectedErrCount: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count = 0
			err := ImmediateTicker(context.Background(), time.Millisecond, tt.f)
			assert.Error(t, err)
			assert.Equal(t, tt.expectedErrCount, count)
		})
	}
}
