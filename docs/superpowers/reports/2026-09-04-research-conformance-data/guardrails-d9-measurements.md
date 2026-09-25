# D9 — guardrails vs the Monte-Carlo rig (read-only measurements, 2026-09-04)

Source of RESOLVED values: running binary PID 878451 (`/home/hoang/nofx/nofx-bin`,
rev 70af663d), boot 8 at 09-04 08:30:11 CT; DB read `mode=ro`; log files
`/home/hoang/nofx/data/nofx_2026-08-16..2026-09-04.log` (20 files).

## M1 — bound strategy (never `LIMIT 1`)
`traders.strategy_id` = `a5b7662e-7bf7-49bb-9f09-7efa48f95ac8` ("MNQ"), trader
`hoang`, account `Sim101`, is_running=1. risk_control lives at
`config.ai_config.risk_control` (root `risk_control` is ABSENT — reading the root
key returns nothing, which is how a naive probe concludes "no risk config").

    guardrails_enabled       = false     daily_loss_limit_usd    = 450
    daily_loss_enabled       = false     daily_profit_target_usd = 900
    daily_profit_enabled     = false     max_daily_trades        = 3
    max_daily_trades_enabled = false     max_contracts_per_order = 2
    max_contracts_enabled    = false     notional_cap_enabled    = false
    blackout_enabled         = false     max_positions           = 3
    min_confidence           = 60        min_risk_reward_ratio   = 2
    btc_eth/altcoin_max_leverage = 5/5
    ABSENT: consecutive_loss_halt, consistency_*, blackout_start_ct/end_ct,
            max_notional_leverage
    day_plan.sessions max_trades = NY 10 / ASIA 7 / LONDON 10

## M2 — boot-8 resolved lines (read from the live log, never a file default)
    08:30:11 trader/auto_trader.go:43   🧾 ledger boot: … guardrails=master=OFF (soft-audit only) …
                                        … trailing=2.0×ATR14 arm=after_breakeven (source: studio) …
    08:30:11 nofx/main.go:335           🛑 exits: … BE=off · trail=off · size=1 · re-arm-after-sweep=on (0B)
    08:30:11 kernel/risk_limits.go:172  daily window reset to CME session-day 2026-09-03
    08:30:11..08:46:10 kernel/engine_analysis.go:173 — master-OFF WARN on 9 of 9 decision cycles

## M3 — enforcement census over 20 days of logs (2026-08-16 → 2026-09-04)
| line | count |
|---|---|
| `🔍 guardrail WOULD have tripped (master OFF, not enforced)` | **2793** |
|   of which `max daily trades would trip` | 2431 |
|   of which `daily loss would trip` | 362 |
| `⚠️ Strategy Studio daily guardrail tripped` (ENFORCING) | **0** |
| `⚠️ concurrent-position gate tripped` | **0** |
| `❌ [RISK CONTROL] Already at max positions` | **0** |
| `⚠️ [RISK CONTROL] … exceeds max … clamping` | **0** |
| `⚠️ [RISK CONTROL] Position … exceeds limit` | **0** |
| `🛑 consecutive-loss halt` | **0** |
| `⚠️ Strategy Studio blackout window active` | **0** |
| `⚠️ Strategy Studio consistency rule` | **0** |
| `not tradeable` (isAccountTradeable refusal) | **0** |
| `🚨 Drawdown close position condition triggered` | **0** |
| `🗓️ session gate: … trade cap reached` (day-plan per-session cap) | **6** |

The 6 session-cap refusals are all 08-19 23:26 → 08-20 00:47 CT, ASIA "3/3 this
session" (the ASIA cap was 3 then; it is 7 now). It is the ONLY trade cap in this
system that has ever refused an entry.

## M4 — the max_daily_trades counterfactual, re-verified at boot 8
Entries per CME session-day (17:00 CT roll), trader `hoang`:

| session-day | entries | first entry CT | last entry CT |
|---|---|---|---|
| 2026-08-30 | 10 | 08-30 17:38:33 | 08-31 13:37:08 |
| 2026-08-31 | 6 | 09-01 02:52:44 | 09-01 13:33:06 |
| 2026-09-01 | 4 | 09-02 00:17:44 | 09-02 10:37:17 |
| 2026-09-02 | 1 | 09-03 09:05:14 | 09-03 09:05:14 |
| 2026-09-03 (current at boot 8) | **0** | — | — |

Session-day 2026-09-01 rows (all qty 1.0, account Sim101):
587 09-02 00:17:44 −62.50 · 588 07:41:05 −65.00 · 589 09:41:04 −155.00 ·
**590 10:37:17 −99.00 ← the 4th entry**. At 590's decision `TradesToday` was 3,
so `TradesToday >= MaxDailyTrades(3)` was TRUE and `Check()` returned
`RiskBlockEntry` — except `MasterEnabled=false` short-circuits at
`kernel/risk_limits.go:241` and returns `RiskAllow`. D38 re-verified.

## M5 — why the "SIZE caps REMAIN enforced" half of the WARN cannot bind
* live equity (latest snapshot 09-04 13:50:09Z) = **$51,906.50**, position_count 0
* notional ceiling = equity × 20 = **$1,038,130**; one MNQ at ~29,000 × $2 = **~$58,000**
  → the ceiling first binds at ~17 contracts while the clamp caps at 1
* per-order clamp `ResolveMaxContracts(2, 2)` → `ClampStageAContracts(2)` →
  `StageAContractCap()` (env `STAGE_A_CONTRACT_CAP` unset) = **1**, and
  `futuresOrderQuantity` floors at 1 — so the clamp can never reduce anything
* arm path does not consult either: `PlaceLimitEntry(..., 1, ...)` is hardcoded at
  `trader/armed_executor.go:965` and `:1603`

## M6 — drawdown auto-close is dead on this instrument
`trader/auto_trader_risk.go:138` fires at `currentPnLPct > 5.0 && drawdownPct >= 40.0`.
NT8 reports `"leverage": 1.0` (`trader/ninjatrader/tcp_trader.go:933`), so
`positionPnLPct` is a raw price percentage. Measured on `trader_positions`
(n=251 CLOSED rows with entry_price>0; 65 carry MFE>0): **max MFE = 156.75 pts =
0.533% of entry**. The 5% arming threshold is ~9.4× the best excursion ever
recorded. 0 fires in 20 days of logs.
