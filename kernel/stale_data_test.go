// B4 — stale-data entry block: a frozen feed refuses NEW entries; a fresh feed
// passes; exits / open-position management are never blocked.
package kernel

import (
	"strings"
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

// THE AGE CONTRACT (stale-bar dispatch 2026-08-19): a TF is stale only when the
// bar that SHOULD exist for the current period is missing/old —
// newest_open < expected_open − period − grace. The old `> 2×interval` math was
// calibrated against END-stamped bars; the open-stamp correction (e9084e38)
// removed one period of slack and armed a false-positive on slow calls.
func TestBarIsStale(t *testing.T) {
	const iv = 60_000
	const grace = DefaultStaleBarGraceSeconds * 1000
	now := int64(10_000_037) // deliberately unaligned
	expectedOpen := (now / iv) * iv

	// T1 — mid-period, open-stamped: the forming bar and the previous bar are
	// both FRESH, at every point inside the period.
	if barIsStale(expectedOpen, iv, now) {
		t.Error("the current forming bar is never stale")
	}
	if barIsStale(expectedOpen-iv, iv, now) {
		t.Error("the immediately previous bar is fresh (forming bar may simply not have arrived)")
	}
	// Boundary: exactly at expected_open − period − grace → NOT stale (strict <).
	if barIsStale(expectedOpen-iv-grace, iv, now) {
		t.Error("the grace boundary itself is not stale")
	}
	// T2 — one ms past the grace boundary → stale.
	if !barIsStale(expectedOpen-iv-grace-1, iv, now) {
		t.Error("past period+grace must be stale (the bar that should exist is missing)")
	}
	// Two full periods gone → stale, plainly.
	if !barIsStale(expectedOpen-2*iv, iv, now) {
		t.Error("two periods behind must be stale")
	}
	if barIsStale(0, iv, now) || barIsStale(now, 0, now) { // unknown → fail-open
		t.Error("unknown inputs must fail-open (not stale)")
	}
}

// T4 — per-timeframe independence: each TF judged against ITS OWN period. The
// same absolute age that kills a 1m feed is comfortably fresh on 15m.
func TestStalePerTimeframeIndependence(t *testing.T) {
	now := int64(20_000_000)
	age := int64(200_000) // 200s old
	if !barIsStale(now-age, 60_000, now) {
		t.Error("a 200s-old 1m newest bar is stale (its period is 60s)")
	}
	if barIsStale(now-age, 900_000, now) {
		t.Error("a 200s-old 15m newest bar is FRESH (its period is 900s)")
	}
	if barIsStale(now-age, 300_000, now) {
		t.Error("a 200s-old 5m newest bar is fresh (previous-bar window)")
	}
}

// The evaluation clock is SNAPSHOT time: a legal slow AI call must never turn a
// live-at-snapshot feed into stale. (The supersession guard owns call latency.)
func TestStaleEvaluatesAtSnapshotTimeNotPostCall(t *testing.T) {
	snapshot := int64(30_000_000)
	postCall := snapshot + 290_000 // a legal 290s call under the 300s ceiling

	fd := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "open_long"}}}
	ctx := &Context{
		SnapshotMs:    snapshot,
		MarketDataMap: map[string]*market.Data{"MNQ": mdWith1mBar(snapshot - 20_000)},
	}
	applyStaleDataBlock(fd, ctx, postCall)
	if fd.Decisions[0].Action != "open_long" {
		t.Fatalf("a feed 20s fresh AT SNAPSHOT must pass regardless of call duration, got %q (%s)",
			fd.Decisions[0].Action, fd.Decisions[0].RefusalReason)
	}

	// Same post-call clock, but the feed was ALREADY dead at snapshot → stale.
	fd2 := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "open_short"}}}
	ctx2 := &Context{
		SnapshotMs:    snapshot,
		MarketDataMap: map[string]*market.Data{"MNQ": mdWith1mBar(snapshot - 400_000)},
	}
	applyStaleDataBlock(fd2, ctx2, postCall)
	if fd2.Decisions[0].Action != "wait" {
		t.Fatal("a feed dead at snapshot must still be refused")
	}
	if !strings.Contains(fd2.Decisions[0].RefusalReason, "hint=") {
		t.Errorf("the discard must self-diagnose with a verdict hint, got %q", fd2.Decisions[0].RefusalReason)
	}
}

// T3 — drift self-identification: with the LIVE provider fresh but the local
// clock skewed past C2 tolerance, the hint says clock, not feed.
func TestStaleDiscardHintsClockOnDrift(t *testing.T) {
	old := market.FuturesBarsProvider
	defer func() { market.FuturesBarsProvider = old }()
	now := int64(40_000_000)
	// A skewed local clock ages every reading uniformly — the snapshot's newest
	// bar reads 400s old AND the live bar reads far older than one period.
	market.FuturesBarsProvider = func(symbol, tf string, n int) []market.Kline {
		// live bar reads 180s old → drift = 180s − 60s = 120s, decisively past
		// the 60s C2 tolerance (a 120s reading would land exactly ON it).
		return []market.Kline{{OpenTime: now - 180_000}}
	}
	fd := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "open_long"}}}
	ctx := &Context{
		SnapshotMs:    now,
		MarketDataMap: map[string]*market.Data{"MNQ": mdWith1mBar(now - 400_000)},
	}
	applyStaleDataBlock(fd, ctx, now)
	if fd.Decisions[0].Action != "wait" {
		t.Fatal("stale snapshot must refuse")
	}
	if !strings.Contains(fd.Decisions[0].RefusalReason, "hint=clock") {
		t.Errorf("want hint=clock for a drift-shaped discard, got %q", fd.Decisions[0].RefusalReason)
	}
}

func TestApplyStaleDataBlock(t *testing.T) {
	now := int64(20_000_000)

	// Fresh 1m bar (30s old) → open stays.
	fresh := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "open_long"}}}
	applyStaleDataBlock(fresh, &Context{SnapshotMs: now, MarketDataMap: map[string]*market.Data{"MNQ": mdWith1mBar(now - 30_000)}}, now)
	if fresh.Decisions[0].Action != "open_long" {
		t.Fatalf("fresh feed must leave the entry, got %q", fresh.Decisions[0].Action)
	}

	// Stale 1m bar (5 min old > 120s) → entry neutralized to wait.
	stale := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "open_short"}}}
	applyStaleDataBlock(stale, &Context{SnapshotMs: now, MarketDataMap: map[string]*market.Data{"MNQ": mdWith1mBar(now - 300_000)}}, now)
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
	applyStaleDataBlock(closing, &Context{SnapshotMs: now, MarketDataMap: map[string]*market.Data{"MNQ": mdWith1mBar(now - 300_000)}}, now)
	if closing.Decisions[0].Action != "close_long" {
		t.Fatalf("a close must never be blocked by stale data, got %q", closing.Decisions[0].Action)
	}

	// No 1m/5m data → fail-open (entry left intact).
	nodata := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "open_long"}}}
	applyStaleDataBlock(nodata, &Context{SnapshotMs: now, MarketDataMap: map[string]*market.Data{"MNQ": {Symbol: "MNQ"}}}, now)
	if nodata.Decisions[0].Action != "open_long" {
		t.Fatalf("no bar data must fail-open (leave the entry), got %q", nodata.Decisions[0].Action)
	}
}

// P8 (ledger-close 2026-08-19) — SnapshotMs ABSENT is a WARN + fail-open pass,
// never a silent time.Now() evaluation (the hidden-clock defect class). The
// injected fake clock proves the predicate cannot run without a snapshot: a
// feed that WOULD read stale on the caller's clock still passes.
func TestStaleGateFailsOpenWithoutSnapshotMs(t *testing.T) {
	now := int64(20_000_000)
	fd := &FullDecision{Decisions: []Decision{{Symbol: "MNQ", Action: "open_long"}}}
	ctx := &Context{ // SnapshotMs deliberately absent
		TraderID:      "t1",
		MarketDataMap: map[string]*market.Data{"MNQ": mdWith1mBar(now - 300_000)},
	}
	applyStaleDataBlock(fd, ctx, now)
	if fd.Decisions[0].Action != "open_long" {
		t.Fatalf("absent SnapshotMs must fail OPEN (pass unassessed), got %q", fd.Decisions[0].Action)
	}
	if fd.Decisions[0].RefusalReason != "" {
		t.Fatalf("fail-open must not stamp a refusal, got %q", fd.Decisions[0].RefusalReason)
	}
}
