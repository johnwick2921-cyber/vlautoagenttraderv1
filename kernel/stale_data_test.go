// B4 — stale-data entry block: a frozen feed refuses NEW entries; a fresh feed
// passes; exits / open-position management are never blocked.
package kernel

import (
	"testing"

	"nofx/market"
)

func mdWith1mBar(barMs int64) *market.Data {
	return &market.Data{
		Symbol: "MNQ",
		TimeframeData: map[string]*market.TimeframeSeriesData{
			"1m": {Timeframe: "1m", Klines: []market.KlineBar{{Time: barMs}}},
		},
	}
}

func TestBarIsStale(t *testing.T) {
	const iv = 60_000
	now := int64(10_000_000)
	if barIsStale(now-iv, iv, now) { // 1×interval old → fresh
		t.Error("1×interval must be fresh")
	}
	if barIsStale(now-2*iv, iv, now) { // exactly 2× → boundary, not stale (>)
		t.Error("exactly 2×interval is the boundary, not stale")
	}
	if !barIsStale(now-2*iv-1, iv, now) { // just past 2× → stale
		t.Error("past 2×interval must be stale")
	}
	if barIsStale(0, iv, now) || barIsStale(now, 0, now) { // unknown → fail-open
		t.Error("unknown inputs must fail-open (not stale)")
	}
}

func TestApplyStaleDataBlock(t *testing.T) {
	now := int64(20_000_000)

	// Fresh 1m bar (30s old) → open stays.
	fresh := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "open_long"}}}
	applyStaleDataBlock(fresh, &Context{MarketDataMap: map[string]*market.Data{"MNQ": mdWith1mBar(now - 30_000)}}, now)
	if fresh.Decisions[0].Action != "open_long" {
		t.Fatalf("fresh feed must leave the entry, got %q", fresh.Decisions[0].Action)
	}

	// Stale 1m bar (5 min old > 120s) → entry neutralized to wait.
	stale := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "open_short"}}}
	applyStaleDataBlock(stale, &Context{MarketDataMap: map[string]*market.Data{"MNQ": mdWith1mBar(now - 300_000)}}, now)
	if stale.Decisions[0].Action != "wait" {
		t.Fatalf("stale feed must block the entry (→wait), got %q", stale.Decisions[0].Action)
	}
	// P0-cleanup (2026-08-19) — a refusal must NEVER be a plain wait: the
	// reason rides the decision (the C2 lesson: six days of silent rewrites).
	if stale.Decisions[0].RefusalReason == "" {
		t.Fatalf("a stale-data refusal must carry RefusalReason — bare wait is forbidden")
	}

	// A CLOSE on a stale feed is NEVER blocked (exits/position-management untouched).
	closing := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "close_long"}}}
	applyStaleDataBlock(closing, &Context{MarketDataMap: map[string]*market.Data{"MNQ": mdWith1mBar(now - 300_000)}}, now)
	if closing.Decisions[0].Action != "close_long" {
		t.Fatalf("a close must never be blocked by stale data, got %q", closing.Decisions[0].Action)
	}

	// No 1m/5m data → fail-open (entry left intact).
	nodata := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "open_long"}}}
	applyStaleDataBlock(nodata, &Context{MarketDataMap: map[string]*market.Data{"MNQ": {Symbol: "MNQ"}}}, now)
	if nodata.Decisions[0].Action != "open_long" {
		t.Fatalf("no bar data must fail-open (leave the entry), got %q", nodata.Decisions[0].Action)
	}
}
