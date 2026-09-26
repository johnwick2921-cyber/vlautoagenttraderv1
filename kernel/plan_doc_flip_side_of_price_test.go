package kernel

import (
	"strings"
	"testing"
)

// W-FLIP-LINE-SIDE-OF-PRICE (2026-09-17) — A FLIP LINE ON THE WRONG SIDE OF
// PRICE.
//
// Evidence [A] (plans table, read-only, 2026-09-17): plan 2026-09-17:ASIA v2,
// created 22:52:04 (structure_mss wake), bias.direction="short",
// flip={29747.50, side "above", rule "5m_close", flip_to "long"},
// death={29755.50, side "above", rule "2x5m"}; the authoring price was
// 29764. Both lines sat BELOW price with side "above". CLASS 140's direction
// check passed (short → flip long on a close ABOVE is the right side), but the
// P1c touch gate (PlanConditionFiredSince) fires a line only after price
// touches it from the near side after birth — a line already beyond price on
// its own side can never be touched from the near side, so the plan could
// never flip by construction. v3 at 23:18 repaired it to flip{29769 above}.
// These cases run at the production call site (ValidatePlanDocWithFactsMachine,
// the write-site validator).

const asiaV2Price = 29764.0

// asiaV2Doc mirrors the 2026-09-17 ASIA v2 shape with the death line kept on
// the far side (v1/v3's 29797.88) so ONLY the flip line is under test.
func asiaV2Doc() *PlanDoc {
	return &PlanDoc{
		Reasoning:       "r",
		Bias:            PlanBias{Direction: "short", FlipCondition: "5m close above 29747.50 flips long"},
		Levels:          []PlanLevel{{Price: 29797.88, Label: "ONH", Grade: "A"}, {Price: 29700, Label: "PDL", Grade: "A"}},
		Scenarios:       []PlanScenario{{ID: "S1", Trigger: "reject at 29797.88", Condition: "reject", Direction: "short", TargetChain: []float64{29700}, Invalid: "5m close above 29810.00", Quality: "B"}},
		NoTrade:         []string{"lunch"},
		DeathCondition:  "2x5m close above 29797.88 kills the plan",
		DeathStructured: &PlanCondition{Price: 29797.88, Side: "above", Rule: "2x5m"},
		FlipStructured:  &PlanCondition{Price: 29747.5, Side: "above", Rule: "5m_close", FlipTo: "long"},
	}
}

func asiaV2Facts() PlanFacts { return PlanFacts{Price: asiaV2Price, DATR: 150} }

func TestFlipLineBeyondPrice_ASIAv2ShapeRejected(t *testing.T) {
	d := asiaV2Doc()
	err := ValidatePlanDocWithFactsMachine(d, asiaV2Facts(), nil, 8, 3)
	if err == nil {
		t.Fatalf("short bias + flip{29747.50 above → long} with price 29764 must be REJECTED: the line is below price on side above")
	}
	want := "flip{above 29747.50 → long} is already below price 29764.00 at authoring: a flip line must sit on the far side of price (it can never be touched from the near side)"
	if err.Error() != want {
		t.Fatalf("the rejection must be the validator's sentence\n got: %s\nwant: %s", err, want)
	}
}

func TestFlipLineBeyondPrice_ASIAv3RepairAccepted(t *testing.T) {
	d := asiaV2Doc()
	d.FlipStructured.Price = 29769 // the v3 repair: above price on side above
	d.Bias.FlipCondition = "5m close above 29769.00 flips long"
	if err := ValidatePlanDocWithFactsMachine(d, asiaV2Facts(), nil, 8, 3); err != nil {
		t.Fatalf("a flip line above price with side above is the correct shape and must pass: %v", err)
	}
}

// Mirror: a long bias flips short on a close BELOW the line; the line must
// therefore sit BELOW price.
func TestFlipLineBeyondPrice_BelowSideMirror(t *testing.T) {
	d := validLongBaseDoc()                     // long, flip{29441 below → short}, death{29648.25 above}
	facts := PlanFacts{Price: 29400, DATR: 150} // price BELOW the flip line
	err := ValidatePlanDocWithFactsMachine(d, facts, nil, 8, 3)
	if err == nil || !strings.Contains(err.Error(), "flip{below 29441.00 → short} is already above price 29400.00 at authoring") {
		t.Fatalf("long bias + flip below with the line above price must be REJECTED with the side-of-price sentence (got %v)", err)
	}
	facts.Price = 29500 // price between the two lines: flip below, death above
	if err := ValidatePlanDocWithFactsMachine(d, facts, nil, 8, 3); err != nil {
		t.Fatalf("flip below price with side below must pass: %v", err)
	}
}

// Unknown authoring price (facts absent) → the rule is SKIPPED, never a
// reject with an invented price.
func TestFlipLineBeyondPrice_UnknownPriceNeverRejects(t *testing.T) {
	d := asiaV2Doc()
	if err := ValidatePlanDocWithFactsMachine(d, PlanFacts{}, nil, 8, 3); err != nil {
		t.Fatalf("with no authoring price the side-of-price rule must not fire: %v", err)
	}
	if err := FlipLineBeyondPrice(d.FlipStructured, 0); err != nil {
		t.Fatalf("price 0 is UNKNOWN, not a price: %v", err)
	}
	if err := FlipLineBeyondPrice(nil, asiaV2Price); err != nil {
		t.Fatalf("no structured flip → no-op: %v", err)
	}
}

func TestFlipLineBeyondPrice_LineAtPriceRejected(t *testing.T) {
	err := FlipLineBeyondPrice(&PlanCondition{Price: asiaV2Price, Side: "above", FlipTo: "long"}, asiaV2Price)
	if err == nil || !strings.Contains(err.Error(), "is already at price 29764.00 at authoring") {
		t.Fatalf("a line AT price is not on the far side (got %v)", err)
	}
	// Empty flip_to is printed without an arrow, never invented.
	err = FlipLineBeyondPrice(&PlanCondition{Price: 29747.5, Side: "above"}, asiaV2Price)
	if err == nil || !strings.HasPrefix(err.Error(), "flip{above 29747.50} is already below price") {
		t.Fatalf("empty flip_to must print without a destination (got %v)", err)
	}
}

// The death line obeys the same law: before W2 D5 the scenario born-dead check
// (validateAuthoredScenariosAt) read ONLY scenario.invalid prose and never
// the death object, so the ASIA v2 death{29755.50 above} at price 29764 was
// a plan born dead that nothing refused.
func TestDeathLineBeyondPrice_ASIAv2DeathRejected(t *testing.T) {
	d := asiaV2Doc()
	d.FlipStructured.Price = 29769 // flip repaired so ONLY the death line is under test
	d.Bias.FlipCondition = "5m close above 29769.00 flips long"
	d.DeathStructured = &PlanCondition{Price: 29755.5, Side: "above", Rule: "2x5m"}
	d.DeathCondition = "2x5m close above 29755.50 kills the plan"
	err := ValidatePlanDocWithFactsMachine(d, asiaV2Facts(), nil, 8, 3)
	want := "death{above 29755.50} is already below price 29764.00 at authoring: a death line must sit on the far side of price (the plan would be born dead)"
	if err == nil || err.Error() != want {
		t.Fatalf("a death line already crossed at authoring must be REJECTED\n got: %v\nwant: %s", err, want)
	}
	if err := DeathLineBeyondPrice(d.DeathStructured, 0); err != nil {
		t.Fatalf("unknown price never rejects the death line: %v", err)
	}
	if err := DeathLineBeyondPrice(nil, asiaV2Price); err != nil {
		t.Fatalf("no structured death → no-op: %v", err)
	}
}

// The flip line is judged BEFORE the death line, so the ASIA v2 doc (both
// wrong) names the flip first — the owner's complaint.
func TestFlipLineBeyondPrice_FlipNamedBeforeDeath(t *testing.T) {
	d := asiaV2Doc()
	d.DeathStructured = &PlanCondition{Price: 29755.5, Side: "above", Rule: "2x5m"}
	d.DeathCondition = "2x5m close above 29755.50 kills the plan"
	err := ValidatePlanDocWithFactsMachine(d, asiaV2Facts(), nil, 8, 3)
	if err == nil || !strings.HasPrefix(err.Error(), "flip{above 29747.50 → long}") {
		t.Fatalf("the flip line must be named first (got %v)", err)
	}
}

// The direction rule (CLASS 140) and the side-of-price rule are siblings that
// ask different questions: a correctly-directed flip on the wrong side of
// price is THIS rule's reject, and the direction rule stays silent on it.
func TestFlipLineBeyondPrice_SiblingOfDirectionRule(t *testing.T) {
	d := asiaV2Doc()
	if err := FlipDirectionContradiction(d.Bias.Direction, d.FlipStructured); err != nil {
		t.Fatalf("ASIA v2's flip points the right way for its bias; the direction rule must be silent: %v", err)
	}
	if err := ValidatePlanDocWithCaps(d, 8, 3); err != nil {
		t.Fatalf("the schema validator (no facts) must still accept the shape: %v", err)
	}
}

// Class-38 style: the law the validator judges by must reach the model — the
// prompt-contract row, the rendered sentence in EVERY plannerOutputContract
// variant, and the repair excerpt routed from BOTH rejection texts.
func TestFlipSideOfPriceLawReachesTheModel(t *testing.T) {
	found := false
	for _, c := range PromptContracts() {
		if strings.Contains(c.Rule, "far side of price") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the prompt-contract registry must carry the side-of-price restriction")
	}
	for _, v := range []struct {
		hasHTF, has1H bool
	}{{true, true}, {true, false}, {false, true}, {false, false}} {
		for _, caps := range [][2]int{{0, 0}, {8, 3}, {8, 5}} {
			p := plannerOutputContract(caps[0], caps[1], v.hasHTF, v.has1H, false)
			if err := ValidatePromptContracts(p); err != nil {
				t.Fatalf("rendered prompt (levels=%d scenarios=%d htf=%v 1h=%v) must state every restriction: %v", caps[0], caps[1], v.hasHTF, v.has1H, err)
			}
			if !strings.Contains(p, "a flip line must sit on the far side of price") {
				t.Fatalf("rendered prompt variant htf=%v 1h=%v must carry the side-of-price sentence", v.hasHTF, v.has1H)
			}
		}
	}
	flipErr := "flip{above 29747.50 → long} is already below price 29764.00 at authoring: a flip line must sit on the far side of price (it can never be touched from the near side)"
	if got := lawExcerptsFor(flipErr); !strings.Contains(got, RepairFlipSideOfPriceLaw) {
		t.Fatalf("repair excerpt for the flip side-of-price error must carry RepairFlipSideOfPriceLaw, got %q", got)
	}
	deathErr := "death{above 29755.50} is already below price 29764.00 at authoring: a death line must sit on the far side of price (the plan would be born dead)"
	if got := lawExcerptsFor(deathErr); !strings.Contains(got, RepairFlipSideOfPriceLaw) {
		t.Fatalf("repair excerpt for the death side-of-price error must carry RepairFlipSideOfPriceLaw, got %q", got)
	}
	// The direction error must NOT pull the side-of-price excerpt (different question).
	dirErr := "flip{below 29474.90 → long} contradicts bias short: a short bias flips to long only on a close above the line"
	if got := lawExcerptsFor(dirErr); strings.Contains(got, RepairFlipSideOfPriceLaw) {
		t.Fatalf("the direction error must not route the side-of-price excerpt, got %q", got)
	}
}
