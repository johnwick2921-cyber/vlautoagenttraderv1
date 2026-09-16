package kernel

import (
	"strings"
	"testing"
)

// TestG3FlipHold tests the flip-hysteresis gate: a plan younger than
// FLIP_MIN_HOLD_MIN cannot flip back; death always wins during the hold.
func TestG3FlipHold(t *testing.T) {
	flip := g7FlipCond()
	death := PlanCondition{Price: 29480.25, Side: "below", Rule: "2x5m"} // closes 29469.x are below → death leg met
	fixture := g7Fixture()                                               // 2 consecutive 5m closes below 29470.25
	now := ctMs(2026, 8, 21, 8, 10)

	// (1) Fresh plan (born 07:45, the 08:00/08:05 touches are in-window): the
	// flip leg is held, death absent → skipped.
	doc := PlanDoc{Bias: PlanBias{FlipCondition: "flip"}, FlipStructured: &flip}
	_, fired, skipped := PlanDeathOrFlipSinceFresh(doc, fixture, "2x5m", ctMs(2026, 8, 21, 7, 45), now)
	if fired {
		t.Fatalf("flip within hold must be suppressed")
	}
	held := false
	for _, s := range skipped {
		if strings.Contains(s, "flip=hold") {
			held = true
		}
	}
	if !held {
		t.Fatalf("want flip=hold skip entry, got %v", skipped)
	}

	// (2) Same plan born 07:10 (60 min old): the flip fires normally.
	_, fired, _ = PlanDeathOrFlipSinceFresh(doc, fixture, "2x5m", ctMs(2026, 8, 21, 7, 10), now)
	if !fired {
		t.Fatalf("flip past the hold must fire")
	}

	// (3) Death always wins during the hold: death condition also met → death
	// fires even though the plan is fresh.
	docBoth := PlanDoc{Bias: PlanBias{FlipCondition: "flip"}, FlipStructured: &flip, DeathStructured: &death}
	killer, fired, _ := PlanDeathOrFlipSinceFresh(docBoth, fixture, "2x5m", ctMs(2026, 8, 21, 7, 45), now)
	if !fired || !strings.Contains(killer, "death-condition") {
		t.Fatalf("death must win during the hold, got fired=%v killer=%q", fired, killer)
	}
}

func TestFlipMinHoldDefault(t *testing.T) {
	if FlipMinHoldMin() != DefaultFlipMinHoldMinutes {
		t.Fatalf("default hold must be %d", DefaultFlipMinHoldMinutes)
	}
}
