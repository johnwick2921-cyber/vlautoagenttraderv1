package trader

import (
	"testing"

	"nofx/store"
)

func bePtr(b bool) *bool { return &b }

func TestBreakevenTrigger(t *testing.T) {
	on, off := bePtr(true), bePtr(false)
	cases := []struct {
		name      string
		rc        store.RiskControlConfig
		side      string
		entry     float64
		mark      float64
		wantFire  bool
		wantPtsGE float64 // sanity: pts should be >= this
	}{
		{"disabled (nil) never fires", store.RiskControlConfig{}, "long", 30352, 30450, false, 0},
		{"disabled (false) never fires", store.RiskControlConfig{BreakevenEnabled: off}, "long", 30352, 30450, false, 0},
		{"long +98 >= default 50 → fire", store.RiskControlConfig{BreakevenEnabled: on}, "long", 30352, 30450, true, 90},
		{"long +40 < default 50 → no", store.RiskControlConfig{BreakevenEnabled: on}, "long", 30352, 30392, false, 0},
		{"long exactly +50 → fire", store.RiskControlConfig{BreakevenEnabled: on}, "long", 30352, 30402, true, 50},
		{"short +60 >= 50 → fire (mirror)", store.RiskControlConfig{BreakevenEnabled: on}, "short", 30400, 30340, true, 60},
		{"short losing → no", store.RiskControlConfig{BreakevenEnabled: on}, "short", 30400, 30460, false, 0},
		{"custom trigger 20, long +25 → fire", store.RiskControlConfig{BreakevenEnabled: on, BreakevenTriggerPoints: 20}, "long", 30352, 30377, true, 25},
		{"custom trigger 100, long +25 → no", store.RiskControlConfig{BreakevenEnabled: on, BreakevenTriggerPoints: 100}, "long", 30352, 30377, false, 0},
		// UPPERCASE side — this is what the real caller passes (NT8 positionMap →
		// upperSideStr → "LONG"/"SHORT"). These rows fail against the pre-fix
		// case-sensitive == "long" comparison and guard the casing regression.
		{"UPPER long +98 >= 50 → fire", store.RiskControlConfig{BreakevenEnabled: on}, "LONG", 30352, 30450, true, 90},
		{"UPPER long losing → no (not inverted)", store.RiskControlConfig{BreakevenEnabled: on}, "LONG", 30352, 30300, false, 0},
		{"UPPER short +60 >= 50 → fire", store.RiskControlConfig{BreakevenEnabled: on}, "SHORT", 30400, 30340, true, 60},
		{"UPPER short losing → no", store.RiskControlConfig{BreakevenEnabled: on}, "SHORT", 30400, 30460, false, 0},
		{"UPPER long exactly +50 → fire (boundary)", store.RiskControlConfig{BreakevenEnabled: on}, "LONG", 30352, 30402, true, 50},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			fire, pts := breakevenTrigger(c.rc, c.side, c.entry, c.mark)
			if fire != c.wantFire {
				t.Fatalf("fire=%v want %v (pts=%.1f)", fire, c.wantFire, pts)
			}
			if fire && pts < c.wantPtsGE {
				t.Fatalf("pts=%.1f want >= %.1f", pts, c.wantPtsGE)
			}
		})
	}
}

// TestBreakevenTrigger_NT8UppercaseNotInverted reproduces the exact casing the
// real production caller passes. checkPositionDrawdown reads pos["side"] straight
// from NT8's GetPositions/positionMap, which emits UPPERCASE "LONG"/"SHORT"
// (upperSideStr). The pre-fix breakevenTrigger compared side == "long" (lowercase),
// so every NT8 long fell into the short branch pts = entry-mark — inverted: a
// WINNING long produced negative pts and never armed, while a LOSING long produced
// positive pts and would have moved the stop to entry on a loser (instant stop-out).
// This test asserts the post-fix, correct direction for the caller's real casing.
func TestBreakevenTrigger_NT8UppercaseNotInverted(t *testing.T) {
	on := bePtr(true)
	rc := store.RiskControlConfig{BreakevenEnabled: on} // default trigger 50

	// A winning long (+62.5) MUST arm — the case the bug silently broke.
	if fire, pts := breakevenTrigger(rc, "LONG", 29129, 29191.5); !fire || pts <= 0 {
		t.Fatalf("winning LONG: fire=%v pts=%.1f — want fire=true, pts>0 (pre-fix bug: never fired on a winner)", fire, pts)
	}
	// A losing long (-62.5) MUST NOT arm — pre-fix it would have (moving the stop to entry on a loser).
	if fire, pts := breakevenTrigger(rc, "LONG", 29129, 29066.5); fire {
		t.Fatalf("losing LONG: fire=%v pts=%.1f — want fire=false (pre-fix bug: fired on a loser)", fire, pts)
	}
	// Mirror for shorts: a winning short (+62.5) MUST arm.
	if fire, pts := breakevenTrigger(rc, "SHORT", 29515.75, 29453.25); !fire || pts <= 0 {
		t.Fatalf("winning SHORT: fire=%v pts=%.1f — want fire=true, pts>0", fire, pts)
	}
	// A losing short (-62.5) MUST NOT arm.
	if fire, _ := breakevenTrigger(rc, "SHORT", 29515.75, 29578.25); fire {
		t.Fatalf("losing SHORT: want fire=false")
	}
}
