# DAY-PLAN CAMPAIGN — P3 · PLANNER CORE (checkpoint: P3.3 done, P3 3/6)

**Date:** 2026-08-15 · **Repo:** /home/hoang/nofx · **Branch:** main
**Range:** `cad1dc2f` → `5d06b063` · 1 feature commit (P3.3)
**Contract:** [docs/VL-DAYPLAN-FULL-SPEC.md](../../VL-DAYPLAN-FULL-SPEC.md)

## LINE 1 — VERDICT
**The PLANNER CORE (P3.3) is built and pushed — the planner can assemble its input
package, call the pinned model, strictly validate the plan JSON, fail-closed to a
NO-TRADE plan, and write append-only plan rows; the scheduler fires per-session
reads.** Suite + `-race` green; no golden touched; all gated on day_plan → dormant.
Checkpointed at the P3.3 boundary — the executor-side integration (P3.4 injection
reorder + golden regen, P3.5 advisory, P3.6 lifecycle + restart recovery) builds
on this core and is a fresh-session amount of work.

## STEP 0 gate
PASS — HEAD `cad1dc2f` · tree clean · bot on the ★1 binary (PID 778475) untouched.

## Shipped — P3.3 (read jobs + the planner call) — `5d06b063`
- **`kernel/plan_doc.go`** — `PlanDoc` schema (bias+flip, graded levels w/
  instruction verbs, 1–3 scenarios in the formal grammar, no-trade, death
  condition). `ParsePlanDoc` extracts the JSON from prose/code-fences
  (brace-balanced, string-aware) + strict `ValidatePlanDoc` (enum + count rules).
  `NoTradePlanDoc` = the fail-closed doc (itself valid → a real row).
- **`kernel/planner_prompt.go`** — `BuildPlannerPrompt` assembles the input
  package (session meta · regime · Go-ranked levels · structure · auction story ·
  session-sliced calendar w/ T1 blackouts · digest chain · owner note · WARMING)
  + the reasoning-first schema-strict output contract.
- **`kernel/levels_assemble.go`** — `AssembleScoredLevels` exposed (shared by the
  executor block and the planner).
- **`trader/auto_trader_planner.go`** — `runPlannerReadCore` (≤2 retries →
  FAIL-CLOSED NO-TRADE plan + `IncGateBlock('planner_fail_closed')` alert event;
  never stale/uncalibrated) writes the append-only plan row `(trade_date, session,
  version)` with `prompt_hash` + resolved `model_id`. `assemblePlannerInput` builds
  from stored+cached data (the 16:55 closed-market path). `maybeRunSessionReads` =
  the per-session scheduler at registry read times (enabled sessions, once/day,
  idempotent via the plan store), wired into `runCycle`.

## EXIT BAR (partial)
- `go build ./...` ✓ · `go vet ./...` ✓ · `go test ./...` ✓ ·
  `go test -race ./trader ./kernel ./store` ✓.
- Goldens: **none changed** (P3.4's deliberate reorder is next-session).
- config-truth: no new config field this checkpoint (`planner_model` proven P0.1/P3.2).

## SAMPLE ① — REAL assembled planner prompt (`BuildPlannerPrompt`)
```
# DAY-PLAN READER — CME MNQ futures
...
## Session
trade_date 2026-08-14 · session NY · read 08:25 CT (live) · price 15600.00 · dATR 120.0
## Regime
REGIME: trend D=up 1h=up · ATR14=118.0 (NORMAL p55) · VIX=n/a
## Ranked levels (Go-graded; you never re-sort)
  15620.00  PDH            grade A  fresh    +20.0
  15450.00  PDL            grade A  fresh    -150.0
## Calendar (this session's window)
  13:00 USD T1 — FOMC (HARD no-trade blackout)
## Owner note
  respect PDH, don't chase
## OUTPUT — one JSON object, reasoning FIRST, no prose outside it
{ "reasoning": ..., "bias": {...}, "levels": [...], "scenarios": [...], "no_trade": [...], "death_condition": ..., "day_type": ... }
Rules: levels chosen ONLY from the ranked table; S/D & FVG are confluence, never standalone. Respect the no-trade windows.
```

## SAMPLE ② — schema-valid plan JSON (passes `ValidatePlanDoc`)
```json
{
  "reasoning": "Overnight swept ONH then reclaimed; balance below PDH. Fade edges, long the reclaim.",
  "bias": {"direction": "long", "conviction": "medium", "flip_condition": "flips short on 2x5m < 15480"},
  "levels": [{"price": 15620, "label": "PDH", "grade": "A", "instruction": "fade first tap"},
             {"price": 15480, "label": "ONL", "grade": "B", "instruction": "reclaim-long"}],
  "scenarios": [{"id": "S1", "trigger": "sweep 15480 then reclaim", "condition": "sweep_reclaim",
                 "direction": "long", "target_chain": [15550, 15620], "invalid": "2x5m < 15470", "quality": "A"},
                {"id": "S2", "trigger": "reject 15620", "condition": "reject", "direction": "short",
                 "target_chain": [15550], "invalid": "acceptance > 15625", "quality": "B"}],
  "no_trade": ["first 5m", "12:00-13:30 lunch"],
  "death_condition": "acceptance above 15620 kills the fade thesis",
  "day_type": "balance"
}
```
**Sample ③ (reordered executor prompt with prefix+plan-block+status-tail) is
DEFERRED** — it is produced by P3.4 (the injection reorder + golden regen).

## What remains (next session)
- **P3.4** executor injection reorder (RECON #4): dynamic blocks (SVP, market
  tables) → prompt-END; byte-stable PLAN BLOCK joins the static prefix; PLAN STATUS
  tail from the P0.4 evaluator (sweep/closes-beyond/acceptance n/2, distances,
  window, re-plans left). Goldens regenerated DELIBERATELY (both states). Cache-hit
  telemetry. **(produces sample ③)**
- **P3.5** advisory mode: executor cites `scenario_id` | "off-plan"; match-rate via
  B6; plan restricts never compels.
- **P3.6** lifecycle: re-plan on death (cap 2/session), activation window k=1.5×dATR,
  scenario re-arm via level-state, sticky owner levels, digest writers, night mode +
  restart recovery (mid-session-restart fixture).

All shipped code is additive + dormant (gated on day_plan) — the running bot is
unaffected. vlauto: DEFERRED.
