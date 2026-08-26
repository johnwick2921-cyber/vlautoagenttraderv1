# Regime wave — executor gate order + planner wake-ups (2026-08-21)

Observed from the shipped code (this order is load-bearing — do not reorder
without a wave). **As-built at the 1h-wave cutover rev (see `deploy/RELEASE`)**
— updated 2026-08-25 for the G6 removal, W6 wake semantics, and the R4 gate.

## Executor entry gate chain (kernel/engine_position.go, `validateDecision`,
open actions)

1. Action enum + leverage/size sanity (pre-existing)
2. SL/TP sanity (pre-existing)
3. F1 real R:R from the actual entry reference (pre-existing)
4. **Min-confidence gate** (pre-existing, `minConfidence > 0 && Confidence < minConfidence`)
5. **G1 — HTF veto** (`htf_veto`: opposed vs CONFIRMED 1h structure → refused;
   RANGING/unconfirmed/detector-unavailable FAIL OPEN with WARN). Studio
   `regime.htf_veto` (default ON); TF = env `HTF_VETO_TF` (default 1h).
6. **G4 — transition stand-down** (`transition_standdown`: plan-direction
   entries paused while an unconfirmed counter-trend CHoCH/MSS is open on the
   plan's bias TF 15m; counter-direction untouched; closes on flip/re-plan,
   BOS resumption, or `TRANSITION_MAX_MIN` (45)). Studio
   `regime.transition_standdown` (default ON).
7. **C6 — executor dead-plan gate** (2026-08-25): while the active plan is
   machine-dead (or planless with day_plan on), new entries are refused;
   management proceeds.
8. **R4 — min-scenario-quality gate** (2026-08-25): `min_scenario_quality`
   floor (A|B, default C = no restriction) refuses entries citing a scenario
   graded below the floor. Fail-open on off-plan/unknown citations.
9. Sizing caps + execution (pre-existing).

Advisory-only (never gates): G2 structure line (prompts), G5 consumed-level
policy (demotion + badge), G7 flip-eval freshness (skips the transition
EVALUATION, never an entry), G8 watcher hooks (zero order authority).

## Planner wake-ups

1. Session first read (read window, once per session-day)
2. Re-plan on death (capped)
3. Owner reread / reset
4. **G4.6 — structure MSS on the plan's bias TF** (`trigger_reason:
   "structure_mss"`, one wake per MSS event)

## Regime boot ledger knobs (value + source, logged at boot)

- `htf_veto` ON (Studio default) · `htf_veto_tf` 1h (env)
- `transition_standdown` ON (Studio default) · cap 45min (env)
- flip hysteresis hold 30min (env `FLIP_MIN_HOLD_MIN`)
- `loss_streak` 4 (Studio default; 0=off) · pause 60min (env)
- structure engine TFs 5m/15m/1h · swing k=2 · min-swing 0.25×ATR · MSS body 1.5×ATR (env-tunable)
- flip-eval freshness cap 90s (env `FLIP_EVAL_MAX_STALE_S`)
