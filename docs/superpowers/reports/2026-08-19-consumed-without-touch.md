# CONSUMED-WITHOUT-TOUCH — ASIA 08-17 v3 audit @ HEAD 32892232

1. **0 of 7 burns justified · 7 false.** Every consumed level was burned with NO in-window touch after its birth (19:18:14 CT, v3 owner_reset): ONL 30062, PDL 30166.25, OR-L 30190.50, nPOC 30203.12, PDC 30234.50, PDH 30254.75, OR-H 30265.75 — touch=F after birth for all 7, each shows 11 consecutive 5m closes beyond since birth (pure proximity), consumed purely by closes-beyond. The 8th (ONH 30104, +6pt) still valid ✓.

2. **Root cause (two sites, same class):** `kernel/scenario_facts.go` — `LevelStillValid`/`EvaluateLevelFacts` count 2+ rule-TF closes beyond on the FULL 2000-bar cache with no touch gate and no birth window; `trader/auto_trader_levelstate.go` ~line 100 persists it: `if !f.StillValid { ls.MarkConsumed(key) }`. The death path already has the correct rule (`plan_lifecycle.go`: touched AND consumed, since birth); the per-level facts path and the state writer don't. Prompt evidence: "PDL … closes-beyond 84 · CONSUMED" (84 = whole cache; only 11 since birth).

3. **State:** session-fresh — the writer re-burned rows at 20:00 CT (done/consumed=1); one inherited ONH/ONL row (15:25, freshness C) shows aging works for freshness but nothing demotes a consumed flag. Purge without the fix is futile: the writer re-burns every cycle.

4. **Consequence:** S1's targets 30166.25→30190.50→30203.12 render CONSUMED. The prompt softens it ("flipped — tradeable both directions") but the card grays them and prior cycles cite "A-levels all consumed" as wait reasons — consumed targets DO bias the model toward waiting.

5. **Fix (SIZED S–M, not shipped):** one shared `LevelConsumedSince(bars, level, rule, birthMs, now)` = in-window touch after birth AND N rule-TF closes beyond after birth; wire it into `EvaluateLevelFacts` (StillValid) and gate `recordLevelState`'s MarkConsumed on it. Then purge the false `done` rows (backup → dry-run → execute → verify) and add the failing test: "a level is never consumed without a recorded in-window touch after its birth".

6. **Not deployed** (no code shipped this pass — the fix and purge need one tested commit at a flat window).
