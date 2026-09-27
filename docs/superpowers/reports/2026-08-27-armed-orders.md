# Wave 2 — Armed-Order Executor (the flagship)

**Branch:** `feat/armed-orders` · **PR:** #86 · **2026-08-27**

> The LLM leaves the execution path. Scenarios compile to RESTING orders,
> managed tick-level by Go. NT8 stays the sole execution venue (SIM).

## Status

| Phase | State | Proof |
|---|---|---|
| 0 recon | done | bridge audit — limit entry / cancel / modify-TP / order_update were missing (≈180 C# lines < 300 gate → proceed) |
| 1 arming contract | done `ecb1961b` | `PlanArmSpec`, `ArmSpecValid`, `ScenarioQualityRank` |
| 2 executor + wire | done `ecb1961b` | Go wire frames + C# dispatcher branches + placement engine |
| 3 planner/UI context | done `24579b85` | arm{} schema, ARMED: prompt lines, `/api/plan` armed map, card chips ⏳/📌/⚡/✕ |
| 4 test matrix | done `fa2762e0` | order_update transitions, churn/reconnect predicates + reconcile seam, long/short R:R twins |
| 5 cutover | pending owner live | see protocol below |

## Phase 2 highlights (the execution path leaves the LLM)

- **Wire (protocol v3, lockstep Go⟷C#):** `SignalPayload.OrderType/LimitPrice`,
  `FrameCancelOrder`, `FrameModifyBracket`, `FrameOrderUpdate`; TCP server
  `order_update` router + subscriptions.
- **TCPTrader:** `PlaceLimitEntry` (bound-account + SIM + B3 guard),
  `CancelOrder`, `ModifyBracket`, `OrderUpdates()`.
- **Executor:** gate AT ARM TIME (ArmSpecValid → plan_mode direction → quality
  floor → R:R → min-SL×ATR5m → HTF veto); armed→working inside
  `ARM_PLACE_TICKS`=100-tick band; stale-working cancel after
  `ARM_WORKING_STALE_MIN`=15m (reconnect reconcile); order_update event machine
  — filled → lineage (`SetPlanLinkFull` band `armed_fill`, **no
  `stale_reeval`**), rejected/cancelled → disarm with reason.
- **Churn guard (2.1):** working arm's bracket re-modified only when the plan
  re-spec'd SL or TP by ≥ 2 ticks.

## Env knobs (zero literals)

`ARM_PLACE_TICKS`=100 · `ARM_WORKING_STALE_MIN`=15 · boot line
`⚔️ armed_orders=on place_band=100t stale_working=15m`

## C# changes — FLAGGED

`ninjascript/VLTraderTCPClient.cs` carries NEW dispatcher branches
(`cancel_order`, `modify_bracket`), `order_type`/`limit_price` parsing, and
`order_update` frames for all states. **The owner must recompile at cutover**
(NinjaScript Editor F5 + FULL NT8 restart). **Partner lockstep:** when
`vlautoagenttraderv1` syncs this wave, Binnie must recompile his AddOn the same
way — a stale AddOn silently ignores the new signals.

## Tests added (phase 4)

- `TestArmedOrderUpdateTransitions` — filled/rejected/cancelled event machine.
- `TestArmedGateRRShortTwin` + `TestArmedOrderUpsertAndGateRR` /
  `TestArmedGateRefusesBadRR` — long/short R:R + quality + plan_mode twins.
- `TestArmedChurnPredicate` / `TestArmedStalePredicate` — pure predicates.
- `TestArmedReconcileStaleWorking` — reconcile wire via the `cancelFn` seam
  (stale row cancelled with reason, fresh row untouched).
- `TestArmedCancelOnDormant` / `TestArmedCancelOnNoActivePlan` — 1.4/2.4.

## Cutover protocol (owner live)

1. **Flat check:** `GET /api/positions` → `[]`.
2. **Owner (NT8):** NinjaScript Editor F5 recompile → FULL NT8 restart → confirm.
3. **Go:** merge PR #86 → dev, build at the merge commit, `deploy/RELEASE` =
   binary rev, move old binary aside, `kill -9 <pid>` (systemd relaunches).
4. **Verify:** TCP link up, boot lines `🔐 BOOT INTEGRITY OK` +
   `🧬 plan lifecycle …` + `⚔️ armed_orders=on …`, one `order_update` frame
   observed.

## E-proofs (collected during/after cutover)

- **E1** full chain quoted: authorize → arm → 📌 working visible in NT8 orders
  tab → ✕ cancel on veto/dormant **or** ⚡ fill with `armed_fill` lineage.
- **E2** a forced cancel demonstrated (veto / dormant / stale-window).
- **E3** grep: zero `stale_reeval` on the `armed_fill` class.

## Parked work (named stashes on `feat/armed-orders`)

Level-truth wave edits were found in the tree (levels*.go, level_stats_wire
retry/once split, backfill cmd, planner pool-stamp hunks). They break
`TestSessionVWAPMoves` under the new VWAP±2σ emission and belong to the
NEXT wave. Parked (nothing lost):

1. `stash@{3}` `level-truth-wave-wip-parked` (12 files + untracked levels_swing.go)
2. `stash@{2}` `level-truth-wave-wip-parked-2` (plan_doc no-trade stamp, level_stats_wire)
3. `stash@{1}` `level-truth-wave-wip-parked-3` (untracked cmd/levelstats-backfill)
4. `stash@{0}` `level-truth-wave-wip-parked-4` (planner pool stamp + carry hunks)

Restore after the armed-orders boot INTEGRITY is green: pop in reverse order
(4→3→2→1) and re-run `go test ./...` BEFORE committing anything.
