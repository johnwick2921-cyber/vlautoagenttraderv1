package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
)

// ENTRY-MECHANICS E6 (2026-08-30) — the time_hold confirm primitive: price
// must HOLD beyond its ref for ACCEPT_HOLD_MIN (default 10) minutes of 1m
// closes with no close back across. Twin fixtures.

func holdTape(closes []float64) ([]market.Kline, int64) {
	start := time.Date(2026, 8, 28, 9, 0, 0, 0, time.Local)
	bars := make([]market.Kline, 0, len(closes))
	for i, cl := range closes {
		at := start.Add(time.Duration(i) * time.Minute)
		bars = append(bars, market.Kline{OpenTime: at.UnixMilli(), CloseTime: at.UnixMilli() + 59_999,
			Open: cl, High: cl + 1, Low: cl - 1, Close: cl})
	}
	return bars, bars[len(bars)-1].CloseTime + 1
}

func TestTimeHoldMetAfterTenMinutes(t *testing.T) {
	// 12 consecutive 1m closes below 100 — the hold is MET at 10.
	closes := []float64{}
	for i := 0; i < 12; i++ {
		closes = append(closes, 99.0)
	}
	bars, now := holdTape(closes)
	c := PlanConfirm{Rule: "time_hold", RefPrice: 100, Side: "below"}
	v := EvaluateConfirm(c, bars, bars[0].OpenTime-1, now)
	if !v.Met {
		t.Fatalf("12 held minutes must be MET: %s", v.Detail)
	}
	if !strings.Contains(v.Detail, "12/10") {
		t.Fatalf("detail must state the run vs need: %s", v.Detail)
	}
}

func TestTimeHoldNineMinutesNotMet(t *testing.T) {
	closes := []float64{}
	for i := 0; i < 9; i++ {
		closes = append(closes, 99.0)
	}
	bars, now := holdTape(closes)
	v := EvaluateConfirm(PlanConfirm{Rule: "time_hold", RefPrice: 100, Side: "below"}, bars, bars[0].OpenTime-1, now)
	if v.Met {
		t.Fatalf("9 held minutes must NOT be MET: %s", v.Detail)
	}
}

func TestTimeHoldCloseBackResetsRun(t *testing.T) {
	// 6 held → one close BACK across → 6 held again: best run 6 < 10, never MET
	// (the run resets on any close back across, not summed).
	closes := []float64{}
	for i := 0; i < 6; i++ {
		closes = append(closes, 99.0)
	}
	closes = append(closes, 101.0) // close back across
	for i := 0; i < 6; i++ {
		closes = append(closes, 99.0)
	}
	bars, now := holdTape(closes)
	v := EvaluateConfirm(PlanConfirm{Rule: "time_hold", RefPrice: 100, Side: "below"}, bars, bars[0].OpenTime-1, now)
	if v.Met {
		t.Fatalf("a close back across must reset the run: %s", v.Detail)
	}
	if !strings.Contains(v.Detail, "6/10") {
		t.Fatalf("best run must read 6/10: %s", v.Detail)
	}
}

func TestTimeHoldClosedBarsOnly(t *testing.T) {
	// 11 closed holds + a still-FORMING 12th — the forming bar never counts.
	closes := []float64{}
	for i := 0; i < 12; i++ {
		closes = append(closes, 99.0)
	}
	bars, _ := holdTape(closes)
	now := bars[len(bars)-1].CloseTime // the last bar is still forming
	v := EvaluateConfirm(PlanConfirm{Rule: "time_hold", RefPrice: 100, Side: "below"}, bars, bars[0].OpenTime-1, now)
	if !v.Met {
		t.Fatalf("11 closed holds must still be MET: %s", v.Detail)
	}
	now = bars[10].CloseTime + 1 // 10 closed bars — exactly the floor
	v = EvaluateConfirm(PlanConfirm{Rule: "time_hold", RefPrice: 100, Side: "below"}, bars, bars[0].OpenTime-1, now)
	if !v.Met {
		t.Fatalf("exactly 10 closed holds must be MET: %s", v.Detail)
	}
	now = bars[8].CloseTime + 1 // 9 closed bars
	v = EvaluateConfirm(PlanConfirm{Rule: "time_hold", RefPrice: 100, Side: "below"}, bars, bars[0].OpenTime-1, now)
	if v.Met {
		t.Fatalf("9 closed holds must NOT be MET: %s", v.Detail)
	}
}

// TestTimeHoldInEntryLaw — acceptance/hold conditions accept time_hold; the
// validator routes it through the enum + prose check like every rule.
func TestTimeHoldInEntryLaw(t *testing.T) {
	d := validBaseDoc()
	d.Scenarios[0].Condition = "acceptance"
	d.Scenarios[0].Direction = "long"
	d.Scenarios[0].Trigger = "acceptance above 29648.25"
	d.Scenarios[0].Invalid = "invalid below 29648.25"
	d.Scenarios[0].Confirm = &PlanConfirm{Rule: "time_hold", RefPrice: 29648.25, Side: "above"}
	if err := ValidatePlanDocWithCaps(d, 8, 3); err != nil {
		t.Fatalf("time_hold on acceptance must validate: %v", err)
	}
}

// TestAcceptHoldMinDefault — the knob resolver's shipped value.
func TestAcceptHoldMinDefault(t *testing.T) {
	if AcceptHoldMin() != 10 {
		t.Fatalf("ACCEPT_HOLD_MIN default = %d, want 10", AcceptHoldMin())
	}
}
