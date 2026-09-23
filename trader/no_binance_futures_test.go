package trader

import (
	"fmt"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"nofx/hook"
	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// ── W-NO-BINANCE A — the CME futures path makes ZERO Binance calls ──────────
//
// market.GetWithExchange called getOpenInterestData + getFundingRate
// (fapi.binance.com) for MNQ on every AI open, close and admission live-price
// read. The trap FAILS the test on any request to a host containing "binance"
// and answers every request offline. It is installed on BOTH ways out of the
// process (critic G3 — the earlier premise that every market HTTP call goes
// through market.NewAPIClient was false): hook.SET_HTTP_CLIENT, which the
// Binance OI/funding client consults, AND http.DefaultTransport, which the
// clients that bypass the hook use (CoinAnk's package client, historical.go,
// any client with a nil Transport). The drivers are the production call sites.

type binanceTrap struct {
	t     *testing.T
	mu    sync.Mutex
	hosts []string
}

func (b *binanceTrap) RoundTrip(r *http.Request) (*http.Response, error) {
	b.mu.Lock()
	b.hosts = append(b.hosts, r.URL.Host+r.URL.Path)
	b.mu.Unlock()
	if strings.Contains(strings.ToLower(r.URL.Host), "binance") {
		b.t.Errorf("the futures path called %s%s — W-NO-BINANCE A forbids any Binance host", r.URL.Host, r.URL.Path)
	}
	return nil, fmt.Errorf("no network in this test (%s)", r.URL.Host)
}

func (b *binanceTrap) seen() []string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]string(nil), b.hosts...)
}

func trapBinance(t *testing.T) *binanceTrap {
	t.Helper()
	b := &binanceTrap{t: t}
	prev, had := hook.Hooks[hook.SET_HTTP_CLIENT]
	prevEnabled := hook.EnableHooks
	hook.EnableHooks = true
	hook.RegisterHook(hook.SET_HTTP_CLIENT, func(args ...any) any {
		return &hook.SetHttpClientResult{Client: &http.Client{Transport: b, Timeout: time.Second}}
	})
	prevDT := http.DefaultTransport
	http.DefaultTransport = b
	t.Cleanup(func() {
		http.DefaultTransport = prevDT
		if had {
			hook.Hooks[hook.SET_HTTP_CLIENT] = prev
		} else {
			delete(hook.Hooks, hook.SET_HTTP_CLIENT)
		}
		hook.EnableHooks = prevEnabled
	})
	return b
}

// futuresTape installs a live-looking NT8 bar provider (5m + 1h, prices that
// move so the staleness guard passes) and returns the newest 5m close.
func futuresTape(t *testing.T) float64 {
	t.Helper()
	mk := func(step time.Duration, n int) []market.Kline {
		out := make([]market.Kline, n)
		start := time.Now().Add(-time.Duration(n) * step)
		for i := range out {
			c := 29000 + float64(i%9)*2.25
			ot := start.Add(time.Duration(i) * step).UnixMilli()
			out[i] = market.Kline{OpenTime: ot, Open: c - 1, High: c + 2, Low: c - 2, Close: c, Volume: 100, CloseTime: ot + step.Milliseconds() - 1}
		}
		return out
	}
	m5, h1 := mk(5*time.Minute, 200), mk(time.Hour, 200)
	prev := market.FuturesBarsProvider
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline {
		switch tf {
		case "5m":
			return tailOf(m5, count)
		case "1h":
			return tailOf(h1, count)
		}
		return nil
	}
	t.Cleanup(func() { market.FuturesBarsProvider = prev })
	return m5[len(m5)-1].Close
}

// The market read every futures call site uses: no Binance request, and OI /
// funding ABSENT (nil / not known), never a fabricated 0.
func TestFuturesMarketReadMakesNoBinanceCall(t *testing.T) {
	trap := trapBinance(t)
	last := futuresTape(t)
	d, err := market.GetWithExchange("MNQ", "ninjatrader")
	if err != nil {
		t.Fatalf("fixture: the futures market read failed: %v", err)
	}
	if d.CurrentPrice != last || d.OpenInterest != nil || d.FundingRateKnown {
		t.Fatalf("futures read: price %.2f (want %.2f), OI %+v (want nil), funding known %v (want false)", d.CurrentPrice, last, d.OpenInterest, d.FundingRateKnown)
	}
	tf, err := market.GetWithTimeframes("MNQ", []string{"5m", "1h"}, "5m", 50)
	if err != nil {
		t.Fatalf("fixture: GetWithTimeframes on futures failed: %v", err)
	}
	if tf.OpenInterest != nil || tf.FundingRateKnown {
		t.Fatalf("GetWithTimeframes on futures must report OI/funding ABSENT, got OI %+v known %v", tf.OpenInterest, tf.FundingRateKnown)
	}
	if hosts := trap.seen(); len(hosts) != 0 {
		t.Fatalf("the futures market read made %d outbound request(s): %v", len(hosts), hosts)
	}
}

// The AI open's send half (executeOpenLongWithRecord) over the real TCPTrader:
// it reads the market (auto_trader_orders.go) and reaches no Binance host.
func TestAIOpenSendHalfMakesNoBinanceCall(t *testing.T) {
	trap := trapBinance(t)
	w := newAIEntryWire(t)
	w.at.positionFirstSeenTime = map[string]int64{} // as NewAutoTrader builds it
	last := futuresTape(t)
	d := &kernel.Decision{Action: "open_long", Symbol: "MNQ", Leverage: 1, Confidence: 70, StopLoss: 28950, TakeProfit: 29100}
	rec := &store.DecisionAction{Action: "open_long", Symbol: "MNQ"}
	_ = w.at.executeOpenLongWithRecord(d, rec)
	if rec.Price != last {
		t.Fatalf("fixture: the send half must have read the market (record price %.2f, want %.2f; err %q)", rec.Price, last, rec.Error)
	}
	for _, h := range trap.seen() {
		if strings.Contains(strings.ToLower(h), "binance") {
			t.Fatalf("the AI open reached %s", h)
		}
	}
}

// The agent door's admission reads the live price (entry_admission.go, the
// EntryGate step) — no Binance request there either.
func TestAdmissionLiveReadMakesNoBinanceCall(t *testing.T) {
	trap := trapBinance(t)
	withMaintenanceDir(t)
	futuresTape(t)
	at := mkPlanTrader(&store.DayPlanConfig{PlanEnabled: true, PlanMode: "advisory"})
	at.id = "no-binance-admission"
	kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{ActivePlan: func(string) *kernel.ActivePlan { return nil }})
	t.Cleanup(func() { kernel.SetTraderPlanProviders(at.id, kernel.TraderPlanProviders{}) })
	nyMidday := time.Date(2026, 9, 15, 16, 0, 0, 0, time.UTC) // 11:00 CT Tuesday
	if reason, refused := at.AdmitManualEntryAt("MNQ", "open_long", nyMidday); refused {
		t.Fatalf("fixture: the door must reach its EntryGate live read (admitted), got %q", reason)
	}
	for _, h := range trap.seen() {
		if strings.Contains(strings.ToLower(h), "binance") {
			t.Fatalf("the admission live read reached %s", h)
		}
	}
}

// Critic G1 — the NinjaTrader venue IS the futures path: a non-CME symbol
// there is REFUSED by the market read, never sent down the crypto branch
// (CoinAnk exchange=Binance + fapi OI/funding). Zero outbound requests, on
// either way out of the process.
func TestNT8VenueRefusesANonCMESymbolWithNoOutboundCall(t *testing.T) {
	trap := trapBinance(t)
	futuresTape(t)
	d, err := market.GetWithExchange("BTCUSDT", "ninjatrader")
	if err == nil || d != nil || !strings.Contains(err.Error(), "non-CME symbol is refused") {
		t.Fatalf("a non-CME symbol on the NinjaTrader venue must be refused, got data=%v err=%v", d, err)
	}
	if hosts := trap.seen(); len(hosts) != 0 {
		t.Fatalf("the refused read still went out: %v", hosts)
	}
}

// CTO F2 — fail-closed at the SOURCE: a trader on the NinjaTrader venue whose
// NT8 symbol or strategy static coin is not a CME futures symbol is REFUSED at
// NewAutoTrader (the load path), named, and never started.
func TestNT8TraderWithANonCMESymbolIsRefusedAtLoad(t *testing.T) {
	for _, tc := range []struct {
		name string
		cfg  AutoTraderConfig
		bad  string
	}{
		{"nt8 symbol", AutoTraderConfig{Name: "nb-f2-a", Exchange: "ninjatrader", NinjaTraderSymbol: "BTCUSDT"}, "BTCUSDT"},
		{"static coin", AutoTraderConfig{Name: "nb-f2-b", Exchange: "ninjatrader", NinjaTraderSymbol: "MNQ",
			StrategyConfig: &store.StrategyConfig{CoinSource: store.CoinSourceConfig{StaticCoins: []string{"MNQ", "ETHUSDT"}}}}, "ETHUSDT"},
	} {
		at, err := NewAutoTrader(tc.cfg, nil, "u")
		if at != nil || err == nil || !strings.Contains(err.Error(), "not CME futures symbols") || !strings.Contains(err.Error(), tc.bad) {
			t.Errorf("%s: an NT8-venue trader trading %s must be REFUSED at load, named; got trader=%v err=%v", tc.name, tc.bad, at != nil, err)
		}
	}
	// A CME symbol passes THIS check (it may still fail later for other reasons).
	_, err := NewAutoTrader(AutoTraderConfig{Name: "nb-f2-c", Exchange: "ninjatrader", NinjaTraderSymbol: "MNQ"}, nil, "u")
	if err != nil && strings.Contains(err.Error(), "not CME futures symbols") {
		t.Fatalf("a CME symbol must not be refused by the venue check: %v", err)
	}
}
