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
	// Regressions: the two existing rules unchanged.
	if got := conditionRule(PlanCondition{Rule: "15m_close"}); got != "15m-close" || acceptanceNeed(got) != 1 || acceptanceTFMinutes(got) != 15 {
		t.Fatal("15m_close semantics changed")
	}
	if got := conditionRule(PlanCondition{Rule: "2x5m"}); got != "2x5m" || acceptanceNeed(got) != 2 || acceptanceTFMinutes(got) != 5 {
		t.Fatal("2x5m semantics changed")
	}
}

// V3 companion: a 5m_close death fires on exactly ONE close-beyond (with the
// touch-gate satisfied) and the reason names the authored rule.
func TestFiveMCloseDeathFiresOnOneClose(t *testing.T) {
	c := PlanCondition{Price: 100, Side: "below", Rule: "5m_close"}
	// 1m source bars: a touch of the level, then one aggregated 5m bucket
	// closing below it. AcceptanceBars aggregates 1m → 5m.
	mk := func(openMs int64, h, l, cl float64) market.Kline {
		return market.Kline{OpenTime: openMs, CloseTime: openMs + 59_999, High: h, Low: l, Close: cl, Open: cl}
	}
	var bars []market.Kline
	base := int64(1_700_000_000_000) - (int64(1_700_000_000_000) % 300_000) // 5m-aligned
	// bucket 1 (touch, closes above): lows touch 100
	for i := int64(0); i < 5; i++ {
		bars = append(bars, mk(base+i*60_000, 101, 99.9, 100.5))
	}
	// bucket 2: closes below 100
	for i := int64(5); i < 10; i++ {
		bars = append(bars, mk(base+i*60_000, 100.2, 99.0, 99.2))
	}
	fired, reason := PlanConditionFiredSince(c, bars, base-1, base+10*60_000)
	if !fired {
		t.Fatalf("one 5m close beyond must fire a 5m_close death (reason=%q)", reason)
	}
	if !strings.Contains(reason, "5m_close") {
		t.Fatalf("reason %q must name the authored rule", reason)
	}
	// And a 2x5m rule on the SAME bars must NOT fire (only one bucket closed
	// beyond) — the phantom mapping would have made these identical.
	c2 := PlanCondition{Price: 100, Side: "below", Rule: "2x5m"}
	if fired2, _ := PlanConditionFiredSince(c2, bars, base-1, base+10*60_000); fired2 {
		t.Fatal("2x5m must still need TWO closes — one bucket fired it")
	}
}
