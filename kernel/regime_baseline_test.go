package kernel

import (
	"math"
	"testing"

	"nofx/market"
)

// synth5m builds `days`×288 5m bars starting at a fixed epoch, with a deterministic
// oscillating close (fixed amplitude → non-zero, stable realized vol; no rand/now).
func synth5m(days int) []market.Kline {
	const base = int64(1_704_400_000_000) // fixed epoch (ms), Jan 2024-ish
	n := days * 288
	bars := make([]market.Kline, 0, n)
	px := 18000.0
	for i := 0; i < n; i++ {
		// small deterministic zig-zag → consistent 5m returns.
		if i%2 == 0 {
			px *= 1.0006
		} else {
			px *= 0.9994
		}
		t := base + int64(i)*300_000
		bars = append(bars, market.Kline{
			OpenTime: t, Open: px, High: px * 1.001, Low: px * 0.999, Close: px, CloseTime: t + 299_999,
		})
	}
	return bars
}

// W10 — RVBaselineFrom5m computes a rolling per-day baseline from stored 5m bars
// once enough complete session-days exist, and reports warming below that.
func TestW10RVBaselineFrom5m(t *testing.T) {
	// too little history (~1 day) → warming.
	if _, ok := RVBaselineFrom5m(synth5m(1), 20, 5); ok {
		t.Fatal("one day of 5m must warm (no baseline)")
	}

	// plenty of complete days → a real baseline.
	baseline, ok := RVBaselineFrom5m(synth5m(8), 20, 5)
	if !ok {
		t.Fatal("8 complete days must yield a baseline")
	}
	if baseline <= 0 || math.IsNaN(baseline) {
		t.Fatalf("baseline must be a positive RV%%, got %v", baseline)
	}

	// empty / nil → warming, never a fake number.
	if _, ok := RVBaselineFrom5m(nil, 20, 5); ok {
		t.Fatal("nil bars must warm")
	}
}

// W10 — once the baseline is supplied, ComputeRegime reports RV as "% of normal"
// (warming cleared); with no baseline it stays honestly warming. VIX stays n/a.
func TestW10RegimeConsumesBaseline(t *testing.T) {
	min5 := synth5m(1) // ~1 day of recent 5m → recent RV computes
	baseline, ok := RVBaselineFrom5m(synth5m(8), 20, 5)
	if !ok {
		t.Fatal("expected a baseline from 8 days")
	}

	warm := ComputeRegime(RegimeInputs{Price: 18000, Min5Bars: min5}) // no baseline
	if !warm.RVWarming {
		t.Fatal("no baseline supplied → RV must be warming")
	}

	fed := ComputeRegime(RegimeInputs{Price: 18000, Min5Bars: min5, RVBaseline20d: baseline})
	if fed.RVWarming {
		t.Fatal("baseline supplied → RV must NOT be warming")
	}
	if fed.RealizedVolPct <= 0 {
		t.Fatalf("RV %%-of-normal must be positive, got %.2f", fed.RealizedVolPct)
	}
	// VIX stays honest n/a (no feed) in both cases.
	if fed.VIXRegime != "unavailable" || fed.VIXLevel != 0 {
		t.Fatal("VIX must remain honest n/a (no feed)")
	}
}
