package trader

import (
	"testing"

	"nofx/store"
)

// W12 — auto-breakeven trigger basis: points = (mark−entry) for LONG, (entry−mark)
// for SHORT; fires when points ≥ trigger (default 50). Sign must be side-correct
// (the historical inversion bug: a case-sensitive "long" compare never matched, so
// breakeven armed on losers). Verified for both sides + the disabled gate.
func TestW12BreakevenTrigger(t *testing.T) {
	on := func(pts float64) store.RiskControlConfig {
		b := true
		return store.RiskControlConfig{BreakevenEnabled: &b, BreakevenTriggerPoints: pts}
	}

	// LONG +55 with trigger 50 → fires.
	if fire, pts := breakevenTrigger(on(50), "LONG", 30000, 30055); !fire || pts != 55 {
		t.Fatalf("LONG +55/trig50: fire=%v pts=%v, want true/55", fire, pts)
	}
	// LONG +40 with trigger 50 → does NOT fire (below trigger).
	if fire, _ := breakevenTrigger(on(50), "LONG", 30000, 30040); fire {
		t.Fatal("LONG +40 must not fire at trigger 50")
	}
	// SHORT +55 (mark BELOW entry) → fires; a winning short is +profit.
	if fire, pts := breakevenTrigger(on(50), "SHORT", 30000, 29945); !fire || pts != 55 {
		t.Fatalf("SHORT +55/trig50: fire=%v pts=%v, want true/55", fire, pts)
	}
	// SHORT that MOVED AGAINST (mark above entry) → negative points, no fire.
	if fire, pts := breakevenTrigger(on(50), "SHORT", 30000, 30055); fire || pts != -55 {
		t.Fatalf("SHORT losing: fire=%v pts=%v, want false/-55", fire, pts)
	}
	// case-insensitive side (NT8 emits UPPERCASE) — lowercase must also work.
	if fire, _ := breakevenTrigger(on(50), "long", 30000, 30055); !fire {
		t.Fatal("lowercase 'long' must be handled (the inversion-bug fix)")
	}
	// disabled → never fires.
	if fire, _ := breakevenTrigger(store.RiskControlConfig{}, "LONG", 30000, 31000); fire {
		t.Fatal("breakeven disabled → must not fire")
	}
	// default trigger 50 when unset (0).
	if fire, _ := breakevenTrigger(on(0), "LONG", 30000, 30050); !fire {
		t.Fatal("unset trigger → default 50; +50 must fire")
	}
}
