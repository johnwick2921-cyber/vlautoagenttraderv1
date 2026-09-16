package trader

import (
	"path/filepath"
	"strings"
	"testing"

	"nofx/kernel"
	ntwire "nofx/provider/ninjatrader"
	"nofx/store"
)

// Class 27 FIX 4 (one-live-arm guard) + FIX 5 (split-leg capacity) tests.

func class27GuardTrader(t *testing.T) (*AutoTrader, *store.Store) {
	t.Helper()
	st, err := store.New(filepath.Join(t.TempDir(), "guard.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	at := &AutoTrader{id: "td1", exchange: "ninjatrader", store: st}
	at.config.Exchange = "ninjatrader"
	at.config.StrategyConfig = &store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	return at, st
}

// SUPERSEDED SPEC (owner ruling 2026-09-03 — ONE OPEN POSITION PER
// INSTRUMENT). This asserted that a SAME-side arm "is outside this guard's
// scope → pass". That scope gap is how a new plan version could re-authorize a
// terminal row and ADD to a position that was still open: the same-version
// re-arm block lives in the store, and nothing refused the cross-version case.
// Both sides are refused now. The opposite-side half of the assertion is
// unchanged.
func TestOneLiveArmGuardRefusesOppositeSide(t *testing.T) {
	at, _ := class27GuardTrader(t)
	openAPosition(t, at.store, at.id) // helper opens MNQ LONG

	verdict := at.oneLiveArmGuard(kernel.PlanScenario{ID: "S3"}, kernel.PlanArmLeg{Entry: 29459}, "short")
	if verdict == "" {
		t.Fatal("opposite-side arm while LONG is open must be refused")
	}
	// And the same side too, now.
	v := at.oneLiveArmGuard(kernel.PlanScenario{ID: "S3"}, kernel.PlanArmLeg{Entry: 29417}, "long")
	if v == "" {
		t.Fatal("a SAME-side arm while a position is open must also be refused (one_open_position)")
	}
	if !strings.Contains(v, "no adds, no flips") {
		t.Errorf("the refusal must say why: %q", v)
	}
}

func TestOneLiveArmGuardExitLegAndFlatPass(t *testing.T) {
	at, st := class27GuardTrader(t)
	openAPosition(t, st, at.id)

	// An explicitly authored exit/flip leg is the escape hatch.
	if v := at.oneLiveArmGuard(kernel.PlanScenario{ID: "S3"}, kernel.PlanArmLeg{Entry: 29459, Kind: "exit"}, "short"); v != "" {
		t.Fatalf("kind=exit leg must pass, got %q", v)
	}

	// Flat account → nothing to net → pass.
	rows, _ := st.Position().GetOpenPositions(at.id)
	for _, r := range rows {
		_, _ = st.Position().ClosePosition(r.ID, r.EntryPrice, "test", 0, 0, "sync")
	}
	if v := at.oneLiveArmGuard(kernel.PlanScenario{ID: "S3"}, kernel.PlanArmLeg{Entry: 29459}, "short"); v != "" {
		t.Fatalf("flat account must pass the guard, got %q", v)
	}
}

func TestSplitLegCapacity(t *testing.T) {
	if c := splitLegCapacity(0); c != 1 {
		t.Fatalf("unset → capacity 1 (netting-safe default), got %d", c)
	}
	if c := splitLegCapacity(2); c != 2 {
		t.Fatalf("explicit 2 → capacity 2, got %d", c)
	}
	if c := splitLegCapacity(-1); c != 1 {
		t.Fatalf("invalid → capacity 1, got %d", c)
	}
}

func TestArmLegCapacityUsesStrategyValue(t *testing.T) {
	at, _ := class27GuardTrader(t)
	at.config.StrategyConfig.RiskControl.MaxContractsPerOrder = 3
	if c := at.armLegCapacity(); c != 3 {
		t.Fatalf("explicit max_contracts_per_order=3 → capacity 3, got %d", c)
	}
}

// TestMaterializeArmedEntryDedupesCaseInsensitive (class 27 FIX 3): a legacy
// lowercase-side open row must block the armed materializer (the 577+578 class
// — the old dedupe queried the lowercase ledger side against the uppercase
// store convention and missed).
func TestMaterializeArmedEntryDedupesCaseInsensitive(t *testing.T) {
	at, st := class27GuardTrader(t)
	// Legacy lowercase-side row for the same fill (pre-FIX3 shape).
	if err := st.Position().CreateOpenPosition(&store.TraderPosition{
		TraderID: "td1", ExchangeType: "ninjatrader",
		ExchangePositionID: "armed_old_1", Symbol: "MNQ", Side: "long",
		Quantity: 1, EntryQuantity: 1, EntryPrice: 29413.0,
		EntryOrderID: "sig-old", Leverage: 1, Status: "OPEN",
		Source: "armed_entry", Account: "Sim101",
	}); err != nil {
		t.Fatal(err)
	}
	at.materializeArmedEntry(store.ArmedOrderDB{
		TraderID: "td1", Scenario: "S1", Side: "long", SignalID: "sig-old",
		PlanID: "2026-08-31:NY:td1",
	}, ntwire.OrderUpdatePayload{FillPrice: 29413.0, Account: "Sim101"})
	// No duplicate may appear.
	rows, _ := st.Position().GetOpenPositions(at.id)
	if len(rows) != 1 {
		t.Fatalf("materializeArmedEntry must dedupe the legacy lowercase row, got %d open rows", len(rows))
	}
}
