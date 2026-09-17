package trader

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
)

// ── PLANNER TAPE IS NT8-ONLY (CTO ruling under the owner's delegation, 2026-09-16) ──

// THE BOOT LINE NAMES EVERY MOVED VALUE (class 82). Excluding 426 imported
// rows from a 12,000-bar tape moves the regime baseline and the levels; the
// line prints the tape length and the baseline BOTH ways, measured through the
// same estimator, so the change is visible and dated — never a literal.
func TestPlannerTapeBootLineNamesTheMovedValues(t *testing.T) {
	now := time.Date(2026, 9, 9, 8, 30, 0, 0, kernel.CTLocation())
	nt8 := rvSessionTape(time.Date(2026, 8, 27, 17, 0, 0, 0, kernel.CTLocation()), 8)
	withImports := rvSessionTape(time.Date(2026, 8, 26, 17, 0, 0, 0, kernel.CTLocation()), 9) // one more session of "import" bars, older
	line := PlannerTapeBootLine("MNQ", "1m", "MNQ 12-26", 426, nt8, withImports, nil, rvBaselineMaxDays, now)
	for _, want := range []string{"NT8-only", "MNQ 12-26", "import rows on contract=426", fmt.Sprintf("tape NT8-only=%d rows", len(nt8)), fmt.Sprintf("with imports=%d rows", len(withImports)), fmt.Sprintf("(Δ%+d)", len(nt8)-len(withImports)), "over 8 day(s)", "over 9 day(s)", "baseline NT8-only=", "with imports=", "chart keeps imports"} {
		if !strings.Contains(line, want) {
			t.Fatalf("boot line lacks %q: %s", want, line)
		}
	}
	if strings.Contains(line, "=0.000000") {
		t.Fatalf("a baseline printed as 0 — UNKNOWN is the word for uncomputed (A24): %s", line)
	}
	// nothing to compute → UNKNOWN, never 0
	empty := PlannerTapeBootLine("MNQ", "1m", "MNQ 12-26", 0, nil, nil, nil, rvBaselineMaxDays, now)
	if !strings.Contains(empty, "UNKNOWN") {
		t.Fatalf("an empty tape must say UNKNOWN: %s", empty)
	}
}

// NO PLANNER DOOR READS THE SHARED READER. Text tripwire (class-113 caveat:
// it matches text, not the call graph; the proof is the A29 grep in the
// report). The planner's files may call only the NT8-only readers; in
// bars_store_depth.go the shared reader is allowed ONLY inside the chart's
// display seam, which starts at `func BarsWithStoreDepthDisplay`. The one
// measurement site is planner_tape_nt8_only.go (logPlannerTapeAccounting),
// whose shared read feeds a boot line and never a planner.
func TestNoPlannerDoorReadsTheSharedBarReader(t *testing.T) {
	shared := regexp.MustCompile(`\.(LastNBarsOn|BarsBetweenOn)\(`)
	for _, f := range []string{"auto_trader_weekly.go", "auto_trader_planner.go", "auto_trader_dayplan.go", "regime_input_window.go"} {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		if loc := shared.FindIndex(b); loc != nil {
			t.Fatalf("%s reads the shared reader at byte %d — planner doors are NT8-only (LastNBarsFromNT8On / BarsBetweenFromNT8On)", f, loc[0])
		}
	}
	b, err := os.ReadFile("bars_store_depth.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	// the two planner doors in this file, checked by FUNCTION BODY (a
	// position cut was wrong: storeBarReader sits after the display seam)
	for _, door := range []string{"func (at *AutoTrader) storeBarReader(", "func BarsWithStoreDepth("} {
		start := strings.Index(src, door)
		if start < 0 {
			t.Fatalf("planner door %q not found", door)
		}
		end := strings.Index(src[start:], "\n}\n")
		if end < 0 {
			t.Fatalf("planner door %q has no end", door)
		}
		body := src[start : start+end]
		if loc := shared.FindStringIndex(body); loc != nil {
			t.Fatalf("%s reads the shared reader — planner doors are NT8-only (LastNBarsFromNT8On)", door)
		}
		if !strings.Contains(body, "LastNBarsFromNT8On") {
			t.Fatalf("%s does not name the NT8-only reader", door)
		}
	}
}

// E7-STYLE GOLDEN: the RULE did not move. Hand the accounting the SAME tape
// both ways and the baselines are identical and Δ is exactly zero — the only
// thing this wave changes is the input, never the estimator.
func TestPlannerTapeAccountingIsZeroWhenTheInputDoesNotChange(t *testing.T) {
	now := time.Date(2026, 9, 9, 8, 30, 0, 0, kernel.CTLocation())
	tape := rvSessionTape(time.Date(2026, 8, 26, 17, 0, 0, 0, kernel.CTLocation()), 9)
	line := PlannerTapeBootLine("MNQ", "1m", "MNQ 12-26", 0, tape, tape, nil, rvBaselineMaxDays, now)
	for _, want := range []string{"import rows on contract=0", "(Δ+0)", "(Δ+0.000000)", "over 9 day(s)"} {
		if !strings.Contains(line, want) {
			t.Fatalf("same input both ways must print %q: %s", want, line)
		}
	}
	if strings.Contains(line, "UNKNOWN") {
		t.Fatalf("a 9-session tape must compute a baseline: %s", line)
	}
}
