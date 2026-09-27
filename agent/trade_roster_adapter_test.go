package agent

import (
	"testing"

	ntwire "nofx/provider/ninjatrader"
	"nofx/trader"
	ntTrader "nofx/trader/ninjatrader"
)

// ── W1b FOLD-5 repair — the resolver reads the manager's roster through the
// PRODUCTION adapter (canon 53) ─────────────────────────────────────────────
//
// The route tests hand resolveTradeExecutionContext a fake candidate list
// (tradeCandidatesOf), which replaces the production adapter whole —
// managedTradeCandidate.tradeUnderlying and the WireSymbol assertion on a real
// NT8 broker never ran. Here the seam is the manager's roster itself
// (tradeRosterOf = GetAllTraders): real *trader.AutoTrader values over real
// TCPTraders go through tradeCandidatesOf's production body.
func TestChatCMEResolverSelectsTheNT8TraderThroughTheProductionAdapter(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	mnq := ntTrader.NewTCPTrader(s, "MNQ", "Sim101")
	atMNQ := trader.NewAutoTraderOnBrokerForTest("b-nt8-mnq", "ninjatrader", mnq, true)
	// Two that must NOT be chosen: an NT8 trader of another instrument, and a
	// stopped NT8 trader of the same one.
	atES := trader.NewAutoTraderOnBrokerForTest("a-nt8-es", "ninjatrader", ntTrader.NewTCPTrader(s, "ES", "Sim101"), true)
	atStopped := trader.NewAutoTraderOnBrokerForTest("c-nt8-mnq-stopped", "ninjatrader", ntTrader.NewTCPTrader(s, "MNQ", "Sim101"), false)

	prev := tradeRosterOf
	tradeRosterOf = func(*Agent) (map[string]*trader.AutoTrader, error) {
		return map[string]*trader.AutoTrader{"a-nt8-es": atES, "b-nt8-mnq": atMNQ, "c-nt8-mnq-stopped": atStopped}, nil
	}
	t.Cleanup(func() { tradeRosterOf = prev })

	a, _ := routeAgent()
	for _, sym := range []string{"MNQ", "mnq", "MNQU6"} {
		wantStock, sel, und, err := a.resolveTradeExecutionContext(&TradeAction{Action: "open_long", Symbol: sym, Quantity: 1})
		if err != nil || wantStock {
			t.Fatalf("%s: the production adapter must resolve the running MNQ NT8 trader, got stock=%v err=%v", sym, wantStock, err)
		}
		mc, ok := sel.(managedTradeCandidate)
		if !ok || mc.AutoTrader != atMNQ {
			t.Fatalf("%s: selected %T %+v, want the manager's MNQ AutoTrader %q", sym, sel, sel, "b-nt8-mnq")
		}
		if got, ok := und.(*ntTrader.TCPTrader); !ok || got != mnq {
			t.Fatalf("%s: underlying %T, want that trader's own TCPTrader", sym, und)
		}
	}
}
