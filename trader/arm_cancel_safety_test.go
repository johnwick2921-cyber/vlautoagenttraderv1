package trader

import (
	"os"
	"strings"
	"testing"

	nt "nofx/provider/ninjatrader"
	"nofx/store"
)

// ── THE 2026-09-06 23:35:02 → 23:37:02 REPLAY ────────────────────────────────
//
// Position 592 lost its stop because a guard acted on a ledger row that was a
// minute out of date. These are the REAL books NT8 sent, copied from
// nt8_order_snapshots 8208 / 8209 / 8212 / 8213 — including the detail that
// settles it: seven seconds before the guard fired, the book already held TWO
// CHILDREN AND NO ENTRY.

const sig592 = "aa07e583-6df3-4148-9d57-667630f1b155"

// snapshot 8208 @ 23:36:55 — the entry has FILLED; only its protections remain.
func book8208() []nt.NT8Order {
	return []nt.NT8Order{
		{OrderID: "o-sl", Name: sig592 + "-sl", State: "Accepted", Type: "stop", StopPrice: 29554},
		{OrderID: "o-tp", Name: sig592 + "-tp", State: "Working", Type: "limit", LimitPrice: 29623},
	}
}

// snapshot 8212 @ 23:37:02 — both children dying, after the cancel landed.
func book8212() []nt.NT8Order {
	return []nt.NT8Order{
		{OrderID: "o-sl", Name: sig592 + "-sl", State: "CancelSubmitted", Type: "stop", StopPrice: 29554},
		{OrderID: "o-tp", Name: sig592 + "-tp", State: "CancelSubmitted", Type: "limit", LimitPrice: 29623},
	}
}

// the book while the entry was genuinely still resting (before the fill).
func bookEntryResting() []nt.NT8Order {
	return []nt.NT8Order{
		{OrderID: "o-entry", Name: sig592, State: "Working", Type: "limit", LimitPrice: 29576},
	}
}

// THE REPLAY. The ledger still says "working" — it had not drained the fill —
// and this is exactly the state the guard saw. The cancel MUST be refused.
func TestReplay592_GuardMustNotCancelAFilledArm(t *testing.T) {
	v := adjudicateArmCancel(store.StateWorking, sig592, book8208(), true)
	if v.Allow {
		t.Fatalf("REPLAY 23:37:02 — the cancel was ALLOWED against a book holding only the OCO children; position 592 lost its 29554 stop exactly here: %s", v.Why)
	}
	t.Logf("refused: %s", v.Why)
}

// The ledger's own word is enough on its own once the fill HAS been drained —
// which is what fix 1 guarantees.
func TestAFilledArmIsNeverCancelledEvenWithNoBook(t *testing.T) {
	if v := adjudicateArmCancel(store.StateFilled, sig592, nil, false); v.Allow {
		t.Fatalf("a FILLED arm must never be cancelled: %s", v.Why)
	}
	if v := adjudicateArmCancel(store.StateFilled, sig592, bookEntryResting(), true); v.Allow {
		t.Fatal("a FILLED arm must never be cancelled, even if something bare-named is still in the book")
	}
}

// The mirror: a genuinely resting ENTRY is still cancellable, or the guard
// would be satisfied by refusing everything and the one-live-arm rule would die.
func TestARestingEntryIsStillCancellable(t *testing.T) {
	v := adjudicateArmCancel(store.StateWorking, sig592, bookEntryResting(), true)
	if !v.Allow {
		t.Fatalf("a resting entry must remain cancellable: %s", v.Why)
	}
}

// Cancelling is the destructive branch — it can remove a live stop — so an
// unreadable book refuses.
func TestNoBookRefusesTheCancel(t *testing.T) {
	if v := adjudicateArmCancel(store.StateWorking, sig592, nil, false); v.Allow {
		t.Fatal("with no broker book the cancel must be refused, not attempted blind")
	}
	// A FRESH book with nothing under this signal is NOT the dangerous case:
	// there is no protection to lose, and refusing would strand the ledger row
	// working forever. It is allowed, deliberately.
	if v := adjudicateArmCancel(store.StateWorking, sig592, []nt.NT8Order{}, true); !v.Allow {
		t.Fatalf("an empty FRESH book has no protection to lose and must not be refused: %s", v.Why)
	}
}

// Children that are already dying still count as children, not as an entry.
func TestDyingChildrenAreStillNotAnEntry(t *testing.T) {
	if v := adjudicateArmCancel(store.StateWorking, sig592, book8212(), true); v.Allow {
		t.Fatalf("CancelSubmitted children are not an entry: %s", v.Why)
	}
}

// entryIsResting must separate the entry from its children by NAME, which is
// the only thing that distinguishes them on the wire.
func TestEntryAndChildrenAreToldApartByName(t *testing.T) {
	resting, children := entryIsResting(book8208(), sig592)
	if resting {
		t.Fatal("book 8208 holds no entry — only -sl and -tp")
	}
	if !children {
		t.Fatal("book 8208 holds two children and they must be seen")
	}
	resting, children = entryIsResting(bookEntryResting(), sig592)
	if !resting || children {
		t.Fatalf("a bare-named order IS the entry: resting=%v children=%v", resting, children)
	}
}

// FIX 1 — THE ORDERING, PINNED IN THE SOURCE.
//
// The whole incident is an ordering bug: every guard in maybeManageArmedOrders
// decided before the fill was drained. This asserts the drain comes FIRST. It
// is a source pin because the defect is a sequence, and no runtime assertion
// can see an ordering that was never exercised.
func TestFillIsDrainedBeforeAnyGuardEvaluatesState(t *testing.T) {
	b, err := os.ReadFile("armed_executor.go")
	if err != nil {
		t.Fatal(err)
	}
	src := string(b)
	fnAt := strings.Index(src, "func (at *AutoTrader) maybeManageArmedOrders(")
	if fnAt < 0 {
		t.Fatal("maybeManageArmedOrders not found — this pin has lost its subject")
	}
	body := src[fnAt:]
	// Match the CALL, not the name: the first draft of this pin searched for
	// "consumeArmedOrderUpdates" and was satisfied by the COMMENT above the
	// call, so deleting the call left it green. A pin its own documentation can
	// satisfy is not a pin (A8).
	drain := strings.Index(body, "at.consumeArmedOrderUpdates(nt, ledger)")
	guard := strings.Index(body, "at.oneLiveArmGuard(")
	if guard < 0 {
		guard = strings.Index(body, "oneLiveArmGuard(sc, leg, side)")
	}
	if drain < 0 {
		t.Fatal("the fill drain is not called inside maybeManageArmedOrders — a guard would again read a stale ledger")
	}
	if guard < 0 {
		t.Fatal("oneLiveArmGuard not found in maybeManageArmedOrders")
	}
	if drain > guard {
		t.Fatalf("THE FILL IS DRAINED AFTER THE GUARD (drain@%d > guard@%d) — this is the 2026-09-06 23:37:02 defect: the guard reads a ledger row that is a minute out of date", drain, guard)
	}
}

// FIX 4 — A DYING ORDER IS NOT AN ACCEPTED ONE.
//
// accepted_risk ids 9 and 10 recorded stop=29554 / target=29623 from a book in
// which the -sl read CancelPending and the -tp CancelSubmitted. The row claimed
// position 592 was protected at 29554 while the broker was withdrawing exactly
// that protection.
func TestAcceptedRiskRefusesACancellingBracket(t *testing.T) {
	// snapshot 8209: -sl CancelPending, -tp Working
	row := &store.AcceptedRisk{SignalID: sig592}
	applyBrokerTerms(row, []nt.NT8Order{
		{Name: sig592 + "-sl", State: "CancelPending", Type: "stop", StopPrice: 29554},
		{Name: sig592 + "-tp", State: "Working", Type: "limit", LimitPrice: 29623},
	}, sig592)
	if row.AcceptedStopPx != nil {
		t.Fatalf("a CancelPending stop must NOT be recorded as accepted, got %.2f — this is exactly accepted_risk id 9", *row.AcceptedStopPx)
	}
	if row.AcceptedTargetPx == nil || *row.AcceptedTargetPx != 29623 {
		t.Fatal("the still-Working target must still be recorded")
	}

	// snapshot 8212: both dying — nothing may be recorded
	row2 := &store.AcceptedRisk{SignalID: sig592}
	applyBrokerTerms(row2, book8212(), sig592)
	if row2.AcceptedStopPx != nil || row2.AcceptedTargetPx != nil {
		t.Fatalf("a fully cancelling bracket records NOTHING: stop=%v target=%v", row2.AcceptedStopPx, row2.AcceptedTargetPx)
	}

	// and a genuinely live bracket still records
	row3 := &store.AcceptedRisk{SignalID: sig592}
	applyBrokerTerms(row3, book8208(), sig592)
	if row3.AcceptedStopPx == nil || *row3.AcceptedStopPx != 29554 {
		t.Fatalf("a live Accepted stop must still be recorded, got %v", row3.AcceptedStopPx)
	}
}

func TestCancelInFlightStatesAreNamedExactly(t *testing.T) {
	for _, s := range []string{"CancelPending", "CancelSubmitted", "cancelpending", " CancelSubmitted "} {
		if !isCancelInFlight(s) {
			t.Fatalf("%q is a cancel in flight", s)
		}
	}
	for _, s := range []string{"Working", "Accepted", "Initialized", "Submitted", ""} {
		if isCancelInFlight(s) {
			t.Fatalf("%q is NOT a cancel in flight", s)
		}
	}
}

// ── THE ORPHAN CASE (2026-09-07) ─────────────────────────────────────────────
//
// The children-without-entry shape means two OPPOSITE things depending on one
// fact the adjudicator was not given:
//
//	position OPEN  → those children are the protection. 2026-09-06 23:37:02.
//	position FLAT  → those children are ORPHANS. Class 27, 2026-08-31: a
//	                 netting close left an arm's SL resting and it fired 26
//	                 minutes later, opening a NAKED SHORT.
//
// Refusing both leaves the orphan alive; allowing both is the naked stop. The
// guard therefore asks whether a position is actually open, and — A24 — an
// UNKNOWN position is treated as open, because that is the non-destructive side.
func TestOrphanBracketIsCancellableOnlyWhenTheBrokerSaysFlat(t *testing.T) {
	book := []nt.NT8Order{
		{Name: sig592 + "-sl", State: "Accepted", StopPrice: 29554},
		{Name: sig592 + "-tp", State: "Working", LimitPrice: 29623},
	}

	// broker FLAT → orphans, sweep them (class 27)
	if v := adjudicateArmCancelWith(store.StateWorking, sig592, book, true,
		positionContext{Known: true, Open: false}); !v.Allow {
		t.Fatalf("orphan legs left alive with the broker FLAT — this is class 27: a resting stop "+
			"fires later and opens a naked position. why=%s", v.Why)
	}

	// broker holds a POSITION → this is protection, never touch it (09-06)
	if v := adjudicateArmCancelWith(store.StateWorking, sig592, book, true,
		positionContext{Known: true, Open: true}); v.Allow {
		t.Fatal("the protective pair of an OPEN position was cleared for cancellation — 2026-09-06")
	}

	// broker FLAT with NO book at all → still a sweep: the book only matters
	// for deciding whether protection is at risk, and a flat account has none.
	if v := adjudicateArmCancelWith(store.StateWorking, sig592, nil, false,
		positionContext{Known: true, Open: false}); !v.Allow {
		t.Fatalf("a confirmed-FLAT account refused an orphan sweep for want of a book — the book "+
			"exists to protect a live position, and there is none. why=%s", v.Why)
	}

	// position UNKNOWN → the non-destructive side, exactly as before
	if v := adjudicateArmCancelWith(store.StateWorking, sig592, book, true,
		positionContext{}); v.Allow {
		t.Fatal("an UNKNOWN position took the destructive branch (A24)")
	}
	// and the plain entry point must keep that conservative default
	if v := adjudicateArmCancel(store.StateWorking, sig592, book, true); v.Allow {
		t.Fatal("the default adjudicator stopped refusing the children case")
	}
}
