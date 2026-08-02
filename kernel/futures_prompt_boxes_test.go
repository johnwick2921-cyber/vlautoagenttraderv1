package kernel

import (
	"os"
	"strings"
	"testing"

	"nofx/store"
)

const futuresEmptyGolden = "testdata/futures_mnq_empty.golden"

// emptyBoxFuturesEngine is futuresTestEngine with explicitly EMPTY prompt boxes
// (the back-compat case: the owner edited nothing).
func emptyBoxFuturesEngine() *StrategyEngine {
	e := futuresTestEngine()
	e.config.PromptSections = store.PromptSectionsConfig{}
	return e
}

// boxedFuturesEngine fills all 4 prompt boxes with sentinel text.
func boxedFuturesEngine() *StrategyEngine {
	e := futuresTestEngine()
	e.config.PromptSections = store.PromptSectionsConfig{
		RoleDefinition:   "ROLE_BOX_SENTINEL — you are a custom futures desk.",
		TradingFrequency: "FREQ_BOX_SENTINEL — at most 3 trades/session.",
		EntryStandards:   "ENTRY_BOX_SENTINEL — only A+ confluence.",
		DecisionProcess:  "DECISION_BOX_SENTINEL — checklist 1-2-3.",
	}
	return e
}

// TestFuturesPromptEmptyBoxesByteIdentical is the Change-4 BACK-COMPAT proof:
// with all 4 prompt boxes EMPTY, BuildFuturesDecisionSystemPrompt must produce
// EXACTLY the pre-change fixed futures prompt (golden captured before the
// builder read the boxes). So existing futures strategies do not change.
// Recapture with: UPDATE_GOLDEN=1 go test ./kernel -run EmptyBoxesByteIdentical
func TestFuturesPromptEmptyBoxesByteIdentical(t *testing.T) {
	got := emptyBoxFuturesEngine().BuildFuturesDecisionSystemPrompt("MNQ", 50000)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll("testdata", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(futuresEmptyGolden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Skip("golden captured/updated")
	}
	want, err := os.ReadFile(futuresEmptyGolden)
	if err != nil {
		t.Fatalf("read golden (capture first with UPDATE_GOLDEN=1): %v", err)
	}
	if got != string(want) {
		t.Fatalf("empty-box futures prompt DIFFERS from the pre-change golden — back-compat broken")
	}
}

// TestFuturesSvpInjection proves Part B3: the SVP line + legend appear ONLY when
// svp_enabled is ON and a non-empty line was threaded in; OFF (nil) or an empty
// line inject nothing — which is what keeps the empty-box golden byte-identical.
func TestFuturesSvpInjection(t *testing.T) {
	line := "SVP: dev POC 21500.62 VAH 21503.75 VAL 21497.50 | prior POC 21480.00 VAH 21490.00 VAL 21470.00"

	// ON (enable_svp) + non-empty line → both the data line and the legend appear.
	on := emptyBoxFuturesEngine()
	on.config.Indicators.EnableSVP = true
	on.SetSVPContext(line)
	got := on.BuildFuturesDecisionSystemPrompt("MNQ", 50000)
	if !strings.Contains(got, line) {
		t.Error("SVP line missing when enable_svp is ON")
	}
	if !strings.Contains(got, "Legend: POC =") {
		t.Error("SVP legend missing when enable_svp is ON")
	}

	// OFF (default) even with a context set → nothing injected.
	off := emptyBoxFuturesEngine()
	off.SetSVPContext(line)
	if strings.Contains(off.BuildFuturesDecisionSystemPrompt("MNQ", 50000), line) {
		t.Error("SVP line leaked while enable_svp is OFF (golden would break)")
	}

	// ON but empty line (insufficient bars) → nothing injected.
	onEmpty := emptyBoxFuturesEngine()
	onEmpty.config.Indicators.EnableSVP = true
	onEmpty.SetSVPContext("")
	if strings.Contains(onEmpty.BuildFuturesDecisionSystemPrompt("MNQ", 50000), "Legend: POC =") {
		t.Error("SVP legend appeared with an empty SVP line")
	}
}

// TestFuturesPromptBoxesHonored proves A (full control restored): ALL 4 prompt
// boxes reach the futures prompt — Role/Decision via override-or-default,
// Frequency/Entry append-when-set. Empty boxes inject nothing (golden holds).
func TestFuturesPromptBoxesHonored(t *testing.T) {
	boxed := boxedFuturesEngine().BuildFuturesDecisionSystemPrompt("MNQ", 50000)
	for _, s := range []string{
		"ROLE_BOX_SENTINEL", "FREQ_BOX_SENTINEL", "ENTRY_BOX_SENTINEL", "DECISION_BOX_SENTINEL",
	} {
		if !strings.Contains(boxed, s) {
			t.Errorf("futures prompt should honor the %q box", s)
		}
	}
	// The Instrument identity + output format stay FIXED regardless of boxes.
	for _, s := range []string{"Symbol: MNQ", "<reasoning>", "<decision>", "Hard Constraints (Risk Control)"} {
		if !strings.Contains(boxed, s) {
			t.Errorf("boxed futures prompt lost FIXED marker %q", s)
		}
	}

	// Empty boxes never inject the box sections.
	empty := emptyBoxFuturesEngine().BuildFuturesDecisionSystemPrompt("MNQ", 50000)
	for _, s := range []string{
		"ROLE_BOX_SENTINEL", "FREQ_BOX_SENTINEL", "ENTRY_BOX_SENTINEL", "DECISION_BOX_SENTINEL",
	} {
		if strings.Contains(empty, s) {
			t.Errorf("empty-box futures prompt unexpectedly contains %q", s)
		}
	}
}

// TestFuturesPromptCustomRoleHonored is the A proof (full control): a futures
// strategy with a CUSTOM Role + Decision box renders THAT text. The earlier
// crypto-leak is fixed at the DATA layer (neutral defaults + a migration), NOT
// by ignoring the box. Empty boxes still fall back to the fixed CME text.
func TestFuturesPromptCustomRoleHonored(t *testing.T) {
	e := futuresTestEngine()
	e.config.PromptSections = store.PromptSectionsConfig{
		RoleDefinition:  "# MY CUSTOM FUTURES DESK\nScalp MNQ with discipline.",
		DecisionProcess: "# MY DECISION FLOW\n1. check trend 2. enter",
	}
	p := e.BuildFuturesDecisionSystemPrompt("MNQ", 50000)
	if !strings.Contains(p, "MY CUSTOM FUTURES DESK") {
		t.Error("futures prompt must honor a custom Role box")
	}
	if !strings.Contains(p, "MY DECISION FLOW") {
		t.Error("futures prompt must honor a custom Decision box")
	}
	// Empty boxes fall back to the fixed CME role (golden holds).
	pe := emptyBoxFuturesEngine().BuildFuturesDecisionSystemPrompt("MNQ", 50000)
	if !strings.Contains(pe, "professional CME") {
		t.Error("empty Role box must fall back to the fixed CME role")
	}
}

// TestFuturesSubModes proves Chunk-2: the 3 futures sub-modes route to the
// futures builder; "futures"/"futures-balanced" == the golden (byte-identical);
// aggressive/conservative add ONLY their "## Mode:" block (TEXT-only — nothing
// else in the prompt changes, so Risk Control / output format are untouched).
func TestFuturesSubModes(t *testing.T) {
	e := emptyBoxFuturesEngine() // empty boxes → only the mode block can vary
	balanced := e.BuildSystemPrompt(50000, "futures", "MNQ")
	balancedExplicit := e.BuildSystemPrompt(50000, "futures-balanced", "MNQ")
	aggressive := e.BuildSystemPrompt(50000, "futures-aggressive", "MNQ")
	conservative := e.BuildSystemPrompt(50000, "futures-conservative", "MNQ")

	golden, err := os.ReadFile(futuresEmptyGolden)
	if err != nil {
		t.Fatalf("read golden: %v", err)
	}
	if balanced != string(golden) {
		t.Error("variant 'futures' must equal the golden (no-block default)")
	}
	if balancedExplicit != balanced {
		t.Error("'futures-balanced' must equal 'futures'")
	}
	if !strings.Contains(aggressive, "## Mode: Aggressive") {
		t.Error("futures-aggressive missing the Aggressive block")
	}
	if !strings.Contains(conservative, "## Mode: Conservative") {
		t.Error("futures-conservative missing the Conservative block")
	}
	// still the FUTURES prompt (CME role), never crypto.
	for _, p := range []string{aggressive, conservative} {
		if !strings.Contains(p, "professional CME") || strings.Contains(strings.ToLower(p), "cryptocurrency") {
			t.Error("a futures sub-mode is not the futures prompt")
		}
	}
	// TEXT-ONLY proof: removing exactly the mode block yields the balanced prompt.
	if strings.Replace(aggressive, futuresModeAggressive, "", 1) != balanced {
		t.Error("futures-aggressive must equal balanced + ONLY the Aggressive block")
	}
	if strings.Replace(conservative, futuresModeConservative, "", 1) != balanced {
		t.Error("futures-conservative must equal balanced + ONLY the Conservative block")
	}
}
