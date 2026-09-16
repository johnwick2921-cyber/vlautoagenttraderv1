package kernel

import (
	"testing"
	"time"

	"nofx/market"
)

// ENTRY-MECHANICS E8 (2026-08-30) — shadow A/B counterfactual fixtures.
// R2: the TEST recomputes the expected fill prices independently from the
// raw bars; the production rows must match.

func abTape(closes []float64, highOff, lowOff float64) ([]market.Kline, int64) {
	start := time.Date(2026, 8, 28, 9, 0, 0, 0, time.Local)
	bars := make([]market.Kline, 0, len(closes))
	for i, cl := range closes {
		at := start.Add(time.Duration(i) * time.Minute)
		bars = append(bars, market.Kline{OpenTime: at.UnixMilli(), CloseTime: at.UnixMilli() + 59_999,
			Open: cl, High: cl + highOff, Low: cl - lowOff, Close: cl})
	}
	return bars, bars[len(bars)-1].CloseTime + 1
}

func abScenario() PlanScenario {
	return PlanScenario{
		ID: "S1", Condition: "reject", Direction: "long",
		Confirm: &PlanConfirm{Rule: "touch", RefPrice: 100, Side: "above"},
		Arm:     &PlanArmSpec{Enabled: true, Entry: 100, Stop: 95, Target: 110},
	}
}

func TestShadowABTouchAndCloseFills(t *testing.T) {
	// 5m buckets: 99×5 → close 99; 101×5 → close 101 (1x5m fill); 102×5 →
	// close 102 (2x5m fill). Bar 1 straddles 100 (touch fill at 100). Then a
	// rally to the target.
	closes := []float64{99, 100.5, 99, 99, 99,
		101, 101, 101, 101, 101,
		102, 102, 102, 102, 102,
		104, 106, 108, 110, 110}
	bars, now := abTape(closes, 0.5, 0.5)
	rows := ShadowABForScenario(abScenario(), bars, "MNQ", bars[0].OpenTime-1, now)
	by := map[string]ShadowABRow{}
	for _, r := range rows {
		by[r.Rule] = r
	}
	// R2 independent recompute: touch fill = 100 at bar 1; 1x5m = 101 at the
	// first beyond 5m bucket; 2x5m = 102 at the second consecutive one.
	if by["touch"].FillPx != 100 {
		t.Fatalf("touch fill wrong: %+v", by["touch"])
	}
	if tt := by["touch"].TimeToFillMs; tt < 60_000 || tt > 60_002 {
		t.Fatalf("touch time-to-fill = %d, want ~60000", tt)
	}
	if by["1x5m_close"].FillPx != 101 || by["2x5m_close"].FillPx != 102 {
		t.Fatalf("close fills wrong: %+v %+v", by["1x5m_close"], by["2x5m_close"])
	}
	// Target hit → MFE = target − fill; MAE ≥ 0; outcome target.
	for _, rule := range []string{"touch", "1x5m_close", "2x5m_close"} {
		r := by[rule]
		if r.Outcome != "target" {
			t.Fatalf("%s outcome=%q, want target", rule, r.Outcome)
		}
		if r.MFE != 110-r.FillPx {
			t.Fatalf("%s MFE=%.2f, want %.2f", rule, r.MFE, 110-r.FillPx)
		}
		if r.MAE < 0 {
			t.Fatalf("%s MAE negative: %.2f", rule, r.MAE)
		}
	}
}

func TestShadowABStopOutResolvesAgainstTrade(t *testing.T) {
	// The same fills, then a collapse through the stop (95) — every row must
	// end in "stop" with MAE = fill − 95.
	closes := []float64{99, 100.5, 99, 99, 99,
		101, 101, 101, 101, 101,
		102, 102, 102, 102, 102,
		98, 96, 94, 93, 93}
	bars, now := abTape(closes, 0.5, 0.5)
	rows := ShadowABForScenario(abScenario(), bars, "MNQ", bars[0].OpenTime-1, now)
	if len(rows) < 3 {
		t.Fatalf("rows=%d want ≥3", len(rows))
	}
	for _, r := range rows {
		if r.Outcome != "stop" {
			t.Fatalf("%s outcome=%q, want stop", r.Rule, r.Outcome)
		}
		if r.MAE != r.FillPx-95 {
			t.Fatalf("%s MAE=%.2f, want %.2f", r.Rule, r.MAE, r.FillPx-95)
		}
	}
}

// TestShadowABZeroRealEffect — the E8 law: the computation is pure and
// deterministic; two calls return identical rows and mutate nothing.
func TestShadowABZeroRealEffect(t *testing.T) {
	sc := abScenario()
	closes := []float64{99, 100.5, 99, 99, 99, 101, 101, 101, 101, 101, 102, 102, 102, 102, 102, 104, 106, 108, 110, 110}
	bars, now := abTape(closes, 0.5, 0.5)
	a := ShadowABForScenario(sc, bars, "MNQ", bars[0].OpenTime-1, now)
	b := ShadowABForScenario(sc, bars, "MNQ", bars[0].OpenTime-1, now)
	if len(a) != len(b) {
		t.Fatalf("nondeterministic: %d vs %d rows", len(a), len(b))
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("row %d differs between calls: %+v vs %+v", i, a[i], b[i])
		}
	}
	// The scenario object is untouched (zero side effects).
	if sc.Confirm.Rule != "touch" || sc.Arm.Entry != 100 {
		t.Fatal("the computation mutated the scenario")
	}
}

// TestShadowABNoArmNoRows — a non-armed scenario has no bracket to replay.
func TestShadowABNoArmNoRows(t *testing.T) {
	sc := abScenario()
	sc.Arm = nil
	if rows := ShadowABForScenario(sc, nil, "MNQ", 0, 0); rows != nil {
		t.Fatalf("non-armed scenario must produce no rows, got %+v", rows)
	}
}

func TestShadowABMSSFill(t *testing.T) {
	// Mirror of the MSS tape shape: quiet base, swing high 100.9, break close
	// 101.5 → the 1m_mss row fills at 101.5.
	start := time.Date(2026, 8, 28, 8, 0, 0, 0, time.Local)
	var bars []market.Kline
	n := 80
	for i := 0; i < n; i++ {
		cl, h, l := 100.0, 100.1, 99.9
		if i == 72 {
			h = 100.9
		}
		if i == n-1 {
			cl, h, l = 101.5, 101.6, 101.4
		}
		at := start.Add(time.Duration(i) * time.Minute)
		bars = append(bars, market.Kline{OpenTime: at.UnixMilli(), CloseTime: at.UnixMilli() + 59_999,
			Open: cl, High: h, Low: l, Close: cl})
	}
	sc := abScenario()
	sc.Confirm.Side = "above"
	now := bars[len(bars)-1].CloseTime + 1
	rows := ShadowABForScenario(sc, bars, "MNQ", bars[0].OpenTime-1, now)
	found := false
	for _, r := range rows {
		if r.Rule == "1m_mss" {
			found = true
			if r.FillPx != 101.5 {
				t.Fatalf("mss fill=%.2f, want 101.5", r.FillPx)
			}
		}
	}
	if !found {
		t.Fatalf("mss counterfactual missing: %+v", rows)
	}
}

// TestShadowABWindowCrossingFiveMBoundary — the 2026-08-30 cutover panic
// regression: a plan born ~3-4 bars ago, window crossing a 5m boundary, with a
// beyond-close in the SECOND bucket. The old i*5 mapping indexed w[5] on a
// 4-bar window → panic. Must return rows (or nil) and never panic.
func TestShadowABWindowCrossingFiveMBoundary(t *testing.T) {
	// Bars at :58,:59,:00,:01 — 4 bars spanning two 5m buckets (epoch-aligned).
	base := int64(1_700_000_000_000) - (int64(1_700_000_000_000) % 300_000) // 5m-aligned
	start := base - 2*60_000                                                // two bars BEFORE the boundary → bucket A
	bars := []market.Kline{
		// Bucket A closes 99 (NOT beyond ref 100) — bucket B closes 102 (beyond).
		{OpenTime: start, CloseTime: start + 59_999, Open: 99, High: 99.2, Low: 98.8, Close: 99},
		{OpenTime: start + 60_000, CloseTime: start + 119_999, Open: 99, High: 99.2, Low: 98.8, Close: 99},
		{OpenTime: base, CloseTime: base + 59_999, Open: 102, High: 102.2, Low: 101.8, Close: 102},
		{OpenTime: base + 60_000, CloseTime: base + 119_999, Open: 102, High: 102.2, Low: 101.8, Close: 102},
	}
	now := bars[len(bars)-1].CloseTime + 1
	sc := abScenario() // long, ref 100 above, arm stop 95 target 110
	sc.Confirm.Rule = "1x5m_close"
	rows := ShadowABForScenario(sc, bars, "MNQ", start-1, now)
	// The beyond-close is in the SECOND bucket — must not panic and must
	// produce the 1x5m row at fill 102.
	found := false
	for _, r := range rows {
		if r.Rule == "1x5m_close" {
			found = true
			if r.FillPx != 102 {
				t.Fatalf("1x5m fill = %.2f, want 102", r.FillPx)
			}
		}
		if r.Rule == "2x5m_close" {
			t.Fatalf("2x5m must not fire on one qualifying bucket: %+v", rows)
		}
	}
	if !found {
		t.Fatalf("1x5m row missing on the boundary-crossing window: %+v", rows)
	}
}
