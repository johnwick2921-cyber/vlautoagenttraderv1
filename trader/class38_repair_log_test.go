package trader

import (
	"errors"
	"strings"
	"testing"
)

// CLASS 38 F6 (2026-09-01) — a repair that returns unparseable output used to
// log one bare sentence ("repair returned unparseable output — falling back to
// a full re-author next attempt"). Rejected-prompt row 79 is exactly that path:
// the repair echoed a token the hint had named, came back malformed, and the
// journal recorded nothing about WHAT came back or WHICH defect was being
// repaired — so the next forensics run had to reconstruct it from the DB.
//
// The line is rendered by a pure function so this fixture pins its wording.
// Retry semantics are NOT changed (stop-line): the fallback to one full
// re-author stays exactly as it was.
func TestClass38RepairUnparseableLineCarriesEvidence(t *testing.T) {
	raw := `{"reasoning":"the tape is one-sided","scenarios":[{"id":"S1","confirm2":{"rule":"2x5m"` // truncated by the model
	reason := `scenario[0].confirm2.rule "1m_mss" not allowed for breakdown_continue — entry law: 1 confirming close + displacement ≥ BD_MIN_DISP_ATR×ATR5m OR stop-entry (E7); 2x5m_close legal ONLY here`
	line := repairUnparseableLine(raw, reason, errors.New("plan JSON unmarshal: unexpected end of JSON input"))

	for _, want := range []string{
		"repair returned UNPARSEABLE output",
		"falling back to a full re-author",
		"plan JSON unmarshal",       // the parse error itself
		"was repairing:",            // the defect the repair was aimed at
		"not allowed for breakdown", // …quoted from that defect
		"raw_head=",                 // the head of what the model actually sent
		"the tape is one-sided",     // …its distinctive content (quoted via %q, so
		"reasoning",                 //    the JSON quotes arrive backslash-escaped)
	} {
		if !strings.Contains(line, want) {
			t.Errorf("repair-unparseable line missing %q\n  line: %s", want, line)
		}
	}

	// The head must be bounded — a 30k-char malformed response must not land
	// in the journal whole (class 12, log-flood retention).
	long := strings.Repeat("x", 30000)
	bounded := repairUnparseableLine(long, reason, errors.New("boom"))
	if len(bounded) > 1200 {
		t.Errorf("repair-unparseable line is unbounded (%d chars) — a malformed 30k response would flood the journal", len(bounded))
	}
	if !strings.Contains(bounded, "xxxx") {
		t.Error("the bounded line must still carry the head of the raw response")
	}
}
