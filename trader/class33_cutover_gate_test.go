package trader

import (
	"strings"
	"testing"
	"time"

	"nofx/store"
)

// E8 — all five legs arrive in ONE payload, numbered and named.
func TestClass33GateReturnsAllFiveLegs(t *testing.T) {
	at := class33Trader(t)
	g := at.CutoverGateStatus()
	if len(g.Legs) != 5 {
		t.Fatalf("want 5 legs, got %d", len(g.Legs))
	}
	want := []string{"db_open_positions", "api_positions", "nt8_positions_snapshot", "working_orders", "planner_in_flight"}
	for i, l := range g.Legs {
		if l.N != i+1 || l.Name != want[i] {
			t.Fatalf("leg %d = %d/%s, want %d/%s", i, l.N, l.Name, i+1, want[i])
		}
	}
	if !strings.Contains(g.Note, store.BootSweptKey) {
		t.Fatalf("note must name the recorded counter: %s", g.Note)
	}
}

// E3 — leg 5: a claimed planner read FAILS the gate; released, it passes.
func TestClass33Leg5PlannerInFlight(t *testing.T) {
	at := class33Trader(t)
	key := store.MakePlanIDForTrader(at.id, "2026-09-02", "NY")
	if held, _ := at.AnyPlannerReadInFlight(); held {
		t.Fatalf("no read should be claimed yet")
	}
	if !claimPlannerRead(key) {
		t.Fatalf("claim failed")
	}
	defer releasePlannerRead(key)

	held, gotKey := at.AnyPlannerReadInFlight()
	if !held || gotKey != key {
		t.Fatalf("in-flight not seen: held=%v key=%q", held, gotKey)
	}
	g := at.CutoverGateStatus()
	leg5 := g.Legs[4]
	if leg5.Pass {
		t.Fatalf("leg 5 must FAIL while a planner read is in flight: %+v", leg5)
	}
	if !strings.Contains(leg5.Detail, key) {
		t.Fatalf("leg 5 must name the claim: %s", leg5.Detail)
	}
	if g.Ready {
		t.Fatalf("gate must not be ready with a read in flight")
	}

	releasePlannerRead(key)
	if g2 := at.CutoverGateStatus(); !g2.Legs[4].Pass {
		t.Fatalf("leg 5 must pass once the read is released: %+v", g2.Legs[4])
	}
}

// Leg 4 FAILS while the ledger holds a non-terminal arm — the exact 00:16 CT
// state that the stub reported as "no working orders".
func TestClass33Leg4FailsWithRestingArm(t *testing.T) {
	at := class33Trader(t)
	if g := at.CutoverGateStatus(); !g.Legs[3].Pass {
		t.Fatalf("leg 4 should pass on an empty ledger: %+v", g.Legs[3])
	}
	now := time.Now()
	row := &store.ArmedOrderDB{TraderID: at.id, PlanID: "p1", Scenario: "S3", Side: "long",
		EntryPx: 29068.05, StopPx: 29040, State: "working", SignalID: "sig-S3",
		CreatedAt: now, UpdatedAt: now}
	if err := at.store.ArmedOrders().UpsertArm(row); err != nil {
		t.Fatalf("seed: %v", err)
	}
	g := at.CutoverGateStatus()
	leg4 := g.Legs[3]
	if leg4.Pass {
		t.Fatalf("CLASS 33: leg 4 must FAIL with an arm resting at the broker: %+v", leg4)
	}
	if !strings.Contains(leg4.Detail, "1 non-terminal arm") {
		t.Fatalf("leg 4 detail: %s", leg4.Detail)
	}
	if !strings.Contains(leg4.Source, "ledger") {
		t.Fatalf("leg 4 must name the ledger as its source: %s", leg4.Source)
	}
	if g.Ready {
		t.Fatalf("gate must not be ready with a resting arm")
	}
}
