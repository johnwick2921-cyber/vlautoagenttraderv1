package trader

import (
	"os"
	"strings"
	"testing"
)

// IN-POSITION CONTRACT guard (in-position silence fix 2026-08-19).
//
// The bug: skip-while-open returned from runCycle BEFORE buildTradingContext
// and saveEquitySnapshot, so for the whole life of every position the equity
// curve froze and the decision feed went silent (position #521: 1 scan in 82
// minutes; #522: 1 scan in 16 minutes — the owner watched cycle #20149 stop at
// 07:17:32 the moment the trade opened and resume only after the stop-loss).
//
// The contract: skipping the AI call in-position to save spend is acceptable
// ONLY as a documented branch AFTER snapshot+broadcast — never before. This
// test pins that source order so a refactor can't silently reintroduce the
// freeze. (Same style precedent as the TZ-guard test: assert on the source.)
func TestSkipWhileOpenSitsBelowSnapshotAndEquity(t *testing.T) {
	src, err := os.ReadFile("auto_trader_loop.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)

	skip := strings.Index(s, "at.skipWhileOpen()")
	ctx := strings.Index(s, "at.buildTradingContext()")
	equity := strings.Index(s, "at.saveEquitySnapshot(")
	if skip < 0 || ctx < 0 || equity < 0 {
		t.Fatalf("expected exactly-once call sites missing: skip=%d ctx=%d equity=%d", skip, ctx, equity)
	}
	if strings.Count(s, "at.skipWhileOpen()") != 1 {
		t.Fatal("runCycle must gate on skipWhileOpen exactly once — a second call site can reintroduce the pre-snapshot return")
	}
	if ctx > skip {
		t.Fatal("IN-POSITION CONTRACT violated: buildTradingContext must run BEFORE the skip-while-open branch (snapshot before AI-skip)")
	}
	if equity > skip {
		t.Fatal("IN-POSITION CONTRACT violated: saveEquitySnapshot must run BEFORE the skip-while-open branch (the equity curve froze in-position for 3 days)")
	}

	// The feed watch must NOT live in runCycle: while holding, the old in-cycle
	// call sat below the skip return, and a dead feed stops bar-close cadence —
	// both made it unreachable exactly when it mattered. It beats on the 60s
	// monitor ticker instead (auto_trader_risk.go monitorTick).
	if strings.Contains(s, "at.checkFeedDown(") {
		t.Fatal("checkFeedDown must live on the wall-clock monitor tick, not inside runCycle")
	}
	risk, err := os.ReadFile("auto_trader_risk.go")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(risk), "at.checkFeedDown(") {
		t.Fatal("monitorTick (auto_trader_risk.go) must carry the feed watch")
	}
}
