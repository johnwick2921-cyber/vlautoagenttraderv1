package trader

import (
	"os"
	"strings"
	"testing"
)

// ── W-ONE-BUTTON M2 — the Guide does not lie about the binary (L5) ─────────
//
// The guide documents the 🔒 boot line and the gate-block names this wave
// emits. Both are pinned to the binary: the documented line carries the real
// line's fields in the real order, and every gate name the code counts has a
// human label in the gate-blocks panel.
func TestGuideDocumentsTheMaintenanceBootLineAndGateNames(t *testing.T) {
	withMaintenanceDir(t)
	real := MaintenanceBootLine(nil) // 🔒 maintenance: hold=clear job=n/a since=n/a addon_ack=n/a
	b, err := os.ReadFile("../web/src/guide/content/status.ts")
	if err != nil {
		t.Fatal(err)
	}
	guide := string(b)
	i := strings.Index(guide, "🔒 maintenance: ")
	if i < 0 {
		t.Fatal("the guide's boot ledger does not document the 🔒 maintenance line")
	}
	doc := guide[i:]
	pos := 0
	for _, field := range []string{"hold=", "job=", "since=", "addon_ack="} {
		if !strings.Contains(real, field) {
			t.Fatalf("fixture: the real line lost %q: %s", field, real)
		}
		k := strings.Index(doc[pos:], field)
		if k < 0 {
			t.Fatalf("the guide's 🔒 line is missing %q (or has it out of order); real line: %s", field, real)
		}
		pos += k + len(field)
	}
	p, err := os.ReadFile("../web/src/components/plan/GateBlocksPanel.tsx")
	if err != nil {
		t.Fatal(err)
	}
	for _, gate := range []string{"maintenance_hold", "maintenance_drop", "maintenance_drop_attempted"} {
		if !strings.Contains(string(p), "\n  "+gate+": {") {
			t.Errorf("GateBlocksPanel has no human label for gate %q", gate)
		}
	}
}
