# CLASS 32 — Scheduled reads must fire on wall-clock

Date: 2026-08-31 CT · Wave: class32-wallclock-reads · Branch: `class32-wallclock-reads` (merged → dev `ebc37e01`)

**STATUS: SHIPPED — cutover 2026-08-31 23:40:22 CT, owner GO ("Send this and GO class 32").**
Live rev now: `ebc37e01d7dd5f19c0e0f0ffa962388e12988f58` (PID 1589096).
Rollback kept: `nofx-bin.prev.boot` = previous live `7004a7f1f726…` (0C, PID 1466535).

Cutover evidence:
- Flat gate pre (23:39-23:40 CT): DB open pos 0 · open orders 0 · armed nonterminal 0 · API positions `[]` · API open-orders MNQ `[]` · NT8 snapshots `count=0` ×2 @23:39:59.
- Swap: `mv nofx-bin nofx-bin.prev.boot` → staged binary in (stamp verified on the deployed file: `vcs.revision=ebc37e01…`, `modified=false`) → `kill -9 1466535` @23:40:22 → systemd relaunch PID 1589096.
- Boot 23:40:27 quoted: `🔐 BOOT INTEGRITY OK — rev ebc37e01d7dd · expected ebc37e01d7dd · goldens PASS` · `🗓 session reads (owner ruling 2026-08-31, open−30): ASIA 16:30 · LONDON 01:30 · NY 08:00 CT` · `🔬 conditions: live [7] · shadow [breakout_retest, fvg_entry]` · `📜 scenario schema: 9 conditions`.
- Post-boot: 0 `[ERRO]`/panic, equity 52216.00 preserved, API positions `[]`, open-orders MNQ `[]`.
- TRUE LIVE PROOF pending: tomorrow ~16:35 CT — quote the actual ASIA read timestamp + `🗓 session read fired during halt …` line (the read is scheduled for 16:30 while CME is halted 16:00-17:00).

Staged:
- Build sha `ebc37e01d7dd5f19c0e0f0ffa962388e12988f58` — clean-clone build, `vcs.revision` matches, `vcs.modified=false`. Binary at `~/nofx-staged/nofx-32-bin`.
- Marker `60ae142d` pushed: `deploy/RELEASE` = `ebc37e01…` + `GUIDE_BUILT_REV` = `ebc37e01…`.
- Suites: `go build ./...` OK · `go test ./...` green · tsc 0 · vitest 36 files / 292 tests.
- Flat gate pre-park (18:44-18:45 CT): DB open pos 0 · open orders 0 · armed nonterminal 0 · API positions `[]` · API open-orders MNQ `[]` · NT8 snapshots `account=Sim101 count=0` + `account=SimAccount1 count=0` @18:44:22.
- Live bot untouched: rev `7004a7f1f726…` PID 1466535, ASIA v1 no_trade, window check passed (18:45 CT, not 16:45-17:10).

Cutover runbook (on owner GO):
1. `mv ~/nofx/nofx-bin ~/nofx/nofx-bin.prev.boot` → `cp ~/nofx-staged/nofx-32-bin ~/nofx/nofx-bin`
2. `kill -9 <PID>` (SIGTERM exits 0 — no relaunch)
3. Boot checklist within 90s: rev `ebc37e01` · `🔐 BOOT INTEGRITY OK` · goldens PASS · `🗓 session reads (owner ruling 2026-08-31, open−30)` line · `🔬 conditions` line (0C)
4. Post-boot flat-gate re-quote (four legs).
5. LIVE PROOF tomorrow ~16:35 CT: quote the actual ASIA read timestamp + the `🗓 session read fired during halt …` line.

Rollback: `mv nofx-bin.prev.boot nofx-bin && <revert deploy/RELEASE> && kill -9 <PID>`
(`nofx-bin.prev.boot` will hold the pre-wave live `7004a7f1f726…`.)

## Root cause (tonight's evidence)

The scheduled session read is carried by the market-data cycle. `tickOnce`
returns at the no-new-data dedup (`trader/auto_trader_clock.go:810` post-fix:
the interval-mode skip) BEFORE `runCycle()` is entered, and the read lived
INSIDE `runCycle` (`trader/auto_trader_loop.go:273` pre-fix). CME is halted
16:00-17:00 CT, so every cycle 16:26→16:38 logged
`⏭ cycle_skip=no_new_data` and the 16:30 ASIA read was skipped WITH the data
work. It fired ~17:00:03 when the reopen tick arrived — 30 minutes late, no
error, no alarm, no plan at the open.

Post-mortem facts:
- session_registry (the LIVE source) reads ASIA 16:30 · LONDON 01:30 · NY 08:00
  — the scheduler resolution was CORRECT.
- Sunday-defer was NOT the cause: defer_count=0, and its predicate correctly
  checks weekly-doc EXISTENCE, not validity.
- traders.is_running = 1 throughout; zero panics.
- The read, once it fired, behaved correctly (attempt 1 rejected on a real
  defect, attempt 2 was a repair call).

## Located paths (LOCATE-FIRST quotes)

- skip decision site: `trader/auto_trader_clock.go:810` (pre-fix: the
  interval-mode `skipNoNewData` return in `tickOnce`; the bar-close-mode gate
  at :802 is the sibling).
- read trigger (old): `trader/auto_trader_loop.go:273` — `at.maybeRunSessionReads()`
  inside `runCycle`.
- read evaluator: `trader/auto_trader_planner.go:190` `maybeRunSessionReadsAt(now)` —
  window gate `inSessionReadWindow`, Sunday defer `sundayAsiaDeferred`, plan-store
  dedupe, in-flight claim (`auto_trader_planner.go:829` "already in flight —
  skipping duplicate call").

## The fix

| # | Item | Evidence |
|---|---|---|
| 1 | Scheduled reads evaluated on wall-clock EVERY tick, before the data-gated skips | `trader/auto_trader_clock.go:791` `at.evaluateWallClockSessionReads()` at the top of `tickOnce`; read call REMOVED from `runCycle` |
| 2 | Halt-fired authoring ruled correct — no staleness gate added | comment block at the evaluator; no validator/knob changes anywhere in the diff |
| 3 | Halt log line with bar age | `🗓 session read fired during halt (ASIA) — authoring from last stored bars (newest 5m 2026-08-18 16:29 CT, age 1m)` — fixture `TestClass32HaltReadLogLine` + live-shaped quote from `TestClass32ReadFiresOnceWithFrozenTape` |
| 4 | Idempotence preserved | existing plan-store dedupe + in-flight claim (quoted above) + fixture `TestClass32ReadFiresOnceWithFrozenTape` (exactly ONE `ASIA_scheduled_read` row across 20 min of frozen ticks) |
| 5 | Data pipeline still idles on a frozen tape | `TestClass32FrozenTapeStillSkipsDataWork` — `runCycle` never entered, `callCount == 0` |
| 6 | Regression pin fails on old code | proven: `TestClass32PinFailsOnOldCode` run against the stashed pre-fix tree → `--- FAIL: OLD-CODE EXPECTED: ASIA read did NOT fire at 16:30 with no new bars` |
| 7 | LONDON 01:30 / NY 08:00 unchanged | existing W1 + P0-B + read_times_lead suites green (no test touched) |
| 8 | Sunday sequencing unchanged | `sundayAsiaDeferred` untouched; `TestP0BAsiaReadDoesNotFireOutsideItsWindow` + `read_times_lead_test` green |
| 9 | Goldens / suites | `go build ./...` OK · `go test ./...` green · web tsc 0 errors · vitest 36 files / 292 tests |

## Sibling audit (4.5 — read-only, NOT fixed here)

| Scheduled action | Call site | Gating today | Halt/quiet-tape exposure |
|---|---|---|---|
| **Weekly read (Sun 16:30)** | `trader/auto_trader_loop.go:202` `maybeRunWeeklyRead` (above the session gate, inside `runCycle`) | behind `tickOnce` data skips | **EXPOSED — same class.** Sunday 16:30 sits inside the halt → delayed until reopen → ASIA then defers on it (compounding). Highest-risk sibling; next wave. |
| Calendar producer | `auto_trader_loop.go:196` `maybeFetchCalendar` | behind `tickOnce` data skips | Exposed on a frozen-tape weekend boot (F0 hoisted it above the SESSION gate only). Idempotent catch-up; benign. |
| EOD flat (14:45/08:30/02:00) | `auto_trader_loop.go:314` `enforceEODFlat` | `skipNoNewData` is FLAT-ONLY → while holding the cycle always runs | Protected-by-construction in interval mode; latent only in `bar_close` mode with a quiet close. |
| T1 force-flat (news −2m) | `auto_trader_loop.go:323` `enforceT1ForceFlat` | same flat-only skip | Protected-by-construction when holding; no-op when flat. |
| Session digests + daily roll | `auto_trader_loop.go:283` `maybeWriteDigests` | behind skips | Delayed digests are idempotent catch-up; cosmetic. |
| Session-profile snapshot / night edge | `auto_trader_loop.go:261/266` | behind skips | Catch-up bookkeeping; benign. |
| Weekly invalidation watch | `auto_trader_loop.go:279` | data-gated by design (closed bar beyond price) | Halt delays it minutes; benign. |
| Gate-block rollover + error-day roll | `auto_trader_loop.go` top | behind skips | Telemetry only. |
| Fast-market / level / MSS wakes | planner wake machinery | data-gated by design (respond to bar events) | Cannot fire without the event; by design. |
| level_stats nightly (17:05 CT) | `trader/ninjatrader/level_stats_wire.go` `WireLevelStatsNightly` | own goroutine + timer | NOT exposed (not on the data cycle). |
| DB backups | systemd user timer | external | NOT exposed. |

## Lock path (1.1)

Path taken: **(a)** — the owner gave GO on 0C earlier tonight; 0C booted at
17:34:21 CT (rev `7004a7f1f726`, PID 1466535) and its lock was released + its
worktree removed in that wave's closeout. This wave re-acquired
`~/nofx-main.lock` live under PID 1437095 (expiry 21:14:08 CT, task
class32-wallclock-reads) on a porcelain-clean main tree at `4be2c73d`.
**1.4 liveness amendment recorded**: the lock rite now carries a `pgrep`/`kill -0`
liveness re-verification step (canon added to CLAUDE.md MAIN-TREE LOCK LAW;
0C's stale lock PID 1129920 was the precedent).

## 0C relationship (1.2)

0C is ALREADY LIVE (cut over at 17:34:21 CT on the owner's GO). This wave builds
on dev INCLUDING 0C's merged code, but the resulting boot activates ONLY the
class-32 change — 0C is not re-booted by this wave beyond being part of the
binary's code state. No separation issue arose.

## Pre-cutover state (1.3)

Live rev `7004a7f1f7266a3d8c354afc7ee27f05b5fda2a4` · PID 1466535 · ASIA v1
no_trade · zero armed/working rows · flat gates quoted at cutover below.

## Rollback

```
mv nofx-bin.prev.boot nofx-bin && <revert deploy/RELEASE> && kill -9 <PID>
```

`nofx-bin.prev.boot` holds the pre-wave live binary (`7004a7f1f726…`, 0C).

## Anything the owner will still see wrong on screen

- Nothing new from this wave. The one honest residual: the WEEKLY read is still
  data-gated (sibling audit above) — Sunday 16:30's weekly read can still be
  delayed until the reopen, and the ASIA read will defer on it until it lands.
  That is the documented next wave, not a regression.
- Test hygiene fix disclosed: `TestArmedOrderUpsertAndGateRR` failed on the
  BASE rev during ASIA hours (0C's shadow map correctly refuses its fvg_entry
  arm). The fixture now declares `condition_status: {fvg_entry: live}` — a
  test-only change; the shadow map has its own fixtures.

## Proof posture (9)

- Relied on NOW: the 6.1 frozen-tape fixture (proven to fail on the old code)
  + halt-line fixture + idempotence fixture + full suites.
- The TRUE live proof is tomorrow's ASIA read firing at 16:30 CT with the
  market halted. FOLLOW-UP: tomorrow ~16:35 CT, quote the actual read
  timestamp and the halt log line — that closes the wave.
