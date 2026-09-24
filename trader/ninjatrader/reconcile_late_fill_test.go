package ninjatrader

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// ── W1b E15 — a late fill materialized through reconcile's untracked path is
// tagged with the signal of its OWN fill (the ring), never with a stale arm's
// identity picked by price similarity ────────────────────────────────────────
//
// CLASS 160 leaves an NT8 AI open whose fill misses the ~3 s poll as an order
// row keyed by its signal (status NEW) and NO position. NT8's own position is
// later materialized by reconcilePositions' untracked branch. Before W1b that
// branch recovered identity only by PRICE (StampArmedLineageIfMatched: any
// filled arm of this trader within one tick, no time bound), so a late AI fill
// was anonymous — or adopted an old arm's plan and signal — while the fill ring
// held the exact signal the whole time.

const lateFillTrader, lateFillExchange = "late-fill", "ex-nt"

type lateFillWire struct {
	st *store.Store
	s  *ntwire.TCPServer
	tr *TCPTrader
}

func newLateFillWire(t *testing.T) *lateFillWire {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "late.db"))
	if err != nil {
		t.Fatal(err)
	}
	s := ntwire.NewTCPServer(nil)
	s.SetAddrForTest("127.0.0.1:0")
	ctx, cancel := context.WithCancel(context.Background())
	if err := s.Start(ctx); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Stop(); cancel() })
	return &lateFillWire{st: st, s: s, tr: NewTCPTrader(s, "MNQ", "Sim101")}
}

// aiOrderRow writes the order row exactly as createOrderRecord does for an NT8
// open whose fill the CLASS 160 poll did not see (keyed by signal, NEW).
func (w *lateFillWire) aiOrderRow(t *testing.T, traderID, sid, action, posSide, side string) {
	t.Helper()
	if err := w.st.Order().CreateOrder(&store.TraderOrder{TraderID: traderID, ExchangeID: lateFillExchange, ExchangeType: "ninjatrader",
		ExchangeOrderID: sid, Symbol: "MNQ", Side: side, PositionSide: posSide, Type: "MARKET", TimeInForce: "GTC",
		Quantity: 1, Status: "NEW", OrderAction: action}); err != nil {
		t.Fatal(err)
	}
}

// fill routes a fill frame through the production fill router into the ring,
// and waits (bounded) until the ring holds it.
func (w *lateFillWire) fill(t *testing.T, sid, side string, px float64) {
	t.Helper()
	w.s.FeedFillForTest(ntwire.FillPayload{SignalID: sid, Symbol: "MNQ", Account: "Sim101", Side: side, Quantity: 1,
		FillPrice: px, Status: "filled", FillTime: time.Now().UTC().Format(time.RFC3339)})
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if _, _, ok := w.tr.RecentFillFor(sid); ok {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("fill %s never reached the ring", sid)
}

// filledArm writes a FILLED armed row the way the armed executor leaves one.
func (w *lateFillWire) filledArm(t *testing.T, planID, scenario, sid string, px float64, age time.Duration) *store.ArmedOrderDB {
	t.Helper()
	r := &store.ArmedOrderDB{TraderID: lateFillTrader, PlanID: planID, Scenario: scenario, Version: 3, State: "armed", Side: "long", EntryPx: px, StopPx: px - 20, TargetPx: px + 50}
	if err := w.st.ArmedOrders().UpsertArm(r); err != nil {
		t.Fatal(err)
	}
	if err := w.st.ArmedOrders().BeginPlacement(r.ID, sid); err != nil {
		t.Fatal(err)
	}
	if err := w.st.ArmedOrders().SetState(r.ID, store.StateFilled, fmt.Sprintf("fill@%.2f", px)); err != nil {
		t.Fatal(err)
	}
	if err := w.st.ArmedOrders().DB().Model(&store.ArmedOrderDB{}).Where("id = ?", r.ID).UpdateColumn("updated_at", time.Now().Add(-age)).Error; err != nil {
		t.Fatal(err)
	}
	return r
}

// materialize seeds NT8's LONG @px and runs the two production reconcile
// passes (sighting, then past the untracked grace).
func (w *lateFillWire) materialize(t *testing.T, px float64) *store.TraderPosition {
	t.Helper()
	w.s.SeedPositionsForTest("Sim101", []ntwire.OpenPosition{{Symbol: "MNQ", Side: "LONG", Quantity: 1, AvgPrice: px}})
	w.tr.reconcilePositions(lateFillTrader, lateFillExchange, "ninjatrader", w.st)
	w.tr.untrackedSince["MNQ|LONG"] = time.Now().UTC().UnixMilli() - untrackedGraceMs - 1
	w.tr.reconcilePositions(lateFillTrader, lateFillExchange, "ninjatrader", w.st)
	rows, err := w.st.Position().GetOpenPositions(lateFillTrader)
	if err != nil || len(rows) != 1 {
		t.Fatalf("expected one materialized row, got %d (%v)", len(rows), err)
	}
	return rows[0]
}

// A late AI fill is tagged with ITS OWN signal and its order row settles
// FILLED at the ring's real price; the plan citation stays UNRESOLVABLE
// (not recoverable here — never fabricated).
func TestLateAIFillMaterializesTaggedWithItsSignal(t *testing.T) {
	w := newLateFillWire(t)
	const sid = "sig-ai-late"
	w.aiOrderRow(t, lateFillTrader, sid, "open_long", "LONG", "BUY")
	w.fill(t, sid, "long", 29001)
	row := w.materialize(t, 29001)
	if row.EntryOrderID != sid {
		t.Fatalf("late AI fill materialized with entry_order_id=%q, want its own signal %q", row.EntryOrderID, sid)
	}
	if row.PlanID != store.PlanUnresolvable || row.PlanVersion != 0 {
		t.Fatalf("the plan citation is not recoverable here and must stay UNRESOLVABLE: plan_id=%q v%d", row.PlanID, row.PlanVersion)
	}
	o, err := w.st.Order().GetOrderByExchangeID(lateFillExchange, sid)
	if err != nil || o == nil {
		t.Fatalf("order row: %v", err)
	}
	if o.Status != "FILLED" || o.AvgFillPrice != 29001 || o.FilledQuantity != 1 {
		t.Fatalf("the signal-keyed order row must settle FILLED at the ring's price: status=%q avg=%.2f qty=%.0f", o.Status, o.AvgFillPrice, o.FilledQuantity)
	}
	if got := w.tr.resolveEntrySignalID("MNQ", "long"); got != sid {
		t.Fatalf("move_stop/trailing must address the late fill's own signal, got %q", got)
	}
}

// With exact ring evidence, a same-price arm that filled 72 h ago is NEVER
// adopted (the price-match fallback does not run).
func TestLateAIFillNeverAdoptsAnOldArmsLineage(t *testing.T) {
	w := newLateFillWire(t)
	const sid = "sig-ai-late"
	w.aiOrderRow(t, lateFillTrader, sid, "open_long", "LONG", "BUY")
	w.filledArm(t, "2026-09-01:NY", "S9", "sig-armed-days-old", 29001, 72*time.Hour)
	w.fill(t, sid, "long", 29001)
	row := w.materialize(t, 29001)
	if row.EntryOrderID != sid || row.PlanID == "2026-09-01:NY" || row.PlanVersion != 0 {
		t.Fatalf("late AI fill adopted an old arm: entry_order_id=%q plan_id=%q v%d", row.EntryOrderID, row.PlanID, row.PlanVersion)
	}
}

// An armed fill in the ring stamps lineage from ITS OWN arm — not from the
// newest same-price filled arm the price matcher would pick.
func TestLateArmedFillStampsItsOwnArmNotTheNewestSamePriceArm(t *testing.T) {
	w := newLateFillWire(t)
	mine := w.filledArm(t, "2026-09-23:NY", "S2", "sig-arm-mine", 29001, 90*time.Second)
	w.filledArm(t, "2026-09-23:NY", "S7", "sig-arm-newer", 29001, 1*time.Second)
	w.fill(t, "sig-arm-mine", "long", 29001)
	row := w.materialize(t, 29001)
	if row.EntryOrderID != "sig-arm-mine" || row.CitedScenarioID != mine.Scenario || row.PlanID != mine.PlanID {
		t.Fatalf("armed fill stamped from the wrong arm: entry_order_id=%q scenario=%q plan=%q", row.EntryOrderID, row.CitedScenarioID, row.PlanID)
	}
}

// Two same-side signals filled in the window: ambiguous. The position stays
// untagged (WARN) and the price matcher does not guess either.
func TestLateFillAmbiguousAcrossTwoSignalsStaysUntagged(t *testing.T) {
	w := newLateFillWire(t)
	w.aiOrderRow(t, lateFillTrader, "sig-ai-a", "open_long", "LONG", "BUY")
	w.aiOrderRow(t, lateFillTrader, "sig-ai-b", "open_long", "LONG", "BUY")
	w.filledArm(t, "2026-09-01:NY", "S9", "sig-armed-days-old", 29001, 72*time.Hour)
	w.fill(t, "sig-ai-a", "long", 29001)
	w.fill(t, "sig-ai-b", "long", 29001)
	row := w.materialize(t, 29001)
	if row.EntryOrderID != "" || row.PlanVersion != 0 || row.PlanID != store.PlanUnresolvable {
		t.Fatalf("an ambiguous late fill must stay untagged: entry_order_id=%q plan=%q v%d", row.EntryOrderID, row.PlanID, row.PlanVersion)
	}
	for _, sid := range []string{"sig-ai-a", "sig-ai-b"} {
		if o, _ := w.st.Order().GetOrderByExchangeID(lateFillExchange, sid); o == nil || o.Status != "NEW" {
			t.Fatalf("an ambiguous fill must not settle either order row: %s %+v", sid, o)
		}
	}
}

// A signal that is already another position row's entry_order_id is excluded:
// a tracked entry's fill inside the window never tags a second position.
func TestLateFillExcludesASignalAnotherPositionAlreadyOwns(t *testing.T) {
	w := newLateFillWire(t)
	// A tracked AI entry: filled, recorded (entry_order_id = its signal), closed.
	prior := &store.TraderPosition{TraderID: lateFillTrader, ExchangeID: lateFillExchange, ExchangeType: "ninjatrader",
		ExchangePositionID: "tracked-1", Symbol: "MNQ", Side: "LONG", Quantity: 1, EntryPrice: 28990, EntryOrderID: "sig-tracked",
		EntryTime: time.Now().UTC().UnixMilli() - 30_000, Status: "OPEN", Account: "Sim101"}
	if err := w.st.Position().CreateOpenPosition(prior); err != nil {
		t.Fatal(err)
	}
	if _, err := w.st.Position().ClosePosition(prior.ID, 28995, "x", 10, 0, "sync"); err != nil {
		t.Fatal(err)
	}
	w.aiOrderRow(t, lateFillTrader, "sig-ai-late", "open_long", "LONG", "BUY")
	w.fill(t, "sig-tracked", "long", 28990)
	w.fill(t, "sig-ai-late", "long", 29001)
	row := w.materialize(t, 29001)
	if row.EntryOrderID != "sig-ai-late" {
		t.Fatalf("the tracked entry's signal must be excluded, leaving the late fill's own: entry_order_id=%q", row.EntryOrderID)
	}
}

// A ring signal that is neither this trader's arm nor this trader's AI open is
// not ours to claim: untagged, and the price matcher does not guess.
func TestLateFillOfAnUnownedSignalStaysUntagged(t *testing.T) {
	w := newLateFillWire(t)
	w.aiOrderRow(t, "some-other-trader", "sig-foreign", "open_long", "LONG", "BUY")
	w.filledArm(t, "2026-09-01:NY", "S9", "sig-armed-days-old", 29001, 72*time.Hour)
	w.fill(t, "sig-foreign", "long", 29001)
	row := w.materialize(t, 29001)
	if row.EntryOrderID != "" || row.PlanVersion != 0 {
		t.Fatalf("an unowned signal must leave the position untagged: entry_order_id=%q v%d", row.EntryOrderID, row.PlanVersion)
	}
	if o, _ := w.st.Order().GetOrderByExchangeID(lateFillExchange, "sig-foreign"); o == nil || o.Status != "NEW" {
		t.Fatalf("another trader's order row must not be touched: %+v", o)
	}
}

// Non-regression pin: with NO same-side entry evidence in the ring (e.g. after
// a restart emptied it), today's price-match fallback still stamps.
func TestUntrackedWithNoRingEvidenceKeepsPriceMatchFallback(t *testing.T) {
	w := newLateFillWire(t)
	w.filledArm(t, "2026-09-23:NY", "S3", "sig-armed", 29001, 90*time.Second)
	w.fill(t, "sig-short-other", "short", 29050) // opposite side: not evidence for a LONG
	row := w.materialize(t, 29001)
	if row.EntryOrderID != "sig-armed" || row.PlanID != "2026-09-23:NY" || row.CitedScenarioID != "S3" {
		t.Fatalf("the price-match fallback must still stamp without ring evidence: entry_order_id=%q plan=%q scenario=%q", row.EntryOrderID, row.PlanID, row.CitedScenarioID)
	}
}
