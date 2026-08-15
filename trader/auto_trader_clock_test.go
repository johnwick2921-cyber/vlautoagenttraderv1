package trader

import (
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

func ctAt(t *testing.T, h, m int) time.Time {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("load chicago: %v", err)
	}
	return time.Date(2026, 8, 14, h, m, 0, 0, loc)
}

func TestClockHelpers(t *testing.T) {
	if v, ok := hhmmToMin("14:45"); !ok || v != 885 {
		t.Fatalf("hhmmToMin(14:45) = %d,%v want 885,true", v, ok)
	}
	if _, ok := hhmmToMin("nope"); ok {
		t.Fatalf("bad input should fail")
	}
	if !timeReachedCT(ctAt(t, 14, 0), "13:00") {
		t.Fatalf("14:00 CT should be past 13:00")
	}
	if timeReachedCT(ctAt(t, 14, 0), "15:00") {
		t.Fatalf("14:00 CT should NOT be past 15:00")
	}
	if !timeReachedCT(ctAt(t, 13, 0), "13:00") {
		t.Fatalf("13:00 CT should count as reached (>=)")
	}
}

func TestEffectiveEODFlat(t *testing.T) {
	reg := kernel.DefaultSessionRegistry()
	if got := effectiveEODFlatCT(reg, "2026-08-14", "14:45"); got != "14:45" {
		t.Fatalf("normal day = %q want 14:45", got)
	}
	reg.HalfDays = map[string]string{"2026-11-27": "12:00"}
	if got := effectiveEODFlatCT(reg, "2026-11-27", "14:45"); got != "12:00" {
		t.Fatalf("half-day should pull flat in to 12:00, got %q", got)
	}
	// A half-day close LATER than the config flat keeps the config flat.
	reg.HalfDays = map[string]string{"2026-11-27": "16:00"}
	if got := effectiveEODFlatCT(reg, "2026-11-27", "14:45"); got != "14:45" {
		t.Fatalf("later half-day must not push flat out, got %q", got)
	}
}

func TestClockConfigDefaultsAndGetters(t *testing.T) {
	// No day_plan → getters return the spec defaults.
	base := mkTrader("ninjatrader", nil, "5m")
	if base.lastEntryCT() != "13:00" || base.eodFlatCT() != "14:45" {
		t.Fatalf("defaults = %s/%s want 13:00/14:45", base.lastEntryCT(), base.eodFlatCT())
	}
	yes := true
	at := mkTrader("ninjatrader", &yes, "5m")
	at.config.StrategyConfig.DayPlan.LastEntryCT = "13:30"
	at.config.StrategyConfig.DayPlan.EODFlatCT = "14:30"
	if at.lastEntryCT() != "13:30" || at.eodFlatCT() != "14:30" {
		t.Fatalf("configured = %s/%s", at.lastEntryCT(), at.eodFlatCT())
	}
}

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
