// F10 — no more ghost records: a guardrail-skip cycle must be saved with a truthful
// execution_status + reason, never as a silently-empty success (the bug: 21 records
// with an empty prompt AND success=1).
package trader

import (
	"testing"

	"nofx/store"
)

func TestStampGuardrailSkip_TellsTheTruth(t *testing.T) {
	// A freshly-built cycle record starts optimistic (Success=true, empty prompt) —
	// exactly the ghost that used to be persisted verbatim on a guardrail hold.
	rec := &store.DecisionRecord{Success: true, ExecutionLog: []string{}}

	stampGuardrailSkip(rec, "daily_guardrail")

	if rec.Success {
		t.Error("a guardrail-skip record must NOT be success=1 (that was the ghost)")
	}
	if rec.ExecutionStatus != "guardrail_skip" {
		t.Errorf("execution_status = %q, want guardrail_skip", rec.ExecutionStatus)
	}
	if rec.RiskCheckPassed {
		t.Error("risk_check_passed must be false on a guardrail skip")
	}
	if rec.RiskCheckError != "daily_guardrail" {
		t.Errorf("risk_check_error = %q, want the gate reason", rec.RiskCheckError)
	}
	if rec.ErrorMessage != "guardrail_skip: daily_guardrail" {
		t.Errorf("error_message = %q, want the labeled reason", rec.ErrorMessage)
	}
	// An empty prompt is explicitly OK on such a record (pre-prompt gates never
	// build a prompt) — the record still tells the truth via status + reason.
	if rec.SystemPrompt != "" || rec.InputPrompt != "" {
		t.Error("this test's record should have no prompt; empty prompt is acceptable")
	}
}
