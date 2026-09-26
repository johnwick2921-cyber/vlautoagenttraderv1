package trader

import (
	"context"
	"encoding/json"
	"net"
	"strconv"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// ── CLASS 33 BOOK GATE — the PRODUCTION sweep call site ─────────────────────
//
// B2 moved the boot sweep's cancel send behind cancelSignalIfSafe, the same
// broker-book gate every other cancel rides. The seams tests
// (sweepPreBootArmsWith + a nil-returning recorder fn) cannot see that gate —
// they pass a send that always succeeds. This test drives the WIRED wrapper
// at.sweepPreBootArms with a real armedTrader on a real TCP server, a dialed
// fake AddOn client to observe the frames, and a FED order-snapshot cache.
//
// The case it protects is the one the gate exists for: a pre-boot arm that
// FILLED during the restart. Its bracket children are the position's
// protection, and a blind boot cancel would take them (2026-09-06 23:37:02).

// class33GateRig is the minimal wire rig for the production sweep path.
type class33GateRig struct {
	t    *testing.T
	at   *AutoTrader
	st   *store.Store
	srv  *ntwire.TCPServer
	conn net.Conn
	ev   chan zoneFrame
	seq  int
}

func newClass33GateRig(t *testing.T, id string) *class33GateRig {
	t.Helper()
	at := plannerTestTrader(t)
	at.id = id
	if err := at.store.ArmedOrders().Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	srv := ntwire.NewTCPServer(nil)
	srv.SetAddrForTest("127.0.0.1:0")
	srv.SetAccountsList([]ntwire.AccountInfo{{Name: "Sim101", IsSim: true}}, "Sim101")
	ctx, cancel := context.WithCancel(context.Background())
	if err := srv.Start(ctx); err != nil {
		t.Fatalf("server start: %v", err)
	}
	t.Cleanup(func() { _ = srv.Stop(); cancel() })
	conn, err := net.Dial("tcp", srv.ListenAddrForTest().String())
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	waitAddonRegistered(t, srv)
	t.Cleanup(func() { _ = conn.Close() })

	at.trader = ntTrader.NewTCPTrader(srv, "MNQ", "Sim101")

	ev := make(chan zoneFrame, 64)
	done := make(chan struct{})
	t.Cleanup(func() { close(done) })
	go func() {
		for {
			env, err := ntwire.ReadFrame(conn)
			if err != nil {
				return
			}
			var f zoneFrame
			switch env.Type {
			case ntwire.FrameCancelOrder:
				var p ntwire.CancelOrderPayload
				if json.Unmarshal(env.Payload, &p) != nil {
					continue
				}
				f.cancel = &p
			case ntwire.FrameBarsHistoryRequest:
				var p ntwire.BarsHistoryRequestPayload
				if json.Unmarshal(env.Payload, &p) != nil || p.Symbol != zoneSentinel {
					continue
				}
				f.sentinel = p.RequestID
			default:
				continue
			}
			select {
			case ev <- f:
			case <-done:
				return
			}
		}
	}()
	return &class33GateRig{t: t, at: at, st: at.store, srv: srv, conn: conn, ev: ev}
}

// feed stamps a FRESH book (received now) into the server's snapshot cache —
// the exact source cancelSignalIfSafe's liveBook reads at the production site.
func (r *class33GateRig) feed(orders []ntwire.NT8Order) {
	r.t.Helper()
	r.srv.OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: orders}, time.Now())
}

// drain barriers on a sentinel and returns every cancel frame written before
// it. A producer that sent no cancel yields an empty (not nil) slice.
func (r *class33GateRig) drain() []ntwire.CancelOrderPayload {
	r.t.Helper()
	r.seq++
	id := zoneSentinel + "-" + strconv.Itoa(r.seq)
	if err := r.srv.SendBarsHistoryRequest(ntwire.BarsHistoryRequestPayload{RequestID: id, Symbol: zoneSentinel}); err != nil {
		r.t.Fatalf("sentinel: %v", err)
	}
	deadline := time.After(3 * time.Second)
	cancels := []ntwire.CancelOrderPayload{}
	for {
		select {
		case f := <-r.ev:
			if f.cancel != nil {
				cancels = append(cancels, *f.cancel)
			}
			if f.sentinel == id {
				return cancels
			}
		case <-deadline:
			r.t.Fatal("sentinel never came back")
			return nil
		}
	}
}

func (r *class33GateRig) row(scenario string) store.ArmedOrderDB {
	r.t.Helper()
	ledger := r.st.ArmedOrders()
	var row store.ArmedOrderDB
	if err := ledger.DB().Where("plan_id = ? AND scenario = ?", "p1", scenario).First(&row).Error; err != nil {
		r.t.Fatalf("read row %s: %v", scenario, err)
	}
	return row
}

func (r *class33GateRig) sweptLatchSet() bool {
	_, done := bootSweepDone.Load(r.at.id)
	return done
}

// TestClass33ProductionSweepRespectsBookGate drives the WIRED sweep (the real
// wrapper, the real armedTrader, a fed book) and pins the three book shapes:
//
//	(a) no fresh book  → NO cancel sent, row stays working, latch unset, and
//	    the sweep retries next cycle;
//	(b) children only  → NO cancel sent (the protection is kept);
//	(c) entry resting  → cancel sent, row becomes cancel_pending.
func TestClass33ProductionSweepRespectsBookGate(t *testing.T) {
	const sig = "pre-boot-sig"
	const traderID = "gate-t1"

	setup := func(t *testing.T) (*class33GateRig, *store.ArmedOrderStore) {
		t.Helper()
		ResetBootSweepForTest()
		rig := newClass33GateRig(t, traderID)
		class33Seed(t, rig.at, "S1", sig, "dead-boot", store.StateWorking)
		return rig, rig.st.ArmedOrders()
	}

	t.Run("a no fresh book", func(t *testing.T) {
		rig, ledger := setup(t)
		// The rig starts with NO snapshot ever received.
		rig.at.sweepPreBootArms(ledger)
		if cancels := rig.drain(); len(cancels) != 0 {
			t.Fatalf("no fresh book: the sweep must NOT send a cancel, wire saw %+v", cancels)
		}
		if row := rig.row("S1"); row.State != store.StateWorking {
			t.Fatalf("no fresh book: the row must stay working, got %q reason %q", row.State, row.StateReason)
		}
		if rig.sweptLatchSet() {
			t.Fatalf("no fresh book: the latch must stay unset — a refused sweep is INCOMPLETE, not done")
		}
		// It retries next cycle: the second call still sends nothing and the
		// row is untouched.
		rig.at.sweepPreBootArms(ledger)
		if cancels := rig.drain(); len(cancels) != 0 {
			t.Fatalf("retry cycle: still no book, still no cancel, wire saw %+v", cancels)
		}
		if row := rig.row("S1"); row.State != store.StateWorking {
			t.Fatalf("retry cycle: the row must stay working, got %q", row.State)
		}
		if rig.sweptLatchSet() {
			t.Fatalf("retry cycle: the latch must still be unset")
		}
	})

	t.Run("b children only", func(t *testing.T) {
		rig, ledger := setup(t)
		// The entry has FILLED: only its bracket children remain. Cancelling
		// now takes the position's protection with it.
		rig.feed([]ntwire.NT8Order{
			{Name: sig + "-sl", State: "Working", Symbol: "MNQ"},
			{Name: sig + "-tp", State: "Working", Symbol: "MNQ"},
		})
		rig.at.sweepPreBootArms(ledger)
		if cancels := rig.drain(); len(cancels) != 0 {
			t.Fatalf("children only: the sweep must NOT cancel the protection, wire saw %+v", cancels)
		}
		if row := rig.row("S1"); row.State != store.StateWorking {
			t.Fatalf("children only: the row must stay working, got %q", row.State)
		}
		if rig.sweptLatchSet() {
			t.Fatalf("children only: the latch must stay unset")
		}
	})

	t.Run("c entry resting", func(t *testing.T) {
		rig, ledger := setup(t)
		rig.feed([]ntwire.NT8Order{{Name: sig, State: "Working", Symbol: "MNQ"}})
		rig.at.sweepPreBootArms(ledger)
		cancels := rig.drain()
		if len(cancels) != 1 || cancels[0].SignalID != sig {
			t.Fatalf("entry resting: want exactly one cancel for %s, wire saw %+v", sig, cancels)
		}
		row := rig.row("S1")
		if row.State != store.StateCancelPending {
			t.Fatalf("entry resting: the row must become cancel_pending after the request, got %q", row.State)
		}
		if !store.IsBootSweepReason(row.StateReason) {
			t.Fatalf("entry resting: the row must carry the boot_sweep reason, got %q", row.StateReason)
		}
		if !rig.sweptLatchSet() {
			t.Fatalf("entry resting: a completed sweep must latch")
		}
		if got := store.CountFromSystemConfig(rig.st, store.BootSweptKey); got != 1 {
			t.Fatalf("the recorded counter must count the SEND, got %d", got)
		}
	})
}
