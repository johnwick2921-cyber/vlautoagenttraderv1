package kernel

import (
	"os"
	"strings"
	"testing"

	"nofx/store"
)

const futuresPlanGolden = "testdata/futures_mnq_plan.golden"

func samplePlanDoc() PlanDoc {
	return PlanDoc{
		Reasoning:      "balance below PDH; fade edges, long the reclaim",
		Bias:           PlanBias{Direction: "long", Conviction: "medium", FlipCondition: "2x5m < 21480"},
		Levels:         []PlanLevel{{Price: 21520, Label: "PDH", Grade: "A", Instruction: "fade"}, {Price: 21480, Label: "ONL", Grade: "B", Instruction: "reclaim-long"}},
		Scenarios:      []PlanScenario{{ID: "S1", Trigger: "sweep 21480 reclaim", Condition: "sweep_reclaim", Direction: "long", TargetChain: []float64{21500, 21520}, Invalid: "2x5m<21470", Quality: "A"}},
		NoTrade:        []string{"first 5m", "12:00-13:30 lunch"},
		DeathCondition: "acceptance above 21520",
		DayType:        "balance",
	}
}

// planActiveEngine: day_plan on + SVP + KEY LEVELS + an active PLAN BLOCK + STATUS.
func planActiveEngine() *StrategyEngine {
	e := emptyBoxFuturesEngine()
	e.config.DayPlan = &store.DayPlanConfig{PlanEnabled: true, MaxLevels: 8}
	e.config.Indicators.EnableSVP = true
	e.SetSVPContext("SVP (today's session, since 17:00 CT open): POC 21500.00 VAH 21503.75 VAL 21497.50")
	e.SetKeyLevelsContext(sampleKeyLevelsBlock)
	e.SetPlanContext(RenderPlanBlock(samplePlanDoc(), "NY"), "# PLAN STATUS (live)\nprice 21500.00 · re-plans left 2\n  21520.00 PDH: dist +20.0 · sweep=F · closes-beyond 0 · acceptance 0/2 · valid")
	return e
}

// TestFuturesPlanReorder proves the RECON #4 reorder: with an active plan, the
// PLAN BLOCK is in the prefix (before Available Data) and SVP/KEY-LEVELS + PLAN
// STATUS are at the END (Live map), each appearing exactly once.
func TestFuturesPlanReorder(t *testing.T) {
	p := planActiveEngine().BuildFuturesDecisionSystemPrompt("MNQ", 50000)

	iPlan := strings.Index(p, "# DAY PLAN (NY)")
	iAvail := strings.Index(p, "# Available Data")
	iLiveMap := strings.Index(p, "# Live map (dynamic")
	iKeyLevels := strings.Index(p, "KEY LEVELS (map")
	iStatus := strings.Index(p, "# PLAN STATUS (live)")

	if iPlan < 0 || iAvail < 0 || iLiveMap < 0 || iKeyLevels < 0 || iStatus < 0 {
		t.Fatalf("missing a section: plan=%d avail=%d livemap=%d keylv=%d status=%d", iPlan, iAvail, iLiveMap, iKeyLevels, iStatus)
	}
	if !(iPlan < iAvail) {
		t.Fatalf("PLAN BLOCK must be in the prefix (before Available Data)")
	}
	if !(iAvail < iLiveMap && iLiveMap < iKeyLevels && iKeyLevels < iStatus) {
		t.Fatalf("KEY LEVELS + PLAN STATUS must be in the END tail after the Live map header")
	}
	// KEY LEVELS appears exactly once (moved, not duplicated).
	if strings.Count(p, "KEY LEVELS (map") != 1 {
		t.Fatalf("KEY LEVELS must appear exactly once (moved to the tail)")
	}
}

// TestFuturesNoPlanUnchanged: day_plan on + KEY LEVELS but NO active plan → the
// P1.7 mid-prompt layout (no reorder). This is the existing keylevels golden.
func TestFuturesNoPlanUnchanged(t *testing.T) {
	e := keyLevelsEnabledEngine(sampleKeyLevelsBlock) // no SetPlanContext → planActive false
	p := e.BuildFuturesDecisionSystemPrompt("MNQ", 50000)
	if strings.Contains(p, "# Live map (dynamic") {
		t.Fatalf("no active plan must NOT trigger the reorder tail")
	}
	iKeyLevels := strings.Index(p, "KEY LEVELS (map")
	iDecision := strings.Index(p, "# Decision Process")
	if iKeyLevels < 0 || !(iKeyLevels < iDecision) {
		t.Fatalf("without a plan, KEY LEVELS stays mid-prompt (P1.7 layout)")
	}
}

func TestFuturesPlanGolden(t *testing.T) {
	got := planActiveEngine().BuildFuturesDecisionSystemPrompt("MNQ", 50000)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(futuresPlanGolden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Skip("golden captured/updated")
	}
	want, err := os.ReadFile(futuresPlanGolden)
	if err != nil {
		t.Fatalf("read golden (capture with UPDATE_GOLDEN=1): %v", err)
	}
	if got != string(want) {
		t.Fatalf("plan-active futures prompt DIFFERS from golden")
	}
}
