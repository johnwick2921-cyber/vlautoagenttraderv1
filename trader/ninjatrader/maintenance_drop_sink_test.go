package ninjatrader

import (
	"fmt"
	"testing"

	ntwire "nofx/provider/ninjatrader"
)

// W-ONE-BUTTON M2, M-2 — the TCPTrader's half of a queue drop: only ITS OWN
// entries reach its handler; a never-attempted drop is forgotten (no fill can
// come, and a stale lastEntrySignalID must not claim a later fill); an
// ATTEMPTED drop may have reached NT8, so it stays tracked.
func TestDroppedEntrySinkRoutesOwnEntriesAndForgetsOnlyNeverAttempted(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	tr := NewTCPTrader(s, "MNQ", "Sim101")
	var heard []ntwire.DroppedEntry
	tr.SetDroppedEntrySink("trader-sink", func(d ntwire.DroppedEntry) { heard = append(heard, d) })
	tr.pendingMu.Lock()
	tr.pending["sig-never"], tr.pending["sig-tried"] = "LONG", "LONG"
	tr.pendingMu.Unlock()
	tr.mu.Lock()
	tr.lastEntrySignalID = "sig-never"
	tr.mu.Unlock()

	s.FeedDroppedEntryForTest(ntwire.DroppedEntry{SignalID: "sig-someone-else"})
	if len(heard) != 0 {
		t.Fatalf("another trader's drop must not reach this handler: %+v", heard)
	}
	s.FeedDroppedEntryForTest(ntwire.DroppedEntry{SignalID: "sig-tried", Attempted: true})
	if !tr.HasPendingEntry("sig-tried") || len(heard) != 1 {
		t.Fatalf("an attempted drop stays tracked and is heard: pending=%v heard=%+v", tr.HasPendingEntry("sig-tried"), heard)
	}
	s.FeedDroppedEntryForTest(ntwire.DroppedEntry{SignalID: "sig-never"})
	if tr.HasPendingEntry("sig-never") || len(heard) != 2 {
		t.Fatalf("a never-attempted drop is forgotten and heard: pending=%v heard=%+v", tr.HasPendingEntry("sig-never"), heard)
	}
	tr.mu.Lock()
	last := tr.lastEntrySignalID
	tr.mu.Unlock()
	if last != "" {
		t.Fatalf("a forgotten entry must not stay the last entry (a later fill would be claimed by it): %q", last)
	}
}

// Review F3: an ATTEMPTED own drop may be at NT8 — it must never read as a
// hold refusal, or the Picture row settles 'refused' and the AI path records
// nothing for an entry that may exist.
func TestAnAmbiguousDropIsNotAHoldRefusal(t *testing.T) {
	if IsMaintenanceHold(fmt.Errorf("send: %w", ntwire.ErrEntryDropAmbiguous)) {
		t.Fatal("ErrEntryDropAmbiguous must not satisfy IsMaintenanceHold")
	}
	if !IsMaintenanceHold(fmt.Errorf("send: %w", ntwire.ErrEntryHeld)) {
		t.Fatal("ErrEntryHeld (provably unsent) must still satisfy IsMaintenanceHold")
	}
}

// M2.1 (CTO: fix first) — sinks are keyed by the OWNING TRADER'S ID, not the
// TCPTrader's pointer: a reloaded trader (same id, new TCPTrader) REPLACES its
// old sink instead of leaving it registered for the life of the process
// (holding the old AutoTrader and TCPTrader alive).
func TestDroppedEntrySinkIsReplacedWhenTheSameTraderReloads(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	oldT, newT := NewTCPTrader(s, "MNQ", "Sim101"), NewTCPTrader(s, "MNQ", "Sim101")
	var oldHeard, newHeard int
	oldT.SetDroppedEntrySink("trader-x", func(ntwire.DroppedEntry) { oldHeard++ })
	newT.SetDroppedEntrySink("trader-x", func(ntwire.DroppedEntry) { newHeard++ })
	for _, tr := range []*TCPTrader{oldT, newT} {
		tr.pendingMu.Lock()
		tr.pending["sig-r"] = "LONG"
		tr.pendingMu.Unlock()
	}
	s.FeedDroppedEntryForTest(ntwire.DroppedEntry{SignalID: "sig-r"})
	if oldHeard != 0 || newHeard != 1 {
		t.Fatalf("a reload must replace the old sink: old heard %d, new heard %d", oldHeard, newHeard)
	}
}
