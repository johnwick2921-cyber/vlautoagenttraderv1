package kernel

import (
	"math"
	"testing"

	"nofx/market"
)

// T9 (2026-08-27) — the σ math is PROVEN correct: volume-weighted standard
// deviation of typical prices. Hand-calc pins: tp = 100/102/104, vol 10 each →
// VWAP = 102, σ = sqrt((4+0+4)·10/30) = 1.63299… The master-recheck's "87pt
// implied σ" was a wrong-window artifact (session-day VWAP spans 17:00 → read,
// legitimately ~60-90pt σ on the 08-26 range day), not an accumulation bug.
func TestT9VwapStdevHandCalc(t *testing.T) {
	bars := []market.Kline{
		{High: 100, Low: 100, Close: 100, Volume: 10},
		{High: 102, Low: 102, Close: 102, Volume: 10},
		{High: 104, Low: 104, Close: 104, Volume: 10},
	}
	vwap, sd := vwapAndStdev(bars)
	if math.Abs(vwap-102) > 0.001 {
		t.Fatalf("VWAP = %.4f, want 102", vwap)
	}
	if math.Abs(sd-1.632993) > 0.001 {
		t.Fatalf("σ = %.4f, want 1.632993", sd)
	}
	// Uneven volume must tilt the VWAP toward the heavier bar.
	bars2 := []market.Kline{
		{High: 100, Low: 100, Close: 100, Volume: 1},
		{High: 110, Low: 110, Close: 110, Volume: 9},
	}
	vwap2, _ := vwapAndStdev(bars2)
	if math.Abs(vwap2-109) > 0.001 {
		t.Fatalf("weighted VWAP = %.4f, want 109", vwap2)
	}
}
