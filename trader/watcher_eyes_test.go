package trader

import (
	"os"
	"strings"
	"testing"

	"nofx/kernel"
	"nofx/market"
)

// hotfix/watcher-eyes (2026-08-19) — regression tests for the anatomy census
// findings U1/U2/U3 (PR #56 FAIL register).

// U2 — a post_exit kick bypasses BOTH cadence gates and the dodge, exactly
// once; a stale_dodge kick bypasses only the dodge.
func TestNoteKickBypassFlags(t *testing.T) {
	at := &AutoTrader{}
	at.noteKick("post_exit")
	if !at.skipCadenceOnce || !at.skipDodgeOnce {
		t.Fatal("post_exit kick must arm both cadence and dodge bypasses (the promised immediate rescan)")
	}
	at = &AutoTrader{}
	at.noteKick("stale_dodge")
	if at.skipCadenceOnce {
		t.Fatal("stale_dodge must NOT bypass the cadence gates")
	}
	if !at.skipDodgeOnce {
		t.Fatal("stale_dodge must bypass the dodge (no re-defer at the boundary)")
	}
	at = &AutoTrader{}
	at.noteKick("")
	if at.skipCadenceOnce || at.skipDodgeOnce {
		t.Fatal("timer ticks arm nothing")
	}
}

// U1 — EnsureMarketData stamps the snapshot instant even when the map is
// already populated (idempotent fetch, never a stale clock).
func TestEnsureMarketDataStampsSnapshot(t *testing.T) {
	// An already-populated map skips the fetch, but SnapshotMs must still be
	// stamped NOW.
	c := &kernel.Context{}
	c.MarketDataMap = map[string]*market.Data{"MNQ": {}}
	if err := kernel.EnsureMarketData(c, nil); err != nil {
		t.Fatalf("populated map must not fetch (got %v)", err)
	}
	if c.SnapshotMs <= 0 {
		t.Fatal("EnsureMarketData must stamp SnapshotMs (the hidden-clock class)")
	}
}

// U1/U3 source contracts — the watch cycle must fetch market truth before the
// observer prompt, and must NOT feed the dodge's latency ring.
func TestWatcherEyesSourceContract(t *testing.T) {
	src, err := os.ReadFile("auto_trader_watcher.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)
	fetchIdx := strings.Index(s, "kernel.EnsureMarketData")
	promptIdx := strings.Index(s, "BuildObserverSystemPrompt")
	if fetchIdx < 0 {
		t.Fatal("runWatchCycle no longer fetches market data — the observer would be market-blind again (U1)")
	}
	if promptIdx >= 0 && fetchIdx > promptIdx {
		t.Fatal("market fetch must happen BEFORE the observer prompt is built (U1)")
	}
	if strings.Contains(s, "recordAICallMs") {
		t.Fatal("watch-call latencies must not pollute the dodge ring (U3)")
	}
}
