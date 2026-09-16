package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
)

// S4 (mega-research 2026-08-26) — nPOC dedupe + ±1-tick retire tolerance.
// One POC = at most one seat; a graze inside ±1 tick keeps it naked; a bar
// from a much later session (fed via the bars-table leg) retires it.

func npocBar(t int64, hi, lo float64) market.Kline {
	return market.Kline{OpenTime: t, CloseTime: t + 59_999, Open: lo, High: hi, Low: lo, Close: (hi + lo) / 2}
}

func TestDedupeSameKind(t *testing.T) {
	levels := []DetectedLevel{
		{Kind: KindNPOC, Price: 29250.00, Label: "nPOC·2026-08-25"},
		{Kind: KindNPOC, Price: 29250.10, Label: "nPOC·2026-08-25"}, // store-fed twin within 1 tick
		{Kind: KindNPOC, Price: 29100.00, Label: "nPOC·2026-08-24"},
		{Kind: KindVWAP, Price: 29250.00, Label: "VWAP"}, // different kind survives
	}
	out := dedupeSameKind(levels)
	if len(out) != 3 {
		t.Fatalf("dedupe must collapse the nPOC twin: got %d (%+v)", len(out), out)
	}
}

func TestNPOCRetireTickTolerance(t *testing.T) {
	now := time.Date(2026, 8, 26, 19, 0, 0, 0, CTLocation()) // 19:00 CT → session-day 08-26
	// A prior-day POC at 100; a later-session bar grazing within ±1 tick (0.25)
	// must NOT retire it; a bar bracketing beyond ±1 tick must.
	pocs := []PriorPOC{{SessionDate: "2026-08-25", POC: 100.0}}
	// Bars in the NEXT session-day (08-26, ≥17:00 CT on 08-25); range
	// [99.90, 100.10] stays inside ±1 tick.
	grazing := []market.Kline{
		npocBar(now.Add(-2*time.Hour).UnixMilli(), 100.10, 99.90),
		npocBar(now.Add(-90*time.Minute).UnixMilli(), 100.10, 99.90),
	}
	if got := NakedPOCs(pocs, grazing, now); len(got) != 1 {
		t.Fatalf("±1-tick graze must keep the POC naked, got %d levels", len(got))
	}
	through := append(grazing, npocBar(now.Add(-time.Hour).UnixMilli(), 100.60, 99.40))
	if got := NakedPOCs(pocs, through, now); len(got) != 0 {
		t.Fatalf("bracket beyond ±1 tick must retire, got %d levels", len(got))
	}
}

// TestNPOCBarsTableLegRetires: a POC from 3 sessions ago touched on day 2 of
// the gap retires — the historical leg the 2000-bar slice could never see.
func TestNPOCBarsTableLegRetires(t *testing.T) {
	now := time.Date(2026, 8, 26, 19, 0, 0, 0, CTLocation())    // session-day 08-26
	pocs := []PriorPOC{{SessionDate: "2026-08-22", POC: 100.0}} // 4 sessions back
	// Combined series: the old bars (bars-table leg) show the touch on
	// session-day 08-24; the recent slice has no touch.
	day3Touch := npocBar(now.AddDate(0, 0, -2).Add(-5*time.Hour).UnixMilli(), 100.50, 99.50)
	recent := []market.Kline{
		npocBar(now.Add(-2*time.Hour).UnixMilli(), 101.00, 100.80),
		npocBar(now.Add(-time.Hour).UnixMilli(), 101.10, 100.90),
	}
	combined := append([]market.Kline{day3Touch}, recent...)
	if got := NakedPOCs(pocs, combined, now); len(got) != 0 {
		t.Fatalf("touch on day 3 (bars-table leg) must retire the POC, got %d", len(got))
	}
	// Same but without the historical leg → the POC leaks (the register bug).
	if got := NakedPOCs(pocs, recent, now); len(got) != 1 {
		t.Fatalf("without the historical leg the scan is blind (sanity of the premise), got %d", len(got))
	}
	if !strings.HasPrefix(NakedPOCs(pocs, recent, now)[0].Label, "nPOC·2026-08-22") {
		t.Fatalf("label must carry the full YYYY-MM-DD date")
	}
}
