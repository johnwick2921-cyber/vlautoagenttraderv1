package kernel

import "time"

// W11b — LEVEL-STATE surfacing (flagged W7 follow-up). W7 WRITES cross-session level
// state (freshness A→B→C, consumed) but the executor's KEY LEVELS + PLAN STATUS
// still rendered every level "fresh" (AssembleScoredLevels passed a nil freshness
// callback — see levels_assemble.go). This hook lets the DB-blind kernel READ the
// persisted state: the trader installs a reader over store.LevelStateStore.
//
// Returns a level's persisted freshness grade ("A"|"B"|"C"|"done"|"") for a level
// IDENTITY (type-from-label + price-bin, the same identity W7's writer uses).
// "" = no persisted state = fresh (the pre-W11b behavior). "done"/"consumed" →
// ScoreLevels role-flips the level at REDUCED weight (freshMult 0.5, label
// "flipped"), never drops it (P1c — the map's best levels stay visible);
// "B"/"C" → shown tested/B. Nil (tests,
// goldens) → everything fresh → byte-identical output.
// P0-cleanup (2026-08-19) — trader-scoped: the provider receives the deciding
// trader's id so two day-plan traders never share burn/freshness state.
// S2 (2026-09-16) — `now` is the READ's now (A28 / class 60), threaded from the
// scoring call site; time-dependent grading rules never consult the wall clock
// of their own.
var LevelStateProvider func(traderID, symbol string, l DetectedLevel, now time.Time) string

// levelFreshnessFn builds the ScoreLevels freshness callback for a symbol from
// the installed LevelStateProvider (nil when no provider → all-fresh, as
// before). `now` is the READ's now, captured here at the scoring call site
// (class 60 / F4): the provider never consults the wall clock of its own.
func levelFreshnessFn(traderID, symbol string, now time.Time) func(DetectedLevel) string {
	if LevelStateProvider == nil {
		return nil
	}
	return func(l DetectedLevel) string { return LevelStateProvider(traderID, symbol, l, now) }
}
