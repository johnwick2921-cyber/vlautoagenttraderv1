package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
)

// A6 (mega-research 2026-08-26) — FVG session-boundary guard: a 3-candle
// window that straddles the 16:00-17:00 CT halt (or any >3×-interval gap) is
// a phantom, never a real imbalance. Normal triples and DST days pass.

func fvgBar(ms int64, o, h, l, c float64) market.Kline {
	return market.Kline{OpenTime: ms, CloseTime: ms + 59_999, Open: o, High: h, Low: l, Close: c, Volume: 100}
}

func TestFvgHaltStraddleRejected(t *testing.T) {
	loc := CTLocation()
	// 15:58, 15:59, then the next bar after the 16:00-17:00 halt: 17:00.
	// Price pattern = bullish FVG (newest low > oldest high).
	bars := []market.Kline{
		fvgBar(time.Date(2026, 8, 25, 15, 58, 0, 0, loc).UnixMilli(), 30000, 30100, 29900, 30050),
		fvgBar(time.Date(2026, 8, 25, 15, 59, 0, 0, loc).UnixMilli(), 30050, 30150, 29950, 30100),
		fvgBar(time.Date(2026, 8, 25, 17, 0, 0, 0, loc).UnixMilli(), 30300, 30350, 30200, 30300), // low 30200 > high[oldest] 30100
	}
	now := time.Date(2026, 8, 25, 18, 0, 0, 0, loc)
	if got := FairValueGaps(bars, 2.0, now); len(got) != 0 {
		t.Fatalf("halt-straddling triple must NOT emit an FVG, got %+v", got)
	}
}

func TestFvgNormalTriplePasses(t *testing.T) {
	loc := CTLocation()
	base := time.Date(2026, 8, 25, 10, 0, 0, 0, loc).UnixMilli()
	bars := []market.Kline{
		fvgBar(base, 30000, 30100, 29900, 30050),
		fvgBar(base+60_000, 30050, 30150, 29950, 30100),
		fvgBar(base+120_000, 30300, 30350, 30200, 30300), // bullish gap
	}
	now := time.Date(2026, 8, 25, 10, 10, 0, 0, loc)
	if got := FairValueGaps(bars, 2.0, now); len(got) != 1 {
		t.Fatalf("contiguous triple must emit one FVG, got %+v", got)
	}
}

// DST spring-forward day (2026-03-08): 1m bars stay 60s apart in wall time →
// the guard's interval check passes; the DST shift must NOT reject the triple.
func TestFvgDSTDayPasses(t *testing.T) {
	loc := CTLocation()
	base := time.Date(2026, 3, 8, 8, 0, 0, 0, loc).UnixMilli()
	bars := []market.Kline{
		fvgBar(base, 30000, 30100, 29900, 30050),
		fvgBar(base+60_000, 30050, 30150, 29950, 30100),
		fvgBar(base+120_000, 30300, 30350, 30200, 30300),
	}
	now := time.Date(2026, 3, 8, 8, 10, 0, 0, loc)
	if got := FairValueGaps(bars, 2.0, now); len(got) != 1 {
		t.Fatalf("DST-day contiguous triple must emit one FVG, got %+v", got)
	}
}

func TestValidateFvgEntryHaltStraddleRejected(t *testing.T) {
	loc := CTLocation()
	bars := []market.Kline{
		fvgBar(time.Date(2026, 8, 25, 15, 58, 0, 0, loc).UnixMilli(), 30000, 30100, 29900, 30050),
		fvgBar(time.Date(2026, 8, 25, 15, 59, 0, 0, loc).UnixMilli(), 30050, 30150, 29950, 30100),
		fvgBar(time.Date(2026, 8, 25, 17, 0, 0, 0, loc).UnixMilli(), 30300, 30350, 30200, 30300),
	}
	s := PlanScenario{
		ID: "S1", Condition: "fvg_entry", Direction: "long",
		Fvg: &PlanFvgEntry{Lo: 30100, Hi: 30200, CE: 30150, EntryMode: "edge", OriginLevel: "PDH", Direction: "long"},
	}
	err := validateOneFvgEntry(s, bars, "MNQ", map[string]bool{"PDH": true}, time.Date(2026, 8, 25, 18, 0, 0, 0, loc))
	if err == nil || !strings.Contains(err.Error(), "no fresh 3-candle gap") {
		t.Fatalf("halt-straddling declared gap must fail validation, got %v", err)
	}
}
