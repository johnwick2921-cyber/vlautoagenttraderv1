package retry

import (
	"fmt"
	"sync"
	"time"
)

// CircuitBreaker implements a simple 3-state breaker (closed / open / half-open).
//
// closed   — all calls allowed; failures counted
// open     — all calls rejected immediately until cooldown elapses
// half-open — single probe call allowed after cooldown; success closes,
//
//	failure re-opens
type CircuitBreaker struct {
	failureThreshold int
	cooldown         time.Duration

	mu       sync.Mutex
	failures int
	openedAt time.Time // zero when closed; non-zero when open
}

// NewCircuitBreaker creates a breaker that opens after `threshold` consecutive
// failures and stays open for `cooldown` before allowing a single probe.
func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: threshold,
		cooldown:         cooldown,
	}
}

// Allow reports whether the next call may proceed.
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	if cb.openedAt.IsZero() {
		return true // closed
	}
	if time.Since(cb.openedAt) >= cb.cooldown {
		// half-open: allow single probe
		return true
	}
	return false
}

// RecordSuccess clears the failure count and closes the breaker.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures = 0
	cb.openedAt = time.Time{}
}

// RecordFailure increments the failure count and opens the breaker once
// the threshold is hit.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.failures++
	if cb.failures >= cb.failureThreshold {
		cb.openedAt = time.Now()
	}
}

// ErrCircuitOpen is returned by callers that consult Allow() and find the
// breaker in the rejection state.
var ErrCircuitOpen = fmt.Errorf("circuit breaker open")
