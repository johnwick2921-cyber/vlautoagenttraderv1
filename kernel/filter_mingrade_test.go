package kernel

import "testing"

// Config-truth (settings fix) — min_grade honoring: the planner assembler drops
// sub-grade levels, but never owner levels (grade A). Empty = no filter.

func TestFilterLevelsByMinGrade(t *testing.T) {
	levels := []ScoredLevel{
		{Grade: "A", DetectedLevel: DetectedLevel{Label: "PDH"}},
		{Grade: "B", DetectedLevel: DetectedLevel{Label: "RN"}},
		{Grade: "C", DetectedLevel: DetectedLevel{Label: "FVG"}},
		{Grade: "A", DetectedLevel: DetectedLevel{Kind: KindOwner, Label: "👤 my"}},
	}

	// empty → no filter
	if got := FilterLevelsByMinGrade(levels, ""); len(got) != 4 {
		t.Fatalf("empty min_grade must not filter: got %d", len(got))
	}
	// B → drops C, keeps A/B (incl. owner A)
	gotB := FilterLevelsByMinGrade(levels, "B")
	if len(gotB) != 3 {
		t.Fatalf("min_grade B → 3 levels, got %d", len(gotB))
	}
	for _, l := range gotB {
		if l.Grade == "C" {
			t.Fatal("min_grade B must drop grade C")
		}
	}
	// A → only grade-A survive; the owner level (A) must remain
	gotA := FilterLevelsByMinGrade(levels, "A")
	if len(gotA) != 2 {
		t.Fatalf("min_grade A → 2 grade-A levels, got %d", len(gotA))
	}
	var sawOwner bool
	for _, l := range gotA {
		if l.Kind == KindOwner {
			sawOwner = true
		}
	}
	if !sawOwner {
		t.Fatal("owner level (grade A) must survive min_grade A (always seated)")
	}
}
