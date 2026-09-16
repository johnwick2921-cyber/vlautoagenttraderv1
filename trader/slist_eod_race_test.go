package trader

import (
	"strings"
	"sync"
	"testing"
	"time"

	"nofx/kernel"
	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	"nofx/trader/types"
)

// ═══════════════════════════════════════════════════════════════════════════
// S-LIST CLOSER FIX1 — EOD race fixtures (2026-08-27).
//
// The race (deep-verify hole 11): enforceEODFlatAt flattened POSITIONS only,
// and the armed cancel ran on the NEXT cycle — a working limit could fill up
// to one 2m cycle after the flat. The fix cancels every working arm FIRST
// (synchronous, ack-waited, one retry), THEN flattens, on all three lifecycle
// paths: EOD flat, session end, dormancy (+ T1 force-flat, same class).
//
// Fixtures: (a) 14:45 + open position + working arm → cancel frames strictly
// precede the flatten in the wire order (twin long/short); (b) cancel-ack
// timeout → flatten still proceeds + loud WARN path (ledger reason); (c)
// dormancy + working arm → wire cancel via the sync path; (d) session end
// (between-session gap) + working arm → same ordering.
// ═══════════════════════════════════════════════════════════════════════════

// wireRecorder embeds MockTrader and records the flatten wire frames
// (close_long / close_short / cancel_stops) so the cancel-before-flatten
// ordering is asserted on the exact sequence the production code emits.
type wireRecorder struct {
	*MockTrader
	mu     sync.Mutex
	events []string
}

var _ types.Trader = (*wireRecorder)(nil)

func (f *wireRecorder) record(ev string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, ev)
}

func (f *wireRecorder) snapshot() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.events...)
}

func (f *wireRecorder) CloseLong(symbol string, quantity float64) (map[string]interface{}, error) {
	f.record("close_long:" + symbol)
	return map[string]interface{}{}, nil
}

func (f *wireRecorder) CloseShort(symbol string, quantity float64) (map[string]interface{}, error) {
	f.record("close_short:" + symbol)
	return map[string]interface{}{}, nil
}

func (f *wireRecorder) CancelStopOrders(symbol string) error {
	f.record("cancel_stops:" + symbol)
	return nil
}

// eodFixture builds the EOD scene: day_plan on, one open MNQ position, one
// WORKING armed row with a signal, the recording trader bound, and the sync
// cancel seam wired so every wire cancel lands in the shared event log.
// onCancel (optional) is invoked with each cancelled signal (ack pumps use it).
// offset (optional) sets the per-session EOD-flat offset override — with
// offset=15 the flat resolves to 14:30, so a 14:30 instant exercises the
// IN-SESSION branch (default offset 0 flattens at session end).
func eodFixture(t *testing.T, now time.Time, side string, timeout time.Duration, onCancel func(sid string), acks <-chan ntwire.OrderUpdatePayload, offset *int) (*AutoTrader, *wireRecorder) {
	t.Helper()
	cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	if offset != nil {
		trueV := true
		cfg.DayPlan.Sessions = []store.DayPlanSessionOverride{{Session: "NY", Enable: &trueV, EODFlatOffsetMin: offset}}
	}
	at, st := resetTrader(t, cfg)
	rt := &wireRecorder{MockTrader: &MockTrader{}}
	at.trader = rt
	at.armedSyncSeam = &armedSyncSeam{
		Cancel: func(sid string) error {
			rt.record("cancel:" + sid)
			if onCancel != nil {
				onCancel(sid)
			}
			return nil
		},
		Stream:  func() <-chan ntwire.OrderUpdatePayload { return acks },
		Timeout: timeout,
	}
	sideU := strings.ToUpper(side)
	if err := st.Position().Create(&store.TraderPosition{
		TraderID: at.id, Symbol: "MNQ", Side: sideU, Account: "Sim101",
		ExchangeType: "ninjatrader", EntryQuantity: 1, Quantity: 1,
		EntryPrice: 30000, EntryTime: now.Add(-2 * time.Hour).UnixMilli(),
		Status: "OPEN",
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.ArmedOrders().UpsertArm(&store.ArmedOrderDB{
		TraderID: at.id, PlanID: "2026-08-18:NY:trader-1", Version: 1, Session: "NY",
		Scenario: "S1", Side: side, EntryPx: 29950, StopPx: 29970, TargetPx: 29910,
		State: "working", SignalID: "sig-eod",
	}); err != nil {
		t.Fatal(err)
	}
	return at, rt
}

// ackPump forwards a cancelled ack for every cancel the seam issues, after a
// small delay (the NT8 round-trip the drain must actually wait on).
func ackPump(t *testing.T, acks chan<- ntwire.OrderUpdatePayload) func(sid string) {
	t.Helper()
	return func(sid string) {
		time.Sleep(5 * time.Millisecond)
		select {
		case acks <- ntwire.OrderUpdatePayload{SignalID: sid, State: "cancelled"}:
		case <-time.After(2 * time.Second):
			t.Fatal("ack pump stalled")
		}
	}
}

// assertWireOrder fails unless every "cancel:" event precedes the first flatten
// frame (close_long/close_short), and both happened.
func assertWireOrder(t *testing.T, side string, ev []string) {
	t.Helper()
	flatten := map[string]string{"long": "close_long:MNQ", "short": "close_short:MNQ"}[side]
	firstClose, firstCancel := -1, -1
	for i, e := range ev {
		if strings.HasPrefix(e, "cancel:sig-eod") && firstCancel < 0 {
			firstCancel = i
		}
		if e == flatten && firstClose < 0 {
			firstClose = i
		}
	}
	if firstCancel < 0 {
		t.Fatalf("%s: no wire cancel in %v", side, ev)
	}
	if firstClose < 0 {
		t.Fatalf("%s: no flatten frame in %v", side, ev)
	}
	if firstCancel > firstClose {
		t.Fatalf("%s: cancel (%d) did not precede flatten (%d): %v", side, firstCancel, firstClose, ev)
	}
}

// TestSListEODFlatCancelsArmsBeforeFlatten — fixture (a), twin long/short.
// 14:45 CT Tuesday (NY flat time) with an open position AND a working arm: the
// cancel frames must strictly precede the flatten on the wire, the ledger must
// end terminal, and a late fill frame must NOT resurrect the cancelled row
// (no new fill possible post-flat).
func TestSListEODFlatCancelsArmsBeforeFlatten(t *testing.T) {
	// 14:30 CT Tuesday with a 15-min EOD offset override → the IN-SESSION
	// EOD-flat branch (default offset 0 would only fire at the boundary).
	now := time.Date(2026, 8, 18, 14, 30, 0, 0, chicagoLoc())
	offset := 15
	for _, side := range []string{"long", "short"} {
		acks := make(chan ntwire.OrderUpdatePayload, 4)
		at, rt := eodFixture(t, now, side, time.Second, ackPump(t, acks), acks, &offset)
		if !at.enforceEODFlatAt(now) {
			t.Fatalf("%s: enforceEODFlatAt must act", side)
		}
		ev := rt.snapshot()
		assertWireOrder(t, side, ev)
		rows, err := at.store.ArmedOrders().ListNonTerminal(at.id)
		if err != nil || len(rows) != 0 {
			t.Fatalf("%s: armed rows still non-terminal: %+v err=%v", side, rows, err)
		}
		// A late fill must not resurrect a cancelled row.
		at.onArmedOrderUpdate(ntwire.OrderUpdatePayload{SignalID: "sig-eod", State: "filled", FillPrice: 29950}, at.store.ArmedOrders())
		all, _ := at.store.ArmedOrders().ListForPlan("2026-08-18:NY:trader-1")
		if len(all) != 1 || all[0].State != "cancelled" {
			t.Fatalf("%s: late fill resurrected a cancelled arm: %+v", side, all)
		}
	}
}

// TestSListEODFlatCancelAckTimeoutStillFlattens — fixture (b). The ack stream
// never delivers: the sync cancel retries once, logs the loud WARN path (the
// ledger carries the honest reason), and the flatten proceeds regardless.
//
// UPDATED 2026-09-10 (settlement wave, Section C1). This asserted the row was
// 'cancelled' after an ack timeout. That was the defect: a TIMEOUT promoted the
// row to the word that frees the arm slot and that cutover leg 4 counts,
// skipping RequestCancel → cancel_pending → ConfirmCancel entirely, so a row
// could be replaced while its order still rested at the broker. Not hearing an
// ack is not evidence the order is gone.
//
// THE TEST'S REAL SUBJECT IS UNCHANGED and still asserted below: the flatten
// PROCEEDS, the cancel is retried exactly once, and the wire order holds. Only
// the ledger claim moved — from "cancelled" to "held cancel_pending, and the
// settlement pass owns it".
func TestSListEODFlatCancelAckTimeoutStillFlattens(t *testing.T) {
	now := time.Date(2026, 8, 18, 14, 30, 0, 0, chicagoLoc())
	offset := 15
	silent := make(chan ntwire.OrderUpdatePayload) // never fed
	at, rt := eodFixture(t, now, "long", 150*time.Millisecond, nil, silent, &offset)
	if !at.enforceEODFlatAt(now) {
		t.Fatal("must flatten despite unacked cancel")
	}
	ev := rt.snapshot()
	var cancels int
	for _, e := range ev {
		if strings.HasPrefix(e, "cancel:sig-eod") {
			cancels++
		}
	}
	if cancels != 2 {
		t.Fatalf("cancel attempts = %d, want 2 (one retry): %v", cancels, ev)
	}
	assertWireOrder(t, "long", ev)
	rows, _ := at.store.ArmedOrders().ListForPlan("2026-08-18:NY:trader-1")
	if len(rows) != 1 || rows[0].State != store.StateCancelPending || !strings.Contains(rows[0].StateReason, "ack timeout") {
		t.Fatalf("an unacked cancel must HOLD the row %q with the honest reason — promoting it on a timeout is the C1 defect: %+v",
			store.StateCancelPending, rows)
	}
	if rows[0].CancelSettledSnapshotID != 0 {
		t.Fatalf("nothing confirmed the cancel, so no settling snapshot may be recorded: %+v", rows[0])
	}
}

// TestSListDormancySyncCancelsWorkingArm — fixture (c). A dormant lifecycle
// flip with a WORKING arm must issue a wire cancel through the sync path (the
// pre-fix code only flipped the ledger — no wire frame would appear here).
func TestSListDormancySyncCancelsWorkingArm(t *testing.T) {
	cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	at, st := resetTrader(t, cfg)
	now := time.Now()
	sess, ok := at.sessionRegistry(now).ActiveSession(now)
	if !ok {
		t.Skip("no active session right now")
	}
	cfg.DayPlan.SessionsEnabled = []string{sess.Name}
	trueV := true
	cfg.DayPlan.Sessions = []store.DayPlanSessionOverride{{Session: sess.Name, Enable: &trueV}}
	td, _ := kernel.PlanChainTradeDate(sess, now)
	pid := store.MakePlanIDForTrader(at.id, td, sess.Name)
	v, err := st.Plan().AppendPlan(&store.PlanDB{PlanID: pid, TradeDate: td, Session: sess.Name, StrategyID: at.id, Lifecycle: "active", Doc: armedDoc(), CreatedAt: now.Add(-30 * time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.ArmedOrders().UpsertArm(&store.ArmedOrderDB{
		TraderID: at.id, PlanID: pid, Version: v, Session: sess.Name, Scenario: "S1",
		Side: "long", EntryPx: 100, StopPx: 95, TargetPx: 110, State: "working", SignalID: "sig-dorm",
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Plan().UpdatePlanLifecycle(pid, v, "dormant", "dormant:flip-condition: test"); err != nil {
		t.Fatal(err)
	}
	installActivePlanProvider(at, st)
	rt := &wireRecorder{MockTrader: &MockTrader{}}
	at.trader = rt
	acks := make(chan ntwire.OrderUpdatePayload, 4)
	at.armedSyncSeam = &armedSyncSeam{
		Cancel: func(sid string) error {
			rt.record("cancel:" + sid)
			go func() {
				time.Sleep(5 * time.Millisecond)
				acks <- ntwire.OrderUpdatePayload{SignalID: sid, State: "cancelled"}
			}()
			return nil
		},
		Stream:  func() <-chan ntwire.OrderUpdatePayload { return acks },
		Timeout: time.Second,
	}
	at.maybeManageArmedOrdersAt(nil, armTestClock(t, at))
	rows, err := st.ArmedOrders().ListNonTerminal(at.id)
	if err != nil || len(rows) != 0 {
		t.Fatalf("dormant must cancel ALL arms (rows=%d err=%v)", len(rows), err)
	}
	ev := rt.snapshot()
	if len(ev) == 0 || ev[0] != "cancel:sig-dorm" {
		t.Fatalf("dormancy must issue a WIRE cancel via the sync path, got %v", ev)
	}
}

// TestSListSessionEndSyncCancelsWorkingArm — fixture (d). 15:30 CT (the
// 14:45→17:00 between-session gap): no active session + open position +
// working arm → the session-end flatten must cancel the arm first, then close.
func TestSListSessionEndSyncCancelsWorkingArm(t *testing.T) {
	now := time.Date(2026, 8, 18, 15, 30, 0, 0, chicagoLoc())
	acks := make(chan ntwire.OrderUpdatePayload, 4)
	at, rt := eodFixture(t, now, "long", time.Second, ackPump(t, acks), acks, nil)
	if !at.enforceEODFlatAt(now) {
		t.Fatal("session-end flatten must act")
	}
	ev := rt.snapshot()
	assertWireOrder(t, "long", ev)
	rows, err := at.store.ArmedOrders().ListNonTerminal(at.id)
	if err != nil || len(rows) != 0 {
		t.Fatalf("session-end must cancel ALL arms (rows=%d err=%v)", len(rows), err)
	}
}

// TestSListSyncCancelSkipsOtherTraders — the sync cancel only touches THIS
// trader's rows (the same filter the async path uses).
func TestSListSyncCancelSkipsOtherTraders(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}})
	other := &store.ArmedOrderDB{TraderID: "trader-2", PlanID: "x:NY", Version: 1, Session: "NY",
		Scenario: "S9", Side: "long", EntryPx: 100, StopPx: 95, TargetPx: 110, State: "working", SignalID: "sig-other"}
	if err := st.ArmedOrders().UpsertArm(other); err != nil {
		t.Fatal(err)
	}
	n, _ := at.cancelArmedOrdersSync("test")
	if n != 0 {
		t.Fatalf("sync cancel touched another trader's row (n=%d)", n)
	}
	rows, _ := st.ArmedOrders().ListNonTerminal("trader-2")
	if len(rows) != 1 || rows[0].SignalID != "sig-other" {
		t.Fatalf("other trader's working arm must survive: %+v", rows)
	}
}
