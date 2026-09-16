package store

import (
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// P0.2 — plans / plan_overlays append-only tables + decisions FK + WAL +
// single-writer goroutine. These tests are written FIRST and lock the
// append-only version monotonicity (the reason a single-writer goroutine
// exists), the json_valid CHECK, and the additive decision FK columns.

func newPlanTestStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() {
		st.Plan().Close()
		_ = st.Close()
	})
	return st
}

// samplePlan builds a plan whose (trade_date, session) is consistent with its
// plan_id — the MakePlanID invariant ("<trade_date>:<session>") the unique
// (trade_date, session, version) index enforces.
func samplePlan(planID string) *PlanDB {
	tradeDate, session := "2026-08-14", "NY"
	if i := strings.IndexByte(planID, ':'); i >= 0 {
		tradeDate, session = planID[:i], planID[i+1:]
	}
	return &PlanDB{
		PlanID:        planID,
		StrategyID:    "strat-1",
		TradeDate:     tradeDate,
		Session:       session,
		TriggerReason: "scheduled",
		Lifecycle:     "active",
		ModelID:       "deepseek-reasoner",
		PromptHash:    "abc123",
		Doc:           `{"bias":"long","levels":[]}`,
	}
}

// TestWALEnabled proves the journal mode flipped to WAL (spec storage req).
func TestWALEnabled(t *testing.T) {
	st := newPlanTestStore(t)
	var mode string
	if err := st.GormDB().Raw("PRAGMA journal_mode").Scan(&mode).Error; err != nil {
		t.Fatalf("read journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Fatalf("journal_mode = %q, want wal", mode)
	}
}

// TestAppendPlanAssignsVersions: append-only versions start at 1 and increment
// per plan_id; a different plan_id has its own sequence.
func TestAppendPlanAssignsVersions(t *testing.T) {
	st := newPlanTestStore(t)
	ps := st.Plan()

	v1, err := ps.AppendPlan(samplePlan("2026-08-14:NY"))
	if err != nil {
		t.Fatalf("append v1: %v", err)
	}
	v2, err := ps.AppendPlan(samplePlan("2026-08-14:NY"))
	if err != nil {
		t.Fatalf("append v2: %v", err)
	}
	if v1 != 1 || v2 != 2 {
		t.Fatalf("versions = %d,%d want 1,2", v1, v2)
	}
	// A separate plan_id has its own sequence.
	other, err := ps.AppendPlan(samplePlan("2026-08-14:ASIA"))
	if err != nil {
		t.Fatalf("append other: %v", err)
	}
	if other != 1 {
		t.Fatalf("other-plan first version = %d want 1", other)
	}
}

// TestAppendPlanConcurrentNoCollision is the core single-writer proof:
// concurrent appends to the SAME plan_id must yield versions {1..N} with no
// gaps, no dupes, no PK violation. Read-max + insert is two statements;
// MaxOpenConns(1) alone does NOT make them atomic — the single-writer
// goroutine does.
func TestAppendPlanConcurrentNoCollision(t *testing.T) {
	st := newPlanTestStore(t)
	ps := st.Plan()

	const n = 25
	var wg sync.WaitGroup
	versions := make([]int, n)
	errs := make([]error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			versions[i], errs[i] = ps.AppendPlan(samplePlan("2026-08-14:NY"))
		}(i)
	}
	wg.Wait()

	seen := map[int]bool{}
	for i := 0; i < n; i++ {
		if errs[i] != nil {
			t.Fatalf("append %d errored: %v", i, errs[i])
		}
		if versions[i] < 1 || versions[i] > n {
			t.Fatalf("version %d out of range: %d", i, versions[i])
		}
		if seen[versions[i]] {
			t.Fatalf("duplicate version %d", versions[i])
		}
		seen[versions[i]] = true
	}
	for v := 1; v <= n; v++ {
		if !seen[v] {
			t.Fatalf("missing version %d (gap)", v)
		}
	}
}

// TestAppendPlanRejectsInvalidJSON proves the CHECK(json_valid(doc)) constraint.
func TestAppendPlanRejectsInvalidJSON(t *testing.T) {
	st := newPlanTestStore(t)
	bad := samplePlan("2026-08-14:NY")
	bad.Doc = `{not valid json`
	if _, err := st.Plan().AppendPlan(bad); err == nil {
		t.Fatalf("expected json_valid CHECK to reject invalid doc")
	}
}

// TestGetLatestPlan / ForSession read the newest version back.
func TestGetLatestPlan(t *testing.T) {
	st := newPlanTestStore(t)
	ps := st.Plan()
	for i := 0; i < 3; i++ {
		if _, err := ps.AppendPlan(samplePlan("2026-08-14:NY")); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	got, err := ps.GetLatestPlan("2026-08-14:NY")
	if err != nil {
		t.Fatalf("get latest: %v", err)
	}
	if got == nil || got.Version != 3 {
		t.Fatalf("latest version = %+v want 3", got)
	}
	bySession, err := ps.GetLatestPlanForSession("2026-08-14", "NY")
	if err != nil {
		t.Fatalf("get by session: %v", err)
	}
	if bySession == nil || bySession.Version != 3 || bySession.PlanID != "2026-08-14:NY" {
		t.Fatalf("by-session latest = %+v", bySession)
	}
}

// TestAppendOverlayMonotonic: overlay_version increments per (plan_id, plan_version).
func TestAppendOverlayMonotonic(t *testing.T) {
	st := newPlanTestStore(t)
	ps := st.Plan()
	if _, err := ps.AppendPlan(samplePlan("2026-08-14:NY")); err != nil {
		t.Fatalf("append plan: %v", err)
	}
	o1, err := ps.AppendOverlay(&PlanOverlayDB{
		OverlayID: "ov-a", PlanID: "2026-08-14:NY", PlanVersion: 1,
		Patch: `[{"op":"replace","path":"/bias","value":"short"}]`, Origin: "owner",
	})
	if err != nil {
		t.Fatalf("append overlay 1: %v", err)
	}
	o2, err := ps.AppendOverlay(&PlanOverlayDB{
		OverlayID: "ov-b", PlanID: "2026-08-14:NY", PlanVersion: 1,
		Patch: `[{"op":"add","path":"/levels/-","value":30000}]`, Origin: "ai",
	})
	if err != nil {
		t.Fatalf("append overlay 2: %v", err)
	}
	if o1 != 1 || o2 != 2 {
		t.Fatalf("overlay versions = %d,%d want 1,2", o1, o2)
	}
	list, err := ps.ListOverlays("2026-08-14:NY", 1)
	if err != nil {
		t.Fatalf("list overlays: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("overlay count = %d want 2", len(list))
	}
}

// TestAppendOverlayRejectsInvalidPatch proves the patch json_valid CHECK.
func TestAppendOverlayRejectsInvalidPatch(t *testing.T) {
	st := newPlanTestStore(t)
	ps := st.Plan()
	if _, err := ps.AppendPlan(samplePlan("2026-08-14:NY")); err != nil {
		t.Fatalf("append plan: %v", err)
	}
	_, err := ps.AppendOverlay(&PlanOverlayDB{
		OverlayID: "ov-x", PlanID: "2026-08-14:NY", PlanVersion: 1, Patch: `[oops`, Origin: "owner",
	})
	if err == nil {
		t.Fatalf("expected json_valid CHECK to reject invalid patch")
	}
}

// TestDecisionPlanFKRoundTrip proves the 4 additive FK columns persist + read.
func TestDecisionPlanFKRoundTrip(t *testing.T) {
	st := newPlanTestStore(t)
	rec := &DecisionRecord{
		TraderID:        "trader-1",
		CycleNumber:     7,
		PlanID:          "2026-08-14:NY",
		PlanVersion:     2,
		OverlayVersion:  1,
		CitedScenarioID: "S2",
	}
	if err := st.Decision().LogDecision(rec); err != nil {
		t.Fatalf("log decision: %v", err)
	}
	got, err := st.Decision().GetLatestRecords("trader-1", 1)
	if err != nil {
		t.Fatalf("get records: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("record count = %d want 1", len(got))
	}
	r := got[0]
	if r.PlanID != "2026-08-14:NY" || r.PlanVersion != 2 || r.OverlayVersion != 1 || r.CitedScenarioID != "S2" {
		t.Fatalf("FK round-trip mismatch: %+v", r)
	}
}

// TestDecisionFKDefaultsEmpty: a decision that cites no plan stores empty/zero
// FK fields (additive; existing decision logging is unchanged).
func TestDecisionFKDefaultsEmpty(t *testing.T) {
	st := newPlanTestStore(t)
	if err := st.Decision().LogDecision(&DecisionRecord{TraderID: "t2", CycleNumber: 1}); err != nil {
		t.Fatalf("log: %v", err)
	}
	got, err := st.Decision().GetLatestRecords("t2", 1)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got[0].PlanID != "" || got[0].PlanVersion != 0 || got[0].OverlayVersion != 0 || got[0].CitedScenarioID != "" {
		t.Fatalf("expected empty FK defaults, got %+v", got[0])
	}
}
