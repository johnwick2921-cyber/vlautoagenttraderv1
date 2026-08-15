package trader

import (
	"testing"
	"time"

	"nofx/market"
	"nofx/store"
)

func TestBarCloseGate(t *testing.T) {
	// Not active → always run; watermark unchanged.
	if run, last := barCloseGate(false, 100, 999, true); !run || last != 100 {
		t.Fatalf("inactive: run=%v last=%v want true/100", run, last)
	}
	// Active + a NEW closed bar → run, advance watermark.
	if run, last := barCloseGate(true, 100, 200, true); !run || last != 200 {
		t.Fatalf("new bar: run=%v last=%v want true/200", run, last)
	}
	// Active + same/older bar → skip (no mid-bar, no re-run).
	if run, last := barCloseGate(true, 200, 200, true); run || last != 200 {
		t.Fatalf("same bar: run=%v last=%v want false/200", run, last)
	}
	if run, _ := barCloseGate(true, 200, 150, true); run {
		t.Fatalf("older bar must not run")
	}
	// Active + no bar available → skip.
	if run, last := barCloseGate(true, 200, 0, false); run || last != 200 {
		t.Fatalf("no bar: run=%v last=%v want false/200", run, last)
	}
}

func mkTrader(exchange string, planEnabled *bool, tf string) *AutoTrader {
	sc := store.StrategyConfig{
		Indicators: store.IndicatorConfig{Klines: store.KlineConfig{PrimaryTimeframe: tf}},
	}
	if planEnabled != nil {
		sc.DayPlan = &store.DayPlanConfig{PlanEnabled: *planEnabled}
	}
	return &AutoTrader{
		exchange: exchange,
		config:   AutoTraderConfig{NinjaTraderSymbol: "MNQ", StrategyConfig: &sc},
	}
}

func TestBarCloseCadenceActive(t *testing.T) {
	yes, no := true, false
	if !mkTrader("ninjatrader", &yes, "5m").barCloseCadenceActive() {
		t.Fatal("futures + day_plan on → active")
	}
	if mkTrader("ninjatrader", &no, "5m").barCloseCadenceActive() {
		t.Fatal("plan off → inactive")
	}
	if mkTrader("ninjatrader", nil, "5m").barCloseCadenceActive() {
		t.Fatal("no day_plan → inactive")
	}
	if mkTrader("binance", &yes, "5m").barCloseCadenceActive() {
		t.Fatal("crypto → inactive even with day_plan")
	}
}

func TestLatestClosedPrimaryBarMs(t *testing.T) {
	prev := market.FuturesBarsProvider
	defer func() { market.FuturesBarsProvider = prev }()

	now := time.Now().UnixMilli()
	market.FuturesBarsProvider = func(symbol, tf string, count int) []market.Kline {
		if symbol != "MNQ" || tf != "5m" {
			t.Errorf("unexpected provider args %s/%s", symbol, tf)
		}
		return []market.Kline{
			{CloseTime: now - 600_000}, // closed
			{CloseTime: now - 300_000}, // closed — the latest
			{CloseTime: now + 300_000}, // still open (future close)
		}
	}
	at := mkTrader("ninjatrader", nil, "5m")
	ms, ok := at.latestClosedPrimaryBarMs()
	if !ok || ms != now-300_000 {
		t.Fatalf("latest closed = %d ok=%v want %d", ms, ok, now-300_000)
	}

	// No provider → not available.
	market.FuturesBarsProvider = nil
	if _, ok := at.latestClosedPrimaryBarMs(); ok {
		t.Fatal("nil provider must return ok=false")
	}
}

func TestPrimaryTimeframeDefault(t *testing.T) {
	if got := mkTrader("ninjatrader", nil, "").primaryTimeframe(); got != "5m" {
		t.Fatalf("default primary tf = %q want 5m", got)
	}
	if got := mkTrader("ninjatrader", nil, "15m").primaryTimeframe(); got != "15m" {
		t.Fatalf("configured primary tf = %q want 15m", got)
	}
}
