package api

import (
	"testing"
	"time"
)

// TestExchangeStateCacheSweepExpired pins P2-17 at the production call site
// (Set → SweepExpired): stale per-user entries must be deleted, not just
// ignored on re-read.
func TestExchangeStateCacheSweepExpired(t *testing.T) {
	c := NewExchangeAccountStateCache()
	now := time.Now()

	c.Set("user-old", map[string]ExchangeAccountState{})
	c.Set("user-fresh", map[string]ExchangeAccountState{})

	// Simulate 31s of wall time: both are past the 30s TTL.
	removed := c.SweepExpired(now.Add(31 * time.Second))
	if removed != 2 {
		t.Fatalf("want 2 expired entries swept, got %d", removed)
	}
	c.mu.RLock()
	n := len(c.entries)
	c.mu.RUnlock()
	if n != 0 {
		t.Fatalf("entries not emptied after sweep: %d", n)
	}

	// A sweep just past a fresh Set removes nothing.
	c.Set("user-new", map[string]ExchangeAccountState{})
	if removed := c.SweepExpired(time.Now().Add(time.Second)); removed != 0 {
		t.Fatalf("fresh entry swept: %d", removed)
	}
	if _, ok := c.Get("user-new"); !ok {
		t.Fatalf("fresh entry must survive a sweep inside the TTL")
	}
}
