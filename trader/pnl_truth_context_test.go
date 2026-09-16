package trader

import (
	"path/filepath"
	"strings"
	"testing"

	"nofx/kernel"
	"nofx/store"
)

// P&L-TRUTH WAVE — the production plumbing (attachTradeContext) carries the
// resolved/unresolved counts into the prompt, and an unresolved recent trade
// renders as UNRESOLVED with no P&L and no percentage.
func TestPnlTruthAttachTradeContextRendersTheTruth(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "ctx.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	f := func(v float64) *float64 { return &v }
	mk := func(id int64, side string, exit, realized float64, corrected *float64, reason string) {
		row := &store.TraderPosition{ID: id, TraderID: "t1", Account: "Sim101", Symbol: "MNQ", Side: side, Quantity: 1,
			EntryPrice: 29459, ExitPrice: exit, RealizedPnL: realized, PnlCorrected: corrected, Status: "CLOSED", CloseReason: reason,
			EntryTime: 1_756_700_000_000 + id*60_000, ExitTime: 1_756_700_000_000 + id*60_000 + 30_000}
		if err := st.GormDB().Create(row).Error; err != nil {
			t.Fatal(err)
		}
	}
	mk(1, "LONG", 29509, 100, f(100), "sync")
	mk(2, "LONG", 29439, -40, f(-40), "sync")
	mk(3, "SHORT", 0, 0, nil, store.CloseReasonUnresolved) // the live "+0.00 (+100.00%)" row
	mk(4, "LONG", 29470, -1458, nil, "sync")               // row-526 shape: raw ×21 artifact, unresolved

	at := &AutoTrader{id: "t1", store: st, exchange: "ninjatrader", config: AutoTraderConfig{NinjaTraderSymbol: "MNQ", StrategyConfig: &store.StrategyConfig{Language: "en"}}}
	ctx := &kernel.Context{}
	at.attachTradeContext(ctx)

	if ctx.TradingStats == nil {
		t.Fatal("stats must be attached")
	}
	// UnresolvedExcluded counts NULL-pnl rows INSIDE the ledger predicate (row 4);
	// row 3 carries close_reason=unresolved, a class the 0A-2 ledger rule already
	// keeps out of every aggregate (it still LISTS as UNRESOLVED below). This is
	// the same definition behind the live figure: 220 rows = 105 resolved + 115
	// unresolved, measured after the close_reason filter.
	if ctx.TradingStats.TotalPnL != 60 || ctx.TradingStats.TotalTrades != 2 || ctx.TradingStats.UnresolvedExcluded != 1 {
		t.Fatalf("track record must be +60 over 2 resolved with 1 unresolved excluded, got %+v", *ctx.TradingStats)
	}
	if ctx.TotalRealizedPnL != 60 {
		t.Fatalf("consistency-rule input must be the strict figure, got %.2f", ctx.TotalRealizedPnL)
	}
	prompt := kernel.NewStrategyEngine(&store.StrategyConfig{Language: "en"}).BuildUserPrompt(ctx)
	if !strings.Contains(prompt, "Track record: +60.00 over 2 resolved trades (1 unresolved trades excluded — see note).") {
		t.Fatalf("prompt track-record line missing/wrong:\n%s", prompt)
	}
	if strings.Contains(prompt, "Total PnL:") || strings.Contains(prompt, "-1458") || strings.Contains(prompt, "100.00%") {
		t.Fatalf("prompt still carries a coerced figure or a fabricated percentage:\n%s", prompt)
	}
	if !strings.Contains(prompt, "#3 MNQ short | Entry 29459.0000→? UNRESOLVED (exit unknown)") {
		t.Fatalf("unresolved row must render as UNRESOLVED with no P&L:\n%s", prompt)
	}
	if !strings.Contains(prompt, "#4 MNQ long | Entry 29459.0000→? UNRESOLVED (exit unknown)") {
		t.Fatalf("NULL pnl_corrected with a raw value must still be UNRESOLVED:\n%s", prompt)
	}
	if !strings.Contains(prompt, "Profit: +100.00 USDT") {
		t.Fatalf("resolved row must render unchanged:\n%s", prompt)
	}
}
