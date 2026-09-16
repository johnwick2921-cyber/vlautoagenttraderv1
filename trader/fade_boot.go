// ── W2 D5 — THE BOOT LINE, and D6 — THE BACKFILL. Both READ, never literal. ──
package trader

import (
	"fmt"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// boundFadeORWideK resolves (a)'s k from THE STRATEGY BOUND TO A TRADER — the
// same row the runtime reads (at.config.StrategyConfig). It reuses
// bootRiskFacts' one-bound-strategy discipline: more than one bound strategy
// is reported as such, never averaged and never the first one found.
func boundFadeORWideK(st *store.Store) (k float64, source string) {
	if cfg, _, _, _, ok := bootRiskFacts(st); ok && cfg != nil && cfg.DayPlan != nil && cfg.DayPlan.FadeORWideK > 0 {
		return cfg.DayPlan.FadeORWideK, "strategy"
	}
	return fadeORWideKDefault, "default:C5 p80/median"
}

// FadePermissionBootLine renders D5. Every count is READ from touch_outcomes;
// k names its resolver; each exclusion carries its coverage note in the help
// text so the line cannot imply protection it lacks (owner ruling 2026-09-10).
func FadePermissionBootLine(st *store.Store, now time.Time, bf store.FadeBackfillResult) string {
	k, ksrc := boundFadeORWideK(st)
	var permitted, excluded, notEval int64
	countsOK := false
	if st != nil {
		if p, e, n, err := st.TouchOutcomes().CountFadeLabels("", kernel.CMESessionDayStart(now).UnixMilli()); err == nil {
			permitted, excluded, notEval, countsOK = p, e, n, true
		}
	}
	counts := "today permitted=n/a excluded=n/a not-evaluated=n/a (table unreadable)"
	if countsOK {
		counts = fmt.Sprintf("today permitted=%d excluded=%d not-evaluated=%d", permitted, excluded, notEval)
	}
	ex := []string{
		fmt.Sprintf("or_wide k=%.2f[I:%s]", k, ksrc),
		"ib_held[I:continuous]", "beyond_map[I]", "t1_news[O:UNKNOWN-never-excludes]",
		fmt.Sprintf("first_n=%d[O:no-trade-band]", kernel.FirstNoTradeMinutes()),
	}
	backfill := "backfill=not-run"
	if bf.Ran {
		backfill = fmt.Sprintf("backfill recomputed=%d unrecomputable=%d untouched=%d",
			bf.Recomputed, bf.Unrecomputable, bf.Untouched)
	}
	return fmt.Sprintf("🚦 fade permission: LABEL ONLY (no refusal) · exclusions=[%s] · %s · %s · coverage: on 2026-09-03 the label would have PERMITTED the filled arm (id 35, 09:02) — E3's null",
		strings.Join(ex, " "), counts, backfill)
}

// BackfillFadePermission (D6) is three-state and NEVER labels a day from its
// completed profile: for each episode it re-runs the predicate with the clock
// set to that episode's OPEN, reading only bars closed before it.
//
//	recomputed          the inputs were recoverable and the label was written
//	unrecomputable:<w>  OR or IB bars missing for that session — marked, not
//	                    guessed
//	untouched           opened before W1's boot; no episode contract existed
func (at *AutoTrader) BackfillFadePermission(sinceMs int64) store.FadeBackfillResult {
	res := store.FadeBackfillResult{Ran: true}
	if at == nil || at.store == nil {
		return res
	}
	ts := at.store.TouchOutcomes()
	rows, err := ts.OpenFadeCandidates(at.id, sinceMs)
	if err != nil {
		at.logWarnf("🚦 fade backfill: read failed: %v", err)
		return res
	}
	for i := range rows {
		r := rows[i]
		if r.OpenedAtMs < sinceMs {
			res.Untouched++
			continue
		}
		openedAt := time.UnixMilli(r.OpenedAtMs)
		facts := at.fadeFactsAt(openedAt, r.Symbol, r.LevelPrice, nil, r.EntrySide)
		// Recoverable means the OR bar for that session was on the tape; a
		// session with no 08:30 bar cannot be labelled from what we have.
		if !facts.ORComplete && !facts.SessionOpen.IsZero() && openedAt.After(facts.SessionOpen.Add(5*time.Minute)) {
			res.Unrecomputable++
			_ = ts.MarkFadeUnrecomputable(r.ID, "no_or_bar")
			continue
		}
		v := kernel.FadePermissionAt(openedAt, facts)
		if !v.Evaluated {
			res.Unrecomputable++
			_ = ts.MarkFadeUnrecomputable(r.ID, "not_evaluable_at_open")
			continue
		}
		stamp := store.FadeStamp{Evaluated: true, Permitted: v.Permitted, AtMs: r.OpenedAtMs}
		for _, ex := range v.Exclusions {
			stamp.Exclusions = append(stamp.Exclusions, ex.Name)
		}
		if err := ts.StampFadePermission(r.ID, stamp); err == nil {
			res.Recomputed++
		}
	}
	return res
}
