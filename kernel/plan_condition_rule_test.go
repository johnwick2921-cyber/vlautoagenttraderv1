package kernel

import (
	"strings"
	"testing"

	"nofx/market"
)

// A2 (fail-register wave) — the rule evaluates AS AUTHORED (anatomy FAIL F4:
// 5m_close silently became 2x5m).
func TestConditionRuleAsAuthored(t *testing.T) {
	if got := conditionRule(PlanCondition{Rule: "5m_close"}); got != "5m-close" {
		t.Fatalf("5m_close maps to %q, want 5m-close (one close, as authored)", got)
	}
	if acceptanceNeed("5m-close") != 1 || acceptanceTFMinutes("5m-close") != 5 {
		t.Fatal("5m-close must need exactly ONE 5-minute close")
	}
	// E1 (entry-mechanics 2026-08-30): 15m AUTHORSHIP is dead (schema rejects
	// condition_rule_15m_removed) — the mapping below is LEGACY evaluation
	// tolerance so stored pre-entry-mechanics docs keep evaluating.
	if got := conditionRule(PlanCondition{Rule: "15m_close"}); got != "15m-close" || acceptanceNeed(got) != 1 || acceptanceTFMinutes(got) != 15 {
		t.Fatal("legacy 15m_close evaluation semantics changed")
	}
	if got := conditionRule(PlanCondition{Rule: "2x5m"}); got != "2x5m" || acceptanceNeed(got) != 2 || acceptanceTFMinutes(got) != 5 {
		t.Fatal("2x5m semantics changed")
	}
}

// PLAN-LIFECYCLE WAVE (2026-08-27) — OLD: a 5m_close death fired on ONE close.
// NEW: FLIP_CONFIRM_CLOSES (default 2) floors every structured death/flip, so
// one 5m close beyond does NOT fire and two consecutive do (anti-whipsaw).
func TestFiveMCloseDeathNeedsConfirmFloor(t *testing.T) {
	c := PlanCondition{Price: 100, Side: "below", Rule: "5m_close"}
	mk := func(openMs int64, h, l, cl float64) market.Kline {
		return market.Kline{OpenTime: openMs, CloseTime: openMs + 59_999, High: h, Low: l, Close: cl, Open: cl}
	}
	base := int64(1_700_000_000_000) - (int64(1_700_000_000_000) % 300_000) // 5m-aligned
	var bars []market.Kline
	for i := int64(0); i < 5; i++ { // bucket 1 (touch, closes above)
		bars = append(bars, mk(base+i*60_000, 101, 99.9, 100.5))
	}
	for i := int64(5); i < 10; i++ { // bucket 2 closes below
		bars = append(bars, mk(base+i*60_000, 100.2, 99.0, 99.2))
	}
	if fired, reason := PlanConditionFiredSince(c, bars, base-1, base+10*60_000); fired {
		t.Fatalf("NEW: one 5m close must NOT fire under the confirm floor (reason=%q)", reason)
	}
	// second below-bucket → fires, reason names the authored rule.
	for i := int64(10); i < 15; i++ {
		bars = append(bars, mk(base+i*60_000, 100.2, 99.0, 99.2))
	}
	fired, reason := PlanConditionFiredSince(c, bars, base-1, base+15*60_000)
	if !fired {
		t.Fatalf("two consecutive closes must fire (reason=%q)", reason)
	}
	if !strings.Contains(reason, "5m_close") {
		t.Fatalf("reason %q must name the authored rule", reason)
	}
}
