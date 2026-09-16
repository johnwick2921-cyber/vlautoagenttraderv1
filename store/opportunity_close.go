// W1 item 2 — closing the episode. E6: every one of them, every time.
//
// A row left open after its session-day ends is this wave's own defect wearing
// a different hat. An experiment counting outcomes would skip it in silence and
// the denominator would be quietly wrong — which is exactly what a missing
// never-confirmed row does today. So the closer is exhaustive BY CONSTRUCTION
// (it closes every row with a NULL outcome for the key, not a list someone
// remembered to build) and the pin is "zero open rows".

package store

import (
	"strings"

	"gorm.io/gorm"
)

// Why an episode ended. Always recorded: a closed row with no cause has lost
// the only thing that distinguishes an orderly close from an abandoned one.
const (
	CloseCauseSessionEnd        = "session_close"
	CloseCauseVersionSuperseded = "version_superseded"
	CloseCauseInvalidated       = "invalidated"
	CloseCauseForcedExit        = "forced_exit"
)

// scopeTrader applies the trader filter, with EMPTY MEANING ALL — the same
// convention BackfillExcursions uses at main.go's boot call, so a boot-time
// caller that has no single trader in hand does not have to invent one. It is
// one definition read by the closer, both counters and the backfill; a second
// hand-written `trader_id = ?` somewhere else is how these drift apart
// (class 97: one source, both readers).
func scopeTrader(q *gorm.DB, traderID string) *gorm.DB {
	if strings.TrimSpace(traderID) == "" {
		return q
	}
	return q.Where("trader_id = ?", traderID)
}

// CloseOpenOpportunities closes every still-open episode for the key and
// returns how many it closed.
//
// factsFor is called PER ROW with that row's scenario link (which may be NULL —
// an unlinked touch still closes, and still gets an outcome). The caller owns
// the facts because only it can see the confirm, the arm and the fill; this
// function owns the guarantee that nothing is left open.
//
// Idempotent: it selects on a NULL outcome, so a second run closes nothing and
// cannot overwrite a recorded verdict.
//
// entryFor is OPTIONAL. When supplied it returns the observations needed to
// resolve the attainable entry for that row; nil leaves attainable_entry NULL,
// which is the honest value for a caller that cannot see prices. It is
// computed HERE and not at record time because a touch has not confirmed,
// armed or filled at the moment it is recorded — writing a basis then would
// stamp "none:never_confirmed_never_armed" on an episode that is still open.
func (s *TouchOutcomeStore) CloseOpenOpportunities(
	traderID, planID string, planVersion int, session, cause string,
	factsFor func(scenario *string) OpportunityFacts,
	entryFor func(scenario *string) *AttainableInputs,
	scenarioFor ...func(TouchOutcomeRow) *string,
) (int, error) {
	if s == nil || s.db == nil || factsFor == nil {
		return 0, nil
	}
	// The plan key is OPTIONAL, with empty meaning ANY — the same convention as
	// scopeTrader above. A session that has ENDED has no active plan to name,
	// and E6 says every open row closes; refusing to close a row because the
	// plan that opened it is already gone would leave exactly the rows this
	// wave exists to account for.
	q := scopeTrader(s.db, traderID).Where("opportunity_outcome IS NULL")
	if strings.TrimSpace(planID) != "" {
		q = q.Where("plan_id = ?", planID)
	}
	if planVersion > 0 {
		q = q.Where("plan_version = ?", planVersion)
	}
	if strings.TrimSpace(session) != "" {
		q = q.Where("session = ?", session)
	}
	var rows []TouchOutcomeRow
	if err := q.Find(&rows).Error; err != nil {
		return 0, err
	}

	closed := 0
	for i := range rows {
		scenario := rows[i].ScenarioNearest
		if len(scenarioFor) > 0 {
			scenario = scenarioFor[0](rows[i])
		}
		outcome := OpportunityOutcomeFor(factsFor(scenario))
		c := cause
		// The attainable entry, resolved from the SAME facts that decided the
		// outcome, so the two can never disagree about whether an entry existed.
		var aePrice *float64
		var aeBasis *string
		if entryFor != nil {
			if in := entryFor(scenario); in != nil {
				got := ResolveAttainableEntry(*in)
				aePrice, aeBasis = got.Price, &got.Basis
			}
		}
		if err := s.db.Model(&TouchOutcomeRow{}).
			Where("id = ?", rows[i].ID).
			Updates(episodeCloseFields(outcome, c, aePrice, aeBasis)).Error; err != nil {
			return closed, err
		}
		closed++
	}
	return closed, nil
}

// CountOpenOpportunities is the E6 probe: after a session-day ends this must be
// zero, and the boot line reports it so an unclosed row is visible rather than
// merely absent from a rate.
func (s *TouchOutcomeStore) CountOpenOpportunities(traderID string) (int64, error) {
	if s == nil || s.db == nil {
		return 0, nil
	}
	var n int64
	err := scopeTrader(s.db.Model(&TouchOutcomeRow{}), traderID).
		Where("opportunity_outcome IS NULL").
		Count(&n).Error
	return n, err
}

// CountClosedByOutcome feeds the boot line's per-rung counts.
func (s *TouchOutcomeStore) CountClosedByOutcome(traderID string, sinceMs int64) (map[string]int64, error) {
	out := map[string]int64{}
	if s == nil || s.db == nil {
		return out, nil
	}
	type row struct {
		Outcome string
		N       int64
	}
	var rs []row
	if err := scopeTrader(s.db.Model(&TouchOutcomeRow{}), traderID).
		Select("opportunity_outcome as outcome, COUNT(*) as n").
		Where("opportunity_outcome IS NOT NULL AND opened_at_ms >= ?", sinceMs).
		Group("opportunity_outcome").Scan(&rs).Error; err != nil {
		return out, err
	}
	for _, r := range rs {
		out[r.Outcome] = r.N
	}
	return out, nil
}

// episodeCloseFields builds the update set. The attainable columns are written
// ONLY when they were resolved: a nil price with a nil basis leaves both NULL,
// so "we did not compute it" stays distinguishable from "there was none", which
// is the whole reason AttainableNone and AttainableNotCaptured are different
// constants.
func episodeCloseFields(outcome, cause string, price *float64, basis *string) map[string]any {
	f := map[string]any{
		"opportunity_outcome": outcome,
		"close_cause":         cause,
	}
	if basis != nil {
		f["attainable_entry"] = price
		f["attainable_entry_basis"] = *basis
	}
	return f
}
