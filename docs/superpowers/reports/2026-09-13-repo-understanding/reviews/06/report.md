# Assignment 06 — crypto broker adapters and history reconstruction

Source base **63968be62e44db2fb07a92883e02127b9064b0be**, read in `/tmp/nofx-understanding-execution-20260913`, whose HEAD and clean porcelain were verified before reading. All **24 assigned files / 9,015 lines** were manually read in full. `functions.json` inventories **200 named functions/methods and 13 closures**, with exact boundaries, local call expressions, purpose and limitations. `reads.json` distinguishes assigned reads from additional dependencies/tests. This is a source review, not an exchange or production certification. No credentials, runtime settings, DB rows or broker accounts were read or changed; no orders, network calls, tests, deployments or source edits were performed.

[A] means a directly read source fact; [B] is its conditional implication. Every issue below is a **latent crypto-path concern**, not an observed MNQ trade failure. The owner describes current execution as NT8 SIM MNQ. The source constructor separately selects `ninjatrader` from crypto exchanges (`trader/auto_trader.go:632–694`), and each historical sync starter is separately exchange-gated (`:867–934`). Thus these files remain callable legacy support, not dead code, but their existence does not establish their use by the owner's current traders. No global claim that all code is SIM-only is justified: e.g. OKX explicitly sends `x-simulated-trading: 0` (`trader/okx/trader.go:232`), while Lighter's current constructor caller passes `false` for testnet (`auto_trader.go:674`). None of that demonstrates a configured live crypto account.

## Rules and freshness

The review used `docs/superpowers/AUDIT-CHECKLIST.md` classes 2/6/7/9/10 and pre-audit R1–R10: fresh source evidence, long/short mirror checks, explicit boundaries, account binding, isolation and no fabricated settlement. No PnL/statistical claims or row counts were made; the corrected-column rule therefore required no DB query. Severity below uses **A = substantial dormant execution/account risk**, **B = reconciliation/interface defect**, **C = robustness/documentation concern**; there is no S-grade current-runtime finding. Static facts are PROVEN by source; venue rejection/fill outcomes remain UNVERIFIED. No archived report is treated as current execution evidence.

Referenced document last-change lines at this base:

- `565e8fbe fix: use owner daily-loss controls without requiring a per-trade cap` — SYSTEM-MAP.md and VL-TRADING-RULEBOOK-v1.md.
- `dfda15e1 test: isolate session clock fixtures from weekly backfill workers` — AUDIT-CHECKLIST.md.

`CLAUDE-canon.md` was read fully: it corrects the stale supplied mirror's manual heartbeat advice; workers did not acquire or touch the main tree. `trader/AGENTS.md` is absent from this tracked worktree despite the root instruction index naming it. The latest daily-loss owner correction was retained; no per-trade cap was proposed.

## Data and control flow

[A] `types.Trader` has 19 methods (`trader/types/interface.go:45–106`). It normalizes many responses through `map[string]interface{}`, so conformance guarantees method signatures, not field types, units, casing, settlement status or completeness. `GetOpenOrders` promises stop, target and ordinary limits; selective cancel methods explicitly promise preservation of the other protective type. `GridTrader` adds placement, per-ID cancellation and depth. Concrete adapters frequently differ from those promises.

[A] Five assigned sync implementations—Aster, Binance, Bitget, Lighter and OKX—take **traderID + exchange account UUID + exchange type + Store**, fetch fills, sort oldest-first, insert `TraderOrder`, insert `TraderFill`, then call `PositionBuilder.ProcessTrade`. Their deduplication uses account UUID plus a **trade ID placed in the order-ID field**. Bitget/OKX fill rows also retain the actual venue order ID, making order-versus-trade identity intentionally uneven across records. `store/order.go:153–166,187–225` verifies existence by exchange UUID/ID, not symbol/trader ID. This review does not assume venue trade IDs collide across instruments; it records the identity boundary requiring confirmation before any repair.

[A] Times enter from venue milliseconds and become UTC Unix milliseconds; Gate closes use seconds (`gate/trader_account.go:155`). Aster/Bitget/Lighter/OKX periodically reread bounded recent pages, nominally last24h with 500/100/100/100 records respectively; Lighter ignores the supplied start time entirely. Binance recovers the DB cursor, adds1000ms, discovers symbols from commission, positions, recent DB fills and realized-PnL income, then uses per-symbol from-ID or time requests. None of these reviewed implementations drains paginated history to exhaustion. Tickers lack a stop channel/context. Binance starts its initial sync in one goroutine and the periodic loop in another, so overlap is possible.

[A] `store/position_builder.go:29–68` forwards crypto fills without an exchange exit reason; it dispatches open/close by action prefix. `:73–120` creates a new position or averages quantity/price into an existing row. The exchange-specific action classification therefore controls accounting state, rather than merely display text.

## File-by-file subsystem map

| Assigned files | Role and distinguishing semantics |
|---|---|
| `aster/trader.go`, `trader_account.go` | ECDSA/ABI/EIP-191 signing;30s client; microsecond nonce; cached tick/lot precision; public prices/depth; balance reconstructed from available balance plus leveraged mark notional; raw fallback on position error. |
| `aster/trader_orders.go`, `trader_sync.go` | One-way BOTH GTC aggressive limits; independent SL/TP orders; broad/selective cancel wrappers; PnL-based classification and last24h fill reconstruction. |
| `binance/futures.go`, `order_sync.go` | SDK/hook setup, server-time offset, constructor hedge-mode mutation; tagged random IDs; incremental history discovery and fill/order/position persistence. |
| `bitget/order_sync.go`, `trader_account.go` | max100 fill normalization and action mapping; cached USDT account equity; crossed/isolated and leverage requests; depth/ticker parsing. |
| `bybit/trader_account.go`, `trader_orders.go`, `trader_positions.go` | Unified-account balance; direct signed closed-PnL HTTP; one-way positionIdx0 market/limit/trigger requests; negative short positionAmt; cached snapshots. |
| `gate/trader.go`, `trader_account.go` | Authenticated SDK context, BTC_USDT conversion, contract multiplier cache; positive base-asset positionAmt; sparse close-history records. |
| `hyperliquid/trader.go`, `trader_orders.go`, `trader_sync.go` | Main/agent wallet checks; standard SDK plus explicit XYZ wire signing; nearest size and five-significant-figure prices; IOC market proxies, triggers and GTC grid orders; on-demand metadata helpers, not a background metadata loop. |
| `indodax/trader.go`, `trader_orders.go` | Spot-only HMAC-SHA512/form transport, monotonic per-instance nonce,5minute pair cache; BUY/SELL limits; explicit short/SL/TP refusal; leverage/margin and stop cancellation no-ops. |
| `kucoin/trader_orders.go` | Quantity-to-lots market entries; fresh-position capped exits; reduceOnly+closeOrder; mark-price triggers; client-ID substring stop classification; sparse/unconverted contract quantities in status/history. |
| `lighter/account.go`, `order_sync.go`, `trader.go` | Wallet→account selection, SDK signing-key comparison,7hour auth token, market-ID lookup; estimated margin and positive position-size maps; fill direction/position-flip reconstruction and periodic persistence. |
| `okx/order_sync.go`, `trader.go` | HMAC timestamp/path/body signing; account-mode detection/switch; lot formatting and instrument conversion; max100 fill history, hedge-mode action classifier and reconstruction. |

Paths in this table are under `trader/`. Individual methods and caller/callee expressions are enumerated in functions.json; those expressions are explicitly syntax-only where receiver type resolution was not independently verified.

## Findings, ordered by conditional consequence

### A1 — Aster quantity/price formatter changes integer values

[A, static PROVEN] `aster/trader.go:199–208` calls `FormatFloat(...,'f',precision,64)` then always trims trailing `0`. With precision0, **100 becomes `1`; 0 becomes empty**. The same helper feeds open/close/SL/TP/grid requests (`aster/trader_orders.go:57–58,204–205,346–347,720–721`). This is a pure deterministic source defect; no venue response is needed to establish the wrong string. [B] A market with zero decimal quantity/price precision can receive a materially different order. Existing Aster mocks only exercise quantityPrecision3 and pricePrecision1/2, so they do not establish this edge case is protected. A focused pure helper regression is appropriate before repair.

### A2 — Aster close and retry semantics lack settlement/idempotency boundaries

[A] `aster/trader_orders.go:159–321` submits GTC close limits without `reduceOnly`, parses acknowledgement, logs closed, and immediately cancels **all** symbol orders. No fill status/remaining exposure check exists between submission and cancellation. SL and TP submissions also omit reduceOnly (`:324–403`). [B] A still-resting close can be canceled by the subsequent sweep, and a stale oversized close/protection may increase reverse exposure under one-way execution semantics. These consequences are conditional; no venue action was observed.

[A] `aster/trader.go:329–369` retries any method—including order POST—on timeout/reset/EOF, copying and newly signing every attempt, up to3times. Assigned open/close order payloads do not provide a stable client key. [B] A response lost after venue acceptance can lead to duplicate submissions. No reproduction against a broker was attempted.

### A3 — Hyperliquid reports fills/IDs not obtained from the exchange

[A] `hyperliquid/trader_orders.go:19–334` discards SDK order responses and returns `status=FILLED, orderId=0`. The close methods then cancel all symbol orders. XYZ handling checks the first inner error but returns only `error`; a resting response or even an empty status array can return success (`:584–742`). [B] Acknowledgement or partial IOC fill can be represented as complete settlement and remove remaining protection.

[A] GTC grid placement creates an ID from `time.Now().UnixNano()` (`:997–1055`), and `CancelOrder` parses the supplied ID and sends it to the venue (`:1059–1075`). That ID is not a venue receipt. This is a direct identity mismatch even without establishing a particular grid caller's runtime use. The actual SDK response contract was not independently audited; the defect here is throwing it away and substituting an unrelated ID.

### A4 — Selective cancellation and open-order completeness differ from interface contracts

[A] Hyperliquid `CancelStopLossOrders` and `CancelTakeProfitOrders` both delegate to cancel-all-symbol behavior (`:337–424`), deleting ordinary entries and the other protective type. The source's comment claiming this is safe is not a verified invariant. Bybit `GetOpenOrders` reads only `orderFilter=StopOrder` (`bybit/trader_orders.go:527–585`), omitting ordinary GTC limits despite the interface promise. Its nonzero RetCode returns empty successful results. Bybit cancellation wrappers also discard business/individual errors (`:384–463`). Aster, Hyperliquid, Indodax, KuCoin bulk cancellation paths often return nil after individual failures. [B] Consumers can interpret incomplete cancellation/snapshots as complete state. This is not a finding about NT8's independent broker snapshot mechanism.

### A5 — Constructor reads are not side-effect-free; OKX local mode can disagree after successful mutation

[A] Binance constructor requests hedge mode (`binance/futures.go:64–108`); OKX detects and may switch to long_short_mode (`okx/trader.go:112–196`). **`setPositionMode` never updates `positionMode`**. If detection returned net_mode and switching succeeds, local field remains net_mode; close payloads condition `posSide` on that cached field (`okx/trader_orders.go:241–244,352–355`). [B] Subsequent requests can use stale-mode fields. Existing OKX margin tests preseed long_short_mode and bypass the constructor, so they do not cover this sequence. This also explains why this review did not instantiate credentialed adapters merely to inspect them.

### A6 — Lighter read-account fallback can break account consistency

[A] `lighter/account.go:15–82` tries stored accountIndex but falls back to the first account if absent; `lighter/trader.go:200–259` initializes from the first account, while TxClient was built for that chosen index (`:148–158`). Index0 is treated as unset. [B] A reordered/partial response lacking the bound nonzero index can make balance/positions describe a different account from the signing client. No account reorder was reproduced or owner binding inspected. A missing explicit target should be an error if this adapter is ever reactivated.

### B1 — Order-first sync dedup prevents repair of partial persistence

[A] Aster `trader_sync.go:46–54,96–132`; Binance `order_sync.go:178–187,231–262`; Bitget `order_sync.go:188–195,232–263`; Lighter `order_sync.go:46–53,99–134`; OKX `order_sync.go:182–189,224–257`: existing order skips the whole fill/position path. Order creation occurs first; fill/position errors are logged and not rolled back. `store/order.go:153–166,187–199` supports independent order/fill idempotency, but the outer skip prevents invoking it for repair. [B] A failure after successful order insertion leaves a persistent missing fill or position update on later scans. Position averaging is not protected by a per-fill marker here, so removing the skip alone is not a safe fix. No DB failure injection was performed.

### B2 — Action/PnL reconstruction loses valid closes

[A] Aster's classifier (`trader_sync.go:149–180`) relies on realizedPnL !=0 even for explicit LONG/SHORT. Binance (`order_sync.go:311–349`) also uses PnL as close evidence, and BOTH falls to opening regardless of PnL. Breakeven closure can therefore become an opening action. Bitget's buy_single/sell_single map always means open_long/close_long (`order_sync.go:111–116`), and OKX net mode always means open_long/open_short (`order_sync.go:116–124`); one-way short/close semantics are not fully represented.

[A] Lighter is internally stronger about direction: it derives action from signed position-before and splits a reversal into close/open IDs with proportional fee (`trader.go:568–688`), placing the close1ms earlier. Nevertheless **every emitted RealizedPnL is0**, while `GetClosedPnL` filters out all zero-PnL records (`:402–448`). Thus that interface method always returns no records on a successful current GetTrades path. This does not mean PositionBuilder can never derive a close; its separate OrderAction is preserved by sync.

### B3 — Unknown data becomes successful zero/empty or unit defaults

[A] Aster GetTrades (`trader_account.go:217–232`) and Lighter GetTrades (`trader.go:493–523`) convert several API/parse failures to empty nil-error results. Lighter startTime is unused and newest limited rows alone are fetched. Gate failed contract lookup defaults multiplier1 (`gate/trader_account.go:86–99`); OKX fill conversion defaults lots as base quantity (`okx/order_sync.go:87–92`), and FormatQuantity metadata failure returns unconverted base size (`okx/trader.go:278–283`). [B] Missing metadata can become wrong-unit data rather than refusal. KuCoin GetOrderStatus exposes executedQty as int64 lots and GetClosedPnL/GetOpenOrders retain contracts; consumers expecting float64 base quantity must adapt explicitly. No blanket claim is made that all current callers assume one unit/type.

### C1 — Concurrency, timeouts and protocol details requiring focused follow-up

[A] Aster getPrecision's final map read follows unlock (`trader.go:142–145`) and can race another cache-miss writer. Bitget GetBalance rereads cachedBalance after unlock (`trader_account.go:16–19`); several other caches return shared maps that callers must not mutate. Lighter reads authToken outside its mutex after token refresh, and Cleanup does not stop its ticker. Hyperliquid's price-significant-figure loop has no finite guard; positive infinity never exits (`trader.go:293–300`). These are static risks, not race-detector or fuzz reproductions.

[A] Hyperliquid XYZ metadata/open-order reads hard-code mainnet while sends honor testnet (`trader_sync.go:62`, `trader_orders.go:440,525–528`), and cancel wire field `a` is assigned the order ID (`:505`) rather than the asset index used in placement. Correct venue handling of that wire remains UNVERIFIED; do not treat the comment “asset index not needed” as proof. Bybit direct closed-PnL and depth use the default HTTP client without timeout (`trader_account.go:133`, `trader_orders.go:693`); this bypasses any SDK-specific transport settings. Bybit closed-PnL side/fee mapping requires venue semantic confirmation: the parser treats Sell as short, which cannot be resolved from this source alone. No web or broker request was made for that unresolved external fact.

## Tests, historical graph and negative evidence

[A] Fully read Aster mock tests, Binance sync diagnostic tests, Hyperliquid metadata race tests and OKX margin-mode recording tests. No tests were executed. Aster POST mocks JSON-unmarshal a **form-encoded production request**, then use defaults; their happy path does not verify precise transmitted fields. Hyperliquid tests establish intended metadata locking paths, not SDK order-response/XYZ behavior. OKX tests exercise actual request builders through a recording transport but initialize mode directly. Binance order-sync tests require live opt-in and are API diagnostics; they were not used as offline proof. Other test paths were inventoried only, not counted as full reads.

[A] Historical Understand Anything graph (July10@7a8adce0) has211 nodes for assigned paths and291 outbound-to-other-path edges. File summaries and representative import/test edges were compared with current source. Corrections: Aster “market open/close” is implemented as **GTC limits**, not market orders; Bybit “pagination parsing” overstates a single list parser with no cursor loop; Hyperliquid trader_sync.go is **on-demand metadata helpers**, not background sync; Lighter API-key generation/registration is **unimplemented**, and closed-PnL is empty by current dataflow. Historical tested_by edges mean a test file exists, not that it covers each risk. CGC was not queried/reindexed by this worker; root owns its exported evidence. No current dynamic call graph is claimed.

No repair was made because root owns the separate implementation lane. The next useful tests would be pure formatter cases, fake transports for rejected/partial acknowledgement and constructor-mode sequences, and isolated in-memory persistence failure tests with explicit per-fill idempotency. None requires a live broker or changing owner accounts. Legacy reactivation should first resolve those contracts; these findings do not justify modifying the owner's active MNQ risk policy or execution configuration.
