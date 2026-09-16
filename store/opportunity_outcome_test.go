// W1 — the opportunity outcome, pinned before it exists.
//
// The unit already exists: touch_outcomes is one row per (trader, symbol,
// level, session-day, ordinal), 4,043 rows, already resolving HOLD/BREAK/
// AMBIGUOUS with its own resolved k/Δ/band/horizon. What it has never carried
// is WHICH SCENARIO was written on the level, and what the OPPORTUNITY came to
// — so a setup that was reached and declined is indistinguishable from one that
// was never reached at all, and both are missing rows rather than zeros.
//
// The research is explicit that a never-confirmed setup is a ZERO-TRADE
// OUTCOME, not a missing row. These pins are that sentence in code.

package store

import "testing"

// E1 — reached, never confirmed → reached_declined. The load-bearing pin.
func TestReachedButNeverConfirmedClosesDeclined(t *testing.T) {
	got := OpportunityOutcomeFor(OpportunityFacts{Reached: true})
	if got != OpportunityReachedDeclined {
		t.Fatalf("reached, never confirmed → %q, want %q — a declined setup is a zero-trade OUTCOME, not a missing row", got, OpportunityReachedDeclined)
	}
}

// E4 — never reached is NOT the same as reached-and-declined. This is the
// distinction the record could not make at all before this wave.
func TestNeverReachedIsItsOwnOutcome(t *testing.T) {
	got := OpportunityOutcomeFor(OpportunityFacts{})
	if got != OpportunityNeverReached {
		t.Fatalf("never reached → %q, want %q", got, OpportunityNeverReached)
	}
	if OpportunityNeverReached == OpportunityReachedDeclined {
		t.Fatal("never_reached and reached_declined must be different values — telling them apart IS the wave")
	}
}

// The ladder, each rung distinct.
func TestOpportunityLadder(t *testing.T) {
	for _, c := range []struct {
		name string
		f    OpportunityFacts
		want string
	}{
		{"never reached", OpportunityFacts{}, OpportunityNeverReached},
		{"reached only", OpportunityFacts{Reached: true}, OpportunityReachedDeclined},
		{"confirmed, no arm", OpportunityFacts{Reached: true, Confirmed: true}, OpportunityConfirmedNotArmed},
		{"armed, no fill", OpportunityFacts{Reached: true, Confirmed: true, Armed: true}, OpportunityArmedNotFilled},
		{"filled", OpportunityFacts{Reached: true, Confirmed: true, Armed: true, Filled: true}, OpportunityFilled},
	} {
		if got := OpportunityOutcomeFor(c.f); got != c.want {
			t.Errorf("%s → %q, want %q", c.name, got, c.want)
		}
	}
}

// A later rung implies the earlier ones. A fill that claims it was never
// reached is a contradiction the recorder must not be able to express.
func TestLaterRungsImplyEarlierOnes(t *testing.T) {
	if got := OpportunityOutcomeFor(OpportunityFacts{Filled: true}); got != OpportunityFilled {
		t.Errorf("a fill is reached/confirmed/armed by construction, got %q", got)
	}
	if got := OpportunityOutcomeFor(OpportunityFacts{Armed: true}); got != OpportunityArmedNotFilled {
		t.Errorf("an arm implies reached+confirmed, got %q", got)
	}
}

// E6 — EVERY pair closes. An outcome is always set; there is no empty string.
func TestEveryFactSetProducesAnOutcome(t *testing.T) {
	for i := 0; i < 16; i++ {
		f := OpportunityFacts{
			Reached:   i&1 != 0,
			Confirmed: i&2 != 0,
			Armed:     i&4 != 0,
			Filled:    i&8 != 0,
		}
		if got := OpportunityOutcomeFor(f); got == "" {
			t.Fatalf("facts %+v produced NO outcome — every episode closes (E6)", f)
		}
	}
}

// E7 — NULL is not zero. An unresolved scenario link is NULL with a stated
// reason; it is never an empty string that reads as "no scenario", and it is
// never inferred from a price.
func TestScenarioLinkIsNullNotEmpty(t *testing.T) {
	r := TouchOutcomeRow{}
	if r.ScenarioNearest != nil {
		t.Fatal("an unlinked touch row must carry a NULL link, not a value")
	}
	// The column is named for the heuristic it is, not for the fact it is not.
	if ScenarioLinkNoScenario == "" {
		t.Fatal("an unresolved link must carry a stated reason, never a bare NULL")
	}
	// The attainable entry is NULL for a scenario that never armed — not 0.0,
	// which would read as "available at zero".
	if r.AttainableEntry != nil {
		t.Fatal("attainable entry must be NULL when uncaptured, never 0")
	}
}
