package trader

import (
	"context"
	"encoding/json"
	"net"
	"path/filepath"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// Class 27 FIX 1 (2026-08-31) — the instant the skip gate sees store-OPEN vs
// broker-FLAT, it must send cancel_order for the open row's bracket (no 60s
// grace). Wire-proven on a loopback TCP server: the cancel_order frame arrives
// carrying the row's entry order id (the bracket key).

func class27DesyncHarness(t *testing.T) (*AutoTrader, *ntwire.TCPServer, chan ntwire.CancelOrderPayload) {
	t.Helper()
	s := ntwire.NewTCPServer(nil)
	s.SetAddrForTest("127.0.0.1:0")
	s.SetAccountsList([]ntwire.AccountInfo{{Name: "Sim101", IsSim: true}}, "Sim101")
	ctx, cancel := context.WithCancel(context.Background())
	if err := s.Start(ctx); err != nil {
		t.Fatalf("server start: %v", err)
	}
	t.Cleanup(func() { _ = s.Stop(); cancel() })

	conn, err := net.Dial("tcp", s.ListenAddrForTest().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	frames := make(chan ntwire.CancelOrderPayload, 4)
	go func() {
		for {
			env, err := ntwire.ReadFrame(conn)
			if err != nil {
				return
			}
			if env.Type != ntwire.FrameCancelOrder {
				continue
			}
			var p ntwire.CancelOrderPayload
			if json.Unmarshal(env.Payload, &p) == nil {
				frames <- p
			}
		}
	}()

	st, err := store.New(filepath.Join(t.TempDir(), "desync-cancel.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	tr := ntTrader.NewTCPTrader(s, "MNQ", "Sim101")
	at := &AutoTrader{id: "td1", exchange: "ninjatrader", store: st, trader: tr}
	at.config.Exchange = "ninjatrader"
	at.config.StrategyConfig = &store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	return at, s, frames
}

func TestDesyncSendsImmediateCancelOrder(t *testing.T) {
	at, _, frames := class27DesyncHarness(t)
	// An open row whose entry_order_id is the bracket key (GAR-F1 identity).
	if err := at.store.Position().CreateOpenPosition(&store.TraderPosition{
		TraderID: "td1", ExchangeType: "ninjatrader",
		ExchangePositionID: "armed_x_1", Symbol: "MNQ", Side: "LONG",
		Quantity: 1, EntryQuantity: 1, EntryPrice: 29413.0,
		EntryOrderID: "f0bbe9af-c6ce-4444-8243-974c1ce03208", Leverage: 1,
		Status: "OPEN", Source: "armed_entry", Account: "Sim101",
	}); err != nil {
		t.Fatalf("create open row: %v", err)
	}

	skip, _ := at.skipWhileOpen() // broker bound, no positions seeded → FLAT
	if skip {
		t.Fatal("desync must NOT skip")
	}

	select {
	case p := <-frames:
		if p.SignalID != "f0bbe9af-c6ce-4444-8243-974c1ce03208" {
			t.Fatalf("cancel_order must carry the row's bracket key, got %q", p.SignalID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no cancel_order frame within 2s — class-27 desync cancel did not fire")
	}
}

// Broker confirms the hold → NO cancel may fire (no false kills).
func TestDesyncNoCancelWhenBrokerHolds(t *testing.T) {
	at, s, frames := class27DesyncHarness(t)
	if err := at.store.Position().CreateOpenPosition(&store.TraderPosition{
		TraderID: "td1", ExchangeType: "ninjatrader",
		ExchangePositionID: "armed_y_1", Symbol: "MNQ", Side: "LONG",
		Quantity: 1, EntryQuantity: 1, EntryPrice: 29413.0,
		EntryOrderID: "f0bbe9af-c6ce-4444-8243-974c1ce03208", Leverage: 1,
		Status: "OPEN", Source: "armed_entry", Account: "Sim101",
	}); err != nil {
		t.Fatalf("create open row: %v", err)
	}
	// Broker truth: MNQ LONG held → the gate must skip and never cancel.
	s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{{Symbol: "MNQ", Side: "LONG", Quantity: 1, AvgPrice: 29413}})

	if skip, _ := at.skipWhileOpen(); !skip {
		t.Fatal("broker-confirmed hold must skip (and must NOT cancel)")
	}
	select {
	case p := <-frames:
		t.Fatalf("unexpected cancel_order for %q while the broker holds the position", p.SignalID)
	case <-time.After(300 * time.Millisecond):
		// expected: silence
	}
}
