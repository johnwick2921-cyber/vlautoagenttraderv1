package ninjatrader

import (
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
)

// TestCloseConfirmedSince_FramePath locks the reconcile-before-open FRAME path
// (PART 1a): a fill-confirmed close (position_close → MarkCloseConfirmed, as
// close-sync calls it) is visible to CloseConfirmedSince ~instantly, keyed by
// (symbol, side) for THIS trader's bound account — independent of the positions
// snapshot (which for a non-active account only refreshes on the 30s heartbeat).
// This is what lets a non-active-account flatten confirm in <1s instead of
// starving against the 6s window.
func TestCloseConfirmedSince_FramePath(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	// 15m on the NON-active account (the starvation case).
	tr := NewTCPTrader(s, "MNQ", "SimAccount1")

	t0 := time.Now().UnixMilli()

	// No close yet → not confirmed.
	if tr.CloseConfirmedSince("MNQ", "long", t0) {
		t.Fatal("no close yet — CloseConfirmedSince must be false")
	}

	// Simulate the position_close frame arriving (what close-sync does on a fill).
	tr.MarkCloseConfirmed("MNQ", "long")

	// Confirmed for our flatten window (>= t0), even though NO positions snapshot
	// was streamed (non-active account) — the frame alone confirms.
	if !tr.CloseConfirmedSince("MNQ", "long", t0) {
		t.Fatal("close was confirmed after t0 — CloseConfirmedSince must be true (frame path)")
	}

	// A future window (after the mark) is NOT confirmed — never accept a stale close.
	future := time.Now().UnixMilli() + 10_000
	if tr.CloseConfirmedSince("MNQ", "long", future) {
		t.Fatal("mark predates the future window — must be false (no stale acceptance)")
	}

	// Wrong side is not confirmed (a short close must not clear a long orphan).
	if tr.CloseConfirmedSince("MNQ", "short", t0) {
		t.Fatal("wrong side must not be confirmed")
	}

	// Case-insensitive symbol/side keying (ntHeldPosition returns lowercase side;
	// the close frame's PositionSide may be either case).
	tr.MarkCloseConfirmed("MNQ", "SHORT")
	if !tr.CloseConfirmedSince("mnq", "short", t0) {
		t.Fatal("keying must be case-insensitive on symbol and side")
	}
}

// TestCloseConfirmed_PerBoundAccountIsolated: two same-symbol traders on different
// accounts track their own close confirmations independently (no cross-talk) — a
// close on one account never confirms the other's flatten.
func TestCloseConfirmed_PerBoundAccountIsolated(t *testing.T) {
	s := ntwire.NewTCPServer(nil)
	hoang := NewTCPTrader(s, "MNQ", "Sim101")
	m15 := NewTCPTrader(s, "MNQ", "SimAccount1")
	t0 := time.Now().UnixMilli()

	// Only 15m's account saw a close.
	m15.MarkCloseConfirmed("MNQ", "long")

	if !m15.CloseConfirmedSince("MNQ", "long", t0) {
		t.Fatal("15m's own close must confirm for 15m")
	}
	if hoang.CloseConfirmedSince("MNQ", "long", t0) {
		t.Fatal("15m's close must NOT confirm hoang's flatten (per-trader state)")
	}
}
