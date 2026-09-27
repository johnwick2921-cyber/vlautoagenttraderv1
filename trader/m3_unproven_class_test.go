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

// M3 PIN (CTO 2026-09-26, on top of the merged #243) — the armed stop-entry
// call site classifies a far-side-UNPROVEN refusal by its true cause:
//
//   - gate-block class stop_entry:far_side_unproven increments (NOT
//     stop_entry:addon_build, which is reserved for a REPORTED old build);
//   - the refusal logs once per change (the dedupe key is the arm spec + the
//     class — a second pass with the same state does not re-count);
//   - the ledger row STAYS armed (the refusal happens before the
//     BeginPlacement stamp, so nothing was sent and nothing is ambiguous).
//
// Production call site: at.runArmedPlacementAt → placeOneStopEntry →
// TCPTrader.PlaceStopEntry over a real loopback server whose far side never
// proved a build id. MUTANT: delete the `errors.Is(perr,
// ntwire.ErrFarSideNotProven)` branch in trader/armed_executor.go and this
// test FAILS — the refusal falls through to the generic path and the
// far_side_unproven counter never increments.
func TestM3UnprovenStopEntryRefusalCountsFarSideClassStaysArmedAndDedupes(t *testing.T) {
	t.Setenv("STOP_ENTRY_SEAM", "on")
	withMaintenanceDir(t)
	at, st := resetTrader(t, store.StrategyConfig{})

	s := ntwire.NewTCPServer(nil)
	s.SetAddrForTest("127.0.0.1:0")
	s.SetAccountsList([]ntwire.AccountInfo{{Name: "Sim101", IsSim: true}}, "Sim101")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Stop() }()
	conn, err := net.Dial("tcp", s.ListenAddrForTest().String())
	if err != nil {
		t.Fatal(err)
	}
	waitAddonRegistered(t, s) // CTO M7: the producer must not race the accept
	defer conn.Close()
	sent := make(chan ntwire.FrameType, 16)
	go func() {
		for {
			env, err := ntwire.ReadFrame(conn)
			if err != nil {
				return
			}
			sent <- env.Type
		}
	}()

	// The M3 premise: NO hello / heartbeat has ever carried a build id — the
	// far side is UNPROVEN on the current connection (the state FIX-P1A makes
	// indistinguishable from "disconnected since the proof").
	if got := s.FarSideBuildID(); got != "" {
		t.Fatalf("fixture: far side must be unproven, got build %q", got)
	}

	broker := nttrader.NewTCPTrader(s, "MNQ", "Sim101")
	// W117 F4 — the AddOn emits a positions frame on connect; seed the
	// known-flat book so the entry path reads empty, not unreadable.
	s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{})
	broker.StartCloseSync(at.id, "fixture", "ninjatrader", st)
	at.trader = broker
	at.config.NinjaTraderSymbol = "MNQ"
	// class 110: pinned to RTH — the admission chain reads THIS clock.
	now := rthInstant()
	s.OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: []ntwire.NT8Order{}}, now)

	ledger := st.ArmedOrders()
	stop := store.ArmedOrderDB{TraderID: at.id, PlanID: "m3", Scenario: "S1", Version: 1, State: "armed",
		Side: "long", EntryPx: 29600, StopPx: 29590, TargetPx: 29630, Kind: "stop_entry", Condition: "reclaim"}
	if err := ledger.UpsertArm(&stop); err != nil {
		t.Fatal(err)
	}

	beforeFar := gateBlocks(at.id, "stop_entry_far_side_unproven")
	beforeOld := gateBlocks(at.id, "stop_entry_addon_build")

	run := func() {
		at.runArmedPlacementAt([]market.Kline{{Close: 29599}}, now.Add(-time.Hour).UnixMilli(), now,
			armAdmission{armAdmitKey("m3", "S1", 0): true}) // the authoring pass admitted the arm (G1)
	}
	run()

	if got := gateBlocks(at.id, "stop_entry_far_side_unproven"); got != beforeFar+1 {
		t.Fatalf("M3: the unproven refusal must increment stop_entry:far_side_unproven (was %d, now %d)", beforeFar, got)
	}
	if got := gateBlocks(at.id, "stop_entry_addon_build"); got != beforeOld {
		t.Fatalf("M3: the unproven refusal must NOT increment stop_entry:addon_build (was %d, now %d)", beforeOld, got)
	}
	var row store.ArmedOrderDB
	if err := ledger.DB().First(&row, stop.ID).Error; err != nil {
		t.Fatal(err)
	}
	if row.State != "armed" {
		t.Fatalf("M3: the row must STAY armed (refused before the ledger stamp, provably unsent), got %q (%q)", row.State, row.StateReason)
	}
	// No SIGNAL frame may reach the wire (bars_subscribe etc. are
	// housekeeping frames the server sends on connect).
	deadline := time.After(200 * time.Millisecond)
drain:
	for {
		select {
		case f := <-sent:
			if f == ntwire.FrameSignal {
				t.Fatalf("M3: no signal frame may reach the wire while the far side is unproven, got %s", f)
			}
		case <-deadline:
			break drain
		}
	}

	// Once per change: a second pass in the same state must not re-count or
	// re-log the same refusal (armRefusalChanged dedupes per arm spec + class).
	run()
	if got := gateBlocks(at.id, "stop_entry_far_side_unproven"); got != beforeFar+1 {
		t.Fatalf("M3: the same refusal must not re-count each pass, now %d (want %d)", got, beforeFar+1)
	}
}
