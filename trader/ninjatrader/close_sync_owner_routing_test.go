package ninjatrader

import (
	"path/filepath"
	"testing"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// TestRecordClose_OwnerRouting locks the fix: a position_close frame received by a
// trader that does NOT own the open row is still matched to the OWNING trader (by
// account+symbol+side across ALL traders) and the P&L is PERSISTED — instead of being
// silently dropped (the old "No matching open position, skipping" → reconcile pnl=0).
func TestRecordClose_OwnerRouting(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "own.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}

	const ownerID = "trader-A"
	// Owner A holds an open MNQ SHORT on Sim101, entry 29804.50.
	if err := st.Position().Create(&store.TraderPosition{
		TraderID: ownerID, Account: "Sim101", Symbol: "MNQ", Side: "SHORT",
		Quantity: 1, EntryQuantity: 1, EntryPrice: 29804.50, EntryTime: 1, Status: "OPEN",
	}); err != nil {
		t.Fatalf("create: %v", err)
	}

	// The frame is received by a DIFFERENT trader's close-sync (trader B, other account).
	s := ntwire.NewTCPServer(nil)
	trB := NewTCPTrader(s, "MNQ", "SimAccountX")
	pb := store.NewPositionBuilder(st.Position())
	trB.recordClose("trader-B", "ex", "ninjatrader", st, pb, ntwire.PositionClosePayload{
		SignalID: "x", Symbol: "MNQ", PositionSide: "short", ExitPrice: 29718.00,
		Quantity: 1, ExitReason: "manual", Account: "Sim101",
	})

	// Owner A's row must now be CLOSED (no open row) with the real exit + P&L.
	if open, _ := st.Position().GetOpenPositionByAccountSymbol("Sim101", "MNQ", "SHORT"); open != nil {
		t.Fatalf("owner's row should be CLOSED after owner-routed record; still open id=%d", open.ID)
	}
	closed, err := st.Position().GetClosedPositions(ownerID, 10)
	if err != nil || len(closed) != 1 {
		t.Fatalf("want 1 closed row for owner, got %d (err=%v)", len(closed), err)
	}
	// SHORT P&L = (entry - exit) * qty * $2 = (29804.50 - 29718.00) * 1 * 2 = 173.0.
	if closed[0].ExitPrice != 29718.00 {
		t.Fatalf("exit_price = %.2f, want 29718.00 (real fill, not entry)", closed[0].ExitPrice)
	}
	if closed[0].RealizedPnL != 173.0 {
		t.Fatalf("realized_pnl = %.2f, want 173.0 (×$2 point value)", closed[0].RealizedPnL)
	}
}

// TestRecordClose_NoOpenRow_ParksPrice: a priced close with no matching open row is
// PARKED (not silently dropped) so reconcile can recover it — and the park is consumed
// exactly once (dedupe) and expires past the grace window.
func TestRecordClose_NoOpenRow_ParksPrice(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "park.db"))
	if err != nil {
		t.Fatalf("store.New: %v", err)
	}
	s := ntwire.NewTCPServer(nil)
	tr := NewTCPTrader(s, "MNQ", "Sim101")
	pb := store.NewPositionBuilder(st.Position())
	// No open row anywhere → the priced close must be parked, not persisted.
	tr.recordClose("trader-A", "ex", "ninjatrader", st, pb, ntwire.PositionClosePayload{
		SignalID: "y", Symbol: "MNQ", PositionSide: "short", ExitPrice: 29718.00,
		Quantity: 1, ExitReason: "manual", Account: "Sim101",
	})

	// reconcile (grace-window fresh) consumes the parked price once...
	if px, ok := takePricedClose("Sim101", "MNQ", "SHORT", 1); !ok || px != 29718.00 {
		t.Fatalf("parked price not recoverable: px=%.2f ok=%v", px, ok)
	}
	// ...and only once (idempotent — a second reconcile gets nothing).
	if _, ok := takePricedClose("Sim101", "MNQ", "SHORT", 1); ok {
		t.Fatal("parked price consumed twice — not idempotent")
	}

	// Stale (past grace) parks are dropped, not returned.
	putPricedClose("Sim101", "ES", "LONG", 5000.25, 1, 0)
	if _, ok := takePricedClose("Sim101", "ES", "LONG", pricedCloseGraceMs+1); ok {
		t.Fatal("stale parked price (past grace) must not be returned")
	}
}
