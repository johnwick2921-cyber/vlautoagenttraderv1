package store

import (
	"path/filepath"
	"testing"
)

// W2 — the stats-window reset (§128 no cross-model pooling): ResetWindow clears
// both the verdicts and the frozen weekly snapshots.
func TestMatchedRandomResetWindow(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	mr := st.MatchedRandom()

	_ = mr.RecordTouch("PDH/PDL/PDC", true, "2026-08-17", 1)
	_ = mr.RecordTouch("ONH/ONL", false, "2026-08-17", 2)
	_, _ = mr.SaveWeeklyIfAbsent("2026-W34", `[{"level_type":"PDH/PDL/PDC","status":"WARMING"}]`, 100)

	counts, _ := mr.CountsByType()
	if len(counts) == 0 || !mr.HasWeekly("2026-W34") {
		t.Fatal("precondition: verdicts + weekly should exist")
	}

	if err := mr.ResetWindow(); err != nil {
		t.Fatalf("ResetWindow: %v", err)
	}
	counts, _ = mr.CountsByType()
	if len(counts) != 0 {
		t.Fatalf("verdicts must be cleared, got %v", counts)
	}
	if mr.HasWeekly("2026-W34") {
		t.Fatal("weekly snapshots must be cleared on reset")
	}
}
