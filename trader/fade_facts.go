// ── W2 — THE FACTS BUILDER: what is knowable at `now`, and nothing else ──────
//
// FadePermissionAt is pure and takes its facts as an argument. This file is
// the ONLY place that assembles them for production, and it is where the
// research law bites: every field below must be computable from information
// available at `now`. Bars whose open time is at or after `now` are not read.
// The completed session is never consulted.
//
// PINNED BY PATH (class 113): the predicate's purity proves nothing about
// this builder. If this file read time.Now(), the stamp would record "now"
// instead of "then" and the predicate would stay provably pure. It does not.
package trader

import (
	"encoding/json"
	"sort"
	"time"

	"nofx/calendar"
	"nofx/kernel"
	"nofx/market"
)

// fadeORWideKDefault is C5's own-tape 80th percentile over the 13-session
// median: p80 104.05 / median 81.25 = 1.28. [I] — 13 sessions is a small n
// and 2026-09-07 (Labor Day, OR 33.75) is in it. It is a DEFAULT: the bound
// strategy's day_plan.fade_or_wide_k overrides it, and the boot line names
// which one is in force.
const fadeORWideKDefault = 1.28

// fadeORMedianSessions is how many prior sessions the OR median is taken over.
// The dispatch asked for 20; the 5m tape reaches 2026-08-24, so at this
// boot the store yields 13. The boot line prints the n it actually got.
const fadeORMedianSessions = 20

// fadeORHistoryProvider is the seam through which the prior-session OR median
// is read. A variable so a test can hand it a fixed history; production never
// reassigns it.
var fadeORHistoryProvider = func(at *AutoTrader, symbol string, before time.Time, n int) (median float64, got int) {
	if at == nil || at.store == nil {
		return 0, 0
	}
	return at.store.BarHistory().PriorSessionORMedian(symbol, before, n)
}

// fadeFactsAt assembles FadeFacts for one scenario evaluation. `now` is the
// evaluation moment — passed in, never read here. `direction` is the
// scenario's own side ("long"|"short"), which (c) needs to know which edge of
// the map matters.
func (at *AutoTrader) fadeFactsAt(now time.Time, symbol string, price float64, seated []kernel.ScoredLevel, direction string) kernel.FadeFacts {
	f := kernel.FadeFacts{Price: price, Direction: direction}
	loc := kernel.CTLocation()
	nowCT := now.In(loc)

	// SESSION OPEN — from the registry the gate uses, never a literal 08:30.
	reg := at.sessionRegistry(now)
	sess, ok := reg.ActiveSession(now)
	var open time.Time
	if ok && sess != nil && sess.WindowStartCT != "" {
		// hhmmToMin is the same parse the clock, cadence and cutoff paths use
		// — one parser, no bare time layout (class 38 / tz guard).
		if startMin, okS := hhmmToMin(sess.WindowStartCT); okS {
			open = time.Date(nowCT.Year(), nowCT.Month(), nowCT.Day(), startMin/60, startMin%60, 0, 0, loc)
			f.SessionOpen = open
			// (e) reuses the existing no-trade band's N (D1e) — never retyped.
			f.FirstNMinutes = kernel.FirstNoTradeMinutes()
		}
	}

	// (a) and (b) need 5m bars at or before now. Bars opening at/after now do
	// not exist yet from the evaluator's point of view and are dropped.
	if !open.IsZero() && market.FuturesBarsProvider != nil {
		bars := market.FuturesBarsProvider(symbol, "5m", 400)
		var sessBars []market.Kline
		for _, b := range bars {
			bt := time.UnixMilli(b.OpenTime)
			if bt.Before(open) || !bt.Before(now) {
				continue
			}
			// a 5m bar is CLOSED only once now >= open+5m
			if !bt.Add(5 * time.Minute).After(now) {
				sessBars = append(sessBars, b)
			}
		}
		sort.Slice(sessBars, func(i, j int) bool { return sessBars[i].OpenTime < sessBars[j].OpenTime })

		// (a) the OR is the FIRST closed 5m bar of the session.
		if len(sessBars) >= 1 && time.UnixMilli(sessBars[0].OpenTime).Equal(open) {
			f.ORComplete = true
			f.OR5mPts = sessBars[0].High - sessBars[0].Low
			f.ORMedianPts, f.ORMedianN = fadeORHistoryProvider(at, symbol, open, fadeORMedianSessions)
			f.ORWideK = at.fadeORWideK()
		}

		// (b) the IB is the first 12 closed 5m bars; CONTINUOUS thereafter.
		if len(sessBars) >= 12 {
			f.IBComplete = true
			hi, lo := sessBars[0].High, sessBars[0].Low
			for _, b := range sessBars[:12] {
				if b.High > hi {
					hi = b.High
				}
				if b.Low < lo {
					lo = b.Low
				}
			}
			f.IBHigh, f.IBLow = hi, lo
			// "held one closed 5m bucket beyond it" — the LAST closed bar
			// closed outside the IB. Confirmation-truth: a close, not a poke.
			last := sessBars[len(sessBars)-1]
			if len(sessBars) > 12 && (last.Close > hi || last.Close < lo) {
				f.ClosedBucketBeyondIB = true
			}
		}
	}

	// (c) the seated map, at this evaluation.
	if len(seated) > 0 {
		f.SeatedRefsKnown = true
		f.MaxSeated, f.MinSeated = seated[0].Price, seated[0].Price
		for _, s := range seated {
			if s.Price > f.MaxSeated {
				f.MaxSeated = s.Price
			}
			if s.Price < f.MinSeated {
				f.MinSeated = s.Price
			}
		}
	}

	// (d) T1 — UNKNOWN unless a REAL slice exists for the day. The gate's
	// static fail-closed fallback is right for refusing and wrong for a
	// label: "we guessed" must not be recorded as "we knew".
	if at.store != nil && ok && sess != nil {
		tradeDate := plannerTradeDateCT(now)
		if slice, err := at.store.Calendar().GetSlice(tradeDate); err == nil && slice != nil {
			var evs []calendar.Event
			if json.Unmarshal([]byte(slice.EventsJSON), &evs) == nil {
				f.CalendarHasSlice = true
				windows := kernel.T1BlackoutWindows(sessionPlannerEvents(evs, sess.Name))
				nowMin := nowCT.Hour()*60 + nowCT.Minute()
				_, f.InT1Blackout = kernel.InT1Blackout(nowMin, windows)
			}
		}
	}
	return f
}

// fadeORWideK resolves (a)'s k from the BOUND strategy, else the C5 default.
// fadeORWideKSource says which, for the boot line (A11: resolved, named).
func (at *AutoTrader) fadeORWideK() float64 {
	k, _ := at.fadeORWideKResolved()
	return k
}

func (at *AutoTrader) fadeORWideKResolved() (float64, string) {
	if at != nil && at.config.StrategyConfig.DayPlan != nil && at.config.StrategyConfig.DayPlan.FadeORWideK > 0 {
		return at.config.StrategyConfig.DayPlan.FadeORWideK, "strategy"
	}
	return fadeORWideKDefault, "default:C5 p80/median n=13"
}
