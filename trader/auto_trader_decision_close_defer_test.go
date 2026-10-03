package trader

import "testing"

// TestDecisionCloseOnNinjaTraderDefersToTheFillFrame is the plan v10 CR-B KEEP
// pin (R4-104-P1-2 / R4-108-P2-1): the NT close-deferral block in
// recordAndConfirmOrderAs. A close_long/close_short DECISION on ninjatrader
// writes NO order row off the mark price — the CLOSED transition happens only
// from the position_close fill frame (trader/ninjatrader close_sync.recordClose,
// the fill-side counterpart). Deleting the deferral block makes this test RED:
// the decision path would record an order row for a close that the fill frame
// also records, recreating the id=45→id=46 net-2 shape.
func TestDecisionCloseOnNinjaTraderDefersToTheFillFrame(t *testing.T) {
	for _, action := range []string{"close_long", "close_short"} {
		at := plannerTestTrader(t)
		at.id = "nt-close-defer-" + action
		at.trader = &stubTrader{} // the poll must survive the deleted-deferral mutant, so the row-count assertion is what fires (CTO P3)
		before, err := at.store.Order().GetTraderOrders(at.id, 100)
		if err != nil {
			t.Fatalf("%s: orders before: %v", action, err)
		}
		orderResult := map[string]interface{}{
			"orderId":  "defer-pin-close",
			"symbol":   "MNQ",
			"price":    20000.0,
			"quantity": 1.0,
			"leverage": 1,
		}
		at.recordAndConfirmOrder(orderResult, "MNQ", action, 1.0, 20000.0, 1, 19990.0, 90)
		after, err := at.store.Order().GetTraderOrders(at.id, 100)
		if err != nil {
			t.Fatalf("%s: orders after: %v", action, err)
		}
		if len(after) != len(before) {
			t.Fatalf("%s: a ninjatrader close decision must defer to the position_close fill frame (no order row off the mark); rows %d -> %d",
				action, len(before), len(after))
		}
	}
}
