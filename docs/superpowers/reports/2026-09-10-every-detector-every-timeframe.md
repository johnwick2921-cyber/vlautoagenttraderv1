# W-TF — every detector, every timeframe

**Dispatch** 103, owner hoang, 2026-09-10 · SELF-CONTAINED · SIM-only (Sim101), MNQ, one contract, `plan_mode=strict`
**Branch** `fix/every-detector-every-timeframe` · claim `81bb9f3c` · composite id `every-detector-every-tf-51524a30/nofx-6d[63eafa]`
**Running rev at measurement** `8941ec68612c` — `/api/health` → `{"revision":"8941ec68612c"}`, PID 1953256, `/proc/1953256/exe → /home/hoang/nofx/nofx-bin`, started 06:56:51 CT; `go version -m` → `vcs.revision=8941ec68612cc019edc3002b999272ab2ed20516`, `vcs.modified=false`
**Evidence** **[A]** directly verified · **[B]** inferred · **[C]** speculation · **[I]** installed-but-untested

---

## 0. THE ONE-LINE RESULT

The owner's configuration has been asking for daily structure since the default
was written. `planner_timeframes` reads `["D","4h","1h","15m","5m"]` and names
`D` **first**. The detector gate only ever recognised the spelling `"1d"`, so
`"D"` fell through a `!isHTFDetectionTF(tf)` branch and was discarded without a
log line. **[A]** Measured: **0 of 297** stored plans carry a level from any
daily timeframe, while the store holds 1d n=1902 back to 2019-05-02.

This wave did not build a timeframe loop. One already existed. It removed the
two reasons the daily family could not reach it — a gate that stopped at 12h,
and an alias nothing resolved — and made a level's timeframe part of its
identity so cross-timeframe references stop silently eating each other.

---

## 1. SECTION C — EVERY PREMISE, MEASURED AT THE RUNNING REV

### C1 — "each detector runs on one hard-coded timeframe" · **NOT REPRODUCED**

A per-timeframe loop has existed since G2/G3 (2026-08-24). `DetectHTFLevels`
(`kernel/levels_assemble.go:284`) runs **four** detector families per timeframe:
`EqualHighsLows`, `SupplyDemandZones`, `FairValueGaps`, `OrderBlocks`. **[A]**

What was actually hard-coded is the *set*:

| call site | timeframes passed | effective detector×tf |
|---|---|---|
| executor `kernel/engine_analysis.go:403` | `[]string{"1h","4h"}` — a literal | 4 × 2 = **8** |
| planner `trader/auto_trader_planner.go:2167` | configured `planner_timeframes` = `["D","4h","1h","15m","5m"]` | 4 × 3 = **12** (`D` dropped, `5m` rejected) |
| gate `isHTFDetectionTF` | `15m 30m 1h 2h 4h 6h 8h 12h` | ceiling 4 × 8 = 32 |
| store holds | 14 distinct timeframes | ceiling 4 × 14 = 56 |

Base swings remain 5m/15m by design (`kernel/levels_swing.go:178-180`) and the
owner's ruling kept them there.

**Section B named three files that do not exist** — `levels_ob.go`,
`levels_fvg.go`, `levels_eq.go`. Order blocks and equal highs/lows live in
`levels_zones.go` and `levels.go`; `fvg_entry.go` is entry mechanics, not a
level detector. Work was done where the detectors actually live, per the owner's
correction.

### C2 — store depth · **NOT REPRODUCED**

The dispatch stated 1m 90d · 5m 180d · 15m 365d. Measured **[A]**, `bars` where
`symbol='MNQ'`:

| tf | rows | first | last | actual depth |
|---|---|---|---|---|
| 1m | 22,006 | 2026-08-19 | 2026-09-10 | **22d** |
| 3m | 4,595 | 2026-08-27 | 2026-09-10 | 14d |
| 5m | 3,556 | 2026-08-24 | 2026-09-10 | **17d** |
| 15m | 2,518 | 2026-08-04 | 2026-09-10 | **37d** |
| 30m | 2,259 | 2026-07-03 | 2026-09-10 | 69d |
| 1h | 2,129 | 2026-05-04 | 2026-09-10 | 129d |
| 2h | 2,066 | 2026-01-12 | 2026-09-10 | 241d |
| 4h | 2,032 | 2025-05-18 | 2026-09-10 | 480d |
| 6h | 2,022 | 2024-09-25 | 2026-09-10 | 715d |
| 8h | 2,019 | 2024-02-01 | 2026-09-10 | 952d |
| 12h | 2,012 | 2022-10-19 | **2026-09-09** | 1,422d |
| 1d | 1,902 | 2019-05-02 | **2026-09-09** | 2,688d |
| 3d | 638 | 2019-05-04 | 2026-09-07 | 2,686d |
| 1w | 384 | 2019-04-26 | **2026-08-28** | 2,692d |

The dispatch also did not mention `3d`, which is present with n=638. The store
is far *shallower* intraday and far *deeper* on coarse timeframes than stated.

### C3 — "a level's identity has no timeframe" · **CONFIRMED**

`DetectedLevel` has carried a `TF` field since 2026-08-24 (`levels.go:79`), but
the **dedupe key never used it**: `dedupeSameKind` (`levels_assemble.go:257`)
matched on `o.Kind == l.Kind && math.Abs(o.Price-l.Price) <= 0.25`, first
occurrence wins. **[A]** A 1h order block and a 1d order block at one price
collapsed to whichever detector emitted first, and the loser left no record.

### C4 — daily/weekly levels ever emitted · **CONFIRMED: none**

**[A]** `plans` n=297. Rows whose doc contains a `1d`/`1w` timeframe marker: **0**.
Rows containing PWH or PWL: **11**. Timeframe suffixes across the 40 most recent
plans: `·1h` 160 · `·5m` 72 · `·15m` 30 · `·4h` 8. The map is dominated by 1h and
has never contained a daily structural level of any kind.

### C5 — the merge is timeframe-blind · **CONFIRMED, and that is the useful direction**

`BuildMapCandidates` groups on `math.Abs(out[i].Price - s.Price) <= width` —
price alone. **[A]** So a 15m level and a 1d level at one price already merge into
one candidate carrying both names rather than competing. `MapCandidate` has **no
TF field**; the timeframe travels in the name (`"Supply·1h · EQH·1d"`), which
`tagHTFLevel` appends. D3 therefore needed no merge change — only for the names
to carry real timeframes, which they now do.

### C6 — what the ×1.2 is applied to · **MEASURED, and D5's premise is INVERTED**

The multiplier is not a timeframe comparison. It fires on a **boolean**:
`levels_score.go:493-495`, `if l.HTF { htf = 1.2 }`, and `HTF` is set only by
`tagHTFLevel` for timeframes the detection gate admits. **[A]**

Before this wave, a daily level would therefore have received:

| | daily level, before | 4h level |
|---|---|---|
| `HTF` flag | **false** (1d not in the gate) → no ×1.2 | true → ×1.2 |
| `zoneTierFor` | falls to default → **`"1m"`** | `"4h"` |
| `zoneTFMult` | **1.0** | 1.3 |
| `KindOB` evidence | **0.40** | 0.72 |
| zone grade | the 1m clause forces **grade C** | floor B, may reach A |

D5 anticipated that exposing daily levels would newly *grant* them the ×1.2 and
move scores up. The opposite was true: they would have arrived as the weakest
rows on the map and been capped at C. **This was surfaced before building and
the owner ruled**: classify `1d`/`3d`/`1w` into the existing `4h` tier, which
changes no weight, no multiplier, no cap and no tolerance — only which tier an
input that was previously impossible is classified into.

---

## 2. THE DETECTOR × TIMEFRAME MATRIX

**Before** — `isHTFDetectionTF`: `15m 30m 1h 2h 4h 6h 8h 12h`

**After** — `HTFDetectionTFs` (the ordered single source the gate ranges over):
`15m 30m 1h 2h 4h 6h 8h 12h` **`1d 3d 1w`**

| | before | after |
|---|---|---|
| gate membership | 8 tfs | **11 tfs** |
| executor call site | literal `["1h","4h"]` → 4×2 = 8 | `DefaultHTFDetectionTFs` `[15m 1h 4h 1d 1w]` → 4×5 = **20** |
| planner call site | configured, `D` dropped → 4×3 = 12 | `D`→`1d` resolved → 4×4 = **16** (`5m` skipped **with a reason**) |
| detectors | four inline loops | a TABLE — the count is a value the boot line READS |

Sub-15m stays out and the original reason stands. `TestE1c_SubFifteenStaysOut`
pins it so widening that gate is a decision rather than an accident.

---

## 3. SECTION E — RED THEN GREEN

RED at `8941ec68`, quoted verbatim from the run:

```
--- FAIL: TestE1_DailyTimeframeReachesTheMap
    no level detected on 1d from 30 daily bars — the daily timeframe never reaches the map
--- FAIL: TestE1b_WeeklyTimeframeReachesTheMap
    no level detected on 1w from 30 weekly bars
--- FAIL: TestE3_TimeframeIsPartOfIdentity
    dedupeSameKind collapsed 2 levels to 1 — a 1h and a 1d level at one price are two references
--- FAIL: TestC6_DailyInheritsTheFourHourTier
    zoneTierFor("1d") = "1m", want "4h"   (also 3d, 1w)
--- FAIL: TestC6c_DailyCarriesTheHTFFlag
    no daily level produced; cannot assert the HTF flag
```

GREEN after D1/D2/C6: all 16 W-TF tests pass, full Go suite green.

**Mutation testing — 10 killed, each verified to have applied before its verdict
was trusted.** Three of the first eight were weak and the mutations found them:

| mutation | first verdict | why it survived | now |
|---|---|---|---|
| `tagHTFLevel` stops suffixing `·tf` | **SURVIVED** | E2 built its labels by hand, so it never drove `tagHTFLevel` | killed by `TestD4_`, which drives detection |
| `LookbackBars` never recorded | **SURVIVED** | nothing asserted D4's field at all | killed by `TestD4_` |
| `dedupeSameKind` drops `tf` | killed | | killed |
| gate drops the daily family | killed | | killed |
| tier drops `1d/3d/1w` | killed | | killed |
| partial window emits instead of skipping | killed | | killed |
| skip reason stops naming the shortfall | killed | | killed |
| dedupe drops `kind` | killed | | killed |

`TestD4b` was **SKIPPING** rather than asserting: the fixture hardcoded pivot
indices 3 and 8, which at n=9 puts a peak on the last bar where the k=2 pivot
window cannot reach it. Peak positions are now derived from the window size. A
skipped test is not evidence.

---

## 4. A15 — WHAT THE OWNER WILL STILL SEE WRONG

1. **`1w`'s last bar is 2026-08-28 — 13 days stale.** `12h` and `1d` last close
   2026-09-09. Weekly detection will run on a window whose newest bar is nearly
   a fortnight old, and nothing says so on the card. Filed as a feed finding for
   batch 2 per the owner's instruction. **[A]**
2. **Daily/weekly inherit the 4h tier as `[I]`.** Round 12 establishes no
   timeframe hierarchy (12e) and finds the ×1.2 untested (12c). Nothing shipped
   here asserts a daily level is stronger — the classification is the least-wrong
   available option, and E4 measures whether it is right. Both facts are on the
   boot line and in the Guide.
3. **The map's composition will move.** Daily and weekly levels have never
   appeared; they now can, at the 4h tier, and will compete for the same
   `max_levels` seats. This is a behaviour change by exposure, not by weight.
4. **`MapCandidate` still has no `TF` field.** A cross-timeframe merged candidate
   shows its timeframes only through the names (`"Supply·1h · EQH·1d"`). A
   consumer wanting to filter or sort by timeframe cannot, without parsing the
   label. Not fixed here — out of A31's scope.
5. **`auto_trader_planner.go:2051` says `"D"` maps to the provider's `"1d"`
   interval.** `structureSummaryLines` does no such mapping. The comment
   describes behaviour the function does not have — checklist class 105, in the
   file whose configuration this whole wave turned on. Recorded, not fixed.
6. **`SYSTEM-MAP.md` §2 said detection ran "off the 1m bar tape"**, which has
   been wrong since G2/G3 landed on 2026-08-24. Corrected in this wave.
7. **No RULEBOOK exists in the repo.** The dispatch's A12 requires "the
   RULEBOOK's Part 3 map table" in the same commit; `git ls-files | grep -i
   rulebook` returns nothing. The Guide and SYSTEM-MAP were updated instead.
8. **The web vitest suite is red on dev, not because of this wave.** 11 test
   files fail at this head; the **same 11 files** fail on `origin/dev`, measured
   by running the identical suite against dev's `web/src` in this environment.
   This wave adds zero failures. `tsc` is clean.
9. **Executor cost measured, not assumed**: the 2→5 timeframe expansion moves one
   detection pass from **217µs to 535µs** (`-benchtime 20x`, same 500-bar input
   per timeframe). Sub-millisecond on a path that runs about once a minute.
11. **`nofx/trader` is RED on dev, and it is not the lunch band.**
    `TestSplitArmWritesTwoLedgerRows` fails with *"split arm must write 2 ledger
    rows (legs), got 0"*. Measured **[A]** at three heads in clean worktrees
    carrying none of this wave's commits:

    | head | what it is | result |
    |---|---|---|
    | my merged head | W-TF + dev | FAIL |
    | `a98a92c7` | dev tip, after the settlement merge | FAIL |
    | `a48af5d7` | dev tip, BEFORE the settlement merge | FAIL |

    So it is neither this wave's nor the settlement wave's. The cause is **not**
    the wall clock: commit `2cc7c28c` (12:26 CT today, *"the arm-path tests read
    the wall clock, so dev is RED for 90 minutes a day"*) reached this test — it
    uses `armTestClock(t, at)` and derives its session, plan and bars from that
    one value, so A28 is honoured. The refusal is:

    ```
    🛑 arm stop NY S1 leg 1 short: stop 29510.75 · anchor none (stop_unanchored) · atr_floor
    🚦 entry-gate REFUSED arm NY: entry_gate: scenario S1 invalidated at an earlier cycle
    ```

    An entry-gate invalidation, not a band refusal. **This blocks Section F's
    "suite at the merged HEAD" requirement** and is recorded, not fixed (A23,
    and it is another lane's file).

10. **`3d` is in the gate but not in `DefaultHTFDetectionTFs`.** It will only run
    for a trader whose configured `planner_timeframes` names it. Deliberate: one
    rung per scale on the default path.

---

## 5. ROLLBACK

Every change is additive and reversible without a migration — no schema change,
no data written, no config touched.

- **Revert the wave**: `git revert` the range on `fix/every-detector-every-timeframe`.
  `DetectedLevel.LookbackBars` is `json:",omitempty"`, so stored docs written
  while it was live remain readable by the reverted binary.
- **Narrow without reverting**: remove `"1d","3d","1w"` from `HTFDetectionTFs`
  (one line, one source — the gate ranges over it). Detection returns to 15m–12h
  immediately; nothing else needs touching.
- **Binary rollback**: `nofx-bin.old.<rev it holds>` per A13, verified with
  `go version -m` before the swap.

---

## 6. WHAT THIS WAVE DID NOT TOUCH (A31)

No detector definition · no score weight · no `zoneTFMult` value · the ×1.2
multiplier's value (it was NAMED as `HTFScoreMultiplier` so the boot line reads
it; the number is unchanged) · `max_levels` · `clusterToleranceFor` · the bar
ring or store · `EntryGate` · `armed_executor` · the planner prompt's
instructions · W1's episode files · settlement's files.

The Stage A parity golden `stage_a_score_legacy.json` is byte-identical.

**One file outside the footprint declared at claim**: `kernel/engine_analysis.go`,
whose executor call site held the literal `[]string{"1h","4h"}`. A31 defines this
wave as changing *which timeframes the detectors run on*, and a hardcoded pair at
a call site is exactly that. Declared in the commit that changed it.
