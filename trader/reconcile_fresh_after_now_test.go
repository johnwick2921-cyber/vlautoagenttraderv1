package trader

import (
	"testing"
	"time"

	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// ── W1b FOLD-11 — a fill stamped AFTER the captured now is fresher than fresh
//
// reconcileBeforeOpenNT captures now (time.Now(), auto_trader_orders.go) BEFORE
// ledgerExplainsPosition reads the ledger. A settle pass that moves an armed
// row working→filled in between stamps UpdatedAt AFTER that now: (i) no longer
// lists it (filled is terminal), the position is not yet materialized, and
// fresh() read d = now − UpdatedAt < 0 as NOT fresh — so the position was
// UNEXPLAINED and FLATTENED: the D10 class W0 (c) exists for, through a
// sub-millisecond window. The same held for a Picture fill. Driven through the
// production call site; the "after now" stamp is written on the ledger row
// before the call, standing for the settle pass's write that lands after the
// capture.
func TestReconcileFillStampedAfterTheCapturedNowExplainsThePosition(t *testing.T) {
	after := func() time.Time { return time.Now().Add(30 * time.Second) } // later than any now the call can capture

	t.Run("armed", func(t *testing.T) {
		w := newReconcileWire(t)
		r := w.armedRow(t, store.StateFilled, "sig-filled-after-now")
		if err := w.st.ArmedOrders().DB().Model(&store.ArmedOrderDB{}).Where("id = ?", r.ID).
			UpdateColumn("updated_at", after()).Error; err != nil {
			t.Fatal(err)
		}
		w.refusedNotFlattened(t, "not yet materialized")
	})

	t.Run("picture", func(t *testing.T) {
		w := newReconcileWire(t)
		if _, _, err := w.st.PictureHtfClaim(&store.PictureHtfOpportunityDB{OppKey: "pic-after-now", TraderID: w.at.id, Account: "Sim101", Symbol: "MNQ", Direction: "short", Stage: "confirmed"}); err != nil {
			t.Fatal(err)
		}
		if err := w.st.PictureHtfTransition("pic-after-now", store.StateFilled, "fixture"); err != nil {
			t.Fatal(err)
		}
		if err := w.st.GormDB().Model(&store.PictureHtfOpportunityDB{}).Where("opp_key = ?", "pic-after-now").
			UpdateColumn("updated_at", after()).Error; err != nil {
			t.Fatal(err)
		}
		w.refusedNotFlattened(t, "picture pic-after-now short filled")
	})
}

// ── FOLD-11 scope pins — dropping the lower bound widened the WINDOW, never the
// OWNER. A fill stamped after the captured now still explains a position only on
// THIS account and instrument: a future-stamped fill of a trader bound to another
// account, or on another instrument, explains nothing and the unexplained
// position keeps today's owner-ruled flatten. Green on both sides of the fix (a
// future stamp was "not fresh" before it); they fail if a later change reads
// "stamped after now" as "owned" ahead of the account/instrument filter.

// flattenedAsBefore drives the production pre-open reconcile and asserts the
// held SHORT is NOT explained: the flatten is sent, and the open then proceeds.
func (w *reconcileWire) flattenedAsBefore(t *testing.T, what string) {
	t.Helper()
	done := make(chan error, 1)
	go func() { done <- w.at.reconcileBeforeOpenNT("MNQ", "long") }()
	if !w.closeSent(3 * time.Second) {
		<-done
		t.Fatalf("%s: a fill stamped after now that is NOT on this account/instrument was taken as the owner — NOT flattened", what)
	}
	w.s.SeedPositionsForTest("Sim101", nil)
	if err := <-done; err != nil {
		t.Fatalf("%s: after the confirmed flatten the open proceeds: %v", what, err)
	}
}

// futureStampedArmedFill writes a filled armed row of traderID whose UpdatedAt
// lies after any now the reconcile can capture.
func (w *reconcileWire) futureStampedArmedFill(t *testing.T, traderID, sid string) {
	t.Helper()
	led := w.st.ArmedOrders()
	r := &store.ArmedOrderDB{TraderID: traderID, PlanID: "p-" + traderID, Scenario: "S1", Version: 1, State: "armed", Side: "short", EntryPx: 29000, StopPx: 29010, TargetPx: 28970}
	if err := led.UpsertArm(r); err != nil {
		t.Fatal(err)
	}
	if err := led.BeginPlacement(r.ID, sid); err != nil {
		t.Fatal(err)
	}
	if err := led.SetState(r.ID, store.StateFilled, "fixture"); err != nil {
		t.Fatal(err)
	}
	if err := led.DB().Model(&store.ArmedOrderDB{}).Where("id = ?", r.ID).
		UpdateColumn("updated_at", time.Now().Add(30*time.Second)).Error; err != nil {
		t.Fatal(err)
	}
}

func TestReconcileFillStampedAfterNowOffThisAccountOrInstrumentExplainsNothing(t *testing.T) {
	for _, c := range []struct{ name, acct, sym string }{{"armed other account", "Sim202", "MNQ"}, {"armed other instrument", "Sim101", "ES"}} {
		t.Run(c.name, func(t *testing.T) {
			w := newReconcileWire(t)
			other := &AutoTrader{id: "reconcile-after-now-" + c.acct + "-" + c.sym, store: w.st, trader: ntTrader.NewTCPTrader(w.s, c.sym, c.acct)}
			other.config.NinjaTraderSymbol = c.sym
			registerPostExitDispatch(other) // running: its scope resolves to (acct, sym)
			t.Cleanup(func() { unregisterPostExitDispatch(other) })
			w.futureStampedArmedFill(t, other.id, "sig-after-now-"+c.acct+"-"+c.sym)
			w.flattenedAsBefore(t, c.name)
		})
	}
	for _, c := range []struct{ name, acct, sym string }{{"picture other account", "Sim202", "MNQ"}, {"picture other instrument", "Sim101", "ES"}} {
		t.Run(c.name, func(t *testing.T) {
			w := newReconcileWire(t)
			key := "pic-after-now-" + c.acct + "-" + c.sym
			if _, _, err := w.st.PictureHtfClaim(&store.PictureHtfOpportunityDB{OppKey: key, TraderID: w.at.id, Account: c.acct, Symbol: c.sym, Direction: "short", Stage: "confirmed"}); err != nil {
				t.Fatal(err)
			}
			if err := w.st.PictureHtfTransition(key, store.StateFilled, "fixture"); err != nil {
				t.Fatal(err)
			}
			if err := w.st.GormDB().Model(&store.PictureHtfOpportunityDB{}).Where("opp_key = ?", key).
				UpdateColumn("updated_at", time.Now().Add(30*time.Second)).Error; err != nil {
				t.Fatal(err)
			}
			w.flattenedAsBefore(t, c.name)
		})
	}
}
