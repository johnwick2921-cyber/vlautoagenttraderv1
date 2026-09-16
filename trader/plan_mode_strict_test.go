package trader

import (
	"strings"
	"testing"
)

// ── R4 — plan_mode "strict" (OWNER RULING 2026-09-03) ────────────────────────
//
// MEASURED FIRST: strict was DOCUMENTED and never implemented. It appears only
// in a doc comment listing "advisory | direction | strict"; `PlanModeFor`
// (store/strategy.go:1387) returns a saved "strict" UNCHANGED — there is no
// self-heal and no "normal" mode — and no consumer anywhere compared against
// it. So this is a FIRST IMPLEMENTATION, i.e. a NEW GATE, ruled by the owner on
// 2026-09-03 rather than a restoration of deprecated behaviour.
//
// Semantics per the ruling: only plan scenarios execute · arm path only ·
// decision-path market entries refused · direction must equal the scenario's ·
// logged as "refused: strict".

func strictIntent(path, action string) EntryIntent {
	return EntryIntent{
		Path: path, Action: action, Symbol: "MNQ",
		Entry: 29000, Stop: 28960, Target: 29120, ATR5m: 10, MinRR: 2.0,
		PlanMode: "strict", CitedScenario: "S2", ScenarioDir: "long",
	}
}

// PIN 1 — strict REFUSES a decision-path market entry, whatever else is fine.
func TestStrictRefusesTheDecisionPath(t *testing.T) {
	reason, refused := EntryGate(strictIntent("decision", "open_long"))
	if !refused {
		t.Fatalf("strict must refuse the decision path, got allow (reason %q)", reason)
	}
	if !strings.Contains(reason, "strict") {
		t.Errorf("the refusal must name strict so the journal is answerable: %q", reason)
	}
	t.Logf("decision path: %s", reason)
}

// PIN 2 — strict ALLOWS an arm whose side matches the scenario it cites.
func TestStrictAllowsAnArmMatchingItsScenario(t *testing.T) {
	reason, refused := EntryGate(strictIntent("arm", "open_long")) // scenario dir = long
	if refused {
		t.Fatalf("strict must allow an arm matching its scenario, refused: %s", reason)
	}
}

// PIN 3 — strict refuses an arm whose side contradicts its scenario, and an arm
// that cites NO scenario at all ("only plan scenarios execute").
func TestStrictRefusesMismatchedOrUncitedArms(t *testing.T) {
	mism := strictIntent("arm", "open_short") // scenario says long
	if reason, refused := EntryGate(mism); !refused {
		t.Errorf("strict must refuse a short arm on a long scenario, got allow (%q)", reason)
	}
	uncited := strictIntent("arm", "open_long")
	uncited.CitedScenario, uncited.ScenarioDir = "", ""
	reason, refused := EntryGate(uncited)
	if !refused {
		t.Fatalf("strict: an arm citing NO scenario must be refused — only plan scenarios execute (got %q)", reason)
	}
	if !strings.Contains(reason, "strict") {
		t.Errorf("refusal must name strict: %q", reason)
	}
	t.Logf("uncited arm: %s", reason)
}

// PIN 4 — the other modes are UNCHANGED: this wave adds a gate, it does not
// widen the existing ones.
func TestStrictDoesNotChangeAdvisoryOrDirection(t *testing.T) {
	for _, mode := range []string{"advisory", "direction", ""} {
		in := strictIntent("decision", "open_long")
		in.PlanMode = mode
		if reason, refused := EntryGate(in); refused && strings.Contains(reason, "strict") {
			t.Errorf("mode %q must not hit the strict leg: %q", mode, reason)
		}
	}
}
