# Wave 1D — per-condition expectancy on the corrected column

**Branch** `fix/expectancy-1d` (worktree `~/nofx-expectancy`, off `origin/dev` 5ebeb5a2)
**Status** BUILT, GREEN, **NOT DEPLOYED** — the cutover needs the owner's GO (A3).
**Scope** a read model, one read-only endpoint, one Studio panel, Guide §14, checklist.
It gates nothing, sizes nothing, prompts nothing. Class 23/A10 throughout: a failure
here WARNs and returns an empty table; it may never stop the loop.

---

## 1 — THE MODEL (D1/D2/D3)

**Row key.** condition × session × level kind × entry path (`armed` | `decision`) ×
era (`pre-0B` | `post-0B`), with roll-ups to condition, session, level kind and path.

**Per cell.** n · wins/losses/flats · Σ `pnl_corrected` · mean · sd · Wilson interval
on the WIN RATE · a normal interval on the MEAN · t-stat · avg realized R · avg
planned R:R · median MAE/MFE · stop-hit and target-hit share · unresolved excluded
(counted) · **the row ids**.

Two intervals, deliberately: `wilson_lo/hi` answers "how often does it win", `mean_lo/hi`
answers "does it pay". The promotion criterion is judged on the second. They are never
interchanged, and both use one `z` constant so a cell cannot mix confidence levels.

**Floor.** `n < 30` → `DESCRIPTIVE ONLY`, no verdict rendered. At or above it the status
is computed from the pre-registered rule: **PASSES ⟺ n ≥ MinN ∧ mean > 0 ∧ mean interval
excludes 0**, else FAILS. `MinN` and the rule travel in the payload, so the panel holds no
copy of either and cannot drift from the binary that computed the statuses.

### Two derivations that had no stored source

Neither is a guess; each is an identity, and each returns "" (counted) when it cannot be
made.

- **Condition is not a column.** `trader_positions` carries `cited_scenario_id`; the
  scenario that id names lives in `plans.doc`. No doc or no scenario → excluded as
  `no_condition`.
- **Level kind is not a column either.** The doc stores only a provenance label
  (`ONL`, `SWG-L·5m`, `OB(bull)·1h (HTF)`, `nPOC·Tue`). One canonicalizer strips the
  decorations and matches the kernel enum **by reference** (`kernel.KindSWGL`, never a
  copy of its spelling), and a scenario is tied to its level by **exact price identity**
  with the confirm's reference price — never the nearest level, which would be a
  fabricated join.

### The era boundary is a resolved instant, not a literal

`Era0BStart = 2026-09-02 07:49:06 CT`, from commit `617faae4` ("lane 2 booted at
07:49:06 CT mid-wave"). **Not** `4175e0b6`'s commit time of 07:44:37 — code that is
committed is not code that is running, and a trade is graded by the binary that took it.
Built from date+time in CT the way `store.DayPlanEraStart` is, and the test asserts the
resolved wall-clock fields so a wrong fallback zone is visible rather than latent.
Split by **timestamp, not session-day**, because 0B booted mid-day (E3).

---

## 2 — FILE:LINES

| What | Where |
|---|---|
| Types, floor, statuses, era constant | `expectancy/model.go` (240) — `MinN` :27 · `Era0BStart` :74 |
| Entry point (caller's clock, A28) | `expectancy/aggregate.go:89 LoadAndBuildAt` |
| Pure aggregation | `expectancy/aggregate.go:186 BuildAt` |
| Era filter (re-aggregates) | `expectancy/aggregate.go:282 FilterEra` |
| Era split by timestamp | `expectancy/aggregate.go:303 eraOf` |
| Condition + level-kind recovery | `expectancy/aggregate.go:344` / `:366` |
| Cell statistics + the floor | `expectancy/aggregate.go:470 computeCell` |
| Wilson score interval | `expectancy/aggregate.go:523 wilson` |
| E8 counterfactual side-table | `expectancy/aggregate.go:564 buildE8` |
| Boot line (READ) | `expectancy/aggregate.go:671 BootLine` |
| Level-kind canonicalizer | `expectancy/levelkind.go:47 LevelKindFromLabel` |
| Tests (13 cases) | `expectancy/expectancy_test.go` (736) |
| Endpoint | `api/handler_expectancy.go:38` · route `api/server.go:569` |
| Boot line emission | `main.go:317` |
| Panel | `web/src/components/plan/ExpectancyPanel.tsx:92`, mounted `PlanCard.tsx:97` |
| Panel tests (9 cases) | `web/src/components/plan/ExpectancyPanel.test.tsx` |
| Guide §14 | `web/src/guide/content/expectancy.ts` |
| Checklist classes 62–63 | `docs/superpowers/AUDIT-CHECKLIST.md` |

---

## 3 — E1 RED → GREEN, quoted

**RED** (`go test ./expectancy/`, before any implementation existed):

```
# nofx/expectancy [nofx/expectancy.test]
expectancy/expectancy_test.go:156:76: undefined: TestSeamSource
expectancy/expectancy_test.go:170:14: undefined: LoadAndBuildAt
expectancy/expectancy_test.go:250:60: undefined: Cell
expectancy/expectancy_test.go:270:5:  undefined: MinN
expectancy/expectancy_test.go:281:19: undefined: StatusNotEnoughData
...
FAIL	nofx/expectancy [build failed]
```

**GREEN**, all 13 cases:

```
--- PASS: TestE1FixtureCellsMatchHandComputedStats
--- PASS: TestE2FloorAt29And30
--- PASS: TestE3EraSplitByTimestampNotSessionDay
--- PASS: TestE4CounterfactualSideTableFlagsShortRows
--- PASS: TestE5FigureIsReproducibleFromStoredRowIDs
--- PASS: TestE6EraBoundaryIsTheRecordedBootInstant
--- PASS: TestE6BuildAtUsesTheCallersClock
--- PASS: TestLevelKindCanonicalizerResolvesLabels
--- PASS: TestKindAndPathRollUpsArePooledNotMeansOfMeans
--- PASS: TestFilterEraReAggregatesInsteadOfSlicing
--- PASS: TestBootLineIsReadNotLiteral
--- PASS: TestBootLineCountsJudgedRollUpsSeparatelyFromCells
--- PASS: TestE8RefusesToMeanAnUncomputableColumn
--- PASS: TestE8MeanIsAbsentWhenNoRowIsUsable
ok  	nofx/expectancy
```

The E1 fixture holds exactly 40 positions across 3 conditions with every exclusion class
present (2 NULL `pnl_corrected`, 1 test-seam, 1 sentinel, 2 crypto-era) and asserts
n/mean/sd/t/Wilson to **1e-6** against values computed independently in Python, not by the
code under test. Crypto-era rows are asserted **ABSENT** rather than excluded-and-counted.

**FE RED observed too** — with the component moved aside: `Failed to resolve import
"./ExpectancyPanel"` / `Test Files 1 failed`; restored, 9/9 pass.

**Suites at this HEAD:** Go `./...` **0 failures** · vitest **42 files / 319 tests** ·
`tsc --noEmit` clean · kernel goldens ok. Run at authoring time; **A28 requires them re-run
at merge time immediately before the build.**

---

## 4 — THE LIVE TABLE (A15)

Read from an online `sqlite3 .backup` snapshot of `data/data.db` (the running bot was not
touched), through the real `LoadAndBuildAt`.

```
📊 expectancy: cells=41 with_n>=30=0 judged_rollups=2 unresolved=3 excluded_test=3
as_of=2026-09-03T14:20:45Z  cells=41 conditions=6 sessions=3 kinds=16 paths=2 e8=8
excluded: unresolved_pnl=3 unresolvable=7 test_seam=3 no_condition=0 crypto_era=0
```

**By condition — the table the rulings are made from:**

| condition | n | W/L/F | Σ | mean | sd | mean 95% | win% | status |
|---|---|---|---|---|---|---|---|---|
| acceptance | 6 | 1/4/1 | +4.57 | +0.76 | 155.42 | [−123.60, +125.12] | 17% | NOT ENOUGH DATA |
| breakout_retest | 9 | 1/8/0 | −581.50 | −64.61 | 57.60 | [−102.24, −26.98] | 11% | NOT ENOUGH DATA |
| hold | 1 | 1/0/0 | +168.00 | +168.00 | 0.00 | [+168.00, +168.00] | 100% | NOT ENOUGH DATA |
| reclaim | 5 | 0/5/0 | −436.50 | −87.30 | 39.42 | [−121.86, −52.74] | 0% | NOT ENOUGH DATA |
| **reject** | **31** | 14/16/1 | +586.00 | **+18.90** | 105.56 | **[−18.26, +56.06]** | 45% | **FAILS** |
| sweep_reclaim | 6 | 1/5/0 | −207.00 | −34.50 | 64.71 | [−86.28, +17.28] | 17% | NOT ENOUGH DATA |

Sample ids for `reject` (A21): 521, 523, 524, 529, 533, 535, 536, 537, 538, 544, 547, 548,
549, 550, 551, 552, 553, 554, 560, 564, 565, 567, 569, 570, 575, 581, 582, 584, 585, 587, 591.

**Exactly one condition clears the floor, and it FAILS** — n=31, mean +18.90, but the 95%
interval on the mean runs from −18.26 to +56.06 and therefore includes zero. The most-traded
play in the system has not been shown to pay. Everything else is DESCRIPTIVE ONLY, which is
what C3 predicted and what the floor is for.

By session: ASIA n=16 mean −34.53 · LONDON n=21 mean +1.14 · NY n=21 mean +2.95 — all below
the floor, no verdicts.

MAE/MFE and the stop/target shares are **blank on every row**: `trade_excursions` holds 0
rows, so wave 1A has not populated them. Blank, never 0.

**The exclusion ledger reconciles exactly.** Nine CLOSED rows carry the `UNRESOLVABLE`
sentinel (530, 539, 545, 546, 566, 571, 573, 574, 580); two of them (573, 574) are
`e7_farside_test` and are attributed to TestSeam first, because exclusions are assigned
most-specific-first so a row lands in exactly one reason. 9 − 2 = the 7 reported.

---

## 5 — SURPRISES (A23) — two, both reported, neither repaired

Both were found by the **first live read**, not by the tests, and both are now checklist
classes appended in this same PR.

### Class 62 — a stored column that reads as money and is not

The E8 side-table's first live output gave cell means of **−29,926 / −29,210 / −32,893**
on an instrument trading near 29,900. `ab_confirm_log.net_pnl` is approximately
**−(entry price × multiplier)** on those rows: the exit was treated as zero.

```
n=188   price-scale 40   bare-zero-beside-a-resolved-outcome 92   usable 56
```

**30% of the column is arithmetic-able and nothing said so.** This is a different defect
from the known short-side sign bug — that one corrupts a sign; this one makes the magnitude
a price. The read model now marks a row usable only when `|net_pnl| < entry_px` and
`net_pnl != 0`, reports `N` beside `UsableN` and both exclusion counts, and returns an
**absent** mean when nothing is usable:

```
             ASIA   touch        n=7   usable=0   mean=—       zero=7    suspect=true
             LONDON touch        n=5   usable=0   mean=—       zero=5    suspect=true
             NY     1x5m_close   n=18  usable=0   mean=—       zero=18   suspect=true
reject       ASIA   1x5m_close   n=39  usable=18  mean=−6.20   price_scale=10 zero=11
reject       LONDON 1x5m_close   n=32  usable=9   mean=−14.32  price_scale=8  zero=15
reject       NY     1x5m_close   n=78  usable=22  mean=−15.93  price_scale=22 zero=34
sweep_reclaim LONDON 1x5m_close  n=4   usable=3   mean=−6.83   suspect=false
sweep_reclaim NY     1m_mss      n=5   usable=4   mean=−8.14   suspect=false
```

**Not repaired here.** `kernel/shadow_ab.go` is outside 1D's footprint, and a read model
that silently patched its input would hide the defect it found. **Owner decision needed:
the E8 table cannot support a shadow/promote ruling in its current state** — 30% usable, no
direction column, and the sign defect still open.

### Class 63 — two true numbers that mislead together

The boot line read `cells=41 with_n>=30=0` while the `reject` condition roll-up stood at
n=31 and judged FAILS. Both true; the juxtaposition implied a false third ("nothing is
judgeable"), because verdicts are made on roll-ups and the line counted only the
five-dimensional cells. Fixed by adding `judged_rollups=`, pinned by a fixture where every
cell is sub-floor while three roll-ups clear the floor — asserting the exact count 3, since
a lax `>0` would still pass if a roll-up set were silently dropped.

---

## 6 — ROLLBACK (A13, by content)

Nothing is deployed, so there is nothing to roll back yet. At cutover the rollback target
is the binary holding **`8d47cc21`** (the rev currently live, per `deploy/RELEASE` and
`GUIDE_BUILT_REV`), restored by content and not by name.

Because 1D is **additive and read-only**, a narrower rollback exists and should be
preferred: the panel and the endpoint can be left in place and only the boot-line call at
`main.go:317` removed — no gate, exit, prompt, validator, detector or attribution writer is
touched by this wave, so backing it out cannot change what the bot does.

---

## 7 — OWED AT MERGE (not done here, deliberately)

1. **`GUIDE_BUILT_REV` bump** to the merged HEAD (A12) — done at merge, so it names the
   rev that actually ships. The new Guide section already stamps `asBuiltRev`.
2. **Class numbers 62/63 confirmed** at merge (A27/A16) — highest occupied at authoring
   was 61.
3. **Re-run the full suite at the MERGED HEAD, immediately before the build** (A28) — a
   branch green alone is not green merged.
4. **Cutover + live proof** (Section F): boot, then quote the live `📊 expectancy` line and
   the top three conditions by n with DESCRIPTIVE ONLY where it belongs. Requires the
   five-leg gate `ready:true`, the owner's explicit GO, and the owner present for the kill.
   **Not attempted** — A3 forbids an unattended deploy and the classifier denies the kill
   to the agent.
5. **A14 commit-ref 200 on dev** — curl'd after the merge lands on `dev`.
