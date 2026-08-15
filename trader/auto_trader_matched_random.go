package trader

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// P5.6 — matched-random verdict recording (per graded close) + the fixed-cadence
// weekly evaluation. All gated on day_plan → dormant for crypto / plan-off.

// recordMatchedRandomForClose records ONE reaction verdict for the just-closed
// trade, keyed by the type of the nearest plan level to the entry. "reacted" =
// the trade saw more favorable than adverse excursion (a level-worked proxy).
// Best-effort: a missing/unreadable plan simply skips (WARMING stays honest).
func (at *AutoTrader) recordMatchedRandomForClose(p *store.TraderPosition, ex kernel.Excursion) {
	if at.store == nil || p == nil || p.EntryPrice <= 0 || p.EntryTime <= 0 {
		return
	}
	entryT := time.UnixMilli(p.EntryTime)
	reg := kernel.DefaultSessionRegistry()
	sess, ok := reg.ActiveSession(entryT)
	if !ok {
		return
	}
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return
	}
	tradeDate := entryT.In(loc).Format("2006-01-02")
	row, err := at.store.Plan().GetLatestPlanForSession(tradeDate, sess.Name)
	if err != nil || row == nil {
		return
	}
	var doc kernel.PlanDoc
	if json.Unmarshal([]byte(row.Doc), &doc) != nil || len(doc.Levels) == 0 {
		return
	}
	// nearest level to the entry → its provenance type
	levelType := "other"
	best := math.MaxFloat64
	for _, l := range doc.Levels {
		if d := math.Abs(l.Price - p.EntryPrice); d < best {
			best = d
			levelType = kernel.LevelTypeFromLabel(l.Label)
		}
	}
	reacted := ex.MFE > ex.MAE && ex.MFE > 0
	if err := at.store.MatchedRandom().RecordTouch(levelType, reacted, tradeDate, time.Now().Unix()); err != nil {
		at.logWarnf("📊 matched-random record failed: %v", err)
	}
}

// maybeRunWeeklyMatchedRandom freezes ONE honesty-gate snapshot per ISO week, on
// Sundays only (fixed cadence — never a nightly re-peek). Idempotent + first-
// writer-wins across traders, so the snapshot is never re-computed mid-week.
func (at *AutoTrader) maybeRunWeeklyMatchedRandom(now time.Time) {
	if !at.dayPlanEnabled() || at.store == nil {
		return
	}
	if now.Weekday() != time.Sunday {
		return
	}
	year, week := now.ISOWeek()
	isoWeek := fmt.Sprintf("%d-W%02d", year, week)
	mr := at.store.MatchedRandom()
	if mr.HasWeekly(isoWeek) {
		return // already frozen this week — no re-peek
	}
	counts, err := mr.CountsByType()
	if err != nil {
		return
	}
	perType := make(map[string]kernel.TypeCounts, len(counts))
	for t, c := range counts {
		perType[t] = kernel.TypeCounts{Touches: c.Touches, Reactions: c.Reactions}
	}
	verdicts := kernel.EvaluateMatchedRandom(perType)
	js, _ := json.Marshal(verdicts)
	if wrote, _ := mr.SaveWeeklyIfAbsent(isoWeek, string(js), now.Unix()); wrote {
		at.logInfof("📊 weekly matched-random eval frozen for %s (%d types)", isoWeek, len(verdicts))
	}
}
