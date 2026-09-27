**VL TRADING SYSTEM**
Implementation Plan & Design
*AI-Powered NQ / MNQ Futures Trading System*
Funded Future Trading · VL Intelligent
Stack: Python + NautilusTrader + Databento + IQFeed + XGBoost + NinjaTrader
Prepared May 2026

# 1. Executive Summary

Goal: build a production-grade automated NQ/MNQ futures day-trading system using ICT/Smart Money Concepts methodology, executed against prop firm funded accounts (Bulenox, MFFU, Apex). The system must support backtest, paper, and live deployment with no code rewrites between modes.

### Core Design Decisions

- NautilusTrader is the execution engine — handles data streaming, indicators, orders, risk, PnL, backtest, live.
- smart-money-concepts is the ICT feature layer — converts raw bars into FVG / OB / BOS / liquidity feature columns.
- Rules first, ML later — prove ICT setups have edge before adding XGBoost meta-labeling.
- NinjaTrader stays as the broker layer — CSV bridge from Python to NinjaScript for prop firm execution.
- Redis + Streamlit for live control panel — read live state, change params, kill switch, all from one UI.
- WSL2 isolated environments via uv — one venv per role, lockfiles for live.

### Build Order (5 Phases)


| **Phase** | **Name** | **Outcome** | **Duration** |
| --- | --- | --- | --- |
| **1** | **Foundation** | **WSL2 envs, IQFeed/Databento Parquet pipeline, NautilusTrader hello-world backtest** | **Week 1** |
| **2** | **Feature Layer** | **smart-money-concepts integration, ICT features written to Parquet, validation vs NT8** | **Week 2** |
| **3** | **Strategy + Backtest** | **ICT strategy class in NautilusTrader, 1-year backtest on NQ, performance baseline** | **Week 3-4** |
| **4** | **Live + Control Panel** | **NinjaTrader CSV bridge, Redis state, Streamlit dashboard, Telegram alerts, paper trading** | **Week 5-6** |
| **5** | **ML Layer (optional)** | **Triple-barrier labels, XGBoost meta-label gate, A/B vs raw strategy** | **Week 7+** |


# 2. Repository Deep-Dive

Each repo, what it ships, exact APIs, integration cost, role in the stack.

## 2.1 NautilusTrader — Execution Engine

github.com/nautechsystems/nautilus_trader · LGPL v3 · 21k+ stars · bi-weekly releases

### What It Ships

- 50+ built-in indicators (auto-updated by engine on every bar/tick)
- All order types — market, limit, stop, stop-limit, trailing, bracket, OCO, MIT, LIT
- Risk engine — pre-trade checks (max order size, position, notional, leverage)
- Native data adapters — Databento (yours), IB, Binance, Bybit, OKX, Coinbase, Tardis
- Backtest + paper + live engines share identical strategy code
- Account types — Cash, Margin (for futures), Betting
- Persistent state via Redis (optional)
- 10 example strategies — EMA cross variants, VolatilityMarketMaker, OrderBookImbalance

### Indicator Reference (50+)


| **Category** | **Indicators** |
| --- | --- |
| **Moving Averages** | **SMA, EMA, DEMA, HMA (Hull), WMA, VWAP, AMA (Kaufman Adaptive), Linear Regression, Moving Average Factory** |
| **Momentum** | **RSI, MACD, Stochastics, CCI, Aroon, Rate of Change, Efficiency Ratio** |
| **Volatility** | **ATR, Bollinger Bands, Donchian Channels, Keltner Channels, Volatility Ratio** |
| **Price Action** | **Swings, Pressure, Fuzzy Candlesticks, On-Balance Volume** |
| **Microstructure** | **Book Imbalance Ratio, Book Imbalance Actor, Spread Analyzer** |


### Strategy Lifecycle Hooks You Implement

on_start()        — register indicators, request history, subscribe live
on_stop()         — flatten, cleanup
on_reset()        — clear state between runs
on_bar(bar)       — main signal logic (called per bar close)
on_quote_tick(t)  — tick-level logic (optional)
on_trade_tick(t)  — trade prints (optional)
on_order_filled() — react to fills
on_position_opened() / on_position_closed() — track positions
on_event()        — catch-all

### What's NOT in NautilusTrader (you bolt on)

- No ICT/SMC indicators → smart-money-concepts
- No ML labeling → mlfinpy
- No NinjaTrader execution adapter → CSV bridge
- No web UI → Streamlit + Redis
- No LLM agents → TradingAgents (if wanted)

## 2.2 smart-money-concepts — ICT Feature Layer

github.com/joshyattridge/smart-money-concepts · MIT · 1.1k+ stars · v0.0.26 (Mar 2025)

### Input Format

Pandas DataFrame with lowercase OHLCV columns: open, high, low, close, volume.

### Full API


| **Function** | **What it returns** |
| --- | --- |
| **smc.fvg(ohlc, join_consecutive=False)** | **Fair Value Gaps: FVG (1=bull,-1=bear), Top, Bottom, MitigatedIndex (where FVG was filled)** |
| **smc.swing_highs_lows(ohlc, swing_length=50)** | **HighLow (1=swing high,-1=swing low), Level (price of swing)** |
| **smc.bos_choch(ohlc, swing_highs_lows, close_break=True)** | **BOS (1=bull break,-1=bear break), CHoCH (character change), Level, BrokenIndex** |
| **smc.ob(ohlc, swing_highs_lows, close_mitigation=False)** | **Order Block: OB (1=bull,-1=bear), Top, Bottom, OBVolume, MitigatedIndex, Percentage (strength)** |
| **smc.liquidity(ohlc, swing_highs_lows, range_percent=0.01)** | **Liquidity (1=bull,-1=bear), Level, End (last liq level idx), Swept (idx of sweep candle)** |
| **smc.previous_high_low(ohlc, time_frame='1D')** | **PreviousHigh, PreviousLow — prior session/day/week levels** |
| **smc.sessions(ohlc, session='London' | 'NewYork' | ...)** | **Active (boolean per bar), High, Low — session high/low tracking** |
| **smc.retracements(ohlc, swing_highs_lows)** | **Fib retracement levels per swing leg** |


### Critical Notes

- swing_length is the most sensitive param — tune per timeframe (5m NQ: try 10-20; 1m NQ: try 20-50).
- liquidity() was O(n²) historically — slow on long histories. Recent PRs sped it up.
- Marked 'educational only' — always validate FVG/OB outputs vs NinjaTrader visuals before trusting.
- Output columns are sparse (NaN where no event) — handle in pipeline.

## 2.3 mlfinpy — ML Labeling & Cross-Validation

github.com/baobach/mlfinpy · MIT · open-source successor to closed-sourced mlfinlab
Skip this entirely until Phase 5. Required only if adding XGBoost meta-labeling.

### Modules You'll Use


| **Module** | **Purpose** |
| --- | --- |
| **labeling.get_events()** | **Triple-barrier event generator — TP / SL / time barrier per setup** |
| **labeling.get_bins()** | **Convert events to binary/categorical labels (win/loss/timeout)** |
| **labeling.meta_labels()** | **Secondary label: given primary signal fired, was it actually a winner?** |
| **filters.cusum_filter()** | **Event sampling — only train on bars with material price movement** |
| **features.fracdiff.frac_diff_ffd()** | **Fractional differentiation — stationary but memory-preserving features** |
| **cross_validation.PurgedKFold** | **K-fold CV that removes train/test leakage from overlapping labels** |
| **sample_weights.get_av_uniqueness_from_triple_barrier()** | **Sample weights for overlapping labels (sklearn sample_weight param)** |
| **bet_sizing.bet_size_probability()** | **Convert model probability → position size (Kelly-style fractional bet)** |


## 2.4 NinjaTrader CSV Bridge — Execution Glue

Reference: github.com/J0shusmc/Claude-Trader-NinjaTrader · NinjaScript .cs file

### How It Works

- Python strategy in NautilusTrader generates a signal.
- Python writes a row to trade_signals.csv on a shared folder (WSL2 /mnt/c/trading).
- NinjaScript strategy polls the CSV every 2 seconds.
- New row → parsed → bracket order placed (entry + SL + TP atomic).
- Signal marked as processed in a status column (or moved to processed file).

### CSV Schema (Recommended)

timestamp,signal_id,action,symbol,qty,entry,stop,target,reason,status
2026-05-20T09:31:00Z,sig_001,BUY,NQ 06-26,1,21500.00,21475.00,21560.00,FVG_LONG,NEW
2026-05-20T10:15:00Z,sig_002,SELL,NQ 06-26,1,21520.00,21545.00,21460.00,LIQ_SWEEP,NEW

### Why CSV over Socket/Pythonnet

- Language-agnostic — Python writes, C# reads, zero coupling
- Debuggable — open the CSV to see exactly what fired
- Persistent audit log — every signal recorded with timestamp
- Crash-safe — if either side restarts, no signals are lost
- 2-second latency is fine for ICT (setups develop over minutes)

## 2.5 TradingAgents — Optional LLM Bias Layer

github.com/TauricResearch/TradingAgents · v0.2.4 (Apr 2026) · 73k+ stars
Optional Phase 6 addition. Runs pre-market only — produces a daily bias label that gates intraday signals.

### Agent Roster

- Fundamentals analyst — earnings, sector rotation
- Sentiment analyst — Reddit, X, news scoring
- News analyst — macro events, Fed, NFP, CPI
- Technical analyst — HTF chart read
- Bull researcher vs Bear researcher (debate)
- Trader agent — synthesizes debate, proposes call
- Risk team (Conservative / Neutral / Aggressive) — vote on final
- Portfolio manager — final sign-off

### Integration Pattern

8:00 AM CT — cron job runs TradingAgents
Output: bias_label ∈ {LONG_ONLY, SHORT_ONLY, BOTH_OK, STAND_ASIDE}
Written to Redis: nq:daily_bias = LONG_ONLY
NautilusTrader strategy reads bias on each bar → filters signals

# 3. System Architecture


## 3.1 End-to-End Flow

┌─────────────────────────────────────────────────────────────────┐
│  DATA LAYER                                                       │
│  ┌──────────────┐         ┌──────────────────┐                   │
│  │  IQFeed      │         │  Databento       │                   │
│  │  (historical)│         │  (live stream)   │                   │
│  └──────┬───────┘         └────────┬─────────┘                   │
│         │                          │                              │
│         ▼                          ▼                              │
│  Parquet files (NQ + MNQ, 1m/5m/15m/1H)                          │
└────────────────────────────┬─────────────────────────────────────┘
▼
┌─────────────────────────────────────────────────────────────────┐
│  FEATURE LAYER                                                    │
│  smart-money-concepts:   smc.fvg, smc.ob, smc.bos_choch,         │
│                          smc.liquidity, smc.sessions             │
│  NautilusTrader builtins: ATR, VWAP, RSI, EMA, Bollinger         │
│  Output: Feature DataFrame (~30-50 columns per bar)              │
└────────────────────────────┬─────────────────────────────────────┘
▼
┌─────────────────────────────────────────────────────────────────┐
│  STRATEGY LAYER (NautilusTrader)                                 │
│  - on_bar(): detect ICT setup using features                     │
│  - 8-factor bias scoring (your existing system)                  │
│  - If setup + bias OK: build bracket order (entry/SL/TP)         │
│  - Risk engine validates order before submission                 │
└────────────────────────────┬─────────────────────────────────────┘
▼
┌─────────────────────────────────────────────────────────────────┐
│  (OPTIONAL) ML META-LABEL GATE                                   │
│  XGBoost model: P(setup wins) > threshold ?                      │
│  If no → skip the trade. If yes → proceed to execution.          │
└────────────────────────────┬─────────────────────────────────────┘
▼
┌─────────────────────────────────────────────────────────────────┐
│  EXECUTION BRIDGE                                                 │
│  Python writes → /mnt/c/trading/signals.csv                      │
│  NinjaScript polls CSV every 2s → places bracket order           │
└────────────────────────────┬─────────────────────────────────────┘
▼
┌─────────────────────────────────────────────────────────────────┐
│  BROKER LAYER                                                     │
│  NinjaTrader 8 → Rithmic/CQG → prop firm account                 │
│  (Bulenox, MFFU Pro, Apex, etc.)                                 │
└────────────────────────────┬─────────────────────────────────────┘
▼
┌─────────────────────────────────────────────────────────────────┐
│  MONITORING LAYER                                                 │
│  Redis (live state) → Streamlit dashboard + Telegram alerts      │
└─────────────────────────────────────────────────────────────────┘

## 3.2 Process Topology

WSL2 (Ubuntu)                              Windows 10 native
─────────────────                          ─────────────────
[nautilus_env]                            NinjaTrader 8
└─ nautilus_strategy.py        ◄──┐     ├─ NQ chart
│                              │   ├─ CSV polling script (.cs)
├─ writes signals.csv ─────────┼───┤
└─ publishes to Redis          │   └─ Rithmic / CQG connector
│
[ml_env]                              │
└─ retrain_xgboost.py               │
└─ saves model.pkl             │   shared folder:
│   C:	radingsignals.csv
[agent_env]                           │   /mnt/c/trading/signals.csv
└─ tradingagents_premarket.py       │
└─ writes bias to Redis        │
│
[control_env]                         │
├─ Redis server :6379               │
├─ Streamlit UI :8501               │
└─ Telegram bot                     │
│
[iqfeed_env]                          │
└─ pyiqfeed historical pull         │
└─ writes to Parquet           │

## 3.3 Environment Isolation Plan


| **Env** | **Packages** | **Lockfile?** |
| --- | --- | --- |
| **nautilus_env** | **nautilus_trader, smartmoneyconcepts, databento, redis, pandas, pyarrow, xgboost (inference only)** | **YES — freeze for live** |
| **ml_env** | **mlfinpy, xgboost, lightgbm, scikit-learn, optuna, pandas, pyarrow, jupyter** | **Snapshot per model version** |
| **agent_env** | **tradingagents, langgraph, langchain, openai/anthropic clients** | **Optional (heavy churn)** |
| **control_env** | **streamlit, redis, fastapi, plotly, python-telegram-bot** | **YES** |
| **iqfeed_env** | **pyiqfeed, pandas, pyarrow** | **YES** |


# 4. Control Panel Design (VL Agent Command)

Dark/gold luxury aesthetic to match your existing TradingView Lightweight Charts dashboard. Streamlit MVP first, migrate to React + TradingView later.

## 4.1 Layout — Desktop View

╔══════════════════════════════════════════════════════════════════════════════╗
║  VL AGENT COMMAND                                          ●LIVE   09:42 CT  ║
║  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━ ║
║                                                                              ║
║  ┌─ KILL SWITCH ─────────────────────────────────────────────────────────┐  ║
║  │  [ 🟢 START ]   [ ⏸ PAUSE ]   [ 🔴 STOP ]   [ ⚠️  FLATTEN ALL ]      │  ║
║  └─────────────────────────────────────────────────────────────────────────┘ ║
║                                                                              ║
║  ┌─ ACCOUNT ──────────────────┐  ┌─ TODAY P&L ─────────────────────────┐   ║
║  │ Bulenox 150K  · $148,250   │  │   Realized:    +$ 420               │   ║
║  │ Daily Loss Limit: $3,000   │  │   Unrealized:  +$ 180               │   ║
║  │ Used: $    420  (14%)      │  │   Total:       +$ 600               │   ║
║  │ Trailing Drawdown: -$4,250 │  │   Trades: 3 (2W / 1L)               │   ║
║  └────────────────────────────┘  └─────────────────────────────────────┘   ║
║                                                                              ║
║  ┌─ POSITION ─────────────────────────────────────────────────────────────┐ ║
║  │  LONG  2 NQ 06-26 @ 21,487.50    SL 21,462.50   TP 21,547.50          │ ║
║  │  Held: 14 min     Unrealized: +$180     Bars to TP: ~8                │ ║
║  └─────────────────────────────────────────────────────────────────────────┘ ║
║                                                                              ║
║  ┌─ ICT STATE ──────────────────────┐  ┌─ BIAS SCORE (8-factor) ──────────┐ ║
║  │ HTF Bias:         BULLISH        │  │  +32   ▓▓▓▓▓▓▓▓░░░░░░░░         │ ║
║  │ Last BOS:         09:18 ⬆        │  │  Threshold: ±25                  │ ║
║  │ Active FVG:       21450-21465 ⬆ │  │  Status: LONG-bias active        │ ║
║  │ Active OB:        21420-21430 ⬆ │  │  Hysteresis: locked until 10:30  │ ║
║  │ Last Liq Sweep:   09:32 (lows)   │  │                                  │ ║
║  │ Session:          NY AM          │  │  Last Signal Prob (XGB): 0.68    │ ║
║  └──────────────────────────────────┘  └──────────────────────────────────┘ ║
║                                                                              ║
║  ┌─ LIVE PARAMETERS (edit live) ─────────────────────────────────────────┐  ║
║  │  Risk %         [▬▬▬●▬▬▬▬▬▬]  0.5%                                   │  ║
║  │  Stop ATR mult  [▬▬▬▬●▬▬▬▬▬]  2.5x                                   │  ║
║  │  Target R/R     [▬▬▬▬▬▬●▬▬▬]  2.5:1                                  │  ║
║  │  Bias threshold [▬▬▬▬●▬▬▬▬▬]  25                                     │  ║
║  │  XGB threshold  [▬▬▬▬▬●▬▬▬▬]  0.60                                   │  ║
║  │  Long only      [○] off                                               │  ║
║  │  Trade window   [✓] 08:30 - 11:30 CT                                  │  ║
║  └─────────────────────────────────────────────────────────────────────────┘ ║
║                                                                              ║
║  ┌─ SYSTEM HEALTH ────────────────────────────────────────────────────────┐ ║
║  │  Databento:     ●  connected   last tick 0.2s ago                     │ ║
║  │  IQFeed:        ●  connected                                          │ ║
║  │  NT8 bridge:    ●  heartbeat 1s ago                                   │ ║
║  │  Redis:         ●  ok                                                 │ ║
║  │  XGB model:     ●  v2026.05.18 loaded                                 │ ║
║  └─────────────────────────────────────────────────────────────────────────┘ ║
║                                                                              ║
║  ┌─ TRADE LOG (today) ────────────────────────────────────────────────────┐ ║
║  │  09:12  LONG  NQ  1 @ 21476.50 → 21501.50 ✓ +$500     FVG_LONG       │ ║
║  │  09:25  LONG  NQ  1 @ 21492.00 → 21467.00 ✗ -$500     FVG_LONG       │ ║
║  │  09:34  LONG  NQ  2 @ 21482.50 → 21500.00 ✓ +$700     LIQ_SWEEP+FVG  │ ║
║  │  09:38  LONG  NQ  2 @ 21487.50 → OPEN     · +$180     FVG_LONG       │ ║
║  └─────────────────────────────────────────────────────────────────────────┘ ║
║                                                                              ║
╚══════════════════════════════════════════════════════════════════════════════╝

## 4.2 Layout — Mobile (Telegram Bot)

┌────────────────────────────┐
│  📊 VL NQ                  │
│  ─────────────────────────│
│  Status: 🟢 LIVE           │
│  Today P&L: +$600 (3 trades)│
│  Position: LONG 2 NQ +$180 │
│  Bias: +32 LONG            │
│  Daily limit: 14% used     │
│                            │
│  Commands:                 │
│   /flatten  /pause  /stop  │
│   /status   /risk 0.3      │
│   /longonly true           │
└────────────────────────────┘

## 4.3 Redis Schema (Single Source of Truth)

Strategy publishes state every bar. UI reads. UI writes params. Strategy reads params next bar.

| **Key** | **Type** | **Owner / Purpose** |
| --- | --- | --- |
| **nq:control:trading_enabled** | **bool** | **UI writes / strategy reads — master killswitch** |
| **nq:control:long_only** | **bool** | **UI writes — disable shorts** |
| **nq:control:flatten_now** | **bool** | **UI writes — strategy flattens on next bar** |
| **nq:params:risk_pct** | **float** | **UI writes — % of account per trade** |
| **nq:params:stop_atr_mult** | **float** | **UI writes — stop = N * ATR** |
| **nq:params:bias_threshold** | **int** | **UI writes — 8-factor score required** |
| **nq:params:xgb_threshold** | **float** | **UI writes — meta-label gate prob** |
| **nq:state:bias_score** | **int** | **Strategy writes — current 8-factor score** |
| **nq:state:htf_bias** | **string** | **Strategy writes — BULLISH/BEARISH/NEUTRAL** |
| **nq:state:active_fvg** | **JSON** | **Strategy writes — {top, bottom, side}** |
| **nq:state:active_ob** | **JSON** | **Strategy writes — {top, bottom, side, vol}** |
| **nq:state:last_signal_prob** | **float** | **Strategy writes — XGB output** |
| **nq:state:open_pnl** | **float** | **Strategy writes — unrealized** |
| **nq:state:realized_pnl_today** | **float** | **Strategy writes** |
| **nq:state:position** | **JSON** | **Strategy writes — {side, qty, avg_px, sl, tp}** |
| **nq:health:databento_last_tick** | **timestamp** | **Strategy writes — for staleness check** |
| **nq:health:nt8_heartbeat** | **timestamp** | **NT8 .cs writes — bridge health** |
| **nq:health:xgb_model_version** | **string** | **Strategy writes on load** |
| **nq:daily_bias** | **string** | **TradingAgents writes premarket — LONG_ONLY etc.** |


## 4.4 Color & Type Tokens


| **Token** | **Value** | **Use** |
| --- | --- | --- |
| **bg-primary** | **#0A0A0A** | **Main background — true black** |
| **bg-panel** | **#141414** | **Card / panel surfaces** |
| **bg-elevated** | **#1F1F1F** | **Modals, dropdowns** |
| **gold-primary** | **#B08D2E** | **VL brand accent, key numbers** |
| **gold-bright** | **#E8B547** | **Hover / active states** |
| **text-primary** | **#F5F5F5** | **Default text** |
| **text-muted** | **#999999** | **Secondary labels** |
| **status-live** | **#2E7D32** | **Connected / profitable** |
| **status-loss** | **#C62828** | **Loss / disconnected / killswitch** |
| **status-warn** | **#E89B2E** | **Approaching daily limit** |
| **font-display** | **Inter / SF Pro Display** | **Headers, big numbers** |
| **font-mono** | **JetBrains Mono** | **Prices, tickers, logs** |


# 5. Phase-by-Phase Implementation


## Phase 1 — Foundation (Week 1)

Goal: working NautilusTrader backtest on NQ with real data. No strategy logic yet — just prove the pipeline runs end-to-end.

### Tasks

- Install uv in WSL2 Ubuntu: curl -LsSf https://astral.sh/uv/install.sh | sh
- Create nautilus_env, iqfeed_env, control_env, ml_env, agent_env
- Verify IQFeed historical pull writes Parquet (you have this working)
- Verify Databento live stream writes Parquet (you have this working)
- Install nautilus_trader, run quickstart EMA cross backtest on test FX data
- Write data_loader.py: Parquet → NautilusTrader BacktestDataConfig
- Run EMA cross backtest on 1 month of NQ 5-min bars from your Parquet

### Done When

- uv pip freeze > nautilus_env.lock saved
- Backtest produces orders, fills, PnL, statistics report
- NQ contract specs (tick size 0.25, tick value $5, MNQ $0.50) verified in instrument config

## Phase 2 — Feature Layer (Week 2)

Goal: ICT features computed once on full history, written to Parquet, validated visually against NinjaTrader.

### Tasks

- uv pip install smartmoneyconcepts in nautilus_env
- Build feature_pipeline.py: load NQ bars Parquet → run smc.* → write features Parquet
- Compute: FVG, OB, BOS/CHoCH, liquidity, swing H/L, session tags, prev day H/L
- Add NautilusTrader-style features computed in Python: ATR(14), VWAP, EMA(10), session VWAP
- Validation: pick 10 random FVGs from Python output, verify they exist on NT8 chart at same bars
- Speed test: full year of 5-min NQ should compute in < 60s

### Feature Matrix Schema (suggested)

timestamp, open, high, low, close, volume,
# ICT
fvg_bull, fvg_bear, fvg_top, fvg_bottom, fvg_mitigated,
ob_bull, ob_bear, ob_top, ob_bottom, ob_vol, ob_pct,
bos, choch, swing_high, swing_low,
liq_bull, liq_bear, liq_level, liq_swept,
# Sessions / context
session_asia, session_london, session_ny_am, session_ny_pm,
prev_day_high, prev_day_low, prev_week_high, prev_week_low,
# Standard TA
atr_14, vwap, vwap_distance, ema_10, ema_20, rsi_14,
# Bias scoring inputs (your 8 factors)
bias_htf, bias_session, bias_liq_sweep, bias_fvg, bias_ob,
bias_vwap, bias_momentum, bias_volume, bias_composite_score

## Phase 3 — Strategy + Backtest (Week 3-4)

Goal: ICT strategy class running in NautilusTrader, full year backtest, baseline metrics established.

### Strategy Class Skeleton

class NQ_ICT_Strategy(Strategy):
def on_start(self):
self.atr = AverageTrueRange(14)
self.vwap = VolumeWeightedAveragePrice()
self.register_indicator_for_bars(self.bar_type, self.atr)
self.register_indicator_for_bars(self.bar_type, self.vwap)
self.smc_buffer = SMCBuffer(window=500)  # rolling SMC features
self.params = ParamLoader('config.yaml')
self.subscribe_bars(self.bar_type)
def on_bar(self, bar):
self.smc_buffer.update(bar)
if not self.atr.initialized:
return
if not self.in_trading_window():
return
if not self.params.trading_enabled:
return
bias = self.compute_8factor_bias()
setup = self.detect_ict_setup(bar, bias)
if not setup:
return
self.submit_bracket(setup, bar)
self.publish_state_to_redis()

### Backtest Targets

- ≥ 100 trades over 1-year period (statistical significance)
- Sharpe ratio (raw rules) baseline established — likely 0.5-1.5 range
- Win rate, average winner, average loser, profit factor logged
- Max drawdown < prop firm daily limit at chosen risk %
- Trade distribution by hour / session / day-of-week analyzed

### Critical Backtest Hygiene

- No look-ahead — features computed only from past bars
- Walk-forward — train on 6 months, test on 1 month, roll
- Slippage modeled — at least 1 tick per fill
- Commissions included — NT8/Rithmic fees per round trip
- Tick size + tick value correct in instrument config

## Phase 4 — Live + Control Panel (Week 5-6)

Goal: paper trading on real Databento stream, full control panel operational, ready to flip to funded account.

### 4A. NinjaTrader CSV Bridge

- Copy J0shusmc/Claude-Trader-NinjaTrader .cs as template
- Modify CSV schema to match your column order
- Add NinjaScript-side: bracket order with NQ tick math (4 ticks/pt)
- Add prop firm rules: max contracts, daily loss check, trade window enforcement
- Add heartbeat: NT8 writes nq:health:nt8_heartbeat to Redis (via small Python helper or file)
- Test with synthetic CSV rows before connecting to real Python

### 4B. Redis State Layer

- Install redis-server on WSL2
- Add publish_state() method to strategy, called at end of every on_bar
- Add reload_params() method called every N bars

### 4C. Streamlit Dashboard

- In control_env: uv pip install streamlit redis plotly
- Build app.py with sections from Section 4.1 layout
- Auto-refresh every 2 seconds (st.rerun() pattern)
- Killswitch buttons publish to Redis pub/sub channel nq:commands
- Strategy subscribes to nq:commands and handles flatten/stop

### 4D. Telegram Bot

- Create bot via @BotFather, get token
- python-telegram-bot polling loop in control_env
- Commands: /status /flatten /pause /resume /stop /risk N /longonly true|false
- Push alerts on: trade open, trade close, daily limit 50/75/90%, disconnect, error

### 4E. Paper Trading Run

- Run for 2 weeks against live Databento stream
- NT8 in simulation mode (Sim101 account)
- Compare paper fills vs backtest expected fills
- Identify slippage / latency / timing discrepancies

## Phase 5 — ML Meta-Label Layer (Week 7+, OPTIONAL)

Only enter Phase 5 if Phase 4 shows positive expectancy with sufficient trades. Otherwise fix rules first.

### When to Skip Phase 5 Permanently

- Rules already profitable at target risk per trade — don't fix what isn't broken
- < 5 trades/day average — not enough samples for meta-labeling to add value
- Win rate already > 60% — XGBoost won't move it meaningfully

### Phase 5 Tasks (if proceeding)

- uv pip install mlfinpy xgboost optuna in ml_env
- Build label_pipeline.py: for each ICT setup in history, apply triple-barrier
- Pick barriers: pt = stop * R/R (your 2.5:1), sl = stop, max hold = 30 bars (5m) or 6 bars (15m)
- CUSUM filter: only sample bars with material price move (threshold ~ 0.5 * ATR)
- Train XGBoost with PurgedKFold (5 splits, 1-day embargo)
- Hyperparameter search via Optuna (max_depth, n_estimators, learning_rate, scale_pos_weight)
- Compare gated vs raw strategy in NautilusTrader backtest
- Accept model only if: gated Sharpe > raw Sharpe AND gated max DD <= raw max DD
- Save versioned model: models/xgb_v2026_05_18.json
- Strategy loads model at startup, inferences in on_bar() (XGBoost CPU inference < 1ms)

### Retrain Schedule

- Monthly retrain on rolling 12-month window (recommended start)
- Triggered manually first, automated via cron later
- Old models archived (never deleted) for audit trail

# 6. Risk Management & Prop Firm Governance


## 6.1 Hard Limits Enforced in Code


| **Limit** | **Where Enforced** | **Action on Breach** |
| --- | --- | --- |
| **Daily loss limit (per firm rules)** | **NautilusTrader risk engine + NT8 .cs** | **Flatten all, disable trading until next day** |
| **Max position size** | **NautilusTrader risk engine** | **Reject order, log warning** |
| **Max contracts open** | **Strategy logic** | **Skip new entries until flat** |
| **Trading window (08:30-11:30 CT)** | **Strategy logic** | **No new entries outside window, manage open positions** |
| **News blackout (high-impact)** | **Strategy logic + economic calendar** | **Pause 5 min before, resume 5 min after** |
| **Data feed staleness > 5s** | **Strategy heartbeat check** | **Flatten, alert, halt new entries** |
| **NT8 bridge stale > 10s** | **Heartbeat in Redis** | **Telegram critical alert, halt new entries** |
| **Consecutive losses N (e.g. 3)** | **Strategy state counter** | **Pause 30 min cooldown** |


## 6.2 Prop Firm Specifics

- Bulenox — 50K/100K/150K accounts, daily loss = 2-3% depending on plan, trailing drawdown rules
- MFFU Pro — algo-friendly, EOD trailing drawdown, news allowed
- Apex — most accounts, but stricter consistency rules (no single big win days)
- Code one config file per firm: bulenox.yaml, mffu.yaml, apex.yaml
- All rules read from config — never hardcode

## 6.3 Operational Safety

- Never run live without paper trading the exact same code for ≥ 2 weeks first
- Never deploy a new model version on a Monday morning — wait for a slow session
- Never disable the killswitch — even briefly
- Always have phone Telegram alerts on when system is live
- Always log every signal, fill, parameter change to disk (immutable audit log)

# 7. Repository / Folder Layout

~/vl_trading/
│
├── .venvs/                          # uv-managed virtual envs
│   ├── nautilus_env/
│   ├── ml_env/
│   ├── agent_env/
│   ├── control_env/
│   └── iqfeed_env/
│
├── locks/                           # frozen requirements per env
│   ├── nautilus_env.lock
│   ├── ml_env.lock
│   └── control_env.lock
│
├── data/
│   ├── raw/
│   │   ├── iqfeed_parquet/         # historical from IQFeed
│   │   │   ├── NQ_1m/
│   │   │   ├── NQ_5m/
│   │   │   └── NQ_15m/
│   │   └── databento_dbn/          # live captures
│   ├── features/                    # SMC-enriched parquet
│   │   └── NQ_5m_features.parquet
│   └── catalog/                     # NautilusTrader Parquet catalog
│
├── src/
│   ├── data_pipeline/
│   │   ├── iqfeed_pull.py
│   │   ├── databento_stream.py
│   │   ├── feature_pipeline.py     # smc.* + ATR/VWAP/etc.
│   │   └── nautilus_loader.py
│   │
│   ├── strategy/
│   │   ├── nq_ict_strategy.py      # main NautilusTrader Strategy
│   │   ├── ict_setups.py           # FVG entry, OB entry, liq sweep entry
│   │   ├── bias_scoring.py         # 8-factor system
│   │   ├── smc_buffer.py           # rolling SMC computation
│   │   ├── param_loader.py         # YAML hot-reload
│   │   └── redis_publisher.py
│   │
│   ├── execution/
│   │   ├── csv_writer.py           # writes signals.csv for NT8
│   │   └── nt8_bridge/
│   │       ├── VLBridge.cs         # NinjaScript strategy
│   │       └── README.md
│   │
│   ├── ml/
│   │   ├── label_pipeline.py       # mlfinpy triple-barrier
│   │   ├── train_xgb.py
│   │   ├── walk_forward.py
│   │   └── inference.py
│   │
│   ├── agents/
│   │   └── premarket_bias.py       # TradingAgents wrapper
│   │
│   └── control/
│       ├── streamlit_app.py        # dashboard
│       ├── telegram_bot.py
│       └── alert_router.py
│
├── configs/
│   ├── strategy.yaml                # hot-reloadable params
│   ├── bulenox.yaml                 # prop firm rules
│   ├── mffu.yaml
│   └── instruments/
│       ├── nq_06_26.yaml
│       └── mnq_06_26.yaml
│
├── models/
│   ├── xgb_v2026_05_18.json
│   └── xgb_v2026_06_15.json
│
├── logs/
│   ├── strategy/                    # daily strategy logs
│   ├── trades/                      # immutable trade log
│   └── signals_csv_archive/         # archived signals.csv per day
│
├── tests/
│   ├── test_feature_pipeline.py
│   ├── test_ict_setups.py
│   ├── test_bias_scoring.py
│   └── test_csv_bridge.py
│
├── notebooks/
│   ├── 01_data_exploration.ipynb
│   ├── 02_smc_validation.ipynb
│   ├── 03_backtest_analysis.ipynb
│   └── 04_xgb_meta_label.ipynb
│
├── scripts/
│   ├── start_live.sh
│   ├── start_paper.sh
│   ├── start_dashboard.sh
│   └── kill_all.sh
│
└── README.md

# 8. Success Metrics & Go/No-Go Gates


## 8.1 Phase Gates


| **Gate** | **Criteria** | **If Failed** |
| --- | --- | --- |
| **P1→P2** | **Backtest runs end-to-end, instrument specs verified, ≥ 1 month NQ data loadable** | **Fix data pipeline before adding features** |
| **P2→P3** | **Feature parquet validates visually against NT8 (≥ 90% match on 10 random samples)** | **Fix SMC integration or swing_length tuning** |
| **P3→P4** | **Backtest Sharpe > 0.8, profit factor > 1.3, ≥ 100 trades, max DD < 50% of daily limit** | **Refine setups or parameters before going live** |
| **P4→Live** | **2 weeks paper trading: realized PnL within 20% of backtest expectation, no system errors > 1/week** | **Extend paper period until stable** |
| **P4→P5** | **Live trading proves positive expectancy over ≥ 100 real trades** | **Don't add ML — rules need work first** |
| **P5 accept** | **Gated strategy: Sharpe > raw Sharpe AND DD ≤ raw DD AND avg trades/day ≥ 50% of raw** | **Discard model, keep raw rules** |


## 8.2 Live KPIs (daily review)

- Realized PnL
- Number of trades
- Win rate (trailing 20)
- Average winner / average loser
- Profit factor (trailing 20)
- Slippage realized vs expected (ticks per trade)
- System uptime / disconnect count
- Time-to-fill (signal → broker fill latency)
- Killswitch activations (should be 0 in normal weeks)

# 9. Appendix


## 9.1 All Repos & Links


| **Repo** | **URL** | **Phase** |
| --- | --- | --- |
| **NautilusTrader** | **github.com/nautechsystems/nautilus_trader** | **1+** |
| **smart-money-concepts** | **github.com/joshyattridge/smart-money-concepts** | **2+** |
| **mlfinpy** | **github.com/baobach/mlfinpy** | **5** |
| **XGBoost** | **github.com/dmlc/xgboost** | **5** |
| **TradingAgents** | **github.com/TauricResearch/TradingAgents** | **6 (opt)** |
| **NT8 CSV bridge ref** | **github.com/J0shusmc/Claude-Trader-NinjaTrader** | **4** |
| **Microsoft Qlib (alt ML)** | **github.com/microsoft/qlib** | **5 (alt)** |
| **uv (env manager)** | **github.com/astral-sh/uv** | **1** |
| **Streamlit (dashboard)** | **github.com/streamlit/streamlit** | **4** |
| **python-telegram-bot** | **github.com/python-telegram-bot/python-telegram-bot** | **4** |


## 9.2 Pre-Flight Checklist (before going live with real money)

- Paper traded ≥ 2 weeks with realized PnL tracking backtest expectations
- All 8 risk limits tested by deliberate breach in paper
- Killswitch tested from UI, Telegram, and SSH
- NT8 bridge survives a forced NinjaTrader restart mid-trade
- Strategy survives a Databento disconnect + reconnect
- XGBoost model (if used) survives a stale-data scenario gracefully
- Daily loss alert at 50% triggers correctly
- Telegram bot reachable from your phone (test from a coffee shop)
- Prop firm rules YAML matches written firm rules verbatim
- First-day cap set: max 1 contract regardless of risk %

## 9.3 Anti-Patterns to Avoid

- Mega-env with everything in one venv → dep hell within a month
- ML before rules are proven → curve-fit overfit disaster
- No paper period → first live week is your most expensive lesson
- Skipping the heartbeat → silent disconnect = silent disaster
- Hardcoded params → can't tune live, every change is a restart
- Trusting backtest blindly → look-ahead bias is sneaky, validate twice
- Editing live env to 'just try' a new version → broken stack, mid-session
- No retrain schedule → 6 months later, model is on old market regime

## 9.4 Open Questions to Resolve Early

- Tick replay vs bar-close backtest — Phase 3 decision
- MNQ for development / scaling, NQ for production — when to switch
- Single-firm vs multi-firm portfolio — risk multiplied across accounts
- Strategy diversification — do you run 3 sleeves (ORB, ICT, VWAP-IB) simultaneously or sequentially?
- Drawdown protocol — auto-pause threshold per week vs hard stop per day
*— End of Plan —*
