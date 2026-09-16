package trader

import (
	"os"
	"regexp"
	"testing"
)

// ── W2 D3 — NOTHING IS REFUSED. THE ARM PATH CANNOT READ THE LABEL. ─────────
//
// A31: an excluded scenario is authorized, armed, placed and traded exactly as
// a permitted one. The behavioural form of that pin ("golden diff on the arm
// path is empty") is only as good as the fixture that produces the diff. The
// STRUCTURAL form is stronger and cannot be satisfied by a lucky fixture: if
// no file on the arm-authorization path can reference the label, no file on
// it can act on the label. This test fails the moment one does.
//
// The list is the files C1 measured as the path between "condition met" and
// "arm authorized" — the same files that hold zero day_type reads. If the
// label ever needs to become a gate, E3 decides that, and the change lands
// here first, RED, with the evidence beside it.

var fadeSymbol = regexp.MustCompile(`FadePermission|FadePermitted|FadeExclusion|FadeFacts|FadeVerdict|FadeStamp|fade_permitted|fade_exclusions`)

var armAuthorizationPath = []string{
	"armed_executor.go",
	"entry_gate.go",
	"auto_trader_orders.go",
	"session_risk.go",
	"../kernel/risk_limits.go",
	"../kernel/plan_authored_invalidation.go",
	"../kernel/engine_analysis.go",
}

func TestArmPathCannotReadTheFadeLabel(t *testing.T) {
	checked := 0
	for _, f := range armAuthorizationPath {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("read %s: %v — the path list names a file that is not there", f, err)
		}
		checked++
		if loc := fadeSymbol.FindIndex(b); loc != nil {
			line := 1 + countNewlines(b[:loc[0]])
			t.Errorf("D3 NO-REFUSAL: %s:%d references the fade label (%q). The arm path must not be able to read it — a label that a gate can see is one decision away from being a gate.",
				f, line, string(b[loc[0]:loc[1]]))
		}
	}
	if checked < 5 {
		t.Fatalf("only %d files checked — the pin is going vacuous", checked)
	}
}

func countNewlines(b []byte) int {
	n := 0
	for _, c := range b {
		if c == '\n' {
			n++
		}
	}
	return n
}
