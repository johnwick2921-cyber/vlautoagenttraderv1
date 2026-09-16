package ninjatrader

import (
	"errors"
	"testing"

	ntwire "nofx/provider/ninjatrader"
	"nofx/trader/types"
)

// openOrdersSourceSetter is asserted dynamically so this pin COMPILES on the
// pre-fix tree (where the setter does not exist) and still fails there — the
// point of E1 is a red-then-green on real behaviour, not a compile error.
type openOrdersSourceSetter interface {
	SetOpenOrdersSource(func(string) ([]types.OpenOrder, error))
}

// E1 (CLASS 33) — THE PIN. Flat-gate leg 4 asks the trader for working orders.
// TCPTrader.GetOpenOrders was `return []types.OpenOrder{}, nil` (tcp_trader.go
// :1149), so the leg passed VACUOUSLY at every cutover 35→41 while the
// armed_orders ledger held a WORKING row (2026-09-02 00:16 CT: arms S1 @29044
// and S3 @29068.05 were resting when the swap landed). A gate that cannot fail
// is not a gate (A24).
func TestClass33PinLeg4Stub(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	s.SetCurrentAccountForTest("Sim101")
	tr := NewTCPTrader(s, "MNQ", "Sim101")

	// The ledger holds ONE working resting limit — exactly the 00:16 CT shape.
	ledgerRows := []types.OpenOrder{{
		OrderID: "sig-S3", Symbol: "MNQ", Side: "BUY", PositionSide: "LONG",
		Type: "LIMIT", Price: 29068.05, Quantity: 1, Status: "NEW",
		Source: "ledger (no NT8 order frame — F12 open)",
	}}
	if setter, ok := any(tr).(openOrdersSourceSetter); ok {
		setter.SetOpenOrdersSource(func(string) ([]types.OpenOrder, error) { return ledgerRows, nil })
	} else {
		t.Log("pre-fix tree: SetOpenOrdersSource does not exist — leg 4 has no source at all")
	}

	got, err := tr.GetOpenOrders("MNQ")
	if err != nil {
		t.Fatalf("leg 4 must answer, got error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("CLASS 33: leg 4 saw %d working order(s) while the ledger holds 1 — the stub returns empty, so the flat gate PASSES with an order resting at the broker", len(got))
	}
	if got[0].OrderID != "sig-S3" || got[0].Price != 29068.05 {
		t.Fatalf("leg 4 row wrong: %+v", got[0])
	}
	if got[0].Source == "" {
		t.Fatalf("leg 4 must name its source so no reader mistakes the ledger for broker truth")
	}
}

// E2 — a ledger error FAILS the leg loudly; it never degrades to empty (A24:
// a plausible-zero gate is not a gate).
func TestClass33Leg4LedgerErrorFailsLoudly(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	tr := NewTCPTrader(s, "MNQ", "Sim101")
	setter, ok := any(tr).(openOrdersSourceSetter)
	if !ok {
		t.Skip("pre-fix tree: no source seam")
	}
	setter.SetOpenOrdersSource(func(string) ([]types.OpenOrder, error) {
		return nil, errors.New("ledger unavailable")
	})
	got, err := tr.GetOpenOrders("MNQ")
	if err == nil {
		t.Fatalf("a ledger failure must FAIL the leg, got %d rows and nil error", len(got))
	}
	if len(got) != 0 {
		t.Fatalf("no rows may be returned alongside the error, got %d", len(got))
	}
}

// E2b — with NO source wired the leg FAILS rather than reporting "no working
// orders". An unwired safety source is fail-closed, never a silent pass.
func TestClass33Leg4UnwiredFailsLoudly(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	tr := NewTCPTrader(s, "MNQ", "Sim101")
	if _, ok := any(tr).(openOrdersSourceSetter); !ok {
		t.Skip("pre-fix tree: no source seam")
	}
	if _, err := tr.GetOpenOrders("MNQ"); err == nil {
		t.Fatalf("an unwired open-orders source must fail the leg, not return empty")
	}
}
