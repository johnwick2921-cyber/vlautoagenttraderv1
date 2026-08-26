package trader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"nofx/market"
	"nofx/store"
)

func readClockSource(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile("auto_trader_clock.go")
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// P10 (owner ruling 2026-08-19) — the Studio scan interval IS the decision
// cadence; bar-close is a selectable legacy mode.

func TestCadenceModeResolution(t *testing.T) {
	at := &AutoTrader{}
	for cfg, want := range map[string]string{
		"":          CadenceInterval, // P10 default
		"interval":  CadenceInterval,
		"bar_close": CadenceBarClose, // legacy, explicit only
		"garbage":   CadenceInterval, // junk never invents the stricter gate
	} {
		at.config.CadenceMode = cfg
		if got := at.cadenceMode(); got != want {
			t.Errorf("cadenceMode(%q) = %q, want %q", cfg, got, want)
		}
	}
}

// E15 logic — the no-new-data dedup: identical newest-bar state while FLAT
// skips; any bar mutation or an open position runs the cycle.
func TestSkipNoNewData(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "c.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	at := &AutoTrader{id: "tc1", exchange: "ninjatrader", store: st}
	at.config.Exchange = "ninjatrader"
	at.config.StrategyConfig = &store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}

	old := market.FuturesBarsProvider
	t.Cleanup(func() { market.FuturesBarsProvider = old })
	bar := market.Kline{OpenTime: 1_000_000, Open: 100, High: 101, Low: 99, Close: 100.5, Volume: 12}
	market.FuturesBarsProvider = func(string, string, int) []market.Kline { return []market.Kline{bar} }

	now := time.Now()
	if at.skipNoNewData(now) {
		t.Fatal("first sighting of a bar state must RUN (signature primed, not skipped)")
	}
	if !at.skipNoNewData(now) {
		t.Fatal("identical bar state + flat must skip (cycle_skip=no_new_data)")
	}
	// The forming bar ticks (close/volume move) → run.
	bar.Close, bar.Volume = 100.75, 13
	if at.skipNoNewData(now) {
		t.Fatal("a mutated forming bar must run the cycle")
	}
	// Identical again, but HOLDING → run (in-position cycles are the heartbeat).
	openAPosition(t, st, "tc1")
	if at.skipNoNewData(now) {
		t.Fatal("an open position must never be skipped by the dedup")
	}
	// No provider → never skip.
	market.FuturesBarsProvider = nil
	if at.skipNoNewData(now) {
		t.Fatal("no bar state to compare → never invent a skip")
	}
}

// E14 — mode=bar_close keeps the legacy gate byte-identical: barCloseGate is
// untouched and still governs; interval mode bypasses it entirely.
func TestBarCloseModeKeepsLegacyGate(t *testing.T) {
	// The pure gate (pinned since day-plan P2): no new close → idle.
	if run, _ := barCloseGate(true, 1000, 1000, true); run {
		t.Fatal("bar_close: same watermark must idle")
	}
	if run, _ := barCloseGate(true, 1000, 2000, true); !run {
		t.Fatal("bar_close: a new close must run")
	}
	// Mode wiring: tickOnce consults the gate ONLY in bar_close mode — pinned
	// at source level (the same guard style as the other ordering contracts).
	src := readClockSource(t)
	gateCall := src[strings.Index(src, "P10 (owner ruling"):]
	if !strings.Contains(gateCall, `at.cadenceMode() == CadenceBarClose`) ||
		!strings.Contains(gateCall, `at.cadenceMode() == CadenceInterval && at.skipNoNewData`) {
		t.Fatal("tickOnce must gate bar-close behavior on the MODE and dedup only interval mode")
	}
}
