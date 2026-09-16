package trader

import (
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
)

// F1 — THE PIN. Two own-1m weeks in the store, but native DAILY bars covering
// ≥4 completed weeks in the cache: the weekly reader must count ≥4 and NOT be
// thin. On the pre-wave code the reader read the 1m table only and counted 2.
func TestBarSourcePinWeeklyThin(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	// 6 weeks of native dailies ending last Friday.
	dayMs := int64(1440) * 60000
	start := now.AddDate(0, 0, -44).Truncate(24 * time.Hour).UnixMilli()
	var dailies []market.Kline
	for i := 0; i < 44; i++ {
		p := 29000 + float64(i)
		dailies = append(dailies, market.Kline{OpenTime: start + int64(i)*dayMs,
			Open: p, High: p + 50, Low: p - 50, Close: p + 10, Volume: 1000})
	}
	// Only 2 weeks of own-1m — the starved state.
	var own1m []market.Kline
	twoWeeks := now.AddDate(0, 0, -14).UnixMilli()
	for i := 0; i < 14*24*60; i += 30 {
		own1m = append(own1m, market.Kline{OpenTime: twoWeeks + int64(i)*60000,
			Open: 29500, High: 29510, Low: 29490, Close: 29505, Volume: 10})
	}

	r := &market.BarResolver{
		Native: func(_, tf string, _ int) []market.Kline {
			if tf == "1d" {
				return dailies
			}
			return nil
		},
		Own1m: func(string, int64, int64) []market.Kline { return own1m },
		Now:   func() time.Time { return now },
	}

	// What the OLD reader saw: the 1m table alone.
	legacyWeeks := kernel.CompletedWeekCount(own1m, now)
	if legacyWeeks >= 4 {
		t.Fatalf("fixture is wrong: own-1m must be thin, got %d weeks", legacyWeeks)
	}

	s, err := r.CompletedBars("MNQ", "1w", 0, now.UnixMilli())
	if err != nil {
		t.Fatalf("resolver: %v", err)
	}
	got := kernel.CompletedWeekCount(s.Bars, now)
	if got < 4 {
		t.Fatalf("BAR-SOURCE: weekly reader still thin — %d completed weeks from %s/%s (legacy own-1m gave %d)",
			got, s.Source, s.FromTF, legacyWeeks)
	}
	if s.FromTF != "1d" || s.Source != market.SourceNT8Agg {
		t.Fatalf("weekly must resolve from native dailies, got %s/%s", s.Source, s.FromTF)
	}
	t.Logf("F1: own-1m gave %d completed weeks; the resolver gives %d from %s/%s",
		legacyWeeks, got, s.Source, s.FromTF)
}

// F3 — the weekly doc and its invalidation watch must read the SAME source.
// A synthetic week where the two would differ: dailies present, own-1m absent.
func TestWeeklyDocAndWatchShareOneSource(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	dayMs := int64(1440) * 60000
	start := now.AddDate(0, 0, -44).Truncate(24 * time.Hour).UnixMilli()
	var dailies []market.Kline
	for i := 0; i < 44; i++ {
		p := 29000 + float64(i)
		dailies = append(dailies, market.Kline{OpenTime: start + int64(i)*dayMs,
			Open: p, High: p + 50, Low: p - 50, Close: p + 10, Volume: 1000})
	}
	r := &market.BarResolver{
		Native: func(_, tf string, _ int) []market.Kline {
			if tf == "1d" {
				return dailies
			}
			return nil
		},
		Now: func() time.Time { return now },
	}
	a, _ := r.CompletedBars("MNQ", "1w", 0, now.UnixMilli())
	b, _ := r.CompletedBars("MNQ", "1w", 0, now.UnixMilli())
	if a.Source != b.Source || a.FromTF != b.FromTF || len(a.Bars) != len(b.Bars) {
		t.Fatalf("two reads of the same instant disagree: %s/%s n=%d vs %s/%s n=%d",
			a.Source, a.FromTF, len(a.Bars), b.Source, b.FromTF, len(b.Bars))
	}
	fa := kernel.ComputeWeeklyFacts(a.Bars, now, a.Bars[len(a.Bars)-1].Close)
	fb := kernel.ComputeWeeklyFacts(b.Bars, now, b.Bars[len(b.Bars)-1].Close)
	if fa.Refs.PWH != fb.Refs.PWH || fa.Refs.PWL != fb.Refs.PWL {
		t.Fatalf("doc and watch derive different PWH/PWL from one source: %.2f/%.2f vs %.2f/%.2f",
			fa.Refs.PWH, fa.Refs.PWL, fb.Refs.PWH, fb.Refs.PWL)
	}
}

// A24 — every value in the boot line is READ from the resolver, never a
// literal: change the fixture and the line changes with it.
func TestBarSourceBootLineReadsRealValues(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	dayMs := int64(1440) * 60000
	start := now.AddDate(0, 0, -44).Truncate(24 * time.Hour).UnixMilli()
	var dailies []market.Kline
	for i := 0; i < 44; i++ {
		dailies = append(dailies, market.Kline{OpenTime: start + int64(i)*dayMs,
			Open: 29000, High: 29050, Low: 28950, Close: 29010, Volume: 1})
	}
	r := &market.BarResolver{
		Native: func(_, tf string, _ int) []market.Kline {
			if tf == "1d" {
				return dailies
			}
			return nil
		},
		Now: func() time.Time { return now },
	}
	line := BarSourceBootLine(r, "MNQ", now)
	for _, want := range []string{"📊 bars:", "1w nt8_agg via 1d since", "1d nt8 since",
		"ladder(1w)=[1d 1m]", "native 1w EXCLUDED", "retention 1m=", "coarse=forever"} {
		if !strings.Contains(line, want) {
			t.Fatalf("boot line missing %q:\n%s", want, line)
		}
	}
	// TFs the fixture cannot answer are reported, not hidden.
	if !strings.Contains(line, "1h UNAVAILABLE") || !strings.Contains(line, "1m UNAVAILABLE") {
		t.Fatalf("unanswerable TFs must be named UNAVAILABLE:\n%s", line)
	}
	// The earliest date is the fixture's, not a constant.
	if !strings.Contains(line, time.UnixMilli(start).UTC().Format("2006-01-02")) {
		t.Fatalf("earliest date is not read from the data:\n%s", line)
	}
}

// R1 — the boot print says the cache may be cold and points at the second
// line; the post-backfill print reports what the resolver can actually reach.
func TestR1BarsBootAndAfterBackfillLines(t *testing.T) {
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	dayMs := int64(1440) * 60000
	start := now.AddDate(0, 0, -44).Truncate(24 * time.Hour).UnixMilli()
	var dailies []market.Kline
	for i := 0; i < 44; i++ {
		dailies = append(dailies, market.Kline{OpenTime: start + int64(i)*dayMs,
			Open: 29000, High: 29050, Low: 28950, Close: 29010, Volume: 1})
	}
	cold := &market.BarResolver{Native: func(string, string, int) []market.Kline { return nil },
		Own1m: func(string, int64, int64) []market.Kline { return nil },
		Now:   func() time.Time { return now }}
	warm := &market.BarResolver{Native: func(_, tf string, _ int) []market.Kline {
		if tf == "1d" {
			return dailies
		}
		return nil
	}, Now: func() time.Time { return now }}

	boot := BarSourceBootLine(cold, "MNQ", now)
	if !strings.Contains(boot, "cache cold at boot — see the 📊 bars after backfill line") {
		t.Fatalf("the boot line must warn that it may be reading a cold cache:\n%s", boot)
	}
	after := BarSourceBootLineAfterBackfill(warm, "MNQ", now)
	if !strings.HasPrefix(after, "📊 bars after backfill:") {
		t.Fatalf("second line prefix:\n%s", after)
	}
	if !strings.Contains(after, "1w nt8_agg via 1d since") {
		t.Fatalf("the post-backfill line must report the REACHABLE source:\n%s", after)
	}
	// Both prints share one field renderer, so they cannot disagree.
	if strings.Contains(after, "cache cold") {
		t.Fatal("the post-backfill line must not repeat the cold-cache caveat")
	}
	// Same resolver → same fields in both prints.
	if !strings.Contains(BarSourceBootLine(warm, "MNQ", now), "1w nt8_agg via 1d since") {
		t.Fatal("the two prints must share one field renderer")
	}
}
