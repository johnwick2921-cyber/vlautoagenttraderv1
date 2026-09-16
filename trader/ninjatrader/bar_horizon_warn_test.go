package ninjatrader

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/telemetry"
)

// PIN 5 — the warn is DEDUPED and every detection is COUNTED.
//
// Why a dedupe at all, by measurement: the planner cadence on 2026-09-09 was
// 32m/10m/37m between reads (planner_read_facts ids 64->65->66->67), the monitor
// tick is 60s, and 57 consumer call sites hit this bridge per cycle. The
// 2026-09-09 log file is 511 MB. An undeduped warn writes thousands of lines.
//
// Why the counter is OUTSIDE the dedupe: counters record, never infer (class
// 35). A dedupe that hides 699 of 700 detections must not also hide the rate.
func TestShortReadWarnIsDedupedAndCounted(t *testing.T) {
	resetBarHorizonWarnsForSessionDay(0)
	get := bhWarnCapture(t)
	cache := ntwire.NewBarCache(2500)
	bhSeed(cache, "MNQ", "1m", bhCT(2026, time.September, 8, 9, 0), 2500)

	// ONE call site for every read below: resolveBarHorizonCaller names the
	// frame, and the frame is part of the dedupe key — two different lines in
	// this test would be two different keys and nothing would ever suppress.
	read := func(at time.Time) { barsFromCache(cache, "MNQ", "1m", 12000, at) }

	base := telemetry.BarHorizonCounts()["short"]
	// The live arm-133 instant, to the second, so the "since" below is a
	// RESOLVED clock rather than a literal.
	start := bhCT(2026, time.September, 9, 13, 18).Add(13 * time.Second)
	for i := 0; i < 700; i++ {
		read(start.Add(time.Duration(i) * time.Second))
	}
	if n := bhCountLines(get(), "bar horizon"); n != 1 {
		t.Fatalf("%d emitted lines inside the 15-minute window, want 1", n)
	}
	if got := telemetry.BarHorizonCounts()["short"] - base; got != 700 {
		t.Fatalf("BarHorizonCounts()[\"short\"] rose by %d, want 700 — every DETECTION is counted, not only the emitted one", got)
	}

	// Past the re-arm window the key speaks again, and it says how many it ate.
	read(start.Add(16 * time.Minute))
	lines := get()
	if n := bhCountLines(lines, "bar horizon"); n != 2 {
		t.Fatalf("after the window re-armed: %d emitted lines, want 2: %v", n, lines)
	}
	if bhCountLines(lines, "suppressed=699 since ") != 1 {
		t.Fatalf("the re-armed line must name what it suppressed and since when: %v", lines)
	}
	if bhCountLines(lines, "suppressed=699 since 13:18:13 CT") != 1 {
		t.Fatalf("the 'since' must be a RESOLVED clock, not a literal or a duration: %v", lines)
	}

	// Build up a fresh suppression run, then cross into the next CME session
	// day. The rollover must CLEAR the state, so the next line carries no
	// "suppressed=" at all — without the rollover the window path would emit
	// the same line WITH suppressed=5, which is what this discriminates.
	for i := 1; i <= 5; i++ {
		read(start.Add(16*time.Minute + time.Duration(i)*time.Second))
	}
	read(start.Add(24 * time.Hour))
	lines = get()
	if n := bhCountLines(lines, "bar horizon"); n != 3 {
		t.Fatalf("after the session-day rollover: %d emitted lines, want 3: %v", n, lines)
	}
	if bhCountLines(lines, "suppressed=5") != 0 {
		t.Fatalf("the CME session-day rollover did not clear the suppression state: %v", lines)
	}
}

// PIN 5b — the EMPTY arm is GRACED across the boot window.
//
// Measured on 2026-09-09: PID 438 started 13:02:25 CT and the first non-zero
// backfill logged at 13:06:48 CT — a ~4.5 minute window in which every one of
// the 57 consumer sites is served 0 bars. Without a grace that is ~40 EMPTY
// warns at every boot. The grace lifts on the backfill OR on a timer, so a TCP
// server that never comes up still gets one loud line (pin the FALLBACK, not
// only the hook).
func TestEmptyArmIsGracedUntilBackfillOrTimeout(t *testing.T) {
	resetBarHorizonWarnsForSessionDay(0)
	get := bhWarnCapture(t)
	cache := ntwire.NewBarCache(2500) // seeded with NOTHING
	boot := bhCT(2026, time.September, 9, 13, 2)
	armBarHorizonGrace(boot)

	barsFromCache(cache, "MNQ", "1m", 2000, boot.Add(1*time.Minute))
	if n := bhCountLines(get(), "bar horizon"); n != 0 {
		t.Fatalf("EMPTY warned %d times inside the boot grace, want 0: %v", n, get())
	}

	// The fallback timer, with the hook never firing.
	barsFromCache(cache, "MNQ", "1m", 2000, boot.Add(barHorizonBootGrace+time.Minute))
	lines := get()
	if bhCountLines(lines, "bar horizon", "EMPTY") != 1 {
		t.Fatalf("after the grace fallback: want 1 EMPTY warn, got %v", lines)
	}
	if bhCountLines(lines, "served=0") != 1 || bhCountLines(lines, "gaps=UNKNOWN") != 1 {
		t.Fatalf("the EMPTY line must say served=0 and gaps=UNKNOWN (never a plausible zero, A24): %v", lines)
	}

	// And the hook path: a fresh grace, lifted by the backfill landing.
	resetBarHorizonWarnsForSessionDay(0)
	get2 := bhWarnCapture(t)
	armBarHorizonGrace(boot)
	noteBarHorizonBackfillLanded()
	barsFromCache(cache, "MNQ", "1m", 2000, boot.Add(1*time.Minute))
	if bhCountLines(get2(), "bar horizon", "EMPTY") != 1 {
		t.Fatalf("after the backfill landed the EMPTY arm must speak immediately: %v", get2())
	}
}

// PIN 5c — A29. Every new function has a production call site, named.
func TestBarHorizonHasProductionCallSites(t *testing.T) {
	for fn, wantIn := range map[string]string{
		"barsFromCache(":                     "trader/ninjatrader/bars_market_bridge.go",
		"barHorizonWarn(":                    "trader/ninjatrader/bars_market_bridge.go",
		"kernel.HorizonOf(":                  "trader/ninjatrader/bars_market_bridge.go",
		"armBarHorizonGrace(":                "trader/ninjatrader/bars_market_bridge.go",
		"noteBarHorizonBackfillLanded(":      "trader/ninjatrader/bar_persist_wire.go",
		"barHorizonBootLine(":                "trader/ninjatrader/bar_persist_wire.go",
		"telemetry.IncBarHorizon(":           "trader/ninjatrader/bar_horizon_warn.go",
		"resetBarHorizonWarnsForSessionDay(": "trader/ninjatrader/bar_horizon_warn.go",
		"cache.MaxBars()":                    "trader/ninjatrader/bars_market_bridge.go",
		// Added in review 2026-09-09: the counter had FIVE arms and no reader.
		"telemetry.BarHorizonCounts(": "trader/ninjatrader/bar_horizon_warn.go",
		"barHorizonCountsTxt(":        "trader/ninjatrader/bar_horizon_warn.go",
	} {
		n, where := prodCallSites(t, fn)
		if n == 0 {
			t.Fatalf("%s: 0 production call sites — a new function with no caller is not shipped (A29)", fn)
		}
		if !strings.Contains(strings.Join(where, " "), wantIn) {
			t.Fatalf("%s: production call sites %v, want one in %s", fn, where, wantIn)
		}
	}
}

// prodCallSites counts NON-TEST .go files under the repo root that contain
// `needle`, and returns their repo-relative paths. Precedent for source-scanning
// wiring pins in this repo: kernel/wiring_riskforceflat_pin_test.go,
// trader/wiring_gate_test.go, researchsnapshot/wiring_test.go.
func prodCallSites(t *testing.T, needle string) (int, []string) {
	t.Helper()
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("repo root: %v", err)
	}
	n := 0
	var where []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "web", "docs", ".understand-anything", ".claude":
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		b, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil
		}
		for _, line := range strings.Split(string(b), "\n") {
			if strings.Contains(line, needle) && !strings.HasPrefix(strings.TrimSpace(line), "//") {
				// The DEFINITION is not a call site.
				if strings.HasPrefix(strings.TrimSpace(line), "func ") {
					continue
				}
				rel, _ := filepath.Rel(root, path)
				n++
				where = append(where, rel)
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	return n, where
}

// PIN D3-E — A29. The boot rehydrate is WIRED, and it uses the ring-safe door.
func TestRingRehydrateIsWiredAtBoot(t *testing.T) {
	for fn, wantIn := range map[string]string{
		"rehydrateRingFromStore(": "trader/ninjatrader/bar_persist_wire.go",
		"cache.RehydrateOlder(":   "trader/ninjatrader/bar_persist_wire.go",
		"bh.LastNBars(":           "trader/ninjatrader/bar_persist_wire.go",
	} {
		n, where := prodCallSites(t, fn)
		if n == 0 {
			t.Fatalf("%s: 0 production call sites — a new function with no caller is not shipped (A29)", fn)
		}
		if !strings.Contains(strings.Join(where, " "), wantIn) {
			t.Fatalf("%s: production call sites %v, want one in %s", fn, where, wantIn)
		}
	}
	// The rehydrate must NEVER reach for SeedHistorical: that door seeds a cold
	// key and lets the INCOMING (here: the store) win an overlap, which is
	// exactly what D3 must not do.
	b, err := os.ReadFile("bar_persist_wire.go")
	if err != nil {
		t.Fatalf("read bar_persist_wire.go: %v", err)
	}
	if strings.Contains(string(b), "SeedHistorical(") {
		t.Fatalf("bar_persist_wire.go calls SeedHistorical — the store must enter through RehydrateOlder, which refuses a cold key and never outranks a live bar")
	}
}

// PIN D3-L — OWNER CONDITION (a), 2026-09-09. THE BOOT REHYDRATE TOUCHES 1m
// AND NOTHING ELSE.
//
// THE DEFECT THIS CATCHES: the first cut rehydrated EVERY (symbol, timeframe)
// pair the cache held, which deepened the 5m ring from the store and silently
// moved a LIVE regime input using NT8 aggregates this repo has already judged
// inconsistent with their own 1m constituents. The owner's ruling allowed the
// regime input to change and required the depth to come from the 1m tape.
func TestRehydrateSelectsOnly1mPairs(t *testing.T) {
	in := [][2]string{
		{"MNQ", "1m"}, {"MNQ", "5m"}, {"MNQ", "15m"}, {"MNQ", "1h"},
		{"MNQ", "4h"}, {"MNQ", "1d"}, {"MNQ", "1w"}, {"ES", "1m"}, {"ES", "5m"},
	}
	got := pairsToRehydrate(in)
	if len(got) != 2 {
		t.Fatalf("selected %d pairs from %d, want exactly the 2 that are 1m: %v", len(got), len(in), got)
	}
	for _, p := range got {
		if p[1] != rehydrateTimeframe {
			t.Fatalf("a non-%s pair was selected for rehydration: %v — stored non-1m rows are NT8 aggregates and must never reach a live regime input", rehydrateTimeframe, p)
		}
	}
	if got[0] != ([2]string{"MNQ", "1m"}) || got[1] != ([2]string{"ES", "1m"}) {
		t.Fatalf("the cache's order was not preserved: %v", got)
	}
	// An all-non-1m cache selects nothing, and says nothing was selected —
	// never a silent full pass (A24).
	if n := len(pairsToRehydrate([][2]string{{"MNQ", "5m"}, {"MNQ", "1d"}})); n != 0 {
		t.Fatalf("a cache with no 1m pair selected %d pair(s)", n)
	}
	// A29 — the production loop consults it.
	if n, where := prodCallSites(t, "pairsToRehydrate("); n == 0 {
		t.Fatalf("pairsToRehydrate has 0 production call sites (A29) — the filter can be bypassed with the suite green (%v)", where)
	}
}

// PIN 5d — THE COUNTER IS READABLE ON THE RUNNING BOT.
//
// THE DEFECT THIS CATCHES: telemetry.BarHorizonCounts had zero production
// readers. Five arms were counted on every detection and nothing on the running
// bot could ever print them — a counter that records into a void, which is the
// opposite of "counters record, never infer" (class 35). There is no API
// surface in this wave's footprint, so the totals ride the one line that
// already survives the dedupe.
func TestBarHorizonWarnLineCarriesTheCounters(t *testing.T) {
	txt := barHorizonCountsTxt()
	for _, arm := range []string{"short=", "holed=", "empty=", "graced=", "suppressed="} {
		if !strings.Contains(txt, arm) {
			t.Fatalf("arm %q is counted but never printed: %q", arm, txt)
		}
	}
	before := telemetry.BarHorizonCounts()["short"]
	telemetry.IncBarHorizon("short")
	after := barHorizonCountsTxt()
	if !strings.Contains(after, fmt.Sprintf("short=%d", before+1)) {
		t.Fatalf("the rendered totals do not READ telemetry (A11): before=%d line=%q", before, after)
	}
}
