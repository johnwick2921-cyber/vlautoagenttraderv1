package trader

import (
	"fmt"
	"strings"
	"testing"

	"nofx/kernel"
	"nofx/mcp"
)

func TestResolvePlannerModelID(t *testing.T) {
	// Empty binding → primary.
	if id, prim := resolvePlannerModelID("", "deepseek"); id != "deepseek" || !prim {
		t.Fatalf("empty → primary: got %q/%v", id, prim)
	}
	if id, prim := resolvePlannerModelID("   ", "deepseek"); id != "deepseek" || !prim {
		t.Fatalf("whitespace → primary: got %q/%v", id, prim)
	}
	// Explicit binding → the pinned ID, not the primary.
	if id, prim := resolvePlannerModelID("deepseek-reasoner", "deepseek"); id != "deepseek-reasoner" || prim {
		t.Fatalf("pinned: got %q/%v want deepseek-reasoner/false", id, prim)
	}
}

func TestResolvePlannerClientEmptyUsesPrimary(t *testing.T) {
	// W2 — empty planner_model → primary client, but the stamped id is the EXACT
	// model, NEVER the provider alias (§125). "deepseek" pins to "deepseek-v4-pro".
	at := mkTrader("ninjatrader", boolp(true), "5m")
	at.aiModel = "deepseek"
	_, id := at.resolvePlannerClient()
	if mcp.IsProviderAlias(id) || id == "" {
		t.Fatalf("empty binding must resolve an EXACT model id (not an alias), got %q", id)
	}
	if id != mcp.DefaultModelForAlias("deepseek") {
		t.Fatalf("primary 'deepseek' should pin to %q, got %q", mcp.DefaultModelForAlias("deepseek"), id)
	}
	// A set planner_model resolves an exact, non-empty id too.
	at.config.StrategyConfig.DayPlan.PlannerModel = "deepseek"
	_, id2 := at.resolvePlannerClient()
	if id2 == "" || mcp.IsProviderAlias(id2) {
		t.Fatalf("resolved planner model id must be an exact non-empty string, got %q", id2)
	}
}

// TestPlannerRejectBookkeepingRewritesPrevReason (WAVE 1a-plan P9, #190) — the
// mechanism that makes the recordRepairOutcome ORDER observable: bookkeeping
// rewrites the reason pointer to THIS attempt's defect. The A1/A2 sites used to
// record BEFORE this rewrite, so "was repairing" logged the PREVIOUS attempt's
// reason while every sibling site logged the current one.
func TestPlannerRejectBookkeepingRewritesPrevReason(t *testing.T) {
	yes := true
	at := mkTrader("ninjatrader", &yes, "5m")
	prev := "schema: missing condition (attempt 1 defect)"
	at.plannerRejectBookkeeping(2, "2026-09-23", "NY", "h", "prompt", "fragment raw", fmt.Errorf("fragment: partial plan document"), &prev)
	if prev != "fragment: partial plan document" {
		t.Fatalf("bookkeeping must rewrite prevReason to this attempt's defect, got %q", prev)
	}
	// The mutation feeds the whack-a-mole comparison on the NEXT attempt.
	if !samePlannerDefect(prev, "fragment: partial plan document") {
		t.Fatalf("same-defect comparison broken after rewrite: %q", prev)
	}
}

// TestRepairWasRepairingNamesThePreviousDefect (skeptic F5, 2026-09-24) — the
// call-site ORDER the pointer-rewrite pin cannot see: attempt 1 is rejected
// with D1 (bias.direction "sideways" invalid), the attempt-2 repair returns a
// fragment, and the 🩹 repair-outcome line's "was repairing" must quote D1 —
// the defect the repair was AIMED at — not repeat the FragmentReason. RED =
// record AFTER bookkeeping (the P9 inversion): the line repeats the fragment
// reason twice.
func TestRepairWasRepairingNamesThePreviousDefect(t *testing.T) {
	at := feasPlannerTrader(t, nil)
	feasStubBars(t)
	logBuf := captureTraderLog(t)
	block := []string{}
	_, _, err := at.runPlannerReadCoreWithFactsGradesClock(feasClock(), "NY", "2026-08-14", "owner_reset",
		"deepseek-v4-pro", "hashF5", "", "", "", "FULLPROMPT",
		kernel.PlanFacts{Price: 15550, DATR: 300}, nil, map[float64]string{}, nil, true,
		func(userPrompt string) (string, error) {
			block = append(block, userPrompt)
			if len(block) == 1 {
				// D1: parses far enough to reject on bias.direction.
				return `{"reasoning":"sideways probe","bias":{"direction":"sideways"},"death_condition":"x","scenarios":[{"id":"S1","trigger":"t","condition":"reclaim","direction":"long","target_chain":[1],"quality":"A"}]}`, nil
			}
			return `{"reasoning":"fragment only"}`, nil // IsPlanFragment → the fragment site
		})
	_ = err // the chain's own terminal state is not what this pin asserts
	if len(block) != 3 {
		t.Fatalf("the chain must make 3 attempts (reject → repair-fragment → reauthor), got %d", len(block))
	}
	wantD1 := `was repairing: bias.direction "sideways" invalid`
	if !strings.Contains(logBuf.String(), wantD1) {
		t.Fatalf("'was repairing' must name the defect the repair was AIMED at (attempt 1's reject), missing %q; log:\n%s", wantD1, logBuf.String())
	}
	// The inverted order prints the fragment reason in BOTH fields — the
	// defect being repaired disappears from the line.
	if strings.Contains(logBuf.String(), "was repairing: repair returned a fragment") {
		t.Fatalf("'was repairing' must not repeat this attempt's fragment reason; log:\n%s", logBuf.String())
	}
}
