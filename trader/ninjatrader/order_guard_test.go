// B3 — dupe guard + rate limiter: replayed order dropped; runaway trips a breaker.
package ninjatrader

import (
	"fmt"
	"strings"
	"testing"
)

func TestOrderGuard_Dedup(t *testing.T) {
	g := newOrderGuard()
	now := int64(1_000_000)
	key := "Sim101|LONG|MNQ|1"

	if _, ok := g.admit(key, now); !ok {
		t.Fatal("first order must be admitted")
	}
	// Replayed / double-fired signal within the bar window → DROPPED.
	if reason, ok := g.admit(key, now+1000); ok || !strings.Contains(reason, "duplicate") {
		t.Fatalf("replayed key within window must be dropped, got ok=%v reason=%q", ok, reason)
	}
	// After the dedup window the same key is admitted again (recovers).
	if _, ok := g.admit(key, now+orderDedupWindowMs+1); !ok {
		t.Fatal("same key after the dedup window must be admitted")
	}
}

func TestOrderGuard_RateBreaker(t *testing.T) {
	g := newOrderGuard()
	base := int64(2_000_000)

	// orderRateMax distinct-key admits within the window → all pass.
	for i := 0; i < orderRateMax; i++ {
		if _, ok := g.admit(fmt.Sprintf("acct|LONG|MNQ|%d", i), base+int64(i)); !ok {
			t.Fatalf("admit %d should pass (under the rate limit)", i)
		}
	}
	// The next order-action within the same minute trips the breaker.
	if reason, ok := g.admit("acct|LONG|MNQ|X", base+50); ok || !strings.Contains(reason, "breaker") {
		t.Fatalf("action past the rate limit must trip the breaker, got ok=%v reason=%q", ok, reason)
	}
	// After the rate window drains, actions are admitted again (breaker recovers).
	if _, ok := g.admit("acct|LONG|MNQ|Y", base+orderRateWindowMs+100); !ok {
		t.Fatal("breaker must recover once the rate window drains")
	}
}

// W1b E12(b) — reserve/done: the key is recorded ONLY by done(true).
func TestOrderGuard_ReserveCommitsOnlyOnSend(t *testing.T) {
	g := newOrderGuard()
	now := int64(3_000_000)
	key := "armed|Sim101|LONG|MNQ|1"

	done, _, ok := g.reserve(key, now)
	if !ok {
		t.Fatal("first reservation must pass")
	}
	// While reserved, the same key is refused (no check/commit race).
	if reason, ok := g.admit(key, now+1); ok || !strings.Contains(reason, "in flight") {
		t.Fatalf("a key in flight must be refused as a duplicate, got ok=%v %q", ok, reason)
	}
	done(false) // refused after B3 — nothing reached the wire
	done(true)  // a second call is a no-op (once)
	if _, seen := g.lastSeen[key]; seen {
		t.Fatal("done(false) recorded the key; a second done must not record it either")
	}
	if len(g.actionTimes) != 0 {
		t.Fatalf("an unsent action counted toward the rate breaker: %v", g.actionTimes)
	}
	// The retry is admitted, and ITS send records the key.
	done2, _, ok := g.reserve(key, now+10)
	if !ok {
		t.Fatal("retry after an unsent reservation must pass")
	}
	done2(true)
	if reason, ok := g.admit(key, now+20); ok || !strings.Contains(reason, "duplicate") {
		t.Fatalf("a SENT key inside the window must still be dropped, got ok=%v %q", ok, reason)
	}
}

// Reservations in flight count toward the breaker (fail-closed across the gap).
func TestOrderGuard_ReservationsCountTowardBreaker(t *testing.T) {
	g := newOrderGuard()
	base := int64(4_000_000)
	for i := 0; i < orderRateMax; i++ {
		if _, _, ok := g.reserve(fmt.Sprintf("k%d", i), base); !ok {
			t.Fatalf("reservation %d should pass", i)
		}
	}
	if reason, ok := g.admit("kX", base+1); ok || !strings.Contains(reason, "breaker") {
		t.Fatalf("in-flight reservations must count toward the breaker, got ok=%v %q", ok, reason)
	}
}
