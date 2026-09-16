package ninjatrader

import (
	"path/filepath"
	"testing"

	"nofx/store"
)

// REGRADE RESET (owner ruling 2026-09-03) — the reset keys on "lineage was just
// stamped on this row", NEVER on a grade letter.
//
// The predecessor read `p.AdherenceGrade == "F"`. That is not an impossible
// value — an uncited close grades base D and steps to F under either penalty
// (InNoTrade, !InKillzone) — it is a SUBSET. So the repair silently succeeded
// on penalised uncited rows (566, 571 → F) and silently failed on clean ones
// (580 → D), which is why it survived: a spot-check lands on a working case.
//
// Keying on the stamp is also the only correct rule. If lineage was just
// written onto a row, whatever grade it carries was computed WITHOUT that
// lineage and is stale by construction, whatever letter it happens to be.

func regradeStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "r.db"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

// A late-stamped row with plan_matched=1 must have its grade CLEARED so the W5
// analytics regrade it with the lineage in hand. This is the 584/586/591 shape.
func TestRegradeResetClearsALateStampedRowWhateverTheLetter(t *testing.T) {
	for _, grade := range []string{"D", "F", "C"} {
		st := regradeStore(t)
		pos := &store.TraderPosition{
			TraderID: "t", Symbol: "MNQ", Side: "SHORT", Quantity: 1,
			EntryPrice: 29285, EntryTime: 1, Status: "OPEN",
			CreatedAt: 1, UpdatedAt: 1,
		}
		if err := st.Position().Create(pos); err != nil {
			t.Fatalf("create: %v", err)
		}
		if _, err := st.Position().ClosePosition(pos.ID, 29115, "", -140, 0, "stop"); err != nil {
			t.Fatalf("close: %v", err)
		}
		if err := st.Position().SetAdherence(pos.ID, grade); err != nil {
			t.Fatalf("grade: %v", err)
		}
		row := store.ArmedOrderDB{
			TraderID: "t", PlanID: "2026-09-03:NY:t", Version: 2, Session: "NY",
			Scenario: "S1", Side: "short", EntryPx: 29285, StopPx: 29362.5,
			TargetPx: 29130, State: "armed", FillPrice: 29285,
		}
		if err := st.ArmedOrders().UpsertArm(&row); err != nil {
			t.Fatalf("arm: %v", err)
		}
		if err := st.ArmedOrders().SetState(row.ID, "filled", "armed_fill"); err != nil {
			t.Fatalf("fill: %v", err)
		}

		if n := RepairArmedLineage(st, "t"); n != 1 {
			t.Fatalf("grade %s: repair stamped %d rows, want 1", grade, n)
		}
		got, err := st.Position().GetClosedPositions("t", 5)
		if err != nil || len(got) != 1 {
			t.Fatalf("read back: %v n=%d", err, len(got))
		}
		if got[0].AdherenceGrade != "" {
			t.Errorf("grade %s: a late-stamped row must be CLEARED for regrading, got %q — the old predicate only cleared F",
				grade, got[0].AdherenceGrade)
		}
		if got[0].PlanVersion != 2 {
			t.Errorf("grade %s: lineage must be stamped, got version %d", grade, got[0].PlanVersion)
		}
	}
}

// An UNCITED close that nothing stamps keeps its D. Row 580 is exactly this:
// plan_version 0, no citation, grade D — and it EARNED that D. Clearing it
// would promote a genuinely off-plan trade.
func TestRegradeResetLeavesAnUnstampedRowAlone(t *testing.T) {
	st := regradeStore(t)
	pos := &store.TraderPosition{
		TraderID: "t", Symbol: "MNQ", Side: "SHORT", Quantity: 1,
		EntryPrice: 29285, EntryTime: 1, Status: "OPEN", CreatedAt: 1, UpdatedAt: 1,
	}
	if err := st.Position().Create(pos); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := st.Position().ClosePosition(pos.ID, 29115, "", -140, 0, "stop"); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := st.Position().SetAdherence(pos.ID, "D"); err != nil {
		t.Fatalf("grade: %v", err)
	}
	// no armed row at all → nothing to stamp
	if n := RepairArmedLineage(st, "t"); n != 0 {
		t.Fatalf("nothing should stamp, got %d", n)
	}
	got, _ := st.Position().GetClosedPositions("t", 5)
	if len(got) != 1 || got[0].AdherenceGrade != "D" {
		t.Fatalf("an uncited close keeps its earned D, got %q", got[0].AdherenceGrade)
	}
}
