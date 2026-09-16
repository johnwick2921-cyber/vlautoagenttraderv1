package store

import (
	"path/filepath"
	"testing"
)

// P&L-TRUTH WAVE (2026-09-01) — corrected-column law on every aggregator:
// NULL pnl_corrected rows are UNRESOLVED — excluded from sums, averages, win
// rates and streaks, and the exclusion count is returned alongside.

func pnlRow(t *testing.T, ps *PositionStore, id int64, trader, account string, exitMs int64, realized float64, corrected *float64, reason string) {
	t.Helper()
	p := &TraderPosition{
		ID: id, TraderID: trader, Account: account, Symbol: "MNQ", Side: "LONG",
		Quantity: 1, EntryPrice: 100, ExitPrice: 100 + realized, RealizedPnL: realized, PnlCorrected: corrected,
		Status: "CLOSED", CloseReason: reason,
		EntryTime: exitMs - 60_000, ExitTime: exitMs, CreatedAt: exitMs, UpdatedAt: exitMs,
	}
	if err := ps.db.Create(p).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}
}

func fp(v float64) *float64 { return &v }

// The fixture: 3 resolved (+50 −20 +10 = +40 over 3) and 2 UNRESOLVED rows
// carrying raw −100 / −300 (coerced would be −360 over 5).
func pnlFixture(t *testing.T) (*Store, *PositionStore) {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "pnl.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ps := st.Position()
	base := int64(1_756_700_000_000)
	pnlRow(t, ps, 1, "t1", "Sim101", base+1_000, 50, fp(50), "sync")
	pnlRow(t, ps, 2, "t1", "Sim101", base+2_000, -20, fp(-20), "sync")
	pnlRow(t, ps, 3, "t1", "Sim101", base+3_000, 10, fp(10), "sync")
	pnlRow(t, ps, 4, "t1", "Sim101", base+4_000, -100, nil, "sync")
	pnlRow(t, ps, 5, "t1", "Sim101", base+5_000, -300, nil, "sync")
	return st, ps
}

func TestPnlTruthGetFullStatsStrict(t *testing.T) {
	_, ps := pnlFixture(t)
	st, err := ps.GetFullStats("t1", "Sim101")
	if err != nil {
		t.Fatal(err)
	}
	if st.TotalPnL != 40 || st.TotalTrades != 3 || st.ResolvedTrades != 3 {
		t.Fatalf("strict: want +40 over 3 resolved, got pnl=%.2f total=%d resolved=%d", st.TotalPnL, st.TotalTrades, st.ResolvedTrades)
	}
	if st.UnresolvedExcluded != 2 {
		t.Fatalf("unresolved exclusion count must ride with the figure, got %d", st.UnresolvedExcluded)
	}
	if st.WinTrades != 2 || st.LossTrades != 1 {
		t.Fatalf("win/loss over resolved rows only, got %d/%d", st.WinTrades, st.LossTrades)
	}
	// All-unresolved trader: figure UNRESOLVED (0 resolved), count surfaced, never a coerced total.
	pnlRow(t, ps, 6, "t2", "Sim101", 1_756_700_009_000, -999, nil, "sync")
	st2, _ := ps.GetFullStats("t2", "Sim101")
	if st2.TotalTrades != 0 || st2.TotalPnL != 0 || st2.UnresolvedExcluded != 1 {
		t.Fatalf("all-unresolved must be 0 resolved / 1 excluded / no total, got %+v", st2)
	}
}

func TestPnlTruthEveryAggregatorExcludesNull(t *testing.T) {
	_, ps := pnlFixture(t)
	m, err := ps.GetPositionStats("t1")
	if err != nil {
		t.Fatal(err)
	}
	if m["total_pnl"].(float64) != 40 || m["total_trades"].(int) != 3 || m["excluded_null_pnl"].(int64) != 2 {
		t.Fatalf("GetPositionStats: %+v", m)
	}
	day, entries, err := ps.GetSessionDayActivity("t1", 0)
	if err != nil || day != 40 || entries != 5 {
		t.Fatalf("GetSessionDayActivity: pnl=%.2f entries=%d err=%v (want +40 strict; entries count ALL opens)", day, entries, err)
	}
	// Streak: newest rows are the two unresolved (−100, −300 raw) — they must
	// NOT extend a losing streak; the newest RESOLVED row is +10 → streak 0.
	if n, _ := ps.CountConsecutiveLossesSince("t1", 0); n != 0 {
		t.Fatalf("unresolved rows must not count as losses, got streak %d", n)
	}
	sym, err := ps.GetSymbolStats("t1", 10)
	if err != nil || len(sym) != 1 || sym[0].TotalPnL != 40 || sym[0].TotalTrades != 3 || sym[0].UnresolvedExcluded != 2 {
		t.Fatalf("GetSymbolStats: %+v err=%v", sym, err)
	}
	hold, err := ps.GetHoldingTimeStats("t1")
	if err != nil {
		t.Fatal(err)
	}
	tc, ue := 0, 0
	for _, h := range hold {
		tc += h.TradeCount
		ue += h.UnresolvedExcluded
	}
	if tc != 3 || ue != 2 {
		t.Fatalf("GetHoldingTimeStats: resolved=%d unresolved=%d", tc, ue)
	}
	sum, err := ps.GetHistorySummary("t1")
	if err != nil || sum.TotalPnL != 40 || sum.TotalTrades != 3 || sum.UnresolvedExcluded != 2 {
		t.Fatalf("GetHistorySummary: %+v err=%v", sum, err)
	}
	if sum.RecentPnL != 40 {
		t.Fatalf("recent-window P&L must be strict, got %.2f", sum.RecentPnL)
	}
}

func TestPnlTruthRecentTradesUnresolvedRowCarriesNoPnlNoPct(t *testing.T) {
	_, ps := pnlFixture(t)
	// An unresolved SHORT with exit 0 — the live row that rendered "+0.00 (+100.00%)".
	p := &TraderPosition{ID: 7, TraderID: "t1", Account: "Sim101", Symbol: "MNQ", Side: "SHORT", Quantity: 1,
		EntryPrice: 29459, ExitPrice: 0, RealizedPnL: 0, Status: "CLOSED", CloseReason: CloseReasonUnresolved,
		EntryTime: 1_756_700_100_000, ExitTime: 1_756_700_200_000}
	if err := ps.db.Create(p).Error; err != nil {
		t.Fatal(err)
	}
	pnlRow(t, ps, 8, "t1", "Sim101", 1_756_700_300_000, 6, fp(6), CloseReasonTestSeam) // quarantined
	trades, err := ps.GetRecentTrades("t1", 10, "Sim101")
	if err != nil {
		t.Fatal(err)
	}
	byID := map[int64]RecentTrade{}
	for _, tr := range trades {
		byID[tr.ID] = tr
	}
	if _, seam := byID[8]; seam {
		t.Fatal("test-seam rows must not appear in the model's recent-trade list")
	}
	u := byID[7]
	if u.Resolved || u.RealizedPnL != 0 || u.PnLPct != 0 {
		t.Fatalf("unresolved row must carry no P&L and no percentage, got %+v", u)
	}
	if u4 := byID[4]; u4.Resolved || u4.RealizedPnL != 0 {
		t.Fatalf("NULL pnl_corrected must never be coerced to raw −100, got %+v", u4)
	}
	r := byID[1]
	if !r.Resolved || r.RealizedPnL != 50 || r.PnLPct == 0 {
		t.Fatalf("resolved row unchanged: %+v", r)
	}
}

func TestPnlTruthLedgerDayTotalMatchesTheFooterRule(t *testing.T) {
	_, ps := pnlFixture(t)
	base := int64(1_756_700_000_000)
	// Footer rule fixtures: a duplicate (reconcile_flat sharing the entry order id
	// with a real close), a hidden test-seam row and an unresolved-reason row —
	// none of them count; the NULL rows are counted as unresolved.
	pnlRow(t, ps, 9, "t1", "Sim101", base+6_000, 92, fp(92), "sync")
	dupe := &TraderPosition{ID: 10, TraderID: "t1", Account: "Sim101", Symbol: "MNQ", Side: "LONG", Quantity: 1, EntryPrice: 100,
		EntryOrderID: "ord-9", Status: "CLOSED", CloseReason: CloseReasonReconcileFlat, EntryTime: base + 5_500, ExitTime: base + 6_500}
	_ = ps.db.Model(&TraderPosition{}).Where("id = ?", 9).Update("entry_order_id", "ord-9").Error
	if err := ps.db.Create(dupe).Error; err != nil {
		t.Fatal(err)
	}
	pnlRow(t, ps, 11, "t1", "Sim101", base+7_000, 6, fp(6), CloseReasonTestSeam)
	pnlRow(t, ps, 12, "t1", "Sim101", base+8_000, 0, nil, CloseReasonUnresolved)
	got, err := ps.GetLedgerDayTotal("t1", "Sim101", base, base+100_000)
	if err != nil {
		t.Fatal(err)
	}
	// resolved: +50 −20 +10 +92 = +132 over 4; unresolved (NULL, normal reason): rows 4, 5 = 2
	if got.Total != 132 || got.Resolved != 4 || got.Unresolved != 2 {
		t.Fatalf("ledger day total = %+v, want +132 / 4 resolved / 2 unresolved (duplicate, test-seam, unresolved-reason rows excluded)", got)
	}
	// Window scoping + account scoping.
	if g2, _ := ps.GetLedgerDayTotal("t1", "Sim101", base+100_000, base+200_000); g2.Resolved != 0 || g2.Total != 0 {
		t.Fatalf("outside the window: %+v", g2)
	}
	if g3, _ := ps.GetLedgerDayTotal("t1", "SimAccount1", base, base+100_000); g3.Resolved != 0 {
		t.Fatalf("other account: %+v", g3)
	}
}

func TestPnlTruthBootLineAndRegistry(t *testing.T) {
	line := PnLSurfacesBootLine()
	if len(PnLSurfaces()) < 10 || !contains(line, "P&L surfaces:") || !contains(line, "strict-corrected, 0 raw") {
		t.Fatalf("boot line %q / registry %d", line, len(PnLSurfaces()))
	}
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
