package trader

import (
	"fmt"
	"strings"
	"sync"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	ntTrader "nofx/trader/ninjatrader"
)

// PICTURE-HTF BROKER STATE (2026-09-20) — the live consumer of RECEIVED
// broker events and the restart-safe reconciliation sweep. Both write only
// through PictureHtfMarkBrokerState / PictureHtfAppendBrokerStatus — stages
// move FORWARD, fills are never downgraded, and an outcome for which NO
// received evidence exists stays UNKNOWN (place_pending, never resent —
// and never fabricated).
//
// RECEIVED-EVIDENCE RULE (CTO findings 1+2, 2026-09-20). The AddOn's
// order_snapshot frame EXCLUDES terminal orders (Filled/Cancelled/Rejected/
// Expired — VLTraderTCPClient.cs SendOrderSnapshot), so the snapshot can
// prove an order is WORKING but can never prove a terminal outcome, and it
// carries NO fill price at all (limit_price is the order's limit, not an
// execution). Terminal outcomes may only be recovered from received
// execution/order history: order_update frames (fill_price = AverageFillPrice
// — actual) and fill frames. A terminal outcome that was never received is
// UNKNOWN, with the fill price left unknown — never LimitPrice.

// pictureEntryStage maps an order_update wire state onto the ledger stage
// vocabulary. accepted/partfilled carry the row forward like working; filled
// passes through as the terminal FILL; any OTHER terminal wire state (per the
// canonical classifier — no re-typed lists) is a terminal refusal.
func pictureEntryStage(u string) string {
	state := strings.ToLower(u)
	switch state {
	case "accepted", "partfilled":
		return store.StateWorking
	case store.StateFilled:
		return store.StateFilled
	}
	if ntwire.ClassifyOrderState(state) == ntwire.LivenessTerminal {
		return store.StateRejected
	}
	return state
}

// pictureStagePriority orders ledger stages so a late/duplicate event can
// never downgrade a row: terminal = 2, working = 1, everything else 0. The
// canonical predicate store.IsTerminalArmState is the ONLY terminal source
// (no re-typed state lists — arm_state_source_guard).
func pictureStagePriority(stage string) int {
	if store.IsTerminalArmState(stage) {
		return 2
	}
	if stage == store.StateWorking {
		return 1
	}
	return 0
}

// pictureLeg returns ("sl"|"tp", true) for a protective-order frame; the C#
// strips the suffix from signal_id but keeps it in order_name.
func pictureLeg(orderName, signalID string) (string, bool) {
	if strings.HasSuffix(orderName, "-sl") && strings.TrimSuffix(orderName, "-sl") == signalID {
		return "sl", true
	}
	if strings.HasSuffix(orderName, "-tp") && strings.TrimSuffix(orderName, "-tp") == signalID {
		return "tp", true
	}
	return "", false
}

// pictureHtfFillRR is the actual-fill R:R from the row's own geometry.
func pictureHtfFillRR(row store.PictureHtfOpportunityDB, fill float64) float64 {
	if fill <= 0 || row.StopPx <= 0 || row.TargetPx <= 0 {
		return 0
	}
	risk := fill - row.StopPx
	reward := row.TargetPx - fill
	if row.Direction == "short" {
		risk = row.StopPx - fill
		reward = fill - row.TargetPx
	}
	if risk <= 0 {
		return 0
	}
	return reward / risk
}

// ── Received broker-history (process lifetime) ───────────────────────────────
//
// Reconciliation may only recover what was RECEIVED. The history is populated
// by the live consumer for every entry-leg order_update frame — even when no
// ledger row matches yet (a frame that raced ahead of the row's write is
// exactly what the sweep exists to catch). After a process restart the
// history is empty, and the sweep says UNKNOWN rather than inventing an
// outcome. Only entry-leg frames qualify: protective legs share the signal id
// but their order_name carries the suffix, and a leg's fill must never read
// as the entry's fill.

type pictureBrokerRecord struct {
	State     string
	FillPrice float64 // wire fill_price = AverageFillPrice — ACTUAL fill evidence
	Quantity  float64 // wire quantity = Filled count
	Reason    string
}

type pictureBrokerHistory struct {
	mu       sync.Mutex
	bySignal map[string]pictureBrokerRecord
}

// pictureBrokerHistories is keyed by trader id — a restarted trader replaces
// its entry.
var pictureBrokerHistories sync.Map

func pictureHistoryFor(at *AutoTrader) *pictureBrokerHistory {
	if at == nil || at.id == "" {
		return nil
	}
	v, _ := pictureBrokerHistories.LoadOrStore(at.id, &pictureBrokerHistory{bySignal: map[string]pictureBrokerRecord{}})
	h, _ := v.(*pictureBrokerHistory)
	return h
}

func pictureRecordReceived(at *AutoTrader, u ntwire.OrderUpdatePayload) {
	if u.SignalID == "" || u.OrderName != u.SignalID {
		return // entry-leg frames only
	}
	h := pictureHistoryFor(at)
	if h == nil {
		return
	}
	stage := pictureEntryStage(u.State)
	h.mu.Lock()
	defer h.mu.Unlock()
	if prev, ok := h.bySignal[u.SignalID]; ok {
		if pictureStagePriority(pictureEntryStage(prev.State)) > pictureStagePriority(stage) {
			return // never downgrade the history
		}
		if pictureStagePriority(pictureEntryStage(prev.State)) == pictureStagePriority(stage) && prev.FillPrice > 0 {
			return // keep the record that carries actual fill evidence
		}
	}
	h.bySignal[u.SignalID] = pictureBrokerRecord{State: u.State, FillPrice: u.FillPrice, Quantity: float64(u.Quantity), Reason: u.Reason}
}

func pictureReceivedRecord(at *AutoTrader, signalID string) (pictureBrokerRecord, bool) {
	h := pictureHistoryFor(at)
	if h == nil {
		return pictureBrokerRecord{}, false
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	rec, ok := h.bySignal[signalID]
	return rec, ok
}

// pictureHtfBroker returns the concrete NT8 trader when there is one.
func pictureHtfBroker(at *AutoTrader) *ntTrader.TCPTrader {
	if at == nil || at.trader == nil {
		return nil
	}
	tcp, _ := at.trader.(*ntTrader.TCPTrader)
	return tcp
}

// pictureHtfConsumeOrderUpdate routes ONE received order_update frame into
// the opportunity it names, AND records it into the received-history that the
// reconciliation sweep recovers from (even when no row matches yet).
// Entry-leg events move the row forward (working → partfilled → filled;
// rejected/cancelled/expired are terminal) and stamp fill price/qty — the
// wire's fill_price, which is AverageFillPrice — plus the actual-fill R:R.
// Protective-leg events (-sl/-tp) APPEND to the protection status WITHOUT
// touching the entry's stage or fill — a filled SL leg is the proof the
// protective order worked, recorded as protection_sl_filled.
func pictureHtfConsumeOrderUpdate(at *AutoTrader, u ntwire.OrderUpdatePayload) {
	if at == nil || at.store == nil || u.SignalID == "" {
		return
	}
	if _, isLeg := pictureLeg(u.OrderName, u.SignalID); !isLeg {
		pictureRecordReceived(at, u)
	}
	rows, err := at.store.PictureHtfBySignal(u.SignalID)
	if err != nil || len(rows) == 0 {
		return
	}
	for i := range rows {
		row := rows[i]
		if leg, isLeg := pictureLeg(u.OrderName, u.SignalID); isLeg {
			note := fmt.Sprintf("protection_%s_%s", leg, strings.ToLower(u.State))
			if u.State == "rejected" && u.Reason != "" {
				note += ": " + u.Reason
			}
			_ = at.store.PictureHtfAppendBrokerStatus(row.OppKey, note)
			continue
		}
		// Entry leg: only move FORWARD, never downgrade.
		stage := pictureEntryStage(u.State)
		pri := pictureStagePriority(stage)
		if pri <= pictureStagePriority(row.Stage) && row.Stage != store.StatePlacePending {
			continue
		}
		_ = at.store.PictureHtfMarkBrokerState(row.OppKey, stage, "", u.State, u.Reason, u.FillPrice, float64(u.Quantity), pictureHtfFillRR(row, u.FillPrice))
	}
}

// pictureHtfApplyTerminal moves a place_pending row to a RECEIVED terminal
// outcome. The fill price comes ONLY from received execution evidence (the
// order_update frame's fill_price, or the fill frame ring); if neither has a
// price the row is terminal with its fill price left UNKNOWN.
func pictureHtfApplyTerminal(at *AutoTrader, row store.PictureHtfOpportunityDB, state string, fillPrice, fillQty float64, reason string) {
	stage := store.StateRejected
	if strings.EqualFold(state, "filled") {
		stage = store.StateFilled
	}
	rr := 0.0
	if stage == store.StateFilled && fillPrice > 0 {
		rr = pictureHtfFillRR(row, fillPrice)
	}
	status := state
	if stage == store.StateFilled && fillPrice <= 0 {
		status = "filled (fill price unknown — no received execution price)"
	}
	_ = at.store.PictureHtfMarkBrokerState(row.OppKey, stage, "", status, reason, fillPrice, fillQty, rr)
}

// pictureHtfReconcilePending recovers unresolved rows — place_pending AND
// working — across disconnects and restarts WITHOUT another entry, from
// RECEIVED evidence only.
//
// PRIORITY: received EXECUTION evidence outranks working receipts. A working
// order_update recorded earlier must never mask a newer received fill: a fill
// frame can arrive with no matching filled order_update, and a row already
// marked working is still an open outcome.
//
//  1. The received order_update history — terminal fill (wire fill price, or
//     the fill ring when the frame carried none) / terminal refusal settle
//     the row; a working/accepted receipt moves the row forward and the sweep
//     CONTINUES to the fill ring instead of exiting.
//  2. The received fill ring — actual fill price/quantity for the signal,
//     checked for place_pending AND working rows (finding 4: this runs even
//     after a working receipt was recorded).
//  3. The broker's WORKING-order book — presence proves the order rests at
//     the broker (working receipt); the snapshot carries no fill price, so a
//     part-filled entry records its filled quantity with the price UNKNOWN.
//     ABSENCE proves nothing (the AddOn excludes terminal orders from the
//     snapshot, so a filled order and a never-placed order look identical).
//  4. Nothing received at all → place_pending rows stay place_pending with a
//     one-time "outcome unknown" marker; working rows stay working (their
//     receipt is the row itself). The ambiguous case is never blindly resent
//     (the atomic claim already blocks re-entry; this sweep only observes).
//
// RESTART LIMITATION (explicit): the received history and the fill ring are
// process-local and disappear on restart. A terminal outcome the machine
// never received stays UNKNOWN after a restart — this sweep is duplicate
// PREVENTION, not complete recovery of broker history.
func pictureHtfReconcilePending(at *AutoTrader) {
	if at == nil || at.store == nil {
		return
	}
	rows, err := at.store.PictureHtfRecoverableByTrader(at.id)
	if err != nil || len(rows) == 0 {
		return
	}
	for _, row := range rows {
		if row.SignalID == "" {
			continue
		}
		haveReceipt := false
		// 1) RECEIVED order_update evidence (the wire's actual events).
		if rec, ok := pictureReceivedRecord(at, row.SignalID); ok {
			haveReceipt = true
			stage := pictureEntryStage(rec.State)
			if stage == store.StateRejected {
				pictureHtfApplyTerminal(at, row, rec.State, 0, 0, rec.Reason)
				continue
			}
			if stage == store.StateFilled {
				px, qty := rec.FillPrice, rec.Quantity
				if px <= 0 {
					if rpx, rqty, okf := pictureRecentFillFor(at, row.SignalID); okf {
						px, qty = rpx, rqty
					}
				}
				pictureHtfApplyTerminal(at, row, rec.State, px, qty, rec.Reason)
				continue
			}
			// working/accepted receipt — forward progress, NOT terminal:
			// record it and FALL THROUGH so newer fill evidence still wins.
			_ = at.store.PictureHtfMarkBrokerState(row.OppKey, store.StateWorking, "", rec.State, "", 0, 0, 0)
		}
		// 2) RECEIVED fill frames (actual execution prices on the fill
		// stream). Checked for place_pending AND working rows — an older
		// working receipt must never mask this evidence.
		if px, qty, ok := pictureRecentFillFor(at, row.SignalID); ok {
			pictureHtfApplyTerminal(at, row, "filled", px, qty, "reconciled from received fill")
			continue
		}
		// 3) The broker's WORKING-order book. Terminal orders are excluded
		// from the wire snapshot, so only a PRESENT working order is evidence;
		// absence here proves nothing and falls through.
		if tcp := pictureHtfBroker(at); tcp != nil {
			if ord, ok := tcp.OrderSnapshotLookup(row.SignalID); ok && ord.IsWorking() {
				haveReceipt = true
				note := "working on broker book"
				if ord.Filled > 0 {
					note = fmt.Sprintf("working on broker book (filled qty %d; fill price unknown — the snapshot carries no fill price)", ord.Filled)
				}
				_ = at.store.PictureHtfMarkBrokerState(row.OppKey, store.StateWorking, ord.OrderID, note, "", 0, 0, 0)
				continue
			}
		}
		// 4) Nothing received at all. Only an UNEVIDENCED place_pending row
		// gets the unknown marker; a working row already carries its receipt
		// (the row itself) and stays working.
		if !haveReceipt && row.Stage == store.StatePlacePending {
			_ = at.store.PictureHtfAppendBrokerStatus(row.OppKey, "reconcile:no_received_evidence (outcome unknown — will not resend)")
		}
	}
}

// pictureRecentFillFor reads the trader's received-fill ring (real execution
// evidence) for a signal.
func pictureRecentFillFor(at *AutoTrader, signalID string) (price, quantity float64, ok bool) {
	tcp := pictureHtfBroker(at)
	if tcp == nil {
		return 0, 0, false
	}
	return tcp.RecentFillFor(signalID)
}

// pictureHtfBrokerConsumers guards the per-trader consumer goroutine
// (keyed by trader id — a restarted trader replaces its entry).
var pictureHtfBrokerConsumers sync.Map

// ensurePictureHtfBrokerConsumer starts the live order_update consumer for
// the concrete NT8 trader (once per trader; cheap LoadOrStore hit otherwise).
// The consumer is a coordinated fan-out LISTENER — it can never evict the
// armed executor's subscription and the armed executor can never evict it.
// Frames with no matching row are still recorded into the received-history
// (the reconciliation sweep's recovery source). If the underlying
// subscription dies, the consumer deletes its registry entry and the next
// evaluator build re-listens.
func (at *AutoTrader) ensurePictureHtfBrokerConsumer() {
	if at == nil || at.id == "" {
		return
	}
	if _, loaded := pictureHtfBrokerConsumers.LoadOrStore(at.id, struct{}{}); loaded {
		return
	}
	tcp := pictureHtfBroker(at)
	if tcp == nil {
		pictureHtfBrokerConsumers.Delete(at.id)
		return
	}
	go func() {
		for {
			ch, _ := tcp.OrderUpdatesListen()
			for u := range ch {
				pictureHtfConsumeOrderUpdate(at, u)
			}
			// The direct subscription died (server teardown / a direct
			// re-subscribe). Self-heal on the next evaluator build.
			pictureHtfBrokerConsumers.Delete(at.id)
			return
		}
	}()
}
