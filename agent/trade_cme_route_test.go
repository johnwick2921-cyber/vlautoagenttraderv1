package agent

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"strings"
	"testing"

	"nofx/mcp"
	"nofx/store"
)

// ── W1b FOLD-5 — a CME futures chat entry reaches the NT8 trader's door ─────
//
// isStockSymbol("MNQ") was true (three uppercase letters, not a known crypto
// base), so resolveTradeExecutionContext looked only for an Alpaca trader and
// a chat MNQ entry could never reach the NinjaTrader trader's OpenManualEntry
// ("no running stock trader (Alpaca) found"). Driven here through the
// PRODUCTION chain — the execute_trade tool (proposal → resolver), then the
// owner's confirm (handleTradeConfirmation → executeTrade → the resolver →
// executeTradeWith → OpenManualEntry) — over a fake roster handed to the
// resolver through its one seam (tradeCandidatesOf).

type routeUnderlying struct {
	admitUnderlying
	wire string // "" = not an NT8 broker (no WireSymbol)
}

type routeNT8Underlying struct{ routeUnderlying }

func (u *routeNT8Underlying) WireSymbol() string { return u.wire }

type routeCandidate struct {
	id, exchange string
	running      bool
	und          tradeUnderlyingTrader

	opens        int
	sym, act     string
	qty          float64
	stop, target float64
}

func (c *routeCandidate) GetStrategyConfig() *store.StrategyConfig { return nil }
func (c *routeCandidate) GetAccountInfo() (map[string]interface{}, error) {
	return map[string]interface{}{"total_equity": 1_000_000.0}, nil
}
func (c *routeCandidate) OpenManualEntry(symbol, action string, qty float64, lev int, stop, target float64) (map[string]interface{}, error) {
	c.opens++
	c.sym, c.act, c.qty, c.stop, c.target = symbol, action, qty, stop, target
	return map[string]interface{}{}, nil
}
func (c *routeCandidate) GetStatus() map[string]interface{} {
	return map[string]interface{}{"is_running": c.running, "trader_id": c.id}
}
func (c *routeCandidate) GetExchange() string                    { return c.exchange }
func (c *routeCandidate) tradeUnderlying() tradeUnderlyingTrader { return c.und }

func nt8Candidate(id, wire string) *routeCandidate {
	u := &routeNT8Underlying{routeUnderlying{wire: wire}}
	return &routeCandidate{id: id, exchange: "ninjatrader", running: true, und: u}
}

func withRoster(t *testing.T, roster ...*routeCandidate) {
	t.Helper()
	prev := tradeCandidatesOf
	tradeCandidatesOf = func(*Agent) ([]tradeCandidate, error) {
		out := make([]tradeCandidate, 0, len(roster))
		for _, c := range roster {
			out = append(out, c)
		}
		return out, nil
	}
	t.Cleanup(func() { tradeCandidatesOf = prev })
}

func routeAgent() (*Agent, context.Context) {
	a := &Agent{
		config:  &Config{AllowTradeExecution: true},
		pending: newPendingTrades(),
		logger:  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	ctx := WithSessionPolicy(context.Background(), SessionPolicy{Authenticated: true, CanExecuteTrade: true})
	return a, ctx
}

func proposeTrade(t *testing.T, a *Agent, ctx context.Context, args string) map[string]any {
	t.Helper()
	raw := a.handleToolCall(ctx, "u1", 1, "en", mcp.ToolCall{Function: mcp.ToolCallFunction{Name: "execute_trade", Arguments: args}})
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("execute_trade returned non-JSON %q: %v", raw, err)
	}
	return out
}

func TestChatMNQEntryReachesTheNT8TraderDoor(t *testing.T) {
	crypto := &routeCandidate{id: "a-crypto", exchange: "binance", running: true, und: &routeUnderlying{}}
	nt8 := nt8Candidate("b-nt8", "MNQ")
	withRoster(t, crypto, nt8)
	a, ctx := routeAgent()

	// The proposal: a CME symbol stays itself (no USDT), is never a stock, and
	// resolves to the NT8 trader. No leverage: a futures contract is not judged
	// by the crypto leverage rule.
	out := proposeTrade(t, a, ctx, `{"action":"open_long","symbol":"mnq","quantity":1,"stop_loss":28950,"take_profit":29100}`)
	if out["status"] != "pending_confirmation" || out["symbol"] != "MNQ" {
		t.Fatalf("a chat MNQ entry must become a pending trade on MNQ (no USDT, never a stock): %v", out)
	}
	id, _ := out["trade_id"].(string)
	confirm := "confirm " + id
	if large, _ := out["requires_large_order_confirmation"].(bool); large {
		confirm = "confirm large " + id
	}

	reply, handled := a.handleTradeConfirmation(ctx, 1, confirm, "en")
	if !handled || !strings.Contains(reply, "Trade executed") {
		t.Fatalf("the owner's confirm must execute the MNQ entry, got handled=%v reply=%q", handled, reply)
	}
	if nt8.opens != 1 || nt8.sym != "MNQ" || nt8.act != "open_long" || nt8.qty != 1 || nt8.stop != 28950 || nt8.target != 29100 {
		t.Fatalf("the NT8 trader's door must receive the chat entry with its own bracket: opens=%d %s %s qty=%.0f SL=%.2f TP=%.2f",
			nt8.opens, nt8.act, nt8.sym, nt8.qty, nt8.stop, nt8.target)
	}
	if crypto.opens != 0 {
		t.Fatalf("a CME entry must never reach a crypto trader (opens=%d)", crypto.opens)
	}
}

// A CME symbol never falls back to a stock or crypto trader, never goes to
// an NT8 trader of ANOTHER instrument (the NT8 broker sends its own wire
// instrument whatever symbol it is handed), and never guesses between two NT8
// traders of the same instrument.
func TestChatCMEEntryResolvesOnlyTheOneNT8TraderOfItsInstrument(t *testing.T) {
	stock := &routeCandidate{id: "a-alpaca", exchange: "alpaca", running: true, und: &routeUnderlying{}}
	crypto := &routeCandidate{id: "b-crypto", exchange: "binance", running: true, und: &routeUnderlying{}}
	es := nt8Candidate("c-es", "ES")
	stopped := nt8Candidate("d-mnq-stopped", "MNQ")
	stopped.running = false
	a, _ := routeAgent()

	withRoster(t, stock, crypto, es, stopped)
	for _, sym := range []string{"MNQ", "MNQU6", "MNQ.c.0"} {
		_, sel, _, err := a.resolveTradeExecutionContext(&TradeAction{Action: "open_long", Symbol: chatTradeSymbol(sym)})
		if err == nil || sel != nil || !strings.Contains(err.Error(), "no running NinjaTrader") {
			t.Fatalf("%s with no running MNQ NT8 trader must refuse by name, got sel=%v err=%v", sym, sel, err)
		}
	}
	_, sel, _, err := a.resolveTradeExecutionContext(&TradeAction{Action: "open_long", Symbol: chatTradeSymbol("ES")})
	if err != nil || sel != es {
		t.Fatalf("ES must resolve to the ES NT8 trader, got sel=%v err=%v", sel, err)
	}

	second := nt8Candidate("e-mnq-2", "MNQ")
	first := nt8Candidate("f-mnq-1", "MNQ")
	withRoster(t, first, second)
	_, sel, _, err = a.resolveTradeExecutionContext(&TradeAction{Action: "open_long", Symbol: "MNQ"})
	if err == nil || sel != nil || !strings.Contains(err.Error(), "never picks") {
		t.Fatalf("two running MNQ NT8 traders must refuse (fail-closed), never pick one: sel=%v err=%v", sel, err)
	}

	// Stocks and crypto resolve exactly as before.
	withRoster(t, stock, crypto)
	if want, sel, _, err := a.resolveTradeExecutionContext(&TradeAction{Symbol: chatTradeSymbol("AAPL")}); err != nil || !want || sel != stock {
		t.Fatalf("AAPL must still resolve to the stock trader: stock=%v sel=%v err=%v", want, sel, err)
	}
	if want, sel, _, err := a.resolveTradeExecutionContext(&TradeAction{Symbol: chatTradeSymbol("btc")}); err != nil || want || sel != crypto {
		t.Fatalf("BTC must still resolve to the crypto trader: stock=%v sel=%v err=%v", want, sel, err)
	}
}

// The classification and the one canonicalizer every chat symbol goes through.
func TestCMESymbolsAreNeverStocks(t *testing.T) {
	for _, sym := range []string{"MNQ", "mnq", "NQ", "ES", "MES", "MNQU6", "MNQ.c.0", "MNQ 06-26"} {
		if isStockSymbol(sym) {
			t.Errorf("isStockSymbol(%q) = true — a CME futures symbol is never a stock", sym)
		}
	}
	for raw, want := range map[string]string{
		"mnq": "MNQ", "MNQ": "MNQ", "MNQU6": "MNQ", "MNQ.c.0": "MNQ", "MNQ 06-26": "MNQ", "es": "ES",
		// Unchanged for stocks and crypto.
		"AAPL": "AAPL", "btc": "BTCUSDT", "ETHUSDT": "ETHUSDT",
	} {
		if got := chatTradeSymbol(raw); got != want {
			t.Errorf("chatTradeSymbol(%q) = %q, want %q", raw, got, want)
		}
	}
	// The watchlist and the crypto-only snapshot keep treating MNQ exactly as
	// they did while it was (wrongly) a stock: no USDT, no crypto fetch.
	if got := normalizeWatchSymbol("mnq"); got != "MNQ" {
		t.Errorf("normalizeWatchSymbol(mnq) = %q, want MNQ", got)
	}
	a, _ := routeAgent()
	if got := a.toolGetMarketSnapshot(`{"symbol":"MNQ"}`); !strings.Contains(got, "crypto symbols only") {
		t.Errorf("get_market_snapshot must refuse a CME symbol (crypto only), got %s", got)
	}
}
