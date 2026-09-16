package kernel

import (
	"math"
	"testing"
	"time"

	"nofx/market"
)

// T3 — the swing-point detector emits the recent fractal extremes on the 5m and
// 15m series. Synthetic 1m bars with a clear 5m peak and trough: the emitted
// SWG-H/SWG-L must sit within a tick of the true extremes.
func TestT3SwingPointLevelsDetectsFractals(t *testing.T) {
	now := time.Date(2026, 8, 27, 10, 0, 0, 0, CTLocation())
	start := now.Add(-180 * time.Minute) // 180 × 1m bars = 36 × 5m bars
	var bars []market.Kline
	price := 100.0
	peak, trough := math.Inf(-1), math.Inf(1)
	for i := 0; i < 180; i++ {
		t0 := start.Add(time.Duration(i) * time.Minute)
		// rise → peak → decline → trough → rise (both fractals interior)
		switch {
		case i < 45:
			price += 0.5
		case i < 90:
			price -= 0.5
		default:
			price += 0.5
		}
		o := price
		h := price + 1.0
		l := price - 1.0
		c := price
		bars = append(bars, market.Kline{OpenTime: t0.UnixMilli(), CloseTime: t0.Add(time.Minute).UnixMilli(),
			Open: o, High: h, Low: l, Close: c, Volume: 10})
		if h > peak {
			peak = h
		}
		if l < trough {
			trough = l
		}
	}
	lv := SwingPointLevels(bars, now)
	if len(lv) == 0 {
		t.Fatal("no swing levels emitted")
	}
	var gotH, gotL bool
	for _, l := range lv {
		switch l.Kind {
		case KindSWGH:
			// 5m series: the only interior fractal high is the first peak (the
			// final rise's high sits in the last 2 bars — not a fractal at k=2).
			if l.TF == "5m" && math.Abs(l.Price-123.5) > 1.5 {
				t.Fatalf("5m SWG-H at %.2f, want 123.5", l.Price)
			}
			gotH = true
		case KindSWGL:
			if l.TF == "5m" && math.Abs(l.Price-98.5) > 1.5 {
				t.Fatalf("5m SWG-L at %.2f, want 98.5", l.Price)
			}
			gotL = true
		}
	}
	if !gotH || !gotL {
		t.Fatalf("want both SWG-H and SWG-L, gotH=%v gotL=%v (levels=%v)", gotH, gotL, lv)
	}
}
