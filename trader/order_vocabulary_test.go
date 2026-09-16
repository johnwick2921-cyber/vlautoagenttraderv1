package trader

import (
	"testing"

	nt "nofx/provider/ninjatrader"
	"nofx/store"
)

// TestLocallyHeldStopIsNotAnAcceptedRisk — C3(c), 2026-09-07.
//
// TriggerPending appeared NOWHERE in this tree (0 occurrences across every .go
// and .cs file). NinjaTrader holds such an order on the local PC — it is not at
// the exchange and will not fire if this machine is down. Recording its price
// as "the stop the broker accepted" writes a protection that does not exist
// where it matters. Same defect as accepted_risk ids 9/10, different state.
func TestLocallyHeldStopIsNotAnAcceptedRisk(t *testing.T) {
	const sig = "aa07e583"
	row := &store.AcceptedRisk{SignalID: sig}
	applyBrokerTerms(row, []nt.NT8Order{
		{Name: sig + "-sl", State: "TriggerPending", StopPrice: 29554},
		{Name: sig + "-tp", State: "Working", LimitPrice: 29623},
	}, sig)

	if row.AcceptedStopPx != nil {
		t.Fatalf("a TriggerPending stop was recorded as accepted at %.2f — NT8 is holding it "+
			"on this PC, not at the exchange; the row claims a protection the exchange has never seen",
			*row.AcceptedStopPx)
	}
	if row.AcceptedTargetPx == nil || *row.AcceptedTargetPx != 29623 {
		t.Fatal("the Working target should still be recorded — the guard must be narrow")
	}
}

// TestUnreadableStateIsNotAnAcceptedRisk — the owner's UNKNOWN ruling applied
// to this writer: a state we could not parse is not evidence of anything.
func TestUnreadableStateIsNotAnAcceptedRisk(t *testing.T) {
	const sig = "b1c2d3e4"
	row := &store.AcceptedRisk{SignalID: sig}
	applyBrokerTerms(row, []nt.NT8Order{
		{Name: sig + "-sl", State: "SomethingNT8NeverSends", StopPrice: 29554},
	}, sig)
	if row.AcceptedStopPx != nil {
		t.Fatalf("a stop in an UNREADABLE state was recorded as accepted at %.2f", *row.AcceptedStopPx)
	}
}
