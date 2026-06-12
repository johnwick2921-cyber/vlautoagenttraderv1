package retry

import (
	"sync"
	"testing"
	"time"
)

func TestCircuitBreaker_FreshAllow(t *testing.T) {
	cb := NewCircuitBreaker(5, 1*time.Second)
	if !cb.Allow() {
		t.Fatalf("expected fresh breaker to Allow")
	}
}

func TestCircuitBreaker_OpensAfterThreshold(t *testing.T) {
	cb := NewCircuitBreaker(5, 1*time.Second)
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	if cb.Allow() {
		t.Fatalf("expected breaker to be open after 5 failures")
	}
}

func TestCircuitBreaker_SuccessResetsCounter(t *testing.T) {
	cb := NewCircuitBreaker(5, 1*time.Second)
	for i := 0; i < 4; i++ {
		cb.RecordFailure()
	}
	cb.RecordSuccess()
	// Now record 4 more failures — total of 4 since reset, still below threshold.
	for i := 0; i < 4; i++ {
		cb.RecordFailure()
	}
	if !cb.Allow() {
		t.Fatalf("expected Allow after success reset (4 fails post-reset)")
	}
}

func TestCircuitBreaker_HalfOpenAfterCooldown(t *testing.T) {
	cb := NewCircuitBreaker(5, 50*time.Millisecond)
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	if cb.Allow() {
		t.Fatalf("expected closed-to-open transition to reject")
	}
	time.Sleep(75 * time.Millisecond)
	if !cb.Allow() {
		t.Fatalf("expected half-open probe after cooldown")
	}
}

func TestCircuitBreaker_HalfOpenProbeSuccessCloses(t *testing.T) {
	cb := NewCircuitBreaker(5, 50*time.Millisecond)
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	time.Sleep(75 * time.Millisecond)
	// Probe succeeds.
	cb.RecordSuccess()
	if !cb.Allow() {
		t.Fatalf("expected breaker closed after probe success")
	}
	// Verify it stays closed under more failures up to threshold-1.
	for i := 0; i < 4; i++ {
		cb.RecordFailure()
	}
	if !cb.Allow() {
		t.Fatalf("expected closed at 4 < 5 post-reset failures")
	}
}

func TestCircuitBreaker_HalfOpenProbeFailureReopens(t *testing.T) {
	cb := NewCircuitBreaker(5, 50*time.Millisecond)
	for i := 0; i < 5; i++ {
		cb.RecordFailure()
	}
	time.Sleep(75 * time.Millisecond)
	// Probe fails — should re-open with current implementation
	// (RecordFailure increments past threshold and updates openedAt).
	cb.RecordFailure()
	if cb.Allow() {
		t.Fatalf("expected breaker re-opened after probe failure")
	}
}

func TestCircuitBreaker_ConcurrentFailures(t *testing.T) {
	cb := NewCircuitBreaker(50, 1*time.Second)
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cb.RecordFailure()
		}()
	}
	wg.Wait()
	// Sanity check — Allow() must not panic and breaker should be open
	// (100 failures >> threshold of 50).
	if cb.Allow() {
		t.Fatalf("expected breaker open after 100 concurrent failures")
	}
}
