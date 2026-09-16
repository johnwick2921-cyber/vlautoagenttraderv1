package kernel

import (
	"path/filepath"
	"strings"
	"testing"

	"nofx/store"
)

// P&L-TRUTH WAVE (2026-09-01) — PIN: the model must never read a fabricated
// track record.
//
// Live evidence: decision record 36090 (23:07:13 CT, Sim101) rendered
//   Total Trades: 220 | … Total PnL: -203.68 USDT
// where the strict corrected truth is +304.32 over 105 resolved trades with
// 115 unresolved trades excluded. GetFullStats loaded every closed row and
// summed EffectivePnL(), which coerces a NULL pnl_corrected to raw
// realized_pnl (row 526: raw −1,458.00 vs corrected −69.43).
//
// This fixture uses ONLY the pre-fix surface (GetFullStats + the engine's
// BuildUserPrompt) so it compiles on the old tree and FAILS there.

func pnlTruthRow(t *testing.T, st *store.Store, id int64, realized float64, corrected *float64) {
	t.Helper()
	row := &store.TraderPosition{
		ID: id, TraderID: "t1", Account: "Sim101", Symbol: "MNQ", Side: "LONG",
		Quantity: 1, EntryPrice: 29000, ExitPrice: 29010, RealizedPnL: realized, PnlCorrected: corrected,
		Status: "CLOSED", CloseReason: "sync",
		EntryTime: 1_756_700_000_000 + id*60_000, ExitTime: 1_756_700_000_000 + id*60_000 + 30_000,
	}
	if err := st.GormDB().Create(row).Error; err != nil {
		t.Fatal(err)
	}
}

func TestPnlTruthPinExecutorPrompt(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "pnl.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	f := func(v float64) *float64 { return &v }
	// 3 RESOLVED: +50 −20 +10 → strict +40.00 over 3.
	pnlTruthRow(t, st, 1, 50, f(50))
	pnlTruthRow(t, st, 2, -20, f(-20))
	pnlTruthRow(t, st, 3, 10, f(10))
	// 2 UNRESOLVED (pnl_corrected NULL) carrying raw realized −100 and −300:
	// the coerced (old) total is −360.00 over 5 — a fabricated track record.
	pnlTruthRow(t, st, 4, -100, nil)
	pnlTruthRow(t, st, 5, -300, nil)

	stats, err := st.Position().GetFullStats("t1", "Sim101")
	if err != nil {
		t.Fatal(err)
	}
	// Build the context exactly the way trader/auto_trader_loop.go does
	// (pre-fix field set), then render the executor prompt.
	ctx := &Context{TradingStats: &TradingStats{
		TotalTrades: stats.TotalTrades, WinRate: stats.WinRate, ProfitFactor: stats.ProfitFactor,
		SharpeRatio: stats.SharpeRatio, TotalPnL: stats.TotalPnL, AvgWin: stats.AvgWin,
		AvgLoss: stats.AvgLoss, MaxDrawdownPct: stats.MaxDrawdownPct,
	}}
	prompt := NewStrategyEngine(&store.StrategyConfig{Language: "en"}).BuildUserPrompt(ctx)
	block := prompt
	if i := strings.Index(prompt, "## Historical Trading Statistics"); i >= 0 {
		block = prompt[i:]
		if j := strings.Index(block, "\n\n"); j > 0 {
			block = block[:j]
		}
	}
	if strings.Contains(block, "-360.00") || strings.Contains(block, "Total PnL:") || strings.Contains(block, "Total Trades: 5") {
		t.Fatalf("P&L TRUTH: the executor prompt carries a COERCED / bare track record (2 unresolved rows folded in as raw realized_pnl):\n%s", block)
	}
	if !strings.Contains(block, "+40.00 over 3 resolved trades") {
		t.Fatalf("P&L TRUTH: the executor prompt must state \"+40.00 over 3 resolved trades\", got:\n%s", block)
	}
}
