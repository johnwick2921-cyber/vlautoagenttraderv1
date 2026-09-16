package store

import (
	"path/filepath"
	"testing"
)

// S3 (mega-research 2026-08-26) — position→plan attribution repair.
// SetPlanLinkFull stamps the active plan's IDENTITY at open; the
// position_plan_join view joins plan_id-first and never drops unresolvables.

func TestSetPlanLinkFullStampsIdentity(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	ps := st.Position()

	pos := &TraderPosition{
		TraderID: "t1", Symbol: "MNQ", Side: "LONG", Quantity: 1, EntryPrice: 29300,
		Status: "OPEN", EntryTime: 1, CreatedAt: 1, UpdatedAt: 1,
	}
	if err := ps.Create(pos); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := ps.SetPlanLinkFull(pos.ID, 9, "S1", true, "ok", "2026-08-25:LONDON", "2026-08-25", "LONDON"); err != nil {
		t.Fatalf("SetPlanLinkFull: %v", err)
	}

	got, err := ps.GetOpenPositions("t1")
	if err != nil || len(got) != 1 {
		t.Fatalf("get: %v n=%d", err, len(got))
	}
	p := got[0]
	if p.PlanID != "2026-08-25:LONDON" || p.PlanTradeDate != "2026-08-25" || p.PlanSession != "LONDON" || p.PlanVersion != 9 {
		t.Fatalf("identity not persisted: %+v", p)
	}

	// Legacy SetPlanLink stays back-compatible (empty identity).
	if err := ps.SetPlanLink(pos.ID, 4, "S2", false, ""); err != nil {
		t.Fatalf("legacy SetPlanLink: %v", err)
	}
}

// TestPositionPlanJoinView: the canonical join resolves plan_id-first and the
// unresolvable row stays visible (LEFT JOIN null side) — counted, never dropped.
func TestPositionPlanJoinView(t *testing.T) {
	st, err := New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	// A resolvable position + a plan row (AppendPlan assigns the version).
	ps := st.Position()
	v, err := st.Plan().AppendPlan(&PlanDB{
		PlanID: "2026-08-24:LONDON", StrategyID: "t1", TradeDate: "2026-08-24",
		Session: "LONDON", Lifecycle: "active", Doc: `{"scenarios":[]}`,
	})
	if err != nil {
		t.Fatalf("append plan: %v", err)
	}
	pos := &TraderPosition{
		TraderID: "t1", Symbol: "MNQ", Side: "LONG", Quantity: 1, EntryPrice: 29300,
		Status: "OPEN", EntryTime: 1, CreatedAt: 1, UpdatedAt: 1,
	}
	if err := ps.Create(pos); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := ps.SetPlanLinkFull(pos.ID, v, "S1", true, "ok", "2026-08-24:LONDON", "2026-08-24", "LONDON"); err != nil {
		t.Fatalf("link: %v", err)
	}

	// An unresolvable position (no plan row at all).
	pos2 := &TraderPosition{
		TraderID: "t1", Symbol: "MNQ", Side: "SHORT", Quantity: 1, EntryPrice: 29350,
		Status: "OPEN", EntryTime: 2, CreatedAt: 2, UpdatedAt: 2,
	}
	if err := ps.Create(pos2); err != nil {
		t.Fatalf("create2: %v", err)
	}
	if err := ps.SetPlanLinkFull(pos2.ID, 7, "S3", true, "ok", "UNRESOLVABLE", "", ""); err != nil {
		t.Fatalf("link2: %v", err)
	}

	var rows []struct {
		PositionID   int64
		PosPlanID    string
		PlansPlanID  string
		PlansVersion int
	}
	if err := st.GormDB().Raw(`SELECT position_id, pos_plan_id, plans_plan_id, COALESCE(plans_version,0) AS plans_version FROM position_plan_join ORDER BY position_id`).Scan(&rows).Error; err != nil {
		t.Fatalf("view query: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("view rows = %d, want 2 (unresolvable counted)", len(rows))
	}
	if rows[0].PlansPlanID != "2026-08-24:LONDON" || rows[0].PlansVersion != v {
		var pv []struct {
			PlanID string
			Ver    int
		}
		_ = st.GormDB().Raw(`SELECT plan_id, version FROM plans`).Scan(&pv).Error
		var pvv []struct {
			PlanID string
			Ver    int
		}
		_ = st.GormDB().Raw(`SELECT plan_id, plan_version FROM trader_positions`).Scan(&pvv).Error
		t.Fatalf("resolvable row wrong: %+v (v=%d plans=%+v positions=%+v)", rows[0], v, pv, pvv)
	}
	if rows[1].PlansPlanID != "" || rows[1].PlansVersion != 0 {
		t.Fatalf("unresolvable row must land on the null side, not be dropped: %+v", rows[1])
	}
}
