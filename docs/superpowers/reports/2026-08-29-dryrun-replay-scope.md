# DRY-RUN REPLAY — SCOPING REPORT (size-first, before build)

- Worktree `~/nofx-dryrun` @ deployed `b0549ff2` · branch `docs/dryrun-replay` · 2026-08-29
- Dispatch rule honored: the fake-provider harness is **well over ~150 lines** — this report states the size, the ONE architectural obstacle, and the options BEFORE any build. Parked bot, live DB, and NT8 untouched.

## The one hard obstacle: no clock abstraction

The bot calls `time.Now()` directly at **181 non-test sites** in `trader/` + `kernel/` + `store/` (session registry, cycle cadence, calendar/T1, confirm windows, plan birth, stale gates — `kernel/session_registry.go:201 ActiveSession(now time.Time)` included). Today is Saturday: `ActiveSession(now)` → no active session → the planner refuses at `api/handler_plan.go:892` ("no active plan to edit (night / disabled session)"). A 60–120× replay with unmodified code therefore cannot produce a single plan — the shadow day would be a no-op by construction. True acceleration requires a **clock seam** (a code modification: ~30 call sites if scoped to the loop/planner subset, ~120 lines), which conflicts with the dispatch's "run UNMODIFIED". The only zero-code alternative — driving `runPlannerRead`/`runCycle` manually — hits the same `time.Now()` wall.

## Component sizes (honest estimate)

| Component | Lines | Notes |
|---|---|---|
| Clock seam (required code change) | ~120 | override `time.Now` at the ~30 loop/planner/session sites (or a package `clock.Now`), behind an env-gated boot flag — SHIPPED code, not harness |
| 1m replay provider (`market.FuturesBarsProvider` setter exists, `market/data.go` var) | ~80 | reads the arbiter-verified 08-28 bars, time-shifted |
| In-process order simulator (no NT8) | ~250 | fake TCPTrader surface (type-asserted at `trader/auto_trader.go:65/83/168`) + fill-on-touch + bracket legs |
| Second-instance plumbing (DB copy path, port, telemetry guards, sandbox flags) | ~100 | |
| Acceleration orchestrator (60–120× clock mapping, session day map to 08-28) | ~150 | |
| Ledger capture + report harness | ~100 | plan versions, arms, refusals, P&L, truncation, latency |
| **Total** | **~800–900 + a shipped code change** | **>150-line threshold: build gate = owner ruling** |

## The second cost: REAL planner calls cannot be accelerated

The dispatch demands real DeepSeek planner calls with per-config reasoning. The real 08-28 day had **13 plan writes** (7 NY versions) at 2–9 min per read → a full-day shadow costs **~2–6 h wall time and real API spend** regardless of bar speed. (Executor-only cycles could be compressed; planner latency dominates.)

## Options (owner ruling needed)

- **A — full-day shadow, as dispatched:** clock-seam wave (S/M, shipped) + harness (~800 lines, harness-only in the worktree) + 2–6 h real AI. Most faithful; largest cost.
- **B — session-slice shadow:** same harness, but ONLY the NY crash window (08-28 08:30–14:45) — captures the −347pt waterfall, immediate-mode, fast-market, and the −$176.5 baseline's biggest day; ~3× less AI time (~1–2 h).
- **C — bar-replay only (no AI),** reuse the existing fixtures: zero new harness; nothing new proven beyond the pre-live-fire sweep.

**Recommendation:** B — the dispatch's money questions (9-condition authoring on the crash, breakdown/immediate behavior, gate chain on real tape) are all NY-session events; the full-day ASIA/LONDON marginal value is thin at 2–6 h of AI. If you want A anyway, say "go A" and I'll build the clock seam as its own S/M wave first.

## Verdict

**Not built yet by design** — the size report is the deliverable this turn. Awaiting your ruling (A / B / C, and approval of the ~120-line shipped clock seam if A or B).

---

# RULING EXECUTION — go B via libfaketime: SMOKE GATE FAILED → CLEAN ABANDON (2026-08-29)

Owner ruled: go B (NY slice) WITHOUT the clock seam, faking the clock at the OS layer via libfaketime, with an explicit fallback if it misbehaves with Go's monotonic clock.

**What ran:**
1. libfaketime built from source (user-local, no sudo): `/tmp/libfaketime/src/libfaketime.so.1` (ELF x86-64, 09:46).
2. 5-minute smoke slice: a 2-line Go program printing `time.Now()` three ways — baseline, under `FAKETIME='2026-08-28 08:25:00' LD_PRELOAD=…`, and with `FAKETIME_DONT_FAKE_MONOTONIC=1`.

**Result (quoted):**
- baseline: `wall: 2026-08-29 09:47:14 CDT` → `+0.5s: 09:47:15`
- under FAKETIME: `wall: 2026-08-29 09:47:15 CDT` (NOT moved — still real Saturday)
- under FAKETIME + DONT_FAKE_MONOTONIC: `wall: 2026-08-29 09:47:15` (wall still NOT moved; only sleep timing normalized)

**Root cause [A]:** Go resolves `clock_gettime` through the **vDSO**, not the libc symbol — `LD_PRELOAD` interception never sees the call, so libfaketime cannot move `time.Now()` in any Go binary. `ActiveSession(now)` would still see Saturday and the planner would refuse every plan. This is exactly the anticipated failure branch.

**Disposition:** shadow session **CANCELLED cleanly** per the owner's fallback — tomorrow's (Sunday 17:00 CT) live-fire is the integration test. Harness not built; no code shipped; parked bot / live DB / NT8 untouched; worktree `~/nofx-dryrun` contains only this report. Done before the Sunday 15:00 CT deadline.

**If the shadow is ever re-attempted, the only working levers are:** (a) the ~120-line clock seam (owner-declined this round), (b) a dedicated VM/container with a rewritten system clock (root + isolation, heavier), or (c) live-fire itself.
