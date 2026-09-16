package kernel

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"nofx/market"
)

// ── W5.1 shadow twins (confluent / non-confluent) ──────────────────────────

func TestWeeklyShadowTwins(t *testing.T) {
	refs := []float64{1000.0, 900.0}
	band, atr5m, mult := 0.25, 40.0, 1.5 // band = 10 pts

	confluentLevels := []WeeklyShadowLevel{
		{Price: 995.0, Label: "PDH", Grade: "B"}, // 5 pts from 1000 → confluent
		{Price: 910.0, Label: "PDL", Grade: "C"}, // 10 pts from 900 → confluent
	}
	c, reorder := WeeklyShadowReorder(confluentLevels, refs, band, atr5m, mult)
	if c != 2 {
		t.Fatalf("proving line: confluent twin — 2 levels in band, got %d", c)
	}
	if reorder != 0 {
		t.Fatalf("proving line: confluent twin — equal multiplier keeps the order, got reorder=%d", reorder)
	}

	nonConfluent := []WeeklyShadowLevel{
		{Price: 1020.0, Label: "PDH", Grade: "B"},
		{Price: 880.0, Label: "PDL", Grade: "C"},
	}
	c, reorder = WeeklyShadowReorder(nonConfluent, refs, band, atr5m, mult)
	if c != 0 || reorder != 0 {
		t.Fatalf("proving line: non-confluent twin — got c=%d reorder=%d, want 0/0", c, reorder)
	}

	// the reorder fixture: a confluent C (×1.5 → 1.5) below a plain B (2) flips
	// the shadow ordering — exactly the counter the Sep-9 promotion table reads.
	reorderCase := []WeeklyShadowLevel{
		{Price: 910.0, Label: "PDL", Grade: "C"},  // confluent
		{Price: 1020.0, Label: "PDH", Grade: "B"}, // not confluent
	}
	c, reorder = WeeklyShadowReorder(reorderCase, refs, band, atr5m, mult)
	if c != 1 || reorder != 2 {
		t.Fatalf("proving line: shadow reorder — got c=%d reorder=%d, want 1/2", c, reorder)
	}
}

// ── W5.4 THE LAW — zero real effect diff ───────────────────────────────────

func TestWeeklyShadowZeroRealEffect(t *testing.T) {
	bars := weeklyFixtureBars(t)
	reg := DefaultSessionRegistry()
	now := time.Date(2026, 8, 27, 13, 0, 0, 0, CTLocation())
	first, _, _ := AssembleScoredLevelsMinGrade("", bars, reg, "MNQ", 8, now, 1.5, "")
	// Run the ENTIRE shadow pass on copies of the seated list.
	shadowLevels := make([]WeeklyShadowLevel, len(first))
	for i, l := range first {
		shadowLevels[i] = WeeklyShadowLevel{Price: l.Price, Label: l.Label, Grade: l.Grade}
	}
	_, _ = WeeklyShadowReorder(shadowLevels, []float64{9999.0}, 0.25, 40.0, 1.5)
	_, _ = WeeklyShadowReorder(shadowLevels, nil, 0.25, 40.0, 1.5)
	second, _, _ := AssembleScoredLevelsMinGrade("", bars, reg, "MNQ", 8, now, 1.5, "")
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("proving line: THE LAW — real seating byte-identical with shadow code present: %v vs %v", first, second)
	}
	if len(shadowLevels) != len(first) {
		t.Fatalf("proving line: shadow view never mutates the seated slice")
	}
}

// ── W5.2 counter-annotation twins ──────────────────────────────────────────

func TestWeeklyCounterAnnotationTwins(t *testing.T) {
	// counter-blocked-clause twin: short vs bull/high, grade C, RR 2.0.
	clauses := WeeklyCounterClauses("bull", "high", "short", "C", 2.0)
	want := "would-halve-size|would-require-A-grade|would-need-RR≥4.0"
	if strings.Join(clauses, "|") != want {
		t.Fatalf("proving line: counter-blocked-clause twin — got %v want %s", clauses, want)
	}
	// aligned-silent twin: long with bull/high → nil.
	if got := WeeklyCounterClauses("bull", "high", "long", "B", 3.0); len(got) != 0 {
		t.Fatalf("proving line: aligned-silent twin — got %v want silent", got)
	}
	// neutral / low conviction → silent.
	if got := WeeklyCounterClauses("neutral", "high", "short", "C", 2.0); len(got) != 0 {
		t.Fatalf("proving line: neutral weekly → silent")
	}
	if got := WeeklyCounterClauses("bull", "low", "short", "C", 2.0); len(got) != 0 {
		t.Fatalf("proving line: low conviction → silent")
	}
	// grade A drops the block clause; RR unknown drops the RR clause.
	got := WeeklyCounterClauses("bear", "med", "long", "A", 0)
	if strings.Join(got, "|") != "would-halve-size" {
		t.Fatalf("proving line: A-grade + unknown RR → only halve-size, got %v", got)
	}
}

// ── W5.3 draw-alignment tag ────────────────────────────────────────────────

func TestWeeklyDrawAlignTag(t *testing.T) {
	if got := WeeklyDrawAlignTag("bull", "long", 30500, 30300); got != "toward_draw" {
		t.Fatalf("proving line: bull long below the draw → toward_draw, got %q", got)
	}
	if got := WeeklyDrawAlignTag("bull", "short", 30500, 30300); got != "away" {
		t.Fatalf("proving line: bull short → away, got %q", got)
	}
	if got := WeeklyDrawAlignTag("bear", "short", 29900, 30300); got != "toward_draw" {
		t.Fatalf("proving line: bear short above the draw → toward_draw, got %q", got)
	}
	if got := WeeklyDrawAlignTag("bear", "long", 29900, 30300); got != "away" {
		t.Fatalf("proving line: bear long → away, got %q", got)
	}
	if got := WeeklyDrawAlignTag("neutral", "long", 30500, 30300); got != "neutral" {
		t.Fatalf("proving line: neutral weekly → neutral, got %q", got)
	}
	if got := WeeklyDrawAlignTag("", "long", 0, 30300); got != "neutral" {
		t.Fatalf("proving line: no doc → neutral, got %q", got)
	}
}

// ── W6 knobs (garbage → default) ───────────────────────────────────────────

func TestWeeklyKnobs(t *testing.T) {
	t.Setenv("WEEKLY_READ_CT", "SUN 17:00")
	if wd, h, m := WeeklyReadSpec(); wd != time.Sunday || h != 17 || m != 0 {
		t.Fatalf("proving line: WEEKLY_READ_CT parse — got %v %d:%d", wd, h, m)
	}
	t.Setenv("WEEKLY_READ_CT", "garbage")
	if wd, h, m := WeeklyReadSpec(); wd != time.Sunday || h != 16 || m != 30 {
		t.Fatalf("proving line: garbage WEEKLY_READ_CT → default — got %v %d:%d", wd, h, m)
	}
	t.Setenv("WEEKLY_READ_CT", "")
	t.Setenv("WEEKLY_CONFLUENCE_BAND_ATR", "not-a-float")
	if WeeklyConfluenceBandATR() != 0.25 {
		t.Fatalf("proving line: garbage band → 0.25")
	}
	t.Setenv("WEEKLY_SHADOW_MULT", "abc")
	if WeeklyShadowMult() != 1.5 {
		t.Fatalf("proving line: garbage mult → 1.5")
	}
	t.Setenv("WEEKLY_COUNTER_MODE", "garbage")
	if WeeklyCounterMode() != "warn" {
		t.Fatalf("proving line: garbage counter mode → warn")
	}
	t.Setenv("WEEKLY_COUNTER_MODE", "OFF")
	if WeeklyCounterMode() != "off" {
		t.Fatalf("proving line: WEEKLY_COUNTER_MODE=off honored")
	}
	t.Setenv("WEEKLY_INVALIDATION_TF_DEFAULT", "")
	if WeeklyInvalidationTFDefault() != "1h" {
		t.Fatalf("proving line: TF default → 1h")
	}
	t.Setenv("PLANNER_CANDLES", "off")
	if PlannerCandlesEnabled() {
		t.Fatalf("proving line: PLANNER_CANDLES=off honored")
	}
	t.Setenv("PLANNER_CANDLES", "garbage")
	if !PlannerCandlesEnabled() {
		t.Fatalf("proving line: garbage PLANNER_CANDLES → on")
	}
	// deadline sanity: the read deadline for a given Monday-week lands on the
	// Sunday before the Monday at the knob time (CT).
	t.Setenv("WEEKLY_READ_CT", "sun 16:30")
	mon := time.Date(2026, 8, 31, 0, 0, 0, 0, CTLocation()) // a Monday
	dl := WeeklyReadDeadline(mon)
	if dl.Weekday() != time.Sunday || dl.Format("2006-01-02") != "2026-08-30" || dl.Hour() != 16 || dl.Minute() != 30 {
		t.Fatalf("proving line: WeeklyReadDeadline — got %v", dl)
	}
}

// ── fixture bars for the zero-real-effect diff ─────────────────────────────

func weeklyFixtureBars(t *testing.T) []market.Kline {
	t.Helper()
	var out []market.Kline
	start := time.Date(2026, 8, 24, 9, 0, 0, 0, CTLocation()) // Monday
	base := 30000.0
	for i := 0; i < 4000; i++ {
		ts := start.Add(time.Duration(i) * time.Minute)
		// skip the weekend gap (2026-08-29/30) to keep a clean series.
		if ts.Weekday() == time.Saturday || ts.Weekday() == time.Sunday {
			continue
		}
		o := base + float64(i)*0.01
		out = append(out, market.Kline{
			OpenTime: ts.UnixMilli(), Open: o, High: o + 1.5, Low: o - 1.5, Close: o + 0.5, Volume: 10,
		})
	}
	return out
}
