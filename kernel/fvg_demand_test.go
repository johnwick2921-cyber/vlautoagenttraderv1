package kernel

import "testing"

// GAR-F5 (grand-audit response, 2026-08-28) — FVG demand compliance, WARN-only.
// Supply exists (6–8 fresh machine gaps/5d) but the model authored ZERO
// fvg_entry scenarios; the demand makes the omission visible at write.
func TestFvgDemandWarnings(t *testing.T) {
	fresh := []FreshFvg{{Direction: "long", Lo: 29620, Hi: 29624, AgeBars: 2, DispATR: 1.6}}

	// Bias agrees with the fresh gap and nothing was authored → WARN.
	doc := PlanDoc{Bias: PlanBias{Direction: "long"}}
	if w := FvgDemandWarnings(doc, fresh); len(w) != 1 {
		t.Fatalf("biased no-author must warn once, got %v", w)
	}

	// An fvg_entry IS authored → compliant.
	doc.Scenarios = []PlanScenario{{ID: "S1", Condition: "fvg_entry", Direction: "long"}}
	if w := FvgDemandWarnings(doc, fresh); len(w) != 0 {
		t.Fatalf("authored fvg_entry must not warn, got %v", w)
	}

	// Bias opposes the only fresh gap → no demand.
	doc2 := PlanDoc{Bias: PlanBias{Direction: "short"}}
	if w := FvgDemandWarnings(doc2, fresh); len(w) != 0 {
		t.Fatalf("against-bias must not warn, got %v", w)
	}

	// Neutral bias: any fresh gap triggers the demand.
	doc3 := PlanDoc{Bias: PlanBias{Direction: "neutral"}}
	if w := FvgDemandWarnings(doc3, fresh); len(w) != 1 {
		t.Fatalf("neutral bias with fresh gaps must warn, got %v", w)
	}

	// No fresh gaps → nothing to demand.
	if w := FvgDemandWarnings(doc, nil); len(w) != 0 {
		t.Fatalf("empty fresh list must not warn, got %v", w)
	}
}
