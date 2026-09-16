package api

import (
	"strings"
	"testing"

	"nofx/kernel"
)

// ITEM 15 — the plain-language diff the owner reads to understand WHY a re-plan
// happened. The v5 → v6 transition of 2026-08-16:ASIA is the case that matters:
// six good levels replaced by a levels:null NO-TRADE plan.

func doc(bias string, levels []kernel.PlanLevel, scenarios int) *kernel.PlanDoc {
	d := &kernel.PlanDoc{
		Bias:   kernel.PlanBias{Direction: bias, Conviction: "medium"},
		Levels: levels,
	}
	for i := 0; i < scenarios; i++ {
		d.Scenarios = append(d.Scenarios, kernel.PlanScenario{ID: "S", Quality: "A"})
	}
	return d
}

func TestPlanDocDiffNamesTheDisappearanceOfEveryLevel(t *testing.T) {
	v5 := doc("long", []kernel.PlanLevel{
		{Price: 30199.5, Label: "EQL"}, {Price: 30203, Label: "ONH"},
		{Price: 30166.25, Label: "ONL"}, {Price: 30146.75, Label: "PDL"},
	}, 3)
	v6 := doc("neutral", nil, 1) // NoTradePlanDoc

	got := strings.Join(planDocDiff(v5, v6), " | ")
	for _, want := range []string{"bias long → neutral", "NO LEVELS in the replacement", "scenarios 3 → 1"} {
		if !strings.Contains(got, want) {
			t.Errorf("diff %q is missing %q — the owner must be able to see what the re-plan cost", got, want)
		}
	}
	if !strings.Contains(got, "dropped") {
		t.Errorf("diff %q never says the levels were dropped", got)
	}
}

func TestPlanDocDiffReportsAddedKeptAndDropped(t *testing.T) {
	from := doc("long", []kernel.PlanLevel{{Price: 30200, Label: "ONH"}, {Price: 30150, Label: "PDL"}}, 2)
	to := doc("long", []kernel.PlanLevel{{Price: 30200, Label: "ONH"}, {Price: 30250, Label: "PDH"}}, 2)

	got := strings.Join(planDocDiff(from, to), " | ")
	if !strings.Contains(got, "added PDH 30250") {
		t.Errorf("diff %q must name the added level", got)
	}
	if !strings.Contains(got, "dropped PDL 30150") {
		t.Errorf("diff %q must name the dropped level", got)
	}
	if strings.Contains(got, "bias") {
		t.Errorf("diff %q reports a bias change that did not happen", got)
	}
}

func TestPlanDocDiffOnAnUnchangedLevelSet(t *testing.T) {
	same := []kernel.PlanLevel{{Price: 30200, Label: "ONH"}}
	got := strings.Join(planDocDiff(doc("long", same, 2), doc("long", same, 2)), " | ")
	if !strings.Contains(got, "same 1 levels") {
		t.Errorf("an unchanged level set must say so, got %q", got)
	}
}

func TestPlanDocDiffToleratesUnparsableVersions(t *testing.T) {
	// A version whose doc failed to unmarshal must not crash the endpoint.
	if planDocDiff(nil, doc("long", nil, 0)) != nil || planDocDiff(doc("long", nil, 0), nil) != nil {
		t.Error("a nil doc must yield a nil diff, never a panic or a fabricated change")
	}
}
