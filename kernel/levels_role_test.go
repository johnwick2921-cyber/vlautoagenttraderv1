package kernel

import "testing"

// PRE-REOPEN F5a (2026-08-28) — SWG levels now bucket into their own
// confluence families instead of falling through to the inert KindRound.
func TestKindForLabelSWG(t *testing.T) {
	cases := []struct {
		label string
		want  LevelKind
	}{
		{"SWG-H · 5m", KindSWGH},
		{"SWG-L · 15m", KindSWGL},
		{"swg-h 5m", KindSWGH}, // lower-case input is normalized
		{"SWG-H", KindSWGH},
	}
	for _, tc := range cases {
		if got := KindForLabel(tc.label); got != tc.want {
			t.Fatalf("KindForLabel(%q) = %v, want %v", tc.label, got, tc.want)
		}
	}
	// Unknown still lands on the inert default.
	if got := KindForLabel("NOPE"); got != KindRound {
		t.Fatalf("unknown label got %v, want KindRound", got)
	}
}
