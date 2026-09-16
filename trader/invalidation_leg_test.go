package trader

import (
	"strings"
	"testing"
)

// ── INVALIDATION-WIRED (owner ruling 2026-09-03) ────────────────────────────
//
// T1, THE PIN. 2026-09-03, NY v2 S1:
//
//   08:50:54  🎯 scenario S1 → ≈invalidated @ 29285.00 (price accepted through
//             the level against the trade — display-only estimate, never
//             execution-wired)
//   09:02:54  ⚔️ armed NY S1 leg 1 short limit 29285.00
//   09:03:53  filled
//   09:20:45  stopped at the widened stop — −$140
//
// The system reached the verdict twelve minutes before it armed the trade the
// verdict condemned, and had no leg that read its own conclusion. The label
// "never execution-wired" was accurate and was the defect.
//
// armed_orders row 35 is the ledger record: version 2, S1, short, entry
// 29285.00, state filled.

func TestInvalidationPinNYv2S1(t *testing.T) {
	// the 09:02 arm intent, as entryGateForArm would build it
	in := EntryIntent{
		Path: "arm", Action: "open_short", Symbol: "MNQ",
		Entry: 29285.00, Stop: 29362.50, Target: 29130.00,
		ATR5m: 15.2, MinRR: 2.0, MinSLMult: 1.0,
		CitedScenario: "S1", ScenarioDir: "short", ScenarioCond: "reject",
		// the 08:50 evaluator state, from the evaluator itself
		ScenarioInvalidation: func(id string) (InvalidationVerdict, bool) {
			if id != "S1" {
				return InvalidationVerdict{}, true
			}
			return InvalidationVerdict{
				Invalidated: true, AtCT: "08:50", Anchor: 29285.00,
				Reason: "price accepted through the level against the trade",
			}, true
		},
	}
	reason, refused := EntryGate(in)
	if !refused {
		t.Fatalf("the 09:02 arm must be REFUSED — the system had already marked S1 invalidated at 08:50. got reason=%q", reason)
	}
	for _, want := range []string{"invalidated at 08:50", "accepted through 29285.00", "S1"} {
		if !strings.Contains(reason, want) {
			t.Errorf("refusal missing %q:\n  %s", want, reason)
		}
	}
}

// T2 — the evaluator could not run. FAIL-OPEN with a line: an unresolved check
// is not a refusal (the zero-threshold rule).
func TestInvalidationUnavailableFailsOpenWithALine(t *testing.T) {
	var noted string
	in := EntryIntent{
		Path: "arm", Action: "open_short", Symbol: "MNQ",
		Entry: 29285.00, Stop: 29362.50, Target: 29130.00,
		ATR5m: 15.2, MinRR: 2.0, MinSLMult: 1.0,
		CitedScenario: "S1", ScenarioDir: "short",
		ScenarioInvalidation: func(string) (InvalidationVerdict, bool) {
			return InvalidationVerdict{}, false // no bars, no verdict
		},
		OnInvalidationUnavailable: func(note string) { noted = note },
	}
	if reason, refused := EntryGate(in); refused {
		t.Fatalf("an unresolved invalidation check must NOT refuse: %q", reason)
	}
	if !strings.Contains(noted, "invalidation check unavailable") {
		t.Errorf("the fail-open must be LOUD, got %q", noted)
	}
}

// T3 — a scenario the evaluator has NOT invalidated passes exactly as before.
func TestInvalidationNotInvalidatedIsUnchanged(t *testing.T) {
	in := EntryIntent{
		Path: "arm", Action: "open_short", Symbol: "MNQ",
		Entry: 29285.00, Stop: 29362.50, Target: 29130.00,
		ATR5m: 15.2, MinRR: 2.0, MinSLMult: 1.0,
		CitedScenario: "S1", ScenarioDir: "short",
		ScenarioInvalidation: func(string) (InvalidationVerdict, bool) {
			return InvalidationVerdict{Invalidated: false}, true
		},
	}
	if reason, refused := EntryGate(in); refused {
		t.Fatalf("a live scenario must pass: %q", reason)
	}
	// and with NO resolver at all the leg is simply off (the decision path)
	in.ScenarioInvalidation = nil
	if reason, refused := EntryGate(in); refused {
		t.Fatalf("no resolver = leg off, got %q", reason)
	}
}

// The leg is ARM-path only, per the dispatch. The decision path never wires the
// resolver, and this pins that a decision intent carrying one is still governed
// only where the wave says — belt and braces against a later copy-paste.
func TestInvalidationLegIsArmPathOnly(t *testing.T) {
	res := func(string) (InvalidationVerdict, bool) {
		return InvalidationVerdict{Invalidated: true, AtCT: "08:50", Anchor: 29285}, true
	}
	// Target 29050 clears the decision path's default 3.0 R:R floor on a 77.5
	// pt risk — the first draft of this test used 29130 and was refused on R:R,
	// which would have "passed" for entirely the wrong reason.
	arm := EntryIntent{
		Path: "arm", Action: "open_short", Entry: 29285, Stop: 29362.5, Target: 29050,
		CitedScenario: "S1", ScenarioDir: "short", ScenarioInvalidation: res,
	}
	reason, refused := EntryGate(arm)
	if !refused || !strings.Contains(reason, "invalidated") {
		t.Errorf("the arm path must refuse ON INVALIDATION, got refused=%v reason=%q", refused, reason)
	}
	dec := arm
	dec.Path = "decision"
	// Assert on the REASON, not merely on refused: another leg may legitimately
	// refuse a decision intent, and this test is only about this leg's scope.
	decReason, decRefused := EntryGate(dec)
	if decRefused && strings.Contains(decReason, "invalidated") {
		t.Errorf("the decision path is out of this wave's scope — it must not refuse on invalidation, got %q", decReason)
	}
}

// The refusal must be COUNTED under its own class, not swept into "other".
func TestInvalidationRefusalIsItsOwnClass(t *testing.T) {
	reason, refused := EntryGate(EntryIntent{
		Path: "arm", Action: "open_short", Entry: 29285, Stop: 29362.5, Target: 29050,
		CitedScenario: "S1", ScenarioDir: "short",
		ScenarioInvalidation: func(string) (InvalidationVerdict, bool) {
			return InvalidationVerdict{Invalidated: true, AtCT: "08:50", Anchor: 29285}, true
		},
	})
	if !refused {
		t.Fatal("expected a refusal")
	}
	if got := armRefusalClass(reason); got != "invalidated" {
		t.Errorf("armRefusalClass(%q) = %q, want \"invalidated\" — an unclassified refusal vanishes into \"other\"", reason, got)
	}
}
