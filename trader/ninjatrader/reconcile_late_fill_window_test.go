package ninjatrader

import (
	"testing"
	"time"

	"nofx/store"
)

// ── W1b FOLD-4 — the price-match fallback is bounded by the fill ring's own
// window ───────────────────────────────────────────────────────────────────
//
// When the ring holds no same-side evidence in the window — it emptied on a
// restart, or every candidate already explains another position — reconcile's
// untracked materialization falls back to a PRICE match against this trader's
// FILLED arms. Before FOLD-4 that match had no time bound (ListFilled(20), ±1
// tick, any age): a fill at an old arm's price adopted that arm's plan and
// signal, rememberEntryOrderID cached a TERMINAL signal, and move_stop went to
// a dead order. The fallback now considers only arms whose updated_at is at or
// after firstSeen − lateEntryFillWindowMs (the ring's window); an older arm
// never matches. updated_at is zone-bearing text (store.LedgerClockSlack), so
// the window is judged on the parsed INSTANT, never on the string.

// assertUntaggedNotAdopted is the FOLD-4 verdict on one materialized row: no
// entry identity, no plan linkage, no cached signal, and the ineligible arm
// untouched.
func assertUntaggedNotAdopted(t *testing.T, w *lateFillWire, row *store.TraderPosition, arm *store.ArmedOrderDB) {
	t.Helper()
	if row.EntryOrderID != "" || row.PlanID != store.PlanUnresolvable || row.PlanVersion != 0 || row.CitedScenarioID != "" {
		t.Fatalf("an arm the fallback may not consider (older than the ring's window, or its signal already explains another position) was adopted: entry_order_id=%q plan_id=%q v%d scenario=%q",
			row.EntryOrderID, row.PlanID, row.PlanVersion, row.CitedScenarioID)
	}
	w.tr.mu.Lock()
	cached := w.tr.entryOrderID[keyFor("MNQ", "LONG")]
	w.tr.mu.Unlock()
	if cached != "" {
		t.Fatalf("rememberEntryOrderID cached the old arm's terminal signal %q — move_stop would address a dead order", cached)
	}
	if got := w.tr.resolveEntrySignalID("MNQ", "long"); got != "" {
		t.Fatalf("move_stop/trailing resolves %q for an untagged position, want none", got)
	}
	got, err := w.st.ArmedOrders().FindBySignal(lateFillTrader, arm.SignalID)
	if err != nil || got == nil {
		t.Fatalf("old arm re-read: %v", err)
	}
	if got.FillQuantity != 0 {
		t.Fatalf("the old arm was stamped as this position's fill (fill_quantity=%d)", got.FillQuantity)
	}
}

// RED (FOLD-4): the ring emptied by a restart, yesterday's arm filled at the
// same price → the position stays untagged and nothing is cached.
func TestPriceMatchFallbackNeverAdoptsYesterdaysArm(t *testing.T) {
	w := newLateFillWire(t)
	arm := w.filledArm(t, "2026-09-22:NY", "S1", "sig-armed-yesterday", 29001, 24*time.Hour)
	arm.SignalID = "sig-armed-yesterday"
	row := w.materialize(t, 29001)
	assertUntaggedNotAdopted(t, w, row, arm)
}

// RED (FOLD-4): the CTO's path — every ring candidate is already in use (the
// same-side fill in the window explains another position), so the verdict is
// no-evidence and the fallback runs; it still never reaches yesterday's arm.
func TestPriceMatchFallbackAfterAllRingCandidatesInUseIsTimeBound(t *testing.T) {
	w := newLateFillWire(t)
	prior := &store.TraderPosition{TraderID: lateFillTrader, ExchangeID: lateFillExchange, ExchangeType: "ninjatrader",
		ExchangePositionID: "tracked-1", Symbol: "MNQ", Side: "LONG", Quantity: 1, EntryPrice: 29001, EntryOrderID: "sig-tracked",
		EntryTime: time.Now().UTC().UnixMilli() - 30_000, Status: "OPEN", Account: "Sim101"}
	if err := w.st.Position().CreateOpenPosition(prior); err != nil {
		t.Fatal(err)
	}
	if _, err := w.st.Position().ClosePosition(prior.ID, 29005, "x", 10, 0, "sync"); err != nil {
		t.Fatal(err)
	}
	arm := w.filledArm(t, "2026-09-22:NY", "S1", "sig-armed-yesterday", 29001, 24*time.Hour)
	arm.SignalID = "sig-armed-yesterday"
	w.fill(t, "sig-tracked", "long", 29001)
	row := w.materialize(t, 29001)
	assertUntaggedNotAdopted(t, w, row, arm)
}

// The window's edge: an arm filled just before firstSeen − lateEntryFillWindowMs
// never matches; one just inside still stamps (the fallback is bounded, not
// removed). firstSeen = the materialize helper's untrackedSince seed.
func TestPriceMatchFallbackWindowEdge(t *testing.T) {
	const margin = 10 * time.Second
	window := time.Duration(lateEntryFillWindowMs) * time.Millisecond
	sinceFirstSeen := time.Duration(untrackedGraceMs) * time.Millisecond // materialize seeds firstSeen this far back
	t.Run("outside", func(t *testing.T) {
		w := newLateFillWire(t)
		arm := w.filledArm(t, "2026-09-23:NY", "S4", "sig-armed-edge-out", 29001, sinceFirstSeen+window+margin)
		arm.SignalID = "sig-armed-edge-out"
		row := w.materialize(t, 29001)
		assertUntaggedNotAdopted(t, w, row, arm)
	})
	t.Run("inside", func(t *testing.T) {
		w := newLateFillWire(t)
		w.filledArm(t, "2026-09-23:NY", "S4", "sig-armed-edge-in", 29001, sinceFirstSeen+window-margin)
		row := w.materialize(t, 29001)
		if row.EntryOrderID != "sig-armed-edge-in" || row.PlanID != "2026-09-23:NY" || row.CitedScenarioID != "S4" {
			t.Fatalf("an arm filled inside the ring's window must still stamp: entry_order_id=%q plan=%q scenario=%q",
				row.EntryOrderID, row.PlanID, row.CitedScenarioID)
		}
	})
}

// updated_at is zone-bearing TEXT written by writers in different zones. The
// window is judged on the parsed instant: an old arm whose text sorts AFTER the
// bound (written in +14:00) never matches, and a fresh arm whose text sorts
// BEFORE it (written in −12:00) still stamps.
func TestPriceMatchFallbackComparesInstantsNotZoneText(t *testing.T) {
	const layout = "2006-01-02 15:04:05.999999999-07:00"
	setText := func(t *testing.T, w *lateFillWire, id int64, text string) {
		t.Helper()
		if err := w.st.GormDB().Exec("UPDATE armed_orders SET updated_at = ? WHERE id = ?", text, id).Error; err != nil {
			t.Fatal(err)
		}
	}
	// The earliest bound the fallback can use in these tests (firstSeen is
	// seeded untrackedGraceMs back; the window reaches lateEntryFillWindowMs
	// further), in both zones a query could format it in.
	bound := time.Now().Add(-time.Duration(untrackedGraceMs+lateEntryFillWindowMs) * time.Millisecond)

	t.Run("old arm, text sorts after the bound", func(t *testing.T) {
		w := newLateFillWire(t)
		arm := w.filledArm(t, "2026-09-23:ASIA", "S2", "sig-armed-east", 29001, 0)
		arm.SignalID = "sig-armed-east"
		oldEast := time.Now().Add(-10 * time.Hour).In(time.FixedZone("E14", 14*3600)).Format(layout)
		if oldEast <= bound.UTC().Format(layout) || oldEast <= bound.Format(layout) {
			t.Fatalf("fixture: %q must sort after the bound %q / %q", oldEast, bound.UTC().Format(layout), bound.Format(layout))
		}
		setText(t, w, arm.ID, oldEast)
		row := w.materialize(t, 29001)
		assertUntaggedNotAdopted(t, w, row, arm)
	})
	t.Run("fresh arm, text sorts before the bound", func(t *testing.T) {
		w := newLateFillWire(t)
		arm := w.filledArm(t, "2026-09-23:NY", "S5", "sig-armed-west", 29001, 0)
		freshWest := time.Now().Add(-90 * time.Second).In(time.FixedZone("W12", -12*3600)).Format(layout)
		if freshWest >= bound.UTC().Format(layout) || freshWest >= bound.Format(layout) {
			t.Fatalf("fixture: %q must sort before the bound %q / %q", freshWest, bound.UTC().Format(layout), bound.Format(layout))
		}
		setText(t, w, arm.ID, freshWest)
		row := w.materialize(t, 29001)
		if row.EntryOrderID != "sig-armed-west" || row.CitedScenarioID != "S5" {
			t.Fatalf("a fresh arm written in another zone must still stamp: entry_order_id=%q scenario=%q", row.EntryOrderID, row.CitedScenarioID)
		}
	})
}

// RED (FOLD-4 finish): the CTO's own path with a FRESH arm. The ring's only
// same-side fill is arm X's signal, and X already explains another position
// (tracked, then closed inside the window), so the verdict is no-evidence and
// the fallback runs. X is FILLED inside the window at the same price — the time
// bound alone lets it through, and the second position adopted X's plan and
// cached X's terminal signal (probe [A] at 95ad925a: entry_order_id="sig-arm-x"
// plan="2026-09-23:NY" cached="sig-arm-x"). E15's invariant binds the fallback
// too: a signal that already explains one position never tags a second.
func TestPriceMatchFallbackNeverReusesAnArmWhoseSignalExplainsAnotherPosition(t *testing.T) {
	w := newLateFillWire(t)
	arm := w.filledArm(t, "2026-09-23:NY", "S1", "sig-arm-x", 29001, 30*time.Second)
	arm.SignalID = "sig-arm-x"
	prior := &store.TraderPosition{TraderID: lateFillTrader, ExchangeID: lateFillExchange, ExchangeType: "ninjatrader",
		ExchangePositionID: "tracked-x", Symbol: "MNQ", Side: "LONG", Quantity: 1, EntryPrice: 29001, EntryOrderID: "sig-arm-x",
		EntryTime: time.Now().UTC().UnixMilli() - 30_000, Status: "OPEN", Account: "Sim101"}
	if err := w.st.Position().CreateOpenPosition(prior); err != nil {
		t.Fatal(err)
	}
	if _, err := w.st.Position().ClosePosition(prior.ID, 28990, "x", -10, 0, "sync"); err != nil {
		t.Fatal(err)
	}
	w.fill(t, "sig-arm-x", "long", 29001)
	row := w.materialize(t, 29001)
	assertUntaggedNotAdopted(t, w, row, arm)
}

// The in-use exclusion is per ARM, not a blanket refusal: with X in use and a
// second, unclaimed in-window arm at the same price, the fallback stamps the
// unclaimed one.
func TestPriceMatchFallbackSkipsTheInUseArmForAnUnclaimedOne(t *testing.T) {
	w := newLateFillWire(t)
	w.filledArm(t, "2026-09-23:NY", "S2", "sig-arm-free", 29001, 60*time.Second)
	w.filledArm(t, "2026-09-23:NY", "S1", "sig-arm-x", 29001, 30*time.Second) // newer: the matcher's first pick
	prior := &store.TraderPosition{TraderID: lateFillTrader, ExchangeID: lateFillExchange, ExchangeType: "ninjatrader",
		ExchangePositionID: "tracked-x", Symbol: "MNQ", Side: "LONG", Quantity: 1, EntryPrice: 29001, EntryOrderID: "sig-arm-x",
		EntryTime: time.Now().UTC().UnixMilli() - 30_000, Status: "OPEN", Account: "Sim101"}
	if err := w.st.Position().CreateOpenPosition(prior); err != nil {
		t.Fatal(err)
	}
	if _, err := w.st.Position().ClosePosition(prior.ID, 28990, "x", -10, 0, "sync"); err != nil {
		t.Fatal(err)
	}
	row := w.materialize(t, 29001)
	if row.EntryOrderID != "sig-arm-free" || row.CitedScenarioID != "S2" {
		t.Fatalf("the unclaimed in-window arm must stamp (the in-use one skipped): entry_order_id=%q scenario=%q", row.EntryOrderID, row.CitedScenarioID)
	}
}
