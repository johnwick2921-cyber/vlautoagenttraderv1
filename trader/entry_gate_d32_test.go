// (3) D32, corrected — the counter was never the missing half.
//
// entryGateDecisionTelemetry DOES call telemetry.IncGateBlock (entry_gate.go),
// so gate blocks were counted all along. What it did not do was SAY anything:
// the decision-path refusal was recorded in decision_records and the counter,
// and nowhere in the log. Its sibling (the arm path) logs a 🚦 line, so a
// reader comparing the two saw refusals from one path only and reasonably
// concluded the other was not firing.
//
// A9: every refusal is one line with the reason.

package trader

import (
	"os"
	"strings"
	"testing"
)

func TestEntryGateDecisionRefusalIsLogged(t *testing.T) {
	src, err := os.ReadFile("entry_gate.go")
	if err != nil {
		t.Fatalf("read entry_gate.go: %v", err)
	}
	text := string(src)

	i := strings.Index(text, "func entryGateDecisionTelemetry(")
	if i < 0 {
		t.Fatal("entryGateDecisionTelemetry not found")
	}
	end := strings.Index(text[i:], "\n}\n")
	if end < 0 {
		t.Fatal("could not bound the function")
	}
	body := text[i : i+end]

	if !strings.Contains(body, "IncGateBlock") {
		t.Error("the counter must stay — D32's premise was that it was missing, and it never was")
	}
	if !strings.Contains(body, "logWarnf") {
		t.Fatal("a decision-path gate refusal must be LOGGED, not only counted (A9)")
	}
	if !strings.Contains(body, "🚦") {
		t.Error("use the same 🚦 marker as the arm path, or the two paths cannot be compared in one grep")
	}
}
