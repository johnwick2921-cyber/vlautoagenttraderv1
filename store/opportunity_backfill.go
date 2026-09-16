// W1 item 4 — the backfill, in three states and no fourth. A30.
//
// THE HONEST ANSWER HERE IS MOSTLY "CANNOT", and that is the finding rather
// than a shortfall. formed_at_ms is present on 164 of 4,163 touch rows (4.1%),
// so unrecomputable dominates by construction. A backfill reporting a high
// recomputed count would be guessing, and a guessed first_reached_at is exactly
// what this wave forbids.
//
// The three states are exhaustive and the counts add up to the rows considered.
// A row that fell through all three would be a silent loss — the same shape as
// the missing never-confirmed row this wave exists to fix — so the pin asserts
// the arithmetic, not the individual branches.

package store

// Why a historical row could not be recomputed. NAMED, never totalled: "966
// unrecomputable" is useless, "966 because no formation time" is the finding.
const (
	BackfillNoFormation    = "unrecomputable:no_formation"
	BackfillNoScenarioLink = "unrecomputable:no_scenario_link"
	// armed_orders is a MUTATED STATE ROW — its entry/stop/target are updated
	// in place, so the terms at the moment they became executable are gone.
	// Recorded forward from now on; never reconstructed from updated_at.
	BackfillTermsMutatedInPlace = "unrecomputable:terms_mutated_in_place"
)

// BackfillResult is the three-state report (A30).
type BackfillResult struct {
	Recomputed      int
	Unrecomputable  map[string]int
	UntouchedPreEra int
}

// BackfillOpportunities classifies historical rows since DayPlanEraStart.
//
// It NEVER invents a reach time. A recomputed row's outcome rests on the touch
// row itself — one exists because price entered the zone — and on nothing
// reconstructed. Terms stay NULL because they are genuinely unrecoverable.
func (s *TouchOutcomeStore) BackfillOpportunities(traderID string) (BackfillResult, error) {
	res := BackfillResult{Unrecomputable: map[string]int{}}
	if s == nil || s.db == nil {
		return res, nil
	}
	era := DayPlanEraStart.UnixMilli()

	var rows []TouchOutcomeRow
	if err := scopeTrader(s.db, traderID).Where("opportunity_outcome IS NULL").
		Find(&rows).Error; err != nil {
		return res, err
	}

	for i := range rows {
		r := rows[i]

		// Outside the era: UNTOUCHED. Not examined, not marked, not counted as
		// a failure — a third state, not a variety of the second.
		if r.OpenedAtMs < era {
			res.UntouchedPreEra++
			continue
		}

		if cause := backfillBlocker(r); cause != "" {
			// MARKED, never skipped: a row nobody could recompute must not be
			// indistinguishable from one nobody has looked at yet.
			c := cause
			if err := s.db.Model(&TouchOutcomeRow{}).Where("id = ?", r.ID).
				Update("close_cause", c).Error; err != nil {
				return res, err
			}
			res.Unrecomputable[cause]++
			continue
		}

		// Recomputable. The touch row's own existence is the evidence that
		// price reached the zone; everything later (confirm, arm, fill) is not
		// recoverable for history, so the honest close is reached_declined.
		outcome := OpportunityOutcomeFor(OpportunityFacts{Reached: true})
		cause := CloseCauseSessionEnd
		if err := s.db.Model(&TouchOutcomeRow{}).Where("id = ?", r.ID).
			Updates(map[string]any{
				"opportunity_outcome": outcome,
				"close_cause":         cause,
			}).Error; err != nil {
			return res, err
		}
		res.Recomputed++
	}
	return res, nil
}

// backfillBlocker names the FIRST missing input, or "" when the row can be
// recomputed. Formation is checked first because it is the dominant cause and
// the one that makes a level's identity unreliable in the first place.
func backfillBlocker(r TouchOutcomeRow) string {
	if r.FormedAtMs <= 0 {
		return BackfillNoFormation
	}
	if r.ScenarioNearest == nil {
		return BackfillNoScenarioLink
	}
	return ""
}
