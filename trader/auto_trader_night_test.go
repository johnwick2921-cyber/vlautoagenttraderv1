package trader

import (
	"testing"

	"nofx/kernel"
)

func TestIsNightMode(t *testing.T) {
	reg := kernel.DefaultSessionRegistry() // NY enabled; ASIA/LONDON disabled
	cases := []struct {
		h, m  int
		night bool
	}{
		{9, 30, false}, // NY (enabled) → day
		{18, 0, true},  // ASIA window but DISABLED → night
		{3, 0, true},   // LONDON window but DISABLED → night
		{16, 0, true},  // interim gap → night
	}
	for _, c := range cases {
		if got := reg.IsNightMode(ctAt(t, c.h, c.m)); got != c.night {
			t.Fatalf("IsNightMode(%02d:%02d) = %v want %v", c.h, c.m, got, c.night)
		}
	}
}

func TestNightEdgeDecision(t *testing.T) {
	day, night := false, true
	// Fresh (re)start: nil prev → NO event (restart during night resumes cleanly).
	if nightEdgeDecision(nil, true) || nightEdgeDecision(nil, false) {
		t.Fatalf("nil prev must never emit (restart-clean)")
	}
	// Same state → no event.
	if nightEdgeDecision(&night, true) || nightEdgeDecision(&day, false) {
		t.Fatalf("unchanged state must not emit")
	}
	// Transition → event.
	if !nightEdgeDecision(&day, true) {
		t.Fatalf("day→night must emit")
	}
	if !nightEdgeDecision(&night, false) {
		t.Fatalf("night→day must emit")
	}
}

func TestObserveNightEdgeRestartClean(t *testing.T) {
	// day_plan off → dormant (no state touched).
	off := mkTrader("ninjatrader", nil, "5m")
	off.observeNightEdge()
	if off.nightPrev != nil {
		t.Fatalf("plan-off must not observe night state")
	}
	// day_plan on → first observation sets state WITHOUT a transition (clean resume).
	on := mkTrader("ninjatrader", boolp(true), "5m")
	on.observeNightEdge()
	if on.nightPrev == nil {
		t.Fatalf("first observation must record the current night state")
	}
}
