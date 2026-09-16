package kernel

import (
	"strings"
	"testing"

	"nofx/market"
)

// DISPLACEMENT FEEDS FORWARD (owner ruling 2026-09-03) — the waterfall floor
// is enforced at write and was never stated, so the author could not know which
// levels were eligible.
//
// Two reads on 2026-09-03 burned attempt 1 on exactly this:
//   00:07  S3 breakdown_continue: measured displacement 0.00 pts < BD_MIN_DISP_ATR 1.0×ATR5m (15.2 pts)
//   00:33  S2 breakdown_continue: measured displacement 0.00 pts < BD_MIN_DISP_ATR 1.0×ATR5m (21.5 pts)
// Both repaired on attempt 2. The model was not shown the number it was judged
// by, nor which levels had any displacement at all.

// chopBars is a tape that CROSSES a level repeatedly without ever delivering:
// the level is touched from both sides and no run beyond it ever extends. This
// is the 0.00-displacement shape.
func chopBars(level float64, n int) []market.Kline {
	var out []market.Kline
	for i := 0; i < n; i++ {
		open := int64(1_000_000 + i*60_000)
		hi, lo := level+2, level-2
		c := level + 1
		if i%2 == 1 {
			c = level - 1
		}
		out = append(out, market.Kline{OpenTime: open, CloseTime: open + 59_999, Open: level, High: hi, Low: lo, Close: c})
	}
	return out
}

// runBars delivers: a clean run away from the level and no close back across.
func runBars(level float64, n int, short bool) []market.Kline {
	var out []market.Kline
	px := level
	for i := 0; i < n; i++ {
		open := int64(1_000_000 + i*60_000)
		if short {
			px -= 6
			out = append(out, market.Kline{OpenTime: open, CloseTime: open + 59_999, Open: px + 6, High: px + 6, Low: px - 1, Close: px})
		} else {
			px += 6
			out = append(out, market.Kline{OpenTime: open, CloseTime: open + 59_999, Open: px - 6, High: px + 1, Low: px - 6, Close: px})
		}
	}
	return out
}

// THE PIN — the 00:07 S3 shape. A level the tape chopped across has 0.00
// measured displacement, and the facts block must say so by name.
func TestDisplacementFeedsForwardChoppedLevelReadsNone(t *testing.T) {
	const level = 29100.0
	bars := chopBars(level, 30)
	nowMs := bars[len(bars)-1].CloseTime

	got := ComputeLevelDisplacements([]ScoredLevel{{DetectedLevel: DetectedLevel{Price: level, Label: "PDL"}}},
		VoidScope{Bars: bars, SinceMs: bars[0].OpenTime}, nowMs)
	if len(got) != 1 {
		t.Fatalf("want one level row, got %d", len(got))
	}
	if got[0].Broken {
		t.Errorf("a chopped level was never broken — got Broken=true with %.2f pts", got[0].Pts)
	}
	if got[0].Pts != 0 {
		t.Errorf("measured displacement = %.2f, want 0.00 — this is the 00:07 S3 case", got[0].Pts)
	}

	line := RenderDisplacementLines(got, 15.2)
	if !strings.Contains(line, "none — no break") {
		t.Errorf("a level with no break must say so verbatim:\n%s", line)
	}
	if !strings.Contains(line, "15.2 pts") {
		t.Errorf("the floor must be stated with the levels it judges:\n%s", line)
	}
}

// The floor line itself: resolved, never a literal, and it names its own basis.
func TestDisplacementFloorLineIsResolved(t *testing.T) {
	const atr5m = 15.2
	line := RenderDisplacementFloorLine(atr5m)
	want := []string{"Waterfall displacement floor this cycle", "15.2 pts", "1.0×ATR", "resolved"}
	for _, w := range want {
		if !strings.Contains(line, w) {
			t.Errorf("floor line missing %q:\n%s", w, line)
		}
	}
	if RenderDisplacementFloorLine(0) != "" {
		t.Error("no ATR → no claim; the line must be empty rather than print a zero floor")
	}
}

// A level that DID deliver reports its real number, on the side that delivered.
func TestDisplacementFeedsForwardMeasuresARealRun(t *testing.T) {
	const level = 29100.0
	bars := runBars(level, 8, true) // short: price runs down away from the level
	nowMs := bars[len(bars)-1].CloseTime

	got := ComputeLevelDisplacements([]ScoredLevel{{DetectedLevel: DetectedLevel{Price: level, Label: "PDL"}}},
		VoidScope{Bars: bars, SinceMs: bars[0].OpenTime}, nowMs)
	if len(got) != 1 || !got[0].Broken {
		t.Fatalf("a delivered run must read as broken: %+v", got)
	}
	if got[0].Pts <= 0 {
		t.Fatalf("measured displacement = %.2f, want a real number", got[0].Pts)
	}
	if !got[0].Short {
		t.Error("the delivering side is SHORT here (price ran down through the level)")
	}
	// and the number is the VALIDATOR's, not a second implementation
	sc := PlanScenario{ID: "probe", Condition: "breakdown_continue", Direction: "short",
		Breakdown: &PlanBreakdownContinue{Level: level, EntryMode: "pullback"}}
	st := BreakdownContinueState(sc, bars, bars[0].OpenTime, nowMs)
	if got[0].Pts != st.BreakLegPts {
		t.Errorf("fed-forward %.2f != validator %.2f — a second implementation has appeared", got[0].Pts, st.BreakLegPts)
	}
}
