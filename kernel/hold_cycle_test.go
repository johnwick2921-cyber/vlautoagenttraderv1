// F10 — a held cycle now returns a non-nil FullDecision carrying the gate reason
// (SkipReason) with no Decisions, so the caller can record the truth instead of a
// silently-empty success.
package kernel

import "testing"

func TestHoldCycle_CarriesReasonNoDecisions(t *testing.T) {
	fd := holdCycle("cme_closed")
	if fd == nil {
		t.Fatal("holdCycle must return a non-nil FullDecision so the caller can stamp the reason")
	}
	if fd.SkipReason != "cme_closed" {
		t.Errorf("SkipReason = %q, want cme_closed", fd.SkipReason)
	}
	if len(fd.Decisions) != 0 {
		t.Errorf("a held cycle must carry no actionable decisions, got %d", len(fd.Decisions))
	}
}
