package trader

import (
	"testing"
	"time"

	"nofx/store"
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
