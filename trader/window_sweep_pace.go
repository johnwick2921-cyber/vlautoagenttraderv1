package trader

import (
	"time"

	"nofx/store"
)

// ── W1b FOLD-12 — the window sweep paces its cancel to a cancel_pending row ──
//
// When a window class refuses (the E13 force-flat window, the no-trade band)
// the armed pass cancels the plan's resting arms through cancelArmedOrdersSync
// — on the 2-minute scan AND on every live-bar event pass. It re-sent the
// cancel to EVERY non-terminal row each time, cancel_pending ones included
// (cancelArmedOrdersSyncWith filtered only SignalID == ""): on a dark book
// every re-send blocked the pass under armedPassMu for up to 2× the ack
// timeout per row and bumped cancel_attempts on no ack, so the 2-minute T1
// lead × per-bar passes could exhaust the re-request cap the settlement pass
// paces by (cancelReRequestMax).
//
// The rule (CTO ruling): the sweep skips a cancel_pending row whose cancel was
// requested less than windowSweepCancelPace ago; past it the cancel is re-sent
// once — so a lost frame is still retried — and the pace runs again from that
// re-send. "Requested" is the latest request this process can see: the
// ledger's cancel_requested_at_ms (the FIRST request, by any path) or the
// sweep's own last send, whichever is later. Judged on the pass's clock (A28);
// a request stamped after now is younger than the pace, never older.
//
// Only the window sweep passes a pace. The EOD flat, session end, news and T1
// enforce callers of cancelArmedOrdersSync pass none and keep today's
// behaviour: every non-terminal row with a signal, every call.
const windowSweepCancelPace = 30 * time.Second

// armedCancelPace is one window sweep's pace: the pass clock and the trader's
// per-row record of the sweep's last send (row id → pass-clock ms). nil is the
// unpaced sweep — every method is a no-op on it and nowMs is the wall clock,
// exactly what the unpaced callers used before.
type armedCancelPace struct {
	now  time.Time
	last map[int64]int64
}

// windowSweepPace returns the window sweep's pace for a pass at now. The map is
// pass state (at.windowSweepSentMs): read and written only by the window
// sweep, which runs inside maybeManageArmedOrdersAtOpts under armedPassMu.
func (at *AutoTrader) windowSweepPace(now time.Time) *armedCancelPace {
	if at.windowSweepSentMs == nil {
		at.windowSweepSentMs = map[int64]int64{}
	}
	return &armedCancelPace{now: now, last: at.windowSweepSentMs}
}

// firstPace unpacks the optional pace argument (absent = unpaced).
func firstPace(pace []*armedCancelPace) *armedCancelPace {
	if len(pace) == 0 {
		return nil
	}
	return pace[0]
}

// paced reports whether the sweep must skip r this pass: a cancel_pending row
// whose latest request is younger than windowSweepCancelPace. A row with no
// request on record (0 = none requested, never a measured zero) is sent.
func (p *armedCancelPace) paced(r store.ArmedOrderDB) bool {
	if p == nil || r.State != store.StateCancelPending {
		return false
	}
	last := r.CancelRequestedAtMs
	if m := p.last[r.ID]; m > last {
		last = m
	}
	if last <= 0 {
		return false
	}
	return p.now.UnixMilli()-last < windowSweepCancelPace.Milliseconds()
}

// sent records that the sweep requested r's cancel at the pass clock.
func (p *armedCancelPace) sent(id int64) {
	if p == nil {
		return
	}
	p.last[id] = p.now.UnixMilli()
}

// nowMs is the stamp a request carries: the pass clock when paced (so the
// ledger's first-request stamp and the pace read one clock), the wall clock
// otherwise (the unpaced callers' stamp, unchanged).
func (p *armedCancelPace) nowMs() int64 {
	if p == nil {
		return time.Now().UnixMilli()
	}
	return p.now.UnixMilli()
}

// prune forgets rows that left the non-terminal set, so the record holds only
// rows the sweep can still see.
func (p *armedCancelPace) prune(rows []store.ArmedOrderDB) {
	if p == nil || len(p.last) == 0 {
		return
	}
	live := make(map[int64]bool, len(rows))
	for _, r := range rows {
		live[r.ID] = true
	}
	for id := range p.last {
		if !live[id] {
			delete(p.last, id)
		}
	}
}
