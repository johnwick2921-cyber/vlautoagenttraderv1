package store

import (
	"path/filepath"
	"strings"
	"testing"
)

// Planner-speed wave 1.4 (2026-08-31) — the rejected-prompt store round-trips
// a verbatim prompt + reason and trims to the cap.
func TestPlannerRejectedRoundTrip(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "pr.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	ps := st.PlannerRejected()
	if err := ps.SaveRejectedPrompt("trader1", "2026-08-31", "LONDON", "hashA", 1,
		"arm on S2 needs EXACTLY 2 legs", "FULL PROMPT TEXT 1"); err != nil {
		t.Fatalf("save: %v", err)
	}
	if err := ps.SaveRejectedPrompt("trader1", "2026-08-31", "LONDON", "hashB", 2,
		"breakdown is void", "FULL PROMPT TEXT 2"); err != nil {
		t.Fatalf("save2: %v", err)
	}
	row, err := ps.Latest()
	if err != nil || row == nil {
		t.Fatalf("latest: %v %+v", err, row)
	}
	if row.PromptText != "FULL PROMPT TEXT 2" || row.RejectReason != "breakdown is void" || row.Attempt != 2 {
		t.Fatalf("latest row wrong: %+v", row)
	}
	// Cap trim: overshoot the cap by 5, exactly the cap stays. Cap-relative so
	// the assertion survives an owner ruling on the value (2026-09-01: 20 → 200).
	for i := 0; i < plannerRejectedCap+5; i++ {
		if err := ps.SaveRejectedPrompt("t", "2026-08-31", "NY", "h", 3, "r", "p"); err != nil {
			t.Fatalf("bulk save: %v", err)
		}
	}
	var count int64
	if err := st.gdb.Model(&PlannerRejectedPrompt{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != plannerRejectedCap {
		t.Fatalf("cap trim: count=%d want %d", count, plannerRejectedCap)
	}
	if !strings.Contains(row.PromptText, "FULL PROMPT") {
		t.Fatalf("verbatim text lost: %q", row.PromptText)
	}
}
