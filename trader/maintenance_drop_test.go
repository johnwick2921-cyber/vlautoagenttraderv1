package trader

import (
	"context"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// ── W-ONE-BUTTON M2, M-2 — an entry the hold dropped from the queue ────────
//
// Production path: a real TCP server, an NT8 TCPTrader and the store. An entry
// sent while NT8 is DISCONNECTED is queued (SendSignal returns nil, so its
// caller already recorded it); the hold then drops it. A never-attempted drop
// provably never reached NT8, so the trader SETTLES what it recorded; an
// attempted one stays ambiguous (place_pending) and keeps the gate closed.
// Nothing is ever resent.

type dropWire struct {
	s    *ntwire.TCPServer
	nt   *ntTrader.TCPTrader
	at   *AutoTrader
	st   *store.Store
	dir  string
	addr string
}

// newDropWire: a started server whose far-side build is proven by a client
// that then DISCONNECTS, so the next entry is queued, not written.
func newDropWire(t *testing.T) *dropWire {
	t.Helper()
	dir := withMaintenanceDir(t)
	st, err := store.New(filepath.Join(t.TempDir(), "drop.db"))
	if err != nil {
		t.Fatal(err)
	}
	s := ntwire.NewTCPServer(nil)
	s.SetAddrForTest("127.0.0.1:0")
	s.SetAccountsList([]ntwire.AccountInfo{{Name: "Sim101", IsSim: true}}, "Sim101")
	ctx, cancel := context.WithCancel(context.Background())
	if err := s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Stop(); cancel(); _ = st.Close() })
	c, err := net.Dial("tcp", s.ListenAddrForTest().String())
	if err != nil {
		t.Fatal(err)
	}
	top := ntwire.MinAddonBuildPictureHtf
	if err := ntwire.WriteFrame(c, ntwire.FrameHeartbeat, ntwire.HeartbeatPayload{BuildID: top}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 200 && !ntwire.FarSideProven(s.FarSideBuildID(), top); i++ {
		time.Sleep(5 * time.Millisecond)
	}
	_ = c.Close()
	for i := 0; i < 200 && s.IsConnected(); i++ {
		time.Sleep(5 * time.Millisecond)
	}
	if s.IsConnected() || !ntwire.FarSideProven(s.FarSideBuildID(), top) {
		t.Fatal("fixture: want a proven far side and NO connected client")
	}
	nt := ntTrader.NewTCPTrader(s, "MNQ", "Sim101")
	at := &AutoTrader{id: "drop-trader-1", store: st, exchange: "ninjatrader", trader: nt}
	at.config.StrategyConfig = &store.StrategyConfig{}
	// Exactly what NewAutoTrader wires.
	nt.SetEntryHoldCheck(maintenanceQueueHeld)
	nt.SetDroppedEntrySink(at.onMaintenanceDroppedEntry)
	return &dropWire{s: s, nt: nt, at: at, st: st, dir: dir, addr: s.ListenAddrForTest().String()}
}

func (w *dropWire) leg(t *testing.T, n int) CutoverLeg {
	t.Helper()
	for _, l := range w.at.CutoverGateStatus().Legs {
		if l.N == n {
			return l
		}
	}
	t.Fatalf("cutover leg %d missing", n)
	return CutoverLeg{}
}

// noSignalOnReconnect: a fresh client receives no signal frame (never resent).
func (w *dropWire) noSignalOnReconnect(t *testing.T) {
	t.Helper()
	c, err := net.Dial("tcp", w.addr)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetReadDeadline(time.Now().Add(600 * time.Millisecond))
	for {
		env, err := ntwire.ReadFrame(c)
		if err != nil {
			return // deadline: nothing more arrived
		}
		if env.Type == ntwire.FrameSignal {
			t.Fatalf("a dropped entry was RESENT on reconnect: %s", env.Payload)
		}
	}
}

func waitDrop(t *testing.T, what string, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out after %s waiting for: %s", d, what)
		}
		time.Sleep(20 * time.Millisecond)
	}
}

func dropArmRow(t *testing.T, st *store.Store, trader string) store.ArmedOrderDB {
	t.Helper()
	r := store.ArmedOrderDB{TraderID: trader, PlanID: "2026-09-22:NY:" + trader, Version: 1, Session: "NY", Scenario: "S1",
		Side: "LONG", EntryPx: 29000, StopPx: 28950, TargetPx: 29100, State: store.StateArmed}
	if err := st.ArmedOrders().UpsertArm(&r); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestDroppedNeverSentArmedEntrySettlesAndLeg4Clears(t *testing.T) {
	w := newDropWire(t)
	r := dropArmRow(t, w.st, w.at.id)
	sid, err := w.nt.PlaceLimitEntry("MNQ", "long", 1, 29000, 28950, 29100, func(sid string) error {
		return w.st.ArmedOrders().BeginPlacement(r.ID, sid)
	})
	if err != nil || sid == "" {
		t.Fatalf("fixture: a disconnected placement queues and returns nil: sid=%q err=%v", sid, err)
	}
	if n := w.s.PendingSignalCount(); n != 1 {
		t.Fatalf("fixture: the entry must be QUEUED (disconnected), queue=%d", n)
	}
	if l := w.leg(t, 4); l.Pass {
		t.Fatalf("before the settle leg 4 must FAIL on the place_pending row: %+v", l)
	}

	setHold(t, w.dir, "job-drop")
	waitDrop(t, "the dropped row to settle", 5*time.Second, func() bool {
		row, _ := w.st.ArmedOrders().FindBySignal(w.at.id, sid)
		return row != nil && row.State == store.StateCancelled
	})
	row, _ := w.st.ArmedOrders().FindBySignal(w.at.id, sid)
	for _, want := range []string{"never sent — queued entry dropped by the maintenance hold", "job-drop"} {
		if !strings.Contains(row.StateReason, want) {
			t.Fatalf("state_reason %q must contain %q", row.StateReason, want)
		}
	}
	if l := w.leg(t, 4); !l.Pass {
		t.Fatalf("after the settle leg 4 must pass: %+v", l)
	}
	if w.nt.HasPendingEntry(sid) {
		t.Fatal("the trader must forget a never-sent entry")
	}
	w.noSignalOnReconnect(t)
}

// An attempted drop (a write was started — bytes may have reached NT8) is
// ambiguous: the row stays place_pending and leg 4 keeps failing.
func TestDroppedAttemptedEntryStaysPending(t *testing.T) {
	w := newDropWire(t)
	r := dropArmRow(t, w.st, w.at.id)
	if err := w.st.ArmedOrders().BeginPlacement(r.ID, "sig-attempted"); err != nil {
		t.Fatal(err)
	}
	setHold(t, w.dir, "job-attempted")
	before := gateBlocks(w.at.id, "maintenance_drop_attempted")
	w.at.onMaintenanceDroppedEntry(ntwire.DroppedEntry{SignalID: "sig-attempted", TraderID: w.at.id, Attempted: true})
	row, _ := w.st.ArmedOrders().FindBySignal(w.at.id, "sig-attempted")
	if row == nil || row.State != store.StatePlacePending {
		t.Fatalf("an attempted drop must stay place_pending: %+v", row)
	}
	if l := w.leg(t, 4); l.Pass {
		t.Fatalf("an ambiguous row must keep leg 4 failing: %+v", l)
	}
	if gateBlocks(w.at.id, "maintenance_drop_attempted") != before+1 {
		t.Fatal("the attempted drop must be counted")
	}
}

// CTO condition 3 — the AI market-entry path. placeEntry returns "submitted"
// for a QUEUED signal and its production caller, recordAndConfirmOrder, then
// writes an order row and — with no matching fill after ~3s — an OPEN position
// at the mark price whose entry_order_id is "<nil>" (placeEntry returns no
// "orderId"). Nothing links that row to the signal, so the drop cannot settle
// it without fabricating a close: the trader FORGETS the entry, says so, and
// the gate stays closed on db_open_positions until an operator reconciles.
func TestDroppedAIEntryIsForgottenAndTheGateStaysClosed(t *testing.T) {
	w := newDropWire(t)
	_ = w.nt.SetStopLoss("MNQ", "LONG", 1, 28950)
	_ = w.nt.SetTakeProfit("MNQ", "LONG", 1, 29100)
	order, err := w.nt.OpenLong("MNQ", 1, 1)
	if err != nil {
		t.Fatalf("fixture: a disconnected AI entry queues and returns nil: %v", err)
	}
	sid, _ := order["signal_id"].(string)
	if sid == "" || !w.nt.HasPendingEntry(sid) || w.s.PendingSignalCount() != 1 {
		t.Fatalf("fixture: the AI entry must be queued and pending: %+v", order)
	}
	// The production caller, exactly as executeOpenLongWithRecord runs it.
	w.at.recordAndConfirmOrder(order, "MNQ", "open_long", 1, 29000, 1, 0, 70)
	open, err := w.st.Position().GetOpenPositions(w.at.id)
	if err != nil || len(open) != 1 || open[0].EntryOrderID != "<nil>" {
		t.Fatalf("[A] the AI path records an OPEN row for a queued entry with entry_order_id \"<nil>\": %+v %v", open, err)
	}

	logs := captureTraderLog(t)
	setHold(t, w.dir, "job-ai")
	waitDrop(t, "the AI entry to be dropped and forgotten", 5*time.Second, func() bool { return !w.nt.HasPendingEntry(sid) })
	if out := logs.String(); !strings.Contains(out, sid) || !strings.Contains(out, "db_open_positions") {
		t.Fatalf("the drop of an unlinkable AI entry must be said out loud (signal + db_open_positions), log:\n%s", out)
	}
	if l := w.leg(t, 1); l.Pass {
		t.Fatalf("the unlinkable OPEN row must keep cutover leg 1 failing: %+v", l)
	}
	if open, _ := w.st.Position().GetOpenPositions(w.at.id); len(open) != 1 {
		t.Fatal("the drop must not fabricate a close for a row it cannot link")
	}
	w.noSignalOnReconnect(t)
}

// CTO on F3: an ambiguous (attempted) drop logs the ambiguity AND raises a P1 —
// an entry that may be at NT8 is an owner-visible event, not a log line.
func TestAttemptedDropRaisesAP1(t *testing.T) {
	w := newDropWire(t)
	w.at.config.StrategyConfig = &store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	setHold(t, w.dir, "job-amb")
	w.at.onMaintenanceDroppedEntry(ntwire.DroppedEntry{SignalID: "sig-amb", TraderID: w.at.id, Symbol: "MNQ", Side: "long", Attempted: true})
	rows, err := w.st.Alert().List(w.at.id, 10)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range rows {
		if r.Level == "P1" && strings.Contains(r.EventID, "sig-amb") {
			return
		}
	}
	t.Fatalf("an attempted drop must raise a P1 naming the signal; alerts: %+v", rows)
}
