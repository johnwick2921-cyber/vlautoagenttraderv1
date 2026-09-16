package kernel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── ROOT-FIX PART A (2026-09-02) — THE CONSUMPTION AUDIT, AS A TEST ──────────
// The dispatch asked for a 40-50% output-token cut by slimming the plan schema.
// The audit found NO removable field (every one has a reader) and, decisively,
// that the plan JSON is not what makes the call long. These tests pin both
// findings so a future wave does not re-derive them — or ship the cut on the
// wrong premise.

// D1 — every top-level plan field has at least one reader OUTSIDE plan_doc.go.
// A field that only the schema mentions is dead weight and may be removed; if
// this test ever finds one, that is a real removal candidate.
func TestRootFixEveryPlanFieldHasAReader(t *testing.T) {
	// field → the Go accessor a reader would use.
	fields := map[string]string{
		"reasoning":       ".Reasoning",
		"bias":            ".Bias",
		"levels":          ".Levels",
		"scenarios":       ".Scenarios",
		"no_trade":        ".NoTrade",
		"death_condition": ".DeathCondition",
		"day_type":        ".DayType",
		"death":           ".DeathStructured",
		"flip":            ".FlipStructured",
	}
	roots := []string{"..", "../trader", "../api"}
	corpus := loadRootFixCorpus(t, roots)
	for jsonName, accessor := range fields {
		readers := 0
		for path, body := range corpus {
			if strings.HasSuffix(path, "plan_doc.go") || strings.HasSuffix(path, "_test.go") {
				continue
			}
			if strings.Contains(body, accessor) || strings.Contains(body, `"`+jsonName+`"`) {
				readers++
			}
		}
		if readers == 0 {
			t.Errorf("ROOT-FIX A-1: plan field %q (%s) has NO reader — removal candidate", jsonName, accessor)
		}
	}
}

func loadRootFixCorpus(t *testing.T, roots []string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, root := range roots {
		entries, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".go") {
				continue
			}
			b, err := os.ReadFile(filepath.Join(root, e.Name()))
			if err != nil {
				continue
			}
			out[filepath.Join(root, e.Name())] = string(b)
		}
	}
	if len(out) < 20 {
		t.Fatalf("corpus too small (%d files) — the audit would pass vacuously", len(out))
	}
	return out
}

// D3 — THE MEASUREMENT, pinned. A representative stored plan is ~3.1 KB ≈ 920
// tokens. Measured against 67 full-author calls (2026-08-31 → 09-02) whose p50
// completion was 23,769 tokens, the plan JSON is ~4% of the output: even
// deleting the ENTIRE schema could not deliver the dispatch's 40-50%.
// This test states the arithmetic so the premise cannot quietly return.
func TestRootFixPlanJSONIsASmallFractionOfOutput(t *testing.T) {
	const (
		measuredP50CompletionTokens = 23769 // n=67 full-author calls, prompt>9k
		charsPerToken               = 3.24  // observed reasoning_chars / completion tokens
	)
	doc := PlanDoc{
		Reasoning: strings.Repeat("the plan rationale, roughly as long as a real one. ", 10),
		Bias:      PlanBias{Direction: "long", Conviction: "medium", FlipCondition: "2x5m < 29500"},
		Levels: []PlanLevel{
			{Price: 29500, Label: "PDL", Grade: "A", Instruction: "reclaim"},
			{Price: 29550, Label: "RN 29550", Grade: "B", Instruction: "reclaim"},
			{Price: 29600, Label: "RN 29600", Grade: "B", Instruction: "fade"},
			{Price: 29800, Label: "PDH", Grade: "A", Instruction: "fade"},
		},
		Scenarios: []PlanScenario{{
			ID: "S1", Trigger: "reclaim of PDL with a 5m close back above",
			Condition: "reclaim", Direction: "long", TargetChain: []float64{29700, 29800},
			Invalid: "2x5m<29480", Quality: "B",
			Confirm: &PlanConfirm{Rule: "2x5m_close", RefPrice: 29500, Side: "above"},
		}},
		NoTrade:        []string{"first 5m", "into the 08:30 print"},
		DeathCondition: "acceptance above 29800",
		DayType:        "balance",
	}
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	planTokens := float64(len(b)) / charsPerToken
	share := planTokens / measuredP50CompletionTokens * 100
	t.Logf("ROOT-FIX A-4: plan JSON %d bytes ≈ %.0f tokens = %.1f%% of the %d-token p50 output; reasoning is the other ~%.1f%%",
		len(b), planTokens, share, measuredP50CompletionTokens, 100-share)
	if share > 15 {
		t.Fatalf("premise changed: the plan JSON is now %.1f%% of output — re-open the schema-slim question", share)
	}
	// The honest ceiling: even a 100% cut of the JSON saves less than a fifth
	// of what the dispatch targeted (40%).
	if maxSaving := share; maxSaving >= 40 {
		t.Fatalf("unreachable: a full schema deletion would save %.1f%%", maxSaving)
	}
}
