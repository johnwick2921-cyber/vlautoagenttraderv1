# DAY-PLAN CAMPAIGN — P3 FINAL: executor injection + advisory + lifecycle-core

**Date:** 2026-08-15 · **Repo:** /home/hoang/nofx · **Branch:** main
**Range:** `c65d9b2b` → `0a292148` · 3 feature commits (P3.4/P3.5/P3.6-core)
**Contract:** [docs/VL-DAYPLAN-FULL-SPEC.md](../../VL-DAYPLAN-FULL-SPEC.md)

## LINE 1 — VERDICT
**The plan→executor loop is closed: the executor now reads the active plan
(RECON #4 reorder), cites scenarios (advisory), and the plan has a lifecycle
(activation window, death→re-plan, restart recovery).** Suite + `-race` green; the
ONLY golden change is the new plan-active `futures_mnq_plan.golden` (existing
byte-identical); all gated on day_plan → dormant until ★2. P3.1–P3.5 ✅ and the
P3.6 core ✅; P3.6 finishing sub-features (digests, scenario re-arm, sticky levels,
night mode) remain a short follow-up.

## STEP 0 gate
PASS — HEAD `c65d9b2b` · clean · bot on the ★1 binary (PID 778475) untouched.

## Shipped
### P3.4 — executor plan injection + RECON #4 reorder — `d0d96a10`
`RenderPlanBlock` (byte-stable prefix) + `RenderPlanStatus` (dynamic tail: per-
level P0.4 evaluator facts) + `ActivePlan`/`ActivePlanProvider`. The futures prompt
reorders WHEN a plan is active: PLAN BLOCK joins the cached prefix (before
Available Data) and SVP + KEY LEVELS + PLAN STATUS move to the END (Live map). No
active plan → prompt unchanged. `installActivePlanProvider` reads the store's
latest ACTIVE plan for the current session.

### P3.5 — advisory mode — `d2d4a975`
`Decision.cited_scenario` (additive; absent=empty). `ClassifyCitation` (pure):
empty/off-plan/unknown→off-plan, known scenario→matched iff the action direction
aligns. `recordPlanCitation` logs + advances match-rate counters via B6
(plan_matched / plan_off_plan / plan_cited_mismatch), advisory-only (never gates).

### P3.6 core — `0a292148`
`ActivePlanLevels` = the 1.5×dATR activation window (Go filter, no plan mutation),
shown in PLAN STATUS. `PlanIsDead` = every level TOUCHED and accepted through
(fixes the distant-never-reached-level false-positive). Re-plan-on-death in the
scheduler: a dead active plan re-plans up to `replan_cap`, then a NO-TRADE plan +
alert. **RESTART RECOVERY (mandatory fixture):** the plan lives in sqlite and
scenario states are a pure function of plan+bars — `TestPlanRestartRecovery` proves
states recompute IDENTICALLY across a store reopen (mid-session restart).

## EXIT BAR
- `go build ./...` ✓ · `go vet ./...` ✓ · `go test ./...` ✓ ·
  `go test -race ./trader ./kernel ./store` ✓.
- **Goldens (deliberate):** existing `futures_mnq_empty.golden` /
  `futures_mnq_keylevels.golden` / `golden/*.txt` **byte-identical** (reorder needs
  an active plan). **NEW** `futures_mnq_plan.golden` (+87) — the ONLY diff, the
  reordered plan-active prompt.
- config-truth: `cited_scenario` is an additive optional Decision field
  (back-compat proven).

## SAMPLE ① — assembled planner prompt (`BuildPlannerPrompt`)
```
# DAY-PLAN READER — CME MNQ futures
## Session   trade_date 2026-08-14 · session NY · read 08:25 CT · price 15600.00 · dATR 120.0
## Regime    REGIME: trend D=up 1h=up · ATR14=118.0 (NORMAL p55) · VIX=n/a
## Ranked levels   15620.00 PDH grade A fresh +20.0 · 15450.00 PDL grade A fresh -150.0
## Calendar   13:00 USD T1 — FOMC (HARD no-trade blackout)
## OUTPUT — one JSON object, reasoning FIRST  { reasoning, bias, levels, scenarios, no_trade, death_condition, day_type }
```

## SAMPLE ② — schema-valid plan JSON (passes `ValidatePlanDoc`)
```json
{"reasoning":"...","bias":{"direction":"long","conviction":"medium","flip_condition":"2x5m < 15480"},
 "levels":[{"price":15620,"label":"PDH","grade":"A","instruction":"fade"}],
 "scenarios":[{"id":"S1","trigger":"sweep 15480 reclaim","condition":"sweep_reclaim","direction":"long","target_chain":[15550,15620],"invalid":"2x5m<15470","quality":"A"}],
 "no_trade":["first 5m","12:00-13:30 lunch"],"death_condition":"acceptance above 15620","day_type":"balance"}
```

## SAMPLE ③ — executor prompt: prefix + PLAN BLOCK + dynamic tail (from the golden)
```
# DAY PLAN (NY) — follow it; entries per plan only
Bias: long (medium) · flips: 2x5m < 21480
Levels:  21520.00 PDH [A] fade · 21480.00 ONL [B] reclaim-long
Scenarios:  S1 [A] sweep_reclaim long: sweep 21480 reclaim → 21500.00,21520.00 · invalid 2x5m<21470
No-trade: first 5m · 12:00-13:30 lunch
Plan dies if: acceptance above 21520
Cite rule: your decision JSON MUST include "cited_scenario" = "S1"|…|"off-plan".

# Available Data … (rules, decision, output — the cached prefix continues) …

# Live map (dynamic — re-read each bar)
SVP (today's session…): POC 21500.00 VAH 21503.75 VAL 21497.50
KEY LEVELS (map, nearest-first; price 21500.00):  21500.00 PDC A fresh +0.0 · …
# PLAN STATUS (live)   price 21500.00 · re-plans left 2
  21520.00 PDH: dist +20.0 · sweep=F · closes-beyond 0 · acceptance 0/2 · valid
```

## What remains (P3.6 finishing — short follow-up, then P4)
Full digest writers (3-line session close · daily roll-up · tapered week chain →
feed P3.3's assembler DigestChain) · scenario re-arm via level-state
(times_triggered · freshness A→B→C · consumed-on-acceptance · 20m cooldown) ·
sticky owner levels across sessions · night mode.

All shipped code is additive + dormant (gated on day_plan) — the running bot is
unaffected. vlauto: DEFERRED.
