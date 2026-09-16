package telemetry

// WEEKLY-BIAS WAVE (2026-08-30) — W2b/W5 shadow counters. Same pattern as the
// B6 gate-block table: in-memory, ephemeral, reset at the CME session-day
// rollover. SHADOW ONLY — these counters feed the Sep-9 promotion table, they
// never gate anything.

import "sync"

var (
	weeklyMu          sync.Mutex
	weeklyCnts        = map[string]int{} // counter name → count
	weeklyCitationDay int64              // session-day the citation count rolled on
)

func weeklyInc(name string) {
	if name == "" {
		return
	}
	weeklyMu.Lock()
	weeklyCnts[name]++
	weeklyMu.Unlock()
}

// IncPlannerCandleCitation (W2b) counts one scenario-rationale line containing
// the marker phrase "per candles" in a written plan — the planner actually
// citing the candle tables. One increment per cited line.
func IncPlannerCandleCitation(trader string) {
	_ = trader // the counter is wave-wide; trader dimension lives in the journal line
	weeklyInc("planner_candle_citations")
}

// IncWeeklyCounter (W5.2) counts one entry decision opposing the weekly bias
// (conviction med|high) — counter_n in the Sep-9 promotion table.
func IncWeeklyCounter(trader string) {
	weeklyInc("weekly_counter")
	IncGateBlock(trader, "weekly_counter")
}

// IncWeeklyCounterBlock counts a counter-trend entry the hypothetical hard rule
// would have BLOCKED (would-require-A-grade) — would_block_n.
func IncWeeklyCounterBlock(trader string) {
	weeklyInc("weekly_counter_block")
	IncGateBlock(trader, "weekly_counter_block")
}

// IncWeeklyCounterResize counts a counter-trend entry the hypothetical hard
// rule would have RESIZED (would-halve-size) — would_resize_n.
func IncWeeklyCounterResize(trader string) {
	weeklyInc("weekly_counter_resize")
	IncGateBlock(trader, "weekly_counter_resize")
}

// IncWeeklyReadFailed counts a weekly read that failed both attempts.
func IncWeeklyReadFailed(trader string) {
	weeklyInc("weekly_read_failed")
	IncGateBlock(trader, "weekly_read_failed")
}

// WeeklyCounterSnapshot returns the wave-wide shadow counters (name → count).
func WeeklyCounterSnapshot() map[string]int {
	weeklyMu.Lock()
	defer weeklyMu.Unlock()
	out := make(map[string]int, len(weeklyCnts))
	for k, v := range weeklyCnts {
		out[k] = v
	}
	return out
}

// ResetWeeklyCounters clears the shadow counters (tests).
func ResetWeeklyCounters() {
	weeklyMu.Lock()
	weeklyCnts = map[string]int{}
	weeklyMu.Unlock()
}

var _ = weeklyCitationDay // reserved for the per-day citation journal line (W2b)
