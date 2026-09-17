package telemetry

import "testing"

// D1'(3) narrow: a confirmed break is counted with the bars it cost, and the
// reader returns what was recorded — never a recomputation (A11).
func TestScaleBreakDropIsCountedWithItsBars(t *testing.T) {
	e0, b0 := ScaleBreakCounts()
	IncScaleBreakDrop(1999)
	IncScaleBreakDrop(1832)
	e1, b1 := ScaleBreakCounts()
	if e1-e0 != 2 || b1-b0 != 3831 {
		t.Fatalf("events +%d bars +%d, want +2 / +3831 (the 09-16 09:22 pair)", e1-e0, b1-b0)
	}
}
