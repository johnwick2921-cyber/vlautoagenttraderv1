# Acceptance interval fix — "2x5m" counted on the rule's timeframe everywhere (2026-08-17)

1. **Root cause:** `EvaluateLevelFacts` counted `ClosesBeyond`/`Acceptance`/`StillValid` on whatever bars it was handed — the 1-minute SVP cache at every call site. "2x5m" (two 5-minute closes) was satisfied by two 1-minute closes: acceptance announced ~5× early, and the level-state writer could burn a level for the day ~5× early too.
2. **Fix:** one resolver in `kernel/scenario_facts.go` — `AcceptanceIntervalMinutes` (interval truth), `AcceptanceBars` (bars bucketed to the rule timeframe), `LevelStillValidOn`; `EvaluateLevelFacts` resolves the timeframe internally. `plan_lifecycle.go` routes through the same resolver. Commits `480a4d51` (fix) + `77623924` (tests).

3. **Site table (before → after):**

| Site | file:line | before | after |
|---|---|---|---|
| death: PlanIsDeadSince | kernel/plan_lifecycle.go:86 | 5m ✓ (correct) | 5m ✓ |
| death: DescribePlanDeath | kernel/plan_lifecycle.go:114 | 5m ✓ | 5m ✓ |
| executor PLAN STATUS tail | kernel/engine_analysis.go:366 → plan_render.go:75 | **1m ✗** | 5m ✓ |
| scenario evaluator | kernel/scenario_state.go:164 | **1m ✗** | 5m ✓ |
| level-state writer (persist consumed) | trader/auto_trader_levelstate.go:85 | **1m ✗** | 5m ✓ |
| card per-level facts | api/handler_plan.go:300 | **1m ✗** | 5m ✓ |
| card live status | api/handler_plan.go:989, :1735 | **1m ✗** | 5m ✓ |

4. **What the AI was told vs the truth:** tail said "closes-beyond N · acceptance n/2 · valid" from 1m closes — e.g. 2 min beyond a level read as "acceptance 2/2" and "CONSUMED". Truth: 10 minutes (two 5m closes) for 2x5m; one 15m close for 15m-close. The card showed the same wrong numbers.
5. **Goldens:** all three embedded prompt goldens re-rendered and compared — **zero diffs** (the golden fixtures hardcode the PLAN STATUS line; they do not compute facts from bars, so the live tail changes without touching golden bytes). `TestFuturesPlanGolden`, `TestVerifyPromptGoldensPasses` green.
6. **Tests (green):** two 1m closes do NOT accept under 2x5m (have=0); two complete 5m buckets do; 15m-close needs one 15m close; `TestAcceptanceIntervalNoHardcodedSites` walks the repo AST and fails if any file outside `scenario_facts.go` calls a raw bar counter.
7. **Deploy:** NOT deployed (never restart the bot). Owner: mandatory order — git pull → `go build -o nofx-bin .` → `git rev-parse HEAD > deploy/RELEASE` → restart. Until then the running binary still counts the tail on 1m.
