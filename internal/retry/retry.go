package retry

import (
	"context"
	"fmt"
	"time"
)

// RetryWithBackoff invokes fn up to maxAttempts times with exponential backoff.
// Initial delay 200ms, doubles each attempt, capped at 5s.
// Respects ctx.Done() between attempts.
func RetryWithBackoff(ctx context.Context, maxAttempts int, fn func() error) error {
	if maxAttempts < 1 {
		return fmt.Errorf("retry: maxAttempts must be >= 1, got %d", maxAttempts)
	}
	var lastErr error
	delay := 200 * time.Millisecond
	for attempt := 0; attempt < maxAttempts; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}
		lastErr = err
		if attempt == maxAttempts-1 {
			break // no sleep after last attempt
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		delay *= 2
		if delay > 5*time.Second {
			delay = 5 * time.Second
		}
	}
	return fmt.Errorf("retry: max attempts (%d) exceeded: %w", maxAttempts, lastErr)
}
