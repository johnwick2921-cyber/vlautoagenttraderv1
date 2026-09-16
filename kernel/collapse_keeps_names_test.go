// fix/collapse-keeps-names — D3 for the ≤3.00pt case.
//
// W-TF promised: same price, different timeframe → ONE candidate, ALL names.
// BuildMapCandidates delivers that when it receives both levels. It never
// received both: collapseLevelClusters runs earlier in the scorer with the
// 3.00pt cluster tolerance, keeps the stronger level, increments its Confluence
// and DISCARDS the weaker one's label. Measured on live bars 2026-09-10:
//
//	LOST: EQL·1d @ 28910.25  →  collapsed into EQL·4h @ 28910.25  (confluence=3)
//
// The map showed one name. The fix carries the collapsed labels on a field that
// is excluded from every marshal, so the Stage A golden stays byte-identical
// (E7) and no survivor, score, tolerance or tier moves (A31).
package kernel

import (
	"encoding/json"
	"strings"
	"testing"
)

func crossTFPair() []ScoredLevel {
	return []ScoredLevel{
		{DetectedLevel: DetectedLevel{Kind: KindEQL, Price: 28910.25, Lo: 28910.25, Hi: 28910.25, Label: "EQL·4h", TF: "4h"}, Score: 1.008, Grade: "A"},
		{DetectedLevel: DetectedLevel{Kind: KindEQL, Price: 28910.25, Lo: 28910.25, Hi: 28910.25, Label: "EQL·1d", TF: "1d"}, Score: 0.786, Grade: "C"},
	}
}

// TestCollapse_CrossTFLoserNameReachesTheMap — RED before the fix. The 1d
// level is collapsed into the 4h one in the scorer; the map then shows only
// "EQL·4h", and the owner cannot see that a daily reference sits at that price.
func TestCollapse_CrossTFLoserNameReachesTheMap(t *testing.T) {
	collapsed := collapseLevelClusters(crossTFPair(), clusterToleranceFor(28910.25))
	if len(collapsed) != 1 {
		t.Fatalf("collapse kept %d levels, want 1 — the survivor set must NOT change (E7)", len(collapsed))
	}
	if collapsed[0].Label != "EQL·4h" {
		t.Fatalf("survivor is %q, want the stronger EQL·4h — the survivor must NOT change (E7)", collapsed[0].Label)
	}
	cs := BuildMapCandidates(collapsed, 29143.5, 20, MapCandidateOpts{})
	if len(cs) != 1 {
		t.Fatalf("map built %d candidates from 1 survivor, want 1", len(cs))
	}
	names := cs[0].NamesLine()
	if !strings.Contains(names, "EQL·4h") || !strings.Contains(names, "EQL·1d") {
		t.Errorf("map candidate names = %q; want BOTH EQL·4h and EQL·1d — a cross-timeframe reference the owner cannot see is one he cannot weigh (D3)", names)
	}
	if cs[0].MergedCredit != 1 {
		t.Errorf("credit = %d, want 1 — carrying a name is not a second credit (W3's rule)", cs[0].MergedCredit)
	}
}

// TestCollapse_SameTFStillOneName — a same-timeframe collapse (two EQLs on 1h,
// 2pt apart) must not sprout a second name that looks like a second reference.
// It carries the label only when the timeframe differs.
func TestCollapse_SameTFStillOneName(t *testing.T) {
	// DIFFERENT labels on the SAME timeframe: identical labels would dedupe in
	// appendDistinct regardless, and a mutation that carried same-tf names
	// survived the first version of this test for exactly that reason.
	in := []ScoredLevel{
		{DetectedLevel: DetectedLevel{Kind: KindEQL, Price: 29000.00, Label: "EQL·1h", TF: "1h"}, Score: 1.0},
		{DetectedLevel: DetectedLevel{Kind: KindSWGL, Price: 29002.00, Label: "SWG-L·1h", TF: "1h"}, Score: 0.9},
	}
	collapsed := collapseLevelClusters(in, clusterToleranceFor(29000))
	cs := BuildMapCandidates(collapsed, 29100, 20, MapCandidateOpts{})
	if len(cs) != 1 {
		t.Fatalf("want 1 candidate, got %d", len(cs))
	}
	if strings.Contains(cs[0].NamesLine(), "SWG-L·1h") {
		t.Errorf("names = %q; a SAME-timeframe collapse is a duplicate and must not carry the loser's name", cs[0].NamesLine())
	}
}

// TestCollapse_CarriedNamesSurviveTheMapMergePath — the survivor carrying
// collapsed names is not always the FIRST level at its price when the map
// groups; a zone (which survives collapse untouched) can seed the candidate
// first, and the line then MERGES in. Both the seed path and the merge path
// must fold CollapsedNames, or the name is lost again one stage later.
func TestCollapse_CarriedNamesSurviveTheMapMergePath(t *testing.T) {
	in := []ScoredLevel{
		// a zone at the price — zones survive collapse and score highest here,
		// so the map seeds the candidate from it
		{DetectedLevel: DetectedLevel{Kind: KindDemand, Price: 28910.25, Lo: 28905, Hi: 28915, Label: "Demand·1h", TF: "1h"}, Score: 1.3, Grade: "B"},
		// the 4h line absorbs the 1d line in collapse, then merges into the zone's candidate
		{DetectedLevel: DetectedLevel{Kind: KindEQL, Price: 28910.25, Lo: 28910.25, Hi: 28910.25, Label: "EQL·4h", TF: "4h"}, Score: 1.008, Grade: "A"},
		{DetectedLevel: DetectedLevel{Kind: KindEQL, Price: 28910.25, Lo: 28910.25, Hi: 28910.25, Label: "EQL·1d", TF: "1d"}, Score: 0.786, Grade: "C"},
	}
	collapsed := collapseLevelClusters(in, clusterToleranceFor(28910.25))
	cs := BuildMapCandidates(collapsed, 29143.5, 20, MapCandidateOpts{})
	if len(cs) != 1 {
		t.Fatalf("want 1 merged candidate, got %d", len(cs))
	}
	names := cs[0].NamesLine()
	for _, want := range []string{"Demand·1h", "EQL·4h", "EQL·1d"} {
		if !strings.Contains(names, want) {
			t.Errorf("names = %q; missing %q — the merge path dropped a carried name", names, want)
		}
	}
}

// TestE7_CollapsedNamesNeverReachTheGolden — the pin. The carrier field must be
// invisible to json.Marshal, or the Stage A golden (685,898 bytes, 64 fixtures)
// moves. This asserts it at the type, so the golden test cannot be the only
// thing standing between this change and a byte diff.
func TestE7_CollapsedNamesNeverReachTheGolden(t *testing.T) {
	a := ScoredLevel{DetectedLevel: DetectedLevel{Kind: KindEQL, Price: 1, Label: "x", TF: "1h"}, Score: 1}
	b := a
	b.CollapsedNames = []string{"EQL·1d", "EQL·1w"}
	ja, _ := json.Marshal(a)
	jb, _ := json.Marshal(b)
	if string(ja) != string(jb) {
		t.Fatalf("CollapsedNames changed the JSON:\n  without: %s\n  with:    %s\nthe carrier must be json:\"-\" or the golden moves", ja, jb)
	}
}
