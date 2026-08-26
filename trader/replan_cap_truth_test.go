package trader

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"nofx/store"
)

// THE PROMPT AND THE CARD MUST QUOTE THE SAME RE-PLAN CAP.
//
// installActivePlanProvider carried `replansLeft := 2 - (row.Version - 1)` with a
// literal 2, while GET /api/plan/today resolved the cap properly via
// DayPlanConfig.ReplanCapFor. On 2026-08-16 the owner raised the ASIA override
// from 2 to 4 mid-session (strategies.updated_at 22:25:11 UTC, the same second
// the log flipped from "cap 2/session" to "cap 4/session"), so at v3 the card
// said "replans left 2" and the executor prompt said 0. Two rulebooks.

func seedTraderWithDayPlan(t *testing.T, st *store.Store, traderID, strategyID string, cfg store.StrategyConfig) {
	t.Helper()
	raw, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := st.GormDB().Create(&store.Strategy{
		ID: strategyID, Name: "t", Config: string(raw),
	}).Error; err != nil {
		t.Fatal(err)
	}
	if err := st.GormDB().Create(&store.Trader{
		ID: traderID, Name: "t", StrategyID: strategyID,
	}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestStoredReplanCapReadsTheSessionOverride(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	four := 4
	cfg := store.StrategyConfig{
		StrategyType: "ai_trading",
		DayPlan: &store.DayPlanConfig{
			PlanEnabled: true,
			ReplanCap:   2, // strategy level
			Sessions: []store.DayPlanSessionOverride{
				{Session: "ASIA", ReplanCap: &four}, // the owner's mid-session edit
			},
		},
	}
	seedTraderWithDayPlan(t, st, "trader-1", "strat-1", cfg)

	if got := storedReplanCap(st, "trader-1", "ASIA"); got != 4 {
		t.Errorf("ASIA cap = %d, want the session override 4 — the prompt would understate replans left", got)
	}
	if got := storedReplanCap(st, "trader-1", "NY"); got != 2 {
		t.Errorf("NY cap = %d, want the strategy-level 2", got)
	}
}

func TestStoredReplanCapFallsBackSafely(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	// Every failure path must yield the shipped default rather than 0, which would
	// tell the AI it has no re-plans left when in fact it has the default budget.
	var dp *store.DayPlanConfig
	want := dp.ReplanCapFor("ASIA")
	for _, tc := range []struct {
		name     string
		st       *store.Store
		traderID string
	}{
		{"nil store", nil, "trader-1"},
		{"empty trader id", st, ""},
		{"unknown trader", st, "does-not-exist"},
	} {
		if got := storedReplanCap(tc.st, tc.traderID, "ASIA"); got != want {
			t.Errorf("%s: cap = %d, want the shipped default %d", tc.name, got, want)
		}
	}
}

// ITEM 4 (2026-08-17) — a re-plan CARRIES the owner's levels forward.
//
// This test previously pinned the interim behaviour: a P1 alert naming how many
// edits a re-plan was about to strand. That was the honest stopgap while the
// rebase was unbuilt. The edits are now re-established by price identity, so the
// contract it guarded is superseded — what must hold is that the carry actually
// writes the owner's level onto the new version.
func TestReplanCarriesOwnerLevelsForward(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	cfg := store.StrategyConfig{
		StrategyType: "ai_trading",
		DayPlan:      &store.DayPlanConfig{PlanEnabled: true},
	}
	seedTraderWithDayPlan(t, st, "trader-1", "strat-1", cfg)
	at := &AutoTrader{id: "trader-1", store: st, exchange: "ninjatrader"}
	at.config.StrategyConfig = &cfg

	planID := store.MakePlanID("2026-08-16", "ASIA")
	base := `{"reasoning":"r","bias":{"direction":"long","conviction":"low","flip_condition":"n/a"},` +
		`"levels":[{"price":30200,"label":"ONH","grade":"A","instruction":"fade"}],` +
		`"scenarios":[{"id":"S1","trigger":"t","condition":"hold","direction":"long","invalid":"n","quality":"A"}],` +
		`"no_trade":[],"death_condition":"x"}`
	if _, err := st.Plan().AppendPlan(&store.PlanDB{
		PlanID: planID, StrategyID: "trader-1", TradeDate: "2026-08-16",
		Session: "ASIA", Lifecycle: "active", Doc: base,
	}); err != nil {
		t.Fatal(err)
	}
	// The owner adds a level of their own to v1.
	if _, err := st.Plan().AppendOverlay(&store.PlanOverlayDB{
		PlanID: planID, PlanVersion: 1, Origin: "owner",
		Patch: `[{"op":"add","path":"/levels/-","value":{"price":30250,"label":"my line","grade":"A","instruction":"reclaim-long"}}]`,
	}); err != nil {
		t.Fatal(err)
	}
	// A re-plan writes v2 WITHOUT that level.
	if _, err := st.Plan().AppendPlan(&store.PlanDB{
		PlanID: planID, StrategyID: "trader-1", TradeDate: "2026-08-16",
		Session: "ASIA", Lifecycle: "active", Doc: base,
	}); err != nil {
		t.Fatal(err)
	}

	at.carryOwnerEditsInto(planID, 1, 2)

	overlays, err := st.Plan().ListOverlays(planID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(overlays) == 0 {
		t.Fatal("the owner's level did not carry into v2 — it was stranded, which is the whole bug")
	}
	found := false
	for _, ov := range overlays {
		if strings.Contains(ov.Patch, "30250") && strings.Contains(ov.Patch, "my line") {
			found = true
			if !strings.Contains(ov.Patch, `"path":"/levels/-"`) {
				t.Errorf("the carry must APPEND by identity, not re-point an index: %s", ov.Patch)
			}
		}
	}
	if !found {
		t.Errorf("v2 overlays do not contain the owner's level: %+v", overlays)
	}
}

// Nothing to carry → no overlay written, no noise.
func TestReplanWithNoOwnerEditsCarriesNothing(t *testing.T) {
	st, err := store.New(filepath.Join(t.TempDir(), "t2.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	cfg := store.StrategyConfig{DayPlan: &store.DayPlanConfig{PlanEnabled: true}}
	seedTraderWithDayPlan(t, st, "trader-1", "strat-1", cfg)
	at := &AutoTrader{id: "trader-1", store: st, exchange: "ninjatrader"}
	at.config.StrategyConfig = &cfg

	planID := store.MakePlanID("2026-08-16", "NY")
	for i := 0; i < 2; i++ {
		if _, err := st.Plan().AppendPlan(&store.PlanDB{
			PlanID: planID, StrategyID: "trader-1", TradeDate: "2026-08-16",
			Session: "NY", Lifecycle: "active", Doc: "{}",
		}); err != nil {
			t.Fatal(err)
		}
	}
	at.carryOwnerEditsInto(planID, 1, 2)
	if ovs, _ := st.Plan().ListOverlays(planID, 2); len(ovs) != 0 {
		t.Errorf("a clean re-plan must write no overlay, got %d", len(ovs))
	}
}
