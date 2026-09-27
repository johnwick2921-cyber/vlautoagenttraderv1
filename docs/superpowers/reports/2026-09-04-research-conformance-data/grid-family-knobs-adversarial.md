# Adversarial re-verification — "Grid-family KnobLive knobs (15)"

Code read at worktree HEAD c28fd337; `git diff --stat 70af663d HEAD -- trader/ kernel/ store/ api/` is EMPTY,
so every file:line below is the DEPLOYED code (rev 70af663d, PID 878451, boot 2026-09-04 08:30:11 CT). [A]

## Corrected row

| field | peer | corrected |
|---|---|---|
| rule | Grid-family KnobLive knobs (15) — live only when a grid strategy runs | same headline, membership wrong (see below) |
| file:line | kernel/grid_engine.go:153,**283**; gate auto_trader.go:940-941 + auto_trader_clock.go:878-879 | gate lines EXACT; **:283 is a mis-citation** — real knob read is trader/auto_trader_grid.go:315-316 |
| resolved NOW | strategy a5b7662e strategy_type='ai_trading' | reproduced + 3 more terms: grid_config ABSENT; 9/9 strategies ai_trading; grid_instances/levels/events/regime_assessments/configs = 0 rows |
| label | [M], "none found" | **[R]** 2026-08-19-strategy-controls-census.md:47 (+[O] 2026-06-02-…-futures-map.md:199,302 = PARK) |
| effect | gate (never reached) | correct |
| CONFORMS | unknown | **yes** — research "LIVE code path, zero live users" == resolved "0 grid strategies / 0 grid rows" |
| prod callers | 0 — DEAD | **>=9 non-test call sites**; state = UNREACHED-BY-CONFIG, not dead-by-no-caller |

## Evidence

- gate: `isGridStrategy := at.IsGridStrategy()` trader/auto_trader.go:940 → `at.tickOnce(isGridStrategy)` :952/:970/:982
  → `if isGrid { at.RunGridCycle() }` trader/auto_trader_clock.go:879-883. [A]
- predicate is a TWO-term AND: `StrategyType == "grid_trading" && GridConfig != nil` trader/auto_trader_grid.go:567. Both fail. [A]
- DB (ro): running trader `8d5c8af5_…_deepseek_1781246265` is_running=1 account=Sim101 strategy_id=a5b7662e-7bf7-49bb-9f09-7efa48f95ac8;
  `json_extract(config,'$.strategy_type')`='ai_trading', `$.grid_config` NULL, for ALL 9 rows. [A]
- non-test callers of the 15 knobs: auto_trader_grid.go:187,304,315-316; grid_levels.go:35,103; grid_orders.go:127,362;
  grid_regime.go:65; grid_engine.go:153. [A]
- second live route into grid code with NO strategy_type gate:
  api/server.go:234-236 `GET /traders/:id/grid-risk` → api/handler_trader_status.go:36 → trader/auto_trader_grid_regime.go:227.
  Early-returns at :229-231 on GridConfig==nil, so no knob is read — but :228 derefs `at.config.StrategyConfig` with NO nil guard
  (IsGridStrategy at :564-566 HAS one). Latent nil-deref. [A]
- 16th grid-named KnobLive knob `grid_config` (store/knob_registry_table.go:73) has consumer trader/auto_trader_decision.go:108,
  inside `GetStatus()` (:67) — called unconditionally from manager/trader_manager.go:136,258,365 and api/handler_trader.go:711,780,821,918.
  That read EXECUTES on the live MNQ trader. [A]

## Registry citation defects (all three verified)

| knob | registry says | actually reads | real knob read |
|---|---|---|---|
| lower_price / upper_price (table:93,161) | kernel/grid_engine.go:283 | `ctx.LowerPrice/UpperPrice` (GridContext, grid-STATE derived; never set from config in BuildGridContextFromMarketData:560-576) | trader/auto_trader_grid.go:315-316 |
| config (table:28) | trader/auto_trader_grid_regime.go:157 | `at.gridState.Config.Symbol` | no `json:"config"` leaf exists under StrategyConfig — phantom entry |
| max_drawdown_pct (table:98) | kernel/formatter.go:162 | `stats.MaxDrawdownPct` (TradingStats, LIVE ai_trading prompt path) | trader/auto_trader_grid.go:180 |

Consequence: the peer's family ("KnobLive whose consumer file matches /grid/") includes the phantom `config` and EXCLUDES the
genuine grid leaf `max_drawdown_pct` (store/strategy.go:1542). 15 is the right count by coincidence, wrong by membership.
The registry itself documents this failure mode at store/knob_registry.go:110-114 ("a leaf name can collide across structs,
and where it does the first classification wins").

## Collision hypothesis that FAILED (reported for honesty)

I tested whether the 15 grid leaf names collide with live AI-path leaves inside the walked schema
(`EnumerateSchemaKnobs`, store/knob_registry.go:62-102, walks `StrategyConfig{}` and SKIPS `json:"-"` at :79-80).
RiskControlConfig / IndicatorConfig / CoinSourceConfig / DayPlanConfig / RegimeConfig share ZERO leaf names with
GridStrategyConfig (store/strategy.go:1522-1553) — RiskControl uses `daily_loss_limit_usd`, not `_pct`. Partition is clean. [A]

## git log -1 for every report cited

- docs/superpowers/reports/2026-08-19-strategy-controls-census.md → `741bfc2a 2026-09-01 07:58:16 -0500 docs: archive 38 stranded research reports to dev + RESEARCH INDEX (docs-only — no code, no content edits; collisions suffixed, originals left in place)`
- docs/superpowers/reports/2026-06-02-strategy-studio-universal-futures-map.md → `72907b9b 2026-06-02 22:31:48 -0500 docs: strategy-studio universal futures change-map (read-only analysis)`
- docs/superpowers/reports/2026-08-30-knob-census.md → `741bfc2a 2026-09-01 07:58:16 -0500 docs: archive 38 stranded research reports…`
- docs/superpowers/reports/2026-09-02-belief-census.md → `ee64a494 2026-09-02 08:50:38 -0500 docs: belief census 2026-09-02 …`

Verbatim grounding, 2026-08-19-strategy-controls-census.md:47:
> | GridConfigEditor + GridRiskPanel | 15 + 1 | All keys consumed inside the grid cycle files — **LIVE code path, zero live users** (no `grid_trading` strategy exists in the DB). |

Same report :42 cites the gate as `auto_trader.go:803-815` — STALE; it is now auto_trader.go:940-941 + auto_trader_clock.go:878-884.

Stronger-than-stated mechanism (2026-06-02 map:124,197): NT8 has no `PlaceLimitOrder`
(implemented only by aster/okx/lighter/hyperliquid/bitget/binance/bybit). But the report's "cannot execute" is
OVERSTATED: trader/auto_trader_grid_orders.go:60-64 falls back to `NewGridTraderAdapter`, which synthesizes limit
orders from SetStopLoss/SetTakeProfit (trader/interface.go:33-63). Interface dispatch, invisible to a `func(` grep. [A]
Whether that fallback works on NT8 is UNTESTED — n=0, all grid tables empty. [C]
