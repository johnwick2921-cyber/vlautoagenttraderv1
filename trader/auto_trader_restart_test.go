package trader

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

// TestPlanRestartRecovery is the P3.6 MANDATORY mid-session-restart fixture: the
// active plan survives in sqlite and the scenario states recompute IDENTICALLY
// after a restart (they are a pure function of the plan + bars — no in-memory
// state to lose).
func TestPlanRestartRecovery(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "t.db")

	const fiveMin = 5 * 60 * 1000
	mk := func(i int, o, h, l, c float64) market.Kline {
		open := int64(i) * fiveMin
		return market.Kline{OpenTime: open, Open: o, High: h, Low: l, Close: c, CloseTime: open + fiveMin - 1}
	}
	bars := []market.Kline{
		mk(0, 15590, 15600, 15580, 15595),
		mk(1, 15595, 15625, 15590, 15610),
		mk(2, 15610, 15615, 15570, 15600),
	}
	nowMs := bars[len(bars)-1].CloseTime + 1
	const price, dATR, rule = 15600.0, 120.0, "2x5m"
	doc := kernel.PlanDoc{
		Reasoning:      "r", Bias: kernel.PlanBias{Direction: "long"},
		Levels:         []kernel.PlanLevel{{Price: 15620, Label: "PDH", Grade: "A"}, {Price: 15480, Label: "ONL", Grade: "B"}},
		Scenarios:      []kernel.PlanScenario{{ID: "S1", Condition: "hold", Direction: "long", Quality: "A"}},
		DeathCondition: "d",
	}
	docJSON, _ := json.Marshal(doc)

	// --- Session 1: write the plan + compute the status.
	st1, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("open st1: %v", err)
	}
	if _, err := st1.Plan().AppendPlan(&store.PlanDB{
		PlanID: "2026-08-14:NY", StrategyID: "t1", TradeDate: "2026-08-14", Session: "NY",
		Lifecycle: "active", ModelID: "m", Doc: string(docJSON),
	}); err != nil {
		t.Fatalf("append plan: %v", err)
	}
	status1 := kernel.RenderPlanStatus(doc, bars, price, dATR, rule, 2, nowMs)
	st1.Plan().Close()
	_ = st1.Close()

	// --- MID-SESSION RESTART: reopen the same DB, reload the plan, recompute.
	st2, err := store.New(dbPath)
	if err != nil {
		t.Fatalf("open st2: %v", err)
	}
	t.Cleanup(func() { st2.Plan().Close(); _ = st2.Close() })

	row, err := st2.Plan().GetLatestPlanForSession("2026-08-14", "NY")
	if err != nil || row == nil {
		t.Fatalf("plan lost across restart: %v", err)
	}
	if row.Lifecycle != "active" || row.ModelID != "m" {
		t.Fatalf("plan metadata changed across restart: %+v", row)
	}
	var reDoc kernel.PlanDoc
	if err := json.Unmarshal([]byte(row.Doc), &reDoc); err != nil {
		t.Fatalf("reload doc: %v", err)
	}
	status2 := kernel.RenderPlanStatus(reDoc, bars, price, dATR, rule, 2, nowMs)

	if status1 != status2 {
		t.Fatalf("scenario states DIVERGED across restart:\n--- before ---\n%s\n--- after ---\n%s", status1, status2)
	}
}
