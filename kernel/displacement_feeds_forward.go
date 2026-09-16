package kernel

import (
	"fmt"
	"sort"
	"strings"
)

// DISPLACEMENT FEEDS FORWARD (owner ruling 2026-09-03) — the waterfall floor is
// ENFORCED at write and was never STATED.
//
// `ValidateBreakdownContinueScenarios` refuses a breakdown_continue whose
// measured displacement is below BD_MIN_DISP_ATR × ATR5m. The author was told
// the rule's NAME in the entry law and never the number, and never which levels
// had any displacement at all — so it authored waterfall plays at levels the
// tape had merely chopped across. On 2026-09-03 that burned attempt 1 of two
// consecutive reads (00:07 S3, 0.00 pts against a 15.2 floor; 00:33 S2, 0.00
// against 21.5), both recovered only by a repair.
//
// Same shape as the class-45 void list: the number the model reads is produced
// by calling the VALIDATOR'S OWN function through a level-oriented entry point,
// over the SAME scope the validator uses. There is no second implementation to
// drift.

// LevelDisplacement is one seated level's measured displacement.
// Broken=false means the tape never delivered a run beyond it — the honest
// "none", which is not the same as a small number.
type LevelDisplacement struct {
	Price          float64
	Label          string
	Short          bool    // the side that delivered (true = price ran DOWN through it)
	Pts            float64 // BreakLegPts from the validator, 0 when never broken
	Broken         bool
	ImmediateShort bool
	Immediate      MinuteDisplacement
}

// displacementProbe asks the validator what one level's displacement is, by
// synthesising the scenario a planner would author there. Mirrors
// BreakdownLevelReclaimed — the same trick for the same reason.
func displacementProbe(level float64, short bool, scope VoidScope, nowMs int64) BreakdownState {
	cond := "breakup_continue"
	dir := "long"
	if short {
		cond, dir = "breakdown_continue", "short"
	}
	sc := PlanScenario{
		ID: "probe", Condition: cond, Direction: dir,
		Breakdown: &PlanBreakdownContinue{Level: level, EntryMode: "pullback"},
	}
	return BreakdownContinueState(sc, scope.Bars, scope.SinceMs, nowMs)
}

// ComputeLevelDisplacements measures every seated level, on whichever side the
// tape actually delivered. A level broken on neither side reports Broken=false
// and 0 points, and the renderer says "none — no break" for it.
func ComputeLevelDisplacements(levels []ScoredLevel, scope VoidScope, nowMs int64) []LevelDisplacement {
	out := make([]LevelDisplacement, 0, len(levels))
	seen := map[float64]bool{}
	for _, l := range levels {
		if l.Price <= 0 || seen[l.Price] {
			continue
		}
		seen[l.Price] = true
		row := LevelDisplacement{Price: l.Price, Label: l.Label}
		for _, short := range []bool{true, false} {
			st := displacementProbe(l.Price, short, scope, nowMs)
			// Leg1Met is the validator's own "the level was broken" verdict;
			// a bigger BreakLegPts on the delivering side wins.
			if st.Leg1Met && st.BreakLegPts > row.Pts {
				row.Short, row.Pts, row.Broken = short, st.BreakLegPts, true
			}
			minute := Evaluate1mDisplacement(scope.Bars, l.Price, short, scope.SinceMs, nowMs)
			if !st.Reclaimed && minute.Observed && (!row.Immediate.Observed || minute.Pts > row.Immediate.Pts) {
				row.ImmediateShort, row.Immediate = short, minute
			}
		}
		out = append(out, row)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Price < out[j].Price })
	return out
}

// RenderDisplacementFloorLine states the floor the validator will judge by.
// Empty when the ATR is unknown: no reading, no claim.
func RenderDisplacementFloorLine(atr5m float64) string {
	if atr5m <= 0 {
		return ""
	}
	mult := bdMinDispATR()
	return fmt.Sprintf("## Waterfall displacement floor this cycle\n%.1f pts (%.1f×ATR5m %.2f, resolved). A breakdown_continue / breakup_continue authored at a level whose MEASURED displacement is below this is REFUSED at write.\n\n",
		mult*atr5m, mult, atr5m)
}

// RenderDisplacementLines is the per-level block. Levels the tape never broke
// say so in words — a level with no break is not a level with a small number.
func RenderDisplacementLines(rows []LevelDisplacement, atr5m float64) string {
	if len(rows) == 0 || atr5m <= 0 {
		return ""
	}
	floor := bdMinDispATR() * atr5m
	var b strings.Builder
	fmt.Fprintf(&b, "## Measured displacement per level (floor %.1f pts)\n", floor)
	for _, r := range rows {
		label := r.Label
		if label == "" {
			label = "level"
		}
		if !r.Broken {
			fmt.Fprintf(&b, "  %.2f %s — none — no break\n", r.Price, label)
			renderImmediateDisplacement(&b, r, floor)
			continue
		}
		side := "up"
		if r.Short {
			side = "down"
		}
		verdict := "BELOW the floor — closed-5m displacement does not permit pullback authoring"
		if r.Pts >= floor {
			verdict = "at or above the floor — closed-5m displacement permits pullback authoring"
		}
		fmt.Fprintf(&b, "  %.2f %s — %.2f pts %s · %s\n", r.Price, label, r.Pts, side, verdict)
		renderImmediateDisplacement(&b, r, floor)
	}
	b.WriteString("\n")
	return b.String()
}

func renderImmediateDisplacement(b *strings.Builder, r LevelDisplacement, floor float64) {
	if !r.Immediate.Observed {
		return
	}
	side := "up"
	if r.ImmediateShort {
		side = "down"
	}
	verdict := "BELOW the immediate-mode displacement floor"
	if r.Immediate.Pts >= floor {
		verdict = "meets the immediate-mode displacement floor"
	}
	fmt.Fprintf(b, "    %s: %.2f pts %s · %s; 5m confirmation and void checked separately\n", r.Immediate.Rule, r.Immediate.Pts, side, verdict)
}
