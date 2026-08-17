package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
)

func TestRoundNumberLevels(t *testing.T) {
	// price 15530, dATR 200 → band ±300 → [15230, 15830].
	levels := RoundNumberLevels(15530, 200, 1.5)
	byPrice := map[float64]string{}
	for _, l := range levels {
		byPrice[l.Price] = l.Label
	}
	// Strongest-granularity dedup: 15500 is a 100, tagged (100), not (50)/(25).
	if lbl, ok := byPrice[15500]; !ok || !strings.Contains(lbl, "(100)") {
		t.Fatalf("15500 should be a 100-round: %q ok=%v", lbl, ok)
	}
	if lbl, ok := byPrice[15550]; !ok || !strings.Contains(lbl, "(50)") {
		t.Fatalf("15550 should be a 50-round: %q ok=%v", lbl, ok)
	}
	if lbl, ok := byPrice[15525]; !ok || !strings.Contains(lbl, "(25)") {
		t.Fatalf("15525 should be a 25-round: %q ok=%v", lbl, ok)
	}
	// Nothing outside the ±1.5×dATR band.
	if _, ok := byPrice[15200]; ok {
		t.Fatalf("15200 is below the band, must not appear")
	}
	if _, ok := byPrice[15900]; ok {
		t.Fatalf("15900 is above the band, must not appear")
	}
	// Degenerate inputs → nil.
	if RoundNumberLevels(0, 200, 1.5) != nil || RoundNumberLevels(15000, 0, 1.5) != nil {
		t.Fatalf("degenerate inputs should return nil")
	}
}

func TestGapLevelsUnfilled(t *testing.T) {
	// b0 high 15500; b1 low 15520 → gap up [15500,15520] size 20; never filled.
	bars := series([][4]float64{
		{15495, 15500, 15490, 15495},
		{15530, 15540, 15520, 15530},
		{15540, 15550, 15525, 15540},
	})
	gaps := GapLevels(bars, 10, 1.0, time.UnixMilli(nowAfter(bars))) // minGap = 10
	if len(gaps) != 1 {
		t.Fatalf("want 1 unfilled gap, got %d (%v)", len(gaps), gaps)
	}
	if gaps[0].Kind != KindGap || gaps[0].Price != 15500 {
		t.Fatalf("gap fill target = %v want KindGap@15500", gaps[0])
	}
	if !strings.Contains(gaps[0].Info, "size 20") {
		t.Fatalf("gap size info = %q", gaps[0].Info)
	}
}

func TestGapLevelsFilled(t *testing.T) {
	// Same gap, but a later bar trades back to/through 15500 → filled → not emitted.
	bars := series([][4]float64{
		{15495, 15500, 15490, 15495},
		{15530, 15540, 15520, 15530},
		{15540, 15550, 15525, 15540},
		{15520, 15530, 15495, 15520}, // low 15495 fills the up-gap; high 15530 overlaps b2 (no new gap)
	})
	if got := GapLevels(bars, 10, 1.0, time.UnixMilli(nowAfter(bars))); len(got) != 0 {
		t.Fatalf("filled gap must not be emitted, got %v", got)
	}
}

func TestGapLevelsBelowThreshold(t *testing.T) {
	// A 20-pt gap but minGapATR×atr = 50 → below threshold → ignored.
	bars := series([][4]float64{
		{15495, 15500, 15490, 15495},
		{15530, 15540, 15520, 15530},
	})
	if got := GapLevels(bars, 50, 1.0, time.UnixMilli(nowAfter(bars))); len(got) != 0 {
		t.Fatalf("sub-threshold gap must be ignored, got %v", got)
	}
}

func TestOpeningRangeLevels(t *testing.T) {
	loc := chicago()
	reg := DefaultSessionRegistry()
	now := time.Date(2026, 8, 14, 10, 0, 0, 0, loc)
	bars := []market.Kline{
		barAt(loc, 2026, 8, 14, 8, 30, 15640, 15650, 15630, 15645), // OR + IB
		barAt(loc, 2026, 8, 14, 9, 0, 15650, 15680, 15620, 15660),  // IB only (09:00 ∉ OR window)
	}
	m := levelMap(OpeningRangeLevels(bars, reg, now))
	if m[KindORH] != 15650 || m[KindORL] != 15630 {
		t.Fatalf("OR = %v/%v want 15650/15630", m[KindORH], m[KindORL])
	}
	// IB-H 15680, IB-L 15620, range 60. levelMap keeps the last-seen per Kind,
	// so verify the extension prices via a full scan.
	prices := map[float64]bool{}
	for _, l := range OpeningRangeLevels(bars, reg, now) {
		prices[l.Price] = true
	}
	for _, want := range []float64{15680, 15620, 15710, 15740, 15590, 15560} {
		if !prices[want] {
			t.Fatalf("missing IB level %.0f in %v", want, prices)
		}
	}
}

func TestOpeningRangePreOpen(t *testing.T) {
	loc := chicago()
	reg := DefaultSessionRegistry()
	now := time.Date(2026, 8, 14, 7, 0, 0, 0, loc) // before 08:30 open
	bars := []market.Kline{barAt(loc, 2026, 8, 14, 4, 0, 15600, 15610, 15590, 15600)}
	if got := OpeningRangeLevels(bars, reg, now); got != nil {
		t.Fatalf("pre-open should return nil, got %v", got)
	}
}
