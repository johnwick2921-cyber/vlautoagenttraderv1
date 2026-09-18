package kernel

import (
	"testing"

	"nofx/market"
)

// W-FLIP-OWNS-THE-BREACH (2026-09-17) — resolver + breach-state units. The
// production call sites are pinned in trader/flip_breach_test.go.

func fbVersions() []PlanVersionFact {
	return []PlanVersionFact{
		{Version: 1, TriggerReason: "NY_scheduled_read", BiasDirection: "short", CreatedAtMs: 1_000, FlipPrice: 100, FlipSide: "above"},
		{Version: 2, TriggerReason: "level_event", BiasDirection: "short", CreatedAtMs: 2_000, FlipPrice: 101.5, FlipSide: "above"},
		{Version: 3, TriggerReason: "structure_mss", BiasDirection: "short", CreatedAtMs: 3_000, FlipPrice: 99, FlipSide: "above"},
	}
}

func TestResolveFlipConditionAnchorSameLineRunAnchorsOnEarliest(t *testing.T) {
	a := ResolveFlipConditionAnchor(fbVersions(), 3, 3.0, 9_999)
	if a.Source != FlipWindowChain || a.AnchorVersion != 1 || a.SinceMs != 1_000 || a.Moved {
		t.Fatalf("v3 (99) within 3.0 of v2 (101.5) and v1 (100) must anchor on v1: %+v", a)
	}
}

func TestResolveFlipConditionAnchorMovedLineWindowsFromBirth(t *testing.T) {
	vs := fbVersions()
	vs[2].FlipPrice = 110
	a := ResolveFlipConditionAnchor(vs, 3, 3.0, 9_999)
	if a.Source != FlipWindowMoved || !a.Moved || a.PrevPrice != 101.5 || a.SinceMs != 3_000 || a.AnchorVersion != 3 {
		t.Fatalf("a line moved 8.5 pt must window from v3's birth and report the previous line: %+v", a)
	}
	// side change is a move too
	vs[2].FlipPrice = 100
	vs[2].FlipSide = "below"
	if a := ResolveFlipConditionAnchor(vs, 3, 3.0, 9_999); !a.Moved {
		t.Fatalf("side change must read as moved: %+v", a)
	}
}

func TestResolveFlipConditionAnchorRunBreaks(t *testing.T) {
	// bias change breaks the run (v2 long) → v3 windows from its own birth, not moved.
	vs := fbVersions()
	vs[1].BiasDirection = "long"
	if a := ResolveFlipConditionAnchor(vs, 3, 3.0, 9_999); a.Source != FlipWindowVersion || a.SinceMs != 3_000 || a.Moved {
		t.Fatalf("bias change must break the run: %+v", a)
	}
	// a version with no line breaks the run.
	vs = fbVersions()
	vs[1].FlipPrice = 0
	if a := ResolveFlipConditionAnchor(vs, 3, 3.0, 9_999); a.Source != FlipWindowVersion || a.SinceMs != 3_000 {
		t.Fatalf("no-line version must break the run: %+v", a)
	}
	// a re-plan version starts the run (included) and stops the walk.
	vs = fbVersions()
	vs[1].TriggerReason = "death_replan"
	if a := ResolveFlipConditionAnchor(vs, 3, 3.0, 9_999); a.Source != FlipWindowChain || a.AnchorVersion != 2 || a.SinceMs != 2_000 {
		t.Fatalf("re-plan v2 must be the run's start: %+v", a)
	}
	// current itself a re-plan → own birth.
	vs = fbVersions()
	vs[2].TriggerReason = "owner_reread"
	if a := ResolveFlipConditionAnchor(vs, 3, 3.0, 9_999); a.Source != FlipWindowVersion || a.SinceMs != 3_000 {
		t.Fatalf("a re-plan current must window from its own birth: %+v", a)
	}
	// missing / unreadable → fallback tagged.
	if a := ResolveFlipConditionAnchor(nil, 3, 3.0, 9_999); a.Source != FlipWindowFallback || a.SinceMs != 9_999 {
		t.Fatalf("empty chain must fall back: %+v", a)
	}
}

// fbTape: flat 100 for `flat` minutes then `beyond` minutes closing at 104
// (first up bar touches 100), all closed at nowMs = end.
func fbTape(flat, beyond int) ([]market.Kline, int64) {
	start := int64(1_700_000_000_000)
	start -= start % 300_000
	var out []market.Kline
	for i := 0; i < flat+beyond; i++ {
		ot := start + int64(i)*60_000
		b := market.Kline{OpenTime: ot, CloseTime: ot + 60_000 - 1, Open: 100, High: 100, Low: 100, Close: 100}
		if i >= flat {
			b = market.Kline{OpenTime: ot, CloseTime: ot + 60_000 - 1, Open: 100, High: 104, Low: 100, Close: 104}
		}
		out = append(out, b)
	}
	return out, start + int64(flat+beyond)*60_000 + 30_000
}

func TestFlipBreachStateMirrorsTheEvaluator(t *testing.T) {
	t.Setenv("FLIP_ATR_BUFFER", "0")
	c := PlanCondition{Price: 100, Side: "above", Rule: "2x5m", FlipTo: "long"}
	// one 5m close beyond → breached 1/2, evaluator not fired
	bars, now := fbTape(30, 5)
	b := FlipBreachState(c, bars, 0, now)
	if !b.Breached || b.Closes != 1 || b.Need != 2 || !b.Touched || b.Stale {
		t.Fatalf("one close beyond must read breached 1/2: %+v", b)
	}
	if fired, _ := PlanConditionFiredSince(c, bars, 0, now); fired {
		t.Fatalf("evaluator must not fire on 1/2")
	}
	// two closes → both agree
	bars, now = fbTape(30, 10)
	b = FlipBreachState(c, bars, 0, now)
	fired, _ := PlanConditionFiredSince(c, bars, 0, now)
	if !b.Breached || b.Closes != 2 || !fired {
		t.Fatalf("2/2 must be breached AND fired: %+v fired=%v", b, fired)
	}
	// a line born beyond price and never touched is NOT a breach (deferring
	// wakes on it would park the plan behind a flip that cannot fire).
	far := PlanCondition{Price: 90, Side: "above", Rule: "2x5m"}
	if b := FlipBreachState(far, bars, 0, now); b.Breached || b.Touched {
		t.Fatalf("untouched line must not be a breach: %+v", b)
	}
	// stale tape → Stale, never Breached
	if b := FlipBreachState(c, bars, 0, now+20*60_000); !b.Stale || b.Breached || b.StaleWhy != "stale_bars" {
		t.Fatalf("stale tape must read stale: %+v", b)
	}
	// windowed: bars before sinceMs are never judged
	if b := FlipBreachState(c, bars, now+1, now); b.Breached || b.Closes != 0 {
		t.Fatalf("empty window must not breach: %+v", b)
	}
}

func TestPlanDeathOrFlipWindowsSeparateFlipFromDeath(t *testing.T) {
	t.Setenv("FLIP_ATR_BUFFER", "0")
	t.Setenv("FLIP_MIN_HOLD_MIN", "0")
	bars, now := fbTape(30, 10)
	doc := PlanDoc{Bias: PlanBias{Direction: "short"}, FlipStructured: &PlanCondition{Price: 100, Side: "above", Rule: "2x5m", FlipTo: "long"}}
	late := bars[len(bars)-4].OpenTime // window opens inside the second beyond-bucket
	hold := FlipHoldAnchor{SinceMs: 0, Source: FlipHoldAnchorBirth}
	if _, fired, _ := PlanDeathOrFlipSinceFreshHold(doc, bars, "2x5m", late, now, hold); fired {
		t.Fatalf("single window from a late birth must not fire")
	}
	if k, fired, _ := PlanDeathOrFlipSinceFreshHoldWindows(doc, bars, "2x5m", late, 0, now, hold); !fired || k == "" {
		t.Fatalf("flip windowed from the chain must fire while death keeps the version window")
	}
	// same value for both windows is byte-identical to the single-window function
	k1, f1, s1 := PlanDeathOrFlipSinceFreshHold(doc, bars, "2x5m", 0, now, hold)
	k2, f2, s2 := PlanDeathOrFlipSinceFreshHoldWindows(doc, bars, "2x5m", 0, 0, now, hold)
	if k1 != k2 || f1 != f2 || len(s1) != len(s2) {
		t.Fatalf("parity broken: %q/%v vs %q/%v", k1, f1, k2, f2)
	}
}
