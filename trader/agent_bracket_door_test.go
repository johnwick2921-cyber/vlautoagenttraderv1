package trader

import (
	"context"
	"encoding/json"
	"math"
	"net"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	nttrader "nofx/trader/ninjatrader"
)

// ── W1b E9 — THE AGENT DOOR SENDS ITS OWN BRACKET ───────────────────────────
//
// The NT8 broker keeps an entry's SL/TP in per-(symbol, side) maps that
// outlive the decision that set them (SetStopLoss / SetTakeProfit; the
// breakeven move writes the same key). The agent door used to call
// OpenLong/OpenShort with nothing set, so the entry went out carrying the
// LAST AI decision's bracket — or a breakeven stop. The door's one send,
// OpenManualEntry, is driven here over a REAL TCPTrader and a real server
// (canon 53): the signal frame on the wire must carry the chat entry's own
// prices, never the stale map values another producer left behind.

type agentDoorWire struct {
	at   *AutoTrader
	nt   *nttrader.TCPTrader
	sigs chan ntwire.SignalPayload
}

func newAgentDoorWire(t *testing.T) *agentDoorWire {
	t.Helper()
	withMaintenanceDir(t)
	at, _ := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true, PlanMode: "advisory"}})
	at.id = "agent-door-wire"
	kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{ActivePlan: func(string) *kernel.ActivePlan { return nil }})
	t.Cleanup(func() { kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{}) })
	s := ntwire.NewTCPServer(nil)
	s.SetAddrForTest("127.0.0.1:0")
	s.SetAccountsList([]ntwire.AccountInfo{{Name: "Sim101", IsSim: true}}, "Sim101")
	ctx, cancel := context.WithCancel(context.Background())
	if err := s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Stop(); cancel() })
	conn, err := net.Dial("tcp", s.ListenAddrForTest().String())
	if err != nil {
		t.Fatal(err)
	}
	waitAddonRegistered(t, s)
	t.Cleanup(func() { _ = conn.Close() })
	sigs := make(chan ntwire.SignalPayload, 8)
	go func() {
		for {
			env, err := ntwire.ReadFrame(conn)
			if err != nil {
				return
			}
			if env.Type != ntwire.FrameSignal {
				continue
			}
			var p ntwire.SignalPayload
			if json.Unmarshal(env.Payload, &p) == nil {
				select {
				case sigs <- p:
				default:
				}
			}
		}
	}()
	nt := nttrader.NewTCPTrader(s, "MNQ", "Sim101")
	at.trader = nt
	at.config.NinjaTraderSymbol = "MNQ"
	return &agentDoorWire{at: at, nt: nt, sigs: sigs}
}

func (w *agentDoorWire) signal(t *testing.T) ntwire.SignalPayload {
	t.Helper()
	select {
	case p := <-w.sigs:
		return p
	case <-time.After(3 * time.Second):
		t.Fatal("no signal frame reached the wire")
		return ntwire.SignalPayload{}
	}
}

func onGrid(px float64) float64 { return math.Round(px*4) / 4 }

// agentTape is futuresTape (5m + 1h, the live read) plus the 1m tape the
// ATR5m seam reads (kernel.AISVPBarInterval), so EntryGate leg 6 has an ATR
// to judge with. Returns the live price.
func agentTape(t *testing.T) float64 {
	t.Helper()
	live := futuresTape(t)
	m1 := make([]market.Kline, 200)
	start := time.Now().Add(-200 * time.Minute)
	for i := range m1 {
		c := live - float64(i%5)*0.5
		ot := start.Add(time.Duration(i) * time.Minute).UnixMilli()
		m1[i] = market.Kline{OpenTime: ot, Open: c, High: c + 2, Low: c - 2, Close: c, Volume: 10, CloseTime: ot + 59_999}
	}
	prev := market.FuturesBarsProvider
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline {
		if tf == kernel.AISVPBarInterval {
			return tailOf(m1, count)
		}
		return prev(symbol, tf, count)
	}
	t.Cleanup(func() { market.FuturesBarsProvider = prev })
	return live
}

func TestChatEntryWithAStopSendsItsOwnBracket(t *testing.T) {
	w := newAgentDoorWire(t)
	live := agentTape(t)
	nyMidday := time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC) // 11:00 CT Tuesday
	wide := onGrid(kernel.MinSLATRMult()*armSeamATR5m("MNQ") + 10)

	// Another producer's bracket is sitting in the maps (the AI's last
	// Set→Open, or a breakeven move) — both sides.
	for _, side := range []string{"LONG", "SHORT"} {
		_ = w.nt.SetStopLoss("MNQ", side, 1, 20000)
		_ = w.nt.SetTakeProfit("MNQ", side, 1, 40000)
	}

	for _, c := range []struct {
		act          string
		stop, target float64
	}{
		{"open_long", live - wide, live + 4*wide},
		{"open_short", live + wide, live - 4*wide},
	} {
		if _, err := w.at.OpenManualEntryAt("MNQ", c.act, 1, 1, c.stop, c.target, nyMidday); err != nil {
			t.Fatalf("%s: a sane chat bracket must be admitted and sent: %v", c.act, err)
		}
		p := w.signal(t)
		if p.StopLoss != c.stop || p.TakeProfit != c.target {
			t.Fatalf("%s: the wire carried SL %.2f / TP %.2f — the chat entry's own bracket is SL %.2f / TP %.2f (stale map values are 20000 / 40000)",
				c.act, p.StopLoss, p.TakeProfit, c.stop, c.target)
		}
	}

	// A stop-less chat entry is refused and NOTHING reaches the wire, even
	// though the maps hold a complete bracket the broker would happily send.
	if _, err := w.at.OpenManualEntryAt("MNQ", "open_long", 1, 1, 0, 0, nyMidday); err == nil || !strings.Contains(err.Error(), "no explicit stop") {
		t.Fatalf("a stop-less chat entry must be refused by name, got %v", err)
	}
	select {
	case p := <-w.sigs:
		t.Fatalf("a refused chat entry reached the wire: %+v", p)
	case <-time.After(300 * time.Millisecond):
	}
}
