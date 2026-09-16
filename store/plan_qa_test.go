package store

import (
	"path/filepath"
	"testing"
)

// P5.4 — Ask-Planner thread store: append/list order, trader-scoped Apply guard
// (IDOR), and the sycophancy KPI (VerdictStats) incl. the defend-on-bare signal.

func newQATestStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func TestPlanQAAppendAndListOrder(t *testing.T) {
	st := newQATestStore(t)
	qa := st.PlanQA()
	if _, err := qa.Append(&PlanQADB{TraderID: "t1", PlanID: "2026-08-17:NY", Role: "owner", Content: "why supply?", CreatedAt: 1}); err != nil {
		t.Fatalf("append owner: %v", err)
	}
	pid, err := qa.Append(&PlanQADB{TraderID: "t1", PlanID: "2026-08-17:NY", Role: "planner", Content: "holding", PointClass: "BARE-DISAGREEMENT", Verdict: "DEFEND", CreatedAt: 2})
	if err != nil {
		t.Fatalf("append planner: %v", err)
	}
	msgs, err := qa.ListForPlan("t1", "2026-08-17:NY", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(msgs) != 2 || msgs[0].Role != "owner" || msgs[1].Role != "planner" {
		t.Fatalf("thread order wrong: %+v", msgs)
	}
	if pid != msgs[1].ID {
		t.Fatalf("planner id mismatch")
	}
}

func TestPlanQAAppendRequiresIDs(t *testing.T) {
	st := newQATestStore(t)
	if _, err := st.PlanQA().Append(&PlanQADB{Role: "owner", Content: "x"}); err == nil {
		t.Fatal("append without trader_id/plan_id must error")
	}
}

func TestPlanQAGetForTraderIDOR(t *testing.T) {
	st := newQATestStore(t)
	qa := st.PlanQA()
	id, _ := qa.Append(&PlanQADB{TraderID: "owner", PlanID: "p", Role: "planner", Verdict: "PROPOSE-MERGE", Patch: "[]", CreatedAt: 1})
	// wrong trader → not found
	got, err := qa.GetForTrader("attacker", id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got != nil {
		t.Fatal("GetForTrader must not return another trader's message")
	}
	// owner → found
	got, _ = qa.GetForTrader("owner", id)
	if got == nil || got.ID != id {
		t.Fatal("owner must retrieve own message")
	}
	// apply is scoped too
	if err := qa.MarkApplied("owner", id); err != nil {
		t.Fatalf("mark: %v", err)
	}
	got, _ = qa.GetForTrader("owner", id)
	if !got.Applied {
		t.Fatal("MarkApplied did not set applied")
	}
}

func TestPlanQAVerdictStatsKPI(t *testing.T) {
	st := newQATestStore(t)
	qa := st.PlanQA()
	rows := []*PlanQADB{
		{TraderID: "t1", PlanID: "p", Role: "planner", PointClass: "BARE-DISAGREEMENT", Verdict: "DEFEND", CreatedAt: 1},
		{TraderID: "t1", PlanID: "p", Role: "planner", PointClass: "NEW-INFO", Verdict: "PROPOSE-MERGE", Patch: "[{}]", Applied: true, CreatedAt: 2},
		{TraderID: "t1", PlanID: "p", Role: "planner", PointClass: "NEW-INFO", Verdict: "CONCEDE", CreatedAt: 3},
		{TraderID: "t1", PlanID: "p", Role: "owner", Content: "q", CreatedAt: 4}, // owner rows excluded
	}
	for _, r := range rows {
		if _, err := qa.Append(r); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	kpi, err := qa.VerdictStats("t1")
	if err != nil {
		t.Fatalf("stats: %v", err)
	}
	if kpi.Total != 3 || kpi.BareDisagree != 1 || kpi.NewInfo != 2 {
		t.Fatalf("class counts wrong: %+v", kpi)
	}
	if kpi.Defend != 1 || kpi.ProposeMerge != 1 || kpi.Concede != 1 {
		t.Fatalf("verdict counts wrong: %+v", kpi)
	}
	if kpi.Applied != 1 || kpi.DefendOnBare != 1 {
		t.Fatalf("applied/defend-on-bare wrong: %+v", kpi)
	}
}
