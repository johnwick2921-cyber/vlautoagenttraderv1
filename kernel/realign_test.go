package kernel

import (
	"encoding/json"
	"strings"
	"testing"
)

// W13 — the re-align prompt carries the owner's change verbatim (including the
// NOTE, their voice into the AI) and the four re-examination questions, and it
// states the DEFEND-unless-new-information rule.
func TestBuildRealignUserPrompt(t *testing.T) {
	p := BuildRealignUserPrompt(
		"BIAS long · levels: PDH 30288",
		"price 30231.5 · re-plans left 2",
		OwnerChange{
			Kind: "add-level", Summary: "added D-4h demand zone", Price: 30156,
			Label: "D-4h", Grade: "A", Instruction: "sweep+reclaim = entry",
			Note: "strong 4h OB — tôi tin zone này", ScenarioTag: "S1",
		},
	)
	for _, want := range []string{
		"CURRENT PLAN", "overlay-resolved", "BIAS long", // the plan the owner sees
		"LIVE FACTS", "price 30231.5", // Go facts
		"OWNER'S CHANGE", "add-level", "30156.00", "D-4h", "A",
		"sweep+reclaim = entry", "tôi tin zone này", "S1", // note survives, any language
		"(a) does this change the bias or the flip line?",
		"(b) does it warrant a NEW scenario",
		"(c) do any scenarios need re-targeting",
		"(d) does it conflict with an AI level?",
		"DEFEND the existing structure unless",
		"Never give vague advice", "the patch IS",
	} {
		if !strings.Contains(p, want) {
			t.Fatalf("realign prompt missing %q\n---\n%s", want, p)
		}
	}
}

// W13 — a bulk-add batch is described as ONE re-examination, not N.
func TestRealignBulkDescribedOnce(t *testing.T) {
	d := OwnerChange{Kind: "bulk-add", Summary: "5 levels", BatchCount: 5}.Describe()
	if !strings.Contains(d, "5 levels saved together") || !strings.Contains(d, "ONE re-examination") {
		t.Fatalf("bulk batch must be framed as one call:\n%s", d)
	}
	// a single-level change must NOT claim a batch
	if s := (OwnerChange{Kind: "add-level", BatchCount: 1}).Describe(); strings.Contains(s, "saved together") {
		t.Fatalf("single change must not render a batch line:\n%s", s)
	}
}

// W13 — the no-change classifier: only a PROPOSE-MERGE carrying real ops is a
// proposal. Everything else renders as the quiet "plan stands" chip.
func TestIsNoChange(t *testing.T) {
	patch := json.RawMessage(`[{"op":"replace","path":"/bias/direction","value":"short"}]`)
	cases := []struct {
		name string
		r    *PlannerReply
		want bool
	}{
		{"nil", nil, true},
		{"defend", &PlannerReply{Verdict: "DEFEND"}, true},
		{"concede without patch", &PlannerReply{Verdict: "CONCEDE"}, true},
		{"propose-merge with EMPTY patch", &PlannerReply{Verdict: "PROPOSE-MERGE", Patch: json.RawMessage(`[]`)}, true},
		{"propose-merge with ops", &PlannerReply{Verdict: "PROPOSE-MERGE", Patch: patch}, false},
	}
	for _, c := range cases {
		if got := IsNoChange(c.r); got != c.want {
			t.Fatalf("%s: IsNoChange = %v, want %v", c.name, got, c.want)
		}
	}
}

// W13 — the re-align reuses the Ask-Planner anti-sycophancy contract VERBATIM, so
// the KPI stays one comparable series. A bare disagreement still never patches.
func TestRealignReusesAntiSycophancyContract(t *testing.T) {
	if !strings.Contains(AskPlannerSystemPrompt, "NOT a sycophant") {
		t.Fatal("the realign path must reuse the anti-sycophancy system prompt verbatim")
	}
	// the shared parser drops a patch on a bare disagreement (contract enforcement)
	raw := `{"evidence":"e","point_class":"BARE-DISAGREEMENT","verdict":"PROPOSE-MERGE",
	         "summary":"s","patch":[{"op":"remove","path":"/levels/0"}]}`
	reply, err := ParsePlannerReply(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if hasPatchOps(reply.Patch) {
		t.Fatal("a BARE-DISAGREEMENT must never carry a patch (contract)")
	}
	if !IsNoChange(reply) {
		t.Fatal("a de-patched bare disagreement must classify as no-change")
	}
}
