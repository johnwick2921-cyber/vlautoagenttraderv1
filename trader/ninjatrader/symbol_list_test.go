package ninjatrader

import (
	"reflect"
	"testing"
)

// TestSplitSymbolList locks the P5.2 symbols-as-list compat shim: a legacy
// single symbol is the one-element case (byte-identical behavior); a comma
// list yields primary=first + extras=rest, whitespace-tolerant.
func TestSplitSymbolList(t *testing.T) {
	cases := []struct {
		in      string
		primary string
		extras  []string
	}{
		{"MNQ", "MNQ", nil},                         // legacy single — the compat shim
		{"MNQ,ES", "MNQ", []string{"ES"}},           // list
		{" MNQ , ES , NQ ", "MNQ", []string{"ES", "NQ"}}, // whitespace
		{"MNQ,,ES", "MNQ", []string{"ES"}},          // empty element skipped
		{"", "", nil},                               // empty config (caller's guard)
	}
	for _, tc := range cases {
		p, e := SplitSymbolList(tc.in)
		if p != tc.primary || !reflect.DeepEqual(e, tc.extras) {
			t.Fatalf("SplitSymbolList(%q) = %q,%v; want %q,%v", tc.in, p, e, tc.primary, tc.extras)
		}
	}
}
