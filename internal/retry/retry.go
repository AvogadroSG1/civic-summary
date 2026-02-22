// Package retry provides generic retry logic with configurable exponential backoff.
package retry

import (
	"context"
	"fmt"
	"log/slog"
	"time"
)

// Config holds retry configuration.
type Config struct {
	MaxRetries    int
	BackoffDelays []time.Duration
}

// NewConfig creates a retry config from max retries and delay seconds.
func NewConfig(maxRetries int, delaySeconds []int) Config {
	delays := make([]time.Duration, len(delaySeconds))
	for i, s := range delaySeconds {
		delays[i] = time.Duration(s) * time.Second
	}
	return Config{
		MaxRetries:    maxRetries,
		BackoffDelays: delays,
	}
}

// Do executes fn with retry logic. Returns the last error if all retries fail.
func Do(ctx context.Context, cfg Config, operation string, fn func() error) error {
	var lastErr error
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := backoffDelay(cfg, attempt-1)
			slog.Info("retrying operation",
				"operation", operation,
				"attempt", attempt+1,
				"delay", delay,
			)
			select {
			case <-ctx.Done():
				return fmt.Errorf("retry cancelled: %w", ctx.Err())
			case <-time.After(delay):
			}
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		slog.Warn("operation failed",
			"operation", operation,
			"attempt", attempt+1,
			"error", lastErr,
		)
	}
	return fmt.Errorf("all %d retries exhausted for %s: %w", cfg.MaxRetries+1, operation, lastErr)
}

// backoffDelay returns the delay for a given retry index.
// If the index exceeds configured delays, uses the last configured delay.
func backoffDelay(cfg Config, index int) time.Duration {
	if len(cfg.BackoffDelays) == 0 {
		return 5 * time.Second
	}
	if index >= len(cfg.BackoffDelays) {
		return cfg.BackoffDelays[len(cfg.BackoffDelays)-1]
	}
	return cfg.BackoffDelays[index]
}
