package kernel

import (
	"testing"
	"time"

	"nofx/market"
)

// plan-lifecycle wave (2026-08-27) — flip/death HYSTERESIS tests.
// Bars are 5m-SPACED 1m-length bars: each bar is its own 5m acceptance bucket,
// so the rule-TF aggregation is exact and every bucket closes with one bar.

func mk5(t0 int64, i int, o, h, l, c float64) market.Kline {
	ot := t0 + int64(i)*300_000
	return market.Kline{OpenTime: ot, CloseTime: ot + 300_000 - 1, Open: o, High: h, Low: l, Close: c}
}

// flatSeries builds n 5m-spaced flat bars at price p.
func flatSeries(t0 int64, n int, p float64) []market.Kline {
	out := make([]market.Kline, n)
	for i := range out {
		out[i] = mk5(t0, i, p, p, p, p)
	}
	return out
}

func closeBelow(base []market.Kline, t0 int64, at, m int, level, close float64) []market.Kline {
	for i := 0; i < m; i++ {
		base = append(base, mk5(t0, at+i, level, level, close, close))
	}
	return base
}

func closeAbove(base []market.Kline, t0 int64, at, m int, level, close float64) []market.Kline {
	for i := 0; i < m; i++ {
		base = append(base, mk5(t0, at+i, level, close, level, close))
	}
	return base
}

func TestFlipConditionWickThroughDoesNotFire(t *testing.T) {
	t.Setenv("FLIP_ATR_BUFFER", "0")
	t0 := time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC).UnixMilli()
	bars := flatSeries(t0, 20, 100)
	bars = append(bars, mk5(t0, 20, 100, 100, 94, 99))        // wick below, closes back
	bars = append(bars, flatSeries(t0+21*300_000, 5, 101)...) // closes stay on the valid side
	c := PlanCondition{Price: 100, Side: "below", Rule: "2x5m"}
	if fired, reason := PlanConditionFiredSince(c, bars, t0, t0+26*300_000); fired {
		t.Fatalf("wick-through must not fire the flip, reason=%q", reason)
	}
}

func TestFlipConditionTwoClosesBeyondBufferFires(t *testing.T) {
	t.Setenv("FLIP_ATR_BUFFER", "0")
	t0 := time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC).UnixMilli()
	bars := flatSeries(t0, 14, 100)
	bars = closeBelow(bars, t0, 14, 2, 100, 96)
	c := PlanCondition{Price: 100, Side: "below", Rule: "2x5m"}
	if fired, reason := PlanConditionFiredSince(c, bars, t0, t0+16*300_000); !fired {
		t.Fatalf("two closes beyond must fire, reason=%q", reason)
	}
}

func TestFlipConditionBufferBlocksNearClose(t *testing.T) {
	t.Setenv("FLIP_ATR_BUFFER", "0.5")
	t0 := time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC).UnixMilli()
	// 14 bars with a 10pt range → ATR14 ≈ 10 → buffer = 5.
	bars := make([]market.Kline, 0, 20)
	for i := 0; i < 14; i++ {
		bars = append(bars, mk5(t0, i, 100, 105, 95, 100))
	}
	// close at 97 is below the RAW line (100) but INSIDE the buffer (100−5=95).
	bars = closeBelow(bars, t0, 14, 2, 100, 97)
	c := PlanCondition{Price: 100, Side: "below", Rule: "2x5m"}
	if fired, reason := PlanConditionFiredSince(c, bars, t0, t0+16*300_000); fired {
		t.Fatalf("close inside the buffer must NOT fire, reason=%q", reason)
	}
	// close at 93 is beyond the buffered line → fires.
	bars = closeBelow(bars, t0, 16, 2, 100, 93)
	if fired, reason := PlanConditionFiredSince(c, bars, t0, t0+18*300_000); !fired {
		t.Fatalf("close beyond the buffered line must fire, reason=%q", reason)
	}
}

func TestFlipConditionLongShortSymmetry(t *testing.T) {
	t.Setenv("FLIP_ATR_BUFFER", "0")
	t0 := time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC).UnixMilli()
	bars := flatSeries(t0, 14, 100)
	bars = closeAbove(bars, t0, 14, 2, 100, 104)
	cAbove := PlanCondition{Price: 100, Side: "above", Rule: "2x5m"}
	if fired, reason := PlanConditionFiredSince(cAbove, bars, t0, t0+16*300_000); !fired {
		t.Fatalf("above-condition: two closes beyond must fire, reason=%q", reason)
	}
	barsL := flatSeries(t0, 14, 100)
	barsL = closeBelow(barsL, t0, 14, 2, 100, 96)
	cBelow := PlanCondition{Price: 100, Side: "below", Rule: "2x5m"}
	if fired, reason := PlanConditionFiredSince(cBelow, barsL, t0, t0+16*300_000); !fired {
		t.Fatalf("below-condition: two closes beyond must fire, reason=%q", reason)
	}
}

func TestFlipConfirmClosesFloor(t *testing.T) {
	t.Setenv("FLIP_ATR_BUFFER", "0")
	t.Setenv("FLIP_CONFIRM_CLOSES", "2")
	t0 := time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC).UnixMilli()
	// rule 5m_close nominally needs ONE close; the confirm floor demands TWO.
	bars := flatSeries(t0, 14, 100)
	bars = closeBelow(bars, t0, 14, 1, 100, 96)
	c := PlanCondition{Price: 100, Side: "below", Rule: "5m_close"}
	if fired, reason := PlanConditionFiredSince(c, bars, t0, t0+15*300_000); fired {
		t.Fatalf("one close must not satisfy the FLIP_CONFIRM_CLOSES=2 floor, reason=%q", reason)
	}
	bars = closeBelow(bars, t0, 15, 1, 100, 96)
	if fired, reason := PlanConditionFiredSince(c, bars, t0, t0+16*300_000); !fired {
		t.Fatalf("two closes must satisfy the floor, reason=%q", reason)
	}
}

func TestPlanConditionClearedSinceRearmLongShort(t *testing.T) {
	t.Setenv("FLIP_ATR_BUFFER", "0")
	t0 := time.Date(2026, 8, 26, 8, 0, 0, 0, time.UTC).UnixMilli()
	bars := flatSeries(t0, 14, 100)
	bars = closeBelow(bars, t0, 14, 2, 100, 96) // fired side
	bars = closeAbove(bars, t0, 16, 2, 100, 104)
	c := PlanCondition{Price: 100, Side: "below", Rule: "2x5m"}
	if cleared, reason := PlanConditionClearedSince(c, bars, t0, t0+18*300_000); !cleared {
		t.Fatalf("close back on the valid side must re-arm (long symmetry), reason=%q", reason)
	}
	barsS := flatSeries(t0, 14, 100)
	barsS = closeAbove(barsS, t0, 14, 2, 100, 104)
	barsS = closeBelow(barsS, t0, 16, 2, 100, 96)
	cA := PlanCondition{Price: 100, Side: "above", Rule: "2x5m"}
	if cleared, reason := PlanConditionClearedSince(cA, barsS, t0, t0+18*300_000); !cleared {
		t.Fatalf("close back below must re-arm (short symmetry), reason=%q", reason)
	}
}
