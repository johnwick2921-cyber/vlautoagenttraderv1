package kernel

import (
	"strings"
	"testing"
)

// Planner-speed wave 3 (2026-08-31) — the repair prompt must carry the
// instruction header, the errors verbatim, the rejected output verbatim, and
// law excerpts for the violated rule ONLY; it must NOT re-send the full
// playbook (no level map / candle tables markers).
func TestBuildPlannerRepairPrompt(t *testing.T) {
	rejected := `{"reasoning":"x","bias":{"direction":"long"},"levels":[...],"scenarios":[{"id":"S2","condition":"sweep_reclaim","arm":{"enabled":true,"entry":29500,"stop":29400,"target":29700,"legs":[{"kind":"limit"}]}}]}`
	err := `arm on S2 needs EXACTLY 2 legs (split contract), got 1`
	got := BuildPlannerRepairPrompt(rejected, err, nil)
	for _, want := range []string{
		"You are repairing a rejected plan",
		"## Validator errors (verbatim)",
		err,
		"## Rejected plan output (verbatim)",
		rejected,
		"## Applicable law (excerpts for the violated rules only)",
		"ARM-SPLIT LAW",
		"EXACTLY 2 legs",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("repair prompt missing %q:\n%s", want, got)
		}
	}
	// No full playbook: these markers only exist in the full author prompt.
	for _, forbidden := range []string{"KEY LEVELS", "candle", "CANDLE"} {
		if strings.Contains(got, forbidden) {
			t.Fatalf("repair prompt must not re-send the playbook (found %q)", forbidden)
		}
	}
}

func TestLawExcerptsForMapping(t *testing.T) {
	cases := map[string]string{
		"arm on S2 needs EXACTLY 2 legs (split contract), got 1":                           "ARM-SPLIT LAW",
		"arm on S2 split requires confirm=touch at the sweep ref":                          "ARM-SPLIT LAW",
		"arm legs on breakdown_continue — arm_legs_sweep_reclaim_only":                     "ARM-SPLIT LAW",
		"S1 breakdown_continue: a close came back across 29437.00 — the breakdown is void": "BREAKDOWN-CONTINUE LAW",
		"scenario[1].confirm2.rule \"1m_mss\" not allowed for breakdown_continue":          "ENTRY-LAW CONFIRM LAW",
		"levels[3] and [4] are 0.50 apart — duplicates":                                    "machine table",
	}
	for err, want := range cases {
		if got := lawExcerptsFor(err); !strings.Contains(got, want) {
			t.Fatalf("lawExcerptsFor(%q) = %q, want %q", err, got, want)
		}
	}
}
