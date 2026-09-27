# CLOCK-SEAM + FLAKE SWEEP (checklist class 72; closes class 60's sweep)

**Branch:** `fix/clock-seam-sweep` off `13680723` · **Scope:** seams and tests only
**Checklist:** entry **72** — number assigned AT MERGE (A16); highest occupied on dev was 71.
**Behaviour change: none.** Every production diff is an entry point delegating to an
`…At(now, …)` variant, or a comment. No rule, threshold, window, gate or exit moved.

---

## 1. The entry-point table

Class 60 listed eight test files that build a fixed clock and call a wall-clock entry point.
Tracing each to the production function it reaches gives thirteen time-dependent rules and
three deliberate exclusions.

| rule | file | before | after |
|---|---|---|---|
| `latestClosedPrimaryBarMs` | auto_trader_clock.go | `nowMs := time.Now().UnixMilli()` in the body | delegate → `…At(now)` |
| `recordClosedTradeAnalytics` | auto_trader_clock.go | two `time.Now()` sites in the body | delegate → `…At(now, p)` |
| `maybeRecordClosedTradeAnalytics` | auto_trader_clock.go | `time.Now()` + called the wall-clock sibling | delegate → `…At(now)`, threads `now` down |
| `entryBlockedByLastEntry` | auto_trader_clock.go | already seamed | unchanged ✓ |
| `enforceEODFlat` | auto_trader_clock.go | already seamed | unchanged ✓ |
| `enforceT1ForceFlat` | auto_trader_clock.go | already seamed | unchanged ✓ |
| `observeTransitionStanddown` | auto_trader_transition.go | `now := time.Now().UnixMilli()` | delegate → `…At(nowT, ctx)` |
| `maybeWakePlannerOnMSS` | auto_trader_transition.go | two `time.Now()` sites | delegate → `…At(now, …)` |
| `maybeWakePlannerOnLevelEvents` | auto_trader_wake_levels.go | seamed by class 60 itself | unchanged ✓ |
| `weeklyConfluenceShadow` | auto_trader_weekly.go | `now := time.Now()` | delegate → `…At(now, …)` |
| `weeklyScenarioGrade` | auto_trader_weekly.go | `now := time.Now()` | delegate → `…At(now, cited)` |
| `ResetDailyPnL` | kernel/risk_limits.go | `CMESessionDayKey(time.Now())` | delegate → `ResetDailyPnLAt(now)` |
| `barPersistSummary` | provider/ninjatrader/bar_persist.go | 60 s wall-clock gate | delegate → `…At(nowT)` |

**Deliberately NOT seamed**, with the reason recorded beside the list rather than left to be
rediscovered — an unexplained absence is how a list stops being trusted:

- `ForceReset`'s `time.Now()` is a **poll deadline for a real wait** on an in-flight read,
  not a rule. Feeding it fixture time would make the wait instant or infinite: a behaviour
  change, which this wave forbids.
- `tickOnce` is the loop entry; its only clock use already delegates to the seamed
  `staleDodgeCheck(now)`.
- `kernel/tz.go` `NowCT` is the clock accessor itself — every consumer (`ClockCT`,
  `ClockCTAndUTC`, `ClockCTSeconds`, `TableTimeCT`) already takes an explicit `t`. **The
  production side of `kernel/tz.go` was already clean**; it appeared on class 60's list on
  the strength of its test file alone.
- `trader/binance/order_sync.go` — live-network request timestamps behind
  `BINANCE_LIVE_TEST`; not in the deploy gate.

## 2. The lint (D2)

`clock-seams.list` at the repo root is the single list; `trader/clock_seam_lint_test.go`
reads it and asserts, per row, that the `…At` variant exists **and** that the entry point is
a one-line delegate. The test carries no copy of the rule names (A24).

The one-line requirement is the half that matters: an entry point that reads the clock *and*
does work is class 60 exactly. Mutation-checked — reinstating a benign two-statement form
(`now := time.Now(); return at.weeklyScenarioGradeAt(now, cited)`) fails it:

```
"weeklyScenarioGrade" must be a ONE-LINE delegate to "weeklyScenarioGradeAt",
has 2 statements: [now := time.Now() return at.weeklyScenarioGradeAt(now, cited)]
```

E2 is pinned permanently against synthetic source inside the test, so the detector's own
correctness does not depend on a real file staying broken.

## 3. E1 — the time-of-day pin

RED first, and the RED is the defect stated exactly:

```
vet: trader/clock_seam_test.go:51:18: at.latestClosedPrimaryBarMsAt undefined
```

There was no way to state a clock. The headline pin is a real bomb rather than a compile
error: `latestClosedPrimaryBarMs` answers "which bar has closed by now", with fixture bars
closing at 11:30 and 14:00 CT. At 11:00 neither has closed; at 14:50 the latest is the 14:00
bar. One assertion, two clocks, and no wall-clock reading can satisfy both. A companion pin
asserts the entry point still reads the real clock, so the seam cannot quietly become a
behaviour change.

## 4. The flake — it was never load (D3/D4)

`TestFanOutClosesLastResortIsHonest` failed ~1 full-suite run in 4-6, passed in isolation
every time, and had been carried as "a load flake" — **including by me, in this checklist.**

**It was a clock.** The drop path increments both counters and then calls
`barPersistSummary()`, which is rate-limited to once per 60 **wall-clock seconds** and, when
it fires, `Swap(0)`s both — erasing the increment one line before the test reads it. At
~6.3 s per iteration, roughly every tenth run crossed a 60-second boundary.

The numbers identified it after two wrong hypotheses:

```
failures: 6.01s   closes_dropped=0  queue_drops=0     ← neither branch's counter survived
passes:   6.30s   closes_dropped=1  queue_drops=1
queue:    4096/4096 in EVERY run, pass and fail alike
```

`closes_dropped=0 AND queue_drops=0` is impossible for either code path — the close branch
increments both, the non-close branch increments one. That contradiction is what pointed at
a destructive reader rather than at the branch taken.

**Two wrong hypotheses, both plausible, both fixed and both useless**, recorded because the
fixes stayed (they made the test honest even though they were not the bug): a stale queue
inherited from the previous `-count` iteration, and the worker freeing a slot mid-retry. The
tell was that the queue read 4096/4096 in the failures too.

Fixed test-side only:
1. fill until the queue is **observably** at capacity, instead of by a magic `cap+512`;
2. confirm the worker is wedged **inside** the persister before asserting;
3. hold the summary's rate limiter open across the assertion window so the destructive reset
   cannot fire.

Production is unchanged; `barPersistSummary` gained a seam and still summarises on its own
schedule in the bot.

**The flake was itself a class-60 instance.** The sweep went looking for wall-clock rules and
found one hiding behind a counter, which is why both halves belong in one wave.

## 5. Results

| id | result |
|---|---|
| E1 | RED quoted (`latestClosedPrimaryBarMsAt undefined`), then GREEN at both 11:00 and 14:50 CT |
| E2 | lint fails a 2-statement entry (quoted above); negative case pinned synthetically |
| E3 | `TestFanOutCloses*` `-race -count=30` → **ok**; whole package `-race -count=10 -p 8` → **ok 181.685s, RC=0, no data race** |
| E4 | full suite **3× consecutively: RUN 1 EXIT=0 · RUN 2 EXIT=0 · RUN 3 EXIT=0**, 28 packages ok each |
| E5 | goldens **3/3 PASS** (`futures-empty 41564a7a` · `futures-keylevels efd93ac6` · `futures-plan 270e4733`) · vitest 42 files / 336 tests · tsc clean |

E4 is the point of the wave and the only result that could have contradicted it: before this
branch the same command failed roughly one run in four to six.

## 6. Standing findings, reported not fixed

- **`barPersistSummary` destroys what it reports.** `Swap(0)` inside a log path means any
  reader racing the summary loses the count. Harmless in production today (nothing reads the
  counters synchronously) but it is the "counters record, never infer" rule pointing the
  other way, and it is exactly what made a gate test lie. Left alone under this wave's
  zero-behaviour-change rule. nofx-47's framing on reading it, which is sharper than mine:
  the log line is the ONLY consumer that can ever observe a nonzero value, so anything else
  that reads those counters will be wrong intermittently and silently — a class-35 problem
  ("counters record, never infer") in waiting. It wants its own wave with its own boot line,
  not a test-side accommodation kept forever.
- `gofmt -l kernel` reports **10 pre-existing unformatted files on dev**, none of them mine
  (verified by stashing this branch's diff and re-running). Not touched — out of scope, and
  reformatting them would bury this wave's diff.
