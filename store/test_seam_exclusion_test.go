// 0A-2 (2026-08-31) — test-seam quarantine: the E7 closeout promised
// e7_farside_test rows are excluded from strategy P&L. Pin it at every real-P&L
// aggregator so the broken promise can never regress.
package store

import (
	"path/filepath"
	"testing"
)

func testSeamRow(t *testing.T, ps *PositionStore, id int64, trader string, exitMs int64, pnl float64, reason string) {
	t.Helper()
	c := pnl
	p := &TraderPosition{
		ID: id, TraderID: trader, Account: "Sim101", Symbol: "MNQ", Side: "LONG",
		Quantity: 1, EntryPrice: 100, RealizedPnL: pnl, PnlCorrected: &c,
		Status: "CLOSED", CloseReason: reason,
		EntryTime: exitMs - 1, ExitTime: exitMs, CreatedAt: exitMs, UpdatedAt: exitMs,
	}
	if err := ps.db.Create(p).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func TestTestSeamRowsExcludedEverywhereRealPnLIsSummed(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "seam.db"))
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	ps := st.Position()

	// One REAL trade and three polluters: +6 / -1 test-seam rows and a
	// reconcile_flat placeholder.
	testSeamRow(t, ps, 1, "trA", 1000, +32.5, "sync")
	testSeamRow(t, ps, 2, "trA", 1100, +6.0, CloseReasonTestSeam)
	testSeamRow(t, ps, 3, "trA", 1200, -1.0, CloseReasonTestSeam)
	testSeamRow(t, ps, 4, "trA", 1300, 0.0, CloseReasonReconcileFlat)

	// Session-day activity (the guardrail P&L) = the real trade only.
	pnl, _, err := ps.GetSessionDayActivity("trA", 0)
	if err != nil {
		t.Fatalf("GetSessionDayActivity: %v", err)
	}
	if pnl != 32.5 {
		t.Fatalf("session-day P&L must exclude test-seam rows, got %.2f", pnl)
	}

	// Full stats = the real trade only (count + P&L).
	stats, err := ps.GetFullStats("trA", "Sim101")
	if err != nil {
		t.Fatalf("GetFullStats: %v", err)
	}
	if stats.TotalTrades != 1 || stats.TotalPnL != 32.5 {
		t.Fatalf("full stats must exclude test-seam rows, got trades=%d pnl=%.2f", stats.TotalTrades, stats.TotalPnL)
	}

	// Streak must ignore both polluter classes (the +6 test-seam would
	// otherwise falsely reset a losing streak).
	if n, _ := ps.CountConsecutiveLossesSince("trA", 0); n != 0 {
		t.Fatalf("streak must ignore test-seam rows, got %d", n)
	}

	// Per-symbol and per-direction stats exclude them too.
	ss, err := ps.GetSymbolStats("trA", 10)
	if err != nil || len(ss) != 1 || ss[0].TotalPnL != 32.5 {
		t.Fatalf("symbol stats must exclude test-seam rows: %+v (err=%v)", ss, err)
	}
	ds, err := ps.GetDirectionStats("trA")
	if err != nil || len(ds) != 1 || ds[0].TotalPnL != 32.5 {
		t.Fatalf("direction stats must exclude test-seam rows: %+v (err=%v)", ds, err)
	}

	// Position stats (the generic aggregate) exclude them too.
	pstats, err := ps.GetPositionStats("trA")
	if err != nil {
		t.Fatalf("GetPositionStats: %v", err)
	}
	if pstats["total_trades"] != 1 || pstats["total_pnl"] != 32.5 {
		t.Fatalf("position stats must exclude test-seam rows: %+v", pstats)
	}
}
