package kernel

import (
	"testing"

	"nofx/market"
)

// P0 2026-08-17 — a plan may only be invalidated by what the market did AFTER it
// was written.
//
// The regression these tests lock: activePlanIsDead fed the FULL 2000×1m cache
// (~33 hours) into PlanIsDead, so every level inside the last day and a half's
// range counted as touched-and-accepted the instant a plan was born. Plan
// 2026-08-16:ASIA died five times in a row this way — v1..v5 each carried 6 good
// levels — until the re-plan budget ran out and a levels:null NO-TRADE plan was
// stored, which is the "No levels in this plan" the owner saw on every version.

func dbar(openMs int64, o, h, l, c float64) market.Kline {
	return market.Kline{OpenTime: openMs, Open: o, High: h, Low: l, Close: c, CloseTime: openMs + 60_000}
}

// The real ASIA v5 levels, in their real ~57pt band.
func asiaV5Levels() []PlanLevel {
	return []PlanLevel{
		{Price: 30199.5, Label: "EQL", Grade: "A", Instruction: "fade_reject"},
		{Price: 30203, Label: "ONH", Grade: "A", Instruction: "breakout_retest"},
		{Price: 30166.25, Label: "ONL", Grade: "A", Instruction: "target_support"},
		{Price: 30146.75, Label: "PDL", Grade: "B", Instruction: "target_support"},
		{Price: 30147.5, Label: "PDC", Grade: "B", Instruction: "watch_reclaim"},
		{Price: 30180, Label: "VWAP", Grade: "C", Instruction: "magnet"},
	}
}

// priorHistory reproduces the live cache the planner actually ran against: 2000
// one-minute bars (~33h) that sweep the whole level band again and again — so
// every level is TOUCHED — and then run clear ABOVE the band and stay there. That
// trailing run is what LevelStillValid reads: `need` consecutive closes beyond on
// one side means the level was accepted through, i.e. consumed.
//
// planWrittenMs is the last bar's open time: the plan is born at the right edge.
func priorHistory(planWrittenMs int64) []market.Kline {
	const n = 2000
	var bars []market.Kline
	t := planWrittenMs - (n-1)*60_000
	for i := 0; i < n; i++ {
		switch {
		case i < n-8:
			// Oscillate across the whole band (30140..30210) — brackets every level.
			if i%2 == 0 {
				bars = append(bars, dbar(t, 30150, 30210, 30140, 30200))
			} else {
				bars = append(bars, dbar(t, 30200, 30210, 30140, 30150))
			}
		default:
			// Then price leaves the band and holds above it: no level is touched
			// here, but every one now has a long run of closes beyond.
			bars = append(bars, dbar(t, 30225, 30240, 30220, 30235))
		}
		t += 60_000
	}
	return bars
}

// THE BUG, REPRODUCED: judged against the whole cache, a brand-new plan is
// already dead the moment it is written.
func TestPlanBornDeadAgainstFullCache(t *testing.T) {
	planWrittenMs := int64(1_760_000_000_000)
	now := planWrittenMs + 60_000 // the plan's own bar has closed
	doc := PlanDoc{Levels: asiaV5Levels(), DeathCondition: "1h close below 30146.75 PDL"}
	bars := priorHistory(planWrittenMs)

	// This is the pre-fix call, and it is why v1..v5 each died on arrival.
	if !PlanIsDead(doc, bars, "2x5m", now) {
		t.Fatal("fixture no longer reproduces the born-dead condition — it must, or this file guards nothing")
	}

	// Same plan, same bars, judged only on what the market did AFTER the write.
	if PlanIsDeadSince(doc, bars, "2x5m", planWrittenMs, now) {
		t.Fatal("a plan must not be killed by evidence that predates it — this is the loop that burned v1..v5")
	}
}

// No post-plan bars at all → never dead. "No evidence" is not "invalidated".
func TestPlanNotDeadWithoutPostPlanEvidence(t *testing.T) {
	now := int64(1_760_000_000_000)
	doc := PlanDoc{Levels: asiaV5Levels()}
	bars := priorHistory(now)
	if PlanIsDeadSince(doc, bars, "2x5m", now+120_000, now) {
		t.Fatal("with sinceMs in the future there are zero qualifying bars; the plan cannot be dead")
	}
}

// It must STILL die when the market genuinely invalidates it after the write.
func TestPlanStillDiesOnPostPlanEvidence(t *testing.T) {
	planWrittenMs := int64(1_760_000_000_000)
	doc := PlanDoc{Levels: []PlanLevel{{Price: 30200, Label: "ONH", Grade: "A"}}}

	var bars []market.Kline
	bars = append(bars, priorHistory(planWrittenMs)...)
	// AFTER the plan: touch the level, then accept decisively above it.
	t0 := planWrittenMs
	bars = append(bars,
		dbar(t0, 30195, 30205, 30190, 30201),         // touched
		dbar(t0+60_000, 30210, 30225, 30208, 30220),  // beyond
		dbar(t0+120_000, 30222, 30240, 30220, 30235), // accepted through
	)
	if !PlanIsDeadSince(doc, bars, "2x5m", planWrittenMs, t0+300_000) {
		t.Fatal("a level touched AND accepted through AFTER the write must still kill the plan")
	}
}

// A plan with no levels never dies this way (unchanged contract).
func TestPlanWithNoLevelsNeverDies(t *testing.T) {
	now := int64(1_760_000_000_000)
	if PlanIsDeadSince(PlanDoc{}, priorHistory(now), "2x5m", 0, now) {
		t.Fatal("a levels-less plan must never be reported dead")
	}
}

// sinceMs=0 preserves the pre-fix behavior for any caller that cannot supply a
// write time (the deprecated PlanIsDead wrapper relies on this).
func TestPlanIsDeadWrapperKeepsWholeWindow(t *testing.T) {
	now := int64(1_760_000_000_000)
	doc := PlanDoc{Levels: []PlanLevel{{Price: 30200, Label: "ONH", Grade: "A"}}}
	bars := priorHistory(now)
	if PlanIsDead(doc, bars, "2x5m", now) != PlanIsDeadSince(doc, bars, "2x5m", 0, now) {
		t.Fatal("PlanIsDead must equal PlanIsDeadSince(sinceMs=0)")
	}
}

func TestBarsSinceWindow(t *testing.T) {
	bars := []market.Kline{dbar(1000, 1, 1, 1, 1), dbar(2000, 2, 2, 2, 2), dbar(3000, 3, 3, 3, 3)}
	if got := barsSince(bars, 2000); len(got) != 2 || got[0].OpenTime != 2000 {
		t.Fatalf("barsSince(2000) = %d bars starting %d, want 2 starting 2000", len(got), got[0].OpenTime)
	}
	if got := barsSince(bars, 9999); got != nil {
		t.Fatalf("a cutoff past every bar must yield nothing, got %d", len(got))
	}
	if got := barsSince(bars, 0); len(got) != 3 {
		t.Fatalf("cutoff 0 must keep every bar, got %d", len(got))
	}
}
