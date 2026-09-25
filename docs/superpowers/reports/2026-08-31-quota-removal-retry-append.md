# 2026-08-31 — quota removal + retry-append (live before NY 08:30)

## Changes (owner ruling 2026-08-31)

- **CHANGE 1 — side-quota count removal.** `kernel/plan_doc.go`
  `ValidatePlanDocWithFactsMachine`: the AI-caused-omission REJECT branches
  (plan side < `min_side_levels` while the machine map supplied ≥ quota) are
  GONE. Demoted to a `thin_side` WARN note on the plan card, never a refusal.
  `min_side_levels` knob / session overrides / env are now WARN-threshold-only.
  KEPT hard fails: 0 levels on a side (2026-08-18 pathology), empty machine
  map, duplicates, illegal confirm rules, split-arm spec, displacement on
  breakdowns, targets in-band, DOA/invalidation-breached.
- **CHANGE 2 — retry-append reject reason (session path).**
  `trader/auto_trader_planner.go`: attempts ≥2 carry
  `## PREVIOUS ATTEMPT REJECTED / Validator reason (verbatim): <err>
  Fix ONLY this defect, keep the rest structurally identical.` — wired at ALL
  6 reject sites (failed, parse/schema, bias, facts, FVG, breakdown).
- Tests: kernel `TestSideQuota*` (warn + 0-side fail + empty-map fail) green;
  trader `TestRunPlannerReadAIOmissionWarnsAndWrites` +
  `TestRunPlannerReadRetryAppendRejectBlock` green. Full `go test ./...` green.
- Guide (5 files) updated: count is WARN-only per owner ruling 2026-08-31;
  zero-side/empty-map still fail-closed.

## Deployment

- Code commit `5d7be58ae17bd1165da11179185bbf01c8568a63` (dev).
- Binary built from temp clone at that sha (`vcs.revision` stamped,
  `vcs.modified=false`).
- Flat-gate pre-swap: DB OPEN positions 0 · non-terminal orders 0 · armed
  non-terminal 0 · API `/api/positions` `[]` · NT8 `positions snapshot
  count=0` ×2.
- Swap: `nofx-bin` → `nofx-bin.prev.boot` (rollback = 59dc9460); new binary
  live via `kill -9` + systemd relaunch (PID 1077758).
- First boot REFUSED (expected: `deploy/RELEASE` still pointed at 59dc9460) —
  by design. RELEASE updated, restart → boot clean:
  - `🔐 BOOT INTEGRITY OK — rev 5d7be58ae17b · goldens PASS`
  - `⚖️ side-quota: per-side COUNT is WARN-only (owner ruling 2026-08-31; thin_side note) — min_side knob/env = warn threshold only · 0-side + empty-map still fail-closed (2026-08-18 pathology guards)`
  - `🎛 entry law: … stop_entry_seam=ON`
  - `⚔️ armed_orders=on … test_seam=ON`

## Live proof (owner reset 07:18:15 → read)

Attempt 1 (07:27:26) — rejected by the ENTRY-LAW confirm rule (unchanged guard):
> `📐 planner attempt 1/3 parse/schema rejected: scenario[1].confirm2.rule
> "1m_mss" not allowed for breakdown_continue — entry law: …`

Attempt 2 (07:32:46) — side-quota WARNed (CHANGE 1 live; previously this
fail-closed ASIA 3×), then rejected by the breakdown-void guard (unchanged):
> `📉 side-quota relaxed: thin-side: 5 below (machine map 53) — count is
> WARN-only per owner ruling 2026-08-31 (previously rejected AI-caused
> omissions)`
> `📉 side-quota relaxed: thin-side: 5 above (machine map 58) — count is
> WARN-only per owner ruling 2026-08-31 (previously rejected AI-caused
> omissions)`
> `📐 planner attempt 2/3 rejected: S1 breakdown_continue: a close came back
> across 29437.00 — the breakdown is void; author a reject/retest play instead`

Attempt 3 (07:40:35, 469.5s) — the named defects fixed (CHANGE 2 working),
thin sides WARNed, plan WRITTEN:
> `📉 side-quota relaxed: thin-side: 4 below (machine map 53) …`
> `📉 side-quota relaxed: thin-side: 6 above (machine map 58) …`
> `🗓️ PLAN written 2026-08-31 LONDON v2 (model deepseek-v4-pro, lifecycle
> active, prompt 5631f606e3a0, ai_config a28d83f159084145)`

## Rollback

`nofx-bin.prev.boot` = rev `59dc9460`. Revert = swap back + `kill -9` (systemd
relaunches) + `deploy/RELEASE` back to `59dc94603e493d9f4a6404a989cfec8d32c32d02`.
