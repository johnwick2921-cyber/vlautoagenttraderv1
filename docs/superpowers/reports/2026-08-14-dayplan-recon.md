# Day-Plan Design — Pre-Build Integration Recon (READ-ONLY)

**Date:** 2026-08-14 · **Repo:** /home/hoang/nofx · **HEAD:** `3624a2a4` (branch `main`)
**Method:** 12 parallel read-only investigators, one per anchor; every claim carries a fresh `file:line` read at HEAD. Evidence tiers: **[A]** saw the exact line · **[B]** inferred from strong evidence · **[C]** speculation.
**Scope:** verify the day-plan design's 12 integration assumptions against real code. No code changes; this report file is the only write.

## VERDICT: 9 surprises (0 hard blockers)

3 anchors hold exactly (1, 8, 9); 8 exist-but-differ (2, 3, 4, 5, 6, 7, 10, 11); 1 is missing (12). None blocks the design — the differences are relocations, additive persistence, and one absent capability whose primitives already exist. The two footguns to internalize before writing code: **(a) StrategyConfig has hand-rolled JSON codecs — a new field is silently dropped unless added to both**, and **(b) the naked-POC tracker needs a NEW durable store because the SVP engine is stateless and the 1m bar cache holds only ~1 prior session and is wiped on restart.**

## Integration map

| # | Anchor | Verdict | Tier | Anchor file:line | Build |
|---|--------|---------|:----:|------------------|:-----:|
| 1 | Strategy config row + hot-read | **EXISTS-AS-ASSUMED** | A | `store/strategy.go:674,682-712,731-823`; `manager/trader_manager.go:626,655`; `api/strategy.go:407-425` | S |
| 2 | SVP engine + session-history | **EXISTS-DIFFERENT** | A | `kernel/svp.go:304,106,78-96,8-11`; `store/strategy.go:915` | M |
| 3 | Bar cache TFs/depth/session-tag | **EXISTS-DIFFERENT** | A | `provider/ninjatrader/bar_cache.go:24,30-34`; `tcp_server.go:384,392`; `tcp_framing.go:353`; `market/data_freshness.go:149` | M |
| 4 | Prompt injection (KEY LEVELS + cached prefix) | **EXISTS-DIFFERENT** | A | `kernel/engine_prompt_futures.go:152-156`; `engine.go:210-220`; `engine_analysis.go:305-331,341`; `mcp/interface.go:17` | S (M if caching) |
| 5 | Executor scan timer + skip-while-open | **EXISTS-DIFFERENT** | A | `trader/auto_trader.go:715,739-758`; `auto_trader_orders.go:329,470`; `auto_trader_risk.go:279-291` | M |
| 6 | 17:00 CT rollover + holidays | **EXISTS-DIFFERENT** | A | `kernel/cme_calendar.go:100-111,17-35,158-227` | M |
| 7 | Chart SVP primitive (LevelOverlay copy) | **EXISTS-DIFFERENT** | A | `web/src/components/charts/primitives/SessionVolumeProfile.ts:411,202-214,134-143`; `AdvancedChart.tsx:1295-1315` | S |
| 8 | B6 gate-block telemetry hook | **EXISTS-AS-ASSUMED** | A | `telemetry/gate_blocks.go:40-47,55-69`; `api/handler_gate_blocks.go:27-41` | S |
| 9 | SQLite additive-migration pattern | **EXISTS-AS-ASSUMED** | A | `store/store.go:118-168`; `store/order.go:139`; `store/strategy.go:1085-1086` | S |
| 10 | Close path + hold-lock exemption | **EXISTS-DIFFERENT** | A | `trader/ninjatrader/tcp_trader.go:397-433`; `auto_trader_orders.go:72-100`; `auto_trader_risk.go:139-161` | S |
| 11 | /api/risk/* handler pattern | **EXISTS-DIFFERENT** | A | `api/handler_risk.go:78-95`; `api/server.go:400-422`; `route_registry.go:28-45` | S |
| 12 | Per-strategy planner-model | **MISSING** | A | `store/trader.go:25`; `store/strategy.go:682-712`; `trader/auto_trader.go:389-433`; `store/ai_model.go:80-236` | M |

---

## Detailed findings

### 1 — Strategy config row + hot-read path — EXISTS-AS-ASSUMED [A] · build S
- Row lives in table `strategies`, one JSON column `Config` (`store/strategy.go:674`, TableName `:679`). Parsed type `StrategyConfig` `:682-712`; top-level keys `strategy_type/language/prompt_variant/grid_config/publish_config`; the 5 AI blocks carry `json:"-"` (`:699-703`) and are nested under `ai_config` by hand.
- **GOTCHA:** `StrategyConfig` has custom `MarshalJSON` (`:731-764`) / `UnmarshalJSON` (`:768-823`). A plain new field is **silently dropped** unless added to BOTH the anonymous `out` struct AND `rawStrategyConfig`. `PublishConfig *PublishStrategyConfig json:"publish_config,omitempty"` (`:711`, marshaled `:743/748`, unmarshaled `:775/793`) is the exact template for a new top-level nested block.
- Hot-read is a cached pointer set ONCE at trader-load: `ParseConfig` (`manager/trader_manager.go:626`) → `StrategyConfig` field (`trader/auto_trader.go:314`), engine built once (`auto_trader.go:572`). Live loop reads `at.config.StrategyConfig.<block>` every cycle (`auto_trader_risk.go:218/267/284`, `auto_trader_orders.go:51/79/111`) — never re-reads the DB (`store/strategy.go:1410-1418`).
- A bare DB write is **not** hot; `handleUpdateStrategy` does `RemoveTrader`+`LoadUserTradersFromStore` on any save (`api/strategy.go:407-425`), else a restart is needed. `MergeStrategyConfig` (`:484-511`) deep-merges a partial `{"day_plan":{...}}` patch cleanly.
- **Design:** attach `DayPlan *DayPlanConfig json:"day_plan,omitempty"` at the **root** of StrategyConfig (not under `ai_config`, which a grid-type switch destroys, `PreserveAIConfigOnTypeSwitch:520-530`), following the PublishConfig template + a round-trip golden. Build S = 1 type + 1 field + 2 codec edits + 1 test + read site.

### 2 — SVP engine + session-history — EXISTS-DIFFERENT [A] · build M
- Engine: `BuildSVPProfile(bars, now) SVPProfile` (`kernel/svp.go:304`), multi-session; every non-live session `Frozen=true` (`:341`). POC/VAH/VAL are `float64` fields on `SVPSession` (`:78-87`), not a dedicated struct. `FormatSVPLine` renders **only the developing session** (`:106,110-119`) — the AI never sees a prior POC today.
- **STATELESS** (`:8-11`): each call recomputes and discards. Only persisted SVP artifact = `EnableSVP bool` (`store/strategy.go:915`). Repo-wide grep: **no** SVP table/migration/cache. Chart path recomputes per HTTP request (`api/handler_svp.go:56,62`); kernel path per cycle (`engine_analysis.go:312,320-324`).
- Backfill depth from the live cache: 1m ring ≈ 2000–2500 bars ≈ **~33–41h ≈ the developing session + ~1 prior** (that prior lands `Partial=true`); volatile in-RAM, **wiped on restart** (`bar_cache.go:24`). Coarse TFs reach deeper but smear volume-at-price and are still lost on restart.
- **Design:** a naked-POC tracker needs a **NEW durable layer** — a table keyed `(symbol, session_date)` storing `{sessionStart, POC, VAH, VAL, partial, filled, filled_at}` + a **session-roll snapshot writer** at the 17:00 CT boundary (mandatory: you cannot reconstruct old POCs from bars later) + backfill-on-boot from residual coarse-TF depth + a naked→filled checker. Math is done; persistence is the M.

### 3 — Bar cache TFs/depth/session-tagging — EXISTS-DIFFERENT [A] · build M (tagging slice S–M)
- `BarCache` = `map["SYMBOL|TIMEFRAME"][]Bar` (`bar_cache.go:30-34,241`), one ring per (symbol,tf), no session dimension. **14 TFs** auto-subscribed 1m→1w (`tcp_server.go:384`), `bars_back=2000` (`:392`), per-ring cap `2500` (`bar_cache.go:24`).
- Depth: 2500 bars ≈ **~41h on 1m, ~8.7d on 5m, ~26d on 15m, ~104d on 1h** — build day-spanning levels off 5m/15m, not 1m.
- `Bar{T,O,H,L,C,V}` has **no** session/RTH/ETH field (`tcp_framing.go:353-360`); `Get()` offers no session filter. **BUT** every bar keeps its UTC-ms `T` → `Kline.OpenTime` (`bars_market_bridge.go:41`), and authorities exist: `IsRTH` (US 08:30–15:00 CT, `market/data_freshness.go:149`), `CMESessionDayStart` (`kernel/cme_calendar.go:100`), and `svp.go` is a working timestamp→session bucket template.
- **MISSING:** any Asia/Tokyo/London sub-session concept (grep `asia|tokyo|london|killzone|opening.range` = 0 hits). **Design:** add Asia/London window helpers next to `IsRTH` + an extractor that buckets `Get()` snapshots by `OpenTime`. Purely additive read-side; no cache write, no wire/schema/AddOn change.

### 4 — Prompt injection sites (KEY LEVELS + cached prefix) — EXISTS-DIFFERENT [A] · build S (M if real caching)
- Live futures assembler `buildFuturesPrompt` (`engine_prompt_futures.go:58`), ordered sections Role→…→SVP(`:147-156`)→Entry(`:159`)→Decision→Output. SVP is injected **gated** (`:152-156`, `if EnableSVP && svpContextLine != ""`) — the only per-cycle-mutable insertion; empty default writes nothing (golden-safe).
- Clone target = the `SetSVPContext` pattern: field `svpContextLine` + setter (`engine.go:215,220`), threaded per cycle (`engine_analysis.go:312,319-324`), assembled at `:341`. Futures routes via early-return (`engine_prompt.go:19-29`) — SVP/KEY-LEVELS/day-plan are **futures-only** unless added to the crypto body.
- **KEY LEVELS:** cleanest site `engine_prompt_futures.go:156-157` (after SVP, before Entry Standards) — add `keyLevelsLine` field + setter + gated inject; mirror-test template ready at `futures_prompt_boxes_test.go:58-90`.
- **Cached day-plan prefix:** everything before the SVP line (`:152`) is positionally stable → a prefix before Role(`:96`)/after Instrument(`:107`) is cacheable. **CRITICAL:** prompt caching is **not wired** — `mcp.AIClient.CallWithMessages` takes a plain-string `Message.Content` (`mcp/interface.go:17`, `mcp/request.go:52-88`), no `cache_control`/content-blocks. Positional stability is free; a real cache HIT needs M transport work, and SVP-mid-prompt truncates any cacheable region (move SVP to END if caching is pursued).
- **Goldens at risk:** `kernel/testdata/futures_mnq_empty.golden` (the only byte-identical LIVE-prompt snapshot; breaks on an UNCONDITIONAL block, **safe if gated**). The `golden/futures_system_*.txt` snapshot a LEGACY standalone builder — untouched unless you edit `BuildFuturesSystemPrompt`.

### 5 — Executor scan timer + skip-while-open — EXISTS-DIFFERENT [A] · build M
- **Timer is MISLOCATED by the design:** `time.NewTicker(at.config.ScanInterval)` is at `trader/auto_trader.go:715` inside `Run()` (`:739-758` select on `ticker.C` → `runCycle`), **not** in `auto_trader_loop.go` (which only holds the `runCycle` body, `:46`). Fixed wall-clock cadence; no bar-close trigger. Bar-close conversion = replace the `case <-ticker.C:` arm with a bar-close signal fed from BarCache ingest + a fallback timer for feed-gap safety → **M**.
- **No loop-level "any open → skip" gate.** The AI runs every tick; the skip is a downstream **same-symbol/same-side** refusal (`auto_trader_orders.go:329,470`) + `enforceMaxPositions` cap (default 3, `auto_trader_risk.go:279-291`) + NT8 orphan reconcile. An opposite-side open is not blocked here. An explicit "any open MNQ position → don't open" gate = S (one check at the top of `executeOpenLong/Short`).

### 6 — 17:00 CT rollover + holidays — EXISTS-DIFFERENT [A] · build M (early-close S)
- Rollover HOLDS: `CMESessionDayStart` (most-recent 17:00 CT, `cme_calendar.go:100-111`), `CMESessionDayKey` (`:116`), `IsCMEOpen` models the 16:00–17:00 daily break (`:17-35`).
- Holiday calendar EXISTS — **hardcoded + algorithmic in-code** (`isCMEHoliday:158-227`: fixed dates + floating MLK/Presidents/GoodFriday(Meeus)/Memorial/Labor/Thanksgiving), no library/feed, fully unit-tested (`cme_calendar_test.go`). Today 2026-08-14 (Fri) is a normal trading day.
- **No session-registry object** — stateless free functions used at ~6 sites (`risk_limits.go:159`, `telemetry/gate_blocks.go:21`, `svp.go:304`, `auto_trader_loop.go:439`); a registry would be an optional centralizing refactor (S), not net capability.
- **MISSING:** early-close/half-days — shortened sessions are deliberately treated as **full closures** (`:14-16,155-157`), so the bot refuses to trade genuine half-days (Jul 3, day-after-Thanksgiving, Dec 24/31). Add a half-day→close-hour map + `IsCMEOpen` override = S. Minor [B]: no weekend-observation shift (low impact, Sat/Sun already closed).

### 7 — Chart SVP primitive (LevelOverlay copy) — EXISTS-DIFFERENT [A] · build S
- `SessionVolumeProfile.ts:411` is a production lightweight-charts v5 `ISeriesPrimitive` attached to the candle series — the exact class LevelOverlay forks. Full triple: `IPrimitivePaneRenderer.draw` (`:116-123`), `IPrimitivePaneView` (`:274-282`), `ISeriesPrimitiveAxisView` (`:224`).
- **Directly copyable:** POC solid + VAH/VAL dashed **horizontal level lines** as `ctx.moveTo(anchorX)/lineTo(endX)` (`:202-214,185-200`); attach-once + `setData`-on-refresh loop against `/api/klines/svp` (`AdvancedChart.tsx:1295-1315`, `syncSVP`); Go↔TS JSON contract matched field-for-field (`kernel/svp.go:65-96`). The ±30min nearest-bar anchor snap (`:303`) handles non-exact 17:00 opens.
- **MISSING:** session **background shading** — only a vertical divider stroke exists (`:134-143`), no filled x-span rect. But `anchorX`/`endX` are already computed per session → shading is one `ctx.fillRect` in the existing loop. Build S. Endpoint pattern (`api/handler_svp.go`, empty-200 degradation, ninjatrader-only gate) is a solved contract to mirror.

### 8 — B6 gate-block telemetry hook — EXISTS-AS-ASSUMED [A] · build S
- `IncGateBlock(trader, gate)` is `+1`, free-form string, **no registry/enum/allowlist** (`telemetry/gate_blocks.go:40-47`); any new name auto-appears in `GateBlockSnapshot`, `by_trader` JSON, and the daily summary (`api/handler_gate_blocks.go:27-41`). 20 call sites today.
- **Caveats:** it is a **counter, not a rate** — a plan match-rate needs TWO counters (`plan_matched` + `plan_total`) + ratio at read. Table is **ephemeral**, resets at the 17:00 CT rollover (`RolloverGateBlocks:55-69`, called `auto_trader_loop.go:99`) — today-only. Names are framed as "gate BLOCKS", so a match counter surfaces in the same table (cosmetic).

### 9 — SQLite additive-migration — EXISTS-AS-ASSUMED [A] · build S
- GORM (`store/gorm.go:8`), **no** external migration framework. Single orchestration site `store.Store.initTables()` (`store/store.go:118-168`) calls each sub-store's `InitTables()`. Best template for plans+overlays: `store/order.go:139` (**two tables, one AutoMigrate**) + its registration line.
- New table = new `store/plan.go`/`overlay.go` (gorm model + `InitTables()`+`AutoMigrate`) + one line in `initTables()` + accessor. Add the Postgres table-exists guard (`position.go:158-179`). New column = gorm-tagged field (AutoMigrate adds it, comment `strategy.go:1085`). **Note:** `consecutive_loss_halt` is a JSON-blob field, **not** a real column — don't cite as a column precedent.

### 10 — Close path + hold-lock exemption — EXISTS-DIFFERENT [A] · build S
- Reusable session-flat primitive: `sendClose` → `SendClosePosition` (flatten at market + cancel bracket, `tcp_trader.go:408-433`); `CloseLong/Short` (`:397-403`); AutoTrader `executeClose*WithRecord` (`auto_trader_orders.go:592-653`).
- Hold-lock gate `holdLockSuppressesClose` (`:72-100`) discriminates **only** on action==close_* + open-position (via `GetOpenPositionBySymbol`) + `HoldDisciplineEnabled` (default OFF); **no origin/source tag** on `kernel.Decision` (`engine.go:131-152`).
- **Design (PREFERRED):** route session-flat **directly** through `at.trader.CloseLong/Short(symbol,0)` exactly like `emergencyClosePosition` (`auto_trader_risk.go:139-161`) and the drawdown monitor — this **inherently bypasses** hold-lock (never enters the loop), zero exemption code. Use the `*WithRecord` variants (or replicate `recordAndConfirmOrder`) so the `position_close` frame still journals PnL. On default configs (HoldDiscipline OFF) it's byte-identical to today.

### 11 — /api/risk/* handler pattern — EXISTS-DIFFERENT [A] · build S
- Pattern: `func (s *Server) handleXxx(c *gin.Context)` registered on the JWT `protected` group via `routeWithSchema(protected, METHOD, "/risk/...", desc, schema, handler)` (`server.go:400-422`); `routeWithSchema` (`route_registry.go:28-45`) wires gin AND feeds the LLM API-docs (bot-callable for free).
- **CORRECTION to the anchor:** risk handlers do **NOT** use `getTraderFromQuery`. They inline `traderID := strings.TrimSpace(c.Query("trader_id"))` → `SafeBadRequest` → `s.traderManager.GetTrader(traderID)` → `SafeNotFound` (`handler_risk.go:78-95`). `getTraderFromQuery` (`server.go:593`) is the order/debug family (defaults to the user's first trader) — a looser contract.
- **Design:** `/api/plan/*` mirrors the risk block (one `handler_plan_*.go` + register at ~`server.go:422`). **Note:** no owner-scoping in the risk family (GetTrader resolves any loaded trader by id) — if `/api/plan/*` must be owner-scoped, that guard is net-new.

### 12 — Per-strategy planner-model — MISSING [A] · build M (S if scored as just the selector)
- Trader→model is **1:1**: `Trader.AIModelID` (`store/trader.go:25`) → one `Provider` → one `mcpClient` (`auto_trader.go:389-427,584`); `StrategyConfig` has **no** model field (`strategy.go:682-712`); the live decision uses the single client (`auto_trader_loop.go:193`). No day-plan sub-task exists (`grep day.?plan` = 0 in the trading path; `agent/planner_runtime.go` is the NOFXi chat planner, separate).
- **Primitives EXIST:** the multi-model list (`store/ai_model.go:80-236`, multiple rows/provider, `GetByID`, `PickProviderModel`) + `NewAIClientByProvider` (`mcp/registry.go:14`). **Build:** add `planner_ai_model_id` (traders col or StrategyConfig JSON) + resolve it in `addTraderFromStore` + build a **second** `mcp.AIClient` (factor out `auto_trader.go:389-433`) + thread into the day-plan call + FE picker (list endpoint exists). Empty→fall back to primary (PromptVariant pattern). **M**; marginal cost of just the selector ≈ S.

---

## Top risks (ranked)

1. **StrategyConfig hand-rolled codecs (#1)** — HIGH-likelihood footgun. A `day_plan` field is **silently dropped** on save/load unless added to BOTH `MarshalJSON.out` and `UnmarshalJSON.rawStrategyConfig`. → Add a round-trip golden test *first*; keep `day_plan` at the root (survives grid-type switch).
2. **Naked-POC durable store (#2)** — the biggest genuine build. Engine is stateless; the 1m cache holds only ~1 prior session and is wiped on restart, so **old POCs are unrecoverable later**. A 17:00-CT session-roll snapshot writer is *mandatory*, not optional. (M, guarded DB writes.)
3. **Prompt-caching payoff is unreachable at HEAD (#4)** — transport is plain-string, no `cache_control`. Positional stability is free; a real cache hit is M transport work, and SVP-mid-prompt must move to the end. Don't scope the day-plan on an assumed cache saving.
4. **Timer relocation + bar-close cadence (#5)** — the ticker is in `auto_trader.go:715` (Run), not the loop file; bar-close needs an ingest tap + channel + Run() rewire + feed-gap fallback (M). A plan that edits `auto_trader_loop.go` for the timer aims at the wrong file.
5. **Planner-model selection missing (#12)** — must build a second binding + second client (primitives exist, M). Until then a day-plan sub-task reuses the trader's single decision model.
6. **Session-tagging / Asia-London windows missing (#3)** — no session tag on bars, no cross-tz session concept; additive read-side extractor (S–M), build day-spanning levels off 5m/15m.
7. **Early-close/half-days = full closures (#6)** — bot won't trade genuine half-days; add a half-day map (S).
8. **B6 is count-only + ephemeral (#8)** — match-rate needs two counters + a read-time ratio; resets daily (no cumulative history).
9. **/api/risk/* has no owner-scoping (#11)** — net-new if `/api/plan/*` must be per-user; and the family uses inline trader_id resolution, not `getTraderFromQuery`.

## Build-cost rollup
- **S (clone/additive):** 1, 4 (levels+prefix, sans caching), 7, 8, 9, 10, 11 — mostly copy-forks of shipped patterns.
- **M (new capability):** 2 (durable POC store + roll writer), 3 (session extractor), 5 (bar-close cadence), 6 (early-close), 12 (planner model). Optional M: real prompt caching (transport).
- **L:** none surfaced.

## Standing constraints carried into the build
SIM-only; `data/data.db` read-only unless a task authorizes a guarded write (back up + WHERE-scope + idempotent); the high-cascade `market.Kline` / `types.Trader` (19 methods) / Decision-JSON / TCP-wire schema must not break; any C# AddOn change needs the cp→F5→full-NT8-restart lockstep; prompt goldens stay byte-identical unless a change is deliberate (gate new prompt blocks like SVP). sqlite3 CLI absent in this env — the `strategies.config` column is Tier B (GORM tag), all else Tier A.
