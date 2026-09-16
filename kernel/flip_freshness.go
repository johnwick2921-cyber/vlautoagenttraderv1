package kernel

import (
	"os"
	"strconv"

	"nofx/market"
)

// G7 (regime wave, 2026-08-21) — the flip/death evaluator may never judge a
// state transition on provably stale bars. Today's London window (shift-day
// forensics §4) showed the evaluator's aggregated rule-TF series lagging
// 1.5–2h behind truth while flips/deaths kept firing (and, conversely, late
// flips went unnoticed). Per the research's fail-open spirit this is NOT an
// entry gate: a stale evaluation is SKIPPED (the transition neither fires nor
// clears) and the next fresh cycle evaluates normally.
//
// The cap is a config knob, no literal on the decision path:
// FLIP_EVAL_MAX_STALE_S (default 90).

const DefaultFlipEvalMaxStaleSeconds = 90

// DefaultFlipMinHoldMinutes (G3, regime wave 2026-08-21) — a freshly-written
// plan cannot flip BACK within this hold unless the death line is breached
// (death always wins — it is evaluated first). Prevents the double-flip chop
// a shift day invites.
const DefaultFlipMinHoldMinutes = 30

// FlipMinHoldMin resolves the hold window (env FLIP_MIN_HOLD_MIN, default 30).
func FlipMinHoldMin() int64 {
	if v := os.Getenv("FLIP_MIN_HOLD_MIN"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return int64(n)
		}
	}
	return DefaultFlipMinHoldMinutes
}

// FlipEvalMaxStaleMs resolves the staleness cap (env FLIP_EVAL_MAX_STALE_S,
// default 90s) as milliseconds.
func FlipEvalMaxStaleMs() int64 {
	if v := os.Getenv("FLIP_EVAL_MAX_STALE_S"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			return int64(n) * 1000
		}
	}
	return DefaultFlipEvalMaxStaleSeconds * 1000
}

// FlipEvalAge returns the age (ms) of the newest CLOSED bar on the condition's
// rule timeframe as derived from the supplied series — the SAME aggregation
// path the evaluator judges on (AcceptanceBars from the 1m cache), so the
// freshness verdict can never disagree with the bars being evaluated. ok=false
// when the series holds no closed bar.
func FlipEvalAge(bars []market.Kline, rule string, nowMs int64) (ageMs int64, ok bool) {
	judge := AcceptanceBars(bars, conditionRule(PlanCondition{Rule: rule}))
	for i := len(judge) - 1; i >= 0; i-- {
		if judge[i].CloseTime >= nowMs {
			continue // forming per the repo bar convention
		}
		return nowMs - judge[i].CloseTime, true
	}
	return 0, false
}

// FlipEvalAllowed is the G7 gate: allowed=false means the evaluation must be
// skipped this cycle (log flip_eval_skipped, never guess).
func FlipEvalAllowed(bars []market.Kline, rule string, nowMs int64) (allowed bool, ageMs int64, why string) {
	age, ok := FlipEvalAge(bars, rule, nowMs)
	if !ok {
		return false, 0, "no_closed_bars"
	}
	periodMs := int64(AcceptanceIntervalMinutes(conditionRule(PlanCondition{Rule: rule}))) * 60_000
	if age > periodMs+FlipEvalMaxStaleMs() {
		return false, age, "stale_bars"
	}
	return true, age, ""
}
