package kernel

import (
	"strings"
	"testing"

	"nofx/store"
	"nofx/telemetry"
)

// TestExecHoldPathsCountGateBlocks pins P2-7 at the PRODUCTION call site:
// GetFullDecisionWithStrategy's two HOLD branches
// (executor_plan_gate, schema_parse_failed) must land in the gate-block
// table that /api/risk/gate-blocks serves — a cycle-holding gate that is not
// counted is invisible to the operator.
func TestExecHoldPathsCountGateBlocks(t *testing.T) {
	trader := "p27-holder"
	ctx := &Context{
		TraderID:    trader,
		Account:     AccountInfo{TotalEquity: 60000},
		OITopDataMap: map[string]*OITopData{},
	}
	engine := &StrategyEngine{config: &store.StrategyConfig{
		RiskControl: store.RiskControlConfig{MaxPositions: 1},
	}}

	// Case 1 — schema_parse_failed: three garbage replies exhaust the retry
	// loop; the cycle HOLDs and the counter must record the class.
	garbage := &mockAIClient{replies: []string{"{not json", "still not json", "nope"}}
	fd, err := GetFullDecisionWithStrategy(ctx, garbage, engine, "futures")
	if err != nil {
		t.Fatalf("a parse HOLD must not error: %v", err)
	}
	if fd == nil || fd.SkipReason != "schema_parse_failed" {
		t.Fatalf("want SkipReason schema_parse_failed, got %+v", fd)
	}
	if !gateBlockCountHas(t, trader, "schema_parse_failed") {
		t.Fatalf("schema_parse_failed HOLD not counted in gate-block table")
	}

	// Case 2 — executor_plan_gate: a VALID open decision against a dead plan
	// must HOLD with the typed refusal and count its class.
	ctx.ExecutorPlanDead = "plan machine-dead (test)"
	valid := &mockAIClient{replies: []string{
		`{"action":"open_long","symbol":"MNQ","entry":30000,"stop_loss":29900,"take_profit":30200,"reasoning":"test","leverage":1,"confidence":80,"position_size_usd":60000}`,
	}}
	fd2, err2 := GetFullDecisionWithStrategy(ctx, valid, engine, "futures")
	if err2 != nil {
		t.Fatalf("a gate HOLD must not error: %v", err2)
	}
	if fd2 == nil || !strings.HasPrefix(fd2.SkipReason, "executor_plan_gate:") {
		t.Fatalf("want SkipReason executor_plan_gate:…, got %+v", fd2)
	}
	if !gateBlockCountHas(t, trader, "executor_plan_gate") {
		t.Fatalf("executor_plan_gate HOLD not counted in gate-block table")
	}
}

func gateBlockCountHas(t *testing.T, trader, class string) bool {
	t.Helper()
	_, table := telemetry.GateBlockSnapshot()
	if table == nil {
		return false
	}
	n := table[trader][class]
	return n > 0
}
