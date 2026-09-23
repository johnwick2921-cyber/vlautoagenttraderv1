package kernel

import (
	"strings"
	"testing"
	"time"
)

// W-EXEC-TRUTH W2 A3/A4 — kernel-seam pins. The write-loop behaviour is pinned
// at the production call site in trader/scenario_write_truth_test.go; these
// cover the seam edges that need a synthetic map (an FVG zone candidate, the
// capacity-cut builder) and the class-38 / repair-routing contracts.

func w2Level(kind LevelKind, lo, hi float64, label string) DetectedLevel {
	now := time.Date(2026, 9, 10, 20, 0, 0, 0, CTLocation())
	closeMs := now.Add(-10 * time.Hour).UnixMilli()
	return DetectedLevel{Kind: kind, Price: (lo + hi) / 2, Lo: lo, Hi: hi, Label: label, OriginDate: "2026-09-09",
		FormedAtMs: closeMs - 60000, FormedCloseMs: &closeMs, FormationTF: "5m", IdentitySymbol: "MNQ", FormationLookback: 400}
}

// No false refusal: an fvg_entry naming its OWN FVG candidate — candidate
// price is the gap midpoint, the evaluator anchor is the distal edge 10 pts
// away (> the 3.00 cluster tolerance) — is admitted because the anchor lies
// inside the level's own zone. The same id on an anchor OUTSIDE the zone is
// refused (the tolerance is the zone, not a blanket pass).
func TestW2A3FvgNamingItsOwnGapIsAdmitted(t *testing.T) {
	cs := BuildMapCandidates([]ScoredLevel{{DetectedLevel: w2Level(KindFVG, 15470, 15490, "FVG·5m"), Grade: "A", Score: 1}}, 15500, 10, MapCandidateOpts{})
	if len(cs) != 1 || cs[0].ID == nil || cs[0].Price != 15480 {
		t.Fatalf("fvg candidate: %+v", cs)
	}
	id := *cs[0].ID
	sc := PlanScenario{ID: "S1", LevelID: &id, Condition: "fvg_entry", Direction: "long", Trigger: "retrace into the gap", Invalid: "close below the gap",
		Fvg: &PlanFvgEntry{Lo: 15470, Hi: 15490, Direction: "long"}}
	d := &PlanDoc{Scenarios: []PlanScenario{sc}}
	anchor, ok := ScenarioAnchor(sc, nil)
	if !ok || anchor != 15470 {
		t.Fatalf("fvg anchor must be the distal edge: %.2f %v", anchor, ok)
	}
	if r := ResolveScenarioIdentity(sc, IdentityLevelsFromCandidates(cs), anchor, ok); !r.Disagreed {
		t.Fatal("precondition: the line predicate must call this a disagreement (10 pts > 3.00)")
	}
	if err := CheckScenarioWriteTruth(d, cs, nil, 0).Err(); err != nil {
		t.Fatalf("an fvg_entry naming its own gap must be admitted: %v", err)
	}
	d.Scenarios[0].Fvg = &PlanFvgEntry{Lo: 15440, Hi: 15460, Direction: "long"}
	if err := CheckScenarioWriteTruth(d, cs, nil, 0).Err(); err == nil || !strings.Contains(err.Error(), "S1 identity≠price") {
		t.Fatalf("an anchor outside the named zone must be refused: %v", err)
	}
}

// UNKNOWN is not []: a nil map judges nothing; an empty (known) map refuses
// any named id as unresolved.
func TestW2NilMapIsUnknownEmptyMapIsKnown(t *testing.T) {
	id := "abc"
	d := &PlanDoc{Scenarios: []PlanScenario{{ID: "S1", LevelID: &id, Trigger: "fade 100"}}}
	if v := CheckScenarioWriteTruth(d, nil, nil, 0.25); v.IdentityChecked || v.ChainChecked || v.Err() != nil {
		t.Fatalf("nil map must be UNKNOWN: %+v", v)
	}
	v := CheckScenarioWriteTruth(d, []MapCandidate{}, nil, 0.25)
	if !v.IdentityChecked || v.Err() == nil || !strings.Contains(v.Err().Error(), "S1 identity unresolved") {
		t.Fatalf("an empty map is a KNOWN map: %+v", v)
	}
}

// CapacityCutCandidates keeps only pool references no seated candidate covers.
func TestW2CapacityCutIsPoolMinusSeated(t *testing.T) {
	seatedLvl := ScoredLevel{DetectedLevel: w2Level(KindPDH, 15520, 15520, "PDH"), Grade: "A", Score: 2}
	cutLvl := ScoredLevel{DetectedLevel: w2Level(KindPDL, 15500, 15500, "PDL"), Grade: "C", Score: 0.1}
	seated := BuildMapCandidates([]ScoredLevel{seatedLvl}, 15550, 10, MapCandidateOpts{})
	cut := CapacityCutCandidates([]ScoredLevel{seatedLvl, cutLvl}, seated, 15550, 10)
	if len(cut) != 1 || cut[0].Price != 15500 {
		t.Fatalf("capacity cut = %+v, want only 15500", cut)
	}
	if CapacityCutCandidates(nil, seated, 15550, 10) != nil {
		t.Fatal("no pool → nil (UNKNOWN), never a fabricated []")
	}
}

// Repair routing: every A3/A4 rejection reaches its own law excerpt, never
// the generic level-label fallback.
func TestW2RepairLawRoutesTheNewRejections(t *testing.T) {
	generic := "Copy the machine table's labels and prices"
	for name, err := range map[string]string{
		"identity≠price":      "S2 identity≠price: level_id names PDH 30917.50 but the scenario trades 31009.75",
		"identity unresolved": `S2 identity unresolved: level_id "x" is not in the frozen map`,
		"obstacle chain":      "S4 obstacle chain: omits SWG-H·5m 31043.00 (4.00 pts from entry)",
	} {
		got := lawExcerptsFor(err)
		if strings.Contains(got, generic) {
			t.Fatalf("%s fell through to the generic excerpt", name)
		}
		want := RepairIdentityPriceLaw
		if name == "obstacle chain" {
			want = RepairObstacleChainLaw
		}
		if !strings.Contains(got, want) {
			t.Fatalf("%s did not route to its law: %q", name, got)
		}
	}
}

// Class 38: the four W2 rows are registered (so both guard tests —
// AllStated and FailsWhenPromptDropsARule — iterate them) and every fragment
// renders in BOTH the knob-OFF and knob-ON contract, BEFORE the write-time
// feasibility sentence.
func TestW2Class38RowsRegisteredAndRendered(t *testing.T) {
	want := map[string]bool{
		"scenarioIdentityWriteIssues ← CheckScenarioWriteTruth": false,
		"scenarioIdentityWriteIssues (anchor_reuse":             false,
		"obstacleChainWriteIssues ← CheckScenarioWriteTruth":    false,
		"obstacleChainWriteIssues (reduce_qty1":                 false,
	}
	off := plannerOutputContract(8, 5, true, true, false)
	on := plannerOutputContract(8, 5, true, true, true)
	feas := strings.Index(on, "WRITE-TIME FEASIBILITY:")
	for _, c := range PromptContracts() {
		for k := range want {
			if strings.Contains(c.Site, k) {
				want[k] = true
				if c.Gate != "" {
					t.Errorf("W2 row %q is a correction and must be unconditional", c.Rule)
				}
				for _, frag := range c.MustAppear {
					if !strings.Contains(off, frag) {
						t.Errorf("fragment %q absent from the OFF rendering", frag)
					}
					if i := strings.Index(on, frag); i < 0 || i > feas {
						t.Errorf("fragment %q must render BEFORE the feasibility sentence (at %d, feasibility at %d)", frag, i, feas)
					}
				}
			}
		}
	}
	for k, ok := range want {
		if !ok {
			t.Errorf("class-38 row %q not registered", k)
		}
	}
	if strings.Contains(off, "A reduce response is a declared intention") {
		t.Fatal("the replaced reduce sentence must not coexist with the new refusal")
	}
}
