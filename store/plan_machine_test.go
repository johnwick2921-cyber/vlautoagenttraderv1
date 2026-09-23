package store

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

// ── W-EXEC-TRUTH W5 foundation — the machine doors into the plan store and
// the one-opportunity-one-order ledger pin.

func machinePlanStore(t *testing.T) *Store {
	t.Helper()
	st, err := New(filepath.Join(t.TempDir(), "w5.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func machinePlanRow(trader string) *PlanDB {
	return &PlanDB{
		PlanID: MakePlanIDForTrader(trader, "2026-09-23", "NY"), TradeDate: "2026-09-23", Session: "NY",
		StrategyID: trader, TriggerReason: "machine:picture_htf", ModelID: "machine", Doc: `{"scenarios":[]}`,
	}
}

func TestAppendPlanIfAbsentWritesOnlyTheFirstRow(t *testing.T) {
	st := machinePlanStore(t)
	wrote, err := st.Plan().AppendPlanIfAbsent(machinePlanRow("t1"))
	if err != nil || !wrote {
		t.Fatalf("no plan row → the machine plan is written as v1: wrote=%v err=%v", wrote, err)
	}
	got, _ := st.Plan().GetLatestPlanForTraderSession("2026-09-23", "NY", "t1")
	if got == nil || got.Version != 1 || !IsMachinePlan(got) {
		t.Fatalf("machine plan = %+v", got)
	}
	again, err := st.Plan().AppendPlanIfAbsent(machinePlanRow("t1"))
	if err != nil || again {
		t.Fatalf("a second machine plan for the same trader+session must not be written: wrote=%v err=%v", again, err)
	}
	if other, err := st.Plan().AppendPlanIfAbsent(machinePlanRow("t2")); err != nil || !other {
		t.Fatalf("another trader's chain is its own: wrote=%v err=%v", other, err)
	}
}

func TestAppendPlanIfAbsentNeverLandsOnAnAIPlan(t *testing.T) {
	st := machinePlanStore(t)
	ai := machinePlanRow("t1")
	ai.TriggerReason, ai.ModelID = "NY_scheduled_read", "deepseek"
	if _, err := st.Plan().AppendPlan(ai); err != nil {
		t.Fatal(err)
	}
	wrote, err := st.Plan().AppendPlanIfAbsent(machinePlanRow("t1"))
	if err != nil || wrote {
		t.Fatalf("an AI plan exists → the machine door writes nothing: wrote=%v err=%v", wrote, err)
	}
	rows, _ := st.Plan().ListVersionsForTrader("2026-09-23", "NY", "t1")
	if len(rows) != 1 || IsMachinePlan(rows[0]) {
		t.Fatalf("the chain must still be the AI plan alone: %+v", rows)
	}
	// A chain under the LEGACY (trader-less) plan id is still this trader's
	// session plan: the (trade_date, session, trader) check is what refuses.
	legacy := machinePlanStore(t)
	old := &PlanDB{PlanID: MakePlanID("2026-09-23", "NY"), TradeDate: "2026-09-23", Session: "NY", StrategyID: "t1", TriggerReason: "NY_scheduled_read", Doc: `{}`}
	if _, err := legacy.Plan().AppendPlan(old); err != nil {
		t.Fatal(err)
	}
	if wrote, err := legacy.Plan().AppendPlanIfAbsent(machinePlanRow("t1")); err != nil || wrote {
		t.Fatalf("a legacy-id AI chain for the session must block the machine door: wrote=%v err=%v", wrote, err)
	}
}

// The AI read supersedes a machine plan by appending v2 to the same chain.
func TestTheAIPlanSupersedesAMachinePlanAsV2(t *testing.T) {
	st := machinePlanStore(t)
	m := machinePlanRow("t1")
	if wrote, err := st.Plan().AppendPlanIfAbsent(m); err != nil || !wrote {
		t.Fatal(err)
	}
	pid := st.Plan().ResolvePlanID("2026-09-23", "NY", "t1")
	if pid != m.PlanID {
		t.Fatalf("the AI read must continue the machine chain: %s vs %s", pid, m.PlanID)
	}
	v, err := st.Plan().AppendPlan(&PlanDB{PlanID: pid, TradeDate: "2026-09-23", Session: "NY", StrategyID: "t1", TriggerReason: "NY_scheduled_read", Doc: `{}`})
	if err != nil || v != 2 {
		t.Fatalf("AI plan = v%d err=%v", v, err)
	}
}

func TestAppendOverlayCheckedRunsTheCheckInsideTheWriter(t *testing.T) {
	st := machinePlanStore(t)
	pid := MakePlanIDForTrader("t1", "2026-09-23", "NY")
	if _, err := st.Plan().AppendOverlay(&PlanOverlayDB{PlanID: pid, PlanVersion: 3, Patch: `[]`, Origin: "owner"}); err != nil {
		t.Fatal(err)
	}
	var sawOwner bool
	v, ok, err := st.Plan().AppendOverlayChecked(&PlanOverlayDB{PlanID: pid, PlanVersion: 3, Patch: `[]`, Origin: "machine:picture_htf", OverlayID: "opp-a"},
		func(existing []*PlanOverlayDB) (bool, error) {
			for _, e := range existing {
				sawOwner = sawOwner || e.Origin == "owner"
				if e.OverlayID == "opp-a" {
					return true, nil
				}
			}
			return false, nil
		})
	if err != nil || !ok || v != 2 || !sawOwner {
		t.Fatalf("first machine overlay: v=%d ok=%v err=%v sawOwner=%v", v, ok, err, sawOwner)
	}
	v, ok, err = st.Plan().AppendOverlayChecked(&PlanOverlayDB{PlanID: pid, PlanVersion: 3, Patch: `[]`, Origin: "machine:picture_htf", OverlayID: "opp-a"},
		func(existing []*PlanOverlayDB) (bool, error) {
			for _, e := range existing {
				if e.OverlayID == "opp-a" {
					return true, nil
				}
			}
			return false, nil
		})
	if err != nil || ok || v != 0 {
		t.Fatalf("the same opportunity must be skipped: v=%d ok=%v err=%v", v, ok, err)
	}
	if _, ok, err := st.Plan().AppendOverlayChecked(&PlanOverlayDB{PlanID: pid, PlanVersion: 3, Patch: `[]`},
		func([]*PlanOverlayDB) (bool, error) { return false, fmt.Errorf("refused") }); ok || err == nil {
		t.Fatal("a check error refuses the append")
	}
	rows, _ := st.Plan().ListOverlays(pid, 3)
	if len(rows) != 2 {
		t.Fatalf("exactly the owner overlay + one machine overlay, got %d", len(rows))
	}
}

// ── the ledger ─────────────────────────────────────────────────────────────

func pictureRow(version int, scenario, ref, state, signal string) *ArmedOrderDB {
	now := time.Date(2026, 9, 23, 15, 0, 0, 0, time.UTC)
	until := now.Add(10 * time.Second).UnixMilli()
	return &ArmedOrderDB{
		TraderID: "t1", PlanID: "2026-09-23:NY:t1", Version: version, Session: "NY", Scenario: scenario, Side: "long",
		EntryPx: 21530, StopPx: 21510, TargetPx: 21560, State: state, SignalID: signal, CreatedAt: now, UpdatedAt: now,
		Policy: ArmPolicyMarketInZone, Kind: "limit", Source: ArmSourcePicture, SourceRef: ref, SourceRule: "h1_close_break",
		EligibleUntilMs: &until,
	}
}

func pictureRows(t *testing.T, st *ArmedOrderStore, ref string) []ArmedOrderDB {
	t.Helper()
	var rows []ArmedOrderDB
	if err := st.db.Where("source_ref = ?", ref).Order("id").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	return rows
}

func TestArmedSourceMigrateAddsColumns(t *testing.T) {
	db := newArmedTestDB(t)
	for _, c := range []string{"source", "source_ref", "source_rule", "eligible_until_ms", "source_run_epoch"} {
		var n int64
		if err := db.Raw("SELECT COUNT(*) FROM pragma_table_info('armed_orders') WHERE name = ?", c).Scan(&n).Error; err != nil || n != 1 {
			t.Fatalf("column %s missing after Migrate (n=%d err=%v)", c, n, err)
		}
	}
	st := NewArmedOrderStore(db)
	legacy := zoneRow(1)
	if err := st.UpsertArm(legacy); err != nil {
		t.Fatal(err)
	}
	var got ArmedOrderDB
	if err := db.First(&got, legacy.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.Source != "" || got.SourceRef != "" || got.SourceRule != "" || got.EligibleUntilMs != nil || got.SourceRunEpoch != nil {
		t.Fatalf("a planner row reads the source absent, never 0: %+v", got)
	}
}

// ONE OPPORTUNITY, ONE LIFE: once any row for the opportunity is terminal, the
// re-appended scenario on a later version never arms again.
func TestArmedSourcePinAcrossVersions(t *testing.T) {
	for _, end := range []struct{ state, signal string }{
		{StateFilled, "sig-1"},
		{StateCancelled, "sig-1"},
		{StateCancelled, ""},
		{"expired", ""},
	} {
		st := NewArmedOrderStore(newArmedTestDB(t))
		if err := st.UpsertArm(pictureRow(1, "P1", "opp-a", StateArmed, "")); err != nil {
			t.Fatal(err)
		}
		if err := st.db.Model(&ArmedOrderDB{}).Where("source_ref = ?", "opp-a").
			Updates(map[string]any{"state": end.state, "signal_id": end.signal}).Error; err != nil {
			t.Fatal(err)
		}
		if err := st.UpsertArm(pictureRow(2, "P1", "opp-a", StateArmed, "")); err != nil {
			t.Fatal(err)
		}
		rows := pictureRows(t, st, "opp-a")
		if len(rows) != 1 || rows[0].State != end.state || rows[0].Version != 1 {
			t.Fatalf("%s/%q: the opportunity must stay ended on v2, rows=%+v", end.state, end.signal, rows)
		}
	}
}

// An ARMED (never placed) Picture row follows its scenario to the new version
// in place — the re-append keeps it alive.
func TestArmedSourceRowFollowsTheReappendedScenario(t *testing.T) {
	st := NewArmedOrderStore(newArmedTestDB(t))
	if err := st.UpsertArm(pictureRow(1, "P1", "opp-a", StateArmed, "")); err != nil {
		t.Fatal(err)
	}
	if err := st.UpsertArm(pictureRow(2, "P1", "opp-a", StateArmed, "")); err != nil {
		t.Fatal(err)
	}
	rows := pictureRows(t, st, "opp-a")
	if len(rows) != 1 || rows[0].State != StateArmed || rows[0].Version != 2 || rows[0].Source != ArmSourcePicture || rows[0].EligibleUntilMs == nil {
		t.Fatalf("one armed row, now under v2, source intact: %+v", rows)
	}
	// A second identity for the same opportunity never gets its own row.
	if err := st.UpsertArm(pictureRow(2, "P2", "opp-a", StateArmed, "")); err != nil {
		t.Fatal(err)
	}
	if rows := pictureRows(t, st, "opp-a"); len(rows) != 1 || rows[0].Scenario != "P1" {
		t.Fatalf("one opportunity, one row: %+v", rows)
	}
}

func TestSupersedeLeavesMachineRowsAlone(t *testing.T) {
	st := NewArmedOrderStore(newArmedTestDB(t))
	if err := st.UpsertArm(pictureRow(1, "P1", "opp-a", StateArmed, "")); err != nil {
		t.Fatal(err)
	}
	planner := zoneRow(1)
	if err := st.UpsertArm(planner); err != nil {
		t.Fatal(err)
	}
	ids, err := st.SupersedeUnplacedArms("t1", "2026-09-23:NY:t1", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(ids) != 0 || pictureRows(t, st, "opp-a")[0].State != StateArmed {
		t.Fatalf("a machine row retires by its own deadline, never by a version move: retired %v", ids)
	}
	ids, err = st.SupersedeUnplacedArms("t1", "2026-09-11:NY:t1", 2)
	if err != nil || len(ids) != 1 || ids[0] != planner.ID {
		t.Fatalf("a planner row is still superseded: %v %v", ids, err)
	}
}
