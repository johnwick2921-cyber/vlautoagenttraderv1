package ninjatrader

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/logger"
	ntwire "nofx/provider/ninjatrader"
)

// ── BARS HORIZON pins (2026-09-09) ───────────────────────────────────────────
//
// These drive the PRODUCTION function barsFromCache, not a copy of it (class
// 86). wireFuturesBarsProvider's closure is a one-line delegate to it.
//
// A28: every clock here is STATED. Nothing calls time.Now().

// bhWarnCapture attaches the repo's WARN+ sink and returns a getter. The sink is
// process-global (logger/db_sink.go:40-45, newest wins), so every log-capture
// pin MUST restore a no-op on cleanup and must not run t.Parallel().
func bhWarnCapture(t *testing.T) func() []string {
	t.Helper()
	var mu sync.Mutex
	var lines []string
	logger.AttachDBSink(func(_ int64, level, _, _, message, _ string) {
		mu.Lock()
		lines = append(lines, level+" "+message)
		mu.Unlock()
	})
	t.Cleanup(func() { logger.AttachDBSink(func(int64, string, string, string, string, string) {}) })
	return func() []string { mu.Lock(); defer mu.Unlock(); return append([]string(nil), lines...) }
}

// bhSeed puts n contiguous 1m bars into the cache, oldest first. Each bar
// carries distinct OHLC and non-zero volume so the cache's placeholder filter
// (NO SYNTHETIC BARS, 2026-08-17) keeps all of them.
func bhSeed(c *ntwire.BarCache, symbol, tf string, startCT time.Time, n int) {
	step := int64(60_000)
	bars := make([]ntwire.Bar, 0, n)
	for i := 0; i < n; i++ {
		p := 29000 + float64(i)*0.25
		bars = append(bars, ntwire.Bar{T: startCT.UnixMilli() + int64(i)*step, O: p, H: p + 1, L: p - 1, C: p + 0.5, V: 10})
	}
	c.SeedHistorical(symbol, tf, bars)
}

func bhCT(y int, mo time.Month, d, h, mi int) time.Time {
	return time.Date(y, mo, d, h, mi, 0, 0, kernel.CTLocation())
}

func bhCountLines(lines []string, subs ...string) int {
	n := 0
	for _, l := range lines {
		ok := true
		for _, s := range subs {
			if !strings.Contains(l, s) {
				ok = false
				break
			}
		}
		if ok {
			n++
		}
	}
	return n
}

// PIN 1 — a read served fewer bars than it asked for WARNS, and names the
// consumer. Covers the two real 12000-bar sites (trader/auto_trader_planner.go
// and trader/auto_trader_weekly.go) against a 2500-bar ring
// (provider/ninjatrader/bar_cache.go:24 DefaultBarCacheMaxBars): 79% short on
// every call since they were written, silently, forever.
func TestBridgeWarnsWhenServedIsShortOfRequested(t *testing.T) {
	resetBarHorizonWarnsForSessionDay(0)
	get := bhWarnCapture(t)
	cache := ntwire.NewBarCache(2500)
	bhSeed(cache, "MNQ", "1m", bhCT(2026, time.September, 8, 9, 0), 2500)
	now := bhCT(2026, time.September, 9, 13, 18)

	out := barsFromCache(cache, "MNQ", "1m", 12000, now)

	if len(out) != 2500 {
		t.Fatalf("returned %d klines, want 2500 — the warn must change nothing that is RETURNED (A10)", len(out))
	}
	lines := get()
	if n := bhCountLines(lines, "bar horizon"); n != 1 {
		t.Fatalf("want 1 bar-horizon warn, got %d captured lines (asked=12000 served=2500): %v", n, lines)
	}
	for _, want := range []string{"asked=12000", "served=2500", "ring=2500", "SHORT", "MNQ", "caller="} {
		if bhCountLines(lines, want) != 1 {
			t.Fatalf("warn line missing %q: %v", want, lines)
		}
	}
	// The caller must be THIS test's file, not the bridge's own closure.
	if bhCountLines(lines, "caller=ninjatrader/bar_horizon_bridge_test.go:") != 1 {
		t.Fatalf("warn line does not name this test as the caller: %v", lines)
	}
}

// PIN 1b — the OTHER direction, per the acceptance: a satisfied read is SILENT.
func TestBridgeSilentWhenReadIsSatisfied(t *testing.T) {
	resetBarHorizonWarnsForSessionDay(0)
	get := bhWarnCapture(t)
	cache := ntwire.NewBarCache(2500)
	bhSeed(cache, "MNQ", "1m", bhCT(2026, time.September, 8, 9, 0), 2500)
	now := bhCT(2026, time.September, 9, 13, 18)

	out := barsFromCache(cache, "MNQ", "1m", 2000, now)

	if len(out) != 2000 {
		t.Fatalf("returned %d klines, want 2000 (tail truncation unchanged)", len(out))
	}
	// The TAIL, not the head: truncation keeps the NEWEST bars. Pinned because
	// the observation must not be able to change what is returned (A10).
	cached := cache.Get("MNQ", "1m")
	if out[len(out)-1].OpenTime != cached[len(cached)-1].T {
		t.Fatalf("newest returned bar %d, want the cache's newest %d — truncation must keep the TAIL",
			out[len(out)-1].OpenTime, cached[len(cached)-1].T)
	}
	if out[0].OpenTime != cached[len(cached)-2000].T {
		t.Fatalf("oldest returned bar %d, want %d", out[0].OpenTime, cached[len(cached)-2000].T)
	}
	if n := bhCountLines(get(), "bar horizon"); n != 0 {
		t.Fatalf("a satisfied, contiguous read emitted %d bar-horizon warns, want 0: %v", n, get())
	}
}

// PIN 1c — a HOLED read that is NOT short still warns. This is the 2026-09-09
// 13:18:13 CT case: 2000 of 2000 served, 696 open-market minutes missing inside
// them. A served-count check alone is silent here.
func TestBridgeWarnsOnAHoledWindowThatIsNotShort(t *testing.T) {
	resetBarHorizonWarnsForSessionDay(0)
	get := bhWarnCapture(t)
	cache := ntwire.NewBarCache(2500)
	// Two contiguous runs with an 80-minute open-market hole between them.
	bhSeed(cache, "MNQ", "1m", bhCT(2026, time.September, 9, 8, 30), 60)
	bhSeed(cache, "MNQ", "1m", bhCT(2026, time.September, 9, 10, 50), 60)
	now := bhCT(2026, time.September, 9, 12, 0)

	out := barsFromCache(cache, "MNQ", "1m", 120, now)

	if len(out) != 120 {
		t.Fatalf("returned %d klines, want 120", len(out))
	}
	lines := get()
	if bhCountLines(lines, "bar horizon", "HOLED", "gaps=80") != 1 {
		t.Fatalf("want one HOLED warn with gaps=80 (served==asked==120), got: %v", lines)
	}
	if bhCountLines(lines, "SHORT") != 0 {
		t.Fatalf("SHORT fired on a fully-served read: %v", lines)
	}
}

// PIN 6 — every warn and boot-line field is READ from the object that enforces
// it, never a literal (A11).
func TestWarnFieldsAreResolvedNotLiteral(t *testing.T) {
	resetBarHorizonWarnsForSessionDay(0)
	get := bhWarnCapture(t)
	cache := ntwire.NewBarCache(1234) // deliberately NOT DefaultBarCacheMaxBars
	bhSeed(cache, "MNQ", "1m", bhCT(2026, time.September, 9, 9, 0), 100)
	now := bhCT(2026, time.September, 9, 10, 39)

	barsFromCache(cache, "MNQ", "1m", 500, now)

	lines := get()
	for _, want := range []string{"ring=1234", "asked=500", "served=100"} {
		if bhCountLines(lines, want) != 1 {
			t.Fatalf("warn line missing %q: %v", want, lines)
		}
	}
	if bhCountLines(lines, "2500") != 0 {
		t.Fatalf("warn line contains 2500 (the DEFAULT ring), want the cache's own 1234 (A11): %v", lines)
	}

	boot := barHorizonBootLine(cache, now)
	if !strings.Contains(boot, "ring=1234") || !strings.Contains(boot, "MNQ 1m=100") {
		t.Fatalf("boot line %q must report ring=1234 and MNQ 1m=100, both READ from the cache (A11)", boot)
	}
	if strings.Contains(boot, "2500") || strings.Contains(boot, "=2000") {
		t.Fatalf("boot line %q contains a literal depth", boot)
	}
	_ = fmt.Sprint()
}
