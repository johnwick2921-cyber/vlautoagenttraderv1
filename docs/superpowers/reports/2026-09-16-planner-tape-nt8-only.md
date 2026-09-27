# Planner tape is NT8-only — dispatch 101 follow-up (2026-09-16)

**Ruling:** CTO nofx-c5 under the owner's delegation ("you are cto just approve as my guideline"):
`historical_import` rows are excluded from EVERY planner door; the chart keeps them, labelled. Follow-up
to dispatch 101 (`9e200002`, booted 14:33 CT), where guard (iii) closed the rehydrate door and A15 §8 named
the second door with counts read from the store.

## What was true before this wave [A]

`data/data.db` read-only, 2026-09-16 12:2x CT: `MNQ 12-26` 1m = 2,873 live + 25 replay + **426
`historical_import`** = 3,324 rows; `plannerCandleTapeBars = weeklyFactsTapeBars = 12,000`; the shared
reader `LastNBarsOn` filters only mixed + off-scale. So all 426 imported bars were inside the planner's
and the weekly reader's 1m tape. On any `09-26` resolve: 75,492 import rows.

## The doors (A29 — every production call site named)

| door | file:line | before | after |
|---|---|---|---|
| planner + weekly 1m tape | `trader/bars_store_depth.go` `storeBarReader` | `LastNBarsOn` | `LastNBarsFromNT8On` |
| exported planner seam | `trader/bars_store_depth.go` `BarsWithStoreDepth` | `LastNBarsOn` | `LastNBarsFromNT8On` |
| weekly bars from epoch | `trader/auto_trader_weekly.go:84` | `BarsBetweenOn` | `BarsBetweenFromNT8On` |
| weekly `Own1m` window | `trader/auto_trader_weekly.go:116` | `BarsBetweenOn` | `BarsBetweenFromNT8On` |
| POC-touch historical leg (level retirement) | `trader/auto_trader_dayplan.go:227` | `BarsBetweenOn` | `BarsBetweenFromNT8On` — **found by the lint, not by the ruling's list** |

Kept on the shared readers (imports included), by design: `BarsWithStoreDepthDisplay` (:108/:123, the
chart [O]); `bar_persist_wire.go:456` (the rehydrate read — its door refuses and COUNTS imports).

**LEFT by CTO ruling (they MEASURE, they do not decide):** `level_stats_wire.go:121` (nightly level
stats), `trade_excursion_hook.go:168` and `trade_excursion_backfill.go:120` (MAE/MFE over a trade's
window), `follow_plan_wiring.go:217` (recorded-only follow plan, [T]). A measurement over imported
minutes is a separate question from a decision over them.

**`one_setup_boot.go:174` — read the code, LEFT:** the call sits inside `BackfillOneSetupVerdicts`
(header :87–95: "D9 for the verdicts, three-state. For every episode opened at or after sinceMs with no
verdict: the LEVEL leg is judged against the last candidate-pool read before the episode's open … price
from the store's bars through the contract filter"), reading the six minutes around a PAST episode's open
(`r.OpenedAtMs-5*60_000, r.OpenedAtMs+60_000`) and writing
`store.OneSetupStamp{… Backfill: "recomputed"}` via `ts.StampOneSetup(r.ID, stamp)` at :199–200. It stamps
a verdict onto an episode that already happened; nothing at boot arms, retires or authorizes from it. The
live one-setup verdict at an open is `one_setup_wiring.go` and reads the ring, not the store.

`LastNBarsOn` / `BarsBetweenOn` are byte-identical. The new readers are theirs plus
`AND COALESCE(source,'') <> 'historical_import'`; `ImportRowsOn` is the census the boot line prints.

## Pins — all RED first (quotes)

- store `TestNT8OnlyReaderExcludesImportsTheSharedReaderReturns`: 500 live + 5 import rows NEWER than the
  live ones (a reader that merely trims the oldest could not pass) → shared 505 (premise) / NT8-only 500 /
  census 5. RED: `NT8-only reader: want 500 rows, got 505`.
- store `TestNT8OnlyRangeReaderExcludesImports`: RED `want 100, got 103`.
- trader `TestPlannerStoreReaderServesNoImportRows` — the REAL door (`&AutoTrader{store}` → `storeBarReader`,
  `currentContract` from the store's latest stamp). RED: `planner reader served 505 rows, want 500`.
  A8 mutation (shared reader put back in `storeBarReader`, build rc=0): RED again; restored, green.
- trader `TestNoPlannerDoorReadsTheSharedBarReader` — text tripwire (class-113 caveat stated) on the four
  planner files + the two door BODIES in `bars_store_depth.go`. Its first form cut the file at the display
  seam and missed `storeBarReader` (which sits after it) — the mutation showed it did not bite; rewritten by
  function body, mutation-proved: `storeBarReader( reads the shared reader`. Its first run is what found the
  POC-touch door.
- trader `TestPlannerTapeBootLineNamesTheMovedValues` (class 82: import count, tape length and baseline both
  ways, 8 vs 9 session-days measured; UNKNOWN never 0) and `TestPlannerTapeAccountingIsZeroWhenTheInputDoesNotChange`
  (E7-style: same 9-session tape both ways → `Δ+0` rows, `Δ+0.000000` baseline — the rule did not move).
- `TestEveryClaimedProductionPathHasACallSite`: `LastNBarsFromNT8On`, `BarsBetweenFromNT8On`, `ImportRowsOn`,
  `PlannerTapeBootLine`, `logPlannerTapeAccounting` registered.

Suite: store, trader, trader/ninjatrader, api ok. Guide vitest 23/23, tsc clean.

## The boot line (A11, every field read)

`🧮 planner tape [NT8-only, CTO ruling 2026-09-16] @<t>: MNQ 1m contract=<c> · import rows on contract=<n>
(excluded from every planner door) · tape NT8-only=<a> rows vs with imports=<b> rows (Δ<a−b>) · regime
baseline NT8-only=<x> vs with imports=<y> (Δ<x−y>) · chart keeps imports, labelled` — emitted beside the
📈 regime line in the dayplan boot hook; the with-imports half is measured through the shared reader in
`planner_tape_nt8_only.go` ONLY (a census, never a planner input — the lint names that file as the exception).

## A15 — what the owner will see / what moved

- Expected at the boot on `12-26`: `import rows on contract=426`, tape Δ **−426** rows — UNLESS the 426 are
  older than the newest 12,000 NT8 rows by then (they were inside on 09-16). The regime Δ is the number
  that says whether the exclusion mattered; it is printed, not predicted.
- Levels: the POC-touch leg no longer sees imported minutes, so a POC an import "touched" may un-retire
  at the next planner read. Named, not measured — no fixture can measure today's levels.
- The chart is unchanged.
- CLASS 129 appended to the checklist (the door shape, the by-body lint, the newer-than-feed fixture, the census line).

## A12 · Docs in this commit
RULEBOOK §A (the planner's tape is NT8's own), SYSTEM-MAP §1 (doors + the kept readers), Guide `status.ts`
(paragraph + the 🧮 boot line). `GUIDE_BUILT_REV` is bumped at the cutover to the merged rev (A27).

## Rollback
Go binary swap only; no migration, no flag. The readers are additive; the previous binary reads the shared readers.

## F3 · THE BOOT — c6579347, 2026-09-16 15:31:59 CT (owner-run runbook; F3 PARTIAL)

PID 2748996. `🔐 BOOT INTEGRITY OK — rev c6579347580a · built 2026-09-16T19:57:23Z · expected c6579347580a
· goldens PASS`; `/api/health` c6579347; `deploy/RELEASE=c6579347`; `nofx-bin.old.9e200002` HOLDS 9e200002
(`go version -m`). `🧯 nt8 history at subscribe: MNQ 1m..30m 2000/2000, HTF n/a`; `🧯 ring rehydrated MNQ 1m
[O]: nt8=2001 store_live=2500 store_hist=0 import=0 (refused at the door — guard iii) total=2500/2500`;
`🧯 ring rehydrate done: 9 of 17 pairs deepened, +1104`; `📼 … mismatches this process: none`; zero
`scale check SKIPPED` / `DIFFERENT PRICE SCALES` lines; 4h horizon 406/500 at 14:5x on the prior boot,
HTF `n/a` at this subscribe (lands late, as before).

**F3 PARTIAL — the `🧮 planner tape` and `📈 regime input window` lines did NOT print this boot.** Read
from timing [A]: both are emitted by the trader's `afterBackfillHook` (`auto_trader_dayplan.go:87` →
`SetAfterBackfillHook`). This boot the replay landed fast — `📦 bars: persisting` 15:32:02, the hook check
and the `🕳 bar horizon:` boot line 15:32:03 — and the trader installed the hook at 15:32:05 (its `📊 bars`
line): `afterBackfillHook.Load()` was nil, the block skipped it silently, and the R1 "📊 bars after
backfill" line is missing with them. At 14:33 the backfill waited until 14:33:11 and the trader had loaded
at 14:33:11 — the hook won by chance. **Class candidate: a hook that loses when its event beats its
installer** (silent by construction — absent, not wrong; A24). The exclusion itself is live (the planner's
reads go through `LastNBarsFromNT8On`; static pins + lint); the LIVE accounting is owed. CTO ruling (b):
no third boot for a log line — the fix (hook fires immediately if the backfill already landed; 🖥 compares
bundle rev to binary rev, not timestamps) ships as the next PR and the `🧮` proof is taken at the next
scheduled boot.

**Correction (CTO, [A]):** my first read said "dist f53f4e94 installed by the owner". Wrong. `web/dist`
serves `assets/index-C0CWYHKw.js` — the **bcd70c0d** dist (bundle `GUIDE_BUILT_REV=9e200002`, mtime
14:40:15), installed after the 14:33 boot (which served the 09-13 bundle). The **f53f4e94** dist
(`index-GYuDKImr.js`, rev c6579347, parked at `scratchpad/cc101d/web-dist-f53f4e94/`) is NOT installed
and is owed — the owner ran the binary runbook only. So the `🖥 ui … STALE` line this boot is REAL (bundle
rev 9e200002 ≠ binary c6579347) and the Guide banner shows drift until that cp runs; the timestamp-vs-rev
point is a separate small fix, not this boot's explanation.

Marker: this commit, from `~/nofx`, `deploy/RELEASE=c6579347` (written by the runbook before the kill, A19).

## FOLLOW-UP PR — the hook race and the 🖥 rule (CTO ruling (b): rides the next scheduled boot)

- `SetAfterBackfillHook` / `fireAfterBackfillHook` share one mutex with `landed` + `fired` flags: install
  after the event fires immediately, exactly once; install before fires once at the event. RED quotes:
  `hook installed AFTER the backfill landed fired 0 time(s), want exactly 1` and `want exactly one firing,
  got 2` (the old mailbox double-fired on a second landing).
- `api.UIServingBootLine(dist, binaryAt, binaryRev)` judges by rev: finds the entry bundle index.html
  references, every distinct 40-hex literal in it, and says `bundle-rev=<x> matches the binary` or
  `bundle-rev=<x> STALE — built for another rev than binary <y>`; no literal → `bundle-rev=UNKNOWN`, not
  judged; the mtime rides as `· bundle mtime predates the binary by …`. RED quotes: the 9e200002 bundle
  with a NEWER mtime under c6579347 printed no STALE; the c6579347 bundle 17 min older printed STALE. The
  timestamp-only function is deleted (class 100: no caller left); its three tests re-pointed.
- CLASS 130 appended. Wiring gate: `fireAfterBackfillHook`, `UIServingBootLine`.
- Live proof owed at the next scheduled boot: the `🧮 planner tape` line (import rows=426 on 12-26, Δ rows,
  Δ baseline), the `📈` line, the R1 line, and `🖥 … bundle-rev=<rev> matches the binary` once the f53f4e94
  dist is installed (or a later one built at the booted rev).

**STAGED AND GREEN (not booted):** PR #135 merged fast-forward → dev `7e87a375`. Clean-clone binary at that
sha: `scratchpad/cc101e/nofx/nofx-bin` — `vcs.revision=7e87a375…`, `vcs.modified=false`, dir `nofx`, md5
`03a82609ff78e88bf83abc8ecbbf7969`, 73,554,832 bytes. Rides the NEXT scheduled boot (CTO ruling (b), no
boot today); at that boot the dist built at the booted rev is installed alongside it (`GUIDE_BUILT_REV`
stamp + dist build follow the 642f8808 precedent), and the proofs are: `🧮 planner tape` (import rows=426
on 12-26, Δ rows, Δ baseline), `📈`, R1, and `🖥 … bundle-rev=<rev> matches the binary`.

## F3 · THE BOOT — 7e87a375, 2026-09-16 16:15:30 CT (owner-run: install-dist-97a1b0f3.sh then cutover-101-7e87a375.sh)

Tree recovered from the THIRD class-45 strike first (15:51 CT, 57 files; the reverted content preserved on
the local branch `junk/class45-strike3-20260916-1551`, never to be merged); `~/nofx` at `97a1b0f3 ==
origin/dev`, porcelain 0. PID 2748996 → **2814144**. Read from the log [A]:

- `🔐 BOOT INTEGRITY OK — rev 7e87a375acf7 · built 2026-09-16T20:47:45Z · expected 7e87a375acf7 · goldens PASS`;
  `/api/health` 7e87a375acf7; `deploy/RELEASE=7e87a375`; `nofx-bin.old.c6579347` HOLDS c6579347.
- **`🖥 ui: served-by=go-static build=2026-09-16T21:00:07Z bundle=index-CVZahq3s.js bundle-rev=7e87a375 matches
  the binary`** — the rev-judged line, first live print; the bundle was built 13 minutes AFTER this binary,
  which the old timestamp rule would also have passed, but for the wrong reason.
- `🧯 nt8 history at subscribe: MNQ 1m..30m 2000/2000 1h=1553 6h=273 8h=205 12h=137 1d=69 3d=23 1w=15 (2h/4h n/a)`
  — the HTF replays landed within the subscribe window this time; `🧯 ring rehydrated MNQ 1m … import=0
  (refused at the door) total=2500/2500`; `🧯 ring rehydrate done: 2 of 24 pairs deepened, +1000`; zero
  `scale check SKIPPED` / `DIFFERENT PRICE SCALES`; 4h 406 (prior read, CTO).
- **The `🧮 planner tape` / `📈` / R1 lines are NOT yet printed at 16:21 CT, and this time the cause is the
  session gate, not the hook:** their installer, `snapshotSessionProfiles()`, runs inside `runCycle`
  AFTER `cmeSessionClosedSkip()` (`trader/auto_trader_loop.go`); this boot landed inside the 16:00–17:00 CT
  daily break, so every cycle returns before reaching it (the 15:32 boot was inside RTH and reached it at
  +6 s). The hook is installed at the 17:00 CT open — and fires immediately on install, which is exactly
  the class-130 fix. **The `🧮` live proof (import rows=426 on 12-26, Δ rows, Δ baseline) is therefore due
  at ~17:00:0x CT and is read then**, appended below when it lands.

Marker: this commit, from `~/nofx`, `deploy/RELEASE=7e87a375` (runbook, before the kill); `GUIDE_BUILT_REV`
= 7e87a375 (dev 97a1b0f3, web-only; the Go binary is the 7e87a375 build, md5 03a82609…).

### The 17:00 open — the 🧮 line landed (2026-09-16 17:01:27 CT, verbatim [A])

The first cycle after the CME break reached `snapshotSessionProfiles()`, installed the hook AFTER the
backfill had landed (16:15:47), and the hook fired immediately — all three lines at once (class 130 fix,
proven live on the exact shape that lost them at 15:32):

- `📊 bars after backfill: 1w nt8_agg via 1d since 2026-06-11 (13) · 1d nt8 since 2026-06-11 (69) · 4h nt8 since 2026-06-12 (406) · 1h nt8 since 2026-06-12 (1553) · 15m nt8 since 2026-08-18 (2000) · 5m nt8 since 2026-09-07 (2000) · 1m nt8 since 2026-09-15 (2499) …`
- `📈 regime input window @17:01:27 CT: BEFORE window=7 complete session-days · baseline=0.966039 · 2001 5m rows via 5m-ring (pre-wave) · AFTER window=7 complete session-days · baseline=0.966039 · 2001 5m rows via 5m-ring-fallback · Δ+0 day(s) · cap=20 days`
- **`🧮 planner tape [NT8-only, CTO ruling 2026-09-16] @17:01:27 CT: MNQ 1m contract=MNQ 12-26 · import rows on contract=426 (excluded from every planner door) · tape NT8-only=3113 rows vs with imports=3539 rows (Δ-426) · regime baseline NT8-only=0.966039 over 7 day(s) vs with imports=0.966039 over 7 day(s) (Δ+0.000000) · chart keeps imports, labelled`**

Reading: the 426 imported rows WERE inside the planner's tape (3,539 → 3,113) and are now out; the regime
baseline did not move (Δ+0.000000) because the imports sit before the 7 complete session-days the estimator
uses — the exclusion changed the tape, not today's regime number. Named, not inferred (class 82). Levels:
the POC-touch leg no longer sees those minutes; any un-retire shows on the next planner read.
