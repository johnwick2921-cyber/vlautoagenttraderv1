package kernel

import (
	"testing"
	"time"
)

func TestIsCMEOpen(t *testing.T) {
	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("failed to load America/Chicago timezone: %v", err)
	}
	cases := []struct {
		name string
		when time.Time
		want bool
	}{
		{"Mon 10am normal trading", time.Date(2026, 6, 15, 10, 0, 0, 0, chicago), true},
		{"Mon daily break 4:30pm CT", time.Date(2026, 6, 15, 16, 30, 0, 0, chicago), false},
		{"Saturday closed", time.Date(2026, 6, 20, 12, 0, 0, 0, chicago), false},
		{"New Year's Day", time.Date(2026, 1, 1, 10, 0, 0, 0, chicago), false},
		{"MLK Day 2026 (Jan 19)", time.Date(2026, 1, 19, 10, 0, 0, 0, chicago), false},
		{"Presidents Day 2026 (Feb 16)", time.Date(2026, 2, 16, 10, 0, 0, 0, chicago), false},
		{"Good Friday 2026 (Apr 3)", time.Date(2026, 4, 3, 10, 0, 0, 0, chicago), false},
		{"Memorial Day 2026 (May 25)", time.Date(2026, 5, 25, 10, 0, 0, 0, chicago), false},
		{"Juneteenth", time.Date(2026, 6, 19, 10, 0, 0, 0, chicago), false},
		{"Independence Day", time.Date(2026, 7, 4, 10, 0, 0, 0, chicago), false},
		{"Labor Day 2026 (Sep 7)", time.Date(2026, 9, 7, 10, 0, 0, 0, chicago), false},
		{"Thanksgiving 2026 (Nov 26)", time.Date(2026, 11, 26, 10, 0, 0, 0, chicago), false},
		{"Day after Thanksgiving 2026 (Nov 27)", time.Date(2026, 11, 27, 10, 0, 0, 0, chicago), false},
		{"Christmas Eve", time.Date(2026, 12, 24, 10, 0, 0, 0, chicago), false},
		{"Christmas Day", time.Date(2026, 12, 25, 10, 0, 0, 0, chicago), false},
		{"New Year's Eve", time.Date(2026, 12, 31, 10, 0, 0, 0, chicago), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsCMEOpen(tc.when); got != tc.want {
				t.Errorf("IsCMEOpen(%v) = %v, want %v", tc.when, got, tc.want)
			}
		})
	}
}

// TestIsCMEOpenBoundaries pins the exact open/close edges (Part A). Friday close,
// Sunday reopen, the Mon–Thu daily break, and a DST-month sample so a tz bug can
// never slip in unnoticed.
func TestIsCMEOpenBoundaries(t *testing.T) {
	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatalf("load tz: %v", err)
	}
	cases := []struct {
		name string
		when time.Time
		want bool
	}{
		// Friday close edge (Jun 19 2026 is Juneteenth → use Jun 12, a normal Friday, CDT)
		{"Fri 15:59 CT open", time.Date(2026, 6, 12, 15, 59, 0, 0, chicago), true},
		{"Fri 16:01 CT closed", time.Date(2026, 6, 12, 16, 1, 0, 0, chicago), false},
		// Sunday reopen edge (Jun 14 2026)
		{"Sun 16:59 CT closed", time.Date(2026, 6, 14, 16, 59, 0, 0, chicago), false},
		{"Sun 17:01 CT open", time.Date(2026, 6, 14, 17, 1, 0, 0, chicago), true},
		{"Sun 17:00 CT open (exact)", time.Date(2026, 6, 14, 17, 0, 0, 0, chicago), true},
		// Mon–Thu daily break edges (Jun 16 2026, a Tuesday)
		{"Tue 15:59 CT open", time.Date(2026, 6, 16, 15, 59, 0, 0, chicago), true},
		{"Tue 16:30 CT break closed", time.Date(2026, 6, 16, 16, 30, 0, 0, chicago), false},
		{"Tue 17:00 CT reopen", time.Date(2026, 6, 16, 17, 0, 0, 0, chicago), true},
		// DST: same wall-clock edges in a CST (winter) month — Jan 9 2026 is a Friday
		{"Fri 15:59 CST open (winter)", time.Date(2026, 1, 9, 15, 59, 0, 0, chicago), true},
		{"Fri 16:01 CST closed (winter)", time.Date(2026, 1, 9, 16, 1, 0, 0, chicago), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsCMEOpen(tc.when); got != tc.want {
				t.Errorf("IsCMEOpen(%v) = %v, want %v", tc.when, got, tc.want)
			}
		})
	}
}

// TestCMEClosedReason asserts the reason bool never disagrees with IsCMEOpen
// (the whole gate depends on that) and that each reason string is right.
func TestCMEClosedReason(t *testing.T) {
	chicago, _ := time.LoadLocation("America/Chicago")
	cases := []struct {
		name       string
		when       time.Time
		wantClosed bool
		wantReason string
	}{
		{"Sat weekend", time.Date(2026, 6, 13, 12, 0, 0, 0, chicago), true, "weekend"},
		{"Sun pre-open weekend", time.Date(2026, 6, 14, 12, 0, 0, 0, chicago), true, "weekend"},
		{"Fri close", time.Date(2026, 6, 12, 16, 30, 0, 0, chicago), true, "Friday close"},
		{"Tue daily break", time.Date(2026, 6, 16, 16, 30, 0, 0, chicago), true, "daily break"},
		{"Holiday (Independence Day)", time.Date(2026, 7, 4, 10, 0, 0, 0, chicago), true, "holiday"},
		{"Mon open (no reason)", time.Date(2026, 6, 15, 10, 0, 0, 0, chicago), false, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			closed, reason := CMEClosedReason(tc.when)
			if closed != tc.wantClosed || reason != tc.wantReason {
				t.Errorf("CMEClosedReason(%v) = (%v,%q), want (%v,%q)", tc.when, closed, reason, tc.wantClosed, tc.wantReason)
			}
			// Invariant: closed == !IsCMEOpen
			if closed == IsCMEOpen(tc.when) {
				t.Errorf("CMEClosedReason closed=%v disagrees with IsCMEOpen=%v at %v", closed, IsCMEOpen(tc.when), tc.when)
			}
		})
	}
}

// TestNextCMEOpen checks the reopen instant is correct from several closed
// states, including a Friday→Sunday jump and a DST fall-back weekend, and that
// the result always satisfies IsCMEOpen.
func TestNextCMEOpen(t *testing.T) {
	chicago, _ := time.LoadLocation("America/Chicago")
	cases := []struct {
		name string
		from time.Time
		want time.Time
	}{
		{"Sat → Sun 17:00", time.Date(2026, 6, 13, 12, 0, 0, 0, chicago), time.Date(2026, 6, 14, 17, 0, 0, 0, chicago)},
		{"Fri close → Sun 17:00", time.Date(2026, 6, 12, 16, 30, 0, 0, chicago), time.Date(2026, 6, 14, 17, 0, 0, 0, chicago)},
		{"Daily break → same-day 17:00", time.Date(2026, 6, 16, 16, 20, 0, 0, chicago), time.Date(2026, 6, 16, 17, 0, 0, 0, chicago)},
		// DST fall-back: Fri Oct 30 close → Sun Nov 1 17:00 CST (clocks fell back 02:00 that morning)
		{"DST fall-back weekend", time.Date(2026, 10, 30, 16, 30, 0, 0, chicago), time.Date(2026, 11, 1, 17, 0, 0, 0, chicago)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := NextCMEOpen(tc.from)
			if !got.Equal(tc.want) {
				t.Errorf("NextCMEOpen(%v) = %v, want %v", tc.from, got.In(chicago), tc.want)
			}
			if !IsCMEOpen(got) {
				t.Errorf("NextCMEOpen(%v) returned %v which is not open", tc.from, got.In(chicago))
			}
		})
	}
}
