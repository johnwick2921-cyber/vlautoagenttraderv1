package store

import (
	"path/filepath"
	"testing"
)

// ITEM 15 — the read path for plan version history.
//
// The plans table has been append-only since P3.6 (PRIMARY KEY (plan_id,
// version); AppendPlan only ever INSERTs max(version)+1), so every superseded
// version's doc and trigger_reason has been durable all along. Nothing could
// read it: GetLatestPlanForSession returns one row, ListRecent is a global feed
// across every plan_id and trader. That is why the owner's v1..v6 chips could
// only ever be decoration.

func TestListVersionsReturnsEveryVersionOldestFirst(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	// The real shape of 2026-08-16: ASIA ran to six versions, NY to two.
	for i := 1; i <= 6; i++ {
		lifecycle := "active"
		reason := "death condition"
		if i == 1 {
			reason = "session open"
		}
		if i == 6 {
			lifecycle, reason = "no_trade", "replans_exhausted"
		}
		if _, err := st.Plan().AppendPlan(&PlanDB{
			PlanID: MakePlanID("2026-08-16", "ASIA"), StrategyID: "trader-1",
			TradeDate: "2026-08-16", Session: "ASIA", TriggerReason: reason,
			Lifecycle: lifecycle, Doc: `{"reasoning":"r"}`,
		}); err != nil {
			t.Fatal(err)
		}
	}
	for i := 1; i <= 2; i++ {
		if _, err := st.Plan().AppendPlan(&PlanDB{
			PlanID: MakePlanID("2026-08-16", "NY"), StrategyID: "trader-1",
			TradeDate: "2026-08-16", Session: "NY", Lifecycle: "active", Doc: `{}`,
		}); err != nil {
			t.Fatal(err)
		}
	}

	got, err := st.Plan().ListVersions("2026-08-16", "ASIA")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 6 {
		t.Fatalf("ASIA versions = %d, want all 6 — a superseded version is not deleted, it is superseded", len(got))
	}
	for i, r := range got {
		if r.Version != i+1 {
			t.Errorf("row %d has version %d, want %d — oldest first", i, r.Version, i+1)
		}
		if r.Session != "ASIA" {
			t.Errorf("row %d leaked session %q — a sibling session must never appear here", i, r.Session)
		}
	}
	if got[5].Lifecycle != "no_trade" || got[5].TriggerReason != "replans_exhausted" {
		t.Errorf("v6 = %s/%s, want no_trade/replans_exhausted — the reason the plan gave up must survive",
			got[5].Lifecycle, got[5].TriggerReason)
	}
	// The superseded versions are still ACTIVE-lifecycle rows: nothing rewrites
	// them, which is what makes the history honest.
	if got[0].Lifecycle != "active" {
		t.Errorf("v1 lifecycle = %q, want the value it was written with", got[0].Lifecycle)
	}

	if n, _ := st.Plan().ListVersions("2026-08-16", "NY"); len(n) != 2 {
		t.Errorf("NY versions = %d, want 2 — sessions must not bleed into each other", len(n))
	}
	if none, err := st.Plan().ListVersions("2026-08-16", "LONDON"); err != nil || len(none) != 0 {
		t.Errorf("a session with no plan must return an empty list, got %d (err %v)", len(none), err)
	}
}
