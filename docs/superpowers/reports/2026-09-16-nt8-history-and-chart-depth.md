# 101 — 5m CHART DEPTH + NT8 HISTORY AT SUBSCRIBE + CHART ACROSS THE ROLL

**Branch** `fix/nt8-history-and-chart-depth` · base dev `8f5f26ee` · running rev `3ce4281a4b6b`
(`/api/health` and `/proc/1834811/exe` agree). **STATUS: STAGED-AND-GREEN at the cutover gate** — stopped at Section C (A23), the
premise refuted, scope revised by the CTO after verifying the refutation, built under the
revised scope. (3a)/(3b) — the AddOn re-request and the all-timeframe store rehydrate —
are HELD pending the owner's direct word in his chat.

---

## THE FINDING — NT8 delivered the history; Go destroyed it eleven hours later

**Premise as dispatched:** "NT8 returns ~0 history at subscribe (historical=139)."

**Measured:** at 22:15:08 CT on 09-15 the AddOn emitted, per its own
`emitted bars_historical` lines:

```
MNQ|1M bars=2000   MNQ|5M bars=2000   MNQ|15M bars=2000   MNQ|30M bars=2000
MNQ|1H bars=1536   MNQ|2H bars=802    MNQ|4H (see below)  MNQ|1D bars=69
MNQ total: 13,054 historical bars
```

**NT8 honoured `bars_back`.** `📼 bar source: … historical=139` is a **store** census
(`bars.source='historical'`, live=57395 cannot be a 2,500-cap ring) — it counts persisted
replay rows, which the replay-hold deliberately keeps out of the store. It was misread as
"delivered".

**The 5m ring was 12 days deep until 09:22 on 09-16:**

| when (09-16) | MNQ 5m horizon |
|---|---|
| before 09:22 | `asked=5000 served=2131 span=285h55m` |
| after 09:22 | `asked=220 served=134 span=11h5m` |

**What happened at 09:22:11.** The data feed flapped five times that morning (07:20,
07:22, 07:34, 08:33, 09:22 — NT8 log: "BarsRequests will be recreated on recovery"). Each
reconnect re-subscribed and the AddOn emitted **`MNQ|5M bars=0`** — a zero-bar replay,
because the feed was still down when the BarsRequest ran. Then Go logged:

```
🚨 P0 — REPLAY AND LIVE ARE ON DIFFERENT PRICE SCALES for MNQ 5m at 09:22:11 CT:
last replay close 29320.50, first live close 29467.25, delta 146.75 pts (> 0.50%).
1999 historical bars DROPPED from the ring and refilled from the store's live rows
```

**`29320.50` is the 09-15 22:10 CT bar** — the LAST bar of the *boot-time* replay, eleven
hours earlier (store: `09-15 22:10 c=29320.5 live`). The detector compared it to the first
live bar after 09:22 and read the overnight move as a scale break.

**Mechanism, `provider/ninjatrader/bar_cache.go` `SeedHistorical` (:230):** the scale-check
re-arm `delete(c.liveSeen, key)` at :246 runs **before** the `len(bars)==0` return at :263.
A zero-bar replay re-arms the check without updating the reference bar, so the next live bar
is judged against whatever the newest historical bar in the ring happens to be — however old.
Then the refill (`rehydrateRingFromStoreWith`, `bar_persist_wire.go:178`) rehydrates **1m
only** (`pairsToRehydrate`, :478 `p[1] == rehydrateTimeframe`), so 5m/15m/1h/4h stay live-only
for the life of the process. Same event on MNQ 1m: 1832 dropped.

**Every hypothesis in D1 is refuted, for the record:**

| H | claim | verdict |
|---|---|---|
| H1 | NT8 local DB shallow for Dec | **No** — `db\minute\MNQ 12-26\` has 83 day-files, 06-12→09-16, 5–10 KB each |
| H2 | TradingHours template starves | **No** — `CME US Index Futures ETH`, and 2000/2000 arrived |
| H3 | rolling literal reaches BarsRequest | **No** — `resolved … => MNQ 12-26`; `Subscribe(symbol, tf, barsBack, instrument)` per tf; reconnect resolves the same |
| H4 | Tradovate caps depth | **No** — 2000 returned for every tf ≤ 30m |
| — | **zero-bar reconnect replay re-arms the scale check; false positive drops the seed; refill is 1m-only** | **Yes** — proven above |

## C · EVIDENCE, each reproduced at `3ce4281a`

- **C1** ✓ (after 09:22) `served=154 span=12h45m oldest=process start`; **✗ before 09:22** — `served=2131 span=285h55m`. The premise held only after the destruction.
- **C2** ✗ as stated — `historical=139` is a store count; NT8 emitted 13,054.
- **C3** ✓ at 22:15:1x — but `1H bars=1536` was emitted at 22:15:08; the horizon line likely preceded the drain (ordering, not emptiness). Unverified; noted.
- **C4** ✓ `26 pair(s) SKIPPED as tf!=1m`, `bar_persist_wire.go:478`.
- **C5** ✓ `bar_history.go:483,540,568 Where("contract = ?")`; `handler_klines.go:385`.
- **C6** ✓ MNQ 12-26 = 581 5m rows (live from 09-14); 09-26 = 20,197; store 284,051 rows, 19 contracts, back to 2022-04-11.
- **C7** ✓ served bundle `index-COvgwytr.js` has `limit=1500`, no 5000; dev-tip source has `limit=${limit}`. L5's.
- **C8** ✗ as stated — the AddOn **does** log the count (`emitted bars_historical <key> bars=<n>`) and the resolve (`resolved MNQ -> MNQ ##-## => MNQ 12-26 (rolling->MNQ 12-26)`). D3(a) is unnecessary.

Position 606 reads CLOSED, not OPEN as dispatched. The running binary's log paths carry
`nofx-deploy-r4/main.go` — built in a directory not named `nofx` (class 75).

## WHAT THIS MEANS FOR THE FIX (proposal, not started)

- **D1 as dispatched is moot.** No AddOn change is needed for depth; the AddOn is correct.
- **The real D1 is Go-side, three lines of intent:** (1) a zero-bar replay must not re-arm the
  scale check; (2) the check must compare *adjacent* bars — last replay bar and first live bar
  within one interval — never a reference hours old; (3) after a true drop, refill every
  timeframe from the store (or re-request from NT8), not 1m only.
- **D2 (chart across the roll) stands** — owner ruling, independent of the above.
- **D3(b) stands** — 8,258 horizon WARNs since boot; my own `fade_facts.go:73` is a caller.
- **D3(a) is unnecessary**; the count is already logged.



---

## D · WHAT SHIPPED (revised scope, CTO rulings 11:35 / 11:50 CT)

| | files | pinned by |
|---|---|---|
| **D1'(1)** an empty replay cannot re-arm the scale check | `bar_cache.go` | `TestZeroBarReplayDoesNotReArmTheScaleCheck` (5m+1m), `TestZeroBarReseedDoesNotRejudgeAVerifiedSeed` |
| **D1'(2)** the check judges ADJACENT bars only (≤2 intervals); older → SKIP + WARN with both ages, stay armed | `bar_source.go`, `bar_persist_wire.go` | `TestStaleReferenceIsSkippedNotDropped`, `TestPastGapBackfillDoesNotDrop`; rule 3 `TestAdjacentRealBreakStillDrops` |
| **D1'(3) narrow** a confirmed break is COUNTED and the P0 says "LIVE-ONLY UNTIL NT8's NEXT FULL REPLAY" for non-1m | `telemetry/scale_break.go`, `bar_persist_wire.go` | `TestScaleBreakDropIsCountedWithItsBars` |
| **D2** chart across the roll, flag ON [O] | `store/bar_history_across_roll.go`, `api/handler_klines.go`, `market.Kline.Contract` (additive, omitempty) | `TestPriorContractsFillBehindTheCurrentContract`, `TestKlinesAcrossRollLabelsEveryBarAndKeepsTheStep`, `TestCurrentContractReaderReturnsImportsUnfiltered` (was `TestDecisionReadersStayCurrentContractOnly` — its 500 measured an empty set, §D'' below) |
| **D3(b)** one horizon WARN per (symbol,tf,why) per 5 min, callers aggregated | `bar_horizon_warn.go` | `TestFourCallersInOneWindowEmitOneLineWithCallersAggregated`, `TestHorizonWindowIsFiveMinutesAndReArmsWithCallers` |
| **E1** `🧯 nt8 history at subscribe:` per-tf received/asked, n/a for absent | `history_at_subscribe.go`, `tcp_server.go` (enqueue count) | `TestHistoryAtSubscribeLineRendersResolvedValues`, `…AbsentIsNotZero` |
| **A12** class 127, SYSTEM-MAP bars section, RULEBOOK §A | docs | same commit |

**No AddOn change. No `.cs` copy. `bar_history.go` byte-identical. `pairsToRehydrate` untouched.**
A31 grep against the do-not-touch list: none. Class 100 deletions vs dev: none.

### D1'(4) — the three facets from batch 2, decided or named LEFT

- **(c) drop all replay bars or only the offending seed — DECIDED: drop all, and here is why.**
  The ring stamps source per bar, not per seed; after a merge (`mergeSeedKeepingLive`) two
  replays' bars are indistinguishable. Dropping "only the offending seed" would require a
  seed id on every bar — a schema change beyond this wave. With rules 1–2 in place a drop
  only happens on an ADJACENT real break, so the destructive response now has the
  precondition it lacked. Documented in `bar_source.go`.
- **(a) the 20×-median-body rule blinds higher timeframes — LEFT**, named: on 15m/1h the
  median body is large enough that a real shift can pass the range check. It is 104's OWED
  item from batch 2 and touching it changes what a break IS, which is a market-belief
  change this wave must not make (research law). Rules 1–2 reduce its exposure (fewer
  spurious checks) without changing its threshold.
- **(b) the re-seed clears `seedOffScale` before the persister's replay hold reads it — LEFT**,
  named: rule 1 now means an EMPTY re-seed no longer clears it (the early return precedes
  the delete), which closes the 09-16 path; a NON-empty re-seed still clears it, which is
  correct (a new replay is a new verdict) but races the hold. Ordering the hold's read
  before the clear is 104's.

## D'' · (3a)/(3b) — SHIPPED under the owner's direct word ("i want fuull data", 2026-09-16, in my chat)

The CTO relayed the ruling first; I held (3a)/(3b) for the owner's own words per the dispatch's
"GO comes from the owner in his chat". They came. Built to the CTO's amendment-2 spec, then
**revised twice on nofx-93's review of `4099cc69` (both objections SUSTAINED by the CTO)**.

| | what | files | pinned by |
|---|---|---|---|
| **(3a)** a CONFIRMED scale break re-requests NT8's full replay — **once per symbol per BOOT** | `history_rerequest.go`, `bar_persist_wire.go` | `TestConfirmedBreakRequestsAFreshReplayOnce` (two breaks → ONE frame, second refused as SPENT), `TestReplayIsNotRequestedWhileTheFeedIsDown`, `TestReplayBudgetIsPerSymbol` |
| **(3b)** every symbol×tf pair rehydrates from the store at boot and after a drop, four guards | `bar_persist_wire.go` (`pairsToRehydrate`, `rehydrateRowsFor`, `rehydrateBarsFromRows`) | `TestRehydrateSelectsEveryPair` (was `TestRehydrateSelectsOnly1mPairs`, the class-86 pin, renamed never deleted), `TestRehydratedRowsEnterTheRingAsHistorical`, `TestPostDropRefillExcludesHistoricalRows`, `TestRehydrateDoorExcludesImportsOnBothPaths` |

**The four guards, as code:**
(i) after a confirmed drop only LIVE store rows refill (the replay rows are what the drop judged);
(ii) every rehydrated row enters the ring stamped `historical` — replay-grade to this process, never a
sacred live bar; (iii) `historical_import` rows are refused **at the door**, on both paths, and the
boot line prints the refused COUNT; (iv) the contract is the current one (`LastNBarsOn`).

**AddOn premise for (3a), verified by 104 (recorded, not assumed):** N4 `Subscribe` on an active
key disposes and recreates the `BarsRequest` → a full `barsBack` replay; deployed AddOn md5 ==
repo. **No AddOn change.**

### nofx-93 objection 1 — guard (iii) was a boot-line literal with no code behind it (SUSTAINED, class 82 + new class 128)

At `4099cc69` the per-tf line printed `import=excluded-by-reader`. It was false: `LastNBarsOn`'s
filter is `COALESCE(source,'') NOT IN ('mixed','off-scale')` (`store/bar_history.go:481/:538`)
and hands imports to every caller. My E4 fixture "measured" 500-not-505 because its 5 import
rows sat on open times the 09-26 series already held — the bars PK is `(symbol, tf,
open_time_ms)`, no contract — so `ImportBars` **skipped all five** and the assertion measured an
empty set. I wrote "measured, not assumed" over a measurement of nothing.

Fix, as 93 specified and the CTO ruled: **exclude at the rehydrate door, not in the shared
reader.** `rehydrateRowsFor` now returns `(kept, importExcluded)`, refuses
`BarSourceHistoricalImport` on both paths, and the line reads
`import=<n> (refused at the door — guard iii)` from that count. RED first, end to end on a real
store: 500 live + 5 import 12-26 rows at NON-colliding times → `LastNBarsOn` returns 505 → door
keeps 500, counts 5 (`bar_horizon_warn_test.go:349: door kept 505 (want 500), counted import=0
(want 5)` before the fix). The store-side premise is pinned on its own:
`TestCurrentContractReaderReturnsImportsUnfiltered` asserts **505 with 5 imports** from the
reader. The fixture now proves its own census (`ImportBars inserted=5 skipped=0` or fatal).

The same fixture defect hid a second thing: `PriorContractBarsBefore`'s per-timestamp
"best source" dedupe can never fire (one row per open time by PK), and with imports in HOLES
of the 09-26 series — the only place production's 426 can be — the display reader returned a
12-26-priced import inside the 09-26 series (RED: `row 2000 is MNQ 12-26`). Fixed by making
"prior" literal: the reader now takes `current` and excludes it. A hole in the prior series
stays a hole. E3 re-pointed to 2,995 (3,000 slots, 5 holes). API E3 unchanged and green.

### nofx-93 objection 2 — a 5-minute re-request floor loops on a TRUE break (SUSTAINED)

`mergeSeedKeepingLive` keeps the existing bar only when it is `live`/`mixed`; historical over
historical, the INCOMING wins. So on a real break: drop (guard (ii) rows included) → post-drop
rehydrate restores them → re-request → NT8 replays the wrong scale again → it overwrites the
store-backed rows → off-scale until the next live bar of that tf judges it → drop → again, every
five minutes. Fix: `historyReplayMaxPerBoot = 1`, per symbol. The second refusal is
`ErrHistoryReplaySpent` and the P0 line reads, verbatim (A24):
`🚨 P0 — second scale break this boot — replay on another contract, restart the AddOn (<sym> <tf>:
NT8 NOT re-asked; the ring keeps its live bars + the store refill below, entered replay-grade)`
(93's wording nit, taken: the post-drop refill (3b) still runs, so "live-only" was false). Counter (`IncScaleBreakDrop`) still increments on every break. A8
mutation: the time floor put back at `history_rerequest.go:64` (build rc=0) → `got <nil>` and
`want exactly ONE bars_subscribe frame, got 2`; restored, green.

### NOTE 4 — guard (ii) changes what `historical` means

The `📼 bar source: live=N historical=M` census and the P0 drop counts now include store-backed
rows entered by (3b), not only what NT8 replayed this boot. A post-drop refill therefore
over-excludes: guard (i) refuses every store row stamped `historical`, which includes rows a
PRIOR boot received as NT8 replay and persisted under that stamp (13 of 598 on `12-26` 5m
today) — never judged wrong, refused because the stamp cannot tell them from what this boot's
drop judged. Rows persisted `live` survive. Read the per-tf `🧯 ring rehydrated … store_live=
store_hist= import=` line to separate them; the 📼 line alone no longer does.

## E · MUTATIONS — line quoted, sed confirmed, build GREEN, RED quoted

| # | file:line | mutation | died on |
|---|---|---|---|
| M1 | `bar_cache.go:254` | `if len(bars)==0 && len(existing)>0` → `if false` | rule 1's own fixture: "an empty re-seed re-opened a verdict already given: historical 30 → 0" — **after the 09:22 fixture alone SURVIVED it** (rule 2 also covers 09:22) |
| M2 | `bar_source.go:186` | drop the adjacency guard | stale-reference and gap-backfill |
| M3 | `bar_source.go` | adjacency = 1000 intervals | stale-reference |
| M4 | `bar_horizon_warn.go:193` | caller back in the key | "four callers … got 4" — **first attempt's sed did not apply (delimiter clash) and passed vacuously; re-run correctly, recorded in a08e0625** |
| M5 | `handler_klines.go` | +292 on prior rows (a back-adjust) | "the basis step is 0.00, want 292.00" |
| M6 | `history_at_subscribe.go` | absent rendered as 0 | "absent must be n/a" |

**A29** call sites: `stampFadePermission`-style wrappers not needed here; each new function's
production caller counted: `IncScaleBreakDrop` 1, `ScaleBreakCounts` 1, `OnScaleCheckSkip` 1,
`barHorizonCallersTxt` 1, `HistoryAtSubscribeLineFor` 1, `ChartAcrossRollResolved` 1,
`PriorContractBarsBefore`/`FirstLiveOn` 2 (api/ only; **0 under kernel/, trader/, provider/** — E4).

**E6** at the merged head `8f5f26ee`, 11:42 CDT (outside 12:00–13:30): **SUITE 32 ok / 0 FAIL**;
**E6 (second, for (3a)/(3b) + the 93 fixes)** at the merged head `9e200002`, 13:30:53–13:35:56 CDT
(outside 12:00–13:30): **SUITE 32 ok / 0 FAIL / 0 panic**. Clean-clone binary at that head:
`vcs.revision=9e200002…`, `vcs.modified=false`, dir `nofx`, md5 `ac4de9293299faaea8d0b92dd4085461`.
Gate on a fresh read 13:22:01 CDT: ready=true, legs 1–5 PASS. Rollback file census (A13): the
existing `nofx-bin.old.3ce4281a` HOLDS `83b76c51` — renamed to its true name before the live
`3ce4281a` takes that name.
lock suite 101/0. One pre-existing pin re-pointed: `TestKlinesNinjaTraderStoreDepthContractFiltered`
asserted the 09-14 rule the 09-16 ruling reverses; it now pins the ruling and, with the flag
off, the 09-14 behaviour exactly.

## A15 · WHAT THE OWNER WILL STILL SEE WRONG

1. **The running binary `3ce4281a` has the 09:22 defect.** Until this boots, any feed flap
   can still empty every non-1m ring. The 5m chart on `:8080` stays 11 hours deep.
2. **The served bundle is `index-COvgwytr.js` with `limit=1500`** — L5's dist rebuild is
   needed for the chart to ask past 1,500 at all; the `contract` label on each kline arrives
   with this boot but the FE that renders it is L5's.
3. **After this boot, a TRUE scale break gets ONE re-request per symbol per boot** (3a); a
   second break in the same boot is diagnosed on the P0 line; NT8 is not re-asked, the store
   refill (3b) still runs (live rows, entered replay-grade), and the AddOn restart is the fix.
   That is the bound, by design.
4. **The five `bars=0` reconnect replays** (07:20, 07:22, 07:34, 08:33, 09:22 — BarsRequest
   run while the feed was down) are an observation for 104; the AddOn returns nothing and
   says so honestly. With rule 1 they are harmless; they are still wasted requests.
5. **"1H EMPTY at boot while 1536 emitted"** — the horizon line at 22:15:1x likely preceded
   the drain. Noted, unverified; did not fall out of the fixtures.
6. **The running binary's log paths say `nofx-deploy-r4/main.go`** — built in a directory
   not named `nofx` (class 75). This build is from a clean clone named `nofx`.
8. **THE SECOND DOOR — the planner's 1m tape reads imports TODAY [A].** `trader/bars_store_depth.go:81/106/121/219`
   feed the planner's and the weekly reader's 1m splice through `LastNBarsOn` with
   `n = plannerCandleTapeBars = 12000` (`auto_trader_planner.go:2957`, `auto_trader_weekly.go:494`).
   Read from `data/data.db` (read-only) at this report: `MNQ 12-26` 1m = **2,873 live + 25
   replay + 426 `historical_import`** = 3,324 < 12,000, so **all 426 import rows are inside the
   planner's 1m tape now**; on any `09-26` resolve the store holds **75,492** 1m import rows
   (5m 16,133 · 15m 4,065 · 1h 66). Guard (iii) keeps them out of the RING; nothing keeps them
   out of the planner's store splice. Excluding there changes today's planner input (class 82)
   and is the owner's yes/no — the CTO has put it to him. **Not in this PR.**
7. The **1833 vs 1832** in the 1m fixture is the ring cap trimming one bar on the live upsert
   — 2,000 seeded + ~667 live > 2,500 — the same arithmetic as the live "1832 dropped". A
   non-defect that reads like one.

## STORE CENSUS AT THE GATE — nofx-93, read-only, against 10f424b0 semantics

Accepted and reproduced. What matters for this PR and the next:

- **10f424b0's 1m rehydrate on MNQ 12-26 selects 2,500 rows, all `live`, 0 `historical_import`,
  filtered 0** (T2). The import yes/no is structurally open, numerically moot for this boot.
  T4: zero `replay:off-scale` rows on MNQ.
- **D2's boundary is right where it matters.** `FirstLiveOn(MNQ,5m,12-26)` = 09-14 10:00 CT,
  dense at 5-minute gaps from the first row; 93's "7 before cutoff" are the 10:00–10:30 bars
  just ahead of their 10:34 CT cutoff, not July. On 1h: 09-14 02:00 CT, 55 rows, largest gap 2h.
- **93's T3 finding, verified here and named LEFT for 104 (the wire/persist path):** MNQ 12-26
  carries ~9 `live`-stamped rows per tf dated BEFORE the roll — 1d from 09-02 at **rowid 438391**
  (db max ≈2.27M), 4h from 09-11, 3d 18 back to 08-10, 1w 8 back to 07-17 — old rows recently
  UPDATED to `live` + `12-26`. That is a fixed-count closed-bar catch-up delivered as
  `bar_update` right after a (re)subscribe, stamped `live` by the persister because the
  frame type says so, and upserted over real rows because only historical-over-live is
  blocked. **A `bar_update` for a bar that closed before the subscribe is a replay.** The
  mis-stamp is in the wire/persist path, not a separate higher-tf writer. For D2 it is
  harmless (the chart labels contract, not source, and the prices are December's); for
  (3b)'s guard (ii) it means the "older than boot" test cannot be store-side — it must be
  ring-side, against this process's first live frame for the key, whatever the stamp says.
  The CTO's optional in-scope WARN (persist path: a `bar_update` whose close predates the
  subscribe) is **LEFT**: a diagnostic line is not worth a rebuild of a binary already parked
  green at a gate blocked on the main tree; it is one fixture and one line for 104.

  **The two refs the fix starts from — nofx-93, path owner, verbatim, at 10f424b0 (read-only;
  assignment is the owner's):**

  1. `trader/ninjatrader/bar_persist_wire.go:57-60` `barRowsForPersist` — `feedSrc :=
     store.BarSourceLive` unless `historical`; the stamp comes from the FRAME TYPE alone, and
     the call site `:124` passes the frame's `historical` straight through. The fixture: a bar
     in `closed` whose close (`b.T + tfDur`) predates the key's subscribe/ACK receipt this
     process → `store.BarSourceHistorical` (replay-grade) + one WARN naming count and key, and
     it then takes the `historical` branch at `:125-129` into `barReplayHold.add` like any
     other unverified replay, so the ring's verdict still gates it. RED first: a `bar_update`
     frame carrying 9 closed bars older than the ACK must produce 9 `historical` rows and 0
     `live`; GREEN today produces 9 `live`.
  2. `provider/ninjatrader/bar_cache.go:347` `Upsert` — `stampSource(bars, BarSourceLive)` on
     every bar of a `bar_update` frame; the same catch-up bars enter the RING as `live`, which
     is why `mergeSeedKeepingLive` shields them from the next replay and `detectScaleMismatch`
     (reference = `Source==historical` only) never judges them. Same predicate, ring side:
     bars whose close predates the key's subscribe time → `BarSourceHistorical` before the
     merge. **Pin both or the store and the ring disagree about the same bar.**

  The timestamp both need: the `subscribed{resolved_contract}` ACK's receipt time
  (`tcp_server.go:842` describes the ACK; `contractFor` already reads it "at receipt", so the
  receipt clock is the one to record per key — not `time.Now()` at the frame, and not the
  bar's own T). One value, one clock, both sites.
- The 09-26 pre-wave `live` rows on every tf (3m 4805 of 5453 before the 09-11 cutoff …)
  are the migration's stamp — the `source` column did not exist before 09-10. Age alone
  cannot distinguish them from real live rows. Same conclusion: ring-side, not store-side.

## OWED (not this PR)

- **The second door** (A15 §8): imports in the planner's 1m store splice — owner's yes/no.
- (3c)/(3d): `weeklyBias.ts:122` (to L5); the regime fallback arm now serves from the 1m tail
  (condition (b)) — its before/after boot line is quoted at the boot.
- The Guide paragraph for `web/src/guide/content/status.ts` — sent to L5 as text.
- Store-count re-run at the gate — nofx-93's offer, accepted.

## F3 · THE BOOT — 9e200002, 2026-09-16 14:33:08 CT (owner-run swap, mid-session by the owner's explicit "go" ×2 in 101's chat + GO to the CTO 13:44 CT)

Tree recovered by the owner first (A2b cleared: `~/nofx` at `50cd0fdd == origin/dev`, porcelain 0;
staged WIP preserved on `wip/class45-staged-20260916`; CLAUDE.md 421 lines). Swap + kill were
OWNER-RUN via `~/nofx-backups/cutover-101-9e200002.sh` — both agent classifiers deny the deploy
path. PID 1834811 → **2704230**. Every line below read from `data/nofx_2026-09-16.log` [A].

- `🔐 BOOT INTEGRITY OK — rev 9e200002d6a2 · built 2026-09-16T17:35:50Z · expected 9e200002d6a2 · goldens PASS`
- `🧯 nt8 history at subscribe: MNQ 1m=2000/2000 3m=2000/2000 5m=2000/2000 15m=2000/2000 30m=2000/2000 1h=n/a/2000 … 1w=n/a/2000` — NT8 delivered the window on every subscribed intraday tf (the dispatch's premise, refuted in §THE FINDING, now on a boot line).
- `🧯 ring rehydrated MNQ 1m [O 2026-09-16]: nt8=2000 store_live=2500 store_hist=0 (post-drop excluded=false) import=0 (refused at the door — guard iii) total=2500/2500 (+500 older, all entered as historical — guard ii)` — `import=0` is a COUNT: the 426 12-26 import rows are older than the newest 2,500 the reader hands the door. Same shape on ES 1m.
- `🧯 ring rehydrate done [O "i want fuull data" 2026-09-16]: 2 of 9 symbol×tf pairs deepened, +1000 bars total, 0 read failure(s), 0 pair(s) not selected` — non-1m pairs did not deepen because the store's 12-26 depth (5m: 623 of the newest 2,500 rows on 12-26) is shallower than NT8's 2,000-bar replay, exactly the CTO's amendment-2 premise; (3a) is what carries those tfs.
- `🕳 bar horizon … MNQ 5m asked=2500 served=2000 span=225h50m0s … gaps=3` — **5m served 2,000 / 9.4 days** (the 09-15 boot on 3ce4281a showed the same 2,000 at boot; the defect fired 11 hours later — the fixtures, not the boot line, prove rules 1–2; zero `scale check SKIPPED` / `DIFFERENT PRICE SCALES` lines since boot).
- `📈 chart: across-roll=on[O] · prior contracts fill strictly before the current contract's first live row · step never adjusted · limit max=20000 · decision readers=current-contract-only`
- `/api/klines?symbol=MNQ&interval=5m&limit=5000&exchange=ninjatrader` → **5,000 klines, `MNQ 09-26`=2,995 + `MNQ 12-26`=2,005**, monotonic, 08-21 08:45 → 09-16 14:35 CT; the roll step **`09-26 29617.00 → 12-26 29918.50`, Δ +301.50 at 09-07 04:40 CT** (`FirstLiveOn` = the first live 12-26 row, [O] never adjusted).
- Condition (c) `📈 regime input window @14:33:12 CT: BEFORE window=7 · baseline=0.965453 · 2000 5m rows via 5m-ring (pre-wave) · AFTER window=7 · baseline=0.965453 · 2000 5m rows via 5m-ring-fallback · Δ+0 day(s)` — unchanged, as the RULEBOOK says it must be.
- `🖥 ui: served-by=go-static build=2026-09-13T06:02:55Z STALE` — expected; L5 owes the dist (A15 §2).
- A13: `nofx-bin.old.3ce4281a` now HOLDS 3ce4281a; the file that held 83b76c51 is `nofx-bin.old.83b76c51`.
- Still 0 at 14:35 CT: **MNQ 1h / 4h / 1d horizons** (NT8 answered `n/a` for every HTF at subscribe) — the AddOn HTF starvation (acceptance-gate F-2, 2026-08-15), not this wave; re-read owed when NT8's HTF replay lands.
- `🚨 CLOCK EARLY-WARNING |drift| 47.5s` (WSL2 time-sync) and `guardrail_would_trip realized today=-2136.00` — pre-existing, logged, not this wave.

Marker: this commit, from `~/nofx` (the SAME tree that booted), `deploy/RELEASE=9e200002` written before the kill (A19).
`GUIDE_BUILT_REV` is NOT bumped here: `web/` is L5's under this dispatch's do-not-touch list; the Guide paragraph went to L5 via the CTO.

## ROLLBACK

Go half: `mv nofx-bin nofx-bin.failed.<rev> && mv nofx-bin.old.<prev-rev> nofx-bin && echo <prev> > deploy/RELEASE && kill -9 $(pgrep -x nofx-bin)`. No AddOn half. No migration (Kline.Contract is wire-only; no DB column). `NOFX_CHART_ACROSS_ROLL=off` disables D2 without a rebuild.
