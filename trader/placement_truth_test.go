package trader

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"nofx/market"
	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	nttrader "nofx/trader/ninjatrader"
)

func TestFourPlacementPathsWaitForEntryReceipt(t *testing.T) {
	for _, path := range []string{"limit", "stop_entry", "debug_limit", "debug_stop"} {
		t.Run(path, func(t *testing.T) {
			t.Setenv("ARMED_TEST_SEAM", "on")
			t.Setenv("STOP_ENTRY_SEAM", "on")
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
			frames := make(chan ntwire.SignalPayload, 4)
			go func() {
				for {
					env, err := ntwire.ReadFrame(conn)
					if err != nil {
						return
					}
					if env.Type == ntwire.FrameSignal {
						var p ntwire.SignalPayload
						if json.Unmarshal(env.Payload, &p) == nil {
							frames <- p
						}
					}
				}
			}()
			if err := ntwire.WriteFrame(conn, ntwire.FrameHeartbeat, ntwire.HeartbeatPayload{BuildID: ntwire.MinAddonBuildStopSlot}); err != nil {
				t.Fatal(err)
			}
			deadline := time.Now().Add(time.Second)
			for s.FarSideBuildID() == "" && time.Now().Before(deadline) {
				time.Sleep(time.Millisecond)
			}
			broker := nttrader.NewTCPTrader(s, "MNQ", "Sim101")
			broker.StartCloseSync(at.id, "fixture", "ninjatrader", st)
			at.trader = broker
			at.config.NinjaTraderSymbol = "MNQ"
			s.OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: []ntwire.NT8Order{}}, time.Now())
			ledger := st.ArmedOrders()
			var row store.ArmedOrderDB
			switch path {
			case "debug_limit":
				row, err = at.TestArmPlace("long", 29600, 29590, 29630)
			case "debug_stop":
				row, err = at.TestArmPlaceStop("long", 29600, 29590, 29630)
			default:
				row = store.ArmedOrderDB{TraderID: at.id, PlanID: "placement", Scenario: "S1", Version: 1, State: "armed", Side: "long", EntryPx: 29600, StopPx: 29590, TargetPx: 29630, Kind: path, Condition: "reclaim"}
				if err = ledger.UpsertArm(&row); err != nil {
					t.Fatal(err)
				}
				price := 29601.0
				if path == "stop_entry" {
					price = 29599
				}
				at.runArmedPlacement([]market.Kline{{Close: price}}, time.Now().Add(-time.Hour).UnixMilli())
			}
			if err != nil {
				t.Fatal(err)
			}
			var p ntwire.SignalPayload
			select {
			case p = <-frames:
			case <-time.After(time.Second):
				t.Fatal("production path did not send")
			}
			if err := ledger.DB().First(&row, row.ID).Error; err != nil {
				t.Fatal(err)
			}
			if row.State != store.StatePlacePending || row.SignalID != p.SignalID {
				t.Fatalf("send falsely settled: %+v", row)
			}
			// A protective leg carries the same signal; it must not promote the entry.
			u := ntwire.OrderUpdatePayload{SignalID: p.SignalID, OrderName: p.SignalID + "-sl", State: "working", Account: p.Account, Symbol: p.Symbol}
			at.onArmedOrderUpdate(u, ledger)
			ledger.DB().First(&row, row.ID)
			if row.State != store.StatePlacePending {
				t.Fatal("protective leg promoted entry")
			}
			u.OrderName = p.SignalID
			at.onArmedOrderUpdate(u, ledger)
			ledger.DB().First(&row, row.ID)
			if row.State != store.StateWorking {
				t.Fatalf("entry receipt did not promote: %+v", row)
			}
			u.State = "rejected"
			u.Reason = "  price check refused this entry  "
			at.onArmedOrderUpdate(u, ledger)
			u.State = "working"
			at.onArmedOrderUpdate(u, ledger)
			ledger.DB().First(&row, row.ID)
			if row.State != store.StateRejected || row.StateReason != "  price check refused this entry  " {
				t.Fatalf("reject lost or overwritten: %+v", row)
			}
		})
	}
}

// Deterministic fast-reply interleaving: registration, received rejection,
// then the placement call returns. No post-send write may erase the receipt.
func TestStopPlacementFastRejectBeforeSendReturns(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{})
	ledger := st.ArmedOrders()
	row := store.ArmedOrderDB{TraderID: at.id, PlanID: "fast", Scenario: "S1", State: "armed", Side: "long", EntryPx: 29600, StopPx: 29590, TargetPx: 29630}
	if err := ledger.UpsertArm(&row); err != nil {
		t.Fatal(err)
	}
	pl := &fakePlacer{sid: "fast-reject", afterRegister: func() {
		at.onArmedOrderUpdate(ntwire.OrderUpdatePayload{SignalID: "fast-reject", OrderName: "fast-reject", State: "rejected", Reason: "stale signal age=715.3s (max 60s)"}, ledger)
	}}
	d := decideStopEntry("long", row.EntryPx, testOffset(), testTick, 29599)
	at.placeOneStopEntry(pl, ledger, row, d, 29599, time.Now(), freeSlot())
	rows, err := ledger.ListForPlan("fast")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].State != "rejected" || rows[0].StateReason != "stale signal age=715.3s (max 60s)" {
		t.Fatalf("fast receipt overwritten: %+v", rows)
	}
}

func TestPlacementBookRequiresLiveEntryAndSilenceKeepsSlot(t *testing.T) {
	at, st, _, _ := shadowWireHarness(t, store.StrategyConfig{})
	ledger := st.ArmedOrders()
	row := &store.ArmedOrderDB{TraderID: at.id, PlanID: "book-proof", Scenario: "S1", State: store.StateArmed}
	if err := ledger.UpsertArm(row); err != nil {
		t.Fatal(err)
	}
	if err := ledger.BeginPlacement(row.ID, "book-entry"); err != nil {
		t.Fatal(err)
	}
	broker := at.armedTrader()
	now := time.Now()
	if err := ledger.DB().Model(row).UpdateColumn("updated_at", now.Add(-time.Hour)).Error; err != nil {
		t.Fatal(err)
	}
	for _, orders := range [][]ntwire.NT8Order{
		{},
		{{Name: "book-entry-sl", State: "working", Symbol: "MNQ"}},
		{{Name: "book-entry", State: "submitted", Symbol: "MNQ"}},
		{{Name: "book-entry", State: "triggerpending", Symbol: "MNQ"}},
	} {
		broker.GetServer().OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: orders}, now)
		confirmed, pending, _ := at.confirmPendingPlacements(ledger, now)
		if confirmed != 0 || pending != 1 {
			t.Fatalf("non-entry evidence settled placement: %d/%d", confirmed, pending)
		}
		rows, err := ledger.ListNonTerminal(at.id)
		if err != nil || len(rows) != 1 || rows[0].State != store.StatePlacePending {
			t.Fatalf("silence released slot: %+v %v", rows, err)
		}
	}
	broker.GetServer().OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: []ntwire.NT8Order{{Name: "book-entry", State: "working", Symbol: "MNQ"}}}, now.Add(-2*placeConfirmMaxWait()))
	confirmed, pending, _ := at.confirmPendingPlacements(ledger, now)
	if confirmed != 0 || pending != 1 {
		t.Fatal("stale book promoted")
	}
	broker.GetServer().OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: []ntwire.NT8Order{{Name: "book-entry", State: "working", Symbol: "MNQ"}}}, now)
	confirmed, pending, _ = at.confirmPendingPlacements(ledger, now)
	if confirmed != 1 || pending != 0 {
		t.Fatalf("live entry not confirmed: %d/%d", confirmed, pending)
	}
}
