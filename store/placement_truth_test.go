package store

import "testing"

func TestPendingPlacementIdentityAndTerminalReceipts(t *testing.T) {
	ledger := NewArmedOrderStore(newArmedTestDB(t))
	row := &ArmedOrderDB{TraderID: "t1", PlanID: "p", Scenario: "S1", State: "armed", Version: 1}
	if err := ledger.UpsertArm(row); err != nil {
		t.Fatal(err)
	}
	if err := ledger.BeginPlacement(row.ID, "sig"); err != nil {
		t.Fatal(err)
	}
	if err := ledger.BeginPlacement(row.ID, "duplicate"); err == nil {
		t.Fatal("second send admitted")
	}
	if err := ledger.UpsertArm(&ArmedOrderDB{TraderID: "t1", PlanID: "p", Scenario: "S1", State: "armed", Version: 2}); err == nil {
		t.Fatal("pending row reauthorized")
	}
	rows, _ := ledger.ListNonTerminal("t1")
	if len(rows) != 1 || rows[0].State != StatePlacePending {
		t.Fatalf("pending lost slot: %+v", rows)
	}
	if err := ledger.ApplyPlacementReceipt("foreign", "sig", StateWorking, ""); err != nil {
		t.Fatal(err)
	}
	rows, _ = ledger.ListNonTerminal("t1")
	if rows[0].State != StatePlacePending {
		t.Fatal("foreign receipt changed row")
	}
	if err := ledger.ApplyPlacementReceipt("t1", "sig", StateRejected, "  precise NT8 reason  "); err != nil {
		t.Fatal(err)
	}
	if err := ledger.ApplyPlacementReceipt("t1", "sig", StateWorking, ""); err != nil {
		t.Fatal(err)
	}
	if err := ledger.BeginPlacement(row.ID, "late-send-return"); err == nil {
		t.Fatal("post-send path overwrote reject")
	}
	rows, _ = ledger.ListForPlan("p")
	if len(rows) != 1 || rows[0].State != StateRejected || rows[0].StateReason != "  precise NT8 reason  " {
		t.Fatalf("terminal receipt overwritten: %+v", rows)
	}
}

func TestAcceptancePreservesCancelIntent(t *testing.T) {
	ledger := NewArmedOrderStore(newArmedTestDB(t))
	row := &ArmedOrderDB{TraderID: "t1", PlanID: "p", Scenario: "S1", State: "armed"}
	if err := ledger.UpsertArm(row); err != nil {
		t.Fatal(err)
	}
	if err := ledger.BeginPlacement(row.ID, "sig"); err != nil {
		t.Fatal(err)
	}
	if err := ledger.RequestCancel(row.ID, "owner cancel", 1); err != nil {
		t.Fatal(err)
	}
	if err := ledger.ApplyPlacementReceipt("t1", "sig", StateWorking, ""); err != nil {
		t.Fatal(err)
	}
	rows, _ := ledger.ListForPlan("p")
	if rows[0].State != StateCancelPending {
		t.Fatalf("late acceptance erased cancel: %+v", rows)
	}
}

func TestSecondRejectionCanSupplyMissingReason(t *testing.T) {
	ledger := NewArmedOrderStore(newArmedTestDB(t))
	row := &ArmedOrderDB{TraderID: "t1", PlanID: "p", Scenario: "S1", State: StateArmed}
	if err := ledger.UpsertArm(row); err != nil {
		t.Fatal(err)
	}
	if err := ledger.BeginPlacement(row.ID, "sig"); err != nil {
		t.Fatal(err)
	}
	for _, reason := range []string{"", "  exact wire reason  ", "", "different later reason"} {
		if err := ledger.ApplyPlacementReceipt("t1", "sig", StateRejected, reason); err != nil {
			t.Fatal(err)
		}
	}
	rows, _ := ledger.ListForPlan("p")
	if rows[0].State != StateRejected || rows[0].StateReason != "  exact wire reason  " {
		t.Fatalf("missing reason did not enrich, or was overwritten: %+v", rows)
	}
}
