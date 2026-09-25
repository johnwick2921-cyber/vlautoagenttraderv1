# 2026-08-17 — COMBINED FIX TRAIN (P1–P7 + latency half) — RESOLVED

**Verdict: SHIPPED.** 9 commits, 020e407a. Every P-item from the dispatch is either fixed end-to-end or proven to be a deploy gap with the chain complete at HEAD. Build/vet/test/-race green; no goldens touched; FE 243/244 tests green (1 pre-existing auth failure + 1 pre-existing e2e suite, both untouched by this session).

## Commits (append-only, pushed to origin)

| Commit | Item |
|---|---|
| `2b4162f6` | Latency half of the P0 decision fix: 180s decision-call timeout + stale-bar discard |
| `6934a18e` | P1/H9 — planner fetches the CONFIGURED timeframes, honest lines |
| `f13ecb86` | P2/H4+H5 — max_levels/scenario_cap reach validation AND the prompt (12/5 hard ceilings) |
| `8338aaea` | P3/H1+H2 — proximity_filter_atr threaded into generation + seating at every call site |
| `a47a51a6` | P4/H7 — SessionRegistryProvider seam (+ H3 k-naming landed with the proximity work) |
| `9199e9f0` | P7 — NO-TRADE plans carry the map, read-only on the card |
| `020e407a` | P6 — the owner reset (abandon chain, re-arm budget, fresh plan) |

## Verified from the stalled session (not re-done)

- **`2ddf3a58` (JSON recovery) — COMPLETE on its half.** Recovery of prose-embedded single-object decisions (`kernel/engine_analysis.go:756-777`), prose-only → error so `callWithSchemaRetry` re-asks, P1 `decision-unparseable` alert per miss (`trader/auto_trader_loop.go:405-413`), 4 tests. The LATENCY half was missing and is now `2b4162f6`: the futures decision client is capped at `decisionCallTimeout=180s` (leaves ≥120s of a 5m bar); `runCycle` captures the decision bar's close before the call and, if a NEWER primary bar closed in-flight, the decision is DISCARDED (named guardrail skip + gate-block counter + P1 `decision-stale-bar` alert). **Expected new failure rate: ≤~0.3% of cycles still lose a decision (was 70/5660 = 1.24%)** — the prose-object recovery saves most misses instantly, the JSON-only retry re-asks cheaply, the timeout converts the 182s-class tail into fast retries, and every residual loss is alerted and visible. Label: estimate from the owner's 14-day sample, not a measured rate.
- **`570c6c32` (H8) — COMPLETE.** Every deciding site now routes through `sessionRunnable`: read scheduler (`auto_trader_planner.go:172`), entry gate (`auto_trader_session.go:38,47,108-120`), executor provider (`auto_trader_planner.go:857`), digests (`:781`), re-read (`auto_trader_reread.go:45`), `RunnableSessions` (`planconfig.go:241`). Proven by `TestW15SessionRunnable` + `w16_date_and_provider_test` (enable=true → read fires AND gate allows AND executor receives; enable=false → none). **Owner action after deploy: ASIA/LONDON are live-enabled in strategy `a5b7662e` — with H8 fixed they now become REAL sessions (entries included), not just LLM spend. Turn them off or keep them deliberately.**
- **P5 (alert ✕ / clear) — ROOT CAUSE: a DEPLOY GAP, not a code bug.** The chain is complete at HEAD: store soft-delete (`store/alert.go` DismissForTrader/DismissAckedForTrader, `dismissed` columns filter List/UnackedCount), handlers (`api/handler_plan.go:710-790`), routes (`api/server.go:487-490`), FE (`AlertCenter.tsx` dismiss/clearRead + mutate), tests green (store/api/vitest W21+P4.4). The RUNNING binary predates ITEM 5: `strings nofx-bin` has `/plan/alerts` + `/plan/alert-ack` but NOT `/plan/alert-dismiss` or `/plan/alert-clear-read`, and the live `day_plan_alerts` table has no `dismissed` columns — so the ✕ and clear POSTs 404'd. The deploy handoff below closes it (AutoMigrate adds the columns at first boot). No code change was needed or made.

## Per-priority receipts

- **H9** — `auto_trader_planner.go` hardcoded 1d/1h/5m fetches (:669-672) while prompt lines claimed the configured set (:720-722). Now `structureSummaryLines` fetches each configured TF (D→1d) and emits `<tf>: structure read` or `<tf>: unavailable`; nil provider → all unavailable. Tests: configured==fetched==lines; missing TF; end-to-end through `assemblePlannerInput`.
- **H4/H5** — `kernel/plan_doc.go:60-61,101,115` hardcoded 8/3 at VALIDATE time → raising max_levels/scenario_cap made every read fail-closed. Now `ValidatePlanDocWithCaps`/`ParsePlanDocCapped` with hard ceilings `PlanHardMaxLevels=12/PlanHardMaxScenarios=5`; the prompt's output contract renders the resolved caps; `runPlannerReadCoreWithTrigger` parses with `resolveSessionPlanCfg` + `at.scenarioCap()`; read-path re-validations use the hard ceilings (write-time is the policy gate). Tests: 12/5 valid, 13/6 rejected, defaults unchanged, write path end-to-end.
- **H1/H2** — `levels_score.go:125` and `levels_intraday.go:24` hardcoded 1.5×dATR. Now `proximityK` threads from `at.proximityFilterATR()` through `RoundNumberLevels → ScoreLevels → AssembleScoredLevels → BuildKeyLevelsBlock` at EVERY call site (planner read, level-state writer ×2, and the kernel executor path via the new `PlanProximityKProvider` installed in the trader's provider Once). Tests: k=2.5 seats levels 1.5 does not (scorer + round numbers).
- **H7** — `engine_analysis.go:333,361` used `DefaultSessionRegistry()`. Now `resolvedSessionRegistry()` reads `SessionRegistryProvider` (installed from `at.sessionRegistry`'s per-day cache); nil/empty → shipped default (byte-identical). Test proves a persisted LONDON-enabled registry reaches the kernel seam.
- **H3** — `auto_trader_levelstate.go:55,164` fed the owner's `proximity_filter_atr` into the ACTIVATION-WINDOW k. Both call sites now pass `kernel.ActivationWindowK` by name with comments: proximity_filter_atr = generation/seating; ActivationWindowK = the spec internal constant for the live candidate filter. Ambiguity named away.
- **P7** — `writeNoTradePlan` + the fail-closed read stored `levels:null`. Now `NoTradePlanDocWithLevels(reason, at.noTradeLevelMap())` carries the current detector/scorer output (owner sticky levels prepended), scenarios stay the single S0 placeholder; a dark detector writes the explicit "detector data unavailable" line. The card renders them read-only (owner door shut for `lifecycle=no_trade`) under the ⛔ banner. Go + FE tests.
- **P6** — GET/POST `/api/plan/reset` + `ResetButton` beside the re-read with one explanatory line each. Semantics: the chain is marked ABANDONED append-only (history + death reasons preserved — rows are never updated), the budget is restored via a baseline marker in `system_config` (`latest+1` = the new chain's free v1), NO-TRADE cleared by writing an ACTIVE plan through the normal read path (`trigger_reason "owner reset"`, fail-closed inside), sticky owner levels carry by price identity. Refusals with reasons: plan off / night / session disabled / market closed / no plan yet. Positions, brackets, guardrail counters and the daily cage are never touched. Budget math became baseline-relative (`ReplansUsedFrom/MayReplanFrom/ReplansLeftFrom`; old functions = baseline 1) threaded through all four consumers (death path, executor provider, re-read gate, card rulebook). NOTE: plan rows are append-only, so the fresh plan is the next version NUMBER (e.g. v7) — the baseline makes it the new chain's v1 budget-wise.

## Not done / honest gaps

- **P5 live fix** = the deploy handoff below (owner restart). Nothing to change in code.
- **FE e2e (`e2e/gate.spec.ts`) and `RegistrationDisabled` logo test** fail — both pre-existing, 0 files under those paths touched this session.
- **Config-truth 4-step**: no config persistence schema changed this session; the enforcement paths reading existing fields (proximity_filter_atr, max_levels, scenario_cap, planner_timeframes) are proven by unit tests. Save→row→UI legs were shipped in prior sessions and unchanged.
- Multi-trader provider globals (SessionRegistry/Proximity/ActivePlan) are last-installer-wins — a known, accepted limitation of the existing provider pattern; flagged, not fixed.
- Stray files left from earlier sessions (untracked): `sandbox-seed` (34MB binary), `trader/demo_seed_test.go`, `trader/demo_verify_test.go`, `web/test-results/`, `web/vite.preview.config.ts`. The junk `tatus --short` file was removed.

## Deploy handoff (owner, at a flat/safe window)

1. `git pull` (done: pushed to origin, HEAD `020e407a`).
2. Binary built: `go build -o nofx-bin .` (done) · `git rev-parse HEAD > deploy/RELEASE` (done → `020e407a7f74ebbf509fbeaa6bfb6020f6453fab`).
3. `sudo systemctl restart nofx` — or `kill -9 <PID 599604>` and let `Restart=on-failure` relaunch the new binary.
4. `cd web && npm run build` (done) + hard reload.
5. Verify after restart: the version mismatch assertion clears; `/plan/alerts` rows gain `dismissed` columns; ✕ and Clear-read now work; decide ASIA/LONDON enablement (they are REAL sessions now).
