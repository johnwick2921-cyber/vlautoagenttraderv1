package kernel

import (
	"math"
	"testing"
	"time"

	"nofx/market"
)

// WEEKLY-BIAS W1 fixtures — R2 independent-math tests: every expectation is
// recomputed by hand from the raw bars; none of them call the function under
// test to produce the expected value.

func wkBar(date string, hhmm string, o, h, l, c float64) market.Kline {
	loc := CTLocation()
	t, err := time.ParseInLocation("2006-01-02 15:04", date+" "+hhmm, loc)
	if err != nil {
		panic(err)
	}
	return market.Kline{OpenTime: t.UnixMilli(), Open: o, High: h, Low: l, Close: c, Volume: 1}
}

func wkNow(date string, hhmm string) time.Time {
	loc := CTLocation()
	t, err := time.ParseInLocation("2006-01-02 15:04", date+" "+hhmm, loc)
	if err != nil {
		panic(err)
	}
	return t
}

// wkFixture builds 3 COMPLETED weeks + one IN-PROGRESS week of 1m bars.
func wkFixture() []market.Kline {
	var bars []market.Kline
	// Week 1 (Mon 2026-08-10) — completed: O=100 H=110 L=95 C=105.
	bars = append(bars, wkBar("2026-08-10", "17:00", 100, 108, 100, 107))
	bars = append(bars, wkBar("2026-08-14", "15:59", 107, 110, 95, 105))
	// Sunday opens (new week's first prints) — also birth the NWOGs.
	bars = append(bars, wkBar("2026-08-16", "17:00", 106, 106, 106, 106))
	// Week 2 (Mon 2026-08-17) — completed: O=106 H=115 L=100 C=112.
	bars = append(bars, wkBar("2026-08-17", "17:01", 106, 109, 106, 108))
	bars = append(bars, wkBar("2026-08-21", "15:58", 108, 115, 100, 112))
	bars = append(bars, wkBar("2026-08-23", "17:00", 113, 113, 113, 113))
	// Week 3 (Mon 2026-08-24) — completed: O=113 H=114 L=96 C=98.
	bars = append(bars, wkBar("2026-08-24", "17:02", 113, 113, 112, 112))
	bars = append(bars, wkBar("2026-08-28", "15:57", 112, 114, 96, 98))
	bars = append(bars, wkBar("2026-08-30", "17:00", 99, 99, 99, 99))
	// Week 4 (Mon 2026-08-31) — IN PROGRESS at now (Wed 09-02 12:00 CT).
	bars = append(bars, wkBar("2026-08-31", "17:03", 99, 102, 99, 101))
	return bars
}

func TestWeeklyAggTwin(t *testing.T) {
	now := wkNow("2026-09-02", "12:00")
	bars := wkFixture()
	weeks := CompletedWeekCandles(bars, now, 12)
	if len(weeks) != 3 {
		t.Fatalf("want 3 completed weeks (current excluded), got %d", len(weeks))
	}
	// R2 hand-computed expectations from wkFixture.
	exp := []WeekCandle{
		{WeekStart: "2026-08-10", Open: 100, High: 110, Low: 95, Close: 105, StructTag: "first"},
		{WeekStart: "2026-08-17", Open: 106, High: 115, Low: 100, Close: 112, StructTag: "HH"},
		{WeekStart: "2026-08-24", Open: 113, High: 114, Low: 96, Close: 98, StructTag: "LL"},
	}
	for i, w := range weeks {
		e := exp[i]
		if w.WeekStart != e.WeekStart || w.Open != e.Open || w.High != e.High || w.Low != e.Low || w.Close != e.Close {
			t.Fatalf("week %d = %+v, want %+v", i, w, e)
		}
		if (i == 0 && w.StructTag != "") || (i > 0 && w.StructTag != e.StructTag) {
			t.Fatalf("week %d tag = %q, want %q", i, w.StructTag, e.StructTag)
		}
	}
}

func TestCurrentWeekExcluded(t *testing.T) {
	now := wkNow("2026-09-02", "12:00")
	bars := wkFixture()
	// The in-progress week (08-31 bars) is present in the INPUT…
	for _, b := range bars {
		if time.UnixMilli(b.OpenTime).In(CTLocation()).Format("2006-01-02") >= "2026-08-31" {
			// present — covered by wkFixture
		}
	}
	// …but the output must never contain it (repaint law).
	for _, w := range CompletedWeekCandles(bars, now, 12) {
		if w.WeekStart == "2026-08-31" {
			t.Fatalf("repaint law broken: in-progress week %s leaked into bias evidence", w.WeekStart)
		}
	}
}

func TestNWOGMathAndFill(t *testing.T) {
	now := wkNow("2026-09-02", "12:00")
	bars := wkFixture()
	gaps := LastNWOGs(bars, now, 5)
	if len(gaps) != 3 {
		t.Fatalf("want 3 weekend gaps, got %d: %+v", len(gaps), gaps)
	}
	// Gap A (born Sun 08-16 → week Mon 08-17): Fri 08-14 close 105 → Sun open 106.
	a := gaps[0]
	if a.Born != "2026-08-17" || a.Lo != 105 || a.Hi != 106 || a.CE != 105.5 {
		t.Fatalf("gap A = %+v, want born 08-17 lo 105 hi 106 ce 105.5", a)
	}
	if !a.Filled {
		t.Fatal("gap A CE 105.5 was traded through by week 2's 100–115 range → filled")
	}
	// Gap B (born Sun 08-23 → week Mon 08-24): Fri 08-21 close 112 → Sun open 113.
	b := gaps[1]
	if b.Born != "2026-08-24" || b.Lo != 112 || b.Hi != 113 || b.CE != 112.5 {
		t.Fatalf("gap B = %+v, want born 08-24 lo 112 hi 113 ce 112.5", b)
	}
	if !b.Filled {
		t.Fatal("gap B CE 112.5 was traded through by week 3's 96–114 range → filled")
	}
	// Gap C (born Sun 08-30 → week Mon 08-31): Fri 08-28 close 98 → Sun open 99.
	c := gaps[2]
	if c.Born != "2026-08-31" || c.Lo != 98 || c.Hi != 99 || c.CE != 98.5 {
		t.Fatalf("gap C = %+v, want born 08-31 lo 98 hi 99 ce 98.5", c)
	}
	if c.Filled {
		t.Fatal("gap C CE 98.5: no post-birth bar trades through it (Mon 08-31 bar L=99) → unfilled")
	}
	// A later bar trading through gap C's CE fills it.
	fill := wkBar("2026-09-01", "09:00", 98.4, 99.2, 98.0, 98.8)
	if gaps2 := LastNWOGs(append(bars, fill), now, 5); !gaps2[2].Filled {
		t.Fatal("a bar with H=99.2 L=98.0 through CE 98.5 must fill gap C")
	}
}

func TestIPDAHandMath(t *testing.T) {
	// 70 session-day daily candles: day i → H=100+i, L=90+i, C=95+i.
	var daily []market.Kline
	for i := 0; i < 70; i++ {
		daily = append(daily, market.Kline{OpenTime: int64(i) * 86_400_000, High: float64(100 + i), Low: float64(90 + i), Close: float64(95 + i)})
	}
	price := 154.5
	ranges := IPDA(daily, price)
	if len(ranges) != 3 {
		t.Fatalf("want 3 IPDA ranges, got %d", len(ranges))
	}
	// R2 hand math: last-20 = days 50..69 → H 169, L 140; last-40 = 30..69 →
	// H 169, L 120; last-60 = 10..69 → H 169, L 100.
	exp := []IPDARange{
		{Days: 20, High: 169, Low: 140, PosPct: (154.5 - 140) / 29},
		{Days: 40, High: 169, Low: 120, PosPct: (154.5 - 120) / 49},
		{Days: 60, High: 169, Low: 100, PosPct: (154.5 - 100) / 69},
	}
	for i, e := range exp {
		r := ranges[i]
		if r.Days != e.Days || r.High != e.High || r.Low != e.Low {
			t.Fatalf("range %d = %+v, want %+v", i, r, e)
		}
		if math.Abs(r.PosPct-e.PosPct) > 1e-9 {
			t.Fatalf("range %d pos%% = %.6f, want %.6f", i, r.PosPct, e.PosPct)
		}
	}
}

func TestIPDAInsufficientHistory(t *testing.T) {
	var daily []market.Kline
	for i := 0; i < 25; i++ {
		daily = append(daily, market.Kline{OpenTime: int64(i) * 86_400_000, High: 100, Low: 90})
	}
	ranges := IPDA(daily, 95)
	if ranges[0].PosPct < 0 || ranges[1].PosPct >= 0 || ranges[2].PosPct >= 0 {
		t.Fatalf("20d must render, 40/60d must be insufficient: %+v", ranges)
	}
}

func TestDepthGuardCount(t *testing.T) {
	now := wkNow("2026-09-02", "12:00")
	if n := CompletedWeekCount(wkFixture(), now); n != 3 {
		t.Fatalf("completed week count = %d, want 3 (< 4 → thin_history)", n)
	}
}

// TestWeekStartMondaySundayMorning — regression for the 2026-08-30 cutover bug:
// Sunday MORNING (before 17:00 CT) must govern the FOLLOWING Monday's week
// (the session-day anchor mis-mapped it one week back → the wrong-week
// boot-backfill fired).
func TestWeekStartMondaySundayMorning(t *testing.T) {
	cases := []struct{ in, want string }{
		{"2026-08-30 09:37", "2026-08-31"}, // Sunday morning → next Monday
		{"2026-08-30 17:00", "2026-08-31"}, // Sunday 17:00 → new week's Monday
		{"2026-08-31 08:00", "2026-08-31"}, // Monday
		{"2026-08-28 15:59", "2026-08-24"}, // Friday → this week's Monday
		{"2026-08-29 12:00", "2026-08-24"}, // Saturday → this week's Monday
	}
	for _, c := range cases {
		ts, err := time.ParseInLocation("2006-01-02 15:04", c.in, CTLocation())
		if err != nil {
			t.Fatal(err)
		}
		got := weekStartMonday(ts).Format("2006-01-02")
		if got != c.want {
			t.Fatalf("weekStartMonday(%s) = %s, want %s", c.in, got, c.want)
		}
	}
}
