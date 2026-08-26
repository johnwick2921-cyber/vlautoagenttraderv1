package store

import (
	"path/filepath"
	"testing"
)

// P5.6 — matched-random store: per-type aggregation + the weekly SaveIfAbsent
// dedupe (the "first-writer-wins, no re-peek" guarantee across traders).

func newMRTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestMatchedRandomCountsByType(t *testing.T) {
	st := newMRTestStore(t)
	mr := st.MatchedRandom()
	rows := []struct {
		typ     string
		reacted bool
	}{
		{"PDH/PDL/PDC", true}, {"PDH/PDL/PDC", true}, {"PDH/PDL/PDC", false},
		{"ONH/ONL", true}, {"ONH/ONL", false},
	}
	for _, r := range rows {
		if err := mr.RecordTouch(r.typ, r.reacted, "2026-08-17", 1); err != nil {
			t.Fatalf("record: %v", err)
		}
	}
	counts, err := mr.CountsByType()
	if err != nil {
		t.Fatalf("counts: %v", err)
	}
	if counts["PDH/PDL/PDC"].Touches != 3 || counts["PDH/PDL/PDC"].Reactions != 2 {
		t.Fatalf("PDH tally wrong: %+v", counts["PDH/PDL/PDC"])
	}
	if counts["ONH/ONL"].Touches != 2 || counts["ONH/ONL"].Reactions != 1 {
		t.Fatalf("ONH tally wrong: %+v", counts["ONH/ONL"])
	}
}

func TestMatchedRandomRecordRequiresType(t *testing.T) {
	st := newMRTestStore(t)
	if err := st.MatchedRandom().RecordTouch("", true, "d", 1); err == nil {
		t.Fatal("empty level_type must error")
	}
}

func TestWeeklySaveIfAbsentDedupes(t *testing.T) {
	st := newMRTestStore(t)
	mr := st.MatchedRandom()
	wrote1, err := mr.SaveWeeklyIfAbsent("2026-W33", `[{"level_type":"PDH/PDL/PDC","status":"WARMING"}]`, 100)
	if err != nil {
		t.Fatalf("save1: %v", err)
	}
	wrote2, err := mr.SaveWeeklyIfAbsent("2026-W33", `[{"level_type":"PDH/PDL/PDC","status":"BEATS-RANDOM"}]`, 200)
	if err != nil {
		t.Fatalf("save2: %v", err)
	}
	if !wrote1 || wrote2 {
		t.Fatalf("first save must win, second is a no-op: wrote1=%v wrote2=%v", wrote1, wrote2)
	}
	if !mr.HasWeekly("2026-W33") {
		t.Fatal("HasWeekly must be true after a save")
	}
	// the FROZEN snapshot is the first write, never the re-peek.
	latest, _ := mr.LatestWeekly()
	if latest == nil || latest.ComputedAt != 100 {
		t.Fatalf("frozen snapshot must be the first write (computed_at=100): %+v", latest)
	}
}
