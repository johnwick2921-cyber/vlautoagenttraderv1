package ninjatrader

import (
	"path/filepath"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// C8 (2026-08-25) — entries whose signal never produced a fill within
// entryConfirmGraceMs are dropped from the pending map by the reconcile sweep:
// they must never linger as would-be positions.
func TestReconcileSweepsStaleUnconfirmedEntries(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "sweep.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	s := ntwire.NewTCPServer(nil)
	s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{}) // ok=true, flat
	tr := NewTCPTrader(s, "MNQ", "Sim101")

	now := time.Now().UTC().UnixMilli()
	tr.pendingMu.Lock()
	tr.pending["stale-signal"] = "LONG"
	tr.pendingAt["stale-signal"] = now - entryConfirmGraceMs - 5_000
	tr.pending["fresh-signal"] = "SHORT"
	tr.pendingAt["fresh-signal"] = now
	tr.pendingMu.Unlock()

	tr.reconcilePositions("trader-sweep", "nt", "ninjatrader", st)

	tr.pendingMu.Lock()
	defer tr.pendingMu.Unlock()
	if _, ok := tr.pending["stale-signal"]; ok {
		t.Fatal("stale unconfirmed entry must be swept from pending")
	}
	if _, ok := tr.pending["fresh-signal"]; !ok {
		t.Fatal("fresh pending entry must survive the sweep")
	}
	if _, ok := tr.pendingAt["stale-signal"]; ok {
		t.Fatal("stale pendingAt stamp must be swept")
	}
}
