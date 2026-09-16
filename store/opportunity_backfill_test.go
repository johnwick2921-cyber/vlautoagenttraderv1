// W1 item 4 — the backfill, in three states and no fourth. A30.
//
// The honest answer here is mostly "cannot". formed_at_ms is present on 164 of
// 4,163 rows (4.1%), so unrecomputable will dominate — and that IS the finding,
// not a failure of the backfill. A backfill that reported a high recomputed
// count would be guessing, and a guessed first_reached_at is exactly what H
// forbids.

package store

import (
	"testing"
)

func seedRow(t *testing.T, st *TouchOutcomeStore, openedMs, formedMs int64, scenario string) uint {
	t.Helper()
	r := &TouchOutcomeRow{
		TraderID: "t1", Symbol: "MNQ", LevelPrice: 29000, LevelKind: "PDH",
		PlanID: "P1", PlanVersion: 1, Session: "NY", Ordinal: 1,
		OpenedAtMs: openedMs, ClosedAtMs: openedMs + 60000,
		Outcome: "hold", FormedAtMs: formedMs, Validity: ValidityValid,
	}
	if scenario != "" {
		s := scenario
		r.ScenarioNearest = &s
		r.ScenarioLinkBasis = ScenarioLinkPriceProximity
	} else {
		r.ScenarioLinkBasis = ScenarioLinkNoScenario
	}
	if err := st.SaveOutcome(r); err != nil {
		t.Fatal(err)
	}
	return r.ID
}

// THREE STATES, and the counts add up to the rows considered. A fourth state —
// or a row falling through all three — is the silent loss this pins against.
func TestBackfillReportsThreeStatesThatAddUp(t *testing.T) {
	st := NewTouchOutcomeStore(newArmedTestDB(t))
	era := DayPlanEraStart.UnixMilli()

	seedRow(t, st, era+86400000, era, "S1") // recomputable
	seedRow(t, st, era+86400000*2, 0, "S2") // no formation
	seedRow(t, st, era+86400000*3, era, "") // no scenario link
	seedRow(t, st, era-86400000, era, "S3") // BEFORE the era — untouched

	res, err := st.BackfillOpportunities("t1")
	if err != nil {
		t.Fatalf("backfill: %v", err)
	}
	total := res.Recomputed + res.UntouchedPreEra
	for _, n := range res.Unrecomputable {
		total += n
	}
	if total != 4 {
		t.Fatalf("three states cover %d of 4 rows — a row fell through", total)
	}
	if res.UntouchedPreEra != 1 {
		t.Errorf("pre-era rows must be UNTOUCHED, got %d", res.UntouchedPreEra)
	}
	if res.Recomputed != 1 {
		t.Errorf("recomputed = %d, want 1", res.Recomputed)
	}
}

// The causes are NAMED, not totalled. "966 unrecomputable" is useless; "966
// because no formation time" is the finding.
func TestUnrecomputableIsBrokenDownByCause(t *testing.T) {
	st := NewTouchOutcomeStore(newArmedTestDB(t))
	era := DayPlanEraStart.UnixMilli()
	seedRow(t, st, era+86400000, 0, "S1")
	seedRow(t, st, era+86400000*2, era, "")

	res, _ := st.BackfillOpportunities("t1")
	if res.Unrecomputable[BackfillNoFormation] != 1 {
		t.Errorf("no-formation count = %d, want 1", res.Unrecomputable[BackfillNoFormation])
	}
	if res.Unrecomputable[BackfillNoScenarioLink] != 1 {
		t.Errorf("no-link count = %d, want 1", res.Unrecomputable[BackfillNoScenarioLink])
	}
	if len(res.Unrecomputable) < 2 {
		t.Error("a single 'unrecomputable' total hides which input was missing")
	}
}

// A row the backfill could not recompute is MARKED, never left indistinguishable
// from one nobody has looked at yet.
func TestUnrecomputableRowsAreMarkedNotSkipped(t *testing.T) {
	st := NewTouchOutcomeStore(newArmedTestDB(t))
	era := DayPlanEraStart.UnixMilli()
	id := seedRow(t, st, era+86400000, 0, "S1")

	if _, err := st.BackfillOpportunities("t1"); err != nil {
		t.Fatal(err)
	}
	var r TouchOutcomeRow
	if err := st.db.First(&r, id).Error; err != nil {
		t.Fatal(err)
	}
	if r.CloseCause == nil || *r.CloseCause != BackfillNoFormation {
		t.Fatalf("an unrecomputable row must carry its reason, got %v", r.CloseCause)
	}
	if r.OpportunityOutcome != nil {
		t.Fatalf("it must NOT be given an outcome it cannot support, got %q", *r.OpportunityOutcome)
	}
}

// H: never a guessed first_reached_at. The backfill has no branch that invents
// a reach time from a price.
func TestBackfillNeverInventsAReachTime(t *testing.T) {
	st := NewTouchOutcomeStore(newArmedTestDB(t))
	era := DayPlanEraStart.UnixMilli()
	id := seedRow(t, st, era+86400000, era, "S1")

	if _, err := st.BackfillOpportunities("t1"); err != nil {
		t.Fatal(err)
	}
	var r TouchOutcomeRow
	st.db.First(&r, id)
	// A recomputed row's outcome rests on the touch itself (a touch row exists
	// because price entered the zone) — never on a reconstructed timestamp.
	if r.OpportunityOutcome == nil || *r.OpportunityOutcome != OpportunityReachedDeclined {
		t.Fatalf("a recomputable row closes reached_declined from the touch alone, got %v", r.OpportunityOutcome)
	}
	// And the terms are NOT reconstructed: armed_orders mutates in place.
	if r.TermsStop != nil || r.TermsTarget != nil {
		t.Error("historical terms are unrecoverable and must stay NULL")
	}
}
