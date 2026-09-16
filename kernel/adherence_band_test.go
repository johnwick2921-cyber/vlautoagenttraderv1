package kernel

import "testing"

// B3 (F6) — the spec test: a same-direction but off-band entry grades B not A.
func TestStrictAdherenceBandGrades(t *testing.T) {
	base := AdherenceInput{Cited: true, Matched: true, InKillzone: true}
	if g, _ := GradeAdherence(base); g != "A" {
		t.Fatalf("in-band matched cite = %s, want A", g)
	}
	off := base
	off.Band = "off_band"
	if g, reasons := GradeAdherence(off); g != "B" {
		t.Fatalf("same-direction off-band entry = %s, want B (%v)", g, reasons)
	}
	st := base
	st.Band = "struct"
	if g, _ := GradeAdherence(st); g != "B" {
		t.Fatalf("SL/TP-inconsistent cite = %s, want B", g)
	}
	// Forward-only: legacy rows ("" band) keep the old A.
	legacy := base
	legacy.Band = ""
	if g, _ := GradeAdherence(legacy); g != "A" {
		t.Fatalf("legacy row regraded to %s — must stay A (no historical rewrite)", g)
	}
}

// CitationStructure verdicts on a real-shaped doc.
func TestCitationStructure(t *testing.T) {
	doc := PlanDoc{
		Levels:    []PlanLevel{{Price: 29648.25, Label: "OR-L", Grade: "A"}},
		Scenarios: []PlanScenario{{ID: "S1", Trigger: "reject at 29648.25", Condition: "reject", Direction: "short", TargetChain: []float64{29514}, Invalid: "15m close above 29648.25", Quality: "B"}},
	}
	dATR := 100.0 // band = 150
	// entry near the anchor, protective SL above, TP below → ok
	if v := CitationStructure("open_short", "S1", doc, 29650, 29672, 29514, dATR); v != "ok" {
		t.Fatalf("clean structure = %q, want ok", v)
	}
	// entry 300 pts away → off_band
	if v := CitationStructure("open_short", "S1", doc, 29950, 29999, 29514, dATR); v != "off_band" {
		t.Fatalf("far entry = %q, want off_band", v)
	}
	// TP pointing the WRONG way for a short → struct
	if v := CitationStructure("open_short", "S1", doc, 29650, 29672, 29750, dATR); v != "struct" {
		t.Fatalf("wrong-way TP = %q, want struct", v)
	}
	// off-plan / unknown → "" (fail-open)
	if v := CitationStructure("open_short", "off-plan", doc, 29650, 29672, 29514, dATR); v != "" {
		t.Fatalf("off-plan = %q, want empty", v)
	}
}
