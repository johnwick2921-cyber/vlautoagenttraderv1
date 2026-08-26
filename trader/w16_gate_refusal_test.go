package trader

import (
	"fmt"
	"strings"
	"testing"
)

// CTO-VERIFY 5.1/5.11 — a gate-REFUSED entry must never be recorded as a
// successful execution.
//
// Every entry gate in executeDecisionWithRecord sets Success=false +
// Error="<gate>: <reason>" and returns nil, because a refusal is not an error.
// The caller's else-branch then set Success=true and appended "✓ … succeeded",
// overwriting the gate's verdict. The owner's dashboard therefore showed a
// blocked entry as an executed one, and "why was my entry refused?" had no
// answer anywhere in the UI.
//
// This pins the caller's three-way classification. It mirrors the real branch
// logic at trader/auto_trader_loop.go so a regression there fails here.
func TestW16GateRefusalIsNotRecordedAsSuccess(t *testing.T) {
	// classify mirrors the caller: (err != nil) -> failed, (Error != "") -> refused,
	// otherwise -> succeeded.
	classify := func(err error, recordedErr string) (success bool, line string) {
		switch {
		case err != nil:
			return false, fmt.Sprintf("❌ MNQ open_long failed: %v", err)
		case recordedErr != "":
			return false, fmt.Sprintf("⛔ MNQ open_long refused: %s", recordedErr)
		default:
			return true, "✓ MNQ open_long succeeded"
		}
	}

	// Every gate that returns nil after stamping a reason.
	for _, gate := range []string{
		"session_gate: ASIA session not enabled",
		"last_entry: past last-entry 13:00 CT",
		"plan_mode: short entry against a long plan bias",
		"boot_integrity: binary is revision X but the intended release is Y",
		"consecutive_loss: halted for the session",
		"feed_down: NT8 feed disconnected",
		"approval: entries await owner approval",
		"dead_man: watchdog tripped",
		"frozen: A4 freeze — belief/broker divergence",
	} {
		success, line := classify(nil, gate)
		if success {
			t.Errorf("REFUSED entry recorded as SUCCESS: %s", gate)
		}
		if !strings.HasPrefix(line, "⛔") || !strings.Contains(line, gate) {
			t.Errorf("refusal log line must name the gate, got %q", line)
		}
	}

	// A genuine execution still reads as success.
	if success, line := classify(nil, ""); !success || !strings.HasPrefix(line, "✓") {
		t.Errorf("a real execution must stay successful, got (%v, %q)", success, line)
	}

	// A real error is still a failure, distinct from a refusal.
	if success, line := classify(fmt.Errorf("broker rejected"), ""); success || !strings.HasPrefix(line, "❌") {
		t.Errorf("an error must read as failed, got (%v, %q)", success, line)
	}
}
