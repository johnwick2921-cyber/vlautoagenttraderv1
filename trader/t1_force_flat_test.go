package trader

import (
	"testing"

	"nofx/kernel"
)

// t1ForceFlatDue is the pure decision boundary for the W3.4 T-2min red-news
// force-flat: nowMin must land inside [window.Start − lead, window.End] for the
// flatten to be due. Pin the boundary both ways + the wrap-around midnight case.

func TestT1ForceFlatDue(t *testing.T) {
	// A synthetic FOMC window: event at 12:00 CT, blackout 11:45–12:15
	// (Start 705, End 735). Lead 2 → force-flat range [703, 735].
	win := []kernel.CTWindow{{Start: 705, End: 735, Label: "FOMC Statement 12:00 CT ±15m"}}

	for _, tc := range []struct {
		name string
		now  int
		lead int
		want bool
	}{
		{"before lead", 702, 2, false},
		{"lead boundary (first due minute)", 703, 2, true},
		{"inside window", 720, 2, true},
		{"window end boundary", 735, 2, true},
		{"after window", 736, 2, false},
		{"lead zero is window start", 705, 0, true},
		{"lead zero before start", 704, 0, false},
	} {
		got := t1ForceFlatDue(tc.now, win, tc.lead) != ""
		if got != tc.want {
			t.Errorf("%s: due=%v want %v", tc.name, got, tc.want)
		}
	}
}

func TestT1ForceFlatDueWrapsMidnight(t *testing.T) {
	// Event at 00:05 CT: Start = 00:05 − 15m wraps to 23:50 (1430), End 20.
	// Lead 2 → due in [1428, 1439] ∪ [0, 20].
	win := []kernel.CTWindow{{Start: 1430, End: 20, Label: "NFP 00:05 CT ±15m"}}
	for _, tc := range []struct {
		now  int
		want bool
	}{
		{1427, false},
		{1428, true},
		{1439, true},
		{0, true},
		{20, true},
		{21, false},
	} {
		got := t1ForceFlatDue(tc.now, win, 2) != ""
		if got != tc.want {
			t.Errorf("wrap now=%d: due=%v want %v", tc.now, got, tc.want)
		}
	}
}

func TestT1ForceFlatDueIgnoresEmptyWindows(t *testing.T) {
	if due := t1ForceFlatDue(720, nil, 2); due != "" {
		t.Errorf("nil windows must never be due, got %q", due)
	}
}
