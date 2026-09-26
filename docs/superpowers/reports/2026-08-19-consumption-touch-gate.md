# CONSUMPTION TOUCH GATE — fix landed @ 91333cfb

1. **0 unjustified burns remain (12 false `done` rows purged, level_state consumed=0, writer now touch-gated) · deploy verdict: `🔐 BOOT INTEGRITY OK — rev 91333cfbc1ec +dirty · built 2026-08-18T02:12:03Z · expected 91333cfbc1ec · goldens PASS` · one PID 1206619.** (The first post-deploy executor tail was not yet written at report time — the count in the next prompt is proven by tests: only touched levels can read CONSUMED.)

2. **One meaning of consumed:** the writer already had `ConsumedSince` (touch AND N rule-TF closes since the row's birth) — the missing half was the DISPLAY. `ActivePlan` now carries `BirthMs` (plan-row created_at); `RenderPlanStatus` and the card's `planLevelFacts` evaluate through `BarsSince(bars, birth)`, so the prompt's closes-beyond is the birth-scoped count (the 84-vs-11 fix) and CONSUMED always means touched+accepted since birth. No fourth interpretation can appear: the AST guard test fails if any kernel display site calls `EvaluateLevelFacts` without `BarsSince` scoping.

3. **Purged:** backup `~/nofx-backups/manual/pre-consumption-purge-2112.db` → dry-run 12 `done` rows → deleted → consumed count 0. The writer recreates rows fresh on the next cycle and can only burn via a real post-birth touch.

4. **Tests (green):** kernel P1C suite (wick/sitting-beyond/pre-window don't consume; touch+accept does; interval not confused) + new birth-scoped count rendering + the AST guard. `go build` ✓ · kernel/trader/api tests ✓.

5. **Live check pending one cycle:** the 21:08 tail (pre-deploy) still showed the old whole-cache counts; the next cycle's tail should show birth-scoped numbers (e.g. PDL ~24 not 97) and only genuinely touched levels as CONSUMED.
