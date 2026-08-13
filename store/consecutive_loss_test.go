// D1 — consecutive-loss halt: the count that drives the entry gate. Verifies the
// TAIL-of-history streak logic: consecutive losers since the session start, a win
// (or break-even) resets, per-trader isolation, session-boundary + reconcile_flat
// exclusion.
package store

import (
	"path/filepath"
	"testing"
)

func TestCountConsecutiveLossesSince(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	ps := st.Position()

	mk := func(trader string, exitMs int64, pnl float64, reason string) {
		p := &TraderPosition{
			TraderID: trader, Account: "Sim101", Symbol: "MNQ", Side: "LONG",
			Quantity: 1, EntryPrice: 100, RealizedPnL: pnl,
			Status: "CLOSED", CloseReason: reason,
			EntryTime: exitMs - 1, ExitTime: exitMs, CreatedAt: exitMs, UpdatedAt: exitMs,
		}
		// Insert the CLOSED row directly — PositionStore.Create is the entry-path
		// constructor and force-sets status=OPEN, which this query filters out.
		if err := ps.db.Create(p).Error; err != nil {
			t.Fatalf("Create: %v", err)
		}
	}

	const since = 1000

	// Two consecutive losses this session → streak 2 (the D1 halt limit is reached).
	mk("trA", 1100, -50, "sync")
	mk("trA", 1200, -30, "sync")
	if n, _ := ps.CountConsecutiveLossesSince("trA", since); n != 2 {
		t.Fatalf("2 consecutive losses → 2, got %d", n)
	}

	// A winning close at the tail RESETS the streak.
	mk("trA", 1300, +40, "sync")
	if n, _ := ps.CountConsecutiveLossesSince("trA", since); n != 0 {
		t.Fatalf("win resets streak → 0, got %d", n)
	}

	// A new loss after the win counts only from the win boundary → 1.
	mk("trA", 1400, -20, "sync")
	if n, _ := ps.CountConsecutiveLossesSince("trA", since); n != 1 {
		t.Fatalf("loss after win → 1, got %d", n)
	}

	// A loss BEFORE the session start (exit_time < since) is excluded.
	mk("trA", 500, -99, "sync")
	if n, _ := ps.CountConsecutiveLossesSince("trA", since); n != 1 {
		t.Fatalf("pre-session loss excluded → still 1, got %d", n)
	}

	// Per-trader isolation: trB's losses never count for trA.
	mk("trB", 1500, -70, "sync")
	mk("trB", 1600, -70, "sync")
	if n, _ := ps.CountConsecutiveLossesSince("trA", since); n != 1 {
		t.Fatalf("trB losses must not count for trA → 1, got %d", n)
	}
	if n, _ := ps.CountConsecutiveLossesSince("trB", since); n != 2 {
		t.Fatalf("trB → 2, got %d", n)
	}

	// A reconcile_flat orphan (unknown P&L) at the tail is excluded, so trA's real
	// tail loss (1400) still governs → 1.
	mk("trA", 1700, -1, CloseReasonReconcileFlat)
	if n, _ := ps.CountConsecutiveLossesSince("trA", since); n != 1 {
		t.Fatalf("reconcile_flat excluded → still 1, got %d", n)
	}
}
