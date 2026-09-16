package store

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// E6 — the E8 counterfactual row carries the class-39 stamp, on create AND on
// the upsert-update path.
func TestClass39E8RowCarriesNormalizationStamp(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "e8.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	ac := st.AbConfirm()
	if err := ac.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	now := time.Now()
	row := &AbConfirmLogDB{
		TraderID: "t", PlanID: "2026-09-01:ASIA:t", Version: 4, Session: "ASIA", Scenario: "S1", Rule: "1x5m_close",
		Condition: "breakdown_continue", Normalized: true,
		DroppedLegs: `[{"entry":29130,"stop":29168,"target":29040,"size":1,"rule":"touch"}]`,
		CreatedAt:   now, UpdatedAt: now,
	}
	if err := ac.Upsert(row); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	var got AbConfirmLogDB
	if err := st.gdb.Where("plan_id = ? AND scenario = ? AND rule = ?", row.PlanID, "S1", "1x5m_close").First(&got).Error; err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !got.Normalized || got.DroppedLegs != row.DroppedLegs {
		t.Fatalf("stamp lost on create: normalized=%v dropped_legs=%q", got.Normalized, got.DroppedLegs)
	}
	// Update path must carry the columns too (the Updates map is explicit).
	row.Normalized = false
	row.DroppedLegs = ""
	if err := ac.Upsert(row); err != nil {
		t.Fatalf("upsert update: %v", err)
	}
	if err := st.gdb.Where("id = ?", got.ID).First(&got).Error; err != nil {
		t.Fatalf("re-read: %v", err)
	}
	if got.Normalized || got.DroppedLegs != "" {
		t.Fatalf("stamp not updated on the upsert path: normalized=%v dropped_legs=%q", got.Normalized, got.DroppedLegs)
	}
}

// E7 — the counter is RECORDED: it survives a store close/reopen (a restart).
func TestClass39CounterPersistsAcrossRestart(t *testing.T) {
	path := filepath.Join(t.TempDir(), "ctr.db")
	st, err := New(path)
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	if n := ArmsNormalizedCount(st); n != 0 {
		t.Fatalf("fresh counter = %d, want 0", n)
	}
	for i := 1; i <= 2; i++ {
		n, err := IncArmsNormalized(st)
		if err != nil || n != i {
			t.Fatalf("inc %d: n=%d err=%v", i, n, err)
		}
	}
	st.Close()
	st2, err := New(path) // the restart
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer st2.Close()
	if n := ArmsNormalizedCount(st2); n != 2 {
		t.Fatalf("counter did not survive the restart: %d, want 2", n)
	}
	if n, _ := IncArmsNormalized(st2); n != 3 {
		t.Fatalf("post-restart increment = %d, want 3", n)
	}
}

// E8 — the 201st rejected prompt trims the OLDEST; survivors are the newest;
// the cap is 200.
func TestClass39RejectedCapTrimsOldestAt201(t *testing.T) {
	if plannerRejectedCap != 200 {
		t.Fatalf("plannerRejectedCap = %d, want 200 (owner ruling 2026-09-01)", plannerRejectedCap)
	}
	st, err := New(filepath.Join(t.TempDir(), "cap.db"))
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	defer st.Close()
	ps := st.PlannerRejected()
	for i := 0; i <= plannerRejectedCap; i++ { // 0..200 = 201 rows
		if err := ps.SaveRejectedPrompt("t", "2026-09-01", "ASIA", "h", 1, fmt.Sprintf("r%d", i), fmt.Sprintf("p%d", i)); err != nil {
			t.Fatalf("save %d: %v", i, err)
		}
	}
	var count int64
	if err := st.gdb.Model(&PlannerRejectedPrompt{}).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != int64(plannerRejectedCap) {
		t.Fatalf("after the 201st insert count=%d want %d", count, plannerRejectedCap)
	}
	var oldest int64
	st.gdb.Model(&PlannerRejectedPrompt{}).Where("reject_reason = ?", "r0").Count(&oldest)
	if oldest != 0 {
		t.Fatalf("the OLDEST row (r0) must be the one trimmed")
	}
	latest, err := ps.Latest()
	if err != nil || latest == nil || latest.RejectReason != fmt.Sprintf("r%d", plannerRejectedCap) {
		t.Fatalf("newest row must survive: %+v err=%v", latest, err)
	}
}
