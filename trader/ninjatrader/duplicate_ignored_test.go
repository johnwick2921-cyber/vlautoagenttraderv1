package ninjatrader

import (
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
)

// FIX-DOUBLE-ENTRY test 7 — a "duplicate_ignored" reply from the AddOn-side
// dedupe must never read as a rejection: no pending drop, no cached fill, no
// re-arm path fires. The original entry's own fill/update owns the state.
func TestDuplicateIgnoredReplyNeverTriggersRearm(t *testing.T) {
	server := ntwire.NewTCPServer(nil)
	tr := NewTCPTrader(server, "MNQ", "Sim101")
	// Plant a pending entry marker for the signal — the exact state a replay
	// could disturb.
	tr.pendingMu.Lock()
	tr.pending["dup-1"] = "long"
	tr.pendingAt["dup-1"] = time.Now().UnixMilli()
	tr.pendingMu.Unlock()

	tr.handleFillInbound(ntwire.FillPayload{
		SignalID:      "dup-1",
		Status:        "duplicate_ignored",
		FillPrice:     0,
		Quantity:      0,
		Symbol:        "MNQ",
		Account:       "Sim101",
		SlippageTicks: 0,
	})

	tr.pendingMu.Lock()
	_, stillPending := tr.pending["dup-1"]
	_, stillPendingAt := tr.pendingAt["dup-1"]
	tr.pendingMu.Unlock()
	if !stillPending || !stillPendingAt {
		t.Fatalf("duplicate_ignored must NOT drop the pending entry marker (a rejection would)")
	}
	tr.mu.Lock()
	hasFill := tr.hasFill
	last := tr.lastFill
	tr.mu.Unlock()
	if hasFill || last.SignalID != "" {
		t.Fatalf("duplicate_ignored must NOT become a fill: hasFill=%v last=%+v", hasFill, last)
	}
}
