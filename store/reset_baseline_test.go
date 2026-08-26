package store

import (
	"path/filepath"
	"testing"
)

// P6 — the reset seam: baseline-relative budget math + the marker's persistence.
// The original chain is baseline 1; after an owner reset the baseline moves to
// the new chain's first version, so the budget math is identical, just re-based.

func TestReplansUsedFromBaseline(t *testing.T) {
	cases := []struct {
		version, baseline, want int
	}{
		{1, 1, 0}, // original chain v1 is free
		{3, 1, 2}, // original chain
		{7, 7, 0}, // the reset chain's first plan is free
		{9, 7, 2}, // two re-plans into the reset chain
		{5, 7, 0}, // a version before the baseline can never spend
		{9, 0, 8}, // a bad baseline falls back to 1
	}
	for _, c := range cases {
		if got := ReplansUsedFrom(c.version, c.baseline); got != c.want {
			t.Errorf("ReplansUsedFrom(%d, %d) = %d, want %d", c.version, c.baseline, got, c.want)
		}
	}
	// The old API stays byte-identical: baseline 1.
	if got := ReplansUsed(3); got != 2 {
		t.Errorf("ReplansUsed(3) = %d, want 2", got)
	}
}

func TestMayReplanAndLeftFromBaseline(t *testing.T) {
	// cap 4, reset baseline 7: v7 free, v8..v11 are the four re-plans, v11's death
	// is the ceiling (NO-TRADE).
	if !MayReplanFrom(7, 7, 4) || ReplansLeftFrom(7, 7, 4) != 4 {
		t.Fatalf("the reset chain's first plan must have the full budget")
	}
	if !MayReplanFrom(10, 7, 4) || ReplansLeftFrom(10, 7, 4) != 1 {
		t.Fatalf("v10 = 3 re-plans used, one left")
	}
	if MayReplanFrom(11, 7, 4) || ReplansLeftFrom(11, 7, 4) != 0 {
		t.Fatalf("v11 = 4 re-plans used, budget spent")
	}
}

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
