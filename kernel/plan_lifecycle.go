package kernel

import (
	"math"

	"nofx/market"
)

// P3.6 — plan lifecycle helpers (pure, Go-side; no plan mutation).

// ActivationWindowK is the default activation half-width in daily-ATR multiples.
const ActivationWindowK = 1.5

// ActivePlanLevels filters plan levels to those currently within k×dATR of price
// — the executor's live candidate set (the "activation window"). Levels beyond
// re-activate when price returns. Pure: it never mutates the plan/cache. k≤0 →
// default 1.5; dATR≤0 → all levels (can't scale).
func ActivePlanLevels(levels []PlanLevel, price, dATR, k float64) []PlanLevel {
	if dATR <= 0 || price <= 0 {
		return levels
	}
	if k <= 0 {
		k = ActivationWindowK
	}
	band := k * dATR
	out := make([]PlanLevel, 0, len(levels))
	for _, l := range levels {
		if math.Abs(l.Price-price) <= band {
			out = append(out, l)
		}
	}
	return out
}

// PlanIsDead reports whether the plan's thesis is spent: it has levels and EVERY
// one has been TOUCHED and then accepted through (consumed) — the evaluator-
// detected death signal that triggers a re-plan (P3.6). A level price never
// reached is NOT consumed (being consistently on one side of a distant level is
// not "acceptance through" it). A plan with no levels never dies this way.
func PlanIsDead(doc PlanDoc, bars []market.Kline, rule string, now int64) bool {
	if len(doc.Levels) == 0 {
		return false
	}
	for _, l := range doc.Levels {
		touched := levelTouched(bars, l.Price, now)
		consumed := !LevelStillValid(bars, l.Price, rule, now)
		if !(touched && consumed) {
			return false // this level is still in play → plan not dead
		}
	}
	return true // every level touched AND accepted through → dead
}

// levelTouched reports whether any closed bar's range bracketed the level.
func levelTouched(bars []market.Kline, level float64, now int64) bool {
	for i := range bars {
		b := bars[i]
		if b.CloseTime >= now {
			continue
		}
		if b.Low <= level && b.High >= level {
			return true
		}
	}
	return false
}
