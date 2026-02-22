package retry_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/AvogadroSG1/civic-summary/internal/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDo_SucceedsFirstTry(t *testing.T) {
	cfg := retry.NewConfig(3, []int{1, 2, 3})
	calls := 0

	err := retry.Do(context.Background(), cfg, "test", func() error {
		calls++
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

func TestDo_SucceedsAfterRetries(t *testing.T) {
	cfg := retry.Config{
		MaxRetries:    3,
		BackoffDelays: []time.Duration{1 * time.Millisecond, 2 * time.Millisecond},
	}
	calls := 0

	err := retry.Do(context.Background(), cfg, "test", func() error {
		calls++
		if calls < 3 {
			return fmt.Errorf("transient error %d", calls)
		}
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 3, calls)
}

func TestDo_ExhaustsRetries(t *testing.T) {
	cfg := retry.Config{
		MaxRetries:    2,
		BackoffDelays: []time.Duration{1 * time.Millisecond},
	}
	calls := 0

	err := retry.Do(context.Background(), cfg, "test", func() error {
		calls++
		return fmt.Errorf("persistent error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "retries exhausted")
	assert.Contains(t, err.Error(), "persistent error")
	assert.Equal(t, 3, calls) // Initial + 2 retries
}

func TestDo_RespectsContext(t *testing.T) {
	cfg := retry.Config{
		MaxRetries:    10,
		BackoffDelays: []time.Duration{100 * time.Millisecond},
	}

	ctx, cancel := context.WithCancel(context.Background())
	calls := 0

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := retry.Do(ctx, cfg, "test", func() error {
		calls++
		return fmt.Errorf("error")
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cancelled")
}

func TestNewConfig(t *testing.T) {
	cfg := retry.NewConfig(3, []int{5, 20, 60})
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Len(t, cfg.BackoffDelays, 3)
	assert.Equal(t, 5*time.Second, cfg.BackoffDelays[0])
	assert.Equal(t, 20*time.Second, cfg.BackoffDelays[1])
	assert.Equal(t, 60*time.Second, cfg.BackoffDelays[2])
}
