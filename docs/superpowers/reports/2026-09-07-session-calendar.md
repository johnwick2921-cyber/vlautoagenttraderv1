# SESSION CALENDAR — a shortened session is a TRADING session

**Wave:** session calendar (takeover) · **Branch:** `fix/session-calendar`
**Lane:** `sessioncal-ee7f9468/nofx-db[ca9c60]` · **Date:** 2026-09-07
**Booted:** `5457ac5a` at 20:39:39 CT, pid 3058590 · **Marker:** `af3472ee`

---

## 0. What it cost, and what it costs now

`isCMEHoliday` answered yes/no and the gate treated every holiday as a FULL
closure. The code conceded it in its own comment — *"for v1 we treat them as
full closures and refuse to trade. Refine in Plan 3 if it becomes
restrictive."* On 2026-09-07 it became restrictive: MNQ traded a full Labor Day
session while the gate called the market shut, no LONDON plan was ever read, and
the dashboard showed `CME CLOSED (holiday)` beside a feed carrying a bar seconds
old.

The day's real shape, now enforced: **trades to 12:00 CT · halted · reopens
17:00 · evening session runs.**

## 1. TAKEOVER, not a fresh claim

Base `be67c688` from lane `session-calendar-554049f5`, which claimed at
09:47:55 and marked its own tip *"superseded mid-flight"* at 09:50:31. Its 233
lines (`kernel/session_calendar.go` + `.json`) were **kept, not re-derived** —
but they were completely dormant: A29 production call sites were **0 for every
function**, and nothing in `kernel/cme_calendar.go`, `trader/` or `api/`
referenced them. The missing half was the wiring.

Step 0 stops two lanes writing one wave. It has no detector for both lanes
yielding to each other, which is what happened: I stood down at 10:01 on the
owner's instruction, and a peer (`nofx-2c`) pointed out the branch had declared
itself superseded 11 minutes *before* I ever read it.

## 2. C1–C5 verified, and three premises CORRECTED

| # | claim | verdict |
|---|---|---|
| C1 | `isCMEHoliday` treats Labor Day as a full closure | **CONFIRMED** — comment quoted above |
| C2 | 156 overrun warnings | **CORRECTED: 165** in `nofx_2026-09-06.log`; 182 across both boot logs. All `3m0.0XXs > 2m0s`, min `3m0.009s`, max `3m0.159s` |
| C2 | log volume ~733 → ~81 lines/hour | **NOT REPRODUCED.** Real collapse 1283 → 171 → ~141/h loop-only (9.1×) |
| C3 | the warning fires by construction | **CONFIRMED** — 3-min backoff (`const cmeClosedBackoff`, hard literal) vs `ScanInterval` = **2 min** from the DB (`scan_interval_minutes=2`, no env fallback) |
| C4 | most CME holidays are early closes | **CONFIRMED**, and the sources were already in the repo (§3) |
| C5 | the strip disagrees with itself | **CONFIRMED** — `CME CLOSED (holiday)` beside a live bar |

**Unmeasured ticks (A24):** kick-path cycles (`auto_trader.go`, the `at.kickCh`
arm) run with **no overrun check at all**, so 2–5 ticks/hour were never
measured. Any "removed N warnings" claim must be measured, never predicted.

## 3. THE FOLD — one fact had three owners

The dispatch's premise was that the calendar was new. It was not: **`half_days.json`
already existed** at the repo root, loaded by `trader/auto_trader_halfdays.go`,
seeded into `SessionRegistry.HalfDays` — a **third** representation. Two key
conventions, and on three dates two different times:

| date | `half_days.json` | inherited calendar | ruling |
|---|---|---|---|
| 2026-09-07 | 12:00 | shortened 12:00 | agree |
| 2026-11-26 | 12:00 | closed | **owner: FULL CLOSURE**; the 12:00 row recorded as `superseded`, citation intact |
| 2026-11-27 | **12:15** | shortened 12:00 | **sourced 12:15 wins** |
| 2026-12-24 | **12:15** | shortened 12:00 | **sourced 12:15 wins** |

`half_days.json` carried **archived CME PDF citations** — far better provenance
than the calendar's convention guesses, three of which said *"NOT owner-verified"*
about themselves. Folded in **verbatim with the citations**, `half_days.json`
deleted, `SessionRegistry.HalfDays` removed, and `EffectiveFlatCT` +
`halfDayCutoffMin` + `effectiveEODFlatCT` all repointed at
`kernel.SessionEarlyCloseCTForKey`.

**Pin:** `TestFoldOneFactOneOwner` — the gate and the EOD flat must return the
**same value** for 2026-11-27 and 2026-12-24 (`12:15`), and Thanksgiving must
expose no early close at all. It fails the moment a second copy reappears.

**Unestablished → closed (C4, owner ruling).** Five rows whose only basis was
"the equity-index convention" fall to `closed` and are flagged `unestablished`:
MLK, Presidents, Memorial, Juneteenth, Jul-3 observed. A guessed trading day is
worse than a missed one. The boot line prints `unknown-dates=5`.

## 4. FOUR DEFECTS THE WORK EXPOSED — all mine, all caught by pins

**(a) A coverage gap that would have traded a closed day.** A day-by-day diff of
the calendar against the superseded boolean across all of 2026 found two dates
the boolean closed and the calendar did not list. `2026-07-04` is a **Saturday**
(benign). **`2026-12-31` is a Thursday** and would have become a fully normal
trading day. Gaps now 0.

**(b) A listed `normal` row fell to CLOSED.** My `switch` had no `SessionNormal`
arm, so it hit `default` → closed. That is how sourced-normal 2026-12-31 would
have stayed shut — and it also corrected my own earlier unestablished-CLOSED
default for that date, because the folded file **sources** it as a normal equity
session (only rates settle early).

**(c) The early close landed on a session that starts after it.** Activating the
dormant half-day path put today's 12:00 close onto **ASIA, which begins at
17:00** — a flat five hours before the session opened. The old `HalfDays` map
was always empty, so the path had never fired. `EffectiveFlatCT` is now pull-in
only and never earlier than the session's own `window_start_ct`. Caught by
`TestW7LevelBurnedStaysBurnedAcrossSessions` — which I **assumed was
pre-existing**, checked against `origin/dev`, and **was wrong**.

**(d) The whole calendar date stayed shut.** CME's own carried-over wording is
*"halt 12:00 CT, **reopen 17:00 CT**"*. I had the date closed through midnight,
so **tonight's ASIA session would have been skipped** — the identical loss this
calendar exists to prevent, one layer down, shipping inside the fix. Found at
cutover by re-reading the source line. `shortenedDayHalted` reuses the weekly
rule's own reopen boundary rather than adding a second session-time literal.

## 5. PINS — every one RED before GREEN

**E1 (RED first, verbatim):**
```
E1: 04:00 CT on a SHORTENED day must be OPEN — IsCMEOpen said closed
E1: 04:00 CT on a SHORTENED day must not be closed; got reason "holiday"
E1: 09:00 / 11:59 CT — same
E1: 12:00 / 13:30 / 15:00 CT reason must name the EARLY CLOSE, not "holiday" — the day traded; got "holiday"
```
Nine failures, every one `reason "holiday"`. → GREEN.

**E5 (RED first):** `undefined: shouldWarnOverrun` → 3/3 GREEN.

**E1.d** evening reopen · **E2** full closure · **E3** normal day is the weekly
rule alone, no status note · **E4** all five unestablished rows refuse AND are
named on both surfaces · **E6** fixed-clock, repeated evaluation, no hidden
`time.Now()` · **E7** A29 below.

**Claim pins 10 → 17** (`deploy/nofx-claim-test.sh`): `CLAIM_RE` tightened to
require the routable composite form with `[unlisted]` accepted; the two
historical bare-session claims relabelled LEGACY and expected to FAIL; seven
ROUTE cases added including the real 2026-09-07 collision message.

## 6. A29 — the dormant island, wired

| function | call sites inherited | now |
|---|---|---|
| `SessionCalendarBootLine` | 0 | 2 |
| `SessionStateAt` | — | 8 |
| `SessionEarlyCloseCTForKey` | — | 4 |
| `SessionDayNote` | — | 2 |
| `SessionShortenedDays` | — | 2 |
| `shouldWarnOverrun` | — | 2 |
| `SessionCalendarUnestablishedCount` | — | 2 |
| `weeklyCMEOpen` | **0 (dead as committed)** | live |

## 7. THE CLEAN-CLONE STEP EARNED ITS PLACE

`TestTZGuardSingleTimeSource` **passed in my worktree and failed in the clean
clone at the same commit** — five bare `"15:04"` layouts
(`cme_calendar.go:67`, `session_calendar.go:148/295/322/378`). All five would
have shipped.

**SURPRISE, reported not fixed (out of footprint):** the tz guard is **blind
inside a git worktree**. WORKTREE LAW requires all work to happen in one, so any
wave can carry bare layouts to cutover. Deserves its own class.

Fixing it exposed a second thing: routing everything through `ClockCT` (which
appends `" CT"`) fed `"12:00 CT"` into a minutes parser expecting `"12:00"`, and
the flat comparison silently stopped matching. Hence **two** helpers —
`CloseClockCT` for display, `CloseHHMMCT` for data.

## 8. CUTOVER

Suite at the merged HEAD in a clean clone named `nofx`: **28/28, 0 FAIL**.
`GUIDE_BUILT_REV` read from the binary's `vcs.revision` (`vcs.modified=false`),
**then** dist rebuilt — new rev ×1 in the bundle, stale rev ×0. `tsc` clean.

**Five-leg gate, own fresh read at 19:41:**

| leg | result |
|---|---|
| 1 db OPEN positions | PASS — 0 |
| 2 api positions | **FAIL — UNEVALUABLE** (401, token denied this lane). Legs 1 and 3 corroborate flat independently |
| 3 NT8 snapshot | PASS — 0 orders, 0 working, 17.3 s fresh |
| 4 armed_orders non-terminal | PASS — 0 *(never overridden)* |
| 5 planner in-flight | PASS — 0 |

**A19 five references, all `5457ac5a`:** RELEASE file · binary `vcs.revision`
(`modified=false`) · `HEAD:deploy/RELEASE` · `GUIDE_BUILT_REV` · `/api/health`.

**Rev vs dev tip:** the five read `5457ac5a` while dev's tip is `4897ce83`+.
Correct, not a mismatch — the check compares `HEAD:deploy/RELEASE` (file
content), not the tip sha, and `5457ac5a..4897ce83` is three `.md` files with
**zero** `.go`/`.ts`/`.cs`.

**Boot lines, READ:**
```
🔐 BOOT INTEGRITY OK — rev 5457ac5accd9 · built 2026-09-08T00:39:02Z · expected 5457ac5a · goldens PASS
🗓 session calendar: today=shortened close=12:00 CT source=CME published ·
   unknown-dates=5 · dates=14 covered=[2026] · backoff=UNKNOWN (not reported by the loop yet)
🗓 half-days: 3 early close(s) in the session calendar · next: 2026-09-07 12:00 CT (Labor Day)
   · source=kernel/session_calendar.json (folded 2026-09-07; half_days.json deleted)
```

## 9. PROOF — measured, never predicted

**E1, PROVEN LIVE.** The last `🌙 CME closed` line is **19:27:04**, under the old
binary, which called today a full holiday. Since the 20:39:39 boot: **zero**
CME-closed lines, and the loop is **cycling** —
```
20:39:39 ⏰ 2026-09-07 20:39 CT - AI decision cycle
20:41:39 ⏰ 2026-09-07 20:41 CT - AI decision cycle
```
Exactly 2 minutes apart, the scan interval. Under the old rule this process
would have been inside `backoffWhileClosed`. **The session the old binary would
have skipped is being traded.**

**D4 — NOT YET PROVEN, stated plainly.** Overrun warnings: **165** in the 09-06
log, **117** in the 09-07 log before the boot, **0** since. But the market is
**open** right now, so the closed path has not been exercised (0 CME-closed
lines since boot). That 0 is consistent with both "the exemption works" and "the
closed path never ran". **The real proof is the next closed tick** — first
CME-closed stretch after this boot, expected at the 16:00–17:00 daily break or
the Friday close.

**Not yet observed:** the DESK MODE row carrying the classification. It is served
by `/api/desk` (JWT-protected) and is not logged, so this lane could not read it.

## 10. A15 — what the owner will STILL see wrong

- **`backoff=UNKNOWN` on the boot line.** `SetClosedBackoffForBootLine` is called
  inside `backoffWhileClosed`, which only runs when the market is CLOSED — so at
  boot the value has never been published. The A11 intent was right (read, never
  a literal) and the **seam is wrong**: the publish belongs at init. The line is
  truthful today and will read UNKNOWN on every boot until moved. **Follow-up.**
- **Five dates say `closed` that CME may trade** (MLK, Presidents, Memorial,
  Juneteenth, Jul-3). That is the ruled safe side, not an accident — but it is
  five sessions the bot will skip until someone checks CME's published calendar.
- **`covered_years` is `[2026]` only.** 2027 falls back to the superseded
  boolean and the boot line says so.
- **The tz guard still can't see a worktree** (§7).
- **Leg 2 of the flat gate remains unevaluable** to any lane without a token.

## 11. Rollback

```
mv nofx-bin nofx-bin.failed.5457ac5a \
  && mv nofx-bin.old.b4195e6f.20260907-204500 nofx-bin \
  && printf '%s' "b4195e6f" > deploy/RELEASE && kill -9 $(pgrep -f '/home/hoang/nofx/nofx-bin')
```
Backups: `~/nofx-backups/sessioncal-2026-09-07/` — binary, RELEASE, `data.db`.

## 12. An operator error, recorded

A one-word reply arrived while my liveness check was minutes stale. I read it as
*stand down*, restored the OLD binary and reverted RELEASE — **while the new
binary was already running under a new pid**. Disk said `b4195e6f`, process said
`5457ac5a`, and a systemd restart would have silently reverted the deploy.

Caught within one command by the VERIFY step, which prints the **running** rev
beside the **disk** rev. A swap step that printed only what it wrote would have
shown a clean `b4195e6f` and hidden it. The lesson is the verify, not the slip.

Separately: my lock heartbeat went **STALE at 2107 s** while the swap was staged
and I waited on a human — caught by a peer, fixed with a detached keeper. The
heartbeat had been living inside the loops that waited on builds, so it stopped
exactly when the work stopped. **Filed as class 88** (highest occupied at merge:
87 — re-checked, not assumed).
