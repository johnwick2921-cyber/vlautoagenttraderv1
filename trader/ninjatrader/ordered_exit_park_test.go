package ninjatrader

import (
	"path/filepath"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// F-A (CTO review, class 40) — the ordered close path keeps the two legacy
// contracts on every Pending / no-row result: the broker's price is parked for
// reconcile's orphan close, and the flat signal is dropped. Without the park, a
// manual NT8 flatten (the #526 class: qty=21 over a 1-lot row) or a frame-before-
// entry close would reach the orphan close with no price → exit=entry, pnl=0.

func newExitParkFixture(t *testing.T) (*TCPTrader, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "fa.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	tr := NewTCPTrader(ntwire.NewTCPServer(nil), "MNQ", "Sim101")
	return tr, st
}

// (a) The #526 replay: a 1-lot OPEN row and a close frame with qty=21. The
// receipt stays PENDING (the row is never grown), the frame's REAL price is
// parked for the orphan close, and the row stays OPEN at 1 lot — so reconcile
// closes (exit−entry)×1×pointValue, never 0 and never ×21.
func TestOversizedCloseParksTheRealPriceAndStaysPending(t *testing.T) {
	tr, st := newExitParkFixture(t)
	now := time.Now().UTC().UnixMilli()
	row := &store.TraderPosition{
		TraderID: "t1", ExchangeType: "ninjatrader", ExchangePositionID: "p1",
		Symbol: "MNQ", Side: "LONG", Quantity: 1, EntryQuantity: 1,
		EntryPrice: 29350, EntryTime: now - 60_000, EntryOrderID: "sig-526",
		Leverage: 1, Status: "OPEN", Source: "armed_entry", Account: "Sim101",
		CreatedAt: now - 60_000, UpdatedAt: now - 60_000,
	}
	if err := st.Position().CreateOpenPosition(row); err != nil {
		t.Fatal(err)
	}
	tr.recordCloseOrdered("t1", "ex", "ninjatrader", st, ntwire.PositionClosePayload{
		SignalID: "sig-526", Symbol: "MNQ", PositionSide: "long", Account: "Sim101",
		Quantity: 21, ExitPrice: 29660.96, ExitReason: "manual",
		ExitTime: time.Now().UTC().Format(time.RFC3339),
	})

	pending, err := st.Position().PendingNT8Exits("Sim101")
	if err != nil || len(pending) != 1 || pending[0].Applied {
		t.Fatalf("the oversized close must be RETAINED pending: %d applied=%v err=%v", len(pending), pending[0].Applied, err)
	}
	openRow, err := st.Position().GetOpenPositionBySymbol("t1", "MNQ", "LONG")
	if err != nil || openRow == nil || openRow.Quantity != 1 {
		t.Fatalf("the 1-lot row must stay OPEN at 1 lot (the row never grows to 21): %+v err=%v", openRow, err)
	}
	if px, ok := takePricedClose("Sim101", "MNQ", "LONG", now); !ok || px != 29660.96 {
		t.Fatalf("the frame's REAL price must be parked for reconcile's orphan close, got (%.2f, %v)", px, ok)
	}
	tr.mu.Lock()
	hasFill := tr.hasFill
	tr.mu.Unlock()
	if hasFill {
		t.Fatal("the Pending path must drop the flat signal exactly like legacy recordClose")
	}
}

// (b) A no-row close: the receipt is pending AND the price is parked — the
// orphan close later consumes the REAL exit, never a fabricated 0.
func TestNoRowCloseParksTheRealPrice(t *testing.T) {
	tr, st := newExitParkFixture(t)
	now := time.Now().UTC().UnixMilli()
	tr.recordCloseOrdered("t1", "ex", "ninjatrader", st, ntwire.PositionClosePayload{
		SignalID: "sig-ghost", Symbol: "MNQ", PositionSide: "long", Account: "Sim101",
		Quantity: 1, ExitPrice: 29475.50, ExitReason: "tp",
		ExitTime: time.Now().UTC().Format(time.RFC3339),
	})
	pending, err := st.Position().PendingNT8Exits("Sim101")
	if err != nil || len(pending) != 1 {
		t.Fatalf("the no-row close must be RETAINED pending: %d err=%v", len(pending), err)
	}
	if px, ok := takePricedClose("Sim101", "MNQ", "LONG", now); !ok || px != 29475.50 {
		t.Fatalf("the no-row close must park the frame's price for the orphan close, got (%.2f, %v)", px, ok)
	}
}
