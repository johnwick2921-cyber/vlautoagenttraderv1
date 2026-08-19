package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/market"
	"nofx/store"
)

// U1 (stale-bar dispatch) — feed problems become loud. T7 runs here with a
// SIMULATED gap: an 11-real-minute NT8 kill is not drivable from WSL (the AddOn
// reconnects every 5s, iptables needs sudo, and stopping the Go listener would
// take the monitoring down with it). The 08-19 00:53 outage is the live
// precedent these assertions encode.

func feedTrader(t *testing.T) (*AutoTrader, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "fw.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	at := &AutoTrader{id: "t1", store: st, exchange: "ninjatrader", startTime: time.Now().Add(-time.Hour)}
	at.config.Exchange = "ninjatrader"
	at.config.StrategyConfig = &store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	return at, st
}

func withProvider(t *testing.T, bars []market.Kline) {
	t.Helper()
	old := market.FuturesBarsProvider
	market.FuturesBarsProvider = func(string, string, int) []market.Kline { return bars }
	t.Cleanup(func() { market.FuturesBarsProvider = old })
}

func feedAlerts(t *testing.T, st *store.Store, kind string) int {
	t.Helper()
	alerts, _ := st.Alert().List("t1", 50)
	n := 0
	for _, a := range alerts {
		if a.Kind == kind {
			n++
		}
	}
	return n
}

// A CME-open Monday noon CT instant, so IsCMEOpen(now)=true deterministically.
func cmeOpenNow(t *testing.T) time.Time {
	return ctTimeAt(t, "2026-08-17", "12:00")
}

// T7a — an 11-minute bar gap while CME is open fires the P0 alert, once per gap.
func TestFeedDownAlertFiresOnAnElevenMinuteGap(t *testing.T) {
	at, st := feedTrader(t)
	now := cmeOpenNow(t)
	withProvider(t, []market.Kline{{OpenTime: now.UnixMilli() - 11*60_000 - 60_000}})

	at.checkFeedDown(now)
	if got := feedAlerts(t, st, "feed-down"); got != 1 {
		t.Fatalf("an 11-min gap while CME is open must fire exactly one P0, got %d", got)
	}
	// Same gap re-checked → deduped (one outage, one alert).
	at.checkFeedDown(now.Add(30 * time.Second))
	if got := feedAlerts(t, st, "feed-down"); got != 1 {
		t.Fatalf("the SAME gap must not re-alert, got %d", got)
	}
}

// A live feed never alerts; a short gap (< 10 min) never alerts.
func TestFeedDownAlertStaysQuietOnALiveFeed(t *testing.T) {
	at, st := feedTrader(t)
	now := cmeOpenNow(t)
	withProvider(t, []market.Kline{{OpenTime: now.UnixMilli() - 90_000}})
	at.checkFeedDown(now)
	withProvider(t, []market.Kline{{OpenTime: now.UnixMilli() - 8*60_000}})
	at.checkFeedDown(now)
	if got := feedAlerts(t, st, "feed-down"); got != 0 {
		t.Fatalf("a live/short-gap feed must not alert, got %d", got)
	}
}

// CME closed → no alert regardless of age (the weekend is not an outage).
func TestFeedDownAlertRespectsTheCalendar(t *testing.T) {
	at, st := feedTrader(t)
	saturday := ctTimeAt(t, "2026-08-15", "12:00")
	withProvider(t, []market.Kline{{OpenTime: saturday.UnixMilli() - 3*3_600_000}})
	at.checkFeedDown(saturday)
	if got := feedAlerts(t, st, "feed-down"); got != 0 {
		t.Fatalf("a closed market must never read as a feed outage, got %d", got)
	}
}

// T7b — the planner preflight refuses on an EMPTY cache with the provider
// wired (the exact 08-19 incident shape), alerts P1, and passes when bars are
// fresh. A nil provider (test/crypto) fails open.
func TestPlannerPreflight(t *testing.T) {
	at, st := feedTrader(t)

	withProvider(t, nil) // provider wired, cache empty — the incident
	if at.plannerPreflight("ASIA", "2026-08-18") {
		t.Fatal("an empty bar cache must refuse the planner call")
	}
	if got := feedAlerts(t, st, "planner-preflight"); got != 1 {
		t.Fatalf("the refusal must alert P1, got %d", got)
	}

	withProvider(t, []market.Kline{{OpenTime: time.Now().UnixMilli() - 60_000}})
	if !at.plannerPreflight("ASIA", "2026-08-18") {
		t.Fatal("a fresh feed must pass preflight")
	}

	market.FuturesBarsProvider = nil
	if !at.plannerPreflight("ASIA", "2026-08-18") {
		t.Fatal("no provider at all (test/crypto) must fail open")
	}
}
