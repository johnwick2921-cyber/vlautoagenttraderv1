// B2c — fuzz the AI-output armor: neither the parser nor the price-sanity check
// may ever panic on arbitrary / malformed / absurd input (they must fail closed
// with an error or a bool, never crash the decision cycle).
package kernel

import "testing"

// FuzzParseDecisionResponse throws arbitrary strings at the parser: it must return
// (decision, error) without panicking, no matter how malformed the "AI response".
func FuzzParseDecisionResponse(f *testing.F) {
	f.Add(`<reasoning>x</reasoning><decision>{"action":"wait"}</decision>`)
	f.Add(`{"action":"open_long","stop_loss":1,"take_profit":2}`)
	f.Add(`garbage {{{ not json`)
	f.Add(``)
	f.Add(`<decision>` + `{"action":` + `}` + `</decision>`)
	f.Fuzz(func(t *testing.T, resp string) {
		// Must not panic on any input; error is fine (that triggers retry/skip).
		_, _ = parseFullDecisionResponse(resp, 50000, 5, 5, 5, 1, 3, 65, 20)
	})
}

// FuzzPriceSanityViolation throws absurd/degenerate prices (incl. 0, negatives,
// huge) at the sanity check: it must return (reason, bool) without panicking.
func FuzzPriceSanityViolation(f *testing.F) {
	f.Add("long", 30000.0, 29900.0, 30100.0, 20.0, 30000.0)
	f.Add("short", 0.0, 0.0, 0.0, 0.0, 0.0)
	f.Add("long", 1e18, -1e18, 1e18, 1e-9, 1e-18)
	f.Fuzz(func(t *testing.T, side string, entryRef, sl, tp, atr, last float64) {
		reason, bad := priceSanityViolation(side, entryRef, sl, tp, atr, last)
		if bad && reason == "" {
			t.Fatal("a violation must always carry a non-empty reason")
		}
	})
}
