package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
)

func TestNakedPOCs(t *testing.T) {
	loc := chicago()
	now := time.Date(2026, 8, 14, 11, 0, 0, 0, loc) // futures session-day 08-13

	pocs := []PriorPOC{
		{SessionDate: "2026-08-11", POC: 15500, Weekly: true},  // untested → naked (weekly)
		{SessionDate: "2026-08-12", POC: 15400, Weekly: false}, // bracketed by a later session → tested
		{SessionDate: "2026-08-13", POC: 15700, Weekly: false}, // untested → naked
	}
	// One closed bar in session-day 08-13 that brackets 15400 (the 08-12 POC).
	bars := []market.Kline{
		barAt(loc, 2026, 8, 14, 10, 0, 15400, 15410, 15390, 15400),
	}

	out := NakedPOCs(pocs, bars, now)
	m := map[float64]DetectedLevel{}
	for _, l := range out {
		if l.Kind != KindNPOC {
			t.Fatalf("expected KindNPOC, got %s", l.Kind)
		}
		m[l.Price] = l
	}
	if len(out) != 2 {
		t.Fatalf("want 2 naked POCs (15400 tested-out), got %d: %v", len(out), out)
	}
	if _, ok := m[15400]; ok {
		t.Fatalf("15400 was bracketed by a later session → must NOT be naked")
	}
	wk, ok := m[15500]
	if !ok || !wk.HTF || !strings.Contains(wk.Label, "wk") {
		t.Fatalf("15500 should be a weekly (HTF) naked POC: %+v", wk)
	}
	daily, ok := m[15700]
	if !ok || daily.HTF {
		t.Fatalf("15700 should be a daily naked POC (not HTF): %+v", daily)
	}
}
