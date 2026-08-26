package kernel

import (
	"strings"
	"testing"
)

// 6.3 (final-bundle 2026-08-19) — every CONFIGURED limit renders a would-trip
// line under master OFF, including the two that used to die silently
// (blackout, consistency — PR #54: 69 live lines were trio-only).
func TestCheckSoftCoversAllFiveChecks(t *testing.T) {
	g := DailyGuardrails{
		MasterEnabled:        false,
		DailyRealizedPnL:     -500,
		TradesToday:          4,
		DailyLossLimitUSD:    450,
		DailyProfitTargetUSD: 1, // trivially reachable? no — pnl negative; use profit case separately
		MaxDailyTrades:       3,
		BlackoutConfigured:   true,
		InBlackoutNow:        true,
		ConsistencyMaxDayPct: 30,
		TotalRealizedPnL:     100,
	}
	hits := g.CheckSoft()
	want := []string{"daily loss would trip", "max daily trades would trip", "blackout window would trip"}
	for _, w := range want {
		found := false
		for _, h := range hits {
			if strings.Contains(h, w) {
				found = true
			}
		}
		if !found {
			t.Errorf("CheckSoft missing %q (got %v)", w, hits)
		}
	}
	// Profit + consistency cases (positive-day variant).
	g.DailyRealizedPnL = 900
	g.TotalRealizedPnL = 2000 // total must exceed today (first-profitable-day guard)
	g.DailyProfitTargetUSD = 900
	hits = g.CheckSoft()
	for _, w := range []string{"daily profit target would trip", "consistency rule would trip"} {
		found := false
		for _, h := range hits {
			if strings.Contains(h, w) {
				found = true
			}
		}
		if !found {
			t.Errorf("CheckSoft missing %q (got %v)", w, hits)
		}
	}
	// And CheckSoft NEVER blocks: it only reports (compile-level truth — it
	// returns strings; this assert pins that unconfigured checks stay silent).
	if hits := (DailyGuardrails{}).CheckSoft(); len(hits) != 0 {
		t.Errorf("unconfigured guardrails must be silent, got %v", hits)
	}
}
