package trader

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"vl/kernel/mentor"
	"vl/market"
	ntwire "vl/provider/ninjatrader"
	"vl/store"
	nttrader "vl/trader/ninjatrader"
)

// ── MENTOR INJECTOR P0 pins (CTO 1791040360324) ───────────────────────────────
//
// Every evaluator action reaches a real executor, at the call site. The
// counters (MentorCountSnapshot) are the refusal ledger: each test asserts
// the exact counter its branch must bump, so a branch that drops an action
// silently turns RED instead of passing.

func resetMentorCounters() {
	mentorCounterMu.Lock()
	mentorCounters = map[string]int{}
	mentorCounterMu.Unlock()
}

// mentorLoopback is a started loopback NT8 server whose far side proves the
// given build, mirroring every frame type into frames.
func mentorLoopback(t *testing.T, buildID string) (at *AutoTrader, st *store.Store, ledger *store.ArmedOrderStore, frames chan ntwire.FrameType) {
	t.Helper()
	at, st = resetTrader(t, store.StrategyConfig{RiskControl: store.RiskControlConfig{
		MentorMode:          true,
		MentorWindowStart:   "00:00", // all-day window so the window gate passes at any test hour
		MentorWindowMinutes: 1440,
	}})
	withMaintenanceDir(t)
	s := ntwire.NewTCPServer(nil)
	s.SetAddrForTest("127.0.0.1:0")
	s.SetAccountsList([]ntwire.AccountInfo{{Name: "Sim101", IsSim: true}}, "Sim101")
	ctx, cancel := context.WithCancel(context.Background())
	if err := s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Stop(); cancel() })
	conn, err := net.Dial("tcp", s.ListenAddrForTest().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	waitAddonRegistered(t, s)
	frames = make(chan ntwire.FrameType, 32)
	go func() {
		for {
			env, err := ntwire.ReadFrame(conn)
			if err != nil {
				return
			}
			frames <- env.Type
		}
	}()
	if buildID != "" {
		if err := ntwire.WriteFrame(conn, ntwire.FrameHeartbeat, ntwire.HeartbeatPayload{BuildID: buildID}); err != nil {
			t.Fatal(err)
		}
		for deadline := time.Now().Add(2 * time.Second); !ntwire.FarSideProven(s.FarSideBuildID(), buildID) && time.Now().Before(deadline); {
			time.Sleep(time.Millisecond)
		}
		if !ntwire.FarSideProven(s.FarSideBuildID(), buildID) {
			t.Fatal("fixture: far-side build never proven")
		}
	}
	broker := nttrader.NewTCPTrader(s, "MNQ", "Sim101")
	s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{})
	broker.StartCloseSync(at.id, "fixture", "ninjatrader", st)
	at.trader = broker
	at.config.NinjaTraderSymbol = "MNQ"
	// A fresh empty book: the slot + one-contract guards read it and an
	// unreadable book is a refusal (A24).
	s.OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: []ntwire.NT8Order{}}, time.Now())
	ledger = st.ArmedOrders()
	return at, st, ledger, frames
}

// mentorWireSeams injects every placement seam the mentorSourcesMissing gate
// reads, so the entry path can reach the wire.
func mentorWireSeams(t *testing.T, at *AutoTrader, ledger *store.ArmedOrderStore) {
	t.Helper()
	mentorOpenStopSource = func() (float64, bool) { return 0, false }
	mentorOpenSideSource = func() string { return "" } // no open position: the never-add gate passes
	mentorLegProtectedSource = func(leg string) bool { return false }
	mentorLatestPriceSource = func() (float64, bool) { return 29590, true } // below the long trigger: the no-chase rule passes
	mentorConfluenceForIntent = func(in mentor.Intent) bool { return in.Confluence }
	mentorDayNetSource = func() float64 { return 0 }
	mentorClosedProfitSource = func() bool { return false }
	mentorSetArmExpiryWire = func(armID int64, expiryMs int64) error {
		return ledger.SetArmExpiry(armID, expiryMs)
	}
	mentorSourceDepthSource = func(name string) (int, bool) { return 1000, true }
	market.FuturesBarsProvider = func(symbol, tf string, n int) []market.Kline {
		return []market.Kline{{Open: 1, High: 2, Low: 0.5, Close: 1.5, OpenTime: time.Now().UnixMilli()}}
	}
	t.Cleanup(func() {
		mentorOpenStopSource = nil
		mentorOpenSideSource = nil
		mentorLegProtectedSource = nil
		mentorLatestPriceSource = nil
		mentorConfluenceForIntent = nil
		mentorDayNetSource = nil
		mentorClosedProfitSource = nil
		mentorSetArmExpiryWire = nil
		mentorSourceDepthSource = nil
		market.FuturesBarsProvider = nil
		mentorLiveMu.Lock()
		mentorLiveArms = map[string]mentorLiveArm{}
		mentorLiveMu.Unlock()
	})
}

// sawMentorFrame drains frames until deadline and reports whether want arrived.
func sawMentorFrame(t *testing.T, frames chan ntwire.FrameType, want ntwire.FrameType, within time.Duration) bool {
	t.Helper()
	deadline := time.After(within)
	for {
		select {
		case f := <-frames:
			if f == want {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

// TestMentorDispatchHandlesEveryAction — the switch pin: every evaluator
// action has a named executor; an unknown action REFUSES with a counter,
// never silent. Each case asserts its exact counter.
func TestMentorDispatchHandlesEveryAction(t *testing.T) {
	at := mentoredTrader(t, store.RiskControlConfig{MentorMode: true})
	t.Setenv("MENTOR_PLACE", "")
	resetMentorCounters()
	cases := []struct {
		name string
		in   mentor.Intent
		want string
	}{
		{"place_stop_limit_entry_held", mentor.Intent{Action: mentor.PlaceStopLimitEntry, Setup: "ISB", Side: mentor.SideLong,
			Price: 29600, Stop: 29595, Target: 29610, StopPts: 5, TargetPts: 10, ExpiryMs: 100_000}, "placement_held"},
		{"extend_unknown_arm", mentor.Intent{Action: mentor.ExtendArm, ArmID: "nope", ExpiryMs: 100_000}, "extend_refused_unknown_arm"},
		{"cancel_unknown_arm", mentor.Intent{Action: mentor.CancelArm, ArmID: "nope"}, "cancel_refused_unknown_arm"},
		{"level_invalid_recorded", mentor.Intent{Action: mentor.LevelInvalid, LevelKey: "k:1@x"}, "intent_level_invalid"},
		{"move_be_unknown_arm", mentor.Intent{Action: mentor.ActionMoveStopBE, ArmID: "nope"}, "move_be_refused_unknown_arm"},
		{"close_unknown_arm", mentor.Intent{Action: mentor.ActionClosePosition, ArmID: "nope"}, "close_refused_unknown_arm"},
		{"unknown_action_refused", mentor.Intent{Action: "frobnicate", Reason: "what is this"}, "unknown_action_frobnicate"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetMentorCounters()
			at.mentorDispatchIntent(tc.in, mentorTierInputs{}, 1000, 1100)
			if got := MentorCountSnapshot()[tc.want]; got != 1 {
				t.Fatalf("counter %q = %d, want 1 (the action must reach its named executor)", tc.want, got)
			}
		})
	}
}

// TestMentorEntryIsAlwaysStopLimit — the ONE mentor entry path at the call
// site: a PlaceStopLimitEntry intent (the ISB order type) creates a ledger
// arm with origin-ready fields and the intent's expiry, and the wire frame is
// ALWAYS stop-limit (stop_limit:true, order_type stop_entry) with
// MENTOR_STOP_LIMIT OFF — the PLAN's never stop-market, knob-independent.
func TestMentorEntryIsAlwaysStopLimit(t *testing.T) {
	t.Setenv("MENTOR_PLACE", "on")
	t.Setenv("MENTOR_STOP_LIMIT", "") // the knob is OFF and must NOT matter
	at, _, ledger, frames := mentorLoopback(t, ntwire.MinAddonBuildStopLimit)
	mentorWireSeams(t, at, ledger)
	resetMentorCounters()

	in := mentor.Intent{Action: mentor.PlaceStopLimitEntry, ArmID: "isb-1", Setup: "ISB", Side: mentor.SideLong,
		Price: 29600, Stop: 29595, Target: 29610, StopPts: 5, TargetPts: 10, ExpiryMs: time.Now().UnixMilli() + 60_000}
	at.mentorDispatchIntent(in, mentorTierInputs{}, 1000, 1100)

	if !sawMentorFrame(t, frames, ntwire.FrameSignal, 2*time.Second) {
		t.Fatal("the mentor entry must reach the wire")
	}
	// The armed row: created with the intent's expiry and the registry entry.
	var row store.ArmedOrderDB
	if err := ledger.DB().Where("scenario = ?", "isb-1").First(&row).Error; err != nil {
		t.Fatalf("the mentor arm row must exist: %v", err)
	}
	if row.Kind != "stop_entry" || row.ExpiryMs != in.ExpiryMs {
		t.Fatalf("mentor arm kind=%q expiry=%d, want stop_entry / %d", row.Kind, row.ExpiryMs, in.ExpiryMs)
	}
	if row.State != store.StatePlacePending || row.SignalID == "" {
		t.Fatalf("mentor arm must be place_pending with a signal id after the send, got %q/%q", row.State, row.SignalID)
	}
	arm, ok := mentorLiveArmFor("isb-1")
	if !ok || arm.RowID != row.ID {
		t.Fatalf("the ArmID registry must resolve isb-1 -> row %d, got %+v ok=%v", row.ID, arm, ok)
	}
	placed := 0
	for k, v := range MentorCountSnapshot() {
		if strings.HasPrefix(k, "placed_") {
			placed += v
		}
	}
	if placed != 1 {
		t.Fatalf("exactly one placement tier must be counted, got %d", placed)
	}
}

// mentorLoopbackRaw is mentorLoopback plus the RAW signal payload bytes (the
// wire-flag pin needs the bytes: a decoded bool cannot tell absent from false).
func mentorLoopbackRaw(t *testing.T, buildID string) (at *AutoTrader, ledger *store.ArmedOrderStore, raw chan []byte) {
	t.Helper()
	at, st := resetTrader(t, store.StrategyConfig{RiskControl: store.RiskControlConfig{
		MentorMode:          true,
		MentorWindowStart:   "00:00",
		MentorWindowMinutes: 1440,
	}})
	withMaintenanceDir(t)
	s := ntwire.NewTCPServer(nil)
	s.SetAddrForTest("127.0.0.1:0")
	s.SetAccountsList([]ntwire.AccountInfo{{Name: "Sim101", IsSim: true}}, "Sim101")
	ctx, cancel := context.WithCancel(context.Background())
	if err := s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Stop(); cancel() })
	conn, err := net.Dial("tcp", s.ListenAddrForTest().String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	waitAddonRegistered(t, s)
	raw = make(chan []byte, 8)
	go func() {
		for {
			env, err := ntwire.ReadFrame(conn)
			if err != nil {
				return
			}
			if env.Type == ntwire.FrameSignal {
				raw <- append([]byte(nil), env.Payload...)
			}
		}
	}()
	if err := ntwire.WriteFrame(conn, ntwire.FrameHeartbeat, ntwire.HeartbeatPayload{BuildID: buildID}); err != nil {
		t.Fatal(err)
	}
	for deadline := time.Now().Add(2 * time.Second); !ntwire.FarSideProven(s.FarSideBuildID(), buildID) && time.Now().Before(deadline); {
		time.Sleep(time.Millisecond)
	}
	if !ntwire.FarSideProven(s.FarSideBuildID(), buildID) {
		t.Fatal("fixture: far-side build never proven")
	}
	broker := nttrader.NewTCPTrader(s, "MNQ", "Sim101")
	s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{})
	broker.StartCloseSync(at.id, "fixture", "ninjatrader", st)
	at.trader = broker
	at.config.NinjaTraderSymbol = "MNQ"
	s.OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: []ntwire.NT8Order{}}, time.Now())
	return at, st.ArmedOrders(), raw
}

// TestMentorEntryFrameCarriesStopLimitTrue — the raw frame pin: the mentor
// entry always carries stop_limit:true (the knob is off in this test).
func TestMentorEntryFrameCarriesStopLimitTrue(t *testing.T) {
	t.Setenv("MENTOR_PLACE", "on")
	t.Setenv("MENTOR_STOP_LIMIT", "")
	at, ledger, raw := mentorLoopbackRaw(t, ntwire.MinAddonBuildStopLimit)
	mentorWireSeams(t, at, ledger)
	at.mentorDispatchIntent(mentor.Intent{Action: mentor.PlaceStopLimitEntry, ArmID: "isb-raw", Setup: "ISB", Side: mentor.SideLong,
		Price: 29600, Stop: 29595, Target: 29610, StopPts: 5, TargetPts: 10, ExpiryMs: time.Now().UnixMilli() + 60_000}, mentorTierInputs{}, 1000, 1100)
	deadline := time.After(2 * time.Second)
	var payload []byte
	select {
	case payload = <-raw:
	case <-deadline:
		t.Fatal("no signal frame reached the wire")
	}
	if !bytes.Contains(payload, []byte(`"stop_limit":true`)) {
		t.Fatalf("the mentor frame must carry stop_limit:true even with MENTOR_STOP_LIMIT off, got %s", payload)
	}
	var p ntwire.SignalPayload
	if err := json.Unmarshal(payload, &p); err != nil {
		t.Fatal(err)
	}
	if p.OrderType != "stop_entry" || !p.StopLimit {
		t.Fatalf("order_type=%q stop_limit=%v, want stop_entry/true", p.OrderType, p.StopLimit)
	}
}

// TestMentorEntryRefusesBelowC2 — an AddOn below the c2 floor REFUSES the
// mentor entry (retired terminal, named counter) and never falls back to a
// stop-market frame.
func TestMentorEntryRefusesBelowC2(t *testing.T) {
	t.Setenv("MENTOR_PLACE", "on")
	at, _, ledger, frames := mentorLoopback(t, ntwire.MinAddonBuildCancelReport) // c1 < c2
	mentorWireSeams(t, at, ledger)
	resetMentorCounters()
	in := mentor.Intent{Action: mentor.PlaceStopLimitEntry, ArmID: "isb-c1", Setup: "ISB", Side: mentor.SideLong,
		Price: 29600, Stop: 29595, Target: 29610, StopPts: 5, TargetPts: 10, ExpiryMs: time.Now().UnixMilli() + 60_000}
	at.mentorDispatchIntent(in, mentorTierInputs{}, 1000, 1100)
	if sawMentorFrame(t, frames, ntwire.FrameSignal, 500*time.Millisecond) {
		t.Fatal("an AddOn below c2 must never receive a mentor entry frame")
	}
	if got := MentorCountSnapshot()["placement_refused_addon_below_c2"]; got != 1 {
		t.Fatalf("placement_refused_addon_below_c2 = %d, want 1", got)
	}
	var row store.ArmedOrderDB
	if err := ledger.DB().Where("scenario = ?", "isb-c1").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if !store.IsTerminalArmState(row.State) {
		t.Fatalf("the refused mentor arm must be retired terminal, got %q", row.State)
	}
}

// TestMentorExtendArmStampsTheRestingRow — ExtendArm reaches SetArmExpiry for
// the registered row (working + unfilled = the resting stop-limit ISB
// stacking extends). Mutant: dropping the case leaves the counter unset.
func TestMentorExtendArmStampsTheRestingRow(t *testing.T) {
	at, _, ledger, _ := mentorLoopback(t, ntwire.MinAddonBuildStopLimit)
	mentorWireSeams(t, at, ledger)
	row := store.ArmedOrderDB{TraderID: at.id, PlanID: "mentor", Version: 1, Session: "MENTOR", Scenario: "isb-ext",
		Side: "long", EntryPx: 29600, StopPx: 29595, TargetPx: 29610, Kind: "stop_entry", Condition: "ISB", ExpiryMs: time.Now().UnixMilli() + 60_000}
	if err := ledger.UpsertArm(&row); err != nil {
		t.Fatal(err)
	}
	if err := ledger.SetState(row.ID, store.StateWorking, "test receipt"); err != nil {
		t.Fatal(err)
	}
	mentorRegisterLiveArm("isb-ext", row.ID, "long", 29600)
	resetMentorCounters()
	newExpiry := time.Now().UnixMilli() + 120_000
	at.mentorDispatchIntent(mentor.Intent{Action: mentor.ExtendArm, ArmID: "isb-ext", ExpiryMs: newExpiry}, mentorTierInputs{}, 1000, 1100)
	var got store.ArmedOrderDB
	if err := ledger.DB().First(&got, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.ExpiryMs != newExpiry {
		t.Fatalf("the resting arm's expiry was not extended: got %d, want %d", got.ExpiryMs, newExpiry)
	}
	if c := MentorCountSnapshot()["extend_ok"]; c != 1 {
		t.Fatalf("extend_ok = %d, want 1", c)
	}
}

// TestMentorCancelArmReachesTheBroker — CancelArm sends the cancel_order on
// the same pass (the F1 shape) and requests the ledger cancel.
func TestMentorCancelArmReachesTheBroker(t *testing.T) {
	at, _, ledger, frames := mentorLoopback(t, ntwire.MinAddonBuildStopLimit)
	now := time.Now()
	row := store.ArmedOrderDB{TraderID: at.id, PlanID: "mentor", Version: 1, Session: "MENTOR", Scenario: "isb-can",
		Side: "long", EntryPx: 29600, StopPx: 29595, TargetPx: 29610, Kind: "stop_entry", Condition: "ISB",
		State: store.StateWorking, SignalID: "isb-can-sig", ExpiryMs: now.UnixMilli() + 60_000}
	if err := ledger.DB().Create(&row).Error; err != nil {
		t.Fatal(err)
	}
	s := at.armedTrader().GetServer()
	s.OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: []ntwire.NT8Order{
		{Name: "isb-can-sig", State: "Working", Type: "stop_limit"}}}, now)
	mentorRegisterLiveArm("isb-can", row.ID, "long", 29600)
	resetMentorCounters()
	at.mentorDispatchIntent(mentor.Intent{Action: mentor.CancelArm, ArmID: "isb-can", Reason: "ISB escaped the mother candle"}, mentorTierInputs{}, 1000, 1100)
	if !sawMentorFrame(t, frames, ntwire.FrameCancelOrder, 1*time.Second) {
		t.Fatal("CancelArm must send cancel_order on the same pass")
	}
	var got store.ArmedOrderDB
	if err := ledger.DB().First(&got, row.ID).Error; err != nil {
		t.Fatal(err)
	}
	if got.State != store.StateCancelPending {
		t.Fatalf("the cancelled row must be cancel_pending, got %q", got.State)
	}
	if c := MentorCountSnapshot()["cancel_requested"]; c != 1 {
		t.Fatalf("cancel_requested = %d, want 1", c)
	}
}

// TestMentorMoveStopBEReachesMoveStopWire — the swing's BE intent moves the
// stop to the registered ENTRY price through mentorMoveStop (never-widen
// guarded). Mutant: dropping the dispatch case leaves the spy uncalled.
func TestMentorMoveStopBEReachesMoveStopWire(t *testing.T) {
	at, _, _, _ := mentorLoopback(t, ntwire.MinAddonBuildStopLimit)
	mentorOpenStopSource = func() (float64, bool) { return 29595, true } // current stop below entry: BE is not a widen
	t.Cleanup(func() { mentorOpenStopSource = nil })
	var mu sync.Mutex
	var moved []float64
	oldWire := moveStopWire
	moveStopWire = func(nt *nttrader.TCPTrader, side string, newStop float64) error {
		mu.Lock()
		moved = append(moved, newStop)
		mu.Unlock()
		return nil
	}
	t.Cleanup(func() { moveStopWire = oldWire })
	mentorRegisterLiveArm("swing-1", 42, "long", 29600)
	resetMentorCounters()
	at.mentorDispatchIntent(mentor.Intent{Action: mentor.ActionMoveStopBE, ArmID: "swing-1", Reason: "swing +1R"}, mentorTierInputs{}, 1000, 1100)
	mu.Lock()
	defer mu.Unlock()
	if len(moved) != 1 || moved[0] != 29600 {
		t.Fatalf("moveStopWire must be called once with the entry price, got %v", moved)
	}
	if c := MentorCountSnapshot()["move_be_sent"]; c != 1 {
		t.Fatalf("move_be_sent = %d, want 1", c)
	}
}

// TestMentorClosePositionReachesTheBroker — the swing's hold-close intent
// closes the registered side at the broker.
func TestMentorClosePositionReachesTheBroker(t *testing.T) {
	at, _, _, frames := mentorLoopback(t, ntwire.MinAddonBuildStopLimit)
	mentorRegisterLiveArm("swing-1", 42, "short", 29600)
	resetMentorCounters()
	at.mentorDispatchIntent(mentor.Intent{Action: mentor.ActionClosePosition, ArmID: "swing-1", Reason: "hold to the 2nd 4h close"}, mentorTierInputs{}, 1000, 1100)
	if !sawMentorFrame(t, frames, ntwire.FrameClosePosition, 1*time.Second) {
		t.Fatal("ClosePosition must reach the broker")
	}
	if c := MentorCountSnapshot()["close_sent"]; c != 1 {
		t.Fatalf("close_sent = %d, want 1", c)
	}
}

// TestMentorEntryStampsExpiryThroughTheBoundWire — the arm-creation site
// calls the bound arm-expiry seam (the unbind-expiry mutant turns this RED).
func TestMentorEntryStampsExpiryThroughTheBoundWire(t *testing.T) {
	t.Setenv("MENTOR_PLACE", "on")
	at, _, ledger, frames := mentorLoopback(t, ntwire.MinAddonBuildStopLimit)
	mentorWireSeams(t, at, ledger)
	var mu sync.Mutex
	var stamped []int64
	mentorSetArmExpiryWire = func(armID int64, expiryMs int64) error {
		mu.Lock()
		stamped = append(stamped, armID)
		mu.Unlock()
		return ledger.SetArmExpiry(armID, expiryMs)
	}
	expiry := time.Now().UnixMilli() + 60_000
	at.mentorDispatchIntent(mentor.Intent{Action: mentor.PlaceStopLimitEntry, ArmID: "isb-stamp", Setup: "ISB", Side: mentor.SideLong,
		Price: 29600, Stop: 29595, Target: 29610, StopPts: 5, TargetPts: 10, ExpiryMs: expiry}, mentorTierInputs{}, 1000, 1100)
	if !sawMentorFrame(t, frames, ntwire.FrameSignal, 2*time.Second) {
		t.Fatal("the mentor entry must reach the wire")
	}
	mu.Lock()
	defer mu.Unlock()
	if len(stamped) != 1 {
		t.Fatalf("the bound arm-expiry seam must be called once at the arm-creation site, got %d calls", len(stamped))
	}
}
