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
		// A-2 (2026-08-28): rows ruled-from must carry a verified correction —
		// NULL pnl_corrected rows are EXCLUDED from the streak query.
		pnlCopy := pnl
		p.PnlCorrected = &pnlCopy
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

	// A reconcile_flat orphan (unknown P&L) at the tail ENDS the run.
	//
	// CHANGED 2026-09-09 (dispatch 104 D2, owner-ruled). This asserted 1: the
	// orphan was excluded by the WHERE, so trA's earlier loss still governed.
	// Excluding a row from the scan does not make it neutral — it makes it
	// TRANSPARENT, and a transparent unknown BRIDGES two runs into one longer
	// one (3 losses + an orphan + 3 losses counted as SIX). Bridging makes a
	// halt MORE likely, and blocking is this counter's destructive branch, so
	// the old semantics had UNKNOWN pushing toward the harmful side — the thing
	// A24 forbids. "We do not know whether that trade won" ends a run of KNOWN
	// losses, so the answer is now 0.
	mk("trA", 1700, -1, CloseReasonReconcileFlat)
	if n, _ := ps.CountConsecutiveLossesSince("trA", since); n != 0 {
		t.Fatalf("an unknown-P&L close at the tail must END the run → 0, got %d", n)
	}
	// ...and a real loss after it starts a fresh run of 1, so the break is a
	// break and not a permanent mute.
	mk("trA", 1800, -40, "sync")
	if n, _ := ps.CountConsecutiveLossesSince("trA", since); n != 1 {
		t.Fatalf("a loss after the unknown starts a new run → 1, got %d", n)
	}
}
