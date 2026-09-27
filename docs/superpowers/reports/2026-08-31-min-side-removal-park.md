# 2026-08-31 — min_side_levels removal (park record)

## What shipped (owner ruling 2026-08-31)

The per-side COUNT concept is deleted from the system — knob
(`min_side_levels`), `MIN_SIDE_LEVELS` env, `MinSideLevelsFor` resolver, WARN
branch, `SideQuotaNote`, `thin_side` plan stamp, ⚖ card chip, Studio base +
session knob rows, i18n label, guide entries. Only survivors: the two
data-earned hard fails — **0-levels-on-a-side** and **empty machine map**
(2026-08-18 pathology guards), reworded with no ≥N language:
`0 levels below/above price X — a plan must map both directions`.

Old stored config JSON with the field loads harmlessly (encoding/json
ignore-unknown; `TestDayPlanConfigIgnoresRemovedMinSideLevels` locks it).

## RIDER answer

`facts` is the **prompt-time snapshot** — built from the same `PlannerInput`
the prompt is rendered from (`trader/auto_trader_planner.go:881-906`), passed by
value into the retry loop and reused across all 3 attempts. No write-time
refetch exists, so the planner is judged on exactly what it was shown. Nothing
to plumb.

## Cutover

- Code commit `e86ae805784b7b0ee10299a3c977738a813d0cd4` (dev).
- Flat-gate: DB OPEN 0 · non-terminal orders 0 · armed non-terminal 0 · API
  `/api/positions` `[]` · NT8 snapshot count=0.
- Swap: `nofx-bin` → `nofx-bin.prev.boot` (rollback = 5d7be58a); `deploy/RELEASE`
  = e86ae805; kill -9 1077758 → systemd relaunch PID 1123319.
- Boot checklist (08:13:51):
  - `🔐 BOOT INTEGRITY OK — rev e86ae805784b · goldens PASS`
  - `🛡 plan facts guards: 0-levels-on-a-side + empty machine map = fail-closed
    (2026-08-18 pathology guards) — per-side counts REMOVED entirely (owner
    ruling 2026-08-31; no quota, no WARN, no thin_side note)`
  - `🎛 entry law: … stop_entry_seam=ON` · `⚔️ armed_orders=on … test_seam=ON`

## Proof

- Owner reset 08:14:44 → LONDON read on the new binary: 3 attempts, all
  rejected on arm-spec rules (split-arm legs / confirm=touch — NOT counts),
  fail-closed → `PLAN written 2026-08-31 LONDON v3 (lifecycle no_trade)`
  at 08:33:54. **Zero side-quota / thin-side lines in the journal** — the
  WARN path no longer exists.
- NY scheduled read fired 08:25:51 on the new binary; result rides the
  planner-latency autopsy report.
- The definitive no-WARN write: LONDON v2 (old binary) warned twice for the
  same shape; on e86ae805 the shape would write silently. Trader test
  `TestRunPlannerReadThinAboveWritesCleanNoArtifacts` + kernel
  `TestSideGuardOneBelowRichMapWritesClean` lock the contract.

## Rollback

`nofx-bin.prev.boot` = rev `5d7be58a`. Revert = swap back + kill -9 +
`deploy/RELEASE` back to `5d7be58ae17bd1165da11179185bbf01c8568a63`.
