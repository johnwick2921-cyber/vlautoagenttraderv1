package trader

import (
	"strings"
	"testing"
)

// ── ONE OPEN POSITION PER INSTRUMENT (owner ruling 2026-09-03) ──────────────
//
// The one-live-arm leg refused only the OPPOSITE side, because on a netting
// account an opposite-side fill silently nets the position. A same-side add was
// "outside this guard's scope" (armed_executor.go:601), and the same-version
// re-arm block lives in the store, so a NEW plan version could re-authorize a
// terminal row and add to a position that was still open.
//
// The ruling closes both: while any position is open, every arm and every
// decision-path open is refused, whatever the side and whatever the version.

func openIntent(side string, version int, scenario string, action string) EntryIntent {
	return EntryIntent{
		Path: "arm", Action: action,
		Entry: 29285, Stop: 29362.5, Target: 29050,
		CitedScenario: scenario, ScenarioDir: strings.TrimPrefix(action, "open_"),
		OpenPositionSide: side, OpenPositionID: 591,
		OpenPositionVersion: version, OpenPositionScenario: "S1",
	}
}

// THE PIN — v2 short open, v3 S1 SHORT arm. The old leg allowed this.
func TestOneLivePositionRefusesASameSideAdd(t *testing.T) {
	r, refused := EntryGate(openIntent("short", 2, "S1", "open_short"))
	if !refused {
		t.Fatalf("a same-side arm while a position is open must be REFUSED: %q", r)
	}
	for _, want := range []string{"position 591 open", "v2 S1 short", "no adds, no flips"} {
		if !strings.Contains(r, want) {
			t.Errorf("refusal missing %q:\n  %s", want, r)
		}
	}
}

// The opposite side stays refused, as it always was.
func TestOneLivePositionStillRefusesTheOppositeSide(t *testing.T) {
	r, refused := EntryGate(openIntent("short", 2, "S1", "open_long"))
	if !refused {
		t.Fatalf("an opposite-side arm must stay refused: %q", r)
	}
	if !strings.Contains(r, "position 591 open") {
		t.Errorf("the refusal must name the open position: %s", r)
	}
}

// Flat → allowed. This is the pin that keeps the leg from refusing everything.
func TestOneLivePositionAllowsWhenFlat(t *testing.T) {
	in := openIntent("", 0, "S1", "open_short")
	in.OpenPositionSide, in.OpenPositionID = "", 0
	if r, refused := EntryGate(in); refused {
		t.Fatalf("flat must be allowed: %q", r)
	}
}

// The DECISION path is governed too — the ruling says every open, not every arm.
func TestOneLivePositionGovernsTheDecisionPath(t *testing.T) {
	in := openIntent("short", 2, "S1", "open_short")
	in.Path = "decision"
	r, refused := EntryGate(in)
	if !refused || !strings.Contains(r, "position 591 open") {
		t.Errorf("a decision-path open while a position is open must be refused, got refused=%v %q", refused, r)
	}
}

// An explicitly authored EXIT leg is how a position is flattened and is NOT an
// open. Refusing it would strand the position — and exits are out of scope for
// this dispatch by its own stop-lines.
func TestOneLivePositionDoesNotBlockAnExitLeg(t *testing.T) {
	in := openIntent("short", 2, "S1", "open_long")
	in.IsExitLeg = true
	if r, refused := EntryGate(in); refused {
		t.Fatalf("an exit leg must still be allowed to flatten: %q", r)
	}
}

// A position with no recorded version renders honestly rather than "v0".
func TestOneLivePositionUnrecordedVersion(t *testing.T) {
	r, refused := EntryGate(openIntent("short", 0, "S1", "open_short"))
	if !refused {
		t.Fatal("still refused")
	}
	if strings.Contains(r, "v0") {
		t.Errorf("a position with no recorded version must not render v0: %s", r)
	}
	if !strings.Contains(r, "version not recorded") {
		t.Errorf("say what is unknown: %s", r)
	}
}
