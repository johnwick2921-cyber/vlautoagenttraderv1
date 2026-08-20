package kernel

import (
	"strings"
	"testing"

	"nofx/market"
)

func confirmBars(base int64, closes ...float64) []market.Kline {
	var out []market.Kline
	for i, cl := range closes {
		o := base + int64(i)*60_000
		out = append(out, market.Kline{OpenTime: o, CloseTime: o + 59_999, Open: cl, High: cl + 1, Low: cl - 1, Close: cl})
	}
	return out
}

// C1 — authored rule == evaluated rule (with A2's semantics): 1x5m_close needs
// ONE 5m bucket beyond; 2x5m_close needs two; 15m_close one 15m bucket.
func TestConfirmRuleIdentity(t *testing.T) {
	base := int64(1_700_000_100_000)
	base -= base % 900_000 // 15m-aligned
	// ten 1m bars: first 5m bucket closes below 100, second closes above.
	bars := confirmBars(base, 99, 99, 99, 99, 99, 101, 101, 101, 101, 101)
	oneBelow := PlanConfirm{Rule: "1x5m_close", RefPrice: 100, Side: "below"}
	if v := EvaluateConfirm(oneBelow, bars, base-1, base+10*60_000); !v.Met {
		t.Fatalf("1x5m_close below: one 5m close beyond must be MET (%s)", v.Detail)
	}
	twoBelow := PlanConfirm{Rule: "2x5m_close", RefPrice: 100, Side: "below"}
	if v := EvaluateConfirm(twoBelow, bars, base-1, base+10*60_000); v.Met {
		t.Fatalf("2x5m_close below: ONE close must NOT satisfy two (%s)", v.Detail)
	}
	touch := PlanConfirm{Rule: "touch", RefPrice: 100, Side: "below"}
	if v := EvaluateConfirm(touch, bars, base-1, base+10*60_000); !v.Met {
		t.Fatalf("touch: highs/lows straddle 100 (%s)", v.Detail)
	}
}

// The MET detail line carries the last rule-TF close (the executor-prompt fact).
func TestConfirmDetailCarriesLastClose(t *testing.T) {
	base := int64(1_700_003_600_000)
	base -= base % 900_000
	bars := confirmBars(base, 99, 99, 99, 99, 99)
	v := EvaluateConfirm(PlanConfirm{Rule: "1x5m_close", RefPrice: 100, Side: "below"}, bars, base-1, base+5*60_000)
	if !strings.Contains(v.Detail, "close") {
		t.Fatalf("detail must state the last close: %q", v.Detail)
	}
}

// Validator: confirm enum + object↔prose number agreement (the A3 contract).
func TestConfirmValidator(t *testing.T) {
	d := validBaseDoc()
	d.Scenarios[0].Confirm = &PlanConfirm{Rule: "15m_close", RefPrice: 29648.25, Side: "above"}
	if err := ValidatePlanDocWithCaps(d, 8, 3); err != nil {
		t.Fatalf("coherent confirm must pass: %v", err)
	}
	d.Scenarios[0].Confirm.Rule = "3x1m"
	if err := ValidatePlanDocWithCaps(d, 8, 3); err == nil || !strings.Contains(err.Error(), "confirm.rule") {
		t.Fatalf("bad rule must fail (got %v)", err)
	}
	d.Scenarios[0].Confirm.Rule = "15m_close"
	d.Scenarios[0].Confirm.RefPrice = 29000 // not in the prose
	if err := ValidatePlanDocWithCaps(d, 8, 3); err == nil || !strings.Contains(err.Error(), "confirm.ref_price") {
		t.Fatalf("prose mismatch must fail (got %v)", err)
	}
}

// The prompt advisory renders one line per confirmed scenario, none when absent.
func TestRenderConfirmLines(t *testing.T) {
	doc := *validBaseDoc()
	if RenderConfirmLines(doc, nil, 0, 0) != "" {
		t.Fatal("no confirm objects → no advisory block")
	}
	doc.Scenarios[0].Confirm = &PlanConfirm{Rule: "15m_close", RefPrice: 29648.25, Side: "above"}
	out := RenderConfirmLines(doc, confirmBars(1_700_000_100_000-(1_700_000_100_000%900_000), 99, 99, 99), 1, 1_700_000_400_000)
	if !strings.Contains(out, "S1 confirm:") || !strings.Contains(out, "advisory") {
		t.Fatalf("advisory block malformed:\n%s", out)
	}
}
