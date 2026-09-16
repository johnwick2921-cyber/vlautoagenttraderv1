package trader

import (
	"fmt"
	"time"
)

// ── FLAT IS A CLAIM ABOUT THE BROKER, NOT ABOUT OUR TABLE ────────────────────
//
// Every flatten path asked ONE question — at.store.Position().GetOpenPositions —
// and read len()==0 as "flat". This codebase documents that table as lagging the
// broker in BOTH directions:
//
//	position_desync.go:18  "for up to ~80s after a real exit, the store row is
//	                        still OPEN"
//	position_desync.go:78   the store showing OPEN while the broker reports FLAT
//
// The second direction is merely noisy for a flatten: we try to close something
// already closed. THE FIRST DIRECTION IS THE ONE THAT COSTS MONEY, and it is the
// one nobody was asking about — our table empty while the broker still holds a
// position. The flatten then declares "book flat", returns, and the position
// rides the close or the red-news print with no stop attached to it.
//
// The cutover gate already refuses to accept one number here: leg 1 reads sqlite
// trader_positions, legs 2 and 3 read the broker, and they are quoted separately
// "so a per-account routing fault cannot hide behind one number"
// (class33_cutover_gate.go). The flatten had only the sqlite leg — the same
// question, asked once, at the one moment where being wrong is unrecoverable.
//
// A flatten that trusts the ledger over the broker is the 2026-09-06 naked-stop
// shape arriving from the exit side: our record says flat, the broker says
// otherwise, and the close is the one moment the two must agree.

// flatVerdict is what the two readers together support. UNVERIFIED is a distinct
// answer from FLAT and must never be rendered as one (A24).
type flatVerdict struct {
	StoreOpen    int  // rows our table calls open
	BrokerOpen   int  // positions the broker reports
	BrokerAsked  bool // was there a broker link to ask at all
	BrokerFailed bool // asked and it errored — UNKNOWN, never "none"
}

// ProvenFlat is true only when BOTH readers were available and BOTH said zero.
// One reader agreeing with itself is not corroboration.
func (v flatVerdict) ProvenFlat() bool {
	return v.StoreOpen == 0 && v.BrokerAsked && !v.BrokerFailed && v.BrokerOpen == 0
}

// Disagrees reports the direction that costs money: our table says flat and the
// broker does not.
func (v flatVerdict) Disagrees() bool {
	return v.StoreOpen == 0 && v.BrokerAsked && !v.BrokerFailed && v.BrokerOpen > 0
}

// Why renders both readers on one line, always naming its sources, so a flat
// claim in the journal can be audited without re-deriving where it came from.
func (v flatVerdict) Why() string {
	broker := "broker n/a (no NT8 link)"
	switch {
	case v.BrokerFailed:
		broker = "broker UNREADABLE"
	case v.BrokerAsked:
		broker = fmt.Sprintf("broker %d", v.BrokerOpen)
	}
	return fmt.Sprintf("store %d · %s", v.StoreOpen, broker)
}

// flatTruthAt asks BOTH readers. It never converts a failure into a zero.
//
// The clock is an argument (A28): the caller owns it, and this takes it so a
// test can pin the moment without reaching the wall.
func (at *AutoTrader) flatTruthAt(storeOpen int, now time.Time) flatVerdict {
	// gatePositions, NOT armedTrader().GetPositions(). armedTrader() type-asserts
	// to the concrete *TCPTrader, so a broker read behind it cannot be exercised
	// by any fixture — and safety code no test can reach is how the flatten came
	// to have only one reader in the first place. gatePositions goes through
	// at.trader (any implementation), and it is already panic-guarded: a gate
	// never panics, A10.
	return flatTruthFrom(storeOpen, at.gatePositions)
}

// flatTruthFrom is the pure half: the reader is an argument, so the rule can be
// pinned without a broker.
func flatTruthFrom(storeOpen int, readBroker func() ([]map[string]interface{}, error)) flatVerdict {
	v := flatVerdict{StoreOpen: storeOpen}
	if readBroker == nil {
		return v // nothing to ask — BrokerAsked stays false, which is not "flat"
	}
	v.BrokerAsked = true
	snap, err := readBroker()
	if err != nil {
		// UNKNOWN. Never folded into zero — that conversion is the whole defect
		// this file exists to stop.
		v.BrokerFailed = true
		return v
	}
	v.BrokerOpen = len(snap)
	return v
}
