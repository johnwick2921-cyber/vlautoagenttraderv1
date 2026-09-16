// PIN 3 — arm A places → arms B and C are cancelled. One plan, one live entry.
//
// This is the wiring half of the invariant, and it needs a real ledger: the
// adjudicator can be right and the plan can still end up with three live
// entries if nothing cancels the losers.
package trader

import (
	"path/filepath"
	"testing"
	"time"

	"nofx/store"
)

func seedArm(t *testing.T, ledger *store.ArmedOrderStore, scen string, leg int, entry float64, signal string) store.ArmedOrderDB {
	t.Helper()
	row := &store.ArmedOrderDB{
		TraderID: "hoang", PlanID: "2026-09-06:ASIA", Version: 3, Session: "ASIA",
		Scenario: scen, Side: "long", EntryPx: entry, StopPx: entry - 26, TargetPx: entry + 77,
		State: store.StateArmed, LegIndex: leg,
	}
	if err := ledger.UpsertArm(row); err != nil {
		t.Fatal(err)
	}
	if signal != "" {
		_ = ledger.SetState(row.ID, store.StateWorking, "")
		_ = ledger.SetSignal(row.ID, signal)
	}
	return *row
}

// The live shape: one plan, three arms (S1 leg0, S1 leg1, S2 leg0) — exactly
// arms 109/110/106 of 2026-09-06. S1 leg0 places; the other two must go.
func TestOneLiveEntryPerPlan_OthersCancelled(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "onelive.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ledger := st.ArmedOrders()

	placed := seedArm(t, ledger, "S1", 0, 29530.25, "c5d8bdde")
	s1leg1 := seedArm(t, ledger, "S1", 1, 29532.00, "")         // armed, never placed
	s2leg0 := seedArm(t, ledger, "S2", 0, 29541.25, "3278aa8c") // working at the broker

	rows, err := ledger.ListNonTerminal("hoang")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 3 {
		t.Fatalf("harness: want 3 non-terminal arms, got %d", len(rows))
	}

	at := &AutoTrader{id: "hoang", store: st}
	now := time.Date(2026, 9, 6, 20, 7, 12, 0, time.UTC)
	at.cancelOtherArmsInPlan(ledger, rows, placed, now)

	after, err := ledger.ListNonTerminal("hoang")
	if err != nil {
		t.Fatal(err)
	}
	byID := map[int64]store.ArmedOrderDB{}
	for _, r := range after {
		byID[r.ID] = r
	}

	// The placed row is untouched.
	if got, ok := byID[placed.ID]; !ok || got.State != store.StateWorking {
		t.Fatalf("the placed arm must keep its state; got %+v (present=%v)", got, ok)
	}

	// A never-placed arm has nothing at the broker → terminal directly.
	if got, ok := byID[s1leg1.ID]; ok {
		t.Fatalf("S1 leg1 was never placed and must be terminal, not still live: %+v", got)
	}

	// A working arm goes to cancel_pending — a send is not a settlement
	// (class 81); only a fresh book may finish it.
	got, ok := byID[s2leg0.ID]
	if !ok {
		t.Fatalf("S2 leg0 was working at the broker and must NOT go terminal on a send")
	}
	if got.State != store.StateCancelPending {
		t.Fatalf("S2 leg0 must be cancel_pending, got %q", got.State)
	}
	if got.StateReason == "" {
		t.Fatalf("the cancel must carry its reason (owner ruling: cancelled WITH the reason)")
	}
}

// A different plan's arms are NOT touched. The ruling is one live entry per
// PLAN; a stale row from yesterday's plan is somebody else's problem.
func TestOneLiveEntryPerPlan_OtherPlansUntouched(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "otherplan.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ledger := st.ArmedOrders()

	placed := seedArm(t, ledger, "S1", 0, 29530.25, "c5d8bdde")
	other := &store.ArmedOrderDB{
		TraderID: "hoang", PlanID: "2026-09-05:NY", Version: 1, Session: "NY",
		Scenario: "S1", Side: "long", EntryPx: 29400, StopPx: 29380, TargetPx: 29470,
		State: store.StateArmed,
	}
	if err := ledger.UpsertArm(other); err != nil {
		t.Fatal(err)
	}

	rows, _ := ledger.ListNonTerminal("hoang")
	at := &AutoTrader{id: "hoang", store: st}
	at.cancelOtherArmsInPlan(ledger, rows, placed, time.Date(2026, 9, 6, 20, 7, 12, 0, time.UTC))

	after, _ := ledger.ListNonTerminal("hoang")
	found := false
	for _, r := range after {
		if r.ID == other.ID {
			found = true
			if r.State != store.StateArmed {
				t.Fatalf("another plan's arm must be untouched; got %q", r.State)
			}
		}
	}
	if !found {
		t.Fatalf("another plan's arm was cancelled — the ruling is per PLAN")
	}
}
