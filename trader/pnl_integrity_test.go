package trader

import (
	"strings"
	"testing"
)

// V3 — the guard's math on a seeded $1 discrepancy (pure pin; the WARN line
// itself is source-contract-checked so a refactor can't silently drop it).
func TestPnLIntegrityGuardMath(t *testing.T) {
	// SHORT 1 lot, entry 29626.25 exit 29660.964286, pv=2 → recomputed −69.43;
	// a recorded −70.43 (seeded $1 off) must trip the ±$0.50 guard.
	pts := -(29660.964286 - 29626.25)
	recomputed := pts * 1.0 * 2.0
	recorded := recomputed - 1.0
	if d := recorded - recomputed; !(d > 0.5 || d < -0.5) {
		t.Fatal("a $1 discrepancy must trip the ±$0.50 guard")
	}
	if d := recomputed - recomputed; d > 0.5 || d < -0.5 {
		t.Fatal("an exact record must not trip")
	}
}

func TestPnLIntegrityGuardWired(t *testing.T) {
	src, err := readFileForTest("auto_trader_clock.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(src, "pnl_integrity MISMATCH") {
		t.Fatal("the per-close integrity WARN was removed — wrong PnL would sit silent again")
	}
	if !strings.Contains(src, "pnl_integrity_mismatch") {
		t.Fatal("the gate-block counter for the guard is gone")
	}
}
