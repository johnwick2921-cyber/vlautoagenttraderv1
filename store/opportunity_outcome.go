// W1 — the opportunity outcome, on the record that already exists.
//
// THE UNIT WAS ALREADY THERE. touch_outcomes is one row per (trader, symbol,
// level, session-day, ordinal) — 4,043 rows carrying their own resolved k, Δ,
// band and horizon, and already resolving HOLD / BREAK / AMBIGUOUS. A new
// table beside it would have been a second source for the same fact, which is
// how two readers come to agree by accident (class 97). So this EXTENDS it.
//
// WHAT WAS MISSING is not the interaction — it is the OPPORTUNITY: which
// scenario was written on the level, and what the chance came to. Without that,
// a setup reached and declined is indistinguishable from one never reached, and
// both are absent rows rather than zeros. The research is explicit that a
// never-confirmed setup is a ZERO-TRADE OUTCOME; an experiment that silently
// drops it computes a rate over survivors.
//
// The filled side already had a home — trade_excursions is scenario-grained but
// UNIQUE(position_id), so it exists ONLY where a fill happened. That is exactly
// the filled-only bias the research objects to. The four unfilled outcomes
// below are what this wave adds.

package store

// The opportunity ladder. Each rung is strictly later than the one above it,
// and EVERY episode closes on exactly one of them (E6).
const (
	// The zone price never traded. A real outcome, not a missing row.
	OpportunityNeverReached = "never_reached"
	// Price entered the zone and the scenario's confirmation never fired.
	// This is the zero-trade outcome the research demands be counted.
	OpportunityReachedDeclined = "reached_declined"
	// Confirmed, but no arm was ever authored.
	OpportunityConfirmedNotArmed = "confirmed_not_armed"
	// Armed, but the order never filled.
	OpportunityArmedNotFilled = "armed_not_filled"
	// Filled. The only rung trade_excursions can currently see.
	OpportunityFilled = "filled"
)

// OpportunityFacts are the four observations the outcome is derived from. They
// are facts about what HAPPENED, not judgements: each is set by the site that
// witnesses it.
type OpportunityFacts struct {
	Reached   bool // price entered the zone
	Confirmed bool // the scenario's confirmation fired
	Armed     bool // an arm was authored for it
	Filled    bool // the order filled
}

// OpportunityOutcomeFor collapses the facts to one rung, and NEVER returns "".
//
// Later rungs imply the earlier ones by construction: a fill that claimed it
// was never reached would be a contradiction, so the ladder is evaluated from
// the bottom up rather than trusting the caller to set every flag.
func OpportunityOutcomeFor(f OpportunityFacts) string {
	switch {
	case f.Filled:
		return OpportunityFilled
	case f.Armed:
		return OpportunityArmedNotFilled
	case f.Confirmed:
		return OpportunityConfirmedNotArmed
	case f.Reached:
		return OpportunityReachedDeclined
	default:
		return OpportunityNeverReached
	}
}
