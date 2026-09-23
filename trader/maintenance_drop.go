package trader

import (
	"fmt"

	ntwire "nofx/provider/ninjatrader"
	"nofx/telemetry"
)

// ── W-ONE-BUTTON M2, M-2 — settling an entry the hold dropped ──────────────
//
// An entry sent while NT8 was disconnected sits in the wire's reconnect queue
// and its caller has already recorded it (SendSignal returned nil). When the
// maintenance hold drops it, those records describe an order NT8 never got.
//
//	never attempted (zero bytes written) → provably never sent:
//	    armed row  place_pending → cancelled
//	    Picture    place_pending → refused
//	    both with "never sent — queued entry dropped by the maintenance hold (job <id>)"
//	    AI path    nothing links its records to the signal (placeEntry returns no
//	               orderId, so recordAndConfirmOrder wrote entry_order_id "<nil>"):
//	               forgotten by the TCPTrader, said out loud, alerted; the
//	               installation gate stays closed on db_open_positions until an
//	               operator reconciles — never a fabricated close
//	attempted (a write was started)       → ambiguous: nothing moves, counted,
//	                                        the gate stays closed
//
// Nothing is resent: a dropped entry has left the queue.
func (at *AutoTrader) onMaintenanceDroppedEntry(d ntwire.DroppedEntry) {
	if d.Attempted {
		telemetry.IncGateBlock(at.id, "maintenance_drop_attempted")
		at.emitAlert("P1", "maintenance-drop-ambiguous", "maintenance-drop-ambiguous:"+d.SignalID,
			fmt.Sprintf("Queued %s %s entry dropped by the update hold AFTER a write started — it may be at NT8", d.Side, d.Symbol),
			"Check NinjaTrader for this order before the update continues; its records stay pending and the installation gate stays closed until it is reconciled.")
		at.logWarnf("🔒 maintenance hold dropped queued entry %s %s %s AFTER a write of it was started — it MAY have reached NT8. Its records stay as they are (place_pending = ambiguous); the installation gate stays closed until it is reconciled against the broker.",
			d.Symbol, d.Side, d.SignalID)
		return
	}
	telemetry.IncGateBlock(at.id, "maintenance_drop")
	job := "n/a"
	if st, ok := maintenanceState(); ok && st.Held && !st.Corrupt {
		job = st.Hold.JobID
	}
	reason := fmt.Sprintf("never sent — queued entry dropped by the maintenance hold (job %s)", job)
	settled := 0
	if at.store != nil {
		if n, err := at.store.ArmedOrders().SettleNeverSent(d.SignalID, reason); err != nil {
			at.logWarnf("🔒 maintenance drop: armed ledger settle failed for %s: %v", d.SignalID, err)
		} else {
			settled += int(n)
		}
		if rows, err := at.store.PictureHtfBySignal(d.SignalID); err == nil {
			for _, r := range rows {
				if r.Stage == "place_pending" {
					if terr := at.store.PictureHtfTransition(r.OppKey, "refused", reason); terr == nil {
						settled++
					}
				}
			}
		}
	}
	if settled > 0 {
		at.logWarnf("🔒 maintenance hold dropped queued entry %s %s %s — it NEVER reached NT8; %d ledger row(s) settled: %s",
			d.Symbol, d.Side, d.SignalID, settled, reason)
		return
	}
	at.logErrorf("🔒 maintenance hold dropped queued entry %s %s %s — it NEVER reached NT8, and no ledger row names it. If the AI path recorded it, trader_positions holds an OPEN row NT8 never had (entry_order_id \"<nil>\"): reconcile it. The installation gate stays closed on db_open_positions until then.",
		d.Symbol, d.Side, d.SignalID)
	at.emitAlert("P1", "maintenance-drop", "maintenance-drop:"+d.SignalID,
		fmt.Sprintf("Queued %s %s entry dropped by the update hold (never sent)", d.Side, d.Symbol),
		"If an OPEN position row exists for it, it is not at NT8 — reconcile before the update continues.")
}
