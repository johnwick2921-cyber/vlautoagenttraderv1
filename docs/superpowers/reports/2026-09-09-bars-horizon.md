# BARS HORIZON — a count is not a horizon

**Branch** `fix/bars-horizon` · **base** `origin/dev` @ **`6a8e14c9`** (rebased THREE times — `05125bd6` → `8dfe6bc1` → `27e062ea` → `6a8e14c9`; see SPEC FRESHNESS). `git merge-base HEAD origin/dev` = `6a8e14c9` = the dev tip, so the branch is a strict fast-forward ahead of dev and is **not merged**.
**Session** bars-horizon-2bdef526/nofx-07[aa8e26] · **worktree** `/home/hoang/nofx-barshorizon`
**Status** pushed, green, **NOT deployed and NOT merged** (A3).
**Rounds** built → 3 reviewers → **REPAIRED (this round, 2026-09-09 evening)**.
**Head** — the code freeze is `54a7bf25` (the history join below); report-only
commits follow it. A18: a push exiting zero proves a ref matched and nothing
more, so `git ls-remote origin fix/bars-horizon` was READ BACK after every push
and compared to local `HEAD`; they matched every time. The final value is in the
dispatch's structured result, not written here — a document cannot name the sha
of the commit that contains it without being wrong by one.

> **HOW THIS WAS PUSHED, because it is not a plain fast-forward.** The dispatch
> required a rebase onto the dev tip, which moved the branch off its own pushed
> tip `53786a44`. Force-push is not available to this session, so the old tip was
> **joined, not overwritten**: `git merge -s ours 53786a44`, a history join that
> changes **no content** (`git diff HEAD^1 HEAD --stat` is empty). Verified
> before the join that `git diff 53786a44 HEAD` is exactly (a) this round's
> additions, (b) the deliberate corrections to sentences the reviewers proved
> false, and (c) dev's own `deploy/RELEASE` + `web/src/guide/types.ts` moving
> forward at `6a8e14c9`. Nothing from the old tip is lost. Nothing was rewritten.
The repair fixes both D1 blockers, implements the owner's D3 ruling with all
four conditions, and corrects every sentence the reviewers proved false.
**Checklist numbers** not assigned — two new classes are appended to
`docs/superpowers/AUDIT-CHECKLIST.md` under PENDING NUMBERS (A16: numbers are assigned AT MERGE).

Evidence tiers: **[A]** verified first-hand · **[B]** inferred · **[C]** speculation.

---

## 0 · RETRACTION — two premises this wave was launched on were WRONG

Both were stated to the owner as fact. Both are corrected here rather than
quietly dropped.

### R1 — `scope_bars` ALREADY RECORDS THE SERVED COUNT. It never recorded the request.

`trader/auto_trader_planner.go` has always written `ScopeBars: len(scope.Bars)`
— the post-truncation, SERVED length. **[A]** On `origin/dev` that is
**line 2499** (`git show origin/dev:trader/auto_trader_planner.go | grep -n
'ScopeBars:'` → `2499:		ScopeBars:    len(scope.Bars),`); on this branch it is
line 2963, moved by the surrounding edits and **not changed**. Any design premised on
"`scope_bars` reports the request" is wrong, and redefining the column would
have silently rewritten the meaning of 67 correct rows.

```
sqlite> select count(*), min(id), max(id), group_concat(distinct scope_bars),
                group_concat(distinct scope_intv) from planner_read_facts;
67|1|67|2000|1m
```

**The void scope has NEVER been short: 67 of 67 rows (ids 1–67) record
`scope_bars=2000` against a 2000 ask.** **[A]** This is the load-bearing
counter-evidence for the whole wave: the bridge's silence is real, but at the
~29 sites that ask 2000 it has cost nothing.

**RE-MEASURED at repair time (2026-09-09 18:3x CT), because the table grew while
the wave was in review** — the conclusion is unchanged and the figure is not:

```
sqlite> select count(*), min(id), max(id), count(distinct scope_bars),
                group_concat(distinct scope_bars) from planner_read_facts;
68|1|68|1|2000
```

**68 of 68 rows (ids 1–68), one distinct value, `scope_bars=2000`.** **[A]**
The dispatch's 67/67 was correct when it was taken; a 68th read has since
landed and records the same thing. Both figures are printed rather than one
silently replacing the other (A21).

`scope_bars` is therefore left **exactly as it was**, with a comment in
`buildReadFactRow` saying why. What was missing is that **a COUNT cannot express
a SPAN or a HOLE** — see §C5 and D4.

### R2 — THE ATR WAS NOT INFLATED, AND ARM 133 IS NOT OVER-SIZED.

The claim rested on comparing today's **NY** read against today's **ASIA** read.
Those are two volatility regimes; the comparison could only produce a difference,
and the difference was read as a defect.

Held to the same session — `planner_read_facts`, verified first-hand **[A]**:

| session | id | atr5m | stop_floor_pts | created (CT) |
|---|---|---|---|---|
| 09-08 NY | 54 | 36.7439730211274 | 55.115959531691 | 2026-09-08 10:02:43 |
| 09-08 NY | 55 | 32.2843805014897 | 48.4265707522345 | 2026-09-08 10:45:01 |
| 09-08 NY | 56 | 28.0296201145494 | 42.044430171824 | 2026-09-08 12:15:00 |
| 09-09 NY | 65 | 20.514183795639 | 30.7712756934586 | 2026-09-09 13:08:08 |
| 09-09 NY | 66 | 19.7631429163709 | 29.6447143745564 | 2026-09-09 13:18:13 |
| 09-09 NY | 67 | 16.6611226001067 | 24.99168390016 | 2026-09-09 13:55:06 |

**Today's NY ATR is roughly HALF yesterday's NY ATR.** Arm 133 (composed
13:18:13 CT, the same second as row 66; long limit 29424.5, stop 29393.5 = 31.0
pts) was **not** composed on an inflated floor. Nothing in this wave touches arm
133 or any order path (owner ruling).

### The method error, stated as its own line

> **A CROSS-SESSION ATR COMPARISON IS A REGIME COMPARISON, NOT A MEASUREMENT.**

A claim about a level, a floor, a size or a threshold is a measurement only when
both sides come from the same session and the same calendar class, and the
report quotes the row ids on both sides (A21). Filed as a new bug class.

---

## SPEC FRESHNESS (class 73)

`git log -1` on origin/dev for every file this wave built against:

```
kernel/planner_prompt.go        486cec46 2026-09-09 14:15:47 -0500  docs(W5): the system describes itself truthfully
kernel/weekly_bias.go           23243670 2026-08-30 09:40:45 -0500  fix(weekly-bias): calendar-anchored week governing Monday
kernel/session_calendar.go      5457ac5a 2026-09-07 19:39:02 -0500  fix(session-calendar): five bare "15:04" layouts
provider/ninjatrader/bar_cache.go 2b22eed2 2026-08-27 16:45:24 -0500 fix(bar-truth): open-stamp the historical persistence path
store/bar_history.go            5821ae55 2026-09-02 12:25:16 -0500  feat(bars): persist EVERY cached TF + per-TF retention
trader/auto_trader_planner.go   0babd090 2026-09-08 16:22:32 -0500  feat(research): link attempts, permissions, broker receipts
trader/auto_trader_weekly.go    21b3e75e 2026-09-03 22:50:35 -0500  fix(class 60/72): clock seams
kernel/regime.go                7b19e753 2026-09-03 23:00:08 -0500  style(kernel): gofmt
kernel/regime_baseline.go       d85f6f13 2026-08-15 12:55:09 -0500  feat(dayplan): W11b
trader/ninjatrader/bar_persist_wire.go 56904ec1 2026-09-02 21:29:52 -0500 feat(no-chase): riders R1+R2
kernel/void_scope.go            60f214d9 2026-09-02 22:38:48 -0500  fix(void-parity): the scope VALUE is the session day
store/planner_read_facts.go     4659874a 2026-09-02 22:38:48 -0500  fix(void-parity): one resolver for the void scope
docs/superpowers/SYSTEM-MAP.md  27e062ea 2026-09-09 14:55:48 -0500  merge dev (W5) + correct three sentences
docs/superpowers/AUDIT-CHECKLIST.md 27e062ea 2026-09-09 14:55:48 -0500  (same commit)
```

**Dev moved under this wave THREE times and it was rebased each time.** The
third move landed during the review round: `6a8e14c9`, a deploy boot marker
setting `RELEASE=27e062ea` + `GUIDE_BUILT_REV=27e062ea` from the main tree. That
marker is exactly what makes the guide's drift banner silent for this branch
(§SCOPE), which would have been invisible without re-reading the tip.

**Dev moved under this wave twice before that and it was rebased both times.** The first
base (`05125bd6`, a `fix/brand-visible` merge) was replaced by `8dfe6bc1`; then
`SYSTEM-MAP.md` and `AUDIT-CHECKLIST.md` — two files this wave edits — moved at
14:55:48 in `27e062ea`, **after** that base, so the branch was rebased again
onto `27e062ea` before the report was written. Without the `git log -1` check
this wave would have written its SYSTEM-MAP section against superseded text.

---

## SECTION C — what actually justifies this wave (all measured)

### C1 · THE HEADING LIES, AND THE MODEL IS TOLD TO TRUST IT **[A]**

`kernel/planner_prompt.go:386-397` (pre-wave):

```go
render := func(title string, bars []market.Kline, n int) {
    if len(bars) > n { bars = bars[len(bars)-n:] }
    fmt.Fprintf(&b, "### %s\n", title)
    FormatCandleTable(&b, KlineBars(bars), true)
}
render("15m (last 12)", AggregateBars(bars1m, 15*60*1000), 12)
render("1h (last 12)",  AggregateBars(bars1m, 60*60*1000), 12)
render("4h (last 8)",   AggregateBars(bars1m, 240*60*1000), 8)
render("daily session candles (last 8)", DailySessionBars(bars1m), 8)
```

Truncate-only-when-long — the bridge's exact semantics one layer up — with the
count baked into the title as a literal.

**Measured over the stored prompts** (`planner_rejected_prompts`, n=54 carrying
a Candles block, ids 70–142), reproduced first-hand:

```
n prompts with a Candles block: 54
  '15m (last 12)':                    12 rows in 54
  '1h (last 12)':                     12 rows in 54
  '4h (last 8)':                       8 rows in 54
  'daily session candles (last 8)':    2 rows in 19 · 3 rows in 35
      2 rows -> ids [70, 71, 72, 73, 74, 95]... (n=19)
      3 rows -> ids [75, 77, 78, 80, 81, 83]... (n=35)
```

**Never 8, in 0 of 54.**

**RE-MEASURED at repair time (2026-09-09 18:3x CT), n has grown by one prompt**
— independently re-derived with a fresh census script, not copied forward:

```
n = 55 ids 70 .. 143
15m                     {12: 55}
1h                      {12: 55}
4h                      {8: 55}
daily session candles   {2: 20, 3: 35}
```

**Never 8, in 0 of 55.** **[A]**

Worse, the OLDEST row of each daily table is a **partial session candle**
presented as a whole one. `DailySessionBars` (`kernel/weekly_bias.go:240-266`)
buckets by `CMESessionDayKey` and takes the first bar it SEES as the session
Open, with no completeness check. Live instance, prompt id 142 **[A]**:

```
### daily session candles (last 8)
Time(CT)       Open      High      Low       Close     Volume
09-07 02:39    29641.5000 29664.5000 29534.0000 29604.2500 277429.00
09-07 17:00    29610.2500 29764.7500 29424.5000 29525.5000 2222809.00
09-08 17:13    29525.0000 29625.5000 29489.7500 29590.7500 235420.00     <- current
```

The first row is stamped `09-07 02:39` for a session that opened at
**09-06 17:00 CT**; the last is stamped `09-08 17:13` for a session that opened
at 17:00. Both are indistinguishable from a true session candle.

And `kernel/planner_prompt.go:428` instructs the model: *"Ground truth for
structure; ranked levels and tags are summaries. On conflict, trust the candles
and say so in the scenario rationale."* The prompt disclosed its tape depth
**nowhere**.

### C2 · NAMES THAT PROMISE MORE HISTORY THAN EXISTS **[A]**

- `RVBaselineFrom5m(min5Long, 20, 5)` (`kernel/regime_baseline.go`) filled the
  struct field **`RVBaseline20d`** from about 7 complete session-days. It is
  honest internally — it drops incomplete days and returns `(0,false)` below
  `minDays=5` — but nothing carried the real count to the reader, and the
  regime line rendered `RV=…%-of-normal`: a baseline with no stated window.
- `kernel/weekly_prompt.go:46` `CompletedWeekCandles(bars1m, now, 12)` — twelve
  weeks from a ring that holds at most 41.7 h.
- `kernel/weekly_prompt.go:48` `LastNWOGs(bars1m, now, 5)` — five weekend gaps
  from the same tape.

### C3 · THREE CALL SITES WERE UNREACHABLE BY CONSTRUCTION **[A]**

Ring ceiling `DefaultBarCacheMaxBars = 2500` (`provider/ninjatrader/bar_cache.go:24`):

| site | ask | implied window | ring ceiling |
|---|---|---|---|
| `trader/auto_trader_planner.go` | 1m × 12000 | 200.0 h | 41.7 h |
| `trader/auto_trader_weekly.go` | 1m × 12000 | 200.0 h | 41.7 h |
| `trader/auto_trader_planner.go` | 5m × 3000 | 250.0 h | 208.3 h |

**No market condition can satisfy any of them.** These are not "short reads on a
quiet day"; they are asks the cache is structurally incapable of serving.

### C4 · THE STORE HELD THE DEPTH AND WAS WIRED TO NOTHING **[A]**

Measured 2026-09-09:

```
symbol  tf   n      oldest_ct            newest_ct
MNQ     1m   20043  2026-08-19 10:00:00  2026-09-09 14:35:00     (21 days)
MNQ     5m    3164  2026-08-24 11:15:00  2026-09-09 14:30:00     (16 days)
```

**RE-MEASURED at repair time (18:4x CT)** — the store keeps growing, the shape
does not **[A]**:

```
MNQ  1m  20929 rows  2026-08-19 15:00 → 2026-09-09 23:45 UTC   (21 days)
MNQ  5m   3342 rows  2026-08-24 16:15 → 2026-09-09 23:40 UTC   (~16 days)
```

`.schema bars` — **PRIMARY KEY (symbol, tf, open_time_ms)**. There is no
contract column, which is a real limitation D2/D3 inherit and extend; see
A15/§KNOWN LIMITATIONS.

Retention per TF (`store/bar_history.go`): 1m 90d · 3m/5m 180d · 15m/30m 365d ·
1h and coarser keep-forever. **Nothing has ever been pruned.**
`market.FuturesBarsProvider` has ONE production assignment
(`trader/ninjatrader/bars_market_bridge.go`) and it reads
`server.BarCache().Get(...)` only. `SeedHistorical` MERGES within a process, so
the ring climbs 2000 → 2500 across a session — and every Go restart drops it
back to the 2000-bar seed (`defaultAutoBarsBack`).

### C5 · CORRECTED, PLAINLY

`scope_bars` already records SERVED. The void scope has never been short:
**67/67 at 2000** when the dispatch measured it, **68/68 at 2000** when this
repair re-measured it. The bridge's silence is real but has cost nothing at the ~29
sites that ask 2000. The defect was never the COUNT — it was **CONTINUITY and
SPAN**, and no field could express either.

The live proof, verified three independent ways **[A]**:

- **The hole.** A 14:49 CT snapshot of `bars` shows one missing run for
  session-day 09-08: `09-09 01:29 → 09-09 13:04 CT`, **696 minutes**
  (613 of 1,309 minutes held).
- **The span.** The newest 2000 MNQ 1m bars at or before 13:18:13 CT span
  `09-07 10:23 → 09-09 13:18` = **3,055 minutes** (3,056 intervals), oldest bar
  **50.92 h** old.
- **The gap count, from the production function.**
  `kernel.OpenIntervalsBetween(09-07 10:23, 09-09 13:19, 60000, 200000)` returns
  **2,696** open 1m intervals. 2,696 − 2,000 served = **696 gaps** — to the
  minute, matching the observed hole.

So a served-count check would have been **silent** on the read that mattered.
`planner_read_facts` id 66 recorded `scope_bars=2000` and was **right**.

> **Dated, because it will not reproduce tomorrow:** between 14:49 and ~15:10 CT
> that hole was BACKFILLED. The live table now holds **1,328 contiguous minutes**
> for the same session with `missing=0`, so an identical read today records
> `gaps=0`. The snapshot at `scratchpad/live_copy.db` is the artefact the 696
> figure rests on. This is exactly why the gap count belongs ON THE ROW: the
> tape heals, the record does not.

---

## WHAT WAS BUILT

Four commits, each revertible on its own **except** that D3 uses
`store.BarHistoryStore.LastNBars` introduced by D2, so the revert order is
**D4 · D3 · D2 · D1**.

### D1 — every history-bearing table declares HELD-vs-CLAIMED, and marks a partial as partial

`kernel/candle_disclosure.go` (new) · `kernel/planner_prompt.go` ·
`kernel/engine_prompt.go` · guide · SYSTEM-MAP.

- `BuildPlannerCandleTables(bars1m)` → **`BuildPlannerCandleTablesAt(bars1m, asked1m, now)`** (A28: the clock comes from the caller).
- **Heading:** `### <label> — HELD h of a requested rows · <clauses>`. When
  short: `SHORT BY n — the 1m tape does not reach back far enough; the n older
  rows are ABSENT from this prompt, not flat, and MUST NOT be inferred`. When
  nothing is marked: `every row PRINTED here is COMPLETE` (it said
  `all held rows COMPLETE` until the F-4 fix — a claim about rows that were
  never printed). The complete case is stated
  deliberately — a disclosure that only appears on failure is one the reader
  learns to skim past.
- **TAPE line**, above all four tables: bars HELD of requested, oldest bar +
  age (minute resolution, rounded DOWN so the tape is never claimed older than
  it is), newest bar, and open 1m intervals missing INSIDE the span.
- **Row markers**, measured against the SAME `kernel/session_calendar.json` the
  trading gate reads (shortened and closed days included):
  - `⚠PARTIAL` — front-truncated rows name the first bar HELD *and* the window
    open; **interior holes** say the minutes are missing INSIDE the window.
    These are different facts and the first live render proved it (a row read
    *"its Open is the first bar HELD (01:15 CT), not the window open (01:15 CT)"*
    — the same clock twice, a false reason; fixed and pinned as D1-G).
  - `⏳FORMING` — the window has not closed.
  - `❓COVERAGE-UNKNOWN — <year> is not covered by kernel/session_calendar.json`
    — never a guess, never a plausible number.
- **THE MINUTE IN PROGRESS IS COUNTED ON NEITHER SIDE** (fixed after review —
  see §REVIEW ROUND F-3). `observableEndMs` now FLOORS to the last CLOSED
  minute (`ms - mod(ms, rowCoverageStepMs)`); the forming minute is excluded
  from the denominator there and from the held count in `aggregateCoverage` /
  `sessionCoverage`, so the answer does not depend on whether the forming bar
  has been delivered yet, and a gapless tape is never marked ⚠PARTIAL.
- **WHOLE ROWS THAT ARE NOT THERE ARE COUNTED AND NAMED** (fixed after review —
  see §REVIEW ROUND F-4). `AggregateBars` emits no row for an empty bucket, so a
  whole absent window was invisible to both the row measure and the row count.
  `absentAggregateRows` / `absentSessionRows` walk the calendar BETWEEN rendered
  rows; the heading gains `N WHOLE ROWS ABSENT BETWEEN HELD ROWS` and the row
  after each break carries
  `⛔N×15m ABSENT BEFORE THIS ROW — the tape holds no bars between HH:MM CT and
  HH:MM CT, so the row ABOVE is NOT the adjacent window`. A bucket with **zero
  open 1m intervals** (weekend, holiday, the 16:00–17:00 CT halt) is correctly
  NOT counted — nothing was expected there. A walk that exceeds
  `absentRowScanCap` reports **UNKNOWN**, never a truncated number (A24).
  The complete-table clause changed from `all held rows COMPLETE` to
  **`every row PRINTED here is COMPLETE`**, because the old wording was a claim
  about rows that were never printed.
- **A CLOSED INSTANT IS NEVER NAMED AS "THE WINDOW OPEN"** (`firstOpenGridMs`).
  A 4h bucket floors to 15:00 CT on a Sunday, two hours before CME reopens, so a
  front-truncated row used to print *"its Open is the first bar HELD (17:30 CT),
  not the window open (15:00 CT)"* — counts right, reason false. Same class as
  the interior-hole false reason D1-G already fixed, one bucket over.
- **TAIL-TRIM FIRST, THEN MEASURE.** Coverage used to be computed for every row
  of the full aggregate — ~1,170 calendar walks per planner read on the
  store-deepened 12,000-bar tape, ~99% of them for rows that were then
  discarded. Only the rows that will be PRINTED are measured now.
- **A24 absolute:** no bar is interpolated, carried forward or synthesised, and
  no rendered O/H/L/C/V is changed. **A10:** discloses only; gates nothing.
- `FormatCandleTable` keeps its single-formatter role —
  `FormatCandleTableNoted` is the same function with a per-row note, and
  `FormatCandleTable` now calls it.
- The pre-existing `TestPlannerCandleTablesRenderAndTokenBudget` **asserted the
  lie** (`"### 15m (last 12)"`) and passed for the life of the defect, because
  it checked the literal in the title rather than the rows underneath. Updated
  deliberately, with that noted in the test.

**Not fixed, same class, reported:** `kernel/weekly_prompt.go` renders
`## Weekly candles (12 completed weeks, oldest → latest)` over as few as one
row. It is the identical defect. It was left alone because `SectionsText` feeds
`WeeklyFactsHash`, so changing the heading changes a stored hash — a separate
decision, not a silent side effect of this wave.

### D2 — callers that ask above the ceiling either read the store, or are corrected and say which

`store/bar_history.go` (`LastNBars`) · `trader/bars_store_depth.go` (new) ·
`trader/auto_trader_planner.go` · `trader/auto_trader_weekly.go` ·
`kernel/regime.go` · `kernel/regime_baseline.go` · guide · SYSTEM-MAP.

| site | choice | reason, recorded at the call site |
|---|---|---|
| `auto_trader_planner.go` 1m × 12000 | **(a) read the store** | the tape feeds the "8 daily session candles" table, which needs ~11,040 open 1m intervals; the ring gives 41.7 h, the store 21 days |
| `auto_trader_weekly.go` 1m × 12000 | **(a) read the store** | `ComputeWeeklyFacts` asks for 12 COMPLETED WEEKS + 5 NWOGs from a ring holding under two days. The store reaches 21 days — still short of 12 weeks, which `ThinHistory` already stamps honestly. **IT DOES CHANGE WHAT THIS COMPUTES — see §REVIEW ROUND F-2.** |
| `auto_trader_planner.go` 5m × 3000 | **(b) correct the ask, stop promising 20d** | two measured reasons, below |

**Why the 5m site refused (a).** ① The store's 5m reaches ~16 days (3,164 MNQ
rows back to 2026-08-24), so reading it could not honour "20d" either — it
would swap one unmet promise for another while changing a live regime input.
② The stored 5m rows are NT8 aggregates this repo has **already judged
inconsistent with their own 1m constituents**: `store/bar_history.go` `Migrate`
step 4 deletes every `tf != '1m'` row for exactly that reason, and the per-TF
persistence restored on 2026-09-02 did not re-establish their agreement.
Feeding them into a LIVE regime input is not a depth improvement; it is an
unverified substitution.

So: ask `3000 → 2500`; `RVBaseline20d` → `RVBaseline` + **`RVBaselineDays`**
carrying the count it was ACTUALLY fed; the regime line stops saying
`RV=103%-of-normal` and now says `RV=…%-of-baseline(N complete session-days)`,
or `(window UNKNOWN)` when the count was not reported.

> **CORRECTED AFTER REVIEW AND AN OWNER RULING.** The sentence that stood here —
> *"the computed value is unchanged"* — was **FALSE**, and it stayed false in
> three shipped artifacts (this report, `SYSTEM-MAP.md` and a guide card).
> See **§REVIEW ROUND F-1** for the ruling, the four conditions, and the measured
> numbers. In short: the RULE is byte-for-byte unchanged and pinned so by an E7
> golden; the INPUT is deliberately corrected, and the report now names every
> value that moves with its measured delta.

`barsWithStoreDepthFrom`'s four rules, each pinned:

1. **An EMPTY ring is NEVER substituted.** An empty ring means the feed is down
   or the seed has not landed; handing a planner 12,000 stored bars would let it
   write a plan on a dead tape — the exact failure "no NT8 → no decisions"
   exists to prevent. The store DEEPENS a live tape; it never stands in for one.
2. A ring that already serves the ask is not second-guessed — **no store read at
   all**, so the ~29 healthy call sites pay nothing.
3. Only bars **strictly older** than the ring's oldest are taken, so a live or
   forming bar is never replaced by a stored one.
4. A failed store read WARNs and degrades to the ring (A10).

### D3 — the ring rehydrates from the store on boot (owner-authorised, 1m ONLY)

`provider/ninjatrader/bar_cache.go` (`RehydrateOlder`) ·
`trader/ninjatrader/bar_persist_wire.go` (`rehydrateRingFromStore`,
`pairsToRehydrate`) · `trader/regime_input_window.go` (new) ·
`trader/auto_trader_planner.go` · `trader/auto_trader_dayplan.go`.

> **OWNER RULING, 2026-09-09 18:18 CT.** *"RULING on D3: the regime input MAY
> change. Rehydrating the ring from the store changes what RVBaseline is fed —
> and what it is fed today is 41 hours labelled as 20 days. A regime value
> computed on a shorter window than its name is the defect; correcting the
> window is not a scope violation, it is the fix. A31 forbids changing the RULE,
> not correcting the INPUT the rule was promised."*

**Four conditions came with it. All four are implemented, none parked:**

| # | condition | where | pin |
|---|---|---|---|
| (a) | rehydrate **1m only** | `pairsToRehydrate` (`bar_persist_wire.go`) — the loop iterates the selection, not the raw pairs | **D3-L** `TestRehydrateSelectsOnly1mPairs` |
| (b) | the 5m ask **served from the rehydrated 1m tail**, renamed to what it receives | `ResolveRVBaselineTape` (`trader/regime_input_window.go`), called at `auto_trader_planner.go`; `rvBaseline5mBarsAsk` → **`rvBaselineFallback5mBarsAsk = nt.DefaultBarCacheMaxBars`** (READ from the ceiling, not copied) | **D3-G** `TestRVBaselineIsServedFromThe1mTail`, **D3-H** fallback never blanks |
| (c) | the **boot line reports the served window in days, BEFORE and AFTER** | `RegimeInputWindowBootLine`, emitted from the `afterBackfillHook` (which now fires AFTER the rehydrate) | **D3-J** `TestRegimeInputWindowBootLineReportsBeforeAndAfterInDays` |
| (d) | an **E7-style golden on the regime label** for a fixture whose window does NOT change | `TestRegimeLabelUnchangedWhenTheWindowDoesNotChange` | **D3-F**, proven to fail under 3 mutations |

**The boot line (condition c), every field resolved (A11):**

```
📈 regime input window @08:30:00 CT: BEFORE window=6 complete session-days · baseline=0.096577 · 1656 5m rows via 5m-ring (pre-wave) · AFTER window=9 complete session-days · baseline=0.096572 · 2484 5m rows via 1m-tail-agg5m · Δ+3 day(s) · cap=20 days (rule UNCHANGED — same estimator, deeper input; owner ruling 2026-09-09)
```

(the numbers above are from the D3-J fixture; on the running bot every one of
them is READ. An unmeasurable half prints `window=UNKNOWN` and the delta prints
`ΔUNKNOWN`, never `Δ+0`.)

**Why 1m only, MEASURED** (`kernel.RVBaselineFrom5mDays(·, 20, 5)`, MNQ, live
store, read-only) **[A]**:

```
stored 5m, newest 2000 rows  → rv 0.876371 over 7 complete session-days
stored 5m, newest 2500 rows  → rv 0.884746 over 9 complete session-days
1m tape 12000 → agg 5m 2400  → rv 0.893543 over 9 complete session-days
```

The last two cover **the same nine session-days** and disagree by **0.9%**: NT8's
stored 5m rows do not agree with their own 1m constituents. `store/bar_history.go`
`Migrate` step 4 once deleted every non-1m row for exactly that reason. So the
builder's refusal of the stored 5m **stands**, and the deeper window comes from
the 1m rows — the feed's own closed bars, the tape every other series is
aggregated FROM.

**A10 — the baseline is never turned OFF.** The 1m ring ALONE cannot support a
baseline (2,000–2,500 1m bars → 400–500 aggregated 5m rows → `ok=false`
[A, measured]), so the depth genuinely depends on the D2 store splice. If that
splice ever degrades, `ResolveRVBaselineTape` falls back to the pre-wave 5m ring
read and the log names which arm answered. Degrade, never gate.

Fired once the AddOn replay lands — **alongside** the AddOn's seed. It is
deliberately **not** `SeedHistorical`, on two counts:

- **A cold key is a NO-OP.** `SeedHistorical` seeds an empty key; this refuses
  to. This is the D3 safety argument in full: before the change, a restart with
  NT8 down left the cache empty and the bot idle. Filling a cold cache from the
  store would have made a dead feed look alive to every reader downstream. By
  restricting rehydration to keys that ALREADY hold live bars, the change can
  only ever restore depth the ring just lost — it can never manufacture a tape.
- **`existing` wins the overlap.** `SeedHistorical` lets `incoming` win because
  incoming is the freshest wire OHLCV; here `incoming` is the STORE, which is by
  definition not fresher than the live ring.

Merges via the same `mergeBarsByTime` discipline, bounded by the ring's own
`maxBars` (trimming the OLDEST, never the live tail), refuses NT8 empty-minute
placeholder bars at this door as at every other, WARNs-and-continues on a store
failure (boot is never blocked), and logs per `(symbol, tf)` with resolved
counts (A9/A11). It touches neither the AddOn, the subscription, nor the
backfill.

### D4 — the served-scope observability, kept AS BUILT, reconciled with R1

`4b2edea3` is kept unchanged. **Nothing in it assumed `scope_bars` was the
request** — its warn arms are SHORT (served < requested), HOLED (open intervals
missing inside the span) and EMPTY, and its own commit message already said the
count was never the defect. So the reconciliation with R1 is a correction of the
*record*, not of the code:

- `store/planner_read_facts.go` — four additive columns
  (`scope_requested_bars`, `scope_span_ms`, `scope_oldest_age_ms`,
  `scope_gap_count`) plus `read_horizons` (JSON `kernel.BarHorizon[]`), with a
  comment stating R1 as NOT REPRODUCED and why the column is not redefined.
- `kernel/void_scope.go` — `VoidScope.Horizon`, computed with the SAME `now` the
  caller passed (A28), so the recorded age is exact.
- `trader/auto_trader_planner.go` — `buildReadFactRow` extracted from
  `persistReadFacts` (class 86) so the pins drive the PRODUCTION builder rather
  than a copy, and pure (`now` comes in).
- **UNKNOWN is not ZERO.** `read_horizons == ""` marks a row whose horizon was
  never computed; its four numerics are UNKNOWN. The 67 pre-wave rows read that
  way. Query with `where read_horizons != ''` and state the excluded COUNT
  (corrected-column law, class 40).

---

## D1 SAMPLE RENDER — live data, FINAL CODE

Rendered from the live `bars` table (12,000 MNQ 1m bars) at **2026-09-09
18:48:20 CDT**, through `kernel.BuildPlannerCandleTablesAt` exactly as the
planner calls it. This is a **HEALTHY** tape, which is the point: after the two
blocker fixes it carries **no ⚠PARTIAL at all**, and the newest row of every
table says ⏳FORMING **only**.

```
TAPE: 12000 1m bars HELD of 12000 requested · oldest 08-27 21:48 CT (309h0m0s ago) · newest 09-09 18:47 CT · gaps 0 open 1m intervals missing INSIDE that span. Every table below is aggregated from THIS tape and can be no deeper than it.
ROW MARKERS: ⚠PARTIAL = only part of the row's window is held, so its O/H/L/C come from the bars HELD and not from the whole window · ⏳FORMING = the window has not closed yet (the minute in progress is counted on neither side) · ⛔N×tf ABSENT BEFORE THIS ROW = that many WHOLE open-market windows between this row and the one above are missing from the tape, so the two rows are NOT adjacent windows · ❓COVERAGE-UNKNOWN = the session calendar cannot classify this window. A missing bar is NEVER interpolated, carried forward or synthesised — it is simply absent, and the row says so.

### 15m — HELD 12 of 12 requested rows · 1 of 12 printed rows are MARKED on the row itself (⚠PARTIAL / ⏳FORMING / ❓COVERAGE-UNKNOWN)
Time(CT)       Open      High      Low       Close     Volume
09-09 15:00    29455.0000 29459.5000 29433.7500 29451.2500 15730.00
09-09 15:15    29451.5000 29454.5000 29448.0000 29450.0000 4012.00
09-09 15:30    29450.0000 29459.5000 29445.2500 29454.2500 4104.00
09-09 15:45    29454.5000 29460.5000 29448.7500 29460.2500 3742.00
09-09 17:00    29460.2500 29474.5000 29425.7500 29428.0000 12479.00
09-09 17:15    29428.0000 29435.0000 29422.7500 29428.7500 4203.00
09-09 17:30    29429.7500 29437.5000 29427.5000 29433.0000 3349.00
09-09 17:45    29433.2500 29437.7500 29425.7500 29431.5000 2843.00
09-09 18:00    29431.2500 29432.7500 29417.7500 29421.7500 4240.00
09-09 18:15    29421.2500 29435.2500 29414.5000 29428.2500 6200.00
09-09 18:30    29428.2500 29432.2500 29424.7500 29429.7500 2764.00
09-09 18:45    29430.5000 29437.0000 29426.5000 29434.0000 2054.00       <- current  ⏳FORMING — the window has not closed; holds 3 of 3 open 1m intervals that have CLOSED so far (the minute in progress is counted on neither side)

### 1h — HELD 12 of 12 requested rows · 1 of 12 printed rows are MARKED on the row itself (⚠PARTIAL / ⏳FORMING / ❓COVERAGE-UNKNOWN)
Time(CT)       Open      High      Low       Close     Volume
09-09 06:00    29391.7500 29451.7500 29375.0000 29396.0000 69541.00
09-09 07:00    29396.5000 29403.7500 29332.5000 29389.7500 69254.00
09-09 08:00    29390.2500 29567.0000 29382.2500 29558.7500 307250.00
09-09 09:00    29558.2500 29594.0000 29459.5000 29528.2500 366432.00
09-09 10:00    29528.7500 29563.2500 29358.2500 29447.5000 373105.00
09-09 11:00    29447.7500 29478.0000 29386.5000 29457.7500 146199.00
09-09 12:00    29457.5000 29483.7500 29428.5000 29468.2500 118615.00
09-09 13:00    29468.5000 29474.7500 29436.5000 29447.7500 87940.00
09-09 14:00    29447.5000 29471.2500 29422.7500 29456.0000 108452.00
09-09 15:00    29455.0000 29460.5000 29433.7500 29460.2500 27588.00
09-09 17:00    29460.2500 29474.5000 29422.7500 29431.5000 22874.00
09-09 18:00    29431.2500 29437.0000 29414.5000 29434.0000 15258.00      <- current  ⏳FORMING — the window has not closed; holds 48 of 48 open 1m intervals that have CLOSED so far (the minute in progress is counted on neither side)

### 4h — HELD 8 of 8 requested rows · 1 of 8 printed rows are MARKED on the row itself (⚠PARTIAL / ⏳FORMING / ❓COVERAGE-UNKNOWN)
Time(CT)       Open      High      Low       Close     Volume
09-08 11:00    29604.5000 29641.5000 29495.0000 29542.5000 579461.00
09-08 15:00    29542.5000 29550.7500 29489.7500 29500.2500 94008.00
09-08 19:00    29500.5000 29625.5000 29499.0000 29562.2500 200379.00
09-08 23:00    29561.7500 29634.0000 29549.2500 29589.0000 154462.00
09-09 03:00    29589.0000 29609.2500 29365.0000 29396.0000 230101.00
09-09 07:00    29396.5000 29594.0000 29332.5000 29447.5000 1116041.00
09-09 11:00    29447.7500 29483.7500 29386.5000 29456.0000 461206.00
09-09 15:00    29455.0000 29474.5000 29414.5000 29434.0000 65720.00      <- current  ⏳FORMING — the window has not closed; holds 168 of 168 open 1m intervals that have CLOSED so far (the minute in progress is counted on neither side)

### daily session candles — HELD 8 of 8 requested rows · 1 of 8 printed rows are MARKED on the row itself (⚠PARTIAL / ⏳FORMING / ❓COVERAGE-UNKNOWN)
Time(CT)       Open      High      Low       Close     Volume
08-31 17:00    29518.5000 29571.0000 29001.7500 29139.0000 2550843.00
09-01 17:00    29137.2500 29212.5000 28927.2500 29143.0000 2177636.00
09-02 17:00    29175.7500 29585.0000 29075.0000 29502.7500 2335268.00
09-03 17:00    29500.0000 29720.0000 29468.2500 29524.7500 1961640.00
09-06 17:00    29534.5000 29683.7500 29530.2500 29604.2500 552400.00
09-07 17:00    29610.2500 29764.7500 29424.5000 29525.5000 2222809.00
09-08 17:00    29528.0000 29634.0000 29332.5000 29460.2500 2244916.00
09-09 17:00    29460.2500 29474.5000 29414.5000 29434.0000 38132.00       <- current  ⏳FORMING — the window has not closed; holds 108 of 108 open 1m intervals that have CLOSED so far (the minute in progress is counted on neither side)
```

**Four things to read off it.**

1. **The daily table renders 8 of 8.** The D2 store read is what makes that
   possible; from the ring alone this table rendered 2 or 3 rows in **55 of 55**
   stored prompts and never 8.
2. **`gaps 0` and no ⚠PARTIAL anywhere.** Before the fix this exact tape put
   `⏳FORMING ⚠PARTIAL — holds only 1309 of the 1310` on the daily row while the
   TAPE line six lines above said `gaps 0` — a self-contradiction on 100% of
   live reads (F-3).
3. **`09-09 15:45` is followed by `09-09 17:00` in the 15m table with NO ⛔
   marker.** The four intervening buckets are the 16:00–17:00 CT maintenance
   halt: **zero open 1m intervals**, so nothing was expected and nothing is
   claimed absent. This is the discrimination that makes F-4's marker readable
   rather than noise.
4. **The wave's flagship live example is now HISTORICAL.** The 696-minute
   09-09 01:29→13:04 CT outage that §C5 rests on **has since been backfilled**.
   Re-measured 18:4x CT, the only gap in the whole of 09-09 in the store is
   `20:59 → 22:00 UTC` = **15:59 → 17:00 CT — the daily halt**, i.e. not a hole
   at all. §C5's figures remain correct as of the snapshot they name; they will
   not reproduce today. The tape heals; the record does not. **This is exactly
   why the gap count belongs ON THE ROW.**

---

## PINS — RED → GREEN → MUTATION

Every pin was quoted RED on a real assertion before it was made green (new
functions were stubbed to the PRE-WAVE behaviour first, so no pin's RED is a
mere compile error), then mutated.

### D1 · `kernel/planner_candle_disclosure_test.go`

**RED** (`TestCandleHeadingsStateHeldVsClaimed`), with the fixture reproducing
the live shape exactly:

```
--- FAIL: TestCandleHeadingsStateHeldVsClaimed (0.00s)
    planner_candle_disclosure_test.go:91: heading missing "### 15m — HELD 12 of 12 requested rows"
        --- rendered ---
        ### 15m (last 12)
        ...
        ### daily session candles (last 8)
        Time(CT)       Open      High      Low       Close     Volume
        09-08 02:39    29500.0000 29511.7500 29498.0000 29500.5000 9612.00
        09-08 17:00    29500.2500 29511.7500 29498.0000 29505.2500 14628.00      <- current
```

**GREEN:** `ok  nofx/kernel  0.028s`

**MUTATIONS** (each: the exact line changed, then the failure text):

| # | line changed | result |
|---|---|---|
| 1 | `RowCoverage.Partial()` body → `return false` | **FAIL** `the front-truncated session ROW is NOT marked partial` · `the partial row does not disclose that its Open is the first bar HELD` |
| 2 | `tableHeading`: `label, held, asked,` → `label, asked, asked,` | **FAIL** `heading missing "### daily session candles — HELD 2 of 8 requested rows"` |
| 3 | `if st := SessionStateAt(...); st.UncoveredFallback {` → `; false && st.UncoveredFallback {` | **FAIL** `must render "❓COVERAGE-UNKNOWN — 2031 is not covered by kernel/session_calendar.json" on the ROW` · `an UNKNOWN row also printed a computed interval count` |
| 4 | `RowCoverage.Marked()` body → `return false` | **FAIL** `### 15m heading claims "all held rows COMPLETE" over 1 MARKED row(s)` · `heading does not count its 1 marked rows` |
| 5 | `case c.Partial() && c.FirstHeldMs > c.WindowOpenMs:` → `case c.Partial() && false:` | **FAIL** `the front-truncated session ROW is NOT marked partial` |
| 6 | `age = msDurTxt(h.OldestAgeMs / 60_000 * 60_000)` → `msDurTxt(h.OldestAgeMs)` | **FAIL** `tape age carries sub-second noise: "… (34h39m13.387s ago) …"` |

**Two pins existed only because mutation found them missing** — both would have
shipped otherwise:

- **Mutation 3 passed on the first version** of the pin. Searching the SECTION
  for `"COVERAGE-UNKNOWN"` matched the **heading's own legend clause**
  (`(⚠PARTIAL / ⏳FORMING / ❓COVERAGE-UNKNOWN)`), so the pin could not fail. A
  `d1Rows()` extractor was added that returns only timestamped candle rows, and
  the same weakness was fixed in the PARTIAL/FORMING assertions of D1-B and
  D1-D, which had it too. **Class 89, exactly.**
- **Mutation 4 passed** with every other pin green: forcing `Marked()` false let
  the heading claim `all held rows COMPLETE` over a row carrying `⚠PARTIAL`. A
  heading contradicting its own rows is the defect this wave exists to remove,
  one layer up. **PIN D1-F** now asserts the heading and the rows agree in both
  directions.
- **Mutation 6 passed** on the first version because the fixture clock had no
  sub-second part, so the pin could not fail; it now states its own clock with
  `387_000_000` ns.

### D2 · `trader/bars_store_depth_test.go`

**RED:**
```
--- FAIL: TestStoreDepthPrependsOlderAndNeverReplacesRing (0.00s)
    bars_store_depth_test.go:57: served 100 bars, want 400 (300 from the store + 100 from the ring)
--- FAIL: TestRVBaselineCarriesItsActualDayCount (0.00s)
    bars_store_depth_test.go:146: complete-day count = 0, want 1..20 — an uncomputed count is UNKNOWN, never a plausible number
--- FAIL: TestD2WiredAtTheThreeCallSites (0.09s)
    bars_store_depth_test.go:177: barsWithStoreDepth(: 0 production call sites (A29)
    bars_store_depth_test.go:188: an ask above the ring ceiling survives: "1m", 12000 in [trader/auto_trader_weekly.go]
    bars_store_depth_test.go:188: an ask above the ring ceiling survives: "5m", 3000 in [trader/auto_trader_planner.go]
```

**GREEN:** `ok  nofx/trader  0.117s`

| # | line changed | result |
|---|---|---|
| 7 | `if k.OpenTime < oldestRing {` → `<= oldestRing` | **FAIL** `result is not strictly ascending at 300 (1788971880000 <= 1788971880000)` |
| 8 | drop `len(ring) == 0 \|\|` from the early return | **FAIL** `TestEmptyRingIsNeverSubstitutedByTheStore` |
| 9 | `return …, len(perDay), true` → `…, maxDays, true` | **FAIL** `complete-day count = 20, want exactly 7` |
| 10 | `} else if r.RVBaselineDays > 0 {` → `} else if false && …` | **FAIL** `the regime line does not name the baseline's real window (7 days): REGIME: … · RV=103%-of-baseline(window UNKNOWN) · VIX=n/a` |

**Mutation 9 passed on the first version** of the pin, which asserted the day
count in a `1..20` RANGE: returning `maxDays` — the ASK — sat inside it. The
wave's entire point is that the fed count and the asked count DIFFER, so a pin
tolerating the ask cannot see the defect. It now asserts **exactly 7**, and
separately that the reported count is not the ask. **Class 89.**

### D3 · `provider/ninjatrader/bar_rehydrate_test.go`

**RED:**
```
--- FAIL: TestRehydrateOlderExtendsBackwardsOnly (0.00s)
    bar_rehydrate_test.go:36: rehydrated 0 bars, want 300 (only those strictly older than the ring's oldest)
--- FAIL: TestRehydrateOlderIsCappedAtMaxBars (0.00s)
    bar_rehydrate_test.go:81: ring holds 100 bars, want the 500 cap
--- FAIL: TestRehydrateOlderRefusesPlaceholderBars (0.00s)
    bar_rehydrate_test.go:102: rehydrated 0 bars, want 4 — the placeholder must be dropped, not stored
```

**GREEN:** `ok  nofx/provider/ninjatrader  0.006s`

| # | line changed | result |
|---|---|---|
| 11 | `if len(existing) == 0 { return 0 }` → `if false && …` | **FAIL** `TestRehydrateOlderIsANoOpOnAColdKey` |
| 12b | `mergeBarsByTime(older, existing)` → `mergeBarsByTime(existing, older)` | **PASS — EQUIVALENT MUTANT** |
| 12c | remove the `b.T < oldest` filter | **PASS — EQUIVALENT MUTANT** |
| 12d | **both** 12b and 12c together | **FAIL** `live bar 0 was replaced by a stored one: {T:940000 … C:-1} want {… C:7}` |
| 13 | `merged = merged[len(merged)-c.maxBars:]` → `merged[:c.maxBars]` | **FAIL** `the cap trimmed a LIVE bar at 0` |
| 14 | remove `dropPlaceholderBars` from `RehydrateOlder` | **FAIL** `rehydrated 5 bars, want 4 — the placeholder must be dropped, not stored` |

**12b and 12c are genuine equivalent mutants and are reported as such, not as
green.** The live-bar guard is DOUBLE — an older-only filter AND `existing`
passed as `mergeBarsByTime`'s `incoming` (which wins every tie). Removing
exactly one leaves the other protecting the live bar, so no test can catch it;
removing both is caught (12d). This is recorded in the test file itself so a
later reader does not "fix" a correct test.

A fixture bug was also caught here: the first version compared against the raw
input slice, but `SeedHistorical` applies the wire's close-stamp → open-stamp
conversion, so the ring's bar times are not the fixture's. The pin now reads the
tape back — which additionally proves store rows (already open-stamped by
`bar_persist_wire`) and ring rows agree.

### D4 · `trader/read_facts_horizon_test.go` · `store/planner_read_facts_test.go`

**RED:**
```
--- FAIL: TestReadFactRowCarriesTheHorizon (0.00s)
    read_facts_horizon_test.go:62: scope_requested_bars=0, want 2000 — the REQUEST is a new column, never a redefinition of scope_bars
--- FAIL: TestAHolyTapeAndAContiguousTapeDifferOnTheRecord (0.00s)
    read_facts_horizon_test.go:107: a contiguous tape and a holed tape are STILL indistinguishable on the record (span 0 gaps 0 both)
--- FAIL: TestReadFactRowBuilderIsWired (0.04s)
    read_facts_horizon_test.go:127: buildReadFactRow(: 0 production call sites (A29)
```

**GREEN:** `ok  nofx/trader  0.050s` · `ok  nofx/store  0.161s`

| # | line changed | result |
|---|---|---|
| 15 | `ScopeBars: len(scope.Bars)` → `ScopeBars: scope.BarCount` (shipping the refuted premise) | **FAIL** `scope_bars=2000, want the SERVED 1200 — it is len(scope.Bars) and always was (R1)` |
| 16 | `row.ScopeGapCount = h.GapCount` → `= 0` | **FAIL** `the holed tape records gaps=0 — a hole must be counted, not implied` |
| 17 | `HorizonRecorded()` → `return true` | **FAIL** `legacy row reports HorizonRecorded()=true, want false — its scope_gap_count=0 is UNKNOWN, not zero` |
| 18 | `if h.Interval == "" {` → `if false && h.Interval == "" {` | **FAIL** `read_horizons="[{"asked":0,…,"gaps":0}]" on a scope that computed no horizon — it must stay "" so the row reads UNKNOWN` |

**Mutations 15 and 18 both passed first time** and both are real gaps that would
have shipped:

- **15**: every live row and the first fixture had `served == requested == 2000`,
  so swapping the two — *shipping the very premise R1 refuted* — was invisible.
  **PIN D4-D** now uses a SHORT fixture (1200 served of 2000 requested) and
  asserts the two columns differ. **Class 89.**
- **18**: no pin covered a scope carrying no horizon. Removing the guard let an
  uncomputed horizon write four zeros AND a fully-zero `read_horizons` entry —
  a row that measured nothing claiming "span 0, age 0, no gaps". **PIN D4-E**
  pins the UNKNOWN. **A24, the plausible zero.**

### ROUND 2 — the pins added after review (RED → GREEN → MUTATION)

**On F-20 (mislabeled mutation quotes).** The mislabeling was in the structured
hand-off's per-pin `mutation_failure` fields, not in the numbered table above —
the table above never claimed a per-pin filing. Both pins the reviewer named ARE
real, and the correct failures are:

- **D1-D** (`TestCompleteSessionRowCarriesNoMarker`) fails under
  `clauses = append(clauses, "every row PRINTED here is COMPLETE")` removed →
  `a complete table does not SAY it is complete`, and under
  `obs := observableEndMs(now) - 120_000` →
  `a complete closed session ROW was marked … ⏳FORMING`.
- **D4-A** (`TestReadFactRowCarriesTheHorizon`) fails under
  `row.ScopeRequestedBars = 0` → `scope_requested_bars=0, want 2000`, and under
  `row.ScopeSpanMs = 0` → `scope_span_ms=0, want 119940000`.

#### RED — the two blockers, reproduced against the committed code

The new pins were run against `53786a44`'s `kernel/candle_disclosure.go` and
`kernel/planner_prompt.go` (the two source files reverted, the new test file
kept). Both REDs are the reviewers' findings, reproduced first-hand:

```
--- FAIL: TestHealthyTapeIsNotMarkedPartial (0.00s)
    planner_candle_disclosure_test.go:333: ### 15m: the NEWEST row of a GAPLESS tape is marked ⚠PARTIAL while the tape line says gaps 0:
        ### 15m — HELD 12 of 12 requested rows · 1 of 12 held rows are MARKED …
        09-09 14:45   …  <- current  ⏳FORMING ⚠PARTIAL — the window is still open AND holds only 4 of the 5 open 1m intervals elapsed so far
    planner_candle_disclosure_test.go:333: ### 1h: … only 49 of the 50 …
    planner_candle_disclosure_test.go:333: ### 4h: … only 229 of the 230 …
    planner_candle_disclosure_test.go:333: ### daily session candles: … only 1309 of the 1310 …
```

```
--- FAIL: TestWholeAbsentRowsAreNamedNotSilentlySkipped (0.00s)
    planner_candle_disclosure_test.go:375: the heading does not count the rows that are NOT there (4×15m, market open):
        ### 15m — HELD 12 of 12 requested rows · all held rows COMPLETE
```

**GREEN after the fixes:** `ok  nofx/kernel  2.126s`

#### MUTATIONS — round 2 (each row: the exact line changed, then the failure)

| # | pin | line changed | result |
|---|---|---|---|
| M1 | D1-K | `if IsCMEOpen(time.UnixMilli(startMs)) {` → `if true \|\| IsCMEOpen(…) {` | **FAIL** `firstOpenGridMs returned 15:00 CT, which is CLOSED` |
| M2 | D1-L | `intTxt(h.GapCount)` → `"0"` | **FAIL** `the TAPE line must state the COMPUTED gap count, by value: … gaps 0 …` |
| M3 | D1-L | `h.Served, h.Requested,` → `h.Requested, h.Requested,` | **FAIL** `the TAPE line must state the SERVED count, by value: TAPE: 12000 1m bars HELD of 12000 requested` |
| M4 | D1-M | `n = -1` → `n = 0` (capped scan) | **FAIL** `a capped between-row walk must report UNKNOWN in the heading` |
| M5 | D1-N | `candleTables = kernel.BuildPlannerCandleTablesAt(…)` → `candleTables = ""` | **FAIL** `kernel.BuildPlannerCandleTablesAt has 0 production call sites in ../trader/auto_trader_planner.go (A29)` |
| M6 | D1-I | `return ms - mod(ms, rowCoverageStepMs)` → `… + rowCoverageStepMs` | **FAIL** `the NEWEST row of a GAPLESS tape is marked ⚠PARTIAL while the tape line says gaps 0` |
| M7 | D1-J | `if gone > 0 {` → `if false && gone > 0 {` | **FAIL** `the heading does not count the rows that are NOT there (4×15m, market open)` |
| M8 | D1-J | `absentRowNote`: `case n > 0:` → `case false:` | **FAIL** `the row FOLLOWING the hole carries no ⛔ marker` |
| M9 | D2-G | `if len(out) > n {` → `if false && len(out) > n {` | **FAIL** `served 400 bars for an ask of 250 — the splice over-serves` |
| M10 | D3-L | `if p[1] == rehydrateTimeframe {` → `if true \|\| …` | **FAIL** `selected 9 pairs from 9, want exactly the 2 that are 1m` |
| M11 | D3-F | `AggregateBars(bars1m, 5*60*1000)` → `3*60*1000` | **FAIL** `GOLDEN MOVED — the RULE changed, not just the input. got: RV=171%… want: RV=103%…` |
| M12 | D3-F | estimator: `sum / float64(len(perDay))` → `… * 1.05` | **FAIL** `GOLDEN MOVED … got: RV=98%… want: RV=103%…` |
| M13 | D3-G | primary read `agg` → `ring5m` (selection order swapped) | **FAIL** `the 1m tail did not widen the window: 1m=6 days, 5m ring=6 days` |
| M14 | D3-H | fallback arm disabled (`; false && ok`) | **FAIL** `a thin 1m tape TURNED THE BASELINE OFF although the ring could answer` |
| M15 | D3-J | `fmt.Sprintf("%+d day(s)", after.Days-before.Days)` → `"+0 day(s)"` | **FAIL** `the boot line does not carry "Δ+3 day(s)"` |

**ONE ROUND-2 MUTATION PASSED, AND IT IS RECORDED RATHER THAN HIDDEN (A8).**

- **M11b** — `const rvBaselineMinBarsPerDay = 200` → `100` in
  `kernel/regime_baseline.go`, run against D3-F: **`ok  nofx/trader  0.014s`**.
  The D3-F fixture builds nine FULL 23-hour session-days, so every day clears
  both thresholds and neither the day count nor the per-day RV moves. This is
  the pin behaving correctly, not a hole: the completeness threshold is a
  property of INCOMPLETE days, and D2-E's own fixture is where it is exercised.
  It is listed because a mutation that passes is not evidence, and pretending it
  did not happen is worse than reporting it.

**Two pin labels collided and were renumbered.** `PIN D1-H` was used twice (tape
age, and the healthy-tape pin added this round). The round-2 pins are
**D1-I … D1-N**; the pre-existing D1-A…D1-H are untouched.

---

## REVIEW ROUND — three reviewers, 22 findings, every one disposed

Three independent reviewers returned findings against `53786a44`. Every one was
**re-measured here** before it was accepted or refuted; two reviewers disagreed
on one figure, so it was measured a third time. Nothing below is taken on a
reviewer's word, and nothing is parked.

### VERIFICATION RECORD

| # | severity as filed | claim (abbreviated) | disposition | evidence |
|---|---|---|---|---|
| F-1 | SCOPE-VIOLATION ×2 | D3 moves the RV baseline (a regime input) while three artifacts say it does not | **ACCEPTED — the claim of no-change was FALSE; the change itself is now OWNER-RULED IN SCOPE.** Implemented with all four conditions; all three false sentences corrected | measured: 5m ring 0.876371/7d → 1m tail 0.893543/9d |
| F-2 | WRONG ×2 | D2 also moves `CompletedWeekCount` / `WeeklyShadowRefs` / `ComputeWeeklyFacts`, undisclosed | **ACCEPTED.** Measured myself (the two reviewers disagreed 1 vs 2). Named at the call site, in the report, in the live-proof list | 2,500 bars (43.6 h) → refs 1, weeks 0, NWOGs 0 · 12,000 (309.0 h) → refs 3, weeks 2, NWOGs 2 |
| F-3 | BLOCKER ×1 | on a gapless tape the newest row of all four tables is ⚠PARTIAL, contradicting `gaps 0` six lines above | **ACCEPTED, FIXED.** `observableEndMs` floors; the forming minute is counted on neither side | RED quoted below (`holds only 1309 of the 1310`) |
| F-4 | BLOCKER ×2 | a whole ABSENT bucket between two rendered rows is measured by nothing; the heading says COMPLETE across a hole | **ACCEPTED, FIXED.** Between-row calendar walk + ⛔ marker + heading clause | RED quoted below (`HELD 12 of 12 requested rows · all held rows COMPLETE` over 4 missing 15m windows) |
| F-5 | WEAK-TEST | the TAPE line's two numbers survive mutation (`gaps` hardcoded 0; served→requested) | **ACCEPTED, PINNED.** D1-L pins both BY VALUE | MUT-2/MUT-3 both fail now; both passed before |
| F-6 | WEAK-TEST | `barsWithStoreDepthFrom`'s tail cap is unpinned; the splice can over-serve to 2n | **ACCEPTED, PINNED.** D2-G | MUT-9 fails: `served 400 bars for an ask of 250` |
| F-7 | UNWIRED | `BuildPlannerCandleTablesAt` can be deleted from the prompt with the suite green | **ACCEPTED, PINNED.** D1-N | MUT-5 fails when the call site is replaced by `candleTables = ""` |
| F-8 | UNWIRED | `telemetry.BarHorizonCounts` has 0 production readers — 5 arms nobody can read | **ACCEPTED, WIRED.** The totals now ride the 🕳 warn line (no API surface is in footprint) | PIN 5d + A29 entries |
| F-9 | UNWIRED | `HorizonRecorded()` has 0 production call sites; the A29 claim over-reached | **ACCEPTED — STATED PLAINLY rather than fabricated a caller.** Same for `RVBaselineFrom5m` (kept as D2-E's parity anchor) | doc comments name both; the A29 sentence in §SCOPE is narrowed |
| F-10 | MINOR | `ReadHorizons` doc promises an `atr5m` entry that is never written | **ACCEPTED, CORRECTED.** Doc now says ONE entry (`who="void"`); atr5m stays OPEN ITEM 2 | `buildReadFactRow` marshals `[]kernel.BarHorizon{h}` |
| F-11 | MINOR | the class-40 "count the excluded" query returns a plausible zero because the legacy rows are NULL, not `''` | **ACCEPTED, CORRECTED and VERIFIED ON A COPY.** Query is now `is null or = ''` | on a COPY of `data.db`: after `ADD COLUMN`, 68/68 rows NULL; `where read_horizons = ''` → **0**, `where … is null or = ''` → **68** |
| F-12 | MINOR | a Sunday 4h bucket floors to 15:00 CT, a CLOSED instant, and the marker names it "the window open" | **ACCEPTED, FIXED.** `firstOpenGridMs` | MUT-1 fails: `firstOpenGridMs returned 15:00 CT, which is CLOSED` |
| F-13 | MINOR | coverage measured for ~1,170 rows per read, ~99% discarded | **ACCEPTED, FIXED.** Tail-trim first, then measure | `planner_prompt.go` `tail()` |
| F-14 | MINOR | the 🕳 SHORT arm is a permanent false alarm at the D2 sites and names an unlisted wrapper | **ACCEPTED, DOCUMENTED — not suppressed.** The wrapper list now has seven entries and says how to pair the 🕳 and 📚 lines | suppressing the ring's line would hide a dead feed on the day the store also fails |
| F-15 | MINOR | `rvBaseline5mBarsAsk = 2500` is a bare copy of `DefaultBarCacheMaxBars` | **ACCEPTED, FIXED.** `= nt.DefaultBarCacheMaxBars` (no import cycle: package `trader` already imports it) | |
| F-16 | MINOR | the guide/SYSTEM-MAP do not say the ring ceiling is still 2,500, so a reader concludes the ring holds 21 days | **ACCEPTED, FIXED** in both surfaces | the rehydrate adds ≤500 bars/pair ≈ 8.3 h at 1m |
| F-17 | MINOR | `📊 bars after backfill` printed pre-rehydrate depths | **ACCEPTED, FIXED.** The rehydrate now runs BEFORE `afterBackfillHook` | `bar_persist_wire.go` ordering |
| F-18 | DOC-DRIFT | running rev / PID / `GUIDE_BUILT_REV` in the report are all wrong | **ACCEPTED — and NOT REPRODUCED A SECOND TIME EITHER.** Both reviewers' replacement figures are also stale. Re-measured: see §OPEN ITEMS 3 | `GET /api/health` → `27e062eab5d5`, PID **368964** since 18:14:23 CT; `deploy/RELEASE` = `27e062ea`; `GUIDE_BUILT_REV` = `27e062ea…` |
| F-19 | DOC-DRIFT | the report says 29 files; the diff is 31 | **ACCEPTED, CORRECTED** (now 34 after this round) | `git diff --name-only origin/dev...HEAD \| wc -l` |
| F-20 | WEAK-TEST | two mutation quotes (D1-D, D4-A) cite a different test's failure than the pin they are filed under | **ACCEPTED, CORRECTED** — see §PINS | both pins are real; the quotes were mislabeled |
| F-21 | WEAK-TEST | D2-E proves function parity, not call-site parity, so it cannot see the input moving | **ACCEPTED — and it was never meant to.** D2-E is kept as the *estimator* parity anchor and explicitly relabeled as such; **D3-F is the call-site golden** | class-53 trap named in the pin's own comment |
| F-22 | WRONG | the store is contract-blind, and D2/D3 extend a roll-seam hazard from 41.7 h to 21 days | **ACCEPTED AS A KNOWN LIMITATION, DOCUMENTED, NOT FIXED** (the wire and the AddOn are out of footprint) | `.schema bars` — PK is `(symbol, tf, open_time_ms)`; no contract column. See A15 |

**Reviewer claims re-measured and found DIFFERENT:**

- Two reviewers reported `CompletedWeekCount` 0→**1** and 0→**2**. Measured
  here at `now=2026-09-09 18:3x CT`: **0 → 2**. A figure two reviewers disagree
  about is measured, not averaged.
- One reviewer reported the RV move as `0.877759 → 0.885826` (7→9 days) and
  another as `0.877507 → 0.885629`. Measured here from the store's newest 5m
  rows: **0.876371 (7 d) → 0.884746 (9 d)**. All three agree on the SHAPE and on
  the day counts; the third decimal differs because the store grew between the
  reads. The number that actually ships is neither of them — it is
  **0.893543 over 9 days**, from the 1m tail (owner condition (b)).
- The reviewers' running-rev corrections (`8dfe6bc1a7ba` / PID 239714) were
  **already stale** when this repair started. Re-measured: `27e062eab5d5`,
  PID 368964.

### F-1 — the sentence that had to be retracted, and the ruling that resolved it

Three shipped artifacts said the computed value did not change:

- this report: *"The computed value is unchanged"*;
- `docs/superpowers/SYSTEM-MAP.md` §1: *"The COMPUTED value is unchanged"*;
- `web/src/guide/content/weeklyBias.ts`: *"The computed value did not change —
  only the ask and the name."*

All three are now corrected. **The owner's words:** *"a report that claims no
behaviour change while the boot line shows one is class 82."*

**EVERY VALUE THIS WAVE MOVES, NAMED, WITH ITS MEASURED DELTA:**

| value | rendered where | before | after | source of the change |
|---|---|---|---|---|
| `RVBaseline` | `RV=N%-of-baseline(…)` in the planner prompt | 0.876371 | **0.893543** | owner (b): 5m ring → 1m tail |
| `RVBaselineDays` | same clause | 7 | **9** | same |
| `CompletedWeekCount` (`nw`) | `(thin history Nw)` in the WEEKLY line | 0 | **2** | D2 1m store splice |
| `WeeklyShadowRefs` | 🌗 SHADOW counters (shadow only, no order path) | 1 | **3** | D2 1m store splice |
| `ComputeWeeklyFacts` Weeks / NWOGs | weekly facts + `FactsHash` | 0 / 0 | **2 / 2** | D2 1m store splice |
| daily session candle rows | the planner prompt's daily table | 2–3 | **8** | D2 1m store splice (the wave's stated deliverable) |

`ThinHistory` stays **true** in both states — 12 completed weeks is still out of
reach, and nothing pretends otherwise.

**What does NOT move:** the RULE. `kernel.RVBaselineFrom5mDays` is byte-for-byte
unchanged, and D3-F pins the rendered regime label to a written-down golden on a
fixture whose window does not change. It fails under three separate mutations
(below), so it is a pin and not a decoration.

---

## SCOPE

`git diff --name-only origin/dev...HEAD` — **34 files** (was 29 when the first
draft was written; 31 at review; 34 after this repair round), all inside the
wave's footprint. **No AddOn `.cs`, no subscription, no backfill, no
order/arm/gate path, no chart, no `deploy/` file, no `api/` route, no DB write.**

**Forbidden-territory check, restated against the owner's ruling.** Nothing in
this branch touches the planner's STRATEGY (only what the prompt DISCLOSES about
its tape), levels, any gate, arm or order path, arm 133, the chart, or the
store's real hole (the hole is DISCLOSED, never filled — and has since been
backfilled by the AddOn, not by this wave).

**Regimes:** the RULE is untouched and pinned byte-for-byte (D3-F golden). The
**INPUT** is deliberately corrected, which A31 would otherwise have forbidden and
which the **owner ruled IN SCOPE on 2026-09-09 18:18 CT** with four conditions,
all four implemented and pinned (§D3). This is stated as an authorised expansion,
not smuggled through as "no change".

**THIS WAVE ADDS NO REJECT GATE.** Every added path is a log line, a marker
string, a column or a degradation. Grepped: every `refuse` / `skip` / `block`
token in the diff is a comment, a log line, or a WARN-and-return-what-we-had.

**A29 — "production call sites: 0" grep.** Asserted by
`TestD2WiredAtTheThreeCallSites`, `TestRegimeInputWindowIsWired`,
`TestRingRehydrateIsWiredAtBoot`, `TestRehydrateSelectsOnly1mPairs`,
`TestReadFactRowBuilderIsWired`, `TestPlannerCandleTablesAreWiredIntoThePrompt`
and `TestBarHorizonHasProductionCallSites`.

**Two functions have ZERO production call sites BY DESIGN, and are named here
rather than counted as covered** (the earlier blanket claim over-reached — F-9):

- `store.PlannerReadFact.HorizonRecorded()` — nothing in Go reads these rows
  back; the readers are SQL. It exists so the NULL/`''` rule has ONE definition
  to cite. Said so in its own doc comment.
- `kernel.RVBaselineFrom5m` — production reads `RVBaselineFrom5mDays` via
  `ResolveRVBaselineTape`. The wrapper is KEPT as the parity anchor D2-E checks
  the day-carrying function against. Said so in its own doc comment.

**A12 — guide + SYSTEM-MAP** are updated in the SAME commit as each behaviour
change (six guide cards, two SYSTEM-MAP bullets, three knob rows).

**`GUIDE_BUILT_REV` — CORRECTED, and it is NOT a free choice.** The earlier text
said it was "deliberately left at `954f11b15f2e`". That value is not in the file
and never was. **Measured 2026-09-09 18:4x CT:**

```
web/src/guide/types.ts:6   GUIDE_BUILT_REV = '27e062eab5d536b5c42b3d0f1e0843ca9fd7537b'   (inherited from dev, untouched by this branch)
GET /api/health            {"revision":"27e062eab5d5","status":"ok"}
```

The two prefixes are **IDENTICAL**, so `GuidePage`'s drift banner is **SILENT**
— it will not warn that these six new cards describe a binary that does not have
the behaviour. The failsafe cannot fire here. Repo convention is that the deploy
marker owns the bump (`deploy: boot marker — RELEASE=… + GUIDE_BUILT_REV=…, from
the MAIN TREE after the passed boot`), so the bump is **not** taken on this
branch — but it becomes a **MANDATORY pre-boot step, not an optional one**:
bump `web/src/guide/types.ts` and rebuild `web/dist` BEFORE the boot (Boot-5
ordering). Recorded in OPEN ITEMS 4 and in A15.

---

## A15 — WHAT THE OWNER WILL STILL SEE WRONG

Stated up front, because a report that only lists what it fixed is a sales
document.

1. **The RV baseline number will STEP at the first boot** — `RV=…%-of-baseline(7
   complete session-days)` becomes `(9 complete session-days)` and the percentage
   moves. This is the fix, not a regression, but it is a visible discontinuity in
   a number the owner reads daily. The `📈 regime input window` boot line exists
   precisely so the step is dated and explained on the record.
2. **The WEEKLY line's `(thin history Nw)` will step 0 → 2** and the 🌗 SHADOW
   counters will move 1 → 3 refs. Shadow only; no order path. Any before/after
   comparison of the Sep-9 promotion table straddles two different input depths.
3. **`kernel/weekly_prompt.go` still lies in exactly the way C1 describes.**
   `## Weekly candles (12 completed weeks, oldest → latest)` renders over as few
   as one row. Left alone because `SectionsText` feeds `WeeklyFactsHash` — a
   heading change moves a stored hash. It is the same defect class, one file over,
   and it is not fixed.
4. **A row still cannot explain its own stop floor.** `read_horizons` carries ONE
   entry (the void 1m tape); `stop_floor_pts` comes from `plannerATR5m`'s separate
   5m×200 fetch. The array shape is additive so the `atr5m` entry needs no
   migration — but it is unbuilt (OPEN ITEM 2).
5. **The store is CONTRACT-BLIND, and this wave extends the blast radius.**
   `bars` has PK `(symbol, tf, open_time_ms)` and no contract column. After the
   MNQ September→December roll, the ring is the new contract while the store's
   older rows are the old one, and rule 3 admits them *because* they are older.
   The quarterly basis is tens of points, so the seam renders as a real price step
   in the 8-session daily table, **unmarked** — D1 measures presence, not identity.
   The hazard pre-existed (the ring merges across a roll too) but its reach was
   41.7 h; D2 extends it to the 1m retention window. **Documented at the splice,
   NOT fixed** — the wire and the AddOn are out of footprint.
6. **The 🕳 SHORT warn now fires on every planner read and every weekly shadow**,
   because `barsWithStoreDepth` asks the 2,500-bar ring for 12,000 and is then
   satisfied by the store one layer up. It is deliberately NOT suppressed:
   suppressing it would hide a genuinely dead feed on the day the store also
   fails. Read it PAIRED with the `📚 bars depth: … ring <horizon> →
   store-deepened <horizon>` line that follows it.
7. **`GUIDE_BUILT_REV` equals the running rev**, so the drift banner is silent
   while this branch's guide cards describe behaviour the running binary lacks.
   The bump is mandatory before the boot, not optional.
8. **Nothing here has run on a deployed binary.** Every figure is measured
   against the store, the stored prompts and the production functions in a test
   harness. See OPEN ITEMS 3 for the exact lines to read on the first boot.
9. **The wave's own flagship live example is now historical.** The 696-minute
   09-09 outage has been backfilled; an identical read today records `gaps=0`.

---

## ROLLBACK

Six commits. Each is revertible on its own **except** that D3 uses
`store.BarHistoryStore.LastNBars`, introduced by D2. Revert order is therefore
**REPAIR · D4 · D3 · D2 · D1**.

| commit | what a revert costs | what it restores |
|---|---|---|
| REPAIR (this round) | the two D1 blocker fixes, the four owner conditions, 8 new pins, every corrected sentence | the pre-review branch: a ⚠PARTIAL on every healthy read and "COMPLETE" across a hole |
| D4 `read record carries a HORIZON` | 4 additive columns + `read_horizons`; rows go back to a bare count | nothing else — the columns are additive and unread by Go |
| D3 `ring rehydrates from the store` | the boot rehydrate and the `📈 regime input window` line; the ring returns to the AddOn seed **and the RV baseline returns to ~7 days via the fallback arm** | pre-wave ring behaviour exactly |
| D2 `store is the horizon` | the 1m store splice; the daily table returns to 2–3 rows; `nw`/`WeeklyShadowRefs` return to 0/1 | pre-wave asks (but the 3 unreachable asks come back) |
| D1 `HELD-vs-REQUESTED` | all candle disclosure; headings return to baked `(last 12)` literals | pre-wave prompt text |
| `a2d85bad` (bridge observability; was `4b2edea3` before three rebases) | the 🕳 warn + counters | pre-wave silence at the choke point |

A revert of any of these is **safe at any time** — none of them gates, sizes,
routes or blocks anything, so a revert can only remove observability and depth,
never unblock a refusal.

---

## OPEN ITEMS

1. **`kernel/weekly_prompt.go` has the identical C1 defect, unfixed.**
   `## Weekly candles (12 completed weeks, oldest → latest)` renders over as few
   as one row; `CompletedWeekCandles(bars1m, now, 12)` and
   `LastNWOGs(bars1m, now, 5)` ask for depth the tape cannot hold. Deferred
   because `SectionsText` feeds `WeeklyFactsHash`, so a heading change moves a
   stored hash — a deliberate decision, not a side effect.
2. **`planner_read_facts` still attributes a floor to the wrong tape.** Row 66
   records `scope_intv='1m' scope_bars=2000` beside `atr5m=19.7631429163709` and
   `stop_floor_pts=29.6447` — but `plannerATR5m` fetches **5m × 200**, a
   DIFFERENT tape. `read_horizons` is already a JSON ARRAY so a second entry
   (`who="atr5m"`) is additive, but wiring it needs a `plannerATR5mWithHorizon`
   variant, outside D4's stated scope. The field doc now says the entry is
   **not** there, instead of claiming it is (F-10).
3. **Live proof owed, against the CURRENT rev.** Re-measured 2026-09-09 18:4x CT
   — every earlier figure in this report and in both reviews was stale:

   ```
   GET /api/health   → {"revision":"27e062eab5d5","status":"ok"}
   ps                → PID 368964  /home/hoang/nofx/nofx-bin  started 18:14:23 CT
   deploy/RELEASE    → 27e062ea
   ```

   Five things to read on the first boot of THIS branch:
   `🧯 ring rehydrate done: N of M symbol×tf pairs deepened … K pair(s) SKIPPED as tf!=1m`
   · `📈 regime input window @… BEFORE window=…d AFTER window=…d Δ…`
   · the first planner prompt's `TAPE:` line and HELD-vs-REQUESTED headings
   (and that the daily table renders **8 of 8**) · the first `planner_read_facts`
   row with a non-NULL `read_horizons` · and the WEEKLY line's
   `(thin history Nw)` clause, which should read **2w**, not **0w**.
4. **`GUIDE_BUILT_REV` bump is MANDATORY before the boot, not optional.** It
   currently EQUALS the running rev (`27e062ea…`), so the drift banner is silent
   and cannot warn anyone. Bump `web/src/guide/types.ts` + rebuild `web/dist`
   BEFORE the boot (Boot-5 ordering). See §SCOPE.
5. **A pre-existing FE test failure**, unrelated to this wave and present on the
   untouched tree: `web/src/guide/GuidePage.test.tsx` fails in a worktree with
   `Error: Denied ID /home/hoang/nofx-barshorizon/branding/product.txt?raw` —
   vite's `fs.allow` root does not cover the worktree path. **Re-verified
   pre-existing at repair time** by stashing the ONLY FE edit
   (`web/src/guide/content/weeklyBias.ts`) and re-running the full suite:
   byte-identical result either way —

   ```
   with the edit:     Test Files  10 failed | 47 passed (57)   Tests  9 failed | 345 passed (354)
   without the edit:  Test Files  10 failed | 47 passed (57)   Tests  9 failed | 345 passed (354)
   ```

   Every failure is `Error: Denied ID …/branding/product.txt?raw`, and no failing
   file imports the edited card. `npx tsc --noEmit` exits **0** with the edits.
6. **Contract-roll marking** (A15 item 5) — a later wave should persist the
   resolved contract per bar and MARK, never drop, a splice across a contract
   boundary. Before the MNQ September→December roll.

## UNCOMMITTED-FILE DISPOSITION

### Round 2 — the four files the interrupted REPAIR attempt left behind

A previous repair attempt was stopped mid-flight by a session limit and left
four files modified but uncommitted. Each was read before anything else was
done, and judged against the owner's ruling:

| file | disposition | reason |
|---|---|---|
| `kernel/candle_disclosure.go` | **KEPT, then extended** | It already contained both owner-mandated D1 fixes (`observableEndMs` floors; the between-row absent-row walk) plus `firstOpenGridMs`. All three re-verified here RED-before-GREEN and mutation-tested (M1, M6, M7, M8). Nothing in it acted on a refuted premise. |
| `kernel/planner_prompt.go` | **KEPT** | Tail-trim-before-measure + the `absent`/`unit` wiring the markers need. Re-verified. |
| `kernel/planner_candle_disclosure_test.go` | **KEPT, relabeled** | Six new pins. Two carried the label `PIN D1-H`, which already existed (tape age); the new ones were renumbered **D1-I…D1-N**. Every one re-run RED then GREEN then mutated here. |
| `trader/ninjatrader/bar_persist_wire.go` | **KEPT, then CORRECTED** | The 1m-only rehydrate is exactly owner condition (a), and the `afterBackfillHook` ordering fix is right. **But its header comment claimed "this changes no computed value anywhere"** — false under the ruling, and the ruling explicitly forbids that claim. Rewritten to quote the ruling and name every value that moves. The `tf != rehydrateTimeframe` inline test was also refactored into `pairsToRehydrate` so a pin can drive the FILTER rather than only assert the constant still exists (a mutation that disabled the filter would otherwise have passed). |

**Nothing was discarded in round 2** — none of the four acted on a refuted
premise; one made a claim the ruling forbids, and that claim was corrected.

### Round 1 — the four files the ORIGINAL build left behind

Four files were left uncommitted when the previous build was stopped mid-flight.

| file | disposition | reason |
|---|---|---|
| `kernel/void_scope.go` | **KEPT** | `VoidScope.Horizon` + the boot line. Correct under the corrected premise; its cited figures (3,055-minute span, 696 gaps) were re-verified first-hand and reproduce exactly. |
| `store/planner_read_facts.go` | **KEPT** | The four additive columns + `HorizonRecorded()`. Its comment already stated R1 as NOT REPRODUCED and explicitly refused to redefine `scope_bars`. |
| `store/planner_read_facts_test.go` | **KEPT** | PIN 4 (a legacy row reads UNKNOWN, not zero). Mutation 17 confirms it is a real pin. |
| `trader/read_facts_horizon_test.go` | **DISCARDED and REWRITTEN** | As written it referenced four symbols that do not exist in the repo — `buildReadFactRow`, `plannerATR5mWithHorizon`, `traderProdCallSites`, `containsAny` — so it could not compile, let alone run. It was also built around the atr5m second tape, which is outside D4's stated scope. Rewritten narrower against the void tape (pins D4-A/B/C, plus D4-D/E added by mutation); the atr5m tape is filed as OPEN ITEM 2 with its evidence rather than dropped. |
