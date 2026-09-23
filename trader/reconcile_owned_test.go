package trader

import (
	"context"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// ── W-EXEC-TRUTH W0 (c) — reconcileBeforeOpenNT never flattens a position a
// ledger explains; an unexplained one keeps today's flatten ─────────────────

type reconcileWire struct {
	at     *AutoTrader
	st     *store.Store
	s      *ntwire.TCPServer
	frames chan ntwire.FrameType
}

func newReconcileWire(t *testing.T) *reconcileWire {
	t.Helper()
	withMaintenanceDir(t)
	at, st := resetTrader(t, store.StrategyConfig{})
	at.id = "reconcile-owned"
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
	frames := make(chan ntwire.FrameType, 64)
	go func() {
		for {
			f, err := ntwire.ReadFrame(conn)
			if err != nil {
				return
			}
			frames <- f.Type
		}
	}()
	nt := ntTrader.NewTCPTrader(s, "MNQ", "Sim101")
	at.trader = nt
	at.config.NinjaTraderSymbol = "MNQ"
	s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{{Symbol: "MNQ", Side: "short", Quantity: 1, AvgPrice: 29000}})
	return &reconcileWire{at: at, st: st, s: s, frames: frames}
}

func (w *reconcileWire) closeSent(d time.Duration) bool {
	deadline := time.After(d)
	for {
		select {
		case f := <-w.frames:
			if f == ntwire.FrameClosePosition {
				return true
			}
		case <-deadline:
			return false
		}
	}
}

func (w *reconcileWire) armedRow(t *testing.T, state, sid string) *store.ArmedOrderDB {
	t.Helper()
	led := w.st.ArmedOrders()
	r := &store.ArmedOrderDB{TraderID: w.at.id, PlanID: "p", Scenario: "S-" + state, Version: 1, State: "armed", Side: "short", EntryPx: 29000, StopPx: 29010, TargetPx: 28970}
	if err := led.UpsertArm(r); err != nil {
		t.Fatal(err)
	}
	if err := led.BeginPlacement(r.ID, sid); err != nil {
		t.Fatal(err)
	}
	if state != "place_pending" {
		if err := led.SetState(r.ID, state, "fixture"); err != nil {
			t.Fatal(err)
		}
	}
	return r
}

// A held SHORT a working armed row explains is REFUSED, named, never flattened.
func TestReconcileRefusesToFlattenALedgerExplainedShort(t *testing.T) {
	w := newReconcileWire(t)
	w.armedRow(t, "working", "sig-armed-short")
	err := w.at.reconcileBeforeOpenNT("MNQ", "long")
	if !errors.Is(err, errPositionOwned) || !strings.Contains(err.Error(), "armed #") {
		t.Fatalf("a ledger-explained position must refuse the open, naming its owner: %v", err)
	}
	if w.closeSent(300 * time.Millisecond) {
		t.Fatal("a ledger-explained position was FLATTENED")
	}
}

// A working Picture row explains the position the same way.
func TestReconcileRefusesToFlattenAPictureExplainedShort(t *testing.T) {
	w := newReconcileWire(t)
	if _, _, err := w.st.PictureHtfClaim(&store.PictureHtfOpportunityDB{OppKey: "pic-short", TraderID: w.at.id, Account: "Sim101", Symbol: "MNQ", Direction: "short", Stage: "confirmed"}); err != nil {
		t.Fatal(err)
	}
	if err := w.st.PictureHtfTransition("pic-short", "working", "fixture"); err != nil {
		t.Fatal(err)
	}
	if err := w.at.reconcileBeforeOpenNT("MNQ", "long"); !errors.Is(err, errPositionOwned) || !strings.Contains(err.Error(), "picture pic-short") {
		t.Fatalf("a Picture-explained position must refuse the open: %v", err)
	}
	if w.closeSent(300 * time.Millisecond) {
		t.Fatal("a Picture-explained position was FLATTENED")
	}
}

// A fill younger than twice the untracked grace explains the position (not yet
// materialized); an older fill does not — that position is flattened.
func TestReconcileRecentFillExplainsOldFillDoesNot(t *testing.T) {
	w := newReconcileWire(t)
	r := w.armedRow(t, "filled", "sig-recent")
	if err := w.at.reconcileBeforeOpenNT("MNQ", "long"); !errors.Is(err, errPositionOwned) || !strings.Contains(err.Error(), "not yet materialized") {
		t.Fatalf("a recent fill must explain the position: %v", err)
	}
	old := time.Now().Add(-10 * time.Minute)
	if err := w.st.ArmedOrders().DB().Model(&store.ArmedOrderDB{}).Where("id = ?", r.ID).UpdateColumn("updated_at", old).Error; err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- w.at.reconcileBeforeOpenNT("MNQ", "long") }()
	if !w.closeSent(3 * time.Second) {
		t.Fatal("an unexplained held SHORT (only an old fill on the ledger) must be flattened as before")
	}
	w.s.SeedPositionsForTest("Sim101", nil) // the flatten confirms flat
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("after the confirmed flatten the open proceeds: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("reconcile never returned after the flatten confirmed")
	}
}

// With nothing on the ledger, a held SHORT is an orphan and is flattened —
// the canon-28 fix made shorts visible, the owner-ruled flatten is unchanged.
func TestReconcileStillFlattensAnUnexplainedShort(t *testing.T) {
	w := newReconcileWire(t)
	done := make(chan error, 1)
	go func() { done <- w.at.reconcileBeforeOpenNT("MNQ", "long") }()
	if !w.closeSent(3 * time.Second) {
		t.Fatal("an unexplained held SHORT must be flattened")
	}
	w.s.SeedPositionsForTest("Sim101", nil)
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("after the confirmed flatten the open proceeds: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("reconcile never returned")
	}
}

// The open paths classify the ledger-owned refusal as a ⛔ gate: counted,
// stamped on the record, and nil — never the ❌ error branch.
func TestReconcileRefusalIsAGateNotAnError(t *testing.T) {
	at, _ := resetTrader(t, store.StrategyConfig{})
	at.id = "reconcile-refusal"
	rec := &store.DecisionAction{}
	before := gateBlocks(at.id, "reconcile_owned")
	if err := at.reconcileRefusal(errPositionOwnedFor("armed #7"), rec); err != nil {
		t.Fatalf("a ledger-owned refusal must return nil (refused, not failed): %v", err)
	}
	if rec.Success || !strings.HasPrefix(rec.Error, "reconcile_owned: ") || gateBlocks(at.id, "reconcile_owned") != before+1 {
		t.Fatalf("the refusal must be stamped and counted: %+v", rec)
	}
	other := errors.New("reconcile-before-open: flatten not confirmed flat")
	if err := at.reconcileRefusal(other, rec); err != other {
		t.Fatalf("any other reconcile error stays the error it was: %v", err)
	}
	src := readSource(t, "auto_trader_orders.go")
	if strings.Count(src, "return at.reconcileRefusal(err, actionRecord)") != 2 {
		t.Fatal("both open paths must route reconcileBeforeOpenNT's error through reconcileRefusal")
	}
}

func errPositionOwnedFor(owner string) error { return errors.Join(errPositionOwned, errors.New(owner)) }

// otherRunningTraderWithFreshFill registers a second MNQ trader on the same
// account the way Run does (the running-trader registry), with NO Picture
// evaluator, and an armed SHORT on it that filled just now.
func (w *reconcileWire) otherRunningTraderWithFreshFill(t *testing.T, register bool) {
	t.Helper()
	other := &AutoTrader{id: "reconcile-other", store: w.st, trader: ntTrader.NewTCPTrader(w.s, "MNQ", "Sim101")}
	other.config.NinjaTraderSymbol = "MNQ"
	if register {
		registerPostExitDispatch(other)
		t.Cleanup(func() { unregisterPostExitDispatch(other) })
	}
	led := w.st.ArmedOrders()
	r := &store.ArmedOrderDB{TraderID: other.id, PlanID: "p-other", Scenario: "S1", Version: 1, State: "armed", Side: "short", EntryPx: 29000, StopPx: 29010, TargetPx: 28970}
	if err := led.UpsertArm(r); err != nil {
		t.Fatal(err)
	}
	if err := led.BeginPlacement(r.ID, "sig-other-fill"); err != nil {
		t.Fatal(err)
	}
	if err := led.SetState(r.ID, store.StateFilled, "fixture"); err != nil {
		t.Fatal(err)
	}
}

// (iii) reads every RUNNING trader in the process — the registry Run and Stop
// maintain — not only the traders that own a Picture evaluator: another MNQ
// trader's fill seconds ago on this account explains the held position.
func TestReconcileSeesAFreshFillOfAnotherRunningTrader(t *testing.T) {
	w := newReconcileWire(t)
	w.otherRunningTraderWithFreshFill(t, true)
	done := make(chan error, 1)
	go func() { done <- w.at.reconcileBeforeOpenNT("MNQ", "long") }()
	if w.closeSent(300 * time.Millisecond) {
		w.s.SeedPositionsForTest("Sim101", nil)
		<-done
		t.Fatal("another running trader's fresh fill was FLATTENED as an orphan")
	}
	if err := <-done; !errors.Is(err, errPositionOwned) || !strings.Contains(err.Error(), "not yet materialized") {
		t.Fatalf("another running trader's fresh fill must explain the position: %v", err)
	}
}

// THE LIMIT, named: a trader that is STOPPED is not in the running registry,
// so its fill seconds ago is not seen by (iii) and the position reads as an
// orphan and is flattened. A stopped trader's in-flight fill is the one case
// (iii) cannot explain without a ledger-wide fill query (not built in W0b).
func TestReconcileFreshFillOfAStoppedTraderIsNotSeen(t *testing.T) {
	w := newReconcileWire(t)
	w.otherRunningTraderWithFreshFill(t, false)
	done := make(chan error, 1)
	go func() { done <- w.at.reconcileBeforeOpenNT("MNQ", "long") }()
	if !w.closeSent(3 * time.Second) {
		t.Fatal("the named limit moved: a stopped trader's fill now explains the position — update this test and the comment")
	}
	w.s.SeedPositionsForTest("Sim101", nil)
	if err := <-done; err != nil {
		t.Fatalf("after the confirmed flatten the open proceeds: %v", err)
	}
}
