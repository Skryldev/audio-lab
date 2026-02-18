package retry

import (
	"context"
	"time"
)

// Config holds retry configuration
type Config struct {
	MaxAttempts int
	Delay       time.Duration
	Multiplier  float64
	MaxDelay    time.Duration
}

// DefaultConfig returns sensible retry defaults
func DefaultConfig() Config {
	return Config{
		MaxAttempts: 3,
		Delay:       time.Second,
		Multiplier:  2.0,
		MaxDelay:    30 * time.Second,
	}
}

// Do executes fn with exponential backoff retry
func Do(ctx context.Context, cfg Config, fn func() error) error {
	var lastErr error
	delay := cfg.Delay

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lastErr = fn()
		if lastErr == nil {
			return nil
		}

		if attempt == cfg.MaxAttempts-1 {
			break
		}

		// Apply exponential backoff
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}

		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return lastErr
}