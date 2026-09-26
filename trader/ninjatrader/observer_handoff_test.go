// W117 F5 (d3e4638e, re-derived) — a delayed old reconcile/close goroutine
// must never subscribe AFTER a same-account replacement and steal its stream.
// Same-account replacement drains: the old close consumer's channel closes, it
// drains its queued receipts, closes observerDone, and the reconcile worker
// retires (reconcileStopped). A foreign account or a socket disconnect does
// NOT retire the observer. RED = remove the close consumer's
// `defer close(done)` + the reconcile select → the old worker survives the
// replacement and the pin times out.

package ninjatrader

import (
	"path/filepath"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

func TestObserverStartReplacementRetiresOldReconcileOnly(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "handoff.db"))
	if err != nil {
		t.Fatal(err)
	}
	srv := ntwire.NewTCPServer(nil)
	old := &TCPTrader{server: srv, symbol: "MNQ", boundAccount: "Sim101"}
	old.StartCloseSync("owned", "ex", "ninjatrader", st)
	old.StartPositionReconcile("owned", "ex", "ninjatrader", st)
	old.mu.Lock()
	stopped := old.reconcileStopped
	old.mu.Unlock()
	if stopped == nil {
		t.Fatal("the reconcile worker must publish its stopped channel")
	}

	foreign := &TCPTrader{server: srv, symbol: "MNQ", boundAccount: "SimOther"}
	foreign.StartCloseSync("other", "ex", "ninjatrader", st)
	select {
	case <-old.observerLifetime():
		t.Fatal("a FOREIGN account must not retire the observer")
	default:
	}

	// A disconnected bridge does not mean the observer was replaced.
	if err := srv.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-old.observerLifetime():
		t.Fatal("a socket disconnect must not retire the observer")
	default:
	}

	// Same-account replacement drains the old observer chain.
	next := &TCPTrader{server: srv, symbol: "MNQ", boundAccount: "Sim101"}
	next.StartCloseSync("owned", "ex", "ninjatrader", st)
	select {
	case <-stopped:
	case <-time.After(time.Second):
		t.Fatal("the old reconcile worker survived the successful replacement")
	}

	// A re-entrant Start on the RETIRED instance cannot steal the replacement.
	old.StartCloseSync("owned", "ex", "ninjatrader", st)
	select {
	case <-next.observerLifetime():
		t.Fatal("the old instance's re-entrant Start stole the replacement's subscription")
	default:
	}
}
