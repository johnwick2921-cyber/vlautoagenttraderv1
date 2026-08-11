package ninjatrader

import (
	"testing"

	ntwire "nofx/provider/ninjatrader"
)

// TestGetPositions_NonActiveBoundAccount_SeesOwnPosition locks the Go side of the
// BUG A fix: once the AddOn streams a `positions` frame for a NON-active account
// (the C# SendAllOpenPositions change), a trader bound to that account reads its
// OWN position instead of an empty book — even while a DIFFERENT account is the
// streamed/current one. This is what un-blinds "15m" and lets its never-compound /
// same-side / max-position gates fire (they all read GetPositions()).
func TestGetPositions_NonActiveBoundAccount_SeesOwnPosition(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	// The connection is streaming Sim101 (hoang's, the "active" account)...
	s.SetCurrentAccountForTest("Sim101")
	// ...but the AddOn now ALSO emits a positions frame for SimAccount1 (15m's).
	s.SeedPositionsForTest("SimAccount1", []ntwire.OpenPosition{
		{Symbol: "MNQ", Side: "long", Quantity: 3, AvgPrice: 29825},
	})

	// Before the fix, PositionsFor("SimAccount1") was !ok (no frame ever emitted).
	if _, ok := s.PositionsFor("SimAccount1"); !ok {
		t.Fatal("PositionsFor(SimAccount1) should be ok once its positions frame arrives")
	}

	m15 := NewTCPTrader(s, "MNQ", "SimAccount1")
	pos, err := m15.GetPositions()
	if err != nil {
		t.Fatalf("GetPositions: %v", err)
	}
	if len(pos) != 1 {
		t.Fatalf("15m must see its OWN SimAccount1 position (qty 3), got %d — still blind", len(pos))
	}
	if side, _ := pos[0]["side"].(string); side != "LONG" {
		t.Fatalf("want LONG (15m's real book), got %q", side)
	}
	if qty, _ := pos[0]["positionAmt"].(float64); qty != 3 {
		t.Fatalf("want qty 3 (the real net it was blind to), got %v", qty)
	}

	// And a trader on the OTHER account is unaffected (reads its own, empty here).
	hoang := NewTCPTrader(s, "MNQ", "Sim101")
	if hp, _ := hoang.GetPositions(); len(hp) != 0 {
		t.Fatalf("hoang (Sim101) has no seeded position; want 0, got %d", len(hp))
	}
}
