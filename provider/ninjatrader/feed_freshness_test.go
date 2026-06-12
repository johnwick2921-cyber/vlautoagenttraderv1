package ninjatrader

import (
	"testing"
	"time"
)

// TestIsFeedConnected_FreshBarOverridesStaleStatus locks the stale-status fix:
// IsFeedConnected default-allows before any status, honors "Connected", returns
// false on a non-"Connected" status with no recent bars, but OVERRIDES a stale
// "Disconnected" when a live bar_update arrived within feedFreshWindow (the bar
// stream is the real proof the feed is up; feed_status is edge-triggered and can
// latch on a momentary flap).
func TestIsFeedConnected_FreshBarOverridesStaleStatus(t *testing.T) {
	s := &TCPServer{}

	// 1. No status yet → DEFAULT-ALLOW (never false-halt at startup).
	if !s.IsFeedConnected() {
		t.Fatal("empty feed status must default-allow (true)")
	}
	// 2. Explicit "Connected" → true.
	s.feedStatus = "Connected"
	if !s.IsFeedConnected() {
		t.Fatal("Connected status must be true")
	}
	// 3. "Disconnected" + no bars → false (the legacy behavior, preserved).
	s.feedStatus = "Disconnected"
	if s.IsFeedConnected() {
		t.Fatal("Disconnected with no recent bars must be false")
	}
	// 4. "Disconnected" + a FRESH bar → override → true (the fix).
	s.lastBarNano.Store(time.Now().UnixNano())
	if !s.IsFeedConnected() {
		t.Fatal("a fresh bar_update must override a stale Disconnected (true)")
	}
	// 5. "Disconnected" + a STALE bar (older than the window) → false.
	s.lastBarNano.Store(time.Now().Add(-2 * feedFreshWindow).UnixNano())
	if s.IsFeedConnected() {
		t.Fatal("a stale bar (older than feedFreshWindow) must NOT override Disconnected")
	}
	// 6. Transient states (Connecting/Disconnecting) also honor the override.
	s.feedStatus = "Connecting"
	s.lastBarNano.Store(time.Now().UnixNano())
	if !s.IsFeedConnected() {
		t.Fatal("a fresh bar must override a non-Connected transient status too")
	}
}
