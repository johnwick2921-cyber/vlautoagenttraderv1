package store

import (
	"fmt"
	"path/filepath"
	"testing"
)

// Owner ruling 2026-09-01: raise the rejected-prompt cap from 20 to 200 so
// class 39 (leg normalization) has a sample to work from. At 20 the store held
// roughly ONE session's rejects — the class-38 forensics found n=1 usable
// instance of the defect it was investigating because the rest had been
// trimmed. 200 spans several sessions of a bad night (the 72h to 2026-09-01
// carried 121 validator rejects) without unbounding disk: prompts run ~25 KB,
// so 200 rows ≈ 5 MB.
func TestPlannerRejectedCapIsTwoHundred(t *testing.T) {
	if plannerRejectedCap != 200 {
		t.Fatalf("plannerRejectedCap = %d, want 200 (owner ruling 2026-09-01: class 39 needs a sample)", plannerRejectedCap)
	}
}

// The trim must still bound the store at exactly the cap — a raised cap that
// silently stopped trimming would be an unbounded verbatim-prompt table.
func TestPlannerRejectedTrimHoldsAtCap(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "cap.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	ps := st.PlannerRejected()

	// One row past the cap, so the trim has to fire exactly once.
	for i := 0; i < plannerRejectedCap+5; i++ {
		if err := ps.SaveRejectedPrompt("t", "2026-09-01", "ASIA", "h", 1,
			fmt.Sprintf("reject %d", i), fmt.Sprintf("PROMPT %d", i)); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}
	var count int64
	if err := st.gdb.Model(&PlannerRejectedPrompt{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != int64(plannerRejectedCap) {
		t.Fatalf("trim: count=%d want %d", count, plannerRejectedCap)
	}

	// The NEWEST rows survive — a sample that kept the oldest would be useless.
	row, err := ps.Latest()
	if err != nil || row == nil {
		t.Fatalf("latest: %v %+v", err, row)
	}
	want := fmt.Sprintf("PROMPT %d", plannerRejectedCap+4)
	if row.PromptText != want {
		t.Fatalf("newest row lost: got %q want %q", row.PromptText, want)
	}
}
