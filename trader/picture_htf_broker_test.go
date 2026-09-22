package trader

import (
	"strings"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
	nttrader "nofx/trader/ninjatrader"
)

// W-PICTURE-HTF (2026-09-20, CTO round 2) — broker-state consumer +
// reconciliation through the REAL receipt paths, reproducing the AddOn's
// actual wire semantics rather than injecting what the wire cannot carry:
//   - order_update frames carry fill_price = AverageFillPrice (ACTUAL fill
//     evidence) and quantity = Filled count;
//   - the order_snapshot EXCLUDES terminal orders (Filled/Cancelled/Rejected/
//     Expired) and carries NO fill price — a filled order is ABSENT from the
//     book exactly like a never-placed one;
//   - fill frames carry the actual fill price on the fill stream.
//
// Therefore no test injects a terminal state or a fill price into the
// snapshot, and no recovered fill price may come from LimitPrice.

func pictureSeedBrokerRows(t *testing.T, st *store.Store) (entryRow, otherRow *store.PictureHtfOpportunityDB) {
	t.Helper()
	r1 := &store.PictureHtfOpportunityDB{
		OppKey: "broker|1", TraderID: "trader-1", Stage: "place_pending", Direction: "long",
		SignalID: "sig-entry", StopPx: 98.25, TargetPx: 110, EntryRef: 101.49,
	}
	r2 := &store.PictureHtfOpportunityDB{
		OppKey: "broker|2", TraderID: "trader-1", Stage: "place_pending", Direction: "long",
		SignalID: "sig-other", StopPx: 98.25, TargetPx: 110, EntryRef: 101.49,
	}
	if _, fresh, err := st.PictureHtfClaim(r1); err != nil || !fresh {
		t.Fatalf("claim r1: %v %v", err, fresh)
	}
	if _, fresh, err := st.PictureHtfClaim(r2); err != nil || !fresh {
		t.Fatalf("claim r2: %v %v", err, fresh)
	}
	// Both rows own their submission.
	if _, err := st.PictureHtfClaimSubmission(r1.OppKey, "sig-entry"); err != nil {
		t.Fatalf("own r1: %v", err)
	}
	if _, err := st.PictureHtfClaimSubmission(r2.OppKey, "sig-other"); err != nil {
		t.Fatalf("own r2: %v", err)
	}
	return r1, r2
}

// pictureBrokerTestReset clears the per-trader process state (received
// history, consumer registry, armed subscription) a test's trader id would
// otherwise inherit.
func pictureBrokerTestReset(at *AutoTrader) {
	pictureBrokerHistories.Delete(at.id)
	pictureHtfBrokerConsumers.Delete(at.id)
	armedSubs.Delete(at.id)
}

// pictureWireBroker installs a REAL TCPTrader on a REAL TCPServer (bound
// like production: symbol MNQ, account Sim101) and returns the server so the
// test can drive the real snapshot cache and the real routers.
func pictureWireBroker(t *testing.T, at *AutoTrader) *ntwire.TCPServer {
	t.Helper()
	s := ntwire.NewTCPServer(nil)
	at.trader = nttrader.NewTCPTrader(s, "MNQ", "Sim101")
	at.config.NinjaTraderSymbol = "MNQ"
	return s
}

// pictureSeedOnePending claims and owns a single place_pending row with the
// given signal id (no received evidence yet).
func pictureSeedOnePending(t *testing.T, st *store.Store, oppKey, signalID string) *store.PictureHtfOpportunityDB {
	t.Helper()
	r := &store.PictureHtfOpportunityDB{
		OppKey: oppKey, TraderID: "trader-1", Stage: "place_pending", Direction: "long",
		SignalID: signalID, StopPx: 98.25, TargetPx: 110, EntryRef: 101.49,
	}
	if _, fresh, err := st.PictureHtfClaim(r); err != nil || !fresh {
		t.Fatalf("claim %s: %v %v", oppKey, err, fresh)
	}
	if _, err := st.PictureHtfClaimSubmission(r.OppKey, signalID); err != nil {
		t.Fatalf("own %s: %v", oppKey, err)
	}
	return r
}

func TestPictureHtfConsumeOrderUpdateEntryLifecycle(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{})
	pictureBrokerTestReset(at)
	r1, _ := pictureSeedBrokerRows(t, st)

	// working — moves forward, no fill yet.
	pictureHtfConsumeOrderUpdate(at, ntwire.OrderUpdatePayload{SignalID: "sig-entry", OrderName: "sig-entry", State: "working"})
	got, _, _ := st.PictureHtfGet(r1.OppKey)
	if got.Stage != "working" {
		t.Fatalf("working must move the row forward: %+v", got)
	}
	// fill at 101.60 → filled, actual R:R computed from the row geometry.
	pictureHtfConsumeOrderUpdate(at, ntwire.OrderUpdatePayload{SignalID: "sig-entry", OrderName: "sig-entry", State: "filled", FillPrice: 101.60, Quantity: 1})
	got, _, _ = st.PictureHtfGet(r1.OppKey)
	if got.Stage != "filled" || got.FillPrice != 101.60 || got.FillQty != 1 {
		t.Fatalf("fill must stamp stage/price/qty: %+v", got)
	}
	wantRR := (110 - 101.60) / (101.60 - 98.25)
	if got.FillRR < wantRR-1e-9 || got.FillRR > wantRR+1e-9 {
		t.Fatalf("actual-fill R:R must be computed from the row geometry: got %.4f want %.4f", got.FillRR, wantRR)
	}
	// A duplicate/late working event must NEVER downgrade a filled row.
	pictureHtfConsumeOrderUpdate(at, ntwire.OrderUpdatePayload{SignalID: "sig-entry", OrderName: "sig-entry", State: "working"})
	got, _, _ = st.PictureHtfGet(r1.OppKey)
	if got.Stage != "filled" {
		t.Fatalf("a late event must never downgrade: %+v", got)
	}
}

func TestPictureHtfConsumeOrderUpdateRejectionCarriesReason(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{})
	pictureBrokerTestReset(at)
	r1, _ := pictureSeedBrokerRows(t, st)
	pictureHtfConsumeOrderUpdate(at, ntwire.OrderUpdatePayload{
		SignalID: "sig-entry", OrderName: "sig-entry", State: "rejected", Reason: "sim: no market data",
	})
	got, _, _ := st.PictureHtfGet(r1.OppKey)
	if got.Stage != "rejected" || !strings.Contains(got.RejectReason, "no market data") {
		t.Fatalf("the rejection reason must ride the row: %+v", got)
	}
}

func TestPictureHtfConsumeProtectiveLegsUpdateProtectionNotFill(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{})
	pictureBrokerTestReset(at)
	r1, _ := pictureSeedBrokerRows(t, st)
	// Entry fills first.
	pictureHtfConsumeOrderUpdate(at, ntwire.OrderUpdatePayload{SignalID: "sig-entry", OrderName: "sig-entry", State: "filled", FillPrice: 101.60, Quantity: 1})
	// The SL leg fills (stop-out): the row must NOT lose its entry fill, but
	// the protection event is recorded.
	pictureHtfConsumeOrderUpdate(at, ntwire.OrderUpdatePayload{SignalID: "sig-entry", OrderName: "sig-entry-sl", State: "filled", FillPrice: 98.25, Quantity: 1})
	got, _, _ := st.PictureHtfGet(r1.OppKey)
	if got.Stage != "filled" || got.FillPrice != 101.60 {
		t.Fatalf("a protective-leg fill must never clobber the entry fill: %+v", got)
	}
	if !strings.Contains(got.BrokerStatus, "protection_sl_filled") {
		t.Fatalf("the SL-leg fill must be recorded as protection evidence: %+v", got)
	}
	// A rejected TP leg keeps the entry fill, records the rejection, AND the
	// SL evidence is preserved (notes append — never overwrite each other).
	pictureHtfConsumeOrderUpdate(at, ntwire.OrderUpdatePayload{SignalID: "sig-entry", OrderName: "sig-entry-tp", State: "rejected", Reason: "cancelled by OCO"})
	got, _, _ = st.PictureHtfGet(r1.OppKey)
	if got.Stage != "filled" || got.FillPrice != 101.60 {
		t.Fatalf("protection updates must not touch the entry fill: %+v", got)
	}
	if !strings.Contains(got.BrokerStatus, "protection_tp_rejected") {
		t.Fatalf("the TP-leg rejection must be recorded: %+v", got)
	}
	if !strings.Contains(got.BrokerStatus, "protection_sl_filled") {
		t.Fatalf("the SL-leg evidence must survive the TP note (append, not replace): %+v", got)
	}
}

func TestPictureHtfConsumeOrderUpdateIsolation(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{})
	pictureBrokerTestReset(at)
	r1, r2 := pictureSeedBrokerRows(t, st)
	pictureHtfConsumeOrderUpdate(at, ntwire.OrderUpdatePayload{SignalID: "sig-other", OrderName: "sig-other", State: "filled", FillPrice: 99.9, Quantity: 1})
	got1, _, _ := st.PictureHtfGet(r1.OppKey)
	got2, _, _ := st.PictureHtfGet(r2.OppKey)
	if got1.Stage != "place_pending" {
		t.Fatalf("another signal's event must not touch this row: %+v", got1)
	}
	if got2.Stage != "filled" || got2.FillPrice != 99.9 {
		t.Fatalf("the named row must get the event: %+v", got2)
	}
}

// ── Reconciliation: RECEIVED-evidence recovery through the real paths ────────
//
// The wire cannot deliver terminal outcomes through the snapshot, so recovery
// tests drive what the wire CAN deliver: order_update frames (with the actual
// average fill price), fill frames, and a working-only snapshot book.

// The frame arrives BEFORE any ledger row exists (the row write racing the
// wire is exactly the gap the history exists for). The sweep must recover the
// fill from the RECEIVED frame — with the frame's actual fill price, never a
// limit — and the recovered row must block re-entry.
func TestPictureHtfReconcileRecoversFillFromReceivedHistory(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{})
	pictureBrokerTestReset(at)
	pictureHtfConsumeOrderUpdate(at, ntwire.OrderUpdatePayload{SignalID: "sig-f", OrderName: "sig-f", State: "filled", FillPrice: 101.60, Quantity: 1})
	r := pictureSeedOnePending(t, st, "broker|rf", "sig-f")
	pictureHtfReconcilePending(at)
	got, _, _ := st.PictureHtfGet(r.OppKey)
	if got.Stage != "filled" || got.FillPrice != 101.60 || got.FillQty != 1 {
		t.Fatalf("the sweep must recover the fill from the received frame: %+v", got)
	}
	wantRR := (110 - 101.60) / (101.60 - 98.25)
	if got.FillRR < wantRR-1e-9 || got.FillRR > wantRR+1e-9 {
		t.Fatalf("actual-fill R:R must use the received fill price: %.4f want %.4f", got.FillRR, wantRR)
	}
	if won, _ := st.PictureHtfClaimSubmission(r.OppKey, "retry"); won {
		t.Fatalf("a recovered row must block re-entry")
	}
}

// A rejection received before the row existed must also recover, with the
// wire's reason preserved.
func TestPictureHtfReconcileRecoversRejectionFromReceivedHistory(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{})
	pictureBrokerTestReset(at)
	pictureHtfConsumeOrderUpdate(at, ntwire.OrderUpdatePayload{SignalID: "sig-r", OrderName: "sig-r", State: "rejected", Reason: "price check refused this entry"})
	r := pictureSeedOnePending(t, st, "broker|rr", "sig-r")
	pictureHtfReconcilePending(at)
	got, _, _ := st.PictureHtfGet(r.OppKey)
	if got.Stage != "rejected" || !strings.Contains(got.RejectReason, "price check") {
		t.Fatalf("the sweep must recover the rejection with its reason: %+v", got)
	}
}

// FINDING 1 (production mismatch). The AddOn EXCLUDES terminal orders from
// the snapshot, so a FILLED order is ABSENT from the book exactly like a
// never-placed one. With no received history the sweep must NOT invent an
// outcome: the row stays place_pending, marked unknown, and is never resent.
// (The old tests injected a Filled order INTO the snapshot — a state the wire
// cannot carry.)
func TestPictureHtfReconcileAbsentBookStaysUnknownNeverResends(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{})
	pictureBrokerTestReset(at)
	s := pictureWireBroker(t, at)
	// The wire's explicit empty book for an account with no working orders.
	s.OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: []ntwire.NT8Order{}}, time.Now())
	r := pictureSeedOnePending(t, st, "broker|ab", "sig-a")
	pictureHtfReconcilePending(at)
	pictureHtfReconcilePending(at) // idempotent — the marker must not duplicate
	got, _, _ := st.PictureHtfGet(r.OppKey)
	if got.Stage != "place_pending" {
		t.Fatalf("an absent book order proves nothing — the row must stay place_pending: %+v", got)
	}
	if got.FillPrice != 0 {
		t.Fatalf("no fill price may be fabricated: %+v", got)
	}
	if strings.Count(got.BrokerStatus, "reconcile:no_received_evidence") != 1 {
		t.Fatalf("the unknown marker must be recorded exactly once: %+v", got)
	}
}

// FINDING 2 (production mismatch). The snapshot's limit_price is the order's
// LIMIT, not an execution price, and the book carries no fill price at all.
// A working book entry with a filled quantity must record the working receipt
// and leave the fill price UNKNOWN — LimitPrice must never become the fill.
func TestPictureHtfReconcileWorkingBookLeavesFillPriceUnknown(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{})
	pictureBrokerTestReset(at)
	s := pictureWireBroker(t, at)
	// Wire shape: a WORKING order with filled qty and its limit — no fill price.
	s.OrderSnapshots().PutAt(ntwire.OrderSnapshotPayload{Account: "Sim101", Orders: []ntwire.NT8Order{
		{OrderID: "nt8-w", Name: "sig-w", State: "working", Filled: 1, LimitPrice: 101.60, Quantity: 1},
	}}, time.Now())
	r := pictureSeedOnePending(t, st, "broker|wb", "sig-w")
	pictureHtfReconcilePending(at)
	got, _, _ := st.PictureHtfGet(r.OppKey)
	if got.Stage != "working" || got.BrokerOrderID != "nt8-w" {
		t.Fatalf("a present working book entry is a working receipt: %+v", got)
	}
	if got.FillPrice != 0 {
		t.Fatalf("LimitPrice must never become the fill price: %+v", got)
	}
	if !strings.Contains(got.BrokerStatus, "fill price unknown") {
		t.Fatalf("the unknown fill price must be stated: %+v", got)
	}
}

// A fill received on the FILL stream (the wire's second execution-evidence
// path) recovers the row with the actual fill price — driven through the real
// fill router and the trader's received-fill ring.
func TestPictureHtfReconcileRecoversFillFromFillStream(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{})
	pictureBrokerTestReset(at)
	s := pictureWireBroker(t, at)
	r := pictureSeedOnePending(t, st, "broker|ff", "sig-fill")
	s.FeedFillForTest(ntwire.FillPayload{SignalID: "sig-fill", Symbol: "MNQ", Account: "Sim101", Status: "filled", FillPrice: 101.60, Quantity: 1})
	broker := at.trader.(*nttrader.TCPTrader)
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, _, ok := broker.RecentFillFor("sig-fill"); ok {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("the fill frame never reached the received-fill ring")
		}
		time.Sleep(5 * time.Millisecond)
	}
	pictureHtfReconcilePending(at)
	got, _, _ := st.PictureHtfGet(r.OppKey)
	if got.Stage != "filled" || got.FillPrice != 101.60 || got.FillQty != 1 {
		t.Fatalf("the sweep must recover the fill from the fill stream: %+v", got)
	}
}

// pictureWaitForRingFill polls the trader's received-fill ring until the
// fill frame routed through the REAL fill router has been recorded.
func pictureWaitForRingFill(t *testing.T, at *AutoTrader, signalID string) {
	t.Helper()
	broker := at.trader.(*nttrader.TCPTrader)
	deadline := time.Now().Add(2 * time.Second)
	for {
		if _, _, ok := broker.RecentFillFor(signalID); ok {
			return
		}
		if time.Now().After(deadline) {
			t.Fatal("the fill frame never reached the received-fill ring")
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// FINDING 4 (a): a working order_update arrives and moves the row to working,
// then a FILL frame arrives — with NO filled order_update ever received. The
// sweep must recover the actual fill for a WORKING row (working is not a
// terminal outcome) and no additional entry may be submitted.
func TestPictureHtfReconcileWorkingReceiptThenFillFrameRecovers(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{})
	pictureBrokerTestReset(at)
	s := pictureWireBroker(t, at)
	r := pictureSeedOnePending(t, st, "broker|wf", "sig-wf")
	pictureHtfConsumeOrderUpdate(at, ntwire.OrderUpdatePayload{SignalID: "sig-wf", OrderName: "sig-wf", State: "working"})
	got, _, _ := st.PictureHtfGet(r.OppKey)
	if got.Stage != "working" {
		t.Fatalf("the working receipt must move the row forward: %+v", got)
	}
	// The fill arrives on the FILL stream only — no filled order_update.
	s.FeedFillForTest(ntwire.FillPayload{SignalID: "sig-wf", Symbol: "MNQ", Account: "Sim101", Status: "filled", FillPrice: 101.60, Quantity: 1})
	pictureWaitForRingFill(t, at, "sig-wf")
	pictureHtfReconcilePending(at)
	got, _, _ = st.PictureHtfGet(r.OppKey)
	if got.Stage != "filled" || got.FillPrice != 101.60 || got.FillQty != 1 {
		t.Fatalf("a working row must still recover the received fill: %+v", got)
	}
	if won, _ := st.PictureHtfClaimSubmission(r.OppKey, "retry"); won {
		t.Fatalf("the recovered row must block re-entry — no additional entry")
	}
}

// FINDING 4 (b): a pending row with OLDER working history and NEWER fill
// evidence. The working receipt must NOT mask the fill — execution evidence
// outranks working receipts.
func TestPictureHtfReconcilePendingPrefersFillOverOlderWorkingHistory(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{})
	pictureBrokerTestReset(at)
	s := pictureWireBroker(t, at)
	// The working receipt arrives BEFORE the row exists — the history records
	// it (the row write racing the wire).
	pictureHtfConsumeOrderUpdate(at, ntwire.OrderUpdatePayload{SignalID: "sig-pw", OrderName: "sig-pw", State: "working"})
	r := pictureSeedOnePending(t, st, "broker|pw", "sig-pw")
	// Newer execution evidence on the fill stream.
	s.FeedFillForTest(ntwire.FillPayload{SignalID: "sig-pw", Symbol: "MNQ", Account: "Sim101", Status: "filled", FillPrice: 101.60, Quantity: 1})
	pictureWaitForRingFill(t, at, "sig-pw")
	pictureHtfReconcilePending(at)
	got, _, _ := st.PictureHtfGet(r.OppKey)
	if got.Stage != "filled" || got.FillPrice != 101.60 || got.FillQty != 1 {
		t.Fatalf("older working history must not mask the received fill: %+v", got)
	}
	if won, _ := st.PictureHtfClaimSubmission(r.OppKey, "retry"); won {
		t.Fatalf("the recovered row must block re-entry — no additional entry")
	}
}

// FINDING 4 (c): restart with a PERSISTED working row, then received
// execution evidence. The process-local history disappears on restart (that
// limitation is explicit and intended — unreceived outcomes stay unknown);
// the persisted working row remains recoverable and post-restart execution
// evidence settles it.
func TestPictureHtfReconcileRestartPersistedWorkingRecoversFill(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{})
	pictureBrokerTestReset(at)
	s := pictureWireBroker(t, at)
	r := pictureSeedOnePending(t, st, "broker|rs", "sig-rs")
	pictureHtfConsumeOrderUpdate(at, ntwire.OrderUpdatePayload{SignalID: "sig-rs", OrderName: "sig-rs", State: "working"})
	got, _, _ := st.PictureHtfGet(r.OppKey)
	if got.Stage != "working" {
		t.Fatalf("the row must be persisted working before the restart: %+v", got)
	}
	// RESTART: the received-history is process-local and gone. The DB row is
	// the only survivor.
	pictureBrokerHistories.Delete(at.id)
	pictureHtfReconcilePending(at)
	got, _, _ = st.PictureHtfGet(r.OppKey)
	if got.Stage != "working" {
		t.Fatalf("a persisted working row with no new evidence stays working: %+v", got)
	}
	// Post-restart execution evidence arrives and settles the row.
	s.FeedFillForTest(ntwire.FillPayload{SignalID: "sig-rs", Symbol: "MNQ", Account: "Sim101", Status: "filled", FillPrice: 101.60, Quantity: 1})
	pictureWaitForRingFill(t, at, "sig-rs")
	pictureHtfReconcilePending(at)
	got, _, _ = st.PictureHtfGet(r.OppKey)
	if got.Stage != "filled" || got.FillPrice != 101.60 || got.FillQty != 1 {
		t.Fatalf("a persisted working row must recover post-restart fill evidence: %+v", got)
	}
	if won, _ := st.PictureHtfClaimSubmission(r.OppKey, "retry"); won {
		t.Fatalf("the recovered row must block re-entry — no additional entry")
	}
}

// FINDING 3 (integration): the picture consumer rides the REAL subscription —
// router → fan-out listener → consumer → ledger — and coexists with the armed
// executor's listener on the same (symbol, account): every fed frame reaches
// BOTH consumers and neither channel is closed by the other.
func TestPictureConsumerAndArmedStreamCoexistOnOneSubscription(t *testing.T) {
	at, st := resetTrader(t, store.StrategyConfig{})
	pictureBrokerTestReset(at)
	s := pictureWireBroker(t, at)
	r := pictureSeedOnePending(t, st, "broker|co", "sig-co")
	broker := at.trader.(*nttrader.TCPTrader)
	armedCh := at.armedUpdateStream(broker)
	at.ensurePictureHtfBrokerConsumer()
	// Poll-feed working frames until the picture consumer proves the full
	// path (the consumer goroutine listens asynchronously).
	sent := 0
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		s.FeedOrderUpdateForTest(ntwire.OrderUpdatePayload{SignalID: "sig-co", OrderName: "sig-co", State: "working", Symbol: "MNQ", Account: "Sim101", Seq: uint64(sent)})
		sent++
		time.Sleep(5 * time.Millisecond)
		got, _, _ := st.PictureHtfGet(r.OppKey)
		if got.Stage == store.StateWorking {
			break
		}
	}
	got, _, _ := st.PictureHtfGet(r.OppKey)
	if got.Stage != store.StateWorking {
		t.Fatalf("the consumer never received a frame through the real subscription: %+v", got)
	}
	// The armed listener must have received EVERY fed frame (no loss) and its
	// channel must still be open (no eviction).
	for i := 0; i < sent; i++ {
		select {
		case u, open := <-armedCh:
			if !open {
				t.Fatalf("armed channel CLOSED — the picture consumer evicted it (frame %d/%d)", i, sent)
			}
			if u.SignalID != "sig-co" {
				t.Fatalf("wrong frame on the armed listener: %+v", u)
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("armed listener lost a frame (%d/%d)", i, sent)
		}
	}
	select {
	case _, open := <-armedCh:
		if open {
			t.Fatal("unexpected extra frame on the armed listener")
		}
		t.Fatal("armed channel closed after the drain")
	default: // open and empty — healthy
	}
	// And the picture consumer still receives after the armed drain.
	s.FeedOrderUpdateForTest(ntwire.OrderUpdatePayload{SignalID: "sig-co", OrderName: "sig-co", State: "filled", FillPrice: 101.45, Quantity: 1, Symbol: "MNQ", Account: "Sim101"})
	deadline = time.Now().Add(2 * time.Second)
	for {
		got, _, _ = st.PictureHtfGet(r.OppKey)
		if got.Stage == store.StateFilled {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("the picture consumer lost frames after the armed drain: %+v", got)
		}
		time.Sleep(5 * time.Millisecond)
	}
	if got.FillPrice != 101.45 {
		t.Fatalf("the consumer must stamp the wire's actual fill price: %+v", got)
	}
}
