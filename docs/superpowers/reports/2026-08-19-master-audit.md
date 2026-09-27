# VL Master Audit — 2026-08-19 (run against checklist v1)

**76 rows verified · 22 findings · 3 can lose money or a trade · 6 UNPROVEN · binary==HEAD: yes (code-identical; HEAD is 2 docs-only commits ahead of the binary)**
Start rev b4f8e05d → End rev b4f8e05d (HEAD did not move). Read-only throughout: no fixes, no deploy, no restart. Tools: 4 parallel Explore subagents (§1/6, §2/3, §4/7, §5/8/9/10/11) + terminal (sqlite3 -readonly on a fresh DB copy, journalctl 48h, pgrep, curl+JWT, python on live bars) + Playwright page earlier this session. `deploy/RELEASE`=c8f91d41, boot line `🔐 BOOT INTEGRITY OK — rev c8f91d413cca +dirty · expected c8f91d413cca · goldens PASS`, 1 PID (1379679), cycling (cycle 19917+). Sessions runnable: ASIA·LONDON·NY (strategy `sessions[].enable` overrides); trader hoang running (Sim101), 15m stopped.

## Verdicts per row (receipts)

**§1 units/windows/zones/scope** — 1.1 PASS+1 residual FAIL: `market/data.go:536-543` chat table still "Time(UTC)" (outside tz guard dirs); trading prompts clean (`engine_prompt.go:718` Time(CT)). 1.2–1.4 PASS (tz.go helpers, `w12_dst_test.go`, `tz_test.go`, guard `tz_guard_test.go`). 1.5 PASS (counters only in `scenario_facts.go`; AST guard). 1.6 PASS (all verdicts birth-windowed; deliberate full-cache reads bounded+stated). 1.7 PASS+gap (owner_levels.CreatedAt never set — unused for verdicts). 1.8 **FAIL**: `level_state.level_key` = symbol|type|date|bin — no trader_id (`store/level_state.go:54`); two MNQ traders share burn/freshness. 1.9 PASS (all money math ×point-value). 1.10 PASS.

**§2 config authority** — 2.1 **FAIL** (UI confidence fallback 75 ≠ backend safe 65; UI contracts hint 10 ≠ venue 2 — `RiskControlEditor.tsx:605/938`). 2.2 **FAIL** (dead fallbacks 60/1.5 in `engine_prompt_futures.go:63-67` shadow config). 2.3 **FAIL** (defaults duplicated: scan-interval ×4, 13:00 ×3, 14:45 ×5, maxLevels ×2). 2.4 PASS (unset→safe 65/3.0). 2.5 **FAIL** (FE mirror constants incl. 14-TF list despite `/strategies/timeframes` "single source"). 2.6 PASS (codec round-trip golden; concurrency-clobber flag). 2.7 **FAIL-clarity** (`sessions_enabled:["NY"]` + `sessions[].enable` coexist; precedence defined, UI should reconcile). 2.8 PASS. 2.9 PASS live (prompt shows stored 60; ASIA ran per override).

**§3 wiring** — 3.1 **FAIL** (orphans: `/plan/approve` un-exercisable from UI; FE `top-traders`/`competition` call dead endpoints). 3.2 PASS (6 buttons traced to persisted state). 3.3 **FAIL** (`fill` alert defined, 0 live emits; DB kinds: armed/decision-stale-bar/decision-unparseable/level-burned/owner-*/plan-died/read-fail/regime-dark). 3.4 **FAIL** (grid_* store orphan — no production writer; day_plan_digests producer only in sandbox-seed — table empty). 3.5/3.6 PASS (equity/positions/gate-blocks producers traced). 3.7 PASS (live decision rows + bar feed).

**§4 truth of what's shown** — 4.1 PASS live (43 guardrail_skip rows all success=0 + stamped reason; 0 refusals-as-success; 0 schema_parse_failed with risk_check_passed). 4.2 PASS+residual (CoT backfill; bare-JSON wait still renders blank). 4.3 PASS (one in-memory map; resets on restart — see finding). 4.4 **FAIL** (MAE/MFE/adherence computed+stored+served, zero FE consumers; dropped from session digest). 4.5 PASS+note (armed dot is plan-derived by design). 4.6 PASS live (position 520 absent, 0 seed rows).

**§5 deployment integrity** — 5.1 PASS (revs above; 17 dirty tree entries = other session churn, accounted). 5.2 UNPROVEN (code path verified `main.go:144-157` refuses trading; sandbox re-run not possible read-only). 5.3 PASS (mtime proof: binary 07:34:53.704 → RELEASE 07:34:53.724, order respected; order itself convention-only). 5.4 PASS (1 PID). 5.5 PASS (boot goldens PASS). 5.6 **FAIL** (no rev exposed in UI/API — bug reports can't be checked against running rev without a shell).

**§6 safety spine** — 6.1 PASS+2 missing fixtures (B3/B4 bypass paths untested). 6.2/6.3 PASS (plan doc has no gate fields; overlays re-validated). 6.4 PASS (6 enumerated account-selection sites all SIM-gated). 6.5 PASS (no widen/cancel action in grammar; breakeven refuses widening). 6.6 UNPROVEN live (code: bracket OCO on entry fill `VLTraderTCPClient.cs:894`). 6.7 PASS (caps master-independent). 6.8 **FAIL**: T1 calendar fail-OPEN — `currentT1Windows` returns nil (entries allowed) on slice/parse error (`auto_trader_calendar.go:104-122`). 6.9 PASS live (trade 515: 30025.00−29901.25 = +123.75 pts × $2 = +$247.50 = realized_pnl, to the cent). 6.10 UNPROVEN: in 48h journal only `clock_drift` fired; all other gates never fired live.

**§7 model I/O** — 7.1 PASS live (record 29840 system prompt opens with `## Clock 07:41 CT (12:41 UTC) — ALL times…`; sampled 29222 numbers all true). 7.2 PASS (one snapshotNow for SVP/levels/status/clock). 7.3 PASS (HEAD advisory header; old records show the forbid text — historical). 7.4 PASS. 7.5 PASS (birth-scoped + touch-gated, live prompt shows `CONSUMED · flipped(tradeable both directions)`). 7.6 PASS. 7.7 PASS (0 stale discards in 24h). 7.8 PASS (0 blank-reason waits in 24h). 7.9 CANNOT PROVE for the population; bounded sample: 7 waits, 0 had a viable 3:1 (max excursion 113 pts vs 165 needed, stop never hit) — sample says gates not suppressing obvious 3:1s in this window.

**§8 data provenance** — 8.1 PASS (0 flat session_profiles rows; placeholder guard `bar_cache.go:50-92`). 8.2/8.3 PASS tick-exact (live recompute: 2026-08-16 session H 30254.75 / L 30166.25 == plan PDH/PDL). 8.4 **FAIL** (plan-card grades are model-written; e.g. OR-L displays A while machine formula gives B; nothing re-grades). 8.5 PASS+gap ("no data" is log-only, never stated in prompt). 8.6 UNPROVEN live (formulas match NT8 semantics; no live NT8-vs-Go diff; MACD signal/histogram not exported).

**§9 answerability** — screens named for: today's plan, armed play, refusal reason (GateBlocksPanel/DecisionCard), owner changes, AI-vs-owner (RealignPanel), plan death (chips tooltip), burned levels (ZoneTable), errors (AlertCenter). GAPs: no counter for lost/guardrail-skipped decisions; gate-blocks are memory-only (reset on restart/rollover) — "why was X refused yesterday" unanswerable.

**§10 multi-instance** — 10.1 PASS (unique index date/session/strategy/version; digests trader-keyed). 10.2 PASS+latent (ActivePlanProvider replaced by per-trader sync.Map; FuturesBarsProvider/NakedPOCProvider still assignable globals). 10.3 PASS (CT hardcoded, no config step, no time.Local in prod). 10.4 PASS (unset→65/3.0). 10.5 PASS (additive idempotent migrations).

**§11 honesty** — UNPROVEN list kept (below); revs recorded start=end; read-only.

## The four closing questions
1. **Engine used what owner set?** Mostly yes (live proof: min-confidence 60 in prompt = DB; sessions override live). FAILs: FE displays stale constants (75 vs 65, hint 10 vs 2) and duplicated defaults (§2).
2. **Every number has a real origin?** Yes in sampled prompts (29222 + 29840); grades are model-written (§8.4), gate counters in-memory only.
3. **Every built thing has a live caller?** No — grid_* store orphan, `/plan/approve` un-exercisable, MAE/MFE never rendered, digest producer unproven (§3/4).
4. **Right unit/window/zone/scope?** Yes at HEAD for the trading paths (guards + tests + live prompt clock). Residuals: chat-path UTC table (§1.1), level_state cross-trader key (§1.8).

## Owner questions answered
- **S/D zones:** detectors exist and run — `SupplyDemandZones` `kernel/levels_zones.go:100`, `FairValueGaps`, `OrderBlocks`, assembled at `levels_assemble.go:63-66` and fed to BOTH the executor KEY LEVELS map and the planner's ranked table. They die at `levels_score.go:168`: any S/D/FVG/OB with zero confluence (no other level within ~3 pts) is dropped ("never stand alone"), then the max_levels=8 cap cuts the rest, and the planner is told "S/D & FVG are confluence, never standalone" (`planner_prompt.go:163`). Live KEY LEVELS block (record 29840): 8 rows, zero zone rows. So: computed → filtered → never surfaces. Suspicion confirmed with one correction: the planner DOES receive them; the confluence filter + cap + instruction kill them.
- **Timezone fix reaches the model?** Yes — post-deploy record 29840 system prompt: `## Clock 07:41 CT (12:41 UTC) — ALL times in this prompt are CT…`. Post-deploy model outputs: 0 lunch mis-citations (n=1 record).

## UNPROVEN (6)
1. 5.2 sandbox refusal re-run (code verified; prior deadbeef run is a secondhand claim). 2. 6.6 bracket-at-exchange live (no NT8 order-history proof). 3. 6.10 all gates except clock_drift have never fired live. 4. 7.9 counterfactual for the full wait population (bounded sample: 0/7 viable). 5. 8.6 live NT8-vs-Go indicator diff (MACD signal/histogram not exported). 6. 7.1 truth of pre-deploy historical records (only HEAD code + 2 sampled records proven).

## REMAINING (numbered, by cost)
1. **HIGH — T1 calendar fail-OPEN** (`auto_trader_calendar.go:104-122`): red-news blackout silently vanishes on slice/parse error → entries allowed in FOMC windows. Fix: fail-closed + P1 alert. Size S.
2. **HIGH — level_state cross-trader** (`store/level_state.go:54`): add trader_id to level_key; migrate rows. Size S-M.
3. **HIGH — B3/B4 bypass fixtures absent** (§6.1): add fixtures for order-dedup/rate-breaker + stale-data bypass paths. Size S.
4. **MED — chat-path Time(UTC) table** (`market/data.go:536`): include market/ in tz guard + TableTimeCT. Size S.
5. **MED — UI shadows backend** (§2.1/2.5): confidence 75/hint 10 mirror the backend or render server truth. Size S.
6. **MED — dead wires** (§3): grid_* store, /plan/approve, top-traders/competition FE calls — wire or remove. Size M.
7. **MED — MAE/MFE/adherence never rendered + dropped from digest** (§4.4). Size M.
8. **MED — no rev in UI** (§5.6). Size S.
9. **MED — defaults duplicated** (§2.3): single consts. Size S.
10. **MED — gate-blocks memory-only** (§4.3): persist or daily-journal. Size M.
11. **LOW — plan grades model-written** (§8.4): display machine grade alongside. Size S.
12. **LOW — no-data never stated in prompt** (§8.5). Size S.
13. **LOW — digest producer unproven, calendar 08-17 slice missing** (§3.4). Size M.
14. **LOW — armed dot is plan-derived** (§4.5): label it. Size S.
15. **LOW — `fill` alert never emitted** (§3.3). Size S.
