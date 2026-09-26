package ninjatrader

import (
	"strings"
	"testing"
)

// ── W1b FOLD-13 — tagLateEntryFill settles FIRST and tags LAST ──────────────
//
// tagLateEntryFill wrote the position's entry_order_id FIRST and settled the
// signal-keyed AI order row after. A failed UpdateOrderStatus then left the
// position tagged and the order NEW forever: the untracked branch never
// re-enters a tracked row, so nothing ever settled it, and the NEW row claimed
// an unresolved open that was in fact this position's entry.
//
// The rule (CTO part 2): settle first — the FILLED status AND its trader_fills
// row — and tag last. On a settle failure: WARN, and leave BOTH untouched (the
// order NEW with no fill row, the position untagged) so the unresolved-order
// sweep keeps the NEW row visible. The settle is one unit: a fill-row write
// that fails after the status moved would otherwise leave FILLED-with-no-fill
// (the half-record TestLateAIFillWritesTheFillRowTheNormalPathWrites forbids).
//
// Every case runs through the production call site: reconcilePositions'
// untracked branch → tagLateEntryFill. Store failures are REAL SQLite failures
// (a trigger that aborts the statement), never a seam.

// failOrderSettle makes the order row's FILLED status write fail from here on.
func failOrderSettle(t *testing.T, w *lateFillWire) {
	t.Helper()
	if err := w.st.GormDB().Exec("CREATE TRIGGER injected_order_settle_fail BEFORE UPDATE OF status ON trader_orders BEGIN SELECT RAISE(ABORT, 'injected order settle failure'); END").Error; err != nil {
		t.Fatal(err)
	}
}

// failFillRow makes every trader_fills INSERT fail from here on.
func failFillRow(t *testing.T, w *lateFillWire) {
	t.Helper()
	if err := w.st.GormDB().Exec("CREATE TRIGGER injected_fill_row_fail BEFORE INSERT ON trader_fills BEGIN SELECT RAISE(ABORT, 'injected fill row failure'); END").Error; err != nil {
		t.Fatal(err)
	}
}

func TestLateAIFillSettleFailureLeavesOrderNewAndPositionUntagged(t *testing.T) {
	for _, tc := range []struct {
		name   string
		inject func(t *testing.T, w *lateFillWire)
	}{
		{name: "order status write fails", inject: failOrderSettle},
		{name: "fill row write fails", inject: failFillRow},
	} {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			w := newLateFillWire(t)
			warns := captureLateWarns(t)
			const sid = "sig-ai-late"
			w.aiOrderRow(t, lateFillTrader, sid, "open_long", "LONG", "BUY")
			w.fill(t, sid, "long", 29001)
			tc.inject(t, w)
			row := w.materialize(t, 29001)

			if row.EntryOrderID != "" {
				t.Fatalf("a failed settle must leave the position UNTAGGED (tag last), entry_order_id=%q", row.EntryOrderID)
			}
			o, err := w.st.Order().GetOrderByExchangeID(lateFillExchange, sid)
			if err != nil || o == nil {
				t.Fatalf("order row: %+v %v", o, err)
			}
			if o.Status != "NEW" || o.AvgFillPrice != 0 || o.FilledQuantity != 0 {
				t.Fatalf("a failed settle must leave the order row untouched (NEW, unfilled) so the unresolved-order sweep still sees it: status=%q avg=%.2f qty=%.0f", o.Status, o.AvgFillPrice, o.FilledQuantity)
			}
			fills, err := w.st.Order().GetOrderFills(o.ID)
			if err != nil {
				t.Fatal(err)
			}
			if len(fills) != 0 {
				t.Fatalf("a failed settle must write no fill row, got %d", len(fills))
			}
			if got := w.tr.resolveEntrySignalID("MNQ", "long"); got != "" {
				t.Fatalf("an untagged position must not be addressable by move_stop/trailing, resolved %q", got)
			}
			if !warns.has(sid) || !warns.has("settle failed") {
				t.Fatalf("the settle failure must be a WARN naming the signal %s and the failed settle", sid)
			}
			line := materializedLine(t, warns)
			for _, sub := range []string{"UNTAGGED", sid, "settle failed", "left NEW"} {
				if !strings.Contains(line, sub) {
					t.Fatalf("the 🧩 line must say %q:\n%s", sub, line)
				}
			}
			for _, sub := range []string{"manual/NT8-side", "late AI fill ("} {
				if strings.Contains(line, sub) {
					t.Fatalf("the 🧩 line claims %q for an attribution that was not established:\n%s", sub, line)
				}
			}
		})
	}
}

// The ORDER of the two writes, pinned from the other side: when the tag is the
// write that fails, the settle has already landed — the order reads FILLED with
// its one fill row (NT8 did fill it; that is broker truth), and only the
// position stays untagged (WARN, 🧩 says why).
func TestLateAIFillSettlesBeforeItTags(t *testing.T) {
	w := newLateFillWire(t)
	warns := captureLateWarns(t)
	const sid = "sig-ai-late"
	w.aiOrderRow(t, lateFillTrader, sid, "open_long", "LONG", "BUY")
	w.fill(t, sid, "long", 29001)
	failPositionUpdates(t, w)
	row := w.materialize(t, 29001)
	if row.EntryOrderID != "" {
		t.Fatalf("fixture: the tag write was made to fail, entry_order_id=%q", row.EntryOrderID)
	}
	o, err := w.st.Order().GetOrderByExchangeID(lateFillExchange, sid)
	if err != nil || o == nil {
		t.Fatalf("order row: %+v %v", o, err)
	}
	if o.Status != "FILLED" || o.AvgFillPrice != 29001 || o.FilledQuantity != 1 {
		t.Fatalf("settle runs BEFORE the tag: the order must already read FILLED at the ring's price when the tag fails: status=%q avg=%.2f qty=%.0f", o.Status, o.AvgFillPrice, o.FilledQuantity)
	}
	fills, err := w.st.Order().GetOrderFills(o.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(fills) != 1 {
		t.Fatalf("the settle writes exactly one fill row before the tag, got %d", len(fills))
	}
	line := materializedLine(t, warns)
	for _, sub := range []string{"UNTAGGED", sid, "settled FILLED", "entry-order-id stamp failed"} {
		if !strings.Contains(line, sub) {
			t.Fatalf("the 🧩 line must say %q:\n%s", sub, line)
		}
	}
}
