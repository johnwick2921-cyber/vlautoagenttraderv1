# VL Day-Plan — full function audit vs contract (read-only, pre-Monday)

**LINE 1 — MONDAY CORE READY, SYSTEM NOT: 0 hard blockers to the 08:25 read (a
plan WILL appear, render, and be cited advisory), but 10 material DEAD WIRES leave
the value-add non-functional** (owner-edits→execution, the whole learning loop,
alerts, calendar/blackouts, cross-session level memory, 6+ config controls,
model-id pinning). Same class as the normalize bug — built + unit-tested, never
wired end-to-end.

**HEAD** 10ce3886 · **audit** 7-agent read-only matrix + my sandbox + spot-checks ·
**live db** mode=ro · binary rebuilt today 10:05 (Sat), all day-plan tables empty
(mostly expected — P3 planner dormant until ★2 rebuild). Contract =
docs/VL-DAYPLAN-FULL-SPEC.md.

## PROVEN working (the Monday core)
- **Sandbox end-to-end [A]** (synthetic bars, no paid API): detectors→scorer (8
  real levels) → `BuildPlannerPrompt` (KEY LEVELS table + REGIME) → `ParsePlanDoc`+
  `ValidatePlanDoc` → `AppendPlan`→read-back (test db) → `RenderPlanBlock`+
  `RenderPlanStatus`. Only the live model call + live NT8 feed remain (fire Mon).
- day_plan **dual-codec round-trip** ✓ (store Marshal/Unmarshal at root + the FE
  `normalizeStrategyConfig` now preserves it, post settings-fix). plans/overlays
  **single-writer** ✓. **scenario evaluator** `EvaluateLevelFacts`→executor tail ✓.
  **KEY LEVELS** reads `DayPlan.PlanEnabled`+`MaxLevels` live ✓ (Sat omit is the
  honest no-bars skip, `engine_analysis.go:362`). **Stats honesty gate** fully
  built ✓ (WARMING short-circuits before significance, Bonferroni 0.05/8). **Ask-
  Planner** bare→DEFEND-never-patches ✓ (unit). **WAL** = `wal` live ✓. **B2 armor**
  on resolved plan ✓. **Assembler now honors** max_levels/min_grade/timeframes ✓
  (settings-fix). **/api/plan/today** graceful found:false ✓.

## DEAD WIRES — built + tested, NOT reachable end-to-end (verified [A])
1. **Learning loop writer unreachable** — `recordExcursionForClosedSymbol` (MAE/MFE
   + **adherence grade** + **matched-random**) sole caller is `recordPositionChange`
   (decision.go:453), reached ONLY on an **AI-emitted close**. Under day_plan
   `skipWhileOpen` suppresses the decision cycle while holding, and real exits go via
   NT8 OCO SL/TP (`close_sync.go`) / EOD-flat (`clock.go` direct `CloseLong`) —
   **neither calls it**. Net: P2.4/P5.5/P5.6 write **nothing** on a normal hold→exit.
2. **Owner overlays never reach the executor** — `installActivePlanProvider`
   (planner.go:442) renders `row.Doc` (BASE), **no `ApplyOverlayPatches`**. Overlays
   feed only GET /plan/today (the card). Contract §42–48 "owner wins on execution"
   unmet; edits are cosmetic. (Ask-Planner also questions the base, not plan_final.)
3. **Notifications never fire** — `AlertStore.Emit` has **zero production callers**;
   bell/feed/ack render but no P0/P1 alert is ever emitted (GAP-HUNT 🔔 dead).
4. **Calendar producer dead** — `calendar.DefaultFetch`/`FetchWeek` +
   `CalendarSliceStore.SaveSliceIfAbsent` have zero callers; planner READS
   `GetSlice` (planner.go:331) over an always-empty store → **no T1 red→HARD
   no-trade blackout** (§80), and half-day flat pull-in (`reg.HalfDays`) starved.
   `calendar_slices=0`.
5. **Level-state memory dead** — `LevelStateStore` (EnsureLevel/RecordPlay/
   MarkConsumed/DecrementFreshness/ListValid/ReArmEligible) + `Store.LevelState()`
   have zero callers; the 17:00-CT writer is deferred. Cross-session freshness /
   consumed / re-arm (§21, §38–39) **never accumulate** — a level burned in one
   session can return "fresh" next session.
6. **Admin session registry not loaded** — every gate hardcodes
   `DefaultSessionRegistry()` (NY-on, ASIA/LONDON off) with TODO(P4);
   `LoadSessionRegistry`+`SessionRegistryConfigKey` round-trip in test only. Admin
   edits to window/read/flat/enabled/killzones can't reach the live gate.
7. **Config controls display-only** (editable + persisted post settings-fix, but NO
   backend reader): `plan_mode` · `proximity_filter_atr` (activation window
   hardcodes k=1.5) · `scenario_cap` (prompt hardcodes 1..3) · `sessions_enabled`
   (hardcoded NY) · `approval_required` (no gate) · `evening_digest` (always writes)
   · per-session `enable`/`replan_cap`/`plan_mode`/`acceptance_rule`/`max_trades`
   (only `min_grade` is read) · `replan_cap` (executor "re-plans left" hardcodes 2).
   *(My settings-fix wired max_levels + min_grade + timeframes — those 3 ARE live.)*
8. **Digest correctness** — Friday daily roll is structurally unreachable (16:00 CT
   CME break + weekend); when it fires (Mon–Thu ~17:00) the P&L window `sinceMs`
   already rolled → summarizes ~empty activity; a Sunday-17:00 reopen may fire a
   **spurious early NY read** (dedup only on trade_date) → Monday's plan could be
   built Sun-evening instead of at 08:25 (needs a live weekday check — flagged).
9. **Model-id pinning** — resolved+logged planner model is `deepseek`, a **provider
   alias**, not an exact model string (§125). No stats-window reset on model change.
10. **Regime inputs partial** — VIX has no feed (§77 vix_level / expected_range
    inert); realized-vol-vs-20d baseline never supplied to `ComputeRegime`.

## MISSING (contract line, no implementation) — beyond the dead wires above
- ⚡ Conflict-chip **backend** (§45): only the FE `detectConflicts` exists; no
  owner-wins-on-execution + AI-ghosted + logged path.
- plan_mode **direction/strict** enforcement (§48) — unbuilt (Stage-3, by-design).
- Blind-mark **10-day** calibration flow (§49) — unbuilt (deferred post-★2).

## Not blockers (correctly dormant-by-design until ★2 rebuild)
Planner reads / plan writes / executor citation / KEY LEVELS / MAE-MFE writer are
`dayPlanEnabled`-gated — they light at the ★2 rebuild (this weekend) and the Monday
08:25 NY read (CME open). The CME-closed skip swallows the day-plan block only when
CME is closed (Sat/break) — NY 08:25 is unaffected.

## Verdict for Monday
The **08:25 read will produce a shown, advisory-cited plan** — that path is intact
and PROVEN. But treat the day-plan as **core-only**: owner edits won't change
execution, the learning loop logs nothing, no alerts fire, no calendar blackouts,
no cross-session level memory, and 6+ Studio controls are inert. Recommend a
follow-up "wire-up" train (calendar producer · overlay→executor fold · learning-
loop writer on the real close path · Emit call-sites · level-state writer · the
inert config readers) before relying on those features. Read-only audit — no code
changed; only this report is committed.
