package kernel

import (
	"testing"
	"time"

	"nofx/market"
)

// W16/R7 — the Plan-3 T22 stale/drift block was UNREACHABLE and has been removed.
//
// It sat above fetchMarketDataWithStrategy guarded by len(ctx.MarketDataMap) > 0,
// but the map is created and filled ~70 lines BELOW, in the same function, and a
// fresh Context is built every cycle. So the guard was false on every production
// evaluation and the gate never once ran.
//
// These tests pin what ACTUALLY protects entries — applyStaleDataBlock (B4),
// which runs after the fetch — so its removal cannot be mistaken for a loss of
// coverage, and so the coverage itself cannot silently rot the way T22 did.
func TestW16StaleEntriesAreStillBlockedAfterT22Removal(t *testing.T) {
	nowMs := time.Now().UnixMilli()
	// Freshest 1m bar is 5 minutes old — well past 2× the 1m interval.
	stale := &market.Data{TimeframeData: map[string]*market.TimeframeSeriesData{
		"1m": {Klines: []market.KlineBar{{Time: nowMs - 5*60_000, Close: 30000}}},
	}}
	ctx := &Context{TraderID: "t1", MarketDataMap: map[string]*market.Data{"MNQ": stale}}

	fd := &FullDecision{Decisions: []Decision{
		{Action: "open_long", Symbol: "MNQ"},
		{Action: "open_short", Symbol: "MNQ"},
		{Action: "close_long", Symbol: "MNQ"},
		{Action: "hold", Symbol: "MNQ"},
	}}
	applyStaleDataBlock(fd, ctx, nowMs)

	if fd.Decisions[0].Action != "wait" {
		t.Errorf("open_long on a stale feed must become wait, got %q", fd.Decisions[0].Action)
	}
	if fd.Decisions[1].Action != "wait" {
		t.Errorf("open_short on a stale feed must become wait, got %q", fd.Decisions[1].Action)
	}
	// Exits and position management are never blocked — a stale feed must not
	// trap an open position.
	if fd.Decisions[2].Action != "close_long" {
		t.Errorf("close_long must NOT be blocked by stale data, got %q", fd.Decisions[2].Action)
	}
	if fd.Decisions[3].Action != "hold" {
		t.Errorf("hold must be untouched, got %q", fd.Decisions[3].Action)
	}
}

func TestW16FreshFeedIsNotBlocked(t *testing.T) {
	nowMs := time.Now().UnixMilli()
	fresh := &market.Data{TimeframeData: map[string]*market.TimeframeSeriesData{
		"1m": {Klines: []market.KlineBar{{Time: nowMs - 30_000, Close: 30000}}},
	}}
	ctx := &Context{TraderID: "t1", MarketDataMap: map[string]*market.Data{"MNQ": fresh}}
	fd := &FullDecision{Decisions: []Decision{{Action: "open_long", Symbol: "MNQ"}}}
	applyStaleDataBlock(fd, ctx, nowMs)
	if fd.Decisions[0].Action != "open_long" {
		t.Fatalf("a 30s-old 1m bar is fresh; the entry must survive, got %q", fd.Decisions[0].Action)
	}
}

// The documented fail-open: with no 1m/5m data at all the gate cannot assess
// freshness and does not block. Pinned so the behavior is a choice, not a
// surprise — it is listed in the W16 report as the residual gap.
func TestW16NoIntradayDataFailsOpen(t *testing.T) {
	nowMs := time.Now().UnixMilli()
	only15m := &market.Data{TimeframeData: map[string]*market.TimeframeSeriesData{
		"15m": {Klines: []market.KlineBar{{Time: nowMs - 10*60_000, Close: 30000}}},
	}}
	ctx := &Context{TraderID: "t1", MarketDataMap: map[string]*market.Data{"MNQ": only15m}}
	fd := &FullDecision{Decisions: []Decision{{Action: "open_long", Symbol: "MNQ"}}}
	applyStaleDataBlock(fd, ctx, nowMs)
	if fd.Decisions[0].Action != "open_long" {
		t.Fatalf("documented fail-open: no 1m/5m data means the gate cannot assess, got %q", fd.Decisions[0].Action)
	}
}
