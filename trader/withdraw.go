package trader

import (
	"strings"
	"time"

	"nofx/kernel"
	"nofx/store"
)

// ── W-EXEC-TRUTH W0 (f) — WITHDRAW RESTING ENTRIES, ENTRIES ONLY ────────────
//
// A hold used to REFUSE new entries only: a resting armed limit or stop kept
// resting and could FILL during an update. And a loss-limit trip (the
// consecutive-loss breaker, the daily force-flat) refused new entries too,
// while an entry already resting at NT8 could still fill after the halt
// (defect 7 — money safety, CTO Q14).
//
// withdrawEntriesIfDue runs from monitorTick — the per-trader wall-clock beat
// that keeps running when runCycle does not (CME closed, EOD flat, a stopped
// scan) — whenever something asks for a withdraw:
//
//	maintenance  a hold written with --withdraw-entries (never implied)
//	breaker      the consecutive-loss halt has tripped
//	force-flat   the daily force-flat has tripped
//
// It cancels ENTRIES ONLY: every armed row that carries a broker signal and
// is not terminal, through the same filled-arm guard (cancelSafetyFor) and the
// AddOn's entry-only cancel_order; the row goes to cancel_pending. A pending
// cancel is NEVER shown as complete: it becomes 'cancelled' only on received
// evidence — an order_update, or the order's absence from a fresh PERSISTED
// snapshot (confirmPendingCancels). Protection, reconciliation and exits are
// never touched. An arm with no signal was never sent: it stays armed and the
// admission gate refuses it.

// WithdrawReasonPrefix marks a withdraw in the ledger's state_reason.
const WithdrawReasonPrefix = "withdraw: "

// withdrawDueReason says why resting entries must be withdrawn now ("" = no).
func (at *AutoTrader) withdrawDueReason(now time.Time) string {
	if st, ok := maintenanceState(); ok && st.Held && !st.Corrupt && st.Hold.WithdrawEntries {
		return "maintenance job " + st.Hold.JobID
	}
	if r := kernel.DailyForceFlatReason(at.id); r != "" {
		return "daily_force_flat"
	}
	if _, halted := at.consecutiveLossHaltedAt(now); halted {
		return "consecutive_loss"
	}
	return ""
}

// withdrawEntriesIfDue is the monitorTick hook.
func (at *AutoTrader) withdrawEntriesIfDue(now time.Time) {
	if at == nil || at.store == nil {
		return
	}
	why := at.withdrawDueReason(now)
	if why == "" {
		return
	}
	ledger := at.store.ArmedOrders()
	if n := at.withdrawRestingEntries(ledger, why, now); n > 0 {
		at.logWarnf("🧹 withdraw (%s): %d resting entry order(s) sent a cancel — each stays cancel_pending until the broker confirms", why, n)
	}
	var resend func(string) error
	if nt := at.armedTrader(); nt != nil {
		resend = func(sid string) error {
			at.cancelSignalIfSafe(nt.CancelOrder, sid, "withdraw re-request", now)
			return nil
		}
	}
	at.confirmPendingCancels(ledger, resend, now)
}

// withdrawRestingEntries sends the entry-only cancel for every resting entry
// of this trader and moves it to cancel_pending. Returns how many it asked.
func (at *AutoTrader) withdrawRestingEntries(ledger *store.ArmedOrderStore, why string, now time.Time) int {
	rows, err := ledger.ListNonTerminal(at.id)
	if err != nil {
		at.logWarnf("🧹 withdraw (%s): ledger read failed — nothing withdrawn this beat: %v", why, err)
		return 0
	}
	nt := at.armedTrader()
	n := 0
	for _, r := range rows {
		if r.TraderID != at.id || strings.TrimSpace(r.SignalID) == "" ||
			store.IsTerminalArmState(r.State) || r.State == store.StateCancelPending {
			continue
		}
		if nt != nil {
			// The filled-arm guard: a row the ledger still calls working may
			// have filled — cancelling it by signal would reach its bracket.
			if v := at.cancelSafetyFor(r, now); !v.Allow {
				if at.admitLast.changed("withdraw|"+r.SignalID, v.Why) {
					at.logWarnf("🛟 withdraw cancel REFUSED %s signal=%s — %s", r.Scenario, shortID(r.SignalID), v.Why)
				}
				continue
			}
			if cerr := nt.CancelOrder(r.SignalID); cerr != nil {
				at.logWarnf("✕ withdraw cancel SEND failed %s signal=%s: %v", r.Scenario, shortID(r.SignalID), cerr)
			}
		}
		// Recorded whether or not the send returned nil: the row is at the
		// broker until evidence says otherwise (class 81).
		if err := ledger.RequestCancel(r.ID, WithdrawReasonPrefix+why, now.UnixMilli()); err != nil {
			at.logWarnf("✕ withdraw: ledger write failed for %s: %v", r.Scenario, err)
			continue
		}
		n++
	}
	return n
}

// WithdrawView is the withdraw half of GET /api/maintenance: the rows the
// current job's withdraw asked for, pending until the broker confirms.
type WithdrawView struct {
	Requested bool    `json:"requested"`
	Pending   []int64 `json:"pending"`
	Confirmed []int64 `json:"confirmed"`
}

// maintenanceWithdrawView reads the withdraw for the current hold's job; nil
// when no hold asks for one.
func maintenanceWithdrawView(st *store.Store) *WithdrawView {
	ms, ok := maintenanceState()
	if !ok || !ms.Held || ms.Corrupt || !ms.Hold.WithdrawEntries {
		return nil
	}
	v := &WithdrawView{Requested: true, Pending: []int64{}, Confirmed: []int64{}}
	if st == nil {
		return v
	}
	rows, err := st.ArmedOrders().ListByReasonPrefix(WithdrawReasonPrefix + "maintenance job " + ms.Hold.JobID)
	if err != nil {
		return v
	}
	for _, r := range rows {
		switch {
		case r.State == store.StateCancelPending:
			v.Pending = append(v.Pending, r.ID)
		case store.IsTerminalArmState(r.State):
			v.Confirmed = append(v.Confirmed, r.ID)
		}
	}
	return v
}
