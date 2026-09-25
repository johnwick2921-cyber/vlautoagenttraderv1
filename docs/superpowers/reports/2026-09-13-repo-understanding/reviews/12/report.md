# Assignment 12 — market and other data providers

Source review at `63968be62e44db2fb07a92883e02127b9064b0be`, isolated worktree `/tmp/nofx-understanding-execution-20260913`. Initial pwd/revision/status checks verified this tree and a clean porcelain status. All **49 assigned files / 7,579 lines** were read fully. `functions.json` inventories **216 named functions/methods**, exact start/end lines, semantics and observed call expressions. Anonymous callbacks are grouped under their containing declaration. `reads.json` distinguishes assigned full reads from additional dependency/test excerpts. No source/config/DB changes, runtime requests, payment calls, trades, deployments, or tests were performed.

[A] below means directly read source or measured static inventory. [B] means a consequence inferred from that source. These are static findings, not reproduced runtime incidents. No live rows were inspected, so no P&L or account claims are made. The owner's daily-loss clarification remains authoritative; the optional pure `market.PositionSize` helper is not evidence that an additional mandatory per-trade cap should exist.

## Operating and documentary evidence

Read the supplied/main instruction file and tracked `CLAUDE-canon.md`, including its corrected keeper-owned heartbeat and checked-worktree verbs. `market/AGENTS.md` and `provider/AGENTS.md` advertised by root instructions are absent in this pinned tree. This review references `docs/superpowers/AUDIT-CHECKLIST.md`, especially timestamp conventions (7), silent gates (2/19), canonicalization, fabricated values, production-boundary parity, and R1–R10 at 2281–2315. Source base is the root-assigned revision, not independently asserted to be the running binary. Historical prose is background, not fresh runtime evidence.

Document revision evidence (`git log -1 --format='%h %s' -- <path>`):

- SYSTEM-MAP: `565e8fbe fix: use owner daily-loss controls without requiring a per-trade cap`
- VL-TRADING-RULEBOOK-v1: `565e8fbe fix: use owner daily-loss controls without requiring a per-trade cap`
- AUDIT-CHECKLIST: `dfda15e1 test: isolate session clock fixtures from weekly backfill workers`

SYSTEM-MAP bars/routing sections and RULEBOOK contract/source sections were consulted; they describe NT8 as the single live futures source, contract-aware store reads, and distinct source labels. This worker did not reverify the AddOn, store migrations or live provenance guards. The additional current bridge source directly confirms NT8 cache injection and warns that the newest bar can be forming.

## End-to-end data ownership

[A] `trader/ninjatrader/bars_market_bridge.go:20–31` installs the global `market.FuturesBarsProvider` seam declared at `market/futures_data.go:14`. Its `barsFromCache:44–62` selects the cache tail and adapts it into ordinary `market.Kline`; horizon checks warn rather than refuse. `barsToKlines:72–86` derives scheduled CloseTime and does not remove forming bars. The seam is a startup-owned function variable, not a per-request transport client; tests overwrite it and restore it.

[A] `market.GetWithTimeframes`, `data.go:190–359`, normalizes symbols, inserts the primary timeframe if absent, and fetches each requested series. CME routes only to that seam; a missing provider or empty secondary series is logged and skipped, while missing primary data returns an error. XYZ assets route to Hyperliquid; other symbols use free CoinAnk Binance data. Fetch depth is `clamp(maxConfiguredPeriod+count,200,2500)`, allowing configured long-period EMA warm-up. Current EMA/RSI maps supplement legacy fixed values. Futures skip OI and funding network calls at 332–343. No fallback from the futures branch to Databento or a crypto candle feed exists here.

[A] `calculateTimeframeSeries`, `data_klines.go:162–304`, retains the requested tail's OHLCV, fixed EMA20/50, MACD line, RSI7/14, BOLL20, ATR14, and configured period maps. Insufficient history yields shortened/empty series; scalar indicator helpers often return zero. Input close order is assumed, and forming inputs remain included. `Data` and `Kline` in `types.go` are shared contracts, not source-specific types. Price changes by bars use ordinal lookback and thus assume regular spacing; preloaded `BuildDataFromKlines:723–751` instead uses CloseTime-based lookback and performs no network access.

[A] There are two different freshness concepts. `isStaleData`, `data.go:773–813`, detects five nearly identical closes with all-zero volume. It is not a timestamp-age gate. The legacy `data_freshness.go` helpers provide 90s RTH / 5m ETH age checks and >5% drift within strictly <60s, but a repository Go search found no non-test calls to `CheckDataHealth`, `IsFresh`, or `IsDriftSuspicious`. The actual additional source read, `kernel/stale_data.go`, evaluates snapshot-time feed age using period/open-time/grace semantics and neutralizes only new `open_long/open_short` decisions. Missing SnapshotMs warns and fails open; exits are exempt. This review does not conclude that futures freshness is absent because the legacy helper is unused.

[A] Completed research bars use a separate `BarResolver`, `bar_resolver.go:142–195`. It chooses the first rung producing any completed bars in the requested half-open window, with source/FromTF provenance. Native bars take precedence; the final 1m rung for coarser resolutions is persisted Own1m, not native 1m. For the 1m request only native 1m is used. Caller wiring in `trader/auto_trader_weekly.go:100–127` scopes Own1m to the currently acknowledged contract via `BarsBetweenOn`. No partial-source splicing occurs inside the resolver. `dropForming` tests elapsed period only; it does not prove all source minutes exist.

## Findings and limits, prioritized

### A-grade static defect: weekly resolution disagrees with its consumer's calendar

[A] `bar_resolver.go:61,80–81` excludes native weekly bars explicitly because they straddle the desired Monday week, and chooses native daily bars. But `CompletedBars:185` immediately aggregates those daily bars into 10,080-minute buckets using `AggregateToTF:249–261`: `start = OpenTime / span * span`. Epoch-based seven-day boundaries are Thursday 00:00 UTC, not Monday-governed CME weeks. `CompletedBar:204–206` uses the same generic floor.

[A] This is on an actual production call chain. `trader.weeklyDailyBars:140–156` asks the resolver for `1w` and returns `s.Bars`; its comment claims these are DAILY. `runWeeklyRead:224–232` passes them into `kernel.ComputeWeeklyFacts:44–57`. That function calls `CompletedWeekCandles`, `PriorWeekRefs`, `LastNWOGs`, and `DailySessionBars`. `CompletedWeekCandles`, `kernel/weekly_bias.go:93–149`, assigns each input bar's entire OHLCV to `weekStartMonday(OpenTime)` (73–80). Once multiple days have been fused into a Thursday week, that assignment cannot recover their original dates. The loss of daily/minute granularity also matters to functions expecting weekend/daily evidence, although those downstream helpers were not fully audited here.

[B] Consequently weekly references can span the wrong constituent days, and a completed prior CME week may be unavailable until the epoch week closes. This is a source-level defect, **not a demonstrated erroneous live plan or trade**. No synthetic reproducer was added because root owns repairs/tests.

[A] Existing tests do not falsify it: `market/bar_resolver_test.go` aligns weekly fixtures by epoch floor and asserts counts/source labels. `trader/bar_source_weekly_test.go` checks that history is no longer thin and calls the same resolver twice to compare doc/watch results. It never pins a known Sunday's/Friday's extreme to independently expected Monday-week OHLC. A repair should exercise the production weekly consumer boundary with distinctive daily values around Sunday/Friday and preserve raw-enough data for all weekly-facts consumers.

### B-grade static inconsistency: legacy futures access still enters crypto metadata paths

[A] `GetWithExchange`, `data.go:33–158`, correctly selects NT8 5m/1h for futures, but unconditionally calls Binance OI/funding at 129–137. Its reported 1h change uses twenty 5m bars (~100 minutes), and reported 4h change uses the preceding 1h bar. `Format:469–505` calls these intraday 3-minute and longer-term 4-hour fields. Thus the legacy representation silently changes units when routed to futures. Search found `trader/auto_trader_orders.go` callers at 97,335,502,650,769,833; these bodies were not fully read, so the conditions reaching each call remain outside this report.

[A] `GetBoxData`, `data_klines.go:470–494`, has only XYZ versus CoinAnk branches and no CME branch. `GetKlinesRange`, `historical.go:17–104`, is explicitly Binance-only after generic Normalize. Caller search links GetBoxData to grid regime code, not proof that NT8 day-plan execution uses it. These concerns must not be described as contamination of the correctly separated main `GetWithTimeframes` path.

### B-grade static data-quality/refusal concerns

- [A] `market/api_client.go:103–124` type-asserts raw JSON fields after only a length check; free and paid CoinAnk Kline methods index nine fields without a length check; websocket `handleResponse:105–152` similarly indexes data. `market/historical.go:73–91` indexes/asserts seven fields without guards. [B] Malformed external rows can panic these paths. No hostile traffic or exploit test was performed, and caller recovery/reachability is not established.
- [A] CoinAnk websocket pumps send to bounded channels without a cancellation select (`depth_ws.go:86–101`, `kline_ws.go:92–152`). Closing the socket does not unblock a goroutine already blocked sending. The demux also requires an exact serialized JSON prefix/key order. [B] Slow/abandoned consumers may leave blocked goroutines; reordered valid JSON may be silently dropped. No runtime leak was measured.
- [A] `getOpenInterestData:389–392` fabricates Average as `Latest*0.999`; failed metadata is often rendered as zero, indistinguishable from a measured zero. NofxOS `GetNetFlowRanking:45–96` and `GetOIRanking:50–84` return non-nil results and nil error even if all constituent requests fail. `GetCoinData:79` and `fetchOIRanking:100` accept `success:false` when code is absent/zero. These are concrete availability/value semantics, not proof of current provider failure.
- [A] `calculateTimeframeSeries` loops the original configured EMA/RSI/BOLL period slices and appends into maps keyed by period. Duplicate periods therefore append multiple values per bar into the same key. Caller configuration deduplication was not audited; this is a local precondition risk, not proof user settings can trigger it.
- [A] `AggregateToTF` assumes ascending input, accepts non-divisible src/dst periods, emits partially held buckets, and carries non-OHLCV fields from the first source bar rather than aggregating quote/trade fields. `CompletedBars` certifies temporal closure rather than data completeness; first-rung nonempty history can be arbitrarily short. Provenance is correct about source, not a coverage guarantee.
- [A] `IsCMEFuturesSymbol:50–81` accepts any string containing lowercase `.c.` and known root-dot forms; it does not accept space-qualified `MNQ 06-26`, although `futuresRoot:142–158` does. Normalize preserves lowercase bare CME roots. Actual NT8 cache callers should supply canonical roots; this review did not audit the cache's own canonicalization.
- [A] Timeframe vocabularies differ. `market/timeframe.go` omits 8h/3d/1w, while resolver and series parsing include them. Hyperliquid maps 3m→5m,2h→1h,6h→4h,3d→1d without aggregation, and `getKlinesFromHyperliquid` leaves callers' requested labels intact. Alpaca/Twelve Data have similar substitutions. These are legacy provider semantics, not NT8 resolver behavior.

### C-grade/historical-provider limitations

[A] Databento remains historical/smoke code, not live data routing here. `GetOHLCV:44–60` advertises 1m/1h/1d and explicit contract support but always sends `stype_in=continuous`; `rawBar.toBar:88–124` accepts only rtype33 and names ohlcv-1m. `ResolveContinuous:13–28` advertises non-continuous passthrough but always requests resolution. Client retries network/5xx failures with a shared circuit breaker; 4xx responses record breaker success, and exported Timeout does not update the already-created internal HTTP client. Fixture server validates paths, not query/auth, so current tests cannot prove these advertised request variants work.

[A] `PriceToTicks:29–34` uses `math.Round`, whose halfway rule is away from zero, despite the bankers-rounding comment. `SafeMultiply:64–69` adds ticks rather than multiplying. These helpers lack finite/range checks; most indicator helpers likewise assume positive periods and finite data. `PositionSize:82–88` is a pure floor calculation only; policy ownership stays with the current daily-loss controls.

[A] Alpaca `GetBars:69–131` ignores NextPageToken and returns one page from a 30-day/2-year lookback. Twelve Data `ParseBar:235–271` parses naive timestamps as UTC without using `Meta.ExchangeTimezone`; request methods check JSON status rather than HTTP status. No provider response freshness or current remote API capability was tested.

[A] NofxOS direct transport uses `security.SafeGet` (implementation outside this slice). Optional Claw402 `DoRequest:77–113` is a paid x402 boundary, not a harmless offline getter: it creates a signing function and invokes `payment.DoX402Request`. Auth query stripping truncates at the auth parameter and can drop parameters after it. No payment key/environment was read and no such call was made.

## File-by-file role inventory

The following file notes summarize every assigned source. Exact function boundaries and body call expressions are in `functions.json`; current hashes and full ranges are in `reads.json`.

- `market/api_client.go`: Binance futures REST with 30s client and SET_HTTP_CLIENT hook; no status validation; array shape/types and numeric strings trusted beyond length check.

- `market/bar_resolver.go`: Completed-bars source ladder with provenance, injected clock, native daily preference for weekly, own persisted 1m final rung. Aggregation is epoch-floor including 1w; elapsed period is completion, not constituent coverage. Native weekly exclusion says Monday but math yields Thursday UTC.

- `market/data.go`: Main multi-TF data assembly routes CME to injected NT8 bars, XYZ to Hyperliquid, crypto to CoinAnk; primary mandatory, secondary errors skipped. Legacy GetWithExchange uses futures 5m/1h under 3m/4h semantics and still requests Binance OI/funding. Configured indicators, formatting, normalization and freeze heuristic.

- `market/data_freshness.go`: Pure legacy freshness/drift helpers (90s RTH, 5m ETH, >5% within <60s); future stamps stale. No non-test callers found. Runtime entry stale gate is kernel/stale_data.go, not this module.

- `market/data_indicators.go`: SMA-seeded EMA, MACD line, Wilder RSI/ATR, population BOLL, Donchian and 72/240/500-bar boxes. Exported wrappers expose pure calculations. Except Donchian periods <=0 are not guarded.

- `market/data_klines.go`: CoinAnk exchange/interval mapping and Binance fallback; Hyperliquid remaps intervals without reaggregation. Series include forming input and retain legacy fields alongside configured maps; duplicate configured periods append duplicate values. GetBoxData has no CME routing.

- `market/databento_adapter.go`: Historical Databento Bar to Kline conversion copies millisecond open and OHLCV, leaves CloseTime/other volumes zero; nil for no input.

- `market/decimal_safe.go`: Integer-tick arithmetic; SafeMultiply adds signed ticks, not multiplication. math.Round uses halfway-away rounding despite bankers comment. PositionSize is an optional pure floor helper, not evidence of an active per-trade cap.

- `market/futures_data.go`: Global startup-injected function seam, count-tail ascending bars; provider ownership is trader/ninjatrader. Seam does not itself guarantee closed bars or freshness.

- `market/futures_symbol.go`: CME recognition and per-root point/tick/month tables; arbitrary .c. accepted, bare matching case-insensitive but Normalize preserves case. Qualified NT8 form accepted by futuresRoot but not IsCMEFuturesSymbol. Energy/metals tick lookup intentionally unknown.

- `market/historical.go`: Binance-only historical range paginator, 1500 rows, cursor=last.CloseTime+1, status checked; raw row indexing/type assertions not guarded, no cursor progress assertion or CME provider branch.

- `market/timeframe.go`: Historical validator supports subset through 1d; case/space normalized, invalid errors or Must panic; differs from resolver and kernel vocabulary (8h/3d/1w absent).

- `market/types.go`: Shared Kline/Data/series shapes, configured-period optional maps, legacy zero fields, grid enums and default alert/cleanup literals. GridDirection ratio default .7 and unknown neutral.

- `provider/alpaca/kline.go`: Stock-only IEX raw bars client with config/explicit keys and 30s timeout; fixed 30d/2y window, returns one page while ignoring next_page_token. Timeframe aliases substitute without aggregation.

- `provider/coinank/base_coin.go`: Paid coin/symbol discovery wrappers, product/exchange filters; decode envelope twice redundantly, reject Success false.

- `provider/coinank/coinank_api/base_coin.go`: Free metadata lookup with optional exchangeName/symbol/baseCoin parameters, typed success envelope.

- `provider/coinank/coinank_api/depth_ws.go`: Depth websocket dial, subscribe/unsubscribe, 1024 buffered read channel; blocked consumer can strand send pump even after conn.Close; no post-dial context selection.

- `provider/coinank/coinank_api/kline.go`: Free crypto kline GET transport with context and 30s timeout, success envelope; positional 9-column array indexed unchecked, HTTP status ignored.

- `provider/coinank/coinank_api/kline_ws.go`: Raw websocket pump and prefix-based demux into typed 1024 queues. Requires exact serialized prefix/key order and 9 array values; sends can block without cancellation. Numeric parse errors become zero.

- `provider/coinank/coinank_enum/exchange.go`: Typed CoinAnk exchange wire names; historic and unsupported runtime brokers can still have constants.

- `provider/coinank/coinank_enum/instrument_agg_sort_by.go`: Typed ranking sort field literals across price/OI/flow/liquidation windows; no executable validation.

- `provider/coinank/coinank_enum/interval.go`: Provider-specific case-sensitive timeframe constants from seconds through years; 1M month differs from 1m minute.

- `provider/coinank/coinank_enum/product_type.go`: SWAP and SPOT wire product literals.

- `provider/coinank/coinank_enum/side.go`: to/from historical traversal direction literals.

- `provider/coinank/coinank_enum/sort_type.go`: asc/desc ranking sort literals.

- `provider/coinank/coinank_enum/url.go`: Mutable global/CN paid CoinAnk base URLs.

- `provider/coinank/coinank_http.go`: Paid transport GET/POST with apikey header and context; generic success/page envelopes; shared 30s client, no HTTP status gate or body size bound.

- `provider/coinank/instrument_agg_rank.go`: Visual screener and OI/long-short/liquidation/price/volume ranks; nested success checks; page>=1, size default10, default openInterest descending.

- `provider/coinank/instruments.go`: Last-price ticker and market-cap typed paid wrappers; rich crypto fields returned after Success check.

- `provider/coinank/kline.go`: Paid kline history with optional start, mandatory end, default10 count; same unchecked 9-column response as free path.

- `provider/coinank/liquidation.go`: Exchange aggregates, coin history, symbol history and individual liquidation wrappers; optional order filters; no trading orders placed.

- `provider/coinank/net_positions.go`: Net-long/short candle wrapper, default10 size, typed history envelope.

- `provider/coinank/open_interest.go`: Seven paid OI endpoints for per-exchange/all/series/candles/aggregate/top/market-cap ratios; typed success checks and optional sizes.

- `provider/databento/client.go`: Historical basic-auth GET client, env/default singleton, mutex key update, shared breaker and retry3 for network/5xx; 4xx returned as APIError after breaker success. Public Timeout not reapplied to internal client.

- `provider/databento/contract_calendar.go`: Single-digit year/month suffix parser and third Friday UTC index expiry estimate; no root/product validation; invalid DaysUntilExpiry returns999.

- `provider/databento/historical.go`: NDJSON nested hd timestamp/rtype decode, int64 prices/1e9; GetOHLCV always continuous stype and rawBar requires rtype33 despite advertised 1h/1d/contract support.

- `provider/databento/mock_server.go`: Testing helper in non-test source hosts fixture files at two v0 routes and cleanup closes server; does not validate request query/auth.

- `provider/databento/resolve.go`: Today UTC continuous-to-raw request; always requests even for non-continuous symbols despite passthrough comment; parser picks first entry, rejects not_found/empty.

- `provider/hyperliquid/coins.go`: Once-created coin provider, meta/contexts join by index, descending volume, 24h cache with copies; refresh outside lock can duplicate concurrent fetches; errors do not serve stale cache.

- `provider/hyperliquid/kline.go`: Contextual mainnet/testnet read-only info API; candles/mids/meta, 30s timeout; interval aliases substituted not aggregated; symbol suffix rules case-sensitive and single-step.

- `provider/nofxos/ai500.go`: AI500 retries3 separated2s, available flag, quadratic score sort, USDT normalization. fetch empty returns[] while available/top zero outputs can be nil.

- `provider/nofxos/claw402.go`: Optional paid x402 data transport, wallet key parsed from explicit/env input; endpoint mapping and auth truncation; payment boundary called only when client enabled, never invoked in review.

- `provider/nofxos/client.go`: Deprecated-service defaults still present; mutex config and optional Claw402 gateway, direct requests use security.SafeGet with auth in query; Error returns body only, auth parsing substring-based.

- `provider/nofxos/coin.go`: Per-symbol/batch quant API and bilingual prompt rendering; success=false accepted if code omitted/zero, batch skips failures; price deltas x100, OI percentages pre-scaled; OI map output order unspecified.

- `provider/nofxos/netflow.go`: Four sequential flow rankings; each failure logged and omitted, returns result,nil even all fail. Bilingual tables and top3 retail summary, FetchedAt request-start.

- `provider/nofxos/oi.go`: Top/low OI ranks and legacy symbol wrappers; both requests may fail yet aggregate returns result,nil. success/code compatibility accepts omitted code zero. Percent fields already percent.

- `provider/nofxos/price.go`: Multi-duration ranking fetch with strict Success, map copy then bilingual renderer includes only1h/4h/24h; price ratio x100.

- `provider/nofxos/util.go`: Language enums and signed K/M/B decimal formatting.

- `provider/twelvedata/kline.go`: API-key query client for time-series/quotes, 30s contextual requests; API status string checked, HTTP status ignored. ParseBar parses numeric OHLC strictly but assumes UTC for naive datetime and ignores exchange timezone metadata.

## Tests, graphs, negative results and remaining scope

[A] Fully read additional tests: `market/bar_resolver_test.go`, `market/data_f8_test.go`, `market/data_emaperiods_test.go`, `trader/bar_source_weekly_test.go`, `provider/databento/historical_test.go`, and `provider/databento/resolve_test.go`. Freshness test lines 1–145 were read as excerpts. The F8 test exercises GetWithTimeframes through the injected provider and asserts EMA200 output count; EMA/BOLL/RSI/ATR tests check configured-versus-legacy equality. Databento tests use nested hd records and captured-shape mock fixtures. No tests were executed during this reading dispatch, and no green-suite claim is made.

The July10@7a8adce0 Understand Anything graph was inspected using its actual `filePath` schema. Assigned-path extraction contains 252 nodes and 1, 130 touching edges: 649 imports, 204 contains, 154 exports, 94 calls, 16 tested_by, 13 documents. `historical-subgraph.json` preserves the complete machine extraction; manual inspection covered relevant node summaries and selected dependency edges, not independent confirmation of every historical edge. `market/bar_resolver.go` is absent. Package imports are expanded into edges toward many files, so an import edge to `market/data_freshness.go` does not establish a CheckDataHealth call. Corrections include SafeMultiply's additive semantics, fabricated OI average instead of an actual delta, Hyperliquid timeout 30s versus old 10s, and default timeframe 5m versus old 1h. `graph.json` records these corrections plus directly read current call boundaries. Root owns CGC evidence export; this worker did not query, reindex, delete or treat its historical index as current.

No source file in the assignment remains unread. Additional dependency files were intentionally read only where necessary and are not represented as full subsystem audits. Live NT8 identity/account/contract protection, all grid/order caller guards, security.SafeGet implementation, x402 payment enforcement, real provider schemas beyond captured fixtures, and whether any static concern has occurred in a live plan remain unresolved by this slice. The supplied task authorizes root's separate repair work; this report itself changes no behavior.
