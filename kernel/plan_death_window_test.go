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
// every level is TOUCHED — and then run clear ABOVE the band for 20 minutes. That
// trailing run is what LevelStillValid reads: `need` consecutive closes beyond on
// one side (counted on the RULE's timeframe — 5m for "2x5m") means the level was
// accepted through, i.e. consumed.
//
// planWrittenMs is the last bar's open time: the plan is born at the right edge.
func priorHistory(planWrittenMs int64) []market.Kline {
	const n = 2000
	var bars []market.Kline
	t := planWrittenMs - (n-1)*60_000
	for i := 0; i < n; i++ {
		switch {
		case i < n-20:
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
	// AFTER the plan: touch the level, then hold above it long enough for TWO
	// five-minute closes — real acceptance under the "2x5m" rule.
	t0 := planWrittenMs
	bars = append(bars, dbar(t0, 30195, 30205, 30190, 30201)) // touched, closed above
	for i := 1; i < 16; i++ {
		bars = append(bars, dbar(t0+int64(i)*60_000, 30210, 30225, 30208, 30220))
	}
	if !PlanIsDeadSince(doc, bars, "2x5m", planWrittenMs, t0+16*60_000) {
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

// SECOND ROOT CAUSE — "2x5m" must mean two FIVE-minute closes.
//
// Measured at the real v5 death check (22:37:29Z), with price at 30193.50 sitting
// INSIDE the level stack, every level read "consumed": PDL 30146.75 had 997
// consecutive 1m closes above it, EQL 30199.50 had 1275 below. Counting bars
// instead of minutes turned the acceptance rule into "is this level more than two
// minutes of price action away", which every level in a day plan trivially is.
func TestAcceptanceIsCountedOnTheRuleTimeframe(t *testing.T) {
	const level = 30200.0
	// Bucket-aligned so "one complete 5m bar" is exactly five 1m bars.
	planWrittenMs := int64(1_760_000_000_000) / 300_000 * 300_000

	// Six 1-minute bars after the write: the level is touched, then price closes
	// above it for five straight MINUTES — but that is only ONE completed 5m bar,
	// so "2x5m" acceptance is NOT met yet.
	var bars []market.Kline
	t0 := planWrittenMs
	bars = append(bars, dbar(t0, 30195, 30205, 30190, 30202)) // touch + close above
	for i := 1; i < 6; i++ {
		bars = append(bars, dbar(t0+int64(i)*60_000, 30202, 30208, 30201, 30206))
	}
	now := t0 + 6*60_000
	doc := PlanDoc{Levels: []PlanLevel{{Price: level, Label: "ONH", Grade: "A"}}}

	if PlanIsDeadSince(doc, bars, "2x5m", planWrittenMs, now) {
		t.Fatal("five one-minute closes beyond is ONE 5m close — 2x5m acceptance must not be satisfied yet")
	}

	// Extend past the second completed 5-minute bucket → now it is genuinely dead.
	for i := 6; i < 16; i++ {
		bars = append(bars, dbar(t0+int64(i)*60_000, 30206, 30212, 30205, 30210))
	}
	if !PlanIsDeadSince(doc, bars, "2x5m", planWrittenMs, t0+16*60_000) {
		t.Fatal("two completed 5-minute closes beyond a touched level IS acceptance — the plan must die")
	}
}

func TestAggregateToMinutes(t *testing.T) {
	// Three 1m bars inside one 5m bucket, then one in the next.
	base := int64(1_760_000_000_000) / 300_000 * 300_000 // bucket-aligned
	bars := []market.Kline{
		{OpenTime: base, CloseTime: base + 60_000, Open: 10, High: 12, Low: 9, Close: 11, Volume: 3},
		{OpenTime: base + 60_000, CloseTime: base + 120_000, Open: 11, High: 15, Low: 8, Close: 14, Volume: 5},
		{OpenTime: base + 120_000, CloseTime: base + 180_000, Open: 14, High: 14, Low: 13, Close: 13, Volume: 2},
		{OpenTime: base + 300_000, CloseTime: base + 360_000, Open: 20, High: 21, Low: 19, Close: 20, Volume: 7},
	}
	got := aggregateToMinutes(bars, 5)
	if len(got) != 2 {
		t.Fatalf("want 2 five-minute bars, got %d", len(got))
	}
	b := got[0]
	if b.Open != 10 || b.High != 15 || b.Low != 8 || b.Close != 13 || b.Volume != 10 {
		t.Errorf("bucket 1 OHLCV = %v/%v/%v/%v vol %v, want 10/15/8/13 vol 10", b.Open, b.High, b.Low, b.Close, b.Volume)
	}
	// CloseTime is the last instant INSIDE the bucket (repo bar convention), so a
	// bucket counts as closed the moment `now` reaches its end — not a tick later.
	if b.OpenTime != base || b.CloseTime != base+300_000-1 {
		t.Errorf("bucket 1 times = %d..%d, want %d..%d", b.OpenTime, b.CloseTime, base, base+300_000-1)
	}
	// A closed market must NOT produce a bucket: there is no bar between the two.
	if got[1].OpenTime != base+300_000 {
		t.Errorf("bucket 2 starts %d, want %d — no empty bucket may be invented", got[1].OpenTime, base+300_000)
	}

	// Gaps stay gaps: a 40-minute hole yields no filler buckets.
	sparse := []market.Kline{
		{OpenTime: base, CloseTime: base + 60_000, Open: 1, High: 1, Low: 1, Close: 1},
		{OpenTime: base + 40*60_000, CloseTime: base + 41*60_000, Open: 2, High: 2, Low: 2, Close: 2},
	}
	if got := aggregateToMinutes(sparse, 5); len(got) != 2 {
		t.Errorf("a 40-minute gap must yield 2 buckets, not %d — no synthetic bars, ever", len(got))
	}
	if got := aggregateToMinutes(bars, 1); len(got) != len(bars) {
		t.Errorf("tf=1 must pass through unchanged")
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
