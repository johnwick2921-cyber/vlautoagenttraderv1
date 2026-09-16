package kernel

import (
	"strings"
	"testing"
)

// H3 — bias spelling normalizes to the set the VALIDATOR accepts.
// The dispatch asked for bearish→bear / bullish→bull; measured, that is the
// WEEKLY doc's vocabulary. A day-plan bias is validated against
// long|short|neutral (biasDirections), so aliasing to "bear" would have
// manufactured the very reject the alias exists to prevent.
func TestH3BiasAliasesTargetTheValidatorsVocabulary(t *testing.T) {
	for in, want := range map[string]string{
		"bearish": "short", "bear": "short", "down": "short", "downside": "short", "sell": "short",
		"bullish": "long", "bull": "long", "up": "long", "upside": "long", "buy": "long",
		"  BEARISH  ": "short", "Bullish": "long",
		"long": "long", "short": "short", "neutral": "neutral",
	} {
		if got := NormalizeBiasDirection(in); got != want {
			t.Errorf("NormalizeBiasDirection(%q) = %q, want %q", in, got, want)
		}
		if !biasDirections[NormalizeBiasDirection(in)] {
			t.Errorf("%q normalized to %q, which the validator REJECTS", in, NormalizeBiasDirection(in))
		}
	}
	// Unknown spellings pass through so validation still rejects them honestly.
	// "sideways" is DELIBERATELY not aliased: it is a semantic judgement, not a
	// spelling, and it is the invalid-direction sentinel in two existing
	// fixtures — aliasing it would have silently weakened both.
	for _, passthrough := range []string{"moonward", "sideways", "flat", "range"} {
		if got := NormalizeBiasDirection(passthrough); got != passthrough {
			t.Errorf("%q must pass through unchanged, got %q", passthrough, got)
		}
		if biasDirections[passthrough] {
			t.Errorf("%q must remain INVALID to the validator", passthrough)
		}
	}
	// And it runs at the chokepoint, not only as a helper.
	d := &PlanDoc{Bias: PlanBias{Direction: "bearish"}}
	NormalizePlanDocRules(d)
	if d.Bias.Direction != "short" {
		t.Errorf("the chokepoint must normalize bias, got %q", d.Bias.Direction)
	}
}

// H4 — the model's no_trade prose renders as NOTES, never as a constraint.
func TestH4NoTradeProseRendersAsNotes(t *testing.T) {
	d := &PlanDoc{
		Bias:    PlanBias{Direction: "short"},
		NoTrade: []string{"skip the 12:00-13:30 lunch chop", "no trades in the first 5m"},
	}
	out := RenderPlanBlock(*d, "ASIA")
	if !strings.Contains(out, "No-trade NOTES") {
		t.Errorf("no_trade must render as NOTES: %q", out)
	}
	if !strings.Contains(out, "NOT machine-enforced") {
		t.Error("the render must say plainly that nothing evaluates these strings")
	}
	// The bare label that read as enforcement must be gone.
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "No-trade:") {
			t.Errorf("the bare constraint-looking label survives: %q", line)
		}
	}
}
