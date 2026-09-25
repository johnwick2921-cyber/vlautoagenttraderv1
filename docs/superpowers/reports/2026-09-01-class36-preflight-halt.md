# CLASS 36 — planner preflight: scheduled reads bypass the freshness check during the halt and weekend

Date: 2026-09-01 · Owner: hoang · Agent: Fable 5 · Worktree: `../nofx-class36` (branch `fix/class36-preflight-halt`)
Evidence tiers: **[A]** directly verified · **[B]** inferred from strong evidence · **[C]** speculation.

## STATUS

| Item | State |
|---|---|
| Code | **MERGED to dev @ `17efeea9`** (fast-forward from `795f67f7`, pushed) |
| Build | clean clone `--no-local` at `17efeea9`, `vcs.modified=false`, built 2026-09-01T22:52:57Z, sha256 `d2f724a92ce3db45…`, 70,859,424 bytes — **STAGED as `~/nofx/nofx-bin.next` at 17:59:30 CT** (main tree fast-forwarded to dev `b2c2ff92`, porcelain empty) |
| Marker | `7089d271` — `deploy/RELEASE` + `GUIDE_BUILT_REV` = `17efeea9…` (one marker, park record in its message) |
| Cutover | **DONE 18:01:06 CT on owner GO** — `🔐 BOOT INTEGRITY OK — rev 17efeea9 · expected 17efeea9 · goldens PASS` 18:01:11, PID 1941026, new `🗓 preflight:` boot line present (see CUTOVER section); rollback `nofx-bin.prev.boot` kept |
| Lock (A2) | `~/nofx-main.lock` is held by `pid=1906840` (`planner-api-failure-0901`, expiry 21:35 CT) — `kill -0` → ALIVE at 17:41, 17:49 and 17:53 CT. **Not cleared.** All work ran in the worktree; the main tree was never touched (porcelain empty, quoted 17:41 CT) |

---

## C — LOCATE (quoted before editing, A17) [A]

**C1 · The preflight.** ONE function, ONE call site, planner-only:
- `trader/auto_trader_feedwatch.go:109` `plannerPreflight(session, tradeDate)` — fail-open when `market.FuturesBarsProvider == nil`; else `age, haveBars := at.feedNewestBarAge(traderNow())` (newest stored **1m** bar's close vs the trader clock, `feedwatch.go:50-61`) and `if haveBars && age <= feedDownAfter() { return true }` where `feedDownAfter()` = `kernel.LoadFeedPolicy().FlatAlertMs` = env `FEED_ALERT_S` → **default 600s** (`kernel/feed_policy.go:47-50`; `.env` sets neither `FEED_ALERT_S` nor `INTRADE_FEED_ALERT_S`, so the resolved value is the 600s default — A11). Otherwise `🛑 planner_preflight_refused … reason=no_bars|stale_bars_<n>s` (ERROR) + P1 alert, return false.
- Call site: `trader/auto_trader_planner.go:878` inside `runPlannerReadWithTriggerClaimedCtx`, after the claim (`claimPlannerRead`) and the F6 clock-hold, before the client resolve. Every planner trigger class passes through it. The executor never calls it (`grep plannerPreflight(` → the definition and this one site). Other gates on the same path: the in-flight claim (`:840-845`), the F6 clock-hold (`:852-859`, defers on a *future*-stamped tape, sign-aware).
- The executor's halt block is separate and untouched: `trader/auto_trader_loop.go:937-947` `cmeSessionClosedSkip()` → `kernel.IsCMEOpen(time.Now())` idles the whole decision cycle; entries are also refused by `sessionGateDecision` outside a window.

**C2 · Today's refusal (data/nofx_2026-09-01.log) [A]:**
```
09-01 16:31:05 [ERRO] … 🛑 planner_preflight_refused session=ASIA trade_date=2026-09-01 reason=stale_bars_1865s — refusing to call the LLM with no market data …
09-01 16:33:05 [ERRO] … 🛑 planner_preflight_refused session=ASIA trade_date=2026-09-01 reason=stale_bars_1985s …
09-01 16:59:05 [ERRO] … 🛑 planner_preflight_refused session=ASIA trade_date=2026-09-01 reason=stale_bars_3545s …
09-01 17:01:05 [INFO] … 🧠 planner model: empty binding → using primary, pinned "deepseek-v4-pro"      ← the read finally launched on the reopen tick
```
15 refusals between 16:31 and 16:59 CT (every 2 minutes), each paired with class 32's `🗓 session read fired during halt (ASIA) — authoring from last stored bars (newest 5m 2026-09-01 15:55 CT, age 46m…58m)`. Then `17:10:09 📐 planner attempt 1/3 rejected` → `🧩 attempt 2/3 repair` → `17:23:14 🗓️ PLAN written 2026-09-01 ASIA v1 (… lifecycle no_trade)` = `planner_fail_closed`.

**C3 · Scheduled session-read path:** `trader/auto_trader_clock.go:98` `evaluateWallClockSessionReads()` (uses `traderNow()`), invoked at the top of `tickOnce` (`clock.go:837` pre-fix) BEFORE the bar-close gate and the no-new-data dedup → `trader/auto_trader_planner.go:190` `maybeRunSessionReadsAt(now)` → `inSessionReadWindow` → `sundayAsiaDeferred` (`:213`) → `SessionInstanceStart`/`IsCMEOpen(instOpen)` (`:218`) → `go at.runPlannerRead(s.Name, tradeDate)` (`:246`) → `runPlannerReadWithTriggerClaimedCtx(…, "", …, true)` → **preflight**.

**C4 · Weekly path:** `trader/auto_trader_loop.go:202` `at.maybeRunWeeklyRead(time.Now())` inside `runCycle` — i.e. BEHIND the bar-close gate and the no-new-data skip in `tickOnce` (`clock.go:848-860`), the exact layer class 32 hoisted the session reads above. `trader/auto_trader_weekly.go:108` `maybeRunWeeklyRead` → `weeklyReadVerdict` (wait/skip/run) → in-flight claim → `go runWeeklyRead`. Spec: `kernel/weekly_knobs.go:18` `WeeklyReadCTDefault = "sun 16:30"`, `WeeklyReadSpec()` → (Sunday, 16, 30). **The weekly read does not call `plannerPreflight`**: it reads STORED 1m bars (`weeklyBars1m` → `store.BarHistory().BarsBetween`) and refuses only on an empty store — so it needed only the wall-clock move, not a bypass.

**C5 · Trigger classes vs the halt (one table):**

| class | trigger label | fires in a halt / weekend? | freshness check today | after class 36 |
|---|---|---|---|---|
| scheduled session read | `""` → `<S>_scheduled_read` | YES (open−30 by design; class 32) | applied → refused (the bug) | **bypassed while `!IsCMEOpen(now)`**; kept on a live-but-silent tape |
| weekly | `sunday_weekly_read` / `weekly_boot_backfill` | YES (Sunday 16:30) | none (own empty-store check) — but data-gated inside runCycle | **moved to wall-clock**; sequencing unchanged |
| level_event | `level_event` | NO — needs a closed bar after the plan's birth; a frozen tape has none (`wake_levels.go:235-260`) | applied | unchanged |
| fast_market | (reasoning MODE of a wake read, `planner.go:875-880`) | NO — rides level_event / structure_mss | applied | unchanged |
| structure_mss | `structure_mss` | NO — needs a fresh MSS on live bars | applied | unchanged |
| death_replan | `death_replan` | can be *evaluated* in the ASIA read window (16:30-17:00) if a chain exists; the read is then **refused on staleness** (no row, no spend) and retried next cycle | applied | **unchanged** (pinned) |
| owner_reread | `owner_reread` | NO — `CanForceReread` refuses "no session is active" at 16:30 (`reread.go:40-43`), and "the market is closed" via `!IsCMEOpen` (`:52`) inside a window on a closed day | applied (never reached) | **unchanged** (pinned) |
| owner_reset | `owner_reset` | NO — `ForceReset` refuses `!IsCMEOpen` (`reset.go:67`) | applied (never reached) | unchanged |
| executor (decide/arm/order) | n/a | NO — `cmeSessionClosedSkip` (`loop.go:937`) | n/a | **untouched** |

**C6 · Facts snapshot during the halt [A]:** `assemblePlannerInputWithCtx` (`planner.go:1794`) reads bars from `market.FuturesBarsProvider` (the in-memory BarCache seeded from NT8 + `SeedHistorical`), candle tables per timeframe, and `AssembleScoredLevelsFullMinGrade` for the level map/ATR — all from the cache, none require a live tick. The 16:30 read today would have seen the same snapshot the 17:01 read saw: newest 5m bar **2026-09-01 15:55 CT** (quoted from the halt lines: "newest 5m 2026-09-01 15:55 CT, age 46m" at 16:41), newest 1m bar 15:59 CT, i.e. the complete pre-halt tape.

---

## D — THE FIX

**D1/D2 — scope the freshness check by trigger class, delete nothing.** `trader/auto_trader_feedwatch.go`:

| line | change |
|---|---|
| 110 | `plannerPreflight(session, tradeDate, trigger string)` — the trigger class is now an input |
| 131-135 | after the normal fresh-pass: `if haveBars && preflightScheduledBypass(trigger, now)` → log the WARN bypass line with the newest 1m bar + age, return true |
| 136-147 | otherwise the refusal, now `⛔ planner preflight refused <class>: <reason>` (ERROR) + the existing P1 alert |
| 151 | `preflightTriggerClass` (`""` → `session_read`) |
| 162 | `preflightIsScheduled`: `""`, `*_scheduled_read`, `weekly`, `sunday_weekly_read`, `weekly_boot_backfill` |
| 171 | `preflightScheduledBypass(trigger, now) = preflightIsScheduled(trigger) && !kernel.IsCMEOpen(now)` — **scheduled AND closed**; a scheduled read into a silent OPEN tape still refuses (the 08-19 outage class, H stop-line: no staleness gates added, none removed) |
| 176 / 182 | pure line builders `preflightBypassLine` / `preflightRefusalLine` (fixture-tested) |
| 188 | `PreflightBootLine()` |

`trader/auto_trader_planner.go:878` — the single call site passes `triggerOverride`. Death re-plans (`death_replan`), owner re-reads (`owner_reread`), wakes (`level_event`, `structure_mss`) and `owner_reset` keep the check exactly (E5 pins the halt behavior of the first two).

**D3 — executor untouched, no split needed.** The preflight was already planner-only (C1); `cmeSessionClosedSkip` / `IsCMEOpen` / arming / order paths have zero diff (`git diff 795f67f7..17efeea9 -- trader/auto_trader_loop.go` touches only the weekly-call comment block).

**D4 — weekly read on the wall-clock evaluator.** `trader/auto_trader_clock.go:122` `evaluateWallClockWeeklyRead()` → `maybeRunWeeklyRead(traderNow())`; called at `clock.go:850` **before** `evaluateWallClockSessionReads()` (`:851`) at the top of `tickOnce`. Removed from `runCycle` (`loop.go:199-204`, comment left in place). Sunday sequencing: weekly claim starts first; `sundayAsiaDeferred` (`planner.go:1277`) **unchanged** — it gates on the weekly doc EXISTING, and the per-tick retry fires ASIA right after the doc lands (E2).

**D5 — loud on both outcomes** (observed in the test log):
```
[WARN] 🗓 preflight bypass (class 36): scheduled session_read read ASIA 2026-09-01 during halt/weekend — freshness check skipped, authoring from last stored bars (newest 1m 2026-09-01 15:59 CT, age 31m)
[ERRO] ⛔ planner preflight refused level_event: no_bars (session=ASIA trade_date=2026-08-18) — refusing to call the LLM with no market data (the 0-scenario fail-closed stub class); the read window retries next cycle
[ERRO] ⛔ planner preflight refused death_replan: stale_bars_1800s (session=ASIA trade_date=2026-09-01) …
```
Class 32's `🗓 session read fired during halt (…) — authoring from last stored bars (newest 5m …, age …m)` line still prints at fire time (INFO). The bypass line is WARN on purpose: INFO is journald-suppressed and absent from `log_events`, so the positive outcome is visible in the journal and the UI's log surface, not only in `data/nofx_*.log`.

**D6 — idempotence.** Unchanged guards, both exercised: the plan-store dedupe (`maybeRunSessionReadsAt` skips when a row exists for the session-day) and the in-flight claim `🗓️ planner read for <key> already in flight — skipping duplicate call.` (`planner.go:840-845`; the weekly has its own `📅 WEEKLY READ already in flight … skipping duplicate call.`). E7 fires the read at 16:30 and re-evaluates at 16:32/16:40/16:50 with a frozen tape → exactly one version.

**D7 — boot line** (`main.go:264`, after the class-35 `🧮` line): `🗓 preflight: scheduled reads bypass freshness in halt/weekend (class 36); executor halt-block unchanged (cmeSessionClosedSkip / IsCMEOpen)`.

**D8 — guide:** `tradingDay.ts:17` (the 16:30 read runs inside the halt on purpose; bypass line), `tradingDay.ts:48` (halt = no executor decisions/orders/arms, unchanged; planner still authors on schedule — the split), `weeklyBias.ts:20` (weekly on wall-clock; sequencing unchanged), `settings.ts:492`. `GUIDE_BUILT_REV` → `17efeea9…` in the marker commit.

**D9 — checklist:** highest was **35** (`AUDIT-CHECKLIST.md:286`, quoted); **36** appended at `:309` as the sibling of 32 — trigger vs preflight, two layers of "scheduled work inheriting the market's calendar". 33 and 27/28/29 untouched.

---

## E — TESTS (A8: written alongside; each quoted) [A]

**E1 · `TestClass36PinAsiaHalt`** (`trader/class36_pin_test.go:37`) — Tuesday 2026-09-01 16:30 CT via `testNow`, frozen provider with the newest 1m bar at 15:59 (age 31 m), scheduled ASIA read through `maybeRunSessionReadsAt`. Uses only the pre-fix surface. **RED on the pre-fix tree `795f67f7`** (throwaway worktree):
```
09-01 17:45:54 [ERRO] [trader_id=t1] 🛑 planner_preflight_refused session=ASIA trade_date=2026-09-01 reason=stale_bars_1800s — refusing to call the LLM …
--- FAIL: TestClass36PinAsiaHalt (5.17s)
    class36_pin_test.go:60: CLASS 36: the scheduled ASIA read fired at 16:30 but the planner call was never MADE — the preflight refused it on staleness during the halt (stale_bars_*); the plan must be on the desk before 17:00
FAIL	nofx/trader	5.177s
```
**GREEN on `17efeea9`:**
```
[WARN] [trader_id=t1] 🗓 preflight bypass (class 36): scheduled session_read read ASIA 2026-09-01 during halt/weekend — freshness check skipped, authoring from last stored bars (newest 1m 2026-09-01 15:59 CT, age 31m)
--- PASS: TestClass36PinAsiaHalt (0.15s)
```

**E2 · `TestClass36PinSundayWeekly`** (`class36_preflight_test.go:326`) — Sunday 2026-08-30 16:30 CT (a real-past Sunday: the F6 clock-hold rightly defers on future-stamped bars), stored 1m bars seeded for the weekly read, provider frozen at Friday 15:59. One tick: `evaluateWallClockWeeklyRead` then `evaluateWallClockSessionReads` → the WEEKLY row for 2026-08-31 lands on the wall-clock path; ASIA has **no** row (deferred); next tick → ASIA lands `active` with the weekend bypass line. **PASS (0.20s).** (Pre-fix the weekly call is not on this path at all — `evaluateWallClockWeeklyRead` does not exist there.)

**E3 · `TestClass36ExecutorHaltBlockUnchanged`** — `IsCMEOpen` false at Tue 16:30, Sun 16:30, Sat 12:00; true at Sun 17:00; `sessionGateDecision` blocks entries at 16:30. PASS.
**E4 · `TestClass36LevelEventDoesNotFireInHalt`** — frozen tape after the plan's birth → `maybeWakePlannerOnLevelEvents` fires nothing (`lastPlannerWakeAt` zero, one version). Fast-market rides that wake (mode, not class). PASS.
**E5 · `TestClass36DeathReplanAndOwnerRereadInHaltUnchanged`** — `runDeathReplan` at 16:30 on the stale tape → `⛔ … refused death_replan: stale_bars_…`, no row, counter 0; `CanForceReread` at 16:30 → refused "no session is active" (the market-closed gate is the next layer). PASS.
**E6 · `TestClass36EveryRefusalEmitsTheLine`** — asserts on the captured WARN+ log: `no_bars` (level_event), `stale_bars_*` (death_replan), a scheduled read on a stale **open** tape (`session_read: stale_bars_*` — still refused), and the bypass line for the scheduled read in the halt. PASS.
**E7 · `TestClass36ScheduledReadFiresOnceInHalt`** — PASS. **E8 · `TestClass36LondonAndNYUnchangedWithLiveBars`** — LONDON 01:30 and NY 08:00 with a 1-minute-old tape land as `<S>_scheduled_read`, no bypass line. PASS. Plus `TestClass36LineBuilders`, `TestClass36TriggerClassTable`.
Adjacent suites still green: `TestClass32*` (4), `TestP0BAsiaRead*` (2), `TestPlannerPreflight` (its three calls now pass `level_event`, a non-scheduled class — the assertions are about the check itself), `TestSundayAsiaDeferred`, `TestWeekly*`.

**E9 ·** `go test ./...` → **26 packages ok**, the only failure being the Sunday fixture's own date (fixed; `nofx/trader` then `ok 27.167s` in full); `go test ./kernel -run Golden` → **PASS**; `vitest run` → **37 files / 295 tests passed**; `tsc --noEmit` clean; `go vet ./trader` clean.

---

## F — CUTOVER STATE

- **F1** merged to dev `17efeea9`; clean-clone build quoted above (`vcs.modified=false`).
- **F2/F3** NOT yet run: the flat-gate quadruple, in-flight check and session window are quoted at cutover time, fresh. The binary sits in scratch (`<scratchpad>/clone36/nofx-bin.next`) and is copied to `~/nofx/nofx-bin.next` only once the main-tree lock is mine; `nofx-bin.prev.boot` is refreshed at swap time.
- **F4** *** OWNER GO REQUIRED. *** Owner unavailable = HOLD.
- **F5** on GO: swap + `kill -9`, then quote the boot checklist — rev, `🔐 BOOT INTEGRITY OK`, goldens PASS, the session-reads line, the conditions line, the validator-hints line, `🧮 replan budget` (class 35) and the NEW `🗓 preflight:` line.
- **Rollback (exact):** `cd /home/hoang/nofx && mv nofx-bin nofx-bin.bad.17efeea9 && cp nofx-bin.prev.boot nofx-bin && printf 'ec6632f9de41060b52398f41f9ffbbf840814c40' > deploy/RELEASE && kill -9 $(pgrep -f '^/home/hoang/nofx/nofx-bin$')` (then revert the marker on dev if the rollback sticks).
- **F7 — the TRUE live proof has not occurred (A20).** It is the next scheduled read inside a halt: **ASIA at 16:30 CT on 2026-09-02** and the **weekly at 16:30 CT on Sunday 2026-09-06**. Until then E1/E2 are the fixture proof. Follow-up owed at ~16:35 CT 2026-09-02: quote the read start timestamp, the `🗓 preflight bypass (class 36)` line, and the `PLAN written … ASIA v1` time — the plan must be on the desk before 17:00.

## A15 — what the owner will still see wrong

- Until cutover, tomorrow's 16:30 ASIA read would refuse exactly as today's did (`stale_bars_*` for 30 minutes, launch at 17:0x). The fix is on dev, built, and parked.
- Until the main tree is fast-forwarded (needs the lock), the vite dev server still serves the class-35 guide text; after the pull and before the swap, the guide drift banner shows (`GUIDE_BUILT_REV` 17efeea9 vs running ec6632f9) — the designed failsafe.
- The halt-read log lines from class 32 remain INFO (journald-suppressed); the class-36 bypass line is WARN and will show. The old `🛑 planner_preflight_refused` wording is gone — anything grepping for it (reports only) must grep `⛔ planner preflight refused`.
- Today's ASIA v1 `planner_fail_closed` chain stays as written; this wave does not re-read it.

## Closeout

Commits on dev: `17efeea9` (fix + tests + guide + checklist) · `7089d271` marker (RELEASE + GUIDE_BUILT_REV = 17efeea9) · this report. Worktree removed and repo memory updated at closeout; the main-tree lock was never mine to release (held live by `1906840`) — the cutover step re-acquires it after that dispatch releases.

---

## CUTOVER — DONE (owner GO, 2026-09-01) [A]

- **GO received 18:00 CT.** Lock re-acquired (pid 1860416, no holder present). Gates at 18:00:23 CT: window OK (outside 16:45–17:10); DB OPEN 0, armed 0, API positions `[]`, open-orders `[]`, NT8 `positions snapshot account=Sim101 count=0` — **but `replan_in_flight: true`**: a level_event wake re-read on the ASIA chain was on `planner attempt 3/3` since 17:53:33. **Held per A6.**
- **Read landed 18:00:41 CT:** `📐 planner attempt 3/3 parse/schema rejected: arm legs on breakdown_continue …` → `🚨 PLANNER FAIL-CLOSED 2026-09-01 ASIA` → `🗓️ PLAN written 2026-09-01 ASIA v2 (… lifecycle no_trade)`; `replan_in_flight` → false. No claim after it. Gates re-quoted 18:00:53 CT: DB OPEN 0, armed 0, positions `[]`, open-orders `[]`, NT8 `count=0` (18:00:52), ASIA `armed: {}`.
- **Swap 18:01:06 CT:** `cp nofx-bin nofx-bin.prev.boot` · `mv nofx-bin nofx-bin.old.ec6632f9` · `mv nofx-bin.next nofx-bin` (rev check `17efeea9` first) · `kill -9 1908258`.
- **Boot checklist (F5), 5 s after the kill, 18:01:11 CT:**
  `🔐 BOOT INTEGRITY OK — rev 17efeea9fc59 · built 2026-09-01T22:52:57Z · expected 17efeea9fc59 · goldens PASS`
  `🗓 session reads (owner ruling 2026-08-31, open−30): ASIA 16:30 · LONDON 01:30 · NY 08:00 CT — windows/flats unchanged; Sunday weekly 16:30 → ASIA follows`
  `📜 scenario schema: 9 conditions [acceptance, breakdown_continue, breakout_retest, breakup_continue, fvg_entry, hold, reclaim, reject, sweep_reclaim]`
  `🧪 validator hints: 6 sites — every condition token legal + live (class 34 guard)`
  `🔐 confirm rules: 5 [1m_mss, 1x5m_close, 2x5m_close, time_hold, touch]`
  `🧮 replan budget: recorded-counter (class 35) — spends: death_replan, owner_reread · free: …`
  **`🗓 preflight: scheduled reads bypass freshness in halt/weekend (class 36); executor halt-block unchanged (cmeSessionClosedSkip / IsCMEOpen)`** ← NEW
  `🎛 entry law: bd_min_closes=1 bd_min_disp_atr=1.00 mss_min_disp_atr=0.50 …` · `📐 NT8 instrument_info MNQ (MNQ 09-26): point_value=2 tick=0.25 — matches table ✓`
  Exactly ONE PID: `1941026`. `go version -m nofx-bin` → `vcs.revision=17efeea9fc5909473a40e60418428b521a2f1574`. Feed re-warmed: `received frame type=bars_historical` ×2 at 18:01:30; newest MNQ 1m bar `2026-09-01 18:00:00 CT`. `[ERRO]`/panic lines since boot: **0**. Positions after boot: `[]`. ASIA via API on the new binary: v2 no_trade, `replans_left 4/4` (class 35 intact), `replan_in_flight false`.
- **Rollback (still valid):** `mv nofx-bin nofx-bin.bad.17efeea9 && cp nofx-bin.prev.boot nofx-bin && printf 'ec6632f9de41060b52398f41f9ffbbf840814c40' > deploy/RELEASE && kill -9 1941026`.
- **Not yet proven live (A20/F7):** no scheduled read has run inside a halt on this binary yet. The proving events are **ASIA 16:30 CT 2026-09-02** (expect `🗓 session read fired during halt (ASIA)` + `🗓 preflight bypass (class 36) …` + `PLAN written … ASIA v1` before 17:00) and the **Sunday 2026-09-06 16:30 CT weekly** (expect `📅 WEEKLY READ starting` from the wall-clock path, then the ASIA read). Follow-up owed at ~16:35 CT 2026-09-02.
