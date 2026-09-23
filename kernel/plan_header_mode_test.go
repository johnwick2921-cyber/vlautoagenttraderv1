package kernel

import (
	"os"
	"strings"
	"testing"

	"nofx/store"
)

// W-EXEC-TRUTH W3 §3 — the executor PLAN BLOCK header states the RESOLVED plan
// mode's rule. Decision 45139 (live, 2026-09-23 01:38 CT, plan 2026-09-22:ASIA
// v3) read "a valid off-plan setup may still be traded" while plan_mode was
// strict and the gate refused every decision-path entry (log_events 92685).

const (
	futuresPlanStrictGolden    = "testdata/futures_mnq_plan_strict.golden"
	futuresPlanDirectionGolden = "testdata/futures_mnq_plan_direction.golden"
)

// planActiveEngineForMode mirrors planActiveEngine with the plan mode set the
// way production resolves it (DayPlanConfig.PlanMode → store.ResolvePlanMode)
// and the PLAN BLOCK picked by the production helper setExecutorPlanContext.
func planActiveEngineForMode(mode string) *StrategyEngine {
	e := emptyBoxFuturesEngine()
	e.config.DayPlan = &store.DayPlanConfig{PlanEnabled: true, MaxLevels: 8, PlanMode: mode}
	e.config.Indicators.EnableSVP = true
	e.SetSVPContext("SVP (today's session, since 17:00 CT open): POC 21500.00 VAH 21503.75 VAL 21497.50")
	e.SetKeyLevelsContext(sampleKeyLevelsBlock)
	e.setExecutorPlanContext(samplePlanDoc(), "NY", "# PLAN STATUS (live)\nprice 21500.00 · re-plans left 2\n  21520.00 PDH: dist +20.0 · sweep=F · closes-beyond 0 · acceptance 0/2 · valid")
	return e
}

func checkGolden(t *testing.T, path, got string) {
	t.Helper()
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) != got {
		t.Fatalf("%s drifted: %s", path, firstDiff(string(want), got))
	}
}

// One golden per mode. advisory through the production helper IS the
// existing boot golden (unchanged); strict and direction are new.
func TestW3FuturesPlanGoldenPerMode(t *testing.T) {
	checkGolden(t, futuresPlanStrictGolden, planActiveEngineForMode("strict").BuildFuturesDecisionSystemPrompt("MNQ", 50000))
	checkGolden(t, futuresPlanDirectionGolden, planActiveEngineForMode("direction").BuildFuturesDecisionSystemPrompt("MNQ", 50000))
	adv := planActiveEngineForMode("advisory").BuildFuturesDecisionSystemPrompt("MNQ", 50000)
	want, err := os.ReadFile(futuresPlanGolden)
	if err != nil {
		t.Fatal(err)
	}
	if adv != string(want) {
		t.Fatalf("advisory through setExecutorPlanContext must equal the boot golden byte-for-byte: %s", firstDiff(string(want), adv))
	}
	// An unset mode resolves advisory (store.ResolvePlanMode) — same bytes.
	if planActiveEngineForMode("").BuildFuturesDecisionSystemPrompt("MNQ", 50000) != adv {
		t.Fatal("an unset plan_mode must render the advisory prompt")
	}
}

// The strict executor prompt never offers an off-plan entry.
func TestW3StrictPromptNeverOffersOffPlan(t *testing.T) {
	p := planActiveEngineForMode("strict").BuildFuturesDecisionSystemPrompt("MNQ", 50000)
	for _, bad := range []string{"may still be traded", "valid non-plan setup"} {
		if strings.Contains(p, bad) {
			t.Errorf("strict prompt says %q — the gate refuses it", bad)
		}
	}
	for _, good := range []string{"# DAY PLAN (NY) — " + PlanHeaderStrictRule, `plan_mode=strict refuses "off-plan"`} {
		if !strings.Contains(p, good) {
			t.Errorf("strict prompt missing %q", good)
		}
	}
	d := planActiveEngineForMode("direction").BuildFuturesDecisionSystemPrompt("MNQ", 50000)
	if !strings.Contains(d, "# DAY PLAN (NY) — "+PlanHeaderDirectionRule) {
		t.Error("direction prompt must state the bias rule in the header")
	}
}

// RenderPlanBlock stays advisory and byte-identical: only the header line
// differs between modes, and an unknown mode is advisory (the gate's reading).
func TestW3RenderPlanBlockForModeOnlyTheHeaderDiffers(t *testing.T) {
	doc := samplePlanDoc()
	adv := RenderPlanBlock(doc, "NY")
	if adv != RenderPlanBlockForMode(doc, "NY", "advisory") || adv != RenderPlanBlockForMode(doc, "NY", "bogus") {
		t.Fatal("advisory/unknown must be RenderPlanBlock exactly")
	}
	body := func(s string) string { return s[strings.Index(s, "\n"):] }
	for _, m := range []string{"strict", "direction"} {
		r := RenderPlanBlockForMode(doc, "NY", m)
		if r == adv || body(r) != body(adv) {
			t.Errorf("%s: only the header line may differ", m)
		}
	}
}

// The production helper resolves the plan's OWN session through
// store.ResolvePlanMode — a per-session override wins.
func TestW3ExecutorPlanModeHelperPerSessionOverride(t *testing.T) {
	strict := "strict"
	dp := &store.DayPlanConfig{PlanEnabled: true, PlanMode: "advisory", Sessions: []store.DayPlanSessionOverride{{Session: "NY", PlanMode: &strict}}}
	if got := executorPlanModeFor(dp, "NY"); got != "strict" {
		t.Fatalf("NY override: got %q", got)
	}
	if got := executorPlanModeFor(dp, "ASIA"); got != "advisory" {
		t.Fatalf("ASIA inherits the strategy mode: got %q", got)
	}
	if got := executorPlanModeFor(nil, "NY"); got != "advisory" {
		t.Fatalf("nil config: got %q", got)
	}
	e := emptyBoxFuturesEngine()
	e.config.DayPlan = dp
	e.setExecutorPlanContext(samplePlanDoc(), "NY", "")
	if !strings.HasPrefix(e.planBlockLine, "# DAY PLAN (NY) — "+PlanHeaderStrictRule) || e.planMode != "strict" {
		t.Fatalf("NY block must be strict: %q mode=%q", strings.SplitN(e.planBlockLine, "\n", 2)[0], e.planMode)
	}
	e.setExecutorPlanContext(samplePlanDoc(), "ASIA", "")
	if e.planBlockLine != RenderPlanBlock(samplePlanDoc(), "ASIA") || e.planMode != "advisory" {
		t.Fatal("ASIA block must be the advisory render")
	}
	// SetPlanContext (tests, legacy callers) resets the mode to advisory.
	e.SetPlanContext("x", "y")
	if e.planMode != "" {
		t.Fatal("SetPlanContext must reset the mode")
	}
}

// Fixture: decision 45139's stored executor prompt — the header the live
// model read under strict IS today's advisory render; the strict render says
// the opposite (the gate's truth). Redacted testdata (plan text only).
func TestW3Decision45139HeaderFixture(t *testing.T) {
	b, err := os.ReadFile("testdata/w3_decision45139_plan_header.txt")
	if err != nil {
		t.Fatal(err)
	}
	stored := strings.SplitN(string(b), "\n", 2)[0]
	adv := strings.SplitN(RenderPlanBlockForMode(PlanDoc{}, "ASIA", "advisory"), "\n", 2)[0]
	if stored != adv {
		t.Fatalf("the stored header must be the advisory render:\n stored %q\n render %q", stored, adv)
	}
	strict := strings.SplitN(RenderPlanBlockForMode(PlanDoc{}, "ASIA", "strict"), "\n", 2)[0]
	if strict == stored || !strings.Contains(strict, "off-plan is refused") || strings.Contains(strict, "may still be traded") {
		t.Fatalf("the strict header must state the gate's rule: %q", strict)
	}
	if !strings.Contains(string(b), `or "off-plan" for a valid non-plan setup`) {
		t.Fatal("fixture: the stored field line (engine_prompt_futures.go) is part of the evidence")
	}
}

// The boot self-check carries the strict case: all four fixtures pass at HEAD.
func TestW3BootSelfCheckHasFourCasesAllPassing(t *testing.T) {
	res, ok := VerifyPromptGoldens()
	names := []string{}
	for _, r := range res {
		names = append(names, r.Name)
		if !r.OK {
			t.Errorf("%s: %s", r.Name, r.Detail)
		}
	}
	if !ok || len(res) != 4 || names[3] != "futures-plan-strict" {
		t.Fatalf("want 4 passing cases incl. futures-plan-strict, got %v ok=%v", names, ok)
	}
}
