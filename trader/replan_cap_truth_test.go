package trader

import (
	"encoding/json"
	"path/filepath"
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

// A re-plan silently orphans every owner overlay attached to the outgoing
// version (they are keyed (plan_id, plan_version) and every reader resolves
// against the LATEST). The rebase is sized M and deliberately deferred; what
// must not happen is the loss being SILENT.
func TestReplanOrphanWarningFiresOnlyWhenOverlaysExist(t *testing.T) {
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

	row := &store.PlanDB{
		PlanID: store.MakePlanID("2026-08-16", "ASIA"), StrategyID: "trader-1",
		TradeDate: "2026-08-16", Session: "ASIA", Lifecycle: "active", Doc: `{}`,
	}
	if _, err := st.Plan().AppendPlan(row); err != nil {
		t.Fatal(err)
	}

	countAlerts := func() int {
		alerts, _ := st.Alert().List("trader-1", 50)
		n := 0
		for _, a := range alerts {
			if a.Kind == "overlays-orphaned" {
				n++
			}
		}
		return n
	}

	// No overlays → nothing to warn about; a re-plan on a clean plan is normal.
	at.warnIfReplanOrphansOverlays(row)
	if got := countAlerts(); got != 0 {
		t.Fatalf("warned %d times with zero overlays — a clean re-plan must stay quiet", got)
	}

	if _, err := st.Plan().AppendOverlay(&store.PlanOverlayDB{
		PlanID: row.PlanID, PlanVersion: row.Version,
		Patch: `[{"op":"add","path":"/levels/-","value":{}}]`, Origin: "owner",
	}); err != nil {
		t.Fatal(err)
	}
	at.warnIfReplanOrphansOverlays(row)
	if got := countAlerts(); got != 1 {
		t.Fatalf("owner edits = %d alerts, want exactly 1 — the loss must not be silent", got)
	}
}
