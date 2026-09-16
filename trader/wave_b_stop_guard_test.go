// WAVE B (2026-09-05) — the stop-side marketable guard.
//
// RED-FIRST NOTE: this table is written against the CORRECT stop-entry
// semantics. In its first form it called the production expression as it stood
// at cb23d9eb — limitMarketableWrongSide(price, trigger, side) — and every one
// of the four substantive cases inverted. See the report's RED quote.

package trader

import (
	"strings"
	"testing"
)

// stopGuardCase is one (side, trigger, price) with the answer a RESTING stop
// entry must give: true = the market is already at/through the trigger, so the
// order would fire the instant it reached NT8 and must never be placed.
type stopGuardCase struct {
	name    string
	side    string
	trigger float64
	price   float64
	through bool
}

// stopGuardCases — the four cells, plus the inclusive boundary on both sides.
// The stop boundary is INCLUSIVE (price == trigger fires) where the limit
// boundary is strict (price == entry still rests); the exact-touch pair pins
// that difference so a strict mirror of the limit predicate cannot pass.
var stopGuardCases = []stopGuardCase{
	// (1) BUY STOP, price BELOW the trigger — the valid resting state.
	{"buy stop rests below", "long", 29610.00, 29590.25, false},
	// (2) BUY STOP, price AT/ABOVE the trigger — already through.
	{"buy stop through above", "long", 29610.00, 29612.25, true},
	{"buy stop exact touch", "long", 29610.00, 29610.00, true},
	// (3) SELL STOP, price ABOVE the trigger — the valid resting state.
	{"sell stop rests above", "short", 29590.50, 29650.00, false},
	// (4) SELL STOP, price AT/BELOW the trigger — already through.
	//     These are the real 2026-09-04 10:05:00 CT numbers (armed_orders id 38).
	{"sell stop through below (09-04 id 38)", "short", 29590.50, 29515.25, true},
	{"sell stop exact touch", "short", 29590.50, 29590.50, true},
	// Case folding + surrounding space, same hygiene as the limit predicate.
	{"case folded long", "  LONG ", 29610.00, 29612.25, true},
	{"case folded short", "Short", 29590.50, 29515.25, true},
}

// TestStopEntryMarketableWrongSide — E3. All four cells of the stop-side guard.
func TestStopEntryMarketableWrongSide(t *testing.T) {
	// A table with no length assertion is a test that passes when it is emptied
	// (the sibling replay pins its own 21; this one did not pin its 8).
	if len(stopGuardCases) != 8 {
		t.Fatalf("the four cells plus both boundaries plus both case-folds = 8 cases, got %d", len(stopGuardCases))
	}
	for _, c := range stopGuardCases {
		got := stopEntryMarketableWrongSide(c.side, c.trigger, c.price)
		if got != c.through {
			t.Errorf("%s: side=%q trigger=%.2f price=%.2f: got through=%v want %v",
				c.name, c.side, c.trigger, c.price, got, c.through)
		}
	}
}

// TestStopEntryGuardUnknownNeverCancels — E5. A guard that cannot be evaluated
// must return UNKNOWN, and UNKNOWN must not be the destructive answer.
func TestStopEntryGuardUnknownNeverCancels(t *testing.T) {
	for _, c := range []struct {
		name    string
		side    string
		trigger float64
		price   float64
	}{
		{"no price", "long", 29610.00, 0},
		{"no trigger", "long", 0, 29612.25},
		{"neither", "short", 0, 0},
		{"unknown side", "sideways", 29610.00, 29612.25},
		{"empty side", "", 29610.00, 29612.25},
	} {
		v, why := stopEntryGuardVerdict(c.side, c.trigger, c.price)
		if v != stopGuardUnknown {
			t.Errorf("%s: verdict=%v, want unknown", c.name, v)
		}
		if strings.TrimSpace(why) == "" {
			t.Errorf("%s: unknown verdict carries no reason — A9 requires it be sayable", c.name)
		}
	}
	// The zero value of the enum is UNKNOWN on purpose: a forgotten verdict
	// reads as the safe branch, never the cancel.
	var zero stopGuardVerdict
	if zero != stopGuardUnknown {
		t.Fatalf("the zero verdict must be unknown, got %v", zero)
	}
	if stopGuardUnknown.String() != "unknown" {
		t.Fatalf("unknown must say so: %q", stopGuardUnknown.String())
	}
}

// TestStopEntryGuardVerdictReasonNamesTheGuard — A9: the refusal names which
// guard ran and the relation it actually evaluated.
func TestStopEntryGuardVerdictReasonNamesTheGuard(t *testing.T) {
	v, why := stopEntryGuardVerdict("long", 29610.00, 29612.25)
	if v != stopGuardThrough {
		t.Fatalf("long price above trigger must be through, got %v", v)
	}
	for _, want := range []string{"accepted through (stop side)", "price 29612.25", ">=", "trigger 29610.00"} {
		if !strings.Contains(why, want) {
			t.Fatalf("reason %q missing %q", why, want)
		}
	}
	v2, why2 := stopEntryGuardVerdict("short", 29590.50, 29515.25)
	if v2 != stopGuardThrough {
		t.Fatalf("short price below trigger must be through, got %v", v2)
	}
	if !strings.Contains(why2, "<=") {
		t.Fatalf("short reason must name the <= relation it evaluated: %q", why2)
	}
}
