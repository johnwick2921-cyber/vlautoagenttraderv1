# One contract, two sources — Section G report (bar-source wave)

**Owner ruling 2026-09-10 22:5x CT** ("ahead of everything") · SIM-only, MNQ, one contract
**Branch:** `fix/bar-source` · **claim:** `barsource-554049f5/nofx-d7[09b1a9]` @ `edd483ef` · base `33fee48e` (dev tip at accept; dev has not moved since)
**Head at report:** see the marker · **Suites:** Go 31 ok / 0 fail / 0 build-fail at `7c344804` (bs3, 23:27–23:3x CDT); store + trader/ninjatrader re-run green at `4a95e023`; merged-head run recorded in the marker · vitest/tsc in the marker
**Specs built from:** `git log -1 -- docs/superpowers/reports/2026-09-10-contract-roll.md` → `1cad9213` (unchanged since)

---

## CORRECTION FIRST (2026-09-11 00:2x–00:4x CT) — THE PREMISE WAS WRONG; THE WAVE IS WITHDRAWN IN PART

**NT8 had not rolled.** Its front month, its charts and its fills were on
**MNQ 09-26**. The AddOn's date rule had put bars AND orders on **MNQ 12-26**.
The ~290 points this wave called "one contract, two sources" was the
**Sep/Dec basis** — the December request's history served at September's
prices under the December name (NT8 serves the prior contract's history before
its own rollover), against December's live ticks. Carry at 2026 rates on
~29,000 over a quarter is ~275–290 points; my "carry implies ~90" was wrong
arithmetic and the tell I read past.

**Evidence [A], NT8's own log** (`log.20260910.00000.en.txt`), position 605:
`23:04:03 Order='0f216950…/Sim101' Name='be8d5968…' New state='Filled'
Instrument='MNQ 12-26' Action='Sell short' Fill price=29361.25`; bracket sl
29478 / tp 29087.25 both `Instrument='MNQ 12-26'`. 299 `MNQ 09-26` order lines
that day before 19:00 CT, 16 `MNQ 12-26` after. No Go log could show this —
the wire frames carry no instrument (roll report C5).

**The fix is C#, one door** (`ed6bac8b` + `c2eef211`, compiled and running
since the owner's 00:40 restart): `VLInstrumentLookup` resolves NT8's rolling
name `MNQ ##-##`, asks its `MasterInstrument.GetNextExpiry` for the platform's
front month, and resolves the CONCRETE contract — for bars, orders,
`close_position` and `place_protective_stop` alike. Verified 00:41 CT, all
three the same: NT8 log `=> MNQ 09-26 (rolling->MNQ 09-26)` · ACK
`instrument_info MNQ (MNQ 09-26)` · live fact 16639341 `MNQ 09-26` 29171.25;
replay fact same minute 29172.25. The rolling instrument itself returns zero
bars (00:33 CT, every timeframe) — the concrete resolution is required, not
optional. Every `bars_historical`/`bar_update` frame now carries `contract`
(Go reads it next wave).

**Disposition of this wave — CORRECTED 2026-09-11 02:2x CT (cleanup batch 2).**
abc420f8 was rolled back to 0070fc79 on the owner's order at 00:45. But the
wave's Go code had already been merged to dev at `0864db9e` (00:4x), and every
binary built from dev since — **06ccaf48 (booted 01:45) and f42e39aa — carries
ALL of it, D4 and D5 included, and runs it.** The earlier text of this
paragraph said D4/D5 were "withdrawn"; that described an intent, not the
binary, and a report that says withdrawn while the binary runs the code is how
the next reader gets misled. What actually shipped and runs: D1 `bars.source`,
D2 the live-wins upsert, D3 the ring's live-over-replay merge, **D4 the
scale-mismatch detector, D5 the replay hold,** D7 readers excluding `mixed` and
`replay:off-scale`. Since the 01:45 boot the hold has released 138 rows as
`historical` — all on the platform's contract, same scale as live, correct.
**Owner ruling 2026-09-11: keep the code; record it.** The two defects found on
the 00:15 boot (the 20×-median-body condition blinding higher timeframes; a
re-seed clearing the per-seed verdict) are therefore LIVE DEFECTS in running
code, harmless while replay = live, and OWED to a named wave — not withdrawn.
D6's measured labels were superseded by the owner-authorised store repair
(next section). Class 118 stands as written about the misread cause.

### The store, repaired under three authorisations (all WHERE-scoped, values untouched, backups quoted)

| write | backup (integrity ok) | md5 | count |
|---|---|---|---|
| `contract='MNQ 09-26' WHERE symbol='MNQ' AND contract='MNQ 12-26' AND open_time_ms < 1789092903000` (21:15:03 ACK); ES same | `pre-relabel-20260911-002105` | `84bbe8b21ac7d052bbbb0147e439505f` | MNQ 24,783 · ES 26,373 (all tfs) · 0 still Dec in the key set after |
| `DELETE WHERE source='historical'` (rows abc420f8 released) | same | same | 60 (MNQ 46, ES 14); non-historical SUM(o+h+l+c+v) 39,165,277,095.0 before = total after |
| `DELETE WHERE source='replay:off-scale'` | `pre-offscale-delete-20260911-002622` | `df7fd8a1e3e0c7b5e4346db529f55bb8` | 38 (MNQ 5, ES 33); others' sum 39,164,888,108.5 unchanged |

The relabel reverses the 22:39 ruling that kept the replay's `12-26` label on
pre-ACK rows. In the owner's words for the marker: *the relabel was kept on
the owner's ruling; it was wrong — September values under a December label is
how the ring got two scales.* Under the corrected premise the relabel is also
simply true: every bar received before the 21:15:03 ACK was a September bar.
Not reached by the WHERE, for a separate call: MNQ 1m rowids 464739/464744 and
ES 464740/464741 (21:16–21:17) open after the ACK, stay `12-26`, and carry
September opens under December closes.

### The two abc420f8 failures, verbatim — both "a verification whose own precondition disappeared"

1. **The 20×-median-body condition blinded the higher timeframes.** Boot line
   00:15:29: `mismatches this process: 1m@00:15:21 Δ=296.50 … 15m … 3m … 5m`
   — and, the same second, `📼 replay verified on the live scale: MNQ 30m —
   1998 row(s) released`, then 1h, 2h, 4h, 6h, 8h, 12h, 1d, 3d, 1w. A 30m
   bar's median body ×20 exceeds a 296-pt gap. The condition was added to stop
   a fixture false positive (`TestBarCache_RingBound`, a 1-pt move at price
   100) and was never measured against the gap it guarded.
2. **The re-seed cleared the per-seed verdict and the re-check had nothing to
   compare.** `1m@00:15:21 Δ=296.50 (replay 29159.50 vs live 29456.00, 2000
   dropped)` — then `00:15:50 📼 replay verified on the live scale: MNQ 1m —
   1998 row(s) released`. Sequence: mismatch → seed dropped → ring refilled
   from the store's live rows (`1 → 2060`) → a second `bars_historical`
   re-armed the check (`delete(c.liveSeen, key)`, `delete(c.seedOffScale,
   key)`) → `mergeSeedKeepingLive` dropped every colliding replay bar → the
   next live bar's `last` search found no historical bar → `return b, false`
   → `SeedVerdict = (checked, !offScale)` → released. Per-seed arming was
   itself the fix for a mutation survivor; ES 1m, whose re-seed did not
   collide the same way, discarded correctly at 00:15:41.

Net damage 60 `historical` rows, all labelled, all deleted (table above).

### The three findings the owner ordered into the record, no softening

**1. `vet-06-risk.md:255` named this window five days early and was not acted
on.** dev `2a66d91c`, 2026-09-05:

> "The uncovered window is **19:00 CT 09-10 → 09-14: orders on DEC26, bars
> and gate on SEP26**, and the gate's 3-day window not yet open. **What I
> would refuse:** no new entries from the Thu 09-10 17:00 CT open until a
> human has confirmed in the log that the bars ACK carries 'MNQ 12-26' …"

19:00 CT flip — as written. Order 152 on December at 19:11 — as written.
Position 605 on December at 23:04. The refusal was specified and never
installed; the report was merged and nothing read it back as a rule. That is
the class (119): a review that predicts an incident and is not converted into
a gate.

**2. The June procedure's premise IS the defect.** `docs/CONTRACT-ROLL.md`
(`17cb3cec`, 2026-06-12, "June→September roll postmortem") prescribed: "if
today ≥ roll date [expiry − 8 days], a fresh NT8 + bot restart lands on the
new quarter automatically." It takes the AddOn's date for the platform's roll.
Followed on 2026-09-11 it would have put the bot further onto December.
Rewritten in this wave: the roll is whatever NT8's front month says, confirmed
in the log — three lines naming one contract, matching the Control Center —
before any entry.

**3. The June plan's refutation is refuted by tonight's log.** Side by side:

> plan `2026-05-22-nq-databento-ninjatrader.md:245` (2026-06-02): "**`GetNextExpiry`
> is UNBOOTSTRAPPABLE on Tradovate.** It needs a `MasterInstrument`, which is
> obtained only from a qualified/continuous `GetInstrument` — and **Tradovate
> has no continuous contracts** → null → chicken-and-egg."

> NT8 log 2026-09-11 00:40:29: `VLBarsSubscriptionManager: reconnect resolved
> MNQ -> MNQ ##-## => MNQ 09-26 (rolling->MNQ 09-26)`

The rolling name resolves; its master answers `GetNextExpiry`; the concrete
contract resolves. The refutation was recorded as settled and closed the
correct path for a quarter. The resolver's own June comment even named
`MasterInstrument.GetNextExpiry` as the "later phase" — for energy and
metals only.

---

## THE ORIGINAL REPORT FOLLOWS, AS WRITTEN AT 23:4x CT 09-10, ITS PREMISE NOW KNOWN TO BE WRONG

## THE FINDING (the roll boot's, restated with its evidence) [A]

The roll wave separated MNQ 09-26 from MNQ 12-26 correctly and booted at
22:39:07 CT into a chart still discontinuous. Research archive, the 22:37 CT
one-minute bar, same subscription, same label:

| fact | frame | open | close |
|---|---|---|---|
| 16516009 | `bar_update` (live) | 29360.00 | **29358.25** |
| 16518205 | `bars_historical` (replay) | 29068.75 | **29068.25** |

NT8's replay serves the SAME contract ~290 points (1.0%) below Tradovate's live
feed. ES the same: replay ~7600 vs live ~7666 (0.86%). A contract column cannot
separate one contract from itself. The 292-point "roll step" the roll wave
measured at 21:15 was two things at once: a real roll AND the replay-vs-live
scale, which is why "Sep/Dec basis" never squared with carry (~90 pts). I read
past that tell; it is recorded in the roll report's premise section.

**Damage, repaired before this wave:** the pre-existing unconditional upsert let
the 22:39 replay overwrite 186 live rows — restored from the 22:15 backup under
two authorisations (markers `ac76b47b` 54 rows, `33fee48e` 132 rows; 0 rows
differ from the backup afterwards).

## C — what the code did before this wave [A]

- **C1 — the upsert was unconditional.** `store/bar_history.go::InsertBars`:
  `ON CONFLICT(symbol,tf,open_time_ms) DO UPDATE SET o=…,h=…,l=…,c=…,v=…`. Any
  later write replaced any earlier one. Harmless while replay and live shared a
  scale; destructive once they did not.
- **C2 — the ring's merge said "incoming is freshest".** `BarCache.SeedHistorical`
  → `mergeBarsByTime(existing, bars)`: on a same-time collision the incoming
  replay won. Correct for a live update over a stale seed, wrong for a replay
  over a bar that traded.
- **C3 — the persister discarded the one bit that named the feed.** The
  callback has received `historical bool` since 2026-08-28 (BAR-TRUTH) and used
  it only for the close-stamp → open-stamp conversion. The row carried no
  source.
- **C4 — the boot minute was a mixed bar and nothing said so.** The AddOn's
  first `bar_update` after a replay copies its open from its own historical
  series: 22:39 read open 29068.25 close 29355.25 — the exact shape 21:15 had,
  which the roll wave (rightly, for 21:15) attributed to the roll.
- **C5 — the second door (found while building D).** For a minute NO live bar
  ever wrote — the restart gap, the seconds after a reconnect — the replay's
  row is the only row. No precedence rule touches it, and the next boot's
  rehydrate (`LastNBarsOn`) reads it back into the ring as its oldest bars.
  Measured on the live DB (read-only, 23:3x CDT): **98 such rows already exist**
  from the night's two boots — see D6.
- **C6 — the reader had one filter (contract) and needed two.** `LastNBarsOn` /
  `BarsBetweenOn` filtered to the current contract and would hand a reader
  every row on it, whichever feed wrote it.

## D — what shipped

- **D1 — `bars.source` (`live` | `historical` | `mixed` | `replay:off-scale`),**
  indexed `(symbol, tf, source, open_time_ms)`. `InsertBars` refuses a row that
  does not name its feed (and refuses `replay:off-scale` from any caller — only
  the migration writes it). `Bar.Source` on the ring, Go-side only (`json:"-"`;
  the AddOn never sends it).
- **D2 — the upsert rule.** `… DO UPDATE SET … WHERE NOT (bars.source IN
  ('live','mixed') AND excluded.source = 'historical')`. Live overwrites
  anything; historical fills only what live never wrote; mixed is kept over a
  later replay (it is the seam's evidence); off-scale is overwritten by a live
  bar or a released replay (the repair path).
- **D3 — the ring rule.** `mergeSeedKeepingLive`: an existing live/mixed bar is
  kept over an incoming historical one. Seed stamps historical; Upsert stamps
  live.
- **D4 — the boot minute is never read as a bar.** `detectScaleMismatch`, on the
  FIRST live bar after EACH seed (per-seed re-arm: the first draft armed once
  per process and a mutation survived on a mid-session reconnect): if the live
  close differs from the last replay close by more than **both** 0.5% of price
  AND 20× the seed's median body (the percent alone fired on a 1-pt move in a
  fixture priced at 100), the seed is dropped from the ring, the straddling bar
  is labelled mixed with its values untouched, the persist wire refills the
  ring from the store's live rows, and one P0 prints both closes and the delta.
  `SeedVerdict(symbol, tf) → (checked, offScale)` exposes the verdict.
- **D5 — the replay hold (C5's fix).** `trader/ninjatrader/replay_hold.go`.
  Every replay row — the persister's historical frames AND the boot backfill's
  historical ring bars — is held per symbol×tf (one row per minute, newest tail
  under a cap) until the ring's verdict. On every LIVE frame the persister
  resolves the hold FIRST, before its own closed-bars check (which is empty for
  the current minute): on-scale → released as `historical`; off-scale →
  discarded, once, loudly; unjudged → held. `backfillBars` counts held rows as
  landed so the boot loop does not retry for five minutes. An unverified replay
  never reaches the store. An empty minute reads as a gap; a wrong-scale minute
  reads as the largest move of the day.
- **D6 — the migration, and its measured exceptions.** Pre-column rows are
  labelled LIVE (the first draft said historical, "conservative" — it would have
  licensed the next boot's replay to repaint the whole store). Exceptions, by
  MEASURED value inside a measured window, never by rowid (`offScale20260910`,
  `straddle20260910`):

  | label | rule | MNQ | ES | rowids |
  |---|---|---|---|---|
  | `replay:off-scale` | December, 21:15–22:38 CT opens, o AND c below 29200 / 7630 | 51 (1m 30 · 3m 9 · 5m 5 · 15m 5 · 30m 2) | 47 (1m 26 · 3m 9 · 5m 5 · 15m 5 · 30m 2) | 464775–465010 |
  | `mixed` | 21:00–22:39 CT opens, one side each scale | 12 | 13 | 1m: 464738, 464739, 464744, 465011 · ES 1m: 464729, 464737, 464740, 464741, 465013 (+ aggregates) |

  The off-scale rows are the 22:14 boot's restart-gap fills (MNQ 21:19, 21:23,
  21:24, 21:36, 21:53; ES 21:35) and the 22:39 boot's replay over 22:14–22:38
  — post-backup, so the restore could not reach them. Their VALUES are
  untouched (A24); their reconstruction from the archive's `bar_update` facts
  is owner-authorised separately and is now a plain live insert. The mixed rule
  recovers the roll's 21:14–21:17 minutes whose `unrecomputable:spans_roll`
  contract label the 22:39 replay had overwritten to `12-26`. Live December
  closes in the window are 29350–29421 / 7660–7677, the replay's 29052–29094 /
  7599–7602; outside the window the value rule means nothing (September traded
  through 29200 for weeks) and is not applied. Live-copy proof: census exactly
  98 / 25, `SUM(o+h+l+c+v)` per symbol×tf unchanged, the reader's 22:13–22:40
  window returns exactly 22:13 and 22:40, second run changes nothing.
- **D7 — readers.** `LastNBarsOn` / `BarsBetweenOn`: `COALESCE(source,'') NOT
  IN ('mixed','replay:off-scale')` (NULL-safe: GORM adds columns nullable).
- **D8 — boot line, every field read.** `📼 bar source: MNQ live=n
  historical=n mixed=n off-scale=n null=n · replay-never-overwrites-live=on ·
  unverified-replay-held=on · replay-hold: held=n released=n discarded=n ·
  scale-mismatch threshold=0.50% AND 20x median body [I] · mismatches this
  process: …|none`. `NOFX_BAR_SCALE_MISMATCH_PCT` / `_MULT` now actually read
  (the comment had promised them — class 19).
- **D9 — docs:** SYSTEM-MAP, RULEBOOK, guide card "One contract, two sources",
  AUDIT-CHECKLIST class 118.

## E — pins, RED first, and the mutation matrix

Ring (`provider/ninjatrader/bar_source_test.go`): replay never overwrites live ·
live overwrites a same-scale seed · mismatch drops the seed + labels the boot
minute + fires once · same scale fires nothing · every ring bar names its
source · reconnect re-seed is checked again · ordinary move at a small price is
not a mismatch · verdict follows the seed. Store (`store/bar_source_test.go`):
replay never overwrites a live row · precedence live > mixed > historical ·
unlabelled/unknown source refused · readers exclude mixed · backfill labels
live + idempotent + a replay cannot repaint a pre-column row · measured
backfill labels by value not rowid (8 shaped cases) · off-scale unreadable and
repairable by live or released replay. Wire (`trader/ninjatrader/`): persister
stamps the feed it was handed the bars from · closure calls the mapping · boot
backfill holds the replay · off-scale replay discarded, gap stays empty, live
rows untouched · on-scale replay released into the gap only · hold waits for a
verdict · dedupe + newest tail · hold wired into persister and backfill (text
pin, class 113 caveat) · 📼 line read-not-literal.

`scripts/mutate.sh`, 28 mutations, **27 KILLED, 1 NOT-APPLIED (my sed typo,
M28, superseded by M26/M27 on the same lines)**, 0 SURVIVED at the final head:
upsert WHERE dropped · merge takes incoming over live · pct threshold infinite ·
mismatch drops nothing · straddling bar not labelled · per-seed re-arm removed
· readers hand out mixed (per reader, M7a/M7b) · unknown source accepted ·
migration labels historical · multiplier zero · multiplier check removed ·
persister stamps live for a replay · mixed label dropped at persist · call
site passes false for historical · backfill writes replay rows · resolve
ignores off-scale · persister no longer holds · verdict never set · verdict
not cleared on reseed · resolve writes unjudged · backfill returns written
only · resolve call removed · off-scale window unbounded · straddle rule
removed · contract scope inverted · reader excludes only mixed · InsertBars
accepts off-scale. One first-draft survivor (per-process arming) drove D4's
per-seed rule.

**Inherited red, fixed here:** `trader/scenario_link_wiring_test.go` (dev
`426b5657`) built its tape to a FIXED 14:00 CDT clock but computed `formed`
from `time.Now()−6h` — green 14:00–20:00 CDT, red otherwise (class 110). The
three pins now evaluate at the clock they build for. `cd1fbef9`.

## NT8 — THE FINDING FILED FOR THE ADDON WAVE

**Claim [A]:** on the `MNQ 12-26` subscription, NT8's `bars_historical` replay
and its `bar_update` live feed report the same minute ~290 points apart (ES
~65). Evidence: research facts **16516009** (live, 22:37 CT close 29358.25) vs
**16518205** (replay, same minute, close 29068.25); the live DB's 22:14–22:38
CT rows (D6) are the replay's values, the 22:39 bar's open is the replay's and
its close is live.

**Hypothesis [B], for the AddOn wave to establish:** NT8's instrument
`MergePolicy` (MergeBackAdjusted / MergeNonBackAdjusted / DoNotMerge) on the
continuous-contract series the AddOn's `BarsRequest` reads, versus the
un-adjusted real-time feed — the replay's December level (~29068) sits within
~60 pts of September's last live level (~29130), which is what a back-adjusted
merge produces. The AddOn wave owns: (1) the `BarsRequest` construction in
`VLBarsSubscriptionManager.cs` (which `Instrument` object, which
`MergePolicy`); (2) whether `bars_historical` can carry the instrument's
merge policy so Go can refuse a back-adjusted replay by name instead of by
measurement; (3) the date-based roll (`VLContractResolver.cs:80`) already filed
by the roll wave (D7 there, class 117). Until then Go survives the replay by
D4/D5; the P0 says so on every boot that trips it.

## OWED / scope calls

1. **The 98 off-scale rows' VALUES** — reconstruction from the archive's
   `bar_update` facts, owner-authorised separately; with D2 it is a plain live
   `InsertBars` (live overwrites off-scale).
2. **The boot minute's replay open.** The AddOn copies the first live bar's
   open from its historical series; D4 labels the bar mixed, it does not fix
   the AddOn. AddOn wave.
3. **`OnContractRoll`'s log line** uses a bare `at.Format("2006-01-02 15:04:05
   MST")` (roll wave); left as-is, noted.
4. **Roll wave items still open:** D5's `retired_contract` mark on
   `touch_episodes` (ids 1886/1887/1888); the AddOn's date-based roll.
5. **A minute only the replay saw stays EMPTY** until a replay on the live
   scale arrives (which, until the AddOn wave, is never). Stated on the boot
   line as `held=n`; the horizon logic already reads gaps as gaps.

## Rollback

- **Binary:** pre-swap binary copied to `~/nofx-backups/bin/` named by the rev
  it holds (A13); `deploy/RELEASE` reverts with it.
- **DB:** backed up BEFORE the migration boot, `PRAGMA quick_check` and md5 in
  the marker. The migration is additive (a column, an index, label UPDATEs on
  `source=''` rows only — no value is touched); pre-wave code never reads the
  column, so rolling the binary back leaves it harmless.
