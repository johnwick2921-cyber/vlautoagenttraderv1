package kernel

import (
	"testing"
	"time"

	"nofx/market"
)

// TestP1CStillValidRequiresTouch is the regression for 2026-08-18 ASIA:
// the card and the PLAN STATUS block showed EVERY level "consumed" a few
// minutes after plan birth because StillValid only looked at acceptance-
// through — a level above price has 2 closes below it (and a support has
// 2 closes above it) almost immediately, so all levels flipped consumed
// without a single touch. Validity must be touch-gated, exactly like
// ConsumedSince and PlanIsDeadSince.
func TestP1CStillValidRequiresTouch(t *testing.T) {
	base := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	const level = 30000.0
	// Price sits ABOVE the level for 40 minutes straight (support below
	// price all session) — 2 consecutive rule-TF closes above are trivially
	// present, but the level was never touched.
	var bars []market.Kline
	for i := 0; i < 40; i++ {
		px := level + 20 + float64(i)
		bars = append(bars, market.Kline{
			OpenTime: base.Add(time.Duration(i) * time.Minute).UnixMilli(),
			Open:     px - 1, High: px + 2, Low: px - 3, Close: px,
			CloseTime: base.Add(time.Duration(i+1) * time.Minute).UnixMilli(),
		})
	}
	now := base.Add(45 * time.Minute).UnixMilli()

	f := EvaluateLevelFacts(bars, level, DirAbove, "2x5m", 3, now)
	if !f.StillValid {
		t.Fatalf("untouched level must stay valid, got cbUp=%d cbDn=%d (the all-consumed-after-2-bars bug)",
			f.ClosesBeyondUp, f.ClosesBeyondDown)
	}
	// And it must NOT be consumed by the ledger verdict either.
	if ConsumedSince(bars, level, "2x5m", base.UnixMilli(), now) {
		t.Fatal("untouched level must not be consumed")
	}
}

// TestP1CStillValidFalseOnlyAfterTouchAndAcceptance: the same series, but
// price touches the level once and then closes beyond for the rest of the
// window → consumed.
func TestP1CStillValidFalseOnlyAfterTouchAndAcceptance(t *testing.T) {
	base := time.Date(2026, 8, 18, 10, 0, 0, 0, time.UTC)
	const level = 30000.0
	bars := []market.Kline{{
		OpenTime: base.UnixMilli(),
		Open:     level - 2, High: level + 2, Low: level - 4, Close: level - 1,
		CloseTime: base.Add(time.Minute).UnixMilli(),
	}} // straddle = touch
	for i := 1; i < 40; i++ {
		px := level + 10 + float64(i)
		bars = append(bars, market.Kline{
			OpenTime: base.Add(time.Duration(i) * time.Minute).UnixMilli(),
			Open:     px - 1, High: px + 2, Low: px - 3, Close: px,
			CloseTime: base.Add(time.Duration(i+1) * time.Minute).UnixMilli(),
		})
	}
	now := base.Add(45 * time.Minute).UnixMilli()

	f := EvaluateLevelFacts(bars, level, DirAbove, "2x5m", 3, now)
	if f.StillValid {
		t.Fatal("touched + accepted-through level must read consumed")
	}
	if !ConsumedSince(bars, level, "2x5m", base.UnixMilli(), now) {
		t.Fatal("ledger verdict must agree: touched + accepted through = consumed")
	}
}
