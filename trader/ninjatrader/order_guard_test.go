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
