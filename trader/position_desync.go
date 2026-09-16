package trader

import (
	"os"
	"strings"
	"time"

	"nofx/store"
	"nofx/telemetry"
)

// P0 hotfix (2026-08-19) — POSITION-STATE RECONCILIATION AT THE SKIP GATE.
//
// Every NT8 bracket exit (SL/TP filled broker-side) reaches the store via the
// reconcile "sync" path: 20s cadence + a DELIBERATE 60s flat-grace so
// close-sync gets first crack and the row closes with the REAL exit price
// (trader/ninjatrader/reconcile.go — the reconcile-FIRST race). Net effect:
// for up to ~80s after a real exit, the store row is still OPEN — and
// skip-while-open would trust it, skipping AI cycles on a position that no
// longer exists (the 2026-08-19 09:10 "phantom" scare; that instance turned
// out to be a genuinely-open position, but the window is real).
//
// THE RULE: the gate must never trust a stale local flag against live NT8
// truth. Before skipping, cross-check the broker's live position state; if the
// broker holds NONE of the store's open rows → CRITICAL position_state_desync
// and DO NOT SKIP. The row itself is deliberately NOT closed here — closing
// belongs to close-sync/reconcile, which land the REAL exit price within their
// grace; a gate-side close would recreate the exact lost-P&L race the
// flat-grace exists to prevent. (The dispatch's "clear store" is thus
// delegated to the existing reconciler — ONE closing authority, no twin path.)
//
// Env POSITION_RECONCILE, default ON; "off"/"0"/"false" disables the check.

func positionReconcileEnabled() bool {
	switch strings.ToLower(os.Getenv("POSITION_RECONCILE")) {
	case "off", "0", "false", "disabled":
		return false
	}
	return true
}

// skipGateDesync reports TRUE when the store's open rows are contradicted by
// live broker truth (broker flat) — the caller then refuses to skip. Fail-safe
// posture: any inability to get fresh truth (nil trader, feed down, error)
// returns false → existing skip behavior is kept.
// skipGateDesync is the entry point and owns the clock (A28/class 60); the
// judgement lives in skipGateDesyncAt so tests state their own.
func (at *AutoTrader) skipGateDesync(storeRows []*store.TraderPosition) bool {
	return at.skipGateDesyncAt(storeRows, time.Now())
}

func (at *AutoTrader) skipGateDesyncAt(storeRows []*store.TraderPosition, now time.Time) bool {
	if !positionReconcileEnabled() || at.trader == nil || len(storeRows) == 0 {
		return false
	}
	// No fresh truth while the feed is down — a dark wire must not un-skip.
	if down, _ := at.ninjaFeedDown(); down {
		return false
	}
	positions, err := at.trader.GetPositions()
	if err != nil {
		return false
	}
	held := map[string]bool{}
	for _, p := range positions {
		sym, _ := p["symbol"].(string)
		side, _ := p["side"].(string)
		if sym != "" {
			held[strings.ToUpper(sym)+":"+strings.ToUpper(side)] = true
		}
	}
	for _, row := range storeRows {
		if held[strings.ToUpper(row.Symbol)+":"+strings.ToUpper(row.Side)] {
			return false // at least one row is live broker truth → genuine hold
		}
	}
	r := storeRows[0]
	at.logErrorf("🚨 position_state_desync: store shows %d OPEN row(s) (%s %s) but the broker reports FLAT — NOT skipping this cycle. The reconciler closes the row with the REAL exit within its grace (≤80s); the AI decides on live flat state.",
		len(storeRows), r.Symbol, r.Side)
	telemetry.RecordError(at.id, "position_state_desync",
		"store open vs broker flat — skip gate deferred to NT8 truth", telemetry.CostNone)
	// CLASS-27 FIX 1 (2026-08-31): the instant the gate sees store-OPEN vs
	// broker-FLAT, kill the orphaned brackets NOW — do NOT wait for the 60s
	// reconcile grace. A netting close leaves the closed arm's SL/TP resting
	// at NT8 (live proof: the S1 stop fired 26 minutes after its long was
	// netted and opened a naked short). cancel_order is idempotent and only
	// targets the bracket legs for that signal.
	if nt := at.armedTrader(); nt != nil {
		for _, row := range storeRows {
			if row.EntryOrderID == "" {
				continue
			}
			// D5 (cancel-confirmation 2026-09-06): the branch is on the
			// FAILURE, never on the success. This site only logs — it writes
			// no ledger state — but a `== nil` block after a cancel send is
			// the shape the defect grows in, so the rule is enforced on the
			// shape rather than on today's contents. The wording was already
			// honest and is unchanged: SENT, not cancelled.
			// D4 (2026-09-07) — GUARDED, BUT WITH THE BROKER'S POSITION TRUTH.
			//
			// This sender was the second one the 09-06 wave missed. It cannot
			// carry the plain guard: children-with-no-entry is exactly the shape
			// it exists to sweep, and the plain guard refuses that shape to
			// protect a live position. Here there IS no live position — every
			// row reaching this loop was checked against the broker's own
			// positions above and found FLAT (the `held[...]` early return).
			// So the guard is given that fact and allows the sweep, while still
			// refusing if the book ever shows an entry still resting.
			if !at.cancelSignalIfSafeWith(nt.CancelOrder, row.EntryOrderID,
				"class-27 desync", now, positionContext{Known: true, Open: false}) {
				continue
			}
			at.logWarnf("🧹 class-27 desync: cancel_order sent for orphan bracket %s (%s %s row=%d) — immediate, no grace; SENT is not CONFIRMED",
				row.EntryOrderID, row.Symbol, row.Side, row.ID)
		}
	}
	return true
}
