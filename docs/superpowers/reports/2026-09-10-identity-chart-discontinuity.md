# 105 post-boot chart discontinuity — read-only investigation

**The planner reads this same ring. It is exposed to the approximately
292-point phantom discontinuity across the September-to-December subscription
change; this is not a chart-only defect. No clean market-data verdict is claimed.**

[A] Running build: `cd8f99780eafe20fbb0b51a9c87955d3a2a4e887`, PID 2826476.
API observed 2026-09-10 21:19:15.308 CT:
`/api/klines?symbol=MNQ&interval=1m&limit=2500&exchange=ninjatrader`.
The handler caps this route at 1500; the following are its last 30 of 1500.
The initial request without `exchange=ninjatrader` returned CoinAnk HTTP 500;
that wrong-source request is not used as evidence about the futures ring.

| Timestamp CT (2026-09-10) | Open | Close |
|---|---:|---:|
| 20:49 | 29112.50 | 29110.00 |
| 20:50 | 29109.75 | 29111.00 |
| 20:51 | 29111.25 | 29108.00 |
| 20:52 | 29108.25 | 29114.25 |
| 20:53 | 29114.25 | 29108.75 |
| 20:54 | 29109.25 | 29110.75 |
| 20:55 | 29111.50 | 29113.25 |
| 20:56 | 29113.50 | 29116.25 |
| 20:57 | 29115.50 | 29120.75 |
| 20:58 | 29121.00 | 29120.50 |
| 20:59 | 29120.25 | 29129.50 |
| 21:00 | 29129.25 | 29134.25 |
| 21:01 | 29134.00 | 29133.00 |
| 21:02 | 29132.75 | 29124.25 |
| 21:03 | 29125.25 | 29124.50 |
| 21:04 | 29124.25 | 29126.50 |
| 21:05 | 29126.25 | 29125.00 |
| 21:06 | 29125.50 | 29124.75 |
| 21:07 | 29124.75 | 29122.25 |
| 21:08 | 29122.75 | 29127.75 |
| 21:09 | 29127.50 | 29125.25 |
| 21:10 | 29126.00 | 29129.25 |
| 21:11 | 29128.75 | 29127.25 |
| 21:12 | 29128.25 | 29128.25 |
| 21:13 | 29128.00 | 29134.75 |
| 21:14 | 29134.00 | 29132.00 |
| 21:15 | 29131.50 | 29416.00 |
| 21:16 | 29124.50 | 29421.25 |
| 21:17 | 29129.25 | 29405.25 |
| 21:18 | 29406.50 | 29406.50 |

[A] Exact discontinuities:
- 21:15 bar: open 29131.50, close 29416.00, within-bar +284.50.
- 21:15 close 29416.00 → 21:16 open 29124.50 = **−291.50**.
- 21:16 close 29421.25 → 21:17 open 29129.25 = **−292.00**.
- Those bars have epoch-open timestamps 1789092900000, 1789092960000,
  1789093020000: all today, not a stale September 8/9 timestamp at the tail.

## Contract and wire evidence

[A] Research archive fact **16480357**, received 21:14:56.969 CT, records
`bar_update`, contract **MNQ 09-26**, 21:14 open=29134, close=29131.
At 21:15:03 the live subscription ACK names **MNQ 12-26**. The contract
field's explicit basis is the preceding subscription acknowledgement; the
archive does not claim a native contract identifier for each historical bar.

[A] On that December subscription:
- Fact **16482371**, historical receipt 21:15:03.148 CT, 21:15 bar
  open=29131.50, close=29131.50.
- Fact **16505304**, live receipt 21:15:26.359 CT, the SAME 21:15 bar
  open=29131.50, close=29418.25.
- Fact **16505334**, live receipt 21:15:34.041 CT, same bar
  open=29131.50, close=29416.00.
- Fact **16505348**, historical receipt 21:16:23.249 CT, 21:16 bar
  open=29124.50, close=29128.00; fact **16505356**, live receipt
  21:16:50.667, same bar close=29421.25.
- Fact **16506228**, live receipt 21:17:45.239, 21:17 bar
  open=29129.25, close=29405.25.

The mixed-price candles are already in the incoming NT8 frames. Go Upsert
replaces the complete same-timestamp bar; it does not merge one frame's open
with another frame's close. The rollover/subscription change is verified.
[B] Historical/live price-scale mixing on the December subscription explains
the phantom jump. NT8's native historical per-bar contract and merge/back-adjust
policy are not recorded, so that finer cause remains UNESTABLISHED.

The resolver's source uses UTC date and expiry minus eight days. September
2026 expiry is September 18; its roll date is September 10. On September 11 UTC
it selects December. The reconnect at 21:15 CT is already September 11 UTC.
This source explanation is consistent with the measured contract switch; no
C# file, resolver, NT8 setting, subscription or bar was changed by this inquiry.

## Full ring and store splice

[A] The actual SSE snapshot bypasses `/api/klines`' 1500 cap. At 21:21:13 CT
it held **2500** MNQ 1m bars:
- oldest: **2026-09-09 01:40 CT**, epoch 1788936000000, O 29609.25 / C 29608.50;
- newest: **2026-09-10 21:20 CT**, epoch 1789093200000, O 29408.50 / C 29412.75.

[A] At 21:15:17.547 CT the rehydrate log states 2000 → 2500 (+500 older),
oldest age 43h39m17.547s, span 43h39m, no missing 1m intervals other than the
calendar's expected closures. That places the boot ring's oldest at
**September 9 01:36 CT**. [B] Reconstructing the 500-row store splice from
that logged boundary and the unchanged ordered store gives:
- `bars.rowid=457826`: Sep 9 01:36, epoch 1788935760000, O 29610.75 / C 29606.00;
- last added `rowid=458325`: Sep 9 09:55, epoch 1788965700000, O 29506.75 / C 29513.00;
- next/seed boundary `rowid=458326`: Sep 9 09:56, epoch 1788965760000,
  O 29512.25 / C 29517.00. The splice close→open is −0.75, not 300.

All 500 reconstructed source rows are attached. They extend the FRONT of the
ring. `RehydrateOlder` accepts only timestamps before the seed's oldest and
keeps live overlaps. The store cannot manufacture a 21:15–21:17 mixed OHLC
bar by this path. Its existing schema keys `(symbol,tf,open_time_ms)` and
the ring keys `(symbol,tf)`; neither retains contract in that identity.
The ring spans two calendar days by design, but the observed 300-point jump
is at today's tail, not at that older store splice.

[A] Persisted rows also now contain the contaminated values:
`464738`: 21:15 O29131.50 C29416.00; `464739`: 21:16 O29124.50 C29421.25;
`464744`: 21:17 O29129.25 C29405.25. Persistence does not repair them.

## Planner path and scope

At the running build: `api/handler_klines.go:346` calls
`market.FuturesBarsProvider`; `trader/ninjatrader/bars_market_bridge.go:24`
binds that provider to `server.BarCache()` via `barsFromCache`.
The planner's `trader/auto_trader_planner.go:2271` calls
`at.barsWithStoreDepth(...,"1m",...)`; `trader/bars_store_depth.go:61`
starts from the same provider and preserves its live tail while adding older
store rows. Thus the planner receives these prices; extending the history does
not remove the discontinuity. A new planner read was observed at 21:17.
The exact corrupted rows inside that already-in-flight prompt are not asserted
without inspecting its captured input; later reads from this ring see them.

No production code, DB row, order, planner state, account or feed setting was
changed. A clean boot-integrity result proves build identity, not tape integrity.
The boot report must carry this incident prominently.


## Owner-requested targeted purge: STOP, store cannot select current contract

Owner authorized purging non-current-contract ring bars, reseeding 1m from
that contract's store rows, then restarting the same revision, with an explicit
condition to report if the store itself mixes contracts. That condition is met.

[A] At 21:27 CT `/api/nt/symbols` reports current subscription **MNQ 12-26**.
The Windows compile-source `VLTraderTCPClient.cs:907` resolves execution
through `VLContractResolver.ResolveFrontMonthContract`, the same resolver
used by the bar subscription. Its current UTC-date result is December.
No test order was sent to prove routing, and no contract was changed.

[A] `PRAGMA table_info(bars)` lists only `symbol,tf,open_time_ms,o,h,l,c,v,convention`.
The unique index is `(symbol,tf,open_time_ms)`. `symbol LIKE 'MNQ%'` finds
only unsuffixed **MNQ**, across 14 timeframes; **22,466 1m rows** at the
observation. There is no `MNQ 09-26` or `MNQ 12-26` partition and no per-row
contract field. The mixed OHLC rows 464738/464739/464744 are persisted in it.
`LastNBars` filters symbol and timeframe only, and `RehydrateOlder` receives
no contract. Neither can implement the owner's requested contract-only reseed.

**Purged: 0. Reseeded: 0. Additional restart: none.** A selective purge count
cannot be computed from missing contract provenance. A price threshold or
timestamp boundary would infer a contract, which the dispatch forbids.
Purging the ring then replaying this store or the same mixed historical feed
would restore the defect. The store is an unsafe reseed source; raw NT8 frames
also already contain the mixed-price bars. Thus clearing only the ring is
insufficient. No behavior/schema fix is smuggled into this targeted-boot request.

Current desk at **21:27:44 CT**, not an after-repair claim:

```text
RANGE: session 29062.25–29421.25 = 359.00 pts on 265 bar(s) · 10.51× ATR5m (34.16)
```

It reports state=ok and verified=true despite the discontinuity; that flag
certifies the calculated surface, not native contract continuity.
[Exact desk payload](2026-09-10-scenario-level-identity-data/purge-blocked-desk.json).
The next repair needs contract-provenance-safe storage and a verified uniform
contract/price basis from NT8; this report does not authorize changing either.
