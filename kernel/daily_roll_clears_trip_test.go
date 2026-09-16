package kernel

import (
	"testing"
	"time"
)

// TestTheCMERollClearsTheDailyForceFlatTrip — D4(c), dispatch 104.
//
// vet-06 listed "it clears only at the first AI cycle after the roll" as the
// daily limit's third hole. The truth at this rev is worse: the trip NEVER
// clears automatically at all.
//
// MaybeResetDaily rolls lastDailyResetDate at the 17:00 CT boundary and does
// not touch forceFlatReason. clearAllDailyForceFlat has exactly one caller —
// ResetDailyPnLAt — whose only production entry is the operator's
// POST /api/risk/force-flat path. So once the limit trips, NEW ENTRIES STAY
// BLOCKED across the roll, across the next session, and across the one after
// that, until a human resets it or the process restarts.
//
// The code says otherwise in its own comment (risk_limits.go, "Cleared by the
// CME session-day reset … so a trip lasts the session-day and lifts with the
// daily window"). The comment describes the intent; nothing wired it.
//
// It has never been observed because the guardrails master is OFF, so the trip
// has never fired — a latent halt waiting for the day the owner turns it on.
func TestTheCMERollClearsTheDailyForceFlatTrip(t *testing.T) {
	ResetDailyPnLAt(time.Date(2026, 9, 8, 9, 0, 0, 0, CTLocation()))

	SetDailyForceFlat("trader-roll", "daily loss -470.00 <= limit -450.00")
	if DailyForceFlatReason("trader-roll") == "" {
		t.Fatal("fixture: the trip did not take")
	}

	// The CME session-day rolls at 17:00 CT.
	rolled := MaybeResetDaily(time.Date(2026, 9, 8, 17, 30, 0, 0, CTLocation()))
	if !rolled {
		t.Fatal("MaybeResetDaily did not see the 17:00 CT rollover")
	}
	if why := DailyForceFlatReason("trader-roll"); why != "" {
		t.Fatalf("the trip SURVIVED the CME roll: %q. New entries stay blocked into the next "+
			"session-day with nothing to lift them but an operator or a restart — while the code's "+
			"own comment promises the trip 'lifts with the daily window'.", why)
	}
}

// The roll must not clear a trip set AFTER it — a same-day trip stands.
func TestATripAfterTheRollSurvivesTheSameDay(t *testing.T) {
	ResetDailyPnLAt(time.Date(2026, 9, 8, 18, 0, 0, 0, CTLocation()))
	SetDailyForceFlat("trader-same", "tripped after the roll")
	if MaybeResetDaily(time.Date(2026, 9, 8, 19, 0, 0, 0, CTLocation())) {
		t.Fatal("no rollover happened between 18:00 and 19:00 CT on the same session-day")
	}
	if DailyForceFlatReason("trader-same") == "" {
		t.Fatal("a trip inside the SAME session-day must stand — the halt is for the rest of the day")
	}
	ClearDailyForceFlat("trader-same")
}
