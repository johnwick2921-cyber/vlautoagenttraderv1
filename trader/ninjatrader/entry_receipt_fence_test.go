// W117 F1 trader half — a positions snapshot received at-or-before the latest
// entry receipt is STALE: adapter replacement must not read the pre-entry flat
// snapshot as truth (double-open). GetPositions fences it and falls to the
// fill-derived cache, which knows the entry.

package ninjatrader

import (
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
)

// TestGetPositionsStaleFlatSnapshotFallsToFillCache: the account is KNOWN-flat
// at t0, then an entry executes (t1 > t0). The flat snapshot predates the
// entry receipt, so GetPositions must serve the fill-derived position, NOT the
// stale flat book. RED = fence off (read PositionsFor without the entry
// comparison) → the stale flat snapshot serves as truth and the pin sees zero
// positions.
func TestGetPositionsStaleFlatSnapshotFallsToFillCache(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{}) // known-flat at t0
	tr := NewTCPTrader(s, "MNQ", "Sim101")

	s.NoteEntryExecution("MNQ", "Sim101", "sig-1", 1) // entry receipt at t1 > t0
	tr.mu.Lock()
	tr.hasFill = true
	tr.lastFill = ntwire.FillPayload{Side: "long", Quantity: 2, FillPrice: 100}
	tr.mu.Unlock()

	pos, err := tr.GetPositions()
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if len(pos) != 1 {
		t.Fatalf("the stale flat snapshot must not read as truth: got %d positions (the flat book), want the 1 fill-derived position", len(pos))
	}
	if entry, _ := pos[0]["entryPrice"].(float64); entry != 100 {
		t.Fatalf("fill-derived entry must be 100, got %v", entry)
	}
}

// TestGetPositionsFreshSnapshotAfterEntryServesTruth: the same fence must NOT
// over-refuse — a snapshot received AFTER the entry receipt is NT8 truth.
func TestGetPositionsFreshSnapshotAfterEntryServesTruth(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	tr := NewTCPTrader(s, "MNQ", "Sim101")

	s.NoteEntryExecution("MNQ", "Sim101", "sig-1", 1) // receipt at t1
	time.Sleep(2 * time.Millisecond)
	s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{
		{Symbol: "MNQ", Side: "long", Quantity: 2, AvgPrice: 30550.25},
	}) // snapshot stamped after the receipt

	pos, err := tr.GetPositions()
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if len(pos) != 1 {
		t.Fatalf("a fresh post-entry snapshot must serve as truth, got %d", len(pos))
	}
	if entry, _ := pos[0]["entryPrice"].(float64); entry != 30550.25 {
		t.Fatalf("snapshot entry must be 30550.25, got %v", entry)
	}
}
