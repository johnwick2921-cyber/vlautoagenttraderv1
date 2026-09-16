package trader

import (
	"fmt"
	"strings"
	"time"

	nt "nofx/provider/ninjatrader"
	"nofx/store"
)

// THE REAPER READS THE BROKER, NOT SILENCE.
//
// The old predicate was `now.Sub(row.UpdatedAt) > staleWindow` — no order_update
// for N minutes means cancel. But order_update is an EVENT frame: a resting
// limit that nothing touches emits nothing at all, so after N minutes a
// perfectly healthy order is indistinguishable from a dead one. The reaper read
// SILENCE AS DEATH and then acted on it by cancelling at the broker — turning an
// absence of evidence into a live order's execution.
//
// F12 put the broker's own book on the wire, so the question "is this order
// still alive?" now has an answer that does not depend on anything having
// happened. Three verdicts, and the third is the one that matters most: when the
// link cannot answer, we say so and DO NOTHING. Cancelling on ignorance is what
// the old reaper did.
type reaperVerdict int

const (
	// reaperUnknown — no book, or a book too old to believe. WARN, never cancel.
	reaperUnknown reaperVerdict = iota
	// reaperAlive — a fresh book lists this order as working. Never reap.
	reaperAlive
	// reaperGone — a fresh book does NOT list it (or lists it terminal). The
	// broker's word; reconcile the ledger to match.
	reaperGone
)

func (v reaperVerdict) String() string {
	switch v {
	case reaperAlive:
		return "alive"
	case reaperGone:
		return "gone"
	default:
		return "unknown"
	}
}

// reaperVerdictAt answers for ONE ledger row against the broker's latest book.
// `now` is the caller's clock (class 60 / A28).
func reaperVerdictAt(
	c *nt.OrderSnapshotCache,
	account, symbol string,
	row store.ArmedOrderDB,
	interval time.Duration,
	now time.Time,
) (reaperVerdict, string) {
	// A row we cannot NAME cannot be asked about: there is no identifier to look
	// up in the broker's book, so every order in it will fail to match and the
	// loop below would fall through to GONE — cancelling on ignorance. This is
	// the same defect this file exists to remove, one level down, and it is
	// checked FIRST because it is the case where a cancel is least wanted: it can
	// only arise when something upstream has already gone wrong.
	// (Review finding, nofx-47, 2026-09-04.)
	if strings.TrimSpace(row.SignalID) == "" {
		return reaperUnknown, "link stale: ledger row carries no signal id — the broker cannot be asked about an order we cannot name"
	}
	if c == nil {
		return reaperUnknown, "link stale: no order-snapshot cache — the broker cannot be asked, so nothing is cancelled"
	}
	snap, ok := c.Latest(account)
	age, haveAge := c.AgeAt(account, now)
	if !ok || !haveAge {
		return reaperUnknown, fmt.Sprintf(
			"link stale: no order_snapshot received for %s — silence is not death, nothing is cancelled", account)
	}
	if age > 2*interval {
		// A stale book is not evidence in EITHER direction: listing the order
		// does not prove it alive, omitting it does not prove it dead.
		return reaperUnknown, fmt.Sprintf(
			"link stale: order_snapshot is %ds old (limit %ds) — neither its presence nor its absence is evidence; nothing is cancelled",
			int(age.Seconds()), int(2*interval.Seconds()))
	}

	for _, o := range snap.WorkingOrdersFor(symbol) {
		if orderMatchesArm(o, row) {
			return reaperAlive, fmt.Sprintf(
				"broker lists it working (order %s, book age %ds) — %d min of order_update silence is not death",
				o.OrderID, int(age.Seconds()), int(now.Sub(row.UpdatedAt).Minutes()))
		}
	}
	return reaperGone, fmt.Sprintf(
		"broker's book (age %ds, build %s) does not list it as working — reconciling the ledger to the broker's word",
		int(age.Seconds()), snap.BuildID)
}

// NOTE ON BROKER STATES (verified against nt8_order_snapshots, 2026-09-04): the
// live book carries Working, Accepted, Initialized, Submitted, CancelPending and
// CancelSubmitted. None is terminal, so all read ALIVE — including an arm
// mid-cancel, which is the safe direction: a cancel already in flight does not
// need a second one, and the order still exists at the broker until it does not.
// Asserted by test rather than inherited from "absent from the terminal list",
// because safety by accident is not safety.

// orderMatchesArm ties a broker order to a ledger row. The ledger keys on
// SignalID; the broker may report it as the order's Name, and bracket legs carry
// the -sl/-tp/-lx suffix on the same signal. A match that only worked one way
// would read a live order as gone, which is the failure this whole change exists
// to remove — so all of them count.
func orderMatchesArm(o nt.NT8Order, row store.ArmedOrderDB) bool {
	sig := strings.TrimSpace(row.SignalID)
	if sig == "" {
		return false
	}
	name := strings.TrimSpace(o.Name)
	if strings.EqualFold(name, sig) || strings.EqualFold(strings.TrimSpace(o.OrderID), sig) {
		return true
	}
	for _, suf := range []string{"-sl", "-tp", "-lx"} {
		if strings.EqualFold(name, sig+suf) {
			return true
		}
	}
	return false
}
