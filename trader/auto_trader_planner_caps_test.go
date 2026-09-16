package trader

import (
	"fmt"
	"strings"
	"testing"
)

// H4/H5 — the WRITE path must honor the owner's max_levels/scenario_cap. Before
// this, a plan the AI emitted with 9–12 levels or 4–5 scenarios was rejected by a
// hardcoded 8/3 validator and the whole read fail-closed into NO-TRADE + P0
// alert — the UI's upper half was unreachable. Prove the seam end to end through
// runPlannerReadCore.

func widePlanJSON(levels, scenarios int) string {
	var lv strings.Builder
	for i := 0; i < levels; i++ {
		if i > 0 {
			lv.WriteString(",")
		}
		fmt.Fprintf(&lv, `{"price": %d, "label": "PDH", "grade": "A", "instruction": "fade"}`, 15000+i*10)
	}
	var sc strings.Builder
	for i := 0; i < scenarios; i++ {
		if i > 0 {
			sc.WriteString(",")
		}
		fmt.Fprintf(&sc, `{"id": "S%d", "trigger": "touch 15000", "condition": "reject", "direction": "long", "target_chain": [15100], "invalid": "n/a", "quality": "B", "confirm":{"rule":"touch","ref_price":15000,"side":"above"},"economics":{"entry_zone":[15000,15000],"geometry":{"entry":15000,"stop":14990,"target":15100},"first_obstacle":{"price":15100,"level":"fixture reference","family":"reference","response":"pass_through"},"r_to_obstacle":10.0,"r_to_arm_target":10.0}}`, i+1)
	}
	return fmt.Sprintf(`{"reasoning": "ok", "bias": {"direction": "neutral", "conviction": "low", "flip_condition": "n/a"}, "levels": [%s], "scenarios": [%s], "no_trade": [], "death_condition": "n/a"}`, lv.String(), sc.String())
}

func TestRunPlannerReadCoreHonorsRaisedCaps(t *testing.T) {
	st := plannerTestTrader(t)
	st.config.StrategyConfig.DayPlan.MaxLevels = 12
	st.config.StrategyConfig.DayPlan.ScenarioCap = 5

	ver, lc, err := st.runPlannerReadCore("NY", "2026-08-14", "deepseek-reasoner", "hashWide", "", "",
		func() (string, error) { return widePlanJSON(12, 5), nil })
	if err != nil || ver != 1 || lc != "active" {
		t.Fatalf("12 levels/5 scenarios with raised caps: ver=%d lc=%q err=%v (must be ACTIVE, not fail-closed)", ver, lc, err)
	}
	got, _ := st.store.Plan().GetLatestPlanForSession("2026-08-14", "NY")
	if got == nil || got.Lifecycle != "active" {
		t.Fatalf("stored plan must be active, got %+v", got)
	}
}

func TestRunPlannerReadCoreShippedCapsStillFailClosed(t *testing.T) {
	st := plannerTestTrader(t) // defaults: max_levels 8, scenario_cap 3

	ver, lc, err := st.runPlannerReadCore("NY", "2026-08-14", "deepseek-reasoner", "hashNarrow", "", "",
		func() (string, error) { return widePlanJSON(9, 3), nil })
	if err != nil {
		t.Fatalf("fail-closed must not be an error: %v", err)
	}
	if lc != "no_trade" || ver == 0 {
		t.Fatalf("9 levels at the shipped cap must fail-closed into a NO-TRADE plan, got lc=%q ver=%d", lc, ver)
	}
}
