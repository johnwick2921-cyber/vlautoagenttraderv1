package trader

import (
	"fmt"
	"strings"

	"nofx/store"
)

// replanCapBootSessions is the order the per-session caps print in — the
// three sessions the planner runs, NY first like every other session line.
var replanCapBootSessions = []string{"NY", "ASIA", "LONDON"}

// ReplanCapBootLine (W1, settings truth, 2026-09-23) renders the re-plan cap
// the gates will run at trader load: the strategy level and each session, each
// READ from store.ResolveReplanCap with its origin — [O] saved (strategy value
// or session override), [I] the shipped default. Before W1 no line said what
// the cap was, and a saved strategy-level 0 was silently run as 2.
//
// The day plan being off is stated, not hidden: the cap is resolved either
// way, but with plan_enabled=false nothing consults it.
func ReplanCapBootLine(dp *store.DayPlanConfig) string {
	parts := make([]string, 0, 1+len(replanCapBootSessions))
	n, src := store.ResolveReplanCap(dp, "")
	parts = append(parts, fmt.Sprintf("strategy=%d%s", n, store.OriginLetter(src)))
	for _, s := range replanCapBootSessions {
		n, src := store.ResolveReplanCap(dp, s)
		parts = append(parts, fmt.Sprintf("%s=%d%s", s, n, store.OriginLetter(src)))
	}
	line := "🧮 replan cap: " + strings.Join(parts, " · ")
	if dp == nil || !dp.PlanEnabled {
		line += " (day plan off — not consulted)"
	}
	return line
}

// BreakerBootLineForStrategy (W1) — the consecutive-loss breaker of ONE bound
// strategy, printed at trader load beside the exit posture: the value READ from
// store.ResolveBreakerHalt with its origin (8[I], 3[O], off[O], 5[E], off[E]).
func BreakerBootLineForStrategy(cfg *store.StrategyConfig) string {
	breaker, n := breakerBootField(cfg)
	return fmt.Sprintf("breaker=%s (consecutive-loss halt; %s)", breaker, breakerTapeNote(n))
}
