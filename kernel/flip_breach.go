package kernel

import (
	"math"
	"strings"

	"nofx/market"
)

// W-FLIP-OWNS-THE-BREACH (2026-09-17, owner: "why at the flip point it
// re-reads and the bias is still the same") — CLASS 139 anchored the flip
// HOLD to the chain, but two things still restarted per version:
//
//   - the CONDITION WINDOW (touch gate + confirm closes) was the version's
//     birth, so a same-bias wake re-read that kept the flip line reset the
//     close count to zero;
//   - ordinary wakes (level_event / structure_mss) kept authoring while the
//     flip line was breached, and even on a tape the flip evaluator had just
//     refused as stale (2026-09-17 22:42:33 CT: flip_eval_skipped
//     flip=stale_bars (age 453s) and, the same second, "structure MSS …
//     waking the planner" — v2 was born of that read at 22:52:04).
//
// Three rules, no knobs:
//
//	R1 the flip owns the breach — while the active plan's flip line is
//	   breached (touched in-window and ≥1 rule-TF close beyond the buffered
//	   line, the same measurement PlanConditionFiredSince makes) and has not
//	   fired, ordinary wakes are DEFERRED. Scheduled reads, death re-plans and
//	   owner reads are untouched. Price closing back inside resumes wakes.
//	R2 a same-bias wake keeps the flip window — a new version whose flip line
//	   is within FlipLineClusterTolerance of the previous same-bias version's
//	   counts closes from the EARLIEST such version's birth (the chain
//	   anchor); a line moved further is a new line and windows from the
//	   version's birth, as before, and is logged as moved.
//	R3 stale bars block wakes too — when the flip evaluation is skipped (G7),
//	   ordinary wakes are deferred for the same reason.

// FlipConditionAnchor is the instant a version's flip CONDITION window opens
// (bars before it are never judged), with the kind of anchor that set it.
type FlipConditionAnchor struct {
	SinceMs       int64
	Source        string  // one of the FlipWindow* kinds
	AnchorVersion int     // the version whose birth SinceMs is
	Moved         bool    // the previous same-bias version carried a line further than the tolerance
	PrevPrice     float64 // that previous line, when Moved
}

const (
	FlipWindowChain    = "chain"             // earliest version of the same-bias, same-line run
	FlipWindowVersion  = "version"           // the version's own birth: first line of its run, a re-plan, or no chain to read
	FlipWindowMoved    = "version(moved)"    // the version's own birth because the wake MOVED the line
	FlipWindowFallback = "version(fallback)" // chain unreadable / version not in it
)

// FlipLineClusterTolerance is the distance within which two flip lines are
// the SAME line — the level map's cluster width (LevelClusterTicks × tick =
// 3.00 pt on MNQ), read from the one constant the map uses.
func FlipLineClusterTolerance() float64 { return clusterToleranceFor(0) }

// ResolveFlipConditionAnchor picks the flip condition window for version
// `current` from the chain's versions (ascending). Walking back from
// `current`, a previous version stays in the run while it carries the same
// bias, the same flip side and a flip price within tol of the current line;
// a deliberate re-plan version (death_replan / owner_reread / owner_reset)
// starts a run and stops the walk; a version with no flip line or a
// different bias breaks it. The anchor is the birth of the earliest run
// member. `current` itself being a re-plan, or missing from the chain, or
// carrying no line, anchors on its own birth.
func ResolveFlipConditionAnchor(versions []PlanVersionFact, current int, tol float64, fallbackMs int64) FlipConditionAnchor {
	idx := -1
	for i, v := range versions {
		if v.Version == current {
			idx = i
		}
	}
	if idx < 0 || versions[idx].CreatedAtMs <= 0 {
		return FlipConditionAnchor{SinceMs: fallbackMs, Source: FlipWindowFallback, AnchorVersion: current}
	}
	cur := versions[idx]
	out := FlipConditionAnchor{SinceMs: cur.CreatedAtMs, Source: FlipWindowVersion, AnchorVersion: cur.Version}
	if cur.FlipPrice <= 0 || flipHoldReplanTriggers[strings.TrimSpace(cur.TriggerReason)] {
		return out
	}
	if tol < 0 {
		tol = 0
	}
	sameLine := func(p PlanVersionFact) bool {
		return p.FlipPrice > 0 && p.FlipSide == cur.FlipSide &&
			p.BiasDirection != "" && p.BiasDirection == cur.BiasDirection &&
			math.Abs(p.FlipPrice-cur.FlipPrice) <= tol
	}
	// Moved: the immediately previous version kept the bias but carried a
	// line further than the tolerance (or on the other side).
	if idx > 0 {
		p := versions[idx-1]
		if p.FlipPrice > 0 && p.BiasDirection != "" && p.BiasDirection == cur.BiasDirection && !sameLine(p) {
			out.Source = FlipWindowMoved
			out.Moved = true
			out.PrevPrice = p.FlipPrice
			return out
		}
	}
	for j := idx - 1; j >= 0; j-- {
		p := versions[j]
		if p.CreatedAtMs <= 0 || !sameLine(p) {
			break
		}
		out = FlipConditionAnchor{SinceMs: p.CreatedAtMs, Source: FlipWindowChain, AnchorVersion: p.Version}
		if flipHoldReplanTriggers[strings.TrimSpace(p.TriggerReason)] {
			break // a deliberate fresh plan starts the run
		}
	}
	return out
}

// FlipBreach is what a wake asks about the active plan's flip line before it
// may author: is the line breached right now (the flip evaluator owns the
// plan), or was the evaluation skipped for stale bars (nobody may author on
// that tape)?
type FlipBreach struct {
	Breached bool // touched in-window AND ≥1 rule-TF close beyond the buffered line
	Stale    bool // G7: the rule-TF series is stale; the flip evaluator skipped
	StaleWhy string
	AgeMs    int64
	Side     string
	Price    float64 // raw line
	Line     float64 // buffered line the closes are judged against
	Closes   int     // consecutive closes beyond, so far
	Need     int     // closes the flip needs to fire
	Touched  bool
}

// FlipBreachState measures the flip line on the SAME window, buffer, touch
// gate and close count PlanConditionFiredSince fires on (conditionCloses), so
// a wake can only be deferred for a breach the evaluator can actually turn
// into a flip: a line born beyond price and never touched is NOT a breach —
// deferring wakes on it would park the plan forever behind a flip that
// cannot fire. sinceMs is the flip condition window (ResolveFlipConditionAnchor).
func FlipBreachState(c PlanCondition, bars []market.Kline, sinceMs, nowMs int64) FlipBreach {
	out := FlipBreach{Side: c.Side, Price: c.Price}
	if c.Price <= 0 {
		return out
	}
	if allowed, age, why := FlipEvalAllowed(bars, c.Rule, nowMs); !allowed {
		out.Stale, out.StaleWhy, out.AgeMs = true, why, age
		return out
	}
	f := conditionCloses(c, bars, sinceMs, nowMs)
	if !f.ok {
		return out
	}
	out.Touched, out.Line, out.Closes, out.Need = f.touched, f.line, f.closes, f.need
	out.Breached = f.touched && f.closes >= 1
	return out
}
