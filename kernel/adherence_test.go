package kernel

import "testing"

// P5.5 — the adherence rubric. Locks: matched+clean = A; off-plan = D; direction
// mismatch = C; no-trade-window + outside-killzone step the grade toward F; and
// that the grade is independent of P&L (no P&L input at all).

func TestGradeAdherenceMatrix(t *testing.T) {
	cases := []struct {
		name string
		in   AdherenceInput
		want string
	}{
		{"clean matched", AdherenceInput{Cited: true, Matched: true, InKillzone: true}, "A"},
		{"matched but bad window", AdherenceInput{Cited: true, Matched: true, InKillzone: true, InNoTrade: true}, "B"},
		{"matched outside killzone", AdherenceInput{Cited: true, Matched: true, InKillzone: false}, "B"},
		{"matched, no-trade AND outside kz", AdherenceInput{Cited: true, Matched: true, InKillzone: false, InNoTrade: true}, "C"},
		{"direction mismatch", AdherenceInput{Cited: true, Matched: false, InKillzone: true}, "C"},
		{"off-plan", AdherenceInput{OffPlan: true, InKillzone: true}, "D"},
		{"off-plan in no-trade outside kz", AdherenceInput{OffPlan: true, InKillzone: false, InNoTrade: true}, "F"},
	}
	for _, c := range cases {
		got, reasons := GradeAdherence(c.in)
		if got != c.want {
			t.Fatalf("%s: grade = %s want %s (reasons=%v)", c.name, got, c.want, reasons)
		}
		if len(reasons) == 0 {
			t.Fatalf("%s: expected reasons", c.name)
		}
	}
}

func TestSummarizeAdherenceGPA(t *testing.T) {
	s := SummarizeAdherence([]string{"A", "A", "C", "F", "bogus"})
	if s.Total != 4 {
		t.Fatalf("total = %d want 4 (bogus excluded)", s.Total)
	}
	// (4+4+2+0)/4 = 2.5
	if s.GPA != 2.5 {
		t.Fatalf("gpa = %v want 2.5", s.GPA)
	}
	if s.Counts["A"] != 2 {
		t.Fatalf("A count = %d want 2", s.Counts["A"])
	}
}
