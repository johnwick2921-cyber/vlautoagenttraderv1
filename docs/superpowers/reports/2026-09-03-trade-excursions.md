# TRADE EXCURSION LOGGING (wave 1A, checklist class 54)

**Branch:** `fix/trade-excursions` off `33de2bef` · worktree `~/nofx-excursions`
**Commits:** 08e0da2f · 44d4bbb7 · 01591040 · d4aee04a · a90f5de3 (+ this report)
**Checklist:** entry **54** (highest occupied at merge: **53**, void-parity)
**Boot line:** `📐 excursions: logging=on rows=… backfilled=… unresolved=…`
**Status:** NOT DEPLOYED. Window per A7; after attribution-integrity by merge order.

This wave measures. It changes no exit, no stop, no gate and no size.

---

## 1. Schema

`trade_excursions`, one row per position, unique on `position_id`.

| group | columns |
|---|---|
| identity | `position_id` `plan_id` `version` `session` `scenario` `condition` `side` |
| entry | `entry_px` `entry_ts` `stop_px_initial` `target_px` `size` `atr5m_at_entry` `atr_mult_stop_at_entry` |
| exit | `exit_px` `exit_ts` `exit_reason` `stop_px_final` `pnl_corrected` `ambiguous_exit` |
| path | `mae_pts` `mae_ts` `mae_bars_after_entry` `mfe_pts` `mfe_ts` `mfe_bars_after_entry` `bars_held` `ambiguous_bars` |
| provenance | `resolution` ("1m" / "5m" / "none") · `source` ("live" / "backfill") |

**Every path column is NULLABLE and starts NULL.** A stored `0` means measured
zero. `resolution="none"` means the tape does not reach this hold, and the row
keeps its NULLs rather than being handed a number.

`source` exists so nobody reads a backfilled corpus as live evidence; the boot
line reports the two separately.

## 2. Files

| file | what |
|---|---|
| `kernel/excursion_path.go` | `ComputePathExcursion` · `ResolveAmbiguousExit` |
| `kernel/excursion_path_test.go` | F1 (the 589 pin), F2, F3, no-coverage |
| `store/trade_excursion.go` | the table, DDL, writer, `Counts`, `ExcursionBootLine` |
| `store/trade_excursion_stats.go` | E6 distributions and the renderer |
| `store/position_excursion_null.go` | E4 migration |
| `trader/trade_excursion_hook.go` | E1/E2/E3 hooks, `safeExcursion`, level resolution |
| `trader/trade_excursion_backfill.go` | E5 backfill, `excursionCondition` |
| `cmd/excursions/main.go` | the read-only CLI |
| `store/decision.go` `StopTargetNear` | reads the levels the opening decision carried |
| wiring | `auto_trader_risk.go` monitorTick · `armed_executor.go` fill · `auto_trader_decision.go` open · `auto_trader_clock.go` close · `main.go` boot line |

## 3. What the old computation was doing

`kernel.ComputeExcursion` returned two floats, and skipped a bar when
`b.OpenTime < entryMs`. Unless a fill landed exactly on a bar boundary, **the
bar containing the entry was excluded** — with the first adverse move in it. The
filter is asymmetric: the bar containing the *exit* was kept. It also recorded
no timestamps, no bar offsets, no hold length and no intrabar ambiguity, so no
exit rule could be derived from it.

`ComputePathExcursion` counts every bar whose own window intersects the hold.

## 4. F1 — quoted RED, then GREEN

Position 589: MNQ LONG, filled 29192.50 at 14:41:04 UTC, stopped at 29115.00 at
14:59:27. The opening decision (`decision_records` 36394) carried
`stop_loss 29115`, `take_profit 29317.25` — the 77.5 pt widened stop of the
forensic. Nineteen 1m bars, read from the `bars` table and embedded verbatim.

RED (no such function):

```
kernel/excursion_path_test.go:55:9: undefined: ComputePathExcursion
```

RED against a stub, for the right reason:

```
--- FAIL: TestExcursionPin589
    589 has full 1m coverage — the path must be computed, not abandoned
--- FAIL: TestExcursion589AgainstTheRecordedRow
    the closed tape must be at least as adverse as the row recorded live: got 0, row 80.5
```

GREEN: MAE **81.25** at 14:59, eighteen bars in · MFE **10.25** at 14:44, three
bars in · **19** bars held (the entry bar included) · **0** ambiguous.

### 4.1 The 0.75 the stored row is missing

`trader_positions.mae` for 589 reads **80.5**; the closed tape says **81.25**.
80.5 implies a low of 29112.00, and no persisted bar has it — the 14:59 bar
closed at 29111.25. MFE matches exactly (10.25), and MFE landed mid-hold.

**[B]** The live computation ran at close time, 27 seconds into the final bar,
and read that bar while it was still forming. One explanation fits both halves:
the extreme that was still moving is understated, the one already set is exact.
Every close-time excursion is therefore biased slightly favourable. The wave's
per-tick recomputation from *closed* bars removes it. Pinned as its own test so
the number is asserted, not described.

## 5. E4 — the default-0 column

Measured on the live store, 586 closed positions:

| pattern | rows | reading |
|---|---|---|
| `mae=0 AND mfe=0` | **517** | never computed — price always moves |
| `mae=0`, mfe set | 4 | genuine zero MAE |
| mae set, `mfe=0` | 5 | genuine zero MFE |
| both set | 60 | measured |

88% of the corpus was unreadable, and round 7 ruled that exits, stop sizes and
targets come from these distributions.

The migration nulls **only the 517-row pair** and leaves the 9 genuine zeros
alone. The dispatch's blanket "existing 0s → NULL" would have destroyed them —
the same disease pointing the other way.

Two more faces, both fixed: `kernel.LearningLine` gated its average on
`if t.MAE > 0 || t.MFE > 0` (dropping a genuine zero, counting an uncomputed row
as merely absent) and printed an average with **no n at all** — it now carries
`Measured` and prints `(n=1 of 2)`. `recordClosedTradeAnalytics` now uses the
same path computation, so `trader_positions` and `trade_excursions` cannot
disagree, and with no coverage it WARNs and leaves NULLs instead of writing 0.

### 5.1 A23 — E4 would not have booted

Running the backfill against a copy of the live store failed to open it:

```
failed to migrate trader_positions table:
  error in view position_plan_join: no such table: main.trader_positions
```

Dropping `DEFAULT 0` makes GORM rebuild `trader_positions` (SQLite's 12-step
ALTER), and the `position_plan_join` VIEW references the table while it is gone.
**Store initialization fails, so the process would not have started.** Confirmed
by building the parent commit, which opens the same copy cleanly. **[A]**

The DDL default stays. Both entry writers now null the two columns explicitly
after the insert — nil-ness captured *before* `Create`, because GORM writes the
default back into the struct, and via raw SQL, because a GORM `Updates` map with
nil values is dropped rather than emitted. `TestNewPositionHasNullExcursions`
pins it and was quoted failing twice on the way there.

## 6. E5 — backfill

```
backfill from 2026-08-15: scanned=70 computed=67 no_coverage=3 levels_resolved=51
excursions: logging=on rows=70 backfilled=70 unresolved=3
```

Run against an online copy of the live store; **production was not written to**.
The 3 no-coverage rows keep their NULLs and say `resolution="none"`. 19 of 70
could not have their levels resolved from an opening decision, so their
ambiguity is not judgeable and is reported as such rather than as 0%.

## 7. E6 — the distribution the stop ruling waits on

n is the number of rows with a **measured** path. Unmeasured rows are reported
beside it and never averaged in. The ambiguous share carries its own
denominator — rows whose levels are known — and is withheld entirely when that
denominator is zero.

```
by condition               n   MAE p50      p80      p95   MFE p50      p80      p95  ambiguous
reject                    30     17.75    38.75    58.75     36.00    69.25   110.25  0.0% of 21  · 2 unmeasured
(unknown)                 10      8.00    22.50    55.25     10.00    25.25    56.00  0.0% of 4
breakout_retest            9     42.25    52.75    81.25     25.75    48.00    71.75  0.0% of 9
sweep_reclaim              7     30.00    43.25   100.75     17.50    25.00    47.25  0.0% of 3
acceptance                 5     49.50    57.50    61.50     25.00    89.75   156.75  0.0% of 5  · 1 unmeasured
reclaim                    5     44.00    50.00    77.00     16.25    19.75    28.00  0.0% of 5
hold                       1     22.50    22.50    22.50     92.00    92.00    92.00  0.0% of 1

by session                 n   MAE p50      p80      p95   MFE p50      p80      p95  ambiguous
NY                        22     31.50    61.50    81.25     25.00    66.25   137.50  0.0% of 13  · 1 unmeasured
LONDON                    19     35.75    50.00    58.75     28.00    73.00   110.25  0.0% of 16  · 2 unmeasured
ASIA                      16     21.75    42.00    57.50     20.25    36.00    69.75  0.0% of 15
(unknown)                  9     12.00    51.00    55.25     14.00    42.25    56.00  0.0% of 4
TEST-E7                    1      5.25     5.25     5.25      8.50     8.50     8.50  — (no levels)

by scenario                n   MAE p50      p80      p95   MFE p50      p80      p95  ambiguous
S1                        25     27.50    42.25    52.00     21.50    58.50   137.50  0.0% of 19  · 2 unmeasured
S2                        23     34.50    49.50    58.75     40.50    73.00    92.00  0.0% of 18
(unknown)                  7     12.00    22.50    51.00     14.00    25.25    56.00  0.0% of 2
S3                         7     36.75    77.00    81.25     16.25    25.75    50.50  0.0% of 5  · 1 unmeasured
S4                         2     52.25   100.75   100.75      6.75    23.50    23.50  0.0% of 2
off-plan                   2      6.25    55.25    55.25     10.00    42.25    42.25  0.0% of 2

by side                    n   MAE p50      p80      p95   MFE p50      p80      p95  ambiguous
SHORT                     39     31.50    49.50    61.50     28.50    69.75   137.50  0.0% of 30  · 3 unmeasured
LONG                      25     33.00    50.00    81.25     17.00    34.75    71.75  0.0% of 18
short                      2      1.25     6.00     6.00     17.00    41.50    41.50  — (no levels)
long                       1     19.25    19.25    19.25      7.00     7.00     7.00  — (no levels)
```

### 7.1 What it says about C3

C3 describes 30–80 pt pullbacks against 30–40 pt stops. The measured adverse
excursion, over 67 trades:

- A **30 pt** stop sits between the p50 and p80 of MAE for every condition —
  it is stopped out by ordinary noise more often than not.
- A **40 pt** stop clears p80 only for `reject` (38.75). For
  `breakout_retest` (52.75), `acceptance` (57.50), `reclaim` (50.00) and
  `sweep_reclaim` (43.25) it is still inside the p80.
- The p95 tail runs 52–101 pts. 589's 81.25 sits at the `breakout_retest` and
  `NY` p95 exactly — it was not an outlier, it was the tail.

**This is a description of what the tape did, not a stop ruling.** n per
condition is 1–30; only `reject` (30) is anywhere near a sample. Two of the
seven conditions have n ≤ 5. The wave's job was to make the number exist.

### 7.2 Ambiguity

**0 of 51** judgeable rows had any bar reaching both the stop and the target.
The C4 machinery is real and fixture-tested (F3), but on this corpus the case
did not occur. Stated with its n, not as a reassuring 0%.

## 8. Findings the tables surfaced

- **`side` casing is inconsistent in `trader_positions`.** `SHORT`/`LONG`
  (39/25 rows) sit beside `short`/`long` (2/1). The grouping is faithful to the
  data; the data is not consistent. Not this wave's to fix. **[A]**
- **A `TEST-E7` session row** is in the live corpus. **[A]**
- **`(unknown)` condition, n=10.** Positions whose plan, version or scenario
  could not be resolved, so the exit study cannot group them. Left unknown
  rather than inferred from side or close reason. **[A]**
- **AI-path entries do not carry stop/target to the position row.** They open
  with the levels NULL and resolve on the next tick from the opening decision,
  within a 2-minute window. 51 of 70 resolved. Threading them through the
  execution path belongs to a wave that owns entry logic. **[A]**

## 9. Tests

| id | pins | state |
|---|---|---|
| F1 | position 589 rebuilt from stored bars | **RED quoted twice, GREEN** |
| — | the 0.75 gap vs the recorded row | GREEN |
| F2 | side-aware MAE/MFE; entry bar included | GREEN |
| F3 | ambiguous bar counted; resolved against the trade | GREEN |
| — | no coverage → computed=false, not zero | GREEN |
| F4 | NULL ≠ 0 on a fresh row; a computed zero stores as 0 | GREEN |
| — | A22: close copies `pnl_corrected`, never `realized_pnl` | GREEN |
| F5 | panic / DB error / index-out-of-range → WARN, loop lives | GREEN |
| — | bare `AutoTrader` hooks are inert | GREEN |
| F6 | backfill counts, no-coverage NULLs, idempotency | GREEN |
| — | E6: unmeasured row does not drag p50 from 20 to 10 | GREEN |
| — | new position writes NULL, not the DDL default | **RED quoted twice, GREEN** |
| — | `LearningLine` prints n and excludes unmeasured | GREEN |
| F7 | `go build ./...` · `go test ./...` · vitest 39/302 · `tsc --noEmit` | GREEN |

## 10. Class 23

Every hook runs through `safeExcursion`, which recovers panics and turns errors
into WARNs. The hooks are inert without a store, a bar provider or day_plan.
Nothing in this wave can refuse an entry, move a stop or block an exit.

## 11. Cutover

Not deployed. Window per A7 (14:45–16:30 CT or after 17:10, no arms, flat).
Rebase onto the tip; full suite; five-leg gate `ready:true`; owner GO; A13
rollback named by the rev it holds; A19 all four halves including the marker
**pushed** before the lock is released; `GUIDE_BUILT_REV` in the marker.

**PROOF OWED:** the first live position's row updating bar by bar, and the E6
table re-run on the live store after the backfill runs there.
