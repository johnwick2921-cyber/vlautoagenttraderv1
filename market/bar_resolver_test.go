package market

import (
	"strings"
	"testing"
	"time"
)

func mkBars(startMs int64, mins, n int, base float64) []Kline {
	span := int64(mins) * 60000
	out := make([]Kline, 0, n)
	for i := 0; i < n; i++ {
		p := base + float64(i)
		out = append(out, Kline{OpenTime: startMs + int64(i)*span, Open: p, High: p + 2, Low: p - 2, Close: p + 1, Volume: 10,
			CloseTime: startMs + int64(i+1)*span - 1})
	}
	return out
}

func fixedNow(ms int64) func() time.Time { return func() time.Time { return time.UnixMilli(ms) } }

// F2 — the ladder is DATA and it is three deep for the coarse TFs (owner
// ruling 2026-09-02): native → next native down → own-1m last.
func TestBarResolverLadderIsData(t *testing.T) {
	want := map[string][]string{
		// native 1w excluded (Fri→Thu stamps) — TestWeeklyLadderExcludesNative1w
		"1w": {"1d", "1m"},
		"1d": {"1d", "1h", "1m"},
		"4h": {"4h", "1h", "1m"},
		"1h": {"1h", "15m", "1m"},
		"1m": {"1m"},
	}
	for tf, w := range want {
		got := LadderFor(tf)
		if len(got) != len(w) {
			t.Fatalf("%s ladder = %v, want %v", tf, got, w)
		}
		for i := range w {
			if got[i] != w[i] {
				t.Fatalf("%s ladder = %v, want %v", tf, got, w)
			}
		}
	}
	if LadderFor("bogus") != nil {
		t.Fatal("unknown TF must have no ladder")
	}
	// mutating the returned slice must not corrupt the table
	l := LadderFor("1w")
	l[0] = "HACKED"
	if LadderFor("1w")[0] != "1d" {
		t.Fatal("ladder table is aliased to callers")
	}
}

// F2 — native first: when the cache has the TF, it is used and labelled nt8.
func TestBarResolverPrefersNative(t *testing.T) {
	dayMs := int64(1440) * 60000
	start := int64(1700000000000) / dayMs * dayMs
	r := &BarResolver{
		Native: func(_, tf string, _ int) []Kline {
			if tf == "1d" {
				return mkBars(start, 1440, 6, 100)
			}
			return nil
		},
		Now: fixedNow(start + 6*dayMs),
	}
	s, err := r.CompletedBars("MNQ", "1d", 0, 0)
	if err != nil {
		t.Fatalf("err %v", err)
	}
	if s.Source != SourceNT8Native || s.FromTF != "1d" {
		t.Fatalf("source=%s fromTF=%s, want nt8/1d", s.Source, s.FromTF)
	}
	if len(s.Bars) != 6 {
		t.Fatalf("want 6 completed days, got %d", len(s.Bars))
	}
	if s.EarliestMs != start {
		t.Fatalf("earliest %d want %d", s.EarliestMs, start)
	}
}

// F2 — rung 2: no native 1w, but native 1d exists → aggregate, labelled
// nt8_agg, and own-1m is NEVER reached.
func TestBarResolverFallsBackToNextNativeNotOwn1m(t *testing.T) {
	dayMs := int64(1440) * 60000
	weekMs := int64(10080) * 60000
	start := int64(1700000000000) / weekMs * weekMs
	own1mCalled := false
	r := &BarResolver{
		Native: func(_, tf string, _ int) []Kline {
			if tf == "1d" {
				return mkBars(start, 1440, 28, 100) // 4 weeks of dailies
			}
			return nil // no native 1w
		},
		Own1m: func(string, int64, int64) []Kline { own1mCalled = true; return mkBars(start, 1, 100, 100) },
		Now:   fixedNow(start + 28*dayMs),
	}
	s, _ := r.CompletedBars("MNQ", "1w", 0, 0)
	if s.Source != SourceNT8Agg || s.FromTF != "1d" {
		t.Fatalf("source=%s from=%s, want nt8_agg/1d", s.Source, s.FromTF)
	}
	if own1mCalled {
		t.Fatal("own-1m must NEVER be reached for weekly when native dailies exist")
	}
	if len(s.Bars) != 4 {
		t.Fatalf("28 dailies must roll into 4 completed weeks, got %d", len(s.Bars))
	}
}

// F2 — last rung: nothing native at all → own-1m, labelled own1m.
func TestBarResolverLastRungOwn1m(t *testing.T) {
	hourMs := int64(60) * 60000
	start := int64(1700000000000) / hourMs * hourMs
	r := &BarResolver{
		Native: func(string, string, int) []Kline { return nil },
		Own1m:  func(string, int64, int64) []Kline { return mkBars(start, 1, 300, 100) },
		Now:    fixedNow(start + 300*60000),
	}
	s, _ := r.CompletedBars("MNQ", "1h", 0, 0)
	if s.Source != SourceOwn1m || s.FromTF != "1m" {
		t.Fatalf("source=%s from=%s", s.Source, s.FromTF)
	}
	if len(s.Bars) != 5 {
		t.Fatalf("300 1m bars = 5 completed hours, got %d", len(s.Bars))
	}
}

// F2 — THE REPAINT LAW: a forming bar is never returned, from any rung.
func TestBarResolverNeverReturnsFormingBar(t *testing.T) {
	hourMs := int64(60) * 60000
	start := int64(1700000000000) / hourMs * hourMs
	native := mkBars(start, 60, 5, 100) // bars 0..4; bar 4 opens at start+4h
	// now is 30 min INTO bar 4 → bar 4 is forming
	r := &BarResolver{Native: func(string, string, int) []Kline { return native },
		Now: fixedNow(start + 4*hourMs + 30*60000)}
	s, _ := r.CompletedBars("MNQ", "1h", 0, 0)
	if len(s.Bars) != 4 {
		t.Fatalf("the forming bar must be dropped: got %d want 4", len(s.Bars))
	}
	if s.Bars[len(s.Bars)-1].OpenTime != start+3*hourMs {
		t.Fatalf("last completed bar wrong: %d", s.Bars[len(s.Bars)-1].OpenTime)
	}
	if !s.Completed {
		t.Fatal("Completed must be true")
	}
	// CompletedBar for the forming instant returns nothing.
	b, _, _ := r.CompletedBar("MNQ", "1h", time.UnixMilli(start+4*hourMs+30*60000))
	if b != nil {
		t.Fatal("CompletedBar must not return a forming bar")
	}
	// ...but the previous hour resolves.
	b2, _, _ := r.CompletedBar("MNQ", "1h", time.UnixMilli(start+3*hourMs+10))
	if b2 == nil || b2.OpenTime != start+3*hourMs {
		t.Fatalf("previous completed hour not resolved: %+v", b2)
	}
}

// F5 — stamp-convention parity with the 1m persister: buckets are keyed by
// floor-aligned OPEN time (class 7 — one convention, one chokepoint).
func TestAggregateStampConventionParity(t *testing.T) {
	span := int64(3600000)
	base := int64(1700000000000)/span*span + 17*60000 // deliberately mid-hour start
	in := mkBars(base, 1, 120, 100)
	out := AggregateToTF(in, 1, 60)
	for _, b := range out {
		if b.OpenTime%span != 0 {
			t.Fatalf("bucket %d is not floor-aligned to the hour", b.OpenTime)
		}
		if b.CloseTime != b.OpenTime+span-1 {
			t.Fatalf("close stamp convention broken: open=%d close=%d", b.OpenTime, b.CloseTime)
		}
	}
	if len(out) < 2 {
		t.Fatalf("expected ≥2 hourly buckets, got %d", len(out))
	}
	// OHLC integrity on a full bucket
	full := out[1]
	if full.High < full.Low || full.Open == 0 || full.Close == 0 {
		t.Fatalf("bad bucket %+v", full)
	}
}

// An unknown TF errors rather than inventing a ladder.
func TestBarResolverUnknownTF(t *testing.T) {
	r := &BarResolver{}
	if _, err := r.CompletedBars("MNQ", "7m", 0, 0); err == nil {
		t.Fatal("unknown TF must error")
	}
	if _, _, err := r.CompletedBar("MNQ", "7m", time.Now()); err == nil {
		t.Fatal("unknown TF must error")
	}
}

// Nothing anywhere → an honest empty answer, never a fabricated series.
func TestBarResolverNoSourceIsHonest(t *testing.T) {
	r := &BarResolver{}
	s, err := r.CompletedBars("MNQ", "1w", 0, 0)
	if err != nil || s.Source != SourceNone || len(s.Bars) != 0 {
		t.Fatalf("want honest empty, got source=%s n=%d err=%v", s.Source, len(s.Bars), err)
	}
}

// A23 FINDING (2026-09-02), pinned: native 1w is DELIBERATELY excluded from
// the weekly ladder because NT8 stamps weekly bars Friday→Thursday while our
// weeks are Monday-governed. If a later wave "restores" nt8-first for 1w, this
// fails and the reason is in the message.
func TestWeeklyLadderExcludesNative1w(t *testing.T) {
	l := LadderFor("1w")
	if len(l) == 0 || l[0] != "1d" {
		t.Fatalf("weekly must resolve from native 1d, got ladder %v", l)
	}
	for _, tf := range l {
		if tf == "1w" {
			t.Fatalf("native 1w must NOT be in the weekly ladder: %v", l)
		}
	}
	why := ExcludedNative("1w")
	if why == "" || !strings.Contains(why, "Friday") {
		t.Fatalf("the exclusion must carry its reason, got %q", why)
	}
	if ExcludedNative("1d") != "" {
		t.Fatal("1d is clean calendar-day and must not be excluded")
	}
}

// Even if the cache serves native 1w, the resolver must not reach for it.
func TestResolverIgnoresNative1wEvenWhenPresent(t *testing.T) {
	dayMs := int64(1440) * 60000
	weekMs := int64(10080) * 60000
	start := int64(1700000000000) / weekMs * weekMs
	used := ""
	r := &BarResolver{
		Native: func(_, tf string, _ int) []Kline {
			used = used + tf + ","
			switch tf {
			case "1w":
				return mkBars(start+3*dayMs, 10080, 8, 999) // Friday-stamped, wrong calendar
			case "1d":
				return mkBars(start, 1440, 28, 100)
			}
			return nil
		},
		Now: fixedNow(start + 28*dayMs),
	}
	s, _ := r.CompletedBars("MNQ", "1w", 0, 0)
	if s.FromTF != "1d" || s.Source != SourceNT8Agg {
		t.Fatalf("weekly resolved from %s/%s, want 1d/nt8_agg", s.FromTF, s.Source)
	}
	if strings.Contains(used, "1w,") {
		t.Fatalf("the resolver asked the cache for 1w: %q", used)
	}
	if s.Bars[0].Open != 100 {
		t.Fatalf("weekly bars came from the wrong series: %+v", s.Bars[0])
	}
}

// The stamp guard catches the mismatch class generically.
func TestStampAlignedCatchesOffsetSeries(t *testing.T) {
	weekMs := int64(10080) * 60000
	base := int64(1700000000000) / weekMs * weekMs
	ok, _ := StampAligned(mkBars(base, 10080, 4, 100), "1w")
	if !ok {
		t.Fatal("aligned series reported misaligned")
	}
	bad, off := StampAligned(mkBars(base+3*1440*60000, 10080, 4, 100), "1w")
	if bad {
		t.Fatal("a Friday-stamped weekly series must be reported misaligned")
	}
	if off == 0 {
		t.Fatal("the offset must be reported")
	}
}
