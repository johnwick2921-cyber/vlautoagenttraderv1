package trader

import (
	"testing"
	"time"

	"vl/calendar"
	"vl/kernel"
	"vl/kernel/mentor"
	"vl/market"
	"vl/store"
)

// TestMentorNoChaseRule pins the pure rule: AT OR BEYOND the trigger the
// entry is skipped (he never enters at market, §3); strictly before it the
// entry passes. The mutant (inverting the comparison) fails this.
func TestMentorNoChaseRule(t *testing.T) {
	cases := []struct {
		name    string
		side    mentor.Side
		latest  float64
		trigger float64
		skip    bool
	}{
		{"long below passes", mentor.SideLong, 99.5, 100, false},
		{"long at trigger skips", mentor.SideLong, 100, 100, true},
		{"long through skips", mentor.SideLong, 100.25, 100, true},
		{"short above passes", mentor.SideShort, 100.5, 100, false},
		{"short at trigger skips", mentor.SideShort, 100, 100, true},
		{"short through skips", mentor.SideShort, 99.75, 100, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			skip, why := mentorNoChase(c.side, c.latest, c.trigger)
			if skip != c.skip {
				t.Fatalf("skip=%v want %v (%q)", skip, c.skip, why)
			}
			if skip && why == "" {
				t.Fatal("a skip must name why")
			}
		})
	}
}

// TestMentorNoChaseAtPlacementCallSite: the placement path consults the latest
// live price BEFORE sending. The mutant that drops the check makes the
// recorder fire on a through-price intent and this test goes RED.
func TestMentorNoChaseAtPlacementCallSite(t *testing.T) {
	ResetMentorCountersForTest()
	at := mentoredTrader(t, store.RiskControlConfig{MentorMode: true})

	in := mentor.Intent{Action: mentor.PlaceStopEntry, Side: mentor.SideLong,
		Price: 21000, Stop: 20988, Target: 21024, Setup: "ISB", StopPts: 12}
	choice := mentorSizeChoice{Contracts: 5, Tier: "base", Why: "test"}

	var placed []mentor.Intent
	mentorPlaceRecorderForTest = func(i mentor.Intent, n int) { placed = append(placed, i) }
	t.Cleanup(func() { mentorPlaceRecorderForTest = nil })

	// price already through the trigger → no placement, counted skip
	mentorLatestPriceSource = func() (float64, bool) { return 21000.50, true }
	t.Cleanup(func() { mentorLatestPriceSource = nil })
	at.mentorPlaceIntent(in, choice, 1000, 1100)
	if len(placed) != 0 {
		t.Fatalf("a through-price entry must NOT be placed (no chase), placed=%v", placed)
	}
	if got := MentorCountSnapshot()["no_chase_skip"]; got != 1 {
		t.Fatalf("the skip must be counted once, got %d", got)
	}

	// price still before the trigger → the placement proceeds
	mentorLatestPriceSource = func() (float64, bool) { return 20999.50, true }
	at.mentorPlaceIntent(in, choice, 1000, 1100)
	if len(placed) != 1 {
		t.Fatalf("a safe-price entry must reach the placement path, placed=%v", placed)
	}
}

// TestMentorEventPassDedup: the event pass evaluates each FINAL bar exactly
// once (a second pass over the same bar is a no-op).
func TestMentorEventPassDedup(t *testing.T) {
	at := mentoredTrader(t, store.RiskControlConfig{MentorMode: true})
	old := market.FuturesBarsProvider
	t.Cleanup(func() { market.FuturesBarsProvider = old })
	open := int64(1_700_000_000_000)
	market.FuturesBarsProvider = func(symbol, tf string, n int) []market.Kline {
		return []market.Kline{
			{OpenTime: open, CloseTime: open + 59_000, Open: 100, High: 101, Low: 99, Close: 100.5},
		}
	}
	if !at.mentorEventPassAt(time.Now()) {
		t.Fatal("the first pass over a new FINAL bar must run")
	}
	if at.mentorEventPassAt(time.Now()) {
		t.Fatal("a second pass over the same bar must be a no-op")
	}
}

// TestMentorLatencySnapshotAndBootLine: p50/p95 from the reservoir; the boot
// line prints n/a until the first sample (canon L7).
func TestMentorLatencySnapshotAndBootLine(t *testing.T) {
	ResetMentorLatencyForTest()
	if line := MentorLatencyBootLine(); line == "" || !textHas(line, "n/a") {
		t.Fatalf("empty reservoir boot line must print n/a, got %q", line)
	}
	base := int64(1_000_000)
	for i := int64(0); i < 100; i++ {
		recordMentorLatency(base, base+10, base+20, base+100+i) // 100..199 ms
	}
	n, p50, p95 := MentorLatencySnapshot()
	if n != 100 {
		t.Fatalf("n=%d want 100", n)
	}
	if p50 < 145 || p50 > 155 {
		t.Fatalf("p50=%dms, want ~150ms", p50)
	}
	if p95 < 190 || p95 > 199 {
		t.Fatalf("p95=%dms, want ~195ms", p95)
	}
	if line := MentorLatencyBootLine(); textHas(line, "n/a") {
		t.Fatalf("a populated reservoir must not print n/a: %q", line)
	}
}

func textHas(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// TestMentorStrongDayFrom5m (S9, D5.2 p2 @05:21): a closed 5m candle in the
// window running 50–80 pts marks a strong day → the size table cuts to 1–2.
func TestMentorStrongDayFrom5m(t *testing.T) {
	mk := func(rng float64) market.Kline { return market.Kline{High: 100 + rng, Low: 100} }
	if mentorStrongDayFrom5m(nil) {
		t.Fatal("no bars must not be a strong day")
	}
	if mentorStrongDayFrom5m([]market.Kline{mk(49.75)}) {
		t.Fatal("a 49.75-pt 5m candle must NOT trigger the strong day")
	}
	for _, rng := range []float64{50, 60, 80, 85} {
		if !mentorStrongDayFrom5m([]market.Kline{mk(rng)}) {
			t.Fatalf("a %.1f-pt 5m candle IS a strong day", rng)
		}
	}
	if !mentorStrongDayFrom5m([]market.Kline{mk(20), mk(25), mk(55)}) {
		t.Fatal("a 55-pt candle anywhere in the 5m window qualifies")
	}
}

// TestMentorNewsHold (F11): the pure gate refuses a placement inside the
// 07:30 CT CPI/PPI/Unemployment print window on a T1 print day, and only then.
func TestMentorNewsHold(t *testing.T) {
	ct := kernel.CTLocation()
	event := func(title string, impact calendar.Impact, hh, mm int) calendar.Event {
		return calendar.Event{Title: title, Impact: impact,
			Time: time.Date(2026, 10, 2, hh, mm, 0, 0, ct).UTC()}
	}
	cpi := []calendar.Event{event("CPI m/m", calendar.T1, 7, 30)}
	cases := []struct {
		name string
		evs  []calendar.Event
		now  time.Time
		want bool
	}{
		{"window start", cpi, time.Date(2026, 10, 2, 7, 20, 0, 0, ct), true},
		{"window end exclusive", cpi, time.Date(2026, 10, 2, 7, 35, 0, 0, ct), false},
		{"just before the window", cpi, time.Date(2026, 10, 2, 7, 19, 0, 0, ct), false},
		{"inside", cpi, time.Date(2026, 10, 2, 7, 28, 0, 0, ct), true},
		{"after", cpi, time.Date(2026, 10, 2, 7, 36, 0, 0, ct), false},
		{"PPI", []calendar.Event{event("PPI m/m", calendar.T1, 7, 30)}, time.Date(2026, 10, 2, 7, 25, 0, 0, ct), true},
		{"unemployment", []calendar.Event{event("Unemployment Rate", calendar.T1, 7, 30)}, time.Date(2026, 10, 2, 7, 25, 0, 0, ct), true},
		{"T2 print does not hold", []calendar.Event{event("CPI m/m", calendar.T2, 7, 30)}, time.Date(2026, 10, 2, 7, 25, 0, 0, ct), false},
		{"other minute", []calendar.Event{event("CPI m/m", calendar.T1, 9, 0)}, time.Date(2026, 10, 2, 7, 25, 0, 0, ct), false},
		{"other title", []calendar.Event{event("GDP q/q", calendar.T1, 7, 30)}, time.Date(2026, 10, 2, 7, 25, 0, 0, ct), false},
		{"no events", nil, time.Date(2026, 10, 2, 7, 25, 0, 0, ct), false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			hold, why := mentorNewsHold(c.evs, c.now)
			if hold != c.want {
				t.Fatalf("hold=%v why=%q, want %v", hold, why, c.want)
			}
			if hold && why == "" {
				t.Fatal("a hold must carry the why")
			}
		})
	}
}

// TestMentorNewsGateAtPlacementCallSite: the placement path consults the news
// gate FIRST. The mutant that removes the gate makes the recorder fire inside
// the print window and this test goes RED.
func TestMentorNewsGateAtPlacementCallSite(t *testing.T) {
	ResetMentorCountersForTest()
	at := mentoredTrader(t, store.RiskControlConfig{MentorMode: true})

	ct := kernel.CTLocation()
	mentorNewsNowSource = func() time.Time { return time.Date(2026, 10, 2, 7, 25, 0, 0, ct) }
	mentorDayEventsForTest = func() []calendar.Event {
		return []calendar.Event{{Title: "CPI m/m", Impact: calendar.T1,
			Time: time.Date(2026, 10, 2, 7, 30, 0, 0, ct).UTC()}}
	}
	t.Cleanup(func() { mentorNewsNowSource = nil; mentorDayEventsForTest = nil })

	in := mentor.Intent{Action: mentor.PlaceStopEntry, Side: mentor.SideLong,
		Price: 21000, Stop: 20988, Target: 21024, Setup: "ISB", StopPts: 12}
	choice := mentorSizeChoice{Contracts: 5, Tier: "base", Why: "test"}

	var placed int
	mentorPlaceRecorderForTest = func(i mentor.Intent, n int) { placed++ }
	t.Cleanup(func() { mentorPlaceRecorderForTest = nil })

	// inside the 07:30 CPI print window: the recorder must NOT fire.
	at.mentorPlaceIntent(in, choice, 1000, 1100)
	if placed != 0 {
		t.Fatalf("a placement inside the 07:30 print window must be refused, placed=%d", placed)
	}
	if got := MentorCountSnapshot()["news_hold"]; got != 1 {
		t.Fatalf("the news hold must be counted once, got %d", got)
	}
	// outside the window (06:00 CT) the same intent proceeds.
	mentorNewsNowSource = func() time.Time { return time.Date(2026, 10, 2, 6, 0, 0, 0, ct) }
	at.mentorPlaceIntent(in, choice, 1000, 1100)
	if placed != 1 {
		t.Fatalf("outside the print window the placement must proceed, placed=%d", placed)
	}
}
