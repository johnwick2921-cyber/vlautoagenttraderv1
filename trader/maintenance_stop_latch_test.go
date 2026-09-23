package trader

import (
	"context"
	"net"
	"testing"
	"time"

	"nofx/market"
	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	nttrader "nofx/trader/ninjatrader"
)

// ── W-ONE-BUTTON M2.1 (review 3 F7) — the stop path's hold refusal, BY
// BEHAVIOUR, in exactly the race it exists for: the pass read "not held" (no
// hold file), the broker permit then refuses the stop's send. The refusal must
// not latch "placed" and must not cancel the plan's other arms — a sibling
// stays armed, and so does the refused stop. (The old pin was a source-text
// check; `if at.placeOneStopEntry(...) && held` — M08 — survived it.)
func TestStopEntryHoldRefusalNeverCancelsTheSiblingArm(t *testing.T) {
	t.Setenv("STOP_ENTRY_SEAM", "on")
	withMaintenanceDir(t) // configured, NO hold file: the pass snapshot says not held
	at, st := resetTrader(t, store.StrategyConfig{})
	s := ntwire.NewTCPServer(nil)
	s.SetAddrForTest("127.0.0.1:0")
	s.SetAccountsList([]ntwire.AccountInfo{{Name: "Sim101", IsSim: true}}, "Sim101")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	conn, err := net.Dial("tcp", s.ListenAddrForTest().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	go func() {
		for {
			if _, err := ntwire.ReadFrame(conn); err != nil {
				return
			}
		}
	}()
	if err := ntwire.WriteFrame(conn, ntwire.FrameHeartbeat, ntwire.HeartbeatPayload{BuildID: ntwire.MinAddonBuildStopSlot}); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(time.Second); s.FarSideBuildID() == "" && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	broker := nttrader.NewTCPTrader(s, "MNQ", "Sim101")
	broker.StartCloseSync(at.id, "fixture", "ninjatrader", st)
	broker.SetEntryPermit(func() (func(), bool) { return nil, false }) // the hold, seen at the permit
	at.trader = broker
	at.config.NinjaTraderSymbol = "MNQ"
	now := time.Now() // ONE clock, controlled by the test (class 60/113: never a wall-clock entry)
	s.OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: []ntwire.NT8Order{}}, now)

	ledger := st.ArmedOrders()
	stop := store.ArmedOrderDB{TraderID: at.id, PlanID: "latch", Scenario: "S1", Version: 1, State: "armed",
		Side: "long", EntryPx: 29600, StopPx: 29590, TargetPx: 29630, Kind: "stop_entry", Condition: "reclaim"}
	sibling := store.ArmedOrderDB{TraderID: at.id, PlanID: "latch", Scenario: "S2", Version: 1, State: "armed",
		Side: "long", EntryPx: 29000, StopPx: 28990, TargetPx: 29030, Condition: "reject"} // limit, far from price
	for _, r := range []*store.ArmedOrderDB{&stop, &sibling} {
		if err := ledger.UpsertArm(r); err != nil {
			t.Fatal(err)
		}
	}
	at.runArmedPlacementAt([]market.Kline{{Close: 29599}}, now.Add(-time.Hour).UnixMilli(), now,
		armAdmission{armAdmitKey("latch", "S1", 0): true, armAdmitKey("latch", "S2", 0): true}) // the authoring pass admitted both (G1)

	for _, r := range []store.ArmedOrderDB{stop, sibling} {
		var got store.ArmedOrderDB
		if err := ledger.DB().First(&got, r.ID).Error; err != nil {
			t.Fatal(err)
		}
		if got.State != store.StateArmed {
			t.Fatalf("%s: a hold refusal of the stop must leave every arm of the plan armed, got %s (%s)", got.Scenario, got.State, got.StateReason)
		}
	}
}
