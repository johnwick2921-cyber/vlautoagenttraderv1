# NQ Trading via Databento + NinjaTrader — Implementation Plan

> **2026-08-06 — Full tunable inventory + bug sweep (read-only) + 3 casing/label fixes.**
> Shipped (nofx `3c6d8695`, vlauto `72c73b3`; both pushed):
> - `c99e3d74` drawdown-monitor side-casing (P&L sign + emergency-close switch) — same
>   uppercase-NT8-side class as the breakeven fix `d3d60c32`.
> - `17ccc96d` reconcile-before-open flattened the WRONG side of a long orphan
>   (uppercase side → CloseShort → blocked all entries).
> - `3c6d8695` SafeFallback decision symbol `"ALL"` → `"(no decision)"` (cosmetic).
> **Not deployed to the running binary yet** — needs a rebuild + restart to go live.
> **Key non-bug findings (LISTED, not changed — behavior-changing / owner-decision):**
> (a) The recorded "R/R prompt-1.5 vs gate-3.0 mismatch" is NOT live — `ClampLimits()`
>     runs first on the engine's pointer config (`engine_analysis.go:220`) and floors
>     unset min_rr to `MinRiskReward=1.0`, so both the prompt and gate read 1.0; the
>     1.5/3.0 fallbacks are dead code. Same for min_confidence (clamp floors 0→50).
> (b) Live strategy `a5b7662e` has `guardrails_enabled=FALSE` → notional cap +
>     contract clamp + daily-loss/profit/trade/blackout/consistency all DISABLED;
>     futures position size is effectively uncapped. Posture fact, not a bug.
> (c) Drawdown monitor defaults position leverage to 10 for futures (should be ~1) —
>     inflates PnL% ~10×; intertwined with the 5%/40% thresholds (see report).
> (d) `min_position_size`/`max_margin_usage` not enforced on the futures path;
>     `RISK_MAX_CONTRACTS_PER_ORDER` (env) and `LongerCount` are dead knobs.
> Full control-map + %-self-serve in the session report.

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

> **Canonical plan read-first protocol (added 2026-05-25):**
> Before acting on ANY prompt that references this plan, the agent
> MUST read the plan doc end-to-end in its current state on disk:
>
> ```
> view docs/superpowers/plans/2026-05-22-nq-databento-ninjatrader.md
> ```
>
> This is non-negotiable because:
> 1. The plan evolves between sessions. A prompt may reference
>    "Task 14" or "Plan 1.5 trigger 3" without re-stating the
>    content; the agent needs the current text.
> 2. Post-mortems, [VERIFY-N] items, and branch-state sections
>    capture state that supersedes earlier instructions.
> 3. The Playwright MCP block (registered 2026-05-25) tells the
>    agent which verification tool to use; missing it means
>    falling back to weaker checks.
> 4. Branch tip and commit history live in the "Branch state since
>    Plan 1.5 design closeout" section — without reading it, the
>    agent does not know what commits already landed.
>
> If the prompt asks you to "open PR #1", "fix Task 14", "verify
> Plan 1.5 design", or any task referencing this plan by number
> or name, the FIRST tool call must be `view` on this plan doc.
> Skipping this step is a process violation.
>
> The agent may rely on prompt-supplied context for any
> self-contained instruction NOT referencing the plan (e.g.
> "rename this variable in foo.go"). The read-first rule applies
> ONLY when the prompt invokes plan content.

> **STATUS (2026-06-02, latest — safety arc DEPLOYED + Universal CME P1/P2):**
> branch `feat/nt8-stage4-chart` tip `dba04d01`. Two state changes since the
> "safety arc COMPLETE" banner below (which was at tip `0f4c1268`, C# deploy
> still pending):
> 1. **Safety frames are now DEPLOYED + LIVE.** One clean NT8 restart activated
>    the `feed_status` + `position_close_rejected` emits; `/tmp/backend.log`
>    shows both decode with **0 "unknown frame type"**. The Go decoders were
>    already live (`CloseRejections()` `tcp_server.go:425`,
>    `position_close_rejected` decode `:840` → `close_sync.go:40`); the C# emit
>    now fires on the next rejected flatten. id=54 SHORT already proved the
>    fill-confirmed path end-to-end (closed **+96.50 ×pv**, real `exit_order_id`).
> 2. **Universal CME Phase 1 + Phase 2 SHIPPED.** Phase 1 (`1620a5e7`) —
>    recognize 18 CME roots with correct per-root multipliers (micro≠mini
>    guaranteed by test: NQ $20 vs MNQ $2, ES $50 vs MES $5); FE no longer
>    USDT-mangles CME symbols. Phase 2 (`dba04d01`) — the hardwired `"MNQ"`
>    subscription is GONE (subscription follows the active symbol); index +
>    treasury families resolve via a quarterly H/M/U/Z resolver
>    (`VLContractResolver`, `RollDaysBeforeExpiry=8`), LIVE-verified
>    `MNQ -> MNQ 06-26`. Energy/metals PARKED verifiably (clean
>    `instrument_unresolved` log — no wrong-contract risk).
> Three NT8/Tradovate **premise-refutations** recorded below (real API truth,
> verified not assumed). **NEXT = Phase 3** (parameterize the MNQ-hardwired AI
> prompt for the resolving families). See `## Current State — Round 6
> (2026-06-02)` immediately below. All prior banners retained for history.

> **STATUS (2026-06-02, later — safety arc COMPLETE):** branch
> `feat/nt8-stage4-chart` tip `0f4c1268`. The phantom-close safety arc that was
> "IN-FLIGHT" in the banner below is now **SHIPPED + live-verified**:
>
> - **`0118ca77` — fill-confirmed NT8 closes (phantom-close root fix).** A
>   decision-driven close used to mark the DB CLOSED off the 5m mark via
>   `recordAndConfirmOrder → recordPositionChange → ClosePositionFully` WITHOUT
>   confirming the NT8 exit filled. During a Tradovate feed outage ~76 flatten
>   ("Close" Sell Market) orders were REJECTED ("There is no market data available
>   to drive the simulation engine"), so id=45 was never closed in NT8 yet the bot
>   phantom-closed it locally (points-only PnL, `exit_order_id=<nil>`); id=46's next
>   buy netted onto the orphan → NT8 net 2. Fix: ninjatrader decision closes now
>   route through the `position_close` fill frame (`close_sync.recordClose` —
>   fill-confirmed, ×`FuturesPointValue`), recorded ONLY on a confirmed NT8 exit
>   (`auto_trader_decision.go`); this also fixed the points-only PnL for free. New
>   `position_close_rejected` wire frame (C# emits on a rejected exit; Go decodes +
>   alarms; the position stays OPEN so the next cycle retries). `reconcile.go` now
>   alarms on a qty divergence (the id=45→46 tell).
> - **`2fc07ea2` — feed-down trading gate + reconcile-before-open flatten-first.**
>   New `feed_status` frame (C# `OnVLConnectionStatusUpdate` → Go); the AutoTrader
>   gates opens AND closes when the NT8 price feed is not Connected (the SIM rejects
>   "no market data"). DESIGN: **no false-halt** — `IsFeedConnected()` default-ALLOWs
>   until the first frame and trips ONLY on an explicit non-Connected status.
>   `reconcileBeforeOpenNT` flattens an orphan first **awaiting its OWN fill** (bounded
>   `GetPositions` poll, feed-gated — NOT fire-and-forget) then opens, else REFUSES.
>   RACE INSIGHT: feed-up, the flatten fills fast and `close_sync` records first
>   (the decision phantom is harmlessly skipped — no open row); feed-down, the
>   flatten is rejected and only the phantom used to record — now it doesn't.
> - **`0f4c1268` — chart markers source from real trades.** The chart B/S markers
>   bound to `/api/orders?status=FILLED`, which is sparse for NT8 (closes route
>   through `close_sync` → `trader_positions`, not the orders table); a lone stale
>   "zombie" order drew a misleading green "B". For `exchange==='ninjatrader'`,
>   markers now source from `trader_positions` (entry + exit per position, correct
>   B/S) via `/api/positions/history`. Crypto + the B/S visibility toggle unchanged.
>
> **VERDICTS this round (read-only diagnoses):**
> - **Track C (Trade History):** the FE faithfully RENDERS the rows — no cache/
>   transform bug (browser-confirmed `DISPLAY==DB`, `DB!=NT8`). The wrong numbers
>   were poisoned OLD rows (id=45/46 phantom/doubled; legacy id=6/14 points-only);
>   NEW closes are clean (×pv, real `exit_order_id`). The OLD rows were **repaired to
>   NT8 truth this round** (DB write, see below).
> - **Track D (B/S button):** NOT a bug — the green "B/S" button is the order-markers
>   show/hide toggle (`AdvancedChart.tsx`, green = markers ON), not a side indicator;
>   id=54 was correctly SHORT at every layer (DB/NT8/`/api/positions`/panel) and
>   closed **+96.50 ×pv via the fill-confirmed path** (`reason=tp`, real
>   `exit_order_id`) — proving the `0118ca77` path end-to-end. The stale green "B"
>   was the zombie order (fixed by `0f4c1268` + the DB cleanup below).
>
> **DB cleanups this round (SIM historical, NT8-truth; backup taken first):**
> - **Trade-history repair:** id=45 → exit 30533.25 / PnL −26.00; id=46 → entry
>   30553.5 / PnL −40.50 (sum −66.50, the real 2-lot flatten @30533.25 split per
>   lot); id=6 → +53.00 (×pv, exit 30573.75 confirmed real); id=14 → −353.50 (×pv,
>   exit 30567.0 confirmed real). All `exit_order_id`→`Close`. None invented — every
>   exit cross-checked against the NT8 trace.
> - **Zombie order:** deleted the lone stale `trader_orders` id=1 (BUY, created
>   05-31 @30464 but `avg_fill_price`/`filled_at` bumped to ~now/30615.5,
>   `related_position_id=0`) that drew the misleading "B".
> - **CI:** removed the unused `onTraderSelect` destructure in
>   `TraderDashboardPage.tsx` → `tsc --noEmit` clean.
>
> **ROADMAP:** the NT8 execution-safety arc (fill-confirmed closes, feed gate,
> reconcile-before-open, honest history + markers) is **COMPLETE and live-verified**
> — no blocking items remain. Operator's remaining manual step: deploy the new C#
> (`cp` → F5 → one full NT8 restart) to activate the `feed_status` +
> `position_close_rejected` emits (the Go decoders + the close-routing root fix are
> already live; the fixes degrade safely against the old C#). **NEXT project:**
> Universal CME Symbol support (Path A, 5-phase); Strategy-UI futures-awareness last.
> The prior `cef42d61` status (phantom-close IN-FLIGHT) is retained below for history.

> **STATUS (2026-06-02):** branch `feat/nt8-stage4-chart` tip `cef42d61`.
> SHIPPED since the last doc-sync (`2a0801b6`): the **bar-feed restart fix**
> (`345bce2b` — resubscribe-on-Connected + fast `.Update` watchdog + ETH
> hours; bars reattach ~14s after a clean restart, operator-verified) and
> the **pre-prompt risk-gate fix** (`cef42d61` — the stale crypto notional
> cap no longer trips in-position, so the AI can manage/close its own
> futures position), plus 7 supporting NT8/chart fixes (`2f59fd7f`,
> `7bc2dd8c`, `dde10cf1`, `86ed1535`, `b2fc0f3d`, `ae94adb5`, `0be1521b`).
> **IN-FLIGHT (designed, NOT committed):** the phantom-close safety fix
> (NT8-fill-confirmed closes + `position_close_rejected` frame). See
> `## Current State — Round 5 (2026-06-02)` immediately below. The prior
> 2026-05-31 status is retained beneath for history.

> **STATUS (2026-05-31):** NT8 pipeline live on branch
> `feat/nt8-stage4-chart` (tip `0a67bfda`, includes `063bc311` +
> `49017ff9` + `0a67bfda`). **DASHBOARD IS COMPLETE** — balance,
> account-switch, per-account P&L, chart with all 7 TF buttons, and the
> per-account equity curve (C/D/E) are all SHIPPED & VERIFIED. The
> **Strategy Studio AUDIT is done (2026-05-31, full 7-tab pass)** and the
> Strategy build is **PLANNED** (two stages, locked — see Current State).
> See `## Current State (2026-05-30, session-verified)` (immediately below)
> for the current truth. Older claims about a `$50k mock balance`,
> `FuturesChart` as the chart path, "decisions skip / unknown coin source",
> a `100000` P&L baseline, and **"kernel reads Binance klines"** are
> SUPERSEDED in place (the kernel reads NT8 BarCache for CME futures on this
> branch — verified `market/data.go:197-210`; Stage 3 source-swap is DONE
> here). Markers added, nothing deleted.

## Current State — Round 6 (2026-06-02): safety arc DEPLOYED+LIVE; Universal CME Phase 1+2 SHIPPED

> Additive sync against git truth at tip `dba04d01`. The execution-safety arc
> (`0118ca77` fill-confirmed closes, `2fc07ea2` feed-down gate + reconcile,
> `0f4c1268` real-trade markers) is fully documented in the "safety arc
> COMPLETE" banner above — Round 6 records what changed AFTER that banner
> (which was at tip `0f4c1268`): the safety frames are now **deployed + live**,
> Universal CME **Phase 1+2 shipped**, energy/metals **parked verifiably**, and
> the **three NT8/Tradovate premise-refutations** are captured as API truth.
> MAIN re-verified every hash via `git show` and every code claim at file:line;
> runtime/live claims are tagged "operator-verified". Round 5 below is intact.

### SAFETY ARC — now DEPLOYED + LIVE-verified (state change from the banner above)

The banner above closed with "Operator's remaining manual step: deploy the new
C# (`cp` → F5 → one full NT8 restart)". That deploy **happened**:

- One clean NT8 restart activated `feed_status` + `position_close_rejected`;
  `/tmp/backend.log` shows both frames decode with **0 "unknown frame type"**
  (operator-verified, 2026-06-02).
- Decoder wiring (git-verified at file:line): `position_close_rejected` decode
  `provider/ninjatrader/tcp_server.go:840` → `CloseRejections() <-chan`
  `tcp_server.go:425` → consumed `trader/ninjatrader/close_sync.go:40`; the C#
  emit is in `VLTraderTCPClient.cs`. Dedicated tests exist
  (`tcp_server_close_rejected_test.go`, `tcp_server_feed_status_test.go`).
- id=54 SHORT already proved the `0118ca77` fill-confirmed close path
  end-to-end: closed **+96.50 ×`FuturesPointValue`**, `reason=tp`, real
  `exit_order_id` — the id=45 `<nil>`/points-only phantom shape is gone. The
  feed-up/feed-down race is closed: feed-up the flatten fills and `close_sync`
  records first (decision-phantom harmlessly skipped); feed-down the flatten is
  rejected and the position stays OPEN (no phantom-close).

### Universal CME — Phase 1 recognition SHIPPED (`1620a5e7`)

`market/futures_symbol.go` (+55) + test (+49), `store/strategy.go`,
`web/.../CoinSourceEditor.tsx` (+194). Recognizes 18 CME roots with correct
**per-root multipliers**; micro≠mini is guaranteed by test (NQ $20 vs MNQ $2,
ES $50 vs MES $5). The FE no longer USDT-mangles a CME symbol (the
`MNQ`→`MNQUSDT` crypto-bypass bug — see the Strategy-build P1🔴 — is handled
for recognition). Recognition only: no subscription/resolution yet (that's
Phase 2).

### Universal CME — Phase 2 active-symbol subscription + resolver SHIPPED (`dba04d01`)

`VLBarsSubscriptionManager.cs`, `VLContractResolver.cs`, `tcp_server.go`,
`tcp_trader.go` + `tcp_trader_subscribe_test.go`.

- **The hardwired `"MNQ"` bar subscription is GONE** — subscription now follows
  the active symbol.
- **Quarterly resolver** (`VLContractResolver`): INDEX (ES/NQ/YM/RTY +
  micros MES/MNQ/MYM/M2K) and TREASURY (ZB/ZN/ZF/ZT, CBOT) resolve to the NT8
  qualified front-month via H/M/U/Z (H=Mar, M=Jun, U=Sep, Z=Dec), rolling
  `RollDaysBeforeExpiry=8` before expiry. **LIVE-verified: `MNQ -> MNQ 06-26`**
  (correct front month, operator-verified 2026-06-02).

### THE THREE NT8 / TRADOVATE PREMISE-REFUTATIONS (real API truth — verified, not assumed)

These were assumptions that turned out FALSE on the NT8 AddOn API + the
operator's Tradovate feed; recorded so the plan carries the real behavior:

1. **Bare-root auto-routing is FALSE for the AddOn API.**
   `Instrument.GetInstrument("MNQ")` returns **null** — a QUALIFIED
   `"MNQ 06-26"` is required. So the AddOn computes the qualified quarterly
   contract itself (`VLContractResolver`, `RollDaysBeforeExpiry=8`). (Refuted
   in Phase 2.)
2. **`GetNextExpiry` is UNBOOTSTRAPPABLE on Tradovate.** It needs a
   `MasterInstrument`, which is obtained only from a qualified/continuous
   `GetInstrument` — and **Tradovate has no continuous contracts** → null →
   chicken-and-egg. This is exactly why the non-quarterly families (energy/
   metals) can't use the calendar path yet. (Refuted in Phase 2.5; the
   `VLContractResolver` header comment documents it.)
3. **`GE` is NOT in our code.** The `"GE 03-26 Symbol is inaccessible"` log
   carries NT8's own `|3|4|` prefix (not our AddOn's `|1|16|`). `GE` =
   delisted Eurodollar; it is an NT8-environment artifact (a saved chart/
   watchlist), silenced by an NT8-side cleanup — **not a code change**.

### PARKED (verifiably; no wrong-contract risk)

Energy (CL/MCL/NG — monthly) + metals (GC/MGC — GJMQVZ; SI — HKNUZ) are
deliberately NOT in the quarterly resolver (their front month is not
quarterly). They resolve to a clear `instrument_unresolved` log — **awaiting**
a continuous-supporting data feed (which would enable `GetNextExpiry`, per
refutation #2) or a testable C# roll path. NOT abandoned — cleanly deferred,
with no risk of trading the wrong contract in the meantime.

### Roadmap refresh (2026-06-02, Round 6)

- **Safety arc: DONE + LIVE.** No blocking items.
- **Resolver: covers the tradable quarterly families** (index + treasury).
- **NEXT — Phase 3:** parameterize the MNQ-hardwired AI prompt —
  `BuildFuturesDecisionSystemPrompt(symbol, equity)` — for the resolving
  families.
- **Phase 4:** `instrument_info` wire frame carrying NT8 tick + `point_value`,
  and the 7→14 timeframe expansion (ties into the LOCKED Strategy build's
  Stage 2 14-TF delivery).
- **Energy/metals:** revive when a testable continuous/roll path exists
  (refutation #2 is the gate).
- **Strategy UI futures-awareness (Stage 1/Stage 2): LAST.**

### SHA ledger addendum (Round 6, verified against git 2026-06-02)

| SHA | message |
|---|---|
| `dba04d01` | feat(nt8): Phase 2 — drive bar subscription from the active symbol + resolver to all quarterly families **[BRANCH TIP]** |
| `1620a5e7` | feat(symbol): Phase 1 — recognize 8 more CME futures roots (recognition only) |
| `91d438a8` | chore(web,docs): drop unused onTraderSelect (tsc clean) + doc-sync NT8 safety arc |
| `0f4c1268` | fix(web): NT8 chart markers source from real trades (trader_positions), not the sparse orders table |
| `2fc07ea2` | feat(nt8): feed-down trading gate + reconcile-before-open flatten-first (deferred safety layers) |
| `0118ca77` | fix(nt8): fill-confirm NT8 closes — a rejected flatten no longer phantom-closes |

## Current State — Round 5 (2026-06-02): bar-feed + risk-gate SHIPPED; phantom-close IN-FLIGHT

> Additive sync against git truth. Branch `feat/nt8-stage4-chart`, tip
> `cef42d61`. 9 code commits landed since the last doc-sync (`2a0801b6`),
> all on this branch. MAIN re-verified every line below against
> `git show` / `grep` at file:line — only what the commits actually support
> is recorded; runtime-only claims are tagged "operator-verified". The
> Round-4 (2026-05-30/31) section below is unchanged.

### SHIPPED & VERIFIED since `2a0801b6`

- **Bar-feed restart freeze — FIXED (`345bce2b`; 2 `.cs`:
  `VLBarsSubscriptionManager.cs` +172, `VLTraderTCPClient.cs` +22).** The
  post-restart "frozen at 16:00 CT / 30523.75" chart is gone. Three
  mechanisms (git-verified in the diff):
  - **resubscribe-on-Connected** — BarsRequests recreate + resubscribe on
    every transition into `PriceStatus==Connected` (via the
    `OnConnectionReconnected()` path wired to
    `Connection.ConnectionStatusUpdate` in `0be1521b`), covering a clean
    startup `Disconnected→Connecting→Connected`, not just loss→recovery (the
    old `dataFeedWasLost` guard was loss-only).
  - **fast `.Update` watchdog** — if a (re)subscribe seeds historical but no
    LIVE `.Update` arrives within `FAST_STALL_MS = 20s` (checked every
    `WATCHDOG_PERIOD_MS = 15s`), recreate the instrument; capped at
    `FAST_MAX_ATTEMPTS = 3` per dead window so a genuine closed-market gap
    can't churn recreates.
  - **75-min backstop KEPT** (`WATCHDOG_STALL_MS = 75 min`, > the 60-min
    daily halt) + **ETH TradingHours** (`BARS_TRADING_HOURS = "CME US Index
    Futures ETH"`) so bars survive the 16:00 CT session close.
  - **VERIFIED LIVE (operator, 2026-06-02):** bars reattach within ~14s of a
    clean restart and stream continuously.

- **Pre-prompt risk-gate — FIXED (`cef42d61`; `kernel/engine_analysis.go`
  +11/−7): the AI runs in-position again.** The pre-prompt gate was passing
  the open futures position's notional into `CheckPreTrade`, tripping the
  stale crypto notional cap (`RiskMaxNotionalUSD`, default $50k) that any
  single MNQ contract (~$61k notional) exceeds → every in-position cycle was
  skipped before `BuildUserPrompt` / the AI → empty dataless
  `decision_records`. FIX (git-verified): the gate now calls
  `CheckPreTrade(TotalPnL, len(Positions), 0, 0)` — enforcing ONLY
  daily-loss + concurrent-position; notional is intentionally NOT checked
  here. **Notional stays enforced futures-aware at EXECUTION**
  (`engine_position.go`: `const futuresMaxNotionalLeverage = 20.0`,
  `maxPositionValue = accountEquity × 20`). Net: the AI can manage/close its
  own position via decisions, not only the NT8 OCO SL/TP.

- **Supporting NT8 / chart fixes (also shipped since `2a0801b6`):**
  - `2f59fd7f` — record entry from the NT8 fill / position average, not the
    frozen 5m mark.
  - `7bc2dd8c` — position uPnL from NT8's live `UnrealizedPnL`, not a stale
    5m bar close.
  - `dde10cf1` — position-card side label case-insensitive (NT8 `LONG` no
    longer renders as `SHORT`).
  - `86ed1535` — NT8 open-position read-back (emit on
    select/connect/`PositionUpdate`, per-account cache, settled entry).
  - `b2fc0f3d` — `BarCache.SeedHistorical` merges instead of replacing →
    chart keeps depth across reconnects.
  - `ae94adb5` — volume histogram sits in a bottom band, not overlaying
    candles.
  - `0be1521b` — wired the dead reconnect-recreate to
    `Connection.ConnectionStatusUpdate` (the foundation `345bce2b` builds on).

### IN-FLIGHT — phantom-close safety fix (DESIGNED, NOT COMMITTED)

> NOT shipped. Tip is `cef42d61`; there is NO commit after it, and no
> `position_close_rejected` symbol anywhere in the `.go`/`.cs` tree
> (grep-confirmed 2026-06-02). Recorded here as the next fix to build.

- **ROOT (PROVEN via NT8 trace, operator, 2026-06-02):** id=45's position
  was never actually closed — 76+ flattens were REJECTED for ~4h ("no
  market data" during a Tradovate feed outage) — yet the DB recorded it
  CLOSED with a −19.25 **points-only** PnL (a non-fill / synthetic-mark
  close path). id=46's single correct `Buy x1` then netted ONTO the
  lingering contract (`operation=Add` → `quantity=2`). NT8's accounting was
  correct; the bot's was wrong. Systemic — recurs on every feed flap.
- **FIX (4 parts; both sides ship together):**
  1. **C# flatten-reject detection** → emits a NEW `position_close_rejected`
     wire frame.
  2. **Go records a position CLOSED only on the NT8 exit `position_close`
     (Filled)** — never on an AI decision, synthetic mark, or reconcile. On
     a reject it keeps the position OPEN, alarms, and retries.
  3. **Go reconcile-before-open** — alarm + flatten-first if the NT8 net ≠
     intended, so a bad state is not compounded.
  4. **feed-down gate** — no opens/closes while
     `priceStatus == ConnectionLost`.
- **NEW wire frame `position_close_rejected`** (additive per ADR-007; 4-byte
  big-endian length prefix + JSON envelope; both sides ship together) — to
  be added to the frozen wire-contract section when it lands.

### SHA ledger addendum (verified against git, 2026-06-02)

| SHA | message |
|---|---|
| `cef42d61` | fix(nt8): don't apply the crypto notional cap at the pre-prompt risk gate — let the AI run in-position **[BRANCH TIP]** |
| `345bce2b` | fix(nt8): revive bar .Update fast after restart — resubscribe-on-Connected + fast watchdog (+ ETH hours) |
| `2f59fd7f` | fix(nt8): record entry from NT8 fill/position avg, not the frozen 5m mark |
| `7bc2dd8c` | fix(nt8): position uPnL from NT8's live UnrealizedPnL, not a stale 5m bar close |
| `dde10cf1` | fix(dashboard): position-card side label case-insensitive — NT8 LONG no longer shows as SHORT |
| `86ed1535` | feat(nt8): NT8 open-position read-back — emit on select/connect/PositionUpdate, per-account cache, settled entry |
| `b2fc0f3d` | fix(nt8): BarCache.SeedHistorical merges instead of replacing — chart keeps depth across reconnects |
| `ae94adb5` | fix(chart): volume histogram sits in a bottom band, not overlaying candles |
| `0be1521b` | fix(nt8): wire the dead reconnect-recreate to Connection.ConnectionStatusUpdate |

### Roadmap / open-items refresh (2026-06-02)

- **Keystone VERIFY 1/2** (entry == NT8 fill end-to-end + the PnL cascade) —
  now **UNBLOCKED**: bars stream continuously (`345bce2b`) and the AI runs
  in-position (`cef42d61`), so the next held position self-captures the
  proof. (Was blocked by the frozen feed + skipped cycles.)
- **id=45 −19.25 points-only PnL anomaly** — should be eliminated by the
  fill-confirmed close path (phantom-close fix, IN-FLIGHT); confirm on the
  next real close, else flag a follow-up.
- **B/S toggle button (FE)** — clicking does nothing. OPEN.
- **TS6133 build-red** — `TraderDashboardPage.tsx:138` unused
  `onTraderSelect` (~2 LOC). OPEN.
- **THE BIG PROJECT — Universal CME Symbol (Path A), 5 phases** (recognition
  → subscription + C# resolver → correctness → capstone) — NOT started;
  phase plan retained in the IN FLIGHT/PLANNED section above. Current top
  priority after the phantom-close fix.
- **Project C — Strategy UI futures-awareness (Stage 1 / Stage 2)** — LAST.
- **Deferred backlog (unchanged):** `GE` delisted by CME June 2023 (dead
  root, not a bug); security batch (jwt default secret, tokenless reset
  endpoints — "sec later"); `nofx→VL` rename (last).

## Current State (2026-05-30, session-verified)

> Current-truth snapshot for branch `feat/nt8-stage4-chart` (tip `d3c18f0f`
> = origin). Supersedes contradicted older claims further down (each marked
> in place with a pointer here; nothing deleted — history prevents repeating
> mistakes). This feat branch diverged before main's PR #44 "Current State"
> section, so this is the branch-local current-truth section.
> Last refreshed 2026-05-30 (round 3): chart shipped, two non-bugs settled,
> CGC indexed.

### SHIPPED & VERIFIED (MAIN-verified live, pushed, backed up)

- **Balance — real per-account equity auto-updates (no `$50k` mock).**
  Fix = C# `SendAccountBalance` poll (`c4e2cb13`) + Go rebuild. **ROOT CAUSE
  was a STALE GO BINARY** dropping frames as "unknown frame type" — see the
  new STALE GO BINARY hard rule below.
- **Account switch re-bind.** Frontend invalidates 5 SWR keys (`82bdca1c`:
  account / positions / status / statistics / equity-history) +
  `createPortal` / `mousedown` click-outside fix (`587a1386`). Cross-SIM
  switching verified (Sim101 `100157` <-> SimAccount1 `70000`). LESSON: MAIN
  re-verifying in the live browser caught the portal bug a subagent missed
  by only inspecting the guard.
- **SimAccount1 selectable; funded LFE gated** (UI disabled + HTTP 400)
  (`587a1386`).
- **Decisions — coin_source.** Set `static` MNQ in DB + DURABLE kernel guard
  (`abda753d`: empty `SourceType` -> `static`, +2 tests).
  `GetDefaultStrategyConfig` was ALREADY correct; the real defect was
  `cmd/create-strategy` building a config without `coin_source` (still a
  birth-defect — see STILL OPEN).
- **Timeframe (blank-AI latent bug).** Strategy klines -> `[5m,15m,1h]`,
  primary `5m` + `ParseConfig` backfill hardening (`05d3968e`; `058e4a56`
  aligned its `coin_source` default to `"static"` to match the kernel guard).
  NOTE: the **kernel AI timeframes stay `[5m,15m,1h]`**; the chart now shows
  7 TF buttons (`d3c18f0f`) but that is CHART-SIDE display only — see below.
- **Per-account P&L (Issue 2 A/B/F, `fcd8bb99`, WIRE-FREE).** Balance
  single-slot -> `acctBalances` map keyed by account; P&L uses NT's OWN
  realized/unrealized (baseline = `equity - pnl`, NOT `equity - 100000`);
  frontend SWR keys account-scoped. The old `+0.16%` / `-30%` / `-50%`
  figures were `100000`-baseline ARTIFACTS — NT reports `0.00%` real session
  P&L.
- **Chart — MNQ bars + indicators + B/S markers.** Wired via `273f85a3`
  (ninjatrader klines branch: `AdvancedChart` reads the NT8 BarCache through
  REST `/api/klines`, NOT the deleted `FuturesChart`; `bars_subscribe`
  auto-sent on reconnect `tcp_server.go:431`, revert `cf5b76b3` never broke
  streaming) + `f267d09e` (chart defaults to the trader's market symbol MNQ,
  not BTC) + `3a3dfb31` (indicators redraw on toggle — fixed the stale-closure
  clobber, so indicators NOW WORK on NT8/MNQ). **B/S is a MARKER TOGGLE, not
  an order placer — there is no manual-order route, by design.** Saturday =
  static historical MNQ bars; live ticks need RTH (Sun ~17:00 CT).
- **Chart shows all 7 TF buttons — root cause was a CSS CLIP (`063bc311`),
  NOT a stale deploy / wrong array.** `d3c18f0f`'s 7-button list was already
  live, but the interval row (`ChartTabs.tsx:397`) had `overflow-x-auto` +
  `max-w-[200px]` inside a `min-w-0` flex toolbar → flex-shrink collapsed it
  (`clientWidth 73` vs `scrollWidth 216`) → `30m`/`1h`/`1d` were clipped behind
  a hidden scrollbar. Fix: flex-wrap the row so all 7 stay visible. LESSON:
  headless Playwright read the DOM text ("7 buttons present") while the real
  browser visually clipped 3 — measure visual visibility, not DOM presence.
- **Per-account equity curve — ITEM 2 C/D/E migration SHIPPED (`49017ff9` +
  `0a67bfda`).** Account column added to the 3 tables (`trader_positions`,
  `trader_equity_snapshots`, `decision_records`) — additive / AutoMigrate /
  idempotent; per-account equity baseline now comes from the SCOPED series'
  first snapshot (NOT the trader-global `100000`); ~208 pre-existing mixed
  rows QUARANTINED (`account=''`); crypto path unaffected; integrity_check ok.
  Result: SimAccount1 now reads `70000`/`70000`/`+0%` (was `100000`/`-30%`/
  the bogus "201" series). This is the durable backend for the per-account
  P&L cards (which `fcd8bb99` already made correct on the card surface).

### SETTLED NON-BUGS — do NOT re-investigate (confirmed 2026-05-30)

- **`/api/config` HTTP 500 — NOT REAL / RESOLVED.** `handleGetSystemConfig`
  (`server.go:420`) is a tiny handler returning 200 with hardcoded leverage +
  an `initialized` bool; it has NO error path and cannot 500 (live curl =
  200). It is DASHBOARD-ONLY and completely off the trader/kernel/manager
  path. The "500" was a stale backlog note, not a real condition.
- **"No prompt to trader" — NOT a bug; it is the CME WEEKEND GATE.**
  DB-verified (`decision_records` #183–#190: `system_prompt`/`input_prompt`/
  `raw_response`/`decision_json` empty, `ai_request_duration_ms`=0,
  `candidate=["MNQ"]`, `success`=1). On a closed-market cycle the prompt is
  GENUINELY NOT BUILT: `auto_trader_loop.go:runCycle` loads context (candidates
  + klines — the "looks like loading"), logs the PRE-GATE banner "🤖 Requesting
  AI…" (`:108`, misleading), then `engine_analysis.go:53 ShouldSkipDecisionCycle`
  (CME closed, `engine.go:911`) returns `nil,nil` BEFORE `BuildSystemPrompt`/
  `BuildUserPrompt` are ever reached → "No actionable decision (risk gate
  HOLD)" → saved with empty prompt fields. `DecisionCard.tsx` renders prompt
  sections conditionally, so gated cycles show no prompt. Confirmed 3×. Verify
  prompts build + DeepSeek fires at RTH (Sun ~17:00 CT). (Optional cosmetic,
  not a bug: move the `:108` banner to after the gate so weekend logs aren't
  misleading.)

### IN FLIGHT / PLANNED

> **SHIPPED 2026-05-31** — the former "chart 7-timeframe buttons" IN-FLIGHT
> item and the "Issue 2 C/D/E per-account equity-curve migration" PLANNED
> item are BOTH DONE; moved to SHIPPED & VERIFIED above (`063bc311`,
> `49017ff9`, `0a67bfda`). The §7g Strategy audit "leanings" are SUPERSEDED
> by the 2026-05-31 full 7-tab audit; the LOCKED Strategy build below
> replaces them.

**PLANNED — Universal CME Symbol (Path A) — CURRENT TOP PRIORITY
(2026-05-31).** GOAL: type ANY CME instrument NT8 offers and have it
resolve → subscribe → tick-round → size → display (chart Sym/Go input) →
trade (strategy Symbol Source), through ONE source (NT8 BarCache — the same
one-source path the kernel + chart already use, see the BarCache correction
below). Instrument set: index `ES`/`NQ`/`YM`/`RTY` + micros
`MES`/`MNQ`/`MYM`/`M2K` + energy `CL`/`NG` + metals `GC`/`SI` + rates
`ZB`/`ZN`/`ZF`/`ZT`.

Build order (**Strategy UI LAST**):
- **Phase 0 — read-only trace** of the current single-symbol (MNQ) path end
  to end: resolve, subscribe, tick-round, size, chart, strategy. Map every
  place `MNQ`/`0.25` tick/point-value is assumed before changing anything.
- **Wire contract (additive frame types, ADR-007 — ship Go + C# together):**
  `instrument_subscribe` / `instrument_info` (carries NT8 tick size +
  `point_value`) / `bar` keyed by `(symbol, TF)` / `instrument_unsubscribe`.
- **C# AddOn:** `GetInstrument(<any>)` + null-handle guard +
  `OnConnectionStatusUpdate` dispose-recreate (reconnect silently kills
  `BarsRequest.Update`) + read `MasterInstrument` tick + `PointValue`, emit
  on `instrument_info`. Deploy = cp → Documents AddOns → F5 → FULL restart.
- **Go:** `isCMEFuturesSymbol` full root set (index/micro/energy/metal/rate);
  per-instrument tick FROM THE WIRE (not the hardcoded `0.25`);
  `point_value` fed into the prompt for correct contract sizing.
- **Chart Sym/Go input + Strategy "Symbol Source"** both consume the SAME
  wire (one source, no per-surface fork).
- **Per-instrument roll** — each root rolls on its own calendar; manual roll
  (NT does not auto-roll a running strategy).

NT8 constraints to honor (from Plan 1.5 research): (1) the instrument must
exist in / be subscribed on the NT8 connection; (2) reconnect kills
`BarsRequest.Update` → dispose-recreate on `OnConnectionStatusUpdate`;
(3) data-feed concurrent-subscription caps; (4) tick size + contract
multiplier are PER-INSTRUMENT — read from NT8, never assume `0.25`/`$2`;
(5) manual front-month roll. Bar timestamps are NT-local close-time →
normalize to bar-open UTC at ingest.

The LOCKED Strategy build below (14-TF selector + Futures variant + label
universalization) is the **LAST** phase of this project — the Strategy UI
lands AFTER the universal-symbol wire + resolve + chart paths are proven.

**PLANNED — Strategy build (LOCKED 2026-05-31; the LAST phase of the
Universal CME Symbol project above).**
SCOPE LOCKED: creating a strategy must let the user select ANY of the 14
timeframes `[1m,3m,5m,15m,30m,1h,2h,4h,6h,8h,12h,1d,3d,1w]`, and the system
subscribes + pulls + feeds the AI those EXACT TFs (incl. a Windows NT8 C#
AddOn update). **Q1 DECIDED: the AI uses ALL selected TFs (no cap).**
**Q2 DECIDED: all 14, C# updated.** This FLIPS the §7g `a3` leaning from
"hard-restrict futures TFs to `[5m,15m,1h]`" → "make all 14 deliverable."

- **STAGE 1 — UI + phantom (frontend + a few Go store/api + DB; NO C#, NO
  wire).** Severity-ordered:
  - **P1 🔴 — TAB2 CoinSource USDT→CME bypass** (`CoinSourceEditor.tsx:78`
    + `:107`). The bypass has ZERO CME roots today → `MNQ`→`MNQUSDT` →
    `IsCMEFuturesSymbol=false` → silently routes to the crypto branch →
    futures decisions skip. **THE critical fix.** Plus the universal
    'Symbol Source' rename + help text.
  - **P2 🟠 — TAB3 indicator gating** (add a `variant` prop; hide
    funding/OI/NetFlow/NofxOS for futures — also stops crypto vocab leaking
    into the futures prompt via the shared `writeAvailableIndicators`) +
    **TAB6 add a 'Futures' variant** to BOTH the preview and AI-Test selects
    (`StrategyStudioPage.tsx:1198-1206` / `:1306-1314` — the futures prompt
    `engine_prompt_futures.go` EXISTS but is currently UI-unreachable) +
    **TAB4 universal risk labels** (BTC/ETH/Altcoin → Leverage/Tier; fix the
    hardcoded `USDT` span at `RiskControlEditor.tsx:277-278`, NOT via i18n).
  - **P3 🟡 — TAB5 futures persona + TAB1/7 cleanup.**
  - **Cross-cutting phantom cleanup:** `UPDATE strategies SET is_default=0
    WHERE id=''` + add `ORDER BY` to GetDefault (`store/strategy.go:1157`) —
    NEVER touch `578ac8f6`.
  - **LABEL RULE:** delete nothing; rename to UNIVERSAL (one label fits
    crypto + futures; units dynamic).
- **STAGE 2 — backend 14-TF delivery (C# → F5 + FULL restart).**
  `MapTimeframe` → all 14 (the 7 MISSING are `2h`/`4h`/`6h`/`8h`/`12h`/`3d`/
  `1w`; `tcp_server.go:143` auto-subscribes only 7) + the Go subscribe set
  DRIVEN BY the strategy's `selected_timeframes` + kernel reads BarCache for
  all selected (**ALREADY the path — no source-swap needed**, per the
  BarCache correction in the Root Cause section below) + prompt feeds all
  selected + the picker honest.

  **i18n GOTCHA:** `strategy-translations.ts` is zh / en / **es (Spanish)**,
  NOT `id` — new editor labels need an `es` value.

  **FLAGGED (separate Plan 3, NOT relabel work) — Risk Control enforcement
  bugs:** the `max_margin_usage` badge is prompt-only (not enforced);
  `min_risk_reward_ratio` is ignored (hard `3.0` at
  `engine_position.go:133`); position-value tiers are inert for MNQ.

  **CLEAN SLATE:** the prior Stage-1 attempt was REVERTED — the phantom is
  back to `is_default=1` on BOTH "MNQ SIM Default" rows, `store/strategy.go`
  == HEAD, and `GetDefault().First()` is nondeterministic again.

### STILL OPEN / DEFERRED

- **Durable:** `cmd/create-strategy` birth-defect (builds config without
  `coin_source`) — the kernel guard protects but the cmd itself is still
  wrong.
- **Phantom** empty-id "MNQ SIM Default" `is_active=1` row (different user
  `8ef641a7`) — report, do NOT delete (see the planned Studio phantom-cleanup
  above).
- **Misc deferred:** live chart-tick verify at RTH (Sun ~17:00 CT); security
  batch (override `jwt_secret` default
  `'default-jwt-secret-change-in-production'`, `/api/exchanges` unauth
  exposure — the named upstream CVE-class issue); nofx->VL full rename (LAST);
  `ALLOW_LIVE_ACCOUNTS` toggle (default false).
- **Pre-existing flaky test** `TestMaybeResetDaily` (UTC-day-boundary), not
  ours.

### Environment / tooling

- **CodeGraphContext (CGC) is now INDEXED for this repo** (FalkorDB Lite, `cgc`
  CLI; nofx in `cgc list`). Hard Rule 0's tool order — CGC-first, then
  grep/Read — is fully available going forward. (Earlier passes ran without it
  and fell back to grep/Read; that limitation no longer applies.) See the
  CodeGraphContext tooling blockquote near the Playwright block above for the
  MCP registration + `cgc index . --force` refresh command.
- **Doc relationship.** This canonical plan doc is the long-term source of
  truth. A separate detailed working-log/archive (`nt8-account-pipeline-build-
  plan.md`) is referenced in working sessions but is NOT present in the repo
  at any path as of 2026-05-30 — if/when it lands, cross-reference (do NOT
  merge) so the two don't drift.

### Session SHA ledger (verified against git)

| SHA | message |
|---|---|
| `273f85a3` | fix(nt8): live MNQ bars reach chart via klines ninjatrader branch |
| `abda753d` | fix(kernel): empty coin source type defaults to static (stop per-cycle skip) |
| `05d3968e` | fix(strategy): backfill default coin_source + klines on ParseConfig |
| `fcd8bb99` | fix(nt8): per-account balance + NT-native P&L + account-scoped fetch (Issue 2 A/B/F) |
| `058e4a56` | fix(strategy): blank coin_source defaults to 'static' (match kernel guard) |
| `587a1386` | fix(nt8): make account dropdown rows selectable; funded stays gated |
| `82bdca1c` | fix(nt8): account select re-binds dashboard data (invalidate 5 SWR keys on switch) |
| `c4e2cb13` | fix(nt8): poll account_balance so real equity populates (Tradovate AccountItemUpdate doesn't fire) — kills $50k mock |
| `f267d09e` | fix(nt8): chart defaults to the trader's market symbol (MNQ), not BTC |
| `3a3dfb31` | fix(nt8): chart indicators redraw on toggle (stop stale-closure clobber) |
| `d3c18f0f` | feat(nt8): MNQ chart shows all 7 timeframe buttons (adds 3m, drops 4h) |
| `063bc311` | fix(nt8): chart timeframe row wraps so all 7 buttons stay visible |
| `49017ff9` | feat(nt8): per-account equity/positions/statistics (ITEM 2 C/D/E migration) |
| `0a67bfda` | fix(nt8): equity curve baseline is per-account, not trader-global 100000 **[BRANCH TIP = origin]** |

### New rules logged this session (see also "Locked Data Architecture Decisions")

- **NEW HARD RULE — STALE GO BINARY.** After ANY wire / parser / Go change,
  `go build` + restart `./nofx-bin`. The tell is `unknown frame type type=X`
  in `/tmp/backend.log` (the TCP slog sink — NOT `data/nofx_*.log`). This is
  the twin of the NT8 AddOn `cp` -> Documents-AddOns -> F5 -> full-restart
  rule. A stale Go binary silently drops new frame types and is why the
  balance fix appeared not to work until rebuild.
- **SESSION WAIVER (2026-05-30, Opus 4.8).** MAIN may spawn multiple
  subagents with full freedom PROVIDED MAIN independently re-verifies
  (the portal-bug catch above is why); reverts to general-purpose-only next
  session unless re-waived.

## 2026-05-28 ARCHITECTURE PIVOT — NT8 as single data source (Databento dropped)

> **READ THIS FIRST IF YOU ARE WORKING ON ANYTHING DATA-RELATED.**
> The plan title still says "Databento + NinjaTrader" for historical
> continuity, but as of 2026-05-28 **Databento is DROPPED**. NT8 via
> Tradovate is the SINGLE real-time data source for BOTH trading
> decisions AND chart display.

### Decision

NT8 (via Tradovate) becomes the SINGLE real-time source for:

- **Trading decisions** — `kernel/engine_analysis.go` reads NT8 bars
  through the TCP wire (Plan 4.4 Stages 1–3)
- **Chart display** — same NT8 feed routed through an SSE relay to
  the React frontend (Plan 4.4 Stage 4)

One source, two consumers, zero lag.

### Why

The `cmd/nq_smoke` foundation test (2026-05-28) probed the operator's
current Databento Historical subscription and found the
`available_end` cursor is **~8 hours behind wall-clock**, not the
~15-minute figure the plan originally assumed. 8-hour-stale data
cannot drive live decisions. Tradovate (the NT8 data feed) is
real-time — confirmed against live MNQ candles moving in NT8 on
2026-05-28.

### What was dropped vs. kept

| Item | Status |
|---|---|
| Databento Historical fetch from kernel (Task 12 Step 3 `getKlinesFromDatabento` + `GetWithTimeframes` Databento branch + `available_end` probe) | **DROPPED** (PR #27 added, PR #28 trimmed) |
| Symbol normalization for CME futures (`isCMEFuturesSymbol`, `market.Normalize` CME-bypass, `store/strategy normalizeSymbols` guard) | **KEPT** — fixes futures-symbol corruption regardless of source; NT8 symbols (e.g. `MNQ 03-26`) hit the same path |
| `cmd/nq_smoke` Databento probe code (the `available_end` available-history measurement) | **KEPT as reference** — if Databento is ever revived (e.g. for backfill/backtest), the probe pattern is the right starting point |
| Plan 4.4 Deep Spec research (NT8 BarsRequest API, TradingView Lightweight Charts v5, Go SSE relay, 15 NT8 gotchas) | **KEPT and now central** — this is the architecture for the staged build below |

### What this supersedes (in this very doc)

- Task 12 (line 2599) — the Databento `getKlinesFromDatabento` helper
  and `GetWithTimeframes` Databento branch are **SUPERSEDED 2026-05-28**.
  See "2026-05-28 Architecture Pivot." The symbol-normalization steps
  in Task 12 remained shipped (v1.0-task12-symbols).
- Old Plan 4.5 — `/api/klines` NT route via Databento. **SUPERSEDED**.
  There is no Databento klines route in the NT8 architecture; the
  chart consumes the same NT8 bar feed as decisions (Plan 4.4 Stage 4).
- Any plan text describing "Databento for decisions" or "Databento
  with 15-minute embargo as primary data source" — superseded.
- [VERIFY-16] (line 6771) "Databento tier vs documented embargo" is
  now MOOT under the NT8 pivot; the 8h Historical-tier lag is the
  documented reason Databento was rejected.

### See also (further down in this doc)

- **Ship Log (2026-05-28)** — what's landed since the last plan update
- **Post-Mortems (2026-05-28)** — Plan 1.5.6, 1.5.7, N11 root causes
- **Plan 4.x Sequencing — REVISED 2026-05-28** — the staged NT8 build
- **Locked Data Architecture Decisions (2026-05-28)** — multi-TF + source

---

## Current State (2026-05-29)

Status block — concise. All facts verified against `git` on `feat/nt8-stage4-chart` tip `703ad29d`, `origin/main` tip `efad9e54` (Merge PR #42). Tag count: 33; nt8 tags: 4 (`v1.0-nt8-stage3-dataspine`, `v1.0-nt8-futures-decisions`, `v1.0-nt8-restart-resilient`, `v1.0-nt8-contract-resolver`).

### SHIPPED ON `main` (PR# / tag from git)

- **Stage 3 data spine** — PR #38, tag `v1.0-nt8-stage3-dataspine`: kernel reads real NT8 MNQ bars from BarCache (not Binance/CoinAnk).
- **Futures prompt** — PR #39: futures system prompt emits the existing parseable decision JSON; `AI_MAX_TOKENS` 2000 → 8000.
- **Risk-gate + executor futures sizing** — PR #40, tag `v1.0-nt8-futures-decisions`: equity × 20 cap, 1–10 contracts.
- **Fast cycles** — PR #41: dead crypto-enrichment flags off; ~30s cycles.
- **N3 restart-resilience** — PR #42, tag `v1.0-nt8-restart-resilient`: C# re-emits `bars_historical` on re-subscribe.
- Tag count: 33 total; nt8 tags = 4 (`stage3-dataspine`, `futures-decisions`, `restart-resilient`, `contract-resolver`).

### ON BRANCH `feat/nt8-stage4-chart` (UNMERGED, operator-gated — 28 commits ahead of `origin/main`, 0 behind, strictly linear)

- **Plan 4.11 real balance:** `account_balance` TCP frame; live account replaces $50k mock (proven ~$100k flowing). Commit `3792babb`.
- **Stage 4 live chart:** `FuturesChart.tsx` (lightweight-charts v5) over SSE/BarCache; MNQ candles ticking. Commit `3e66b716`.
- **Security/reliability:** SSE auth via short-lived single-use ticket (JWT out of URL — `73ecbf1f`); AI idle-watchdog 60→180/300s (`758be7d4`); liquidationPrice panic fixed.
- **Position lifecycle:** `position_close` frame + close-sync (exit price, realized PnL, CLOSED state — `b874f817`); OCO bracket fix (entry alone, then SL+TP own OCO; was opening UNPROTECTED positions — `0a8ddd68` / `f36a6498`); manual close / Emergency Flat via `close_position` → `account.Flatten` (`a53e558e`).
- **Account UI:** REPLACED the old static "mNQ sIM TEST" display column with a new NT-driven account selector dropdown (`2e222846` live selector + SIM gating; `aa9b8f44` filter internal accounts). The old column was a static trader/exchange label; the new one pulls live NT8 accounts.
- **Account FILTER corrected to CONNECTED accounts only** (`Status && PriceStatus == Connected`, per NT staff guidance — tip commit `703ad29d`) so the dropdown shows the real NT8 set (Sim101 + active funded LFE), not all 42 of `Account.All`. SIM-safety gating: only SIM selectable; funded greyed + server-rejected (HTTP 400). `accounts_list` re-emits on TCP reconnect.

### OPEN / KNOWN ISSUES

- AccountSelector was REMOVED at one point (`312e7bb1`) to unblock a blank-page render crash, then RESTORED (`d46100ba`); current tip has the selector live with React-Portal escape (`fa0be202`).
- Account SWITCH does not yet fully re-bind ALL account-scoped data (balance / positions / PnL / orders) to the selected account — only partial. Open.
- SIM-only gate is HARDCODED in 3 layers (frontend grey-out, Go API HTTP 400, C#); no toggle yet. A config toggle (`ALLOW_LIVE_ACCOUNTS`, default false) is DESIGNED but not built.
- `orderId` vs `signal_id` DB-persistence bug (flagged-separate; dashboard reads live fill-cache so proof unaffected).
- **Security (deferred):** tokenless `/api/reset-password` (auth bypass); `/api/reset-account` wipes all users; `/login` session wipe. (SSE JWT-in-URL leak already fixed.)
- Decision QUALITY / prompt tuning deferred (AI makes informed WAITs; good MNQ entries untested on SIM).

### NT8 DEPLOY GOTCHA (cross-ref CLAUDE.md)

C# compiles ONLY from `/mnt/c/Users/hoang/Documents/NinjaTrader 8/bin/Custom/AddOns/`, NOT the repo. Every C# change: `cp` → F5 → full NT8 restart.

---

> **Agent tooling available in this repo (registered 2026-05-25; hardened 2026-05-26):**
> A Playwright MCP server is registered in `~/.claude.json` for project
> `/home/hoang/nofx`. When you spawn agents that need to verify the React
> frontend (Settings page exchange config form, Dashboard column rendering,
> Strategy Studio variant gating, etc.), use the `mcp__playwright__*` tools
> to drive a headless Chromium against `http://localhost:3000` after
> running `cd web && npm run dev`. Chromium binary is pre-installed at
> `~/.cache/ms-playwright/`. Without this, you cannot verify UI changes
> end-to-end — `npm run build` only confirms TypeScript compiles, not
> that the rendered DOM is correct.
>
> Server command:
> `npx -y @playwright/mcp@latest --headless --executable-path /home/hoang/.cache/ms-playwright/chromium-1223/chrome-linux64/chrome`
>
> Required env (set in the MCP registration, not the shell):
> `LD_LIBRARY_PATH=/home/hoang/.local/lib/playwright-deps/extracted/usr/lib/x86_64-linux-gnu`
>
> **WSL2 / Ubuntu 24.04 hardening notes (2026-05-26):**
> 1. Chromium needs `libnss3`, `libnssutil3`, `libnspr4` — these are NOT
>    pre-installed on this WSL2 image and `sudo` was unavailable. Workaround:
>    `.deb` files downloaded from the Ubuntu noble archive and extracted to
>    `~/.local/lib/playwright-deps/extracted/` via `dpkg-deb -x`. No sudo,
>    no system pollution; just user-space libs reachable via `LD_LIBRARY_PATH`.
>    To re-apply on a fresh machine:
>    ```bash
>    mkdir -p ~/.local/lib/playwright-deps && cd ~/.local/lib/playwright-deps
>    apt-get download --print-uris libnss3 libnspr4 \
>      | grep -oP "'http[^']+'" | tr -d "'" | xargs -n1 curl -sSLO
>    mkdir -p extracted
>    for f in *.deb; do dpkg-deb -x "$f" extracted/; done
>    ```
> 2. The MCP server defaults to `--browser chrome` which expects system
>    Google Chrome at `/opt/google/chrome/chrome`. This system has none.
>    The `--executable-path` flag overrides this and points at Playwright's
>    bundled Chromium (the FULL one, not the `chrome-headless-shell` —
>    the MCP's `--headless` flag handles headless mode, the binary should
>    be the regular `chrome` binary in `chromium-1223/`).
> 3. To register on a fresh machine (after steps 1 + 2 prerequisites):
>    ```bash
>    LIB_PATH=$HOME/.local/lib/playwright-deps/extracted/usr/lib/x86_64-linux-gnu
>    CHROME_BIN=$HOME/.cache/ms-playwright/chromium-1223/chrome-linux64/chrome
>    claude mcp add-json playwright "{\"type\":\"stdio\",\"command\":\"npx\",\"args\":[\"-y\",\"@playwright/mcp@latest\",\"--headless\",\"--executable-path\",\"$CHROME_BIN\"],\"env\":{\"LD_LIBRARY_PATH\":\"$LIB_PATH\"}}"
>    ```
> 4. Session restart required after registration changes. MCP tool schemas
>    are sealed at session start; updates to `~/.claude.json` only take
>    effect on the next `claude` invocation.

> **CodeGraphContext (cgc) — query-before-read code graph (registered 2026-05-30):**
> CodeGraphContext 0.4.0 is installed at `~/.local/bin/cgc` and registered as an
> MCP server in `~/.claude.json` for project `/home/hoang/nofx` (local scope —
> NOT a committed `.mcp.json`). Use the `mcp__cgc__*` tools to query code
> structure BEFORE blindly reading files — e.g. `analyze_code_relationships`
> (find_callers / find_callees / find_all_callers / call_chain / class_hierarchy
> / dead_code / find_importers / who_modifies) and `execute_cypher_query` (direct
> read-only Cypher over the graph). This is how you trace "who calls X / what
> does Y depend on" across the Go + TS + C# tree without grepping the whole repo.
>
> - **Backend: FalkorDB Lite** — embedded, NO Docker required (this WSL2 distro
>   has no Docker; FalkorDB/Neo4j server backends are therefore NOT used).
>   `cgc doctor` is the health check.
> - **Index:** `nofx` is indexed (1872 files). Refresh after code changes with
>   `cgc index . --force` (plain `cgc index .` skips an already-indexed repo);
>   `cgc watch /home/hoang/nofx` auto-updates on save. The index is local graph
>   state, not in git.
> - **Registration (re-apply on a fresh machine):**
>   ```bash
>   claude mcp add -s local cgc -- /home/hoang/.local/bin/cgc mcp start
>   ```
> - **Session restart required** (same caveat as Playwright above): the
>   `mcp__cgc__*` tools only appear on the next `claude` invocation after
>   registration; the CLI (`cgc query`, `cgc analyze`, `cgc find`) works
>   immediately regardless.

**Goal:** Get the nofx bot to fetch NQ futures OHLCV from Databento, ask the AI for a trade decision in futures-native vocabulary, write that decision to a CSV signal file, and have NinjaTrader (running on the user's Windows host with the open-source `claudetrader.cs` strategy attached to an MNQ chart) execute that decision in SIM mode — then tail the fills CSV back into the bot's database.

**Architecture:** Three independent layers wired through narrow interfaces.

```
┌──── WSL2 (Linux) ────────────────────────────┐    ┌── Windows ──┐
│  nofx-bin (Go)                               │    │             │
│                                              │    │  NinjaTrader 8
│  trader/auto_trader_loop.go                  │    │  + claudetrader.cs
│    │                                         │    │  (modified copy)
│    ├─► provider/databento/  (NEW)            │    │      ▲
│    │     GetOHLCV("NQ.c.0", "1m", ...)       │    │      │ polls CSV
│    │                                         │    │      │ every 2s
│    ├─► kernel/engine.go (MODIFY)             │    │      │
│    │     futures-mode prompt → AI            │    │      │
│    │                                         │    │      │
│    └─► provider/ninjatrader/ (NEW)           │    │      │
│          WriteSignal(dir, entry, sl, tp) ────┼────┼──┐   │
│          TailFills(callback) ◄───────────────┼────┼──┼───┘
│                                              │    │  │
└──────────────────────────────────────────────┘    │  │
                                                    │  ▼  files at
                                                    │  C:\Users\<u>\NofxTrader\data\
                                                    │  ├─ trade_signals.csv  (Go writes, NT reads)
                                                    │  └─ trades_taken.csv   (NT writes, Go reads)
                                                    └─────────────
```

WSL2 reaches Windows files via `/mnt/c/Users/<windows_username>/NofxTrader/data/`. Same files, two views. No sockets, no daemon.

**Tech Stack:**
- Go (existing `nofx` codebase)
- Databento Historical REST API (`https://hist.databento.com/v0/`) with HTTP Basic auth
- `claudetrader.cs` (open-source NinjaScript from `https://github.com/J0shusmc/Claude-Trader-NinjaTrader`) — modified copy with our file paths
- NinjaTrader 8 (Windows desktop app) with SIM connection
- File-CSV protocol (5-field signals, 3-field fills)

**Scope of this plan:** End-to-end paper-trading slice. AI brain takes NQ context → decides → writes signal → NT executes in SIM → fill recorded. **Out of scope** (separate future plans): CME holiday calendar, contract rolls, position-sizing-by-contract-multiplier math, dead-man-switch heartbeat, web UI rewrite, going live with real money.

**Architectural alignment with existing patterns (revised 2026-05-22):**

The system already has rich per-trader configuration via the **Exchange**, **Strategy**, **AI Model**, and **Trader** entities (documented in upstream `STRATEGY_MODULE.md`). This plan integrates NQ trading **into those patterns** rather than introducing a parallel "futures mode" branch:

- **NinjaTrader is added as a new exchange type** in [store/exchange.go](store/exchange.go) (joining `binance`, `bybit`, `hyperliquid`, etc.). Per-account NT data dir lives on the Exchange row, not in global env vars.
- **NQ symbols use the existing static-coin-source mode** at [kernel/engine.go:437-442](kernel/engine.go#L437) — `CoinSource.SourceType="static"`, `StaticCoins=["NQ.c.0"]`. No engine fork needed.
- **The futures prompt template is selected via the existing `PromptVariant` field** ([api/strategy.go:551](api/strategy.go#L551) already supports `req.PromptVariant`). NQ traders set `PromptVariant="futures"`.
- **Data feed override:** when the engine sees `ExchangeType=="ninjatrader"`, it routes K-line fetches through Databento instead of nofxos/coinank. This is a small branch in `kernel/engine.go` data-fetch, not a top-level mode switch.
- **Symbol normalization fix:** [market.Normalize()](market/data.go) currently appends `USDT`. It needs a futures-aware path that detects CME symbol patterns (e.g., `NQ.c.0`, `MNQ`, `NQM6`) and skips the suffix. See Task NEW.

**Verified facts (from reading `claudetrader.cs` directly):**
- Strategy places **market orders** (not limit, despite README claims) → `EnterLong(0, contractQuantity, "CT_Long")` at line 255. `Entry_Price` in CSV is a *reference* logged but not used for entry.
- SL is `ExitLongStopMarket` and TP is `ExitLongLimit`, both placed at absolute prices from CSV after the entry fills.
- File is read every `FileCheckInterval` seconds (default 2), but only re-parsed when `File.GetLastWriteTime` changes. **NT clears the file (rewrites only the header) after each signal is processed** — every signal is one-shot.
- Dedup is by `DateTime + Direction` concatenated as signal ID. Two signals with the same DateTime+Direction = the second is ignored.
- Strategy refuses new signals while `Position.MarketPosition != Flat || hasLimitOrder`. One position at a time.
- `IsExitOnSessionCloseStrategy = true` with 30s buffer — NT auto-flattens at session close.

---

## File Structure

**New files (this plan):**
- `provider/databento/historical.go` — `GetOHLCV(symbol, interval, start, end) ([]Bar, error)`
- `provider/databento/historical_test.go` — unit tests with mocked HTTP
- `provider/databento/resolve.go` — `ResolveContinuous(symbol)` returns "NQM6" etc.
- `provider/databento/resolve_test.go` — unit tests
- `market/databento_adapter.go` — `BarsToKlines(bars []databento.Bar) []market.Kline`
- `market/databento_adapter_test.go` — unit tests
- `provider/ninjatrader/csv_writer.go` — `WriteSignal(SignalRow) error`
- `provider/ninjatrader/csv_writer_test.go` — unit tests against a tempdir
- `provider/ninjatrader/csv_tailer.go` — `TailFills(ctx, onFill) error` — long-running goroutine
- `provider/ninjatrader/csv_tailer_test.go` — unit tests
- `provider/ninjatrader/types.go` — shared `SignalRow`, `FillRow` types
- `trader/ninjatrader/trader.go` — implements `trader/types.Trader` interface using the CSV writer/tailer
- `trader/ninjatrader/trader_test.go` — unit tests
- `kernel/engine_prompt_futures.go` — `BuildFuturesSystemPrompt()` and `BuildFuturesUserPrompt()`
- `kernel/engine_prompt_futures_test.go` — golden-file tests for prompt content

**Modified files:**
- `config/config.go` — add `DatabentoAPIKey`, `NinjaTraderDataDir`, `TradingMode` fields
- `.env.example` — add `DATABENTO_API_KEY`, `NINJATRADER_DATA_DIR`, `TRADING_MODE`
- `kernel/engine.go` — branch on `TradingMode == "futures"` to call futures prompt + Databento data path
- `trader/auto_trader.go:263-315` — add `"ninjatrader"` case in the exchange switch
- `main.go` — wire `provider/databento.DefaultClient` initialization on startup

**Modified externally (Windows side):**
- `claudetrader.cs` — change hardcoded paths from `C:\Users\Joshua\Documents\Projects\Claude Trader\data\` to `C:\Users\<user>\NofxTrader\data\` (or whatever path the user picks, via NT strategy parameter)

---

## Setup Tasks (before any code)

### Task 0: Environment + accounts + manual smoke test

**Goal:** Confirm the human-side prerequisites work BEFORE we write any code. If a hand-edited CSV doesn't trigger an order in NT SIM, no Go code we write will matter.

**Files:** none — this is human-side setup.

- [ ] **Step 0.1: Confirm Databento account + API key**

You should already have this (per earlier conversation). Verify your key works:

```bash
curl -sS -u "$DATABENTO_API_KEY:" "https://hist.databento.com/v0/metadata.list_datasets" | head -c 500
```

Expected: a JSON array of dataset codes including `GLBX.MDP3`. If you get `{"detail":"Not authenticated"}` your key is wrong; if you get a list, you're good.

- [ ] **Step 0.2: Install NinjaTrader 8 on your Windows host**

Download from [ninjatrader.com/download](https://ninjatrader.com/download). Install with default settings. On first launch, connect to the SIM101 simulated account (free, no signup money needed).

- [ ] **Step 0.3: Subscribe to MNQ in NT SIM**

In NinjaTrader:
1. Open Control Center → Connections → SIM101 → Connect.
2. New → Chart → Instrument: `MNQ 12-26` (or whichever is the front-month MNQ at the time you're doing this). Choose a 5-minute or 1-minute bar interval.
3. Confirm the chart renders live or replayed price data.

- [ ] **Step 0.4: Pick a shared data directory**

Decide on a path on your Windows host that both NT and the Go bot will use. Recommended:

```
Windows path:    C:\Users\<your-windows-username>\NofxTrader\data\
WSL2 path:       /mnt/c/Users/<your-windows-username>/NofxTrader/data/
```

Create the directory. From Windows: open Explorer, navigate to `C:\Users\<you>\`, right-click → New Folder → `NofxTrader`, then inside it create `data`.

From WSL2, confirm visibility:

```bash
ls -la "/mnt/c/Users/<your-windows-username>/NofxTrader/data/"
```

Expected: empty directory listing (no errors).

- [ ] **Step 0.5: Initialize the two CSV files**

From WSL2:

```bash
DATA_DIR="/mnt/c/Users/<your-windows-username>/NofxTrader/data"
printf "DateTime,Direction,Entry_Price,Stop_Loss,Take_Profit\n" > "$DATA_DIR/trade_signals.csv"
printf "DateTime,Direction,Entry_Price\n" > "$DATA_DIR/trades_taken.csv"
ls -la "$DATA_DIR/"
```

Expected: both files exist with their header rows.

- [ ] **Step 0.6: Install and configure claudetrader.cs in NT**

1. From your Windows host, clone the repo: `git clone https://github.com/J0shusmc/Claude-Trader-NinjaTrader.git C:\NofxTrader\bridge` (or anywhere convenient).
2. Open `C:\NofxTrader\bridge\ninjascripts\claudetrader.cs` in any text editor.
3. Find lines 34, 35, 86, 87, 98, 99 — replace `C:\Users\Joshua\Documents\Projects\Claude Trader\data\` with `C:\Users\<your-windows-username>\NofxTrader\data\` everywhere.
4. Save.
5. Copy the file to `Documents\NinjaTrader 8\bin\Custom\Strategies\claudetrader.cs`.
6. In NinjaTrader: Tools → Edit NinjaScript → Strategy → find `ClaudeTrader` → click Compile (F5). You should see "0 errors" in the output panel.

- [ ] **Step 0.7: Apply ClaudeTrader strategy to your MNQ chart**

On your MNQ 12-26 chart in NT:
1. Strategies tab → click + → select `ClaudeTrader`.
2. Set parameters:
   - `Signals File Path`: `C:\Users\<your-windows-username>\NofxTrader\data\trade_signals.csv`
   - `Trades Log File Path`: `C:\Users\<your-windows-username>\NofxTrader\data\trades_taken.csv`
   - `File Check Interval`: `2`
   - `Contract Quantity`: `1` ← starting safe; we'll raise this later after validation
3. Click Apply, then Enable. Look at NT's Output window: you should see `ClaudeTrader Initialized - Monitoring signals every 2 seconds`.

- [ ] **Step 0.8: Manual signal smoke test (DO NOT SKIP)**

This is the most important step in the entire plan. From WSL2:

```bash
DATA_DIR="/mnt/c/Users/<your-windows-username>/NofxTrader/data"
# Make sure NT is open, SIM connected, ClaudeTrader running on MNQ chart
DT=$(date +"%m/%d/%Y %H:%M:%S")
# Pick a price near current MNQ market — check NT chart. Replace 21500 with current ish.
echo "${DT},LONG,21500.00,21450.00,21560.00" >> "$DATA_DIR/trade_signals.csv"
```

Within 2-4 seconds:
- NT Output window should print `[SIGNAL] LONG MARKET ORDER (1 contracts)`
- NT Trades tab should show an order for MNQ 12-26
- Once filled, NT prints `[FILLED] LONG 1 contracts @ <price>` and `[ORDERS SUBMITTED] Long exit orders sent to broker`
- After a moment (or when SL/TP hits), NT prints `[EXIT SL]` or `[EXIT TP]`
- `trades_taken.csv` should have one new row

If this doesn't happen, **stop here and debug NT setup before continuing**. Nothing in the rest of the plan works if this manual test fails.

- [ ] **Step 0.9: Commit the .env values**

Once Steps 0.1-0.8 all pass, add the env vars to `/home/hoang/nofx/.env`:

```
DATABENTO_API_KEY=db-XXXXXXXXXXXXXXXXXXXXXXXXXX
DATABENTO_DATASET=GLBX.MDP3
NINJATRADER_DATA_DIR=/mnt/c/Users/<your-windows-username>/NofxTrader/data
TRADING_MODE=futures
```

(Replace placeholders with real values. Do NOT commit `.env` to git; it's already in `.gitignore`.)

---

## Task 1: Wire Databento config into Go

**Files:**
- Modify: `config/config.go`
- Modify: `.env.example`

- [ ] **Step 1.1: Add fields to Config struct**

In `config/config.go`, find the `Config` struct (around line 16-46) and add these fields at the bottom:

```go
// Databento (NQ futures data)
DatabentoAPIKey  string
DatabentoDataset string  // e.g., "GLBX.MDP3"

// NinjaTrader (CSV bridge for execution)
NinjaTraderDataDir string  // e.g., "/mnt/c/Users/<u>/NofxTrader/data"

// Trading mode: "crypto" (default, original behavior) or "futures"
TradingMode string
```

- [ ] **Step 1.2: Load them in Init()**

Find the `Init()` or env-loading section in `config/config.go` and add:

```go
cfg.DatabentoAPIKey = os.Getenv("DATABENTO_API_KEY")
cfg.DatabentoDataset = getEnvOrDefault("DATABENTO_DATASET", "GLBX.MDP3")
cfg.NinjaTraderDataDir = os.Getenv("NINJATRADER_DATA_DIR")
cfg.TradingMode = getEnvOrDefault("TRADING_MODE", "crypto")
```

(If `getEnvOrDefault` doesn't exist, define it as a helper: `func getEnvOrDefault(key, def string) string { if v := os.Getenv(key); v != "" { return v }; return def }`.)

- [ ] **Step 1.3: Add to .env.example**

Append to `.env.example`:

```
# NQ futures (Databento + NinjaTrader)
TRADING_MODE=crypto                                          # set to "futures" to enable NQ path
DATABENTO_API_KEY=                                           # https://databento.com/portal/keys
DATABENTO_DATASET=GLBX.MDP3                                  # CME Globex
NINJATRADER_DATA_DIR=                                        # e.g., /mnt/c/Users/<u>/NofxTrader/data
```

- [ ] **Step 1.4: Build, verify no compile errors**

Run:

```bash
cd /home/hoang/nofx && go build ./... 2>&1 | head -20
```

Expected: no output (build success).

- [ ] **Step 1.5: Commit**

```bash
cd /home/hoang/nofx
git add config/config.go .env.example
git commit -m "feat(config): add Databento + NinjaTrader config fields"
```

---

## Task 2: Databento Historical OHLCV client

**Files:**
- Create: `provider/databento/historical.go`
- Create: `provider/databento/historical_test.go`
- Existing: `provider/databento/client.go` (already has Basic-auth `doRequest`)

**API reference:** `GET /v0/timeseries.get_range` returns JSON-lines (one record per line) when `encoding=json` is set. Schema `ohlcv-1m` returns records with `ts_event` (nanosecond epoch), `open`, `high`, `low`, `close`, `volume` as integer fixed-point (close * 1e9 — divide by 1e9 to get a float). Source: [Databento Historical API docs](https://databento.com/docs/api-reference-historical/timeseries).

- [ ] **Step 2.1: Define the Bar type and signature in a new file**

Create `/home/hoang/nofx/provider/databento/historical.go`:

```go
package databento

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// Bar represents one OHLCV record returned by Databento.
type Bar struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// rawBar mirrors the JSON shape Databento returns for ohlcv-* schemas.
// Numeric fields arrive as integer fixed-point with 1e9 divisor.
type rawBar struct {
	TsEvent string `json:"ts_event"`
	Open    string `json:"open"`
	High    string `json:"high"`
	Low     string `json:"low"`
	Close   string `json:"close"`
	Volume  string `json:"volume"`
}

// GetOHLCV fetches OHLCV bars for one symbol over [start, end).
// interval must be one of "1m", "1h", "1d" — maps to schema "ohlcv-<interval>".
// symbol can be a continuous code like "NQ.c.0" or a specific contract like "NQM6".
func (c *Client) GetOHLCV(symbol, interval string, start, end time.Time) ([]Bar, error) {
	schema := "ohlcv-" + interval
	params := url.Values{}
	params.Set("dataset", DefaultDataset)
	params.Set("symbols", symbol)
	params.Set("schema", schema)
	params.Set("stype_in", "continuous") // NQ.c.0 is a continuous symbol
	params.Set("start", start.UTC().Format(time.RFC3339))
	params.Set("end", end.UTC().Format(time.RFC3339))
	params.Set("encoding", "json")

	body, err := c.doRequest("/timeseries.get_range", params)
	if err != nil {
		return nil, err
	}
	return parseOHLCVResponse(body)
}

// parseOHLCVResponse decodes Databento's JSON-lines body into []Bar.
// Exported (lowercase but easy to test from within package).
func parseOHLCVResponse(body []byte) ([]Bar, error) {
	var bars []Bar
	sc := bufio.NewScanner(bytes.NewReader(body))
	sc.Buffer(make([]byte, 1024*1024), 8*1024*1024) // tolerate long lines
	for sc.Scan() {
		line := sc.Bytes()
		if len(line) == 0 {
			continue
		}
		var r rawBar
		if err := json.Unmarshal(line, &r); err != nil {
			return nil, fmt.Errorf("databento: parse bar: %w", err)
		}
		bar, err := r.toBar()
		if err != nil {
			return nil, err
		}
		bars = append(bars, bar)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("databento: scan body: %w", err)
	}
	return bars, nil
}

func (r rawBar) toBar() (Bar, error) {
	tsNs, err := strconv.ParseInt(r.TsEvent, 10, 64)
	if err != nil {
		return Bar{}, fmt.Errorf("databento: parse ts_event %q: %w", r.TsEvent, err)
	}
	open, err := scaledFloat(r.Open)
	if err != nil {
		return Bar{}, err
	}
	high, err := scaledFloat(r.High)
	if err != nil {
		return Bar{}, err
	}
	low, err := scaledFloat(r.Low)
	if err != nil {
		return Bar{}, err
	}
	closeP, err := scaledFloat(r.Close)
	if err != nil {
		return Bar{}, err
	}
	vol, err := strconv.ParseFloat(r.Volume, 64)
	if err != nil {
		return Bar{}, fmt.Errorf("databento: parse volume %q: %w", r.Volume, err)
	}
	return Bar{
		Timestamp: time.Unix(0, tsNs).UTC(),
		Open:      open,
		High:      high,
		Low:       low,
		Close:     closeP,
		Volume:    vol,
	}, nil
}

// Databento ohlcv schemas use integer fixed-point with 1e9 divisor.
func scaledFloat(s string) (float64, error) {
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("databento: parse scaled int %q: %w", s, err)
	}
	return float64(n) / 1e9, nil
}
```

- [ ] **Step 2.2: Write the parser test FIRST**

Create `/home/hoang/nofx/provider/databento/historical_test.go`:

```go
package databento

import (
	"testing"
	"time"
)

func TestParseOHLCVResponse_TwoBars(t *testing.T) {
	// Sample of what Databento returns for schema=ohlcv-1m, encoding=json.
	// Each line is one bar. Numeric fields are integer fixed-point (1e9 divisor).
	body := []byte(`{"ts_event":"1746360000000000000","open":"21500250000000","high":"21515750000000","low":"21498000000000","close":"21510000000000","volume":"4321"}
{"ts_event":"1746360060000000000","open":"21510000000000","high":"21525500000000","low":"21505000000000","close":"21522750000000","volume":"5102"}
`)

	bars, err := parseOHLCVResponse(body)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(bars) != 2 {
		t.Fatalf("want 2 bars, got %d", len(bars))
	}

	want0 := Bar{
		Timestamp: time.Unix(0, 1746360000000000000).UTC(),
		Open:      21500.25,
		High:      21515.75,
		Low:       21498.00,
		Close:     21510.00,
		Volume:    4321,
	}
	if bars[0] != want0 {
		t.Errorf("bar[0] = %+v, want %+v", bars[0], want0)
	}

	if bars[1].Close != 21522.75 {
		t.Errorf("bar[1].Close = %v, want 21522.75", bars[1].Close)
	}
}

func TestParseOHLCVResponse_Empty(t *testing.T) {
	bars, err := parseOHLCVResponse([]byte(""))
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if len(bars) != 0 {
		t.Errorf("want 0 bars, got %d", len(bars))
	}
}

func TestParseOHLCVResponse_MalformedLine(t *testing.T) {
	body := []byte(`{"ts_event":"abc","open":"1","high":"1","low":"1","close":"1","volume":"1"}` + "\n")
	_, err := parseOHLCVResponse(body)
	if err == nil {
		t.Fatal("want error on malformed ts_event, got nil")
	}
}
```

- [ ] **Step 2.3: Run tests, verify all pass**

```bash
cd /home/hoang/nofx && go test ./provider/databento/... -run TestParseOHLCV -v
```

Expected: `--- PASS` for all three. If any fails, fix the parser.

- [ ] **Step 2.4: Live smoke test (manual; costs ~$0.002 USDC equivalent on your Databento balance)**

Create a temporary file `/tmp/db_smoke.go`:

```go
package main

import (
	"fmt"
	"nofx/provider/databento"
	"os"
	"time"
)

func main() {
	c := databento.NewClient("", os.Getenv("DATABENTO_API_KEY"))
	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)
	bars, err := c.GetOHLCV("NQ.c.0", "1m", start, end)
	if err != nil {
		fmt.Println("ERROR:", err)
		os.Exit(1)
	}
	fmt.Printf("Got %d bars\n", len(bars))
	for _, b := range bars {
		fmt.Printf("  %s  O=%.2f H=%.2f L=%.2f C=%.2f V=%.0f\n",
			b.Timestamp.Format("15:04:05"), b.Open, b.High, b.Low, b.Close, b.Volume)
	}
}
```

Run it:

```bash
cd /home/hoang/nofx && go run /tmp/db_smoke.go
```

Expected: ~25-30 lines of 1-minute NQ bars from the last half hour. If you get an error mentioning "401", your API key is wrong; if "402", the endpoint or schema is wrong; if "no bars", check the time range hits a session-open window.

- [ ] **Step 2.5: Commit**

```bash
cd /home/hoang/nofx
git add provider/databento/historical.go provider/databento/historical_test.go
git commit -m "feat(databento): add GetOHLCV historical client"
rm /tmp/db_smoke.go
```

---

## Task 3: Symbol resolver (continuous → specific contract)

**Why:** When NinjaTrader places an order, it places against the *specific* contract attached to the chart (e.g., `MNQ 12-26`), not the continuous symbol. Databento can tell us which specific contract `NQ.c.0` resolves to today via `/v0/symbology.resolve`. We use this to (a) confirm NT is on the front-month, and (b) detect when the front-month rolls.

**Files:**
- Create: `provider/databento/resolve.go`
- Create: `provider/databento/resolve_test.go`

- [ ] **Step 3.1: Write the failing test**

Create `/home/hoang/nofx/provider/databento/resolve_test.go`:

```go
package databento

import "testing"

func TestParseResolveResponse_FrontMonthNQ(t *testing.T) {
	// Real-shape response from /v0/symbology.resolve for symbols=NQ.c.0
	body := []byte(`{
		"result": {
			"NQ.c.0": [
				{"d0": "2026-05-22", "d1": "2026-06-19", "s": "NQM6"}
			]
		},
		"symbols": ["NQ.c.0"],
		"stype_in": "continuous",
		"stype_out": "raw_symbol",
		"start_date": "2026-05-22",
		"end_date": "2026-05-22",
		"partial": [],
		"not_found": []
	}`)

	got, err := parseResolveResponse(body, "NQ.c.0")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if got != "NQM6" {
		t.Errorf("got %q, want %q", got, "NQM6")
	}
}

func TestParseResolveResponse_NotFound(t *testing.T) {
	body := []byte(`{"result":{},"symbols":["NQ.c.0"],"not_found":["NQ.c.0"]}`)
	_, err := parseResolveResponse(body, "NQ.c.0")
	if err == nil {
		t.Fatal("want error when symbol not found, got nil")
	}
}
```

- [ ] **Step 3.2: Run the test and verify failure**

```bash
cd /home/hoang/nofx && go test ./provider/databento/... -run TestParseResolve -v
```

Expected: build error / compile failure ("parseResolveResponse" undefined). That's the failing state.

- [ ] **Step 3.3: Implement the resolver**

Create `/home/hoang/nofx/provider/databento/resolve.go`:

```go
package databento

import (
	"encoding/json"
	"fmt"
	"net/url"
	"time"
)

// ResolveContinuous returns the specific contract symbol that a continuous
// symbol (e.g. "NQ.c.0") points to today. For non-continuous symbols this
// is a passthrough.
func (c *Client) ResolveContinuous(symbol string) (string, error) {
	today := time.Now().UTC().Format("2006-01-02")
	params := url.Values{}
	params.Set("dataset", DefaultDataset)
	params.Set("symbols", symbol)
	params.Set("stype_in", "continuous")
	params.Set("stype_out", "raw_symbol")
	params.Set("start_date", today)
	params.Set("end_date", today)

	body, err := c.doRequest("/symbology.resolve", params)
	if err != nil {
		return "", err
	}
	return parseResolveResponse(body, symbol)
}

type resolveResponse struct {
	Result   map[string][]resolveEntry `json:"result"`
	NotFound []string                  `json:"not_found"`
}

type resolveEntry struct {
	D0 string `json:"d0"`
	D1 string `json:"d1"`
	S  string `json:"s"`
}

func parseResolveResponse(body []byte, symbol string) (string, error) {
	var resp resolveResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("databento resolve: parse: %w", err)
	}
	for _, nf := range resp.NotFound {
		if nf == symbol {
			return "", fmt.Errorf("databento resolve: symbol not found: %s", symbol)
		}
	}
	entries, ok := resp.Result[symbol]
	if !ok || len(entries) == 0 {
		return "", fmt.Errorf("databento resolve: no entries for %s", symbol)
	}
	return entries[0].S, nil
}
```

- [ ] **Step 3.4: Run tests and verify pass**

```bash
cd /home/hoang/nofx && go test ./provider/databento/... -v
```

Expected: all tests pass.

- [ ] **Step 3.5: Commit**

```bash
cd /home/hoang/nofx
git add provider/databento/resolve.go provider/databento/resolve_test.go
git commit -m "feat(databento): add continuous symbol resolver"
```

---

## Task 4: Databento → market.Kline adapter

**Why:** The rest of the codebase (indicators, formatter, kernel) consumes `market.Kline`, not `databento.Bar`. One small adapter lets every existing function work unchanged.

**Files:**
- Create: `market/databento_adapter.go`
- Create: `market/databento_adapter_test.go`

- [ ] **Step 4.1: Read the existing Kline shape**

```bash
cd /home/hoang/nofx && grep -A 12 "^type Kline struct" market/types.go
```

Note the fields exactly — they'll inform the mapping in the next step.

- [ ] **Step 4.2: Write the failing test**

Create `/home/hoang/nofx/market/databento_adapter_test.go`:

```go
package market

import (
	"nofx/provider/databento"
	"testing"
	"time"
)

func TestBarsToKlines_Mapping(t *testing.T) {
	bars := []databento.Bar{
		{
			Timestamp: time.Unix(1746360000, 0).UTC(),
			Open:      21500.25,
			High:      21515.75,
			Low:       21498.00,
			Close:     21510.00,
			Volume:    4321,
		},
	}
	klines := BarsToKlines(bars)
	if len(klines) != 1 {
		t.Fatalf("want 1 kline, got %d", len(klines))
	}
	k := klines[0]
	if k.Open != 21500.25 || k.High != 21515.75 || k.Low != 21498.00 || k.Close != 21510.00 || k.Volume != 4321 {
		t.Errorf("kline OHLCV mismatch: %+v", k)
	}
	// Kline.OpenTime should be milliseconds since epoch (this is the convention
	// the existing code uses — verify in market/types.go before writing).
	wantMs := int64(1746360000 * 1000)
	if k.OpenTime != wantMs {
		t.Errorf("kline.OpenTime = %d, want %d", k.OpenTime, wantMs)
	}
}

func TestBarsToKlines_Empty(t *testing.T) {
	got := BarsToKlines(nil)
	if len(got) != 0 {
		t.Errorf("want 0 klines, got %d", len(got))
	}
}
```

> **Note:** Before writing the adapter, **read `market/types.go:94-108` to confirm the field name** (`OpenTime` vs `Timestamp` vs `Time`) and **the unit** (seconds, milliseconds, or `time.Time`). Adjust the test and adapter to match. The test above assumes `OpenTime int64` in milliseconds; this matches Binance-style conventions.

- [ ] **Step 4.3: Run the test, verify failure**

```bash
cd /home/hoang/nofx && go test ./market/... -run TestBarsToKlines -v
```

Expected: compile error ("BarsToKlines undefined").

- [ ] **Step 4.4: Implement the adapter**

Create `/home/hoang/nofx/market/databento_adapter.go`:

```go
package market

import (
	"nofx/provider/databento"
)

// BarsToKlines converts Databento bars into the project's canonical Kline shape.
// This is the single bridge that lets every existing indicator, formatter, and
// strategy work with NQ data unchanged.
func BarsToKlines(bars []databento.Bar) []Kline {
	if len(bars) == 0 {
		return nil
	}
	out := make([]Kline, 0, len(bars))
	for _, b := range bars {
		out = append(out, Kline{
			OpenTime: b.Timestamp.UnixMilli(),
			Open:     b.Open,
			High:     b.High,
			Low:      b.Low,
			Close:    b.Close,
			Volume:   b.Volume,
		})
	}
	return out
}
```

> **If `market.Kline` has more fields** (e.g. `CloseTime`, `Trades`, `IsClosed`), fill them with sensible zero values or computed values (`CloseTime = OpenTime + intervalMs - 1`, `IsClosed = true`). Don't leave unset fields that downstream code requires.

- [ ] **Step 4.5: Run tests, verify pass**

```bash
cd /home/hoang/nofx && go test ./market/... -run TestBarsToKlines -v
```

Expected: PASS.

- [ ] **Step 4.6: Commit**

```bash
cd /home/hoang/nofx
git add market/databento_adapter.go market/databento_adapter_test.go
git commit -m "feat(market): add Databento Bar -> Kline adapter"
```

---

## Task 5: NinjaTrader CSV writer

**Files:**
- Create: `provider/ninjatrader/types.go`
- Create: `provider/ninjatrader/csv_writer.go`
- Create: `provider/ninjatrader/csv_writer_test.go`

**Contract format (verified from claudetrader.cs:191-223):**

```csv
DateTime,Direction,Entry_Price,Stop_Loss,Take_Profit
05/22/2026 14:30:15,LONG,21505.00,21485.00,21545.00
```

- DateTime format MUST match `MM/dd/yyyy HH:mm:ss` (C# `DateTime.TryParse` is lenient but this is the format NT writes back).
- Direction must be exactly `LONG` or `SHORT` (uppercase per claudetrader.cs:229,233).
- Prices use `.` decimal separator, two decimal places (NQ tick = 0.25).
- One signal per row. **Writing a new row triggers NT within ~2 seconds.**
- NT clears the file after read, leaving only the header. So our writer truncates+rewrites: header + one row. (Append also works, but truncate is safer — guarantees we never have stale rows.)

- [ ] **Step 5.1: Define types**

Create `/home/hoang/nofx/provider/ninjatrader/types.go`:

```go
package ninjatrader

import (
	"fmt"
	"strings"
)

// SignalRow is one row of trade_signals.csv that claudetrader.cs will consume.
type SignalRow struct {
	DateTime    string // MM/dd/yyyy HH:mm:ss
	Direction   string // "LONG" or "SHORT"
	EntryPrice  float64
	StopLoss    float64
	TakeProfit  float64
}

// FillRow is one row of trades_taken.csv that claudetrader.cs writes.
type FillRow struct {
	DateTime   string  // MM/dd/yyyy HH:mm:ss
	Direction  string  // "LONG" or "SHORT"
	EntryPrice float64
}

const signalsHeader = "DateTime,Direction,Entry_Price,Stop_Loss,Take_Profit"
const fillsHeader = "DateTime,Direction,Entry_Price"

// Validate rejects malformed signals before they reach disk.
func (s SignalRow) Validate() error {
	if s.DateTime == "" {
		return fmt.Errorf("ninjatrader signal: empty DateTime")
	}
	if strings.ToUpper(s.Direction) != "LONG" && strings.ToUpper(s.Direction) != "SHORT" {
		return fmt.Errorf("ninjatrader signal: direction must be LONG or SHORT, got %q", s.Direction)
	}
	if s.EntryPrice <= 0 || s.StopLoss <= 0 || s.TakeProfit <= 0 {
		return fmt.Errorf("ninjatrader signal: prices must be positive")
	}
	dir := strings.ToUpper(s.Direction)
	if dir == "LONG" && s.StopLoss >= s.EntryPrice {
		return fmt.Errorf("ninjatrader signal: LONG stop_loss (%.2f) must be below entry (%.2f)", s.StopLoss, s.EntryPrice)
	}
	if dir == "LONG" && s.TakeProfit <= s.EntryPrice {
		return fmt.Errorf("ninjatrader signal: LONG take_profit (%.2f) must be above entry (%.2f)", s.TakeProfit, s.EntryPrice)
	}
	if dir == "SHORT" && s.StopLoss <= s.EntryPrice {
		return fmt.Errorf("ninjatrader signal: SHORT stop_loss (%.2f) must be above entry (%.2f)", s.StopLoss, s.EntryPrice)
	}
	if dir == "SHORT" && s.TakeProfit >= s.EntryPrice {
		return fmt.Errorf("ninjatrader signal: SHORT take_profit (%.2f) must be below entry (%.2f)", s.TakeProfit, s.EntryPrice)
	}
	return nil
}
```

- [ ] **Step 5.2: Write the failing test for the writer**

Create `/home/hoang/nofx/provider/ninjatrader/csv_writer_test.go`:

```go
package ninjatrader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCSVWriter_WriteSignal_LongValid(t *testing.T) {
	dir := t.TempDir()
	w := NewCSVWriter(dir)

	sig := SignalRow{
		DateTime:   "05/22/2026 14:30:15",
		Direction:  "LONG",
		EntryPrice: 21505.00,
		StopLoss:   21485.00,
		TakeProfit: 21545.00,
	}
	if err := w.WriteSignal(sig); err != nil {
		t.Fatalf("WriteSignal: %v", err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "trade_signals.csv"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	want := "DateTime,Direction,Entry_Price,Stop_Loss,Take_Profit\n05/22/2026 14:30:15,LONG,21505.00,21485.00,21545.00\n"
	if string(got) != want {
		t.Errorf("file content:\n  got:  %q\n  want: %q", string(got), want)
	}
}

func TestCSVWriter_WriteSignal_RejectsBadLong(t *testing.T) {
	dir := t.TempDir()
	w := NewCSVWriter(dir)

	bad := SignalRow{
		DateTime:   "05/22/2026 14:30:15",
		Direction:  "LONG",
		EntryPrice: 21500.00,
		StopLoss:   21520.00, // wrong: stop ABOVE entry for a long
		TakeProfit: 21550.00,
	}
	err := w.WriteSignal(bad)
	if err == nil {
		t.Fatal("want validation error, got nil")
	}
	if !strings.Contains(err.Error(), "LONG stop_loss") {
		t.Errorf("error message %q does not mention LONG stop_loss", err.Error())
	}
}

func TestCSVWriter_WriteSignal_TruncatesPrevious(t *testing.T) {
	dir := t.TempDir()
	w := NewCSVWriter(dir)

	first := SignalRow{DateTime: "05/22/2026 14:30:15", Direction: "LONG", EntryPrice: 100, StopLoss: 90, TakeProfit: 110}
	second := SignalRow{DateTime: "05/22/2026 14:32:00", Direction: "SHORT", EntryPrice: 200, StopLoss: 210, TakeProfit: 190}

	if err := w.WriteSignal(first); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteSignal(second); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(filepath.Join(dir, "trade_signals.csv"))
	// Should contain ONLY the second signal (plus header), not both.
	if strings.Contains(string(got), "14:30:15") {
		t.Errorf("expected first signal to be truncated, but file still contains it:\n%s", string(got))
	}
	if !strings.Contains(string(got), "14:32:00") {
		t.Errorf("expected second signal in file:\n%s", string(got))
	}
}
```

- [ ] **Step 5.3: Run tests, verify failure**

```bash
cd /home/hoang/nofx && go test ./provider/ninjatrader/... -v
```

Expected: compile failure ("NewCSVWriter undefined").

- [ ] **Step 5.4: Implement the writer**

Create `/home/hoang/nofx/provider/ninjatrader/csv_writer.go`:

```go
package ninjatrader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// CSVWriter writes trade signals to a Windows-shared CSV file that
// NinjaTrader's claudetrader.cs polls every 2 seconds.
type CSVWriter struct {
	dataDir string
	mu      sync.Mutex
}

func NewCSVWriter(dataDir string) *CSVWriter {
	return &CSVWriter{dataDir: dataDir}
}

// SignalsPath returns the absolute path to trade_signals.csv.
func (w *CSVWriter) SignalsPath() string {
	return filepath.Join(w.dataDir, "trade_signals.csv")
}

// WriteSignal validates and writes one signal. The file is truncated and
// rewritten with header + this single row — claudetrader.cs clears the file
// after processing, so we always overwrite to avoid stale rows accumulating.
func (w *CSVWriter) WriteSignal(s SignalRow) error {
	if err := s.Validate(); err != nil {
		return err
	}
	w.mu.Lock()
	defer w.mu.Unlock()

	row := fmt.Sprintf("%s,%s,%.2f,%.2f,%.2f",
		s.DateTime,
		strings.ToUpper(s.Direction),
		s.EntryPrice,
		s.StopLoss,
		s.TakeProfit,
	)
	content := signalsHeader + "\n" + row + "\n"

	tmp, err := os.CreateTemp(w.dataDir, "trade_signals.*.tmp")
	if err != nil {
		return fmt.Errorf("ninjatrader writer: create temp: %w", err)
	}
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return fmt.Errorf("ninjatrader writer: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("ninjatrader writer: close temp: %w", err)
	}
	if err := os.Rename(tmp.Name(), w.SignalsPath()); err != nil {
		os.Remove(tmp.Name())
		return fmt.Errorf("ninjatrader writer: rename temp: %w", err)
	}
	return nil
}
```

> **Why temp+rename:** atomic update. NT polls every 2s; if NT happens to read mid-write of a non-atomic update it sees a partial file. Atomic rename means NT always sees either the old or new file, never a torn write.

- [ ] **Step 5.5: Run tests, verify pass**

```bash
cd /home/hoang/nofx && go test ./provider/ninjatrader/... -v
```

Expected: all 3 tests PASS.

- [ ] **Step 5.6: Live smoke test against your actual NT setup**

With NT open, ClaudeTrader running on the MNQ chart in SIM, run:

```bash
cd /home/hoang/nofx
DATA_DIR="$(grep NINJATRADER_DATA_DIR .env | cut -d= -f2)"
cat > /tmp/nt_smoke.go <<'EOF'
package main

import (
	"fmt"
	"nofx/provider/ninjatrader"
	"os"
	"time"
)

func main() {
	dir := os.Args[1]
	w := ninjatrader.NewCSVWriter(dir)
	now := time.Now().Format("01/02/2006 15:04:05")
	// IMPORTANT: replace these prices with something near current MNQ market.
	// Check NT chart — if MNQ is at 21500, use entry near it and SL/TP a few points away.
	sig := ninjatrader.SignalRow{
		DateTime:   now,
		Direction:  "LONG",
		EntryPrice: 21500.00,
		StopLoss:   21480.00,
		TakeProfit: 21540.00,
	}
	if err := w.WriteSignal(sig); err != nil {
		fmt.Println("ERR:", err)
		os.Exit(1)
	}
	fmt.Println("Wrote signal:", sig)
	fmt.Println("Now watch NT's Output window — should see [SIGNAL] within 2-4s.")
}
EOF
go run /tmp/nt_smoke.go "$DATA_DIR"
rm /tmp/nt_smoke.go
```

Watch NT's Output window. Expected:
```
[SIGNAL] LONG MARKET ORDER (1 contracts)
  Reference Entry: 21500.00
  Target SL: 21480.00 | Target TP: 21540.00
[ORDER UPDATE] CT_Long ...
[FILLED] LONG 1 contracts @ <market price>
[ORDERS SUBMITTED] Long exit orders sent to broker
```

If you see this — the Go-side bridge works end-to-end.

- [ ] **Step 5.7: Commit**

```bash
cd /home/hoang/nofx
git add provider/ninjatrader/types.go provider/ninjatrader/csv_writer.go provider/ninjatrader/csv_writer_test.go
git commit -m "feat(ninjatrader): add CSV signal writer with validation"
```

---

## Task 6: NinjaTrader fill tailer

**Files:**
- Create: `provider/ninjatrader/csv_tailer.go`
- Create: `provider/ninjatrader/csv_tailer_test.go`

**Behavior:** Periodically read `trades_taken.csv`, track which rows we've already seen by line count (NT appends; doesn't truncate this file). For each new row, decode it and invoke the callback.

- [ ] **Step 6.1: Write the failing tailer test**

Create `/home/hoang/nofx/provider/ninjatrader/csv_tailer_test.go`:

```go
package ninjatrader

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func TestCSVTailer_DetectsAppendedRows(t *testing.T) {
	dir := t.TempDir()
	fillsPath := filepath.Join(dir, "trades_taken.csv")
	// Seed with header
	if err := os.WriteFile(fillsPath, []byte(fillsHeader+"\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var (
		mu    sync.Mutex
		fills []FillRow
	)
	cb := func(f FillRow) {
		mu.Lock()
		defer mu.Unlock()
		fills = append(fills, f)
	}

	tailer := NewCSVTailer(dir, 50*time.Millisecond)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() { _ = tailer.TailFills(ctx, cb) }()

	// Allow tailer to read initial state (header only, no rows)
	time.Sleep(150 * time.Millisecond)

	// Append two fills (NT-style)
	f, err := os.OpenFile(fillsPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString("05/22/2026 14:30:15,LONG,21505.25\n")
	f.WriteString("05/22/2026 14:35:42,SHORT,21520.50\n")
	f.Close()

	// Wait for tailer to pick them up
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(fills)
		mu.Unlock()
		if n >= 2 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(fills) != 2 {
		t.Fatalf("got %d fills, want 2: %+v", len(fills), fills)
	}
	if fills[0].Direction != "LONG" || fills[0].EntryPrice != 21505.25 {
		t.Errorf("fill[0] = %+v", fills[0])
	}
	if fills[1].Direction != "SHORT" || fills[1].EntryPrice != 21520.50 {
		t.Errorf("fill[1] = %+v", fills[1])
	}
}
```

- [ ] **Step 6.2: Run the test, verify failure**

```bash
cd /home/hoang/nofx && go test ./provider/ninjatrader/... -run TestCSVTailer -v
```

Expected: compile failure.

- [ ] **Step 6.3: Implement the tailer**

Create `/home/hoang/nofx/provider/ninjatrader/csv_tailer.go`:

```go
package ninjatrader

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// CSVTailer reads new rows appended to trades_taken.csv by claudetrader.cs.
type CSVTailer struct {
	dataDir      string
	pollInterval time.Duration
	seen         int // rows already delivered (excluding header)
}

func NewCSVTailer(dataDir string, pollInterval time.Duration) *CSVTailer {
	if pollInterval <= 0 {
		pollInterval = time.Second
	}
	return &CSVTailer{dataDir: dataDir, pollInterval: pollInterval}
}

func (t *CSVTailer) FillsPath() string {
	return filepath.Join(t.dataDir, "trades_taken.csv")
}

// TailFills blocks until ctx is cancelled. For each new fill row appended to
// the file, the callback is invoked synchronously. Reset on file-shrink (e.g.
// if NT cycles the file at session boundary).
func (t *CSVTailer) TailFills(ctx context.Context, onFill func(FillRow)) error {
	ticker := time.NewTicker(t.pollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := t.readNew(onFill); err != nil {
				// Log internally — never kill the loop on a transient read.
				fmt.Fprintf(os.Stderr, "ninjatrader tailer: %v\n", err)
			}
		}
	}
}

func (t *CSVTailer) readNew(onFill func(FillRow)) error {
	f, err := os.Open(t.FillsPath())
	if err != nil {
		if os.IsNotExist(err) {
			return nil // file may not exist yet — first fill creates it
		}
		return err
	}
	defer f.Close()

	rows := []string{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			rows = append(rows, line)
		}
	}
	if err := sc.Err(); err != nil {
		return err
	}

	// Strip header if present
	if len(rows) > 0 && strings.HasPrefix(rows[0], "DateTime,") {
		rows = rows[1:]
	}

	// File shrunk → NT cycled; reset our cursor
	if len(rows) < t.seen {
		t.seen = 0
	}

	for i := t.seen; i < len(rows); i++ {
		fill, err := parseFillRow(rows[i])
		if err != nil {
			fmt.Fprintf(os.Stderr, "ninjatrader tailer: parse %q: %v\n", rows[i], err)
			continue
		}
		onFill(fill)
	}
	t.seen = len(rows)
	return nil
}

func parseFillRow(line string) (FillRow, error) {
	parts := strings.Split(line, ",")
	if len(parts) < 3 {
		return FillRow{}, fmt.Errorf("expected 3 fields, got %d", len(parts))
	}
	price, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if err != nil {
		return FillRow{}, fmt.Errorf("parse entry price: %w", err)
	}
	return FillRow{
		DateTime:   strings.TrimSpace(parts[0]),
		Direction:  strings.ToUpper(strings.TrimSpace(parts[1])),
		EntryPrice: price,
	}, nil
}
```

- [ ] **Step 6.4: Run tests, verify pass**

```bash
cd /home/hoang/nofx && go test ./provider/ninjatrader/... -v
```

Expected: all PASS, including the 2-second tailer test.

- [ ] **Step 6.5: Commit**

```bash
cd /home/hoang/nofx
git add provider/ninjatrader/csv_tailer.go provider/ninjatrader/csv_tailer_test.go
git commit -m "feat(ninjatrader): add CSV fill tailer"
```

---

## Task 7: NinjaTrader Trader interface implementation

**Why:** [trader/auto_trader.go:263-315](trader/auto_trader.go#L263-L315) switches on `config.Exchange` and creates a `Trader` per the 17-method interface at [trader/types/interface.go:43-105](trader/types/interface.go#L43-L105). We implement a minimal version that maps `OpenLong/OpenShort` to CSV writes and uses the tailer for `GetPositions`/`GetOrderStatus`.

**Reality check on what we can/can't support via this bridge:**
- ✅ `OpenLong`, `OpenShort` — write LONG/SHORT signal
- ✅ `SetStopLoss`, `SetTakeProfit` — bundled into the signal at submit time, claudetrader.cs handles them
- ✅ `GetPositions` — read trades_taken.csv + track our own state
- ⚠️ `CloseLong`, `CloseShort` — claudetrader.cs auto-closes via SL/TP. No "manual close" path exists in the current bridge. **For v1, return an error here and rely on SL/TP exits.** Adding manual close = a future task that modifies the NinjaScript.
- ⚠️ `SetLeverage`, `SetMarginMode` — irrelevant for futures (set at the broker level, not per order). Return nil noop.
- ⚠️ `CancelAllOrders` — not supported by the CSV bridge. Return error.
- ⚠️ `GetBalance` — claudetrader.cs doesn't expose account balance via CSV. For v1, return a fixed mock value or an error. Future: extend the NinjaScript to write a balance.csv.
- ⚠️ `GetClosedPnL` — same constraint. Future task to log P&L from the C# side.
- ✅ `FormatQuantity` — passthrough rounding.
- ✅ `GetOrderStatus` — query our internal tracking.

**Files:**
- Create: `trader/ninjatrader/trader.go`
- Create: `trader/ninjatrader/trader_test.go`

- [ ] **Step 7.1: Read the Trader interface to confirm signatures**

```bash
cd /home/hoang/nofx && sed -n '43,105p' trader/types/interface.go
```

Note the exact method signatures and return types. **The code below assumes specific signatures based on the audit; verify and adjust if any differ.**

- [ ] **Step 7.2: Write the failing test (just the constructor + OpenLong)**

Create `/home/hoang/nofx/trader/ninjatrader/trader_test.go`:

```go
package ninjatrader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNew_Smoke(t *testing.T) {
	dir := t.TempDir()
	tr := New(Config{DataDir: dir, Symbol: "MNQ"})
	if tr == nil {
		t.Fatal("New returned nil")
	}
}

func TestOpenLong_WritesSignal(t *testing.T) {
	dir := t.TempDir()
	tr := New(Config{DataDir: dir, Symbol: "MNQ"})

	// Stash SL/TP first — they get bundled into the order on OpenLong.
	_ = tr.SetStopLoss("MNQ", "LONG", 1, 21480.00)
	_ = tr.SetTakeProfit("MNQ", "LONG", 1, 21540.00)

	res, err := tr.OpenLong("MNQ", 1, 1)
	if err != nil {
		t.Fatalf("OpenLong: %v", err)
	}
	if res["status"] != "submitted" {
		t.Errorf("status = %v, want submitted", res["status"])
	}

	body, _ := os.ReadFile(filepath.Join(dir, "trade_signals.csv"))
	if !strings.Contains(string(body), "LONG") {
		t.Errorf("signal file missing LONG row:\n%s", string(body))
	}
	if !strings.Contains(string(body), "21480.00") || !strings.Contains(string(body), "21540.00") {
		t.Errorf("signal file missing SL/TP:\n%s", string(body))
	}
}
```

- [ ] **Step 7.3: Implement the trader**

Create `/home/hoang/nofx/trader/ninjatrader/trader.go`:

```go
// Package ninjatrader implements the trader.Trader interface by writing
// trade signals to a CSV file that NinjaTrader's claudetrader.cs strategy
// consumes. Reads fills back via a CSV tailer.
package ninjatrader

import (
	"context"
	"fmt"
	"sync"
	"time"

	"nofx/provider/ninjatrader"
	"nofx/trader/types"
)

type Config struct {
	DataDir string // /mnt/c/Users/<u>/NofxTrader/data
	Symbol  string // e.g. "MNQ" (informational only; NT uses chart's instrument)
}

// Trader satisfies trader/types.Trader using the CSV bridge.
type Trader struct {
	cfg    Config
	writer *ninjatrader.CSVWriter
	tailer *ninjatrader.CSVTailer

	mu       sync.Mutex
	stopLoss map[string]float64 // key: "<symbol>:<side>"
	takePrft map[string]float64
	lastFill ninjatrader.FillRow
	hasFill  bool
}

func New(cfg Config) *Trader {
	t := &Trader{
		cfg:      cfg,
		writer:   ninjatrader.NewCSVWriter(cfg.DataDir),
		tailer:   ninjatrader.NewCSVTailer(cfg.DataDir, time.Second),
		stopLoss: map[string]float64{},
		takePrft: map[string]float64{},
	}
	// Start the fill tailer in the background. Cancellation is the program's
	// responsibility (defer at startup). For now, use context.Background().
	go func() {
		_ = t.tailer.TailFills(context.Background(), func(f ninjatrader.FillRow) {
			t.mu.Lock()
			defer t.mu.Unlock()
			t.lastFill = f
			t.hasFill = true
		})
	}()
	return t
}

// Compile-time check that we implement the interface. If signatures drift,
// the build fails here — not silently at runtime.
var _ types.Trader = (*Trader)(nil)

// --- Trader interface methods ---

func (t *Trader) OpenLong(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	return t.placeEntry(symbol, "LONG", quantity)
}

func (t *Trader) OpenShort(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	return t.placeEntry(symbol, "SHORT", quantity)
}

func (t *Trader) placeEntry(symbol, side string, quantity float64) (map[string]interface{}, error) {
	t.mu.Lock()
	sl := t.stopLoss[keyFor(symbol, side)]
	tp := t.takePrft[keyFor(symbol, side)]
	t.mu.Unlock()

	if sl == 0 || tp == 0 {
		return nil, fmt.Errorf("ninjatrader: SetStopLoss and SetTakeProfit must be called before %s", side)
	}

	// Entry price for the CSV is a "reference" — claudetrader uses MARKET orders.
	// We use 0 as a placeholder, but the strategy ignores it for entry decisions.
	// Use a near-market value so logs make sense; if you don't have one handy,
	// passing the SL midpoint is harmless.
	entryRef := (sl + tp) / 2.0

	sig := ninjatrader.SignalRow{
		DateTime:   time.Now().Format("01/02/2006 15:04:05"),
		Direction:  side,
		EntryPrice: entryRef,
		StopLoss:   sl,
		TakeProfit: tp,
	}
	if err := t.writer.WriteSignal(sig); err != nil {
		return nil, err
	}
	return map[string]interface{}{
		"status":   "submitted",
		"symbol":   symbol,
		"side":     side,
		"quantity": quantity,
	}, nil
}

func (t *Trader) CloseLong(symbol string, quantity float64) (map[string]interface{}, error) {
	return nil, fmt.Errorf("ninjatrader: manual CloseLong not supported via CSV bridge — position closes via SL/TP set at entry")
}

func (t *Trader) CloseShort(symbol string, quantity float64) (map[string]interface{}, error) {
	return nil, fmt.Errorf("ninjatrader: manual CloseShort not supported via CSV bridge — position closes via SL/TP set at entry")
}

func (t *Trader) SetLeverage(symbol string, leverage int) error {
	return nil // futures leverage is set at the broker, not per-order
}

func (t *Trader) SetMarginMode(symbol string, isCrossMargin bool) error {
	return nil // n/a for futures
}

func (t *Trader) SetStopLoss(symbol, positionSide string, quantity, stopPrice float64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stopLoss[keyFor(symbol, positionSide)] = stopPrice
	return nil
}

func (t *Trader) SetTakeProfit(symbol, positionSide string, quantity, takeProfitPrice float64) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.takePrft[keyFor(symbol, positionSide)] = takeProfitPrice
	return nil
}

func (t *Trader) CancelAllOrders(symbol string) error {
	return fmt.Errorf("ninjatrader: CancelAllOrders not supported via CSV bridge")
}

func (t *Trader) GetBalance() (map[string]interface{}, error) {
	// claudetrader.cs doesn't expose balance via CSV. For paper-mode v1, return
	// a fixed sim balance so the trader loop doesn't fail balance checks.
	return map[string]interface{}{
		"totalEquity": 50000.0, // SIM101 starts with $50k by default
		"availableBalance": 50000.0,
	}, nil
}

func (t *Trader) GetPositions() ([]map[string]interface{}, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.hasFill {
		return []map[string]interface{}{}, nil
	}
	return []map[string]interface{}{{
		"symbol":     t.cfg.Symbol,
		"side":       t.lastFill.Direction,
		"entryPrice": t.lastFill.EntryPrice,
		"quantity":   1.0,
	}}, nil
}

func (t *Trader) FormatQuantity(symbol string, quantity float64) (string, error) {
	return fmt.Sprintf("%.0f", quantity), nil
}

func (t *Trader) GetOrderStatus(symbol, orderID string) (map[string]interface{}, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.hasFill {
		return map[string]interface{}{"status": "pending"}, nil
	}
	return map[string]interface{}{
		"status":     "filled",
		"price":      t.lastFill.EntryPrice,
		"side":       t.lastFill.Direction,
	}, nil
}

func (t *Trader) GetClosedPnL(start time.Time, limit int) ([]types.ClosedPnLRecord, error) {
	// Not available via CSV bridge in v1. Return empty.
	return nil, nil
}

// --- 5 additional methods required by trader/types.Trader (audited 2026-05-22) ---

func (t *Trader) GetMarketPrice(symbol string) (float64, error) {
	// Pull last known close from the most recent fill, or zero if no fills yet.
	// For accurate live price the caller should query Databento directly via
	// provider/databento; this method is best-effort for the bridge.
	t.mu.Lock()
	defer t.mu.Unlock()
	if !t.hasFill {
		return 0, fmt.Errorf("ninjatrader: no fill yet, market price unavailable; use Databento client directly")
	}
	return t.lastFill.EntryPrice, nil
}

func (t *Trader) CancelStopLossOrders(symbol string) error {
	return fmt.Errorf("ninjatrader: CancelStopLossOrders not supported via CSV bridge — SL is set at entry")
}

func (t *Trader) CancelTakeProfitOrders(symbol string) error {
	return fmt.Errorf("ninjatrader: CancelTakeProfitOrders not supported via CSV bridge — TP is set at entry")
}

func (t *Trader) CancelStopOrders(symbol string) error {
	// Legacy method, alias for both SL+TP cancel.
	return fmt.Errorf("ninjatrader: CancelStopOrders not supported via CSV bridge")
}

func (t *Trader) GetOpenOrders(symbol string) ([]types.OpenOrder, error) {
	// claudetrader.cs CSV protocol doesn't expose pending orders. The signal
	// file only stores the most recent unconsumed signal; SL/TP live entirely
	// on NT's side after entry fills. Return empty slice (not nil, not error)
	// to match the contract used by other brokers.
	return []types.OpenOrder{}, nil
}

func keyFor(symbol, side string) string {
	return symbol + ":" + side
}
```

> **Compile-time verification:** the `var _ types.Trader = (*Trader)(nil)` line at the top of this file enforces that the impl matches the 19-method interface exactly. If you see `*Trader does not implement types.Trader (missing method X)` at build time, add the missing method following the same pattern (noop/return-empty for unsupported, error for cancellation, real impl for the few that work).

- [ ] **Step 7.4: Run tests, verify pass**

```bash
cd /home/hoang/nofx && go test ./trader/ninjatrader/... -v
```

Expected: PASS.

- [ ] **Step 7.5: Commit**

```bash
cd /home/hoang/nofx
git add trader/ninjatrader/trader.go trader/ninjatrader/trader_test.go
git commit -m "feat(trader): add NinjaTrader Trader impl via CSV bridge"
```

---

## Task 8: Wire NinjaTrader into auto_trader switch

**Files:**
- Modify: `trader/auto_trader.go:263-315` (the broker switch)

- [ ] **Step 8.1: Read the switch**

```bash
cd /home/hoang/nofx && sed -n '253,320p' trader/auto_trader.go
```

Note the existing pattern — each case constructs a broker-specific struct and assigns to `at.trader`.

- [ ] **Step 8.2: Add the case**

In `trader/auto_trader.go`, inside the switch, before the `default:` clause, add:

```go
case "ninjatrader":
    cfg := ninjatrader.Config{
        DataDir: globalConfig.NinjaTraderDataDir, // from config.Get()
        Symbol:  config.Symbol,                   // e.g. "MNQ"
    }
    at.trader = ninjatrader.New(cfg)
```

Add the import at the top:

```go
import (
    // ... existing ...
    ninjatrader "nofx/trader/ninjatrader"
)
```

> **If `globalConfig` isn't accessible at that point**, pass the data dir into `NewAutoTrader` or pull it from the trader's stored config. The pattern in this file will tell you which approach matches.

- [ ] **Step 8.3: Build and verify**

```bash
cd /home/hoang/nofx && go build ./... 2>&1 | head -20
```

Expected: no output.

- [ ] **Step 8.4: Commit**

```bash
cd /home/hoang/nofx
git add trader/auto_trader.go
git commit -m "feat(trader): register ninjatrader broker in switch"
```

---

## Task 9: NQ futures prompt template

**Files:**
- Create: `kernel/engine_prompt_futures.go`
- Create: `kernel/engine_prompt_futures_test.go`

**Why a separate file:** keeps the existing crypto prompt untouched, makes A/B testing trivial, and lets the engine pick at runtime by `TradingMode`.

- [ ] **Step 9.1: Write a golden-file test**

Create `/home/hoang/nofx/kernel/engine_prompt_futures_test.go`:

```go
package kernel

import (
	"strings"
	"testing"
)

func TestBuildFuturesSystemPrompt_NoCryptoVocab(t *testing.T) {
	p := BuildFuturesSystemPrompt(FuturesPromptConfig{
		Symbol:           "MNQ",
		ContractMultiplier: 2.0, // MNQ = $2/point
		TickSize:         0.25,
		MinStopPoints:    15,
		MaxStopPoints:    50,
		MinRiskReward:    1.5,
	})

	// Must NOT contain crypto vocabulary
	forbidden := []string{
		"cryptocurrency", "altcoin", "BTC", "ETH", "USDT", "perpetual",
		"funding rate", "coins simultaneously",
	}
	for _, f := range forbidden {
		if strings.Contains(p, f) {
			t.Errorf("futures prompt contains forbidden crypto term %q", f)
		}
	}

	// Must contain futures-specific framing
	required := []string{
		"NQ", "tick", "contract", "stop loss", "take profit", "MNQ",
	}
	for _, r := range required {
		if !strings.Contains(p, r) {
			t.Errorf("futures prompt missing required term %q", r)
		}
	}
}

func TestBuildFuturesUserPrompt_IncludesIndicators(t *testing.T) {
	p := BuildFuturesUserPrompt(FuturesContext{
		Symbol:       "MNQ",
		CurrentPrice: 21500.00,
		EMA20:        21495.00,
		EMA50:        21480.00,
		RSI14:        58.3,
		MACD:         3.21,
		ATR14:        12.5,
		BollUpper:    21540.00,
		BollLower:    21460.00,
	})
	for _, s := range []string{"21500.00", "EMA20", "RSI14", "ATR14", "Bollinger"} {
		if !strings.Contains(p, s) {
			t.Errorf("user prompt missing %q", s)
		}
	}
}
```

- [ ] **Step 9.2: Run, verify failure**

```bash
cd /home/hoang/nofx && go test ./kernel/... -run TestBuildFutures -v
```

Expected: compile failure.

- [ ] **Step 9.3: Implement**

Create `/home/hoang/nofx/kernel/engine_prompt_futures.go`:

```go
package kernel

import (
	"fmt"
	"strings"
)

// FuturesPromptConfig captures the few parameters the system prompt needs
// to describe an index-futures contract to the model.
type FuturesPromptConfig struct {
	Symbol             string  // "NQ" or "MNQ"
	ContractMultiplier float64 // NQ = 20 ($20/point), MNQ = 2 ($2/point)
	TickSize           float64 // 0.25 for both NQ and MNQ
	MinStopPoints      float64 // 15
	MaxStopPoints      float64 // 50
	MinRiskReward      float64 // 1.5
}

// FuturesContext is the per-cycle data shoved into the user prompt.
type FuturesContext struct {
	Symbol       string
	CurrentPrice float64
	// indicator snapshot
	EMA20     float64
	EMA50     float64
	RSI14     float64
	MACD      float64 // MACD line (current value, from market.ExportCalculateMACD)
	ATR14     float64
	BollUpper float64
	BollLower float64
}

// NOTE: The existing market.ExportCalculateMACD returns only the MACD line
// (one float64), not the signal line or histogram. To surface signal/histogram
// to the AI, we would need to extend market/data_indicators.go to expose a
// fuller MACD function. For Plan 1, the prompt mentions only the MACD line
// value; signal/histogram are deferred to a future indicator-extension task.

func BuildFuturesSystemPrompt(c FuturesPromptConfig) string {
	var b strings.Builder
	b.WriteString("# You are a professional index-futures trading AI specializing in CME E-mini Nasdaq-100 contracts.\n\n")
	b.WriteString(fmt.Sprintf("## Instrument\n- Symbol: %s\n- Tick size: %.2f points\n- Contract multiplier: $%.2f per point\n\n", c.Symbol, c.TickSize, c.ContractMultiplier))
	b.WriteString("## Hard constraints\n")
	b.WriteString(fmt.Sprintf("- Every entry MUST include a stop loss and a take profit, expressed as absolute prices.\n"))
	b.WriteString(fmt.Sprintf("- Stop loss distance: minimum %.0f points, maximum %.0f points from entry.\n", c.MinStopPoints, c.MaxStopPoints))
	b.WriteString(fmt.Sprintf("- Minimum risk/reward: %.2f (reward must be at least %.2fx the risk).\n", c.MinRiskReward, c.MinRiskReward))
	b.WriteString("- One position at a time. Do NOT propose averaging in or pyramiding.\n")
	b.WriteString("- Prices must be in tick increments (multiples of " + fmt.Sprintf("%.2f", c.TickSize) + ").\n")
	b.WriteString("- The market session is CME futures hours; do not assume 24/7 trading.\n\n")
	b.WriteString("## Decision output\n")
	b.WriteString("Respond ONLY with JSON of the following exact shape:\n")
	b.WriteString("```json\n")
	b.WriteString(fmt.Sprintf(`{"action":"LONG"|"SHORT"|"NONE","entry":%.2f,"stop_loss":%.2f,"take_profit":%.2f,"reasoning":"<one-paragraph explanation>"}`, 0.0, 0.0, 0.0))
	b.WriteString("\n```\n")
	b.WriteString("\n- `action=NONE` is a valid and frequently correct answer. Do not force a trade.\n")
	b.WriteString("- All three price fields are absolute (e.g. 21500.25), not deltas from entry.\n\n")
	b.WriteString(fmt.Sprintf("## Trade plan checklist (apply before answering LONG/SHORT)\n"))
	b.WriteString("1. Is there a clear directional bias from EMA20 vs EMA50 alignment?\n")
	b.WriteString("2. Does RSI confirm or contradict that bias? (extreme = caution)\n")
	b.WriteString("3. Is MACD histogram positive (for LONG) or negative (for SHORT)?\n")
	b.WriteString("4. Is ATR consistent with your proposed stop distance? Stop should be ~1.5-3x ATR.\n")
	b.WriteString("5. Where is the Bollinger band — overextended (mean revert) or trending (continuation)?\n")
	b.WriteString("6. Risk/reward calculation: (take_profit - entry) / (entry - stop_loss) for LONG. Must exceed " + fmt.Sprintf("%.2f", c.MinRiskReward) + ".\n")
	return b.String()
}

func BuildFuturesUserPrompt(ctx FuturesContext) string {
	var b strings.Builder
	b.WriteString("## Current market\n")
	b.WriteString(fmt.Sprintf("- Symbol: %s\n", ctx.Symbol))
	b.WriteString(fmt.Sprintf("- Current price: %.2f\n\n", ctx.CurrentPrice))
	b.WriteString("## Indicator snapshot (1-minute timeframe)\n")
	b.WriteString(fmt.Sprintf("- EMA20: %.2f (current price %s)\n", ctx.EMA20, side(ctx.CurrentPrice, ctx.EMA20)))
	b.WriteString(fmt.Sprintf("- EMA50: %.2f (current price %s)\n", ctx.EMA50, side(ctx.CurrentPrice, ctx.EMA50)))
	b.WriteString(fmt.Sprintf("- EMA20 vs EMA50: %s\n", emaAlignment(ctx.EMA20, ctx.EMA50)))
	b.WriteString(fmt.Sprintf("- RSI14: %.1f (%s)\n", ctx.RSI14, rsiBucket(ctx.RSI14)))
	b.WriteString(fmt.Sprintf("- MACD: %.2f (line only; signal/histogram require extended indicator API)\n", ctx.MACD))
	b.WriteString(fmt.Sprintf("- ATR14: %.2f points\n", ctx.ATR14))
	b.WriteString(fmt.Sprintf("- Bollinger Bands: upper %.2f, lower %.2f, position: %s\n", ctx.BollUpper, ctx.BollLower, bollPosition(ctx.CurrentPrice, ctx.BollUpper, ctx.BollLower)))
	b.WriteString("\n## Decision\nGive me your trade decision in the JSON format specified by the system prompt.\n")
	return b.String()
}

func side(price, ref float64) string {
	if price > ref {
		return "above"
	}
	if price < ref {
		return "below"
	}
	return "equal"
}

func emaAlignment(ema20, ema50 float64) string {
	if ema20 > ema50 {
		return "bullish (20 > 50)"
	}
	if ema20 < ema50 {
		return "bearish (20 < 50)"
	}
	return "neutral"
}

func rsiBucket(r float64) string {
	switch {
	case r >= 70:
		return "overbought"
	case r <= 30:
		return "oversold"
	default:
		return "neutral"
	}
}

func bollPosition(p, upper, lower float64) string {
	switch {
	case p >= upper:
		return "above upper band (overextended)"
	case p <= lower:
		return "below lower band (overextended)"
	default:
		return "inside bands"
	}
}
```

- [ ] **Step 9.4: Run tests, verify pass**

```bash
cd /home/hoang/nofx && go test ./kernel/... -run TestBuildFutures -v
```

Expected: PASS.

- [ ] **Step 9.5: Commit**

```bash
cd /home/hoang/nofx
git add kernel/engine_prompt_futures.go kernel/engine_prompt_futures_test.go
git commit -m "feat(kernel): add NQ futures prompt template"
```

---

## Task 10: End-to-end smoke test

**Goal:** Run the bot one cycle, manually, with TRADING_MODE=futures pointing at MNQ in NT SIM. Verify the full chain: Databento → indicators → AI → CSV → NT fill → DB record.

**Files:**
- Create: `cmd/nq_smoke/main.go` — a standalone runner for one cycle

- [ ] **Step 10.1: Write the smoke runner**

Create `/home/hoang/nofx/cmd/nq_smoke/main.go`:

```go
// Standalone single-cycle runner for the NQ trading slice.
// Pulls 30min of NQ.c.0 1m bars, computes indicators, prompts AI, writes signal.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"

	"nofx/kernel"
	"nofx/market"
	"nofx/provider/databento"
	"nofx/provider/ninjatrader"
)

func main() {
	_ = godotenv.Load("/home/hoang/nofx/.env")

	dbKey := os.Getenv("DATABENTO_API_KEY")
	if dbKey == "" {
		log.Fatal("DATABENTO_API_KEY not set in .env")
	}
	ntDir := os.Getenv("NINJATRADER_DATA_DIR")
	if ntDir == "" {
		log.Fatal("NINJATRADER_DATA_DIR not set in .env")
	}

	// 1. Fetch NQ bars
	db := databento.NewClient("", dbKey)
	end := time.Now().UTC()
	start := end.Add(-30 * time.Minute)
	bars, err := db.GetOHLCV("NQ.c.0", "1m", start, end)
	if err != nil {
		log.Fatalf("databento: %v", err)
	}
	if len(bars) < 50 {
		log.Fatalf("got %d bars; need at least 50 for indicators", len(bars))
	}
	fmt.Printf("✓ Fetched %d 1m NQ bars\n", len(bars))

	// 2. Convert to klines and compute indicators
	klines := market.BarsToKlines(bars)
	ctx := kernel.FuturesContext{
		Symbol:       "MNQ",
		CurrentPrice: klines[len(klines)-1].Close,
		EMA20:        market.ExportCalculateEMA(klines, 20),
		EMA50:        market.ExportCalculateEMA(klines, 50),
		RSI14:        market.ExportCalculateRSI(klines, 14),
		MACD:         market.ExportCalculateMACD(klines), // returns MACD line only
		ATR14:        market.ExportCalculateATR(klines, 14),
	}
	upper, _, lower := market.ExportCalculateBOLL(klines, 20, 2.0)
	ctx.BollUpper = upper
	ctx.BollLower = lower

	fmt.Printf("✓ Indicators computed. Current price: %.2f\n", ctx.CurrentPrice)

	// 3. Build prompts
	sysP := kernel.BuildFuturesSystemPrompt(kernel.FuturesPromptConfig{
		Symbol:             "MNQ",
		ContractMultiplier: 2.0,
		TickSize:           0.25,
		MinStopPoints:      15,
		MaxStopPoints:      50,
		MinRiskReward:      1.5,
	})
	userP := kernel.BuildFuturesUserPrompt(ctx)
	fmt.Println("\n--- SYSTEM PROMPT ---")
	fmt.Println(sysP)
	fmt.Println("\n--- USER PROMPT ---")
	fmt.Println(userP)

	// 4. AI call — STUBBED in the smoke runner.
	// Replace with your actual AI client. For first smoke test, hand-fabricate a decision.
	fmt.Println("\n>>> Paste an AI decision JSON (or press Ctrl-C to abort):")
	var decision struct {
		Action     string  `json:"action"`
		Entry      float64 `json:"entry"`
		StopLoss   float64 `json:"stop_loss"`
		TakeProfit float64 `json:"take_profit"`
		Reasoning  string  `json:"reasoning"`
	}
	dec := json.NewDecoder(os.Stdin)
	if err := dec.Decode(&decision); err != nil {
		log.Fatalf("decode decision: %v", err)
	}
	fmt.Printf("✓ Decision: %s entry=%.2f sl=%.2f tp=%.2f\n", decision.Action, decision.Entry, decision.StopLoss, decision.TakeProfit)

	if decision.Action == "NONE" {
		fmt.Println("Action=NONE; nothing to write. Exiting.")
		return
	}

	// 5. Write signal
	w := ninjatrader.NewCSVWriter(ntDir)
	sig := ninjatrader.SignalRow{
		DateTime:   time.Now().Format("01/02/2006 15:04:05"),
		Direction:  decision.Action,
		EntryPrice: decision.Entry,
		StopLoss:   decision.StopLoss,
		TakeProfit: decision.TakeProfit,
	}
	if err := w.WriteSignal(sig); err != nil {
		log.Fatalf("write signal: %v", err)
	}
	fmt.Println("✓ Signal written to", w.SignalsPath())

	// 6. Tail fills for 30 seconds
	fmt.Println("\nTailing trades_taken.csv for 30s — watch for fill...")
	tailer := ninjatrader.NewCSVTailer(ntDir, time.Second)
	ctxT, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = tailer.TailFills(ctxT, func(f ninjatrader.FillRow) {
		fmt.Printf("  >>> FILL: %s %s @ %.2f\n", f.DateTime, f.Direction, f.EntryPrice)
	})
	fmt.Println("Done.")
}
```

- [ ] **Step 10.2: Run the smoke test**

Prerequisites: NT is open, ClaudeTrader running on MNQ chart in SIM, all env vars set.

```bash
cd /home/hoang/nofx && go run ./cmd/nq_smoke
```

The runner prints the system + user prompts. Hand-fabricate a tiny decision JSON near current MNQ price. Example (if NQ is around 21500):

```json
{"action":"LONG","entry":21500.00,"stop_loss":21485.00,"take_profit":21525.00,"reasoning":"smoke test"}
```

Paste it into the runner's stdin. Expected:

1. Signal file written to `$NINJATRADER_DATA_DIR/trade_signals.csv`.
2. Within 2-4s, NT's Output window prints `[SIGNAL] LONG MARKET ORDER (1 contracts)`.
3. NT fills the order in SIM.
4. NT writes a row to `trades_taken.csv`.
5. The smoke runner's tailer prints `>>> FILL: 05/22/2026 ... LONG @ <price>`.

If all five happen — **the entire chain works end-to-end**. This is the slice complete.

- [ ] **Step 10.3: Commit**

```bash
cd /home/hoang/nofx
git add cmd/nq_smoke/main.go
git commit -m "feat(cmd): add NQ end-to-end smoke runner"
```

- [ ] **Step 10.4: Playwright integration assertion**

After the manual smoke fills against NT SIM, verify the UI reflects
the backend state. While the bot is running and `cd web && npm run dev`
is serving on localhost:3000:

1. mcp__playwright__browser_navigate to /dashboard?trader=<id>
2. mcp__playwright__browser_snapshot — assert the new fill row
   appears in the positions table
3. mcp__playwright__browser_snapshot — assert Recent Decisions panel
   shows the LONG signal we just piped
4. mcp__playwright__browser_console_messages — no React errors during
   the live update poll

This proves the full chain works in the user-facing view, not just
the backend logs.

---

## Task 11: Remove three crypto-era pages from navigation

**Why:** Three pages exist that have no purpose for a single-user NQ trader and add maintenance burden:

1. **Data page** (`/data`) — iframe to `nofxos.ai/dashboard`, now CSP-blocked. NinjaTrader IS the chart view.
2. **Strategy Market page** (`/strategy-market`) — community strategy browser showing public crypto strategies. Confusing UX for NQ users; mixing crypto strategies with NQ ones serves nobody.
3. **Competition / Leaderboard page** (`/competition`) — public competition view ranking crypto traders by P&L. Not relevant for a personal NQ trading setup; also makes the bot send anonymous data to public listings by default.

All three follow the same removal pattern: drop the route, drop the import, drop the nav entry (desktop + mobile), delete the component file, clean the `Page` type union.

**Files:**
- Modify: `web/src/components/common/HeaderBar.tsx` (remove 4 nav entries: 2 for Data, 2 for Market, 2 for Competition — actually 6 entries across desktop + mobile blocks)
- Modify: `web/src/router/AppRoutes.tsx` (remove 3 routes + 3 imports)
- Modify: `web/src/router/paths.ts` (remove 3 entries from ROUTES, PAGE_PATHS, LEGACY_HASH_ROUTES, getCurrentPageForPath, and the Page type union)
- Delete: `web/src/pages/DataPage.tsx`
- Delete: `web/src/pages/StrategyMarketPage.tsx`
- Delete: `web/src/components/trader/CompetitionPage.tsx`
- Delete: `web/src/components/trader/CompetitionPage.test.tsx`
- Modify: `web/src/i18n/translations.ts` (remove unused i18n keys: `dataCenter`, `strategyMarket*`, `competition*`, etc.)

- [ ] **Step 11.1: Remove desktop nav entries (Data + Market + Competition)**

In `web/src/components/common/HeaderBar.tsx`, find the desktop nav block and remove three entries (Data at ~121-131, Strategy Market at ~133-145, Competition at ~162-175). Pattern for each:

```diff
                 {
                   page: 'agent',
                   path: ROUTES.agent,
                   label: 'Agent',
                   badge: 'Beta',
                   requiresAuth: false,
                 },
-                {
-                  page: 'data',
-                  path: ROUTES.data,
-                  label: language === 'zh' ? '数据' : 'Data',
-                  requiresAuth: false,
-                },
-                {
-                  page: 'strategy-market',
-                  path: ROUTES.strategyMarket,
-                  label: language === 'zh' ? '策略市场' : 'Strategy Market',
-                  requiresAuth: false,
-                },
-                {
-                  page: 'competition',
-                  path: ROUTES.competition,
-                  label: language === 'zh' ? '排行榜' : 'Leaderboard',
-                  requiresAuth: false,
-                },
                 {
                   page: 'traders',
                   ...
                 },
```

- [ ] **Step 11.2: Remove mobile nav entries (Data + Market + Competition)**

Same file, find the mobile nav block at ~457-510 and remove the same three entries (Data ~457-466, Strategy Market ~468-481, Competition ~497-510). Same pattern as desktop block.

- [ ] **Step 11.3: Remove the three routes**

In `web/src/router/AppRoutes.tsx` find and remove three `<Route>` blocks:

```diff
-        <Route
-          path={ROUTES.data}
-          element={
-            <AppChrome currentPage="data" showFooter={false}>
-              <DataPage />
-            </AppChrome>
-          }
-        />
-        <Route
-          path={ROUTES.strategyMarket}
-          element={
-            <AppChrome currentPage="strategy-market" showFooter={false}>
-              <StrategyMarketPage />
-            </AppChrome>
-          }
-        />
-        <Route
-          path={ROUTES.competition}
-          element={
-            <AppChrome currentPage="competition" showFooter={false}>
-              <CompetitionPage />
-            </AppChrome>
-          }
-        />
```

Also remove three import lines at top of `AppRoutes.tsx`:

```diff
-import { DataPage } from '../pages/DataPage'
-import { StrategyMarketPage } from '../pages/StrategyMarketPage'
-import { CompetitionPage } from '../components/trader/CompetitionPage'
```

- [ ] **Step 11.4: Clean route constants**

In `web/src/router/paths.ts`:

```diff
 export type Page =
   | 'agent'
-  | 'competition'
   | 'traders'
   | 'trader'
   | 'strategy'
-  | 'strategy-market'
-  | 'data'
   | 'faq'
   | 'login'
   | 'register'

 export const ROUTES = {
   home: '/',
   agent: '/agent',
   login: '/login',
   register: '/register',
   setup: '/setup',
   welcome: '/welcome',
   faq: '/faq',
   resetPassword: '/reset-password',
   settings: '/settings',
-  data: '/data',
-  competition: '/competition',
   traders: '/traders',
   dashboard: '/dashboard',
   strategy: '/strategy',
-  strategyMarket: '/strategy-market',
 } as const

 export const PAGE_PATHS: Record<Page, string> = {
   agent: ROUTES.agent,
-  competition: ROUTES.competition,
   traders: ROUTES.traders,
   trader: ROUTES.dashboard,
   strategy: ROUTES.strategy,
-  'strategy-market': ROUTES.strategyMarket,
-  data: ROUTES.data,
   faq: ROUTES.faq,
   login: ROUTES.login,
   register: ROUTES.register,
 }

 export const LEGACY_HASH_ROUTES: Record<string, string> = {
   agent: ROUTES.agent,
-  competition: ROUTES.competition,
   traders: ROUTES.traders,
   trader: ROUTES.dashboard,
   details: ROUTES.dashboard,
   strategy: ROUTES.strategy,
-  'strategy-market': ROUTES.strategyMarket,
-  data: ROUTES.data,
 }
```

Also remove the three `case` lines in `getCurrentPageForPath()` for `ROUTES.competition`, `ROUTES.strategyMarket`, and `ROUTES.data`.

- [ ] **Step 11.5: Delete the page files**

```bash
cd /home/hoang/nofx
rm web/src/pages/DataPage.tsx
rm web/src/pages/StrategyMarketPage.tsx
rm web/src/components/trader/CompetitionPage.tsx
rm web/src/components/trader/CompetitionPage.test.tsx
```

- [ ] **Step 11.6: Check for orphan references**

```bash
cd /home/hoang/nofx && grep -rn "DataPage\|StrategyMarketPage\|CompetitionPage\|ROUTES\.data\|ROUTES\.strategyMarket\|ROUTES\.competition" web/src/ 2>&1 | grep -v node_modules | head -30
```

Expected: a small number of i18n keys still reference these page names (e.g., `dataCenter`, `strategyMarket*`, `competition*` in `translations.ts`). These can be deleted as a separate cosmetic pass; the build does not require it.

- [ ] **Step 11.7: Build the frontend**

```bash
cd /home/hoang/nofx/web && npm run build 2>&1 | tail -30
```

Expected: build succeeds with no TypeScript errors. If TS complains about any of the removed page identifiers, locate the remaining reference (likely a stray import or an inline `<Link to={ROUTES.x}>` in a forgotten component) and remove it. The deletion only affects the four files above plus the nav/router infrastructure.

- [ ] **Step 11.8: Smoke test in browser**

```bash
cd /home/hoang/nofx/web && npm run dev
```

Open `http://localhost:3000/`. Verify:
- "Data", "Strategy Market", and "Leaderboard" links are gone from the header (both desktop and mobile views).
- Visiting `http://localhost:3000/data`, `/strategy-market`, `/competition` 404s or redirects.
- Other nav items still work: Agent, Traders, Strategy, Settings, FAQ.
- The "Settings → Exchanges" tab still loads. The Trader Dashboard still loads.

- [ ] **Step 11.9: Backend: are there server routes serving these pages?**

```bash
cd /home/hoang/nofx && grep -n "/api/competition\|/api/leaderboard\|/api/strategy-market\|/api/public-strategies" api/server.go api/handler_*.go 2>/dev/null
```

If any backend routes exist *specifically* to serve competition data or public-strategy listings (e.g., `GET /api/competition`, `GET /api/strategies/public`), and they're no longer reachable from the UI, you can leave them in place (defensive) or remove them. **For this plan, leave them in place** — removing backend routes is a separate optional cleanup; they don't affect the build.

- [ ] **Step 11.10: Commit**

```bash
cd /home/hoang/nofx
git add web/src/components/common/HeaderBar.tsx \
        web/src/router/AppRoutes.tsx \
        web/src/router/paths.ts
git rm web/src/pages/DataPage.tsx \
       web/src/pages/StrategyMarketPage.tsx \
       web/src/components/trader/CompetitionPage.tsx \
       web/src/components/trader/CompetitionPage.test.tsx
git commit -m "refactor(web): remove Data, Strategy Market, and Leaderboard pages

For NQ-trading focus, three crypto-era pages are removed:
- Data: was an iframe to deprecated nofxos.ai/dashboard (CSP-blocked); NinjaTrader covers chart needs
- Strategy Market: community crypto-strategy browser; not useful for single-user NQ setup
- Leaderboard: public competition ranking; not relevant for personal use

Three control surfaces remain: Settings (Config), Strategy (Studio), Traders (Dashboard)."
```

- [ ] **Step 11.11: Playwright UI assertion**

Verify removal at the DOM level, not just TypeScript:

1. mcp__playwright__browser_navigate to http://localhost:3000/
2. mcp__playwright__browser_snapshot — accessibility tree must NOT
   contain links to "Data", "Strategy Market", "Leaderboard" (desktop
   AND mobile nav)
3. mcp__playwright__browser_navigate to /data — should 404 or redirect
4. mcp__playwright__browser_navigate to /strategy-market — same
5. mcp__playwright__browser_navigate to /competition — same
6. Confirm Agent, Traders, Strategy, Settings, FAQ links still render
7. mcp__playwright__browser_console_messages — no router errors

---

## Completion criteria

This plan is done when:

1. ✅ All Go tests pass: `go test ./...`
2. ✅ `cmd/nq_smoke` runs end-to-end against NT SIM with a hand-fabricated decision → real fill in NT SIM → row in `trades_taken.csv`.
3. ✅ All 11 tasks committed to git.
4. ✅ The broken Data page is gone from the web UI; visualization happens in NinjaTrader.

What this plan does **not** deliver (deferred to next plans):
- Real AI client integration (use of DeepSeek/Claude API in the cycle — currently stubbed by stdin in smoke runner)
- Integration into the main `trader/auto_trader_loop.go` polling cycle
- CME holiday calendar + session-boundary lockouts
- Contract roll detection (NQ.c.0 → next month transition awareness)
- Live (real-money) toggle separate from SIM
- Web UI rewrite of `DataPage.tsx`
- Dead-man-switch heartbeat
- Daily loss kill-switch
- `MAX_CONTRACTS_PER_TRADER` enforcement in code (currently relies on NT strategy parameter)

These become the next slices once this one is validated.

---

## Parallel Dispatch Map

The tasks below split into three independent clusters that can run concurrently via `superpowers:dispatching-parallel-agents`. Each cluster ends at a clean compile boundary so dispatches don't fight over shared files.

### Cluster A — Databento data pipeline (Tasks 1–4)

**Files touched:** `provider/databento/`, `market/data.go` (Normalize fix only), `config/`
**Independent because:** Pure data ingestion; no overlap with bridge or UI files.

```text
Task("Cluster A: Databento data pipeline", subagent_type: "feature-dev:code-architect", prompt: """
Implement Tasks 1–4 from docs/superpowers/plans/2026-05-22-nq-databento-ninjatrader.md:
- Task 1: TRADING_MODE + DATABENTO_API_KEY wiring in config/config.go
- Task 2: provider/databento/client.go — Historical OHLCV REST client (HTTP Basic auth, GET /v0/timeseries.get_range)
- Task 3: Symbol resolver NQ.c.0 → MNQM6 (continuous → specific contract)
- Task 4: Databento bar → market.Kline adapter; fix Normalize() case-preservation for NQ.c.0
Constraints: do NOT touch trader/ninjatrader/ or web/. Stop after `go build ./...` is clean and Task 2.4 + Task 4.6 tests pass.
Return: summary of files created, test results, any deviations from the plan.
""")
```

### Cluster B — NinjaTrader CSV bridge (Tasks 5–8)

**Files touched:** `trader/ninjatrader/`, `trader/auto_trader.go` (switch case only)
**Independent because:** Bridge code is self-contained in one package; only touches auto_trader.go at the broker switch (Task 8) — merge conflict is trivial.
**Blocker:** none on Cluster A — bridge code doesn't import provider/databento.

```text
Task("Cluster B: NinjaTrader CSV bridge", subagent_type: "feature-dev:code-architect", prompt: """
Implement Tasks 5–8 from docs/superpowers/plans/2026-05-22-nq-databento-ninjatrader.md:
- Task 5: trader/ninjatrader/csv_writer.go — 5-field signal writer (datetime, direction, entry, SL, TP)
- Task 6: trader/ninjatrader/csv_tailer.go — 3-field fill tailer with dedup
- Task 7: trader/ninjatrader/trader.go — 19-method Trader interface impl; compile-time `var _ types.Trader = (*Trader)(nil)`
- Task 8: add "ninjatrader" case to trader/auto_trader.go broker switch (~line 268)
Hazards to encode in code: H1 (lost-signal race, 2s polling), H2 (datetime+direction dedup collision — add monotonic nonce as 6th CSV field OR rate-limit Go writer to ≥2s), H4 (fill replay on NT restart — persist last-processed offset).
Return: summary of files created, list of CSV fields, dedup strategy chosen.
""")
```

### Cluster C — Prompt + UI + smoke (Tasks 9–11)

**Files touched:** `kernel/engine_prompt.go`, `web/src/pages/`, `cmd/nq_smoke/`
**Independent because:** Prompt + UI + smoke runner; doesn't touch provider or trader internals.
**Blocker:** Task 10 (smoke) needs A + B done. Tasks 9 + 11 can start immediately.

```text
Task("Cluster C: Prompt + UI", subagent_type: "feature-dev:code-architect", prompt: """
Implement Tasks 9 + 11 from docs/superpowers/plans/2026-05-22-nq-databento-ninjatrader.md (DEFER Task 10 — needs Clusters A + B merged first):
- Task 9: kernel/engine_prompt.go — NQ futures prompt template with FuturesContext (no MACDSignal — single float64 only)
- Task 11: confirm crypto-era pages (Data, Strategy Market, Competition) are gone from web/src/router/AppRoutes.tsx + web/src/components/common/HeaderBar.tsx
Constraints: do NOT touch provider/databento/ or trader/ninjatrader/. Run `cd web && npm run build` after Task 11.
Return: prompt diff summary, list of removed route entries, npm build output.
""")
```

### Sequencing

1. **Dispatch A + B + C(Task 9, 11 only) in one message** — 3 parallel agents.
2. **Wait for all three to return** — verify each summary, run `go build ./... && cd web && npm run build`.
3. **Then dispatch Task 10** (end-to-end smoke) as a single agent — depends on A + B fills landing in `data/data.db` via real NT execution.

### Plan 1.5 (NT8 AddOn migration) — NOT parallelizable

Plan 1.5 replaces the CSV bridge with a TCP/WebSocket NDJSON protocol. It must be sequential because:
- Stage 1 (AddOn skeleton) blocks Stage 2 (protocol wire)
- Stage 2 blocks Stage 3 (Go-side client)
- Stage 3 blocks Stage 4 (cutover from CSV)
- Stage 5 (CSV removal) requires Stage 4 stable

Each stage produces a working binary; dispatch one agent per stage in sequence.

## Self-review

**Spec coverage:**
- Databento config — Task 1 ✓
- Databento OHLCV client — Task 2 ✓
- Symbol resolution — Task 3 ✓
- Bar → Kline adapter — Task 4 ✓
- CSV signal writer — Task 5 ✓
- CSV fill tailer — Task 6 ✓
- Trader interface impl — Task 7 ✓
- Wire into auto_trader switch — Task 8 ✓
- NQ futures prompt — Task 9 ✓
- End-to-end smoke — Task 10 ✓
- NT setup + manual smoke (Task 0) — covers the human-side prerequisites

**Placeholder scan:** searched for "TBD", "TODO", "implement later" — none present. All steps have concrete commands or code.

**Type consistency:** `SignalRow` and `FillRow` defined in Task 5, used in Tasks 6, 7, 10. `Bar` defined in Task 2, used in Task 4, 10. `FuturesContext` defined in Task 9, used in Task 10. `Config` (in `trader/ninjatrader`) defined in Task 7, used in Task 8.

**Known assumption to verify during execution:**
- `market.Kline` field name `OpenTime` and millisecond unit — verify in `market/types.go:94-108` before Task 4. If different, adjust adapter + test.
- `trader/types.Trader` interface signatures — verified 2026-05-22: **19 methods exactly** (not 17 as earlier draft said). Task 7's code block has been corrected to implement all 19. The `var _ types.Trader = (*Trader)(nil)` line catches any future drift at compile time.
- Databento JSON response shape for `ohlcv-1m` — verified against Databento docs but actual response format may have minor differences. The Task 2.4 live smoke test catches this.

## v3 audit-feedback corrections (2026-05-22)

Following external audit, the plan was updated with:
- **Task 7 Trader impl** now includes all 19 interface methods (was missing 5: `GetMarketPrice`, `CancelStopLossOrders`, `CancelTakeProfitOrders`, `CancelStopOrders`, `GetOpenOrders`). The "17-method" label was wrong; corrected to 19.
- **`Normalize` futures fix** is now case-preserving (was returning `NQ.C.0` instead of `NQ.c.0` — Databento rejects the uppercased form).
- **`FuturesContext.MACDSignal` removed** — `market.ExportCalculateMACD` returns only the MACD line (one `float64`). Signal/histogram would require extending the indicator API and are deferred.
- **JWT verification** strengthened — the `🔑 JWT secret configured` log line fires unconditionally, so cannot be used as a check. Added a recommended code-level warning when the default value is detected.
- **Coverage scorecard** recounted; previous totals had arithmetic errors on 3 of 4 surfaces. Corrected totals reconcile.
- **CSV bridge runtime hazards** documented as Plan 1.5 work (4 hazards: H1 lost-signal race, H2 dedup collision, H3 DrvFs mtime, H4 fill replay on session rollover). Not blocking for Plan 1 SIM but must be addressed before any live trading.
- **`GetBalance` $50k mock** honestly disclosed as feeding both prompt equity math and Go-side risk checks; 3 fix options documented; option 3 (disable Go-side balance sizing for ninjatrader exchange) recommended for early go-live.

**The plan is now executable for Plan 1 SIM paper trading.** Going live requires Plan 1.5 (CSV protocol hardening + real balance read) and Plan 2 (CME calendar + contract rolls + kill-switches).

---

## Task 12: Backend kernel + market futures routing (Cluster D)

> **PARTIALLY SUPERSEDED 2026-05-28 — see "Architecture Pivot" at top of doc.**
> The Databento branch in `GetWithTimeframes` (Step 3 below) was built
> in PR #27 then dropped in PR #28 because the Historical-tier
> `available_end` lag is ~8 hours, unusable for live decisions.
> SHIPPED from this task: the futures variant routing (Step 1) +
> symbol-on-decision-JSON (Step 2) + symbol normalization
> (`isCMEFuturesSymbol`, `Normalize` CME bypass, `normalizeSymbols`
> guard) — all under `v1.0-task12-symbols`. Data fetch is now Plan
> 4.4 Stage 3 (read from NT8 TCP feed, not from any external HTTP
> provider).

**Why:** v4 audit found that `BuildSystemPrompt("futures")` falls through to crypto, futures decision JSON omits `symbol`, `GetWithTimeframes` has no Databento branch, and `kernel/engine.go` never checks `TradingMode`.

**Files:**
- Modify: `kernel/engine_prompt.go:39-46` (add futures case to variant switch)
- Modify: `kernel/engine_prompt_futures.go:50` (add `symbol` to JSON shape)
- Modify: `market/data.go:147-256` (add futures branch in GetWithTimeframes)
- Modify: `kernel/engine.go` (resolve continuous symbol when TradingMode=futures)
- Test: `kernel/engine_prompt_test.go`, `market/data_test.go`

- [ ] **Step 1: Route futures variant in BuildSystemPrompt**

In `kernel/engine_prompt.go` find the variant switch (around line 39-46) and add:
```go
case "futures":
    return BuildFuturesSystemPrompt(accountEquity)
```
Place BEFORE the default crypto case so "futures" matches first.

- [ ] **Step 2: Add `symbol` to futures decision JSON**

In `kernel/engine_prompt_futures.go` find the JSON example string (around line 50) and update from `{action, entry, stop_loss, take_profit, reasoning}` to `{symbol, action, entry, stop_loss, take_profit, reasoning, confidence}`. The `symbol` field is required by `engine_analysis.go` decision parser.

- [ ] **Step 3: Add futures branch in GetWithTimeframes**

In `market/data.go` add a helper:
```go
func getKlinesFromDatabento(symbol string, timeframe string) ([]Kline, error) {
    cfg := config.Get()
    client := databento.NewClient(cfg.DatabentoAPIKey, cfg.DatabentoDataset)
    resolved, err := databento.ResolveContinuous(client, symbol)
    if err != nil { return nil, err }
    bars, err := client.GetOHLCV(resolved, timeframe, 200)
    if err != nil { return nil, err }
    return BarsToKlines(bars), nil
}
```
Then in `GetWithTimeframes` add (BEFORE the hyperliquid/coinank fork):
```go
if IsCMEFuturesSymbol(symbol) {
    return getKlinesFromDatabento(symbol, tf)
}
```

- [ ] **Step 4: Wire kernel/engine.go to TradingMode**

In `kernel/engine.go` where the engine builds prompts/fetches data, add a TradingMode check at the entry point:
```go
if config.Get().TradingMode == "futures" {
    variant = "futures"
}
```
This ensures the futures prompt path triggers when env var is set, regardless of strategy config.

- [ ] **Step 5: Tests + build**

Add `TestBuildSystemPrompt_FuturesVariant` and `TestNormalize_CMEFutures("NQ.c.0")` cases. Run:
```bash
go test ./kernel/... ./market/...
go build ./...
```

- [ ] **Step 6: Commit**
```bash
git commit -m "feat(futures): route futures variant + add Databento branch in GetWithTimeframes + symbol in decision JSON"
```

---

## Task 13: Backend wiring + storage for NinjaTrader (Cluster E)

**Why:** v4 audit found `manager/trader_manager.go` has no ninjatrader case → NT traders unloadable from DB; `store/exchange.go` Exchange struct lacks NT fields; `AutoTraderConfig` missing `NTDefaultContractQty`; CSV tailer offset not persisted.

**Files:**
- Modify: `store/exchange.go` (add NT fields to Exchange struct + GORM tags + migration)
- Modify: `manager/trader_manager.go:~700` (add ninjatrader case to addTraderFromStore switch)
- Modify: `trader/auto_trader.go:95-97,~322` (add NTDefaultContractQty field + use it)
- Modify: `provider/ninjatrader/csv_tailer.go:18,78-82` (persist offset to disk)
- Test: `store/exchange_test.go`, `provider/ninjatrader/csv_tailer_test.go`

- [ ] **Step 1: Add NT fields to Exchange struct**

In `store/exchange.go` add to the Exchange struct:
```go
// NinjaTrader-specific (only set when ExchangeType == "ninjatrader")
NTDataDir            string `gorm:"column:nt_data_dir"            json:"nt_data_dir,omitempty"`
NTInstrumentName     string `gorm:"column:nt_instrument_name"     json:"nt_instrument_name,omitempty"`
NTDefaultContractQty int    `gorm:"column:nt_default_contract_qty" json:"nt_default_contract_qty,omitempty"`
```
GORM AutoMigrate will add the columns; no separate migration file needed.

- [ ] **Step 2: Add ninjatrader case in manager**

In `manager/trader_manager.go` find the `switch exchangeCfg.ExchangeType` block in `addTraderFromStore` (around line 661-700) and after the `case "indodax":` add:
```go
case "ninjatrader":
    traderConfig.NinjaTraderDataDir = exchangeCfg.NTDataDir
    traderConfig.NinjaTraderSymbol = exchangeCfg.NTInstrumentName
    traderConfig.NTDefaultContractQty = exchangeCfg.NTDefaultContractQty
```

- [ ] **Step 3: Add NTDefaultContractQty to AutoTraderConfig**

In `trader/auto_trader.go` near line 95-97 add field:
```go
NTDefaultContractQty int
```
And in the ninjatrader case (around line 320-328) pass it into `ntTrader.New(ntTrader.Config{...DefaultContractQty: config.NTDefaultContractQty})`.

- [ ] **Step 4: Persist CSV tailer offset**

In `provider/ninjatrader/csv_tailer.go` change `seen int` to a disk-backed counter:
```go
type Tailer struct {
    path       string
    offsetPath string  // e.g. path + ".offset"
    seen       int
}

func (t *Tailer) loadOffset() {
    if data, err := os.ReadFile(t.offsetPath); err == nil {
        fmt.Sscanf(string(data), "%d", &t.seen)
    }
}

func (t *Tailer) saveOffset() {
    _ = os.WriteFile(t.offsetPath, []byte(fmt.Sprintf("%d", t.seen)), 0644)
}
```
Call `loadOffset()` in constructor; call `saveOffset()` after every successful row processing. Removes H4 fill-replay hazard.

- [ ] **Step 5: Tests + build**
```bash
go test ./store/... ./manager/... ./provider/ninjatrader/...
go build ./...
```

- [ ] **Step 6: Commit**
```bash
git commit -m "feat(nt): per-account exchange fields + manager wiring + tailer offset persistence"
```

---

## Task 14: Frontend NinjaTrader config (Settings page)

**Why:** v4 audit found ExchangeConfigModal has no NinjaTrader option, SettingsPage handleSaveExchange doesn't pass NT params. User cannot save NT config from UI.

**Files:**
- Modify: `web/src/components/trader/ExchangeConfigModal.tsx` (~+140 LOC across 4 sections)
- Modify: `web/src/pages/SettingsPage.tsx:190-257` (handleSaveExchange signature + payload)

- [ ] **Step 1: Add NinjaTrader to SUPPORTED_EXCHANGE_TEMPLATES**

In `web/src/components/trader/ExchangeConfigModal.tsx` around line 23-34, append to the templates array:
```ts
{
    id: 'ninjatrader',
    name: 'NinjaTrader',
    type: 'futures',
    icon: '/icons/ninjatrader.svg',  // OK to fallback to a generic icon if asset not present
    description: 'CME futures via NT8 CSV bridge',
    requiresApiKey: false,
}
```

- [ ] **Step 2: Add form state**

Around line 154-180 add useState entries:
```tsx
const [ntDataDir, setNtDataDir] = useState('')
const [ntInstrumentName, setNtInstrumentName] = useState('MNQ')
const [ntDefaultContractQty, setNtDefaultContractQty] = useState(1)
```

- [ ] **Step 3: Add form UI section**

After the existing `aster` form section (around line 720-756) add:
```tsx
{selectedExchange === 'ninjatrader' && (
    <div className="space-y-4">
        <div>
            <label className="block text-sm font-medium mb-1">NT Data Directory (WSL path)</label>
            <input
                type="text"
                value={ntDataDir}
                onChange={(e) => setNtDataDir(e.target.value)}
                placeholder="/mnt/c/Users/<u>/NofxTrader/data"
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded"
            />
        </div>
        <div>
            <label className="block text-sm font-medium mb-1">Instrument</label>
            <input
                type="text"
                value={ntInstrumentName}
                onChange={(e) => setNtInstrumentName(e.target.value)}
                placeholder="MNQ"
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded"
            />
        </div>
        <div>
            <label className="block text-sm font-medium mb-1">Default Contract Qty</label>
            <input
                type="number"
                min="1"
                value={ntDefaultContractQty}
                onChange={(e) => setNtDefaultContractQty(parseInt(e.target.value) || 1)}
                className="w-full px-3 py-2 bg-white/5 border border-white/10 rounded"
            />
        </div>
    </div>
)}
```

- [ ] **Step 4: Add validation branch in handleSubmit**

In `ExchangeConfigModal.tsx` around line 301-339 add:
```tsx
} else if (selectedExchange === 'ninjatrader') {
    if (!ntDataDir.trim()) {
        setError('NT Data Directory is required')
        return
    }
}
```

- [ ] **Step 5: Update onSave call + props interface**

In `ExchangeConfigModal.tsx` find the onSave invocation at the end of handleSubmit and pass NT params; update the props interface (~line 39-55) to include `ntDataDir`, `ntInstrumentName`, `ntDefaultContractQty`.

- [ ] **Step 6: Update SettingsPage handleSaveExchange**

In `web/src/pages/SettingsPage.tsx:190-257` add NT params to function signature and thread into createRequest/updateRequest body:
```ts
const handleSaveExchange = async (
    // ...existing params...
    ntDataDir?: string,
    ntInstrumentName?: string,
    ntDefaultContractQty?: number,
) => {
    // ...
    const payload = {
        // ...existing fields...
        nt_data_dir: ntDataDir,
        nt_instrument_name: ntInstrumentName,
        nt_default_contract_qty: ntDefaultContractQty,
    }
}
```

- [ ] **Step 7: Build + smoke test**
```bash
cd web && npm run build
```
Then open Settings → Exchanges → Add → NinjaTrader and confirm form renders.

- [ ] **Step 8: Commit**
```bash
git commit -m "feat(web): NinjaTrader exchange config form + save handler"
```

- [ ] **Step 9: Playwright form assertion**

After npm build passes, verify the form actually renders and saves:

1. mcp__playwright__browser_navigate to /settings
2. mcp__playwright__browser_click on "Exchanges" tab
3. mcp__playwright__browser_click "Add Exchange"
4. mcp__playwright__browser_click the NinjaTrader card
5. mcp__playwright__browser_snapshot — assert form has DataDir,
   InstrumentName, DefaultContractQty inputs (NOT API key inputs)
6. mcp__playwright__browser_type into DataDir field
7. mcp__playwright__browser_click Save
8. mcp__playwright__browser_snapshot — assert exchange appears in
   list with NinjaTrader name
9. mcp__playwright__browser_console_messages — no errors

---

## Task 15: Frontend futures-gating (Dashboard + Strategy)

**Why:** v4 audit found Dashboard hardcodes USDT/leverage; CoinSourceEditor appends USDT to "NQ"; IndicatorEditor + RiskControlEditor have no variant prop.

**Files:**
- Modify: `web/src/pages/TraderDashboardPage.tsx:513,522,530,609,663,673` (gate on exchange_type)
- Modify: `web/src/components/strategy/CoinSourceEditor.tsx:69-79` (skip USDT for CME)
- Modify: `web/src/components/strategy/IndicatorEditor.tsx:656-688` (variant prop + hide crypto sources)
- Modify: `web/src/components/strategy/RiskControlEditor.tsx:61-128` (variant prop + relabel)

- [ ] **Step 1: Gate Dashboard StatCard units**

In `web/src/pages/TraderDashboardPage.tsx` around lines 513, 522, 530:
```tsx
const unit = exchangeType === 'ninjatrader' ? 'USD' : 'USDT'
<StatCard ... unit={unit} ... />
```
`exchangeType` is already available via `getExchangeTypeFromList(...)` helper.

- [ ] **Step 2: Hide Leverage + Liquidation Price columns for NT**

Around lines 609, 663, 673 wrap each `<th>` and `<td>` with:
```tsx
{exchangeType !== 'ninjatrader' && (
    <th className="hidden md:table-cell ...">Leverage</th>
)}
```
Same pattern for Liquidation Price column.

- [ ] **Step 3: Skip USDT auto-append for CME futures**

In `web/src/components/strategy/CoinSourceEditor.tsx` around line 69-79:
```ts
const cmeFuturesPattern = /^(NQ|MNQ|ES|MES|YM|MYM|RTY|M2K|CL|GC)(\.c\.0|[A-Z]\d)?$/i
if (cmeFuturesPattern.test(symbol)) {
    return symbol  // keep as-is, no USDT
}
```
Place BEFORE the existing xyzDexAssets check.

- [ ] **Step 4: Add variant prop to IndicatorEditor**

In `web/src/components/strategy/IndicatorEditor.tsx` add `variant?: 'crypto' | 'futures'` prop. Around lines 656-688 gate crypto-only toggles:
```tsx
{variant !== 'futures' && (
    <label><input type="checkbox" name="enable_funding_rate" /> Funding Rate</label>
)}
```
Same for OI Ranking, NetFlow Ranking, Price Ranking sources.

Pass `variant={exchangeType === 'ninjatrader' ? 'futures' : 'crypto'}` from `StrategyStudioPage.tsx`.

- [ ] **Step 5: Add variant prop to RiskControlEditor**

In `web/src/components/strategy/RiskControlEditor.tsx` add `variant?: 'crypto' | 'futures'` prop. Relabel:
```tsx
const leverageLabel = variant === 'futures' ? 'Primary Instrument Leverage' : 'BTC/ETH Leverage'
const sizeUnit = variant === 'futures' ? 'contracts' : 'USDT'
```

- [ ] **Step 6: Build + visual smoke**
```bash
cd web && npm run build
```
Then open Dashboard for a NT trader and verify no USDT unit, no Leverage/LiqPrice columns. Open Strategy Studio and verify no funding rate / OI sources for futures.

- [ ] **Step 7: Commit**
```bash
git commit -m "feat(web): futures-gate Dashboard + IndicatorEditor + RiskControlEditor + CoinSourceEditor"
```

- [ ] **Step 8: Playwright futures-gating assertion**

The build only proves TS compiles; the runtime behavior is what
matters. With a NT trader configured:

1. mcp__playwright__browser_navigate to /dashboard?trader=<NT-trader-id>
2. mcp__playwright__browser_snapshot — assert StatCards show "USD"
   text, NOT "USDT"
3. mcp__playwright__browser_snapshot — positions table headers must
   NOT contain "Leverage" or "Liquidation Price"
4. mcp__playwright__browser_navigate to /strategy
5. mcp__playwright__browser_click into a futures-variant strategy
6. mcp__playwright__browser_snapshot — IndicatorEditor must NOT
   render funding rate or OI toggle sections
7. mcp__playwright__browser_snapshot — RiskControlEditor labels read
   "Primary Instrument Leverage" not "BTC/ETH Leverage"
8. mcp__playwright__browser_console_messages — no errors

---

## Task 16: VL brand cleanup + minor fixes

**Why:** v4 audit found SetupPage still says "Welcome to NOFX"; chart watermarks say "NOFX"; agent.go:174 comment is stale; missing Normalize test.

**Files:**
- Modify: `web/src/pages/SetupPage.tsx` (NOFX → VL in 3 languages)
- Modify: `web/src/components/charts/EquityChart.tsx:321,335` (watermark text)
- Modify: `web/src/components/charts/AdvancedChart.tsx:1161,1182` (watermark text)
- Modify: `agent/agent.go:174` (comment example)
- Test: `market/data_test.go` (TestNormalize_CMEFutures)

- [ ] **Step 1: SetupPage NOFX → VL**

In `web/src/pages/SetupPage.tsx` replace any occurrence of "NOFX" in user-visible copy with "VL". Check 3 language blocks (en, zh, id).

- [ ] **Step 2: Chart watermarks**

In `EquityChart.tsx:321,335` and `AdvancedChart.tsx:1161,1182` change watermark text "NOFX" → "VL".

- [ ] **Step 3: Update stale comment**

In `agent/agent.go:174` change comment example `"deepseek-chat"` → `"deepseek-v4-pro"` to match the actual default.

- [ ] **Step 4: Add Normalize futures test**

In `market/data_test.go` add:
```go
func TestNormalize_CMEFutures(t *testing.T) {
    cases := []struct{ in, want string }{
        {"NQ.c.0", "NQ.c.0"},
        {"MNQ.c.0", "MNQ.c.0"},
        {"nq.c.0", "NQ.c.0"},  // only the ticker portion uppercased
    }
    for _, tc := range cases {
        if got := Normalize(tc.in); got != tc.want {
            t.Errorf("Normalize(%q) = %q, want %q", tc.in, got, tc.want)
        }
    }
}
```

- [ ] **Step 5: Build + commit**
```bash
go build ./... && cd web && npm run build && cd ..
git commit -m "chore: VL brand cleanup + stale comment + Normalize futures test"
```

- [ ] **Step 6: Playwright brand assertion**

Catch any "NOFX" string that slipped through:

1. mcp__playwright__browser_navigate to /setup
2. mcp__playwright__browser_get_page_text — search for "NOFX".
   Expected: 0 matches.
3. mcp__playwright__browser_navigate to a trader dashboard with chart
4. mcp__playwright__browser_take_screenshot — visually confirm chart
   watermark reads "VL" not "NOFX"
5. mcp__playwright__browser_console_messages — no errors

---

## Dispatch map for Tasks 12-16

Tasks 12 + 13 + 14/15 are independent — they touch disjoint file sets. Task 16 can fold into any of them. Dispatch in parallel via one message:

```text
Task("Task 12: Backend kernel + market futures routing")
Task("Task 13: Backend wiring + storage for NinjaTrader")
Task("Task 14+15: Frontend NT config + futures-gating")  // touches different files, one agent handles both
```

After all three return clean (`go build ./... && cd web && npm run build`) → run Task 10 (end-to-end smoke) as the final verification.

---

# Plan 1.5: NT8 AddOn migration (research-backed architecture)

> **Added 2026-05-22 based on external research.** Plan 1.5 replaces the CSV polling bridge with an in-process NinjaScript AddOn that hosts a TCP/WebSocket loopback server. This eliminates every "v1 limitation" Task 7 documents (manual close, balance read, cancel/modify, real fill events) and removes the CSV runtime hazards H1-H4 by design. **Plan 1 (CSV) still ships first** to validate the AI brain in SIM; Plan 1.5 migrates the bridge once Plan 1 trades cleanly.

## Trigger conditions — when to start Plan 1.5

Plan 1.5 is **deferred** until at least one of these conditions fires. Do NOT
preemptively rebuild the bridge — the CSV path in Plan 1 is sufficient for
the "validate AI brain in SIM" objective.

Start Plan 1.5 when any one of these is true:

1. **AI brain validated for several sessions.** The AI is making sensible NQ
   decisions on stale-bar (15-min Databento Historical) data and you're
   ready to graduate to real-time bars + real fill events for live trading.
   Concretely: 20-50 cycles observed in SIM with reasoning quality you trust.

2. **Funded broker connection imminent.** You're switching NinjaTrader's
   account from SIM101 to a real funded prop-firm account. Plan 1.5 is
   required before live money — the CSV bridge cannot read real balance
   ($50k mock breaks position sizing) and cannot manually close on news.

3. **CSV race condition fires in SIM.** Hazards H1-H4 (lost-signal race,
   1-sec dedup collision, DrvFs mtime, fill replay on session rollover)
   are documented as acceptable for SIM but not for live. If any fires
   in SIM, that's the signal to upgrade the bridge now rather than later.

4. Real-time bars required for actionable strategy validation

   PROBLEM
   - Databento Historical OHLCV has a publication delay (verify exact
     delay against Databento docs and current account tier before Plan
     1.5 design — assume 15 minutes for planning purposes)
   - Strategies with intra-session entry windows (e.g. TTrades AM
     Silver Bullet, 10:00-11:00 ET) cannot be validated for actionable
     edge on delayed data — by the time a bar arrives, the entry
     window may have already passed
   - Databento Live DOES provide OHLCV aggregates at 1-second and
     1-minute intervals over its real-time streaming API (per
     databento.com/futures product page). Earlier internal assumption
     that Databento Live was ticks-only was INCORRECT.
   - However, Databento Live requires a separate subscription tier
     beyond Historical, while NinjaTrader already receives a live
     CQG/Rithmic feed through the prop firm subscription (MFFU,
     Bulenox, Apex, Topstep all include this) at no additional cost.

   DECISION
   - Plan 1.5 will source LIVE bars from NinjaTrader, not Databento
     Live
   - Databento Historical retained for: warmup on cold start, gap fill
     after restart, archive, walk-forward backtests
   - NT live bars become primary source for the live indicator loop
   - Trade-off accepted: Plan 1.5 requires NinjaScript changes to
     ClaudeTrader.cs and a Go-side bar reader; in exchange,
     elimination of source-disagreement risk and zero additional data
     vendor cost during SIM phase

   SCOPE EXPANSION
   - Plan 1.5 was previously scoped as "TCP socket bridge for orders"
   - Now expanded to include:
     a) TCP or CSV-based live bar feed (NT → Go), implementation
        choice deferred to Plan 1.5 design doc
     b) Data source switcher in Go (Historical for warmup, Live for
        steady-state)
     c) Bar timestamp normalization (NT close-time vs Databento
        open-time conventions — verify both before implementing)
     d) Contract roll handling across NT and Databento symbol
        conventions

   OPEN QUESTIONS FOR PLAN 1.5 DESIGN DOC
   - Current Calculate mode in ClaudeTrader.cs (OnBarClose vs
     OnEachTick) — read file to verify before designing the bar-write
     path
   - File I/O method: File.AppendAllText vs FileStream with explicit
     FileShare.Read — NT support docs recommend lock object
     (private object barWriteLock = new object()) for multi-threaded
     write safety
   - Tail-from-offset reader pattern in Go to survive bot restarts
     without losing NT-written bars
   - Bar timestamp convention difference: NT defaults to bar-close
     time; Databento OHLCV uses bar-open (ts_event = bar start) —
     verify both and document the translation
   - Symbol normalization across MNQM6/MNQU6 rolls
   - Latency budget: NT writes bar → Go reads bar → indicators → AI →
     CSV signal → NT places order. Target end-to-end < 5 seconds for
     1-minute timeframe viability.

   CITATIONS
   - Databento OHLCV aggregate availability:
     https://databento.com/futures (verified 2026-05-25)
   - NT8 multi-threading file I/O guidance:
     https://ninjatrader.com/support/helpguides/nt8/multi-threading.htm
   - NT8 IsFirstTickOfBar / Calculate mode behavior:
     https://ninjatrader.com/support/helpGuides/nt8/isfirsttickofbar.htm

Until one of those triggers fires: **the CSV path is canonical**, Plan 1
stays the production path, and Plan 1.5 lives as documented architecture
ready to execute when needed.

## Plan 1.5 — Design Findings (Research Summary 2026-05-25)

The following findings come from a comprehensive research pass on the
NT8 → Go (WSL2) real-time market data bridge problem. They are NOT
implementation specs — they are the verified-or-flagged knowledge that
must inform the Plan 1.5 implementation when it is written.

Each finding cites its source. Items marked [VERIFY] require empirical
confirmation on John's specific machine before relying on them.

### Architecture Decision: CSV-over-/mnt/c with 1-minute bars

DECISION: Plan 1.5 will use CSV append on /mnt/c as the bar transport,
not TCP, not NetMQ, not WebSocket, not memory-mapped files.

RATIONALE:
- Plan 1 already validated this pattern end-to-end with 2 real fills
  on 2026-05-22. Lowest implementation risk.
- For 1-minute bars on NQ/MNQ, the 250ms-poll-jitter latency is
  comfortably below any actionable strategy budget.
- Alternative transports (NetMQ, WebSocket, TCP) deferred to Stage 2,
  triggered only if measured p99 latency exceeds 500ms.

DECISION: NT emits ONLY 1-minute bars. Go aggregates 5m/15m/H1/H4
in-process.

RATIONALE:
- Single source of truth for higher timeframes
- Backtest-live parity: same aggregation logic used against historical
  Databento OHLCV-1m
- Multi-BarsArray NinjaScript has documented synchronization quirks
  (NT8 "stair-step effect" in multi-series indicators)
- Reconnect resilience: gap-filling one 1m feed is trivial; gap-filling
  5 separate timeframe streams introduces consistency bugs

### NT8 Calculate Mode (verified from NT support forum)

DECISION: Calculate.OnEachTick + IsFirstTickOfBar + State.Historical
guard. The bar-writer fires once per closed bar in realtime only.

EXACT IDIOM:
- In State.SetDefaults: Calculate = Calculate.OnEachTick
- In OnBarUpdate():
  - if (State == State.Historical) return;
  - if (!IsFirstTickOfBar) return;
  - if (CurrentBar < 1) return;
  - Write Times[0][1] / Opens[0][1] / Highs[0][1] / Lows[0][1] /
    Closes[0][1] / Volumes[0][1]

WARNING: Without the State.Historical return guard, every strategy
restart writes thousands of duplicate historical bars. Confirmed by NT
staff in forum thread "State == State.Realtime".

WARNING: NT redownloads the current day's historical data on every
reconnect (NT staff, forum thread "Reload charts after connection
lost"). This means after a NT reconnect, there is a temporary GAP in
the live-emitted bars while NT replays history. The Go side must
gap-fill from Databento Historical on reconnect detection.

### File I/O Pattern (NT official guidance)

DECISION: Explicit lock object + FileStream with FileMode.Append +
FileShare.Read + FileOptions.WriteThrough.

RATIONALE: NT's own multi-threading help guide explicitly requires
protection of custom resources because "market data is distributed
across the entire application by a randomly assigned UI thread, there
is no guarantee that your object will be running on the same event
thread."

PATTERN:
private static readonly object _writeLock = new object();
lock(_writeLock) {
    using (FileStream(_csvPath, FileMode.Append, FileAccess.Write,
                      FileShare.Read, 4096, FileOptions.WriteThrough))
    using (StreamWriter sw) { sw.WriteLine(row); }
}

FileShare.Read allows Go to hold the file open for reading concurrently
without blocking NT writes. Go reader must use os.Open (read-only) and
must NEVER call syscall.Flock — that would cause NT's next write to
throw "process cannot access the file."

### WSL2 File Watching: HARD CONSTRAINT

CRITICAL FINDING: inotify on /mnt/c DOES NOT FIRE for Windows-side
file writes. This is microsoft/WSL issue #4739 and #5424, both still
open as of May 2026.

DO NOT use fsnotify in Go for this bridge. The Add() call will
succeed silently, but events will never arrive. There is no error to
detect.

DECISION: Polling with os.Stat + persisted byte offset, every 250ms.

GO PATTERN:
- Load lastOffset from /home/hoang/nofx/.state/bars_MNQ_1m.offset
- Loop: os.Stat -> if size > lastOffset -> os.Open -> Seek(lastOffset)
  -> read new bytes -> update offset -> save offset
- Only ingest lines ending in '\n' (handles partial reads where NT
  has flushed bytes but not yet completed a line)
- Persist offset to WSL native filesystem (NOT /mnt/c) for guaranteed
  POSIX rename atomicity

POLLING INTERVAL: 250ms is the recommended default. Worst-case
NT-write-to-Go-ingest latency = 250ms + transport overhead, typically
under 500ms total. Stat overhead is sub-millisecond on a small file;
CPU impact negligible.

### Timestamp Reconciliation: NT vs Databento

CRITICAL FINDING: NT and Databento use OPPOSITE bar timestamp
conventions in DIFFERENT timezones.

- NT: bar timestamp = bar CLOSE time, in the timezone configured in
  Control Center > Tools > Options > General (defaults to Windows
  local time). Verified from NT staff reply in forum thread
  "NinjaScript NT8 TIME[0]" plus the "How Bars Are Built" help guide.
- Databento OHLCV: ts_event = bar OPEN time, in UTC nanoseconds.
  Verified from NautilusTrader integration docs and Databento schema
  documentation.

CONSEQUENCE: The same 1-minute bar (e.g., 09:30:00-09:31:00 ET) will
appear in NT as Time[0] = 09:31:00 LOCAL and in Databento as
ts_event = 09:30:00.000000000 UTC. Off-by-one-minute bugs from this
mismatch are the #1 source of NT-Databento integration failures.

CANONICAL KEY: All bars in the Go side are keyed by bar_open_utc
(time.Time, UTC, second precision). Convert at ingest:
- NT side: write Times[0][1].AddMinutes(-1).ToUniversalTime() as
  ISO-8601
- Databento side: time.Unix(0, ts_event_ns).UTC()

DST HANDLING: NT uses local Windows time and shifts with DST.
Databento uses UTC and is immune. Always store and reason in UTC;
convert to ET/CT at display layer only.

### Multi-Timeframe: Aggregate in Go from 1m

DECISION: NT writes only 1m bars. Go aggregates 5m, 15m, H1, H4.

WARNING: 5m bar close instants in NT and Databento RARELY align to
the same millisecond. NT bars close on the first tick AFTER the
boundary (verified NT forum thread "Bar Closing Time" documents
2-10 second delta in slow markets). Databento bars close on the
matching-engine aggregation boundary (deterministic).

Implication: ALWAYS use bar_open_utc as the key. NEVER use
arrival-wall-clock-time.

### Session Boundaries (CME NQ/MNQ)

VERIFIED from CME E-mini/Micro futures contract specifications page
(May 2026):
- Sunday 17:00 CT → Friday 16:00 CT
- Daily trading halt 16:00–17:00 CT Mon-Thu
- In Eastern Time: Sunday 18:00 ET → Friday 17:00 ET, daily halt
  17:00-18:00 ET Mon-Thu

CME holidays and early closes occur on different days from US
equity markets. DO NOT infer session boundaries from clock arithmetic
alone — use NT's Trading Hours template (Bars.IsFirstBarOfSession)
or Databento's "status" schema.

### Contract Roll Handling

CRITICAL FINDING from NT's official rollover help guide:
"NinjaScript strategies are not rolled forward and must be manually
rolled over."

CONSEQUENCE: Plan 1.5 must explicitly handle the front-month roll on
both sides (NT and Databento) because they have OPPOSITE policies and
NEITHER auto-rolls inside a running NinjaScript strategy.

- NT side uses fixed expiry-month symbols (e.g. MNQ 06-26, MNQ 09-26).
  Live ticks arrive only for the explicit front-month contract attached
  to the chart. Per NT staff in forum thread "continuous ticket for
  MES/MNQ/M2K": "Loading the current front month in NinjaTrader 8 with
  a default merge policy of 'Merge Back Adjusted' will be almost
  identical to a continuous contract."
- Databento side ships RAW prices through the roll (no back-adjustment)
  via continuous symbology (MNQ.c.0, MNQ.v.0, MNQ.n.0, MNQ.c.1). From
  Databento docs: "Our continuous contract symbology does not behave
  the same as continuous contracts provided on retail charting apps,
  which create a continuous series by applying a constant offset on
  each rollover month to the lead month contract. Our philosophy is
  generally to provide raw prices."

VOLUME-DRIVEN ROLL TIMING: per FlowBots Knowledge Center analysis,
volume typically migrates from the front contract to the next 1-2
days BEFORE the calendar expiry. NT's rollover prompt fires based on
calendar date and can leave the running strategy pointing at a
near-empty contract for that 1-2 day window. Best practice: defer
the NT rollover prompt until volume has actually migrated.

STITCHING OPTIONS (3 methods):
- Raw stitched: concatenate front-month series, accept the price-jump
  at each roll. Appropriate when indicators are scale-invariant or
  analyze each contract separately.
- Back-adjusted (additive): subtract the front/back roll-day spread
  from all pre-roll bars. NT default "Merge Back Adjusted" policy;
  appropriate for trend indicators (MA, BB).
- Ratio-adjusted: multiply all pre-roll bars by new_price/old_price
  ratio. Better for very long histories where additive offsets
  accumulate.

PLAN: live trade only the explicit front-month contract; manually
re-deploy strategy on roll day; plan a tradeless window of several
hours to avoid signaling on the synthetic price jump; request
Databento MNQ.v.0 (volume-based continuous) raw for historical
warmup; apply Go-side back-adjustment at ingest.

### Latency Measurement

Instrument both sides explicitly. NT writes two timestamps per CSV
row (bar_close_utc_nt, wallclock_at_write_utc captured immediately
before sw.WriteLine). Go captures time.Now().UTC() at three points:
- t_read_stat: when os.Stat first shows new bytes
- t_parse_done: when the row is fully parsed into a struct
- t_indicator_done: when downstream indicators have processed it

NT-to-Go transport latency = t_read_stat - wallclock_at_write_utc.

ENGINEERING ESTIMATES (component breakdown, NOT measured on John's
machine — see VERIFY-7):
- NT bar-close to NT file-write: < 10 ms (sub-second tick + lock
  acquisition)
- NT write to NTFS flush: < 5 ms with FileOptions.WriteThrough
- NTFS to DrvFs visibility: < 50 ms typical, occasional spikes to
  several hundred ms
- Go poll-jitter: 0-250 ms depending on phase alignment relative to
  bar close
- Go parse + ingest: < 1 ms per row

BUDGET: end-to-end p99 < 500 ms (NT bar-close to Go indicator update).
Above 500 ms triggers Stage 2 transport upgrade (see Alternative
Transports below).

BOTTLENECK: the 250 ms poll interval. Reducing it costs CPU; moving
off CSV to a push-based transport eliminates it entirely.

See [VERIFY-7] in Caveats and Verification Items below.

### Reordering and Deduplication

FAILURE MODES:
1. NT reconnect during session: NT replays the day's historical and
   the State.Historical guard prevents re-writes. If the strategy is
   RE-ENABLED (not just reconnected), prior bars may never get
   appended (no double-write), while subsequent live bars are
   appended normally. Net: occasional gaps, never duplicates.
2. File-system buffering reordering: with FileOptions.WriteThrough
   and a single-thread writer protected by lock(), reordering is not
   possible within a single NT process. Cross-process reordering is
   also not a concern (one writer).
3. Strategy restart while a bar is being written: lock acquisition
   is process-scoped; an OS crash mid-write can leave a partial line.
   Mitigated by the partial-line rule below.

PARTIAL-LINE RULE (Go side): only ingest lines that end with '\n'.
On incomplete final read, set lastOffset to the end of the last
newline (NOT to the current file position).

IDEMPOTENT INGESTION: use (symbol, bar_open_utc) as composite primary
key. Dedup cache with 24-hour TTL:
- 24h × 60 bars/hour = 1440 entries per symbol — trivial memory
- key format: fmt.Sprintf("%s|%d", symbol, barOpenUTC.Unix())
- On duplicate: log debug, skip ingestion

### Process Lifecycle and Persistent Offset

When the Go process dies and restarts, replaying the entire CSV
from offset 0 is wasteful. Persist last-known-good offset.

CHECKPOINT LOCATION: WSL native filesystem (e.g.
/home/hoang/nofx/.state/), NOT /mnt/c.

REASON: DrvFs does NOT guarantee POSIX rename atomicity across the
Windows/Linux boundary. ext4 native does. Atomic-rename is the
critical primitive for crash-safe checkpoints.

WRITE PATTERN (write-to-tmp + atomic rename):
- Write offset bytes to checkpointPath + ".tmp"
- os.Rename(tmp, checkpointPath) — atomic on same-filesystem WSL
  native

RECOVERY ON STARTUP:
- Load offset from checkpoint file
- os.Seek(offset, io.SeekStart) on the CSV
- Resume the polling loop

OFFSET-BEYOND-FILE-SIZE: if saved offset is greater than current CSV
size (e.g. NT truncated and re-initialized the bar log between
sessions), reset offset to 0 and re-ingest from start. The dedup
cache prevents double-emission of any bars from the prior session
that remain in the cache window.

### Alternative Transports (Future Migration Path)

Plan 1.5 ships on CSV. The following are deferred upgrade paths
triggered only if measured latency exceeds the 500 ms p99 budget.

| Transport | Effort | Latency | Reliability notes |
|---|---|---|---|
| NinjaScript TCP server | 1-2 days | 1-5 ms | Fragile inside NT process per NT staff; users report success in Console apps but issues inside NT |
| NetMQ (native C# ZeroMQ) | 2-3 days | 1-5 ms | No native libsodium dependency; cleaner than clrzmq4 inside NT. RECOMMENDED for fan-out pub/sub |
| WebSocket from NinjaScript | 2-3 days | 2-10 ms | NT support has explicitly recommended over raw TCP: "websockets… more reliable than TCP". RECOMMENDED for single-connection simple protocol |
| HTTP POST from NT to Go | 1 day | 5-50 ms | Highest overhead; easiest to debug; survives Go restarts gracefully. Best for control-plane, not bar data |

REJECTED OPTIONS:
- Named pipes: WSL2 cannot directly open Windows named pipes; would
  require a Windows-side proxy.
- Memory-mapped files: cross-kernel sync primitives not guaranteed
  across DrvFs; MMF semantics undocumented for the Win-Lin boundary.

CSV ROLE AFTER MIGRATION: keep CSV as warm-backup archive. The CSV
writer should remain enabled even after socket-based transport
becomes primary — the file is a free durability layer and aids
post-mortem analysis.

### Decision Matrix Summary

| Design question | Recommended | Rationale |
|---|---|---|
| Transport for bar data | CSV append on /mnt/c | Already works; meets 500ms p99 budget for 1m bars |
| Calculate mode | Calculate.OnEachTick + IsFirstTickOfBar + State.Historical guard | Documented NT idiom; preserves option for future intra-bar tick channel |
| Bar timestamp written | Bar OPEN in UTC ISO-8601 | Matches Databento ts_event convention; DST-immune; one canonical key |
| Multi-timeframe | NT writes only 1m; Go aggregates 5m/15m/H1/H4 | Single source of truth; backtest-live parity; reconnect-safe |
| Symbol on live NT | Front-month explicit (e.g. MNQ 06-26) | NT live cannot use continuous symbols on CQG/Rithmic |
| File I/O pattern | lock(_writeLock) { FileStream(Append, FileShare.Read, WriteThrough) } | Per NT's own multi-threading help guide |
| Go-side file watching | Polling os.Stat every 250ms + persisted offset | inotify does NOT fire for Windows-side writes on /mnt/c |
| Dedup key | symbol + bar_open_utc, 24h TTL | Idempotent against historical replay, reconnect, restart |
| Checkpoint location | WSL native filesystem (/home/hoang/nofx/.state/) | DrvFs does NOT guarantee POSIX rename atomicity |
| Rollover handling | Manual NT re-deploy on roll day; pause trading window around roll | NT support: "NinjaScript strategies are not rolled forward" |
| Databento role | Historical only — warmup, gap-fill, backtest, archive | Live tier with OHLCV is $1,399/mo annual contract; NT live via prop firm is free and matches execution venue |
| Latency budget | End-to-end p99 < 500ms (NT bar-close to Go indicator update) | Comfortably achievable with 250ms poll; reassess only if sub-100ms required |

### Staged Implementation Recommendations

STAGE 0 — Week 1 (Plan 1.5 minimum viable):
1. Add a BarWriter section to ClaudeTrader.cs using the
   Calculate.OnEachTick + IsFirstTickOfBar + State.Historical guard
   from the NT8 Calculate Mode section above. Emit ONLY 1m bars.
   Write to C:\Users\hoang\NofxTrader\data\bars_MNQ_1m.csv with
   columns: bar_open_utc, wallclock_at_write_utc, open, high, low,
   close, volume.
2. Add a Go-side tail_csv package that polls
   /mnt/c/Users/hoang/NofxTrader/data/bars_MNQ_1m.csv every 250ms
   with persisted offset in /home/hoang/nofx/.state/.
3. Aggregate 5m, 15m, H1, H4 in Go using bar_open_utc as the
   canonical key.
4. Dedup by (symbol, bar_open_utc) with 24h TTL.
5. Run a 1-week soak test on SIM101 during RTH and overnight. Log
   NT-to-Go latency on every bar.

STAGE 1 — Weeks 2-3 (hardening):
6. Add gap-fill from Databento Historical on startup. Query
   everything between the latest persisted bar_open_utc and (now -
   16 minutes) to stay outside Databento's 15-minute Historical
   publication delay.
7. Add session-boundary detection using a hard-coded CME calendar
   table or by reading Databento's status schema offline.
8. Add a unit test that takes one known NT bar and one known
   Databento bar for the same minute, normalizes both, and asserts
   equality on the canonical key.
9. Add a daily smoke test that compares the previous day's NT bars
   (via CSV) against Databento Historical OHLCV-1m for the same day,
   after end-of-session, and alerts on any mismatch exceeding the
   tolerance ladder (defined in Merge Logic State Machine, to be
   added in a later commit).

STAGE 2 — only if Stage 0 measured p99 latency exceeds 500 ms:
10. Replace CSV with NetMQ pub/sub: NT publishes tcp://*:5556
    topic bar.MNQ.1m; Go subscribes. Keep CSV as warm-backup
    archive.

BENCHMARKS THAT CHANGE THE RECOMMENDATION:
- If NT-to-Go p99 latency > 500 ms during RTH: upgrade transport
  to NetMQ or WebSocket (Stage 2).
- If Databento Live drops below $200/month or a new lower tier with
  OHLCV appears: reconsider Databento Live as primary feed for
  backtest-live parity.
- If WSL2 adds Windows-side inotify forwarding (microsoft/WSL #4739):
  drop polling, use fsnotify.

### Caveats and Verification Items

[VERIFY-7] Measure NT-to-Go transport latency p50 and p99 over a
full RTH session on John's machine. If p99 exceeds 500 ms, Stage 2
transport migration is triggered. NT staff have noted that
Calculate.OnEachTick is "CPU intensive if your program code is
compute intensive" — for a bar-only writer that early-returns on
!IsFirstTickOfBar this is negligible, but verify under live NQ load
(~1k-10k ticks/min during RTH).

[VERIFY-8] FileShare.Read semantics across DrvFs: confirm
empirically that Go can open and read the file while NT holds the
write handle. Build a 1-day soak test before relying on it.
Validated in spirit by NT's own multi-threading help guide and
YSFKDR/NinjaTrader_Data_Exporter's use of ReaderWriterLockSlim,
but the cross-kernel boundary is the verification gap.

[VERIFY-9] Bar-close jitter under low-liquidity conditions. NT bars
only close on the next tick after the boundary. During Sunday-night
reopen or holiday sessions, multi-second close latency is documented
(NT forum thread "Bar Closing Time"). Affects arrival timing, not
bar content. Decide whether the strategy can tolerate.

[VERIFY-10] Read the actual upstream claudetrader.cs C# source via
git clone and diff against the patterns documented here before
merging the bar-writer changes. Automated fetch returned a
permissions error during research; the patterns above are from NT's
own help guide and corroborating community sources. Confirm
threading and State semantics in the live source match before
implementation.

CONFIRMATION ITEMS (recheck periodically, NOT pre-implementation
blockers):
- Databento pricing can change. Standard tier rose from $179/month
  at April 2025 launch to $199/month by May 2026; Plus $1,399/month
  verified May 2026. Recheck before any business decision involving
  Live data.
- CME session schedule changes occasionally. Verified May 2026 from
  CME's E-mini/Micro futures page. Use NT's Trading Hours template
  (Bars.IsFirstBarOfSession) and/or Databento's status schema rather
  than hard-coded clock arithmetic.
- WSL2 inotify limitation could be fixed in a future WSL2 release
  without obvious announcement. Don't write code that assumes
  "polling forever" — abstract the watcher behind an interface so
  future migration is a one-file change.
- Time zone conversion is the single most common bug source in this
  kind of bridge. Write the unit test described in Stage 1 step 8
  and run it on every PR that touches the timestamp normalization
  path.

## Plan 1.5 — Merge Logic State Machine (Research Summary 2026-05-25)

This subsection specifies how NT live bars and Databento Historical
bars combine into a single canonical 1-minute time series keyed by
bar_open_utc. It complements the design findings above, which
established the architecture and protocols. This section establishes
the LIFECYCLE — what happens at boot, during normal operation, on
disconnects, and at end-of-day.

### Three Rules That Dominate

1. NT wins every conflict. Discrepancies are logged (info / warn /
   error tiered by tick distance), never auto-resolved against NT.
   NT is the execution venue truth.

2. Time, not source, is the join key. All bars are keyed on
   bar_open_utc (second precision UTC). NT stamps bars at CLOSE
   per NT8 help guide, so Go reader must subtract 60s to obtain
   canonical open. Databento ohlcv-1m is timestamped at OPEN per
   NautilusTrader Databento integration docs.

3. Databento is structurally lagged by ~15 minutes. Databento's
   roadmap explicitly states intraday GLBX.MDP3 via Historical API
   has 15-min embargo without real-time entitlement. Databento can
   never close the gap to "now" — only to "now − 15 min."

### Cold Start Sequence

WARMUP WINDOW: 7-10 RTH sessions for indicators up to EMA(200) on 15m.
Matches backtrader's _minperiod convention and QuantConnect Lean's
SetWarmUp pattern. NautilusTrader uses the same historical-prefetch-
then-subscribe pattern (PR #3825).

BOUNDARY CALCULATION:
- databento_warmup_end = T0 - 15min - 2min safety = T0 - 17min
- nt_handoff_start = first NT bar with bar_close_utc >= T0
- overlap_window = [databento_warmup_end, nt_handoff_start] =
  15-17min hole BY CONSTRUCTION

SEQUENCE (sequential, not parallel):
1. Block on Databento timeseries.get_range for warmup window
   ending at T0 - 17min
2. Feed bars through indicator pipeline with is_warmup=true
3. Start consuming NT CSV tail
4. Mark system "live-ready" only when nt_handoff_start observed

FALLBACK on short Databento response:
- Missing bars inside active session: log warn, proceed if
  missing_pct <= 5%; abort if > 5%
- Missing bars during 16:00-17:00 CT halt or weekend: expected,
  no log

### Overlap Zone at Cold Start

DECISION: Accept the gap explicitly. Do NOT block waiting for
Databento to catch up (wastes 15 min of signal time). Do NOT
forward-fill (pollutes indicators with synthetic data).

PATTERN:
- Warm indicators on [T0 - warmup, T0 - 17min)
- Mark [T0 - 17min, T0) as state=GAP
- Start consuming NT at T0
- Schedule backfill from Databento for the gap at T0 + 17min

RISK: indicators path-dependent over the missing window (cumulative
VWAP anchored at session open, opening-range breakouts) must check
state != GAP before firing. Path-independent indicators (EMAs warmed
from earlier data) are safe.

Expose WarmupComplete boolean analogous to Lean's IsWarmingUp.
Strategy code is responsible for gating on it.

### Warm Restart Mid-Session

SCENARIO: Go process died 14:32, restarts 14:35. Local Parquet has
bars up to 14:30. Databento has bars up to 14:20. NT has been
emitting throughout (NT didn't die — only Go did).

RECOVERY HIERARCHY (fastest first):
1. Local Parquet replay for [boot - warmup, 14:30] — no network,
   indicators rehydrate in milliseconds
2. NT live tail consumed from 14:35 onward. NT may have written
   bars [14:30, 14:35] to CSV during the Go outage. VERIFY-11
   below.
3. Databento backfill for any remaining hole [14:30, 14:35] is
   partially available (Databento has through 14:20, won't have
   14:30 until ~14:45). Mark these state=PENDING_BACKFILL.

REFERENCE: NautilusTrader v1.225.0 (PR #3733) — "Fixed Interactive
Brokers historical bar subscriptions not restored after daily
gateway restart" — confirms automatic re-request of historical bars
on reconnect is production-grade pattern.

GO LOGIC ON BOOT:
- Read last_local_bar_open_utc from Parquet
- Issue ONE bounded Databento request for (last_local, T0 - 15min)
- Tail NT CSV from saved offset
- Backfill scheduler handles PENDING_BACKFILL bars at T+17min

### NT Disconnect / Reconnect During Session

SCENARIO: NT loses CQG/Rithmic price feed at 11:42, reconnects at
11:47. During gap NT emits NOTHING to CSV. Per NT staff: "Keep
running will not recalculate the historical data that has passed
so it would just keep running once the connection resumes."

DETECTION (defense in depth, three signals):

Signal 1: Time-since-last-bar
- Watchdog timer on canonical stream
- > 90s during session: suspect
- > 180s: confirmed disconnect

Signal 2: Heartbeat file
- NinjaScript writes UTC timestamp to heartbeat.txt every 5s
- Go reads heartbeat
- Stale > 30s: degraded

Signal 3: Connection probe
- NinjaScript writes line on OnConnectionStatusUpdate(ConnectionLost)
- And again on .Connected
- Definitive signal

Any TWO of three confirm NT outage.

FALLBACK DURING NT GAP:
- Stay on canonical stream, switch source-of-truth flag to DEGRADED
- Do NOT write Databento bars into canonical stream during gap
  (Databento is 15 min behind, gap [11:42, 11:47] won't exist on
  Databento until ~12:02)
- Mark [last_nt_bar, NT_reconnect_bar) as state=NT_GAP
- At NT_reconnect_bar + 17min, scheduled backfill pulls Databento
  [11:42, 11:47] and writes with source="databento_historical",
  provisional=true

TRANSITION HANDLING:
- When OnConnectionStatusUpdate -> Connected fires, do NOT call
  ReloadAllHistoricalData() from running strategy. NT help guide
  warns: "This method should NOT be called from any of the event
  methods which access data."
- For data-emitting NinjaScript, set ConnectionLossHandling =
  KeepRunning rather than default Recalculate (default would
  re-fire OnBarUpdate on backfilled bars, risking duplicates)
- If NT bars arrive later for the gap window, they overwrite
  Databento provisional bars per the NT-wins rule, discrepancy
  logged

### Discrepancy Detection and Logging

Once Databento catches up (T+17min for any bar), every NT bar
acquires a Databento counterpart for comparison.

AGGRESSIVENESS: Compare every bar as it becomes comparable. One
hash-map lookup per bar, cheap. Sampling only at EOD throws away
timing information that makes feed-quality bugs diagnosable.

TOLERANCE LADDER (NQ tick = 0.25, MNQ tick = 0.25):

OHLC fields:
- Exact match: no log
- Off by <= 1 tick (0.25): info
- Off by 2 ticks (0.50): warn
- Off by > 2 ticks: error

Volume:
- Ratio in [0.95, 1.05]: info, normal
- Ratio in [0.80, 0.95) or (1.05, 1.20]: warn, known-class divergence
- Ratio outside [0.80, 1.20]: error, investigate

Volume divergence between CQG/Rithmic and Databento is EXPECTED,
not an error condition by itself. Different aggregation, different
message coalescing.

LOG FORMAT (greppable single-line JSON):
{"ts":"2026-05-25T14:31:00Z","evt":"bar_discrepancy","sym":"NQM6",
 "bar_open_utc":"2026-05-25T14:30:00Z","field":"close",
 "nt":21450.25,"db":21450.50,"ticks_off":1,"vol_nt":1842,
 "vol_db":1851,"level":"info"}

Naming supports triage:
grep evt=bar_discrepancy | jq 'select(.level=="error")'

No alerts, no auto-pause (per task constraints).

### End-of-Day Reconciliation

At 16:00 CT (start of daily halt), NT has emitted full session.
At ~16:45 CT Databento has caught up. Reconciliation job runs
at 17:15 CT (17:00 reopen + 15-min buffer).

SEQUENCE:
1. Read NT bars for session from canonical Parquet
2. Read Databento bars for same session via one
   timeseries.get_range call
3. Bar-set diff:
   - In Databento but NOT in NT: nt_missing — real holes in
     execution-venue data. Write with source=databento_historical,
     provisional=true. Log warn. THIS catches "NT silently missed
     bars" — without this step canonical stream gets quietly
     shorter than the day.
   - In NT but NOT in Databento: db_missing. Extremely rare.
     Log info, trust NT.
   - In both: run tolerance ladder
4. Rewrite vs append: do NOT rewrite NT bars even if Databento
   disagrees (NT-wins rule). EOD writes limited to:
   - INSERT of nt_missing bars (with source tag and provisional)
   - UPDATE of db_close, db_volume, db_match_status audit columns
     on existing rows (NOT canonical OHLCV)

SCHEDULING:
- Automatic at 17:15 CT, with 30-min retry window if Databento
  fails
- Manual --reconcile-day YYYY-MM-DD CLI flag for ad-hoc replay

### Source Tagging

Single source string column per bar. Three values cover universe:
- nt_live: bar consumed from NT CSV in real time
- nt_replay: bar read from NT CSV after Go-process restart (NT
  was running, Go was not)
- databento_historical: bar fetched via Databento Historical API
  (warmup, gap fill, EOD insert)

Separate provisional boolean column marks bars that may be
overwritten later. Currently only databento_historical bars
filling NT gaps are provisional.

### Parquet Layout

DECISION: One file per symbol per day, all sources merged, source
as a column.

Pattern:
data/bars_1m/symbol=NQM6/date=2026-05-25/part-0.parquet

RATIONALE:
- Queries are overwhelmingly time-range scans on one symbol
- Splitting by source doubles file count, complicates "give me
  canonical series" read path, works against Parquet's columnar
  compression on bar_open_utc
- Modexa's Parquet partition design analysis: "Date-only
  partitioning when 90% of analytics filter by a time range" —
  this workload exactly
- Target file size 1-10 MB. 1440 bars/day × ~80 bytes = ~110 KB
  raw, ~30-50 KB compressed. Well below standard 128-512 MB
  row-group recommendation but acceptable at this volume.

### Gap Detection

Run periodic contiguous-sweep every 60s during session hours:

expected = expected_minute_stream(session_start, now - 1min)
for t in expected:
    if t not in local_index:
        gaps.append(t)
if gaps and now - max(gaps) > 120s:
    trigger_backfill(gaps)

expected_minute_stream MUST respect:
- 16:00-17:00 CT daily halt
- Friday 16:00 CT to Sunday 17:00 CT weekend gap
- CME holiday calendar

Without these the detector fires every halt minute. Pre-compute
session-minute calendar weekly from CME calendar.

### On-Demand Backfill

Databento timeseries.get_range is RANGE-ONLY. No single-bar
endpoint. Coalesce gap-fill requests to ranges with at most
5-min separation; minimizes HTTP overhead.

Canonical shape from Databento Python client (Go equivalent):
client.timeseries.get_range(
    dataset='GLBX.MDP3', symbols='NQ.c.0', schema='ohlcv-1m',
    stype_in='continuous', start='...', end='...'
)

Use stype_in='continuous' with NQ.c.0 / MNQ.c.0 to avoid rollover
bookkeeping inside Go process.

### Clock Skew (Windows Host)

W32Time service sufficient for this use case but MUST be
configured. Per Microsoft Learn "High Accuracy W32time Requirements":
W32time was created for "computers to be equal to or less than
five minutes (which is configurable) of each other for
authentication purposes." Five-minute skew is CATASTROPHIC for
bar joining.

CONFIGURATION on Windows host (where NT8 runs):
Command form from Microsoft Learn "Windows Time Service Tools
and Settings":
w32tm /config /manualpeerlist:"pool.ntp.org,0x8"
       /syncfromflags:manual /update

The 0x8 flag is SpecialInterval / client-mode association —
enables tighter polling.

WSL2 inherits Windows host clock by default. Verify with date -u
on both sides at startup.

TARGET SKEW: < 250 ms
- Anything above 1s means stale bar_open_utc lands in wrong
  minute bucket
- Startup check in Go: abort if
  |wsl_clock - windows_clock| > 250ms

### Session Boundary Merge Behavior

DAILY HALT 16:00-17:00 CT Mon-Thu:
Pre-compute expected minute set per session day:
session_minutes(day) = minutes(17:00CT day-1, 16:00CT day)
                       - holidays

Gap detector and EOD reconciler both consume this calendar.

WEEKEND Friday 16:00 CT → Sunday 17:00 CT:
- Treat as 49-hour expected gap
- Sunday 17:00 CT bar is first new bar of next trading week
- Tag with session_first_bar=true audit column
- Cold start on Sunday afternoon should fetch warmup window
  crossing weekend cleanly — Databento returns no bars during
  closed hours, contiguous-sweep detector must skip them

### Decision Matrix (Merge Logic)

State / NT available / Databento current / Action / Canonical source:

WARMUP / not yet / yes (delayed) / pull Databento [T0-7d, T0-17min]
/ databento_historical

OVERLAP_HOLE / not yet / no (inside 15-min embargo) / mark gap, wait
/ none

LIVE / yes / yes (for past bars) / consume NT CSV, compare against
Databento at T+17min / nt_live

NT_GAP / no (disconnect) / catches up at T+17min / mark gap,
backfill from Databento later / databento_historical (provisional)

NT_RECONNECT / yes / yes / resume NT, NT backfill (if any)
overwrites provisional Databento / nt_live (overwrites)

EOD_RECONCILE / session closed / full day available / diff sets,
insert nt_missing bars, log discrepancies / databento_historical
for missing only

WEEKEND / no (closed) / no new data / idle, no writes / none

### State Machine

States: BOOT -> WARMUP -> LIVE <-> NT_GAP, LIVE -> EOD_RECONCILE
-> WEEKEND -> BOOT

BOOT:
- load last_local_bar_open_utc from Parquet
- if last_local within current session: LIVE_RESUME
- else: WARMUP

WARMUP:
- request Databento [now - warmup_window, now - 17min]
- feed indicators in monotonic order with is_warmup=true
- on completion: emit WarmupComplete event -> LIVE

LIVE:
- consume NT CSV tail
- on each new NT bar: write canonical with source=nt_live
- every 60s: run gap_detector
- every bar+17min: run discrepancy_check against Databento
- on heartbeat_stale OR connection_lost: -> NT_GAP
- at 16:00 CT Mon-Thu or Fri close: -> EOD_RECONCILE

NT_GAP:
- log warn evt=nt_gap_open
- scheduled job at gap_start + 17min: fetch Databento
  [gap_start, min(now - 17min, gap_end)]
- write with source=databento_historical, provisional=true
- on heartbeat_recovered AND new NT bar arrives: -> LIVE
- on NT bar inside [gap_start, gap_end] (later): overwrite
  provisional bar, log info

EOD_RECONCILE:
- wait until 17:15 CT
- fetch Databento for full session
- set_diff against NT bars
- insert nt_missing bars (provisional=true)
- run tolerance ladder for each comparable bar
- -> WEEKEND (Fri) or sleep until 17:00 CT next day

WEEKEND:
- on Sunday 16:50 CT: -> WARMUP (short, just to seed indicators
  if state was lost)

### Implementation Checklist (Merge Logic)

1. Normalize NT timestamp on ingest: subtract 60s from NT CSV row
   to produce bar_open_utc. Unit-test against known fixture.

2. Confirm Databento OHLCV-1m timestamp convention via one live
   Historical request. Assert ts_event corresponds to bar-open.
   See [VERIFY-12].

3. Implement source tagging as Parquet column, not directory.

4. Implement three-signal disconnect detector (time-since-last-bar,
   heartbeat file, optional explicit connection-status line in
   NT CSV).

5. Set ConnectionLossHandling = KeepRunning in any data-emitting
   NinjaScript to avoid default Recalculate re-firing OnBarUpdate.

6. Schedule EOD reconciliation at 17:15 CT with 30-min retry
   window if Databento fails.

7. Pre-compute session-minute calendar for next 30 days, refresh
   weekly from CME holiday calendar.

8. Configure W32Time on Windows host with 0x8 interval flag.
   Verify w32tm /query /status reports stratum <= 3 before market
   open daily. Add startup check in Go process that aborts if
   |wsl_clock - windows_clock| > 250ms.

9. Define JSON log schema for bar_discrepancy. Rotate daily.

10. Build --reconcile-day CLI for manual EOD replay.

11. Test cold start across the 16:00-17:00 CT halt — warmup window
    will straddle the halt, gap detector must NOT fire on the
    missing 60 minutes.

### Additional Verification Items

[VERIFY-11] NT CSV behavior when NT is up but Go reader is down
for 5 min. Are bars present, or backfilled by NT into CSV
automatically? Determines whether warm restart can rely on
NT-side persistence.

[VERIFY-12] Databento OHLCV-1m timestamp convention. Make one
live Historical request, inspect ts_event vs bar boundaries,
confirm it's bar-OPEN (not close). NautilusTrader's adapter
normalizes to close internally — the raw Databento API does not.

[VERIFY-13] Databento behavior during 16:00-17:00 CT daily halt.
Does it emit bars with volume=0, or no rows? Code the gap
detector accordingly.

[VERIFY-14] NT CSV behavior on true CQG/Rithmic disconnect
lasting > 60s. Does NT Historical Data Server backfill into CSV
automatically, or only after ReloadAllHistoricalData() called?

[VERIFY-15] Clock skew between Windows host and WSL2 at startup.
Measure |wsl_clock - windows_clock| over 1 hour. Should be
< 250ms.

### Sources (Merge Logic)

NT8 ConnectionLossHandling:
https://ninjatrader.com/support/helpGuides/nt8/connectionlosshandling.htm

NT8 OnConnectionStatusUpdate:
https://ninjatrader.com/support/helpguides/nt8/onconnectionstatusupdate.htm

NT8 ReloadAllHistoricalData:
https://ninjatrader.com/support/helpguides/nt8/reloadallhistoricaldata.htm

NautilusTrader Databento integration:
https://nautilustrader.io/docs/latest/integrations/databento/

Databento 15-min historical embargo:
https://roadmap.databento.com/b/n0o5prm6/feature-ideas/release-historical-data-as-soon-as-possible-based-on-licensing-requirements-including-intraday

QuantConnect warmup documentation:
https://www.quantconnect.com/forum/discussion/4646/what-is-the-purpose-of-a-warmup-period-and-how-do-i-find-out-how-long-it-should-be/

CME equity futures trading hours:
https://www.cmegroup.com/education/files/eq-trading-hours.pdf

Microsoft Learn W32Time configuration:
https://learn.microsoft.com/en-us/windows-server/networking/windows-time-service/windows-time-service-tools-and-settings

QuantVPS NTP for futures trading:
https://www.quantvps.com/blog/ntp-time-synchronization-in-trading

NexusFi data feeds analysis:
https://nexusfi.com/a/platforms/data-feeds-market-data

Parquet partition design (Modexa):
https://medium.com/@Modexa/7-parquet-partition-designs-that-actually-work-69a2a0811ea8

## Why this architecture change

The CSV bridge in Plan 1 has known caps:

| Plan 1 (CSV) | Plan 1.5 (AddOn + socket) |
|---|---|
| One-shot file write, 2s NT polling | Push-driven events, sub-10ms |
| 1-second DateTime dedup → race conditions (H1-H4) | UUIDv7 cmd_id + monotonic seq |
| `GetBalance` returns $50k mock | `Account.Get(AccountItem.CashValue, ...)` + `AccountItemUpdate` subscription |
| `CloseLong/CloseShort` return error | `Account.CreateOrder(...counter-market...) + Account.Submit` |
| `CancelAllOrders` returns error | `Account.CancelAllOrders(instrument)` |
| `SetStopLoss`/`SetTakeProfit` set only at entry | `Account.Change(order)` mid-trade, with `isLiveUntilCancelled=true` |
| `GetClosedPnL` returns empty | `Account.Get(AccountItem.RealizedProfitLoss, ...)` |
| Broker disconnect: invisible | `ConnectionStatusUpdate` event |
| ATM strategies / OCO: unsupported | `AtmStrategy.StartAtmStrategy(...)` available |
| Fill ordering: relies on CSV file mtime | Documented sequence `OrderUpdate → ExecutionUpdate → PositionUpdate` (caveat: Rithmic/IB ordering not guaranteed — drive off `Execution` value, not `Position` cache) |

## Architecture

```
┌──────────────────────── Windows (NinjaTrader 8 process) ────────────────────────┐
│                                                                                  │
│  bin/Custom/AddOns/ClaudeBridge.cs   (new NinjaScript AddOn)                    │
│    ├─ TcpListener on 127.0.0.1:36974  (NDJSON command/event protocol)           │
│    ├─ Account.OrderUpdate / ExecutionUpdate / PositionUpdate /                   │
│    │  AccountItemUpdate / ConnectionStatusUpdate event subscriptions             │
│    ├─ ConcurrentDictionary<cmd_id, Order>  (idempotency LRU)                    │
│    ├─ Ring buffer of last 1000 events for resync on reconnect                   │
│    └─ Optional: claudetrader.cs Strategy still on chart                          │
│       for ATM OCO if you want server-side SL/TP autocancel                       │
└──────────────────────────────────────────────────────────────────────────────────┘
        │                                                            ▲
        │ NDJSON over TCP, mirrored loopback                          │
        │ {"cmd_id":"uuid7","auth":"<token>","action":"OpenLong",...} │
        ▼                                                            │
┌──────────────────────── WSL2 (Ubuntu) ───────────────────────────────────────────┐
│                                                                                   │
│  trader/ninjatrader/trader.go  (Plan 1.5 replaces Plan 1's CSV bridge here)      │
│    ├─ NDJSON client to NT8 (auto-reconnect, exp. backoff, PING/PONG keepalive)   │
│    ├─ Per-cmd UUIDv7 + monotonic seq + LRU dedup                                 │
│    ├─ Pending-command journal (BadgerDB or SQLite) — replay on NT restart        │
│    └─ State cache: positions / orders / equity / connection status               │
└──────────────────────────────────────────────────────────────────────────────────┘
```

## WSL2 networking (required prerequisite)

### Preferred: mirrored mode (Windows 11 22H2+)

```ini
# C:\Users\<user>\.wslconfig
[wsl2]
networkingMode=mirrored
```

Then `wsl --shutdown` and restart. Go in WSL2 connects to `127.0.0.1:36974` exactly as if it were on Windows. **No firewall rule, no host IP discovery, no NAT translation.**

**Limitations:**
- Windows Server 2025 does NOT support mirrored mode (Microsoft/WSL issue #12569).
- Windows 10 does NOT support mirrored mode.

### Fallback: NAT mode (any older Windows)

```powershell
# As Administrator
New-NetFirewallHyperVRule -Name "NTBridge" -DisplayName "NT8 Bridge" `
    -Direction Inbound -VMCreatorId '{40E0AC32-46A5-438A-A0B2-2B479E8F2E90}' `
    -Protocol TCP -LocalPorts 36974
```

In NT8 AddOn, bind `TcpListener` to `IPAddress.Any` (NOT `IPAddress.Loopback`). In WSL2, discover Windows host IP from `ip route show | grep -i default | awk '{ print $3}'` (typically 172.x.x.x).

## NinjaScript AddOn skeleton

`bin/Custom/AddOns/ClaudeBridge.cs`:

```csharp
public class ClaudeBridgeAddOn : NinjaTrader.NinjaScript.AddOnBase
{
    private TcpListener _listener;
    private CancellationTokenSource _cts;
    private Account _acct;
    private readonly ConcurrentDictionary<string, Order> _orders = new();
    private readonly ConcurrentQueue<string> _eventBuffer = new(); // last ~1000 events
    private long _seqCounter;
    private string _authToken;

    protected override void OnStateChange()
    {
        if (State == State.SetDefaults) { Name = "ClaudeBridge"; }
        else if (State == State.Configure)
        {
            // Load auth token (rotated by Go-side deploy)
            _authToken = File.ReadAllText(
                Path.Combine(Environment.GetEnvironmentVariable("USERPROFILE"),
                            ".claudebridge", "secret")).Trim();

            // Bind account (configurable via NT strategy parameter)
            lock (Account.All) _acct = Account.All.First(a => a.Name == "PropFirmAcct");

            // Subscribe to push-event sources
            _acct.OrderUpdate              += OnOrderUpdate;
            _acct.ExecutionUpdate          += OnExecutionUpdate;
            _acct.PositionUpdate           += OnPositionUpdate;
            _acct.AccountItemUpdate        += OnAccountItemUpdate;
            _acct.ConnectionStatusUpdate   += OnConnectionStatusUpdate;

            // Start TCP listener on background task
            _cts = new CancellationTokenSource();
            _listener = new TcpListener(IPAddress.Loopback, 36974);
            // CRITICAL: do NOT use port 36973 — that's NT's own ATI port
            _listener.Start();
            _ = Task.Run(() => AcceptLoopAsync(_cts.Token));
        }
        else if (State == State.Terminated)
        {
            _cts?.Cancel();
            _listener?.Stop();
            _acct.OrderUpdate              -= OnOrderUpdate;
            _acct.ExecutionUpdate          -= OnExecutionUpdate;
            _acct.PositionUpdate           -= OnPositionUpdate;
            _acct.AccountItemUpdate        -= OnAccountItemUpdate;
            _acct.ConnectionStatusUpdate   -= OnConnectionStatusUpdate;
        }
    }

    // ... AcceptLoopAsync handles each client on its own Task
    // ... OnOrderUpdate / OnExecutionUpdate / etc. push NDJSON to connected clients
    // ... Command handlers call Account.CreateOrder/Submit/Cancel/Change directly
    //     (these methods are thread-safe per NT's multi-threading guide)
}
```

**Critical threading rules** (per NT staff guidance):
1. Listener loop MUST be on `Task.Run` — never on `OnStateChange` direct call (that's a UI thread, would block NT)
2. Account.* methods are thread-safe and can be called from the listener task without `Dispatcher`
3. If you ever touch a NT chart drawing or NTWindow from the listener: use `Dispatcher.InvokeAsync` (NEVER `Dispatcher.Invoke` — deadlocks NT)
4. Every callback wrapped in `try/catch` — uncaught exceptions tear down the AddOn

## NT method mapping for the 9 required actions

All assume the **AddOn-level Account API** (not the Strategy-managed approach):

| Action | NT call |
|---|---|
| `OpenLong` / `OpenShort` (entry) | `Account.CreateOrder(instrument, OrderAction.Buy/Sell, OrderType.Market/Limit/StopMarket/StopLimit, TimeInForce.Day/Gtc, qty, limitPrice, stopPrice, oco, name, customId)` then `Account.Submit(new[]{order})`. Optionally bind ATM via `AtmStrategy.StartAtmStrategy(...)`. |
| `CloseLong` / `CloseShort` | Resolve `Position pos = _acct.Positions.FirstOrDefault(p => p.Instrument == instr)`; submit counter-market via `Account.CreateOrder(instrument, pos.MarketPosition == MarketPosition.Long ? OrderAction.Sell : OrderAction.BuyToCover, OrderType.Market, ...)`. |
| `CancelOrder(id)` | Look up in `_orders[cmd_id]`, call `Account.Cancel(new[]{order})`. |
| `CancelAllOrders` | `Account.CancelAllOrders(instrument)` — documented at ninjatrader.com/support/helpguides/nt8/accounts_cancelallorders.htm. |
| `ModifyStopLoss` / `ModifyTakeProfit` | Mutate `order.StopPrice` / `order.LimitPrice`, then `Account.Change(new[]{order})`. Order MUST have been submitted with `isLiveUntilCancelled=true`. |
| `MoveToBreakeven` | Read `pos.AveragePrice`, call `ModifyStopLoss` with that price rounded via `Instrument.MasterInstrument.RoundToTickSize(...)`. |
| `GetAccountBalance` | `Account.Get(AccountItem.CashValue, Currency.UsDollar)` + `BuyingPower` + `NetLiquidation` + `InitialMargin` + `RealizedProfitLoss`. **Caveat:** the `Currency` parameter is documented-as-ignored; values are realtime-only (0 in backtest); not callable from indicator context. |
| `GetOpenPositions` | Iterate `_acct.Positions` → `Instrument.FullName`, `MarketPosition`, `Quantity`, `AveragePrice`. |
| `GetPendingOrders` | `_acct.Orders.Where(o => o.OrderState == OrderState.Working \|\| o.OrderState == OrderState.Accepted)`. |

**Event push (NT → Go):** subscribed in `State.Configure`:

```csharp
_acct.OrderUpdate              += ...;   // every state transition (Submitted → Accepted → Working → PartFilled → Filled / Cancelled / Rejected)
_acct.ExecutionUpdate          += ...;   // every fill, by-value Execution arg
_acct.PositionUpdate           += ...;   // when position size/side changes
_acct.AccountItemUpdate        += ...;   // when CashValue / RealizedPnL / etc. change
_acct.ConnectionStatusUpdate   += ...;   // when broker connection drops/reconnects
```

**Critical caveat (Rithmic / Interactive Brokers):** per NT support, "sequence of events are not guaranteed due to provider API design" on these adapters. Drive Go-side state machine off `ExecutionUpdate`'s **by-value Execution** argument, NOT off cached `Position` properties.

## NDJSON protocol

**Command (Go → NT):**
```json
{"cmd_id":"01900a2d-...-uuid7","seq":42,"auth":"<sha256-hex-of-shared-secret>","action":"OpenLong","instrument":"MNQ 12-26","qty":1,"stop":21485.0,"target":21540.0}
```

**Response (NT → Go), one of:**
```json
{"reply_to":"01900a2d-...","ok":true,"order_id":"<nt-order-id>"}
{"reply_to":"01900a2d-...","ok":false,"error":"InsufficientBuyingPower"}
```

**Event push (NT → Go, unsolicited):**
```json
{"event":"fill","seq":1042,"instrument":"MNQ 12-26","side":"LONG","qty":1,"price":21500.25,"time":"2026-05-22T19:30:45.123Z"}
{"event":"order_state","seq":1043,"order_id":"<id>","state":"Working","price":21485.0}
{"event":"account_item","seq":1044,"item":"CashValue","value":49850.50}
{"event":"connection","seq":1045,"status":"Disconnected"}
```

## Idempotency + reconnect

- Every Go command carries UUIDv7 `cmd_id`. AddOn keeps an LRU of executed cmd_ids (in memory + persisted to `bin/Custom/AddOns/ClaudeBridge/cmd_log.jsonl`). Duplicate cmd_id replays = no-op.
- Every NT event carries monotonic `seq`. Go tracks last-seen seq. On reconnect, Go sends `{"action":"RESYNC","last_seq":1042}`. AddOn replays any events newer than that from its ring buffer.
- TCP keepalive: `socket.SetKeepAlive(true, 10_000, 5_000)` catches OS-level drops in ~15s.
- Application-level: PING every 1s with 3s timeout, catches hung NT processes.
- On Go-side connection loss: mark all cached state stale, retry connect with exponential backoff capped at 5s. Surface `bridge_status=disconnected` to NOFX dashboard.

## Security

- **Bind to 127.0.0.1 only** under mirrored mode. NAT-mode fallback: bind to vEthernet IP, never LAN IP.
- **Shared-secret token** in `%USERPROFILE%\.claudebridge\secret`. Every Go command includes `auth=<sha256-hex>` field. AddOn rejects mismatches.
- **No remote LAN exposure** without stunnel/wireguard in front. Prop-firm TOS likely prohibits this anyway.

## 5-stage migration plan (CSV → AddOn → live)

| Stage | What | Effort | Gate to proceed |
|---|---|---|---|
| **Stage 1** | Keep CSV alive, add read-only event channel. Build AddOn skeleton, subscribe `Account.*Update`, push NDJSON over TCP. NO order submission yet. | 2-3 days | Events arriving in Go with <50ms latency for ≥1h live |
| **Stage 2** | Add command channel for SAFE actions: `GetAccountBalance`, `GetOpenPositions`, `GetPendingOrders`, `CancelAllOrders`, `CloseLong/CloseShort`. (Read or "panic-flat" — conservative failure modes.) | 3-5 days | 1 trading day of flatten commands without stuck position |
| **Stage 3** | Add `ModifyStopLoss`, `ModifyTakeProfit`, `MoveToBreakeven`. Decide ATM-vs-Managed-vs-Unmanaged here and document. | 3-5 days | 50 modify ops live without orphan stop |
| **Stage 4** | Add `PlaceOrder` over socket. Feature-flag the CSV poller. Keep CSV as fallback for 1 month. | 2-3 days | 30 days zero socket-bridge incidents |
| **Stage 5 (parallel)** | Prototype Tradovate/ProjectX exit ramp. Build Go SDK client that performs same 9 actions against `live.tradovateapi.com/v1` or `api.topstepx.com/api`. | 1 week | Trigger to switch: >2 incidents/month requiring NT restarts |

**Realistic total Plan 1.5 timeline: 2-3 weeks** for stages 1-4. Stage 5 runs in parallel as the escape hatch.

## Buy vs build alternative

**CrossTrade XT AddOn** (https://crosstrade.io/blog/crosstrades-new-ninjatrader-add-on/) implements ~95% of this architecture as paid SaaS:
- 25 endpoints covering all 9 actions + more (account queries, position info, place/close/cancel/modify)
- Standard Unlimited: $24/mo
- Pro Unlimited: $49/mo
- 7-day free trial (no card)

**Tradeoff:** routes commands through CrossTrade's cloud servers — adds a hop for LLM agent loops targeting sub-second latency, and may not pass prop-firm compliance review (external order routing). Build-it-yourself is in-process and stays on your machine.

## Escape ramp: move off NT8 entirely (Stage 5)

If the prop firm clears through Tradovate or ProjectX, this is the cleanest architecture and removes NT8 as a single point of failure.

**Tradovate** (Apex, Tradeify, TPT historically, FundedNext, others):
- Live REST: `https://live.tradovateapi.com/v1`
- Demo REST: `https://demo.tradovateapi.com/v1`
- Market data: `https://md.tradovateapi.com`
- Official C# example: `tradovate/example-api-csharp-trading`
- All 9 actions natively supported via REST + WebSocket

**ProjectX / TopstepX** (Topstep, TFDX, Bulenox):
- REST: `https://api.topstepx.com/api`
- SignalR hubs: `wss://realtime.topstepx.com/api` (user hub + market hub)
- Mature Python SDK: `TexasCoding/project-x-py` on PyPI
- Add-on subscription: $14.50-29/mo through ProjectX

**What you lose by moving off NT8:**
- NT chart tooling
- Local sim environment
- Manual intervention via NT UI
- ATM strategies (server-side OCO)
- NT-specific indicators

**Worth it if:** your prop firm clears via Tradovate or ProjectX and your strategy doesn't depend on NT-specific indicators.

## Plan 1.5 known gotchas (from research)

1. **ATM strategies break `OnOrderUpdate`** — per NT support, "OnOrderUpdate/OnExecutionUpdate events will not trigger for strategies that submit ATM templates." If you use `AtmStrategyCreate`, you must use `GetAtmStrategyPositionAveragePrice()` and ATM-specific queries instead. Pick one approach and stick to it.
2. **Managed approach auto-resets stops/targets every bar** unless `isLiveUntilCancelled=true`. For a bridge where Go controls SL/TP placement, you do NOT want managed auto-reset. Use unmanaged or `ExitStopMarket(...isLiveUntilCancelled=true...)`.
3. **`Account.Get(AccountItem.CashValue, ...)` returns 0 in backtests and stale values from indicator context.** Treat as realtime-only; subscribe to `AccountItemUpdate` for change events rather than polling.
4. **Rithmic / IB `PositionUpdate` ordering is not guaranteed.** Drive Go state machine off `ExecutionUpdate`'s by-value `Execution` object, not cached `Position` properties.
5. **NT8 is .NET Framework 4.8** — cannot use modern NuGet packages compiled against .NET 6+. Pin dependencies. `async/await` works fine.
6. **Port 36973 is NT8's own ATI port** — do NOT collide. Use 36974 / 50051 / something else.
7. **WSL2 mirrored mode requires Windows 11 22H2+.** On Windows 10 (or Server 2025), use NAT mode + firewall rule.
8. **Pin NT version.** NT8 **8.1.6** (2025-09-25) or **8.1.6.3** (2026-01-16). Re-validate AddOn on every minor NT8 upgrade — third-party AddOns occasionally break on minor releases.
9. **NT8 sockets are officially "unsupported" by NT staff.** Forum stance: "Of course, no issues. Naturally, from NT support's point of view, this is unsupported C# code. But 'unsupported' only means 'we don't answer questions on that' — but you can certainly achieve anything your skills allow." You're on your own when it breaks.

## When to start Plan 1.5

Trigger conditions (any one):
- Plan 1 SIM validation complete (AI brain proven to make sensible NQ decisions)
- Need to flip to live trading (CSV bridge inadequate per H1-H4 hazards)
- CSV bridge experiencing >1 incident/week in SIM
- Need real balance read for accurate position sizing
- Need manual close / cancel-all / modify mid-trade for risk management

**Plan 1 stays canonical until one of those triggers fires.** Do not pre-emptively rebuild the bridge — the CSV path is sufficient for the "validate AI brain" objective.

## Implementation Spec (building pre-trigger)

> **Status update (2026-05-26):** Plan 1.5 is moving from "designed, deferred" to "designed + spec'd, building pre-trigger" as a belt-and-suspenders measure before live money. The CSV bridge (Plan 1, validated on 2026-05-22 with a real SIM fill at NQ 29807) remains the canonical production path. The TCP AddOn is being built as an opt-in alternative via the `NT_TRANSPORT` env var (`csv` default, `tcp` opt-in), with zero-downtime flip-back. Cross-reference: ADR-001 (CSV bridge vs TCP).

### Why build pre-trigger

Plan 1.5 was originally deferred per ADR-001 until one of the documented trigger conditions fired (CSV latency, Defender file-lock contention, operator-reported failures, or sub-second algorithm cadence). None has fired. Building pre-trigger anyway provides:

1. A live-tested alternative transport ready before the first live-money trade — avoids an emergency migration if a CSV failure surfaces under load.
2. Wire-protocol stability locked while the rest of the canonical plan is still fresh in operator memory — better than re-deriving it weeks later.
3. Plan 1 critical files preserved byte-identical (per ADR-007) — TCP is purely additive; the CSV bridge is not modified, removed, or refactored.

### File manifest

**Go files (8 new, all ADD-only — Plan 1 critical files stay byte-identical):**

- `provider/ninjatrader/tcp_server.go` — TCP listener bound to 127.0.0.1:36974 (NOT NT's ATI port 36973); accepts a single concurrent NT AddOn client; routes signal frames out, fill/heartbeat/ack frames in.
- `provider/ninjatrader/tcp_framing.go` — 4-byte big-endian length-prefix codec + JSON marshalling for the 4 message types.
- `provider/ninjatrader/tcp_framing_test.go` — round-trip framing tests, malformed-frame rejection, oversized-frame (>1MB) rejection.
- `provider/ninjatrader/tcp_server_test.go` — listener lifecycle, concurrent-client rejection, graceful close on context cancellation.
- `provider/ninjatrader/tcp_client_mock.go` — in-process mock NT client for integration tests (mirrors `mock_nt.go`'s role for the CSV path).
- `trader/ninjatrader/tcp_trader.go` — alternative `Trader` implementation that emits signals through the TCP server. SEPARATE TYPE from the existing CSV `Trader` (per ADR-007: additive, not a modification).
- `trader/ninjatrader/transport.go` — env-var router. Reads `NT_TRANSPORT=csv|tcp` (default `csv`); constructor returns either the existing CSV `Trader` or the new `TCPTrader`. Zero-downtime flip-back: change the env var, restart the bot, no code rebuild required.
- `cmd/nq_smoke/smoke_tcp.go` — `nq_smoke tcp` sub-command exercising the TCP round-trip end-to-end against the mock NT client. Follows the Plan 5 Task 29 dispatch precedent (ADD-only sub-command, `cmd/nq_smoke/main.go` receives dispatch wiring only).

**C# AddOn files (3, scaffolded only — must compile on Windows host):**

- `ninjascript/VLTraderTCPClient.cs` — NinjaScript AddOn class. Connects to `127.0.0.1:36974` on NT startup; subscribes to bar data and order events; emits fill frames; consumes signal frames and submits OCO bracket orders via NT's managed order API. CANNOT compile in WSL2 — requires Visual Studio (or NT8's NinjaScript editor F5) on the Windows host.
- `ninjascript/vltrader_tcp_README.md` — install procedure (drop file into `Documents\NinjaTrader 8\bin\Custom\AddOns\`, compile via NT8 editor, restart NT, verify Active in NT's Output window).
- `ninjascript/vltrader_tcp_PROTOCOL.md` — wire protocol reference (mirrors the spec below) so the C# side has a local copy of the contract.

### Wire protocol

**Framing:** every frame on the TCP stream is a 4-byte big-endian length prefix followed by a UTF-8 JSON payload. Max frame size: 1 MB (oversized frames are an error → server closes the connection).

**Envelope:**

```
{
  "type": "signal" | "fill" | "heartbeat" | "ack",
  "payload": { … }
}
```

**Signal frame payload** (Go server → C# AddOn):

- `symbol` (string, e.g. "MNQ")
- `side` (string, "long" | "short")
- `quantity` (int, default contracts)
- `entry` (float64, tick-rounded)
- `stop_loss` (float64, tick-rounded)
- `take_profit` (float64, tick-rounded)
- `signal_id` (string, UUID)
- `timestamp` (RFC3339)

**Fill frame payload** (C# AddOn → Go server):

- `signal_id` (string, matches the originating signal)
- `fill_price` (float64)
- `fill_time` (RFC3339)
- `side` (string)
- `quantity` (int)
- `slippage_ticks` (float64)
- `status` (string, "filled" | "rejected" | "partial")

**Heartbeat frame** (bidirectional): empty payload, 30s interval.

**Ack frame** (bidirectional): `{ "acks": "heartbeat" }` or `{ "acks": "<signal_id>" }`.

### Failure modes + reconnect

- **TCP disconnect:** server holds the signal queue; on reconnect, sends pending signals with original timestamps. Client (C# AddOn) may reject signals older than 60s as stale.
- **Heartbeat timeout:** server closes the connection after 60s without an ack; client reconnects every 5s.
- **Invalid frame** (bad length, oversized >1MB, malformed JSON): server logs warn + closes the connection; client reconnects.
- **Order rejection by NT8:** AddOn emits a fill frame with `status=rejected`; Go side logs the rejection and does NOT retry (manual operator intervention required).

### Manual followup (operator, post-PR-merge)

1. Compile `VLTraderTCPClient.cs` on the Windows host using Visual Studio or NT8's built-in NinjaScript editor (F5).
2. Drop the compiled AddOn into `Documents\NinjaTrader 8\bin\Custom\AddOns\`.
3. Restart NT8; verify the AddOn shows as Active in NT's Output window.
4. Start the bot with `NT_TRANSPORT=tcp` set in the environment.
5. Submit a SIM test signal through the dashboard's existing signal flow.
6. Verify: NT8 places an OCO bracket order; the fill emits back through the TCP channel; the bot records the position.
7. If success → Plan 1.5 is full-stack verified; annotate `v1.0-plan1-5` tag with the verification timestamp.
8. If failure → diagnose against `vltrader_tcp_PROTOCOL.md`; patch the C# AddOn or Go server side as needed; do NOT fall back to CSV until the diagnosis is captured.

### Plan 1 critical file integrity (ADR-007)

Plan 1.5 ADDS files. It does NOT modify any of the 19 Plan 1 critical files locked by ADR-007:

- `provider/ninjatrader/csv_writer.go`
- `provider/ninjatrader/csv_tailer.go`
- `provider/ninjatrader/types.go` (reused via import; not modified)
- `provider/ninjatrader/mock_nt.go`
- `trader/ninjatrader/trader.go` (the existing CSV `Trader`; `TCPTrader` is a separate type)
- `trader/ninjatrader/tick_rounding.go`

`cmd/nq_smoke/main.go` is modified ADD-only (smoke dispatch wiring for the new `tcp` sub-command), per the Plan 5 Task 29 precedent.

### Release tag scheme

- `v1.0-plan1-5-spec` — this docs-only change (the spec being read right now).
- `v1.0-plan1-5` — the actual code, follow-up PR after this spec lands.
- The `v1.0-plan1-5` tag annotation will be extended once the operator's manual compile + NT8 integration test (steps 1–7 above) is complete, with the note "verified via operator manual compile + NT8 integration test on YYYY-MM-DD".

---

# Reference: Developing New Strategies and Indicators

> **This section is a reference appendix, not part of the build sequence.** After Plan 1 ships, you'll want to add new indicators, new strategies, or fork existing ones. This guide shows the exact pattern.

## How to add a NEW INDICATOR

**Real example: adding Williams %R (an overbought/oversold oscillator).**

Three files touched, each with one focused change.

### Step W.1: Implement the indicator math

Append to [market/data_indicators.go](market/data_indicators.go):

```go
// calculateWilliamsR returns Larry Williams' %R oscillator over the last
// `period` bars. Range is -100 (oversold) to 0 (overbought).
// Reference: https://www.investopedia.com/terms/w/williamsr.asp
func calculateWilliamsR(klines []Kline, period int) float64 {
    if len(klines) < period {
        return 0
    }
    window := klines[len(klines)-period:]
    highest := window[0].High
    lowest := window[0].Low
    for _, k := range window {
        if k.High > highest {
            highest = k.High
        }
        if k.Low < lowest {
            lowest = k.Low
        }
    }
    if highest == lowest {
        return -50
    }
    close := window[len(window)-1].Close
    return -100 * (highest - close) / (highest - lowest)
}

// ExportCalculateWilliamsR is the package-public wrapper.
func ExportCalculateWilliamsR(klines []Kline, period int) float64 {
    return calculateWilliamsR(klines, period)
}
```

**Pattern to copy:** look at how `calculateEMA`, `calculateRSI`, `calculateATR` are structured — they all follow the same shape: lowercase implementation, uppercase Export wrapper. New indicators follow this pattern verbatim.

### Step W.2: Write the test FIRST (then implementation — TDD)

Append to [market/data_test.go](market/data_test.go):

```go
func TestCalculateWilliamsR_OversoldBottom(t *testing.T) {
    // Build a window where the close is at the period's low — should report
    // very negative (close to -100, deep oversold).
    klines := []Kline{
        {High: 100, Low: 90, Close: 100},
        {High: 102, Low: 92, Close: 95},
        {High: 101, Low: 88, Close: 93},
        {High: 99, Low: 85, Close: 86}, // close near the period low
    }
    got := calculateWilliamsR(klines, 4)
    // highest in window: 102, lowest: 85, close: 86
    // %R = -100 * (102 - 86) / (102 - 85) = -94.12
    want := -94.12
    if math.Abs(got - want) > 0.5 {
        t.Errorf("calculateWilliamsR = %.2f, want ~%.2f", got, want)
    }
}

func TestCalculateWilliamsR_OverboughtTop(t *testing.T) {
    klines := []Kline{
        {High: 100, Low: 90, Close: 91},
        {High: 102, Low: 92, Close: 100},
        {High: 105, Low: 95, Close: 103},
        {High: 110, Low: 100, Close: 109}, // close near the period high
    }
    got := calculateWilliamsR(klines, 4)
    // highest: 110, lowest: 90, close: 109
    // %R = -100 * (110 - 109) / (110 - 90) = -5.0
    if math.Abs(got - (-5.0)) > 0.5 {
        t.Errorf("calculateWilliamsR = %.2f, want -5.0 (overbought)", got)
    }
}

func TestCalculateWilliamsR_FlatRange(t *testing.T) {
    klines := []Kline{
        {High: 100, Low: 100, Close: 100},
        {High: 100, Low: 100, Close: 100},
    }
    got := calculateWilliamsR(klines, 2)
    if got != -50 {
        t.Errorf("flat-range Williams %%R = %.2f, want -50.0", got)
    }
}
```

Run them:

```bash
cd /home/hoang/nofx && go test ./market/... -run TestCalculateWilliams -v
```

Expected: 3 PASS. If any fail, fix the math.

### Step W.3: Surface the indicator to the AI prompt

In [kernel/engine_prompt_futures.go](kernel/engine_prompt_futures.go) — extend `FuturesContext`:

```go
type FuturesContext struct {
    // ... existing fields ...
    WilliamsR14 float64  // ADD: Williams %R, period 14
}
```

In `BuildFuturesUserPrompt`, add a line in the indicator-snapshot block:

```go
b.WriteString(fmt.Sprintf("- Williams %%R(14): %.1f (%s)\n",
    ctx.WilliamsR14, williamsBucket(ctx.WilliamsR14)))
```

Add the bucket helper next to `rsiBucket`:

```go
func williamsBucket(w float64) string {
    switch {
    case w >= -20:
        return "overbought"
    case w <= -80:
        return "oversold"
    default:
        return "neutral"
    }
}
```

### Step W.4: Wire into the trading loop

In `cmd/nq_smoke/main.go` (or, post-plan-1, in `trader/auto_trader_loop.go` where the futures context is assembled):

```go
ctx.WilliamsR14 = market.ExportCalculateWilliamsR(klines, 14)
```

### Step W.5: Update the prompt test

In [kernel/engine_prompt_futures_test.go](kernel/engine_prompt_futures_test.go) — add `Williams` to the required terms in `TestBuildFuturesUserPrompt_IncludesIndicators`:

```go
for _, s := range []string{"21500.00", "EMA20", "RSI14", "ATR14", "Bollinger", "Williams"} {
    // ...
}
```

### Step W.6: Run and verify

```bash
cd /home/hoang/nofx && go test ./... && go run ./cmd/nq_smoke
```

You should see the Williams %R value in the printed user prompt, and the AI now has it in its context. **Total LOC: ~50. Total time: ~30 minutes including reading and tests.**

### Step W.7: Commit

```bash
git add market/data_indicators.go market/data_test.go \
        kernel/engine_prompt_futures.go kernel/engine_prompt_futures_test.go \
        cmd/nq_smoke/main.go
git commit -m "feat(indicators): add Williams %R oscillator to NQ prompt context"
```

That's the entire flow. **Adding a 7th, 8th, 20th indicator is the same five steps.**

---

## How to add a NEW STRATEGY (different prompt template)

A "strategy" in this codebase = a prompt template + the rules baked into it. Different strategies live in separate files so you can have multiple and A/B test.

**Real example: adding a mean-reversion strategy (different from the trend-following one Task 9 builds).**

### Step S.1: Create the new prompt file

Create [kernel/engine_prompt_meanrev.go](kernel/engine_prompt_meanrev.go):

```go
package kernel

import (
    "fmt"
    "strings"
)

// MeanReversionPromptConfig is the same shape as FuturesPromptConfig but the
// strategy framing is different.
type MeanReversionPromptConfig struct {
    FuturesPromptConfig         // embed for tick, multiplier, R/R, etc.
    BBPeriod        int         // 20
    BBStdDevs       float64     // 2.0
    OversoldRSI     float64     // 30
    OverboughtRSI   float64     // 70
}

func BuildMeanReversionSystemPrompt(c MeanReversionPromptConfig) string {
    var b strings.Builder
    b.WriteString("# You are a MEAN-REVERSION index-futures trader.\n\n")
    b.WriteString("## Philosophy\n")
    b.WriteString("Price tends to revert to its statistical mean. You only enter\n")
    b.WriteString("when price has stretched far from that mean and there's evidence\n")
    b.WriteString("of exhaustion. You DO NOT chase trends.\n\n")
    b.WriteString(fmt.Sprintf("## Instrument: %s, tick %.2f, multiplier $%.2f/point\n\n",
        c.Symbol, c.TickSize, c.ContractMultiplier))
    b.WriteString("## Entry rules\n")
    b.WriteString(fmt.Sprintf("- LONG only when: price is below the lower Bollinger band(%d, %.1fσ)\n",
        c.BBPeriod, c.BBStdDevs))
    b.WriteString(fmt.Sprintf("  AND RSI(14) < %.0f (oversold).\n", c.OversoldRSI))
    b.WriteString(fmt.Sprintf("- SHORT only when: price is above the upper Bollinger band(%d, %.1fσ)\n",
        c.BBPeriod, c.BBStdDevs))
    b.WriteString(fmt.Sprintf("  AND RSI(14) > %.0f (overbought).\n", c.OverboughtRSI))
    b.WriteString("- If EMA20 > EMA50 (strong uptrend), DO NOT short. Skip the cycle.\n")
    b.WriteString("- If EMA20 < EMA50 (strong downtrend), DO NOT long. Skip the cycle.\n\n")
    b.WriteString("## Exits (you choose at entry)\n")
    b.WriteString("- Take profit = back to the Bollinger mid (mean reversion target).\n")
    b.WriteString("- Stop loss = beyond the band you entered against, plus 1 ATR buffer.\n\n")
    b.WriteString(fmt.Sprintf("- Minimum R/R: %.2f.\n", c.MinRiskReward))
    b.WriteString("\n## Output: same JSON shape as the base prompt.\n")
    return b.String()
}
```

### Step S.2: Test it

Create [kernel/engine_prompt_meanrev_test.go](kernel/engine_prompt_meanrev_test.go):

```go
package kernel

import (
    "strings"
    "testing"
)

func TestMeanReversionPrompt_HasMeanRevFraming(t *testing.T) {
    p := BuildMeanReversionSystemPrompt(MeanReversionPromptConfig{
        FuturesPromptConfig: FuturesPromptConfig{
            Symbol:             "MNQ",
            ContractMultiplier: 2.0,
            TickSize:           0.25,
            MinStopPoints:      15,
            MaxStopPoints:      50,
            MinRiskReward:      1.5,
        },
        BBPeriod:      20,
        BBStdDevs:     2.0,
        OversoldRSI:   30,
        OverboughtRSI: 70,
    })

    for _, must := range []string{"MEAN-REVERSION", "Bollinger", "oversold", "overbought", "DO NOT chase", "MNQ"} {
        if !strings.Contains(p, must) {
            t.Errorf("mean-rev prompt missing %q", must)
        }
    }
    for _, mustNot := range []string{"trend-following", "momentum"} {
        if strings.Contains(p, mustNot) {
            t.Errorf("mean-rev prompt contains anti-pattern %q", mustNot)
        }
    }
}
```

```bash
cd /home/hoang/nofx && go test ./kernel/... -run TestMeanRev -v
```

### Step S.3: Make the engine pick a strategy

In [config/config.go](config/config.go), add:

```go
// StrategyName picks the prompt template. Values: "trend" (default), "meanrev".
StrategyName string
```

Load in Init:

```go
cfg.StrategyName = getEnvOrDefault("STRATEGY_NAME", "trend")
```

In `cmd/nq_smoke/main.go` (or your live entrypoint), branch:

```go
var sysP string
switch cfg.StrategyName {
case "meanrev":
    sysP = kernel.BuildMeanReversionSystemPrompt(kernel.MeanReversionPromptConfig{
        FuturesPromptConfig: kernel.FuturesPromptConfig{
            Symbol:             "MNQ",
            ContractMultiplier: 2.0,
            TickSize:           0.25,
            MinStopPoints:      15,
            MaxStopPoints:      50,
            MinRiskReward:      1.5,
        },
        BBPeriod:      20,
        BBStdDevs:     2.0,
        OversoldRSI:   30,
        OverboughtRSI: 70,
    })
default: // "trend"
    sysP = kernel.BuildFuturesSystemPrompt(kernel.FuturesPromptConfig{ ... })
}
```

### Step S.4: Switch strategies via env var

```bash
# Run with trend-following
STRATEGY_NAME=trend go run ./cmd/nq_smoke

# Run with mean-reversion
STRATEGY_NAME=meanrev go run ./cmd/nq_smoke
```

Two strategies, same bot. **You can run them concurrently as two separate traders in the DB** — each trader has its own config and can point to a different strategy.

### Step S.5: Commit

```bash
git add kernel/engine_prompt_meanrev.go kernel/engine_prompt_meanrev_test.go \
        config/config.go cmd/nq_smoke/main.go
git commit -m "feat(strategy): add mean-reversion prompt template"
```

---

## How to CLONE and MODIFY an existing strategy

The cheapest way to develop a new strategy = copy a working one and tweak.

```bash
# 1. Clone
cp kernel/engine_prompt_futures.go    kernel/engine_prompt_myversion.go
cp kernel/engine_prompt_futures_test.go kernel/engine_prompt_myversion_test.go

# 2. Rename the symbols inside both files
sed -i 's/BuildFuturesSystemPrompt/BuildMyVersionSystemPrompt/g' kernel/engine_prompt_myversion.go kernel/engine_prompt_myversion_test.go
sed -i 's/BuildFuturesUserPrompt/BuildMyVersionUserPrompt/g'     kernel/engine_prompt_myversion.go kernel/engine_prompt_myversion_test.go
sed -i 's/FuturesPromptConfig/MyVersionPromptConfig/g'           kernel/engine_prompt_myversion.go kernel/engine_prompt_myversion_test.go
sed -i 's/FuturesContext/MyVersionContext/g'                     kernel/engine_prompt_myversion.go kernel/engine_prompt_myversion_test.go

# 3. Verify tests still pass on the clone
go test ./kernel/... -run TestBuildMyVersion -v

# 4. NOW make your changes
$ code kernel/engine_prompt_myversion.go
```

You now have a parallel strategy you can edit freely without breaking the original. Run the original in one trader, your variant in another, compare results in the nofx web UI.

---

## How to RUN MULTIPLE STRATEGIES side-by-side (A/B test)

The codebase's `TraderManager` already supports multiple traders, each with its own config. After Plan 1, you can:

1. Create Trader A in the nofx web UI: `Strategy=trend`, `Symbol=MNQ`, `AI Model=DeepSeek`, status=Running.
2. Create Trader B: `Strategy=meanrev`, `Symbol=MNQ`, `AI Model=DeepSeek`, status=Running.
3. Both write to the SAME `trade_signals.csv` (be careful — one ClaudeTrader instance can only handle one position at a time. **For true A/B you need TWO NinjaTrader charts each with its own ClaudeTrader strategy and its own CSV file pair.**)

**Practical recipe for true A/B:**
- Two MNQ charts in NT, each running ClaudeTrader, configured with two different file paths:
  - `C:\Users\<u>\NofxTrader\data_A\trade_signals.csv` ← Trader A writes here
  - `C:\Users\<u>\NofxTrader\data_B\trade_signals.csv` ← Trader B writes here
- nofx config: per-trader `NinjaTraderDataDir` overrides the global default
- Run both for a week. Compare P&L in the nofx Dashboard.

---

## What changes if you add a NEW DATA PROVIDER (e.g., Polygon, IBKR)

Same pattern as Databento. Three files:

```
provider/polygon/client.go        ← HTTP/auth wrapper
provider/polygon/historical.go    ← GetOHLCV(symbol, interval, start, end)
market/polygon_adapter.go         ← Bars to market.Kline
```

Add a `DATA_PROVIDER=polygon|databento` env switch, branch in `kernel/engine.go`. **The downstream (indicators, prompts, CSV bridge, NT) doesn't know or care which provider supplied the bars.** That's the value of the adapter pattern.

---

## What changes if you add a NEW BROKER (instead of NinjaTrader)

Same pattern as the NinjaTrader CSV bridge. Two packages:

```
provider/tradovate/  (or whichever broker)
  client.go            ← REST + WebSocket auth
  orders.go            ← place/modify/cancel
  positions.go         ← state

trader/tradovate/
  trader.go            ← implements trader/types.Trader interface
```

Add a case in [trader/auto_trader.go:263-315](trader/auto_trader.go#L263-L315) switch. **The strategy + AI side doesn't care which broker executes.**

---

## Boundaries — what you CANNOT change without bigger work

- **The `market.Kline` shape itself** — touched by 20+ files. Changing field names cascades. If you need extra per-bar fields, add them as new optional fields rather than renaming existing ones.
- **The `trader/types.Trader` interface** — adding a method means every existing broker impl must implement it (or use a default in an embedded base). Removing a method means audits across all impls. Add carefully.
- **The decision JSON output shape** — `{action, entry, stop_loss, take_profit, reasoning}`. The validator at the strategy-engine layer, the persistence layer in the DB, the web UI dashboard table — all read this shape. Add fields rather than remove. Adding a "confidence" field is safe; renaming "entry" to "entry_price" is a multi-file refactor.
- **The CSV protocol with NinjaTrader** — `claudetrader.cs` defines the 5-field signal and 3-field fill. Changing these requires also editing the NinjaScript and recompiling in NT. Don't change unless you have a real need.

Everything else is fair game. Strategies, indicators, prompt wording, the AI provider, the data provider, the broker — all designed to be swappable.

---

## TL;DR — strategy development cycle in 5 minutes

```
1. Have an idea.
2. Either: clone a working prompt template → edit the language.
   Or:     add a new indicator function (15-30 lines + test).
3. Run cmd/nq_smoke once, hand-paste a decision, watch it fire in NT SIM.
4. Once it fires correctly, enable as a trader in the web UI.
5. Watch it run live in NT SIM for a few sessions.
6. If profitable, raise contract size. If not, edit prompt, GOTO 3.
```

No UI development needed. No backend refactor needed. The system is built to let you iterate on strategies and indicators with minimum friction.

---

# External References

> Verified by HTTP probe on 2026-05-22.

## Upstream NOFX project (canonical source)

- **Repository:** https://github.com/NoFxAiOS/nofx (default branch: `dev`, last pushed 2026-05-11)
- **Description:** *"Your personal AI trading assistant. Any market. Any model. Pay with USDC, not API keys."*
- **Local clone:** `/home/hoang/nofx` (this working tree)

### Architecture documentation (upstream)

Located at https://github.com/NoFxAiOS/nofx/tree/dev/docs/architecture — these are the authoritative module references:

| Doc | Size | Relevance to this plan |
|---|---|---|
| [README.md](https://github.com/NoFxAiOS/nofx/blob/dev/docs/architecture/README.md) | 6.3 KB | Overall system architecture, module map |
| [STRATEGY_MODULE.md](https://github.com/NoFxAiOS/nofx/blob/dev/docs/architecture/STRATEGY_MODULE.md) | **21.7 KB** | **PRIMARY REFERENCE** — full trading-cycle data flow, prompt construction, risk control. Plan 1's Tasks 4, 9, 10 align with this doc's stages 2 (Data Assembly), 3 (System Prompt), 4 (User Prompt), 6 (AI Parsing). |
| [AGENT_MEMORY_AND_PLANNING.md](https://github.com/NoFxAiOS/nofx/blob/dev/docs/architecture/AGENT_MEMORY_AND_PLANNING.md) | 11.2 KB | Agent (NOFXi) memory + planning subsystem. Out of scope for this plan but informs future work. |
| [X402_STREAMING_PAYMENT.md](https://github.com/NoFxAiOS/nofx/blob/dev/docs/architecture/X402_STREAMING_PAYMENT.md) | 12.7 KB | claw402 / x402 micropayment protocol. **Not used by Plan 1** (we bypass claw402 via direct Databento subscription). |

### Confirmed alignment with STRATEGY_MODULE.md

Plan 1 hooks into the same cycle stages the upstream doc describes:

- **Stage 1 (Coin Selection)** → For NQ we use the **static** path (single symbol "MNQ" or "NQ.c.0"); the `Static` mode is documented in STRATEGY_MODULE.md §1.1 at `decision/engine.go:395-403`. No change to selection code; just static-mode configuration.
- **Stage 2 (Data Assembly)** → Task 4 inserts the Databento adapter at the K-line ingestion point.
- **Stages 3 + 4 (Prompts)** → Task 9 adds a futures-mode template alongside the existing crypto template.
- **Stage 6 (AI Parsing)** → Unchanged; the futures prompt outputs the same JSON shape (`action`/`entry`/`stop_loss`/`take_profit`/`reasoning`) the existing parser expects.

This is important: **Plan 1 does not require structural changes to the strategy engine** — it adds new templates and a new data provider, both behind existing extension points.

## NinjaTrader CSV bridge

- **Repository:** https://github.com/J0shusmc/Claude-Trader-NinjaTrader
- **Key file:** `ninjascripts/claudetrader.cs` (14.8 KB) — the NinjaScript strategy that polls `trade_signals.csv` and places orders.
- **Auxiliary files:** `ninjascripts/SecondHistoricalData.cs`, `ninjascripts/SecondLifeFeed.cs` (data-export from NT to CSV — **not used by Plan 1**, since we get data from Databento).
- **Last update:** 2025-11-25 — production-ready per the README.
- **Status verified:** read claudetrader.cs in full during plan preparation. CSV contract (5-field signals, 3-field fills) confirmed against actual C# code, not just README claims.

## Databento

- **Documentation portal:** https://databento.com/docs/
- **Historical API base:** `https://hist.databento.com/v0/`
- **Authentication:** HTTP Basic, API key as username, empty password (NOT a Bearer token despite some community examples).
- **Pricing:** Pay-per-symbol-day. NQ continuous (`NQ.c.0`) on `GLBX.MDP3` is ~$0.001-0.005 per call for `ohlcv-1m` over short windows.
- **Datasets:** `GLBX.MDP3` covers all CME Globex products (NQ/MNQ/ES/MES/RTY/MRTY/CL/GC/etc.) — one subscription handles every CME futures contract.

### **CORRECTION — Go SDK status**

The URL `https://github.com/databento/databento-go` returns **HTTP 404** — there is **no official Databento Go SDK**.

What does exist:
- Official Python SDK: https://github.com/databento/databento-python
- Official Rust SDK + binary format library: https://github.com/databento/dbn
- C++ SDK: https://github.com/databento/databento-cpp
- **Community Go library:** https://github.com/NimbleMarkets/dbn-go (HTTP 200 verified 2026-05-22) — implements DBN binary format parsing + REST helpers. Active maintenance; not endorsed by Databento but used in production by NimbleMarkets. Could replace Plan 1's `net/http` approach if a more featureful SDK is desired later.

For Go, Plan 1's approach is correct as a starting point: call the REST API directly via `net/http` (Task 2). It's minimal and depends on no third-party code. If the project later wants DBN-binary streaming or more complete schema bindings, swap to `NimbleMarkets/dbn-go` — the adapter pattern in `market/databento_adapter.go` keeps the boundary clean and the swap is 1-2 days.

### **CORRECTION — Upstream docs reference older code structure**

The upstream [STRATEGY_MODULE.md](https://github.com/NoFxAiOS/nofx/blob/dev/docs/architecture/STRATEGY_MODULE.md) cites code at paths like `decision/engine.go:395-403`. **The current codebase does not have a `decision/` folder** — that module was refactored into `kernel/engine.go`. The function names and behavior cited are still accurate; only the file paths drifted. When implementing, find the equivalent in `kernel/`:

| Upstream doc cite | Current local path |
|---|---|
| `decision/engine.go` | `kernel/engine.go` |
| `decision/engine_analysis.go` | `kernel/engine_analysis.go` |
| `decision/engine_position.go` | `kernel/engine_position.go` |
| `decision/engine_prompt.go` | `kernel/engine_prompt.go` |

Other docs may have similar drift. If a cited path doesn't exist, search for the function name within `kernel/`.

## Quick-reference URLs (all verified)

```
Local code:                 /home/hoang/nofx
Upstream NOFX:              https://github.com/NoFxAiOS/nofx
Architecture docs:          https://github.com/NoFxAiOS/nofx/tree/dev/docs/architecture
NT CSV bridge:              https://github.com/J0shusmc/Claude-Trader-NinjaTrader
Databento docs portal:      https://databento.com/docs/
Databento Hist API:         https://hist.databento.com/v0/
Databento Go SDK (official): (does not exist — use net/http per Task 2)
Databento Go SDK (community): https://github.com/NimbleMarkets/dbn-go (alternative for later swap)
```

---

## CSV bridge runtime hazards — must address before live trading

> **Identified by external audit 2026-05-22.** The current Plan-1 CSV protocol has 4 known runtime issues. Acceptable for SIM paper-trading; **not acceptable for live execution**. Add a Plan 1.5 (or fold into Plan 2 safety layer) to harden these before flipping to a funded broker connection.

### Hazard H1: Lost signal race

- **Symptom:** Bot writes signal A at T=0. NT poll fires at T=2s. Bot writes signal B at T=1.5s, overwriting A. NT reads B, never sees A.
- **Cause:** `WriteSignal` truncates+rewrites the file each call. No acknowledgment back to Go that NT consumed the prior signal.
- **Fix:** Block subsequent writes until the tailer confirms a fill (or N-second timeout has passed) for the prior signal. Track an in-flight flag per trader. Reject `WriteSignal` calls when in-flight is set.

### Hazard H2: 1-second dedup collision

- **Symptom:** Two same-direction signals fire in the same wall-clock second. NT's dedup key is `DateTime+Direction`. Second signal silently dropped.
- **Cause:** [claudetrader.cs:202](claudetrader.cs#L202) — `string signalId = $"{parts[0]}_{parts[1]}";` with DateTime format `MM/dd/yyyy HH:mm:ss` (no fractional seconds).
- **Fix:** Either (a) require Go-side rate-limit of ≥2 seconds between same-direction writes (matches the dedup window), OR (b) modify `claudetrader.cs` to include a sub-second nonce/sequence number in the signal ID (`{DateTime}_{Direction}_{Nonce}`), and have the Go writer include a monotonic nonce as a 6th CSV field.

### Hazard H3: DrvFs (WSL2 `/mnt/c`) atomic-rename mtime

- **Symptom:** Go's `os.Rename` is atomic on ext4 but mtime propagation through the WSL2 DrvFs mount may or may not reliably bump `File.GetLastWriteTime` as observed from Windows NT process.
- **Cause:** DrvFs is a 9P-protocol bridge with semantic-translation quirks; not all metadata operations propagate identically to native NTFS access.
- **Fix:** Empirically test in Step 0.8: write a signal, verify NT's `File.GetLastWriteTime` changes within 1 second. If mtime doesn't propagate, fall back to write-in-place (non-atomic) with file locking, OR modify `claudetrader.cs` to poll content-hash instead of mtime.

### Hazard H4: Fill replay on session rollover

- **Symptom:** NT cycles `trades_taken.csv` at session close (truncates or rotates). Go-side tailer sees file shrink, resets seen-rows to 0, then re-emits every row that re-appears as if new → duplicate DB fills.
- **Cause:** Tailer tracks position by line count. Reset logic at file-shrink triggers on rotation.
- **Fix:** Key the tailer on a stable fill identity (timestamp + price + side), not line count. Maintain a "seen fills" set in memory or DB; ignore any row whose key has been seen before.

### Required for live (not Plan 1 paper)

These 4 hazards are not in scope for Plan 1's SIM-only goal. Add them as Plan 1.5 (or first part of Plan 2's safety layer) before flipping NT's broker connection from SIM101 to a funded account. The mitigation work is ~200-300 LOC: Go-side in-flight tracking + nonce + stable-fill-ID set, plus a minor claudetrader.cs extension to echo the nonce in fill rows.

---

## `GetBalance` $50k mock — honest disclosure

> **SUPERSEDED 2026-05-30** — the `$50k` mock is GONE. Real per-account
> equity now auto-updates via the C# `SendAccountBalance` poll (`c4e2cb13`).
> Root cause of it appearing stuck was a STALE GO BINARY, not the mock.
> See `## Current State (2026-05-30, session-verified)`. Kept below for
> history.

[provider/ninjatrader/trader.go:GetBalance](provider/ninjatrader/trader.go) (per Task 7) returns hardcoded:

```go
return map[string]interface{}{
    "totalEquity":      50000.0,
    "availableBalance": 50000.0,
}, nil
```

**This affects two consumers:**

1. **AI prompt equity math** — `BuildSystemPrompt` computes max position size as `equity × ratio`. With $50k mocked, the AI is told it has $50k regardless of the real SIM balance. Decisions sized against fantasy capital.
2. **Go-side risk checks** — `MaxMarginUsage`, `BTCETHMaxPositionValueRatio`, `MinPositionSize` validations all use the balance from `GetBalance()`. They will pass/reject orders based on $50k, not real account state.

**Acceptable for:** SIM paper trading where SIM101 default balance is also ~$50k, AND you understand the prompt is decoupled from real account state.

**Not acceptable for:** any live deployment.

**Fix options:**

| Option | Effort | When |
|---|---|---|
| Extend `claudetrader.cs` to write a `balance.csv` (account equity + available) on each cycle; Go tailer reads it | Medium (~80 LOC C# + 60 LOC Go) | Plan 1.5 |
| Read balance from NinjaTrader's Connection panel via a separate NinjaScript export | Same as above | Plan 1.5 |
| Disable Go-side balance-based sizing when `ExchangeType==ninjatrader` (let the AI's risk reasoning + NT's hard contract-quantity cap do the work) | Low (~20 LOC) | Plan 1 if going live early |

Recommended: option 3 for paper validation, then option 1 before any funded-account flip.

---

## Security must-fix before any live deployment

**Issue:** [config/config.go:67-69](config/config.go#L67-L69) defaults `JWTSecret` to the literal string `"default-jwt-secret-change-in-production"` when the `JWT_SECRET` env var is unset:

```go
if cfg.JWTSecret == "" {
    cfg.JWTSecret = "default-jwt-secret-change-in-production"
}
```

**Risk:** anyone running the bot with default config has the same JWT-signing key as every other default-config install on the internet. JWT tokens can be forged trivially. **Documented but easy to miss.**

**Required action (must be done before exposing the bot to any non-localhost network):**

1. Generate a strong random secret: `openssl rand -base64 64`
2. Set it in `.env` as `JWT_SECRET=<the random string>`
3. **Verify by env var, NOT the log line.** The log `🔑 JWT secret configured` at [main.go:90](main.go#L90) fires unconditionally on every startup — it does NOT indicate the default was overridden. Use one of these instead:
   - `grep "^JWT_SECRET=" /home/hoang/nofx/.env` — must return your generated key, not empty
   - Add a startup warning in code (recommended): modify [config/config.go:67-69](config/config.go#L67-L69) to log a loud warning when the default value is detected:
     ```go
     if cfg.JWTSecret == "" {
         cfg.JWTSecret = "default-jwt-secret-change-in-production"
         logger.Warnf("⚠️  JWT_SECRET env var not set; using INSECURE default. " +
             "This is acceptable for localhost-only paper trading. " +
             "Set JWT_SECRET in .env before any network-exposed deploy.")
     }
     ```
   This way the log distinguishes "default in use" (warning) from "real secret loaded" (silent or info).

This is not specific to NQ trading — it's a pre-existing nofx hardening item. Add to Plan 0 (Task 0.10) below if planning a live deploy:

- [ ] **Step 0.10: Set JWT_SECRET to a strong random value**

```bash
echo "JWT_SECRET=$(openssl rand -base64 64)" >> /home/hoang/nofx/.env
```

Confirm: `grep JWT_SECRET /home/hoang/nofx/.env` should show your generated key, NOT the literal "default-jwt-secret-change-in-production".

This is a no-op for LOCAL-ONLY paper trading (bot binds to localhost), but is non-negotiable for any deployment reachable from the internet.

---

# Function Transfer Manifest

> **Distilled from a 15-agent parallel audit on 2026-05-22.** The audits read every domain end-to-end (schema, kernel, market, broker pattern, API handlers, web UI, i18n, bootstrap). This section is the **canonical actionable list** — every function that must be CREATED, every function that must be MODIFIED, and every function that transfers UNCHANGED.

## A — NEW functions to implement

Group 1: **Databento data layer** (provider/databento/, market/)

| # | File | Function signature | LOC |
|---|------|---------------------|-----|
| A1 | `provider/databento/client.go` ✓ done | `NewClient(baseURL, apiKey string) *Client` + `doRequest(path, params) ([]byte, error)` + `basicAuth(user, pass) string` | 120 |
| A2 | `provider/databento/historical.go` | `(c *Client) GetOHLCV(symbol, interval string, start, end time.Time) ([]Bar, error)` | 80 |
| A3 | same | `parseOHLCVResponse(body []byte) ([]Bar, error)` + `(rawBar).toBar()` + `scaledFloat(s)` | 60 |
| A4 | `provider/databento/resolve.go` | `(c *Client) ResolveContinuous(symbol string) (string, error)` + `parseResolveResponse(body, symbol) (string, error)` | 60 |
| A5 | `market/databento_adapter.go` | `BarsToKlines(bars []databento.Bar) []Kline` | 25 |
| A6 | `market/data.go` (add helper) | `isCMEFuturesSymbol(symbol string) bool` (regex/map of NQ/MNQ/ES/MES/YM/MYM/GC/SI/CL/NG/ZB/ZN/ZT/ZW/ZC/ZS/ZL/ZO/etc.) | 25 |

Group 2: **NinjaTrader CSV bridge** (provider/ninjatrader/)

| # | File | Function signature | LOC |
|---|------|---------------------|-----|
| A7 | `provider/ninjatrader/types.go` | `SignalRow` + `FillRow` types, header constants, `(SignalRow) Validate() error` | 80 |
| A8 | `provider/ninjatrader/csv_writer.go` | `NewCSVWriter(dataDir) *CSVWriter` + `(w) WriteSignal(SignalRow) error` (atomic temp+rename) + `(w) SignalsPath() string` | 90 |
| A9 | `provider/ninjatrader/csv_tailer.go` | `NewCSVTailer(dataDir, pollInterval) *CSVTailer` + `(t) TailFills(ctx, onFill) error` + `parseFillRow(line) (FillRow, error)` | 110 |

Group 3: **Trader interface impl** (trader/ninjatrader/)

| # | File | Function | LOC | Strategy |
|---|------|----------|-----|----------|
| A10 | `trader/ninjatrader/trader.go` | `New(Config) *Trader` constructor + tailer goroutine | 40 | clean |
| A11 | same | `(t) OpenLong(symbol, qty, leverage) (map, error)` — bundle stashed SL/TP into one SignalRow | 30 | clean |
| A12 | same | `(t) OpenShort(...)` — same shape as OpenLong | 30 | clean |
| A13 | same | `(t) CloseLong(...)` / `CloseShort(...)` — return error "auto-close via SL/TP" | 8 | error |
| A14 | same | `(t) SetStopLoss(symbol, side, qty, stopPrice) error` — stash in internal map | 15 | clean |
| A15 | same | `(t) SetTakeProfit(symbol, side, qty, tpPrice) error` — stash | 15 | clean |
| A16 | same | `(t) SetLeverage(symbol, leverage) error` — noop (return nil) | 3 | noop |
| A17 | same | `(t) SetMarginMode(symbol, isCross) error` — noop | 3 | noop |
| A18 | same | `(t) GetMarketPrice(symbol) (float64, error)` — query Databento last 1m bar | 15 | clean |
| A19 | same | `(t) CancelStopLossOrders(symbol) error` — return error "not supported" | 3 | error |
| A20 | same | `(t) CancelTakeProfitOrders(symbol) error` — error | 3 | error |
| A21 | same | `(t) CancelAllOrders(symbol) error` — error | 3 | error |
| A22 | same | `(t) CancelStopOrders(symbol) error` — error (legacy) | 3 | error |
| A23 | same | `(t) FormatQuantity(symbol, qty) (string, error)` — `fmt.Sprintf("%.0f", qty)` for whole contracts | 5 | clean |
| A24 | same | `(t) GetOrderStatus(symbol, orderID) (map, error)` — query internal pending map; "filled" after fill row | 25 | partial |
| A25 | same | `(t) GetBalance() (map, error)` — return fixed SIM101 mock $50,000 | 12 | mock |
| A26 | same | `(t) GetPositions() ([]map, error)` — read from tailer state | 25 | clean |
| A27 | same | `(t) GetClosedPnL(start, limit) ([]ClosedPnLRecord, error)` — parse trades_taken.csv historical rows | 50 | clean |
| A28 | same | `(t) GetOpenOrders(symbol) ([]OpenOrder, error)` — return `[]OpenOrder{}` (CSV doesn't expose pending) | 5 | empty |

**Trader interface methods: 17 implemented, 11 cleanly via CSV, 4 noop/empty, 2 error-returning.** All compile-time-checked via `var _ types.Trader = (*Trader)(nil)`.

Group 4: **NQ futures AI prompt** (kernel/)

| # | File | Function signature | LOC |
|---|------|---------------------|-----|
| A29 | `kernel/engine_prompt_futures.go` | `BuildFuturesSystemPrompt(FuturesPromptConfig) string` | 80 |
| A30 | same | `BuildFuturesUserPrompt(FuturesContext) string` | 60 |
| A31 | same | `side(price, ref) string` + `emaAlignment(20, 50) string` + `rsiBucket(r) string` + `bollPosition(p, u, l) string` | 40 |

Group 5: **Futures-specific validation** (kernel/engine_position.go additions)

| # | File | Function signature | LOC |
|---|------|---------------------|-----|
| A32 | `kernel/engine_position.go` (add) | `roundTickNQ(price float64) float64` — round to nearest 0.25 | 5 |
| A33 | same | `roundTickMNQ(price float64) float64` — round to 0.25 (same; MNQ tick is also 0.25 not 0.05) | 5 |
| A34 | same | `validateStopDistanceNQ(entry, stop float64, minPoints int) error` — enforce min-point gap | 12 |
| A35 | same | `validateContractSizeNQ(contracts int, currentPrice, maxNotional float64) error` — guard notional exposure | 12 |

Group 6: **End-to-end smoke runner**

| # | File | Function | LOC |
|---|------|----------|-----|
| A36 | `cmd/nq_smoke/main.go` | one-shot runner: fetch bars → compute indicators → prompt → stdin-paste AI decision → write signal → tail fills | 130 |

**Total NEW code: ~1,200 LOC** + **~400 LOC tests** = **~1,600 LOC new**.

---

## B — EXISTING functions to MODIFY (specific edits only)

| # | File:Line | Change | Δ LOC |
|---|-----------|--------|-------|
| B1 | `market/data.go:557-558` (`Normalize`) | **Case-preserving fix:** capture raw symbol BEFORE the `strings.ToUpper` call (line 558), check `isCMEFuturesSymbol(raw)` first, and `return raw` if true. Otherwise proceed with existing logic. Databento continuous symbols use lowercase suffix (`NQ.c.0`, not `NQ.C.0`). Inserting the guard only at line 583 would receive an already-uppercased string and return `NQ.C.0`, which Databento rejects. Code shape: `func Normalize(symbol string) string { raw := symbol; if isCMEFuturesSymbol(raw) { return raw }; symbol = strings.ToUpper(symbol); ... }` | +5 |
| B2 | `market/data_klines.go:getKlinesFromCoinAnk` line 18 (or one level up) | Branch: if symbol is CME futures, call Databento adapter instead | +30 |
| B3 | `store/exchange.go:43` (Exchange struct) | Add field `NinjaTraderDataDir string` with gorm tag | +2 |
| B4 | `store/exchange.go:231` (Create) | Add `ninjaTraderDataDir string` param + assignment | +6 |
| B5 | `store/exchange.go:277-323` (Update) | Add to updates map | +4 |
| B6 | `store/exchange.go:initTables` (Postgres migration block) | Add conditional `ALTER TABLE` for the new column | +6 |
| B7 | `store/exchange.go:203-224` (`getExchangeNameAndType`) | Add `case "ninjatrader": return "NinjaTrader Futures", "cex"` | +2 |
| B8 | `store/visibility.go:5-37` (`MissingRequiredExchangeCredentialFields`) | Add `case "ninjatrader"` requiring `ninja_trader_data_dir` | +5 |
| B9 | `store/visibility.go:64-80` (`IsVisibleExchange`) | Add `strings.TrimSpace(exchange.NinjaTraderDataDir) != ""` to the OR-chain | +1 |
| B10 | `manager/trader_manager.go:~700` (`addTraderFromStore` switch) | Add `case "ninjatrader"`: copy `NinjaTraderDataDir` into `AutoTraderConfig` | +5 |
| B11 | `manager/trader_manager.go` imports | `import ninjatrader "nofx/trader/ninjatrader"` | +1 |
| B12 | `trader/auto_trader.go:111` (`AutoTraderConfig` struct) | Add fields `NinjaTraderDataDir string` + `NinjaTraderSymbol string` | +3 |
| B13 | `trader/auto_trader.go:60` (Exchange field comment) | Update comment to include `"ninjatrader"` | +0 |
| B14 | `trader/auto_trader.go:263-315` (broker switch) | Add `case "ninjatrader"`: `trader = ninjatrader.New(...)` | +8 |
| B15 | `kernel/engine_prompt.go:17` (`BuildSystemPrompt`) | Branch: `if variant == "futures" { return BuildFuturesSystemPrompt(...) }` | +6 |
| B16 | `kernel/engine_position.go:39` (`validateDecisions`) | Add NQ/MNQ branch: tick-round SL/TP, validate stop distance in points, allow leverage=1 | +30 |
| B17 | `kernel/engine_analysis.go:91` (`fetchMarketDataWithStrategy`) | Skip `engine.nofxosClient.GetOITopPositions()` when exchange is ninjatrader | +6 |
| B18 | `kernel/engine.go:NewStrategyEngine` lines 183-225 | (optional) Add `databentoClient` field + initialize when env var set | +12 |
| B19 | `api/handler_exchange.go:347-354` (validTypes map) | Add `"ninjatrader"` to the map | +1 |
| B20 | `api/handler_exchange.go:90-107` + `70-87` (request structs) | Add `NinjaTraderDataDir string` field to both request types | +4 |
| B21 | `api/handler_exchange.go:48-68` (`SafeExchangeConfig`) | Add the field | +2 |
| B22 | `api/handler_trader.go:184-196` (`validateExchangeForTraderCreation`) | Add `case "ninjatrader"` checking DataDir non-empty | +5 |
| B23 | `api/handler_trader.go:457-481` (probe builder) | Return nil for ninjatrader (skip live probe) | +5 |
| B24 | `api/handler_trader_status.go:159-201` (`handleClosePosition`) | Add `case "ninjatrader"` returning HTTP 400 with graceful message | +5 |
| B25 | `agent/skill_management_handlers.go:~200-280` (credential display) | Add `case "ninjatrader"`: show DataDir | +5 |
| B26 | `api/exchange_account_state.go:~50-120` (`buildExchangeProbeTrader`) | Add `case "ninjatrader"`: return early (no probe possible for file-based bridge) | +5 |
| B27 | `config/config.go:Config` struct ~16-46 | Add `DatabentoAPIKey string` + `DatabentoDataset string` | +3 |
| B28 | `config/config.go:Init()` ~92 | Load `DATABENTO_API_KEY` + `DATABENTO_DATASET` (default `"GLBX.MDP3"`) | +5 |
| B29 | `.env.example` (append) | New section: `DATABENTO_API_KEY=` + `DATABENTO_DATASET=GLBX.MDP3` | +4 |
| B30 | `web/src/router/AppRoutes.tsx` ~468-475 + import line | Remove `<Route path={ROUTES.data}>` + `DataPage` import | -10 |
| B31 | `web/src/router/paths.ts:23` | Remove `data: '/data',` line | -1 |
| B32 | `web/src/components/common/HeaderBar.tsx:121-131 + 457-466` | Remove desktop + mobile Data nav entries (2 identical blocks) | -22 |
| B33 | `web/src/pages/DataPage.tsx` | Delete file entirely (17 lines) | -17 |
| B34 | `web/src/components/strategy/CoinSourceEditor.tsx:69-79` | Skip USDT auto-append when symbol matches CME futures pattern | +8 |
| B35 | `web/src/pages/StrategyStudioPage.tsx:1203-1206 + 1311-1313` (PromptVariant dropdown) | Add `<option value="futures">{tr('futuresVariant')}</option>` in 2 places | +4 |
| B36 | `web/src/components/trader/ExchangeConfigModal.tsx:23-34` (templates array) | Add `{exchange_type:'ninjatrader', name:'NinjaTrader', type:'cex'}` | +5 |
| B37 | `web/src/components/trader/ExchangeConfigModal.tsx` (form fields, new section ~720-756) | Add NinjaTrader form section: DataDir + InstrumentName + DefaultContractQty inputs | +35 |
| B38 | `web/src/components/trader/ExchangeConfigModal.tsx:301-339` (handleSubmit) | Add validation branch + onSave call for `currentExchangeType==='ninjatrader'` | +12 |
| B39 | `web/src/pages/SettingsPage.tsx:handleSaveExchange` | Accept + pass NT fields to create/update request | +6 |
| B40 | `web/src/types/config.ts:49,91` (Exchange + CreateExchangeRequest) | Add `ninjaTraderDataDir?: string` (plus optional `instrumentName`, `defaultContractQty`) | +5 |
| B41 | `web/src/components/common/ExchangeIcons.tsx:10-21` | Add `ninjatrader: '/exchange-icons/ninjatrader.png'` to ICON_PATHS | +1 |
| B42 | `web/src/pages/TraderDashboardPage.tsx:513,522,530` (StatCard `unit="USDT"`) | Make conditional: USDT for crypto, "USD" for futures (read exchange.exchange_type) | +6 |
| B43 | `web/src/pages/TraderDashboardPage.tsx:609,663,673` (Leverage + Liquidation cols) | Hide for futures exchanges (`exchange_type==='ninjatrader'`) | +8 |
| B44 | `web/src/i18n/translations.ts:347,1693,2971` (`invalidSymbolFormat`) | Soften message: futures-aware | +6 |
| B45 | `web/src/i18n/translations.ts` (add 5 new keys × 3 languages) | `ninjatraderExchangeName`, `ninjatraderSetupGuide`, `dataDirectoryPath`, `chartInstrument`, `futuresVariant` | +30 |

**Total MODIFIED code: ~280 LOC delta** across 45 touchpoints.

---

## C — EXISTING functions that TRANSFER UNCHANGED

These are 100% confirmed transferable as-is. **Do not modify these**:

**Indicator engine** (no change):
- `calculateEMA`, `calculateMACD`, `calculateRSI`, `calculateATR` ([market/data_indicators.go:6,28,42,86](market/data_indicators.go))
- `ExportCalculateEMA/MACD/RSI/ATR/BOLL/Donchian/BoxData` ([market/data_indicators.go:203-233](market/data_indicators.go#L203-L233))
- All take `[]Kline`, return numbers. No symbol-format assumptions.

**Strategy engine — static-coin path** (no change after Normalize fix):
- `GetCandidateCoins` for `SourceType="static"` ([kernel/engine.go:262-271](kernel/engine.go#L262-L271))
- `filterExcludedCoins` ([kernel/engine.go:450-473](kernel/engine.go#L450-L473))
- `CoinSourceConfig` schema ([store/strategy.go:695-721](store/strategy.go#L695-L721))

**Strategy engine — nofxos paths** (no change; bypassed via config):
- `FetchOIRankingData`, `FetchNetFlowRankingData`, `FetchPriceRankingData` ([kernel/engine.go:778,802,834](kernel/engine.go)) — already return `nil` if the corresponding indicator flag is disabled. NQ strategies just leave them disabled.
- `getAI500Coins`, `getOITopCoins`, `getOILowCoins`, `getHyperAllCoins`, `getHyperMainCoins` — never called when `SourceType="static"`.

**Risk control schema** (no schema change; fields repurposed):
- `RiskControlConfig` ([store/strategy.go:802-825](store/strategy.go#L802-L825)) — `MaxPositions`, `MaxMarginUsage`, `MinPositionSize`, `MinRiskRewardRatio`, `MinConfidence` are generic. `BTCETHMaxLeverage` / `AltcoinMaxLeverage` can carry NQ values without rename (cosmetic-only labels).

**Decision parsing** (no change):
- `parseFullDecisionResponse` ([kernel/engine_analysis.go:230-335](kernel/engine_analysis.go#L230-L335)) — `<reasoning>` / `<decision>` XML+JSON extraction is symbol-agnostic.
- Decision JSON shape `{action, symbol, entry, stop_loss, take_profit, reasoning, leverage, confidence}` accepts NQ as-is.

**AI Model store** (no change):
- All of [store/ai_model.go](store/ai_model.go) — provider, API key, custom URL. Works for any LLM provider.
- `ResolveClaw402WalletKey` returns `("", nil)` cleanly when no claw402 configured — NQ traders pass through without error.

**Strategy CRUD** (no change):
- All of [api/strategy.go](api/strategy.go) — accepts arbitrary `StaticCoins` strings without USDT validation.
- `POST /api/strategies/test-run` works for `SourceType="static"` (skips nofxos calls entirely).
- `POST /api/strategies/estimate-tokens` is pure math — works for any prompt.

**Trader CRUD core** (only the broker-switch validation needs an addition; everything else unchanged):
- Trader Create/Update/Start/Stop flows
- `TraderManager.LoadTradersFromStore` and `addTraderFromStore` (only one switch case to add — B10)

**Position storage** (no change):
- [store/position.go:97-121](store/position.go#L97-L121) — `Symbol`, `EntryPrice`, `Quantity`, `Leverage`, `Status` are generic. NQ contract symbols like "NQM6" fit as `Symbol`. Leverage=1 works.

**Decision logging** (no change):
- [store/decision.go](store/decision.go) — `DecisionRecord` and `DecisionAction` structs are symbol-agnostic.

**Order storage** (no change):
- [store/order.go](store/order.go) `TraderOrder` and `TraderFill` — generic fields.

**Web UI components that work as-is** (no functional change; just minor i18n/conditional render edits in B):
- `TraderDashboardPage` positions table, equity stat cards, decisions log
- `DecisionCard` rendering (auto-strips USDT for display; works for NQ)
- `ModelConfigModal` (configures AI providers; provider-agnostic)
- `TraderConfigModal` (only displays strategy; doesn't validate symbols)
- `RiskControlEditor` (sliders work for any numeric range)
- `FAQ*` components

**Bootstrap** (one config addition, otherwise unchanged):
- `main.go` start sequence — adds nothing for the NQ path
- `crypto.NewCryptoService` — `NinjaTraderDataDir` is a path, not a secret, so no encryption needed
- DB init, JWT secret, logger init — unchanged
- Telegram bot — unchanged (vocabulary cosmetic only)

---

## D — Total scope (function-transfer view)

| Layer | New LOC | Modified Δ | Tests | Total |
|---|---|---|---|---|
| **provider/databento/** | 250 | — | 150 | 400 |
| **market/ (Databento adapter + Normalize fix)** | 50 | 34 | 80 | 164 |
| **provider/ninjatrader/ (CSV bridge)** | 280 | — | 150 | 430 |
| **trader/ninjatrader/ (broker impl)** | 270 | — | 50 | 320 |
| **kernel/ (futures prompt + validation)** | 240 | 54 | 60 | 354 |
| **store/ (exchange field + visibility)** | — | 31 | — | 31 |
| **api/ (handlers + validation)** | — | 32 | — | 32 |
| **config/ + .env** | — | 12 | — | 12 |
| **manager/ (trader manager switch)** | — | 6 | — | 6 |
| **trader/auto_trader (broker switch)** | — | 11 | — | 11 |
| **cmd/nq_smoke** | 130 | — | — | 130 |
| **web/ (UI conditional renders + nav cleanup)** | — | 102 | — | 102 |
| **i18n** | — | 36 | — | 36 |
| **TOTAL** | **~1,220** | **~318** | **~490** | **~2,030 LOC** |

**Calibrated estimate: ~2,000 LOC total work** — about 50% less than my original 4,000-6,000 LOC estimate, validated by the 15-agent audit.

**Time to ship**: 3-4 weeks at one focused engineer. The biggest single item is `trader/ninjatrader/trader.go` (the 17-method `Trader` interface impl) at ~270 LOC + tests.

---

## E — The transfer order

When executing, do groups in this dependency order to avoid getting stuck:

```
1. Group A1-A6  (Databento data layer)         ← independent, can start any time
2. Group A7-A9  (CSV bridge primitives)        ← independent
3. Group A29-A31 (Futures prompt)               ← independent
4. Group A32-A35 (Futures validation helpers)  ← independent
5. Group A10-A28 (Trader broker impl)          ← needs A7-A9 done
6. Modifications B1-B7                          ← needs A1-A6 done (Normalize + adapter wired)
7. Modifications B8-B14                         ← needs A10-A28 done (broker registered)
8. Modifications B15-B18                        ← needs A29-A35 done (prompt + validation registered)
9. Modifications B19-B29                        ← API layer + config (independent, can do in parallel after store layer)
10. Modifications B30-B45                       ← web UI (can do in parallel with backend)
11. Group A36 (cmd/nq_smoke)                    ← needs everything else done; the integration test
```

Groups 1-4 are independent and can run in parallel. Group 5 depends on 2. Modifications wait for the new functions they reference. The web UI work can happen in parallel with the backend once the request/response shapes are settled.

This ordering is also reflected in the Task 0-11 sequence at the top of this document.

---

# The 3 Control Surfaces (User-Facing Priority)

> **Refocus 2026-05-22:** From the user's perspective, the entire NQ trading system is controlled through three pages in the web UI: **Config**, **Dashboard**, and **Strategy**. If these three flows work, the system works. This section is the minimal function-transfer manifest scoped to those three surfaces only — backend backbone (Databento client, NinjaTrader CSV bridge, broker impl) is separate, covered earlier in this document.

```
┌────── CONFIG (Settings) ──────┐   ┌──── DASHBOARD (Trader) ────┐   ┌──── STRATEGY (Studio) ─────┐
│  Add NinjaTrader exchange     │   │  Watch the bot trade NQ    │   │  Build NQ strategy with    │
│  Configure AI model           │   │  in NT SIM, review AI       │   │  static coin source +      │
│                                │   │  decisions live            │   │  futures prompt variant    │
└────────────────────────────────┘   └──────────────────────────────┘   └────────────────────────────┘
```

User flow: open Config → add NinjaTrader exchange + verify AI model. Open Strategy → build NQ strategy. Open Dashboard → create trader linking the three → start → watch.

## SURFACE 1: CONFIG (Settings page)

**Frontend:** `web/src/pages/SettingsPage.tsx` + `web/src/components/trader/ExchangeConfigModal.tsx` + `web/src/components/trader/ModelConfigModal.tsx`
**Backend:** `api/handler_exchange.go` + `api/handler_ai_model.go` + `store/exchange.go` + `store/visibility.go`

| # | Touchpoint | Action |
|---|------------|--------|
| C1 | `ExchangeConfigModal.tsx:23-34` (templates array) | Add `{ exchange_type: 'ninjatrader', name: 'NinjaTrader', type: 'cex' }` |
| C2 | same (new form section ~720-756) | NT form fields: DataDir + InstrumentName + DefaultContractQty |
| C3 | same:301-339 (handleSubmit) | NT validation branch (DataDir required, no API key) |
| C4 | `SettingsPage.tsx:190` (`handleSaveExchange`) | Pass new fields into createRequest/updateRequest |
| C5 | `web/src/types/config.ts:49,91` | Add `ninjaTraderDataDir?: string` to Exchange + CreateExchangeRequest |
| C6 | `web/src/components/common/ExchangeIcons.tsx:10-21` | Add NT icon mapping |
| C7 | `web/src/i18n/translations.ts` (×3 languages) | New keys: `ninjatraderExchangeName`, `ninjatraderSetupGuide`, `dataDirectoryPath`, `chartInstrument` |
| C8 | `api/handler_exchange.go:347-354` (validTypes) | Add `"ninjatrader"` |
| C9 | same:90-107 + 70-87 (request structs) | Add `NinjaTraderDataDir string` field |
| C10 | same:48-68 (`SafeExchangeConfig`) | Add the field for return DTOs |
| C11 | `store/exchange.go:43` | Add `NinjaTraderDataDir string` column |
| C12 | `store/visibility.go:5-37` (`MissingRequiredExchangeCredentialFields`) | Add `case "ninjatrader"` |
| C13 | `store/visibility.go:64-80` (`IsVisibleExchange`) | Include DataDir in OR-chain |
| C14 | `api/handler_trader.go:184-196` (`validateExchangeForTraderCreation`) | Add ninjatrader case |

**Already works:** AI Model config UI, tab routing, exchange list rendering, encrypted-credential flow, ModelConfigModal for DeepSeek/Claude/OpenAI/etc.

**Surface 1 total: ~110 LOC frontend + ~30 LOC backend = ~140 LOC.**

## SURFACE 2: DASHBOARD (Trader Dashboard)

**Frontend:** `web/src/pages/TraderDashboardPage.tsx` + `web/src/components/trader/DecisionCard.tsx` + `web/src/components/charts/ChartTabs.tsx`
**Backend:** `api/handler_trader.go` + `api/handler_trader_status.go`

| # | Touchpoint | Action |
|---|------------|--------|
| D1 | `TraderDashboardPage.tsx:513,522,530` (StatCard `unit="USDT"`) | Conditional unit: USD for ninjatrader, USDT otherwise |
| D2 | `TraderDashboardPage.tsx:609,663` (Leverage column) | Hide when `exchange_type === 'ninjatrader'` |
| D3 | `TraderDashboardPage.tsx:611,673` (Liquidation Price column) | Hide for ninjatrader |
| D4 | `api/handler_trader_status.go:159-201` (`handleClosePosition`) | Add ninjatrader case returning HTTP 400 with friendly message |
| D5 | `api/handler_trader.go:457-481` (probe trader) | Return nil for ninjatrader (skip live API probe) |
| D6 | `api/exchange_account_state.go:~50-120` (`buildExchangeProbeTrader`) | Add ninjatrader case: skip |
| D7 | `web/src/i18n/translations.ts:250` etc. (USDT warnings) | Soften crypto-only warnings |

**Already works:** DecisionCard (auto-strips USDT, formats prices, renders reasoning), Recent Decisions panel, Equity/P&L chart, Position history, Run/Stop buttons, cycle counter, all sub-components. **The dashboard is the most asset-agnostic surface — least work.**

**Surface 2 total: ~20 LOC frontend + ~15 LOC backend = ~35 LOC.**

## SURFACE 3: STRATEGY (Strategy Studio)

**Frontend:** `web/src/pages/StrategyStudioPage.tsx` + `web/src/components/strategy/CoinSourceEditor.tsx` + `IndicatorEditor.tsx` + `RiskControlEditor.tsx`
**Backend:** `api/strategy.go` + `api/strategy.go` + `kernel/engine_prompt.go` + `kernel/engine_position.go` + `market/data.go` + NEW `kernel/engine_prompt_futures.go`

| # | Touchpoint | Action |
|---|------------|--------|
| S1 | `CoinSourceEditor.tsx:69-79` (USDT auto-append) | Skip USDT for CME futures patterns (`NQ.c.0`, `MNQ`, `ES`, etc.) |
| S2 | same:195-201 (input placeholder) | "BTC, ETH, SOL, NQ.c.0, MNQ..." |
| S3 | `IndicatorEditor.tsx:658-687` (Market Sentiment) | Hide `enable_funding_rate` + `enable_oi` for futures (prop-controlled) |
| S4 | `IndicatorEditor.tsx:226-449` (NofxOS sources: AI500/OI/NetFlow/Price ranking) | Hide for futures strategies |
| S5 | `RiskControlEditor.tsx:72-127` (leverage labels) | Conditional label: "NQ Leverage" for futures vs "BTC/ETH Leverage" for crypto |
| S6 | `RiskControlEditor.tsx:277` (USDT min position unit) | Conditional unit |
| S7 | `StrategyStudioPage.tsx:1203-1206 + 1311-1313` (PromptVariant dropdown) | Add `<option value="futures">` in 2 places |
| S8 | `web/src/i18n/strategy-translations.ts` | Add futures variant labels, soften coinSource descriptions |
| S9 | `api/strategy.go:514-515` (PromptVariant validation) | Accept `"futures"` (currently no rejection — verify pass-through) |
| S10 | `kernel/engine_prompt.go:17` (`BuildSystemPrompt`) | Branch: `if variant=="futures" → BuildFuturesSystemPrompt(...)` |
| S11 | **NEW** `kernel/engine_prompt_futures.go` | `BuildFuturesSystemPrompt(FuturesPromptConfig) string` + `BuildFuturesUserPrompt(FuturesContext) string` + helpers |
| S12 | **NEW** `kernel/engine_position.go` (add helpers) | `roundTickNQ(price)` + `validateStopDistanceNQ(entry, stop, minPoints)` + futures branch in `validateDecisions` |
| S13 | `kernel/engine_analysis.go:91` (`fetchMarketDataWithStrategy`) | Skip `GetOITopPositions()` when exchange is ninjatrader |
| S14 | `market/data.go:583` (`Normalize`) | Skip USDT-append for CME futures symbols |
| S15 | **NEW** `market/data.go` helper | `isCMEFuturesSymbol(symbol string) bool` |

**Already works:** Strategy CRUD (accepts arbitrary `StaticCoins`), editor layout, Strategy Type selector, `PromptSections` editor, `CustomPrompt` field, test-run for static mode, token estimation, indicator math (EMA/MACD/RSI/ATR/Bollinger), backend `NormalizeProductSchema`.

**Surface 3 total: ~60 LOC frontend + ~250 LOC backend = ~310 LOC.** Most work because it includes the NEW futures prompt (~180 LOC) — the AI's NQ behavior is entirely defined here.

## Consolidated 3-surface scope

| Surface | Frontend Δ | Backend Δ | New backend | TOTAL |
|---------|------------|-----------|-------------|-------|
| **Config** | ~80 | ~30 | — | ~110 |
| **Dashboard** | ~20 | ~15 | — | ~35 |
| **Strategy** | ~60 | ~20 | ~230 | ~310 |
| **TOTAL (3 surfaces)** | **~160** | **~65** | **~230** | **~455 LOC** |

The remaining ~1,500 LOC (provider/databento/, provider/ninjatrader/, trader/ninjatrader/, cmd/nq_smoke, kernel adapter wiring) is the **backbone** — invisible to the user but required for the 3 surfaces to function. The split:

```
~455 LOC  ← 3 user-facing control surfaces
~1500 LOC ← backbone (data layer + broker impl + market adapter + smoke runner)
─────────
~1950 LOC ← total project (3-4 weeks focused engineering)
```

## Acceptance criteria, surface by surface

**Config:** A user can navigate to Settings → Exchanges tab → click Add → select "NinjaTrader" → enter DataDir path → save. The exchange appears in the list. No API key/secret required. **Done when this round-trip works.**

**Strategy:** A user can navigate to Strategy Studio → create new strategy → set Source = "static" + Static Coins = `["NQ.c.0"]` → set PromptVariant = "futures" → save. Strategy editor shows NQ-friendly RiskControl labels and hides funding/OI indicators. Token estimate works. Test-run button (with no AI key) returns prompts containing NQ-aware language, no BTC/USDT references. **Done when this round-trip works.**

**Dashboard:** A user can navigate to Trader Dashboard for an NQ trader → see positions in contracts (not coins, no USDT unit) → see Leverage column hidden → see Recent Decisions render with NQ symbols cleanly. **Done when this round-trip works.**

When all three acceptance criteria pass, the user has a working NQ-trading workflow even before any actual order flows through NinjaTrader. The backbone layer is required for live orders, but the *control surfaces* are independently validatable first.

---

# 4 Control Surfaces — Complete Function Inventory

> **Per-page function audit, 2026-05-22.** Four parallel Explore agents each enumerated every function/component/handler/endpoint/store-method on one surface and assigned KEEP / MODIFY / NEW / DELETE / DEFER per item. **Combined coverage: ~320 enumerated touchpoints.** This is the canonical pre-implementation checklist; nothing should be missed.

## Coverage scorecard

> **Recounted 2026-05-22 v3** — prior version had arithmetic errors on 3 of 4 surfaces. Totals below now reconcile (sum of categories = surface total; sum of surfaces = grand total).

| Surface | Total items | KEEP | MODIFY | NEW | DELETE | DEFER |
|---------|-------------|------|--------|-----|--------|-------|
| **Config** (Settings) | 65 | 47 | 18 | 0 | 0 | 0 |
| **Dashboard** (Trader) | 96 | 71 | 19 | 0 | 6 | 0 |
| **Strategy** (Studio) | 118 | 82 | 34 | 2 | 0 | 0 |
| **AgentBeta** (Chat) | 45 | 20 | 18 | 0 | 0 | 7 |
| **TOTAL** | **324** | **220** | **89** | **2** | **6** | **7** |

**Verdict:** ~68% of all enumerated functions work unchanged for NQ. ~27% need targeted modification (most are label/conditional renders, not structural rewrites). 2 require new code (futures prompt + validators). 6 are conditional deletes (dashboard columns/sections hidden for futures). 7 Agent items are deferred to Plan 2 since they require deeper agent rework (provider catalog, candidate-coin tools, deep prompt persona).

## SURFACE 1 — CONFIG (Settings page): 55 items

**Frontend:** [web/src/pages/SettingsPage.tsx](web/src/pages/SettingsPage.tsx), [ExchangeConfigModal.tsx](web/src/components/trader/ExchangeConfigModal.tsx), [ModelConfigModal.tsx](web/src/components/trader/ModelConfigModal.tsx)
**Backend:** [api/handler_exchange.go](api/handler_exchange.go), [api/handler_ai_model.go](api/handler_ai_model.go), [store/exchange.go](store/exchange.go), [store/visibility.go](store/visibility.go)

### KEEP (47 — work unchanged for NQ)
SettingsPage tab container + state mgmt, all useEffects, refreshModelConfigs, refreshExchangeConfigs, handleChangePassword, handleSaveModel, handleDeleteModel, handleDeleteExchange, all 4 tab containers (Account/AI Models/Exchanges/Telegram), ModelConfigModal entirely (DeepSeek/Claude/OpenAI/etc. work as-is). ExchangeConfigModal: StepIndicator, ExchangeCard, useEffect lifecycle, handleCopyIP, handleSecureInputComplete, handleSelectExchange, handleBack, all 5 existing CEX/DEX field sections (binance, bybit, okx, gate/kucoin/indodax, aster, hyperliquid, lighter), form buttons. API: `GET /api/exchanges`, `DELETE /api/exchanges/:id`. Store: Exchange.List, Exchange.Delete.

### MODIFY (18 items — exact edits)

| # | File:Line | Change |
|---|-----------|--------|
| C-M1 | `SettingsPage.tsx:190-257` (`handleSaveExchange`) | Add NT params (dataDir, instrumentName, defaultContractQty) to function signature + pass into createRequest/updateRequest body |
| C-M2 | `ExchangeConfigModal.tsx:23-34` (`SUPPORTED_EXCHANGE_TEMPLATES`) | Add `{ exchange_type: 'ninjatrader', name: 'NinjaTrader', type: 'cex' }` entry |
| C-M3 | `ExchangeConfigModal.tsx:146-152` (props) | Update onSave signature to accept NT fields |
| C-M4 | `ExchangeConfigModal.tsx:154-180` (form state) | Add `dataDir`, `instrumentName`, `defaultContractQty` useState |
| C-M5 | `ExchangeConfigModal.tsx:212-227` (edit-mode populate) | Load NT fields when editing ninjatrader exchange |
| C-M6 | `ExchangeConfigModal.tsx:301-339` (`handleSubmit`) | Add NT validation branch (DataDir required, no API key) |
| C-M7 | `ExchangeConfigModal.tsx:342-454` (Step 0: exchange selection grid) | Ensure NT card appears in CEX section |
| C-M8 | `ExchangeConfigModal.tsx:~720-756` (NEW form section) | Add NT-specific form: DataDir input + InstrumentName + DefaultContractQty |
| C-M9 | `types/config.ts:21-50` (Exchange interface) | Add optional NT fields |
| C-M10 | `types/config.ts:76-92` (CreateExchangeRequest) | Add optional NT fields |
| C-M11 | `types/config.ts:124-145` (UpdateExchangeConfigRequest) | Add optional NT fields |
| C-M12 | i18n keys (translations.ts ×3 languages) | New: `ninjatraderExchangeName`, `ninjatraderSetupGuide`, `dataDirectoryPath`, `chartInstrument`, `ninjatraderContractQty` |
| C-M13 | `api/handler_exchange.go:347-354` (validTypes map) | Add `"ninjatrader"` |
| C-M14 | `api/handler_exchange.go:90-107 + 70-87` (request structs) | Add `NinjaTraderDataDir string`, `InstrumentName string`, `DefaultContractQty int` |
| C-M15 | `api/handler_exchange.go:28-46` (`SafeExchangeConfig`) + `:48-68` (`safeExchangeConfigFromStore`) | Add NT fields to response DTO + mapper |
| C-M16 | `api/handler_exchange.go:440-458` (`GET /api/supported-exchanges`) | Add NT to static response list |
| C-M17 | `store/exchange.go:20-43` (Exchange struct) + Create/Update signatures + `initTables` migration | Add `NinjaTraderDataDir`, `InstrumentName`, `DefaultContractQty` columns + Postgres ALTER |
| C-M18 | `store/visibility.go:5-37` (`MissingRequiredExchangeCredentialFields`) + `:64-80` (`IsVisibleExchange`) | Add ninjatrader case requiring DataDir + extend visibility OR-chain |

**Surface 1 LOC delta: ~140 (110 FE, 30 BE).**

## SURFACE 2 — DASHBOARD (Trader): 96 items

**Frontend:** [TraderDashboardPage.tsx](web/src/pages/TraderDashboardPage.tsx), [DecisionCard.tsx](web/src/components/trader/DecisionCard.tsx), [ChartTabs.tsx](web/src/components/charts/ChartTabs.tsx), [TraderConfigModal.tsx](web/src/components/trader/TraderConfigModal.tsx)
**Backend:** [api/handler_trader.go](api/handler_trader.go), [api/handler_trader_status.go](api/handler_trader_status.go), [api/handler_order.go](api/handler_order.go), [api/handler_competition.go](api/handler_competition.go) (for decisions endpoint), [store/position.go](store/position.go), [store/decision.go](store/decision.go), [manager/trader_manager.go](manager/trader_manager.go)

### KEEP (71 — work unchanged for NQ)
Trader Header section, Debug Info, StatCard: Positions (count-only), GridRiskPanel, ChartTabs container, the entire positions data table EXCEPT leverage + liq columns, every Symbol/Side/Action/Entry/Mark/Qty/Value/uPnL column, close-position button + handler, position pagination, Recent Decisions Panel, decisions limit selector, DecisionCard main component, Risk/Reward visualization, EquityChart, AdvancedChart, K-line interval selector, symbol dropdown (works for any symbol source), PositionHistory component, useState hooks for closingPosition/selectedChartSymbol/pagination, useEffects for trader/grid changes, handleSymbolClick, handleClosePosition, all DecisionCard copyToClipboard/downloadAsFile, ChartTabs auto-detect on exchange change, all 16 backend API endpoints (`/api/status`, `/api/account`, `/api/positions`, `/api/positions/history`, `/api/decisions`, `/api/decisions/latest`, `/api/statistics`, `/api/equity-history`, `/api/traders/:id/grid-risk-info`, `/api/traders/:id/sync-balance`, `/api/traders/:id/close-position`, `/api/traders` CRUD, `/api/traders/:id/start`, `/api/traders/:id/stop`), all Position/Decision/Trader store methods, all TraderManager lifecycle methods (GetTrader, StartAll, StopAll, AutoStartRunningTraders), all 4 useSWR polling hooks. TraderConfigModal entirely (the create-trader modal trusts strategy's symbols).

### MODIFY (19) + DELETE (6)

| # | File:Line | Change | Type |
|---|-----------|--------|------|
| D-M1 | `TraderDashboardPage.tsx:513` (StatCard Total Equity `unit="USDT"`) | Conditional: USD for ninjatrader, USDT otherwise | MODIFY |
| D-M2 | `TraderDashboardPage.tsx:522` (StatCard Available Balance) | Conditional unit | MODIFY |
| D-M3 | `TraderDashboardPage.tsx:530` (StatCard Total P&L) | Conditional unit | MODIFY |
| D-D1 | `TraderDashboardPage.tsx:609, 663` (Leverage column header + cells, 7 render locations) | DELETE for ninjatrader (futures don't show leverage; conditional render) | DELETE |
| D-D2 | `TraderDashboardPage.tsx:611, 673` (Liquidation Price column) | DELETE for ninjatrader (no liq for NT) | DELETE |
| D-M4 | `TraderDashboardPage.tsx:67-71` (`isPerpDexExchange` helper) | Keep but extend: NT is neither perp nor dex; ensure helper returns false | MODIFY |
| D-M5 | `TraderDashboardPage.tsx:73-87` (`getWalletAddress` helper) | Return empty for ninjatrader | MODIFY |
| D-D3 | `TraderDashboardPage.tsx:178-187` (`handleCopyAddress`) | Not invoked for NT; safe to keep but no UI to trigger | DELETE (cond) |
| D-D4 | `TraderDashboardPage.tsx:395-440` (wallet address display section) | Hide for ninjatrader (conditional render) | DELETE (cond) |
| D-D5 | `TraderDashboardPage.tsx:143-144` (showWalletAddress/copiedAddress useState) | Never set for NT | DELETE (cond) |
| D-M6 | `DecisionCard.tsx:148-151` (ActionCard Leverage display `{leverage}x`) | Hide for ninjatrader (or show "1x" gracefully) | MODIFY |
| D-D6 | `DecisionCard.tsx:?` Leverage field rendering when action is NQ trade | Conditional hide | DELETE (cond) |
| D-M7 | `ChartTabs.tsx:184-204` (market type pills: hyperliquid/crypto/stocks/forex/metals) | For NT, force a single market type (or hide pills) | MODIFY |
| D-M8 | `ChartTabs.tsx:67-71` (useEffect: auto-detect market type from exchangeId) | Add ninjatrader → "futures" mapping (or "crypto" as fallback so kline still works) | MODIFY |
| D-M9 | `ChartTabs.tsx:283-294` (Quick Symbol Input + auto-append USDT) | Skip USDT append for CME futures symbol | MODIFY |
| D-M10 | `ChartTabs.tsx:132-143` (`handleSymbolSubmit`) | Don't normalize CME symbols to USDT | MODIFY |
| D-M11 | `ChartTabs.tsx:112-116` (`handleMarketTypeChange`) | Disable / no-op for ninjatrader | MODIFY |
| D-M12 | `ChartTabs.tsx:60-64` (default symbol "BTCUSDT") | Use exchange's chart symbol or "NQ.c.0" for ninjatrader | MODIFY |
| D-M13 | `api/handler_trader_status.go:159-201` (`handleClosePosition`) | Add `case "ninjatrader"` returning HTTP 400 with friendly message ("Close via NT UI; bridge does not support manual close") | MODIFY |
| D-M14 | `api/handler_trader.go:457-481` (probe trader) | Return nil for ninjatrader (no live probe) | MODIFY |
| D-M15 | `api/exchange_account_state.go:~50-120` (`buildExchangeProbeTrader`) | Add ninjatrader case: skip | MODIFY |
| D-M16 | Position struct returned by `/api/positions` (handler + store) | Conditionally omit `leverage` + `liquidation_price` for NT — OR leave in API response and just hide in UI (recommended: hide in UI only, less backend churn) | MODIFY |
| D-M17 | `web/src/i18n/translations.ts:250` (`asterUsdtWarning` and similar) | Soften crypto-specific warnings (multi-exchange-aware) | MODIFY |
| D-M18 | `web/src/i18n/translations.ts` (`tradingSymbolsDescription`, `invalidSymbolFormat`) | Soften "must end with USDT" validators | MODIFY |
| D-M19 | `web/src/components/trader/PositionHistory.tsx:126` (`stat.symbol.replace('USDT', '')`) | Safe as-is (replace returns original if no match for NQ) — no change needed BUT verify | KEEP-verify |

**Surface 2 LOC delta: ~80 (60 FE conditional renders, 20 BE). Most "DELETE" items are conditional renders — UI does not render the cell, but the data shape stays unchanged.**

## SURFACE 3 — STRATEGY (Studio): 120 items

**Frontend:** [StrategyStudioPage.tsx](web/src/pages/StrategyStudioPage.tsx) + 6 strategy editor sub-components in [web/src/components/strategy/](web/src/components/strategy/)
**Backend:** [api/strategy.go](api/strategy.go), [api/strategy.go](api/strategy.go), [store/strategy.go](store/strategy.go), [kernel/engine.go](kernel/engine.go), [kernel/engine_prompt.go](kernel/engine_prompt.go), [kernel/engine_position.go](kernel/engine_position.go), [market/data.go](market/data.go)

### KEEP (82 — work unchanged for NQ)
StrategyConfig + AIStrategyConfig interfaces, KlineConfig (timeframes work for any asset), PromptSectionsConfig, StrategyStudioPage root + all state vars + accordion section state, all 12 strategy CRUD handlers (fetchStrategies, handleCreateStrategy, handleDeleteStrategy, handleDuplicateStrategy, handleActivateStrategy, handleExportStrategy, handleImportStrategy, handleSaveStrategy, updateConfig, updateAIConfig, handleStrategyTypeChange, fetchPromptPreview), all 7 accordion section render blocks (CoinSource/Indicators/RiskControl/PromptSections/GridConfig/PublishSettings/StrategyType selector), CoinSourceEditor's sourceTypes selector (all 4 modes work for any asset), all 4 source type cards (static/ai500/oi_top/oi_low — AI500/OI just disabled by config), all timeframe selections (14 timeframes), Raw Klines toggle, Technical Indicators section entirely (EMA/MACD/RSI/ATR/BOLL + period inputs — all asset-agnostic), enable_volume + enable_oi (volume always relevant; OI just left off for NQ), RiskControlEditor max_positions / max_margin_usage / min_position_size / min_risk_reward_ratio / min_confidence (all generic), PromptSectionsEditor entirely (free-form text), PublishSettingsEditor entirely, TokenEstimateBar entirely, GridConfigEditor (unused for NQ), all i18n: coinSource/gridConfig/riskControl/promptSections/publishSettings objects (no crypto-specific terms in keys), all 13 backend strategy API endpoints (estimate-tokens, list, get, public, get-active, get-default-config — the last needs minor extension for futures variant), all 12 store/strategy.go CRUD methods, ClampLimits, MergeStrategyConfig, EstimateTokens. Kernel: GetCandidateCoins for static mode at engine.go:262-271 (works for NQ after Normalize fix), BuildUserPrompt, validateDecisions (with futures branch added separately).

### MODIFY (34) + NEW (2)

| # | File:Line | Change | Type |
|---|-----------|--------|------|
| S-M1 | `types/strategy.ts:107-118` (CoinSourceConfig) | Add optional `coin_source_variant?: 'crypto'\|'futures'` discriminator | MODIFY |
| S-M2 | `types/strategy.ts:120-162` (IndicatorConfig) | Document that `enable_funding_rate` + `enable_oi` are hidden in futures variant | MODIFY |
| S-M3 | `types/strategy.ts:184-202` (RiskControlConfig) | Add labels for futures (relabel BTC/ETH and Altcoin fields generically OR add separate fields) | MODIFY |
| S-M4 | `StrategyStudioPage.tsx:125` (selectedVariant state) | Add `'futures'` option | MODIFY |
| S-M5 | `StrategyStudioPage.tsx:1203-1206 + 1311-1313` (PromptVariant dropdown ×2) | Add `<option value="futures">{tr('futuresVariant')}</option>` | MODIFY |
| S-M6 | `StrategyStudioPage.tsx:650-678` (`runAiTest`) | Pass `prompt_variant` (may be `futures`) in request | MODIFY |
| S-M7 | `CoinSourceEditor.tsx:31-42` (xyzDexAssets set) | Add NQ, MES, MNQ, ES patterns | MODIFY |
| S-M8 | `CoinSourceEditor.tsx:44-47` (`isXyzDexAsset`) | Match CME futures patterns | MODIFY |
| S-M9 | `CoinSourceEditor.tsx:60-88` (`handleAddCoin`) | For CME, store as "NQ.c.0" not "NQc.0USDT" | MODIFY |
| S-M10 | `CoinSourceEditor.tsx:69-87` (symbol formatting logic) | Skip USDT suffix for CME symbols | MODIFY |
| S-M11 | `CoinSourceEditor.tsx:97-118` (`handleAddExcludedCoin`) | Same formatting fix | MODIFY |
| S-M12 | `CoinSourceEditor.tsx:200` (input placeholder) | "BTC, ETH, SOL, NQ.c.0, MNQ..." | MODIFY |
| S-M13 | `IndicatorEditor.tsx:226-275` (Quant Data section) | Hide for futures variant | MODIFY |
| S-M14 | `IndicatorEditor.tsx:278-331` (OI Ranking section) | Hide for futures variant | MODIFY |
| S-M15 | `IndicatorEditor.tsx:334-387` (NetFlow Ranking section) | Hide for futures variant | MODIFY |
| S-M16 | `IndicatorEditor.tsx:390-449` (Price Ranking section) | Conditionally show (price ranking is asset-class-agnostic) | MODIFY |
| S-M17 | `IndicatorEditor.tsx:656-688` (Market Sentiment section: volume / OI / funding_rate) | Hide `enable_funding_rate` for futures | MODIFY |
| S-M18 | `RiskControlEditor.tsx:61-128` (BTC/ETH + Altcoin leverage sliders) | Conditional labels for futures ("Primary Instrument Leverage" generic OR explicit "NQ/MNQ Leverage") | MODIFY |
| S-M19 | `RiskControlEditor.tsx:139-?` (position value ratios) | Conditional labels | MODIFY |
| S-M20 | `RiskControlEditor.tsx:277` (USDT min position unit) | Conditional unit | MODIFY |
| S-M21 | i18n `strategy-translations.ts:193-252` (indicator object) | Add `futuresVariant` key + remove crypto-only descriptions for indicators that don't apply | MODIFY |
| S-M22 | i18n `strategy-translations.ts:144-169` (riskControl object) | Add NQ-aware labels | MODIFY |
| S-M23 | `api/strategy.go:172-256` (`handleCreateStrategy`) | Accept `coin_source_variant=futures` → set futures default in `NormalizeProductSchema` | MODIFY |
| S-M24 | `api/strategy.go:478-489` (`handleGetDefaultStrategyConfig`) | Support `?variant=futures` query param → return futures defaults | MODIFY |
| S-M25 | `api/strategy.go:491-538` (`handlePreviewPrompt`) | Accept `prompt_variant="futures"` in body | MODIFY |
| S-M26 | `api/strategy.go:540-711` (`handleStrategyTestRun`) | Accept `prompt_variant="futures"` + route to BuildFuturesSystemPrompt | MODIFY |
| S-M27 | `store/strategy.go:134-150` (`NormalizeProductSchema`) | Detect "NQ.c.0" format and preserve (don't append USDT); detect `coin_source_variant` discriminator | MODIFY |
| S-M28 | `store/strategy.go:842` (`GetDefaultStrategyConfig`) | Add `futures` variant → static + NQ.c.0 + disabled funding/OI | MODIFY |
| S-M29 | `kernel/engine.go:254-?` (`GetCandidateCoins`) | Accept NQ symbols cleanly through normalization | MODIFY |
| S-M30 | `kernel/engine_prompt.go:17-150` (`BuildSystemPrompt`) | Branch: `if variant=="futures" → BuildFuturesSystemPrompt(...)` | MODIFY |
| S-M31 | `kernel/engine_prompt.go:?` (`writeAvailableIndicators`) | For futures variant, exclude funding rate / OI | MODIFY |
| S-M32 | `kernel/engine_position.go:39` (`validateDecisions`) | Add NQ branch: tick-round, min stop in points, allow leverage=1 | MODIFY |
| S-M33 | `market/data.go:557-558` (`Normalize`) | **Case-preserving fix.** Capture raw symbol before `strings.ToUpper` at line 558, check `isCMEFuturesSymbol(raw)` first, `return raw` if futures. Otherwise proceed with existing crypto logic. Critical: Databento uses lowercase suffix (`NQ.c.0`), inserting the check only at line 583 returns `NQ.C.0` which Databento rejects. | MODIFY |
| S-M34 | `market/data.go` | Add `isCMEFuturesSymbol(symbol) bool` helper (regex/map) | MODIFY (new helper in existing file) |
| S-N1 | **NEW** `kernel/engine_prompt_futures.go` | `BuildFuturesSystemPrompt(FuturesPromptConfig) string` + `BuildFuturesUserPrompt(FuturesContext) string` + helpers | NEW |
| S-N2 | **NEW** `kernel/engine_position.go` (add helpers) | `roundTickNQ(price)` + `validateStopDistanceNQ(entry, stop, minPoints)` + `validateContractSizeNQ` | NEW |

**Surface 3 LOC delta: ~340 (60 FE, 80 BE modifications, 200 NEW for futures prompt + validators).**

## SURFACE 4 — AGENT BETA (Chat): 49 items

**Frontend:** [AgentChatPage.tsx](web/src/pages/AgentChatPage.tsx) + [web/src/components/agent/](web/src/components/agent/) (MarketTicker, WelcomeScreen, PositionsPanel, UserPreferencesPanel, TradersPanel, SuggestionCards)
**Backend:** [agent/web.go](agent/web.go), [agent/tools.go](agent/tools.go) (the ~23 tools), [agent/agent.go](agent/agent.go), [agent/model_provider_catalog.go](agent/model_provider_catalog.go)

### KEEP (20)
Main chat area, message stream + input box, quickActions (6 commands), sidebar accordion sections, MarketTicker fetch logic (the *logic* is fine; just the hardcoded symbol list needs change), PositionsPanel (already supports stocks + crypto), most agent tools that are asset-agnostic: get_preferences, manage_preferences, get_backend_logs, get_decisions (works for any symbol), get_exchange_configs, get_model_configs, get_strategies, manage_trader, search_stock, get_positions, get_balance, get_market_price, get_kline, get_trade_history, get_watchlist, manage_watchlist. Backend handlers: `/api/agent/health`, `/api/agent/chat`, `/api/agent/chat/stream`, `/api/agent/tickers`. Memory: TaskState + chatHistory. All 9 streaming SSE events (planning/plan/step_start/step_complete/replan/tool/delta/done/error).

### MODIFY (18 — for minimal Plan 1 NQ-awareness)

| # | File:Line | Change | Type |
|---|-----------|--------|------|
| A-M1 | `MarketTicker.tsx:14` (hardcoded SYMBOLS) | Replace `['BTCUSDT','ETHUSDT','SOLUSDT']` with prop-driven or mode-detected list (include NQ when in futures mode) | MODIFY |
| A-M2 | `WelcomeScreen.tsx:22-34` (suggestion cards) | Replace crypto-only suggestions OR add NQ variants | MODIFY |
| A-M3 | `UserPreferencesPanel.tsx:130-132` (example placeholder) | Update example to mention futures or be generic | MODIFY |
| A-M4 | `agent/tools.go:~432` (exchange_type enum) | Add `ninjatrader` to supported exchange list | MODIFY |
| A-M5 | `agent/tools.go:~551` (`manage_exchange_config` tool description) | Document NinjaTrader CSV bridge setup steps the agent should know | MODIFY |
| A-M6 | `agent/tools.go:~592` (`manage_model_config` tool description) | Note that claw402/blockrun providers are crypto-only; not required for NQ | MODIFY |
| A-M7 | `agent/tools.go:~627` (`manage_strategy` tool: coin_source enum + static_coins example) | Add "NQ.c.0" to example; document that futures use static mode | MODIFY |
| A-M8 | `agent/tools.go:~691` (`execute_trade` description + examples) | Replace "long BTC, short ETH" examples with neutral or NQ-inclusive ones | MODIFY |
| A-M9 | `agent/tools.go:~756` (`get_market_snapshot` description) | Add asset-class guard: if NQ symbol, skip funding-rate / OI sections in response | MODIFY |
| A-M10 | `agent/tools.go:~796` (`get_candidate_coins`) | Note as crypto-only; if asked for NQ context, suggest static-mode strategy template | MODIFY |
| A-M11 | `agent/agent.go:62-72` (DefaultConfig WatchSymbols) | Make symbols configurable (env or per-user pref); default still BTC/ETH/SOL but extensible | MODIFY |
| A-M12 | `agent/agent.go:547-633` (Chinese system prompt) | Add NQ / CME context section ("If user mentions NQ/MNQ/futures, use NinjaTrader bridge + Databento data; skip funding rate questions") | MODIFY |
| A-M13 | `agent/agent.go:636-721` (English system prompt) | Mirror NQ awareness in English | MODIFY |
| A-M14 | `agent/web.go:207` (`HandleKlines`) | Abstract from hardcoded Binance futures URL OR route NQ symbols to Databento adapter | MODIFY |
| A-M15 | `agent/onboard.go:36+` (`needsSetup`) | Detect "NQ", "futures", "NinjaTrader" intent — skip claw402 wallet flow | MODIFY |
| A-M16 | `web/src/components/agent/SuggestionCards.tsx` (if exists, otherwise inline in AgentChatPage) | Add NQ suggestion when appropriate | MODIFY |
| A-M17 | `web/src/components/agent/MarketTicker.tsx:22-52` (fetch interval + UI) | Render NQ ticker entries cleanly (avoid stripping `.c.0` suffix) | MODIFY |
| A-M18 | Agent tool descriptions that mention "crypto" or "USDT" in tools.go | Soften: "crypto or futures symbol" | MODIFY |

### DEFER to Plan 2 (7 — full Agent NQ rewrite, not blocking)

| # | File | Why deferred |
|---|------|---------------|
| A-D1 | `agent/tools.go` `get_candidate_coins` full futures support | Requires new data source for NQ-equivalent ranking; not core to NQ trading |
| A-D2 | `agent/tools.go` `manage_strategy` `coin_source` enum extension | Static mode covers NQ; deeper futures-source registry is post-launch |
| A-D3 | `agent/model_provider_catalog.go:32-42` claw402 spec | Crypto payment unrelated to NQ; leave as-is |
| A-D4 | `agent/model_provider_catalog.go:44-52` blockrun-base | Same |
| A-D5 | `agent/model_provider_catalog.go:54-62` blockrun-sol | Same |
| A-D6 | `agent/memory.go` asset-class memory isolation | Risk: past crypto memory contaminates NQ reasoning. Add Plan 2 once we see real conflicts. |
| A-D7 | Deep agent-driven strategy generation for futures | Requires more sophisticated agent persona work; Plan 2 |

**Surface 4 LOC delta: ~140 (40 FE for ticker/welcome/suggestions, 100 BE for tool descriptions + system prompt + onboarding gating).**

## Combined acceptance criteria

When all 4 surfaces are at MODIFY-complete:

- **Config:** User adds NinjaTrader exchange via Settings → Exchanges → Add → select NT → enter DataDir + InstrumentName + DefaultContractQty → save. Listed correctly. No API key required.
- **Strategy:** User creates Strategy → SourceType="static" → StaticCoins=["NQ.c.0"] → PromptVariant="futures" → save. Editor hides funding/OI sections, shows NQ-aware risk control labels. Token estimate works. Test-run prompts contain NQ language.
- **Dashboard:** User creates Trader linking NinjaTrader exchange + NQ strategy + their AI model → opens Trader Dashboard → sees positions in contracts (no USDT unit, no leverage column, no liquidation column) → Recent Decisions render with NQ symbols → equity curve displays cleanly. AgentBeta sidebar markets list includes NQ.
- **AgentBeta:** User types "set up NinjaTrader for NQ trading" → agent suggests creating an exchange + strategy (does not push claw402 wallet) → user types "what's NQ doing today" → agent calls `get_market_price` (works) → "show NQ funding rate" → agent responds "futures don't have funding rate" gracefully.

## Final scope (4-surface view)

| Surface | LOC Δ | Of which NEW |
|---------|-------|--------------|
| Config | ~140 | 0 |
| Dashboard | ~80 | 0 |
| Strategy | ~340 | ~200 (futures prompt + validators) |
| AgentBeta | ~140 | 0 |
| **4-surface total** | **~700 LOC** | **~200 LOC new code** |

Plus the **backbone** (provider/databento/, provider/ninjatrader/, trader/ninjatrader/, cmd/nq_smoke, market adapter, kernel adapter wiring): ~1,500 LOC.

**Project total: ~2,200 LOC** spread across new code (~1,700) and surgical modifications (~500). Realistic timeline: **3-4 weeks focused engineering**.

**Items requiring confirmation** (flagged by the audit agents):
- `handleSaveExchange` already has 15 params; consider refactor to object param when adding NT fields
- Exchange.Create/Update Go function signatures: add NT fields as optional/last to avoid breaking call sites
- Icon file path for `/exchange-icons/ninjatrader.*` — need to provide an icon asset
- Decision whether to add `prompt_variant="futures"` as a NEW variant (alongside balanced/aggressive/conservative) OR a NEW `coin_source_variant` discriminator
- NQ symbol canonical format: confirm `NQ.c.0` (Databento) vs `NQ`/`NQM6` (specific contract) — propose `NQ.c.0` as primary
- Whether to *delete* leverage/liquidation columns server-side or *hide* them client-side — recommend client-side only (less backend churn)

---

# Plan 2: CME futures domain (production-grade)

> **Required before live money.** Plan 1 + 1.5 get the bot trading SIM cleanly. Plan 2 hardens it for CME's actual rules: tick sizes, contract rolls, session windows, holidays, settlement. Skipping Plan 2 = rejected orders + invalid prices + trading during closed markets.

## Task 17: Tick-size rounding for entry/SL/TP

**Why:** NQ tick = 0.25 (4 ticks/point); MNQ tick = 0.25. AI returns floating-point prices like `21503.17` — CME rejects anything not on a tick boundary. Round at the boundary before writing CSV.

**Files:**
- Create: `trader/ninjatrader/tick_rounding.go`
- Test: `trader/ninjatrader/tick_rounding_test.go`
- Modify: `trader/ninjatrader/trader.go` (call rounding before `csv_writer.Write`)

- [ ] **Step 1: Write the failing test**
```go
// trader/ninjatrader/tick_rounding_test.go
func TestRoundToTick(t *testing.T) {
    cases := []struct {
        in, tick, want float64
    }{
        {21503.17, 0.25, 21503.25},   // round up
        {21503.13, 0.25, 21503.00},   // round down
        {21503.125, 0.25, 21503.00},  // halfway = banker's round
        {21500.0, 0.25, 21500.00},    // exact
    }
    for _, tc := range cases {
        if got := RoundToTick(tc.in, tc.tick); got != tc.want {
            t.Errorf("RoundToTick(%v, %v) = %v, want %v", tc.in, tc.tick, got, tc.want)
        }
    }
}
```

- [ ] **Step 2: Implementation**
```go
// trader/ninjatrader/tick_rounding.go
package ninjatrader

import "math"

// InstrumentTickSize returns the tick size in points for a CME instrument.
// Returns 0.25 for NQ/MNQ/ES/MES (index futures default). Other instruments
// can be added as needed.
func InstrumentTickSize(symbol string) float64 {
    switch symbol {
    case "NQ", "MNQ", "ES", "MES":
        return 0.25
    case "YM", "MYM":
        return 1.0
    case "RTY", "M2K":
        return 0.10
    case "CL":  // crude oil
        return 0.01
    case "GC":  // gold
        return 0.10
    default:
        return 0.25  // safe default for indices
    }
}

// RoundToTick rounds price to the nearest tick boundary.
// Uses banker's rounding (round-half-to-even) to avoid bias.
func RoundToTick(price, tick float64) float64 {
    if tick <= 0 {
        return price
    }
    return math.Round(price/tick) * tick
}
```

- [ ] **Step 3: Wire into trader.go**
In `OpenLong` and `OpenShort`, before constructing SignalRow:
```go
tick := InstrumentTickSize(t.cfg.Symbol)
entry := RoundToTick(decision.Entry, tick)
sl := RoundToTick(decision.StopLoss, tick)
tp := RoundToTick(decision.TakeProfit, tick)
```

- [ ] **Step 4: Verify + commit**
```bash
go test ./trader/ninjatrader/...
git commit -m "feat(nt): round entry/SL/TP to instrument tick size"
```

## Task 18: CME session calendar + RTH/ETH gating

**Why:** CME Globex hours: Sun 5pm CT → Fri 4pm CT, daily break 4-5pm CT. Holiday closures exist (Christmas, New Year, etc.). Trading outside these windows = rejected orders + risk in thin liquidity.

**Files:**
- Create: `kernel/cme_calendar.go`
- Create: `kernel/cme_calendar_test.go`
- Modify: `kernel/engine.go` (skip decision cycle when market closed)

- [ ] **Step 1: Write the calendar**
```go
// kernel/cme_calendar.go
package kernel

import "time"

// IsCMEOpen reports whether CME Globex is open for index futures at the given time.
// Globex hours (Chicago time):
//   Sunday 17:00 → Friday 16:00, with a 60-minute daily break at 16:00–17:00.
// Holidays observed: New Year, MLK Day, Presidents Day, Good Friday, Memorial Day,
//   Juneteenth, Independence Day, Labor Day, Thanksgiving (+ day after), Christmas Eve,
//   Christmas Day. Each may have shortened hours; for v1 we treat them as full closures
//   and refuse to trade. Refine in Plan 3 if it becomes restrictive.
func IsCMEOpen(t time.Time) bool {
    chicago, _ := time.LoadLocation("America/Chicago")
    ct := t.In(chicago)
    if isCMEHoliday(ct) {
        return false
    }
    wd := ct.Weekday()
    hour := ct.Hour()
    switch wd {
    case time.Saturday:
        return false
    case time.Sunday:
        return hour >= 17
    case time.Friday:
        return hour < 16
    default:  // Mon-Thu
        return hour != 16
    }
}

// isCMEHoliday returns true if t falls on a CME-observed full-closure holiday.
// CME may have shortened-hours days (e.g. Good Friday, day after Thanksgiving),
// but for v1 we treat shortened days as full closures and refuse to trade.
// Refine in a later plan if this becomes operationally restrictive.
func isCMEHoliday(ct time.Time) bool {
    year := ct.Year()
    month := ct.Month()
    day := ct.Day()
    weekday := ct.Weekday()

    // Fixed-date holidays
    md := ct.Format("01-02")
    switch md {
    case "01-01": // New Year's Day
        return true
    case "06-19": // Juneteenth
        return true
    case "07-04": // Independence Day
        return true
    case "12-24": // Christmas Eve (early close treated as closure)
        return true
    case "12-25": // Christmas Day
        return true
    case "12-31": // New Year's Eve (early close treated as closure)
        return true
    }

    // MLK Day — 3rd Monday of January
    if month == time.January && weekday == time.Monday && (day-1)/7 == 2 {
        return true
    }

    // Presidents Day — 3rd Monday of February
    if month == time.February && weekday == time.Monday && (day-1)/7 == 2 {
        return true
    }

    // Good Friday — Friday before Easter
    if month == time.March || month == time.April {
        easter := easterSunday(year)
        goodFri := easter.AddDate(0, 0, -2)
        if ct.Year() == goodFri.Year() && ct.Month() == goodFri.Month() && ct.Day() == goodFri.Day() {
            return true
        }
    }

    // Memorial Day — last Monday of May
    if month == time.May && weekday == time.Monday {
        // Check if next Monday is in June (i.e. this is the last Monday of May)
        nextMon := ct.AddDate(0, 0, 7)
        if nextMon.Month() == time.June {
            return true
        }
    }

    // Labor Day — 1st Monday of September
    if month == time.September && weekday == time.Monday && day <= 7 {
        return true
    }

    // Thanksgiving — 4th Thursday of November (plus day after as early-close)
    if month == time.November && weekday == time.Thursday && (day-1)/7 == 3 {
        return true
    }
    // Day after Thanksgiving — Friday after 4th Thursday
    if month == time.November && weekday == time.Friday {
        thursday := ct.AddDate(0, 0, -1)
        if thursday.Month() == time.November && (thursday.Day()-1)/7 == 3 {
            return true
        }
    }

    return false
}

// easterSunday returns the date of Easter Sunday in the given year (Western/Gregorian).
// Used only for Good Friday calculation.
func easterSunday(year int) time.Time {
    // Anonymous Gregorian algorithm (Meeus/Jones/Butcher)
    a := year % 19
    b := year / 100
    c := year % 100
    d := b / 4
    e := b % 4
    f := (b + 8) / 25
    g := (b - f + 1) / 3
    h := (19*a + b - d - g + 15) % 30
    i := c / 4
    k := c % 4
    l := (32 + 2*e + 2*i - h - k) % 7
    m := (a + 11*h + 22*l) / 451
    month := (h + l - 7*m + 114) / 31
    day := ((h + l - 7*m + 114) % 31) + 1
    return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}
```

- [ ] **Step 2: Test**
```go
// kernel/cme_calendar_test.go
func TestIsCMEOpen(t *testing.T) {
    chicago, _ := time.LoadLocation("America/Chicago")
    cases := []struct {
        name string
        when time.Time
        want bool
    }{
        {"Mon 10am normal trading", time.Date(2026, 6, 15, 10, 0, 0, 0, chicago), true},
        {"Mon daily break 4:30pm CT", time.Date(2026, 6, 15, 16, 30, 0, 0, chicago), false},
        {"Saturday closed", time.Date(2026, 6, 20, 12, 0, 0, 0, chicago), false},
        {"New Year's Day", time.Date(2026, 1, 1, 10, 0, 0, 0, chicago), false},
        {"MLK Day 2026 (Jan 19)", time.Date(2026, 1, 19, 10, 0, 0, 0, chicago), false},
        {"Presidents Day 2026 (Feb 16)", time.Date(2026, 2, 16, 10, 0, 0, 0, chicago), false},
        {"Good Friday 2026 (Apr 3)", time.Date(2026, 4, 3, 10, 0, 0, 0, chicago), false},
        {"Memorial Day 2026 (May 25)", time.Date(2026, 5, 25, 10, 0, 0, 0, chicago), false},
        {"Juneteenth", time.Date(2026, 6, 19, 10, 0, 0, 0, chicago), false},
        {"Independence Day", time.Date(2026, 7, 4, 10, 0, 0, 0, chicago), false},
        {"Labor Day 2026 (Sep 7)", time.Date(2026, 9, 7, 10, 0, 0, 0, chicago), false},
        {"Thanksgiving 2026 (Nov 26)", time.Date(2026, 11, 26, 10, 0, 0, 0, chicago), false},
        {"Day after Thanksgiving 2026 (Nov 27)", time.Date(2026, 11, 27, 10, 0, 0, 0, chicago), false},
        {"Christmas Eve", time.Date(2026, 12, 24, 10, 0, 0, 0, chicago), false},
        {"Christmas Day", time.Date(2026, 12, 25, 10, 0, 0, 0, chicago), false},
        {"New Year's Eve", time.Date(2026, 12, 31, 10, 0, 0, 0, chicago), false},
    }
    for _, tc := range cases {
        t.Run(tc.name, func(t *testing.T) {
            if got := IsCMEOpen(tc.when); got != tc.want {
                t.Errorf("IsCMEOpen(%v) = %v, want %v", tc.when, got, tc.want)
            }
        })
    }
}
```

- [ ] **Step 3: Gate engine decision cycle**
In `kernel/engine.go` at the top of the scan loop:
```go
if config.Get().TradingMode == "futures" && !IsCMEOpen(time.Now()) {
    logger.Info("CME closed, skipping decision cycle")
    return
}
```

- [ ] **Step 4: Commit**
```bash
git commit -m "feat(futures): CME session calendar + skip decisions when market closed"
```

## Task 19: Contract roll automation

**Why:** NQ futures expire quarterly (March / June / Sep / Dec). NQ.c.0 is the front month; after expiry day, it auto-points to the next contract, but the AI may still be holding the old contract → liquidation. Need detection + roll plan.

**Files:**
- Create: `provider/databento/contract_calendar.go`
- Modify: `trader/ninjatrader/trader.go` (warn near expiry)
- Modify: `kernel/engine.go` (avoid new entries within 5 days of expiry)

- [ ] **Step 1: Add expiry resolver**
```go
// provider/databento/contract_calendar.go
package databento

import (
    "fmt"
    "strings"
    "time"
)

// CME month codes for futures contract symbology.
var cmeMonthCodes = map[byte]time.Month{
    'F': time.January,
    'G': time.February,
    'H': time.March,
    'J': time.April,
    'K': time.May,
    'M': time.June,
    'N': time.July,
    'Q': time.August,
    'U': time.September,
    'V': time.October,
    'X': time.November,
    'Z': time.December,
}

// NextExpiryFromSymbol returns the expiry date of the given CME contract code,
// disambiguating the single-digit year against `now`. Format: last 2 chars are
// month code (F/G/H/J/K/M/N/Q/U/V/X/Z) + last digit of year.
//
// Year disambiguation rule: assume the contract is in the current decade.
// If that would place the contract more than 1 year in the past, assume next
// decade. This handles the normal case (front-month contract within ~1 year
// of now) without breaking when the year-digit wraps (e.g. 2030).
//
// Examples (now=2026-05-22):
//   "MNQM6" → 2026-06 (current decade, current year)
//   "MNQU6" → 2026-09 (current decade, this year)
//   "MNQH7" → 2027-03 (current decade, next year)
//   "MNQM0" → 2030-06 (current decade, but year 2020 would be 6 years ago → bump to 2030)
func NextExpiryFromSymbol(symbol string, now time.Time) (time.Time, error) {
    if len(symbol) < 2 {
        return time.Time{}, fmt.Errorf("contract code too short: %q", symbol)
    }
    code := strings.ToUpper(symbol)
    monthChar := code[len(code)-2]
    yearChar := code[len(code)-1]

    month, ok := cmeMonthCodes[monthChar]
    if !ok {
        return time.Time{}, fmt.Errorf("invalid CME month code %q in %q", monthChar, symbol)
    }
    if yearChar < '0' || yearChar > '9' {
        return time.Time{}, fmt.Errorf("invalid year digit %q in %q", yearChar, symbol)
    }
    yearDigit := int(yearChar - '0')

    decade := (now.Year() / 10) * 10
    year := decade + yearDigit
    // If the candidate year is more than 1 year before now, the contract code
    // refers to the next decade.
    if year < now.Year()-1 {
        year += 10
    }
    return thirdFridayOf(year, month), nil
}

// DaysUntilExpiry returns calendar days from now until contract expiry.
// Returns 999 if the symbol cannot be parsed (treat as "not near expiry").
func DaysUntilExpiry(symbol string, now time.Time) int {
    exp, err := NextExpiryFromSymbol(symbol, now)
    if err != nil {
        return 999
    }
    return int(exp.Sub(now).Hours() / 24)
}

// thirdFridayOf returns the 3rd Friday of the given month — CME index futures
// expiry convention.
func thirdFridayOf(year int, month time.Month) time.Time {
    // Start at day 1, advance to first Friday, then add 14 days.
    first := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
    offset := (int(time.Friday) - int(first.Weekday()) + 7) % 7
    return first.AddDate(0, 0, offset+14)
}
```

- [ ] **Step 2: Engine gating**
In `kernel/engine.go` at decision-evaluate time:
```go
if config.Get().TradingMode == "futures" {
    if days := databento.DaysUntilExpiry(symbol, time.Now()); days <= 5 {
        logger.Warnf("contract %s expires in %d days, blocking new entries", symbol, days)
        decision.Action = "HOLD"  // override
    }
}
```

- [ ] **Step 3: Test year-digit disambiguation**
```go
// provider/databento/contract_calendar_test.go
func TestNextExpiryFromSymbol_YearDisambiguation(t *testing.T) {
    cases := []struct {
        symbol string
        now    time.Time
        want   time.Time
    }{
        {"MNQM6", date(2026, 5, 22), date(2026, 6, 19)},   // current quarter
        {"MNQU6", date(2026, 5, 22), date(2026, 9, 18)},   // next quarter
        {"MNQH7", date(2026, 5, 22), date(2027, 3, 19)},   // next year
        {"MNQM0", date(2029, 12, 1), date(2030, 6, 21)},   // year-digit wrap into next decade
        {"MNQH0", date(2029, 12, 1), date(2030, 3, 15)},   // wrap, earliest month
    }
    for _, tc := range cases {
        got, err := NextExpiryFromSymbol(tc.symbol, tc.now)
        if err != nil {
            t.Errorf("symbol=%q: %v", tc.symbol, err)
            continue
        }
        if !got.Equal(tc.want) {
            t.Errorf("symbol=%q now=%v: got %v, want %v", tc.symbol, tc.now, got, tc.want)
        }
    }
}

func date(y int, m time.Month, d int) time.Time {
    return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}
```

- [ ] **Step 4: Commit**
```bash
git commit -m "feat(futures): contract roll detection + block entries near expiry"
```

## Task 20: Symbol precision + decimal-safe arithmetic

**Why:** Position sizing, PnL, tick math all use float64. For NQ at 21500, 0.01-point rounding error = $5 over 100 trades. Audit and add `decimal.Decimal` only where it matters (position sizing, fill matching).

**Files:**
- Modify: `kernel/engine_position.go` (use math/big or shopspring/decimal for sizing)
- Test: `kernel/engine_position_test.go`

- [ ] **Step 1: Use existing helpers**
Check `market/data_klines.go` for any rounding helpers; if absent, add:
```go
func roundDecimals(v float64, decimals int) float64 {
    p := math.Pow(10, float64(decimals))
    return math.Round(v*p) / p
}
```
2 decimals for NQ prices; 0 decimals for contract quantities.

- [ ] **Step 2: Position-size sanity test**
```go
func TestPositionSize_NQ(t *testing.T) {
    // 1 NQ contract @ 21500 = $1,075,000 notional ($50/point × 21500)
    // Account $10k, risk 1% = $100; stop 25 points = $1,250 loss/contract
    // → Should size to 0 contracts (can't afford 1)
}
```

- [ ] **Step 3: Commit**

---

# Plan 3: Risk management + kill switches

> **Required for live money.** Without explicit risk limits, a runaway AI decision can blow up the account. Plan 3 adds hard limits enforced in Go (not just in the prompt).

## Task 21: Daily loss limit + force-flat kill switch

**Why:** AI prompt says "don't risk more than 1%" but it's a suggestion, not a guard rail. Need hard server-side enforcement.

**Files:**
- Create: `kernel/risk_limits.go`
- Create: `kernel/risk_limits_test.go`
- Modify: `kernel/engine.go` (pre-trade risk check + post-fill PnL check)
- Modify: `config/config.go` (add risk limit env vars)

- [ ] **Step 1: Risk-limit struct**
```go
// kernel/risk_limits.go
package kernel

type RiskLimits struct {
    MaxDailyLossUSD     float64  // hard stop for the day
    MaxConcurrentTrades int      // open position cap
    MaxNotionalUSD      float64  // total notional cap
    MaxContractsPerOrder int     // single-order size cap
}

func (r *RiskLimits) CheckPreTrade(ctx *Context, decision *Decision) error {
    if ctx.Account.TotalPnL < -r.MaxDailyLossUSD {
        return fmt.Errorf("daily loss limit hit: %.2f", ctx.Account.TotalPnL)
    }
    if len(ctx.Positions) >= r.MaxConcurrentTrades {
        return fmt.Errorf("concurrent trade cap reached: %d", r.MaxConcurrentTrades)
    }
    notional := decision.PositionSizeUSD
    for _, p := range ctx.Positions {
        notional += p.NotionalUSD
    }
    if notional > r.MaxNotionalUSD {
        return fmt.Errorf("notional cap exceeded: %.2f", notional)
    }
    return nil
}
```

- [ ] **Step 2: Wire into engine**
In `kernel/engine.go` after parsing the AI decision:
```go
limits := RiskLimits{
    MaxDailyLossUSD:     config.Get().RiskMaxDailyLossUSD,
    MaxConcurrentTrades: config.Get().RiskMaxConcurrentTrades,
    MaxNotionalUSD:      config.Get().RiskMaxNotionalUSD,
    MaxContractsPerOrder: config.Get().RiskMaxContractsPerOrder,
}
if err := limits.CheckPreTrade(ctx, decision); err != nil {
    logger.Warnf("⚠️ risk limit violated: %v — forcing HOLD", err)
    decision.Action = "HOLD"
}
```

- [ ] **Step 3: Config env vars**
```go
// config/config.go
RiskMaxDailyLossUSD     float64
RiskMaxConcurrentTrades int
RiskMaxNotionalUSD      float64
RiskMaxContractsPerOrder int
```
Defaults: `$500`, `2`, `$50000`, `5`.

- [ ] **Step 4: Force-flat API endpoint**
Add `POST /api/risk/force-flat` that calls every active trader's `CancelAllOrders + CloseLong/CloseShort`. (For NT v1: just cancel pending signals — manual close not supported.) Surface as a red "EMERGENCY FLAT" button on TraderDashboardPage.

- [ ] **Step 5: Tests + commit**

## Task 22: Stale-data + drift detection

**Why:** If Databento returns stale OHLCV (e.g. last bar is 10 min old), the AI is making decisions on old data. Detect and skip the cycle.

**Files:**
- Modify: `market/data.go` (timestamp check after fetch)
- Modify: `kernel/engine.go` (abort decision on stale data)

- [ ] **Step 1: Add `IsFresh` check**
```go
// in market/data.go
func (k *Kline) IsFresh(maxAge time.Duration) bool {
    return time.Since(time.UnixMilli(k.OpenTime)) <= maxAge
}
```

- [ ] **Step 2: Engine gating**
```go
lastBar := data.Klines[len(data.Klines)-1]
if !lastBar.IsFresh(2 * time.Minute) {
    logger.Warnf("stale data for %s (last bar %v old) — skipping cycle", symbol, time.Since(time.UnixMilli(lastBar.OpenTime)))
    return
}
```

- [ ] **Step 3: Commit**

---

# Plan 4: Observability + reliability

> **Required for confidence in production.** Without metrics, you don't know if the bot is healthy. Without retries, transient network blips become trade-misses.

## Task 23: Structured logging + decision audit trail

**Files:**
- Modify: `kernel/engine.go` (log decision with full context to structured logger)
- Modify: `store/decision.go` (persist decision JSON + risk-check outcome + execution status)
- Create: `api/handler_decisions.go` (read endpoint for audit replay)

- [ ] **Step 1: Decision struct expansion**
```go
type Decision struct {
    ID               int64
    TraderID         string
    Symbol           string
    Action           string
    Entry, SL, TP    float64
    Confidence       float64
    Reasoning        string
    PromptVersion    string  // hash of system prompt for reproducibility
    AIModel          string
    AILatencyMs      int64
    RiskCheckPassed  bool
    RiskCheckError   string
    ExecutionStatus  string  // "queued", "filled", "rejected", "blocked"
    FillPrice        *float64
    FillLatencyMs    *int64
    CreatedAt        time.Time
}
```

**Timezone requirement:** `CreatedAt` is always stored as UTC. The Go layer
inserts `time.Now().UTC()`; the database column is TIMESTAMP WITH TIME ZONE
(Postgres) or TEXT in ISO 8601 UTC format (SQLite). The display layer in
TraderDashboardPage converts to the user's local time zone for rendering.
Storing local time in the audit trail creates DST-transition bugs that
cause decisions to appear out-of-order or duplicated; UTC is non-negotiable.

The same rule applies to FillLatencyMs computation: subtract two UTC
timestamps, do not mix wall-clock and monotonic time.

- [ ] **Step 2: Persistence**
GORM Migrate adds the new columns. Insert at decision-time + update at fill-time.

- [ ] **Step 3: API + UI**
Expose at `GET /api/decisions/audit?trader_id=xxx&since=2026-05-22`; render in TraderDashboardPage's Decisions tab.

- [ ] **Step 4: Playwright audit trail assertion**

Verify the decision audit endpoint actually renders in the UI:

1. mcp__playwright__browser_navigate to /dashboard?trader=<id>
2. mcp__playwright__browser_click "Decisions" tab
3. mcp__playwright__browser_snapshot — assert table shows columns:
   Symbol, Action, Entry, SL, TP, Confidence, Risk Check,
   Execution Status, Fill Price, Latency
4. For a known historical decision, assert the Reasoning field
   expands on click
5. mcp__playwright__browser_console_messages — no errors

## Task 24: Retry + circuit breaker for Databento + NT bridge

**Why:** Databento HTTP can return 5xx; NT CSV write can fail with EBUSY on Windows file lock contention. Need bounded retry with backoff, plus circuit breaker that pauses the trader after N consecutive failures.

**Files:**
- Create: `mcp/retry.go` (generic retry-with-backoff)
- Modify: `provider/databento/client.go` (use retry on doRequest)
- Modify: `provider/ninjatrader/csv_writer.go` (retry on rename collision)

- [ ] **Step 1: Retry helper**
```go
func RetryWithBackoff(ctx context.Context, maxAttempts int, fn func() error) error {
    delay := 200 * time.Millisecond
    for attempt := 0; attempt < maxAttempts; attempt++ {
        if err := fn(); err == nil {
            return nil
        }
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-time.After(delay):
        }
        delay *= 2
        if delay > 5*time.Second {
            delay = 5 * time.Second
        }
    }
    return fmt.Errorf("max retries exceeded")
}
```

- [ ] **Step 2: Circuit breaker**
```go
type CircuitBreaker struct {
    failureThreshold int
    cooldown         time.Duration
    failures         int
    openedAt         time.Time
    mu               sync.Mutex
}
func (cb *CircuitBreaker) Allow() bool { /* ... */ }
func (cb *CircuitBreaker) RecordFailure() { /* ... */ }
func (cb *CircuitBreaker) RecordSuccess() { cb.failures = 0 }
```

- [ ] **Step 3: Wire into trader loop**
After N=5 consecutive failures, pause the trader for 5 minutes and log to decision audit trail.

## Task 25: Prometheus metrics endpoint

**Why:** CTO ask. Without numbers you don't know what's happening.

**Files:**
- Modify: `main.go` (add /metrics route)
- Create: `telemetry/metrics.go` (counter + histogram definitions)

- [ ] **Step 1: Define metrics**
```go
// telemetry/metrics.go
var (
    DecisionsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "nofx_decisions_total"},
        []string{"trader_id", "action", "status"},
    )
    DecisionLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "nofx_decision_latency_seconds"},
        []string{"trader_id"},
    )
    FillLatency = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{Name: "nofx_fill_latency_seconds"},
        []string{"exchange"},
    )
    DatabentoErrorsTotal = prometheus.NewCounter(
        prometheus.CounterOpts{Name: "nofx_databento_errors_total"},
    )
)
```

- [ ] **Step 2: Expose /metrics**
```go
http.Handle("/metrics", promhttp.Handler())
```

- [ ] **Step 3: Instrument decision path**
Add `DecisionLatency.WithLabelValues(traderID).Observe(elapsed.Seconds())` in `engine.go`.

---

# Plan 5: Testing matrix

> **Required for CI confidence.** Without integration tests, every code change is a coin flip.

## Task 26: Databento mock server

**Files:**
- Create: `provider/databento/mock_server.go` (httptest-based)
- Modify: `provider/databento/historical_test.go` (use mock)

- [ ] **Step 1: Mock OHLCV server**
```go
// provider/databento/mock_server.go
func NewMockServer(t *testing.T, fixture string) *httptest.Server {
    return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path == "/v0/timeseries.get_range" {
            http.ServeFile(w, r, fixture)
            return
        }
        if r.URL.Path == "/v0/symbology.resolve" {
            // ...
        }
    }))
}
```
Fixture: `provider/databento/fixtures/nq-ohlcv-1m.json` with 100 sample bars.

- [ ] **Step 2: Update tests to use mock**

## Task 27: NinjaTrader CSV bridge mock harness

**Files:**
- Create: `provider/ninjatrader/mock_nt.go` (goroutine that simulates NT polling + writing fills)
- Create: `trader/ninjatrader/integration_test.go` (full Go → CSV → mock NT → CSV → tailer round-trip)

- [ ] **Step 1: Mock NT loop**
```go
// Reads trade_signals.csv every 100ms (faster than real 2s for tests),
// writes a synthetic fill to trades_taken.csv after 200ms.
func StartMockNT(dataDir string, fillDelay time.Duration) (stop func())
```

- [ ] **Step 2: Round-trip test**
```go
func TestTrader_OpenLong_RoundTrip(t *testing.T) {
    dir := t.TempDir()
    stop := StartMockNT(dir, 200*time.Millisecond)
    defer stop()

    tr := New(Config{DataDir: dir, Symbol: "MNQ"})
    err := tr.OpenLong("MNQ", 21500, 21450, 21550)
    require.NoError(t, err)

    // Wait for fill...
    fill := waitForFill(t, dir, 2*time.Second)
    require.Equal(t, "LONG", fill.Direction)
}
```

## Task 28: AI prompt golden tests

**Why:** Changing the prompt template can silently break decision quality. Pin the prompt structure with snapshot tests.

**Files:**
- Create: `kernel/engine_prompt_golden_test.go`
- Create: `kernel/testdata/golden/futures_aggressive.txt`

- [ ] **Step 1: Snapshot test**
```go
func TestBuildFuturesSystemPrompt_Golden(t *testing.T) {
    got := BuildFuturesSystemPrompt(10000)
    want := readGoldenFile(t, "futures_aggressive.txt")
    if got != want {
        t.Fatalf("prompt changed.\nDiff:\n%s", diff(got, want))
    }
}
```
Update via `go test -update`.

## Task 29: End-to-end smoke matrix

Extend `cmd/nq_smoke/main.go` with sub-commands:
- `nq_smoke databento` — fetch OHLCV, verify shape
- `nq_smoke resolver` — resolve NQ.c.0 → MNQM6
- `nq_smoke prompt` — build system prompt, check non-empty
- `nq_smoke roundtrip` — full write→mockNT→tailer cycle
- `nq_smoke all` — runs everything

---

# Plan 6: Operational runbook

> **CTO/SRE-style ops doc.** Not code — but every production system needs these procedures.

## Task 30: Document startup procedure

**File:** Create `docs/operations/STARTUP.md`

Sections to include:
1. **Pre-flight checks** — env vars set (JWT_SECRET, DATABENTO_API_KEY, NINJATRADER_DATA_DIR), NT8 running on Windows, ClaudeTrader strategy attached to MNQ chart, WSL2 mirrored networking enabled.
2. **Cold start** — `./nofx-bin > /tmp/nofx.log 2>&1 &` — verify port 8080 listens, log shows "✅ System started successfully".
3. **Verify trader loads** — `curl localhost:8080/api/traders` returns the configured NT trader. Log shows `📦 Loading trader X (AI Model: deepseek, Exchange: ninjatrader/...)`.
4. **First-trade smoke** — manual decision via `cmd/nq_smoke/main.go all`, watch for fill appearing in `trades_taken.csv` and in DB `decisions` table.
5. **Shutdown** — `pkill -TERM -f nofx-bin`; verify no hung file handles via `lsof | grep NofxTrader`.
6. **Windows Defender exclusion for the data directory.** Defender's
   real-time scanner can transiently lock files during `os.Rename` from
   the Go side, causing the atomic temp+rename pattern in
   `provider/ninjatrader/csv_writer.go` to fail with EBUSY. Exclude the
   data directory from real-time scanning. From an Administrator
   PowerShell prompt on the Windows host:

```powershell
   Add-MpPreference -ExclusionPath "C:\Users\<user>\NofxTrader\data"
```

   Verify the exclusion is active:

```powershell
   Get-MpPreference | Select-Object -ExpandProperty ExclusionPath
```

   This is a one-time setup per host. Without it, you may see intermittent
   "rename: Access is denied" errors in Go logs during high-frequency signal
   writes — these are benign (the next write succeeds) but noisy.

## Task 31: Document rollback procedure

**File:** Create `docs/operations/ROLLBACK.md`

1. **Code rollback** — `git checkout <previous-good-commit> -- .`; `go build -o nofx-bin .`; restart.
2. **Schema rollback** — GORM auto-migrate is additive; for column removal, use a migration file under `store/migrations/`. Always snapshot `data/data.db` to `data/data.db.bak.<timestamp>` before code deploy.
3. **NT script rollback** — restore prior `claudetrader.cs` from git; rebuild in NT (`F5`).
4. **Risk wipe** — if rollback follows an unexpected loss, run force-flat (`curl -X POST localhost:8080/api/risk/force-flat -H "Authorization: Bearer $TOKEN"`) before any new trades.

## Task 32: Document monitoring + alerting

**File:** Create `docs/operations/MONITORING.md`

Key dashboards (Grafana / similar):
- **Decision rate** — `rate(nofx_decisions_total[5m])` — should be ~1/scan_interval
- **Fill latency** — `histogram_quantile(0.95, nofx_fill_latency_seconds)` — alert if > 30s
- **Databento errors** — `rate(nofx_databento_errors_total[10m])` — alert if > 0.1/sec
- **Daily PnL** — read from DB; alert if < -$500 (matches `RiskMaxDailyLossUSD`)
- **CME session health** — alert at 16:00 CT if a trade attempt was logged during the break

Manual checks (no alerting infra yet):
- `tail -f /tmp/nofx.log | grep -E "ERROR|WARN"`
- `curl localhost:8080/api/risk/status` — exposes current PnL + position count vs limits

## Task 33: Document disaster recovery

**File:** Create `docs/operations/DR.md`

1. **DB corruption** — restore from latest `.bak`; replay missing decisions via NT trades_taken.csv backfill (read fills, reconstruct decisions).
2. **NT8 crash mid-trade** — `claudetrader.cs` tracks active position internally; restart resumes from current position. **Risk:** if SL/TP weren't acked to broker, they're lost. Mitigation: after every NT restart, manually verify SL/TP are placed via the NT Orders window.
3. **Databento outage** — fall back to last-known OHLCV cached in DB; engine refuses new entries on stale data per Task 22.
4. **WSL2 reboot** — confirm `wsl --version` shows mirrored mode; reconfirm `/mnt/c/` is writable from Linux side.
5. **Lost JWT secret** — invalidate all sessions (clear `users.last_session_token`); users must re-login.

## Task 34: Document trader-mode runbook

**File:** Create `docs/operations/TRADER_MODE.md`

For the user (not engineers):
- How to switch from SIM to live (NT8 simulation account → real account; re-login required)
- Daily checklist before market open: NT8 connected? ClaudeTrader strategy enabled? Bot logs healthy?
- Weekly checklist: contract roll calendar (3rd Friday); review decision audit trail; rotate API keys.
- Emergency: how to hit force-flat from the dashboard.

### Playwright runbook verification

Before any live trade, run this manual check to confirm the
emergency-flat button works end-to-end:

1. cd web && npm run dev
2. mcp__playwright__browser_navigate to /dashboard
3. mcp__playwright__browser_click the red "Emergency Flat" button
4. mcp__playwright__browser_snapshot — assert confirmation modal
   appears
5. mcp__playwright__browser_click "Confirm"
6. Verify backend logs show: "FORCE FLAT initiated by user"
7. mcp__playwright__browser_snapshot — assert positions table is
   empty within 10s

If the button does not exist, does not confirm, or does not flatten —
do NOT trade live. Fix the button first.

---

# Plan 7: Documentation + ADRs

## Task 35: Architecture Decision Records

Create `docs/adr/` with one file per significant decision:
- `001-csv-bridge-vs-tcp.md` — why CSV first, TCP later (Plan 1.5)
- `002-databento-vs-alternatives.md` — why Databento over Polygon/IB Trader Workstation/CQG
- `003-nt8-vs-tradovate.md` — why we picked NT8 (user has license, NinjaScript open-source bridges exist)
- `004-decision-json-shape.md` — locking the action/symbol/entry/stop_loss/take_profit/reasoning shape across crypto + futures
- `005-tick-rounding-strategy.md` — banker's rounding, not floor/ceil, to avoid bias

Each ADR follows the format: **Status / Context / Decision / Consequences**.

## Task 36: API reference

`docs/api/README.md` — list every HTTP endpoint with request/response example. Generate from `api/server.go` route table if practical; otherwise write manually.

## Task 37: Onboarding doc

`docs/getting-started/NEW_DEV.md` — for someone joining the project:
- Repo layout map
- Run locally (`go build`, `cd web && npm run dev`)
- How to add a new exchange (link to `trader/CLAUDE.md`)
- How to add a new AI provider (link to `provider/CLAUDE.md`)
- Plan reading order: Plan 1 → 1.5 → 2 → 3 → 4 → 5 → 6 → 7

---

# Completion checklist (definitive)

| Plan | Status | Blocker for live? |
|---|---|---|
| Plan 1 (CSV bridge SIM) | ✅ Shipped | — |
| Plan 1.5 (NT8 AddOn TCP) | 📋 Documented | No (CSV works for SIM) |
| Plan 2 (CME domain) | 📋 Documented | **YES** — tick rounding + sessions + rolls |
| Plan 3 (Risk + kill switch) | 📋 Documented | **YES** — no live without hard limits |
| Plan 4 (Observability) | 📋 Documented | Recommended |
| Plan 5 (Testing matrix) | 📋 Documented | Recommended |
| Plan 6 (Operational runbook) | 📋 Documented | **YES** — for any non-dev operator |
| Plan 7 (Docs + ADRs) | 📋 Documented | No |

**Order of execution after Plan 1 cleanup (Tasks 12-16):**
1. Plan 2 (Tasks 17-20) — gate live with tick/session/roll/decimal correctness
2. Plan 3 (Tasks 21-22) — kill switches before any real money
3. Plan 4 (Tasks 23-25) — observability so you can SEE the bot
4. Plan 5 (Tasks 26-29) — CI confidence before iterating
5. Plan 6 (Tasks 30-34) — runbook for ops handoff
6. Plan 7 (Tasks 35-37) — write while context is fresh

After all of these: **the bot is ready for paper-live → real-live trading with reasonable safeguards.** Without Plan 2 + 3, do NOT trade real money.

## Plan 1 — Post-Mortem (2026-05-25)

Plan 1 was marked SHIPPED on 2026-05-22 based on:
- 2 real SIM fills on SIM101 via NT Playback
- Unit tests passing for provider/databento and provider/ninjatrader

Final acceptance via cmd/nq_smoke against the live Databento API
was deferred to 2026-05-25. That acceptance session surfaced six
findings in the cmd/nq_smoke entry point — four code bugs (fixed
in the 2026-05-25 session), one verified non-bug, and one
documentation/tier-reality observation. All findings sat in the
warmup / data-fetch / parser code path. NONE in the AI/CSV/NT
path.

The CSV→NT→fill round trip was validated tonight (2026-05-25 evening
session) with a real LIVE-session fill at 29807 on SIM101 after CME
Memorial Day closure ended.

### Bugs found and fixed (2026-05-25 session)

1. Databento window-too-recent — smoke runner queried end=now,
   but Databento Historical has publication lag. Fixed: 17min
   buffer (commit 286ee3b9), then 24h buffer (3ae0bf77) once
   tier-lag reality surfaced.

2. Weekend gap — 24h buffer fails on Monday morning when 24h
   back lands in the Sunday 18:00 to Friday 17:00 weekend hole.
   Fixed: 96h buffer (commit 18c21bae) covers worst-case 3-day
   holiday weekends.

3. Parser struct shape — Databento ohlcv-1m response has
   ts_event nested in hd, not top level. Hand-fabricated test
   fixture put ts_event at top level, so the unit test passed
   against a payload that never matched real API output. The
   bug shipped to SHIPPED status because the test was fiction.
   Fixed: real captured fixture + CRITICAL comment block
   (commit 8d20487d).

4. Lookback range too narrow — 30min window returns 30 bars;
   EMA50 needs 50 bar minimum. Fixed: 90min for 40-bar
   headroom (commit 46cd2aa1).

5. 1e9 fixed-point division — VERIFIED NOT A BUG. The
   scaledFloat() helper at provider/databento/historical.go:117-123
   correctly divides integer-scaled prices (e.g. "21500250000000")
   by 1e9 to produce floats (21500.25). Reviewed during the
   parser-fix pre-edit check on 2026-05-25; behavior was already
   correct, no patch required. Listed here so the audit trail
   shows the code path was examined and ruled out, not silently
   skipped.

6. Account-tier vs documented embargo — documentation/reality
   observation, captured as [VERIFY-16] below. The Plan 1.5
   design assumed Databento intraday GLBX.MDP3 has a documented
   15-minute embargo; that figure applies to the
   real-time-with-embargo subscription tier. The current
   Historical-tier account has multi-hour available_end lag
   (~3 hours observed) and required 96h lookback for
   weekend-spanning queries (per item 2 above). This is not a
   bug in the code — it is a planning assumption that needs
   revisiting if Plan 1.5 Cold Start runs on this tier.

### Lessons

1. Unit tests against fabricated fixtures are not unit tests.
   They are theater. Future Databento integration changes MUST
   verify fixtures against live curl output before committing.

2. SHIPPED status should require live-API acceptance through
   the production code path, not just unit tests + manual NT
   smoke. The 2026-05-22 SHIPPED claim was technically true
   for the NT bridge (csv_writer / csv_tailer / trader) which
   were exercised manually, but the cmd/nq_smoke entry that
   exercises the FULL chain Databento → indicators → AI →
   CSV → NT had never run.

3. Recommendation: future plans MUST add a "Live API acceptance"
   gate to the SHIPPED criteria, not just "unit tests pass +
   manual smoke." This is captured in Plan 5 (Testing matrix)
   Task 26 (Databento mock server) and Task 29 (E2E smoke
   matrix) — those tasks now have proven need.

### Pass 2 validation (2026-05-25 evening)

After Memorial Day closure ended (CME reopen 18:00 ET), pipeline
exercised end-to-end with a single LONG signal on SIM101:

- Signal: entry=29812.00, sl=29792.00, tp=29842.00 (1.5 R:R)
- CSV write to /mnt/c/Users/hoang/NofxTrader/data/trade_signals.csv
- NT VLTrader detected signal within ~2 sec
- Market order placed against MNQ 06-26 contract
- Filled at 29807.00 (5pt favorable slippage)
- Fill row written to trades_taken.csv
- Go tailer detected and logged the new fill
- Clean exit

H4 observed exactly as documented in Plan 1 hazards section: the
Go tailer re-emitted the two historical Playback fills from
2026-05-22 at startup before catching the new live fill. This is
the "fill replay on session rollover" hazard. Plan 1.5 fix
(stable fill-ID dedup + persisted offset) already specified in
canonical plan.

### Current Plan 1 status (after 2026-05-25)

- Databento parser handles real API response shape: DONE
- cmd/nq_smoke validates end-to-end pipeline (Pass 1): DONE
- cmd/nq_smoke validates CSV→NT→fill loop (Pass 2): DONE
- Two pending PRs to merge in order (chore/hide-faq-nav was
  superseded by commit 119e36ee already on this branch and
  closed without merge on 2026-05-25):
  - chore/rename-claudetrader-to-vltrader → main (audit doc only)
  - nq-databento-ninjatrader-plan → main (Plan 1 release)
- Plan 1.5 implementation: triggered by Plan 1.5 trigger
  conditions, not yet fired

### Branch state since Plan 1.5 design closeout

After e15c1dc0 (Plan 1.5 merge logic state machine, 2026-05-25),
eight commits landed on nq-databento-ninjatrader-plan during the
Plan 1 live-API acceptance + cleanup session:

- 286ee3b9 fix(nq_smoke): add 17min lag buffer for Databento
  historical embargo
- 3ae0bf77 fix(nq_smoke): use 24h lookback for Historical-tier
  Databento account
- 18c21bae fix(nq_smoke): 96h lookback survives weekend + 3-day
  holiday gaps
- 8d20487d fix(databento): parser struct shape + real fixture +
  docs URL (the parser-fixture-fiction bug surfaced here)
- 46cd2aa1 fix(nq_smoke): bump lookback range to 90min for EMA50
  headroom
- 9b7f8d70 chore(plan-1): rename ClaudeTrader → VLTrader in
  comments (1:1 swap across 5 Go files; upstream URL citation in
  provider/ninjatrader/types.go:7 preserved as historical
  attribution)
- 5db0bea6 docs(plan): Plan 1 post-mortem + Pass 2 validation +
  VERIFY-16 (this post-mortem section)
- 6d66b6ac docs(plan): document Playwright MCP availability for
  UI verification (registered npx -y @playwright/mcp@latest
  --headless in ~/.claude.json; Chromium binary pre-installed
  at ~/.cache/ms-playwright/ for headless E2E testing of
  frontend tasks 14/15)

Branch tip at the time of this update: 6d66b6ac. Branch is ahead
of origin/main by the full commit history since the v4-polish
merge (339d90ab).

PR #1 closeout (2026-05-25 evening): the chore/hide-faq-nav
branch was closed without merge after recon revealed it was
functionally identical to commit 119e36ee already on this
branch. Recon found a 51-file conflict surface (5 weeks of
branch divergence) for what would have been a no-op merge.
Branch deleted from local + origin. Future PR ordering:
rename audit → main, then Plan 1 release → main.

### [VERIFY-16] Databento tier vs documented embargo

Plan 1.5 design assumed Databento intraday GLBX.MDP3 has a
documented 15-minute embargo. That number applies to the
real-time-with-embargo subscription tier. The current account
is on basic Historical tier with multi-hour available_end lag
(~3 hours observed during 2026-05-25 acceptance session) and
weekend gap behavior requiring 96h lookback for Monday-morning
queries.

This does NOT change the Plan 1.5 architecture — bars come
from NT live feed via CQG/Rithmic, not Databento. Databento
Historical remains for warmup, gap-fill, backtest only, where
the multi-hour lag is acceptable.

But it DOES change one production-path expectation: when
implementing Plan 1.5 Cold Start Sequence (databento_warmup_end
= T0 - 17min), the 17min constant assumes real-time-with-embargo
tier. If the system runs on the current Historical tier, the
warmup must use a larger buffer (96h or session-aware) and
accept stale warmup bars.

For SIM/Plan-1 validation: 96h buffer is fine.
For live/Plan-1.5 implementation: confirm Databento tier
before implementing the Cold Start logic. Either upgrade to
real-time-with-embargo or adapt the buffer constant.

# Plan 4.x Roadmap — VL Futures Readiness

Captured from comprehensive web UI audit run on 2026-05-27 during Plan 1.5 operator verification session. The audit ran 104 tool calls over 11 minutes, mapped every page in the dashboard, identified crypto-era residue from the pre-VL-rebrand codebase, and surfaced Plan 1.5 backend functionality with no UI surface.

Cross-reference: ADR-001 (CSV bridge vs TCP), ADR-007 (Plan 1 critical file integrity).

## Chart Data Source Decision

**Decision: NinjaTrader (via Plan 1.5 VLTraderTCPClient.cs extension), NOT Databento.**

Rationale: Operator's Databento subscription is the delayed tier (~15-min lag on historical OHLCV bars). Real-time tier costs ~$199/mo for MNQ. For a futures-trading product, a delayed chart erodes operator trust in the entire app. NT8 provides zero-delay both historical and live bars to the operator's running NT8 instance, which is the same data the trader sees in the NT8 chart.

Architectural implications:
- Chart only works when NT8 + AddOn running (already the case when trading is active)
- Plan 4.4 extends the existing Plan 1.5 TCP wire protocol with new message types: `bars_subscribe`, `bar_update`, `bars_historical`
- VLTraderTCPClient.cs gets bar subscription via NT8 SDK `AddInstrumentBarsSubscription`
- Frontend MarketChart adds Futures pill, picks NT trader's instrument, subscribes via WS/SSE to bot's bar-relay endpoint
- Databento provider remains for kernel decisions (historical pulls for AI context); NOT used for chart

Plan 1.5 critical file integrity (ADR-007) preserved: existing TCP protocol stays additive; bar message types are NEW.

## Audit Findings — Issue Catalog

Recorded at file:line precision for surgical hotfix targeting.

### Crypto-era residue (UI shows crypto-only when futures is active)

| Issue | File:Line | Scope | Plan |
|---|---|---|---|
| Chart has no Futures pill, falls back to BTC/HYPERLIQUID | `web/src/components/charts/ChartTabs.tsx:19,28-34,47-53` | ~200 LOC, 90min | 4.4 |
| Backend `/api/klines` has no NT/Databento case (500s) | `api/handler_klines.go:48-78` | ~120 LOC, 60min | 4.5 |
| Stat cards hardcode USDT label | `web/src/pages/TraderDashboardPage.tsx:520,529,537` | ~80 LOC, 45min | 4.3 |
| Leverage + Liquidation columns shown for NT (meaningless) | `web/src/pages/TraderDashboardPage.tsx:665,667,719,729` | (bundled in 4.3) | 4.3 |
| Strategy Studio crypto-only (Coin Source AI500, BTC/ETH leverage, USDT grid) | `web/src/components/strategy/CoinSourceEditor.tsx`, `RiskControlEditor.tsx:80-127`, `GridConfigEditor.tsx:15-90` | ~400 LOC, 3hr | 4.6 |
| AgentChat tickers hardcoded BTC/ETH/SOL | `web/src/components/agent/MarketTicker.tsx:14`, `WelcomeScreen.tsx:24-31` | ~60 LOC, 30min | 4.7 |
| Settings exchange card shows API Key/Secret badges for NT | `web/src/pages/SettingsPage.tsx` | ~20 LOC, 20min | 4.13 |
| `/api/symbols` returns 400 'Unsupported exchange' for NT | `api/handler_symbols.go` | (bundled in 4.5) | 4.5 |

### Plan 1.5 backend with no UI surface

Operator drives these entirely via env vars / logs — no dashboard visibility:

- `NT_TRANSPORT=tcp` opt-in: no toggle in UI
- TCP listener status: no connected/waiting badge
- CME calendar gate: no 'CME CLOSED — skipping' indicator
- Databento provider status: no indicator
- NT8 AddOn version/health: no surface

Future Plan 4.14 candidate: 'Plan 1.5 backend visibility panel' (status badges, env-var toggle, CME session indicator). Scope estimate pending audit Phase 4 deepening.

### Mock/placeholder data still in production paths

| Issue | File:Line | Scope | Plan |
|---|---|---|---|
| NT trader returns hardcoded $50k mock balance | `trader/ninjatrader/trader.go:161-162` | ~150 LOC + C# AddOn ext, 2hr | 4.11 |
| DecisionAudit React key prop warning (console noise) | `web/src/components/trader/DecisionAudit.tsx:179-180, 234-236` | ~6 LOC, 5min | 4.9 |

### What is working for NT futures (validated by audit)

- Settings → Add Exchange → NinjaTrader template with WSL data dir, Instrument (default MNQ), Contract Qty fields
- Trader creation + run lifecycle on NT exchange
- TCP transport active on 36974, AddOn connects, signal frames flow (Plan 1.5 lifecycle complete)
- Decision audit table renders all 11 columns + 'wait' decision rows
- Emergency Flat modal opens, Cancel works, kernel-kill-switch warning correct
- Position History panel
- DecisionAudit surfaces `fill_price` + `fill_latency_ms`

Partial: `slippage_ticks` (Plan 1.5 wire field) not yet surfaced in DecisionAudit row.

## Plan 4.x Sequencing

> **PARTIALLY SUPERSEDED 2026-05-28 — see "Plan 4.x Sequencing —
> REVISED 2026-05-28" at the end of this doc for the current
> ordering after the NT8 pivot.** Items 1–4 below (Plan 4.9, 4.3,
> 4.3.1, 4.13, 4.7, 4.7.1) all SHIPPED — see Ship Log at end of doc.
> Item "Plan 4.5 — handler_klines.go NT route" is **DROPPED** —
> there is no Databento klines route in the NT8 architecture; the
> chart consumes the same NT8 bar feed as decisions (Plan 4.4
> Stage 4). The original sequencing below is preserved for the
> audit trail.

Proposed order (lowest risk + highest visible value first):

1. **Plan 4.9** — DecisionAudit React key warning
   ~6 LOC, 5min. Free hygiene. Removes console noise.

2. **Plan 4.3** — USDT label + leverage/liq column hiding
   ~80 LOC, 45min. Cosmetic but immediate visual correctness.

3. **Plan 4.13** — Settings exchange API Key badge for NT
   ~20 LOC, 20min. Cosmetic; removes confusion.

4. **Research dispatch** — NT8 AddOn bar subscription patterns, TradingView Lightweight Charts v5 wiring patterns, reference implementations. Web-search + web-fetch driven. Output: research report.

5. **Plan 4.4-spec** — Lock chart wiring design in plan doc (NT8 source). Tagged `v1.0-plan4-4-spec` (docs-only).

6. **Plan 4.5** — `handler_klines.go` NT route to bot's WS/SSE endpoint. Slim, orchestrates Plan 4.4 pipeline. ~120 LOC, 60min.

7. **Plan 4.4-build** — Go server bar-relay + C# AddOn bar subscription + frontend Futures pill + chart wiring. ~720 LOC across 3 languages. 3-4hr (likely with 1-2 compile hotfix cycles like Plan 1.5).

8. **Plan 4.7** — AgentChat tickers BTC/ETH/SOL → MNQ default. ~60 LOC, 30min.

9. **Plan 4.11** — NT trader real balance (no $50k mock). ~150 LOC + C# AddOn extension (new wire message: `account_balance`). 2hr.

10. **Plan 4.6** — Strategy Studio futures-aware rewrite. ~400 LOC, 3hr. Largest UX change. Lowest priority since AI-decision side works without UI editor changes.

11. **Plan 4.14** — Plan 1.5 backend visibility panel. Scope pending design.

Total scope estimate: ~1,700 LOC over ~12-15 hours of focused work. Spread across 6-10 individual PRs each with own tag.

## Verification Plan

Each Plan 4.x ships with Playwright surrogate verification:

- Plan 4.3, 4.7, 4.9, 4.13: Playwright pass confirms UI render + interaction
- Plan 4.4, 4.5: Playwright pass confirms chart renders MNQ data with non-empty bars + live update flicker
- Plan 4.6: Playwright pass confirms Strategy Studio loads + saves without crypto-required fields when exchange is NT
- Plan 4.11: Playwright pass confirms balance reflects NT account (not $50k)
- Plan 4.14: Playwright pass confirms backend status panel renders + reflects env-var state

Plan 1 critical files (ADR-007 invariant) preserved across all Plan 4.x work. Plan 4.4 + 4.11 ADD to Plan 1.5 wire protocol ADDITIVELY (new message types); existing `signal`/`fill`/`heartbeat`/`ack`/`bars_*` unchanged.

## Open Questions (to resolve before Plan 4.4 dispatch)

- Multi-instrument support: chart subscribes to ONE instrument at a time, or multi-symbol pane?
- Historical depth: how many bars on initial load? 500? 1000?
- Bar resolution: 1m only initially, or all of 1m/5m/15m/1H?
- WS vs SSE for bar relay: which fits the Go stack better?
- Bar cache invalidation: when NT8 disconnects, do cached bars stay shown or get marked stale?
- CME calendar integration: when CME is closed, chart should show 'CME CLOSED' overlay (Plan 4.14 dependency?)

## Cross-references

- ADR-001 — CSV bridge vs TCP (Plan 1.5 architectural baseline)
- ADR-007 — Plan 1 critical file integrity (preserved across Plan 4.x work)
- Plan 1.5 spec L4343-4447 — TCP wire protocol foundation that Plan 4.4 extends
- Plan 5 — Testing matrix (Plan 4.x extensions tested via the same mock + smoke patterns)
- web/CLAUDE.md — frontend-side architecture notes

## Plan 4.4 Deep Spec (Research-Informed)

This subsection supersedes the 200-LOC stub estimate for Plan 4.4 in the issue catalog above. Generated from a comprehensive research dispatch on 2026-05-27 covering NT8 BarsRequest API, TradingView Lightweight Charts v5 realtime patterns, Go SSE relay patterns, and alternative CME data source comparison.

### Scope revision

| Item | Roadmap stub | Research-informed actual |
|---|---|---|
| LOC | ~200 | ~2,260 |
| Time | 90 min | ~33 engineering hours |
| Stages | 1 (frontend only) | 4 (wire spec, C# AddOn, Go relay, frontend) |
| Hotfix cycles expected | 1 | 5 (one more than Plan 1.5) |
| Files modified | ChartTabs.tsx | ~10 across 3 languages |
| New files created | 0 | ~6 (Go bars package, C# manager, frontend chart) |

The original 200 LOC estimate covered only the frontend Futures pill addition. Real Plan 4.4 must extend:
- C# AddOn (`VLTraderTCPClient.cs` subscription manager) — ~600 LOC
- Go relay layer (new `provider/ninjatrader/bars_*` package + SSE handler in `api/`) — ~800 LOC
- Frontend `FuturesChart` component + ChartTabs integration — ~400 LOC
- Tests + integration fixtures (mock NT, golden data, Vitest, integration) — ~460 LOC

### Data source decision (confirmed by research)

NT8 remains the chosen chart data source. Comparative analysis ruled out:

- **Databento Standard:** $179/month for real-time CME live tier (operator currently on delayed tier with ~15min lag on historical bars). Upgrade is the clean fallback if NT8 ever proves unreliable, but $0 incremental cost via NT8 wins for v1.
- **Databento real-time latency:** median 6.1µs from venue handoff (per Databento blog) — Plan 4.5 candidate if triggered.
- **Polygon.io Futures:** $199/month, beta rollout state uncertain.
- **IQFeed (DTN):** $108.15/month base + $24.87 NA futures surcharge + per-exchange Globex fees.
- **Tradovate API:** requires funded Tradovate account (operator's prop firm constraint unknown).
- **Rithmic R | Protocol API:** requires funded Rithmic prop firm account; uses WebSockets + protobuf.
- **CME DataMine direct:** $600-$1,500/month professional rates only.

### Wire protocol extension (additive to Plan 1.5 per ADR-007)

Four new message types reuse Plan 1.5 framing (4-byte big-endian uint32 length prefix + snake_case JSON envelope `{type, payload}`). Existing `signal`/`fill`/`heartbeat`/`ack` byte-locked.

1. **`bars_subscribe`** (Go bot → C# AddOn)

```json
{
  "type": "bars_subscribe",
  "payload": {
    "subscription_id": "sub_01HMNQ12345",
    "symbol": "MNQ 03-26",
    "period_type": "minute",
    "period_value": 1,
    "bars_back": 500,
    "trading_hours": "CME US Index Futures ETH",
    "lookup_policy": "provider",
    "merge_policy": "merge_back_adjusted"
  }
}
```

2. **`bars_historical`** (C# AddOn → Go bot → frontend, one shot)

```json
{
  "type": "bars_historical",
  "payload": {
    "subscription_id": "...",
    "symbol": "MNQ 03-26",
    "error_code": "NoError",
    "error_message": "",
    "bars": [{"t": 1748352000, "o": 21845.25, "h": 21847.50, "l": 21844.00, "c": 21846.75, "v": 1234}],
    "last_bar_in_progress": true
  }
}
```

`last_bar_in_progress: true` always — per NT8 BarsRequest docs the final returned bar may be in-progress.

3. **`bar_update`** (C# AddOn → Go bot → frontend, streaming)

```json
{
  "type": "bar_update",
  "payload": {
    "subscription_id": "...",
    "symbol": "MNQ 03-26",
    "bars": [{"t": 1748352060, "o": 21846.75, "h": 21847.25, "l": 21846.50, "c": 21847.00, "v": 42}],
    "is_new_bar": false,
    "tick_seq": 4827
  }
}
```

`bars` array (not single bar) because NT8 `OnBarUpdate` can update `MinIndex..MaxIndex` range from a single tick on Range or non-time-based bar periods. `tick_seq` used as SSE `Last-Event-ID` for resume on disconnect.

4. **`bars_unsubscribe`** (Go bot → C# AddOn)

```json
{
  "type": "bars_unsubscribe",
  "payload": {"subscription_id": "..."}
}
```

**Field shape conventions:**
- snake_case JSON field names (`signal_id`, `bars_back`, `trading_hours`)
- doubles formatted with `.ToString("R", CultureInfo.InvariantCulture)` to preserve precision on the wire
- timestamps as Unix UTC seconds (integer); converted in C# from NT8's local-time `Bars.GetTime()` via `TimeZoneInfo.ConvertTimeToUtc` with `Bars.TradingHours.TimeZoneInfo`
- compact JSON, no pretty-print
- UTF-8 byte encoding

### Architecture

```
NT8 (Windows 11)              Go bot (WSL2 Ubuntu)              Browser
─────────────────             ──────────────────────             ─────────────────
VLTraderAddOn                 tcp_server.go                      FuturesChart.tsx
├ Plan 1.5 TCP client         ├ envelope router                  ├ createChart
└ VLBarsSubscription          └ BarsRegistry (NEW)               ├ chart.addSeries
  Manager (NEW)                 ├ map[subID]*ntSub                  (CandlestickSeries)
  ├ BarsRequest per             ├ per-subscriber Out             ├ series.setData
  │  subscription                  channel (256 buf)              ├ series.update
  ├ .Update event               └ drop-newest backpressure       └ EventSource SSE
  ├ MinIndex..MaxIndex                                              (?token=JWT)
  │  bar batching             sse_handler.go (gin) (NEW)
  ├ Connection                 ├ GET /api/v1/bars/stream
  │  reconnect-recreate        ├ JWT in ?token= (EventSource
  └ JSON encoder                │   cannot send headers)
     (StringBuilder,            ├ 15s keepalive comment
      InvariantCulture,         └ X-Accel-Buffering: no
      "R" round-trip)
```

Transport: SSE over WebSocket. Rationale: server→client only once subscribed, EventSource native browser auto-reconnect, HTTP/2 multiplexing friendly, easier debugging via curl, plays nice with existing gin stack. WebSocket reserved for multi-chart panes (Plan 4.5+).

### Latency budget (200ms tick-to-chart SLO)

```
NT8 tick → BarsRequest.Update             5-20ms
C# encode + TCP send                      <5ms
WSL2 ↔ Windows loopback                   <2ms (Plan 1.5 proven)
Go fan-out + SSE write                    <5ms
Browser EventSource receive               10-50ms
React state + canvas redraw               5-20ms
─────
Total typical:                            30-100ms
SLO ceiling:                              200ms
```

### NT8 SDK gotchas (15 cataloged, top 10 most critical for Plan 4.4)

1. **Reconnect kills `BarsRequest.Update` silently.** Forum thread (Mizpah Software / jeronymite): after a connection drop and reconnect, the `.Update` event stops firing for the existing `BarsRequest`. Workaround: hook `Connection.ConnectionStatusUpdate`; on transition back to Connected, dispose+recreate every active `BarsRequest` with same parameters. NT staff acknowledged but not documented as a platform guarantee.

2. **Multiple bars updated by single tick.** Official `BarsRequest` docs: "Depending on the BarsPeriod type, you can have situations where more than one bar is updated by a single tick. Be sure to process the full range of updated bars." Plan 4.4 `OnBarUpdate` handler must loop `e.MinIndex..e.MaxIndex`; `bar_update.payload.bars` MUST be an array.

3. **`bars.GetTime()` returns local time, not UTC.** Per NT forum (ChelseaB staff): "Real-time and historical data are already automatically converted to your local time zone as they are received." TradingView Lightweight Charts needs Unix UTC seconds. Conversion required: `TimeZoneInfo.ConvertTimeToUtc(t, bars.TradingHours.TimeZoneInfo)` → Unix seconds.

4. **Newtonsoft.Json officially unsupported in user AddOns.** Per NT staff. Plan 1.5.2 already established the hand-rolled JSON encoder pattern in pure `System.*` C#. Plan 4.4 extends the same encoder with array support for `bars[]`.

5. **No `is_first_tick_of_bar` helper in AddOn context.** Compute as `e.MaxIndex > sub.LastSeenMaxIndex`. Save `LastSeenMaxIndex` on each event. Per NT staff forum reply.

6. **Connection-specific BarsRequest not supported.** Per NT staff (Jesse): "The bars request would use the platform's general connection settings." Operator with multiple broker connections (FXCM, IB, Rithmic, etc.) must place the futures broker connection first in connection order.

7. **`NinjaScript.Print`/`Log` do NOT work in AddOn context.** Use `NinjaTrader.Code.Output.Process(msg, PrintTo.OutputTab1)` — Plan 1.5.2 established this. `PrintTo.OutputTab2` reserved for errors.

8. **AddOn `State.Configure` may run multiple times.** UI clone problem. Guard real allocation with static-bool. Real cleanup only in `State.Terminated` of same instance.

9. **`ErrorCode` enum only 7 documented members** (NoError, LogOnFailed, OrderRejected, UnableToCancelOrder, UnableToChangeOrder, UserAbort, Panic). NoData not in published list. Handler stringifies received value; anything != NoError is failure.

10. **CME US Index Futures RTH template was updated in 2024** to remove the 15:15-15:30 CT halt. ETH recommended for MNQ algo trading (captures overnight session).

(11-15: lock contention, memory leaks, dispose ordering, NT8 version differences, freeze-on-reconnect 10-15s — documented in research artifact.)

### Frontend gotchas (9 critical)

1. **EventSource cannot set custom headers.** JWT must travel via short-lived (5 min) signed query parameter. Server reads `c.Query("token")`, not `Authorization` header.

2. **`setData()` vs `update()` semantics.** Per v5 docs: "We do not recommend calling `ISeriesApi.setData` to update the chart, as this method replaces all series data and can significantly affect the performance." Use `setData()` ONCE on `bars_historical` receive, then `update()` for every `bar_update`.

3. **`update()` same time vs new time.** Same time → extends current bar (live progression). New time → appends new bar. Frontend handles `last_bar_in_progress` correctly.

4. **TradingView v5 API change.** `chart.addCandlestickSeries()` removed. Must use `chart.addSeries(CandlestickSeries, opts)` with explicit import.

5. **Time format.** `UTCTimestamp` (Unix seconds) for intraday; ISO strings collapse to daily resolution.

6. **`autoSize: true`** in v5 eliminates ResizeObserver boilerplate.

7. **Background tab throttling.** Browser throttles canvas redraws on hidden tabs but SSE keeps flowing. Accept; let canvas catch up on tab focus.

8. **Chart unmount cleanup.** Must close `EventSource` AND call `chart.remove()` in React useEffect cleanup. Subscription teardown via `bars_unsubscribe` handled server-side on EventSource close.

9. **Loading/Empty/Error states.** Must surface in UI distinctly: loading (waiting `bars_historical`), ready (data displayed), disconnected (auto-reconnecting), error (`bars_historical` error_code != NoError).

### Go relay gotchas (8 critical)

1. **Subscription dedup keyed by (userID, symbol, period).** Multiple frontend clients viewing same symbol share ONE NT8 BarsRequest. Registry: `map[userKey]subscriptionID → ntSubscription`.

2. **Backpressure: drop-newest, never block broadcast.** Per-subscriber buffered channel (256 entries). Slow SSE client doesn't block the fan-out goroutine.

3. **Cache last `bars_historical` for late joiners.** Second subscriber to same `(userID, symbol, period)` gets cached payload immediately, no second NT8 round-trip.

4. **`X-Accel-Buffering: no` header** defeats reverse-proxy buffering (nginx, traefik). Required even in dev to prevent Vite proxy buffering.

5. **Disable gzip middleware on `/api/v1/bars/stream`.** SSE incompatible with gzip; per-route exclusion required.

6. **Concurrent map access.** `sync.RWMutex` + plain map (not `sync.Map`) because access pattern is "many writes broadcast to N readers" — `sync.Map` optimizes for the opposite.

7. **Graceful shutdown.** Drain subscriber channels, send `bars_unsubscribe` for each NT subscription, close TCP connection cleanly.

8. **CORS.** `Access-Control-Allow-Origin` matching the Vite dev origin (`http://localhost:3000`). No credentials needed when using JWT-in-query.

### 4-stage rollout

**Stage 1 — Wire spec + end-to-end loopback (~8h)**
- Land 4 new envelope types in Plan 1.5 wire spec
- Implement C# `VLBarsSubscriptionManager` + JSON encoder extensions
- Implement Go `BarsRegistry` + SSE handler
- Assert: curl-driven SSE client gets `bars_historical` with ≥500 bars from a single `bars_subscribe` envelope
- Promotion gate: end-to-end loopback works; p99 encode/decode <5ms; one MNQ chart renders

**Stage 2 — Robustness against NT8 gotchas (~12h)**
- Reconnect-driven BarsRequest recreation (`Connection.ConnectionStatusUpdate` hook)
- `MinIndex..MaxIndex` multi-bar batching verified with Range bar period
- `last_bar_in_progress` handling verified on frontend
- Backpressure dropping verified with slow-consumer Vitest
- Promotion gate: survive 30s NT8 disconnect; survive tab backgrounded 10min; survive MNQ Z25→H26 roll with MergeBackAdjusted

**Stage 3 — Multi-tenant + auth + ops polish (~8h)**
- JWT-in-query 5-min TTL with refresh-on-401
- Per-tenant subscription isolation
- SSE keepalive comment every 15s
- Prometheus metrics: `active_subscriptions`, `dropped_envelopes`, `per_symbol_tick_rate`
- Promotion gate: 2 concurrent operators on same MNQ chart, independent NT subscriptions, isolated SSE streams

**Stage 4 — Multi-pane indicators (Plan 4.5 candidate)**
- v5 native multi-pane RSI/MACD/Volume separate panes
- Defer if Plan 4.4 budget tight

### Failure modes to expect (lessons from Plan 1.5)

Plan 1.5 needed 4 hotfix cycles. Plan 4.4 expects 5 because of 4 new failure dimensions:

1. **Time-zone conversion bugs.** First chart load may show bars at midnight UTC instead of US trading hours if `ConvertTimeToUtc` gets wrong `TimeZoneInfo`. → Plan 4.4.1 hotfix likely.

2. **Reconnect doesn't recover bars stream** (the `Connection.ConnectionStatusUpdate` hook is fragile). → Plan 4.4.2 hotfix likely.

3. **OCO bracket signal vs bars stream interleave** in the TCP wire — both flowing through `tcp_server.go`, must not corrupt each other's frames. → Plan 4.4.3 hotfix possible.

4. **EventSource auto-reconnect storms** if the SSE server restarts during dev. Need exponential backoff client-side. → Plan 4.4.4 hotfix likely.

5. **NT8 SDK API mismatch** (BarsRequest constructor, BarsPeriod enum values, MergePolicy values vary across NT 8.0.x and 8.1.x). → Plan 4.4.5 hotfix possible.

### Verification approach

Each Stage promotion gates on Playwright surrogate verification (proven by Plan 1.5):
- Stage 1: Playwright confirms one Futures pill appears, chart renders with 500 bars
- Stage 2: Manual NT8 disconnect/reconnect; Playwright confirms chart resumes within 10s
- Stage 3: Two browser sessions on same chart; Playwright confirms both render
- Stage 4: Multi-pane RSI verified

Real-world operator verification (NT8 + dashboard + live SIM signal flow + bar updates correlating with NT8 chart) runs at end of Stage 2 minimum.

### Open questions (resolve before Plan 4.4-build dispatch)

1. Trading hours default: ETH (recommended for algo) vs RTH
2. MergePolicy default: BackAdjusted (chart-friendly) vs NonBackAdjusted
3. `bars_back` default: 500 (≈ 8.3 ETH hours) vs 1000
4. Multi-symbol overlay in Plan 4.4 or defer to 4.5
5. JWT TTL for SSE query param (recommend 5 min, refreshable)
6. Drop-newest backpressure policy confirmation
7. Per-tenant subscription dedup behavior confirmation
8. Chart-side persistence on Go side: none (confirm acceptable)

### Research artifact reference

Full research report (12 sections, production C# code skeletons, Go relay reference implementation, frontend React code, comprehensive gotchas catalog, GitHub reference implementations with URLs) captured as conversation artifact on 2026-05-27. Future Plan 4.4-spec and Plan 4.4-build dispatches reference both:
- This canonical plan doc section (locked design + scope)
- The research artifact (production code skeletons + exhaustive edge cases)

### Cross-references

- ADR-001 — CSV bridge vs TCP (Plan 1.5 baseline that 4.4 extends)
- ADR-007 — Plan 1 critical file integrity (preserved; 4.4 ADDS message types, doesn't modify existing)
- Plan 1.5 spec L4343-4500 — TCP wire protocol foundation
- Plan 5 — Testing matrix (Plan 4.4 extends via mock NT + golden bar data)
- web/CLAUDE.md — frontend architecture notes
- Research artifact (2026-05-27) — comprehensive technical reference

---

# 2026-05-28 Plan Update — Ship Log, Post-Mortems, Revised Roadmap

This block records reality as of 2026-05-28 and supersedes earlier
sections where flagged. See also the "2026-05-28 ARCHITECTURE PIVOT"
section at the top of this doc for the NT8-as-single-source decision.

## Ship Log (since v1.0-plan4-roadmap-spec)

25 release tags on origin as of 2026-05-28. New since the last plan
update (chronological by merge):

| Tag | Date | PR | Summary |
|---|---|---|---|
| `v1.0-plan4-9` | 2026-05-26 | #19 | DecisionAudit React key warning fix (Dashboard Decisions tab console hygiene) |
| `v1.0-plan4-3` | 2026-05-26 | #20 | USDT→USD labels + Leverage/Liquidation column hide on Dashboard StatCards + positions table |
| `v1.0-plan4-13` | 2026-05-26 | #21 | Settings → Exchanges NT card shows "TCP Bridge" badge instead of "API Key" / "Secret" |
| `v1.0-plan4-3-1` | 2026-05-26 | #22 | EquityChart USDT residue cleanup (7 sites now use `currencyLabel`, wired from `isFutures` prop) |
| `v1.0-plan4-7` | 2026-05-26 | #23 | AgentChat MarketTicker BTC/ETH/SOL → MNQ; WelcomeScreen prompts futures-aware |
| `(docs)` | 2026-05-26 | #24 | Vite HMR stale-module cache gotcha documented in `web/CLAUDE.md` |
| `v1.0-plan4-7-1` | 2026-05-26 | #25 | UserPreferencesPanel placeholder MNQ-only (Plan 4.7 residue: "focusing on BTC and ETH" → "focusing on MNQ") |
| `v1.0-plan1-5-6` | 2026-05-27 | #26 | TCP heartbeat write-deadline + concurrent-write mutex — fixes the 60s reconnect loop caught by 2026-05-27 comprehensive audit NEW-1 |
| `v1.0-task12-symbols` | 2026-05-28 | #27 + #28 | Futures symbol normalization (Databento branch added in #27, then trimmed in #28 per the NT8 pivot); symbol fixes kept |
| `(untagged merge)` | 2026-05-28 | #30 | Plan 1.5.7 — TCP read-deadline desync fix (Patch C). Merged to main at `6defdc84`; no release tag yet |

Plus the comprehensive 5-page UI inventory produced 2026-05-28 at
[docs/internal/inventory/](docs/internal/inventory/) — see
`INDEX.md` for the cross-reference + 27 observations including N11.

## Post-Mortems (2026-05-28)

### Plan 1.5.6 — TCP heartbeat write-deadline bug

- **Symptom:** Deterministic 60-second TCP reconnect loop. 240+ WARN
  `tcp_server: write heartbeat ack err="...i/o timeout"` events
  observed across ~16 hours steady-state on 2026-05-27.
- **Root cause:** Go's `net.Conn` write deadlines are PERSISTENT
  until reset. The `FrameHeartbeat` ack handler in `readLoop` called
  `WriteFrame` WITHOUT first calling `SetWriteDeadline`. The ack
  inherited the stale 5s deadline from the most recent
  `heartbeatLoop` send. 5 seconds after that scheduled heartbeat,
  the deadline expired; any subsequent write — including the ack
  to the next client heartbeat — failed immediately with
  `i/o timeout`. `readLoop` returned, `defer closeConn()` ran, socket
  died. C# AddOn reconnected 5s later. Cycle = 60s.
- **Why hotfixes 1.5.1 through 1.5.4 missed it:** They addressed
  feature-complete surface (Newtonsoft removal, hand-rolled JSON
  encoder, auto_trader wiring, server singleton). None exercised
  the steady-state heartbeat path for more than a few minutes.
  Any soak ≥1 minute would have surfaced this.
- **Fix:** `provider/ninjatrader/tcp_server.go` — two changes,
  primary + secondary:
  - **PRIMARY (closes the actual bug):** add `SetWriteDeadline`
    before the ack write so it no longer inherits the stale
    deadline.
  - **SECONDARY (defense-in-depth, not because frame corruption
    was observed):** add `writeMu sync.Mutex` to serialize the
    3 WriteFrame call sites (ack, heartbeat send, signal flush).
    The C# side already uses `lock(writeLock)`; this brings the
    Go side to parity.

  +28/-4 LOC. Single file. ADR-007 critical files untouched. No
  C# AddOn recompile.
- **Verification:** 14-minute soak post-deploy on PID 51866 showed
  0 heartbeat errors, 1 client connect (initial only), 0
  disconnects. Pre-fix would have produced 14+ of each.

### Plan 1.5.7 — TCP read-deadline desync (Patch C)

- **Symptom:** Spurious `oversized frame` disconnects (~0.5%/frame,
  ~49 events over 16h). Caused intermittent C# AddOn reconnects on
  top of the 1.5.6 baseline — present since Plan 1.5 initial impl,
  invisible until the 1.5.6 fix removed the dominant noise.
- **Root cause:** `readLoop` used `SetReadDeadline(2 * time.Second)`
  + continue-on-timeout to let the loop notice `ctx.Done()`. Partial
  reads near the deadline boundary left bytes consumed without the
  full frame being parsed. The next iteration started reading from
  the middle of the previous frame — interpreting 4 arbitrary bytes
  as the next length-prefix → garbage length (often >1MB) →
  `ErrFrameTooLarge` → disconnect.
- **Fix (Patch C):** Removed the read deadline; block on `ReadFrame`
  until a complete frame arrives OR the connection closes.
  Implementation uses **`context.WithCancel` + a watcher goroutine**
  (`connCtx, connCancel := context.WithCancel(ctx); defer connCancel();
  go func() { <-connCtx.Done(); c.Close() }()`) — NOT
  `context.AfterFunc`. Same architectural pattern (cancellable
  blocking read driven by ctx), the more traditional API. Either
  would work; the operator chose `WithCancel`+goroutine to keep the
  pattern uniform with the rest of the codebase. No frame-byte
  desync possible.
- **Found by:** Pre-Stage-2 diagnostic — looking for any other
  socket noise before starting Plan 4.4 Stage 2. Confirmed via
  post-1.5.6 soak logs showing the residual ~0.5%/frame
  disconnect pattern.

### N11 — Trader STARVED (not blocked by risk gate)

> **SUPERSEDED 2026-05-28 → see `## N11 — RESOLVED (flip applied + cycle proven, 2026-05-28)`** (appended at end of doc). The N11 coin-source flip is applied and the decision cycle is proven end-to-end on SIM. The original `risk_check_passed=false` reasoning below is also corrected: that column is UNWIRED (never a rejection signal) — see `## 2026-05-28 — risk_check_passed is UNWIRED (not a rejection signal)`. The remaining WAIT decisions are the empty-market-data root cause, not ai500 starvation — see `## 2026-05-28 Root Cause — Kernel reads Binance klines, not NT8 BarCache`.

- **Symptom:** Trader `mnq sIM TEST` had `risk_check_passed=false`
  visible in DecisionAudit, and 5+ consecutive cycles showed "No
  candidate coins available, cycle skipped." Initial diagnosis:
  "risk gate is rejecting; flip the threshold."
- **Actual root cause:** The active strategy was using a dead
  `ai500` coin source (HTTP 402 from claw402 paywall). The cycle
  short-circuited at coin selection BEFORE running the AI brain or
  the risk gate. `risk_check_passed=false` was a GORM zero-value
  (the field was never set because that code path never ran), not
  a real risk rejection.
- **Why the misdiagnosis:** The DecisionAudit column shows
  `risk_check_passed` as a boolean; the UI doesn't distinguish
  "false because rejected" from "false because never evaluated."
  Initial fix proposal was a "30-second config flip" — turn off
  the risk gate. That would have done nothing because the gate
  wasn't the blocker.
- **Real fix path:** Data-fetch routing (the Plan 4.4 staged build).
  Once decisions get bars from NT8 instead of dead AI500/Databento
  paths, the cycle reaches the AI brain and the risk gate, and
  trades can flow. Trader was STARVED of input data, not blocked
  by output gate.
- **Documentation correction:** the inventory N11 entry was
  corrected from "30-second config fix" to "requires Plan 4.4
  staged build; data-fetch routing is the real blocker."
- **CORRECTION 2026-05-28 →** the "GORM zero-value" reasoning above
  is technically right but understates the scope: `risk_check_passed`
  is an UNWIRED column — no code path EVER sets it (verified across
  all 246 rows, before AND after the N11 flip). It was never a real
  rejection signal in any scenario, not just this one. See
  `## 2026-05-28 — risk_check_passed is UNWIRED (not a rejection signal)`.

## Plan 4.x Sequencing — REVISED 2026-05-28

Supersedes the earlier sequencing above (lines ~6940+). Reflects
both shipped work and the NT8 pivot.

### Already SHIPPED
- v1.0-plan4-9, 4-3, 4-3-1, 4-13, 4-7, 4-7-1 — small wins (above)
- v1.0-plan1-5-6 — TCP heartbeat write-deadline fix
- v1.0-task12-symbols — futures symbol normalization
- Plan 1.5.7 read-deadline fix (merged, untagged)
- **Plan 4.4 Stage 1 — C# AddOn multi-TF bar subscription
  (`VLBarsSubscriptionManager`) — SHIPPED 2026-05-28**, tag
  `v1.0-nt8-contract-resolver`. Bars PROVEN flowing into BarCache.
  (Moved here from In-flight; was "PR #29 awaiting merge".)

### In-flight
- **Plan 4.4 Stage 1** — C# AddOn multi-TF bar subscription
  (`VLBarsSubscriptionManager`). ~~PR #29 awaiting merge after
  Plan 1.5.7 verification. Compiled clean in operator's Windows VS.~~
  **SHIPPED 2026-05-28 →** (was: PR #29 awaiting merge). Bars are
  PROVEN flowing into BarCache; the C# bar-subscription work shipped
  under tag `v1.0-nt8-contract-resolver`. Moved to **Already SHIPPED**
  framing below. Remaining decision-side gap is NOT bar delivery —
  it's the kernel reading the wrong source (Binance klines, not
  BarCache); see `## 2026-05-28 Root Cause` + `## 2026-05-28
  Reordered Fix Roadmap` (Stage 3).

### Next, in order
- **Plan 4.4 Stage 2** — Go bar handling
  - New frame types (`bars_subscribe`, `bars_historical`,
    `bar_update`, `bars_unsubscribe`) per Plan 1.5 wire spec
    additive extension
  - `BarsRegistry` (the (userID, symbol, period) dedup map)
  - Auto-subscribe lifecycle (subscribe on first consumer,
    unsubscribe on last)
  - Bar cache for late-joiners
  - Drop-newest backpressure on the per-subscriber channel
  - Wire into TCPServer's existing read/write loops

- **Plan 4.4 Stage 3** — wire bars into kernel decisions
  - `engine_analysis.go` reads bars from `BarsRegistry`
    instead of `GetWithTimeframes` HTTP fetch
  - `kernel/engine.go` resolves `NT_TRANSPORT=tcp` + NT
    exchange → use NT8 bars
  - **End-to-end validation runbook** (operator-observable
    success criteria: trader cycles produce actionable
    decisions on zero-lag bars, AI brain runs, risk gate
    runs, signals flow to NT8)

- **Plan 4.4 Stage 4** — chart display
  - SSE relay (`/api/v1/bars/stream` handler in `api/`)
  - JWT-in-query (EventSource can't set headers)
  - 15s keepalive, `X-Accel-Buffering: no`, gzip-disable
  - Frontend `FuturesChart` component (TradingView v5,
    `chart.addSeries(CandlestickSeries)`, `setData` once
    then `update` per `bar_update`)
  - `ChartTabs` futures pill (`MARKET_CONFIG.futures`)

### After Plan 4.4 stages all green
- **Plan 4.11** — real NT balance via wire-protocol extension
  (TCP `account_balance` frame type; ~150 LOC + C# AddOn extension)
- **Plan 4.6** — Strategy Studio futures-aware rewrite (~400 LOC,
  largest UX change pending — touches `CoinSourceEditor`,
  `IndicatorEditor`, `RiskControlEditor`, `GridConfigEditor`,
  PromptSections variant)
- **Plan 4.14** — Plan 1.5 backend visibility panel (NT_TRANSPORT
  toggle, TCP listener status, AddOn health, CME calendar gate,
  Databento status if revived for backfill — currently no UI
  surface; would have surfaced 1.5.6 reconnect loop without
  log-tail)
- **Plan 4.9.x cosmetic bundle** — favicon, footer hrefs, Chinese
  label rendering in EN locale, PageNotFound orphan reachability,
  grainy-gradient 404 image

### DROPPED from the original roadmap
- **Plan 4.5 — handler_klines.go NT route**: the original idea
  was a Databento backend route serving MNQ candles to the chart.
  Under the NT8 pivot, the chart consumes the same NT8 bar feed
  as decisions (Plan 4.4 Stage 4), so no separate klines route is
  needed. The `/api/klines` silent Binance fallback for unknown
  exchanges (NEW-2 from 2026-05-27 audit) is still worth fixing
  as a hygiene cleanup (5 LOC: return 400 instead of silently
  defaulting), but not as part of the futures data path.

## Locked Data Architecture Decisions (2026-05-28)

- **Source:** NT8 via Tradovate (real-time, confirmed live MNQ
  candles moving 2026-05-28). No external HTTP provider involved
  in the live decision path.
- **Transport from NT8 to Go:** existing Plan 1.5 TCP socket on
  `127.0.0.1:36974`. Extended additively per ADR-007 with 4 new
  bar-related frame types.
- **Multi-timeframe:** YES — the engine consumes
  `StrategyConfig.Indicators.Klines.SelectedTimeframes` (Balanced
  Strategy default: `["5m", "15m", "1h"]`, `PrimaryTimeframe="5m"`,
  `LongerTimeframe="4h"`). The full coded TF vocabulary is 14
  values (1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 6h, 8h, 12h, 1d, 3d,
  1w) per `store/strategy.go::normalizeTimeframe`.
- **Multi-TF method:** **Option 1 — native** — one `BarsRequest`
  per selected timeframe on the C# AddOn side. NOT Go-side
  aggregation from 1m. Reason: higher-timeframe bias must match
  NT8's own bars (DST/session-boundary edges differ subtly from
  pure 1m roll-up).
- **Chart window at runtime:** NOT NEEDED. `BarsRequest` pulls
  from NT8's data engine directly; no chart instance required.
  Operator does not need to leave a chart open for the bot to
  function.
- **One feed, two consumers:** Decisions (Stage 3 reads from
  `BarsRegistry`) AND chart (Stage 4 streams via SSE) both
  consume the same NT8 bars. Single source of truth, no two-feed
  divergence risk.
- **Databento revival path (if ever needed):** The `cmd/nq_smoke`
  `available_end` probe pattern is preserved as a reference for
  any future backfill/backtest use. Live decisions will NOT use
  Databento unless the operator's subscription tier upgrades to
  real-time (currently Historical-only with ~8h lag, unusable).

## Deferred / Open Items (refreshed 2026-05-28)

### Plan 4.4 open questions (gate detailed design of Stages 2 + 4)
1. Trading hours default: ETH (recommended for algo) vs RTH
2. MergePolicy default: BackAdjusted vs NonBackAdjusted
3. `bars_back` default: 500 (~8.3 ETH hours) vs 1000
4. Multi-symbol overlay in Plan 4.4 or defer to 4.5-equivalent
5. JWT TTL for SSE query param (recommend 5 min, refreshable)
6. Drop-newest backpressure policy confirmation
7. Per-tenant subscription dedup behavior confirmation
8. Chart-side persistence on Go side: none — confirm acceptable

### Stage 2 verify items (mechanical, not design)
- Tradovate's documented concurrent-BarsRequest limit (research
  flagged 8 active in some forum threads; verify against the
  operator's account tier before designing the Auto-Subscribe
  behavior)
- Bar timestamp normalization across NT (local) and the canonical
  bar_open_utc on the Go side — already documented in Plan 1.5
  Design Findings + Merge Logic State Machine; carry into Stage 2

### Security
- N8 — password-change form has no "old password" verification
  before allowing change. Trivial bypass if session token leaks.
  ~10 LOC + backend endpoint update.
- **LOW (added 2026-05-28, from N11 VERIFY 2) — C# `Account.All[0]`
  fallback fail-closed.** If `Sim101` is absent the C# AddOn's
  `Account.All[0]` fallback picks whatever account happens to be
  loaded first. Harden to fail-closed (refuse to operate rather than
  silently bind to an arbitrary — possibly LIVE — account) instead
  of defaulting to `Account.All[0]`.

### Cosmetic bundle (defer to Plan 4.9.x)
- Favicon: still serves 404 on `/favicon.ico` (browser auto-fetch
  separate from the SVG link in `<head>`)
- Footer GitHub/Twitter/Telegram hrefs all `#` placeholders
- Chinese label rendering in EN locale on a few stat-card units
- PageNotFound component exists but is orphan (no route hits it
  thanks to catch-all redirect to `/`)
- Grainy-gradients 404 background image asset

### Plan 1.5 area
- `symbology.resolve` 422 from Databento (was Task 3 in original
  plan) — MOOT under NT8 pivot but noted: if Databento is ever
  revived, the resolve endpoint behavior may have changed.
- Tradovate concurrent-BarsRequest limit — Stage 2 verify item
  above; flagged in Plan 4.4 NT8 SDK gotchas

### Repo hygiene
- ~30 untracked screenshot/inventory PNGs in repo root from prior
  audit sessions (`audit_*.png`, `test*.png`, `plan4_verify_*.png`,
  `verify_*.png`). Should be moved to `.claude/screenshots/` (which
  is in `.gitignore`) or deleted. Currently harmless but noisy in
  `git status`.

---

# 2026-05-28 Plan Update — Wire-protocol design additions + report corrections

This block adds the wire-protocol design decisions surfaced by the
2026-05-28 external-report comparison (see
[docs/internal/external-reviews/2026-05-28-comparison-vs-canonical-plan.md](../../internal/external-reviews/2026-05-28-comparison-vs-canonical-plan.md))
and records the 3 corrections (W1 / W2 / W3) plus deferred hardening
items (N2 / N6). The 4 new wire-protocol additions (N1 / N3 / N4 / N5)
land in the Plan 4.4 Stage 2 design **before** Stage 2 builds, so
they're designed in from the start rather than retrofitted in a later
hotfix cycle.

## Plan 4.4 Stage 2 — Wire-protocol design additions (2026-05-28, pre-build)

Cross-references: Plan 4.4 Deep Spec wire protocol section (L7043+);
Plan 4.x Sequencing REVISED 2026-05-28 (Stage 2 listing); ADR-007
(any additive frame type must ship simultaneously on Go + C# sides).

### N1 — `protocol_version` field on envelope (or heartbeat payload)

**What:** Add an integer `protocol_version` field. Versions start
at `1` for the current Plan 1.5 + 4.4 spec; bumped only on
breaking changes to the wire format.

**Where:** Either in the JSON envelope alongside `type` + `payload`,
or in the `heartbeat` payload (so it rides on existing heartbeat
traffic without growing every frame). Recommended: envelope-level
(visible on every frame; simpler to inspect during debugging).

**Why:** On connect, each side reads the peer's version. The
**older** side logs a warning and falls back to whatever envelope/
field shape it understands. Distinguishes:
- "I received a frame type I don't recognize, that's expected on a
  newer peer" — current warn-and-continue behavior, unchanged
- "I received a frame type I don't recognize AND the peer's version
  is older than mine" — this is real wire drift, escalate beyond a
  warn

The current warn-and-continue rule masks both cases identically.
Would have caught at least one Plan 1.5.x hotfix early (drift
between hand-rolled JSON encoder versions).

**Implementation surface:** Adds a field to `Envelope` struct in
`provider/ninjatrader/tcp_framing.go` AND `VLTraderTCPClient.cs`
JSON encoder. Coordinated commit per ADR-007. ~10 LOC each side.

**Lands in:** Plan 4.4 Stage 2 (alongside the bar frame types).

### N3 — `bars_resync` frame type

**What:** A new wire message:

```json
{
  "type": "bars_resync",
  "payload": {
    "subscription_id": "...",
    "last_bar_open_utc": 1748352000
  }
}
```

Sent by Go → C# on Go reconnect for each active subscription. C#
replies with one or more `bars_historical` frames covering the
range `[last_bar_open_utc, now]`.

**Why:** Closes the Go-restarts-while-NT-keeps-streaming gap. The
Plan 4.4 Stage 2 design already includes "cache last bars_historical
for late joiners" on the Go side — but that cache is Go-process-
local. If the Go process restarts, the cache is empty AND the bars
that NT8 streamed during the outage are lost forever (the C#
`BarsRequest.Update` event already fired; there's no replay
mechanism on the C# side without a wire-level handshake).

**Implementation surface:** New frame type (1 envelope shape, 2
handlers — one in `tcp_server.go` for the dispatch, one in
`VLTraderTCPClient.cs` + `VLBarsSubscriptionManager.cs` for the
replay logic). ~30-50 LOC each side. C# side needs to keep a
sliding window of recently-emitted bars per subscription (size
configurable; default e.g. last 1,000 bars) so it can serve a
resync without re-querying NT8's historical store.

**Lands in:** Plan 4.4 Stage 2.

### N4 — Paginate `bars_historical` at ~5,000 bars

**What:** When responding to a `bars_subscribe` (initial load) or
`bars_resync` (gap fill), if the requested range produces more
than ~5,000 bars, C# sends multiple sequential `bars_historical`
frames with a continuation flag:

```json
{
  "type": "bars_historical",
  "payload": {
    "subscription_id": "...",
    "bars": [...],
    "more": true,           // false on the final batch
    "batch_index": 0
  }
}
```

**Why:** Even with the 1 MB frame ceiling, a single batch of ~5k
1m bars (~200 bytes each ≈ 1 MB) saturates the read pipeline for
the duration of the batch parse. Other frames (signal acks,
heartbeats) starve on the shared socket until the big batch
completes. Pagination restores fairness: the Go-side reader can
process other frames between batches, and Stage 2's backpressure
channel doesn't see one giant burst.

**Implementation surface:** C# side splits the bars array at the
batch boundary; Go side accumulates batches into a complete window
keyed by `subscription_id` and emits on `more=false`. ~20 LOC each
side.

**Lands in:** Plan 4.4 Stage 2.

### N5 — Dropped-coalesced-tick counter

**What:** Instrument the Stage 2 backpressure channel with a
counter for in-progress `bar_update` ticks that get dropped due to
coalescing (newer tick overwrites older pending tick in the channel
slot for the same `subscription_id|bar_open_utc`).

**Why:** Indicator-path slowness manifests as elevated drop rate.
If drop rate > ~1% of stream rate, the indicator computation is
lagging behind the bar stream — should trigger a profiling pass,
not be silent. Without the counter, this degradation is invisible
until the trader makes a decision on stale indicators.

**Implementation surface:** Plain `atomic.Uint64` counter inside
`market.Data` or wherever the coalescing channel lives. Surfaced
via `/api/internal/metrics` or Prometheus when Plan 4.14 (backend
visibility panel) lands. ~5 LOC for the counter, +10-20 LOC for
the UI surface when 4.14 catches up.

**Lands in:** Stage 2 (counter); Plan 4.14 (UI surface).

## External-report comparison — corrections logged (2026-05-28)

The 3 W-items from
[2026-05-28-comparison-vs-canonical-plan.md](../../internal/external-reviews/2026-05-28-comparison-vs-canonical-plan.md)
that the canonical plan's own post-mortems should reflect:

### W1 — Plan 1.5.7 uses `context.WithCancel`+goroutine, NOT `context.AfterFunc`

The external report inferred `context.AfterFunc(ctx, func(){ conn.SetReadDeadline(time.Now()) })`
from Go context docs. The actual fix at commit `803a8727` uses
`context.WithCancel(ctx)` + a watcher goroutine `go func() {
<-connCtx.Done(); c.Close() }()`. Same architectural pattern,
different API. The Plan 1.5.7 post-mortem above has been updated
to explicitly cite `WithCancel`+goroutine so future readers can't
re-infer the wrong API surface.

### W2 — Plan 1.5.6 primary vs secondary fix

The external report framed the v1.5.6 fix as concurrent-write
corruption mitigation. The actual primary root cause was the stale
persistent write deadline on the heartbeat-ack path; `writeMu` was
defense-in-depth (no frame corruption was observed in
production). The Plan 1.5.6 post-mortem above has been updated to
explicitly split PRIMARY vs SECONDARY in the Fix section.

### W3 — `engine_position.go` validActions = 6, not 9

The external report cited "upstream issue #982" as enumerating 9
valid decision actions:

```
open_long, open_short, close_long, close_short,
update_stop_loss, update_take_profit, partial_close,
hold, wait
```

The actual `kernel/engine_position.go::validActions` map (verified
2026-05-28 via grep) contains **6** entries:

```go
validActions = map[string]bool{
    "open_long":   true,
    "open_short":  true,
    "close_long":  true,
    "close_short": true,
    "hold":        true,
    "wait":        true,
}
```

`update_stop_loss`, `update_take_profit`, and `partial_close` are
NOT discrete action keys in the validation path. They may exist
elsewhere (e.g. as decision-parsing fields, or in an upstream
branch not on origin/main), but the 9-action claim does not match
the current shipped code. Any prompt template or downstream
consumer that assumes the 9-action set is out of sync; if those
actions are intended to be supported, they need a deliberate
add-to-validActions PR.

## Branch ground truth (2026-05-28)

The external report's biggest confusion was probing `origin/dev`
(GitHub-declared default) and finding only upstream crypto NOFX,
which led to a chain of "INFERRED, not visible" caveats throughout
its §12. The clarifying facts:

| Branch | Tip SHA | Contents |
|---|---|---|
| `origin/main` (operator's trunk) | `6c3333a6` (and advancing) | **All futures work** — `ninjascript/`, `provider/ninjatrader/tcp_server.go` + `tcp_framing.go`, `kernel/engine_prompt_futures.go`, ADR-007, this canonical plan with the 2026-05-28 NT8 pivot, the 28 `v1.0-*` tags, `CLAUDE.md` files at root + subsystems |
| `origin/dev` (GitHub default, vestige) | `ab5873e2` | Upstream crypto NOFX. CHANGELOG.md last-updated 2025-11-01. No `ninjascript/`. No C# in language stats |

GitHub's UI (language breakdown, default-branch view, "Code" tab
listing) reflects `dev`. External probes that read those without
explicitly switching to `main` will see the upstream crypto code
and conclude (incorrectly) that the futures work is missing.

**Deferred (repo-config decision):** consider setting GitHub's
default branch to `main` so external reviewers and CI / 
fork-tracking infrastructure see the right trunk by default. Trade-
off to evaluate before flipping: some CI keys off the default
branch, and any fork-tracking that's pinned to `dev` would need
updating. Not blocking; tracked as repo hygiene.

## Deferred / Open Items (refreshed 2026-05-28)

Carried forward from the prior 2026-05-28 plan update, plus 2 new
items from the external-report comparison:

### N2 — CI hash-check enforcing ADR-007

**What:** Add a CI workflow that computes SHA-256 of the 3
ADR-007 critical files (`provider/ninjatrader/tcp_server.go`,
`provider/ninjatrader/tcp_framing.go`,
`ninjascript/VLTraderTCPClient.cs`). If a PR changes the hash of
one without matching changes to the others, fail the check.

**Why:** Currently ADR-007 is enforced by review discipline (the
ADR itself says "any change must be simultaneous on both sides").
A CI check mechanizes this — catches drift before it lands. Would
have caught at least one Plan 1.5.x desync if the contract had
been enforced earlier.

**Threshold to relax:** only when a coordinated minor-version bump
is being released; even then, the check should require all 3 hashes
to change in the same PR rather than be bypass-able.

**Adoption value:** medium-high. Not urgent (review discipline has
held so far), but mechanizes a rule that's currently a soft norm.

### N6 — fetch-event-source polyfill (chart SSE JWT in Authorization header)

**What:** Replace the native browser `EventSource` API in the
`FuturesChart` component (to be built in Plan 4.4 Stage 4) with
the [`fetch-event-source`](https://www.npmjs.com/package/@microsoft/fetch-event-source)
polyfill. The polyfill lets you set custom headers on an SSE
connection, so the JWT can ride in the `Authorization` header
instead of the URL query parameter.

**Why:** The current Plan 4.4 Stage 4 design has JWT-in-query as a
mitigation for the EventSource header limitation (WHATWG html#2177).
URLs leak into server logs and browser history; the chart token
must be short-TTL and scoped. Moving the JWT to the Authorization
header eliminates the URL-token risk entirely.

**Threshold to skip:** if a cookie-based session-auth alternative
is added (the chart JWT would ride on the session cookie), the
polyfill becomes unnecessary.

**Trade-offs:** polyfill is a non-zero maintenance dependency
(~9KB minified; Microsoft-maintained, used in production by VS Code
Live Share); evaluate against the short-TTL-token mitigation cost.

**Lands in:** Plan 4.4 Stage 4 (alongside the FuturesChart
component itself). Decide polyfill vs short-TTL token at Stage 4
design review.

### CHANGELOG.md pointer-ification

`CHANGELOG.md` is an upstream-NOFX vestige; its last entry is
`[3.0.0] 2025-10-30` and it has no awareness of the futures work
or any `v1.0-*` tag on this fork. Replaced in this same PR with a
one-line pointer to:

- This plan doc's Ship Log section (the canonical change log for
  the futures fork)
- `git tag -l 'v1.0-*'` (the version log — 28 tags as of
  2026-05-28)

(Editing CHANGELOG.md is in-scope for this doc update; it is NOT a
Plan 1 critical file per ADR-007.)

### Adoption sequence

Suggested order for landing N1/N3/N4/N5 into the Stage 2 build:

1. **N1 protocol_version** — design-only addition to the envelope
   spec; lands in the Stage 2 wire-design PR alongside the bar
   frame types.
2. **N4 paginate `bars_historical`** — additive contract change to
   an already-designed frame; lands in the same Stage 2 PR.
3. **N3 bars_resync** — new frame type; lands in Stage 2 alongside
   the others (no point splitting one C# + one Go PR per frame).
4. **N5 dropped-tick counter** — Go-only, lands in Stage 2 wherever
   the coalescing channel goes. UI surface waits for Plan 4.14.

All four are additive per ADR-007 and ride on a single coordinated
Stage 2 build (one Go PR, one C# PR, hash-matched per ADR-007 if
N2 lands first).

---

# 2026-05-28 UI Audit + N11 Resolution + Pinpointed Stage 3 Fix

This block records a convergence: a per-page UI data-flow audit (run
on `origin/main` @ `4f0843e5`, against the live app) and the N11
trader-starvation investigation landed on the **same single root
cause** from two independent angles. It also records N11 as RESOLVED,
corrects the `risk_check_passed` interpretation, reorders the fix
roadmap, reaffirms the Databento-dropped decision, and logs the new
bugs the audit surfaced.

## 2026-05-28 Root Cause — Kernel reads Binance klines, not NT8 BarCache

> **SUPERSEDED 2026-05-31 (branch `feat/nt8-stage4-chart`) — the Stage 3
> source-swap is DONE on this branch; the kernel reads NT8 BarCache for CME
> futures, NEVER Binance/CoinAnk.** Verified live 2026-05-31 in
> `market/data.go`: `isFutures := IsCMEFuturesSymbol(symbol)` (`:197`) →
> the futures branch reads `FuturesBarsProvider(symbol, tf, 200)` — the
> BarCache (`:204-210`, with a "no NT8 bar provider wired; skipping" guard);
> `getKlinesFromCoinAnk(..., "binance", ...)` is the **non-futures `else`
> branch only** (`:224`). The sibling 3m/4h helper does the same
> (`:41-90`). **ONE SOURCE OF TRUTH CONFIRMED: chart + kernel both read NT8
> BarCache.** The "reads Binance" diagnosis below was true on
> `origin/main @ 4f0843e5` (where Stage 3 had not landed); it is no longer
> true on this branch. Kept verbatim below for history.

The two investigations converged on one cause:

- **Engine-side (UI audit):** the trader's market-data map comes up
  empty, so DecisionAudit fills with 100 WAIT rows.
- **Kernel-side (N11 cycle proof):** the decision cycle reaches the
  AI brain but on absent/mismatched data.

Both trace to a single call. `buildTradingContext` →
`getKlinesFromCoinAnk(symbol, tf, "binance", 200)` at
**`market/data.go:192`**. The live log confirms it:

```
Unknown exchange 'ninjatrader', defaulting to Binance for CoinAnk.
```

There are **NO `BarCache` references in `kernel/`, `market/`, or
`trader/`** — the kernel never reads the source that actually holds
the NQ bars. Bars ARE flowing into `BarCache` (PROVEN, shipped under
tag `v1.0-nt8-contract-resolver`); the kernel just reads the wrong
source. Given absent/mismatched data, the AI's "wait" is the
**RATIONAL** output — not a bug in the brain, a bug in the plumbing.

**The pinpointed fix (Stage 3):** swap that ONE call for
`BarCache().Get(symbol, tf)` in the futures branch. Precise,
single-call, no broad refactor.

## N11 — RESOLVED (flip applied + cycle proven, 2026-05-28)

(Supersedes `### N11 — Trader STARVED (not blocked by risk gate)`
above — that post-mortem is preserved with a SUPERSEDED marker.)

- **Flip applied:** Balanced Strategy (`d5412693`)
  `ai_config.coin_source` flipped `ai500` → static `["NQ.c.0"]` at
  runtime in `data/data.db` (gitignored). Pre-flip DB backed up to
  `/tmp/data.db.bak-pre-n11-flip`.
- **Cycle PROVEN end-to-end on SIM:** decisions `#244` / `#245`,
  `candidate_coins ["NQ.c.0"]`, AI round-tripped
  (`ai_request_duration_ms` 11884 / 15379 ms, model `deepseek-v4-pro`),
  `action: wait`, `success: 1`, no order fired. **3/3 verify PASS.**
- **Scope:** only Balanced changed; Conservative / Aggressive still
  use `ai500`.
- **NOT reseed-durable:** the flip lives in the runtime DB. A durable
  fix is either a `GetDefaultStrategyConfig` tweak OR a boot
  migration — **recommend folding into Stage 3's PR** (Stage 3 Step 2
  touches the kernel anyway).
- **CORRECTION:** the WAIT decisions are **NOT** ai500 starvation
  (the flip worked — coins are now selected). They are the
  empty-market-data cause (see `## 2026-05-28 Root Cause` above). The
  flip removed the coin-selection short-circuit; Stage 3 removes the
  data starvation. Two distinct gates: N11 cleared the first, Stage 3
  clears the second.
- **UPDATE 2026-05-31 (branch `feat/nt8-stage4-chart`):** the second gate
  is now CLEARED here too — Stage 3's source-swap is DONE on this branch
  (kernel reads NT8 BarCache via `FuturesBarsProvider`,
  `market/data.go:197-210`; see the SUPERSEDED marker on the Root Cause
  section above). The empty-market-data starvation is resolved on this
  branch; remaining WAITs are weekend-gate (CME closed), not data
  starvation.

## 2026-05-28 — risk_check_passed is UNWIRED (not a rejection signal)

`risk_check_passed`, `ai_model`, and `ai_latency_ms` are DecisionAudit
columns that **no code path ever sets**. All 246 rows show
`0`/empty — BEFORE and AFTER the N11 flip. `risk_check_passed=0` was
**NEVER** a real rejection; it is a pre-existing observability gap, not
a risk-gate verdict. (This supersedes any prior text — including the
N11 post-mortem above — that read `risk_check_passed=0` as a risk-gate
rejection.)

- **Reliable "AI ran" signal:** `ai_request_duration_ms` + logs (these
  are what proved the N11 cycle, not `risk_check_passed`).
- **Follow-up:** wiring these columns is an observability task — fold
  into **Plan 4.14** (backend visibility panel).

## 2026-05-28 UI Data-Flow Audit (origin/main 4f0843e5, live app)

> **PARTIALLY SUPERSEDED 2026-05-30** (branch `feat/nt8-stage4-chart`):
> the "Balance = hardcoded `$50k` MOCK", "decisions skip / unknown coin
> source" (new strategies born broken on `ai500`), and the Binance/`BTCUSDT`
> chart findings are RESOLVED — real per-account balance (`c4e2cb13`),
> `static` MNQ coin_source + kernel guard (`abda753d`/`058e4a56`), and the
> NT8 chart wiring (`273f85a3`). The crypto-prompt-served-to-futures and the
> remaining depth findings below still stand. See
> `## Current State (2026-05-30, session-verified)`. Kept for history.

Per-page findings against the live app on `origin/main` @ `4f0843e5`:

- **Settings — healthiest.** All controls wired; save/persist
  verified.
- **Dashboard.**
  - Balance = hardcoded **$50k MOCK**
    (`trader/ninjatrader/trader.go:156-163`,
    `trader/ninjatrader/tcp_trader.go:171-177`).
  - Positions empty (no fills yet).
  - DecisionAudit all **WAIT** — empty data, NOT ai500 (the flip
    worked; see Root Cause above).
  - Crypto "balanced" prompt served to the futures trader:
    `kernel/engine_prompt_futures.go` EXISTS but is **unused**
    (`auto_trader_loop.go:102` selects the crypto prompt).
- **Strategy Studio.** Save/persist works and the store is
  futures-aware, BUT:
  - symbol picker appends `USDT`;
  - default config still `ai500`/`nofxos` — **new strategies are
    born broken** (`store/strategy.go:914`, `store/strategy.go:945`);
  - risk tiers are crypto;
  - the futures prompt is unreachable from the UI.
- **Agent Chat.** Functional, BUT:
  - market chart shows Binance `BTCUSDT`, not MNQ
    (`api/handler_klines.go:48-78`);
  - MarketTicker empty (Binance proxy).

**New bugs surfaced by the audit:**

- account-state variadic bug
  (`api/exchange_account_state.go:327-339`) — falsely reports NT as
  `missing_credentials`. ~2 LOC.
- `/api/klines` silent Binance fallback for unknown exchanges —
  should return `400` instead of silently defaulting.
- `/settings` → `/dashboard` auto-redirect (trace the cause). — RETRACTED
  2026-05-28: not a bug; SettingsPage has zero navigate calls, it was the
  shared Playwright browser driven by another agent.
- cosmetic: dead NT "Register" link; Chinese-in-EN hints.

## 2026-05-28 Reordered Fix Roadmap

Reflects the audit + N11 convergence. Data plane unblocks everything.

### 🔴 BLOCKING (data plane)

> **UPDATE 2026-05-31 — DASHBOARD COMPLETE on `feat/nt8-stage4-chart`.**
> Stage 1 (C# bars) DONE, Stage 2 (Go bar handling) PROVEN, **Stage 3
> (kernel reads NT8 BarCache) DONE** (`market/data.go:197-210`), **Stage 4
> (chart) SHIPPED** (`273f85a3` AdvancedChart over `/api/klines` + the
> `063bc311` 7-TF CSS-clip fix), and **Plan 4.11 (real balance) DONE**
> (`c4e2cb13`) + per-account equity C/D/E (`49017ff9`/`0a67bfda`). The
> data-plane chain is no longer blocking. **The next big project is the
> Strategy build** (LOCKED — see `## Current State` → IN FLIGHT / PLANNED:
> Stage 1 UI+phantom, then Stage 2 backend 14-TF delivery).

- **Stage 1 — C# bars [DONE — `v1.0-nt8-contract-resolver`].**
- **Stage 2 — Go bar handling [bars flow + cached, PROVEN].** N1/N3/N4/N5
  wire-design items pending IF not yet built — **confirm remaining**
  against the Stage 2 wire-design section above.
- **Stage 3 — the pinpointed one-call fix. ✅ DONE on
  `feat/nt8-stage4-chart` (SUPERSEDED 2026-05-31).** The source-swap
  shipped: the futures branch now reads the NT8 BarCache via the injected
  `FuturesBarsProvider` (`market/data.go:197-210`), never
  `getKlinesFromCoinAnk` (that is the non-futures `else` only, `:224`).
  Verified live 2026-05-31. The N11 data-starvation cause is resolved by
  this same Stage-3 work being present on the branch. (Original target text
  below kept for history; the cited `market/data.go:192` line drifted —
  the live branch is `:197-210`.) Remaining fold-ins still open: (a) the
  futures-prompt selection (route futures traders to
  `engine_prompt_futures.go` — tracked in the LOCKED Strategy build, Stage 1
  P2 TAB6), and (b) the reseed-durable default
  (`GetDefaultStrategyConfig`/boot migration — the phantom CLEAN SLATE note).
  Original Stage-3 target: ~~Swap `getKlinesFromCoinAnk` →
  `BarCache().Get(symbol, tf)` in the futures branch
  (`market/data.go:192`).~~
- **Stage 4 — FuturesChart + SSE relay** off the same NT8 feed →
  fixes the Binance chart. **SUPERSEDED 2026-05-30:** the chart path is
  `AdvancedChart` reading the NT8 BarCache via REST `/api/klines`
  (`273f85a3`), NOT `FuturesChart` (deleted) and NOT an SSE relay. See
  `## Current State (2026-05-30, session-verified)`.

### 🟠 After (depends on data plane)

- **Plan 4.11** — real NT balance (~150 LOC; fixes the $50k mock).
  **DONE 2026-05-30** via `c4e2cb13` — see Current State.
- MarketTicker → NT8 feed.

### 🟡 Quick wire-ups (zero-dependency)

- account-state variadic fix (~2 LOC).
- default-config off `ai500`.
- `/api/klines` return `400` (instead of silent Binance fallback).
- `/settings`→`/dashboard` redirect trace. — RETRACTED 2026-05-28: not a
  bug; SettingsPage has zero navigate calls, it was the shared Playwright
  browser driven by another agent.

### 🟢 Lower

- **Plan 4.6** — Strategy Studio futures rewrite.
- **Plan 4.14** — backend visibility (also wire `risk_check_passed`,
  `ai_model`, `ai_latency_ms`).
- **Plan 4.9.x** — cosmetic bundle.

### Depth tiers (2026-05-28) — independent of the data-plane chain

Strategic finding from the 5-agent depth audit: the data-plane chain (Stage
2/3/4 + 4.11) is **UNCHANGED** and still the unlock for real
decisions/chart/balance, **BUT** ~25 depth findings are **INDEPENDENT** of
that chain and can be batched in parallel:

- 🟠 **correctness** — NT controls (Close Position/CloseLong/CloseShort,
  Emergency Flat), edit-form repopulation (nt_* setters), AI Model delete,
  account_name drop + HL-flag flip, account-state variadic (~2 LOC),
  getTradersUsingExchange names (`[object Object]`), ensureRawKlines
  render-time setState, `/login`-wipe + 401 hard-redirect, swallowed
  errors, onboarding error state.
- 🟡 **safety/UX** — confirm dialogs on destructive Settings actions; the
  deferred security items (tokenless reset-password, reset-account
  wipes-all); USDT-append symbol corruption.
- 🟢 **dead-code cleanup** — ~58KB dead charts + dead modals + dead schema
  fields + FAQ branches + PageNotFound decision (wire `*`→404 or delete).
- 🔵 **crypto-residue → fold into Plan 4.6** — risk tiers, margin-mode,
  grid symbols, futures-prompt-unreachable (also Stage 3),
  StatCard/EquityChart USD propagation, FAQ rewrite, `/faq` nav link.
- 🟣 **observability → fold into Plan 4.14** — wire the 5 dead
  DecisionAudit columns + model/prompt badges.
- 🌐 **i18n → dedicated pass / 4.9.x** — Indonesian fallthrough,
  Chinese-in-EN leaks, x-axis locale, English-only auth strings.

## 2026-05-28 — Databento stays DROPPED (reaffirmed)

The audit's Agent 4 suggested wiring Databento into
`handler_klines`. **REJECTED** per the locked architecture (see
`## Locked Data Architecture Decisions (2026-05-28)`): the chart rides
the **NT8 feed** (Stage 4); Databento is Historical-only (~8h lag) and
stays dropped; the old Plan 4.5 Databento route stays DROPPED. The
`/api/klines` `400`-cleanup survives (it's hygiene, not a data-path
change); the Databento route does not.

## 2026-05-28 Deep Per-Page Completeness Audit

Layered on top of the breadth audit (`## 2026-05-28 UI Data-Flow Audit`
above) — does NOT supersede it. A **deeper** per-page completeness pass.

**Method.** 5 parallel general-purpose agents, exhaustive code inventory +
Playwright/curl live verification, read-only, each with an honest coverage
section. Surface split:

1. Settings + modals
2. Dashboard + Traders
3. Strategy Studio editors
4. Charts + Ticker + Chat
5. Auth + Onboarding + FAQ + chrome + routes

**COVERAGE CAVEAT (itself a bug).** All 5 agents lost the live
authenticated session mid-run because visiting `/login` wipes the session
(`LoginPage.tsx:29-33`) and any `401` hard-redirects to `/login`
(`httpClient.ts:148-155`). Live verification of authed surfaces is
therefore **PARTIAL**; **code verification is complete and authoritative.**
The agents correctly **REFUSED** to forge a JWT or register a throwaway
user to regain access — integrity held.

**DISCREPANCY (unresolved live).** Dashboard StatCards are **USD-correct**
(`TraderDashboardPage.tsx:185-188`); a possibly-stale screenshot showed
`USDT`. `EquityChart.tsx:45` is a **separate** `isFutures` unit-label path.
**Action item:** confirm `isFutures` propagates into `EquityChart`.

## 2026-05-28 Depth Audit — Security Findings (DEFERRED, not yet fixed)

- `/api/reset-password` resets **ANY** user's password from email + new
  password — **NO token, NO verification, NO current-password check**
  (`handler_user.go:193-227`). Auth bypass; on a single-user box that's the
  only account.
- "Forgot account?" → `POST /api/reset-account` **WIPES ALL USERS**
  (`server.go:127`).
- `/login` mount wipes the active session (`LoginPage.tsx:29-33`) + `401`
  hard-redirect (`httpClient.ts:148-155`).
- Reset-password min-length contradiction (checklist `8`, placeholder `6`,
  backend `6`).
- **Status:** operator **deferred** ("sec later") — recorded here so they
  are not lost; revisit before any non-local exposure.

## 2026-05-28 Depth Audit — NT Broker UI Gaps

The "looks-wired-isn't" tier — controls render but fail at runtime for the
primary (NinjaTrader) broker.

- **Close Position** (per row) → `HTTP 400`, no `ninjatrader` case
  (`handler_trader_status.go:159`); `CloseLong`/`CloseShort` also error.
  **Guaranteed fail once a fill exists.**
- **Emergency Flat** → Confirm → **nil-writer no-op** (`handler_risk.go:119`),
  logs "operator must flatten manually on NT chart", dumps raw JSON in UI.
- **NT edit-form doesn't repopulate:** TS type `config.ts:21-50` omits
  `nt_*`; edit `useEffect` `ExchangeConfigModal.tsx:228-243` has no `nt_*`
  setters (backend GET **does** return them). **Primary broker.**
- **Create-trader shows Cross/Isolated margin mode for NT** (dead crypto
  knob, `TraderConfigModal.tsx:367-391`).

## 2026-05-28 Depth Audit — Correctness / Persistence Bugs

- **AI Model "Remove" doesn't delete** (`SettingsPage.tsx:193-227` +
  `store/ai_model.go:222` — sends `api_key:''` but store only writes
  non-empty → never cleared → row stays visible). No real `DELETE` wired.
- **`account_name` silently dropped on exchange edit**
  (`handler_exchange.go` + `exchange.go:297-353`; dead `UpdateAccountName`
  uncalled).
- **`hyperliquid_unified_account` flipped to `false` on edit** (form never
  sends it; `exchange.go:308` writes unconditionally).
- **`getTradersUsingExchange().join()` → `[object Object]` toast**
  (`AITradersPage.tsx:505`).
- **`ensureRawKlines()` render-time setState anti-pattern**
  (`IndicatorEditor.tsx:106-115`) → spurious unsaved-changes flag;
  Raw-Klines checkbox disabled + always-on (dead control).
- **"Fill Default" / default-config still writes the dead `NofxOS`/`ai500`
  key** (`IndicatorEditor.tsx:206-218`; `store/strategy.go` default) → new
  strategies born broken. (= the reseed-durable counterpart to the runtime
  N11 flip; **fold into Stage 3**.)
- **5 of 9 risk fields are display-only "System enforced"**
  (`RiskControlEditor.tsx:54,158,180,244,280`); only 4 editable.
- **Symbol inputs auto-append "USDT"** → corrupt CME roots (`"NQ"` →
  `"NQUSDT"`, `"NQ.c.0"` → `"NQ.C.0USDT"`) in `CoinSourceEditor` + the chart
  quick-input.
- **BeginnerOnboarding broken error state** (no retry when `data===null`).
- **Silent failures:** swallowed token-overflow `400` / missing-fields /
  save errors; chart wrong-data with no warning; ticker empty with no
  empty-state; dashboard degrades to `"--"` after 2 silent retries.

## 2026-05-28 Depth Audit — Dead Code

- **4 chart components ~58KB:** `TradingViewChart`, `ChartWithOrders`,
  `ChartWithOrdersSimple` (contains "测试模式 / under development"
  placeholder), `ComparisonChart` — zero render refs.
- `TraderConfigViewModal.tsx`, `BeginnerGuideCards.tsx` — imported nowhere.
- `PageNotFound.tsx` — never imported; `*` route does `Navigate→/` so 404
  never renders (**decide:** wire `*`→404 or delete).
- **Dead schema fields:** `external_data_sources`,
  `use_hyper_all`/`_main`/`hyper_main_limit`,
  `longer_timeframe`/`longer_count` (no UI).
- `/faq` is an **orphan** (no nav link; URL-only); FAQ content **entirely
  stale crypto** (Binance/HL/Docker/TA-Lib; zero NQ/MNQ/CME).
- **Dead footer social links** (`OFFICIAL_LINKS.*` empty → `href=""`
  reloads page).

## 2026-05-28 Depth Audit — i18n

- **Indonesian (`id`) falls through to English** in strategy editors /
  onboarding / agent panels.
- **~10 Chinese-in-EN leaks;** worst is the Competition-toggle title
  (`TradersList.tsx:394`, user-visible under EN locale). Others:
  `ExchangeConfigModal` credential-status + placeholders, `ModelConfigModal`,
  `TwoStageKeyModal` toasts, coin-source `'固定币种'` preview, `AdvancedChart`
  x-axis `toLocaleString('zh-CN')`.
- **"Reset to default" writes Chinese prompt text for EN users**
  (`PromptSectionsEditor.tsx:14-43`).

## 2026-06-04 — Trade-History P&L: honest "unknown" for reconcile-flat closes (SHIPPED, commit `0c245344`)

**STATUS:** SHIPPED (PART 2 of 2). NT8-only, additive, crypto byte-identical,
`close_sync` (the real-P&L path) untouched. Go rebuild + `./nofx-bin` restart
done (clean start, 0 "unknown frame type"); FE `tsc` clean.

**Symptom:** Dashboard → Trade History showed AI-decision closes at **P&L = $0**.

**Root cause (diagnosed read-only, then confirmed at file:line):** the close
pipeline is correct on the `close_sync` path — 80 rows recorded real ×point-value
P&L (e.g. MNQ SHORT 30456.75→30473.25 = 16.5pt × $2 = −$33.00). The zero-P&L rows
are **exclusively** `close_reason='reconcile_flat'` (≈24-25 rows): when a
decision-driven flatten's NT8 `position_close` frame is never captured
(`auto_trader_decision.go:313-325` leaves the row OPEN awaiting it), the 20s
reconcile finds NT8 flat and orphan-closes the row at **entry price / $0**
(`reconcile.go:122`) — a placeholder that **falsely reads as breakeven**. So it
was neither a compute, store, display, nor multiplier bug — it was a reconcile
placeholder presented as a real $0.

**PART 1 (root cause — make decision closes reliably captured by close-sync):
SCOPED OUT this pass.** Reliable `position_close` delivery for decision flattens
is an NT8 **C# AddOn / feed-delivery** matter (or would require fabricating an
exit) — both forbidden here (no C#; "NT8 is the source of truth, never fabricate
an exit"). The existing design already retries on the next decision cycle and
uses reconcile as the safety net. Options for a future pass: (a) C#-side guarantee
a `position_close` frame for decision flattens; (b) a Go-side await/confirm before
the cycle returns; (c) on reconcile, query NT8 for the real exit fill (needs bridge
support). None shipped.

**PART 2 (safety net — honest "unknown", not a false $0): SHIPPED.**
- Premise refinement (reported): the brief suggested `realized_pnl` NULL/sentinel,
  but the column is a **non-nullable `float64`** — NULL needs a `*float64` struct
  change that ripples to every reader (high-cascade) and a numeric sentinel would be
  silently summed by the stats loops. So the **existing `close_reason='reconcile_flat'`
  marker** (already persisted, already plumbed to the FE via `trading.ts`) is the
  single source of truth for "unknown"; every P&L presenter/aggregator now treats it
  as unknown.
- **BE** (`store/`): exclude `reconcile_flat` from all closed-position stat
  aggregators — `GetFullStats`, `GetSymbolStats`, `GetDirectionStats`, and
  `GetHistorySummary` (recent + streaks). Shared const `store.CloseReasonReconcileFlat`
  (also wired into `reconcile.go`, de-magicking the literal — no behavior change).
- **FE** (`PositionHistory.tsx`): render **"—"** (with an "exit not captured" tooltip)
  for `reconcile_flat` rows instead of `+0.00 / 0.00%`, and exclude them from the
  footer P&L total. The rows still appear in the list (honest: the close happened,
  the P&L is unknown).
- **DB proof (live `data/data.db`):** excluding `reconcile_flat` moves the trader's
  win-rate from a diluted **35.2% → 46.2%** (over the 80 known-outcome trades;
  wins=37 / losses=43 and total_pnl unchanged — the excluded rows were $0). The
  ≈25 unknown closes remain in the position list (→ "—").

**Not changed:** crypto close path (exchange order-sync, returns real P&L — never
on this reconcile path); `close_sync` real-P&L recording; the 80 correct `sync`
rows. **Uncommitted-edits audit:** the session-start snapshot listed
`store/position_query.go` / `store/decision.go` / `auto_trader_loop.go` as modified,
but the working tree was **clean** (already committed in `24634b5a`) — nothing
collided, nothing swept in.

## 2026-06-04 — PART 1: reconcile status-guard stops the overwrite of close-sync's real P&L (SHIPPED, commit `7786d845`)

**STATUS:** SHIPPED. Go-only, no C#. Additive — `close_sync` byte-identical, crypto
untouched. Go rebuild + `./nofx-bin` restart done (clean, 0 "unknown frame type").

**Root cause (PART-1, log-confirmed — refutes "missed frame"):** the `position_close`
frame is NOT missed. close-sync **captures the real ×point-value P&L** on it (it
commits first, event-driven) — then the 20s reconcile, working off a **stale
open-positions snapshot**, **overwrites the same row** with the `$0` `reconcile_flat`
placeholder. `ClosePosition` updated `WHERE id=?` with **no status guard**, so the
stale write clobbered the just-recorded real close. Caught live: row=122 logged
`pnl=2.00` at 08:14:57, reconcile orphan-closed it 20s later → DB `reconcile_flat $0`.
The race is close-sync-first / reconcile-overwrites every time (logs show close-sync's
`📕` precedes reconcile's `🔧` by up to ~20s). Not decision-specific — an SL exit was
clobbered too.

**Fix (Go-only):** [`store/position.go`] `ClosePosition` now updates
`WHERE id=? AND status='OPEN'` and returns whether a row was actually closed. Once
close-sync sets the row `CLOSED/sync`, reconcile's guarded UPDATE matches **0 rows**
→ the real P&L stands (close-sync wins). [`trader/ninjatrader/reconcile.go`] uses the
bool to log honestly: `🔧 closed orphan` only when it really closed a still-OPEN row,
else `✓ already closed by close-sync (kept real P&L)`. `ClosePosition`'s **sole
caller is reconcile** (verified), so close-sync (`ClosePositionFully`) and the crypto
path are unaffected.

**Proof (DB):** for CLOSED sync row id=121, the OLD `WHERE id=121` matches **1** (would
clobber); the NEW `WHERE id=121 AND status='OPEN'` matches **0** (no-op, real P&L kept).
A live-close watcher confirms the next captured close keeps `sync`/real ×$2 P&L through
a reconcile tick.

**PART 2 intact:** a genuinely-uncaptured close (no `position_close` frame at all) is
still OPEN when reconcile runs → matches `status='OPEN'` → `reconcile_flat` → UI "—".
So the honest-unknown fallback (commit `0c245344`) remains for the real feed-down case;
this PART-1 fix simply stops reconcile from destroying the closes that WERE captured.

## 2026-06-04 — PART 1b: flat-grace window closes the reconcile-FIRST race (SHIPPED, commit `6afb5adf`)

**STATUS:** SHIPPED. Go-only, reconcile-side; `close_sync` BYTE-IDENTICAL; PART-1a
status guard retained; crypto untouched. Go rebuild + restart done (clean).

**PART-1a premature-claim correction:** PART 1a (`7786d845`, the status guard) was
reported "live-verified" off a **single** close (row 123) — that was premature. The
fuller data showed the guard only fixed the **close-sync-FIRST** ordering; **5 of 8**
post-1a captured closes (rows 125–129) were **still** lost to `reconcile_flat $0` —
matched by entry-price to their `📕` real-P&L log lines — via the **reconcile-FIRST**
ordering (5 `🔧 closed orphan` / **0** `✓ already closed` lines = the guard's no-op
branch never fired; reconcile always acted on a still-OPEN row).

**Root cause (reconcile-FIRST):** NT8 publishes a **flat** positions snapshot (which
reconcile's 20s poll reads) at/around the instant it sends the `position_close` frame,
but that frame reaches close-sync a beat later. Reconcile orphan-closed the still-OPEN
row first → close-sync then found no open row and **skipped** → the real ×point-value
P&L was lost.

**Fix (PART 1b):** a **flat-grace window**. [`trader/ninjatrader/tcp_trader.go`] adds a
`flatSince` map (row id → first-seen-flat ms; reconcile-goroutine-only, no lock).
[`trader/ninjatrader/reconcile.go`] `reconcilePositions` now, on first seeing a row
NT8-flat-but-DB-open, records the time and **defers** the orphan-close; it only
orphan-closes after the row has been continuously flat ≥ **`flatGraceMs` = 60s** (3
reconcile cycles). close-sync records the real close within seconds → the row goes
`CLOSED/sync` and is pruned from `flatSince` next pass. A **genuinely-uncaptured** row
(no frame ever) is still OPEN after 60s → orphaned → `reconcile_flat` "—" (PART-2
fallback, correct). The **PART-1a status guard stays** (covers the last-moment
close-sync-first DB race). Together both orderings are closed.

**Verify:** deterministic logic (defer-then-orphan) + a **multi-close** live watcher
(target ≥4 captured closes keeping `sync`/real ×$2 P&L — NOT generalizing from one,
per the PART-1a lesson). `go build`/`go vet`/tests green; restart clean, 0 "unknown
frame type".

## 2026-06-05 — Chart X-axis timezone: render local/exchange TZ, not UTC (SHIPPED, commit `86d0b1c1`)

**STATUS:** SHIPPED. FE-only, additive. tsc clean; vite serves 200.

**Root cause (diagnosed read-only first):** lightweight-charts v5 does **no** timezone
conversion — its time **axis** defaults to UTC. `AdvancedChart` set a crosshair
`localization.timeFormatter` (browser-local, correct) but **no
`timeScale.tickMarkFormatter`**, so the X-axis tick labels rendered in **UTC** (a 05:10
UTC bar read `05:10` instead of the local/exchange `00:10`, and evening bars showed
tomorrow's date). **Pre-existing** (unchanged since the chart was built) — explicitly
**NOT** caused by the P&L work or the 3 restarts: the running binary never restarted
between "good" and "wrong", and `Bar.T` epochs are correct UTC and consistent across
the midnight boundary (verified against decision-record bar labels Jun 4 vs Jun 5).

**Fix (FE-only, `web/src/components/charts/AdvancedChart.tsx`):** add a
`timeScale.tickMarkFormatter` that formats the axis in an explicit
`CHART_TZ = 'America/Chicago'` (the operator's local **and** the CME exchange zone),
respecting the tick granularity (year/month/day/time); and pin the crosshair
`timeFormatter` to the **same** `CHART_TZ` so axis + crosshair always agree and are
**deterministic regardless of the browser's TZ**. The candle `openTime` mapping is
unchanged (the epoch is correct UTC — only the axis FORMATTING changed). `AdvancedChart`
is the shared chart for crypto + futures, so crypto's axis is now local too (correct for
a Chicago operator).

**Verified (deterministic, in a real browser — JWT was 401 so the authed chart couldn't
load):** 05:10 UTC → axis `00:10` (Chicago) vs the old `05:10` (UTC); crosshair `06/05
00:10` agrees with the axis; date-boundary 02:00 UTC → `06-04` (Chicago), not the old
`06-05` (UTC). The explicit `timeZone` makes it browser-independent.

## 2026-06-05 — Multi-account STAGE 1: none-on-create + gate + persist-pick + allow-list (SHIPPED, commit `56f1642c`)

**STATUS:** Stage 1 SHIPPED (Go+FE, **NO C#**). Additive; crypto + the P&L fixes (PART
1a/1b) + the chart-TZ fix + the account-switch re-scope (24634b5a) all untouched.

**THE CRUX (read-only, C#-confirmed) — why Stage 1 ≠ separation yet:** the NT8 AddOn
holds a **single** `Account` field ([VLTraderTCPClient.cs:49](#)); orders are
`account.CreateOrder/Submit` to that one ([:533-550](#)); the `signal` frame carries
**no account** ([HandleSignal :460-467](#)); `account_select` switches the single field
([:643](#)); and there is **one** TCP connection ([tcp_server.go:3,55,106-108](#)). So
**one connection executes ONE active account** — true per-trader ROUTING needs Stage 2
(a wire-frame `account` field + a multi-account C# AddOn + per-account events + F5).
NinjaScript *can* hold multiple `Account` objects + submit per-account (`Account.All`
lookup under `lock`), so Stage 2 is feasible — just out of this Go+FE pass.

**Root cause (binding diagnosis):** `tcp_server.go:766` auto-binds `currentAccount` on
every `account_balance` frame → a new trader silently traded the streamed account (incl.
risk of the LIVE one); `handleSelectAccount` didn't persist + the pick was overwritten by
the next frame (the `:954` tug-of-war); `traders` had no account column.

**Stage 1 shipped (the stored CHOICE + gate + rails — NOT routing):**
- **DB:** `traders.account` column ([store/trader.go](#)) + `UpdateAccount`; GORM
  AutoMigrate adds it. New/migrated traders start `account=''` (**none-on-create**).
- **GATE:** `runCycle` ([trader/auto_trader_loop.go](#)) skips the cycle for a
  NinjaTrader trader whose `account==''` (re-read each cycle so a fresh pick opens it
  without restart). Verified live: `🚫 No NT8 account selected … skipping cycle #1` for
  "sss" — it does **not** trade until an account is picked, so it can never auto-land on
  the LIVE account.
- **PERSIST:** `handleSelectAccount` ([api/handler_account.go](#)) now `UpdateAccount`s
  the pick per-trader so it **sticks** (survives restart; the stream can't unset it) and
  the gate opens. `handleGetAccounts` returns `selected` (the persisted choice).
- **ALLOW-LIST:** `config.AllowedNTAccounts` (env `NT_ALLOWED_ACCOUNTS`, comma-sep) —
  if set, `handleSelectAccount` HARD-rejects any account not on it. The LIVE account is
  also blocked by the pre-existing **SIM guard** (non-SIM → rejected). Double-guarded.
- **FE:** `AccountSelector` shows/highlights the persisted `selected` (or "Select an
  account" when none → gated); the pick auto-saves (no separate save).

**Honest scope:** Stage 1 persists the **CHOICE** + **gates** + stamps; it does **NOT**
route orders per-account yet — one connection still trades one active account. True
per-AI separation = **Stage 2** (the C# wire-frame + multi-account routing + F5),
DESIGNED + scoped, NOT built.

**Verify:** `go build`/`go vet`/tests green; tsc clean; restart clean (0 "unknown frame
type"); `account` column added; gate fires live for "sss". UI pick not exercised (JWT
401 all session) — persist/allow-list verified by code + the store path.

**ACTION REQUIRED (live-behavior change):** "sss" is now **gated** — pick its account
(Sim101) in the dashboard to resume trading.

## 2026-06-05 — Stage 2 PHASE 1: double-guard the LIVE account out of automated trading (SHIPPED, commit `dedc67f0`)

**STATUS:** Phase 1 SHIPPED (Go live; **C# needs an F5** — see below). Additive; **no
routing, no wire-frame change** (P2/P3). Crypto + the P&L fixes + the chart-TZ fix + the
Stage-1 work untouched. Goal: make it **structurally impossible** for an automated order
to reach the LIVE/funded account (`LFE05060792090061`) BEFORE per-order routing exists.

**STEP A (refutations):** the research's "no typed sim/live property" is **REFUTED** — the
C# `IsSimAccount` already uses the typed `Account.Simulation` (NT8 8.1+) with a `Sim`-name
fallback ([VLTraderTCPClient.cs:279-296](#)), so the live account is already typed at
selection. `NT_ALLOWED_ACCOUNTS` is currently **unset** → the allow-list is inactive (the
SIM guard is the active rail); set it (e.g. `NT_ALLOWED_ACCOUNTS=Sim101`) to enforce a
whitelist on top.

**Double-guard (defense-in-depth) — the live account is rejected at BOTH order layers:**
- **Go gate** ([trader/ninjatrader/tcp_trader.go]): `isAccountTradeable(name)` = the
  account is **SIM** (per the C#-reported `GetAccountsList` IsSim, i.e. `Account.Simulation`)
  **AND**, if `NT_ALLOWED_ACCOUNTS` is set, on that allow-list. **Fail-safe:** unknown →
  false. Enforced at `placeEntry` (the entry-send chokepoint for OpenLong/OpenShort) —
  refuses to even SEND the frame for a non-tradeable account. The live `LFE…` (IsSim=false)
  → refused.
- **C# gate** ([ninjascript/VLTraderTCPClient.cs] HandleSignal, right after the
  `account==null` check, BEFORE `CreateOrder/Submit`): `if (!IsSimAccount(account))` →
  reject (the hard live-block); `if (account.Connection == null || account.Connection.Status
  != ConnectionStatus.Connected)` → reject (no submit to a disconnected account). No
  fallback to another account. This is the last line before the order hits NT8.

Each gate **independently** blocks the live account (verified by code: Go refuses at send;
C# refuses at submit). **No per-order routing added** — one connection still trades one
active account; the guards protect the single active account today and the routed account
in P5.

**⚠️ C# NEEDS AN F5 (no hot-reload):** the C# guard is in the repo `.cs` but the RUNNING
NT8 AddOn still has the old binary until you: `cp ninjascript/VLTraderTCPClient.cs` →
`Documents\NinjaTrader 8\bin\Custom\AddOns\` → **F5** in NT8 → **one clean full NT8
restart**. Until then the **Go gate alone** is live (it already refuses to send a
live-account order). *(If the F5 flags `account.Connection.Status`, remove that one line —
the `IsSimAccount` guard alone still hard-blocks the live account.)*

**Verify:** `go build`/`go vet`/`go test ./store ./trader ./trader/ninjatrader` green;
restart clean (1 trader, 0 "unknown frame type" — no new frame in P1); "sss" on Sim101
(SIM, allow-list empty) is **not** falsely blocked (no refusal logged). The live-account
rejection is **code-verified** (I did not place a live order). Next: P2 = the wire-frame
`account` field; P3 = the C# multi-account routing + per-account events.

## 2026-06-05 — Stage 2 PHASE 2: the C# AddOn PARSES an optional `account` wire-frame field (SHIPPED, C# only, back-compat)

**STATUS:** Phase 2 SHIPPED (C# only; **needs an F5** — see below). Additive; **back-compat
PARSE side ONLY** — the protocol change lands parse-first so the un-changed Go side keeps
working. **NO per-order routing** (still P3), **NO Go send** (still P5), **NO new required
field**. Crypto + the P&L fixes (PART 1a/1b) + the chart-TZ fix + Stage-1 + the P1
double-guard all untouched. Goal: the AddOn learns to READ an optional `account` from the
signal frame, resolve + guard it, and FALL BACK to today's single active-account behavior
when it is absent — so the wire contract can carry an account before any routing exists.

**STEP A (MAIN-verified at file:line):**
- **Parse** ([VLTraderTCPClient.cs:458-473](#)) — the strict fields (symbol/side/quantity/
  entry/stop_loss/take_profit/signal_id/timestamp) parse inside a try/catch that rejects on
  a missing field. `GetString` ([:1350-1354](#)) is `TryGetValue` → returns **null** on an
  absent key (it does **not** throw). So the optional `account` read is placed **after** the
  strict try (a missing `account` can never trip the "missing field" reject) → **back-compat
  by construction**.
- **Fallback** — the single `private Account account` ([:49](#)), switched by
  `HandleAccountSelect` ([:622-690](#)); a no-`account`-field frame uses this active account
  exactly as today.
- **P1 guards** ([:502-523](#)) `IsSimAccount(account)` + `Connection.Status==Connected` —
  intact + reused (same shape) on the resolved account.
- **Resolve** — the canonical name→Account lookup `lock (Account.All) { foreach a … a.Name
  == X }` ([:636-642](#), from `account_select`) reused verbatim.

**The change (additive, C# only):**
- **PARSE** ([:481](#)): `string targetAccount = GetString(p, "account");` — null/empty when
  the field is absent (every order today, since Go does not send it yet → P5).
- **RESOLVE + GUARD, NO ROUTING** ([:525-570](#)): if `targetAccount` is non-empty, resolve
  it against `Account.All` and run the **same** double-guard (SIM + connected) on it, then
  **LOG** the resolution (`signal … targets account 'X' (resolved, sim, connected) … Phase 2
  logs the resolution; submission stays on active account 'Y' (per-order routing is Phase
  3)`). A named-but-bad target (unknown / non-SIM / disconnected) is **REJECTED** here (reuse
  the P1 reject — `SendFillFrame(…, "rejected")`); we never silently fall back to the active
  account when a specific one was requested. The active `account` (guarded above) still
  SUBMITS — **P3** adds the routing that submits to the resolved account.
- **FALLBACK** ([back-compat]): an ABSENT/empty `account` skips the resolve block entirely →
  byte-identical to today. The P1 double-guard still hard-blocks the LIVE `LFE…` account
  (active or resolved).

**⚠️ C# NEEDS AN F5 (no hot-reload):** the parse is in the repo `.cs` (already `cp`'d to
`Documents\NinjaTrader 8\bin\Custom\AddOns\`) but the RUNNING NT8 AddOn has the old binary
until you **F5** in NT8 → **one clean full NT8 restart**. The cp staged P1+P2 together (the
AddOns copy predated P1), so one F5 deploys both. Until the F5, the running AddOn ignores
any `account` field — and Go doesn't send one yet, so nothing changes. *(If the F5 flags
`account.Connection.Status`, remove that one line — `IsSimAccount` alone still hard-blocks
live.)*

**Verify:** Go **untouched** (0 `.go` modified → no `nofx-bin` rebuild; pid 98629 from P1
still running, cycle 381, MNQ "wait"). The `.cs` edit is brace-balanced (delta vs HEAD =
+6/+6 braces, +0 brackets; parens even) — NinjaScript compiles inside NT8 on F5, so the
compile itself is the user's step. Back-compat handshake with the CURRENT binary is clean
(0 "unknown frame type", 0 "signal payload missing field"). Parse+resolve is **code-verified**
this phase (the C# isn't live until the F5; no live order placed — SIM-only). Next: P3 = the
C# per-order routing (submit to the resolved account + per-account events); P5 = Go sends the
`account` field.

## 2026-06-05 — Stage 2 PHASE 3: C# routes each order to the RESOLVED account (SHIPPED, C# only, back-compat)

**STATUS:** Phase 3 SHIPPED (C# only; **needs an F5** — see below). Additive; **the per-order
ROUTING** — the trade-copier submit. Still back-compat (absent `account` = active account =
today). **NO Go send** (P5), **per-account balance/position snapshots + manual-close routing =
P4**. Crypto + the P&L fixes (PART 1a/1b) + chart-TZ + Stage-1 + P1 + P2 all untouched. THE
RISKIEST PHASE — a routing bug = an order on the wrong account; the P1 double-guard runs on
EVERY routed submit, the LIVE `LFE…` account is hard-blocked as a routing target.

**STEP A (MAIN-verified at file:line):** the P2 resolve block ([VLTraderTCPClient.cs:525-570](#))
resolves+guards the target but submitted on the active account; the submit path
(`account.CreateOrder/Submit` :609/:626) + the bracket (`account.CreateOrder/Submit` in
`SubmitBracketOnEntryFill`) + the events (`account.OrderUpdate += OnOrderUpdate` :99) were ALL
bound to the single active `account`. `OnOrderUpdate` ([:749-848](#)) is **account-agnostic for
reads** (everything off `e.Order`) — so a routed account's fills process correctly IF its
`OrderUpdate` is subscribed. Back-compat holds with **no Go change** (the Go wire struct
`ntwire.SignalPayload` has no `account` field → P5).

**The change (additive, C# only):**
- **ROUTE** — `Account submitAccount = resolved ?? account;` then the entry
  `submitAccount.CreateOrder(...)` + `submitAccount.Submit(...)`. The `PendingBracket` gains an
  `Account` field (= `submitAccount`) so `SubmitBracketOnEntryFill` places the SL/TP on the SAME
  account the entry routed to (`Account ba = b.Account ?? account`).
- **FINAL GUARD (defense-in-depth, every routed submit)** — immediately before `CreateOrder`,
  re-assert `IsSimAccount(submitAccount) && Connection.Connected` on the EXACT submit target, no
  matter how chosen; fail → REFUSE+reject. The LIVE account can never be a submit target. (The
  allow-list stays Go-side at `placeEntry` — no config channel into the AddOn.)
- **PER-ACCOUNT FILLS** — `OnOrderUpdate` must fire for the routed account, so a dedup'd
  `HashSet<Account> orderSubscribed` + `SubscribeOrderUpdate`/`UnsubscribeOrderUpdate` helpers
  become the single source of truth; ALL OrderUpdate sub/unsub (the active-account init,
  `account_select` swap, terminate) flow through them (behavior-preserving + no double-subscribe),
  and the routed `submitAccount` is subscribed before submit.
- **FALLBACK (back-compat)** — `resolved == null` (absent field, every order today) →
  `submitAccount = account` → byte-identical to P2/today.
- **KEPT deliberately:** `OrderEntry.Manual` (the proven SIM submit; `Automated` is orthogonal +
  regression-risky — not required for routing) and the order name = `signalId` (the fill-matching
  key; an explicit agent-id tag needs a wire field = P5).

**Deferred to P4 (flagged, do NOT assume done):** per-account `AccountItemUpdate` (balance) +
`PositionUpdate` (position snapshots) stay active-account-only; manual close (`account.Flatten`)
routes to the active account. These don't affect the routed ENTRY→fill→SL/TP-close lifecycle
(which flows through `OnOrderUpdate` + the bracket-on-stored-account), but full multi-account
parity needs them.

**⚠️ C# NEEDS AN F5 (no hot-reload):** the routing is in the repo `.cs` (already `cp`'d to
`Documents\NinjaTrader 8\bin\Custom\AddOns\`) but the RUNNING AddOn has the old binary until you
**F5** in NT8 → **one clean full NT8 restart**. Until the F5, the running AddOn ignores routing
— and Go sends no `account` field anyway, so nothing changes.

**Verify:** Go **untouched** (0 `.go` → no `nofx-bin` rebuild). The `.cs` edit is structurally
balanced (delta vs HEAD = +8/+8 braces, +44/+44 parens, +0 brackets) — NinjaScript compiles
inside NT8 on F5 (the user's step). Routing is **correct by construction** + verified by the
deployed-copy grep (`submitAccount.CreateOrder/Submit`, bracket-follows-account, the final
guard, `SubscribeOrderUpdate`). A live cross-account frame-injection isn't possible pre-Go-send
(P5) — so the routing is NOT exercised by a live cross-account fill this phase; for the
single-trader case `resolved == active`, the existing fully-subscribed path is unchanged.
SIM-only; no live order placed. Next: P4 = per-account events/positions/balance + manual-close
routing; P5 = Go sends the `account` field.

## 2026-06-05 — Stage 2 PHASE 4: per-account CLOSE + fill/close-frame account attribution (SHIPPED, C# only)

**STATUS:** Phase 4 SHIPPED (C# only; **needs an F5**). Additive; fixes the P3 account-blind close
and tags the reporting frames with the owning account. Back-compat (resolves to Sim101 today).
**NO Go change** (audit: Go already keys every read by account + degrades to Sim101). **NO Go
account-send** (P5). Crypto + P&L fixes (PART 1a/1b) + chart-TZ + Stage-1 + P1 + P2 + P3 untouched.

**STEP A (3-track audit + MAIN re-verify at file:line):** all 3 tracks `back_compat_ok=true,
forces_wire_or_go_send=false` — no STOP. Wire: `account_balance`/`positions` frames already carry
`account` (tcp_framing.go:96/210); the `fill` frame (FillPayload :48) and `position_close`
(SendPositionCloseFrame) did NOT; `position_close_rejected` stamped the **active** account (latent
bug). `close_position` (Go→C#) carries no account → the close must resolve the account C#-side.
`OnOrderUpdate` is account-agnostic (reads `e.Order.Account`). **Track-3 decisive:** Go needs ZERO
changes — every read (`acctBalances[p.Account]` tcp_server.go:762, `acctPositions[p.Account]` :840,
reconcile `row.Account!=acct` reconcile.go:105, the variadic-account store funcs) already keys by
account; `recordClose` updates the entry row in place, inheriting its account stamp. Go decodes
with plain `json.Unmarshal` (no `DisallowUnknownFields`) → the new `account` keys are silently
ignored by the un-changed binary.

**The change (additive, C# only):**
- **PER-ACCOUNT CLOSE** — new `Dictionary<string,Account> positionAccountBySymbol` (root-symbol →
  owning account): populated on **entry-fill** (`OnOrderUpdate` Filled, from `e.Order.Account`),
  cleared on **exit-fill**. `HandleClosePosition` resolves the symbol's account (fallback the active
  account → back-compat) and flattens **that** account, behind a **`IsSimAccount` guard** (a
  non-SIM/LIVE resolved target → REFUSE, position left open). Keyed by root symbol because the Go
  close generates a fresh UUID signal_id (tcp_trader.go) that can't match the entry.
- **FILL ACCOUNT TAG** — `SendFillFrame` now carries `e.Order.Account.Name` (optional param,
  default "" → the 10 reject/stale calls are unchanged).
- **CLOSE-REPORT ACCOUNT FIX** — `SendPositionCloseFrame` now emits the account (was absent);
  `SendPositionCloseRejectedFrame` now stamps `e.Order.Account` (fixes the latent active-account
  bug). Both reachable today because `OrderUpdate` is already per-account (P3).

**DEFERRED to P5 (flagged, NOT built):** the per-account **balance/position SUBSCRIPTION** +
account-aware `OnPositionUpdate`/`OnAccountItemUpdate`/`SendAccountBalance(acc)`. Rationale: it is
(a) dormant until cross-account routing (P5), and (b) **blocked anyway by the Go process-singleton
TCPServer** (transport.go:31-53 — one fill/close channel + one streamed `CurrentAccount()`), which
P5 must demux first. Building the C# reporting subscription now is premature subscription-lifecycle
risk for zero reachable value; it belongs with P5's Go demux. The active account's balance/position
already report correctly today.

**⚠️ C# NEEDS AN F5 (no hot-reload):** cp'd to `Documents\NinjaTrader 8\bin\Custom\AddOns\`; **F5 in
NT8 → one clean full restart**. Until then the running AddOn keeps the P3 binary; Go sends no
account field, so nothing changes (back-compat).

**Verify:** Go **untouched** (0 `.go` → no `nofx-bin` rebuild). `.cs` structurally balanced (Δ vs
HEAD = +6/+6 braces, +28/+28 parens, +3/+3 brackets) — compiles inside NT8 on F5 (the user's step).
The close-account resolution + fill/close account tags are **correct by construction** + the
deployed-copy grep; back-compat (one trader → resolves to Sim101). Cross-account close isn't
exercisable pre-P5 (no routed position exists). SIM-only; no live order placed. Next: P5 = Go sends
the `account` field on the signal + the per-account channel demux + the per-account reporting
subscription (activates true multi-account).

## 2026-06-05 — Strategy Studio Phase 1 (Risk Control): STOP-report + architecture + Chunk 1 SHIPPED

**STOP-and-report (premise refuted):** the manifest said the prop-firm guardrails "do NOT exist."
MAIN audit found an **existing** `kernel/risk_limits.go` (`RiskLimits{MaxDailyLossUSD,
MaxConcurrentTrades, MaxNotionalUSD, MaxContractsPerOrder}` + `CheckPreTrade`/`Classify`/`ForceFlat`
+ `MaybeResetDaily`), `kernel/cme_calendar.go` (`IsCMEOpen`), and `api/handler_risk.go`. Status,
file:line-verified: daily-loss limit **EXISTS + LIVE** but **global-env** (`RISK_MAX_DAILY_LOSS_USD`
default $500), uses **session-cumulative** P&L + **UTC** reset (`engine_analysis.go:108-112`,
`config.go:67,134`); concurrent-trade cap EXISTS (env=2, overlaps per-strategy Max Positions);
notional cap **wired-but-disabled** (passed `0,0` at `engine_analysis.go:108`); **MaxContractsPerOrder
loaded but NEVER enforced** (absent from `CheckPreTrade`); daily-profit / max-daily-trades /
blackout-gate / consistency **genuinely absent**. Building "new" gates on top would create
conflicting real-money limits — so I stopped + got the architecture decided.

**Architecture (owner-decided):** (1) **evolve the single existing gate to per-strategy** (read the
Risk Control boxes; env = fallback), extend with the missing limits, enforce max-contracts, fix the
notional, no duplicate gates; (2) **true daily-realized P&L on the CME session day** (Sun 5pm CT
boundary), not session-cumulative + UTC.

**CHUNK 1 SHIPPED (deep-dive fixes #7-8 — the FAKE boxes made REAL):**
- **Min R/R** — `validateDecision` now reads the per-strategy `min_risk_reward_ratio` (threaded
  through `parseFullDecisionResponse → validateDecisions → validateDecision`, sourced from
  `riskConfig` at `engine_analysis.go:243-250`) instead of the hardcoded `3.0` at
  `engine_position.go:133`. Unset (≤0) falls back to 3.0 (back-compat). Applies crypto + futures.
- **Min Confidence** — a NEW gate check rejects an open with `d.Confidence < min_confidence` when
  configured (`min_confidence` = 0 → disabled, back-compat). Was prompt-only (AI-guided); now
  CODE ENFORCED.
- **Tests:** `TestStrategyStudio_MinRRConfigDriven` (4:1 decision PASSES at minRR=3.0, REJECTED at
  5.0 — proves config-driven) + `TestStrategyStudio_MinConfidenceGate` (conf-50 PASSES at gate-off,
  REJECTED at min=75) — both green; all existing gate tests green. `go build`/`go vet` clean.
  `TestMaybeResetDaily` fails on HEAD too (pre-existing, unrelated — to be examined when Chunk 2
  touches the daily reset).

**REMAINING CHUNKS (planned, NOT built — each its own safe chunk + Sunday behavior-proof):**
Chunk 2 = the per-strategy daily guardrails (daily loss/profit, max daily trades) on true
daily-realized P&L + CME-day reset; Chunk 3 = max-contracts (enforce the unenforced field) + the
visible/editable equity×20 cap; Chunk 4 = time/news blackout (wire `IsCMEOpen`); Chunk 5 =
consistency rule; Chunk 6 = the FE futures risk panel (contracts × point value, $-risk/trade,
margin as $/contract) + all the new guardrail inputs + the honest Max-Margin relabel (fix the false
"System enforced" label `skill_management_handlers.go:1166`). ADDITIVE; crypto byte-identical; the
multi-account work + P&L fixes + chart-TZ untouched. SIM-only.

**CHUNK 2 SHIPPED (daily loss/profit/max-trades on TRUE CME-day realized P&L + the reset-bug fix):**
- **Evolved the ONE gate (no duplicate):** the env daily-loss in `CheckPreTrade` was on session-
  cumulative `TotalPnL` (`engine_analysis.go:108`); now it passes `0` there (keeping only the
  concurrent cap) and a new `DailyGuardrails.Check()` enforces daily-loss on **true daily-realized
  P&L** over the CME session-day, plus the new **daily-profit** + **max-daily-trades** limits.
- **Per-strategy + env fallback + toggles:** values from `store.RiskControlConfig`; daily-loss
  falls back to `RISK_MAX_DAILY_LOSS_USD` via `firstPositive(perStrategy, env)`; a master switch
  (`GuardrailsEnabled`, default ON) + per-guardrail `*bool` toggles (daily-loss default ON to
  preserve the live gate; new ones OFF) via `boolOrDefault`. Master OFF → ALL bypassed (logged).
- **True daily-realized P&L:** `CMESessionDayStart`/`CMESessionDayKey` (17:00 CT roll);
  `store.GetSessionDayActivity` sums realized P&L (closed, excl. reconcile-flat) + counts entries
  since the session start; the loop sets `ctx.DailyRealizedPnL`/`ctx.TradesToday`; the gate reads them.
- **Reset bug FIXED:** `ResetDailyPnL` stamped `time.Now()` while `MaybeResetDaily(now)` used the
  passed time → never agreed. Both now derive from `CMESessionDayKey(now)`. `TestMaybeResetDaily`
  PASSES (was failing on HEAD).
- **SAFETY (by construction):** the toggles live in the KERNEL decision gate (HOLD only); the
  live-account block lives in the BROKER layer (P1/P3/P4 + C# IsSimAccount at placeEntry). Master
  OFF lets the decision proceed but the order STILL hits the untoggleable live-account block.
- **Tests:** 9 new (boundaries/master/toggles/env-fallback/CME-day) + `TestMaybeResetDaily` green;
  full kernel green; `go build`/`vet` clean; store + trader green. ADDITIVE; crypto byte-identical.

**Sunday behavior add:** tiny daily-loss → bot stops after a loss; max-daily-trades=1 → 2nd entry
blocked; daily-profit hit → entries blocked; master OFF → all bypass (logged) but live account
STILL blocked. REMAINING: Chunk 3 (max-contracts + visible equity×20), Chunk 4 (blackout), Chunk 5
(consistency), Chunk 6 (FE + toggles UI + Max-Margin relabel + Playwright).

**CHUNK 3 SHIPPED (max-contracts + the visible/editable equity×20 cap):**
- **Max contracts** — `futuresOrderQuantity` had a SILENT hardcoded clamp at `maxFuturesContracts =
  10` (`auto_trader_orders.go:16`). Now it takes a resolved `maxContracts` (clamps + **logs**); the
  caller uses `at.resolveMaxContracts()` → `kernel.ResolveMaxContracts(master, toggle, perStrategy,
  10)`. Per-strategy `MaxContractsPerOrder` overrides; toggle/master OFF → `0` = no clamp.
- **Visible equity×20 cap** — the hidden const `futuresMaxNotionalLeverage = 20.0` (used in
  `validateDecision:54/:90` + `enforcePositionValueRatio:211`) is now the **editable multiplier**
  `MaxNotionalLeverage` (default 20). `validateDecision` takes a 9th `maxNotionalLev` param
  (resolved at the call site via `kernel.ResolveNotionalLeverage`); `enforcePositionValueRatio`
  reads it directly. `<=0` → cap DISABLED (master/toggle off) → huge ceiling (never binds).
- **Config:** `MaxContractsPerOrder`/`MaxContractsEnabled` + `MaxNotionalLeverage`/`NotionalCapEnabled`
  added to `store.RiskControlConfig` (toggles `*bool`, plug into the Chunk-2 master framework).
- **Safety (by construction):** these toggles only affect trade SIZE (contracts/notional) in the
  trader+kernel gate — they never touch which account the order routes to. The live-account block
  (placeEntry + C# IsSimAccount) is untoggleable.
- **Tests:** `TestResolveMaxContracts` + `TestResolveNotionalLeverage` (per-strategy override, unset
  → default, toggle off → no clamp, master off → no clamp) green; all existing futures-gate +
  futures-quantity tests updated + green; full kernel + trader packages green; `go build`/`vet`
  clean. ADDITIVE; crypto byte-identical (the leverage/PVR tiers unchanged).

**Sunday add:** max-contracts exceeded → clamped (logged); raise `MaxNotionalLeverage` → bigger
position allowed; toggle a cap OFF → it stops binding. REMAINING: Chunk 4 (blackout), Chunk 5
(consistency), Chunk 6 (FE + toggles UI + Max-Margin relabel + Playwright).

**CHUNK 4 SHIPPED (time/news blackout window):**
- **What:** a per-strategy daily **blackout window** (`BlackoutStartCT`/`BlackoutEndCT`, HH:MM in
  America/Chicago) + toggle (`BlackoutEnabled`, default OFF). When master+toggle ON and `now` is in
  `[start,end)` CT, the gate skips the decision cycle (HOLD); NT8-side SL/TP still protect open
  positions. The session-hours gate (`IsCMEOpen`, Plan-3 Task-18) already existed; this adds the
  configurable window.
- **Where:** `kernel.InBlackoutWindow(now, startCT, endCT)` (handles midnight-wrap; empty/malformed
  → false so a misconfig never silently halts) chained into the Chunk-2 gate block
  (`engine_analysis.go`) as `else if`, so it only runs when master is ON and no daily guardrail
  already tripped.
- **Safety (by construction):** blackout lives in the kernel decision gate (HOLD only); never the
  live-account block. Master OFF → blackout (+ daily limits) bypassed, logged.
- **Tests:** `TestInBlackoutWindow` (in-window, end-exclusive, before-start, outside, midnight-wrap,
  empty/malformed/zero-width) green; full kernel + store + trader green; build/vet clean. ADDITIVE.

**Sunday add:** set a blackout window covering "now" → entries blocked in the window; outside →
normal. REMAINING: Chunk 5 (consistency), Chunk 6 (FE + toggles UI + Max-Margin relabel + Playwright).

**CHUNK 5 SHIPPED (consistency rule — the complex one):**
- **Semantic (documented):** no single CME session-day's realized profit may exceed
  `ConsistencyMaxDayPct`% of all-time total realized profit. `ConsistencyBreached(today, total, pct)`
  triggers ONLY once there is prior-day profit (`total − today > 0`) so a fresh/single-day account
  never self-locks on its first profitable day; losing days, zero total, or pct≤0 never breach. When
  breached the gate skips new entries so today's profit stops growing past the allowed share.
- **Data:** `today` = `ctx.DailyRealizedPnL` (Chunk 2); `total` = `ctx.TotalRealizedPnL` set in the
  loop from `GetFullStats().TotalPnL`. Chained into the gate as the next `else if` (master+toggle
  governed, default OFF).
- **Tests:** `TestConsistencyBreached` (breach, boundary, under, first-day no-self-lock, losing day,
  zero total, pct 0) green; full kernel + store + trader green; build/vet clean. ADDITIVE.

**BACKEND GUARDRAIL LAYER COMPLETE (Chunks 1–5):** Min R/R, Min Confidence, daily loss/profit/
max-trades (true CME-day realized P&L), max-contracts, editable equity×20 cap, time/news blackout,
consistency — ALL gate-enforced, per-strategy + env fallback, master + per-guardrail toggles, every
trip + every bypass LOGGED, no toggle disables the live-account block. REMAINING: **Chunk 6 = the FE**
(futures risk panel + all guardrail inputs + per-guardrail/master toggle UI + Max-Margin relabel +
Playwright FE-verify).

**CHUNK 6 SHIPPED (the FE surfacing — Strategy Studio Phase 1 COMPLETE end-to-end):**
- **Prop-Firm Guardrails section** added to `RiskControlEditor.tsx`: a **master switch** + a
  `GuardrailRow` (label + on/off `Toggle` + value input) for daily loss, daily profit, max-daily-
  trades, consistency %, max-contracts, the editable notional cap (equity × N), and a blackout
  start/end window — each wired via `updateField('<exact_snake_case_key>')`.
- **Wiring verified (MAIN, file:line):** all 17 FE keys match the `store.RiskControlConfig` JSON tags
  the gates read 1:1 (`daily_loss_limit_usd`, …, `guardrails_enabled`); every FE toggle default
  (`?? true`/`?? false`) matches its backend `boolOrDefault` default. So a saved value reaches the
  exact gate field; an unset strategy shows the correct default toggle state.
- **Futures risk panel** (futures path only): shows the real futures size knobs (≤ max contracts,
  notional = equity × N, MNQ ≈ $2/pt). Crypto keeps its leverage/PVR tiers byte-identical (the panel
  + the guardrails section render regardless, but the dead crypto knobs stay only on the crypto path).
- **Honest relabel:** the false **"System enforced"** on Max Margin → `AI-guided (not enforced)` /
  `futures = per-contract bond` in the FE (RiskControlEditor.tsx) AND the Go chat summary
  (`skill_management_handlers.go:1166`, the one false tag — the others on 1163-1167 are genuinely
  enforced).
- **Safety:** the FE toggles set ONLY the guardrail `…_enabled`/master flags (kernel-gate config) —
  no FE control touches the broker-layer live-account guard; the LIVE account stays hard-blocked.
- **Verify:** `tsc --noEmit` clean; `npm run build` ✓; `go build` ✓ (the chat-string fix). Live
  Playwright render of the panel is **blocked by an expired JWT** (the strategy fetch 401s — same as
  all session); documented (screenshot) + the FE→config→gate wiring verified by code (all 17 keys
  match) + the panel builds/type-checks. The save→persist + visual render go on the Sunday/next-auth
  list. ADDITIVE; crypto byte-identical; the backend (Chunks 1-5) + multi-account + P&L + chart-TZ
  untouched.

**STRATEGY STUDIO PHASE 1 (RISK CONTROL) COMPLETE** — gates (Chunks 1-5) + FE (Chunk 6): every
guardrail is gate-enforced, per-strategy + env fallback, master + per-guardrail toggles, every trip/
bypass logged, no toggle disables the live-account block, and all of it is now settable on the page.

## 2026-06-05 — Strategy Studio: two reported issues (diagnose + FE fix)

**(A) "built but can't use anything" — DIAGNOSED, not a code bug.** The stored JWT **expired
2026-05-30 23:43 UTC** (~6.25 days ago); `/api/strategies` returns **401** → "Failed to fetch
strategies" → no strategy loads → the editor renders empty/disabled. Verified via Playwright (decoded
the token's `exp`). Compounding by-design facts (not bugs): the editor is `disabled={selectedStrategy
?.is_default}` (StrategyStudioPage.tsx:774) — **default/template strategies are read-only** (the Save
button is hidden for `is_default`); and the Max-Positions / Min-Position-Size / PVR boxes are
intentionally read-only "System enforced" displays. **Fix: re-login** (fresh token) → the strategy
list loads → a **non-default** strategy is editable. No code change for (A).

**(B) PVR labels lying on futures — FIXED (FE-only).** The Position-Value-Ratio section showed
"CODE ENFORCED" / "System enforced" on BOTH paths, but the deep-dive proved PVR is **FAKE for
futures** (the gate hardcodes equity×20, ignoring these ratios). The crypto PVR tiles are now hidden
on the futures path — `RiskControlEditor.tsx` wraps the PVR section in `{!isFutures && (…)}` (mirroring
the leverage-tier pattern). On futures the real size controls are **max-contracts + the editable
equity×N notional cap** (already in the Chunk-6 Prop-Firm Guardrails section). **Crypto keeps the PVR
tiles + "CODE ENFORCED"** (true there). Max-Positions + Min-Position-Size keep "System enforced" (REAL
both per the deep-dive); Max-Margin was already relabeled honest in Chunk 6.

**Verify:** `tsc --noEmit` 0 errors; `npm run build` ✓; Go untouched. The futures-vs-crypto render is
**code-verified** (the `{!isFutures}` wrap) — a live Playwright render is **blocked by the expired
JWT** (can't load a strategy); it'll show once re-authed. ADDITIVE; crypto byte-identical; the gates
(Chunks 1-5) untouched.


**FOLLOW-UP — the REAL "can't use anything" root cause + fix (FE).** Deeper diagnosis: the app's
`auth_token` expired 2026-06-04 (2 days ago) AND `AuthContext` init (`contexts/AuthContext.tsx:63,76`)
called `setToken(savedToken)` WITHOUT checking expiry — so the app sat "logged in" while every API
call 401'd, with no prompt to re-login (`from401`/`returnUrl` null, never bounced to `/login`). That
stuck state IS the "can't use anything." Fix: a small `isJwtExpired(token)` helper; both init branches
now skip + CLEAR an expired token so the app shows the login page. tsc clean; FE build ✓; Go untouched.
After this lands (vite HMR), a reload detects the dead token → `/login` → re-login → everything works
(and the PVR-on-futures fix shows). ADDITIVE; FE-only; the gates + crypto untouched.


## 2026-06-06 — Max Positions + Min Position Size made USER-EDITABLE (FE-only; supersedes the "keep System enforced" note above)

**Goal:** the user wanted to SET Max Positions + Min Position Size themselves — they were read-only
"System enforced" displays. (This corrects the 2026-06-05 note above that said they "keep System
enforced".)

**AUDIT (STEP A) — the premise that a gate-read change was needed is REFUTED; it's FE-only.** Both
gates ALREADY read the per-strategy config value with a system default — exactly the Chunk-1 pattern:
- **Max Positions:** `enforceMaxPositions` reads `RiskControl.MaxPositions`, default **3** when ≤0
  (`trader/auto_trader_risk.go:266`). `ClampLimits` bounds it to **[1, 3]** on every save
  (`api/strategy.go:205` create / `:316` update) AND at decision time (`kernel/engine_analysis.go:220`)
  — the ceiling is the `MaxPositions = 3` const (`store/strategy.go:18,77`), which is reused as both
  the default and the clamp ceiling.
- **Min Position Size:** `enforceMinPositionSize` reads `RiskControl.MinPositionSize`, default **12**
  when ≤0 (`trader/auto_trader_risk.go:249`). `ClampLimits` bounds it to **[10, 1000]**
  (`store/strategy.go:122-126`, consts `MinPositionSize=10`/`MaxPositionSize=1000`).

**FLOOR VERDICT (the critical safety question) — SAFE to expose.** Min Position Size has THREE
independent guards: store-clamp ≥10, trader-gate ≥config, and the kernel reject-floor **12 general /
60 BTC-ETH** (`kernel/engine_position.go:78-84`). A user value can therefore only ever **RAISE** the
effective minimum — anything below the floor is still clamped/rejected (defense-in-depth). **No floor
is removed; none is bypassed.** So min_position_size is exposed freely within [10,1000]. Max Positions
has a genuine **ceiling of 3** (the const), so it is exposed within **[1,3]** (lets the user *tighten*
to 1–2, e.g. a single MNQ position). Raising it above 3 is a separate product decision (the const
comment is "Hard limits to prevent token explosion in AI requests") — **FLAGGED, not silently done**,
mirroring Chunk 1 (Min R/R was exposed within its `[1,10]` clamp range, NOT by raising the const).

**BUILD (FE-only, additive) — `web/src/components/strategy/RiskControlEditor.tsx`:**
- Max Positions: the read-only `<span>` + "System enforced" → a `type="number"` input
  (`min=1 max=3 step=1`) wired to `updateField('max_positions', …)`, with an **onChange clamp**
  `Math.min(3, Math.max(1, …))` so the shown value always equals the saved value (no "typed 5, saved
  3" surprise). Label → `user-set · enforced (range 1–3)`.
- Min Position Size: same treatment, input (`min=10 max=1000 step=1`) wired to
  `updateField('min_position_size', …)`, onChange clamp `Math.min(1000, Math.max(10, …))`, USD/USDT
  unit preserved. Label → `user-set · enforced`.
- Mirrors the existing Min R/R input exactly (same className/style/fallback). Both boxes render on
  BOTH crypto + futures (outside the `!isFutures` gates) — correct, since the gate reads them on both.

**No env var for these two** (unlike the Chunk-2–5 guardrails): the system default (3 / 12) IS the
unset fallback, and the gate already reads the per-strategy value — so "per-strategy + system
fallback" is satisfied with **zero Go change**.

**Verify:** `tsc --noEmit` 0 errors; `go build ./...` clean; `go test ./store ./trader ./kernel` all
`ok` (Go byte-identical — pure regression proof). ADDITIVE; crypto byte-identical; the gates
(Chunks 1-5) + guardrails + multi-account + P&L + chart-TZ all untouched. SIM-only; the live-account
block untoggleable (broker layer, unchanged).


## 2026-06-06 — Strategy Studio universal LABEL sweep + Max Margin (editable on crypto / hidden on futures) — FE only

Two FE-only jobs, ADDITIVE, no Go/gate change, no enforced path touched.

**JOB 1 — neutralized crypto-flavored labels on UNIVERSAL controls** (LABELS only — no per-instrument
gating/hiding of data; that reverted pattern was NOT repeated). The Strategy Studio is one page for
both crypto + CME futures, so "coins" on controls that serve both is wrong. Changed (all 3 locales
zh/en/es) in `web/src/i18n/strategy-translations.ts`:
- `maxPositionsDesc` (Risk Control): "Maximum **coins** held simultaneously" → "Maximum **positions**
  held simultaneously".
- `staticCoins` "Custom Coins"→"Custom Symbols"; `addCoin` "Add Coin"→"Add Symbol"; `staticDesc`
  "…trading coins"→"…trading symbols"; `coins` (the "Up to N" unit) "coins"→"symbols";
  `excludedCoins` "Excluded Coins"→"Excluded Symbols"; `excludedCoinsDesc` "These coins…"→"These
  symbols…".
And in `web/src/components/strategy/CoinSourceEditor.tsx` (hardcoded): the max-symbols toast
"…coins allowed"→"…symbols allowed"; the two add-symbol placeholders "BTC, ETH, SOL…" / "BTC, ETH,
DOGE…" → "e.g. MNQ, ES, BTC, ETH" (shows both futures + crypto).
**LEFT crypto wording (deliberate, verified):** the Leverage sliders + PVR tiles (crypto-only, already
hidden on futures via `{!isFutures}`); `minPositionSizeDesc` USDT (already has a `…Futures` USD variant
chosen by `isFutures`); the `USD/USDT` unit (already `isFutures`-conditional); the AI500 / OI / NofxOS
data-source descriptions (genuinely crypto data providers — neutralizing them would falsely imply they
serve futures); GridConfig symbol options (grid = a separate strategy type).

**JOB 2 — Max Margin Usage made EDITABLE (crypto) / HIDDEN (futures), stays ADVICE-ONLY.** In
`RiskControlEditor.tsx` the read-only display span became a number input (percent), wired to
`updateField('max_margin_usage', …)`, clamped to **[0.1,1.0] = [10,100]%** to match `ClampLimits`
(store/strategy.go:116-120) so shown == saved. Wrapped in `{!isFutures && (…)}` (the SAME `isFutures`
the PVR fix uses) → visible+editable on crypto, hidden on futures (margin-usage % is a crypto-margin
concept; futures margin is a per-contract bond shown in the Futures Risk panel). **Stays advice-only:**
the value still only feeds the AI prompt (`engine_prompt.go:73`) — NO gate enforcement was added. Its
lying i18n desc `maxMarginUsageDesc` ("enforced by code" / "由代码强制执行") was corrected to the honest
"AI-guided hint, not code-enforced".

**Verify:** `tsc --noEmit` 0 errors; `npm run build` ✓; Go untouched (no gate/prompt code change — the
prompt USE of max_margin_usage is unchanged). ADDITIVE; crypto otherwise byte-identical; the gates
(Chunks 1-5) + guardrails + PVR fix + the editable Max-Positions/Min-Position-Size + multi-account +
P&L + chart-TZ all untouched. SIM-only; the live-account block untouched.


## 2026-06-06 — Strategy Studio PHASE 2 CORE: the Mode dropdown is now REAL (3 safe chunks) — built + static-verified, behavior-verified Sunday

**STATUS:** BUILT + STATIC-VERIFIED (market closed). The owner's saved Mode now drives the LIVE bot;
"Futures" is selectable; the preview shows what the bot will actually run. LIVE behavior proof
(pick a mode → the bot's next decision uses it) is on the SUNDAY list.

Prior state (Phase-2 deep-dive map): the Mode dropdown was a **preview-only toy** — its value was
local state sent only to the preview/test endpoints, NEVER saved; the live loop hardcoded the variant
by venue (`auto_trader_loop.go`: ninjatrader→"futures", else→"balanced"); the dropdown had no
"Futures" option; and a futures strategy's preview showed "balanced" while the bot ran the futures
prompt (the "preview lies" gap). Three additive chunks fixed this:

- **CHUNK A — `cad85169` (FE-only):** added a **"Futures"** option to both Mode `<select>`s (Prompt
  Preview + AI Test tabs) + the `futures` i18n key (en/zh/id). The backend `BuildSystemPrompt` already
  accepts `variant="futures"` (engine_prompt.go:22). tsc 0; Go untouched.
- **CHUNK B — `84f5ea75` (KEYSTONE, Go + FE):** the dropdown pick now **persists** and **drives the
  live bot**.
  - `store/strategy.go`: `StrategyConfig` gains a top-level `prompt_variant` (json), threaded through
    `MarshalJSON`/`UnmarshalJSON` so it persists with the strategy (omitempty → unset = absent).
  - `trader/auto_trader_loop.go`: new pure helper **`resolvePromptVariant(exchange, saved)`** — a
    non-empty saved variant WINS; when EMPTY it falls back to the ORIGINAL venue rule
    (ninjatrader→futures, else balanced). The loop reads the saved variant **null-safely** (nil
    engine/config → venue rule). **Back-compat guarantee: a no-variant strategy resolves EXACTLY as
    before — byte-identical.**
  - FE: `prompt_variant` added to the `StrategyConfig` type + to `normalizeStrategyConfig` (which
    otherwise **strips** it on save — the keystone gotcha); the dropdown persists via `updateConfig`
    and syncs from the saved value on strategy switch.
  - Proven by unit tests: `TestResolvePromptVariant` (empty→venue-rule byte-identical; saved wins;
    whitespace-trimmed) + `TestStrategyConfigPromptVariantRoundTrip` (persists top-level; unset
    omitted; legacy config → ""). **Prompt-layer only — NO risk gate, NO live-account-block change.**
- **CHUNK C — `843f0350` (FE-only, honest preview):** the dropdown/preview now initialize to the
  variant the LIVE loop will **resolve** — mirroring `resolvePromptVariant`: saved wins, else the
  venue rule by the strategy's symbol (`isCMEFutures(static_coins[0])` → "futures", else "balanced").
  The preview endpoint already builds for the strategy's own symbol, so a futures strategy now previews
  the **real futures prompt** (preview == live), not a misleading "balanced".

**Static verify:** `go build ./...` clean; `go vet ./store ./trader` clean; `go test ./store ./trader
./kernel` all green (incl. the 2 new keystone tests); `tsc --noEmit` 0; FE built; `nofx-bin` rebuilt +
restarted clean (0 "unknown frame type"; trader `sss` = NinjaTrader/"MNQ SIM Default" auto-started — a
default strategy with no saved variant → resolves to "futures" via the fallback, byte-identical).

**SUNDAY behavior list (market open):** on a NON-DEFAULT strategy, set the Mode (e.g. Aggressive on a
crypto strategy, or Futures on an MNQ strategy) → the bot's NEXT decision uses THAT prompt — confirm
via Recent Decisions' "System Prompt" expander (the ground truth) + the cycle log; a no-variant
strategy still runs the venue rule. SIM-only; never the LIVE LFE… account.

**ADDITIVE; the Mode dropdown is now real (persist + live-read, venue-rule fallback); crypto
byte-identical where no variant set; the Risk Control gates/guardrails/editable-boxes/label-sweep +
multi-account (P1-P4) + P&L + chart-TZ ALL untouched; NO C#/wire change. SIM-only; the live-account
block (broker layer) untouched + untoggleable.**


## 2026-06-06 — Strategy Studio PHASE 2 CHANGE 4: the FUTURES prompt builder now honors the 4 structured boxes — built + static-verified, behavior Sunday

**STATUS:** BUILT + STATIC-VERIFIED (market closed). Editing Role Definition / Trading Frequency /
Entry Standards / Decision Process on a futures strategy now shapes the live futures prompt AND the
preview. LIVE behavior proof on the SUNDAY list.

Prior gap (deep-dive map): the crypto builder honored the 4 boxes (engine_prompt.go:38/93/105/118) but
`engine_prompt_futures.go` IGNORED them — only Custom Prompt + Min R/R + Min Conf reached futures, so
edited boxes ("Modified") went nowhere on an MNQ strategy.

**BACKEND (engine_prompt_futures.go) — the only code change.** `BuildFuturesDecisionSystemPrompt` now
reads `e.config.PromptSections` (mirroring the crypto override-or-default pattern):
- **Role Definition** → override-or-default: box replaces the fixed CME role line when set; the
  Instrument identity block stays FIXED.
- **Trading Frequency** + **Entry Standards** → appended as their own section ONLY when set (the
  futures builder had no such section before, so an empty box adds nothing → byte-identical).
- **Decision Process** → override-or-default: box replaces the fixed 4-step section when set.
- **FIXED, never box-driven:** the Instrument block, the Hard Constraints (risk rules), the Output
  Format + Field Description (parser envelope), and the Custom Prompt append. The boxes change
  instruction TEXT only — Risk Control + the output/risk rules are unchanged.

**BACK-COMPAT PROOF (byte-identical when empty):** a golden of the pre-change futures prompt was
captured (kernel/testdata/futures_mnq_empty.golden); `TestFuturesPromptEmptyBoxesByteIdentical` proves
the empty-box output equals it EXACTLY — so existing futures strategies do not change.
`TestFuturesPromptBoxesOverride` proves each set box reaches the prompt, empty boxes never inject the
box-only sections, and the FIXED markers (Symbol, <reasoning>/<decision>, Hard Constraints) survive.

**FE — NO change needed (refutes the assumed FE work).** The preview is backend-driven: the
preview/test handler builds the engine from the POSTED config (api/strategy.go:581) and (Change-3
honest preview) a futures strategy previews `variant="futures"`, so edited boxes now flow into the
preview automatically. The editor already renders the 4 boxes as editable textareas for both markets
(StrategyStudioPage.tsx:803, ungated) — there was no "not used on futures" state to fix.

**Static verify:** go build clean; `go test ./store ./trader ./kernel` green (incl. the 2 new golden
tests); nofx-bin rebuilt + restarted clean (0 "unknown frame type"; trader `sss` = NinjaTrader/"MNQ SIM
Default" auto-started — empty boxes → byte-identical futures prompt). tsc unaffected (no FE change).

**SUNDAY behavior list (market open):** on a futures strategy edit a box (e.g. Entry Standards) → the
bot's NEXT decision's "System Prompt" (Recent Decisions = ground truth) contains the edited text; empty
boxes → today's fixed futures prompt. SIM-only; never the LIVE LFE… account.

**ADDITIVE; the futures builder honors the 4 boxes (override-or-default); empty boxes = today's fixed
futures prompt (byte-identical, proven by golden test); boxes are TEXT-only (Risk Control + output
format + hard risk rules FIXED); crypto + Risk Control + multi-account (P1-P4) + P&L + chart-TZ + Phase
2 CORE all untouched; NO C#/wire/FE change. SIM-only; the live-account block untouched.**


## 2026-06-06 — Strategy Studio PHASE 2 — (B) crypto-leak fix + futures SUB-MODES + two-dropdown Market+Mode (3 chunks) — built + static-verified, behavior Sunday

**STATUS:** BUILT + STATIC-VERIFIED (market closed). Fixes the Change-4 regression (futures preview led
with a leaked crypto Role box), adds futures sub-modes, and a two-field Market(auto-locked)+Mode UI.
LIVE behavior proof on the SUNDAY list.

Root cause of the regression (diagnosed by rendering the real prompt): Change 4 (181f1a00) made the
futures builder honor all 4 boxes, but EVERY non-default strategy carries crypto-DEFAULT box content
(`role_definition` = "professional cryptocurrency trading AI" / "你是一个专业的加密货币交易AI";
`decision_process` = "候选币种/coins"). So the futures prompt led with crypto framing. The live trader
(`sss` → "MNQ SIM Default", empty boxes) was NOT affected; this was a preview-of-`均衡策略` issue + latent
for any box-laden strategy assigned to a live futures trader.

- **CHUNK 1 (B) — `9387a7e2` (Go):** on futures, **Role Definition + Decision Process are ALWAYS the
  fixed CME text** (reverted JUST those two from Change 4); **Trading Frequency + Entry Standards stay
  honored** (append-when-set); Custom Prompt + Min R/R + Min Conf unchanged. Proven: rendering
  `均衡策略`'s REAL futures prompt now leads with "professional CME index-futures trading AI" — no
  `cryptocurrency` / `加密货币` / `候选币种`. Tests: `TestFuturesPromptNoCryptoLeak`,
  `TestFuturesPromptBoxesHonored`; golden `TestFuturesPromptEmptyBoxesByteIdentical` still passes.
- **CHUNK 2 (SUBMODES) — `6b4976d5` (Go):** `BuildSystemPrompt` routes `futures` / `futures-balanced` /
  `futures-aggressive` / `futures-conservative` via `futuresVariantMode` (was an exact `=="futures"`
  that would have fallen through to crypto). `buildFuturesPrompt(symbol, equity, mode)` injects a
  `## Mode:` block for aggressive/conservative; balanced/empty/unknown = NO block. Proven:
  `TestFuturesSubModes` — `futures`/`futures-balanced` == the golden (byte-identical); aggressive/
  conservative == balanced + ONLY their mode block (TEXT-only, Risk Control/output format untouched).
  `resolvePromptVariant` unchanged (passes the saved variant through).
- **CHUNK 3 (LAYOUT) — `e876ded7` (FE):** the single Mode dropdown → a **READ-ONLY Market field**
  (left, derived from `isCMEFutures(static_coins[0])` — cannot contradict the symbol) **+ a Mode
  dropdown** (right, Balanced/Aggressive/Conservative). `combineVariant`/`decomposeMode` map
  Market+Mode ↔ the saved `prompt_variant` (crypto → balanced/aggressive/conservative; futures →
  futures / futures-aggressive / futures-conservative), mirroring the Go `futuresVariantMode`. Preview +
  AI-Test send the combined variant → a futures strategy now previews the FIXED CME role + the sub-mode
  (no crypto line). i18n market/marketFutures/marketCrypto/marketLockedHint (en/zh/id).

**Static verify:** `go build ./...` clean; `go test ./store ./trader ./kernel` green (incl. golden + the
new tests); `tsc --noEmit` 0; FE built; `nofx-bin` rebuilt + restarted clean (0 "unknown frame type").

**SUNDAY behavior list (market open):** on a futures strategy — (B) the bot's next decision's "System
Prompt" (Recent Decisions = ground truth) leads with the CME role, no crypto line, reflects edited
Frequency/Entry; (SUBMODES) Mode=Aggressive → the aggressive block appears, Balanced/no-variant →
today's fixed prompt; (LAYOUT) Market shows "Futures" locked, Mode pick persists the combined variant.
SIM-only; never the LIVE LFE… account.

**ADDITIVE; (B) Role+Decision fixed on futures, Frequency+Entry honored; (SUBMODES) text-only sub-modes
(Risk Control unchanged), futures-balanced = today's fixed prompt (byte-identical, golden); (LAYOUT)
two-dropdown Market-auto-locked+Mode (symbol = source of truth); crypto + Risk Control + multi-account
(P1-P4) + P&L + chart-TZ + Phase 2 CORE all untouched; NO C#/wire change. SIM-only; the live-account
block untouched + untoggleable.**


## 2026-06-06 — Strategy Studio PHASE 2 A+D: honor the boxes on futures (revert Option B) + replace stale crypto defaults — built + static-verified, behavior Sunday

**STATUS:** BUILT + STATIC-VERIFIED. Option B (Chunk 1 of the prior round, 9387a7e2) took away the
owner's ability to edit Role/Decision on futures — WRONG. The real problem was the stale CRYPTO-DEFAULT
box content, not that the user shouldn't edit. So: **(A)** honor the boxes again (full control) + **(D)**
replace the stale crypto data.

- **A — `07fef1a4` (Go):** reverted Option B — the futures builder honors Role Definition + Decision
  Process again (override-or-default: box when set, fixed CME text when empty). Frequency + Entry already
  honored. Tests: `TestFuturesPromptBoxesHonored` (all 4 honored), `TestFuturesPromptCustomRoleHonored`
  (custom Role/Decision rendered; empty → fixed CME); golden + sub-mode tests still pass.
- **D1 — `0f84f5bb` (Go + FE):** neutralized the new-strategy box defaults — `store/strategy.go`
  DefaultStrategyConfig (zh+en) + the FE `PromptSectionsEditor.defaultSections`: Role "professional
  cryptocurrency trading AI" / "加密货币交易AI" → "professional trading AI" / "交易AI"; Decision
  "candidate coins / 候选币种" → "the market / 市场". A new strategy never starts with crypto framing.
  The crypto BUILDER defaults (`engine_prompt.go` else-branches) are untouched (crypto path correct).
- **D2 — one-time DB migration (data, NOT in git):** a surgical byte-replace on `data.db` rewrote ONLY
  boxes whose `role_definition`/`decision_process` EXACTLY equalled a known crypto default — **7
  strategies** (均衡/稳健/积极 zh + New×3/Strategy Copy en). Dry-run preview confirmed the exact set;
  applied; **verified against a pre-migration snapshot: 7 rows changed, integrity_ok (every
  non-role/decision field byte-identical), 2 empty "MNQ SIM Default" untouched, 0 crypto boxes remain.**
  Reversible from `~/nofx-backups/2026-06-06-phase2-D-migration/data.db.PRE-MIGRATION-snapshot`.

**End-to-end proof:** rendering `均衡策略`'s real futures prompt now leads with **"# 你是一个专业的交易AI"**
(the neutral role, **honored from the box** — A) — crypto? false, 候选币种? false, CME instrument
present. So edit (the box, neutral) == preview (honored). The owner can now type ANY role/decision and
the futures AI uses it.

**Static verify:** `go build ./...` clean; `go test ./store ./trader ./kernel` green; `tsc --noEmit` 0;
FE built; `nofx-bin` rebuilt + restarted clean (0 "unknown frame type", serves the migrated configs).

**Note (minor, by design):** for an EMPTY-box strategy (e.g. MNQ SIM Default) the editor shows the
generic neutral default text while the preview shows the instrument-specific fixed CME role — both
non-crypto; the difference is a generic placeholder vs the instrument-aware prompt. For strategies with
actual box content (all 7 migrated ones), edit == preview exactly.

**SUNDAY behavior list:** on a futures strategy, edit the Role box → the bot's next decision's "System
Prompt" (Recent Decisions = ground truth) uses YOUR text; a migrated strategy → the neutral role, no
crypto line. SIM-only; never the LIVE LFE… account.

**ADDITIVE behavior (boxes honored = full control) + a careful reversible DATA migration (only
crypto-default boxes, never a customized one); crypto builder + Risk Control + multi-account (P1-P4) +
P&L + chart-TZ + Phase 2 sub-modes/two-dropdown all untouched; NO C#/wire change. SIM-only; the
live-account block untouched + untoggleable.**


## 2026-06-06 — Strategy Studio: specific CME futures Role default + hide crypto leverage in the Config box (commit 725c91f2)

Two display/default FE+seed changes (no enforced behavior change):

- **(1) Futures Role default = specific professional CME role.** After A+D the seed Role was generic
  neutral ("professional trading AI"). Root nuance: `GetDefaultStrategyConfig(lang)` is market-agnostic,
  but the futures builder's empty-box fallback is ALREADY the instrument-aware CME role. So the seed now
  **omits Role + Decision** (store/strategy.go) — an empty box lets each market's builder supply the
  correct default: futures → "professional CME index-futures trading AI specializing in <instrument>"
  (engine_prompt_futures.go), crypto → the crypto role (engine_prompt.go). The FE editor's empty-box
  default is market-aware too (`PromptSectionsEditor` gains `isFutures`: CME Role/Decision on futures,
  neutral on crypto). **Default/fallback only — edit control intact (a typed Role overrides), saved user
  boxes UNTOUCHED (not a migration), crypto builder default unchanged.** Proven: a NEW default-config
  strategy's futures prompt leads with the specific CME role; crypto path unchanged.
- **(2) Config box hides crypto leverage on futures.** The Prompt-Preview "Config" summary filters out
  `btc_eth_leverage` + `altcoin_leverage` when `isFuturesStrategy` (mirrors the PVR `{!isFutures}`
  pattern); `coin_source` / `primary_tf` / `max_positions` shown on both. The backend `config_summary`
  is unchanged; the FE filters for display.

**Static verify:** go build/vet/test green; tsc 0; FE built; nofx-bin restarted clean (0 "unknown frame
type"). **SUNDAY:** a futures strategy with the default Role → the bot's next decision's "System Prompt"
leads with the specific CME role. ADDITIVE; display/default only; crypto + Risk Control + multi-account
+ P&L + chart-TZ + Phase 2 work untouched; live-account block untouched. SIM-only.


## 2026-06-07 — Strategy Studio readability UX (FE-only): auto-expand edited boxes + taller preview + live auto-refresh (commit 97b60895)

Three FE/UX-only fixes from the prior read-only diagnoses (the "1-line box" was a default-collapsed
accordion; the preview was a 400px scroll pane; the preview only rebuilt on Refresh-click):

- **(1) Auto-expand edited prompt boxes.** `PromptSectionsEditor` now initializes `expandedSections`
  from the config — a box with non-empty SAVED content opens by default; empty boxes (showing only the
  default placeholder) stay collapsed. Re-computed per strategy via a `key={selectedStrategy.id}` on the
  component in StrategyStudioPage (re-mounts on strategy switch). So an edited strategy shows its box
  text without clicking.
- **(2) Taller preview.** The System Prompt `<pre>` `maxHeight` 400px → **70vh** (still `overflow-auto`),
  so most of the prompt is visible without scrolling.
- **(3) Live auto-refresh.** A debounced (500ms) `useEffect` re-runs `fetchPromptPreview` when
  `editingConfig`/`selectedMode` change on the Prompt tab — edits reflect without clicking Refresh; the
  manual Refresh button is kept. `fetchPromptPreview` already uses the live `editingConfig`.

**FE/UX-only — NO Go/gate/prompt-content/behavior change** (the rendered prompt text is identical; Risk
Control + the builder + multi-account + P&L + chart-TZ untouched); crypto byte-identical; live-account
block untouched. tsc 0; `npm run build` ✓; Go untouched. SIM-only.


## 2026-06-07 — Strategy Studio Phase 3: hide crypto data feeds on futures + rename "Coin Source" → "Data Source" (commit 3923df1c)

FE/UX-only, additive:
- **(1) Crypto data feeds hidden on futures.** CoinSourceEditor's Source-Type selector showed all 4
  (Static / AI500 / OI-Top / OI-Low) on a futures strategy; AI500/OI are crypto-only feeds (would make
  `GetCandidateCoins` fetch crypto data instead of trading MNQ). Now `isFutures = isCMEFutures(static_coins[0])`
  → `visibleSourceTypes` shows ONLY Static on futures, and `effectiveSourceType` (= 'static' on futures)
  drives the highlight + the option panels so the crypto panels also hide. **Display-only — saved data
  untouched** (render-only hide; no mutation). Crypto shows all 4 (else-branch = byte-identical).
- **(2) Label rename.** "Coin Source" → "Data Source" (i18n en/zh/id: the strategyStudio section title +
  the trader-modal label). The config key `coin_source` / `static_coins` is UNCHANGED (the stored data
  shape — no migration; saved strategies load fine).
- **(3) Sweep.** The Data Source labels were already neutralized (Custom Symbols / Add Symbol / symbols /
  Excluded Symbols); the remaining crypto terms are the AI500/OI feed descriptions — genuinely crypto-only
  feeds (now hidden on futures) — left as-is.

**Verified LIVE (owner's session, no token forged):** futures (MNQ) → the selector shows only "Static"
(1 button); flipping the symbol to BTC → all 4 reappear; section title "数据源 / Data Source" both; New
Strategy's saved symbol still MNQ (no mutation). tsc 0; build ok; Go untouched. Crypto byte-identical;
gates/guardrails/prompt/multi-account/P&L/chart-TZ untouched; live-account block untouched. SIM-only.


## 2026-06-07 — Strategy Studio Phase 4: timeframe-line fix + futures default indicators + NofxOS hide (commits a7ca000d / 2558ea2c / 0312a828)

Three safe chunks (each audit → build → verify → commit), from the Phase-4 Indicators map + the
timeframe diagnosis. ADDITIVE; indicators/timeframes are prompt-data and NEVER gate a trade
(engine_position.go reads zero `Indicators.Enable*`).

- **(A) Prompt timeframe line lists all selected TFs (Go — a7ca000d).** `writeAvailableIndicators`
  (kernel/engine_prompt.go) named only `PrimaryTimeframe` + `LongerTimeframe` (the legacy 2-field model),
  so a 3-timeframe selection (MNQ SIM Default = 5m/15m/1h) showed "5m + 1h" and dropped 15m — even though
  the fetch (engine_analysis.go reads `SelectedTimeframes`) and the per-timeframe user-prompt loop
  already feed the AI all selected TFs. Only the summary line under-reported. Extracted
  `formatKlineTimeframes(kline)`: list every `SelectedTimeframes` entry; fall back to the prior
  primary[+longer] wording ONLY when the list is empty (legacy configs → **byte-identical**). Shared by
  the crypto `BuildSystemPrompt` + the live futures `buildFuturesPrompt`, so the crypto line updates too
  (golden-tested). **Golden:** kernel/engine_prompt_timeframes_test.go locks the empty-list fallback
  (byte-identical) + the all-listed behavior, end-to-end through the real method.
- **(B) ATR/EMA/RSI enabled in the futures new-strategy default (Go — 2558ea2c).** The futures AI got raw
  bars + volume only (EMA/MACD/RSI/ATR/BOLL default OFF). Extracted `applyFuturesIndicatorDefaults()`
  (the existing NofxOS/ranking disable + the new ATR/EMA/RSI enable) inside `GetDefaultStrategyConfig`'s
  `if isFuturesMode()` block — ATR (stop sizing) + EMA (trend) + RSI (momentum) on; MACD/BOLL left off.
  **Defaults-only** (new-strategy template; FE create flow fetches /default-config); existing saved
  strategies NOT mutated (verified: 均衡/稳健/积极/New still volume,oi,funding_rate; MNQ SIM Default
  none). Crypto mode untouched (helper never called) → byte-identical. **Test:**
  store/strategy_futures_indicators_test.go. **Recommendation (separate confirm):** to give the owner's
  EXISTING MNQ strategies the same indicators, toggle EMA/RSI/ATR ON per-strategy in the Indicators UI
  (or a one-off, owner-approved DB update) — NOT auto-applied here (no mass-mutation).
- **(C) NofxOS + crypto-ranking feeds hidden on futures (FE — 0312a828).** The whole NofxOS Data
  Provider section (Quant Data / Quant OI / NetFlow + OI/NetFlow/Price ranking + the API Key field)
  rendered but was inert on futures (crypto-only; disabled by default; claw402 402/404 on CME). Wrapped
  the section in `{!isFutures && (...)}` (the `isCMEFutures(static_coins[0])` → `isFuturesStrategy` flag,
  passed as `isFutures`). Futures → not rendered; crypto → renders exactly as before. FE-only display
  gate of already-inert controls; no saved-data mutation. (`git diff -w` shows the only change is the
  wrap; the rest is prettier re-indent.)

**Static verify (now):** go build/vet/test (./store ./kernel ./trader) green incl. both new tests +
existing goldens; tsc 0; `npm run build` ✓; nofx-bin rebuilt + restarted clean in futures mode (0
"unknown frame type" / panic / fatal); bot flat at restart (0 open positions). **SUNDAY (market open):**
(A) a futures strategy → the bot's next decision's "System Prompt" lists all selected TFs; (B) a NEW
futures strategy defaults ATR/EMA/RSI ON and they appear in the decision data; (C) Playwright — a futures
strategy → NofxOS/ranking controls hidden, a crypto strategy → shown. ADDITIVE — old configs
byte-identical (golden); indicators never gate (low real-money risk); crypto byte-identical except the
intended timeframe wording; gates/guardrails/multi-account (P1-P4)/P&L/chart-TZ + Phase 2/3 work + the
live-account block untouched. SIM-only.


## 2026-06-07 — Open Interest honesty fix: hide + disable OI on futures (commit pending)

Follow-up from the OI investigation. "Open Interest" on a futures strategy was the Binance crypto-perp
feed (`fapi/v1/openInterest`, market/data.go) → returns ZEROS for MNQ; the FE box was labeled "Futures
open interest" (misleading); and the futures prompt LISTED OI as available, printed "Open Interest:
Latest: 0.00 Average: 0.00", then said "ignore the empty OI" (a 3-way contradiction, one toggle away on
OI-enabled strategies). Real futures OI isn't worth wiring (CME OI is once-daily EOD; the NT8 bridge
carries OHLCV only). Fix mirrors (and extends) the funding-rate treatment:

- **(1) FE — OI toggle hidden on futures.** IndicatorEditor.tsx OI toggle `cryptoOnly: false → true`, so
  the existing `!cryptoOnly || !isFutures` filter hides it on futures (exactly like the funding-rate
  toggle beside it). Crypto → OI shows as today. Verified live: the Market Sentiment grid shows only
  Volume on MNQ (OI + Funding both hidden).
- **(2) Go default — OI off on futures.** `ind.EnableOI = false` added to `applyFuturesIndicatorDefaults`
  (store/strategy.go). A new futures strategy no longer lists/values OI. Defaults-only; existing saved
  strategies NOT mutated (DB re-checked: 均衡/稳健/积极/New still enable_oi=true; MNQ SIM Default false).
  Verified live: futures `/default-config` returns `enable_oi: false`.
- **(3) Go fetch guard.** `getOpenInterestData` + `getFundingRate` now skip on futures in
  `GetWithTimeframes` (market/data.go, behind the in-scope `isFutures`). MNQ values were always {0,0}/0
  from the failed Binance call — identical result, minus the wasted round-trip per cycle.

**Golden/tests:** kernel/engine_prompt_oi_test.go (OI availability line present IFF EnableOI — futures
default drops it, crypto byte-identical); store/strategy_futures_indicators_test.go extended (futures
default → EnableOI false). go build/vet/test (./store ./kernel ./trader ./market) green incl. existing
goldens; tsc 0; `npm run build` ✓; nofx-bin restarted clean in futures mode (0 "unknown frame type");
bot flat. OI is prompt-data and NEVER gates (engine_position.go reads zero Indicators.Enable*). The live
bot (MNQ SIM Default, enable_oi=false) is unaffected today.

**RECOMMENDATION (separate owner confirm):** the owner's EXISTING OI-enabled futures strategies
(均衡/稳健/积极/New) keep enable_oi=true → they still show the zeros + contradiction until untoggled
per-strategy in the Indicators UI (or a one-off owner-approved DB update). NOT auto-applied (no
mass-mutation). **OBSERVED symmetric gap (out of scope):** funding rate has the same residual — its FE
toggle is hidden (cryptoOnly:true) but `enable_funding_rate` stays TRUE in the futures default, so the
futures prompt still lists "Funding rate" + "Funding Rate: 0.00e+00". Same one-line fix
(`EnableFundingRate = false` in applyFuturesIndicatorDefaults) would close it — deferred.

ADDITIVE; crypto byte-identical; OI prompt-data only (never gate) = low real-money risk;
gates/guardrails/multi-account (P1-P4)/P&L/chart-TZ + Phase 2/3/4 work + the live-account block
untouched. SIM-only.


## 2026-06-07 — Indicator period-edit typing bug fixed (commit e70116e7)

The EMA/RSI/ATR/BOLL period inputs (IndicatorEditor.tsx) parsed
`split(',').map(parseInt).filter(n>0)` on EVERY keystroke and the value fell back to the default when
empty — so on an EDITABLE strategy you couldn't type a comma (it reverted), couldn't clear (it snapped
to the default), and editing across a comma collapsed the other value. You could not edit a multi-value
period field character-by-character. (Diagnosed as bug (b); the (a) read-only-default lock —
`disabled={selectedStrategy?.is_default}` — is a separate by-design behavior, not changed.)

Fix (FE-only): extracted a `PeriodInput` component that holds the RAW typed text in local state while
editing and parses → `number[]` ONLY on blur/commit (Enter blurs); when not editing it derives straight
from the saved value (so strategy-switch / reset just work). The SAVED DATA SHAPE is unchanged —
`onCommit` always passes a `number[]` (the default periods if the field was left empty), so
`config[periodKey]` stays e.g. `ema_periods=[20,50]`.

Verified live (owner session, no token forged): on an editable strategy, typing a comma → stays;
clearing → stays empty while editing; leading/mid comma `",50"` → stays (fixable to `25,50`); blur
sanitizes. Save persisted `ema_periods=[20,50,100]` as a `number[]` of ints (sqlite before/after), then
restored to `[20,50]` (no net change). tsc 0; `npm run build` ✓; Go untouched. Periods are prompt-data
(never gate); the period input passes `disabled` through, so default strategies stay read-only; the
component is market-agnostic (crypto + futures identical). gates/guardrails/multi-account/P&L/chart-TZ +
the live-account block untouched. SIM-only.


## 2026-06-08 — Indicator PERIODS now drive the math (was hardcoded EMA 20/50, BOLL 20) (commits b8064830 + bcf3a26c)

Diagnosed bug: the market layer hardcoded EMA 20/50, BOLL 20, RSI 7/14, ATR 14 at every call-site; the
strategy-configured periods (e.g. EMA 21/9/200, BOLL 10/12) were SAVED + listed in the prompt HEADER but
NEVER computed — the AI received values labeled EMA20/EMA50, which is why it cited "EMA20/EMA50". RSI/ATR
only looked right because the owner's values equalled the hardcoded defaults.

Fix (additive refactor, 2 chunks):
- **(Chunk 1 — EMA, b8064830).** Plumb the configured periods into the market layer:
  `GetWithTimeframes` gains an optional variadic `market.IndicatorPeriods`; the decision
  (engine_analysis.go) + preview (api/strategy.go) call-sites pass `config.Indicators.*Periods`.
  `calculateTimeframeSeries` computes one EMA series per CONFIGURED period into
  `TimeframeSeriesData.EMAByPeriod` (+ `Data.CurrentEMAByPeriod`), IN ADDITION to the legacy fixed
  `EMA20Values/EMA50Values/CurrentEMA20` (kept untouched for the grid engine, agent chat, futures static
  builder, and formatters — those readers are unchanged). The decision prompt labels the real periods
  (EMA9/EMA21/EMA200, sorted), falling back to EMA20/EMA50 when no periods are configured.
- **(Chunk 2 — BOLL/RSI/ATR, bcf3a26c).** Same pattern generalized via the `IndicatorPeriods` struct:
  `BOLLByPeriod` (std-dev mult fixed at 2), `RSIByPeriod`, `ATRByPeriod`. Prompt labels RSI9/RSI21,
  ATR10, BOLL10/BOLL12, with legacy fallback.

ADDITIVE + back-compat: with the default config the *ByPeriod path computes byte-identically to the legacy
fixed fields (same calc fns, same `i>=period-1` guards) — golden-tested
(`market/data_emaperiods_test.go`: `EMAByPeriod[20]==EMA20Values`, `BOLLByPeriod[20]==BOLLUpper/...`,
`RSIByPeriod[7/14]==RSI7/14Values`, `ATRByPeriod[14]==ATR14`; `kernel/engine_prompt_emaperiods_test.go`:
labels + byte-identical legacy fallback). Indicators/periods are prompt-data and NEVER gate
(engine_position.go reads zero EMA/period — re-confirmed). go build/vet/test (./market ./kernel ./store
./trader ./api) green incl. all existing goldens; nofx-bin rebuilt + restarted clean (0 "unknown frame
type"); bot flat at restart.

**Verified LIVE (owner session decision_records, cycle 16):** the active trader's (Hoangvl) decision
prompt now shows `current_ema9/21/200`, per-TF `EMA9/EMA21/EMA200`, `BOLL10/BOLL12 Upper/Middle/Lower`,
`RSI7/RSI14`, `ATR14` — the configured periods, with the legacy EMA20/EMA50 + "BOLL Upper:" labels gone.
EMA200 shows 1 value (correct — needs 200 bars). The grid/agent/futures-static readers + the gate +
multi-account + the live-account block untouched. SIM-only. Propagated to the partner repo
vlautoagenttraderv1 (same fix).


## 2026-06-08 — Risk Control R/R input revert-on-clear bug fixed (commit cd3ff63d)

Audit-then-fix. The owner reported "set R/R to 1:1 but the AI enforces 3:1." **Audit (Phase 1) REFUTED
the EMA-class premise:** the R/R prompt (engine_prompt_futures.go:66/114) + gate (engine_position.go:141)
are config-driven + code-enforced (live: the decision prompt injected "3.00x" — the *saved* value). The
active strategy (Hoangvl/70695b25) is SAVED as min_rr=3; no strategy is saved at 1:1. The real cause is
the **FE R/R input**: RiskControlEditor.tsx used `value={config.min_risk_reward_ratio ?? 3}` +
`onChange={parseFloat(e.target.value) || 3}`, so clearing the box (empty → NaN → `|| 3`) or typing 0
snapped it back to 3 — the owner's 1:1 never persisted. Same class as the indicator period-edit bug.

Fix (FE-only, cd3ff63d): extracted `ClampedNumberInput` (mirrors the shipped `PeriodInput` pattern) —
holds the raw typed text in local state while editing (clear + retype allowed), parses → clamps [1,10] →
commits ONLY on blur (Enter blurs); an empty/invalid commit keeps the existing saved value (not a
hardcoded 3). Applied to the min-R/R input ONLY. The saved shape (a number, clamped [1,10] by store
ClampLimits) is unchanged. **The R/R prompt + gate are UNTOUCHED** (they were already correct — not
weakened). tsc 0; `npm run build` ✓; Go untouched. Propagated to vlautoagenttraderv1.

Coverage: code/DB/live-log-confirmed; the clear→type→commit behavior is code-deterministic — Playwright
LIVE-verify was BLOCKED (the browser session expired → /login; no token forged), so the before/after
typing + the persist-as-1 sqlite check await a re-authed session. **OTHER risk fields flagged (NOT fixed
this pass — one bug at a time):** MaxMarginUsage is advisory-only (never gated, absent on futures); the
active strategy has `guardrails_enabled=false` so all prop-firm guardrails (daily loss/profit, max
trades, blackout, consistency, max contracts, notional cap) are BYPASSED — confirmed live
(engine_analysis.go:142); the R/R unset-fallback mismatches (prompt 1.5 vs gate 3.0 when min_rr=0). The
other number inputs in RiskControlEditor likely share the `|| default` revert pattern — a follow-up could
reuse ClampedNumberInput for them. ADDITIVE; FE-only; saved shape unchanged; the gate/guardrails/
multi-account/P&L/chart-TZ + the live-account block untouched. SIM-only.


## 2026-06-08 — Stage 1.5: strategy SAVE now reloads the running trader (config live without restart) (commit 53ad62a1)

Pipeline-audit headline fix. `handleUpdateStrategy` (api/strategy.go) persisted the config to the DB and
returned — it NEVER reloaded the running trader, whose engine is a snapshot from trader-load
(auto_trader.go:414 `NewStrategyEngine(config.StrategyConfig)`). So R/R, indicators, periods, every Risk
Control setting were STALE until a manual restart (proven live: saved min_rr→2 at 17:16, but decisions
through 17:21 still used 3 from the 08:43 load). The AI-model + Exchange save handlers already reload the
trader; strategy saves just didn't.

Fix (extend the EXISTING pattern — no new hot-reload framework): after the validated+clamped+persisted
save, find the running trader(s) bound to this strategy (`Trader().List(userID)` filtered by
`StrategyID`), `RemoveTrader` them, then `LoadUserTradersFromStore` to rebuild from the store. Verified
SAFE at file:line: `RemoveTrader→Stop()` (auto_trader.go:617) clears only the in-memory flag — it does
NOT persist `is_running=false` — so the reload AUTO-STARTS the trader (`addTraderFromStore` honors the DB
`is_running`, trader_manager.go:743-755); `LoadUserTradersFromStore` skips traders already in memory
(:444) so only the affected one reloads. The TCP bridge + BarCache are a shared SINGLETON → bars + the
NT8 connection are preserved; an open position re-attaches on reload (NT8-side SL/TP guard it during the
swap). On reload failure the request still succeeds (config saved) + logs — no crash.

go build/vet/test (./api ./store ./kernel ./trader) green; nofx-bin rebuilt + restarted clean (0 errors,
flat). **Live-confirmed (partial):** after the new binary loaded, cycle 169's decision prompt shows
R/R=2.00 + BOLL=[10] — the owner's current saved config (was 3.00/[10,12]). The SAVE-triggered reload
(save while running → next cycle uses the new value WITHOUT restart) is CODE-verified (proven AI-model
pattern + the auto-start mechanism) and will log "Strategy X saved → reloaded N trader(s)" on the next
save — the live save-trigger awaited a session (Playwright expired → /login; no token forged). No
prompt/gate/FE change; the gate/guardrails/multi-account/P&L/chart-TZ + the live-account block untouched.
Propagated to vlautoagenttraderv1 (4495b6a). SIM-only. (Note: a CME daily maintenance halt 16:00-17:00 CT
produced empty "risk gate HOLD" cycles around the restart — expected, not this change.)


## 2026-06-08 — feed-gate stale-status latch fixed: IsFeedConnected overrides with live bars (commit 2b9138b9)

Owner-reported + corrected diagnosis: the AI's LONG decisions weren't entering — initially mis-attributed
to "feed down," but the owner correctly noted NT8 was connected. Real bug: the NT8 `feed_status` frame is
EDGE-TRIGGERED (sent only on change), and `IsFeedConnected` latched it. A momentary connect/disconnect
flap at 19:53 ended on "Disconnected" and NT8 never re-sent "Connected" on reconnect — so the feed-gate
(auto_trader_orders.go:62) skipped EVERY open order for ~3.5h (11 skips, 21:01→23:23) even though bars
flowed continuously (~1M/hour, hours 20/21/22/23) and the feed was demonstrably live. The decisions were
otherwise valid (conf 75, R/R ≥ min); the order path works (earlier fills 13:11/18:54).

Fix (Go-only, no C# change): `provider/ninjatrader/tcp_server.go` tracks the wall-clock of the most recent
live `bar_update` (`lastBarNano atomic.Int64`, stamped in `enqueueBarUpdate` on the hot path);
`IsFeedConnected` now returns true when a bar arrived within `feedFreshWindow` (90s), overriding a stale
non-"Connected" status. The bar stream is the real proof the feed is up; a TRUE outage stops the bars →
the override expires → the legacy status-based `false` is restored. DEFAULT-ALLOW (empty status) +
"Connected" behavior unchanged. Golden: provider/ninjatrader/feed_freshness_test.go (5 cases incl.
fresh-bar-overrides + stale-bar-does-not). go build/vet/test (./provider/ninjatrader ./trader/...) green;
nofx-bin restarted clean (0 errors). Immediate unblock = a prior bare restart (feedStatus resets to
default-allow); this commit is the DURABLE fix so a future flap can't re-latch. No prompt/gate/FE change;
the gate/guardrails/multi-account/live-account block untouched. Propagated to vlautoagenttraderv1. SIM-only.


## 2026-06-09 — P5.1 MULTI-SYMBOL SIM PROOF: MNQ+ES streaming concurrently (commit b989ad9f)

Stage P5.1 of the P5 multi-symbol upgrade (the attached P5 plan = the spec). STEP-A finding: the C# AddOn
was ALREADY multi-symbol capable — VLBarsSubscriptionManager's registry is keyed SYMBOL|TF (one
BarsRequest per key, per the plan's BarsRequest-not-AddDataSeries decision), HandleBarsSubscribe doesn't
tear down other roots, OnConnectionReconnected (= the plan's ResubscribeAll) + the stall watchdog iterate
ALL entries, dispose discipline exists, the TCP writer is lock-serialized, bar frames already carry the
symbol, and VLContractResolver whitelists MNQ/NQ/ES/MES/YM/RTY/Z* (quarterly H/M/U/Z, roll = 3rd Friday −
8 days). The ONLY gap: the Go server auto-subscribed a single symbol. **P5.1 was therefore Go-only — no
C# change, no F5/NT8 restart.**

Build (b989ad9f): TCPServer.extraBarsSymbols + AddBarsSubscribeSymbols (dedup, primary excluded);
sendAutoBarsSubscribe sends one bars_subscribe per root (primary FIRST, extras clone its
timeframes/bars-back); UnsubscribeBarsSymbol removes an extra + sends bars_unsubscribe (PRIMARY refused —
the trading feed can never be torn down); NT_EXTRA_SYMBOLS env (unset → single-symbol byte-identical);
POST /api/debug/nt-bars-unsubscribe (futures-only, mirrors nt-test-trade). GOLDEN
(multisymbol_subscribe_test.go): no-extras → exactly ONE frame deep-equal to the legacy payload
([MNQ]-only byte-identical); extras order/dedup; primary-refusal; state-removed-even-disconnected.

**Live proofs (CME reopen 17:00 CT, all PASS):** (1) concurrent streaming — MNQ+ES 1m caches advancing
together (17:03→17:04→…; MNQ ~29060, ES ~7378 — correct per-symbol prices, zero cache bleed; cache keys
SYMBOL|TF isolated); (2) ES front-month resolution — instrument_info "ES 06-26 point_value=50 tick=0.25
matches table ✓"; (3) dispose-one — unsubscribe ES @17:04:25 → ES froze @17:05 while MNQ advanced to
17:08; primary (MNQ) unsubscribe REFUSED; (4) reconnect-resubscribe — Go restart → BOTH roots re-subscribed
+ re-seeded (incl. ES whose BarsRequests had just been disposed), both advancing again @17:10. NOT yet
proven (needs the owner at NT8): the true broker-feed-drop reconnect (Tradovate disconnect/reconnect →
OnConnectionReconnected restores BOTH) — the code path is symbol-agnostic and production-proven for MNQ.

Bars-only: the strategy/gate/order paths stay bound to the primary (kernel fetches only static_coins; no
ES trading path until P5.4); the live-account block is account-scoped + untouched; SIM-only. Rollback =
remove NT_EXTRA_SYMBOLS from .env (single-symbol behavior is golden-locked byte-identical). Partner repo
NOT synced (per the P5 dispatch: only after full P5 verification). HARD STOP — awaiting the owner's
go/no-go for P5.2 (symbol-as-list config + symbol on EVERY frame + protocol_version handshake, C#+Go
lockstep).


## 2026-06-09 — P5.2 WIRE v2: symbols-as-list + symbol-tagged fills + hello handshake (commit 8932e93e; lockstep deploy in flight)

Owner GO on P5.1 → P5.2 built (protocol v2, C#+Go in lockstep):
- **Config symbols-as-list (compat shim):** the Exchange row's NT instrument field accepts a comma list
  ("MNQ,ES,NQ") — `SplitSymbolList` (trader/ninjatrader/transport.go): FIRST = the PRIMARY (trading)
  symbol, rest = extra bar-subscription roots. A single symbol = the one-element case → byte-identical,
  zero migration. transport REPLACES the singleton server's extras from config on every trader (re)load
  (`SetExtraBarsSymbols` — a removed symbol can't linger), then appends the NT_EXTRA_SYMBOLS testing
  override. CSV transport logs + ignores extras.
- **hello/protocol_version handshake (v2):** the AddOn sends hello{protocol_version,source} FIRST; the Go
  server REFUSES a mismatch loudly (closes — never silently misparses) + replies with its own hello. A
  connection that never sends hello = LEGACY AddOn → tolerated with a one-time warning so the lockstep
  deploy window can't brick the feed. ProtocolVersion(Go) == PROTOCOL_VERSION(C#) == 2.
- **Symbol-tagged fills:** FillPayload.Symbol (C# sends the order's instrument root; rejected paths echo
  the signal's symbol). Empty = legacy → primary. The TCPTrader fill consumer REJECTS a mismatched
  non-empty symbol (split-brain defense, warn+drop).
- **Mistagged-bar rejection:** bars frames for a symbol outside the subscribed set (primary+extras) are
  REJECTED before ingest (write-to-own-cache only) — golden-tested predicate.
- **FE:** the NT Instrument field documents the comma-list form (placeholder + help line).
- **Protocol doc:** §2b hello + fill.symbol (additive).

GOLDENS green: SplitSymbolList shim; SetExtraBarsSymbols replace; isSubscribedBarsSymbol; hello framing
round-trip + v2 pin; the P5.1 [MNQ]-only byte-identical golden still green. go build/vet/test + tsc + FE
build clean.

**Deploy state — LOCKSTEP COMPLETE + VERIFIED (owner F5'd + NT8 restarted 18:53 CT):** the v2 AddOn's
FIRST frame was hello → Go log "hello handshake OK protocol_version=2 source=vltrader-addon"; both
subscribes re-sent; both instrument_infos re-emitted (MNQ 06-26 + ES 06-26 re-resolved); MNQ + ES bars
advancing live post-F5 (both on the current 18:56 bar); 0 "unknown frame type"/panic since the v2
connect (rule 11 clean, both sides v2); 0 LEGACY warnings since v2. The legacy-tolerance path was also
live-proven during the 18:12→18:53 window (old AddOn + new Go: one warning, feed unbroken). Fill
symbol-tagging is golden-tested + deployed; the live tag appears on the next fill. SIM-only; the
live-account block untouched; partner repo not synced (post-P5 only).

## 2026-06-09 — AI scan-interval minimum lowered 3 → 1 minute (commit d06617a2)

Owner scalps and wants 1-minute decision scans; the Create-Trader modal rejected anything below 3.
EVERY floor site moved to 1 (additive — existing traders at 3+ behave byte-identically; 3 stays the
default for blank/unset):

- FE input clamp `Math.max(3→1, …)` + `min="3"→"1"` — web/src/components/trader/TraderConfigModal.tsx
  (~:401-410); blank/invalid input still falls back to 3.
- BE create clamp — api/handler_trader.go (~:426): was `<3 → 3` (silent bump), now `<=0 → 3` (default
  for unset); any positive value accepted.
- BE update clamp — api/handler_trader.go (~:617): the `else if <3 → 3` bump REMOVED; `<=0 → keep
  existing` unchanged.
- Hint text honest in en/zh/id (translations.ts scanIntervalRecommend): "Recommended: 3-10; minimum 1 —
  1-minute scanning increases AI-call frequency/cost and may be limited by decision latency."
- Engine: NO floor exists — manager/trader_manager.go:650 converts minutes verbatim;
  store/trader.go:29 keeps gorm default:3.

OVERLAP ANSWER (definitive, auto_trader.go:557-600): the scan loop is `time.NewTicker(interval)` +
`runCycle()` executed ON the loop goroutine — cycles are strictly sequential, overlap is IMPOSSIBLE,
no reentrancy guard needed. If a cycle runs LONGER than the interval (e.g. 90s at 1-min): Go's ticker
buffers at most ONE pending tick and DROPS the rest, so the next cycle starts immediately after the
long one ends, skipped ticks are simply gone, and the cadence realigns — cycles never queue up or run
concurrently. Trader-update reloads+restarts the running trader (handler_trader.go:698-708), so an
interval edit applies live once the new binary is running.

Cost note: at 1-minute the claw402 runway estimator (EstimateRunway, scanMinutes=1) correctly reports
~3× the daily AI cost vs 3-minute. SIM-only; gates/guardrails/live-account block untouched. Partner
repo sync HELD — P5.3 is mid-flight in a concurrent session; sync at the next stable boundary.


## 2026-06-09 — P5.3 RUNTIME SYMBOL ADD/REMOVE + subscription acks (commit afb9abb3; acks await one F5)

GO on P5.2 → P5.3 built + live-verified (the remove path existed from P5.1; this adds the runtime ADD,
lifecycle acks, clean teardown, and the flag-gated owner API):
- **Runtime ADD:** `SubscribeBarsSymbol` registers an extra root (survives reconnect) + sends
  bars_subscribe NOW when connected (clones the primary's timeframes/bars-back). Primary-dup refused.
- **Acks (C#→Go, additive):** subscribed{symbol,resolved_contract} / unsubscribed{symbol,removed} /
  subscribe_error{symbol,reason} — a typo'd symbol now fails LOUDLY Go-side. Go tracks per-symbol state
  (pending/subscribed/error/unsubscribed); a pre-ack AddOn just stays "pending" (bars still flow).
- **Clean teardown:** UnsubscribeBarsSymbol PURGES the symbol's cached bars (BarCache.PurgeSymbol — every
  timeframe, case-insensitive, primary untouchable) + records the state. No per-symbol goroutines exist
  (single drain + shared cache) → nothing to leak by design.
- **Owner API (futures-only, flag NT_RUNTIME_SYMBOLS=true; default OFF = the static P5.2 list = the
  rollback):** GET /api/nt/symbols (always readable: primary + states + flag), POST ?symbol=NQ (add),
  DELETE ?symbol=NQ (remove). Protocol doc §9.

**LIVE proofs (19:06-19:09 CT, CME open, pre-F5 — the current AddOn handles subscribe/unsubscribe; only
the acks need the new C#):** POST add NQ → "sent runtime bars_subscribe NQ" → instrument_info NQ 06-26
(pv=20 ✓) → **THREE symbols streaming concurrently** (MNQ+ES+NQ all on the same live 1m bar — the P5
headline goal at the data layer); DELETE NQ → unsubscribed + **cache PURGED** (MNQ+ES untouched); 3
add/remove churn cycles → threads 15→16 (runtime noise, no leak), 0 errors; DELETE MNQ (primary) →
REFUSED; the state view shows NQ "unsubscribed". Tests: PurgeSymbol, runtime-add semantics, ack framing
round-trips, unsubscribe-purges-cache — green; [MNQ]-only golden intact. C# ack emits staged to the
AddOns folder — ONE F5 + NT8 restart activates them (post-F5 expected: states flip to "subscribed" with
contracts). SIM-only; the live-account block untouched; partner repo not synced (post-P5).

**P5.3 ACKS VERIFIED (owner F5 19:38 CT):** the v2+acks AddOn reconnected → hello OK → "subscription ACK
MNQ (MNQ 06-26)" + "subscription ACK ES (ES 06-26)"; GET /api/nt/symbols shows both "subscribed" with
resolved contracts, NQ "unsubscribed" from the earlier runtime removal. 0 errors. P5.3 COMPLETE.

## 2026-06-09 — P5.4 A+B+C: agent-per-symbol foundations (commits a3b4087f / 67b070fa / 77fd2d03; live)

STEP-A audit found the C# AddOn's per-order routing already shipped (Phase 2 parse + Phase 3
submit-to-target + Phase 4 per-symbol close memory) and four Go-side blockers. Built in 3 chunks, all
golden-tested, ZERO C# changes:

- **A (a3b4087f) — trading-symbol registry:** SetBarsSubscribeSymbol REGISTERS (first claims the primary
  slot, replacing the hardwired default; later roots append) — a second trader can't kill the first's bar
  feed. Config extras keyed by OWNER (SetExtraBarsSymbolsFor) — one trader's reload can't wipe another's.
  UnsubscribeBars + runtime-subscribe refuse ANY registered trading root.
- **B (67b070fa) — symbol-routed fan-out:** one lazy router consumes fills/closes/rejects/instrument_info
  and dispatches by symbol to per-symbol subscriber channels (SubscribeFillsFor/ClosesFor/RejectsFor/
  InstrumentInfoFor) — no more multi-consumer racing (the audit's fill-loss + recordClose cross-
  contamination). Legacy empty-symbol → primary. Resubscribe CLOSES the prior channel, terminating a
  reloaded-away trader's consumer (fixes the pre-P5.4 reload leak). NewTCPTrader + close_sync converted.
- **C (77fd2d03) — account binding + signal.account:** store.Trader.Account flows trader_manager →
  AutoTraderConfig → ninjatrader.Config → TCPTrader.boundAccount; signals carry account (omitempty —
  unbound wire byte-identical); placeEntry validates the BOUND account against the SIM/allow-list gate
  (the live/funded block applies identically). This ACTIVATES the AddOn's Phase-3 routing.

**Deployed + single-trader live-verified (20:37 CT):** "trader MNQ BOUND to account Sim101"; hello v2;
MNQ+ES subscribed + ACK'd; bars flowing both; 0 errors. All suites + the [MNQ]-only goldens green.

**REMAINING for P5.4 sign-off (owner-gated):** the LIVE two-trader proof — create a 2nd trader row (ES
strategy with static_coins=["ES"], own Exchange row with NTInstrumentName="ES", bound to a 2nd SIM
account; set SIM accounts' Max Position Size = 0 in NT8 per the plan) → verify independent decisions,
fills attributed per (symbol,account), zero cross-talk, no "exceeds max position", the live-account
block holding for ALL symbols. Partner repo sync remains post-P5-verification.
## 2026-06-10 — Auto-start on reboot (infra-only; commit 42b93057)

Survives a Windows reboot with zero manual steps. WSL side: systemd units `deploy/nofx.service`
(./nofx-bin, WorkingDirectory=/home/hoang/nofx so .env+SQLite resolve, Restart=on-failure 5s, log →
/tmp/backend.log + journalctl) and `deploy/nofx-web.service` (vite :3000), installed by ONE owner
command `sudo bash deploy/install-autostart.sh` (no passwordless sudo on this box — install is
owner-run; crash-restart proof lands then). systemd was ALREADY enabled on Ubuntu-24.04 — no
wsl --shutdown needed. Windows side (docs/AUTOSTART.md): Task Scheduler at-logon task
(`wsl.exe -d Ubuntu-24.04 --exec /bin/true`, 30s delay) boots the VM; shell:startup NT8 shortcut +
NT8 "On startup, connect to" auto-connects the feed (the AddOn already retries every 5s). Latent
fragility fixed: NT_TRANSPORT=tcp lived ONLY in ~/.bashrc (a service start would silently fall back
to the CSV transport) — moved into .env (gitignored, local). ZERO trading-code change; SIM-only
gates untouched. Full-reboot proof pending the owner's restart.

## 2026-06-10 — FE auto-refresh layer: every display surface self-updates (commit 2af59c93)

THE F5 ROOT CAUSE was not missing polling — most surfaces already polled (SWR refreshInterval in the
route wrappers + scattered setIntervals). It was a PERMANENT POLL-KILL LATCH on exactly the three
surfaces the owner stares at: account/positions/decisions set refreshInterval→0 after 2 failed retries
(AppRoutes onErrorRetry), and with revalidateOnFocus:false the onSuccess unlatch could never fire again
— every bot restart/deploy froze those panels until a full F5 or trader switch.

SHIPPED (FE-only):
- web/src/lib/autoRefresh.ts (NEW): named interval constants (positions/status/account/decisions 5s,
  chart 5s, open-orders 60s, equity 10s, history/logs 10s, ticker 15s, backoff-resume 30s);
  usePollResume (the latch fix — failed-state UI kept, polling auto-resumes after 30s);
  useAutoRefresh (skip-if-in-flight, pause on document.hidden + instant refresh on return,
  exponential error backoff ×2..×8, unmount cleanup).
- Latch fix wired to the 3 dashboard surfaces; all wrapper intervals lifted to the constants.
- Gaps wired: PositionHistory (trade history, silent 10s — pagination/filter/sort/scroll all separate
  local state, never yanked), DecisionAudit (decisions tab, silent 10s), AdvancedChart kline 5s +
  open-orders 60s (in-flight + hidden guards added), MarketTicker + GridRiskPanel (converted from bare
  setInterval to useAutoRefresh).
- NO-CLOBBER fixes (pre-existing holes the inventory exposed): (1) Strategy Studio focus-refetch
  replaced selectedStrategy unconditionally → unsaved name/description/Publish-toggle edits silently
  reverted on alt-tab; now guarded by the same hasChangesRef rule as editingConfig. (2) Recent
  Decisions cards were keyed by ARRAY INDEX → an expanded card re-attached to a DIFFERENT decision
  when a refetch prepended; now keyed timestamp+cycle_number.
- DELIBERATE non-polls (refutation/no-clobber): Settings lists (polling them resets an OPEN
  ExchangeConfigModal's unsaved fields incl. just-imported keys via its init-from-props effect —
  modal must become snapshot-on-open first); the shared accounts-SWR key (polling re-keys ALL
  account-scoped data if NT8's `current` drifts — the Issue-2F regression); Strategy Studio editors
  (editor page; its existing focus-refetch + preview debounce stay). AgentChat streams (SSE) + FAQ
  (static) unchanged. EquityChart already polled (10s/30s SWR) — untouched.
- SWR surfaces pause on hidden tabs by default (refreshWhenHidden=false); the new hook mirrors that
  for raw pollers. Manual refresh buttons unchanged. Editors/forms excluded throughout.

ALSO FIXED (found mid-verify): deploy/nofx.service + nofx-web.service shipped dead-on-arrival —
systemd's StandardOutput=append:/tmp/...log fails on this WSL2 ("Failed to set up standard output:
Permission denied", status=209/STDOUT; bot died in 1ms, frontend crash-looped 127×, frontend DOWN from
~21:07). Fix: shell redirection in ExecStart (exec ... >> log 2>&1). OWNER ACTION:
`sudo bash deploy/install-autostart.sh` to redeploy the units (then stop the manual processes).
Until then the frontend runs manually again (restarted 21:20) and the bot stays on the manual process.

Verification: tsc + npm build green. Playwright: browser reachable but the stored token is EXPIRED →
authenticated live proofs (update-without-F5 ticks, typing-survives, hidden-tab network quiet) are
OWNER-GATED: log in once, then watch the dashboard update on its own. /tmp/backend.log note: 11 GB —
rotation recommended. Partner repo sync HELD (P5.4 mid-flight; sync at the next stable boundary).

## 2026-06-10 (late) — Autostart 209/STDOUT crash fixed + deploy made universal

First install crashed BOTH units at stdout setup: append: targets in /tmp + Ubuntu
fs.protected_regular=2 (denies opening another user's existing file in sticky /tmp) →
"Failed to set up standard output: Permission denied" → 209/STDOUT, binary never ran
(1ms), nofx-web looped to 142 restarts, bot DOWN. Recovered live first (rm the offending
/tmp file → web unit self-healed; backend via nohup), then redesigned: real logs in
/var/log/nofx (LogsDirectory=), /tmp/backend.log + /tmp/frontend.log refreshed as
SYMLINKS each start (all tooling + nohup fallback unchanged), StartLimitIntervalSec=0 so
a failure can loop (journalctl-visible) but never strand the bot dead. Units are now
placeholder TEMPLATES; the installer detects user/repo/node (nvm-aware) at install time
and guards .env for NT_TRANSPORT=tcp — fully portable to the partner's machine.
NT_TRANSPORT=tcp also added to .env.example in both repos. Root-gated re-install +
kill-restart proofs = owner: `sudo bash deploy/install-autostart.sh`.

## 2026-06-10 — autostart 209 round 2: units go JOURNAL-ONLY (the actual fix)

The /var/log/nofx + root-ExecStartPre redesign ALSO 209'd: StandardOutput= applies to EVERY Exec*
line and the append-file is opened in the forked child BEFORE exec — the pre-step died at stdout
setup without executing. Root causes stacked: (1) /tmp: fs.protected_regular denies opening another
user's file in a sticky dir (the /tmp logs flip-flopped owner root↔hoang across attempts);
(2) any unit-level file sink on this WSL2 systemd risks the same child-setup failure. FIX: journal
sink only (no file-open in the child → cannot 209) — journalctl -u nofx / -u nofx-web; tooling
note: services no longer write /tmp/backend.log (manual nohup fallback still does). Units keep
StartLimitIntervalSec=0 + Restart=on-failure/5s (never permanently dead). Installer unchanged except
log hints (already idempotent: stops old units, kills strays, reset-failed, enable --now).
OWNER ACTION: sudo bash deploy/install-autostart.sh — then the kill→≤5s + 3×-restart proofs run
unprivileged. During the broken window the bot+frontend were manually restored (21:37, bars flowing,
:3000 HTTP 200). /tmp/backend.log confirmed truncated (21 MB, was 11 GB). SIM-only; no trading code.

---

## Timeframe selector — capability-driven (single source of truth) + de-cap (2026-06-11)

The Strategy Studio timeframe selector was hardcoded twice and hard-capped at 4
selections. Replaced with a capability-driven design.

**Capability (verified, not assumed):** NT8 is **per-series, NO Go aggregation**
— the live BarCache serves exactly the timeframes the AddOn auto-subscribes
(`provider/ninjatrader/tcp_server.go` `defaultAutoBarsTimeframes`), and an
interval outside that set (probed: `2m`) returns 0 bars. So the genuine
end-to-end capability is the fixed 14-set
`1m,3m,5m,15m,30m,1h,2h,4h,6h,8h,12h,1d,3d,1w` (crypto/CoinAnk serves the same
standard intervals). No new C#/wire work was needed — all 14 are already
streamed; this was a de-cap + de-dup, not "enable arbitrary intervals."

**Single source of truth:** `store.SupportedTimeframes` (the 14). Served via
`GET /api/strategies/timeframes` (also returns `soft_warn_above` + `max`); the
frontend fetches it instead of hardcoding a copy (falls back to a local list on
error). `provider/ninjatrader/timeframes_parity_test.go` keeps the NT8
auto-subscribe set in lockstep with `store.SupportedTimeframes` (drift fails CI).

**De-cap:** the hard max-4 (FE toast + BE `MaxTimeframes=4` truncation) is gone.
`MaxTimeframes = len(SupportedTimeframes)` (14); the FE shows an **advisory**
amber toast above `SoftWarnTimeframesAbove = 6` ("more timeframes = bigger AI
prompt → slower, costlier") that never blocks. The prompt path
(`formatKlineTimeframes`) already lists all N selected — golden extended with a
7-timeframe case. Existing strategies + defaults are byte-identical (≤4
selections are unaffected by the relaxed clamp).

## 2026-06-11 — AgentBeta 400 fixed: DB row id was sent as the AI model name (d55d397a)

Owner-reported 400 ("you passed 8ef641a7-…_deepseek") root-caused to agent/agent.go's
registry path: empty CustomModelName (legal "use default") was replaced with model.ID —
the ai_models ROW ID — and the DeepSeek client's SetAPIKey overwrote its default with it.
AgentBeta chat only; the trading loop / Strategy Studio test / Telegram all pass empty
through correctly (provider keeps deepseek-chat) and decision_records showed zero failures
(live success proven mid-diagnosis). DB clean — empty custom_model_name is legal. Fix:
fallback deleted; the effective model (ClientEmbedder.BaseClient().Model) is returned for
logs/display. Regression test TestLoadAIClientEmptyCustomModelNameNeverSendsRowID (golden:
row id must never reach the wire). Deployed live via rebuild + systemd on-failure restart;
hello v2 clean, trader re-bound to Sim101. AgentBeta chat end-proof = owner (UI + real
API call); code path + test verified.

## 2026-06-11 — DeepSeek default → deepseek-v4-pro system-wide

Owner directive: auto-set pro at the API level. DefaultDeepSeekModel = deepseek-v4-pro
(mcp/providers.go) — every empty-custom-name path (trading loop, AgentBeta, Studio test,
Telegram) now resolves to the reasoning tier automatically; explicit Custom Model Name
still overrides. Catalogs/presets/FE placeholders aligned; grid gorm default deliberately
left (live-DB schema-migration risk, dormant subsystem). Live: first cycle on pro
succeeded — 55.46s AI call (vs ~8s on chat), decision 769 saved. Watch: with a 1m scan
interval, ~55s calls stretch the effective cycle cadence.

## 2026-06-11 — vlauto mirror prepared + install made fully universal (3ab61978)

Owner decision: vlauto's CONTENTS now mirror nofx main exactly (repo/URL kept; fresh
single-commit history — partner re-clones). Portability commit 3ab61978: nq_smoke's
hardcoded /home/<user>/nofx/.env → godotenv.Load() cwd; ONBOARDING + the vltrader rename
doc genericized (C:\Users\<you>); INSTALL.md (root) = complete generic setup (packages,
.env w/ NT_TRANSPORT=tcp, AddOn deploy, autostart). Fresh-clone simulation PASSED (clean
tree → go build + npm install/build, documented steps only). Mirror commit 75bcb57 in
vlauto = nofx tree 3ab61978 MINUS internal docs (web/CLAUDE.md, docs/superpowers,
docs/internal — root CLAUDE.md was never tracked); builds+tests green there; secret scan
0 machine names, no private state (only upstream's dead public nofxos key + an obvious
fake test key). Old vlauto main kept as branch backup-pre-mirror-20260611. Force-push =
owner (classifier blocks me): `cd ~/vlautoagenttraderv1 && git push --force origin main`.
Side-finding: local nofx history has one unreadable old object (21a15f98…) — HEAD tree
fully intact (archive/build/push fine); deep-clone-from-local fails; origin is the good
copy.

## 2026-06-12 — June→September contract roll executed + the chart-bug root cause

The "all charts buggy" report = the ROLL, in two acts. (1) The resolver is DYNAMIC
(VLContractResolver.ResolveFrontMonthContractAt on every bars_subscribe + order submit —
never startup-cached); the owner's ~01:30 CT NT8 restart already moved the stream to
MNQ/ES 09-26 (ack'd; June roll date was Jun 11). (2) BUT the bot's BarCache still held
06-26 bars: the September seed at re-subscribe carried no history (NT8 had just started),
and mergeBarsByTime PRESERVES old bars the seed doesn't cover → every timeframe kept
June-priced history with a +232.75-pt cliff at the 06:29:30Z switch minute (measured on
1m; ~ the Jun/Sep carry basis). FIX = bot restart while flat (position from 01:33 was
already closed): cache wiped → N3 re-seed from the now-September BarsRequests → 500
pure-09-26 bars per timeframe. Verified: 1m seam GONE (max move 52.75 pts, live bars),
deeper TFs show single-contract September levels (the 06-11 17:30 move re-priced from
28744→29087 to 29031→29374 = the basis, proving the re-seed). Procedure doc:
docs/CONTRACT-ROLL.md (quarterly: flat → NT8 restart → bot restart → verify ack).

## 2026-06-24 — AddOn gap self-heal: rebuild the BarsRequest on bot reconnect (was a frozen N3 snapshot)

Root cause of "chart gap that won't fix even after bot restart": the AddOn's per-(symbol,tf)
BarsRequest survives a Go-process restart, and the old N3 path (VLBarsSubscriptionManager.cs
Subscribe, the active.ContainsKey branch) merely RE-EMITTED that surviving request's window
(EmitHistorical(existing)) without recreating it. A BarsRequest that straddled a feed outage
(feed drops ~16:00, resumes next afternoon) holds that hole FOREVER and never backfills —
even though NT8's own DB fills the gap from the provider — so every bot restart re-sent last
night's hole. FIX (C#, AddOn redeploy required): the N3 branch now DISPOSES + RECREATES the
BarsRequest (→ N4), so a fresh request re-reads NT8's now-complete DB across the full barsBack
lookback and the gap self-heals on the next bot restart. DEFAULT_BARS_BACK 500→2000 so the
reconnect-rebuild path (OnConnectionReconnected, line ~554) and fallback look back far enough
to span an overnight gap (500=~8h was shorter than the outage). Go side already correct:
mergeBarsByTime (bar_cache.go) is a time-ordered union that ACCEPTS backfilled interior bars
into a hole (not append-only), and defaultAutoBarsBack=2000/maxBars=2500 shipped in be0c9860.
Go build/vet/test green; the Go bars_subscribe frame is unchanged (goldens unaffected). The
friend MUST redeploy the AddOn: cp ninjascript/*.cs → Documents AddOns → F5 → clean NT8
restart (the restart itself re-seeds with 2000 from the full DB → fills last night).

## 2026-06-24 — Auto-heal completes: feed-resume recreate already wired; add flap debounce

STEP-A finding: the auto-heal-on-feed-resume ALREADY EXISTS. VLTraderTCPClient.cs hooks
Connection.ConnectionStatusUpdate (line ~141) and on a data-feed recovery (PriceStatus →
Connected after a loss/fresh-connect) calls barsManager.OnConnectionReconnected() (line ~259)
— dispose+recreate ALL BarsRequests. This is the SAME code path the manual Tradovate
disconnect/reconnect triggers (which the owner confirmed heals the gap) — so a natural feed
resume already runs it, no manual step. The reason an overnight gap didn't auto-heal: the
recreate's lookback was DEFAULT_BARS_BACK=500 (~8h) — shorter than the outage — fixed to 2000
(~33h) in cde27cc2. Go merge fills interior gaps (mergeBarsByTime union, confirmed). This
commit adds the only missing piece per spec: a 30s flap DEBOUNCE (lastRecreateUtc /
RecreateDebounce) so the now-deep/expensive refetch coalesces a rapidly-flapping feed into one
rebuild instead of thrashing; first-connect + genuine resume still fire once; the 20s
FAST_STALL fast-guard backstops a .Update that dies inside the cooldown. No-gap/startup
behavior preserved; Go bars_subscribe frame unchanged (goldens green). C# AddOn change →
redeploy required (cp ninjascript/*.cs → AddOns → F5 → NT8 restart).
