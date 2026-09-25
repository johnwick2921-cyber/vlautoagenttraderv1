# 2026-08-26 · Pipeline Confirmation (CTO 7-stage model vs deployed rev 657e813b)

Read-only verification against live code at the deployed revision. Answer format: stage · CTO claim · verdict · file:line.

## 1. Wake triggers — CONFIRM (5) + notes

| Trigger | Call site | Budget rule |
|---|---|---|
| 1. Clock read (session first read) | `trader/auto_trader_planner.go:229` (`runPlannerRead`) inside `maybeRunSessionReadsAt` (`:180`), gated by `inSessionReadWindow` (`:195`), CME-open, `sessionRunnable` | once per session-day (plan-store dedupe) |
| 2. Death/flip re-plan | `:238` `describeActivePlanDeath` → `runPlannerReadWithCtx(…detail.Killer…)` `:275` | `replanCapFor`/`MayReplanFrom` (`:244-259`); exhausted → `writeNoTradePlan` (`:260`) |
| 3. Owner reread/reset | `api/handler_plan.go:997` `handlePlanReread`, `:1049` `handlePlanReset` (API, not the loop) | reread budget via `planRulesWithCap` (`:142`); reset re-arms baseline (`GetResetBaseline`) |
| 4. structure_mss (G4.6) | `trader/auto_trader_transition.go:144` `maybeWakePlannerOnMSS` (called `auto_trader_planner.go:286`) | one per MSS event, dedupe plan-version+event |
| 5. W6 level events | `trader/auto_trader_wake_levels.go:235` `maybeWakePlannerOnLevelEvents` (called `:291`, death-first ordering) | UNLIMITED/budget-free, `wake_min_interval_min` throttle (PIPELINE-MAP.md:60-63) |

**No 6th path.** The `post_exit` rescan kick is an executor cycle trigger (`cycle_trigger=post_exit`, PIPELINE-MAP.md:169), not a planner wake. `replans_exhausted` (`:617`) is a terminal no-trade write, not a wake.

## 2. Detector census — CONFIRM, count updated

`kernel/levels_assemble.go:47/94` (`AssembleScoredLevels`/`MinGrade`) wires **10 detector groups**:
1. `ExtractMultiDayLevels` — PDH/PDL/PDC/RTH-H/RTH-L/AS-H/AS-L/LDN-H/LDN-L/ONH/ONL/PWH/PWL/PMH/PML
2. `RoundNumberLevels` · 3. `OpeningRangeLevels` (OR-H/OR-L/IB-H/IB-L) · 4. `GapLevels` · 5. `EqualHighsLows` (EQH/EQL) · 6. `SupplyDemandZones` · 7. `FairValueGaps` (FVG/iFVG) · 8. `OrderBlocks` · 9. **`VolumeLevels` (Pack B, `kernel/levels_volume.go`) — SETT ✅ live, MID-O ✅ live, eVWAP ✅ live, SessionVWAP±1σ, pdPOC/VAH/VAL, nPOC retire-on-touch, pdVWAP** · 10. extras: durable nPOC (`kernel/naked_poc.go`) + `DetectHTFLevels` (EQH/EQL/S/D/FVG/OB on 15m/30m/1h/2h/4h/6h/8h/12h, `levels_assemble.go:157`).
Live proof: `nPOC·wk·2026-08-18` seated in today's tables; SETT/MID-O are detected but lose seat races when not in-band. Boot: `🎛 volume wave: detectors=on`.

## 3. Scoring order — CORRECT (CTO's list has order errors)

Actual order in `scoreLevelsPool` (`kernel/levels_score.go:~285`):
**proximity lock FIRST** (`|level−price| ≤ 1.5×dATR`) → per level: freshness (`freshMult`; zones use the Pack-B ladder `1.0/0.6/0.3/0.15` `:339`) → family confluence (distinct families, cap 3) → zone-standalone rule → evidence: zones = `zoneEvidence`(**reversal ×1.1 inside** `:207`) × `zoneSizeMult` × freshness × confluence × `zoneTFMult`; lines = `typeEvidence` × freshness × confluence × HTF ×1.2 → `gradeFromScore` → zone floors/caps → **B2 Tier-1 gate** → **role** (`RoleFor`) → collapse → sort → `seatHTF` → **`SeatVolumeFamily` (new)** → `seatBothSides` → topN(8) → nearest-first.
CTO's claimed tail "topN(8)→proximity(1.5)" is inverted — proximity is step 1; reversal is inside evidence, not a separate stage; `SeatVolumeFamily` sits between seatHTF and bothSides.

## 4. Planner prompt sections (verbatim, render order — `kernel/planner_prompt.go:BuildPlannerPrompt`)

`## Session` → `## Regime` → `## Indicators (executor mirror — …)` ⚠conditional → `## Ranked levels (Go-graded; you never re-sort)` → `## Consumed levels (already role-flipped — plan AROUND them…)` ⚠ → `## Level roles (machine-assigned, 5-line playbook)` → `bias_ctx:` line → `## Structure (1 line/TF)` ⚠ → `## HTF zones (nearest first — confluence references, NEVER standalone triggers)` ⚠ → `## Auction story` ⚠ → `## Calendar (this session's window — times CT)` → `## Recent context (digest chain)` ⚠ → `## Owner note` ⚠ → `## Prior plan invalidation (MANDATORY context)` ⚠ → `## Prior plan levels (continuity — carry over or re-anchor…)` ⚠ → `## Anchor roles (week evidence, advisory)` → `## OUTPUT — one JSON object, reasoning FIRST, no prose outside it`.

## 5. Plan schema — CONFIRM with corrections

`PlanDoc`: reasoning · bias{direction,conviction,flip_condition} · levels[]{price,label,grade,instruction,machine_grade} · scenarios[]{id,trigger,condition,direction,target_chain,invalid,confirm{rule,ref_price,side},quality,consumed,**fvg{lo,hi,ce,entry_mode,displacement_atr,origin_level,direction}**} · no_trade[] · death_condition · day_type · death{price,side,rule,flip_to} · flip{…}.
Enforced: all enums (`plan_doc.go:54-76`), caps, both-side levels, continuation-on-gap, duplicate seats, reachable targets, label provenance, flip-fired bias, confirm{} after grace, **fvg re-verification**. Advisory: instruction verbs, quality letters, reasoning prose.
**Condition types = 7, not 5**: reclaim · hold · sweep_reclaim · reject · acceptance · breakout_retest · **fvg_entry** (new today, PR #79).

## 6. Post-birth lifecycle — CONFIRM

Card: `api/handler_plan.go` handlePlanToday → `SessionPlanCard` (+ `ScenarioList`, chips, `fvg_states`, `touch_state`). Executor consumption points per cycle: PLAN STATUS (`kernel/plan_render.go:157`), confirm lines (`kernel/plan_confirm.go:71`), TOUCH lines (`kernel/touch_telemetry.go:RenderTouchLines`), fvg lines (`kernel/fvg_entry.go:RenderFvgEntryLines`), bias_ctx (`kernel/engine.go:SetBiasContext` → `engine_prompt_futures.go`). Birth = write in `runPlannerReadCoreWithFactsGrades` (`auto_trader_planner.go:836`); mid-session death checked every cycle at `:238` → re-plan (capped) or no-trade; consumed levels role-flip in place.

## 7. Gate chain — CORRECT vs the claimed order

`validateDecision` (`kernel/engine_position.go:34`) actual order:
1. action enum + leverage/size sanity (`:44-113`)
2. SL/TP sanity (`:113-120`)
3. **F1 R:R** (`:125+`, live trace: `engine_position.go:178 📐 R:R eval MNQ open_long: entry=29282.5000 SL=29247.2500 …` 10:53:22)
4. **min-confidence** (`:190`)
5. **min-SL (A3)** — ATR leg then anchor-clearance leg (`:195-226`)
6. **HTF veto** (`:245-250`)
7. **transition stand-down** (`:266`)
8. **C6 dead-plan** (`:275`)
9. **R4 min-scenario-quality** (`:289`)
10. sizing (`:300+`).
The session gate + multi-account + dead-man watchdog live in `runCycle` (`auto_trader_loop.go:91,107,175`) BEFORE validateDecision — not inside it. The CTO's "R/R after minconf" is inverted; sanity legs and gates 8-9 are unmentioned. PIPELINE-MAP.md is stale for gates 5/9 (pre-A3/pre-R4).

## Components the CTO's model does not mention

A5 prompt-ownership assertion (per-cycle, trader-scoped) · the ONE-SNAPSHOT contract · dodge (defer near bar-close) · dead-man watchdog · sticky owner levels + overlays (carry/re-anchor on re-plan) · level_stats nightly + touch_episodes telemetry (the forward-validation layer) · gate-block counters/telemetry · `SeatVolumeFamily` + `Seat1HZone` seat guarantees · BOOT INTEGRITY golden assertion · post-exit rescan kick.
