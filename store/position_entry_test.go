// Deterministic proof of the entry-anchor write the NinjaTrader reconcile uses:
// UpdateEntryPrice must replace entry_price (the stale 5m-mark reference) with
// the NT8 position average, WITHOUT disturbing status/quantity/side. The
// reconcile LOOP that decides when to call this is proven live (orphan-close);
// this pins the write primitive itself.
package store

import (
	"path/filepath"
	"testing"
)

func TestUpdateEntryPrice(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("New store: %v", err)
	}
	ps := st.Position()

	// Open row written with the stale decision-time mark (as the AI-decision path does).
	pos := &TraderPosition{
		TraderID:  "tr1",
		Account:   "Sim101",
		Symbol:    "MNQ",
		Side:      "SHORT",
		Quantity:  1,
		EntryPrice: 30523.75, // frozen 5m-mark
		Status:    "OPEN",
		EntryTime: 1,
		CreatedAt: 1,
		UpdatedAt: 1,
	}
	if err := ps.Create(pos); err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Reconcile anchors entry to the NT8 average (the real fill).
	const nt8Avg = 30450.25
	if err := ps.UpdateEntryPrice(pos.ID, nt8Avg); err != nil {
		t.Fatalf("UpdateEntryPrice: %v", err)
	}

	got, err := ps.GetOpenPositionBySymbol("tr1", "MNQ", "SHORT")
	if err != nil || got == nil {
		t.Fatalf("refetch: got=%v err=%v", got, err)
	}
	if got.EntryPrice != nt8Avg {
		t.Errorf("entry_price = %v, want %v (NT8 avg)", got.EntryPrice, nt8Avg)
	}
	// Must NOT disturb anything else — it's an entry anchor, not a close/average.
	if got.Status != "OPEN" {
		t.Errorf("status = %q, want OPEN (unchanged)", got.Status)
	}
	if got.Quantity != 1 {
		t.Errorf("quantity = %v, want 1 (unchanged)", got.Quantity)
	}
	if got.Side != "SHORT" {
		t.Errorf("side = %q, want SHORT (unchanged)", got.Side)
	}
}
