package store

import (
	"path/filepath"
	"testing"
)

// P6 — the reset seam: the marker's persistence. The original chain is baseline
// 1; after an owner reset the baseline moves to the new chain's first version.
// CLASS 35 (2026-09-01): the budget itself is a RECORDED counter keyed under the
// baseline (see replan_budget_test.go) — the version arithmetic is gone.

func TestResetBaselinePersistence(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })

	// Absent → the original chain baseline.
	if got := GetResetBaseline(st, "tid-1", "2026-08-18", "NY"); got != 1 {
		t.Fatalf("absent marker = %d, want 1", got)
	}
	if err := SetResetBaseline(st, "tid-1", "2026-08-18", "NY", 7); err != nil {
		t.Fatalf("set: %v", err)
	}
	if got := GetResetBaseline(st, "tid-1", "2026-08-18", "NY"); got != 7 {
		t.Fatalf("round-trip = %d, want 7", got)
	}
	// Per-session scoping: another session keeps the original baseline.
	if got := GetResetBaseline(st, "tid-1", "2026-08-18", "ASIA"); got != 1 {
		t.Fatalf("ASIA must keep baseline 1, got %d", got)
	}
	// A malformed value can never inflate or destroy budget.
	_ = st.SetSystemConfig(ResetBaselineKey("tid-1", "2026-08-18", "LONDON"), "not-a-number")
	if got := GetResetBaseline(st, "tid-1", "2026-08-18", "LONDON"); got != 1 {
		t.Fatalf("malformed marker = %d, want the safe fallback 1", got)
	}
}
