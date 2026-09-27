package trader

import (
	"testing"
	"time"

	"nofx/store"
	"nofx/telemetry"
)

// TestDecisionSaveFailureIsLoudAndCounted is the P1-D pin (audit 2026-09-26):
// a failed decision-record save must WARN (auto-promotes to the WARN+ DB sink)
// and record a counter on the errors API — never an INFO line that journald
// prunes at the 2G cap, and never silent. The 12 discard sites ride saveDecision,
// so this single pin covers every one of them.
func TestDecisionSaveFailureIsLoudAndCounted(t *testing.T) {
	at := plannerTestTrader(t)
	at.id = "p1d-decision-save"
	if err := at.store.GormDB().Exec("DROP TABLE IF EXISTS decision_records").Error; err != nil {
		t.Fatalf("break decision table: %v", err)
	}
	record := &store.DecisionRecord{
		TraderID:  at.id,
		Account:   "Sim101",
		Timestamp: time.Now().UTC(),
	}
	if err := at.saveDecision(record); err == nil {
		t.Fatalf("a broken decision table must fail the save, got nil")
	}
	found := false
	for _, row := range telemetry.ErrorSummary(at.id) {
		if row.Type == "decision_save_failed" && row.DecisionsLost >= 1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("the failed decision save must be RECORDED (type=decision_save_failed, decisions_lost>=1); ErrorSummary(%q) has no such row", at.id)
	}
}
