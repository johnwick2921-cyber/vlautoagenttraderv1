# Bar-source wave — one resolver, every timeframe persisted, and a calendar that is not ours

**Dispatch:** BAR-SOURCE WAVE, owner hoang, 2026-09-02. Audit basis `2026-09-02-bar-source-audit.md` @ 593dcf9e.
**Base:** dev at merge. **Live rev at build:** `0465a10b`. Worktree `~/nofx-bars`, no lock held during the build.
**Tiers:** [A] verified directly · [B] inferred · [C] speculation.

## 0. The measurement that set the shape (A23)

Measured on the LIVE cache of the running binary through the futures klines path [A]:

| TF | bars | range |
|---|---|---|
| 1w | **383** | 2019-05-03 → 2026-08-28 |
| 1d | 1500 | 2020-11-11 → 2026-09-01 |
| 4h | 1500 | 2025-09-11 → today |
| 1h | 1500 | 2026-06-03 → today |
| 15m | 1500 | 2026-08-11 → today |
| 5m | 1500 | 2026-08-26 → today |
| 1m | 1500 | 2026-09-01 → today |

Against a persisted `bars` table holding **1m only**, from 2026-08-19 [A][DB]. The weekly reader
therefore saw 2 completed weeks against a ≥4 guard and rendered "WEEKLY thin · low" beside 383
unread weekly bars. The audit's central claim is confirmed and quantified.

**And the surprise that changed the design.** NT8's native weekly bars are stamped [A]:

```
open 2026-08-21 00:00 (Fri)  close 2026-08-27 23:59 (Thu)
open 2026-08-28 00:00 (Fri)  close 2026-09-03 23:59 (Thu)
```

**Friday → Thursday.** Our weekly vocabulary is Monday-governed everywhere (`weekStartMonday`;
"Sunday 17:00 CT first print"; PWH/PWL from the prior Monday week). The dispatch's "nt8-native
first" would have bucketed a Friday-stamped bar into the wrong Monday and shifted every weekly
open, PWH and PWL by three days — silently wrong data replacing an honestly-labelled *thin*.
Native **1d/4h/1h are clean** (calendar-day and hour-aligned, verified) [A].

**Owner ruling (2026-09-02):** exclude native 1w from the weekly ladder (1d → 1m); persist it
anyway, labelled `convention=fri_thu`, research only.

## 1. What shipped (file:line)

| Item | Location |
|---|---|
| The one resolver | `market/bar_resolver.go` — `CompletedBars`, `CompletedBar`, `BarSeries{Source,FromTF,EarliestMs}` |
| Ladder as DATA | `barLadder` (per-TF chain) + `LadderFor` / `LadderTFs`; `tfMinutes` is the single duration table |
| The exclusion, with its reason | `ladderExclusions["1w:1w"]` + `ExcludedNative()` — an omission without a reason is indistinguishable from an oversight |
| Repaint law, one chokepoint | `dropForming()` — every rung passes through it; a forming bar is never returned |
| Stamp guard | `StampAligned()` — catches the same mismatch class on any TF the provider changes underneath us |
| Stamp convention label | `market.StampConvention()` → `epoch_floor` / `fri_thu` |
| Persistence of every TF | `store/bar_history.go` — removed `if r.TF != "1m" { continue }`; `convention` column added idempotently |
| Per-TF retention | `store/bar_history.go` `tfRetentionDays` + `RetentionDaysFor` + `PruneByTF`; wired at `trader/ninjatrader/bar_persist_wire.go` (`pruneOnce`) |
| Integrity check | now REPORTS the tf set instead of asserting `{1m}` |
| Weekly reader re-pointed | `trader/auto_trader_weekly.go` — `weeklyDailyBars()` resolves "1w"; `runWeeklyRead` uses it and logs source + completed-week count; `barResolver()` wires NT8 native + our 1m |
| Boot line | `trader/bar_source_boot.go` `BarSourceBootLine` — every value READ from the resolver (A24) |
| Research export | `cmd/bars-export/main.go` |
| Guide | `web/src/guide/content/levels.ts` (new section), `status.ts` (boot ledger line) |
| Checklist | class **45** (44 was the highest at merge) |

**Consumers re-pointed: exactly one — the weekly reader.** The audit's census found every other
consumer already reads native per-TF bars or genuinely needs 1m intrabar resolution. Re-pointing
them would have been churn without benefit, so they were left alone (stop-line honoured: no
change to 1m-native consumers, no prompt changes, no grader changes, no weekly SIGNAL change).

## 2. Retention — the trap inside the fix

The old prune was `DELETE FROM bars WHERE open_time_ms < cutoff`, TF-blind, at
`BAR_RETENTION_DAYS` = 90. Persisting 383 weekly bars back to 2019 and then running that prune
would have deleted them on the first nightly pass. Retention is now per-TF:

| TF | retention | rationale |
|---|---|---|
| 1m | `BAR_RETENTION_DAYS` (90) | bulky and re-fetchable; the existing knob keeps its meaning |
| 3m, 5m | 180d | |
| 15m, 30m | 365d | |
| 1h and coarser | **forever** | tiny and irreplaceable |

**Storage projection** (measured base 23,470 rows = 1.34 MB ≈ 60 B/row, 2 symbols):
1m 90d ≈ 261k rows ≈ 16 MB · 5m 180d ≈ 104k ≈ 6 MB · 15m 365d ≈ 70k ≈ 4 MB · 1h and coarser
kept forever ≈ 90k ≈ 5 MB → **≈ 31 MB** against a 634 MB database. One-off backfill of the
current cache: ≈ 15,800 rows ≈ 0.95 MB.

## 3. Tests

**F1 (`TestBarSourcePinWeeklyThin`)** — the red and the green in one run, because the resolver
did not exist before the wave so a pre-wave run is a compile error, not evidence:

```
F1: own-1m gave 2 completed weeks; the resolver gives 6 from nt8_agg/1d
--- PASS
```

The live red is stronger and independent: the running binary renders "WEEKLY thin · low" today.

Others: `TestBarResolverLadderIsData` · `TestBarResolverPrefersNative` ·
`TestBarResolverFallsBackToNextNativeNotOwn1m` (asserts own-1m is NEVER reached when native
dailies exist) · `TestBarResolverLastRungOwn1m` · `TestBarResolverNeverReturnsFormingBar` (F2
repaint law, both APIs) · `TestAggregateStampConventionParity` (F5, class-7 pin) ·
`TestWeeklyLadderExcludesNative1w` and `TestResolverIgnoresNative1wEvenWhenPresent` (the A23
finding, pinned so a later wave cannot "restore" nt8-first for weekly) ·
`TestStampAlignedCatchesOffsetSeries` · `TestBarInsertAndDedup` (rewritten to the new storage
ruling) · `TestPruneByTFKeepsCoarseHistory` (F4) · `TestWeeklyDocAndWatchShareOneSource` (F3) ·
`TestBarSourceBootLineReadsRealValues` (A24).

Full `go test ./...` green · `go build ./...` clean · vitest guide 10/10 · `tsc --noEmit` clean.

## 4. Export (F6) — run against the LIVE store, read-only

```
tf        rows  earliest   → latest      convention   file
1w           0  -            -           -            (skipped — nothing persisted)
1d           0  -            -           -            (skipped — nothing persisted)
…
```

Correct and expected: the running binary is still `0465a10b`, which persists 1m only. **The
export fills after this wave boots and the backfill writes every cached TF.** Verified the run
did NOT alter the live schema (`pragma table_info(bars)` unchanged, no `convention` column) and
the bot stayed clean (0 `[ERRO]`) — `store.New` does not invoke `BarHistory.Migrate` [A].

## 5. Cutover

_(filled at swap time — five-leg gate, then the boot checklist with the real 📊 line)_

## 6. What the owner will still see wrong (A15)

- **The weekly card stays "thin" until this boots.** Nothing is fixed on the running binary.
- **Deep history is one restart away from being permanent, but not yet.** The first boot backfills
  the cache into the table; until then a restart still loses everything but 1m.
- **Native 1w will sit in the table unread.** By ruling. It is labelled `convention=fri_thu` and
  the export prints a warning next to it. A research reader who ignores the label will get weeks
  that are not our weeks.
- **The weekly bias SIGNAL is unchanged and still shadow/WARN.** This wave fixed the DATA. Do not
  read a fuller weekly card as the signal having been promoted.
- **Other consumers still aggregate their own way.** Four hand-rolled 1m→TF aggregators remain
  (`AggregateBars`, `StructureAggregateToMinutes`, `aggregateBars`, `DailySessionBars`). They are
  correct for their callers and were left alone; the resolver does not yet own them.
- **`cmd/bars-export` opens the store read-write** (GORM) though it issues no writes. A strictly
  read-only handle would be better.
- **The 2026-09-02 main-tree corruption was a VS Code save**, not an agent and not git — stale
  editor buffers from three different vintages written over the working tree at 08:46. The lock
  and the worktree law cannot prevent that, because an editor is not an agent.

## 7. Rollback

```
cp nofx-bin.prev.boot nofx-bin && echo <previous rev> > deploy/RELEASE && kill -9 <MainPID>
```

The `convention` column and any newly persisted coarse rows are additive: the previous binary
lists explicit columns on insert and reads only `tf='1m'`, so a rollback needs no data undo.

---

## 8. Cutover record — 17:51 CT 2026-09-02 (owner GO)

Gate re-quoted at 17:47:49 → **red on leg 5** (an ASIA read in flight); held per A6; re-quoted
17:50:59 → `ready: True`, all five legs. Swap 17:51:09 (`status=9/KILL`). Boot 17:51:14, **PID
585014**, `🔐 BOOT INTEGRITY OK — rev 4d159022c114 · built 2026-09-02T17:30:48Z · expected
4d159022c114 · goldens PASS`, **0 `[ERRO]`**, boot sweep `cancelled 0 pre-boot arm(s)`. Every
prior wave's ledger line survived (27 · 34+38 · 39 · root-fix A · 44 · 33 · 36 · 0B · 40 · 41 ·
root-fix B).

### The result: 2 symbol×tf → 14, and 22 YEARS of weekly

`📦 bars: persisting 14 symbol×tf retention=90d rows=48017 (backfilled 27988)` at 17:51:29 —
27,988 rows written in 15 s. `✅ bars integrity OK: dups=0 total=48017`.

Persisted state, measured [A][DB] `SELECT tf,COUNT(*),MIN,MAX,convention FROM bars GROUP BY tf`:

| tf | rows | earliest | convention |
|---|---|---|---|
| 1w | 1,331 | **2004-12-03** | **fri_thu** |
| 3d | 2,191 | 2004-12-10 | epoch_floor |
| 1d | 3,895 | **2018-12-05** | epoch_floor |
| 8h | 3,998 | 2024-02-01 | epoch_floor |
| 6h | 3,997 | 2024-09-25 | epoch_floor |
| 4h | 3,997 | 2025-05-18 | epoch_floor |
| 2h | 3,998 | 2026-01-12 | epoch_floor |
| 1h | 3,998 | 2026-05-04 | epoch_floor |
| 30m | 3,998 | 2026-07-03 | epoch_floor |
| 15m | 3,998 | 2026-08-04 | epoch_floor |
| 5m | 3,998 | 2026-08-24 | epoch_floor |
| 3m | 3,999 | 2026-08-27 | epoch_floor |
| 1m | 24,028 | 2026-08-19 | epoch_floor |

**~4× deeper than the pre-build estimate.** The API probe was capped at `limit=2000`; the cache
actually holds ~4,000 bars per TF. Weekly reaches 2004 and daily 2018 — this morning the table
held two weeks of 1m and nothing else. The convention labelling is live and correct: 1w is the
only `fri_thu` series, exactly as ruled.

### A15 addendum — the boot line was honest but misleading at that instant

`📊 bars:` printed `own1m via 1m` for EVERY TF. That was TRUE when it ran: the boot line fires
before the AddOn replays `bars_historical`, so the BarCache was still cold and the resolver
correctly fell to its last rung. It is a true statement about that instant and a misleading
report of capability — a reader would conclude the resolver cannot reach NT8. **Defect, not a
lie:** the line should run after the first backfill, or state `cache cold — replay pending`.
Logged here rather than hot-patched. Everything else on the line (ladder, exclusion, retention)
was correct.
