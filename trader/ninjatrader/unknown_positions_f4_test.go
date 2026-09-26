// W117 F4 (c20d0a82 trader half) — no NT8 snapshot AND no confirmed entry fill
// is UNKNOWN, never flat. A fabricated empty book let callers read "no
// position" as truth and re-enter on a position NT8 holds. RED = restore the
// old `return []map[string]interface{}{}, nil` fallback → the unknown pin
// fails with a nil error.

package ninjatrader

import (
	"strings"
	"testing"

	ntwire "nofx/provider/ninjatrader"
)

func TestGetPositionsUnknownBeforeFirstSnapshot(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	tr := NewTCPTrader(s, "MNQ", "Sim101")

	// No snapshot, no fill: UNKNOWN.
	pos, err := tr.GetPositions()
	if err == nil || !strings.Contains(err.Error(), "account positions unknown") {
		t.Fatalf("a trader with no snapshot and no fill must report UNKNOWN, got pos=%v err=%v", pos, err)
	}

	// A fresh snapshot (even empty) is KNOWN-flat.
	s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{})
	pos, err = tr.GetPositions()
	if err != nil {
		t.Fatalf("a fresh empty snapshot is known-flat, got err=%v", err)
	}
	if len(pos) != 0 {
		t.Fatalf("known-flat must be empty, got %d", len(pos))
	}
}
