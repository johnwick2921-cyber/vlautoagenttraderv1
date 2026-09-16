package trader

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// ── D4 PINS (wave BARS HORIZON, 2026-09-09) ─────────────────────────────────
//
// REFUTATION R1, ON THE RECORD. The dispatch that opened this wave held that
// scope_bars records the REQUESTED window. IT DOES NOT and never did:
// persistReadFacts has always written `ScopeBars: len(scope.Bars)` — the
// SERVED count, post-truncation. All 67 live rows (ids 1..67) record
// scope_bars=2000 against a 2000 ask, so the void scope has NEVER been short.
// Any design premised on "scope_bars reports the request" is wrong, and
// renaming its meaning would silently reinterpret 67 correct rows.
//
// WHAT WAS ACTUALLY MISSING is that A COUNT CANNOT EXPRESS A SPAN OR A HOLE.
// planner_read_facts id 66 (2026-09-09 13:18:13 CT) and id 64 are identical in
// every recorded field, yet id 66's tape reached back to 09-07 10:23 CT — a
// 3,055-minute span — with 696 OPEN-MARKET minutes missing inside it (the
// machine was off 09-09 01:29→13:04 CT; VERIFIED against a 14:49 CT snapshot of
// the bars table, and verified again through the production
// kernel.OpenIntervalsBetween: 2,696 open intervals in that window, 2,000
// served, 696 gaps).
//
// So D4 adds the three fields a count cannot carry: the REQUEST beside the
// served count, the oldest bar's AGE, and a GAP COUNT.

func d4Bars(n int, firstMs int64) []market.Kline {
	out := make([]market.Kline, 0, n)
	for i := 0; i < n; i++ {
		out = append(out, market.Kline{OpenTime: firstMs + int64(i)*60000, Open: 1, High: 2, Low: 0, Close: 1, Volume: 1})
	}
	return out
}

// PIN D4-A — THE ROW CARRIES SPAN, AGE AND GAPS, AND scope_bars STAYS SERVED.
func TestReadFactRowCarriesTheHorizon(t *testing.T) {
	now := time.Date(2026, 9, 9, 13, 18, 13, 0, kernel.CTLocation())
	// The live shape of id 66: 2000 served of 2000 asked, oldest 09-07 10:23 CT.
	oldest := time.Date(2026, 9, 7, 10, 23, 0, 0, kernel.CTLocation())
	bars := d4Bars(2000, oldest.UnixMilli())
	scope := kernel.VoidScope{
		Bars: bars, SinceMs: 1788904800000, Interval: "1m", BarCount: 2000, Source: "provider",
		Horizon: kernel.HorizonOf(bars, "1m", 2000, now),
	}
	scope.Horizon.Who = "void"

	row := buildReadFactRow("hoang", kernel.PlannerInput{Session: "NY"}, scope, nil, 19.7631429163709, now)

	if row.ScopeBars != 2000 {
		t.Fatalf("scope_bars=%d, want 2000 — it has ALWAYS been len(scope.Bars), the SERVED count (R1)", row.ScopeBars)
	}
	if row.ScopeRequestedBars != 2000 {
		t.Fatalf("scope_requested_bars=%d, want 2000 — the REQUEST is a new column, never a redefinition of scope_bars", row.ScopeRequestedBars)
	}
	if row.ScopeSpanMs != 1999*60000 {
		t.Fatalf("scope_span_ms=%d, want %d — a COUNT cannot express a SPAN", row.ScopeSpanMs, 1999*60000)
	}
	wantAge := now.UnixMilli() - oldest.UnixMilli()
	if row.ScopeOldestAgeMs != wantAge {
		t.Fatalf("scope_oldest_age_ms=%d, want %d", row.ScopeOldestAgeMs, wantAge)
	}
	if !row.HorizonRecorded() {
		t.Fatalf("HorizonRecorded()=false on a computed row — its zeros would read as UNKNOWN")
	}
	var hs []kernel.BarHorizon
	if err := json.Unmarshal([]byte(row.ReadHorizons), &hs); err != nil {
		t.Fatalf("read_horizons is not a JSON array: %v (%q)", err, row.ReadHorizons)
	}
	if len(hs) != 1 || hs[0].Who != "void" || hs[0].Interval != "1m" {
		t.Fatalf("read_horizons = %+v, want one entry who=void intv=1m", hs)
	}
}

// PIN D4-B — TWO READS THAT DIFFER ONLY IN CONTINUITY MUST NOT LOOK IDENTICAL.
//
// This is the whole defect: id 64 (a contiguous tape) and id 66 (a tape with a
// 696-minute open-market hole) both recorded scope_bars=2000 and were
// indistinguishable on the record.
func TestAHolyTapeAndAContiguousTapeDifferOnTheRecord(t *testing.T) {
	now := time.Date(2026, 9, 9, 13, 18, 13, 0, kernel.CTLocation())
	mk := func(bars []market.Kline) *store.PlannerReadFact {
		sc := kernel.VoidScope{Bars: bars, Interval: "1m", BarCount: 2000, Source: "provider",
			Horizon: kernel.HorizonOf(bars, "1m", 2000, now)}
		sc.Horizon.Who = "void"
		return buildReadFactRow("hoang", kernel.PlannerInput{Session: "NY"}, sc, nil, 19.76, now)
	}
	// Contiguous: 2000 consecutive open minutes ending at the read.
	contig := d4Bars(2000, now.Add(-2000*time.Minute).Truncate(time.Minute).UnixMilli())
	// Holed: the live id-66 window — 2000 bars spread over 3,056 intervals.
	holed := append(d4Bars(1000, time.Date(2026, 9, 7, 10, 23, 0, 0, kernel.CTLocation()).UnixMilli()),
		d4Bars(1000, time.Date(2026, 9, 9, 0, 0, 0, 0, kernel.CTLocation()).UnixMilli())...)

	a, b := mk(contig), mk(holed)
	if a.ScopeBars != b.ScopeBars {
		t.Fatalf("fixture broken: both rows must record the SAME served count (%d vs %d)", a.ScopeBars, b.ScopeBars)
	}
	if a.ScopeSpanMs == b.ScopeSpanMs && a.ScopeGapCount == b.ScopeGapCount {
		t.Fatalf("a contiguous tape and a holed tape are STILL indistinguishable on the record (span %d gaps %d both)",
			a.ScopeSpanMs, a.ScopeGapCount)
	}
	if b.ScopeGapCount <= 0 {
		t.Fatalf("the holed tape records gaps=%d — a hole must be counted, not implied", b.ScopeGapCount)
	}
	if a.ScopeGapCount != 0 {
		t.Fatalf("the contiguous tape records gaps=%d, want a COMPUTED 0", a.ScopeGapCount)
	}
}

// PIN D4-C — A29. The extracted row builder is the PRODUCTION path, not a copy.
func TestReadFactRowBuilderIsWired(t *testing.T) {
	for fn, wantIn := range map[string]string{
		"buildReadFactRow(":      "trader/auto_trader_planner.go",
		"scope.Horizon":          "trader/auto_trader_planner.go",
		"sc.Horizon = HorizonOf": "kernel/void_scope.go",
	} {
		n, where := d2ProdCallSites(t, fn)
		if n == 0 {
			t.Errorf("%s: 0 production call sites (A29)", fn)
			continue
		}
		if !strings.Contains(strings.Join(where, " "), wantIn) {
			t.Errorf("%s: production call sites %v, want one in %s", fn, where, wantIn)
		}
	}
}

// PIN D4-D — SERVED AND REQUESTED MUST BE DISTINGUISHABLE ON THE ROW.
//
// Found by MUTATION, not design: every live row and the first fixture both had
// served == requested == 2000, so swapping `len(scope.Bars)` for
// `scope.BarCount` — shipping the very premise R1 refuted — passed green. A
// pin whose two numbers are equal cannot tell them apart (class 89).
func TestServedAndRequestedAreSeparateColumns(t *testing.T) {
	now := time.Date(2026, 9, 9, 13, 18, 13, 0, kernel.CTLocation())
	bars := d4Bars(1200, now.Add(-1200*time.Minute).Truncate(time.Minute).UnixMilli()) // SHORT: 1200 of 2000
	scope := kernel.VoidScope{
		Bars: bars, Interval: "1m", BarCount: 2000, Source: "provider",
		Horizon: kernel.HorizonOf(bars, "1m", 2000, now),
	}
	scope.Horizon.Who = "void"
	row := buildReadFactRow("hoang", kernel.PlannerInput{Session: "NY"}, scope, nil, 19.76, now)

	if row.ScopeBars != 1200 {
		t.Fatalf("scope_bars=%d, want the SERVED 1200 — it is len(scope.Bars) and always was (R1)", row.ScopeBars)
	}
	if row.ScopeRequestedBars != 2000 {
		t.Fatalf("scope_requested_bars=%d, want the REQUESTED 2000", row.ScopeRequestedBars)
	}
	if row.ScopeBars == row.ScopeRequestedBars {
		t.Fatalf("fixture broken: this pin requires served != requested, got %d for both", row.ScopeBars)
	}
}

// PIN D4-E — A SCOPE WITH NO HORIZON WRITES UNKNOWN, NOT ZEROS.
//
// Also found by mutation: removing the `h.Interval == ""` guard let an
// uncomputed horizon write four zeros AND a read_horizons entry, so a row that
// measured nothing would have claimed "span 0, age 0, no gaps" — the exact
// plausible-zero A24 forbids.
func TestScopeWithoutAHorizonWritesUnknown(t *testing.T) {
	now := time.Date(2026, 9, 9, 13, 18, 13, 0, kernel.CTLocation())
	// A pre-wave-shaped scope: bars and interval, but no Horizon computed.
	scope := kernel.VoidScope{
		Bars:     d4Bars(2000, now.Add(-2000*time.Minute).Truncate(time.Minute).UnixMilli()),
		Interval: "1m", BarCount: 2000, Source: "provider",
	}
	row := buildReadFactRow("hoang", kernel.PlannerInput{Session: "NY"}, scope, nil, 19.76, now)

	if row.ScopeBars != 2000 {
		t.Fatalf("scope_bars=%d — the SERVED count is recorded with or without a horizon", row.ScopeBars)
	}
	if row.ReadHorizons != "" {
		t.Fatalf("read_horizons=%q on a scope that computed no horizon — it must stay \"\" so the row reads UNKNOWN", row.ReadHorizons)
	}
	if row.HorizonRecorded() {
		t.Fatalf("HorizonRecorded()=true on an uncomputed row — its scope_gap_count=%d would be read as \"no gaps\"", row.ScopeGapCount)
	}
	if row.ScopeRequestedBars != 0 || row.ScopeSpanMs != 0 || row.ScopeOldestAgeMs != 0 || row.ScopeGapCount != 0 {
		t.Fatalf("an uncomputed horizon wrote values: requested=%d span=%d age=%d gaps=%d — the discriminator is read_horizons, and these must stay at their UNKNOWN zero",
			row.ScopeRequestedBars, row.ScopeSpanMs, row.ScopeOldestAgeMs, row.ScopeGapCount)
	}
}
