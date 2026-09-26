# DAY-PLAN CAMPAIGN — P3.6 FINISH → P3 COMPLETE

**Date:** 2026-08-15 · **Repo:** /home/hoang/nofx · **Branch:** main
**Range:** `0dfcd532` → `e19403fc` · 4 feature commits (A/B/C/D)
**Contract:** [docs/VL-DAYPLAN-FULL-SPEC.md](../../VL-DAYPLAN-FULL-SPEC.md)

## LINE 1 — VERDICT
**P3 · THE PLANNER is COMPLETE.** The last four P3.6 sub-features — digest writers,
scenario re-arm, sticky owner levels, night mode — all landed with fixtures; suite
+ `-race` green; NO golden touched (A–D are storage/logic only); everything gated
on day_plan → dormant until ★ RESTART 2. Campaign: **P0 ✅ P1 ✅ P2 ✅ (★1 live) P3 ✅.**

## STEP 0 gate
PASS — HEAD `0dfcd532` · tree clean · bot on the ★1 binary (PID 778475) untouched.

## Shipped
### A — digest writers — `f9f5460f`
`store/day_plan_digests` (append-only, idempotent). `kernel` FormatSessionDigest /
FormatDailyDigest (pure 3-line) + `BuildDigestChain` (tapered: current-date session
digests + 3 FULL dailies + days 4–7 one-liners). Trader `maybeWriteDigests` (session
digest at each enabled session's close; daily roll-up at 16:00 CT) wired into
runCycle; `assemblePlannerInput` now feeds `DigestChain`. Fixture: a synthetic week
renders the exact tapered chain (2 sessions + 3 full + 4 one-liners = 9).

### B — scenario re-arm via level-state — `29727291`
Extends the P0.5 level-state: `last_play_ms` column + `RecordPlay` (times_tested++,
stamp, freshness A→B→C→done) + pure `ReArmEligible` (not consumed · above floor ·
setup re-formed not a bare re-touch · 20m cooldown). Permanent consume on
acceptance-through reuses `MarkConsumed`. Fixtures for every transition.

### C — sticky owner levels — `9f000637`
`store/owner_levels` — persist ACROSS sessions (independent of any plan/session,
survive plan expiry) until consumed/deleted; P5's overlay API posts here.
`assemblePlannerInput` prepends active owner levels to the ranked table as
always-seated ScoredLevels tagged 👤 (grade A, note + scenario tag). Fixture: an
owner level set in one session appears in a LATER session's assembled planner input.

### D — night mode — `e19403fc`
`SessionRegistry.IsNightMode` (no active ENABLED session = night; clock-derived).
Trader `observeNightEdge` emits a night↔day transition event on the edge; a nil
prev (fresh/restart) never emits, so **a restart during night resumes cleanly**.
Reads + entries are already night-safe (the P3.1 session gate). Fixtures: IsNightMode
day/night, edge decision (nil no-emit / unchanged no-emit / both transitions emit),
restart-clean.

## EXIT BAR
- `go build ./...` ✓ · `go vet ./...` ✓ · `go test ./...` ✓ ·
  `go test -race ./trader ./kernel ./store` ✓.
- Goldens: **none changed** (`git diff 0dfcd532..HEAD -- kernel/testdata/` empty) —
  A–D are storage + logic, no prompt changes.
- config-truth: new store tables (day_plan_digests, owner_levels) + the additive
  level_state.last_play_ms column are created by AutoMigrate on next start; empty,
  additive, dormant.

## P3 — full recap (all six)
- P3.1 session gates · P3.2 planner model binding · P3.3 read jobs + planner call
  (schema/assembler/fail-closed/plan rows) · P3.4 executor injection + RECON #4
  reorder (the ONLY new golden, futures_mnq_plan.golden) · P3.5 advisory
  (cited_scenario + match-rate) · P3.6 lifecycle (activation window, death→re-plan,
  restart recovery, digests, scenario re-arm, sticky owner levels, night mode).

## What's next — P4 · THE CARD
SessionTabs + 24h timeline strip + HandoverBanner + SessionPlanCard, the chart
level overlay, the in-app alert center, the Studio Day Plan block, and /api/plan/*
(mirrors /api/risk/*). Then P5 · THE DOOR, then ★ RESTART 2.

All shipped code is additive + dormant (gated on day_plan) — the running bot is
unaffected. vlauto: DEFERRED to the next propagation train.
