package kernel

import (
	"os"
	"strings"
	"testing"
	"time"
)

// P7 (ledger-close 2026-08-19) — ONE SNAPSHOT INSTANT. The cycle used to carry
// four clocks; plan/level math read the bar cache ~2 min after the market
// block in the SAME prompt (U4). These guards pin the fix the same way the
// TZ-guard pins layouts: on the source, so a refactor can't silently
// reintroduce a second capture point.

func TestSingleSnapshotInstantContract(t *testing.T) {
	src, err := os.ReadFile("engine_analysis.go")
	if err != nil {
		t.Fatal(err)
	}
	s := string(src)

	if n := strings.Count(s, "snapshotNow := time.Now()"); n != 1 {
		t.Fatalf("exactly ONE snapshot instant may be captured per cycle, found %d", n)
	}
	capture := strings.Index(s, "snapshotNow := time.Now()")
	stamp := strings.Index(s, "ctx.SnapshotMs = snapshotNow.UnixMilli()")
	fetch := strings.Index(s, "fetchMarketDataWithStrategy(ctx, engine)")
	barsRead := strings.Index(s, "snapshotBars = market.FuturesBarsProvider(activeSymbol")
	clockLine := strings.Index(s, `" · Snapshot: " + ClockCTSeconds(snapshotNow)`)

	if stamp < 0 {
		t.Fatal("ctx.SnapshotMs must be re-stamped from snapshotNow (B4 evaluates at data-assembly time)")
	}
	if clockLine < 0 {
		t.Fatal("the prompt must self-document its instant: Snapshot: HH:MM:SS CT via ClockCTSeconds")
	}
	if !(capture < stamp && stamp < fetch) {
		t.Fatalf("order violated: capture(%d) < SnapshotMs stamp(%d) < market fetch(%d) required", capture, stamp, fetch)
	}
	if !(fetch < barsRead && barsRead < clockLine) {
		t.Fatalf("the 1m snapshot window must be read back-to-back with the fetch (fetch=%d barsRead=%d clock=%d)", fetch, barsRead, clockLine)
	}
}

func TestClockCTSecondsFormat(t *testing.T) {
	// 2026-08-19 14:30:07 UTC = 09:30:07 CT (CDT, UTC-5).
	instant := time.Date(2026, 8, 19, 14, 30, 7, 0, time.UTC)
	if got := ClockCTSeconds(instant); got != "09:30:07 CT" {
		t.Fatalf("ClockCTSeconds = %q, want 09:30:07 CT", got)
	}
}
