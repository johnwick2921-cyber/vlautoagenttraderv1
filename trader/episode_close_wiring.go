package trader

import (
	"strings"

	"nofx/kernel"
	"nofx/store"
)

// ── W1 EPISODE CONTRACT — THE PRODUCTION CALL PATH FOR THE CLOSER ────────────
//
// The 95f387ae boot shipped this wave with the closer built, unit-tested,
// reported, and called by nobody. This file is the call site, and
// trader/wiring_gate_test.go now fails if it disappears.
//
// A31 SCOPE: recording only. Nothing here refuses, cancels, sizes or places
// anything; it writes an outcome onto rows that already exist.

// closeCauseFor maps the arm-cancel reason to the episode's recorded cause.
// The reasons are the ones maybeManageArmedOrders already computed, so there is
// no second vocabulary to drift (class 97).
func closeCauseFor(reason string) string {
	switch {
	case strings.Contains(reason, "session ended"):
		return store.CloseCauseSessionEnd
	case strings.Contains(reason, "lifecycle"):
		return store.CloseCauseVersionSuperseded
	default:
		return store.CloseCauseSessionEnd
	}
}

// closeEpisodesForSessionClose closes every open touch row for this trader.
//
// THE FACTS COME FROM THE SCENARIO, NOT FROM A MUTATED STATE ROW. armed_orders
// is overwritten in place, so reading it here would ask a row that has already
// forgotten. It is also the reason historical terms are unrecomputable at all
// (see the backfill's terms_mutated_in_place). Under the settlement wave a
// timed-out cancel now rests at cancel_pending rather than reaching
// "cancelled", so a state read would additionally be reading a value that is
// deliberately not yet final.
//
// KNOWN LIMITATION, stated rather than hidden: the touch → scenario link is a
// price-proximity HEURISTIC and is NULL whenever two levels sit inside the band
// or nothing is close. A row with a NULL link closes as reached_declined —
// correct for a touch nothing was armed at, and not yet distinguishable from
// one whose arm this wave cannot see. The identity wave is what fixes that;
// until it lands, the honest reading of reached_declined is "no arm was linked
// to this touch", not "no arm existed".
func (at *AutoTrader) closeEpisodesForSessionClose(reason string) {
	if at == nil || at.store == nil {
		return
	}
	to := at.store.TouchOutcomes()
	if to == nil {
		return
	}

	// Scenario state, read ONCE from the plan the scenarios belong to. A nil
	// plan means the session ended with nothing active, in which case no
	// scenario confirmed or armed and Reached is all any row can claim.
	confirmed := map[string]bool{}
	armed := map[string]bool{}
	filled := map[string]bool{}
	entries := map[string]store.AttainableInputs{}
	var activeDoc *kernel.PlanDoc
	activePlanID, activeVersion := "", 0
	if ap := kernel.ActivePlanFor(at.id, at.futuresSymbol()); ap != nil {
		activeDoc, activePlanID, activeVersion = &ap.Doc, ap.PlanID, ap.Version
		for _, sc := range ap.Doc.Scenarios {
			if sc.Confirm != nil && sc.Confirm.RefPrice > 0 {
				confirmed[sc.ID] = true
			}
			if sc.Arm != nil && sc.Arm.Enabled && sc.Arm.Entry > 0 {
				armed[sc.ID] = true
				// PlanArmSpec.Entry documents itself as "resting limit price"
				// and the struct carries no kind, so "limit" is READ from the
				// shape of the type rather than guessed. A stop-entry arm
				// reaches the broker by a different path that this wave does
				// not link to a touch; when it does, this is the line that
				// gains the branch, not a literal somewhere else.
				entries[sc.ID] = store.AttainableInputs{
					Confirmed: confirmed[sc.ID],
					Armed:     true,
					ArmKind:   "limit",
					ArmEntry:  sc.Arm.Entry,
				}
			}
		}
	}

	facts := func(scenario *string) store.OpportunityFacts {
		f := store.OpportunityFacts{Reached: true}
		if scenario == nil {
			return f
		}
		f.Confirmed = confirmed[*scenario]
		f.Armed = armed[*scenario]
		f.Filled = filled[*scenario]
		return f
	}
	entryFor := func(scenario *string) *store.AttainableInputs {
		if scenario == nil {
			return nil
		}
		in, ok := entries[*scenario]
		if !ok {
			return nil
		}
		return &in
	}

	n, err := to.CloseOpenOpportunities(at.id, "", 0, "", closeCauseFor(reason), facts, entryFor, func(row store.TouchOutcomeRow) *string {
		if row.LevelID == nil {
			return row.ScenarioNearest
		}
		// Identity must not attach this row to a different plan version's
		// mutable facts merely because both scenarios happen to be called S1.
		if activeDoc == nil || row.PlanID != activePlanID || row.PlanVersion != activeVersion {
			return nil
		}
		return kernel.EpisodeScenarioByID(row.LevelID, activeDoc)
	})
	if err != nil {
		at.logWarnf("🎫 episode close failed (%s): %v", reason, err)
		return
	}
	if n > 0 {
		at.logInfof("🎫 episodes closed: %d (%s)", n, closeCauseFor(reason))
	}
}
