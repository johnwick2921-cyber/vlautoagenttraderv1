package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
)

// ENTRY-MECHANICS E5 (2026-08-30) — 1m-MSS confirm primitive fixtures: the
// last confirmed 1m fractal swing (k=2) broken by a qualifying 1m CLOSE.
// Closed bars only, wick never, direction-aware. Twin fixtures + an R2 twin
// (the test recomputes the swings INDEPENDENTLY of the production code).

// mssTape builds a quiet base (ATR-establishing) + a confirmed swing + a
// break bar as the LAST bar (so the break bar can never be a confirmed swing
// itself — a confirmed swing needs k=2 CLOSED bars after it).
func mssTape(swingIdx int, swingPrice float64, quietRange, tailClose, tailHigh float64, bullish bool) []market.Kline {
	start := time.Date(2026, 8, 28, 8, 0, 0, 0, time.Local)
	n := 70 + swingIdx + 3
	bars := make([]market.Kline, 0, n)
	for i := 0; i < n; i++ {
		cl := 100.0
		h, l := cl+quietRange/2, cl-quietRange/2
		at := start.Add(time.Duration(i) * time.Minute)
		if i == swingIdx {
			// the swing extreme (high or low), strictly beyond its k=2 neighbors
			if bullish {
				h = swingPrice
			} else {
				l = swingPrice
			}
		}
		if i == n-1 { // the break bar — the LAST bar, closed when now = CloseTime+1
			// tailHigh is the WICK EXTREME: the high for bullish tapes, the low
			// for bearish ones. OHLC stays valid either way.
			cl = tailClose
			if bullish {
				h, l = tailHigh, cl-quietRange/2
			} else {
				h, l = cl+quietRange/2, tailHigh
			}
		}
		bars = append(bars, market.Kline{OpenTime: at.UnixMilli(), CloseTime: at.UnixMilli() + 59_999,
			Open: cl, High: h, Low: l, Close: cl})
	}
	return bars
}

// mssBullishTape: quiet 70 bars (ATR base), swing HIGH at index 72, break bar
// at index 75.
func mssBullishTape(tailClose, tailHigh float64) []market.Kline {
	return mssTape(72, 100.9, 0.2, tailClose, tailHigh, true)
}

// mssBearishTape: mirror — swing LOW at index 72.
func mssBearishTape(tailClose, tailLow float64) []market.Kline {
	return mssTape(72, 99.1, 0.2, tailClose, tailLow, false)
}

func TestMSSBullishBreakMet(t *testing.T) {
	bars := mssBullishTape(101.5, 101.6) // close 0.6 beyond the 100.9 swing high
	now := bars[len(bars)-1].CloseTime + 1
	v := EvaluateMSS(bars, "above", now)
	if !v.Met {
		t.Fatalf("bullish MSS break must be MET: %s", v.Detail)
	}
	if v.SwingPrice != 100.9 || !v.SwingHigh {
		t.Fatalf("wrong swing: %+v", v)
	}
	if v.BreakClose != 101.5 {
		t.Fatalf("break close = %.2f, want 101.5", v.BreakClose)
	}
}

func TestMSSWickNeverCounts(t *testing.T) {
	// The tail bar WICKS beyond the swing (high 101.6) but CLOSES back at
	// 100.5 — a wick beyond the swing is NOT a break.
	bars := mssBullishTape(100.5, 101.6)
	now := bars[len(bars)-1].CloseTime + 1
	if v := EvaluateMSS(bars, "above", now); v.Met {
		t.Fatalf("a wick beyond the swing must NEVER count: %s", v.Detail)
	}
}

func TestMSSWeakDisplacementRejected(t *testing.T) {
	// Break close 101.0 → displacement 0.1 vs the swing 100.9. With quiet range
	// 0.2 the 5m ATR ≈ 0.2, threshold = 0.5×ATR ≈ 0.1 — borderline by
	// construction; widen the quiet range so the threshold clearly exceeds the
	// 0.1 displacement (0.5×0.4 = 0.2 > 0.1).
	bars := mssTape(72, 100.9, 0.4, 101.0, 101.05, true)
	now := bars[len(bars)-1].CloseTime + 1
	if v := EvaluateMSS(bars, "above", now); v.Met {
		t.Fatalf("weak displacement must not MET: %s", v.Detail)
	}
}

func TestMSSNoSwingNotMet(t *testing.T) {
	start := time.Date(2026, 8, 28, 8, 0, 0, 0, time.Local)
	var bars []market.Kline
	for i := 0; i < 4; i++ { // fewer than 2k+1 closed bars — no swing possible
		at := start.Add(time.Duration(i) * time.Minute)
		bars = append(bars, market.Kline{OpenTime: at.UnixMilli(), CloseTime: at.UnixMilli() + 59_999,
			Open: 100, High: 100.2, Low: 99.8, Close: 100})
	}
	if v := EvaluateMSS(bars, "above", bars[len(bars)-1].CloseTime+1); v.Met {
		t.Fatalf("no swing → never MET: %s", v.Detail)
	}
}

func TestMSSBearishMirrorMet(t *testing.T) {
	// Mirror tape: swing LOW 99.1 broken by a 1m close below it.
	bars := mssBearishTape(98.5, 98.4)
	now := bars[len(bars)-1].CloseTime + 1
	v := EvaluateMSS(bars, "below", now)
	if !v.Met {
		t.Fatalf("bearish MSS break must be MET: %s", v.Detail)
	}
	if v.SwingPrice != 99.1 || v.SwingHigh {
		t.Fatalf("wrong bearish swing: %+v", v)
	}
}

// TestMSSR2IndependentSwingRecompute — the R2 twin: the TEST recomputes the
// k=2 fractal swings from the raw bars with its own loop and asserts the
// production verdict used the SAME last confirmed swing.
func TestMSSR2IndependentSwingRecompute(t *testing.T) {
	bars := mssBullishTape(101.5, 101.6)
	now := bars[len(bars)-1].CloseTime + 1
	v := EvaluateMSS(bars, "above", now)

	// Independent recompute: closed bars, k=2, strict highs.
	const k = 2
	closed := make([]market.Kline, 0, len(bars))
	for _, b := range bars {
		if b.CloseTime < now {
			closed = append(closed, b)
		}
	}
	var lastHigh float64
	found := false
	for i := len(closed) - 1 - k; i >= k; i-- {
		ok := true
		for j := i - k; j <= i+k; j++ {
			if j == i {
				continue
			}
			if closed[j].High >= closed[i].High {
				ok = false
				break
			}
		}
		if ok {
			lastHigh, found = closed[i].High, true
			break
		}
	}
	if !found {
		t.Fatal("independent recompute found no swing — tape is wrong")
	}
	if lastHigh != v.SwingPrice {
		t.Fatalf("R2 mismatch: independent last swing high %.2f ≠ production %.2f", lastHigh, v.SwingPrice)
	}
	if !v.Met {
		t.Fatalf("R2: production must still be MET")
	}
}

// TestMSSInConfirmPipeline — the 1m_mss rule routes through EvaluateConfirm and
// renders the machine swing in the detail line.
func TestMSSInConfirmPipeline(t *testing.T) {
	bars := mssBullishTape(101.5, 101.6)
	now := bars[len(bars)-1].CloseTime + 1
	c := PlanConfirm{Rule: "1m_mss", RefPrice: 100.9, Side: "above"}
	v := EvaluateConfirm(c, bars, bars[0].OpenTime-1, now)
	if !v.Met {
		t.Fatalf("1m_mss through EvaluateConfirm must MET: %s", v.Detail)
	}
	if !strings.Contains(v.Detail, "swing") {
		t.Fatalf("detail must name the swing: %s", v.Detail)
	}
	// The authored ref_price must also be prose-anchored like every rule.
	d := validBaseDoc()
	d.Scenarios[0].Condition = "reclaim" // 1m_mss is legal on reclaim
	d.Scenarios[0].Trigger = "reclaim below 29648.25"
	d.Scenarios[0].Invalid = "invalid above 29648.25"
	d.Scenarios[0].Confirm = &PlanConfirm{Rule: "1m_mss", RefPrice: 29648.25, Side: "below"}
	if err := ValidatePlanDocWithCaps(d, 8, 3); err != nil {
		t.Fatalf("1m_mss confirm must validate on reclaim: %v", err)
	}
}
