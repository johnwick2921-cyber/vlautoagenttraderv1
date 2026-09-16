package kernel

import (
	"strings"
	"testing"
	"time"

	"nofx/market"
)

// TestTwoLegConfirmRenderS2Fixture — F2: the exact S2 10:54 artifact. Leg 1 MET
// (close below VWAP 29657.39) but the retest leg never fired — the render must
// show BOTH legs and an overall NOT MET, never a bare "MET".
func TestTwoLegConfirmRenderS2Fixture(t *testing.T) {
	start := time.Date(2026, 8, 28, 10, 25, 0, 0, time.Local)
	lvl := 29657.39
	bars := []market.Kline{}
	// 15 minutes above the level, then the breakdown: closes 29613.50, 29589.50
	seq := []float64{29780, 29775, 29783, 29779, 29772, 29778, 29763, 29759, 29733, 29721, 29697, 29689, 29671, 29662, 29646,
		29634, 29613.50, 29589.50, 29562.50, 29557.50, 29565.50, 29552.50, 29545.50, 29528.25, 29542.50, 29549.50, 29531.50, 29551.75, 29556.25, 29574.50}
	for i, c := range seq {
		bars = append(bars, mkTapeBar(start.Add(time.Duration(i)*time.Minute), c+0.25, c+1.5, c-1.5, c))
	}
	doc := PlanDoc{Scenarios: []PlanScenario{{
		ID: "S2", Trigger: "2x5m close below VWAP 29657.39 then retest fails",
		Condition: "breakout_retest", Direction: "short",
		Confirm:  &PlanConfirm{Rule: "1x5m_close", RefPrice: lvl, Side: "below"},
		Confirm2: &PlanConfirm{Rule: "touch", RefPrice: lvl, Side: "below"},
	}}}
	out := RenderConfirmLines(doc, bars, 0, bars[len(bars)-1].CloseTime, 29570, 15.0)
	if !strings.Contains(out, "leg 1/2") || !strings.Contains(out, "leg 2/2") {
		t.Fatalf("want both legs rendered, got:\n%s", out)
	}
	if !strings.Contains(out, "overall NOT MET") {
		t.Fatalf("partial must be overall NOT MET, got:\n%s", out)
	}
	if strings.Contains(out, "— MET (") || strings.Contains(out, "confirm: 1x 5m close below 29657.39 — MET") {
		t.Fatalf("a partial must never print as a bare MET, got:\n%s", out)
	}
	// Now the retest-that-fails: touch back to the level, then a 5m close below.
	ext := append(append([]market.Kline{}, bars...),
		mkTapeBar(start.Add(30*time.Minute), 29650, 29660, 29649, 29658), // touches
		mkTapeBar(start.Add(31*time.Minute), 29658, 29659, 29640, 29644), // fails
	)
	out2 := RenderConfirmLines(doc, ext, 0, ext[len(ext)-1].CloseTime, 29644, 15.0)
	if !strings.Contains(out2, "overall MET") {
		t.Fatalf("retest-fail must be overall MET, got:\n%s", out2)
	}
}

// TestBreakdownConfirmRenderTwoLeg — the waterfall class renders its machine
// legs (breakdown + retest-fail) and a partial never reads MET.
func TestBreakdownConfirmRenderTwoLeg(t *testing.T) {
	bars, start := waterfallTape(29657.39)
	plan := waterfallPlan()
	plan.Scenarios[0].Arm = nil // render path doesn't care about arms
	birth := start.Add(26 * time.Minute)
	out := RenderConfirmLines(plan, bars, birth.UnixMilli(), bars[len(bars)-1].CloseTime, 29500, 15.0)
	if !strings.Contains(out, "leg 1/2") || !strings.Contains(out, "leg 2/2") || !strings.Contains(out, "overall NOT MET") {
		t.Fatalf("waterfall two-leg render wrong:\n%s", out)
	}
}
