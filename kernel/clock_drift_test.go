// C2 — clock-drift guard: since 2026-08-18 signals are feed-stamped
// (trader/ninjatrader feedNowUTC), so a skewed LOCAL clock can no longer age a
// signal into NT8's 60s stale rejection. The guard now only LOGS skew; entries
// are never converted to wait, exits were never touched.
package kernel

import (
	"testing"

	"nofx/market"
)

func TestClockDriftMs(t *testing.T) {
	const iv = int64(60_000)
	now := int64(10_000_000)

	// Correct clock + live feed: the just-closed 1m bar is labeled one interval ago
	// → drift ≈ 0.
	if d := clockDriftMs(now, now-iv, iv); d != 0 {
		t.Errorf("correct clock drift = %d, want 0", d)
	}
	// Clock AHEAD (or feed lagging) by 120s.
	if d := clockDriftMs(now, now-iv-120_000, iv); d != 120_000 {
		t.Errorf("clock-ahead drift = %d, want 120000", d)
	}
	// Clock BEHIND: feed labeled a bar 120s in the future.
	if d := clockDriftMs(now, now-iv+120_000, iv); d != -120_000 {
		t.Errorf("clock-behind drift = %d, want -120000", d)
	}
}

func TestApplyClockDriftBlock_InjectedSkew(t *testing.T) {
	const iv = int64(60_000)
	now := int64(20_000_000)
	freshBar := now - iv // correct-clock freshest 1m bar (labeled one interval ago)

	ctxWith := func(barMs int64) *Context {
		return &Context{MarketDataMap: map[string]*market.Data{"MNQ": mdWith1mBar(barMs)}}
	}

	// Correct clock → entry stays.
	ok := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "open_long"}}}
	applyClockDriftBlock(ok, ctxWith(freshBar), now)
	if ok.Decisions[0].Action != "open_long" {
		t.Fatalf("correct clock must leave the entry, got %q", ok.Decisions[0].Action)
	}

	// Boundary: exactly 60s skew → NOT blocked (guard uses strict >).
	edge := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "open_long"}}}
	applyClockDriftBlock(edge, ctxWith(freshBar-clockDriftToleranceMs), now)
	if edge.Decisions[0].Action != "open_long" {
		t.Fatalf("exactly 60s skew is the boundary (not blocked), got %q", edge.Decisions[0].Action)
	}

	// Injected clock-AHEAD skew (+5 min): entry must now PROCEED — the signal
	// is stamped with the feed clock, so local skew cannot mis-time it. The
	// guard only logs the skew.
	ahead := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "open_long"}}}
	applyClockDriftBlock(ahead, ctxWith(freshBar), now+300_000)
	if ahead.Decisions[0].Action != "open_long" {
		t.Fatalf("feed-stamped entries must survive local clock-ahead skew, got %q", ahead.Decisions[0].Action)
	}

	// Injected clock-BEHIND skew (−5 min): same — observed, not blocked.
	behind := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "open_short"}}}
	applyClockDriftBlock(behind, ctxWith(freshBar+300_000), now)
	if behind.Decisions[0].Action != "open_short" {
		t.Fatalf("feed-stamped entries must survive local clock-behind skew, got %q", behind.Decisions[0].Action)
	}

	// A CLOSE under a skewed clock is NEVER blocked.
	closing := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "close_long"}}}
	applyClockDriftBlock(closing, ctxWith(freshBar+300_000), now)
	if closing.Decisions[0].Action != "close_long" {
		t.Fatalf("a close must never be blocked by clock drift, got %q", closing.Decisions[0].Action)
	}

	// No 1m/5m data → fail-open (entry left intact).
	nodata := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "open_long"}}}
	applyClockDriftBlock(nodata, &Context{MarketDataMap: map[string]*market.Data{"MNQ": {Symbol: "MNQ"}}}, now+300_000)
	if nodata.Decisions[0].Action != "open_long" {
		t.Fatalf("no bar data must fail-open, got %q", nodata.Decisions[0].Action)
	}
}
