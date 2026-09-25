**VL TRADING SYSTEM**
Build Plan
*AI-powered NQ / MNQ futures day-trading system*
Funded Future Trading · VL Intelligent
Data · Strategy · ML · Frontend · Backend · Deployment

# 1. Stack & Build Phases

Goal: production-grade automated NQ/MNQ futures day-trading using ICT methodology, executed against prop firm funded accounts. Backtest, paper, and live all run the same code.

## 1.1 Stack


| **Layer** | **Technology** |
| --- | --- |
| **Data — historical** | **IQFeed via pyiqfeed → Parquet** |
| **Data — live** | **Databento (CMBP-1) → in-memory + Parquet** |
| **Features** | **smart-money-concepts (github.com/joshyattridge/smart-money-concepts)** |
| **Engine** | **NautilusTrader (github.com/nautechsystems/nautilus_trader) — backtest, paper, live in one engine** |
| **ML (optional)** | **XGBoost + mlfinpy (github.com/baobach/mlfinpy)** |
| **Premarket bias (opt)** | **TradingAgents (github.com/TauricResearch/TradingAgents)** |
| **Execution bridge** | **CSV polled by NinjaScript (pattern from github.com/J0shusmc/Claude-Trader-NinjaTrader)** |
| **Broker** | **NinjaTrader 8 → Rithmic/CQG → Bulenox/MFFU/Apex** |
| **State bus** | **Redis (KV + pub/sub)** |
| **Backend API** | **FastAPI (REST + WebSocket)** |
| **Frontend** | **React + Vite + TypeScript + Tailwind + Lightweight Charts v5** |
| **Mobile alerts** | **Telegram bot** |
| **Process mgmt** | **systemd on WSL2; NT8 auto-start on Windows** |
| **Env mgmt** | **uv (github.com/astral-sh/uv) — one venv per role, lockfiles** |


## 1.2 Phases


| **#** | **Phase** | **Outcome** | **Time** |
| --- | --- | --- | --- |
| **1** | **Foundation** | **WSL2 envs, data pipeline, NautilusTrader runs** | **Week 1** |
| **2** | **Features** | **SMC + standard features in Parquet, validated** | **Week 2** |
| **3** | **Strategy + Backtest** | **ICT strategy class, 1-year backtest, baseline metrics** | **Week 3-4** |
| **4** | **Backend + State** | **FastAPI, Redis, NT8 CSV bridge, Telegram bot** | **Week 5** |
| **5** | **Frontend** | **React dashboard with live chart and control panel** | **Week 6-7** |
| **6** | **Paper → Live** | **2 weeks paper, then funded account with 1-contract cap** | **Week 8-9** |
| **7** | **ML (optional)** | **Triple-barrier + XGBoost meta-label gate** | **Week 10+** |


# 2. Architecture


## 2.1 End-to-end flow

┌──────────────────────────────────────────────────────────────────┐
│  DATA                                                             │
│   IQFeed historical ───┐     ┌─── Databento live (CMBP-1)        │
│                        ▼     ▼                                    │
│              Parquet store (NQ + MNQ, 1m/5m/15m/1H)              │
└────────────────────────┬─────────────────────────────────────────┘
▼
┌──────────────────────────────────────────────────────────────────┐
│  FEATURES                                                         │
│   smart-money-concepts: fvg, ob, bos_choch, liquidity, sessions  │
│   + NautilusTrader builtins: ATR, VWAP, RSI, EMA, BB             │
└────────────────────────┬─────────────────────────────────────────┘
▼
┌──────────────────────────────────────────────────────────────────┐
│  STRATEGY (NautilusTrader)                                        │
│   on_bar: setup detection + 8-factor bias + risk engine           │
│   Publishes state to Redis every bar                              │
└────────────────────────┬─────────────────────────────────────────┘
▼
┌──────────────────────────────────────────────────────────────────┐
│  ML GATE (optional, Phase 7)                                      │
│   XGBoost meta-label: P(setup wins) > threshold?                  │
└────────────────────────┬─────────────────────────────────────────┘
▼
┌──────────────────────────────────────────────────────────────────┐
│  EXECUTION                                                        │
│   Python writes signals.csv → NinjaScript reads → bracket order   │
└────────────────────────┬─────────────────────────────────────────┘
▼
NT8 → Rithmic/CQG → Prop firm funded account
STATE BUS (parallel)
Redis ◄── Strategy + NT8 + TradingAgents publish
──► FastAPI ──► React dashboard via WebSocket
──► Telegram bot for mobile alerts

## 2.2 Process topology

WSL2 (Ubuntu)                              Windows 10 native
──────────────                              ────────────────
[iqfeed_env]                               IQFeed terminal app
└─ pyiqfeed historical pull              (always running)
[nautilus_env]                             NinjaTrader 8
├─ data ingest                           ├─ NQ chart
├─ feature pipeline                      ├─ VLBridge.cs strategy
├─ NautilusTrader strategy ──────────────┤
│    └─ writes signals.csv ─────► /mnt/c/trading/signals.csv
│    └─ publishes to Redis               │
└─ logs to disk                          └─ Rithmic/CQG
│
[ml_env]                                           ▼
└─ retrain (monthly cron)                Prop firm account
└─ saves model.pkl
[agent_env]
└─ tradingagents premarket (cron 8 AM)
└─ writes daily bias to Redis
[control_env]
├─ redis-server :6379
├─ fastapi backend :8000
├─ react served via nginx :80
└─ telegram bot

## 2.3 Environment isolation


| **Env** | **Packages** | **Lockfile** |
| --- | --- | --- |
| **nautilus_env** | **nautilus_trader, smartmoneyconcepts, databento, redis, pandas, pyarrow, xgboost (inference)** | **Yes — freeze for live** |
| **ml_env** | **mlfinpy, xgboost, lightgbm, sklearn, optuna, jupyter** | **Per model snapshot** |
| **agent_env** | **tradingagents, langgraph, langchain** | **Optional** |
| **control_env** | **fastapi, uvicorn, redis, python-telegram-bot, pydantic** | **Yes** |
| **iqfeed_env** | **pyiqfeed, pandas, pyarrow** | **Yes** |
| **frontend (node)** | **react, vite, typescript, tailwindcss, lightweight-charts, zustand** | **package-lock.json** |


# 3. Repo APIs You'll Call


## 3.1 NautilusTrader

github.com/nautechsystems/nautilus_trader · LGPL v3
Strategy lifecycle hooks you implement:
on_start, on_stop, on_reset, on_bar, on_quote_tick, on_trade_tick,
on_order_filled, on_position_opened, on_position_closed, on_event
Built-in indicators (auto-fed by the engine):

| **Category** | **Indicators** |
| --- | --- |
| **Moving Avg** | **SMA, EMA, DEMA, HMA, WMA, VWAP, AMA (Kaufman), Linear Regression** |
| **Momentum** | **RSI, MACD, Stochastics, CCI, Aroon, Rate of Change, Efficiency Ratio** |
| **Volatility** | **ATR, Bollinger Bands, Donchian, Keltner, Volatility Ratio** |
| **Price action** | **Swings, Pressure, Fuzzy Candlesticks, OBV** |
| **Microstructure** | **Book Imbalance Ratio, Book Imbalance Actor, Spread Analyzer** |

Order types: market, limit, stop, stop-limit, trailing, bracket, OCO, MIT, LIT. Account types: Cash, Margin, Betting. Native Databento adapter (DBN files drop in).

## 3.2 smart-money-concepts

github.com/joshyattridge/smart-money-concepts · MIT · v0.0.26 (Mar 2025)
Input: pandas DataFrame with lowercase OHLCV. Output: same DataFrame plus new columns.

| **Function** | **Returns** |
| --- | --- |
| **smc.fvg(ohlc, join_consecutive=False)** | **FVG (±1), Top, Bottom, MitigatedIndex** |
| **smc.swing_highs_lows(ohlc, swing_length=50)** | **HighLow (±1), Level** |
| **smc.bos_choch(ohlc, swings, close_break=True)** | **BOS (±1), CHoCH (±1), Level, BrokenIndex** |
| **smc.ob(ohlc, swings, close_mitigation=False)** | **OB (±1), Top, Bottom, OBVolume, MitigatedIndex, Percentage** |
| **smc.liquidity(ohlc, swings, range_percent=0.01)** | **Liquidity (±1), Level, End, Swept** |
| **smc.previous_high_low(ohlc, time_frame='1D')** | **PreviousHigh, PreviousLow** |
| **smc.sessions(ohlc, session='London'|'NewYork'|...)** | **Active (bool), High, Low** |
| **smc.retracements(ohlc, swings)** | **Fib retracement levels** |

Notes:
- swing_length is the most sensitive param. NQ 5m: 10-20. NQ 1m: 20-50.
- liquidity() historically O(n²) — test speed on long histories
- Validate visually against NT8 for the first 10 events before trusting

## 3.3 mlfinpy (Phase 7 only)

github.com/baobach/mlfinpy · MIT · open-source successor to closed-sourced mlfinlab

| **Module** | **Purpose** |
| --- | --- |
| **labeling.get_events()** | **Triple-barrier event generator (TP/SL/time)** |
| **labeling.get_bins()** | **Convert events to win/loss/timeout labels** |
| **labeling.meta_labels()** | **Secondary label for primary-signal filtering** |
| **filters.cusum_filter()** | **Sample only bars with material price moves** |
| **features.fracdiff.frac_diff_ffd()** | **Fractional diff: stationary + memory-preserving** |
| **cross_validation.PurgedKFold** | **Leak-free k-fold CV for overlapping labels** |
| **bet_sizing.bet_size_probability()** | **Model probability → fractional position size** |


# 4. Control Panel


## 4.1 Desktop wireframe

╔══════════════════════════════════════════════════════════════════════════════╗
║  VL AGENT COMMAND                                          ●LIVE   09:42 CT  ║
║  ════════════════════════════════════════════════════════════════════════════║
║                                                                              ║
║  [ 🟢 START ]   [ ⏸ PAUSE ]   [ 🔴 STOP ]   [ ⚠️  FLATTEN ALL ]             ║
║                                                                              ║
║  ┌─ ACCOUNT ─────────┐  ┌─ TODAY P&L ────┐  ┌─ POSITION ──────────────────┐ ║
║  │ Bulenox 150K      │  │ Realized +$420 │  │ LONG 2 NQ @ 21487.50        │ ║
║  │ Daily used 14%    │  │ Unreal   +$180 │  │ SL 21462.50  TP 21547.50    │ ║
║  │ Trailing -$4,250  │  │ 3 trades 2W/1L │  │ Held 14m  Unreal +$180      │ ║
║  └───────────────────┘  └────────────────┘  └─────────────────────────────┘ ║
║                                                                              ║
║  ┌─ LIVE CHART (TradingView Lightweight) ─────────────────────────────────┐ ║
║  │   NQ 5m candles + FVG zones (gold dashed) + OB boxes (gold solid)     │ ║
║  │   + liq sweep markers + BOS/CHoCH triangles + entry/exit pins         │ ║
║  └────────────────────────────────────────────────────────────────────────┘ ║
║                                                                              ║
║  ┌─ ICT STATE ──────────────────┐  ┌─ 8-FACTOR BIAS ────────────────────┐   ║
║  │ HTF: BULLISH                 │  │ +32  ▓▓▓▓▓▓▓▓░░░░░░░░              │   ║
║  │ Last BOS: 09:18 ⬆            │  │ Threshold: ±25                     │   ║
║  │ Active FVG: 21450-21465 ⬆    │  │ Status: LONG-bias active           │   ║
║  │ Active OB:  21420-21430 ⬆    │  │ Locked until: 10:30                │   ║
║  │ Last Liq Swp: 09:32 lows     │  │ XGB prob: 0.68                     │   ║
║  │ Session: NY AM               │  │                                    │   ║
║  └──────────────────────────────┘  └────────────────────────────────────┘   ║
║                                                                              ║
║  ┌─ LIVE PARAMETERS ──────────────────────────────────────────────────────┐ ║
║  │  Risk %         [▬▬▬●▬▬▬▬▬▬]  0.5%                                    │ ║
║  │  Stop ATR mult  [▬▬▬▬●▬▬▬▬▬]  2.5x                                    │ ║
║  │  Target R/R     [▬▬▬▬▬▬●▬▬▬]  2.5:1                                   │ ║
║  │  Bias threshold [▬▬▬▬●▬▬▬▬▬]  25                                      │ ║
║  │  XGB threshold  [▬▬▬▬▬●▬▬▬▬]  0.60                                    │ ║
║  │  Long only [○]    Trade window [✓] 08:30 - 11:30 CT                   │ ║
║  └────────────────────────────────────────────────────────────────────────┘ ║
║                                                                              ║
║  ●Databento  ●NT8  ●Redis  ●XGB v2026.05.18                                 ║
║                                                                              ║
║  ┌─ TRADES TODAY ─────────────────────────────────────────────────────────┐ ║
║  │  09:12  LONG  1 NQ  21476.50 → 21501.50  ✓ +$500  FVG_LONG            │ ║
║  │  09:25  LONG  1 NQ  21492.00 → 21467.00  ✗ -$500  FVG_LONG            │ ║
║  │  09:34  LONG  2 NQ  21482.50 → 21500.00  ✓ +$700  LIQ_SWEEP+FVG       │ ║
║  │  09:38  LONG  2 NQ  21487.50 → OPEN      · +$180  FVG_LONG            │ ║
║  └────────────────────────────────────────────────────────────────────────┘ ║
╚══════════════════════════════════════════════════════════════════════════════╝

## 4.2 Mobile / Telegram

┌────────────────────────────┐
│  📊 VL NQ                  │
│  ─────────────────────────│
│  Status: 🟢 LIVE           │
│  Today P&L: +$600 (3)      │
│  Position: LONG 2 NQ +$180 │
│  Bias: +32 LONG            │
│  Daily limit: 14% used     │
│                            │
│  Commands:                 │
│   /flatten  /pause  /stop  │
│   /status   /risk 0.3      │
│   /longonly true           │
└────────────────────────────┘

## 4.3 Redis schema (the data contract)


| **Key** | **Type** | **Owner / Purpose** |
| --- | --- | --- |
| **nq:control:trading_enabled** | **bool** | **UI writes / strategy reads — killswitch** |
| **nq:control:long_only** | **bool** | **UI writes — disable shorts** |
| **nq:control:flatten_now** | **bool** | **UI writes — flatten next bar** |
| **nq:params:risk_pct** | **float** | **UI writes — % of account per trade** |
| **nq:params:stop_atr_mult** | **float** | **UI writes — stop = N * ATR** |
| **nq:params:bias_threshold** | **int** | **UI writes — 8-factor score required** |
| **nq:params:xgb_threshold** | **float** | **UI writes — meta-label gate prob** |
| **nq:state:bias_score** | **int** | **Strategy writes — current 8-factor** |
| **nq:state:htf_bias** | **string** | **Strategy — BULLISH/BEARISH/NEUTRAL** |
| **nq:state:active_fvg** | **JSON** | **Strategy — {top, bottom, side}** |
| **nq:state:active_ob** | **JSON** | **Strategy — {top, bottom, side, vol}** |
| **nq:state:last_signal_prob** | **float** | **Strategy — XGB output** |
| **nq:state:open_pnl** | **float** | **Strategy — unrealized** |
| **nq:state:realized_pnl_today** | **float** | **Strategy** |
| **nq:state:position** | **JSON** | **Strategy — {side, qty, avg_px, sl, tp}** |
| **nq:state:trades_today** | **LIST** | **Strategy LPUSH on each close** |
| **nq:health:databento_last_tick** | **timestamp** | **Strategy — staleness check** |
| **nq:health:nt8_heartbeat** | **timestamp** | **NT8 — bridge health** |
| **nq:health:xgb_model_version** | **string** | **Strategy — model loaded** |
| **nq:daily_bias** | **string** | **TradingAgents premarket — LONG_ONLY etc.** |
| **nq:commands (pub/sub)** | **channel** | **UI publishes → strategy subscribes** |
| **nq:events (pub/sub)** | **channel** | **Strategy publishes → UI subscribes** |


## 4.4 Color & type tokens


| **Token** | **Value** | **Use** |
| --- | --- | --- |
| **bg-primary** | **#0A0A0A** | **True black background** |
| **bg-panel** | **#141414** | **Card surfaces** |
| **bg-elevated** | **#1F1F1F** | **Modals** |
| **gold** | **#B08D2E** | **Brand accent, key numbers** |
| **gold-bright** | **#E8B547** | **Hover / active** |
| **gold-dim** | **#8A6F24** | **Disabled** |
| **text-primary** | **#F5F5F5** | **Default text** |
| **text-muted** | **#999999** | **Secondary** |
| **status-live** | **#22C55E** | **Connected / profit** |
| **status-loss** | **#EF4444** | **Loss / disconnect** |
| **status-warn** | **#E89B2E** | **Approaching limit** |
| **font-sans** | **Inter** | **Headers, body** |
| **font-mono** | **JetBrains Mono** | **Prices, tickers, logs** |


# Phase 1 — Foundation (Week 1)

Goal: working NautilusTrader backtest on NQ. End-to-end pipeline runs.

## Tasks

- Install uv in WSL2: curl -LsSf https://astral.sh/uv/install.sh | sh
- Create all envs: nautilus_env, iqfeed_env, control_env, ml_env, agent_env
- Verify IQFeed pulls historical → Parquet
- Verify Databento live stream → in-memory + Parquet
- Install nautilus_trader; run quickstart EMA cross backtest on test data
- Write data_loader.py: Parquet → NautilusTrader BacktestDataConfig
- Run EMA cross backtest on 1 month of NQ 5-min from your Parquet
- Freeze: uv pip freeze > locks/nautilus_env.lock

## Done when

- Backtest produces orders, fills, PnL, statistics report
- NQ contract specs verified: tick size 0.25, tick value $5 (MNQ $0.50)
- nautilus_env.lock committed

# Phase 2 — Features (Week 2)

Goal: SMC + standard features in Parquet, validated visually against NinjaTrader.

## Tasks

- uv pip install smartmoneyconcepts in nautilus_env
- Build feature_pipeline.py: NQ bars → smc.* → features Parquet
- Compute: FVG, OB, BOS/CHoCH, liquidity, swings, sessions, prev day H/L
- Add: ATR(14), VWAP, EMA(10), session VWAP
- Validation: pick 10 random FVGs from Python, verify they exist on NT8 chart at same bars
- Speed check: full year of NQ 5-min should compute < 60s

## Feature matrix schema

timestamp, open, high, low, close, volume,
# ICT
fvg_bull, fvg_bear, fvg_top, fvg_bottom, fvg_mitigated,
ob_bull, ob_bear, ob_top, ob_bottom, ob_vol, ob_pct,
bos, choch, swing_high, swing_low,
liq_bull, liq_bear, liq_level, liq_swept,
# Context
session_asia, session_london, session_ny_am, session_ny_pm,
prev_day_high, prev_day_low, prev_week_high, prev_week_low,
# Standard TA
atr_14, vwap, vwap_distance, ema_10, ema_20, rsi_14,
# 8-factor bias inputs
bias_htf, bias_session, bias_liq_sweep, bias_fvg, bias_ob,
bias_vwap, bias_momentum, bias_volume, bias_composite_score

## Done when

- ≥ 90% match on 10 visual FVG validations vs NT8
- Features Parquet covers full historical period
- Speed test passes

# Phase 3 — Strategy + Backtest (Weeks 3-4)

Goal: ICT strategy class running in NautilusTrader. 1-year backtest. Baseline metrics established.

## Tasks

- Build NQ_ICT_Strategy(Strategy) class
- Implement detect_ict_setup(): FVG entry, OB entry, liquidity-sweep reversal
- Implement compute_8factor_bias() (port your existing logic)
- Implement submit_bracket(): 2.5x ATR stop, 2.5:1 R/R
- Run full-year backtest on NQ 5-min, target ≥ 100 trades
- Walk-forward: train 6 months / test 1 month, roll
- Slippage: model 1 tick min; commissions per round trip

## Strategy class skeleton

# src/strategy/nq_ict_strategy.py
from decimal import Decimal
from nautilus_trader.config import StrategyConfig
from nautilus_trader.model.data import Bar, BarType
from nautilus_trader.model.identifiers import InstrumentId
from nautilus_trader.model.enums import OrderSide
from nautilus_trader.trading.strategy import Strategy
from nautilus_trader.indicators.atr import AverageTrueRange
from nautilus_trader.indicators.vwap import VolumeWeightedAveragePrice
from src.strategy.smc_buffer import SMCBuffer
from src.strategy.bias_scoring import BiasScorer
from src.strategy.ict_setups import detect_ict_setup
from src.strategy.redis_publisher import RedisPublisher
from src.strategy.param_loader import ParamLoader
class NQ_ICT_Config(StrategyConfig, frozen=True):
instrument_id: InstrumentId
bar_type: BarType
trade_size: Decimal
class NQ_ICT_Strategy(Strategy):
def __init__(self, config: NQ_ICT_Config):
super().__init__(config)
self.atr = AverageTrueRange(14)
self.vwap = VolumeWeightedAveragePrice()
self.smc = SMCBuffer(window=500)
self.bias = BiasScorer()
self.params = ParamLoader("configs/strategy.yaml")
self.redis = RedisPublisher()
self.last_signal_prob = 0.0
def on_start(self):
self.register_indicator_for_bars(self.config.bar_type, self.atr)
self.register_indicator_for_bars(self.config.bar_type, self.vwap)
self.subscribe_bars(self.config.bar_type)
# Background thread: Redis commands
import threading
threading.Thread(
target=self.redis.subscribe_commands,
args=(self._handle_command,),
daemon=True,
).start()
def _handle_command(self, cmd: str):
if cmd == "flatten_all":
self.close_all_positions(self.config.instrument_id)
def on_bar(self, bar: Bar):
if self.bar_count % 30 == 0:
self.params.reload()
self.smc.update(bar)
if not self.atr.initialized or not self.smc.ready():
return
if not self.redis.is_trading_enabled():
self.publish_state(bar); return
if not self.in_trading_window(bar.ts_event):
self.publish_state(bar); return
bias_data = self.bias.compute(bar, self.smc, self.atr, self.vwap)
if abs(bias_data.score) < self.params.bias_threshold:
self.publish_state(bar, bias_data); return
setup = detect_ict_setup(bar, self.smc, bias_data, self.params)
if not setup:
self.publish_state(bar, bias_data); return
if self.params.use_meta_label:
prob = self.predict_meta_label(setup, bias_data)
self.last_signal_prob = prob
if prob < self.params.xgb_threshold:
self.publish_state(bar, bias_data); return
self.submit_bracket(setup, bar)
self.publish_state(bar, bias_data)
def submit_bracket(self, setup, bar):
atr_val = float(self.atr.value)
stop_dist = atr_val * self.params.stop_atr_mult
target_dist = stop_dist * self.params.target_rr
entry = float(bar.close)
if setup.side == "LONG":
sl, tp, side = entry - stop_dist, entry + target_dist, OrderSide.BUY
else:
sl, tp, side = entry + stop_dist, entry - target_dist, OrderSide.SELL
order_list = self.order_factory.bracket(
instrument_id=self.config.instrument_id,
order_side=side,
quantity=self.config.trade_size,
entry_price=self.instrument.make_price(entry),
sl_trigger_price=self.instrument.make_price(sl),
tp_price=self.instrument.make_price(tp),
)
self.submit_order_list(order_list)

## Done when (gate to Phase 4)

- Backtest Sharpe > 0.8
- Profit factor > 1.3
- ≥ 100 trades
- Max drawdown < 50% of daily limit at chosen risk %
- If failed: refine setups or parameters before going live

# Phase 4 — Backend + State (Week 5)

Goal: Redis + FastAPI live. NT8 CSV bridge works. Telegram bot sends alerts.

## Tasks

- Install Redis on WSL2: sudo apt install redis-server
- Add publish_state() to strategy (called at end of every on_bar)
- Add reload_params() polling Redis every N bars
- Build FastAPI backend in control_env
- Build WebSocket endpoint for live state stream
- Modify J0shusmc CSV bridge .cs for your schema + prop firm rules
- NT8 writes heartbeat to file every 5 seconds (Redis-bridge daemon picks it up)
- Build Telegram bot: commands + push alerts

## Backend stack

- FastAPI — async REST + WebSocket
- uvicorn — ASGI server
- redis-py async
- Pydantic v2 for schemas
- loguru for structured logging

## REST endpoints


| **Method** | **Path** | **Purpose** |
| --- | --- | --- |
| **POST** | **/api/control/start** | **Set trading_enabled = true** |
| **POST** | **/api/control/pause** | **trading_enabled = false (manage open)** |
| **POST** | **/api/control/stop** | **Pause + close all at market** |
| **POST** | **/api/control/flatten** | **PUBLISH nq:commands 'flatten_all'** |
| **GET** | **/api/params** | **All nq:params:*** |
| **PUT** | **/api/params/{key}** | **Set nq:params:{key} (validated)** |
| **GET** | **/api/state** | **Snapshot of all nq:state:*** |
| **GET** | **/api/trades/today** | **LRANGE nq:state:trades_today** |
| **GET** | **/api/system/health** | **All nq:health:*** |
| **GET** | **/api/chart/bars?tf=5m&n=300** | **Last N candles + active FVG/OB** |
| **WS** | **/ws** | **Push state + events + ticks** |


## WebSocket message types

// Envelope
{ "type": "state_update" | "trade_event" | "tick" | "alert", "data": {...} }
// state_update — every ~1s with full snapshot
{
"type": "state_update",
"data": {
"account": { "balance": 148250, "dailyUsed": 420, "dailyLimit": 3000, "trailingDD": -4250 },
"pnl": { "realized": 420, "unrealized": 180, "tradesCount": 3, "wins": 2, "losses": 1 },
"position": { "side": "LONG", "qty": 2, "avgPx": 21487.5, "sl": 21462.5, "tp": 21547.5 },
"bias": { "score": 32, "htf": "BULLISH", "xgbProb": 0.68, "lockedUntil": "10:30" },
"ict": {
"fvg": { "top": 21465, "bottom": 21450, "side": "BULL" },
"ob":  { "top": 21430, "bottom": 21420, "side": "BULL", "vol": 12500 },
"lastBos": { "time": "09:18", "side": "BULL" },
"session": "NY_AM"
},
"health": { "databento": true, "nt8": true, "redis": true, "xgbVersion": "v2026.05.18" }
}
}
// trade_event — on every position open/close
{
"type": "trade_event",
"data": { "event": "OPENED" | "CLOSED",
"trade": { "time": "09:38", "side": "LONG", "qty": 2,
"entry": 21487.5, "exit": null, "pnl": 0, "reason": "FVG_LONG" } }
}
// tick — for live chart
{ "type": "tick", "data": { "ts": "2026-05-20T14:42:13Z", "price": 21492.25 } }
// alert — system warning
{ "type": "alert", "data": { "level": "WARN" | "CRITICAL", "msg": "Databento stale 8s" } }

## FastAPI main.py skeleton

# src/backend/main.py
from fastapi import FastAPI, WebSocket, WebSocketDisconnect, Header, HTTPException, Depends
from fastapi.middleware.cors import CORSMiddleware
from redis.asyncio import Redis
import asyncio, json, os
from loguru import logger
API_TOKEN = os.environ["VL_API_TOKEN"]
REDIS_URL = os.environ.get("REDIS_URL", "redis://localhost:6379")
app = FastAPI(title="VL Backend")
app.add_middleware(CORSMiddleware,
allow_origins=["http://localhost:3000", "https://vl.yourdomain.com"],
allow_methods=["*"], allow_headers=["*"])
redis: Redis = None
@app.on_event("startup")
async def startup():
global redis
redis = Redis.from_url(REDIS_URL, decode_responses=True)
def auth(authorization: str = Header(None)):
if authorization != f"Bearer {API_TOKEN}":
raise HTTPException(401, "Unauthorized")
@app.post("/api/control/start", dependencies=[Depends(auth)])
async def control_start():
await redis.set("nq:control:trading_enabled", "true")
return {"ok": True}
@app.post("/api/control/flatten", dependencies=[Depends(auth)])
async def control_flatten():
await redis.publish("nq:commands", "flatten_all")
return {"ok": True}
@app.put("/api/params/{key}", dependencies=[Depends(auth)])
async def set_param(key: str, payload: dict):
await redis.set(f"nq:params:{key}", str(payload["value"]))
return {"ok": True}
@app.websocket("/ws")
async def websocket_endpoint(ws: WebSocket, token: str = ""):
if token != API_TOKEN:
await ws.close(code=4401); return
await ws.accept()
pubsub = redis.pubsub()
await pubsub.subscribe("nq:events")
try:
while True:
snap = await build_snapshot(redis)
await ws.send_json({"type": "state_update", "data": snap})
try:
async with asyncio.timeout(1.0):
msg = await pubsub.get_message(ignore_subscribe_messages=True, timeout=1.0)
if msg:
await ws.send_json({"type": "trade_event", "data": json.loads(msg["data"])})
except asyncio.TimeoutError:
pass
except WebSocketDisconnect:
await pubsub.unsubscribe("nq:events")

## Strategy-side Redis publisher

# src/strategy/redis_publisher.py
import redis, json, time
class RedisPublisher:
def __init__(self, url="redis://localhost:6379"):
self.r = redis.Redis.from_url(url, decode_responses=True)
def publish_state(self, s: dict):
pipe = self.r.pipeline()
pipe.set("nq:state:bias_score", s["bias_score"])
pipe.set("nq:state:htf_bias", s["htf_bias"])
pipe.set("nq:state:active_fvg", json.dumps(s.get("active_fvg")))
pipe.set("nq:state:active_ob", json.dumps(s.get("active_ob")))
pipe.set("nq:state:last_signal_prob", s.get("last_signal_prob", 0))
pipe.set("nq:state:open_pnl", s.get("open_pnl", 0))
pipe.set("nq:state:realized_pnl_today", s.get("realized_pnl_today", 0))
pipe.set("nq:state:position", json.dumps(s.get("position")))
pipe.set("nq:health:databento_last_tick", time.time())
pipe.execute()
def publish_trade_event(self, event_type: str, trade: dict):
self.r.publish("nq:events", json.dumps({"event": event_type, "trade": trade}))
def is_trading_enabled(self) -> bool:
return self.r.get("nq:control:trading_enabled") == "true"
def subscribe_commands(self, handler):
ps = self.r.pubsub()
ps.subscribe("nq:commands")
for msg in ps.listen():
if msg["type"] == "message":
handler(msg["data"])

## NinjaScript CSV bridge (excerpt)

// VLBridge.cs (NinjaTrader 8 Strategy)
// Polls signals.csv every 2s, places bracket orders, writes status back
protected override void OnBarUpdate()
{
if ((DateTime.Now - lastCheck).TotalSeconds < 2) return;
lastCheck = DateTime.Now;
ProcessSignals();
WriteHeartbeat();
}
private void ProcessSignals()
{
if (!File.Exists(csvPath)) return;
var lines = File.ReadAllLines(csvPath);
var output = new List<string> { lines[0] };
for (int i = 1; i < lines.Length; i++) {
var f = lines[i].Split(',');
if (f[9] == "NEW") {
bool ok = PlaceBracket(f[2], int.Parse(f[4]),
double.Parse(f[5]), double.Parse(f[6]),
double.Parse(f[7]));
f[9] = ok ? "FILLED" : "REJECTED";
}
output.Add(string.Join(",", f));
}
File.WriteAllLines(csvPath, output);
}
private bool PlaceBracket(string action, int qty,
double entry, double stop, double target)
{
if (Performance.AllTrades.TradesPerformance.Currency.CumProfit < -DailyLossLimit)
return false;
if (action == "BUY") {
EnterLong(qty, "VL_LONG");
SetStopLoss("VL_LONG", CalculationMode.Price, stop, false);
SetProfitTarget("VL_LONG", CalculationMode.Price, target);
} else if (action == "SELL") {
EnterShort(qty, "VL_SHORT");
SetStopLoss("VL_SHORT", CalculationMode.Price, stop, false);
SetProfitTarget("VL_SHORT", CalculationMode.Price, target);
}
return true;
}
private void WriteHeartbeat() {
File.WriteAllText(@"C:\trading\nt8_heartbeat.txt",
DateTime.UtcNow.ToString("o"));
}

## Done when (gate to Phase 5)

- Backend handles 100 req/s, WebSocket stable
- NT8 bridge fills a test signal end-to-end (Sim101)
- Telegram bot reachable from your phone

# Phase 5 — Frontend (Weeks 6-7)

Goal: React dashboard with live chart and control panel.

## Stack rationale


| **Tech** | **Why** |
| --- | --- |
| **React 18** | **Components fit the panel-based dashboard** |
| **Vite** | **Sub-second HMR, fast prod build** |
| **TypeScript** | **Catches Redis schema drift early** |
| **Tailwind** | **Utility-first matches dark/gold tokens** |
| **Lightweight Charts v5** | **You already use this; native price line overlays** |
| **Zustand** | **Tiny global state; simpler than Redux for this** |
| **react-use-websocket** | **Auto-reconnect, queuing — saves a week of work** |


## Project init

cd ~/vl_trading
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install -D tailwindcss postcss autoprefixer
npx tailwindcss init -p
npm install lightweight-charts zustand react-use-websocket clsx tailwind-merge lucide-react
npm install -D @types/node prettier prettier-plugin-tailwindcss

## Tailwind config (dark/gold)

// tailwind.config.ts
import type { Config } from "tailwindcss";
export default {
content: ["./index.html", "./src/**/*.{ts,tsx}"],
theme: {
extend: {
colors: {
bg: { primary: "#0A0A0A", panel: "#141414", elevated: "#1F1F1F" },
gold: { DEFAULT: "#B08D2E", bright: "#E8B547", dim: "#8A6F24" },
text: { primary: "#F5F5F5", muted: "#999999", dim: "#666666" },
status: { live: "#22C55E", loss: "#EF4444", warn: "#E89B2E" },
},
fontFamily: {
sans: ["Inter", "sans-serif"],
mono: ["JetBrains Mono", "monospace"],
},
borderRadius: { panel: "8px" },
},
},
} satisfies Config;

## Component tree

<App>
├── <TopBar>              (logo, status, clock)
├── <KillSwitchBar>       (4 big buttons)
├── <Grid cols=3>
│   ├── <AccountCard>
│   ├── <PnLCard>
│   └── <PositionCard>
├── <ChartPanel>          (Lightweight Charts + FVG/OB overlays)
├── <Grid cols=2>
│   ├── <ICTStatePanel>
│   └── <BiasGauge>
├── <ParamSlidersCard>
├── <SystemHealthBar>
└── <TradeLog>

## App.tsx

// src/App.tsx
import { useLiveState } from "./hooks/useWebSocket";
import { KillSwitchBar } from "./components/KillSwitchBar";
import { AccountCard } from "./components/AccountCard";
import { PnLCard } from "./components/PnLCard";
import { PositionCard } from "./components/PositionCard";
import { ChartPanel } from "./components/ChartPanel";
import { ICTStatePanel } from "./components/ICTStatePanel";
import { BiasGauge } from "./components/BiasGauge";
import { ParamSlidersCard } from "./components/ParamSlidersCard";
import { SystemHealthBar } from "./components/SystemHealthBar";
import { TradeLog } from "./components/TradeLog";
export default function App() {
const { connected } = useLiveState();
return (
<div className="min-h-screen bg-bg-primary text-text-primary font-sans p-6">
<header className="flex items-center justify-between mb-6">
<h1 className="text-3xl font-bold tracking-wide">
<span className="text-gold">VL</span> AGENT COMMAND
</h1>
<div className="flex items-center gap-3 text-sm">
<span className={connected ? "text-status-live" : "text-status-loss"}>
● {connected ? "LIVE" : "DISCONNECTED"}
</span>
<span className="text-text-muted font-mono">
{new Date().toLocaleTimeString()} CT
</span>
</div>
</header>
<KillSwitchBar />
<div className="grid grid-cols-3 gap-4 mb-4">
<AccountCard />
<PnLCard />
<PositionCard />
</div>
<div className="mb-4"><ChartPanel /></div>
<div className="grid grid-cols-2 gap-4 mb-4">
<ICTStatePanel />
<BiasGauge />
</div>
<ParamSlidersCard />
<SystemHealthBar />
<TradeLog />
</div>
);
}

## useWebSocket hook

// src/hooks/useWebSocket.ts
import useWebSocketRaw, { ReadyState } from "react-use-websocket";
import { useEffect } from "react";
import { useStore } from "@/store";
const WS_URL = import.meta.env.VITE_WS_URL || "ws://localhost:8000/ws";
export function useLiveState() {
const setState = useStore(s => s.setLiveState);
const { lastMessage, readyState } = useWebSocketRaw(WS_URL, {
shouldReconnect: () => true,
reconnectAttempts: 999,
reconnectInterval: 2000,
});
useEffect(() => {
if (lastMessage) {
const data = JSON.parse(lastMessage.data);
if (data.type === "state_update") setState(data.data);
}
}, [lastMessage]);
return { connected: readyState === ReadyState.OPEN };
}

## Zustand store

// src/store/index.ts
import { create } from "zustand";
interface LiveState {
account: { balance: number; dailyLimit: number; dailyUsed: number; trailingDD: number };
pnl: { realized: number; unrealized: number; tradesCount: number; wins: number; losses: number };
position: { side: "LONG" | "SHORT"; qty: number; avgPx: number; sl: number; tp: number } | null;
bias: { score: number; htf: "BULLISH" | "BEARISH" | "NEUTRAL"; xgbProb: number; lockedUntil?: string };
ict: {
fvg?: { top: number; bottom: number; side: "BULL" | "BEAR" };
ob?:  { top: number; bottom: number; side: "BULL" | "BEAR"; vol: number };
lastBos?: { time: string; side: "BULL" | "BEAR" };
session: "ASIA" | "LONDON" | "NY_AM" | "NY_PM" | "OFF";
};
health: { databento: boolean; nt8: boolean; redis: boolean; xgbVersion?: string };
}
interface Store extends LiveState {
setLiveState: (s: Partial<LiveState>) => void;
}
export const useStore = create<Store>((set) => ({
account: { balance: 0, dailyLimit: 0, dailyUsed: 0, trailingDD: 0 },
pnl: { realized: 0, unrealized: 0, tradesCount: 0, wins: 0, losses: 0 },
position: null,
bias: { score: 0, htf: "NEUTRAL", xgbProb: 0 },
ict: { session: "OFF" },
health: { databento: false, nt8: false, redis: false },
setLiveState: (s) => set((prev) => ({ ...prev, ...s })),
}));

## KillSwitchBar.tsx

// src/components/KillSwitchBar.tsx
import { useState } from "react";
import { Play, Pause, Square, AlertTriangle } from "lucide-react";
import { api } from "@/api/client";
import { clsx } from "clsx";
export function KillSwitchBar() {
const [confirmFlatten, setConfirmFlatten] = useState(false);
const [loading, setLoading] = useState<string | null>(null);
const handle = async (action: string, fn: () => Promise<any>) => {
setLoading(action);
try { await fn(); } finally { setLoading(null); }
};
return (
<div className="bg-bg-panel rounded-panel p-4 mb-4 border border-bg-elevated">
<div className="flex gap-3">
<Btn onClick={() => handle("start", api.control.start)} variant="success" icon={<Play size={18}/>}>START</Btn>
<Btn onClick={() => handle("pause", api.control.pause)} variant="warn" icon={<Pause size={18}/>}>PAUSE</Btn>
<Btn onClick={() => handle("stop", api.control.stop)} variant="danger-outline" icon={<Square size={18}/>}>STOP</Btn>
<Btn onClick={() => setConfirmFlatten(true)} variant="danger" icon={<AlertTriangle size={18}/>}>FLATTEN ALL</Btn>
</div>
{confirmFlatten && (
<ConfirmModal
title="Flatten all positions?"
msg="Closes all open NQ at market."
onConfirm={() => { handle("flatten", api.control.flatten); setConfirmFlatten(false); }}
onCancel={() => setConfirmFlatten(false)} />
)}
</div>
);
}
function Btn({ children, onClick, variant, icon }: any) {
return (
<button onClick={onClick} className={clsx(
"flex-1 flex items-center justify-center gap-2 py-3 rounded font-bold tracking-wide transition",
variant === "success" && "bg-status-live/20 text-status-live hover:bg-status-live/30",
variant === "warn"    && "bg-status-warn/20 text-status-warn hover:bg-status-warn/30",
variant === "danger-outline" && "border border-status-loss text-status-loss hover:bg-status-loss/10",
variant === "danger"  && "bg-status-loss text-white hover:bg-status-loss/80",
)}>{icon}{children}</button>
);
}

## ChartPanel.tsx (live chart + ICT overlays)

// src/components/ChartPanel.tsx
import { useEffect, useRef } from "react";
import { createChart, ColorType, LineStyle, IChartApi, ISeriesApi } from "lightweight-charts";
import { useStore } from "@/store";
import { api } from "@/api/client";
const VL_DARK = "#0A0A0A";
const VL_GOLD = "#B08D2E";
const VL_GREEN = "#22C55E";
const VL_RED = "#EF4444";
export function ChartPanel() {
const ref = useRef<HTMLDivElement>(null);
const chart = useRef<IChartApi | null>(null);
const candles = useRef<ISeriesApi<"Candlestick"> | null>(null);
const fvgLines = useRef<any[]>([]);
const obLines = useRef<any[]>([]);
const ict = useStore(s => s.ict);
useEffect(() => {
if (!ref.current) return;
chart.current = createChart(ref.current, {
layout: { background: { type: ColorType.Solid, color: VL_DARK }, textColor: "#F5F5F5" },
grid: { vertLines: { color: "#1F1F1F" }, horzLines: { color: "#1F1F1F" } },
timeScale: { timeVisible: true, secondsVisible: false, borderColor: "#333" },
rightPriceScale: { borderColor: "#333" },
watermark: { visible: true, color: "#666", text: "VL · NQ 06-26 · 5m",
fontSize: 14, horzAlign: "right", vertAlign: "bottom" },
crosshair: { mode: 1 },
});
candles.current = chart.current.addCandlestickSeries({
upColor: VL_GREEN, downColor: VL_RED,
borderUpColor: VL_GREEN, borderDownColor: VL_RED,
wickUpColor: VL_GREEN, wickDownColor: VL_RED,
});
api.chart.bars("5m", 300).then((bars: any) => candles.current!.setData(bars));
const ro = new ResizeObserver(() => {
chart.current!.applyOptions({ width: ref.current!.clientWidth });
});
ro.observe(ref.current);
return () => { ro.disconnect(); chart.current?.remove(); };
}, []);
useEffect(() => {
if (!candles.current) return;
fvgLines.current.forEach(l => candles.current!.removePriceLine(l));
obLines.current.forEach(l => candles.current!.removePriceLine(l));
fvgLines.current = []; obLines.current = [];
if (ict.fvg) {
fvgLines.current.push(candles.current.createPriceLine({
price: ict.fvg.top, color: VL_GOLD, lineStyle: LineStyle.Dashed,
lineWidth: 1, axisLabelVisible: true, title: "FVG↑" }));
fvgLines.current.push(candles.current.createPriceLine({
price: ict.fvg.bottom, color: VL_GOLD, lineStyle: LineStyle.Dashed,
lineWidth: 1, axisLabelVisible: true, title: "FVG↓" }));
}
if (ict.ob) {
obLines.current.push(candles.current.createPriceLine({
price: ict.ob.top, color: VL_GOLD, lineStyle: LineStyle.Solid,
lineWidth: 2, title: "OB↑" }));
obLines.current.push(candles.current.createPriceLine({
price: ict.ob.bottom, color: VL_GOLD, lineStyle: LineStyle.Solid,
lineWidth: 2, title: "OB↓" }));
}
}, [ict.fvg, ict.ob]);
return <div ref={ref} className="h-[500px] w-full rounded-panel bg-bg-panel" />;
}

## API client

// src/api/client.ts
const API = import.meta.env.VITE_API_URL || "http://localhost:8000";
const TOKEN = import.meta.env.VITE_API_TOKEN;
async function req<T>(method: string, path: string, body?: any): Promise<T> {
const r = await fetch(`${API}${path}`, {
method,
headers: { "Content-Type": "application/json", "Authorization": `Bearer ${TOKEN}` },
body: body ? JSON.stringify(body) : undefined,
});
if (!r.ok) throw new Error(`${r.status} ${r.statusText}`);
return r.json();
}
export const api = {
control: {
start:   () => req("POST", "/api/control/start"),
pause:   () => req("POST", "/api/control/pause"),
stop:    () => req("POST", "/api/control/stop"),
flatten: () => req("POST", "/api/control/flatten"),
},
params: {
get: () => req("GET", "/api/params"),
set: (key: string, val: any) => req("PUT", `/api/params/${key}`, { value: val }),
},
state: { snapshot: () => req("GET", "/api/state") },
trades: { today: () => req("GET", "/api/trades/today") },
system: { health: () => req("GET", "/api/system/health") },
chart: { bars: (tf: string, n: number) => req("GET", `/api/chart/bars?tf=${tf}&n=${n}`) },
};

## Done when (gate to Phase 6)

- All components render with live WebSocket data
- Killswitch buttons work and trigger strategy actions
- Chart shows live candles + active FVG/OB overlays
- Reachable from your phone via Tailscale

# Phase 6 — Paper → Live (Weeks 8-9)

Goal: 2 weeks paper trading, then funded account with hard cap.

## Tasks

- Run live Databento stream into NautilusTrader paper mode
- NT8 in simulation mode (Sim101 account)
- Compare paper fills to backtest expected fills daily
- Identify slippage / latency / timing issues, fix
- Day-1 live: 1 contract cap, regardless of risk %
- Week 1 live: 1-2 contracts
- Promote risk only after 50+ live trades with positive expectancy

## Pre-live checklist (must pass all)

- ≥ 2 weeks paper traded, realized PnL within 20% of backtest expectation
- All 8 risk limits tested by deliberate breach in paper
- Killswitch tested from UI, Telegram, and SSH
- NT8 bridge survives a forced NinjaTrader restart mid-trade
- Strategy survives a Databento disconnect + reconnect
- Daily loss alert at 50% triggers correctly
- Telegram bot reachable from your phone (test on cellular)
- Prop firm rules YAML matches written firm rules verbatim
- First-day cap: max 1 contract

# Phase 7 — ML Meta-Label (Week 10+, OPTIONAL)

Only enter if Phase 6 shows positive expectancy with enough trades.

## Skip Phase 7 if

- Rules already profitable — don't fix what isn't broken
- < 5 trades/day — not enough samples for meta-labeling
- Win rate already > 60% — XGBoost won't move it

## Tasks (if proceeding)

- uv pip install mlfinpy xgboost optuna in ml_env
- Build label_pipeline.py: triple-barrier on each historical ICT setup
- Barriers: pt = stop * R/R (2.5:1), sl = stop, max hold 30 bars (5m)
- CUSUM filter for event sampling (~0.5 * ATR threshold)
- Train XGBoost with PurgedKFold (5 splits, 1-day embargo)
- Hyperparameter search via Optuna
- Compare gated vs raw strategy in NautilusTrader backtest
- Accept only if: gated Sharpe > raw Sharpe AND gated DD ≤ raw DD
- Save versioned model, strategy loads at startup

## Retrain schedule

- Monthly retrain on rolling 12-month window (recommended start)
- Manual first, automated via cron after stable
- Archive old models, never delete (audit trail)

# Risk Management


## Hard limits in code


| **Limit** | **Where** | **Action on breach** |
| --- | --- | --- |
| **Daily loss limit** | **Risk engine + NT8** | **Flatten, halt until next day** |
| **Max position size** | **Risk engine** | **Reject order** |
| **Max contracts open** | **Strategy** | **Skip new entries** |
| **Trading window** | **Strategy** | **No new entries outside** |
| **News blackout** | **Strategy + calendar** | **Pause 5min before/after** |
| **Data stale > 5s** | **Heartbeat check** | **Flatten + alert** |
| **NT8 bridge stale > 10s** | **Redis heartbeat** | **Critical alert, halt** |
| **Consecutive losses N** | **Strategy counter** | **30-min cooldown** |


## Prop firm config

- One YAML per firm: bulenox.yaml, mffu.yaml, apex.yaml
- All rules read from config — never hardcode
- Bulenox: 2-3% daily loss, trailing DD by plan
- MFFU Pro: algo-friendly, EOD trailing DD
- Apex: stricter consistency — no single big-win days

## Operational rules

- Never live without ≥ 2 weeks paper on identical code
- Never deploy new model on Monday morning
- Never disable killswitch, even briefly
- Always have Telegram alerts on phone when live
- Always log every signal, fill, param change (immutable audit)

# Folder Layout

~/vl_trading/
├── .venvs/                          # uv-managed envs
│   ├── nautilus_env/
│   ├── ml_env/
│   ├── agent_env/
│   ├── control_env/
│   └── iqfeed_env/
├── locks/                           # frozen requirements
│   ├── nautilus_env.lock
│   ├── ml_env.lock
│   └── control_env.lock
├── data/
│   ├── raw/{iqfeed_parquet, databento_dbn}
│   ├── features/NQ_5m_features.parquet
│   └── catalog/                     # NautilusTrader catalog
├── src/
│   ├── data_pipeline/
│   │   ├── iqfeed_pull.py
│   │   ├── databento_stream.py
│   │   ├── feature_pipeline.py
│   │   └── nautilus_loader.py
│   ├── strategy/
│   │   ├── nq_ict_strategy.py
│   │   ├── ict_setups.py
│   │   ├── bias_scoring.py
│   │   ├── smc_buffer.py
│   │   ├── param_loader.py
│   │   └── redis_publisher.py
│   ├── execution/
│   │   ├── csv_writer.py
│   │   └── nt8_bridge/{VLBridge.cs, README.md}
│   ├── ml/
│   │   ├── label_pipeline.py
│   │   ├── train_xgb.py
│   │   └── inference.py
│   ├── agents/premarket_bias.py
│   ├── backend/
│   │   ├── main.py
│   │   ├── routes/{control.py, state.py, trades.py, system.py}
│   │   ├── websocket.py
│   │   └── redis_client.py
│   └── control/
│       ├── telegram_bot.py
│       └── alert_router.py
├── frontend/
│   ├── index.html
│   ├── package.json
│   ├── tailwind.config.ts
│   └── src/
│       ├── App.tsx
│       ├── components/
│       │   ├── KillSwitchBar.tsx
│       │   ├── AccountCard.tsx
│       │   ├── PnLCard.tsx
│       │   ├── PositionCard.tsx
│       │   ├── ChartPanel.tsx
│       │   ├── ICTStatePanel.tsx
│       │   ├── BiasGauge.tsx
│       │   ├── ParamSlidersCard.tsx
│       │   ├── SystemHealthBar.tsx
│       │   └── TradeLog.tsx
│       ├── hooks/useWebSocket.ts
│       ├── api/client.ts
│       └── store/index.ts
├── configs/
│   ├── strategy.yaml
│   ├── bulenox.yaml
│   ├── mffu.yaml
│   └── instruments/{nq_06_26.yaml, mnq_06_26.yaml}
├── models/
│   └── xgb_v2026_05_18.json
├── logs/
│   ├── strategy/
│   ├── trades/
│   └── signals_csv_archive/
├── deploy/
│   ├── systemd/
│   │   ├── vl-nautilus.service
│   │   ├── vl-backend.service
│   │   └── vl-telegram.service
│   └── nginx/vl.conf
└── scripts/
├── start_live.sh
├── start_paper.sh
├── start_backend.sh
└── kill_all.sh

# Deployment


## Process layout


| **Process** | **Host** | **Manager** | **Port** |
| --- | --- | --- | --- |
| **redis-server** | **WSL2** | **systemd** | **:6379** |
| **nautilus_strategy** | **WSL2** | **systemd** | **—** |
| **fastapi backend** | **WSL2** | **systemd + uvicorn** | **:8000** |
| **frontend (prod)** | **WSL2** | **nginx serves /dist** | **:80** |
| **telegram_bot** | **WSL2** | **systemd** | **—** |
| **ninjatrader 8** | **Windows** | **Auto-start on logon** | **desktop** |
| **nginx** | **WSL2** | **systemd** | **:80/:443** |


## systemd unit (vl-nautilus)

# /etc/systemd/system/vl-nautilus.service
[Unit]
Description=VL NautilusTrader live strategy
After=network.target redis-server.service
Requires=redis-server.service
[Service]
Type=simple
User=hoang
WorkingDirectory=/home/hoang/vl_trading
Environment="VL_ENV=live"
Environment="REDIS_URL=redis://localhost:6379"
ExecStart=/home/hoang/vl_trading/.venvs/nautilus_env/bin/python \
-m src.strategy.run_live --config configs/strategy.yaml
Restart=on-failure
RestartSec=10s
StandardOutput=append:/home/hoang/vl_trading/logs/strategy/strategy.log
StandardError=append:/home/hoang/vl_trading/logs/strategy/strategy.err
[Install]
WantedBy=multi-user.target

## systemd unit (vl-backend)

# /etc/systemd/system/vl-backend.service
[Unit]
Description=VL FastAPI backend
After=network.target redis-server.service
[Service]
Type=simple
User=hoang
WorkingDirectory=/home/hoang/vl_trading
EnvironmentFile=/home/hoang/vl_trading/.env
ExecStart=/home/hoang/vl_trading/.venvs/control_env/bin/uvicorn \
src.backend.main:app --host 0.0.0.0 --port 8000
Restart=on-failure
[Install]
WantedBy=multi-user.target

## nginx reverse proxy

# /etc/nginx/sites-available/vl.conf
server {
listen 80;
server_name vl.local;
root /home/hoang/vl_trading/frontend/dist;
index index.html;
try_files $uri $uri/ /index.html;
location /api/ {
proxy_pass http://127.0.0.1:8000;
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
}
location /ws {
proxy_pass http://127.0.0.1:8000;
proxy_http_version 1.1;
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "upgrade";
proxy_read_timeout 86400;
}
}

## Build & deploy

- cd frontend && npm run build → outputs frontend/dist/
- sudo systemctl reload nginx
- sudo systemctl restart vl-backend
- sudo systemctl restart vl-nautilus  (off-session only!)
- Verify: curl http://localhost:8000/api/system/health

## Remote access

- Tailscale on WSL2 + phone: zero-config private VPN, free
- https://your-laptop.tailnet/ from phone — full dashboard
- Telegram bot covers remote control on cellular
- Never expose backend to public internet without TLS + proper auth
*— End of Build Plan —*
