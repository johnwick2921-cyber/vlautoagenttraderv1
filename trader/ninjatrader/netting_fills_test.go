package ninjatrader

import (
	"path/filepath"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// Class 27 (2026-08-31 netting-orphan) — exit reconstruction tests.
//
// Today's live stream (the replay): S1 Buy 29413 opened LONG; the S3 SellShort
// 29459 NETTED it flat (no position_close frame ever); the S1 stop later fired
// while flat (the naked short). The ledger must reconstruct the long's exit as
// 29459 → +92.00, never exit=entry.

func fillAt(side string, price float64, ts int64, symbol, account string) recentFill {
	return recentFill{Side: side, Price: price, TimeMs: ts, Symbol: symbol, Account: account}
}

func TestTakeNettingExitFindsOppositeFill(t *testing.T) {
	tr := NewTCPTrader(ntwire.NewTCPServer(nil), "MNQ", "Sim101")
	now := time.Now().UTC().UnixMilli()
	flat := now - flatGraceMs - 1_000
	// Today's replay: SHORT 29459 at flat-10s (the netting fill), a same-side
	// SHORT before it (must be ignored), and a LONG earlier (same side as row).
	tr.mu.Lock()
	tr.recentFills = append(tr.recentFills,
		fillAt("long", 29413.0, flat-40_000, "MNQ", "Sim101"),
		fillAt("short", 29363.25, flat-30_000, "MNQ", "Sim101"),
		fillAt("short", 29459.0, flat-10_000, "MNQ", "Sim101"))
	tr.mu.Unlock()

	px, ok := tr.takeNettingExit("Sim101", "MNQ", "LONG", flat, now)
	if !ok || px != 29459.0 {
		t.Fatalf("netting exit: want (29459.0, true), got (%.2f, %v)", px, ok)
	}
}

func TestTakeNettingExitRejectsSameSideAndStale(t *testing.T) {
	tr := NewTCPTrader(ntwire.NewTCPServer(nil), "MNQ", "Sim101")
	now := time.Now().UTC().UnixMilli()
	flat := now - flatGraceMs - 1_000
	tr.mu.Lock()
	tr.recentFills = append(tr.recentFills,
		fillAt("short", 29459.0, flat-nettingFillWindowMs-5_000, "MNQ", "Sim101"), // too old
		fillAt("long", 29397.5, flat+2_000, "MNQ", "Sim101"))                      // same side as LONG row
	tr.mu.Unlock()

	if px, ok := tr.takeNettingExit("Sim101", "MNQ", "LONG", flat, now); ok {
		t.Fatalf("no opposite fill in window: want ok=false, got (%.2f, true)", px)
	}
}

// TestReconcileReconstructsNettingExitReplayToday replays today's 12:25–13:09
// stream: a LONG row @29413 orphan-closes while the ring holds the netting
// SELL @29459 → the row must close with exit 29459 and pnl +92.00.
func TestReconcileReconstructsNettingExitReplayToday(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "netting.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	const traderID = "trader-netting"

	s := ntwire.NewTCPServer(nil)
	// NT8 reports FLAT (the netting close already happened).
	s.SeedPositionsForTest("Sim101", nil)
	tr := NewTCPTrader(s, "MNQ", "Sim101")

	nowMs := time.Now().UTC().UnixMilli()
	// The S1 long row (as the armed materializer would have written it).
	row := &store.TraderPosition{
		TraderID: traderID, ExchangeType: "ninjatrader",
		ExchangePositionID: "armed_f0bbe9af_1", Symbol: "MNQ", Side: "LONG",
		Quantity: 1, EntryQuantity: 1, EntryPrice: 29413.0, EntryTime: nowMs - 400_000,
		EntryOrderID: "f0bbe9af-c6ce-4444-8243-974c1ce03208", Leverage: 1,
		Status: "OPEN", Source: "armed_entry", Account: "Sim101",
		CreatedAt: nowMs - 400_000, UpdatedAt: nowMs - 400_000,
	}
	if err := st.Position().CreateOpenPosition(row); err != nil {
		t.Fatalf("create open row: %v", err)
	}
	// The ring holds today's netting fill (SHORT @29459) just before flat.
	tr.mu.Lock()
	tr.recentFills = append(tr.recentFills, fillAt("short", 29459.0, nowMs-30_000, "MNQ", "Sim101"))
	tr.mu.Unlock()

	// First pass: defer (records flatSince), within grace.
	tr.reconcilePositions(traderID, "nt", "ninjatrader", st)
	if tr.flatSince[1] == 0 {
		t.Fatal("first pass must start the flat debounce")
	}
	// Age past the grace → orphan-close must reconstruct the netting exit.
	tr.flatSince[1] = nowMs - flatGraceMs - 1_000
	tr.reconcilePositions(traderID, "nt", "ninjatrader", st)

	closed, _ := st.Position().GetClosedPositions(traderID, 10, "Sim101")
	if len(closed) != 1 {
		t.Fatalf("expected 1 closed row, got %d", len(closed))
	}
	c := closed[0]
	if c.ExitPrice != 29459.0 {
		t.Fatalf("exit must be the netting fill 29459.0, got %.2f", c.ExitPrice)
	}
	if c.RealizedPnL != 92.0 {
		t.Fatalf("replay P&L must be +92.00 (29413→29459 × $2), got %.2f", c.RealizedPnL)
	}
	if c.CloseReason != "sync" {
		t.Fatalf("reconstructed close must be reason sync, got %q", c.CloseReason)
	}
}

// TestReconcileUnresolvedWhenNoEvidence: flat + no frame + no netting fill →
// UNRESOLVED (never exit=entry).
func TestReconcileUnresolvedWhenNoEvidence(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "unres.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	const traderID = "trader-unres"

	s := ntwire.NewTCPServer(nil)
	s.SeedPositionsForTest("Sim101", nil)
	tr := NewTCPTrader(s, "MNQ", "Sim101")

	nowMs := time.Now().UTC().UnixMilli()
	row := &store.TraderPosition{
		TraderID: traderID, ExchangeType: "ninjatrader",
		ExchangePositionID: "armed_ghost_1", Symbol: "MNQ", Side: "SHORT",
		Quantity: 1, EntryQuantity: 1, EntryPrice: 29459.0, EntryTime: nowMs - 400_000,
		EntryOrderID: "060192ea-7281-4b55-b7f7-11a9c11adbe7", Leverage: 1,
		Status: "OPEN", Source: "armed_entry", Account: "Sim101",
		CreatedAt: nowMs - 400_000, UpdatedAt: nowMs - 400_000,
	}
	if err := st.Position().CreateOpenPosition(row); err != nil {
		t.Fatalf("create open row: %v", err)
	}
	tr.reconcilePositions(traderID, "nt", "ninjatrader", st)
	tr.flatSince[1] = nowMs - flatGraceMs - 1_000
	tr.reconcilePositions(traderID, "nt", "ninjatrader", st)

	closed, _ := st.Position().GetClosedPositions(traderID, 10, "Sim101")
	if len(closed) != 1 {
		t.Fatalf("expected 1 closed row, got %d", len(closed))
	}
	c := closed[0]
	if c.CloseReason != store.CloseReasonUnresolved {
		t.Fatalf("no evidence → close_reason must be unresolved, got %q", c.CloseReason)
	}
	if c.ExitPrice != 0 {
		t.Fatalf("unresolved exit must NOT fabricate a price, got %.2f", c.ExitPrice)
	}
	// Stats must exclude the unresolved row (visible gap, not a fake zero).
	stats, err := st.Position().GetFullStats(traderID, "Sim101")
	if err != nil {
		t.Fatalf("GetFullStats: %v", err)
	}
	if stats.TotalPnL != 0 || stats.TotalTrades != 0 {
		t.Fatalf("unresolved row must be excluded from stats, got pnl=%.2f trades=%d", stats.TotalPnL, stats.TotalTrades)
	}
}
