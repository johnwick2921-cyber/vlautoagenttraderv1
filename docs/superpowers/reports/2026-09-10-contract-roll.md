# The contract roll — Section G report

**Dispatch 104 (roll wave)** · owner hoang · 2026-09-10 · SIM-only, MNQ, one contract
**Branch:** `fix/contract-roll` · **claim:** `roll-554049f5/nofx-8e[88742a]` @ `86b26c5f`
**Binary rev:** `0070fc79` · guide commit `1caf7264` (frontend-only)
**Running rev at start:** `cd8f9978` (105's boot), pid 2826476 — per 105's report
**Suites at `0070fc79`:** Go **30 ok / 3 test failures, all in 101's
`scenario_link_wiring_test.go` and ALL red on clean dev `90fa7ce2` too** (see
§E8) · vitest **421/421** · tsc clean · environment: vite 6.4.1 installed vs
6.4.3 pinned (class 110), runs at 22:12 CDT.

---

## THE PREMISE THAT WAS WRONG, FIRST (A17)

The dispatch pinned the roll at **19:00 CT**. The tape says **21:15 CT**, and so
does the report the dispatch cites. 19:00 CT is when the resolver's UTC date
rule *decided* (00:00 UTC on the 11th); 21:15:03 CT is when the subscription
*re-ACKed* on reconnect and the bars changed. **The 135 minutes between are clean
September bars** — every close-to-close delta from 18:55 to 21:13 is under seven
points. A backfill keyed on 19:00 would have labelled two hours of September
bars as December.

The dispatch itself says *"that is the roll instant, MEASURED — the backfill
boundary."* Measured, it is 21:15. Used as measured.

## C1 — no contract column [A]

`pragma_table_info('bars')`: `symbol, tf, open_time_ms, o, h, l, c, v,
convention`. PK `(symbol, tf, open_time_ms)`. MNQ 1m rows at start: 22,481
(the dispatch's 22,466 was the count at 105's read; 15 more had landed). ES has
the same 14 timeframes. No row carried a contract.

## C2 — the boundary, from the tape [A]

| rowid | CT | o | c | body | |
|---|---|---|---|---|---|
| 464732 | 21:14 | 29134.00 | 29132.00 | −2.00 | **last clean MNQ 09-26** |
| 464738 | 21:15 | 29131.50 | 29416.00 | **+284.50** | spans roll |
| 464739 | 21:16 | 29124.50 | 29421.25 | **+296.75** | spans roll |
| 464744 | 21:17 | 29129.25 | 29405.25 | **+276.00** | spans roll |
| 464749 | 21:18 | 29406.50 | 29406.50 | 0.00 | **first clean MNQ 12-26** |

Close→open gaps: 21:15→21:16 **−291.50**, 21:16→21:17 **−292.00**. The three
spans-roll bars carry a September open and a December close — the AddOn's
historical replay landed the open, the live update landed the close, and Go's
upsert keeps whole bars.

**ES rolled on the same reconnect, one bar earlier:** last clean 464728 (21:13),
spans 464729/464737/464740/464741 (21:14–21:17, ~+65 bodies), first Z26 464745
(21:18).

**Wire evidence (research archive, `data.db.research.db`):** last `bar_update`
on `MNQ 09-26` is fact **16480370** at 21:14:56; first `bars_historical` on
`MNQ 12-26` is fact **16480372** at 21:15:03; first `bar_update` on `MNQ 12-26`
is **16505304** at 21:15:26. The switch is pinned to seven seconds.

## C3 — who read the mixed tape [A]

Every kernel consumer reads through `market.FuturesBarsProvider` →
`barsFromCache` → the ring: `kernel/void_scope.go`, `naked_poc.go`,
`stale_data.go`, `engine_analysis.go`, `map_projections.go`,
`api/handler_klines.go:346`, `api/handler_svp.go`, `trader/auto_trader_clock.go`
(`deskRange`). None carried a contract; none read the store. **The ring is
single-contract by construction after this wave**, so they are covered by the
purge + filtered rehydrate without being edited (A31).

The **direct** store readers — the ones that bypassed the ring — were:
`trader/bars_store_depth.go:78` (**the planner's depth path**, the read that put
~2,000 September bars under December ones), `auto_trader_weekly.go:75,103`,
`auto_trader_dayplan.go:214`, `ninjatrader/level_stats_wire.go:110`,
`trade_excursion_hook.go:157`, `trade_excursion_backfill.go:110`. All now
filtered (D4).

## C4 — does the bar frame carry the instrument? NO [A]

`HelloPayload` carries only `build_id`. `BarUpdatePayload` carries `symbol` and
`timeframe` — no instrument. The only frames that name it: `SubscribedPayload.
resolved_contract` (the ACK) and `InstrumentInfoPayload.contract`. **The writer
stamps from the most recent `subscribed` ACK**, with `contract_source=
subscribed@<receipt>`; before the first ACK of a process, from the store's newest
usable stamp, labelled `store-fallback`. Never from a date.

## C5 — routing vs bars: they resolve INDEPENDENTLY, and disagreed [A]

- Bars resolve at **subscribe-time**: `VLBarsSubscriptionManager.cs:177`
  (and `:556`), cached until reconnect.
- Orders resolve at **request-time**: `VLTraderTCPClient.cs:907` (submit) and
  `:1140` (close_position), fresh from `DateTime.UtcNow` via
  `VLContractResolver.cs:80`.

So from 19:00 CT (rule flip) to 21:15 CT (reconnect) the bars were on
`MNQ 09-26` and any order would resolve to `MNQ 12-26`. **Order 152 is the
row:** placed 19:11:05 CT, buy limit at **29094** — a September price — it
resolved to December and sat ~291 points below December's market. It never
filled; the boot sweep cancelled it at 21:15:03, the same second the December
ACK arrived. The order snapshots carry only the root `MNQ`, so the routing
contract is not in the record; this is [A] from the C# and the ledger row, [B]
on December's exact price at 19:11 (no December bars exist before 21:15).

**That is D7's finding, filed with its lines.**

## C6 — the planner's levels since the roll [A]

Plan **v7** (ASIA, authored **21:29 CT**, fourteen minutes after the roll)
carries 12 identity levels. **Seven are on the retired September scale:**

| idx | level | price |
|---|---|---|
| 5 | SWG-H | 29267.00 |
| 6 | pdVWAP | 29229.83 |
| 7 | VWAP±2σ | 29172.31 |
| 8 | VWAP | 29147.47 |
| 9 | OR-H | 29123.50 |
| 10 | SWG-L | 29104.25 |
| 11 | SWG-L | 29094.25 |

And **idx 4, IFVG at 29282.38, sits inside the 29130→29410 void** — the roll gap
itself was detected as a fair-value gap. The VWAP family's *values* are
contaminated (computed over the mixed tape), not merely their scale.

**Touch episodes since the roll:** 13 opened; **3 on the September scale — ids
1886 (SWG-H·15m 29267.0), 1887 (pdVWAP 29229.83), 1888 (OR-H 29123.5), all at
21:16 CT.** The mixed 21:16 bar (open 29124.50, close 29421.25) swept through
every September-scale level on its way to December and recorded a touch on each.

## What shipped (D1–D6)

**D1** `bars.contract TEXT`, index `(symbol, tf, contract, open_time_ms)`.
`InsertBars` **refuses** an unstamped row. GORM's `AutoMigrate` adds the column
*nullable*; the first backfill filtered on `''`, touched zero rows and reported
success — NULLs are normalised first now.

**D2** three-state backfill by **window intersection** per `(symbol, tf)`:
`open + tf ≤ MixedFrom` → old; `open ≥ NewFrom` → new; else
`unrecomputable:spans_roll`. On a copy of the live DB: **MNQ 50,899 U26 · 34 Z26
· 7 spans-roll · 0 unstamped; ES 48,095 · 37 · 11 · 0.** Spans-roll rows across
all timeframes: MNQ 1m×3, 3m, 5m, 15m, 30m; ES 1m×4, 3m×2, 5m×2, 15m×2, 30m.
Idempotent; nothing deleted (E6 pins both).

**D3** `TCPServer.CurrentContract(symbol)` → `ContractFact{Contract, Source,
ReceivedAt, Previous, RolledAt}` from the ACK. Trader-side `currentContract()`
and `contractFactFor()`.

**D4** every live reader filtered — `LastNBarsOn` / `BarsBetweenOn` for the
current tape; `WindowContract` for a historical window, which returns
`unrecomputable:spans_roll` for a window across the roll and the caller treats
it as unresolved. `bar_reader_filter_lint_test.go` fails on a new unfiltered
read in live trader code (text-match; class 113 caveat stated in the file).

**D5** `observeContract` at the ACK: different name → **purge the symbol's ring
on every timeframe**, record, notify once. Persist-wire listener reseeds from
the store for the new contract only and raises a P0. Same name → nothing.

**D6** MODE: `contract=MNQ 12-26 since 21:15:03 (subscribed@21:15:03) · ROLLED
from MNQ 09-26 at …`. Boot line `📜 contract: current=… · bars by contract: …
· readers filtered=kept/read · ring reseeded=y/n · last roll=…`, pure formatter,
every field read.

## E — pins, RED first

- **E1 gap pin:** a ring holding both contracts reads a ~294-point range (the
  fixture asserts this BEFORE the fix acts — the red half); after the ACK names
  December the ring is purged, refills on December only, no September bar
  returns, range < 50. On the pre-wave server `observeContract` does not exist.
- **E3** roll fires once with both names and the ACK's receipt; a repeated ACK
  before or after does nothing.
- **E4** window-intersection three-state incl. the dispatch's own 1h example.
- **E5** `CurrentContract` answers only from a subscribed ACK — a pending or
  errored state names nothing *even with a contract string beside it*.
- **E6** retired rows survive; a second backfill changes nothing.
- **D1/D4** refuse-unstamped; filtered vs unfiltered vs window.
- **E2** reader-bypass lint.

**Mutations via `scripts/mutate.sh`, 7 applied / 7 killed:** no purge on roll ·
repeated ACK counts as roll · pending state counts as named · spans-roll window
dropped · `LastNBarsOn` ignores the filter · `InsertBars` accepts unstamped ·
backfill deletes retired rows. Two first drafts were weak and the script named
it: the state-gating mutation SURVIVED because every non-subscribed case tried
also had an empty contract (fixed: a contract string beside a pending state), and
one sed spanned a newline and was NOT-APPLIED.

**E7 (A29):** all 16 new symbols have ≥2 live production callers; list in the
commit. Class 113 caveat: this counts names, not reachability.

**E8 at `0070fc79`:** Go 30 ok; **3 failures, all `scenario_link_wiring_test.go`
(101's), all red on clean dev `90fa7ce2`** — `"pre-formation bars
skipped=2000"`, every bar on the fixture's tape skipped, so nothing recorded.
Not mine; A31; reported to the owner here because 101's session had ended by
the time I could tell them. vitest 421/421, tsc clean.

## OWED / scope calls surfaced

1. **D5's `retired_contract` mark on the episode row — NOT done.** It needs a
   column on `touch_episodes`, which is 101's table and under active wiring.
   Adding a column to another lane's active table mid-flight is the class-70
   collision. The three ids are above. Recommend 101 adds the mark in their wave,
   or this lane adds it after theirs lands.
2. **D7 — the AddOn's date-based roll and independent resolution**, filed with
   `VLContractResolver.cs:80`, `VLBarsSubscriptionManager.cs:177`,
   `VLTraderTCPClient.cs:907`/`:1140` for the next AddOn wave. Order 152 is its
   proof.
3. **The 21:15–21:17 bars' VALUES are contaminated** and are left as they are
   (A31: no bar's values). They are labelled `unrecomputable:spans_roll` and no
   reader accepts them.
4. **A2b disclosure:** to run vitest I copied one file into the main tree and
   restored it (porcelain empty after). That is an edit under A2b and I should
   not have; `node_modules` is now symlinked into the worktree so it never
   recurs.
5. **One commit message lost a backtick to bash** (`a53359ce`, "subscribed"
   rendered as a command). A24: no rewrite.

## Rollback

- **Binary:** `~/nofx-backups/bin/` — the pre-swap binary is copied there named
  by the rev it holds before the swap (A13); `deploy/RELEASE` reverts with it.
- **DB:** `data.db` is backed up BEFORE the migration boot with `PRAGMA
  integrity_check` quoted in the marker. The migration is additive (a column +
  an index + UPDATEs on empty-contract rows only); rolling the binary back
  leaves the column in place and harmless — pre-wave code never reads it.
