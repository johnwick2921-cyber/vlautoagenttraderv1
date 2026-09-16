// W1 item 2 — EVERY episode closes. E6.
//
// A row left open after its session-day ends is the defect this wave exists to
// remove wearing a different hat: an experiment counting outcomes would skip it
// silently, and the denominator would quietly be wrong. So the closer is
// exhaustive by construction, and the pin is "zero open rows", not "the ones I
// remembered are closed".

package store

import (
	"testing"
	"time"
)

func openRow(t *testing.T, st *TouchOutcomeStore, scenario string, openedAtMs int64) uint {
	t.Helper()
	r := &TouchOutcomeRow{
		TraderID: "t1", Symbol: "MNQ", LevelPrice: 29000, LevelKind: "PDH",
		PlanID: "P1", PlanVersion: 1, Session: "NY", Ordinal: 1,
		OpenedAtMs: openedAtMs, ClosedAtMs: openedAtMs + 60000,
		Outcome: "hold", Validity: ValidityValid,
	}
	if scenario != "" {
		s := scenario
		r.ScenarioNearest = &s
		r.ScenarioLinkBasis = ScenarioLinkPriceProximity
	} else {
		r.ScenarioLinkBasis = ScenarioLinkNoScenario
	}
	if err := st.SaveOutcome(r); err != nil {
		t.Fatalf("seed: %v", err)
	}
	return r.ID
}

// E6 — after the close runs, NOTHING is open. Not "the linked ones"; nothing.
func TestEveryOpportunityClosesAtSessionEnd(t *testing.T) {
	st := NewTouchOutcomeStore(newArmedTestDB(t))
	day := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC).UnixMilli()

	openRow(t, st, "S1", day)
	openRow(t, st, "S2", day+1000)
	openRow(t, st, "", day+2000) // an UNLINKED touch still has to close

	n, err := st.CloseOpenOpportunities("t1", "P1", 1, "NY", CloseCauseSessionEnd,
		func(scenario *string) OpportunityFacts {
			// A touch row exists BECAUSE price entered the zone: reached is
			// true by construction. Nothing else fired here.
			return OpportunityFacts{Reached: true}
		}, nil)
	if err != nil {
		t.Fatalf("close: %v", err)
	}
	if n != 3 {
		t.Fatalf("closed %d rows, want 3 — an unlinked touch closes too", n)
	}

	openLeft, err := st.CountOpenOpportunities("t1")
	if err != nil {
		t.Fatal(err)
	}
	if openLeft != 0 {
		t.Fatalf("%d row(s) still open after the session ended (E6)", openLeft)
	}
}

// The outcome recorded is the one the facts imply, per row — not one verdict
// stamped across the batch.
func TestCloseRecordsPerRowOutcomes(t *testing.T) {
	st := NewTouchOutcomeStore(newArmedTestDB(t))
	day := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC).UnixMilli()
	openRow(t, st, "S1", day)
	openRow(t, st, "S2", day+1000)

	if _, err := st.CloseOpenOpportunities("t1", "P1", 1, "NY", CloseCauseSessionEnd,
		func(scenario *string) OpportunityFacts {
			if scenario != nil && *scenario == "S2" {
				return OpportunityFacts{Reached: true, Confirmed: true, Armed: true, Filled: true}
			}
			return OpportunityFacts{Reached: true}
		}, nil); err != nil {
		t.Fatal(err)
	}

	rows, _ := st.AllOutcomes()
	got := map[string]string{}
	for _, r := range rows {
		if r.ScenarioNearest != nil && r.OpportunityOutcome != nil {
			got[*r.ScenarioNearest] = *r.OpportunityOutcome
		}
	}
	if got["S1"] != OpportunityReachedDeclined {
		t.Errorf("S1 → %q, want %q", got["S1"], OpportunityReachedDeclined)
	}
	if got["S2"] != OpportunityFilled {
		t.Errorf("S2 → %q, want %q", got["S2"], OpportunityFilled)
	}
}

// Closing twice must not re-close, re-count, or overwrite a recorded outcome.
func TestCloseIsIdempotent(t *testing.T) {
	st := NewTouchOutcomeStore(newArmedTestDB(t))
	day := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC).UnixMilli()
	openRow(t, st, "S1", day)

	f := func(*string) OpportunityFacts { return OpportunityFacts{Reached: true} }
	first, _ := st.CloseOpenOpportunities("t1", "P1", 1, "NY", CloseCauseSessionEnd, f, nil)
	second, _ := st.CloseOpenOpportunities("t1", "P1", 1, "NY", CloseCauseSessionEnd, f, nil)
	if first != 1 || second != 0 {
		t.Fatalf("first close %d, second %d — a closed episode must not re-close", first, second)
	}
}

// The cause is always stated. A closed row with no cause loses why it ended.
func TestCloseAlwaysStatesItsCause(t *testing.T) {
	st := NewTouchOutcomeStore(newArmedTestDB(t))
	day := time.Date(2026, 9, 10, 12, 0, 0, 0, time.UTC).UnixMilli()
	openRow(t, st, "S1", day)

	if _, err := st.CloseOpenOpportunities("t1", "P1", 1, "NY", CloseCauseSessionEnd,
		func(*string) OpportunityFacts { return OpportunityFacts{Reached: true} }, nil); err != nil {
		t.Fatal(err)
	}
	rows, _ := st.AllOutcomes()
	r := rows[0]
	if r.CloseCause == nil || *r.CloseCause != CloseCauseSessionEnd {
		t.Fatalf("cause = %v, want %q", r.CloseCause, CloseCauseSessionEnd)
	}
}
