package retry

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestRetryWithBackoff_SuccessFirstAttempt(t *testing.T) {
	calls := 0
	err := RetryWithBackoff(context.Background(), 3, func() error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestRetryWithBackoff_SuccessSecondAttempt(t *testing.T) {
	calls := 0
	err := RetryWithBackoff(context.Background(), 3, func() error {
		calls++
		if calls == 1 {
			return errors.New("transient")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected nil err, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestRetryWithBackoff_AllAttemptsFail(t *testing.T) {
	calls := 0
	sentinel := errors.New("boom")
	err := RetryWithBackoff(context.Background(), 3, func() error {
		calls++
		return sentinel
	})
	if err == nil {
		t.Fatalf("expected err, got nil")
	}
	if calls != 3 {
		t.Fatalf("expected 3 calls, got %d", calls)
	}
	if !strings.Contains(err.Error(), "max attempts") {
		t.Fatalf("expected 'max attempts' in err, got %v", err)
	}
	if !errors.Is(err, sentinel) {
		t.Fatalf("expected wrapped sentinel, got %v", err)
	}
}

func TestRetryWithBackoff_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()
	err := RetryWithBackoff(ctx, 5, func() error {
		calls++
		return errors.New("fail")
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if calls < 1 {
		t.Fatalf("expected at least 1 call, got %d", calls)
	}
}

func TestRetryWithBackoff_ZeroAttempts(t *testing.T) {
	err := RetryWithBackoff(context.Background(), 0, func() error {
		return nil
	})
	if err == nil {
		t.Fatalf("expected err for maxAttempts=0, got nil")
	}
	if !strings.Contains(err.Error(), "maxAttempts must be >= 1") {
		t.Fatalf("expected validation message, got %v", err)
	}
}

func TestRetryWithBackoff_BackoffTiming(t *testing.T) {
	var timestamps []time.Time
	start := time.Now()
	err := RetryWithBackoff(context.Background(), 3, func() error {
		timestamps = append(timestamps, time.Now())
		return fmt.Errorf("always fail")
	})
	if err == nil {
		t.Fatalf("expected err, got nil")
	}
	if len(timestamps) != 3 {
		t.Fatalf("expected 3 attempts, got %d", len(timestamps))
	}
	gap1 := timestamps[1].Sub(timestamps[0])
	gap2 := timestamps[2].Sub(timestamps[1])
	// First gap should be ~200ms (tolerate 150-350ms).
	if gap1 < 150*time.Millisecond || gap1 > 350*time.Millisecond {
		t.Fatalf("expected gap1 ~200ms, got %v", gap1)
	}
	// Second gap should be ~400ms (tolerate 350-550ms).
	if gap2 < 350*time.Millisecond || gap2 > 550*time.Millisecond {
		t.Fatalf("expected gap2 ~400ms, got %v", gap2)
	}
	total := time.Since(start)
	if total < 500*time.Millisecond {
		t.Fatalf("expected total >= 500ms, got %v", total)
	}
}
