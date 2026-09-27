# Historical backfill — pull years, not weeks (wave 101)

**Dispatch:** 101 — SELF-CONTAINED, owner's order: end-to-end. Branch `fix/historical-backfill` (claim `daa0e9ae`, NOFX_SESSION `historical-backfill-4abbb353/copilot[unlisted]`), code PR #106 merged → dev `7ad41317`. Follow-up branch `fix/historical-backfill-pins` carries the brand-scope rebaseline + this report.
**Running rev verified (measured):** `/api/health` → `{"revision":"6c96683c704f","status":"ok"}`; `/proc/3497330/exe` → `/home/hoang/nofx/nofx-bin`. Dev tip at cut: `64d75cb9`.
**A13 backup (before the first import):** `/home/hoang/nofx-backups/historical-backfill/data.db.pre-import.20260911-214126` — `PRAGMA integrity_check` = **ok**, md5 **`a032fb3491e6fdf7c360dc015234df76`**, 793 MB. Re-run this backup immediately before the first import (command in §D6).

---

## C1 — what NT8 will serve on THIS account (the answer the owner wants most)

Measured on this machine, not from vendor claims: the NT8 minute database at
`C:\Users\hoang\Documents\NinjaTrader 8\db\minute\` already holds **4.4 years of
MNQ 1m, contract by contract, 2022-04-11 → 2026-09-11**. Per contract (`.ncd`
file counts and first/last file dates — full listing committed at
`2026-09-11-historical-backfill-data/contract-inventory.txt`):

| Contract | Files | First day | Last day | Quarter covered |
|---|---|---|---|---|
| MNQ 06-22 | 53 | 2022-04-11 | 2022-06-08 | ✓ |
| MNQ 09-22 | 85 | 2022-06-08 | 2022-09-07 | ✓ |
| MNQ 12-22 | 87 | 2022-09-07 | 2022-12-09 | ✓ |
| MNQ 03-23 | 80 | 2022-12-11 | 2023-03-10 | ✓ |
| MNQ 06-23 | 80 | 2023-03-12 | 2023-06-09 | ✓ |
| MNQ 09-23 | 79 | 2023-06-11 | 2023-09-08 | ✓ |
| MNQ 12-23 | 82 | 2023-09-10 | 2023-12-08 | ✓ |
| MNQ 03-24 | 80 | 2023-12-10 | 2024-03-08 | ✓ |
| MNQ 06-24 | 85 | 2024-03-10 | 2024-06-14 | ✓ |
| MNQ 09-24 | 78 | 2024-06-16 | 2024-09-13 | ✓ |
| MNQ 12-24 | 83 | 2024-09-15 | 2024-12-13 | ✓ |
| MNQ 03-25 | 81 | 2024-12-15 | 2025-03-14 | ✓ |
| MNQ 06-25 | 84 | 2025-03-16 | 2025-06-13 | ✓ |
| MNQ 09-25 | 84 | 2025-06-15 | 2025-09-12 | ✓ |
| MNQ 12-25 | 86 | 2025-09-14 | 2025-12-14 | ✓ |
| MNQ 03-26 | 80 | 2025-12-14 | 2026-03-13 | ✓ |
| MNQ 06-26 | 78 | 2026-03-15 | 2026-06-11 | ✓ |
| MNQ 09-26 | 158 | 2026-03-13 | 2026-09-11 | live + overlap (see below) |
| MNQ 12-26 | 6 | 2026-09-06 | 2026-09-11 | the roll-incident contract |
| MNQ 03-21, 06-19, ##-## | 0 | — | — | **unavailable: empty on this account** |

**Well over one year** — the H-stop-line ("feed serves less than a year → stop
before importing") does not trigger. Two data-quality notes, handled at import
(D4), not silently: (a) **MNQ 09-26 holds files from 2026-03-13** while 06-26
ends 2026-06-11 — the folders overlap; the import pulls each NAMED contract and
the store's per-contract rows make any overlap visible instead of stitched; (b)
the C# resolver's date rule (the 09-10 roll bug, `VLContractResolver.cs:80`)
made 12-26 the early subscriber — the platform's own front month is still
09-26, which is why 09-26 carries live bars through today.

## C2 — the download path

Established: **the AddOn cannot serve an expired contract today** — the live
`bars_subscribe` path resolves through `VLInstrumentLookup.Resolve`, which
answers "the platform's front month" and nothing else. The wire holds exactly
ONE client (`tcp_server.go:59-61`), so a separate import process can never
claim the connection; the import must ride the running bot.

Built (this wave): three new wire frames — `bars_history_request` (Go→C#:
explicit contract name, timeframe, `[from,to)`), `bars_history_data` (C#→Go:
~8k-bar chunks, seq/last, the instrument's real `ContractName` echoed),
`bars_history_error` (a pull that cannot be served is ANSWERED, never silent).
C# side: `ninjascript/VLHistoryPull.cs` — `Instrument.GetInstrument(contract)`
directly (never the resolver, never a date rule), `BarsRequest(instrument,
from, to)`, **`MergePolicy.DoNotMerge` always** (a back-adjusted series is a
different scale wearing the same label — the 09-10 replay damage),
`TradingHours` CME US Index Futures ETH, request disposed on completion.
Go side: env-gated `POST /api/admin/bars/import` (authed + `HISTORICAL_IMPORT_SEAM=on`),
which streams chunks through `store.ImportBars`.

**The owner's hands, exactly:** the C# files are already copied to the AddOns
folder (`VLTraderTCPClient.cs` md5 `0a0a253f`, `VLHistoryPull.cs` md5
`30e06eb4`, verified byte-equal against the repo). In the NT8 UI:
**Control Center → (New) → NinjaScript Editor → Compile (F5)** — the Output
window must show zero errors and "Successfully compiled". Then **fully close
and restart NinjaTrader** (AddOns do not hot-reload; a restart without F5 or
F5 without restart = the old binary). Nothing needs clicking per contract — the
pull is per contract over the wire, driven from the Go side; the owner repeats
zero clicks. Wait for this lane's next message before any other step.

## C3 — the store's shape

`bars(symbol, tf, open_time_ms, o, h, l, c, v, convention, contract, source)`,
natural key `(symbol, tf, open_time_ms)` (`store/bar_history.go:22-60`),
`contract` since the roll wave, `source` since the bar-source wave, retention
per TF `tfRetentionDays` (1m = `BAR_RETENTION_DAYS`, default 90d; 3m/5m 180d;
15m/30m 365d; 1h+ forever). Today: 102,785 rows ≈ 7.8 MB (measured estimate).

Importing all 17 quarters × {1m, 5m, 15m, 1h} ≈ **1.8M rows ≈ 110–150 MB +
index ≈ 250 MB total** against the 793 MB database — trivial. What changes:
`ImportBars` is the ONLY new writer (no upsert); `PruneByTF` now exempts
`source='historical_import'` (D2's chosen retention: imported history is never
pruned — the sweep exists to bound live churn, and pulled tape is
irreplaceable); readers are unchanged: live readers keep `IsReadableSource`
(live|historical) so imported rows are invisible to every live path, and the
ring rehydrate / `/api/klines` / detectors keep the current-contract filter
(`LastNBarsOn`/`BarsBetweenOn`; the reader-filter lint still fails a new
unfiltered read).

## C4 — the seam

Already answered by the roll wave, verified for this import: **no live reader
joins across contracts today** — every live store reader asks for a contract
(`LastNBarsOn`/`BarsBetweenOn`) or `WindowContract` first, and a window across
the roll is `unrecomputable:spans_roll`, never read mixed
(`bar_contract_roll.go:230-243`; lint `trader/bar_reader_filter_lint_test.go`).
Imported bars carry OLD contracts, so the current-contract filter excludes them
automatically — the second lock is `IsReadableSource` (imported rows are not
live-readable at all). A backtest continuous series is built deliberately via
`store.BuildContinuous` with an **explicit per-seam basis in points** (missing
or zero basis = refusal), labelled `continuous:adjusted`, and the store
REFUSES to persist it — the accidental join cannot even be written. **What this
wave built: per-contract raw access + the labelled builder; no silent join
anywhere.**

## C5 — what a backtest needs

- **Gap fade (daily opens/closes):** years of daily already present in the store (1d rows back to 2019) — answerable today, no import needed.
- **Zone / level fade, multi-TF agreement (1m/5m/15m/1h intraday):** needs this import — answerable once §D5 runs.
- **Forming-candle test (1m):** needs this import — answerable once §D5 runs.
- The import plan is the full 17-quarter set × {1m, 5m, 15m, 1h} (~1.8M rows).

## D — the work as shipped

- **D1** backup quoted above (A13).
- **D2** retention: exemption flag, not a raised window — `PruneByTF` skips `source='historical_import'`; resolved before/after: 1m=90d/5m=180d/15m=365d/1h=forever unchanged, plus "imported history is never pruned" now explicit and pinned (E4).
- **D3** import per contract stamped: contract from the wire's echoed `ContractName` (never a date); `source='historical_import'` (a third feed, distinct from the live-path `historical`); `ImportBars` is `ON CONFLICT DO NOTHING` — a key collision keeps the existing row's values and counts the skip. No upsert exists in the import door (E1).
- **D4** seam: per-contract rows + current-contract filter + `BuildContinuous` labelled builder (E3); a bar whose 5m+ bucket spans a roll is on the old contract's scale, kept per contract, never mixed.
- **D5** three-state import report: `imported / skipped:<why> / unavailable:<reason>` per contract × tf — the importer returns it and logs it (A9: contract, row count, first and last bar, skipped). **Runs after the owner's F5+restart + GO (next message).**
- **D6** verify plan (runs with D5): per contract — row count, first/last bar, three-bar spot-check against NT8's own chart, and a checksum over the pre-existing rows before/after (no live row's values change — pinned by E1).
- **D7** Go changed → boot required: staged, **NOT booted** (A3: owner GO). The boot line adds `📚 history held: <contract>·<tf>=n [first→last] …` (read from the store, never a literal) beside the existing `📊 bars` line.

## E — tests

- **E1 overwrite pin** `TestImportBarsNeverOverwritesExistingRow` — mutation: `DO NOTHING → DO UPDATE SET c=excluded.c` → **KILLED** (`--- FAIL: TestImportBarsNeverOverwritesExistingRow`). Quote, passing after restore.
- **E2 stamp pin** `TestImportBarsRefusesUnstampedRows` — empty contract and wrong source refused, nothing written; source-check mutation → **KILLED**.
- **E3 seam pin** `TestImportSeamContractFilterAndContinuousLabel` — per-contract filter returns exactly the asked contract; adjusted series labelled `continuous:adjusted` and refused by BOTH writers.
- **E4 prune pin** `TestPruneNeverDeletesImportedHistory` — imported rows survive the sweep, live rows past cutoff don't; mutation (exemption dropped) → **KILLED**.
- Wire roundtrip `TestRoundTrip_BarsHistoryFrames` for the three new frames. E5 A29 grep: the import path's only call sites are the seam itself (handler → importer) — production call sites: **0**. A28: one `time.Now()` per import run for elapsed; bar timestamps from the wire.
- **E6** full Go suite green at the merged head · goldens PASS · vitest **427/427** · tsc 0. The brand-scope byte-pins for the three intentionally-changed files were re-baselined with the wave named in the header comment (`fix/historical-backfill-pins`) — the pre-existing failure set is otherwise unchanged.

## A15 — what a backtest still cannot do after this wave

Fills are assumed; round 10 measured ~66% of real passive fills go adverse
immediately, so a fade backtest overstates. The store holds OHLCV only — no
intrabar path, no bid/ask, no tick sequence, so forming-candle questions stay
limited to closed bars (round 13f's constraint). A continuous series needs a
measured basis per seam and remains `continuous:adjusted`, never raw. NT8's
own merge/back-adjust policy is filed with the AddOn wave.

## Rollback

- **DB:** every import writes only new `historical_import` rows; a bad pull is
  removed by `DELETE … WHERE source='historical_import' AND contract=<name>`
  (history is never dropped wholesale); the A13 backup restores anything else.
- **Binary:** the A4 clean-clone build at `7ad41317` (md5 `b1b2c5d2`,
  `vcs.modified=false`) is the staged replacement; the running binary is
  untouched until owner GO. C#: restore the `.bak-20260911-*` files in the
  AddOns folder + F5 + restart.

## Sha-pinned

Report committed at the merge head; blob byte size == `git ls-tree` (verified
after commit). Import-run results (D5/D6 three-state tables) append to this
report at the gate, after the owner's F5+restart + GO.

---

## APPENDIX — THE CUTOVER AND IMPORT RUN (2026-09-12, executed on owner GO)

**Cutover (A19):** five-leg gate 5/5 (DB OPEN=0 · armed non-terminal=0 · API `[]` ·
NT8 snapshots both accounts count=0, 00:16:38 · no planner read in flight) →
`deploy/RELEASE` + `HISTORICAL_IMPORT_SEAM=on` written BEFORE the kill → binary
swap (`nofx-bin.old.6c96683c` = rollback) → `kill -9 3497330` → systemd
(`Restart=on-failure`, verified) relaunched PID 3671783 →
**`🔐 BOOT INTEGRITY OK — rev 400ea26c12c8 · built 2026-09-12T05:19:33Z · expected 400ea26c12c8 · goldens PASS`**
(A4 clean-clone build: `vcs.modified=false`, md5 `93e8bd17`). Marker `379713e3`
(RELEASE + GUIDE_BUILT_REV → 400ea26c, five references) pushed before the lock
released. Boot notes: the `📚 history held:` census line prints at the FIRST
cycle (Sunday 17:00 CT) together with the other per-trader boot lines — market
closed Saturday; the CLOCK CRITICAL boot line is the stale-feed reading before
the first bar (system time verified correct) and is log-only.

**A13 pre-import backup:** `~/nofx-backups/historical-backfill/data.db.pre-import.20260912-002200`
— integrity **ok**, md5 **`f1ab180acd549d22cc200e3ea0306280`**.

**D5 — three-state result** (raw responses:
`2026-09-11-historical-backfill-data/import-results.jsonl`):

| Contract | 1m | 5m | 15m | 1h | skips (where) |
|---|---|---|---|---|---|
| MNQ 06-22 | 51,779 | 10,356 | 3,452 | 863 | 0 |
| MNQ 09-22 | 85,716 | 17,205 | 5,736 | 1,434 | 0 |
| MNQ 12-22 | 85,264 | 17,081 | 5,695 | 1,425 | 0 |
| MNQ 03-23 | 78,178 | 15,636 | 5,212 | 1,303 | 0 |
| MNQ 06-23 | 82,095 | 16,419 | 5,473 | 1,369 | 0 |
| MNQ 09-23 | 82,093 | 16,419 | 5,473 | 1,369 | 0 |
| MNQ 12-23 | 82,332 | 16,467 | 5,489 | 1,373 | 0 |
| MNQ 03-24 | 79,551 | 15,911 | 5,304 | 1,326 | 0 |
| MNQ 06-24 | 81,155 | 16,231 | 5,411 | 1,353 | 0 |
| MNQ 09-24 | 76,302 | 15,265 | 5,089 | 1,273 | 0 |
| MNQ 12-24 | 76,810 | 15,363 | 5,121 | 1,281 | 0 |
| MNQ 03-25 | 71,967 | 14,395 | 4,799 | 1,201 | 0 |
| MNQ 06-25 | 75,655 | 15,132 | 5,044 | 1,261 | 0 |
| MNQ 09-25 | 77,715 | 15,543 | 5,181 | 1,296 | 0 |
| MNQ 12-25 | 77,549 | 15,510 | 5,170 | 1,294 | 0 |
| MNQ 03-26 | 73,743 | 14,749 | 4,917 | 1,230 | 0 |
| MNQ 06-26 | 77,955 | 15,591 | 5,197 | 832 | 468 (1h — live-era rows kept) |
| MNQ 09-26 | 75,492 | 16,133 | 4,065 | 66 | 34,983 / 8,001 / 4,072 / 1,969 (live-era rows kept) |
| MNQ 12-26 | 4 | 4 | 4 | 4 | 5,164 / 1,234 / 411 / 100 (live-era rows kept) |
| MNQ 03-21, MNQ 06-19 | 0 | 0 | 0 | 0 | **zero bars served — empty on this account (named, not silent)** |

Totals: **1,784,150 imported** · ~56,402 skipped (all collisions with existing
rows — kept, counted) · 0 unavailable-with-error. Store arithmetic exact:
102,785 pre-existing + 1,784,150 = 1,886,935 rows.

**D6 — verification (not the tool's message):**
- Store per-contract×tf counts match the API's per-run numbers exactly.
- **No existing row's values changed:** sha256 over all 102,785 non-import rows,
  pre-import backup vs live DB — identical (`66523b63…`).
- **No cross-contract timestamp collisions** among imported rows (the seam is
  clean by construction; readers stay per-contract via `BarsBetweenOn`).
- Spot-check (3 bars, MNQ 06-22 1m, ids in store): OHLC relations hold, prices
  on the April-2022 MNQ scale (14,209–14,216). Visual against the NT8 chart is
  the owner's one-minute check; the skip pattern above is the overlap proof.
- Surprises, recorded: the feed served MNQ 09-26 from **2026-04-30** (not
  03-13 as the folder listing suggested) and up to 2026-09-02; MNQ 12-26 served
  09-06→09-11; the overlapping windows contain NO identical timestamps. The
  `📚 history held:` boot line fires at the first cycle (Sunday reopen) — the
  live proof rides the Sunday 17:00 CT boot census.
