package trader

import (
	"strings"
	"testing"

	nt "nofx/provider/ninjatrader"
)

func px(v float64) *float64 { return &v }

// E5 — the reconciler's whole contract, one table.
//
// The 09-06 position ran 8h19m with no stop and nothing in this process was
// looking. Every mechanism watched orders it believed in; none asked "is there
// a position, and does the broker hold a stop for it".
func TestProtectionAdjudication(t *testing.T) {
	const sym, side = "MNQ", "LONG"
	live := nt.NT8Order{Symbol: sym, Name: "sig-sl", Action: "sell", Type: "stop",
		State: "Accepted", StopPrice: 29554, Quantity: 1}

	cases := []struct {
		name       string
		book       []nt.NT8Order
		haveBook   bool
		posQty     int
		accepted   *float64
		planStop   float64
		wantAction protectionAction
		wantSource string
		wantPx     float64
	}{
		{"protected by an Accepted stop", []nt.NT8Order{live}, true, 1, px(29554), 29500, protectionOK, "", 0},
		{"flat needs nothing", nil, true, 0, nil, 0, protectionOK, "", 0},

		{"no book is not permission", nil, false, 1, px(29554), 29500, protectionUnknown, "", 0},
		{"an unreadable protective state is not absence",
			[]nt.NT8Order{{Symbol: sym, Name: "sig-sl", Action: "sell", Type: "stop",
				State: "SomethingNT8NeverSends", Quantity: 1}},
			true, 1, px(29554), 29500, protectionUnknown, "", 0},
		{"a live stop with no quantity leaves coverage unknown",
			[]nt.NT8Order{{Symbol: sym, Name: "sig-sl", Action: "sell", Type: "stop",
				State: "Working", Quantity: 0}},
			true, 1, px(29554), 29500, protectionUnknown, "", 0},

		// THE ACTED-ON CASE
		{"nothing protective → place the ACCEPTED stop",
			[]nt.NT8Order{{Symbol: sym, Name: "sig", Action: "buy", Type: "limit", State: "Working", Quantity: 1}},
			true, 1, px(29554), 29500, protectionPlace, "accepted_risk", 29554},
		{"nothing protective and no accepted record → the plan's stop",
			nil, true, 1, nil, 29500, protectionPlace, "plan", 29500},
		{"nothing protective and no price at all → raised, never invented",
			nil, true, 1, nil, 0, protectionAlertOnly, "", 0},

		// A DYING OR LOCAL STOP IS NOT PROTECTION
		{"a CancelPending stop is not protection",
			[]nt.NT8Order{{Symbol: sym, Name: "sig-sl", Action: "sell", Type: "stop",
				State: "CancelPending", Quantity: 1}},
			true, 1, px(29554), 0, protectionPlace, "accepted_risk", 29554},
		{"a TriggerPending stop is held on this PC, not at the exchange",
			[]nt.NT8Order{{Symbol: sym, Name: "sig-sl", Action: "sell", Type: "stop",
				State: "TriggerPending", Quantity: 1}},
			true, 1, px(29554), 0, protectionPlace, "accepted_risk", 29554},

		// PARTIAL COVER IS REPORTED, NOT PATCHED
		{"partially covered is raised, not doubled",
			[]nt.NT8Order{live}, true, 2, px(29554), 29500, protectionAlertOnly, "", 0},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			v := adjudicateProtection(sym, side, c.posQty, c.book, c.haveBook, c.accepted, c.planStop)
			if v.Action != c.wantAction {
				t.Fatalf("action = %s, want %s (why: %s)", v.Action, c.wantAction, v.Why)
			}
			if c.wantSource != "" && v.Source != c.wantSource {
				t.Fatalf("source = %q, want %q", v.Source, c.wantSource)
			}
			if c.wantPx != 0 && v.StopPx != c.wantPx {
				t.Fatalf("stop = %.2f, want %.2f", v.StopPx, c.wantPx)
			}
			if v.Why == "" {
				t.Fatal("every verdict must say why — this one is silent")
			}
		})
	}
}

// A short position is protected by a BUY stop; the side logic must not be
// long-only. (The 592 incident was long, which is exactly how a long-only bug
// would have survived review.)
func TestShortPositionProtectionIsRecognised(t *testing.T) {
	book := []nt.NT8Order{{Symbol: "MNQ", Name: "whatever", Action: "BuyToCover", Type: "StopMarket",
		State: "Accepted", StopPrice: 29700, Quantity: 1}}
	if v := adjudicateProtection("MNQ", "SHORT", 1, book, true, nil, 0); v.Action != protectionOK {
		t.Fatalf("a buy stop above a SHORT is its protection; got %s — %s", v.Action, v.Why)
	}
	// the same order does NOT protect a long
	if v := adjudicateProtection("MNQ", "LONG", 1, book, true, nil, 29500); v.Action != protectionPlace {
		t.Fatalf("a BUY stop was counted as protection for a LONG; got %s", v.Action)
	}
}

// D7 — the boot line says what it KNOWS and n/a for what it cannot.
func TestBracketsBootLineReadsRatherThanAsserts(t *testing.T) {
	// At process start there is no book and no far-side build id.
	cold := BracketsBootLine(nil, false, "", 0)
	for _, want := range []string{"entry-oco=n/a", "bracket-oco=n/a", "state-source=none", "protective-tif=n/a"} {
		if !strings.Contains(cold, want) {
			t.Errorf("cold boot line is missing %q — a field this process cannot know must print n/a, "+
				"never a constant dressed as a measurement.\n  got: %s", want, cold)
		}
	}
	if strings.Contains(cold, "can-place-stop=yes") {
		t.Errorf("an unknown AddOn build was reported as capable:\n  %s", cold)
	}

	// With a book carrying the shape this wave creates.
	book := []nt.NT8Order{
		{Name: "aa07e583", OCO: "", State: "Working", TimeInForce: "Day"},
		{Name: "aa07e583-sl", OCO: "aa07e583-exit", State: "Accepted", TimeInForce: "Gtc"},
		{Name: "aa07e583-tp", OCO: "aa07e583-exit", State: "Working", TimeInForce: "Gtc"},
	}
	warm := BracketsBootLine(book, true, "2026-09-07-h1", 3)
	for _, want := range []string{
		"entry-oco=own(none)", "bracket-oco=on-fill(shared)", "state-source=broker",
		"protective-tif=Gtc", "can-place-stop=yes", "unprotected-found=3",
	} {
		if !strings.Contains(warm, want) {
			t.Errorf("warm boot line is missing %q\n  got: %s", want, warm)
		}
	}

	// THE REGRESSION THIS LINE EXISTS TO SHOW: an entry back in the bracket's
	// group must be visible on sight, not inferred from an incident report.
	bad := []nt.NT8Order{
		{Name: "aa07e583", OCO: "aa07e583-exit", State: "Working"},
		{Name: "aa07e583-sl", OCO: "aa07e583-exit", State: "Accepted", TimeInForce: "Gtc"},
	}
	if got := BracketsBootLine(bad, true, "2026-09-07-h1", 0); !strings.Contains(got, "entry-oco=SHARED(") {
		t.Errorf("an entry sharing its bracket's OCO group did not show as SHARED:\n  %s", got)
	}
}
