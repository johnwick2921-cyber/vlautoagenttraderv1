package kernel

// W11b — LEVEL-STATE surfacing (flagged W7 follow-up). W7 WRITES cross-session level
// state (freshness A→B→C, consumed) but the executor's KEY LEVELS + PLAN STATUS
// still rendered every level "fresh" (AssembleScoredLevels passed a nil freshness
// callback — see levels_assemble.go). This hook lets the DB-blind kernel READ the
// persisted state: the trader installs a reader over store.LevelStateStore.
//
// Returns a level's persisted freshness grade ("A"|"B"|"C"|"done"|"") for a level
// IDENTITY (type-from-label + price-bin, the same identity W7's writer uses).
// "" = no persisted state = fresh (the pre-W11b behavior). "done"/"consumed" →
// ScoreLevels drops the level (freshMult 0); "B"/"C" → shown tested/B. Nil (tests,
// goldens) → everything fresh → byte-identical output.
// P0-cleanup (2026-08-19) — trader-scoped: the provider receives the deciding
// trader's id so two day-plan traders never share burn/freshness state.
var LevelStateProvider func(traderID, symbol string, l DetectedLevel) string

// levelFreshnessFn builds the ScoreLevels freshness callback for a symbol from the
// installed LevelStateProvider (nil when no provider → all-fresh, as before).
func levelFreshnessFn(traderID, symbol string) func(DetectedLevel) string {
        if LevelStateProvider == nil {
                return nil
        }
        return func(l DetectedLevel) string { return LevelStateProvider(traderID, symbol, l) }
}
