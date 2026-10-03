package trader

import (
	"testing"
	"time"

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
