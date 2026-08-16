// W15.B — /api/plan/today?session=… . The card's session tabs were pure
// highlighting: the request had no session dimension, so clicking ASIA rendered
// the LIVE session's plan. These lock the resolution the handler now applies.

package api

import (
	"testing"

	"nofx/kernel"
	"nofx/store"
)

func TestW15ResolveRequestedSession(t *testing.T) {
	reg := kernel.DefaultSessionRegistry()
	ny, _ := reg.SessionByName(kernel.SessionNY)

	// no param → the live session, explicit=false (pre-W15 behavior exactly)
	name, def, explicit := resolveRequestedSession(reg, "", kernel.SessionNY, ny)
	if name != kernel.SessionNY || def != ny || explicit {
		t.Fatalf("no param must pass through unchanged, got (%q,%v,%v)", name, def != nil, explicit)
	}

	// an explicit sibling session resolves and is marked explicit, so the handler
	// serves it even though it is NOT live right now — the point of the tabs.
	name, def, explicit = resolveRequestedSession(reg, "asia", kernel.SessionNY, ny)
	if name != kernel.SessionAsia || !explicit {
		t.Fatalf("asia must resolve explicitly, got (%q,%v)", name, explicit)
	}
	if def == nil || def.Name != kernel.SessionAsia {
		t.Fatal("the returned SessionDef must be ASIA's, not the live one")
	}

	// case-insensitive + whitespace tolerant (query strings are user input)
	if n, _, _ := resolveRequestedSession(reg, "  LoNdOn  ", kernel.SessionNY, ny); n != kernel.SessionLondon {
		t.Fatalf("expected LONDON, got %q", n)
	}

	// garbage falls back to the live session instead of 500ing or echoing back
	name, def, explicit = resolveRequestedSession(reg, "NOPE", kernel.SessionNY, ny)
	if name != kernel.SessionNY || def != ny || explicit {
		t.Fatalf("unknown session must fall back to live, got (%q,%v)", name, explicit)
	}

	// night (no live session) + no param → still no session, no crash
	if n, d, e := resolveRequestedSession(reg, "", "", nil); n != "" || d != nil || e {
		t.Fatalf("night with no param must stay empty, got (%q,%v,%v)", n, d != nil, e)
	}
	// night + an explicit request → readable (that is how the owner reviews a
	// session's plan outside its hours)
	if n, _, e := resolveRequestedSession(reg, "NY", "", nil); n != kernel.SessionNY || !e {
		t.Fatalf("explicit request at night must resolve, got (%q,%v)", n, e)
	}
}

// planRules must RESOLVE the rulebook. With no config it returns exactly what the
// old hardcodes emitted, so the shipped path is unchanged; with config it follows
// the strategy and per-session overrides.
func TestW15PlanRulesFallsBackToShippedDefaults(t *testing.T) {
	var dp *store.DayPlanConfig // the unknown-trader case inside planRules
	if got := dp.AcceptanceRuleFor(kernel.SessionNY); got != store.DefaultAcceptanceRule {
		t.Errorf("acceptance = %q, want the previously-hardcoded %q", got, store.DefaultAcceptanceRule)
	}
	if got := dp.PlanModeFor(kernel.SessionNY); got != "advisory" {
		t.Errorf("mode = %q, want the previously-hardcoded advisory", got)
	}
	// replans_left was hardcoded as 2-(version-1)
	for version, want := range map[int]int{1: 2, 2: 1, 3: 0, 9: 0} {
		if got := maxI(0, dp.ReplanCapFor(kernel.SessionNY)-(version-1)); got != want {
			t.Errorf("version %d → replans_left %d, want %d (matches the old formula)", version, got, want)
		}
	}
}

// W16/R1 — a plan that is no longer active has no armed plays.
func TestW16ScenarioStatusExpiresWithThePlan(t *testing.T) {
	doc := kernel.PlanDoc{Scenarios: []kernel.PlanScenario{{ID: "S1"}, {ID: "S2"}}}
	stored := map[string]string{"S1": kernel.ScenarioArmed, "S2": kernel.ScenarioTriggered}

	// active → untouched
	if got := scenarioStatusForLifecycle(stored, "active", doc); got["S1"] != kernel.ScenarioArmed {
		t.Fatalf("an active plan must keep its live statuses, got %v", got)
	}
	// empty lifecycle behaves as active (back-compat)
	if got := scenarioStatusForLifecycle(stored, "", doc); got["S2"] != kernel.ScenarioTriggered {
		t.Fatalf("an unset lifecycle must not expire anything, got %v", got)
	}
	// rolled / dead → every scenario expires
	for _, lc := range []string{"expired", "died", "superseded", "no_trade"} {
		got := scenarioStatusForLifecycle(stored, lc, doc)
		if got["S1"] != kernel.ScenarioExpired || got["S2"] != kernel.ScenarioExpired {
			t.Errorf("lifecycle %q must expire every scenario, got %v", lc, got)
		}
	}
	// a doc with no scenarios cannot invent rows
	if got := scenarioStatusForLifecycle(stored, "expired", kernel.PlanDoc{}); len(got) != len(stored) {
		t.Errorf("no scenarios in the doc → fall back to the stored map, got %v", got)
	}
}
