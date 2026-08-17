package kernel

import (
	"fmt"
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
//
// Deprecated in favour of PlanIsDeadSince: judging a plan against the WHOLE bar
// cache asks "was this level ever traded through in the last ~33 hours", which is
// true of almost any level the moment real history exists. Retained so existing
// callers/tests keep compiling; it passes sinceMs=0 (no lower bound).
func PlanIsDead(doc PlanDoc, bars []market.Kline, rule string, now int64) bool {
	return PlanIsDeadSince(doc, bars, rule, 0, now)
}

// PlanIsDeadSince is PlanIsDead judged ONLY on bars that closed at/after sinceMs
// — normally the moment the plan was written.
//
// THE BUG THIS FIXES (2026-08-17): the caller fed the full 2000×1m cache (~33
// hours), so every level inside the last day and a half's range counted as both
// touched and accepted-through the instant a plan was born. 2026-08-16:ASIA died
// five times in a row that way — v1..v5 each held 6 perfectly good levels
// (EQL 30199.5 / ONH 30203 / ONL 30166.25 …) clustered in a ~57pt band that the
// session had obviously traded through earlier — until the re-plan budget ran out
// and writeNoTradePlan stored a levels:null NO-TRADE plan. The card then showed
// "No levels in this plan", which is what the owner saw on "every version".
//
// It is latent whenever the cache is thin and fires the moment real history
// arrives, which is exactly what happened when the weekend bar-starvation
// (AddOn watchdog livelock, 7aa521a1) ended and 2000 bars/TF were seeded.
//
// A plan can only be invalidated by what the market did AFTER it was written.
func PlanIsDeadSince(doc PlanDoc, bars []market.Kline, rule string, sinceMs, now int64) bool {
	if len(doc.Levels) == 0 {
		return false
	}
	if sinceMs > 0 {
		bars = barsSince(bars, sinceMs)
		// Too little post-plan history to judge — a plan is never dead on no evidence.
		if len(bars) == 0 {
			return false
		}
	}
	// SECOND ROOT CAUSE, same P0: acceptance must be counted on the timeframe the
	// rule names. ClosesBeyond counts BARS, not minutes, and the series handed in
	// is the 1-minute SVP cache — so "2x5m" (two 5-minute closes = 10 minutes of
	// acceptance) was being decided by two ONE-minute closes. Measured at v5's real
	// death check: every level in the stack read "consumed" while price sat at
	// 30193.50, inside the stack — PDL had 997 consecutive closes above it, EQL
	// 1275 below. That is a proximity test, not a consumption test: any level more
	// than two 1m closes away from price is automatically "accepted through".
	judge := AcceptanceBars(bars, rule)
	for _, l := range doc.Levels {
		touched := levelTouched(judge, l.Price, now)
		consumed := !LevelStillValidOn(judge, l.Price, rule, now)
		if !(touched && consumed) {
			return false // this level is still in play → plan not dead
		}
	}
	return true // every level touched AND accepted through → dead
}

// PlanDeathDetail names WHAT killed a plan, so a death can never again be a bare
// "DIED" line. 2026-08-16 lost a whole session to six deaths whose only record
// was five identical log lines carrying no condition and no price.
type PlanDeathDetail struct {
	// Killer is the human-readable condition, e.g.
	// "all 6 levels touched and accepted through (last: ONH 30203.00)".
	Killer string
	// Price is the latest closed price at the moment of the check.
	Price float64
	// Levels lists each level with the evidence that consumed it.
	Levels []string
}

// DescribePlanDeath explains a death decision using the SAME window and timeframe
// the decision itself used, so the explanation can never disagree with the
// verdict. Returns ok=false when the plan is not dead.
func DescribePlanDeath(doc PlanDoc, bars []market.Kline, rule string, sinceMs, now int64) (PlanDeathDetail, bool) {
	if !PlanIsDeadSince(doc, bars, rule, sinceMs, now) {
		return PlanDeathDetail{}, false
	}
	judge := AcceptanceBars(barsSince(bars, sinceMs), rule)
	price, _ := latestClosedClose(judge, now)
	d := PlanDeathDetail{Price: price}
	last := ""
	for _, l := range doc.Levels {
		f := EvaluateLevelFacts(judge, l.Price, DirAbove, rule, 3, now)
		up, dn := f.ClosesBeyondUp, f.ClosesBeyondDown
		side, n := "above", up
		if dn > up {
			side, n = "below", dn
		}
		d.Levels = append(d.Levels, fmt.Sprintf("%s %.2f accepted %s (%d× %dm closes)",
			l.Label, l.Price, side, n, AcceptanceIntervalMinutes(rule)))
		last = fmt.Sprintf("%s %.2f", l.Label, l.Price)
	}
	d.Killer = fmt.Sprintf("all %d levels touched and accepted through on %s bars (last: %s); price %.2f",
		len(doc.Levels), rule, last, price)
	return d, true
}

// barsSince returns the bars whose OPEN time is at/after sinceMs — the window a
// plan may legitimately be judged on. Ascending order is preserved.
func barsSince(bars []market.Kline, sinceMs int64) []market.Kline {
	for i := range bars {
		if bars[i].OpenTime >= sinceMs {
			return bars[i:]
		}
	}
	return nil
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
