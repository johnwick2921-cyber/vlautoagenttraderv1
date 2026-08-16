package kernel

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"strings"

	"nofx/store"
)

// P1 — PROMPT-GOLDEN SELF-CHECK (boot integrity, the Knight-Capital control).
//
// The goldens are the contract for what the executor actually says to the model.
// A binary can be the right revision and STILL be wrong if the prompt builders
// drifted, so the boot assertion re-renders the same three fixtures the golden
// tests use and compares them to the goldens EMBEDDED IN THIS BINARY. Embedding
// matters: it checks the code against the contract it was built with, not against
// whatever happens to be on disk next to it.
//
// Pure + allocation-light (three prompt renders) — it runs once at startup.

//go:embed testdata/futures_mnq_empty.golden testdata/futures_mnq_keylevels.golden testdata/futures_mnq_plan.golden
var goldenFS embed.FS

// sampleKeyLevelsBlockSelfCheck mirrors the test fixture byte-for-byte.
const sampleKeyLevelsBlockSelfCheck = "KEY LEVELS (map, nearest-first; price 21500.00):\n" +
	"  21500.00  PDC            A  fresh     +0.0\n" +
	"  21520.00  PDH            A  fresh·x2  +20.0\n" +
	"  21480.00  RTH-L          B  fresh     -20.0\n" +
	"Anchor: react AT these levels (grade A>B>C); do not chase price between them."

// selfCheckEngine rebuilds the futures fixture engine used by the golden tests.
func selfCheckEngine() *StrategyEngine {
	cfg := &store.StrategyConfig{
		Language: "en",
		RiskControl: store.RiskControlConfig{
			MinConfidence:      75,
			MinRiskRewardRatio: 1.5,
			MaxPositions:       1,
		},
	}
	e := &StrategyEngine{config: cfg}
	e.config.PromptSections = store.PromptSectionsConfig{} // empty-box
	return e
}

func selfCheckKeyLevelsEngine() *StrategyEngine {
	e := selfCheckEngine()
	e.config.DayPlan = &store.DayPlanConfig{PlanEnabled: true, MaxLevels: 8}
	e.SetKeyLevelsContext(sampleKeyLevelsBlockSelfCheck)
	return e
}

// selfCheckPlanDoc mirrors samplePlanDoc() in futures_prompt_plan_test.go.
func selfCheckPlanDoc() PlanDoc {
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

// selfCheckPlanEngine mirrors planActiveEngine() — day_plan on + SVP + KEY LEVELS
// + an active PLAN BLOCK + STATUS.
func selfCheckPlanEngine() *StrategyEngine {
	e := selfCheckEngine()
	e.config.DayPlan = &store.DayPlanConfig{PlanEnabled: true, MaxLevels: 8}
	e.config.Indicators.EnableSVP = true
	e.SetSVPContext("SVP (today's session, since 17:00 CT open): POC 21500.00 VAH 21503.75 VAL 21497.50")
	e.SetKeyLevelsContext(sampleKeyLevelsBlockSelfCheck)
	e.SetPlanContext(
		RenderPlanBlock(selfCheckPlanDoc(), "NY"),
		"# PLAN STATUS (live)\nprice 21500.00 · re-plans left 2\n  21520.00 PDH: dist +20.0 · sweep=F · closes-beyond 0 · acceptance 0/2 · valid",
	)
	return e
}

// GoldenSelfCheckResult reports one fixture's outcome.
type GoldenSelfCheckResult struct {
	Name    string
	OK      bool
	Detail  string // on failure: where it diverged
	GotHash string
}

// VerifyPromptGoldens re-renders the three golden fixtures and compares them to
// the embedded goldens. Returns (results, allPassed).
func VerifyPromptGoldens() ([]GoldenSelfCheckResult, bool) {
	cases := []struct {
		name string
		file string
		got  func() string
	}{
		{"futures-empty", "testdata/futures_mnq_empty.golden",
			func() string { return selfCheckEngine().BuildFuturesDecisionSystemPrompt("MNQ", 50000) }},
		{"futures-keylevels", "testdata/futures_mnq_keylevels.golden",
			func() string { return selfCheckKeyLevelsEngine().BuildFuturesDecisionSystemPrompt("MNQ", 50000) }},
		{"futures-plan", "testdata/futures_mnq_plan.golden",
			func() string { return selfCheckPlanEngine().BuildFuturesDecisionSystemPrompt("MNQ", 50000) }},
	}
	out := make([]GoldenSelfCheckResult, 0, len(cases))
	all := true
	for _, c := range cases {
		want, err := goldenFS.ReadFile(c.file)
		got := c.got()
		res := GoldenSelfCheckResult{Name: c.name, GotHash: shortSum(got)}
		switch {
		case err != nil:
			res.OK, res.Detail = false, "embedded golden unreadable: "+err.Error()
		case string(want) != got:
			res.OK, res.Detail = false, firstDiff(string(want), got)
		default:
			res.OK = true
		}
		if !res.OK {
			all = false
		}
		out = append(out, res)
	}
	return out, all
}

// firstDiff names the first differing line so a failure is actionable in one line.
func firstDiff(want, got string) string {
	w := strings.Split(want, "\n")
	g := strings.Split(got, "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		var wl, gl string
		if i < len(w) {
			wl = w[i]
		}
		if i < len(g) {
			gl = g[i]
		}
		if wl != gl {
			return fmt.Sprintf("line %d: want %q, got %q", i+1, trunc(wl), trunc(gl))
		}
	}
	return "length differs"
}

func trunc(s string) string {
	if len(s) > 70 {
		return s[:70] + "…"
	}
	return s
}

func shortSum(s string) string {
	h := sha256.Sum256([]byte(s))
	return hex.EncodeToString(h[:4])
}
