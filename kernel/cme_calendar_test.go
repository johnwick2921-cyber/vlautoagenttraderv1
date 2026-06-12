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
