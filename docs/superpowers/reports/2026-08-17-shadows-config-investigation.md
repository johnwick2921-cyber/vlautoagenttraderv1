# 7 SHADOWS-CONFIG confirmed (H8 is live-harmful), 1 TRUE-CONSTANT, 1 safe — replan_cap: RESOLVED, config=4 wins

Read-only root-cause investigation of the 9 SHADOWS-CONFIG sites (H1–H9) from the CTO final-verification report. **Nothing fixed in this pass.** Every verdict carries file:line receipts and a three-way runtime proof (engine effective value · DB stored value · UI displayed value).

## Tools actually used

- `read_file` / `search_files` — walked `kernel/`, `trader/`, `store/`, `api/`, `web/src/` source.
- `git show <rev>:<path>` + `git diff 8e7b816a..HEAD` — pinned the **running binary** (`deploy/RELEASE = 8e7b816a`, bot PID 599604) against HEAD (`184fe200`).
- `sqlite3 -readonly "file:data/data.db?mode=ro"` — read live `strategies`, `plans`, `system_config` (no writes).
- `journalctl --no-pager --since/--until` (bounded) + `data/nofx_2026-08-16.log` — death-loop runtime receipts.
- **Not used:** subagents (direct reads were cheaper and had to stay precise), Playwright (the three-way UI side is a resolved API value in `handler_plan.go` + plan rows; no rendering claim required), MCP tools (none relevant).

## Deployment state (read this first — it changes what "live" means)

| Field | Value | Receipt |
|---|---|---|
| HEAD | `184fe200` | `git rev-parse HEAD` |
| **Running binary** | `8e7b816a` (7 commits behind HEAD) | `pgrep -af nofx-bin` → `599604 …/nofx-bin`; `deploy/RELEASE` first line |
| Bot state | `active`, PID 599604 | `systemctl is-active nofx` |
| Live DB | `data/data.db` (WAL; `data.db-wal` non-empty at read time) | `ls -la data/` |

The source files carrying H1, H2, H3, H4, H5, H7, H8 are **byte-identical between the running binary and HEAD** (`git diff --stat 8e7b816a HEAD` touches only `kernel/engine_analysis.go` — item-6 reasoning backfill — and `trader/auto_trader_planner.go` among the H-sites). So the line numbers below are what is executing on the bot right now, except where I mark a HEAD-only refinement.

---

## THE REPLAN-CAP ANSWER (the question you asked directly)

**Is "2-(version-1)" still deciding how many re-plans a session gets while the owner's config says 4? — NO. It is dead, and the config is honored.**

1. **The literal is gone from production code.** `8b24c85e` ("the executor prompt quotes the real re-plan cap, not a literal 2") is **in the running binary** (`git merge-base --is-ancestor 8b24c85e 8e7b816a` → yes). The old `trader/auto_trader_planner.go` line `replansLeft := 2 - (row.Version - 1)` now reads `store.ReplansLeftFor(row.Version, storedReplanCap(st, row.StrategyID, sess.Name))` (`trader/auto_trader_planner.go:869`), and `storedReplanCap` (`:822-840`) resolves from the LIVE stored strategy config via `ReplanCapFor` (`store/strategy.go:957-966`) — per-session override → strategy level → default 2.

2. **Engine effective value = 4.** Runtime log (`data/nofx_2026-08-16.log`) shows the enforcer re-reading the cap MID-SESSION and flipping:
   ```
   17:17:53 … ASIA v1 DIED — re-planning (cap 2/session).
   17:20:58 … ASIA v2 DIED — re-planning (cap 2/session).
   17:25:11 … ASIA v3 DIED — re-planning (cap 4/session).   ← owner raised 2→4
   17:31:20 … ASIA v4 DIED — re-planning (cap 4/session).
   17:37:29 … NO-TRADE … re-plans exhausted after death condition
   ```

3. **DB stored value = 4.** Live strategy `a5b7662e` (name `MNQ`) `day_plan.sessions[ASIA]` has `"replan_cap":4` (strategy-level `replan_cap:2`, override wins). All three sessions carry the same override.

4. **UI displayed value = 4.** `api/handler_plan.go:70-81` `planRulesWithCap` returns `replanCap = dp.ReplanCapFor(session)` and `replansLeft = store.ReplansLeftFor(version, replanCap)`, surfaced as `replan_cap`/`replans_left` (`:222,:250,:254`).

5. **The "six versions" were correct, not a cap failure.** `plans` shows ASIA v1–v5 as `ASIA_scheduled_read`/`active` and v6 as `replans_exhausted`/`no_trade`. Under the settled semantics (`store/strategy.go:974-985`): cap counts **re-plans**, v1 is free, so cap=4 ⇒ real versions v1…v5 (four re-plans) and the 5th death writes the terminal NO-TRADE marker as v6. That is exactly 4 re-plans spent. HEAD-only commit `301df4be` centralizes this (`ReplansUsed`/`MayReplan`/`ReplansLeftFor`) and re-renders the marker; it is a **display/semantics clarification, not a behavior change** — the arithmetic was already correct in the running binary.

**Verdict: SAFE / RESOLVED.** No literal decides the budget; config rules on all three surfaces.

---

## THE 9 SITES, RANKED BY HARM

### H8 — `sessions[].enable` (registry flag vetoes the owner's explicit `enable=true`) — **LIVE AND HARMFUL, highest harm**

- Receipts:
  - `trader/auto_trader_planconfig.go:104-121` `sessionRunnable()` — the correct resolver: explicit per-session `enable` override wins over registry + `sessions_enabled`.
  - Read scheduler honors it: `trader/auto_trader_planner.go:171-173` (`if runnable, _ := at.sessionRunnable(s); !runnable { continue }`) — **so ASIA reads fire and spend LLM.**
  - Entry gate vetoes it *anyway*: `trader/auto_trader_session.go:47` calls `sessionGateDecision`, whose `auto_trader_session.go:108` (`if !sess.Enabled`) checks **the registry's `Enabled` flag** — still false for ASIA/LONDON. So `sessionEntryBlocked` returns `"ASIA session not enabled"` and blocks entries even after `sessionRunnable(ASIA)=true`.
  - Executor prompt vetoes it *too*: `trader/auto_trader_planner.go:850` (`if !ok || !sess.Enabled { return nil }`) inside `installActivePlanProvider` — the executor **never sees the ASIA plan it just wrote**.
- Three-way proof:
  - **Engine:** two rules in one gate — `sessionRunnable(ASIA)=true` (read fires) vs `sessionGateDecision` `!sess.Enabled=true` (entry refused); `ActivePlanProvider` returns nil.
  - **DB:** strategy `a5b7662e` `day_plan.sessions[ASIA].enable=true`, `[LONDON].enable=true`; **`system_config.key='session_registry'` is ABSENT** (count 0) → `loadStoredRegistry` (`trader/auto_trader_registry.go:19-26`) → `kernel.DefaultSessionRegistry()` where ASIA/LONDON `Enabled:false` (`kernel/session_registry.go:88,97`).
  - **UI:** `api/handler_plan.go:135-138` `RunnableSessions()` → `sessionRunnable` → ASIA/LONDON tabs render runnable, while the executor eats the plan's absence. The read runs, a plan row is written, **the dashboard and the brain narrate different rulebooks.**
- Blast radius: owner flipped ASIA and LONDON `enable=true` → the bot spends LLM on reads it then discards, entries silently blocked. This is the P-G "live-reachable bypass" from the CTO report, still present.
- Verdict: **SHADOWS-CONFIG** (the registry `Enabled` literal-copy wins over the owner's per-session toggle at 2 deciding sites).
- Size: **M**. Single source of truth must be `sessionRunnable` (and `ActivePlanProvider` must use it); only 2 of the deciding sites were converted. Until then: **switch ASIA+LONDON back OFF** (they cost money and change nothing).

### H4 — `max_levels > 8` rejected by a hardcoded cap (fail-closed NO-TRADE when the owner raises it)

- Receipts: `kernel/plan_doc.go:60` (`planMaxLevels = 8`), enforced `plan_doc.go:101-102`. Config field `store/strategy.go:878` `MaxLevels` (FE offers up to 12, `web/src/components/strategy/DayPlanEditor.tsx:34,439-442`). The planner **input** honors config (`trader/auto_trader_planner.go:607-609` → `resolveSessionPlanCfg` → `dp.MaxLevels`), but `ParsePlanDoc` → `ValidatePlanDoc` (`kernel/plan_doc.go:68,77`) **rejects any doc with 9–12 levels before it is stored.**
- Three-way proof: engine key-levels uses config (`kernel/engine_analysis.go:316-321`); DB for `a5b7662e` `max_levels:8` (currently equals the literal → latent); UI editor advertises 3–12.
- Blast radius: raising `max_levels` to 9–12 makes every read fail-closed (whole plan rejected → NO-TRADE + P0 alert). The upper half of the documented range is unreachable.
- Verdict: **SHADOWS-CONFIG**. Size **M**.

### H5 — `scenario_cap > 3` unreachable (same class, same fail-closed consequence)

- Receipts: `kernel/plan_doc.go:61` (`planMaxScenarios = 3`), enforced `plan_doc.go:115-116`. Config `store/strategy.go:880` `ScenarioCap` (1–5); `at.scenarioCap()` (`trader/auto_trader_planconfig.go:133-138`) truncates **post-parse** (`trader/auto_trader_planner.go:562-565`), but parse already rejects >3 — so the resolver can only lower below 3, never raise above.
- Verdict: **SHADOWS-CONFIG**. Size **M** (parameterize `ValidatePlanDoc` caps, keep 12/5 as hard ceilings).

### H1 — `proximity_filter_atr` hardcoded 1.5 in the scorer (day-trade lock)

- Receipts: `kernel/levels_score.go:125` (`band := 1.5 * dATR`). Config `store/strategy.go:876` `ProximityFilterATR`, resolver `trader/auto_trader_planconfig.go:125-130` `proximityFilterATR()`. **The resolver is called only by `trader/auto_trader_levelstate.go:55,164`** (level-state writer + scenario evaluator); it is **not threaded into `ScoreLevels`**, so the detector/scorer proximity filter ignores the config.
- Three-way proof: DB `a5b7662e` `proximity_filter_atr:1.5` = literal today (latent); engine `AssembleScoredLevels` (`kernel/levels_assemble.go:67`) → `ScoreLevels` uses the literal; UI editor shows 1.5.
- Blast radius: changing `proximity_filter_atr` changes the *activation window* paths but does nothing to *which levels the detector generates/seats*. Half-honored setting.
- Verdict: **SHADOWS-CONFIG**. Size **M**.

### H2 — the upstream twin: round numbers generated only within 1.5×dATR

- Receipts: `kernel/levels_intraday.go:24` (`band := 1.5 * dATR` in `RoundNumberLevels`). Called from `kernel/levels_assemble.go:55`. No config parameter.
- Blast radius: round-number levels beyond 1.5×dATR are never even generated, so fixing H1 alone still can't honor a wider `proximity_filter_atr` for the RN family.
- Verdict: **SHADOWS-CONFIG**. Size **M** (thread `proximityK` through `RoundNumberLevels`/`ScoreLevels` + the callers, all three sites or it stays half-honored).

### H9 — `planner_timeframes` asserts a read-set the planner never received

- Receipts: config read at `trader/auto_trader_planner.go:610-611` (`timeframes = dp.PlannerTimeframes`), emitted only as placeholder lines `"tf: structure read"` (`trader/auto_trader_planner.go:720-722`); the bars actually fetched are hardcoded `"1d",300` / `"1h",300` / `"5m",300` / `"5m",3000` (`trader/auto_trader_planner.go:669-672`).
- Three-way proof: DB `planner_timeframes:["D","4h","1h","15m"]`; engine fetch set `{1d,1h,5m}`; UI editor offers the multiset. Selecting "4h" tells the planner it read 4h structure it never saw.
- Verdict: **SHADOWS-CONFIG** (worse than dead: the prompt asserts a false read-set). Size **S** (fetch the configured TFs, or stop claiming them).

### H7 — executor prompt uses `DefaultSessionRegistry`, not the persisted admin registry

- Receipts: `kernel/engine_analysis.go:333` (`BuildKeyLevelsBlock(bars, DefaultSessionRegistry(), …)`) and `:361` (`AssembleScoredLevels(bars, DefaultSessionRegistry(), …)`). The trader layer has the correct seam (`trader/auto_trader_registry.go:19` `loadStoredRegistry`, `:32` `sessionRegistry`) and **does** use it on the planner-read path (`auto_trader_planner.go:629,848`) — only the executor-prompt level extraction bypasses it.
- Three-way proof: DB `system_config.session_registry` **absent** → `loadStoredRegistry` returns the default → engine `DefaultSessionRegistry()` equals the live registry **today** (latent); UI session-registry editor (`/api/plan/session-registry`) would write a value the prompt path would then ignore.
- Verdict: **SHADOWS-CONFIG** (latent until an admin edits the registry). Size **M** (add a registry-provider seam to the kernel, mirroring bars/nPOC).

### H3 — activation window `ActivationWindowK` — TRUE-CONSTANT, but cross-fed with config

- Receipts: `kernel/plan_lifecycle.go:13` (`const ActivationWindowK = 1.5`); promp-path `kernel/plan_render.go:65,67` uses the constant. The spec (`docs/VL-DAYPLAN-FULL-SPEC.md:73-74`) lists "activation-window k=1.5" as an **INTERNAL CONSTANT**, not an owner-facing knob — so the prompt path using the constant is correct per contract.
- The seam: `trader/auto_trader_levelstate.go:55,164` feeds `at.proximityFilterATR()` (the owner's `proximity_filter_atr`) into the *same* `ActivePlanLevels(…, k, …)` parameter that the prompt path fills with the constant. Two callers of one function disagree on what "k" is — spec's internal constant vs the owner's day-trade-lock field.
- Three-way proof: DB `proximity_filter_atr:1.5` = literal = constant today → no live divergence.
- Verdict: **TRUE-CONSTANT** on the prompt path; the level-state writer **mixes config into it** (semantic blur, latent). Size **S** to name/kill the ambiguity.

---

## Summary table

| Site | Receipt | Config shadowed | Live today? | Verdict | Size |
|---|---|---|---|---|---|
| H8 | `auto_trader_session.go:108`, `auto_trader_planner.go:850` | `sessions[].enable` | **YES — harmful** | SHADOWS-CONFIG | M |
| H4 | `plan_doc.go:60,101` | `max_levels` (9–12) | latent (value=8) | SHADOWS-CONFIG (fail-closed) | M |
| H5 | `plan_doc.go:61,115` | `scenario_cap` (4–5) | latent (value=3) | SHADOWS-CONFIG (fail-closed) | M |
| H1 | `levels_score.go:125` | `proximity_filter_atr` | latent (value=1.5) | SHADOWS-CONFIG | M |
| H2 | `levels_intraday.go:24` | `proximity_filter_atr` | latent (value=1.5) | SHADOWS-CONFIG | M |
| H9 | `auto_trader_planner.go:669-672` vs `610,721` | `planner_timeframes` | **YES** (writes false "structure read" claims) | SHADOWS-CONFIG | S |
| H7 | `engine_analysis.go:333,361` | persisted `session_registry` | latent (registry absent) | SHADOWS-CONFIG | M |
| H3 | `plan_render.go:65` vs `levelstate.go:55,164` | `proximity_filter_atr` | no (both 1.5) | TRUE-CONSTANT (spec) / mixed at one caller | S |
| H6 | `auto_trader_planner.go:869` (`8b24c85e`) | `replan_cap` | **no — config wins** | **SAFE / RESOLVED** | — |

## Correct single source of truth per site

- **H8** → `sessionRunnable()` (already correct; finish the conversion of the 7 remaining deciding sites and make `ActivePlanProvider` use it).
- **H4/H5** → `DayPlanConfig.MaxLevels` / `ScenarioCap` must reach `ValidatePlanDoc` (12/5 as hard ceilings).
- **H1/H2** → a `proximityK` parameter threaded from `proximityFilterATR()` into `ScoreLevels` + `RoundNumberLevels` at every call site.
- **H9** → fetch the configured `PlannerTimeframes`, not hardcoded 1d/1h/5m.
- **H7** → a kernel `SessionRegistryProvider` seam (mirror `FuturesBarsProvider`/`NakedPOCProvider`).
- **H6** → already single-sourced (`store.ReplansUsed` / `MayReplan` / `ReplansLeftFor` + `ReplanCapFor`).

## Owner actions

1. **Do not deploy anything for this report — it is findings only.** No code changed.
2. **Until H8 is fixed, switch ASIA + LONDON back OFF.** They are live-enabled (`enable:true`) in strategy `a5b7662e`; reads fire (LLM spend) and then the entry gate + executor provider veto them. Zero trading value, real cost.
3. **`replan_cap` needs no action.** Your 4 is honored on the engine, the DB, and the card. The "v6" row is the append-only NO-TRADE marker, not a cap failure.
