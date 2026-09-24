package trader

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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

// ── W1b FOLD-2 — THE CHAT DOOR SENDS THROUGH THE AI DECISION'S EXECUTE PATH ─
//
// The door (OpenManualEntry → sendManualEntry) called the broker's
// OpenLong/OpenShort directly: no reconcile-before-open, no max-positions or
// same-side check, no max-contracts cap (a chat "buy 3 MNQ" sent 3 against
// the Stage-A cap), and no order record — the position materialized untracked
// and E15's AI-order branch could never tag it. Driven here at the door the
// agent's executeTradeWith calls (AutoTrader.OpenManualEntryAt), over a REAL
// TCPTrader, a real TCP server and a real store (canon 53).

type chatDoorWire struct {
	at   *AutoTrader
	st   *store.Store
	s    *ntwire.TCPServer
	nt   *nttrader.TCPTrader
	conn net.Conn
	sigs chan ntwire.SignalPayload
	cls  chan struct{}
}

func newChatDoorWire(t *testing.T, rc store.RiskControlConfig) *chatDoorWire {
	t.Helper()
	withMaintenanceDir(t)
	at, st := resetTrader(t, store.StrategyConfig{RiskControl: rc, DayPlan: &store.DayPlanConfig{PlanEnabled: true, PlanMode: "advisory"}})
	at.id = "chat-door-wire"
	at.positionFirstSeenTime = map[string]int64{} // as NewAutoTrader builds it
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
	w := &chatDoorWire{at: at, st: st, s: s, conn: conn, sigs: make(chan ntwire.SignalPayload, 8), cls: make(chan struct{}, 8)}
	go func() {
		for {
			env, err := ntwire.ReadFrame(conn)
			if err != nil {
				return
			}
			switch env.Type {
			case ntwire.FrameSignal:
				var p ntwire.SignalPayload
				if json.Unmarshal(env.Payload, &p) == nil {
					w.sigs <- p
				}
			case ntwire.FrameClosePosition:
				w.cls <- struct{}{}
			}
		}
	}()
	w.nt = nttrader.NewTCPTrader(s, "MNQ", "Sim101")
	at.trader = w.nt
	at.config.NinjaTraderSymbol = "MNQ"
	return w
}

// nothingSent fails if an entry or a flatten reached the wire.
func (w *chatDoorWire) nothingSent(t *testing.T, what string) {
	t.Helper()
	select {
	case p := <-w.sigs:
		t.Fatalf("%s: an entry reached the wire: %+v", what, p)
	case <-w.cls:
		t.Fatalf("%s: a flatten reached the wire", what)
	case <-time.After(300 * time.Millisecond):
	}
}

var chatDoorMidday = time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC) // 11:00 CT Tuesday

// chatBracket is a bracket the admission chain admits at the tape's live price.
func chatBracket(t *testing.T, action string) (stop, target float64) {
	t.Helper()
	live := agentTape(t)
	wide := onGrid(kernel.MinSLATRMult()*armSeamATR5m("MNQ") + 10)
	if action == "open_short" {
		return live + wide, live - 4*wide
	}
	return live - wide, live + 4*wide
}

// On CME the owner's contract count is sent AS TYPED or REFUSED: above the
// max-contracts cap the AI's sizing clamps to, or not a whole contract, the
// entry is refused by name and nothing reaches the wire — never clamped.
func TestChatEntryOverTheMaxContractsCapIsRefusedNeverClamped(t *testing.T) {
	w := newChatDoorWire(t, store.RiskControlConfig{MaxContractsPerOrder: 1})
	stop, target := chatBracket(t, "open_long")
	capN := w.at.resolveMaxContracts()
	if capN < 1 {
		t.Fatalf("fixture: max-contracts cap %d", capN)
	}
	for _, q := range []float64{float64(capN + 2), 1.5} {
		_, err := w.at.OpenManualEntryAt("MNQ", "open_long", q, 1, stop, target, chatDoorMidday)
		want := fmt.Sprintf("refused: quantity %d exceeds the max-contracts cap %d", capN+2, capN)
		if q == 1.5 {
			want = "refused: quantity 1.5 is not a whole number of contracts"
		}
		var ref *ManualEntryRefusal
		if !errors.As(err, &ref) || ref.Reason != want || !ref.Execute {
			t.Fatalf("quantity %v: want the execute-side refusal %q, got %T %v", q, want, err, err)
		}
		w.nothingSent(t, fmt.Sprintf("quantity %v", q))
	}
}

// A position NT8 holds that the ledger explains refuses the chat entry at
// reconcile-before-open, exactly as it refuses an AI entry — refused (typed,
// named), counted under reconcile_owned, never flattened, nothing sent.
func TestChatEntryIsReconciledBeforeOpenLikeAnAIEntry(t *testing.T) {
	w := newChatDoorWire(t, store.RiskControlConfig{})
	stop, target := chatBracket(t, "open_long")
	w.s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{{Symbol: "MNQ", Side: "short", Quantity: 1, AvgPrice: 29000}})
	led := w.st.ArmedOrders()
	r := &store.ArmedOrderDB{TraderID: w.at.id, PlanID: "p", Scenario: "S1", Version: 1, State: "armed", Side: "short", EntryPx: 29000, StopPx: 29010, TargetPx: 28970}
	if err := led.UpsertArm(r); err != nil {
		t.Fatal(err)
	}
	if err := led.BeginPlacement(r.ID, "sig-armed-short"); err != nil {
		t.Fatal(err)
	}
	if err := led.SetState(r.ID, "working", "fixture"); err != nil {
		t.Fatal(err)
	}
	before := gateBlocks(w.at.id, "reconcile_owned")

	_, err := w.at.OpenManualEntryAt("MNQ", "open_long", 1, 1, stop, target, chatDoorMidday)
	var ref *ManualEntryRefusal
	if !errors.As(err, &ref) || !ref.Execute || !strings.HasPrefix(ref.Reason, "reconcile_owned: ") || !strings.Contains(ref.Reason, "armed #") {
		t.Fatalf("a ledger-explained held position must refuse the chat entry at reconcile, typed and named: %T %v", err, err)
	}
	if got := gateBlocks(w.at.id, "reconcile_owned"); got != before+1 {
		t.Fatalf("the refusal must be counted under reconcile_owned (%d → %d)", before, got)
	}
	w.nothingSent(t, "reconcile-owned")
}

// The max-positions limit binds a chat entry like an AI entry: the account
// already holds MaxPositions positions (another instrument, so reconcile has
// nothing of MNQ to act on) → refused before any send.
func TestChatEntryAtMaxPositionsIsRefusedLikeAnAIEntry(t *testing.T) {
	w := newChatDoorWire(t, store.RiskControlConfig{MaxPositions: 1})
	stop, target := chatBracket(t, "open_long")
	w.s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{{Symbol: "ES", Side: "long", Quantity: 1, AvgPrice: 6500}})
	_, err := w.at.OpenManualEntryAt("MNQ", "open_long", 1, 1, stop, target, chatDoorMidday)
	var ref *ManualEntryRefusal
	if !errors.As(err, &ref) || !ref.Execute || !strings.Contains(ref.Reason, "max positions (1/1)") {
		t.Fatalf("a chat entry at the max-positions limit must be refused before any send, named: %T %v", err, err)
	}
	w.nothingSent(t, "max positions")
}

// After a send the chat entry has an ORDER ROW keyed by its signal id (the
// AI path's recordAndConfirmOrder), so a fill is matched to it and its
// position row carries that signal as its entry order — and it never takes
// the AI decision's pending plan citation.
func TestChatEntrySendIsRecordedUnderItsSignalID(t *testing.T) {
	w := newChatDoorWire(t, store.RiskControlConfig{})
	stop, target := chatBracket(t, "open_long")
	w.at.lastCitation = planCitation{planVersion: 7, scenarioID: "S-AI", matched: true, planID: "plan-ai", valid: true}

	fillPx := onGrid(stop + 1)
	go func() {
		select {
		case p := <-w.sigs:
			w.sigs <- p // hand it back to the assertion below
			_ = ntwire.WriteFrame(w.conn, ntwire.FrameFill, ntwire.FillPayload{
				SignalID: p.SignalID, Symbol: "MNQ", Account: "Sim101", Side: "long", Quantity: 1,
				FillPrice: fillPx, Status: "filled", FillTime: time.Now().UTC().Format(time.RFC3339),
			})
		case <-time.After(5 * time.Second):
		}
	}()
	if _, err := w.at.OpenManualEntryAt("MNQ", "open_long", 1, 1, stop, target, chatDoorMidday); err != nil {
		t.Fatalf("an admitted chat entry must send: %v", err)
	}
	var sig ntwire.SignalPayload
	select {
	case sig = <-w.sigs:
	case <-time.After(3 * time.Second):
		t.Fatal("no signal frame reached the wire")
	}
	if sig.StopLoss != stop || sig.TakeProfit != target || sig.Quantity != 1 {
		t.Fatalf("the wire must carry the chat entry's own bracket and quantity: %+v (want SL %.2f TP %.2f qty 1)", sig, stop, target)
	}
	orders, err := w.st.Order().GetTraderOrders(w.at.id, 10)
	if err != nil {
		t.Fatal(err)
	}
	var row *store.TraderOrder
	for _, o := range orders {
		if o.ExchangeOrderID == sig.SignalID {
			row = o
		}
	}
	if row == nil || row.OrderAction != "open_long" || row.Status != "FILLED" {
		t.Fatalf("a sent chat entry must leave an order row keyed by its signal id %q (filled): %+v", sig.SignalID, orders)
	}
	pos, err := w.st.Position().GetOpenPositionBySymbol(w.at.id, "MNQ", "LONG")
	if err != nil || pos == nil || pos.EntryOrderID != sig.SignalID || pos.EntryPrice != fillPx {
		t.Fatalf("the filled chat entry must be recorded as a position under its signal id: %+v err=%v", pos, err)
	}
	if pos.CitedScenarioID != "" || pos.PlanVersion != 0 || !w.at.lastCitation.valid {
		t.Fatalf("a chat entry must never take the AI decision's plan citation: pos scenario=%q version=%d, citation still pending=%v",
			pos.CitedScenarioID, pos.PlanVersion, w.at.lastCitation.valid)
	}
}

// The NT8 venue reads CME futures only: a non-CME chat symbol on it is
// refused by the open path's own market read — before any broker write, typed
// as a refusal (the AI open refuses it at the same read).
func TestChatNonCMEEntryOnTheNT8VenueIsRefusedBeforeAnySend(t *testing.T) {
	at, _ := resetTrader(t, store.StrategyConfig{})
	b := &doorBroker{openOrderPayload: map[string]interface{}{"orderId": "o-1"}}
	at.trader = b
	_, err := at.sendManualEntry("BTCUSDT", "open_long", 1, 1, 60000, 70000)
	var ref *ManualEntryRefusal
	if !errors.As(err, &ref) || !ref.Execute || b.opens != 0 || !strings.Contains(ref.Reason, "CME futures only") {
		t.Fatalf("a non-CME entry on the NT8 venue must be refused before any send: %T %v (opens=%d)", err, err, b.opens)
	}
}

// The same-side check binds a chat entry like an AI entry. On NT8 a held
// position is met FIRST by reconcile-before-open (refused when the ledger
// explains it — pinned above — else the owner-ruled orphan flatten), so the
// check is reached on a venue with no reconcile: an existing long refuses a
// chat open_long before any broker write, typed as a refusal.
func TestChatEntryOnAnAlreadyHeldSideIsRefusedLikeAnAIEntry(t *testing.T) {
	at, _ := resetTrader(t, store.StrategyConfig{})
	at.exchange = "binance"
	prev := openEntryMarketRead
	openEntryMarketRead = func(string, string) (*market.Data, error) { return &market.Data{CurrentPrice: 65000}, nil }
	t.Cleanup(func() { openEntryMarketRead = prev })
	b := &doorBroker{openOrderPayload: map[string]interface{}{"orderId": "o-1"}}
	b.positions = []map[string]interface{}{{"symbol": "BTCUSDT", "side": "long", "positionAmt": 0.01}}
	at.trader = b
	_, err := at.sendManualEntry("BTCUSDT", "open_long", 0.01, 1, 60000, 70000)
	var ref *ManualEntryRefusal
	if !errors.As(err, &ref) || !ref.Execute || !strings.Contains(ref.Reason, "already has long position") || b.opens != 0 || b.stopSet != 0 {
		t.Fatalf("a chat entry on an already-held side must be refused before any broker write: %T %v (opens=%d stopSet=%.2f)", err, err, b.opens, b.stopSet)
	}
}
