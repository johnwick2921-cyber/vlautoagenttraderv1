// B1 — stop-widen ban: a stop modification may only TIGHTEN (reduce risk), never
// widen. The pure predicate + the MoveStopToBreakeven refuse path.
package ninjatrader

import (
	"strings"
	"testing"

	ntwire "nofx/provider/ninjatrader"
)

func TestStopWouldWiden(t *testing.T) {
	// LONG (stop below price): tighten = move UP toward price; widen = move DOWN.
	if !stopWouldWiden("long", 100, 95) {
		t.Error("long 100→95 (lower) must WIDEN")
	}
	if stopWouldWiden("long", 100, 105) {
		t.Error("long 100→105 (higher) tightens — not a widen")
	}
	if stopWouldWiden("long", 100, 100) {
		t.Error("long equal is not a widen")
	}
	// SHORT (stop above price): tighten = move DOWN toward price; widen = move UP.
	if !stopWouldWiden("short", 100, 105) {
		t.Error("short 100→105 (higher) must WIDEN")
	}
	if stopWouldWiden("SHORT", 100, 95) {
		t.Error("short 100→95 (lower) tightens — not a widen (case-insensitive)")
	}
	if stopWouldWiden("short", 100, 100) {
		t.Error("short equal is not a widen")
	}
}

func TestMoveStopToBreakeven_WidenRefused(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	tr := NewTCPTrader(s, "MNQ", "Sim101")
	_ = tr.SetStopLoss("MNQ", "long", 1, 29900) // current resting stop
	tr.mu.Lock()
	tr.lastEntrySignalID = "sig-1" // simulate an open entry
	tr.mu.Unlock()

	// A widening move (lower stop for a long) must be REFUSED before any wire send.
	err := tr.MoveStopToBreakeven("long", 29850)
	if err == nil || !strings.Contains(err.Error(), "stop-widen ban") {
		t.Fatalf("widen must be refused with the stop-widen ban error, got: %v", err)
	}
	// The tracked stop must stay UNCHANGED after a refused widen.
	tr.mu.Lock()
	got := tr.stopLoss[keyFor("MNQ", upperSideStr("long"))]
	tr.mu.Unlock()
	if got != 29900 {
		t.Fatalf("tracked stop must stay 29900 after a refused widen, got %.2f", got)
	}
}
