// Unit test for the keystone fill-poll fix: GetOrderStatus must report the real
// NT8 fill (AverageFillPrice) in the canonical shape recordAndConfirmOrder
// expects ("FILLED" + "avgPrice"), and ONLY for the fill that belongs to the
// current entry (signal_id correlation) — so the recorded entry_price becomes
// NT8 truth instead of the stale 5m-mark, and a stale/close-path fill is never
// reported. No NT8/server needed — drives the lastFill/correlation fields.
package ninjatrader

import (
	"testing"

	ntwire "nofx/provider/ninjatrader"
)

func TestGetOrderStatus_SignalIDCorrelation(t *testing.T) {
	tr := &TCPTrader{symbol: "MNQ"}

	// 1) No fill yet → pending.
	if st, _ := tr.GetOrderStatus("MNQ", "x"); st["status"] != "pending" {
		t.Fatalf("no fill: status=%v, want pending", st["status"])
	}

	// 2) Fill arrives for the CURRENT entry signal → FILLED + avgPrice == fill.
	tr.lastEntrySignalID = "sig-A"
	tr.hasFill = true
	tr.lastFill = ntwire.FillPayload{SignalID: "sig-A", FillPrice: 30412.25, Quantity: 1, Side: "short", Status: "filled"}
	st, _ := tr.GetOrderStatus("MNQ", "x")
	if st["status"] != "FILLED" {
		t.Fatalf("matched fill: status=%v, want FILLED", st["status"])
	}
	if av, ok := st["avgPrice"].(float64); !ok || av != 30412.25 {
		t.Fatalf("matched fill: avgPrice=%v, want 30412.25", st["avgPrice"])
	}
	if eq, ok := st["executedQty"].(float64); !ok || eq != 1 {
		t.Fatalf("matched fill: executedQty=%v, want 1", st["executedQty"])
	}

	// 3) Stale fill (signal_id from a PRIOR trade) → pending, never reported.
	tr.lastEntrySignalID = "sig-B" // new entry placed
	// lastFill still carries sig-A (the prior fill) → must not match.
	if st, _ := tr.GetOrderStatus("MNQ", "x"); st["status"] != "pending" {
		t.Fatalf("stale fill: status=%v, want pending", st["status"])
	}

	// 4) Correlation cleared (close path) → pending even though a fill exists.
	tr.lastEntrySignalID = ""
	if st, _ := tr.GetOrderStatus("MNQ", "x"); st["status"] != "pending" {
		t.Fatalf("cleared correlation: status=%v, want pending", st["status"])
	}

	// 5) Rejected entry → REJECTED (poll skips the position record).
	tr.lastEntrySignalID = "sig-C"
	tr.lastFill = ntwire.FillPayload{SignalID: "sig-C", FillPrice: 0, Quantity: 0, Side: "long", Status: "rejected"}
	if st, _ := tr.GetOrderStatus("MNQ", "x"); st["status"] != "REJECTED" {
		t.Fatalf("rejected: status=%v, want REJECTED", st["status"])
	}
}
