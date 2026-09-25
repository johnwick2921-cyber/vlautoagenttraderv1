**VL TRADING SYSTEM**
Final Build Plan
*v5 · Qlib + RD-Agent Architecture*
AI-Powered Multi-Timeframe Trading System
Microsoft Qlib · RD-Agent · NautilusTrader · NinjaTrader 8
7 Timeframes × 5 Models × LLM-Driven Discovery
Funded Future Trading · VL Intelligent · May 2026

# Contents

**Part I — System Foundation**
**  1. Executive Summary**
**  2. Stack & Repositories**
**  3. System Architecture**
**  4. Data Flow & Process Topology**
**Part II — Dashboard Specification**
**  5. Design Tokens (Colors, Fonts, Spacing)**
**  6. Page Layout & Sidebar Navigation**
**  7. Operations Section**
**  8. Daily Summary & News Calendar**
**  9. Strategy Management Section**
**  10. Build & Test Section**
**  11. Live Trading Section**
**  12. Risk & Models Section**
**  13. System & Logs Section**
**  14. Trade Journal Page**
**  15. Models Page (full registry view)**
**Part III — Backend & Data Contract**
**  16. Redis Schema**
**  17. FastAPI Endpoints**
**  18. WebSocket Protocol**
**  19. NinjaTrader Bridge**
**  20. Qlib + RD-Agent Integration**
**Part IV — Phases & Build Order**
**  21. 7-Phase Build Plan**
**  22. Acceptance Criteria per Phase**
**  23. Code Templates**
**Part V — Deployment**
**  24. Folder Layout**
**  25. Process Management (systemd)**
**  26. Reverse Proxy (nginx)**
**  27. Remote Access (Tailscale)**
**  28. Scheduler (cron + automation)**
**Appendices**
**  A. Alpha158 / Alpha360 Factor Reference**
**  B. Qlib Model Registry (5 algos × 7 timeframes)**
**  C. Risk Math**
**  D. Data Sources & API Keys**
**  E. NinjaTrader Integration Resolution**
**  F. First Qlib Workflow Build Plan**
**  G. Default Configuration Files**
**  H. Glossary**
**  I. Qlib Quickstart (7 timeframes)**
**  J. RD-Agent Quickstart (GPT-4o-mini)**
**  K. Ensemble Strategy Specification**

# Part I — System Foundation


## 1. Executive Summary

Goal: AI-driven automated NQ/MNQ futures trading system. Instead of hand-coding ICT strategies, the system uses Microsoft Qlib's prebuilt ML models trained on Alpha158/Alpha360 factor libraries, with Microsoft RD-Agent (LLM-powered research agent) autonomously discovering new factors and refining models. NautilusTrader executes signals via a NinjaTrader 8 bridge. The full React-based VL Agent Command dashboard provides control, monitoring, and journaling.

### What this system does

- Streams live NQ/MNQ price data from NinjaTrader 8 to Python via a TCP socket bridge
- Computes Qlib Alpha158/Alpha360 features (158 to 360 engineered factors) across 7 timeframes: 1m, 2m, 3m, 5m, 1h, 1d, 1w
- Trains 5 prebuilt ML models per timeframe (LightGBM, XGBoost, CatBoost, LSTM, Transformer) — 35 models total
- Ensembles predictions across models and timeframes into a single trade signal per bar
- NautilusTrader strategy consumes the ensemble signal, applies risk rules, and submits bracket orders to NT8
- RD-Agent runs nightly (LLM: GPT-4o-mini) to propose new factors, test them, and merge winners into the model pool
- All 35 models retrain every Sunday night on the most recent data
- Provides a full web dashboard for control, monitoring, model management, backtest, paper trade, and trade journaling

### What this system explicitly does NOT include (for now)

- Hand-coded ICT setups — Qlib's prebuilt models and Alpha158 factors replace this entirely
- Custom 8-factor bias scorer — RD-Agent discovers factors autonomously; ensemble model output replaces the bias score
- smart-money-concepts library — Qlib features cover momentum, volatility, volume, and structural signals natively (RD-Agent can add ICT-like factors if it finds them profitable)
- Telegram, email, or external communication (deferred — in-app UI notifications only)
- Prop-firm-specific rule templates (deferred — generic risk rules only)
- Tax export and mobile-responsive view (deferred)

### Why this architecture (vs hand-coding strategies)

- Speed: Qlib's prebuilt models work out-of-the-box. No 8-factor formulas to debug. No setup catalog to refine over months.
- Coverage: Alpha158 captures 158 engineered factors — more than any single human would code by hand
- Continuous improvement: RD-Agent autonomously discovers new factors while you sleep. Per the NeurIPS 2025 paper, ~2× returns over the Alpha158 baseline at <$10 in LLM costs per run (CSI300, not NQ — treat as best case)
- Multi-timeframe: 7 timeframes capture different market structures (1m scalping → 1w macro)
- Ensemble diversity: 5 model architectures reduce overfitting risk vs any single model
- Auditable: every prediction traces back to a specific model + timeframe + factor set

## 2. Stack & Repositories

The full system requires 12 core open-source installs. Optional Phase 7 ML and observability layers add a few more.

### Core install list (14)


| **#** | **Name** | **Repo** | **Role** |
| --- | --- | --- | --- |
| **1** | **Qlib** | **microsoft/qlib** | **ML platform · 23 prebuilt models · Alpha158/360 factors** |
| **2** | **RD-Agent** | **microsoft/RD-Agent** | **LLM-driven factor + model discovery** |
| **3** | **LightGBM** | **microsoft/LightGBM** | **Fast tree model (default Qlib baseline)** |
| **4** | **XGBoost** | **dmlc/xgboost** | **Tree model for ensemble diversity** |
| **5** | **CatBoost** | **catboost/catboost** | **Tree model with categorical handling** |
| **6** | **PyTorch** | **pytorch/pytorch** | **Backend for LSTM + Transformer models** |
| **7** | **NautilusTrader** | **nautechsystems/nautilus_trader** | **Execution engine (consumes Qlib predictions)** |
| **8** | **ninja-socket** | **mattalford/ninja-socket** | **NT8 → Python data stream** |
| **9** | **VLBridge.cs** | **(custom in this repo)** | **Python → NT8 order receiver** |
| **10** | **Redis** | **redis/redis** | **State bus (KV + pub/sub)** |
| **11** | **FastAPI** | **fastapi/fastapi** | **Backend REST + WebSocket** |
| **12** | **React + Vite + Tailwind** | **facebook/react + vitejs/vite** | **Frontend (your VL Agent Command UI)** |
| **13** | **Lightweight Charts v5** | **tradingview/lightweight-charts** | **Live chart with model overlays** |
| **14** | **uv** | **astral-sh/uv** | **Python env / lockfile manager** |


### LLM provider for RD-Agent


| **Provider** | **Model** | **Cost / run** |
| --- | --- | --- |
| **OpenAI (default)** | **gpt-4o-mini** | **$1-5 per discovery loop · start here** |
| **OpenAI (upgrade)** | **gpt-4o** | **$10-40 · move up if mini produces weak factors** |
| **Anthropic (alt)** | **claude-sonnet-4-5** | **$15-50 · best reasoning quality** |
| **Local (free)** | **ollama/deepseek-r1:70b** | **Free but slow + lower quality** |


## 3. System Architecture

Everything runs on a single workstation: NinjaTrader 8 on the Windows side, the rest of the stack on WSL2 Ubuntu. Communication between the two sides uses TCP sockets (data stream) and a shared filesystem (CSV orders).

### End-to-end flow

NinjaTrader 8 (Windows) ──────────────────────────────────────────
│
├─ ninja-socket add-on ──► sends NQ tick/bar data
│                                  │
│                                  ▼
│                          Python bridge process
│                          (writes ticks to Redis +
│                           Parquet per timeframe)
│                                  │
│                                  ▼
│                          Qlib data layer (7 timeframes)
│                          ~/.qlib/qlib_data/nq_{tf}/
│                          tf in {1m, 2m, 3m, 5m, 1h, 1d, 1w}
│                                  │
│                  ┌───────────────┼───────────────┐
│                  ▼               ▼               ▼
│           LightGBM model    XGBoost model    LSTM/Transformer
│           (×7 timeframes)   (×7 timeframes)  (×7 timeframes)
│                  │               │               │
│                  └───────────────┼───────────────┘
│                                  ▼
│                          Ensemble (weighted mean / stacker)
│                                  │
│                                  ▼
│                          NautilusTrader strategy
│                          (reads ensemble pred, applies risk,
│                           submits bracket order)
│                                  │
│                                  ▼
│                          signals.csv (on shared folder)
│                                  │
└─ VLBridge.cs ◄───────────────── reads CSV
│
▼
Bracket order in NT8 ──► Rithmic/CQG ──► account
Independent automation services (cron-scheduled):
─────────────────────────────────────────────────
Sun 02:00 CT  ──► vl-qlib-retrain    : retrain all 35 models on latest data
Nightly 23:00 ──► vl-rdagent-discover: RD-Agent proposes & tests new factors
(LLM: GPT-4o-mini, ~$1-5 per run)
FastAPI backend ◄─► Redis ─► React dashboard (VL Agent Command)
│
├─► Dashboard pages: Dashboard, Strategies, Models (35+ row table),
│   Backtest, Paper Trade, Replay, Trade Journal, etc.
└─► Live tail of Qlib + RD-Agent training jobs

### Process topology

WSL2 Ubuntu                                Windows 10 native
──────────────                              ────────────────
systemd-managed (long-running):             NT8 desktop app
├─ redis-server     :6379                 ├─ NQ 06-26 chart
├─ vl-bridge        (NT8 socket → Redis)  ├─ ninja-socket add-on
├─ vl-nautilus      (ensemble strategy)   ├─ VLBridge.cs add-on
├─ vl-backend       :8000 (FastAPI)       └─ Rithmic / CQG feed
├─ vl-journal       (screenshot service)         │
└─ nginx            :80 (reverse proxy)          ▼
Account
cron-scheduled (periodic):
├─ vl-qlib-retrain     Sun 02:00 CT
│                      → trains all 35 models
│                      → publishes vl:models:registry
└─ vl-rdagent-discover Daily 23:00 CT
→ runs RD-Agent loop
→ proposes new factors
→ publishes vl:rdagent:trace
Frontend (Vite production build) served by nginx
from /home/user/vl/frontend/dist

### Environment isolation (uv-managed)


| **Env** | **Packages** | **Lockfile?** |
| --- | --- | --- |
| **qlib_env** | **pyqlib, lightgbm, xgboost, catboost, torch, pandas, pyarrow, redis** | **Yes — freeze for retrains** |
| **rdagent_env** | **rdagent, openai, anthropic, pyqlib (reads same data), pydantic** | **Yes** |
| **nautilus_env** | **nautilus_trader, pandas, pyarrow, redis (consumes Qlib predictions)** | **Yes — freeze for live** |
| **control_env** | **fastapi, uvicorn, redis, pydantic, loguru** | **Yes** |
| **bridge_env** | **websockets, redis, pyarrow** | **Yes** |
| **frontend (node)** | **react, vite, typescript, tailwindcss, lightweight-charts, zustand** | **package-lock.json** |


## 4. Data Flow & Communication


### 4.1 Data lane (NT8 → Python)

ninja-socket NinjaScript add-on opens a TCP WebSocket server on port 9001. The Python bridge connects and receives every tick/bar as JSON.
// NT8 → Python message format (JSON over WebSocket)
{
"type": "tick",
"ts": 1747754533123,
"symbol": "NQ 06-26",
"bid": 21492.00,
"ask": 21492.50,
"last": 21492.25,
"volume": 3
}
{
"type": "bar",
"ts": 1747754400000,
"symbol": "NQ 06-26",
"tf": "5m",
"open": 21485.50,
"high": 21495.00,
"low": 21484.25,
"close": 21492.25,
"volume": 2143
}

### 4.2 Order lane (Python → NT8)

Python writes a row to signals.csv on a shared folder. Claude-Trader NinjaScript polls the file every 2 seconds and places a bracket order on NT8 for each NEW row.
# signals.csv format
timestamp,signal_id,action,symbol,qty,entry,stop,target,reason,status
2026-05-20T14:38:22Z,sig_0142,BUY,NQ 06-26,2,21487.50,21462.50,21547.50,FVG_LONG,NEW
# After NT8 processes the row, it rewrites status:
# NEW → FILLED   (order placed and filled)
# NEW → REJECTED (risk check failed or no buying power)
# NEW → PARTIAL  (partial fill)

### 4.3 State lane (everywhere ↔ Redis)

All live state lives in Redis. The strategy writes, the backend reads, the UI subscribes via WebSocket. Full schema in Part III §14.

# Part II — Dashboard Specification

This part captures the locked design from the v3 mockup. Every panel, every color, every measurement. Build the React app to match these specs.

## 5. Design Tokens


### 5.1 Colors


| **Token** | **Hex** | **Use** |
| --- | --- | --- |
| **bg.primary** | **#0A0A0A** | **Page background (true black)** |
| **bg.panel** | **#141414** | **Card surfaces** |
| **bg.elevated** | **#1F1F1F** | **Modals, sliders track** |
| **bg.sidebar** | **#0D0D0D** | **Left navigation column** |
| **bg.input** | **#0F0F0F** | **Inputs, code blocks** |
| **gold.primary** | **#B08D2E** | **Brand accent (deeper)** |
| **gold.bright** | **#E8B547** | **Hover, active state, headlines** |
| **gold.dim** | **#8A6F24** | **Disabled state** |
| **text.primary** | **#F5F5F5** | **Default text** |
| **text.muted** | **#999999** | **Secondary labels** |
| **text.dim** | **#666666** | **Tertiary, build info** |
| **status.live** | **#22C55E** | **Profit, connected, long** |
| **status.loss** | **#EF4444** | **Loss, disconnect, short** |
| **status.warn** | **#E89B2E** | **Approaching limit, paper mode** |
| **status.test** | **#60A5FA** | **Backtest, info logs** |
| **border** | **rgba(232,181,71,0.15)** | **Default panel border** |
| **border.strong** | **rgba(232,181,71,0.30)** | **Active/selected panel** |


### 5.2 Typography


| **Token** | **Font** | **Use** |
| --- | --- | --- |
| **font.sans** | **Inter (300, 400, 500, 700)** | **All UI text** |
| **font.mono** | **JetBrains Mono (400, 500, 700)** | **Prices, numbers, IDs, code, timestamps** |


### 5.3 Type scale


| **Role** | **Size / weight** | **Example** |
| --- | --- | --- |
| **Page title** | **19px / 500** | **"Dashboard", "Trade Journal"** |
| **Card label** | **9px / 500 · 2px letter-spacing · uppercase · muted** | **"ACCOUNT", "TODAY P&L", "ICT STATE"** |
| **Big metric** | **22-30px / 500 mono** | **"$148,250", "+$600", "+32"** |
| **Body / row** | **11-12px / 400** | **Table rows, lists** |
| **Small / dim** | **9-10px / 400 dim** | **Timestamps, helper text** |
| **Code** | **11px mono** | **Editor, diff, system logs** |


### 5.4 Spacing & radius

- Page padding: 18-22px horizontal, 16-20px vertical
- Card padding: 16px (most), 11-14px header (cards with header bars)
- Card border-radius: 10px
- Button border-radius: 6px
- Card gap (between cards in a row): 10-12px
- Section gap (between major sections): 10-14px
- Pill border-radius: 4px
- Slider thumb: 14px circle

### 5.5 Component tokens

Reusable component variants used across the dashboard:

| **Component** | **Variants** |
| --- | --- |
| **Button** | **default · gold · green · amber · red · red-outline · blue** |
| **Pill (status)** | **live (green) · paper (gold) · test (blue) · off (grey) · edit (amber) · warn (red) · hi/med/lo (news impact)** |
| **Toggle** | **on (gold) · off (grey)** |
| **Slider** | **gold thumb, 3px track, hover halo** |
| **Input** | **bg.input + gold-on-focus border + monospace text** |
| **Mini progress bar** | **4px height, gold gradient (default), amber/red gradient when warn/danger** |


## 6. Page Layout & Sidebar Navigation


### 6.1 Two-column layout

Fixed-width left sidebar (200px) + flexible main content area. Sidebar stays visible on all pages. Main content scrolls.
┌─────────┬──────────────────────────────────────────────────┐
│ SIDEBAR │  MAIN CONTENT                                    │
│ 200px   │  flex: 1                                         │
│         │  padding: 16px 20px                              │
│ #0D0D0D │  bg: #0A0A0A                                     │
│         │                                                  │
│         │  ┌─ Top bar ─────────────────────────────────┐   │
│         │  │ Page title          [Search][Export][+]   │   │
│         │  └───────────────────────────────────────────┘   │
│         │                                                  │
│         │  ... section cards stack vertically ...          │
│         │                                                  │
└─────────┴──────────────────────────────────────────────────┘

### 6.2 Sidebar — exact nav structure


| **Section** | **Item** | **Badge / notes** |
| --- | --- | --- |
| **LOGO** | **VL · AGENT COMMAND v3** | **Gold gradient mark, bordered bottom** |
| **OPERATIONS** | **Dashboard** | **(active state shown by default)** |
|   | **Strategies** | **Badge: count (4)** |
|   | **Accounts** | **Badge: count (3)** |
|   | **Notifications** | **Badge: unread count, red when >0** |
| **BUILD & TEST** | **Editor** | **Opens last-edited strategy** |
|   | **Backtest** | **Full-page backtest runner** |
|   | **Paper Trade** | **Full-page paper trade view** |
|   | **Replay** | **Historical replay tool** |
|   | **Models** | **Full-page model management** |
| **RECORDS** | **Trade Log** | **Full trade history** |
|   | **Trade Journal** | **Badge: total journaled count (142). Highlighted nav.** |
|   | **System Logs** | **Full-page log viewer** |
| **SYSTEM** | **Risk Rules** | **Full-page risk config** |
|   | **Connections** | **Full-page connection health** |
|   | **Settings** | **Account / theme / shortcuts** |
| **FOOTER** | **System Status card** | **Green pulse + uptime + CT clock** |
|   | **EMERGENCY STOP button** | **Full-width red. Confirmation modal required.** |


## 7. Operations Section (top of dashboard)


### 7.1 Top Bar

Page title + subtitle showing active strategy/mode/account + global action buttons.
- Title: 19px / weight 500 / 1px letter-spacing
- Subtitle: 10px text.dim. Format: "Active: <strategy> · <mode> · <account>"
- Right side: 3-4 buttons (Search, Export, + New Strategy gold)

### 7.2 Accounts Bar

Multi-account row at top. 3 cards (live, paper, paused) in grid.

| **Element** | **Spec** |
| --- | --- |
| **Label** | **⌬ Accounts (N) + "+ Add" button right-aligned** |
| **Grid** | **repeat(3, 1fr), gap 7px** |
| **Active card** | **bg gold-tinted, border gold strong, ★ prefix, active label gold-bright** |
| **Inactive card** | **bg.input, border subtle** |
| **Card content** | **Account name + status pill (LIVE / PAPER / PAUSED), balance (13px mono), today's P&L** |
| **Click action** | **Switches active account context across entire dashboard** |


### 7.3 Kill Switch Bar

5 buttons in a grid, full-width, equal columns. Always visible.
┌────────────┬────────────┬────────────┬────────────┬─────────────────┐
│ ▶ START    │ ⏸ PAUSE    │ ↻ RESTART  │ ■ STOP     │ ⚠ FLATTEN ALL   │
│ (green)    │ (amber)    │ (default)  │ (red-out)  │ (red filled)    │
└────────────┴────────────┴────────────┴────────────┴─────────────────┘
- Start: bg rgba(34,197,94,0.12), color green
- Pause: bg rgba(232,155,46,0.12), color amber
- Restart: default button styling (kills + restarts strategy without stopping engine)
- Stop: red outline only, fills with red on hover
- Flatten All: full red fill, white text. ALWAYS requires confirm modal.

### 7.4 Account / PnL / Position Cards (3-card row)

Stack vertically inside each card; standardized internal structure.

#### Account card

- Header label: "ACCOUNT N · 150K" or similar (depends on active account)
- Main value: balance, 22px mono
- Row 1: Daily limit (label muted, value mono) + 4px progress bar gold gradient
- Row 2: Max DD (label muted, value mono red) + 4px progress bar (gold → amber → red as % rises)
- DD alert message: appears when bar > 75%, format: "⚠ DD ALERT @ 85%" in red 9px

#### PnL card

- Header label: "TODAY P&L · ALL"
- Main value: total P&L, 28px mono, color green/red based on sign
- Sub-row (top): REALIZED / UNREAL / TRADES — 3 columns, 9px label + 12px value
- Sub-row (bottom): WEEK / MONTH / SHARPE — 3 columns, divider above

#### Position card

- Header label: "OPEN POSITION" or "FLAT" when no position
- Pill (LONG green / SHORT red) + avg entry price 19px mono
- Held time + unrealized P&L
- Two-column: STOP (red mono) + TARGET (green mono)
- Three action buttons (5-9px font): Trail · ½ Out · Close (red-outline)

## 8. Daily Summary & News Calendar (paired row)


### 8.1 Daily Summary card (left, ~1.2 fr)

- Card header (bordered bottom): "📊 Daily Summary · <date>" + subtitle session/duration
- Header right: [Yesterday] [⤓ PDF] [Full Report] (gold)
- Section 1 — Top metrics row (4 columns): NET P&L, WIN RATE, BEST TRADE, WORST TRADE
- Section 2 — Intraday equity sparkline (SVG line + gradient fill). Caption shows current and peak
- Section 3 — Bottom two columns: BY SETUP list + BY HOUR list

### 8.2 News Calendar card (right, ~1 fr)

- Card header (bordered bottom): "📅 News Calendar · Today" + count of HIGH-impact events
- Header right: [Week] button to expand to 7-day view
- Event list row format: time (60px mono) · event name + countdown/details · impact pill
- PAST events: opacity 0.5 with strikethrough "past" label
- UPCOMING HIGH-impact: red background tint, countdown shown red
- Impact pills: HIGH (red bg/text), MED (amber), LOW (green)
- Tomorrow section: small "TOMORROW · <date>" header above next-day events
- Footer (bordered top): auto-blackout toggle (on by default) + ±5 min note

### 8.3 Notifications Panel

Below the Daily Summary + News row. Card with up to 5 most-recent notifications.
- Header: "⧖ Notifications" + [Mark all read] right-aligned
- Each row: colored left border (red/amber/green by severity) · dot · title 10px bold · timestamp 9px
- Background: light tint of severity color (4-6% alpha)

## 9. Strategy Management Section


### 9.1 Strategies Card with tabs

Tabbed card with active tab selector at the top: Strategies · Editor · Backtest · Paper · Replay. + New button right-aligned (gold).
- Tab style: 10px / 500 / 1.5px letter-spacing / uppercase
- Active tab: gold-bright color + 2px bottom border gold-bright
- Inactive: text.muted, hover -> text.primary

### 9.2 Strategy table

Default tab shows all strategies. 7-column grid (CSS grid-template-columns: 22px 1fr 70px 60px 60px 60px 180px):

| **Column** | **Content / behavior** |
| --- | --- |
| **Select** | **▣ if active (gold) or ▢ (grey)** |
| **Name** | **Strategy name + version + 🔒 LOCKED icon when live + short description below in 9px muted** |
| **Mode** | **Pill: LIVE / PAPER / BACKTEST / EDITING / OFF** |
| **Trades** | **Total trade count (10px mono)** |
| **Win%** | **Color-coded: green if >55%, amber if 45-55%, red if <45%** |
| **Today** | **Today's P&L (only shown if live or paper)** |
| **Actions** | **Edit ✎ · Versions ⧖ · Clone ⎘ · Promote ↑ (paper→live) · Kill ⊘ (red-outline). Save button (gold) shown when EDITING.** |


### 9.3 Editor + Diff Panel (paired row)

Two cards side-by-side after the strategy table.

#### Editor card (left)

- Header: filename in gold-bright + "Saved Nm ago · unsaved" indicator (amber)
- Action row: ↶ Undo · Test · Save Draft · Deploy (gold)
- Tab bar inside: CODE · PARAMS · SIGNALS · EXITS · FILTERS
- Code area: bg.input, max-height 240-280px, scroll. 11px mono. Syntax-colored: keywords gold, functions blue, strings green, numbers orange, comments dim italic, properties purple.
- Active line: highlight rgba(232,181,71,0.04) + gold line number
- Footer: Python version · Nautilus version + lint/syntax status + cursor position

#### Diff card (right)

- Header: "⊟ Diff: v<old> → v<new>" + change summary
- Toggle: side-by-side vs unified view
- Header right: ↺ Rollback button
- Diff lines: removed lines red-tinted bg + red line number with −; added lines green-tinted bg + green line number with +

### 9.4 Version History card

Sortable table of all versions for the currently-selected strategy.
- Columns: VER · DATE · NOTES · SHARPE · WIN% · TRADES · ACTIONS
- Current version: gold + ★ suffix
- Notes column: change summary entered when saved
- Actions: [View] [Restore] or [Diff vs current]

## 10. Build & Test Section


### 10.1 Backtest + Walk-Forward card

Three-column layout inside one card. Header: "▷ Backtest + Walk-Forward" + status pill + [Monte Carlo] [⤓ Config] [▶ RUN] (gold) buttons.

#### Column 1 — Setup

- DATE RANGE: two date inputs
- WF (walk-forward window): select (IS 6mo / OOS 1mo · roll)
- CAPITAL: text input
- Toggles: Slippage · Commission · Walk-forward CV

#### Column 2 — IS vs OOS results

- Grid table: metric label + IS column + OOS column
- Metrics: Sharpe · Win% · PF · MaxDD · Net · Trades
- OOS values colored green when within acceptable range vs IS
- Bottom row: "OOS robustness" % with ✓ when above threshold

#### Column 3 — Equity curve

- Combined IS + OOS line chart
- Vertical dashed amber line marking IS→OOS boundary, labeled "OOS →"
- Gradient fill below the line

### 10.2 Paper Trade card

Single row with 4 metrics. Header: "↻ Paper Trade" + day-counter pill + [Reset] [⏸ Pause] [↑ Promote] (gold) buttons.
- PAPER P&L (large mono green/red)
- WIN DRIFT — current % with backtest baseline in parens, amber if drifted
- AVG SLIP — average slippage in ticks
- PRE-CONFIRM — toggle for manual approve mode

### 10.3 Historical Replay card

Three-column row: date input · transport controls · speed slider.
- Date input: "2026-05-15 NY AM" format
- Transport: ⏮ ◀ ▶▶(gold) ▶ ⏭ centered. Bar counter below: "Bar 142/384 · 09:14"
- Speed slider: 1x — 100x range with labeled stops at 4x, 10x, 50x

## 11. Live Trading Section


### 11.1 Indicator Library

Complete list of every indicator the system computes. Each one feeds the strategy AND/OR draws on the chart. Indicators are computed on every bar close in the strategy process and published to Redis under vl:state:* so the chart can render them.

| **Indicator** | **Source** | **Used for** | **On chart?** | **Redis key** |
| --- | --- | --- | --- | --- |
| **Fair Value Gap (FVG)** | **smc.fvg** | **Entry zone** | **YES — gold dashed rect** | **vl:state:active_fvg** |
| **Order Block (OB)** | **smc.ob** | **Entry zone, support/resist** | **YES — solid gold rect** | **vl:state:active_ob** |
| **Break of Structure (BOS)** | **smc.bos_choch** | **HTF bias change** | **YES — green up-triangle** | **vl:state:last_bos** |
| **Change of Character (CHoCH)** | **smc.bos_choch** | **Reversal confirmation** | **YES — amber triangle** | **vl:state:last_choch** |
| **Liquidity Sweep** | **smc.liquidity** | **Reversal setup trigger** | **YES — red triangle below/above** | **vl:state:last_sweep** |
| **Equal Highs / Lows** | **smc.eq_h_l** | **Liquidity targets** | **YES — dashed horizontal** | **vl:state:eq_highs, eq_lows** |
| **Swing Highs / Lows** | **smc.swing_highs_lows** | **Structure tracking** | **YES — small dot markers** | **vl:state:swings** |
| **Previous Day H/L/C** | **internal** | **Liquidity targets** | **YES — horizontal lines** | **vl:state:pdh, pdl, pdc** |
| **Session High/Low** | **internal** | **Liquidity targets** | **YES — dashed horizontal** | **vl:state:session_hl** |
| **Killzone Bands** | **internal** | **Trade window filter** | **YES — shaded background** | **vl:state:session** |
| **ATR(14)** | **nautilus.atr** | **Stop distance, vol filter** | **NO — number only** | **vl:state:atr14** |
| **VWAP** | **nautilus.vwap** | **Mean reversion, bias factor** | **YES — orange line** | **vl:state:vwap** |
| **Volume Profile / POC** | **custom** | **Support/resistance** | **YES — horizontal bars right margin** | **vl:state:vp_poc** |
| **Volume bars** | **from feed** | **Filter low-vol bars** | **YES — bottom histogram** | **(per bar)** |
| **8-Factor Bias Score** | **BiasScorer** | **Long/short gate** | **NO — Bias card** | **vl:state:bias_score** |
| **HTF Bias (4H, 1D)** | **BiasScorer** | **Trend filter** | **NO — ICT State card** | **vl:state:htf_bias** |
| **XGB probability** | **xgb.predict** | **ML meta-label gate** | **NO — Bias card footer** | **vl:state:xgb_prob** |
| **Entry line** | **position** | **Open trade marker** | **YES — dashed + pin** | **vl:state:position.avg** |
| **Stop Loss line** | **position** | **Risk visualization** | **YES — red dashed** | **vl:state:position.sl** |
| **Take Profit line** | **position** | **Target visualization** | **YES — green dashed** | **vl:state:position.tp** |
| **Closed trade markers** | **trade history** | **Visual trade log** | **YES — colored circles** | **vl:state:trades_today** |


### 11.2 Live Chart Card

Full-width card, no card padding (chart fills the space). Rendered with TradingView Lightweight Charts v5. Indicators from above are layered on top of the candle series.

#### Chart header

- Left: symbol "NQ 06-26 · LIVE" (gold-bright, 11px 500 weight, letter-spacing 1px)
- Timeframe tabs: 1m · 5m (active gold) · 15m · 1H
- Right: current price 14px mono, green/red based on tick direction (live update)
- Indicator toggle button (top-right corner of chart): opens a popover to show/hide each indicator from §11.1

#### Chart canvas (base layer)

- Background: #0A0A0A
- Grid: rgba(232,181,71,0.05)
- Crosshair: gold dashed, gold label background
- Watermark: "VL · NQ 06-26 · 5m" subtle gold center
- Candles: up green (#22C55E) / down red (#EF4444), wicks match body
- Volume histogram: bottom 20% of chart, gold bars with 0.4 opacity

#### Indicator drawing rules (overlay layer)


| **Indicator** | **Drawing spec** |
| --- | --- |
| **FVG zone** | **Rectangle: top = fvg.top, bottom = fvg.bottom, from FVG bar to current bar. Fill rgba(232,181,71,0.08), border gold dashed 0.5px. Label "FVG↑ <price>" or "FVG↓ <price>" right-aligned in gold.** |
| **Order Block** | **Rectangle: top = ob.top, bottom = ob.bottom, from OB bar to current. Fill rgba(176,141,46,0.10), border solid gold 1px. Label "OB↑" or "OB↓" right-aligned.** |
| **BOS marker** | **Small green up-triangle at break price + "BOS" 8px mono label above. Bearish BOS = red down-triangle below.** |
| **CHoCH marker** | **Same shape as BOS but amber color + "CHoCH" label.** |
| **Liquidity sweep** | **Red up-triangle BELOW the swept low (for low sweep) or above the swept high. "SWEEP" 8px mono label.** |
| **Equal H/L** | **Dashed horizontal line connecting the two equal points, extends right. Color gold-dim. No label.** |
| **Swing H/L** | **Small dot (3px) on the swing wick: green dot for swing low, red dot for swing high.** |
| **PDH / PDL / PDC** | **Horizontal lines spanning full chart width. PDH white dashed, PDL white dashed, PDC dotted. Labels right-aligned in dim white.** |
| **Session H/L** | **Dashed horizontal lines from session open to now. Color matches session (NY = blue, London = gold).** |
| **Killzone bands** | **Background shading on time axis: NY AM rgba(96,165,250,0.04), London rgba(232,181,71,0.04). No labels (subtle).** |
| **VWAP** | **Solid orange line (#EF8B47) 1.5px. Anchored to session open. Label "VWAP" right-aligned.** |
| **Volume Profile** | **Horizontal bars on the right margin (last 30 bars width). POC line highlighted gold. Value Area (70%) shaded.** |
| **Entry line** | **Horizontal dashed line at position.avg, full chart width. Green for long, red for short. Label "Entry <price>" right.** |
| **Stop Loss line** | **Horizontal dashed red line at position.sl. Label "SL <price>" right.** |
| **Take Profit line** | **Horizontal dashed green line at position.tp. Label "TP <price>" right.** |
| **Entry pin** | **Gold-bright filled circle 4px radius, black border 1px, placed at entry bar + price.** |
| **Closed trade dots** | **Each closed trade: green circle at entry + green or red circle at exit, connected by faint line. Hover shows P&L tooltip.** |


#### Implementation notes (Lightweight Charts v5)

- Candles: standard CandlestickSeries
- FVG/OB zones: Series Primitive plugin (v5 supports custom rectangles via primitives)
- Horizontal lines (Entry/SL/TP/PDH/PDL/Session H/L/VWAP): createPriceLine API
- Markers (BOS, CHoCH, SWEEP, swing dots, entry pin, trade circles): setMarkers API
- Killzone background shading: custom time-axis overlay primitive
- Volume histogram: HistogramSeries with priceScaleId='' and scaleMargins.top=0.8
- All overlays re-render on WebSocket state_update message (~1Hz). Tick updates only refresh the last candle.

### 11.3 ICT State · Bias · Live Parameters (3-column row)


#### ICT State (col 1)

- Six rows divided by 0.5px lines: HTF · LAST BOS · FVG · OB · SWEEP · SESSION
- Each row: 9px label muted (left) + mono value (right)
- HTF/BOS direction shown with ⬆ green or ⬇ red glyph

#### 8-Factor Bias (col 2)

- Big score number (36px mono, color = sign)
- Scale labels under: -50 -25 0 +25 +50
- Gauge bar: 7px height, gradient bg, glowing fill from center to current score
- Center 0 mark: vertical gold line
- Status pill: LONG BIAS / SHORT BIAS / NEUTRAL
- Footer: lock time + XGB probability

#### Live Parameters (col 3)

- 4 sliders stacked: RISK % · STOP ATR · TARGET R/R · BIAS MIN
- Each slider row: label (left) + current value gold-bright (right) + slider full-width
- Bottom row (divider above): Long Only toggle · Trade Window toggle
- On change: debounce 500ms then PUT to /api/params/{key}

## 12. Risk & Models Section


### 12.1 Account Risk Rules + Per-Strategy Overrides (paired row)


#### Account Risk Rules (left)

- Header: "⚙ Account Risk Rules" + [Save] gold button
- 6 inputs in 2-col grid: Daily Loss · Max DD · Max Contracts · Consec Loss · Trade Window · News Blackout
- Below divider — DD ALERT THRESHOLDS as 4 pills: 25% / 50% (paper-pill) · 75% (amber-pill) · 90% auto-stop (red-pill)

#### Per-Strategy Overrides (right)

- Header: "⌬ Per-Strategy Overrides" + [+ Add] button
- Table: STRATEGY · RISK% · MAX · STATUS · ✎ edit
- Each strategy can override the global risk settings

### 12.2 Models card (3-column layout)

Header: "⬢ Model Management" + active version pill + [A/B] [↻ Retrain] (gold) buttons.

#### Column 1 — Models list

- Active model: gold-tinted bg, ★ suffix, green ACTIVE pill right
- Archived models: bg.input, [↶ Revert] button right
- Training candidate: dashed border, amber tag, progress bar showing %

#### Column 2 — Live Performance

- 2x2 grid of metrics: ACCURACY · CALIBRATION · GATED · UPLIFT
- Below — "NEXT RETRAIN" date display

#### Column 3 — Top Features

- Up to 6 features, each: feature name + importance % + mini-bar
- Mini-bars use gold gradient, width = importance%

## 13. System & Logs Section


### 13.1 System Logs + Connections (paired row)


#### System Logs (left, 1.5fr)

- Header: "≡ System Logs · Live" + green pulse dot + [Filter] [⤓] buttons
- Log body: bg.input, 9px mono
- Each log: timestamp (dim) · level tag (color-coded INFO/SIGNAL/FILL/CLOSE green, WARN amber, ERROR red) · message
- Auto-scrolls to bottom; pauses on hover; tail subscription via WebSocket

#### Connections (right, 1fr)

- List of 5 services: NinjaTrader 8 · Data Feed · Redis bus · FastAPI :8000 · XGBoost
- Each row: pulse dot (green ok / amber degraded / red down) · service name · latency or version on right
- Border tint matches status color (green/amber/red)

### 13.2 Trade Log (bottom of dashboard)

Today's trades only. Full history lives on the Trade Log nav page.
- Header: "⧗ Trades Today" + filter dropdowns (All strategies) + [⤓ CSV] + [📔 Journal] (goes to Journal page)
- Columns: TIME · SIDE pill · QTY · ENTRY→EXIT · P&L · ACCT · SETUP · JNL
- JNL column: 📔 button that jumps to that trade's journal entry
- OPEN trade row: exit shown as "OPEN" in gold-bright

## 14. Trade Journal Page (separate route)

Accessed from sidebar nav. Full-page view for annotating, screenshotting, and tagging trades.

### 14.1 Page header

- Title: "📔 Trade Journal"
- Subtitle: "<total> trades · <noted> with notes · <screenshot count> with screenshots"
- Right: [Search] [Export] [+ Manual Entry] (gold)

### 14.2 Filter bar card

- 6-column grid: DATE RANGE (two date inputs) · STRATEGY (select) · SETUP (select) · OUTCOME (select wins/losses/all) · HAS NOTES (toggle) · [Apply] gold

### 14.3 Stats row (5 cards)

- TOTAL TRADES · NET P&L · WIN RATE · AVG R:R · JOURNAL %

### 14.4 Two-column body


#### Left column — Selected entry detail

- Header: outcome pill (WIN +$700 / LOSS −$500) · trade description
- Subtitle: "<date> · <strategy> · <account>"
- Header right: [⟳ Replay] [⤓ Export]
- Section 1 — Auto-captured trade screenshot (chart at fill moment with FVG/OB/entry/SL/TP overlays). 160px height SVG. Section label "📷 TRADE SCREENSHOT (auto-captured at fill)"
- Section 2 — Trade details 4-col grid: ENTRY · EXIT · HELD · R MULTIPLE
- Section 3 — Tags row: "🏷 TAGS" with chips (A+ setup, London sweep, NY AM, etc.) + [+ Tag] button
- Section 4 — Notes textarea: editable, ~70px min-height, font Inter 11px. Below: "Last edited Nh ago" + [Save Notes] (gold)

#### Right column — Recent entries list

- Header: "Recent Entries"
- Each row: trade description (LONG 2 NQ · LIQ+FVG) + P&L · date/time · indicators showing what's attached (📷 + 📝 + N tags)
- Selected row: gold-tinted bg, gold left-border 2px
- Unrated/no-notes rows: amber "no notes" indicator
- Footer: "Showing N of M · Load more"

## 15. Models Page (full sidebar nav page)

Dedicated page for the 35+ model registry. With 5 algorithms × 7 timeframes plus RD-Agent-discovered models, the Models card on the dashboard can only show top-line stats; this full page provides the table view, filters, and individual model inspection.

### 15.1 Page header

- Title: "⬢ Models"
- Subtitle: "<active count> active · <total count> total · last retrain <date>"
- Right: [+ Add Model] (gold) · [Retrain All] · [Import] · [Export]

### 15.2 Status bar (4 stat cards)


| **Card** | **Content** |
| --- | --- |
| **Active Models** | **Count of models with status=ACTIVE / total slots (e.g. 35/35)** |
| **Best Sharpe (live)** | **Top-performing model name + Sharpe value in mono green** |
| **Next Retrain** | **Countdown to Sunday 02:00 CT scheduled retrain** |
| **RD-Agent Status** | **RUNNING / IDLE pill + last run timestamp + new factors discovered count** |


### 15.3 Filter bar

Sticky filter row above the table:
- ALGORITHM: dropdown (All / LightGBM / XGBoost / CatBoost / LSTM / Transformer / RD-Agent discovered)
- TIMEFRAME: chips (1m · 2m · 3m · 5m · 1h · 1d · 1w · All) — multi-select
- STATUS: chips (Active · Training · Failed · Archived)
- FEATURE SET: dropdown (Alpha158 / Alpha360 / Custom)
- SORT: dropdown (Sharpe desc / IC desc / Date desc / Name)
- SEARCH: text input for model name

### 15.4 Model registry table

Main table — sortable, filterable, paginated. One row per model.

| **Column** | **Width** | **Content** |
| --- | --- | --- |
| **Status** | **60px** | **Pill: ACTIVE (green) / TRAINING (amber) / FAILED (red) / ARCHIVED (grey)** |
| **Model name** | **1.5fr** | **e.g. "lightgbm_alpha158_5m" + small subtitle showing algorithm + feature set** |
| **Algorithm** | **100px** | **LightGBM / XGBoost / CatBoost / LSTM / Transformer / rdagent_<id>** |
| **Timeframe** | **70px** | **Pill: 1m / 2m / 3m / 5m / 1h / 1d / 1w** |
| **Features** | **90px** | **Alpha158 / Alpha360 / custom feature set name** |
| **IC** | **60px** | **Information Coefficient — backtested + live. Green if >0.03, amber 0.01-0.03, red <0.01** |
| **Sharpe** | **60px** | **Live Sharpe (last 30d) — green/amber/red coding** |
| **Last train** | **90px** | **Date in YYYY-MM-DD format, dim if stale (>30d)** |
| **Weight** | **60px** | **Ensemble weight 0.00-1.00 (auto-tuned)** |
| **Actions** | **200px** | **[Inspect ⊙] [Retrain ↻] [Disable ⊘] [Clone ⎘]** |


### 15.5 Model inspection drawer (slides in from right when row clicked)

Click Inspect ⊙ to open a 600px-wide drawer over the right side of the page. Contains:
- Header: model name + algorithm + version hash
- Tabs: OVERVIEW · FEATURES · BACKTEST · LIVE · CODE
- OVERVIEW: model card with last-train date, hyperparameters, target column (e.g. "label_5m_5" = 5-bar forward return), data range used for training
- FEATURES: feature importance list (top 20 from Alpha158 or custom set) with mini-bars
- BACKTEST: IS/OOS metrics table + equity curve SVG
- LIVE: rolling 30-day IC, Sharpe, hit rate, drift vs backtest
- CODE: raw Qlib workflow YAML — read-only, with [Open in Editor] button

### 15.6 RD-Agent panel (collapsible, top-right of page)

Live trace of RD-Agent's nightly discovery loop:
- Status: RUNNING (animated pulse) / IDLE / FAILED
- Current iteration: "Trying factor #12: rolling 20-bar volume-weighted RSI..."
- Results so far: "3 new factors accepted, 9 rejected (IC < 0.02)"
- Token usage: "$0.42 spent · GPT-4o-mini"
- Next run: countdown to 23:00 CT
- [Open Full Trace] button → modal with complete LLM transcript

### 15.7 Empty / Training states

- Empty (first install): show "No models yet. Click [+ Add Model] or [Retrain All] to begin." with friendly gold-accent illustration
- Training row: replaces metrics with a 3px progress bar + percentage + ETA
- Failed row: red row background, error message in expandable footer

# Part III — Backend & Data Contract


## 16. Redis Schema

Single source of truth for live state. Strategy writes; backend reads; UI subscribes.

| **Key** | **Type** | **Owner / Purpose** |
| --- | --- | --- |
| **vl:control:trading_enabled** | **bool** | **UI writes / strategy reads — global killswitch** |
| **vl:control:long_only** | **bool** | **UI writes — disable shorts** |
| **vl:control:active_account** | **string** | **UI writes — currently selected account ID** |
| **vl:control:active_strategy** | **string** | **UI writes — currently active strategy** |
| **vl:params:risk_pct** | **float** | **UI writes — % of account per trade** |
| **vl:params:stop_atr_mult** | **float** | **UI writes — stop = N × ATR** |
| **vl:params:target_rr** | **float** | **UI writes — target R/R** |
| **vl:params:bias_threshold** | **int** | **UI writes — 8-factor min** |
| **vl:state:bias_score** | **int** | **Strategy writes — current 8-factor** |
| **vl:state:htf_bias** | **string** | **Strategy — BULLISH/BEARISH/NEUTRAL** |
| **vl:state:active_fvg** | **JSON** | **{ top, bottom, side }** |
| **vl:state:active_ob** | **JSON** | **{ top, bottom, side, vol }** |
| **vl:state:last_bos** | **JSON** | **{ time, side }** |
| **vl:state:last_sweep** | **JSON** | **{ time, side }** |
| **vl:state:session** | **string** | **ASIA / LONDON / NY_AM / NY_PM / OFF** |
| **vl:state:position** | **JSON** | **{ side, qty, avg, sl, tp, age_sec }** |
| **vl:state:open_pnl** | **float** | **Strategy — unrealized** |
| **vl:state:realized_today** | **float** | **Strategy** |
| **vl:state:trades_today** | **LIST** | **LPUSH on each close** |
| **vl:health:nt8** | **timestamp** | **Bridge writes — heartbeat** |
| **vl:health:data_feed** | **timestamp** | **Strategy writes — last tick** |
| **vl:health:xgb_version** | **string** | **Strategy writes on load** |
| **vl:news:today** | **LIST<JSON>** | **News service — today's events** |
| **vl:journal:<trade_id>** | **JSON** | **Journal entry: notes, tags, screenshot path** |
| **vl:commands (pub/sub)** | **channel** | **UI publishes commands → strategy subscribes** |
| **vl:events (pub/sub)** | **channel** | **Strategy publishes events → UI/journal subscribe** |
| **vl:logs (pub/sub)** | **channel** | **All services publish logs → UI live-tails** |


## 17. FastAPI Endpoints


| **Method** | **Path** | **Purpose** |
| --- | --- | --- |
| **POST** | **/api/control/start** | **Set trading_enabled = true** |
| **POST** | **/api/control/pause** | **trading_enabled = false (keep open)** |
| **POST** | **/api/control/restart** | **Restart active strategy** |
| **POST** | **/api/control/stop** | **Pause + close all at market** |
| **POST** | **/api/control/flatten** | **PUBLISH flatten_all command** |
| **POST** | **/api/control/emergency** | **Flatten all + halt all strategies + lock UI** |
| **GET** | **/api/state** | **Snapshot of vl:state:*** |
| **GET** | **/api/params** | **All vl:params:*** |
| **PUT** | **/api/params/{key}** | **Set vl:params:{key}** |
| **GET** | **/api/accounts** | **List configured accounts** |
| **PUT** | **/api/accounts/active** | **Switch active account** |
| **GET** | **/api/strategies** | **List all strategies** |
| **POST** | **/api/strategies** | **Create new strategy** |
| **PUT** | **/api/strategies/{id}** | **Save / deploy strategy** |
| **POST** | **/api/strategies/{id}/clone** | **Clone strategy to a new id** |
| **GET** | **/api/strategies/{id}/versions** | **Version history** |
| **POST** | **/api/strategies/{id}/rollback** | **Restore prior version** |
| **POST** | **/api/backtest/run** | **Kick off backtest job** |
| **GET** | **/api/backtest/results/{id}** | **Get IS/OOS metrics + equity** |
| **POST** | **/api/paper/start|stop|reset** | **Paper trade control** |
| **GET** | **/api/paper/status** | **Paper trade metrics** |
| **POST** | **/api/replay/load** | **Load a historical date for replay** |
| **GET** | **/api/chart/bars?tf=5m&n=300** | **Bars + active FVG/OB for chart** |
| **GET** | **/api/news/today** | **Today's economic events** |
| **GET** | **/api/summary/daily** | **Daily summary computation** |
| **GET** | **/api/trades/today** | **Today's trades** |
| **GET** | **/api/journal** | **List journal entries (paginated)** |
| **GET** | **/api/journal/{trade_id}** | **Single journal entry** |
| **PUT** | **/api/journal/{trade_id}** | **Update notes/tags** |
| **POST** | **/api/journal/{trade_id}/screenshot** | **Upload or regenerate screenshot** |
| **GET** | **/api/risk** | **Risk rules for active account** |
| **PUT** | **/api/risk** | **Save risk rule changes** |
| **GET** | **/api/models** | **List XGBoost models** |
| **POST** | **/api/models/retrain** | **Trigger retrain** |
| **GET** | **/api/system/health** | **All vl:health:*** |
| **GET** | **/api/notifications** | **Recent notifications** |
| **WS** | **/ws** | **Push state + events + ticks + logs** |


## 18. WebSocket Protocol

Server pushes JSON messages on five topics. Frontend dispatches by type.
// Envelope
{ "type": "<msg_type>", "data": {...} }
// Types:
//   state_update — full snapshot, every ~1s
//   tick         — single price tick (high frequency)
//   trade_event  — position opened/closed
//   log          — system log line (for the live tail)
//   alert        — system alert (drives notifications panel)
//   news         — economic event notification
// state_update example
{
"type": "state_update",
"data": {
"account": { "balance": 148250, "daily_used": 420, "daily_limit": 3000, "max_dd": -4250, "dd_limit": -5000 },
"pnl": { "realized": 420, "unreal": 180, "trades": 3, "wins": 2, "losses": 1, "week": 3240, "month": 12180, "sharpe": 1.84 },
"position": { "side": "LONG", "qty": 2, "avg": 21487.5, "sl": 21462.5, "tp": 21547.5, "age_sec": 840 },
"bias": { "score": 32, "htf": "BULLISH", "xgb_prob": 0.68, "locked_until": "10:30" },
"ict": {
"fvg": { "top": 21465, "bottom": 21450, "side": "BULL" },
"ob":  { "top": 21430, "bottom": 21420, "side": "BULL", "vol": 12500 },
"last_bos":   { "time": "09:18", "side": "BULL" },
"last_sweep": { "time": "09:32", "side": "LOW" },
"session": "NY_AM"
},
"health": { "nt8": true, "data_feed": true, "redis": true, "xgb_version": "v2026.05.18" }
}
}
// tick — drives the chart and current-price flash
{ "type": "tick", "data": { "ts": 1747754533123, "price": 21492.25 } }
// trade_event — drives Trade Log + auto-screenshot
{
"type": "trade_event",
"data": {
"event": "OPENED" | "CLOSED",
"trade": { "id": "trd_0142", "time": "09:38:22", "side": "LONG", "qty": 2,
"entry": 21487.50, "exit": null, "pnl": 0, "reason": "FVG_LONG",
"strategy_id": "fvg_reversal_v3", "account_id": "acc_1" }
}
}
// log — drives the System Logs live tail
{ "type": "log", "data": { "ts": "09:42:13", "level": "INFO|SIGNAL|FILL|CLOSE|WARN|ERROR", "msg": "..." } }
// alert — drives Notifications panel
{ "type": "alert", "data": { "level": "INFO|WARN|CRITICAL", "msg": "...", "timestamp": "..." } }
// news — drives News Calendar countdown / blackout banner
{ "type": "news", "data": { "event": "FOMC Statement", "in_minutes": 18, "impact": "HIGH" } }

## 19. NinjaTrader Bridge


### 19.1 NT8 → Python (data via ninja-socket)

Install the ninja-socket NinjaScript add-on into NinjaTrader 8. Configure it to broadcast ticks and 5m bars to port 9001.
# bridge_env/python/nt_socket_client.py
import asyncio, json, redis.asyncio as redis
import websockets
async def main():
r = redis.from_url("redis://localhost:6379")
async with websockets.connect("ws://host.docker.internal:9001/stream") as ws:
async for raw in ws:
msg = json.loads(raw)
if msg["type"] == "tick":
await r.publish("vl:ticks", json.dumps(msg))
await r.set("vl:health:data_feed", msg["ts"])
elif msg["type"] == "bar":
await r.publish("vl:bars", json.dumps(msg))
# also write to today's parquet
await append_to_parquet(msg)
if __name__ == "__main__":
asyncio.run(main())

### 19.2 Python → NT8 (orders via Claude-Trader CSV)

The strategy appends rows to signals.csv. The Claude-Trader NinjaScript polls every 2 s and rewrites status fields as orders are placed/filled.
# strategy/csv_writer.py
import csv, os
from datetime import datetime, timezone
SIGNAL_CSV = "/mnt/c/trading/signals.csv"
def write_signal(action, qty, entry, stop, target, reason):
sig_id = f"sig_{datetime.now(timezone.utc).strftime('%Y%m%d%H%M%S%f')[:-3]}"
row = [
datetime.now(timezone.utc).isoformat(),
sig_id, action, "NQ 06-26",
qty, entry, stop, target, reason, "NEW"
]
new_file = not os.path.exists(SIGNAL_CSV)
with open(SIGNAL_CSV, "a", newline="") as f:
w = csv.writer(f)
if new_file:
w.writerow(["timestamp","signal_id","action","symbol","qty","entry","stop","target","reason","status"])
w.writerow(row)
return sig_id

## 20. Qlib + RD-Agent Integration


### 20.1 Qlib data layout (7 timeframes)

Each timeframe gets its own Qlib binary data directory. The bridge writes ticks to Parquet first, then a converter script populates Qlib bins on a schedule.
~/.qlib/qlib_data/
├── nq_1m/         # 1-minute bars
│   ├── calendars/day.txt
│   ├── instruments/all.txt
│   └── features/NQ/{open,high,low,close,volume,factor}.day.bin
├── nq_2m/
├── nq_3m/
├── nq_5m/
├── nq_1h/
├── nq_1d/
└── nq_1w/
# Conversion CLI (run nightly or after backfill)
python -m vl.qlib.convert_parquet_to_qlib \
--parquet ~/vl_trading/data/parquet/nq_5m.parquet \
--out ~/.qlib/qlib_data/nq_5m \
--instrument NQ

### 20.2 Qlib workflow YAML pattern

Each (algorithm, timeframe) pair is one YAML file. 5 algos × 7 timeframes = 35 YAML files in configs/qlib/. Example for LightGBM on 5m:
# configs/qlib/lightgbm_5m.yaml
qlib_init:
provider_uri: "~/.qlib/qlib_data/nq_5m"
region: us
market: &market all
benchmark: &benchmark NQ
data_handler_config: &data_handler_config
start_time: 2022-01-01
end_time: 2026-05-01
fit_start_time: 2022-01-01
fit_end_time: 2025-05-01
instruments: *market
port_analysis_config: &port_analysis_config
strategy:
class: TopkDropoutStrategy
module_path: qlib.contrib.strategy
kwargs:
topk: 1
n_drop: 0
backtest:
start_time: 2025-05-01
end_time: 2026-05-01
account: 150000
benchmark: *benchmark
exchange_kwargs:
freq: 5min
limit_threshold: 0.095
deal_price: close
open_cost: 0.0005
close_cost: 0.0015
task:
model:
class: LGBModel
module_path: qlib.contrib.model.gbdt
kwargs:
loss: mse
num_leaves: 64
learning_rate: 0.05
n_estimators: 500
dataset:
class: DatasetH
module_path: qlib.data.dataset
kwargs:
handler:
class: Alpha158
module_path: qlib.contrib.data.handler
kwargs: *data_handler_config
segments:
train: [2022-01-01, 2024-12-31]
valid: [2025-01-01, 2025-04-30]
test:  [2025-05-01, 2026-05-01]
record:
- class: SignalRecord
module_path: qlib.workflow.record_temp
- class: SigAnaRecord
module_path: qlib.workflow.record_temp
- class: PortAnaRecord
module_path: qlib.workflow.record_temp
kwargs:
config: *port_analysis_config

### 20.3 Reading Qlib predictions in NautilusTrader

# src/strategy/qlib_consumer.py
import qlib
from qlib.data import D
import pandas as pd
import redis
REDIS = redis.Redis(host='localhost', port=6379, decode_responses=True)
class QlibEnsembleSignal:
"""Loads latest predictions from all 35 models and combines them."""
def __init__(self):
qlib.init(provider_uri="~/.qlib/qlib_data/nq_5m", region="us")
self.weights = self._load_weights()
def _load_weights(self):
# Loaded from Redis: vl:ensemble:weights
# Stored as JSON: {"lightgbm_5m": 0.18, "xgboost_1h": 0.12, ...}
import json
return json.loads(REDIS.get("vl:ensemble:weights") or "{}")
def latest_signal(self, ts):
"""Returns weighted prediction in [-1, +1]. >0 = long, <0 = short."""
preds = {}
for model_id, weight in self.weights.items():
pred_key = f"vl:predictions:{model_id}:{ts}"
raw = REDIS.get(pred_key)
if raw is not None:
preds[model_id] = float(raw) * weight
if not preds:
return 0.0
return sum(preds.values()) / sum(self.weights.values())

### 20.4 RD-Agent integration

RD-Agent runs as a separate process (its own systemd timer). It reads the Qlib data, proposes new factors via LLM, tests them, and writes accepted factors back to the Qlib workspace.
# scripts/rdagent_loop.sh
#!/bin/bash
# Runs nightly at 23:00 CT via systemd timer.
source ~/vl_trading/.venvs/rdagent_env/bin/activate
export OPENAI_API_KEY="$(cat ~/vl_trading/secrets/openai.key)"
export QLIB_DATA_PATH=~/.qlib/qlib_data/nq_5m
export RDAGENT_LOG_PATH=~/vl_trading/logs/rdagent
cd ~/vl_trading/rdagent_workspace
# Run factor discovery loop with budget cap
rdagent fin_factor \
--max_iter 20 \
--llm_model gpt-4o-mini \
--budget_usd 5.00 \
--output_dir ./discoveries/$(date +%Y%m%d)
# Publish trace to Redis for the dashboard
python -m vl.rdagent.publish_trace \
--trace_dir ./discoveries/$(date +%Y%m%d)

### 20.5 Sunday weekly retrain

# scripts/weekly_retrain.sh
#!/bin/bash
# Runs every Sunday at 02:00 CT via systemd timer.
source ~/vl_trading/.venvs/qlib_env/bin/activate
cd ~/vl_trading
# Step 1: convert latest Parquet to Qlib bins for all 7 timeframes
for tf in 1m 2m 3m 5m 1h 1d 1w; do
python -m vl.qlib.convert_parquet_to_qlib \
--parquet data/parquet/nq_${tf}.parquet \
--out ~/.qlib/qlib_data/nq_${tf} \
--instrument NQ
done
# Step 2: retrain all 35 models in parallel (5 algos × 7 tfs)
for algo in lightgbm xgboost catboost lstm transformer; do
for tf in 1m 2m 3m 5m 1h 1d 1w; do
qrun configs/qlib/${algo}_${tf}.yaml \
--experiment_name vl_weekly_$(date +%Y%m%d) &
done
done
wait
# Step 3: recompute ensemble weights based on rolling 30-day Sharpe
python -m vl.qlib.update_ensemble_weights
# Step 4: publish notification
redis-cli PUBLISH vl:alerts '{"level":"INFO","msg":"Weekly retrain complete: 35 models updated"}'

# Part IV — Phases & Build Order


## 21. 7-Phase Build Plan


| **#** | **Phase** | **Outcome** | **Time** |
| --- | --- | --- | --- |
| **1** | **Foundation** | **WSL2 envs, NT8 socket bridge, NautilusTrader hello-world, Qlib install** | **Week 1** |
| **2** | **Qlib data + first model** | **Convert 1 year NQ 5m to Qlib bins, run LightGBM Alpha158 benchmark** | **Week 2** |
| **3** | **Full registry + ensemble** | **All 7 timeframes × 5 algos = 35 models trained, ensemble logic working** | **Week 3-4** |
| **4** | **Backend + State** | **Redis, FastAPI, NT8 CSV bridge, NautilusTrader consumes Qlib predictions** | **Week 5** |
| **5** | **Frontend** | **Full dashboard + Models page + Trade Journal** | **Week 6-7** |
| **6** | **Paper → Live** | **2 weeks paper, then funded account with 1-contract cap** | **Week 8-9** |
| **7** | **RD-Agent + automation** | **RD-Agent nightly discovery + Sunday weekly retrain + News Calendar** | **Week 10+** |


## 22. Acceptance Criteria per Phase


### Phase 1 — Foundation

- ninja-socket sends ticks; Python bridge writes to Redis with < 200 ms latency
- NautilusTrader runs an EMA-cross backtest on 1 month of NQ 5-min and reports stats
- All 5 venvs created, frozen lockfiles committed

### Phase 2 — Qlib data + first model

- Qlib installed in qlib_env; `qrun examples/benchmarks/LightGBM/workflow_config_lightgbm_Alpha158.yaml` runs successfully on bundled sample data
- 1 year NQ 5m Parquet successfully converted to Qlib binary format via scripts/convert_parquet_to_qlib.py
- LightGBM + Alpha158 trained on the NQ 5m data; IC > 0.02 on out-of-sample test
- Backtest report (Sharpe, drawdown, equity curve) renders correctly in Jupyter

### Phase 3 — Full registry + ensemble

- All 7 timeframes (1m, 2m, 3m, 5m, 1h, 1d, 1w) have Qlib binary data
- All 5 algos (LightGBM, XGBoost, CatBoost, LSTM, Transformer) train successfully on each timeframe = 35 models
- Ensemble weights computed via rolling 30-day Sharpe of each model
- Combined ensemble Sharpe > best individual model Sharpe (diversity benefit)
- Walk-forward (6mo IS / 1mo OOS rolling) produces robustness ≥ 70% for the ensemble

### Phase 4 — Backend + State

- All REST endpoints respond < 100 ms
- WebSocket stays connected through a forced restart of the strategy
- NT8 CSV bridge fills a paper bracket order end-to-end with status round-trip

### Phase 5 — Frontend

- All dashboard panels render with live data via WebSocket
- Sliders write back through PUT /api/params/{key} within 500ms debounce
- Trade Journal page can save notes, attach screenshot, tag a trade
- Killswitch buttons trigger the strategy correctly (verified in paper mode)

### Phase 6 — Paper → Live

- 2 weeks paper, realized PnL within 20% of backtest expectation
- All 8 risk limits tested by deliberate paper breach
- Day-1 live: 1 contract cap regardless of risk %

### Phase 7 — RD-Agent + automation (optional but recommended)

- RD-Agent successfully runs `fin_factor` loop with gpt-4o-mini, completes 20 iterations under $5
- At least one RD-Agent-discovered factor has IC > 0.03 on out-of-sample, not already in Alpha158
- Weekly Sunday retrain systemd timer fires successfully and updates all 35 models
- Nightly RD-Agent discovery timer fires and publishes trace to Redis for the Models page
- News calendar auto-blackout fires correctly on next FOMC release
- Daily summary auto-generates at session close

## 23. Code Templates


### 23.1 Strategy class skeleton (Qlib ensemble consumer)

# src/strategy/nq_qlib_ensemble.py
from decimal import Decimal
from nautilus_trader.config import StrategyConfig
from nautilus_trader.model.data import Bar, BarType
from nautilus_trader.model.identifiers import InstrumentId
from nautilus_trader.model.enums import OrderSide
from nautilus_trader.trading.strategy import Strategy
from nautilus_trader.indicators.atr import AverageTrueRange
from src.strategy.qlib_consumer import QlibEnsembleSignal
from src.strategy.redis_publisher import RedisPublisher
from src.strategy.param_loader import ParamLoader
from src.strategy.csv_writer import write_signal
class NQ_Qlib_Config(StrategyConfig, frozen=True):
instrument_id: InstrumentId
bar_type: BarType
trade_size: Decimal
class NQ_Qlib_Ensemble(Strategy):
"""Reads ensemble prediction from 35 Qlib models, applies risk, fires order."""
def __init__(self, config: NQ_Qlib_Config):
super().__init__(config)
self.atr = AverageTrueRange(14)
self.signal_source = QlibEnsembleSignal()
self.params = ParamLoader("configs/strategy.yaml")
self.redis = RedisPublisher()
def on_start(self):
self.register_indicator_for_bars(self.config.bar_type, self.atr)
self.subscribe_bars(self.config.bar_type)
def on_bar(self, bar: Bar):
if self.bar_count % 30 == 0:
self.params.reload()
if not self.atr.initialized:
return
if not self.redis.is_trading_enabled():
self.publish_state(bar); return
if not self.in_killzone(bar.ts_event):
self.publish_state(bar); return
# Ensemble prediction in [-1, +1]
signal = self.signal_source.latest_signal(bar.ts_event)
# Gate by configurable threshold (replaces 8-factor bias threshold)
if abs(signal) < self.params.signal_threshold:
self.publish_state(bar, signal); return
# Build bracket
side = "LONG" if signal > 0 else "SHORT"
self.submit_bracket(side, signal, bar)
self.publish_state(bar, signal)
def submit_bracket(self, side, signal, bar):
atr_val = float(self.atr.value)
stop_dist = atr_val * self.params.stop_atr_mult
target_dist = stop_dist * self.params.target_rr
entry = float(bar.close)
if side == "LONG":
sl, tp = entry - stop_dist, entry + target_dist
action = "BUY"
else:
sl, tp = entry + stop_dist, entry - target_dist
action = "SELL"
write_signal(action, int(self.config.trade_size), entry, sl, tp,
f"ensemble_{signal:.3f}")
self.redis.publish_trade_event("OPENED", {
"side": side, "qty": int(self.config.trade_size),
"entry": entry, "sl": sl, "tp": tp,
"signal_strength": signal,
"reason": f"ENSEMBLE_{side}"
})

### 23.2 FastAPI main.py skeleton

# src/backend/main.py
from fastapi import FastAPI, WebSocket, WebSocketDisconnect, Header, HTTPException, Depends
from fastapi.middleware.cors import CORSMiddleware
from redis.asyncio import Redis
import asyncio, json, os
from loguru import logger
API_TOKEN = os.environ["VL_API_TOKEN"]
app = FastAPI(title="VL Backend")
app.add_middleware(CORSMiddleware, allow_origins=["*"], allow_methods=["*"], allow_headers=["*"])
redis: Redis = None
@app.on_event("startup")
async def startup():
global redis
redis = Redis.from_url("redis://localhost:6379", decode_responses=True)
def auth(authorization: str = Header(None)):
if authorization != f"Bearer {API_TOKEN}":
raise HTTPException(401, "Unauthorized")
@app.post("/api/control/start", dependencies=[Depends(auth)])
async def start():
await redis.set("vl:control:trading_enabled", "true")
return {"ok": True}
@app.post("/api/control/flatten", dependencies=[Depends(auth)])
async def flatten():
await redis.publish("vl:commands", "flatten_all")
return {"ok": True}
@app.put("/api/params/{key}", dependencies=[Depends(auth)])
async def set_param(key: str, payload: dict):
await redis.set(f"vl:params:{key}", str(payload["value"]))
return {"ok": True}
@app.get("/api/state", dependencies=[Depends(auth)])
async def get_state():
keys = await redis.keys("vl:state:*")
return { k.split(":")[-1]: await redis.get(k) for k in keys }
@app.websocket("/ws")
async def ws_endpoint(ws: WebSocket, token: str = ""):
if token != API_TOKEN:
await ws.close(code=4401); return
await ws.accept()
ps = redis.pubsub()
await ps.subscribe("vl:events", "vl:logs", "vl:ticks")
try:
while True:
# Push snapshot every 1s
snap = await build_snapshot(redis)
await ws.send_json({"type": "state_update", "data": snap})
# Drain pub/sub
for _ in range(20):
msg = await ps.get_message(ignore_subscribe_messages=True, timeout=0.05)
if not msg: break
channel = msg["channel"].decode() if isinstance(msg["channel"], bytes) else msg["channel"]
t = {"vl:events": "trade_event", "vl:logs": "log", "vl:ticks": "tick"}[channel]
await ws.send_json({"type": t, "data": json.loads(msg["data"])})
await asyncio.sleep(1)
except WebSocketDisconnect:
await ps.unsubscribe()

### 23.3 React App.tsx (top-level routing)

// src/App.tsx
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Sidebar } from "./components/Sidebar";
import { Dashboard } from "./pages/Dashboard";
import { TradeJournal } from "./pages/TradeJournal";
import { Backtest } from "./pages/Backtest";
import { Models } from "./pages/Models";
import { useLiveState } from "./hooks/useWebSocket";
export default function App() {
useLiveState(); // bootstraps WebSocket and feeds Zustand store
return (
<BrowserRouter>
<div className="flex min-h-screen bg-bg-primary text-text-primary">
<Sidebar />
<main className="flex-1 p-5">
<Routes>
<Route path="/" element={<Dashboard />} />
<Route path="/journal" element={<TradeJournal />} />
<Route path="/journal/:tradeId" element={<TradeJournal />} />
<Route path="/backtest" element={<Backtest />} />
<Route path="/models" element={<Models />} />
{/* ... other pages */}
</Routes>
</main>
</div>
</BrowserRouter>
);
}

### 23.4 Tailwind config tokens

// tailwind.config.ts
import type { Config } from "tailwindcss";
export default {
content: ["./index.html", "./src/**/*.{ts,tsx}"],
theme: {
extend: {
colors: {
bg: {
primary:  "#0A0A0A",
panel:    "#141414",
elevated: "#1F1F1F",
sidebar:  "#0D0D0D",
input:    "#0F0F0F",
},
gold: {
DEFAULT: "#B08D2E",
bright:  "#E8B547",
dim:     "#8A6F24",
},
text: {
primary: "#F5F5F5",
muted:   "#999999",
dim:     "#666666",
},
status: {
live: "#22C55E",
loss: "#EF4444",
warn: "#E89B2E",
test: "#60A5FA",
},
},
fontFamily: {
sans: ["Inter", "sans-serif"],
mono: ["JetBrains Mono", "monospace"],
},
borderRadius: { panel: "10px", btn: "6px", pill: "4px" },
},
},
} satisfies Config;

# Part V — Deployment


## 24. Folder Layout

~/vl_trading/
├── .venvs/
│   ├── qlib_env/        (Phase 2 — pyqlib, lightgbm, xgboost, catboost, torch)
│   ├── rdagent_env/     (Phase 7 — rdagent, openai)
│   ├── nautilus_env/    (Phase 1 — consumes Qlib predictions)
│   ├── control_env/     (Phase 4 — fastapi)
│   └── bridge_env/      (Phase 1 — NT8 socket)
├── locks/               (frozen requirements per env)
├── data/
│   ├── parquet/         (raw ticks/bars from NT8, partitioned by date)
│   └── catalog/         (NautilusTrader catalog)
├── ~/.qlib/qlib_data/   (Qlib binary format, 7 timeframes)
│   ├── nq_1m/
│   ├── nq_2m/
│   ├── nq_3m/
│   ├── nq_5m/
│   ├── nq_1h/
│   ├── nq_1d/
│   └── nq_1w/
├── configs/
│   ├── qlib/            (35 YAML workflows)
│   │   ├── lightgbm_1m.yaml ... lightgbm_1w.yaml
│   │   ├── xgboost_1m.yaml ... xgboost_1w.yaml
│   │   ├── catboost_*.yaml
│   │   ├── lstm_*.yaml
│   │   └── transformer_*.yaml
│   ├── strategy.yaml    (NautilusTrader runtime config)
│   ├── risk.yaml
│   └── instruments/nq.yaml
├── models/              (trained Qlib model artifacts)
│   ├── lightgbm_5m/v2026.05.20/model.pkl
│   ├── transformer_1h/v2026.05.20/model.pt
│   └── ...
├── rdagent_workspace/   (RD-Agent's discovery output)
│   └── discoveries/
│       ├── 20260518/    (one folder per nightly run)
│       └── 20260519/
├── src/
│   ├── bridge/
│   │   ├── nt_socket_client.py
│   │   └── csv_writer.py
│   ├── qlib/
│   │   ├── convert_parquet_to_qlib.py
│   │   ├── update_ensemble_weights.py
│   │   └── ensemble_signal.py
│   ├── rdagent/
│   │   ├── publish_trace.py
│   │   └── apply_discoveries.py
│   ├── strategy/
│   │   ├── nq_qlib_ensemble.py     (the NautilusTrader strategy)
│   │   ├── qlib_consumer.py         (reads predictions from Redis)
│   │   ├── redis_publisher.py
│   │   └── param_loader.py
│   ├── backend/
│   │   ├── main.py
│   │   ├── routes/
│   │   │   ├── control.py
│   │   │   ├── strategies.py
│   │   │   ├── backtest.py
│   │   │   ├── paper.py
│   │   │   ├── journal.py
│   │   │   ├── risk.py
│   │   │   ├── models.py            (Qlib model registry endpoints)
│   │   │   ├── rdagent.py           (RD-Agent trace endpoints)
│   │   │   ├── news.py
│   │   │   └── summary.py
│   │   ├── websocket.py
│   │   └── redis_client.py
│   ├── journal/
│   │   ├── screenshot_service.py
│   │   └── storage.py
│   └── nt8/
│       ├── ninja-socket/    (NinjaScript add-on source)
│       └── VLBridge/        (your custom NinjaScript order receiver)
├── frontend/
│   ├── index.html
│   ├── package.json
│   ├── tailwind.config.ts
│   └── src/
│       ├── App.tsx
│       ├── pages/
│       │   ├── Dashboard.tsx
│       │   ├── Models.tsx           (full 35+ row registry)
│       │   ├── TradeJournal.tsx
│       │   ├── Strategies.tsx
│       │   ├── Backtest.tsx
│       │   ├── PaperTrade.tsx
│       │   ├── Replay.tsx
│       │   ├── TradeLog.tsx
│       │   ├── SystemLogs.tsx
│       │   ├── RiskRules.tsx
│       │   ├── Connections.tsx
│       │   ├── Notifications.tsx
│       │   └── Settings.tsx
│       ├── components/
│       │   ├── Sidebar.tsx
│       │   ├── KillSwitchBar.tsx
│       │   ├── ModelRegistryTable.tsx
│       │   ├── ModelInspectorDrawer.tsx
│       │   ├── RDAgentTracePanel.tsx
│       │   ├── EnsembleWeightsViz.tsx
│       │   ├── ... (all other components from v4 spec)
│       │   └── ui/            (Card, Button, Pill, Toggle, Slider, Input)
│       └── store/index.ts
├── scripts/
│   ├── weekly_retrain.sh        (Sunday 02:00 CT)
│   ├── rdagent_loop.sh          (nightly 23:00 CT)
│   └── backup_models.sh
├── logs/
├── secrets/
│   ├── openai.key
│   └── api_token.txt
├── deploy/
│   ├── systemd/
│   │   ├── vl-bridge.service
│   │   ├── vl-nautilus.service
│   │   ├── vl-backend.service
│   │   ├── vl-redis.service
│   │   ├── vl-journal.service
│   │   ├── vl-qlib-retrain.service (one-shot, triggered by timer)
│   │   ├── vl-qlib-retrain.timer
│   │   ├── vl-rdagent-discover.service
│   │   └── vl-rdagent-discover.timer
│   └── nginx/vl.conf
└── README.md

## 25. Process Management (systemd)

# /etc/systemd/system/vl-bridge.service
[Unit]
Description=VL NinjaTrader 8 socket bridge
After=network.target redis-server.service
[Service]
Type=simple
User=hoang
WorkingDirectory=/home/hoang/vl_trading
ExecStart=/home/hoang/vl_trading/.venvs/bridge_env/bin/python \
-m src.bridge.nt_socket_client
Restart=on-failure
RestartSec=5s
StandardOutput=append:/home/hoang/vl_trading/logs/bridge.log
StandardError=append:/home/hoang/vl_trading/logs/bridge.err
[Install]
WantedBy=multi-user.target
# /etc/systemd/system/vl-nautilus.service
[Unit]
Description=VL NautilusTrader strategy
After=network.target redis-server.service vl-bridge.service
[Service]
Type=simple
User=hoang
WorkingDirectory=/home/hoang/vl_trading
Environment="REDIS_URL=redis://localhost:6379"
ExecStart=/home/hoang/vl_trading/.venvs/nautilus_env/bin/python \
-m src.strategy.run_live --config configs/strategies/active.yaml
Restart=on-failure
RestartSec=10s
StandardOutput=append:/home/hoang/vl_trading/logs/strategy.log
StandardError=append:/home/hoang/vl_trading/logs/strategy.err
[Install]
WantedBy=multi-user.target
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

## 26. Reverse Proxy (nginx)

# /etc/nginx/sites-available/vl.conf
server {
listen 80;
server_name vl.local;
# Frontend static (built React)
root /home/hoang/vl_trading/frontend/dist;
index index.html;
try_files $uri $uri/ /index.html;
# API
location /api/ {
proxy_pass http://127.0.0.1:8000;
proxy_set_header Host $host;
proxy_set_header X-Real-IP $remote_addr;
}
# WebSocket
location /ws {
proxy_pass http://127.0.0.1:8000;
proxy_http_version 1.1;
proxy_set_header Upgrade $http_upgrade;
proxy_set_header Connection "upgrade";
proxy_read_timeout 86400;
}
# Journal screenshots
location /journal-media/ {
alias /home/hoang/vl_trading/journal/;
autoindex off;
}
}

## 27. Remote Access (Tailscale)

- Install Tailscale on the workstation + your phone
- Visit https://<your-host>.tailnet/ from any device on your tailnet
- No public exposure; auth is your Tailscale identity + the API_TOKEN bearer
- If you ever want public access: put a real auth proxy in front (Authelia / Caddy + OIDC). Not part of this build.

## 28. Scheduler (cron + systemd timers)

Two periodic tasks: Sunday weekly retrain and nightly RD-Agent discovery. Both implemented as systemd timer units (more reliable than cron, integrate with journalctl).

### 28.1 Weekly retrain — Sunday 02:00 CT

# /etc/systemd/system/vl-qlib-retrain.service
[Unit]
Description=VL Qlib weekly retrain (all 35 models)
After=network.target redis-server.service
[Service]
Type=oneshot
User=hoang
WorkingDirectory=/home/hoang/vl_trading
EnvironmentFile=/home/hoang/vl_trading/.env
ExecStart=/home/hoang/vl_trading/scripts/weekly_retrain.sh
StandardOutput=append:/home/hoang/vl_trading/logs/retrain.log
StandardError=append:/home/hoang/vl_trading/logs/retrain.err
TimeoutStartSec=6h
# /etc/systemd/system/vl-qlib-retrain.timer
[Unit]
Description=Trigger weekly Qlib retrain every Sunday 02:00 CT
Requires=vl-qlib-retrain.service
[Timer]
# Sunday 02:00 America/Chicago
OnCalendar=Sun *-*-* 02:00:00
Persistent=true
RandomizedDelaySec=300
[Install]
WantedBy=timers.target

### 28.2 Nightly RD-Agent discovery — Daily 23:00 CT

# /etc/systemd/system/vl-rdagent-discover.service
[Unit]
Description=VL RD-Agent nightly factor discovery
After=network.target redis-server.service
[Service]
Type=oneshot
User=hoang
WorkingDirectory=/home/hoang/vl_trading
EnvironmentFile=/home/hoang/vl_trading/.env
ExecStart=/home/hoang/vl_trading/scripts/rdagent_loop.sh
StandardOutput=append:/home/hoang/vl_trading/logs/rdagent.log
StandardError=append:/home/hoang/vl_trading/logs/rdagent.err
TimeoutStartSec=3h
# /etc/systemd/system/vl-rdagent-discover.timer
[Unit]
Description=Trigger RD-Agent factor discovery nightly at 23:00 CT
Requires=vl-rdagent-discover.service
[Timer]
OnCalendar=*-*-* 23:00:00
Persistent=true
RandomizedDelaySec=600
[Install]
WantedBy=timers.target

### 28.3 Enable and verify

# Enable both timers
sudo systemctl daemon-reload
sudo systemctl enable --now vl-qlib-retrain.timer
sudo systemctl enable --now vl-rdagent-discover.timer
# Verify both are scheduled
systemctl list-timers --all | grep vl-
# Manual trigger (to test without waiting)
sudo systemctl start vl-qlib-retrain.service
sudo systemctl start vl-rdagent-discover.service
# Watch logs
journalctl -u vl-qlib-retrain -f
journalctl -u vl-rdagent-discover -f

### 28.4 Budget caps

- RD-Agent script enforces --budget_usd 5.00 per run (hard cap on LLM spend)
- Weekly retrain has 6-hour timeout (kill if it hangs)
- If retrain takes longer than 4 hours consistently, reduce model count or split timers per algorithm
- Both timers respect RandomizedDelaySec to avoid colliding with NT8 market-open prep

# Appendices

Reference material referenced by the main spec. Read on demand.

## Appendix A — Alpha158 / Alpha360 Factor Reference

Qlib's prebuilt factor libraries replace the hand-coded 8-factor bias scorer. Alpha158 ships 158 engineered factors covering K-line shape, momentum, volume, volatility, and rolling statistics. Alpha360 extends this to 360 factors including longer rolling windows. Source: qlib/contrib/data/handler.py

### A.1 Factor families in Alpha158


| **Family** | **Count** | **Examples** |
| --- | --- | --- |
| **K-line shape** | **9** | **KMID (close/open-1), KLEN (high-low/open), KSFT (close-(high+low)/2/open), KUP, KLOW, KMID2, KUP2, KLOW2, KSFT2** |
| **OHLCV ratios** | **4** | **OPEN0 (open/close), HIGH0, LOW0, VWAP0 — current bar values normalized by close** |
| **Rolling momentum (ROC)** | **18** | **ROC5, ROC10, ROC20, ROC30, ROC60 across 6 base columns (close, open, high, low, vwap, volume) — rate of change over N bars** |
| **Rolling mean (MA)** | **12** | **MA5, MA10, MA20, MA30 — close / mean(close, N) — current price vs N-bar SMA** |
| **Rolling std (STD)** | **12** | **STD5, STD10, STD20, STD30 — std(close, N) / close — volatility** |
| **Rolling beta** | **6** | **BETA5-60 — regression slope (close vs index over N bars) / close** |
| **Rolling max/min** | **12** | **MAX5-60, MIN5-60 — highest/lowest close ratio over N bars** |
| **Rolling quantile** | **12** | **QTLU5-60 (upper quantile), QTLD5-60 (lower quantile) — close percentile in window** |
| **Rolling rank** | **6** | **RANK5-60 — close's rank in last N bars / N** |
| **Rolling correlation** | **12** | **CORR5-60, CORD5-60 — corr(close, volume) and corr(close, log_volume_diff)** |
| **Counts up/down** | **12** | **CNTP5-60 (count of up bars), CNTN5-60 (count of down bars), CNTD5-60 (net direction count)** |
| **Volume features** | **28** | **VMA5-60 (volume MA), VSTD5-60 (volume std), WVMA5-60 (weighted volume MA), VSUMP/N/D5-60 (volume sum positive/negative/net)** |
| **Rolling sum of returns** | **15** | **SUMP5-60 (sum positive returns), SUMN, SUMD — directional accumulation** |
| **Residual momentum** | **0+** | **RESI5-60 — residual after detrending; not in default Alpha158 but trivial to add** |


### A.2 How features replace the old 8-factor bias


| **Old hand-coded factor** | **Replaced by (Alpha158)** |
| --- | --- |
| **HTF trend (4H + 1D)** | **Run the same model on the 1h and 1d timeframes — ensemble combines them naturally** |
| **Session (NY AM = +5)** | **Added as a custom calendar feature in Alpha158 + filter on bar_in_session** |
| **Active FVG** | **Captured implicitly by KMID/KSFT/KLEN K-line features + ROC/momentum — RD-Agent can add explicit FVG factor if it improves IC** |
| **Active OB** | **Captured by volume-weighted features (WVMA, VSUMP, VSTD) + price-volume correlation (CORR, CORD)** |
| **BOS direction** | **MAX5-60, MIN5-60, RANK5-60 — break-of-structure shows up as rank flipping** |
| **Liquidity sweep** | **CNTP/CNTN reversal + abnormal volume (VSTD spike) + price wick (KUP/KLOW) — multi-factor signature** |
| **VWAP position** | **VWAP0 (current vwap ratio) + ROC of VWAP** |
| **Time-of-day quality** | **Custom feature: hour_of_day + minute_of_day added to handler config** |


### A.3 Adding custom features (ICT-specific) to Alpha158

If you want to keep ICT structural features (e.g. explicit FVG detector) in the model, subclass Alpha158 and add expressions. Qlib's expression engine supports rolling operators on OHLCV columns.
# src/qlib/handlers/alpha158_ict.py
from qlib.contrib.data.handler import Alpha158
from qlib.data.dataset.handler import DataHandlerLP
class Alpha158_ICT(Alpha158):
"""Alpha158 + custom ICT-style features."""
@staticmethod
def get_feature_config(config={}):
# Inherit Alpha158 defaults
fields, names = Alpha158.get_feature_config(config)
# Add ICT-style features as Qlib expressions
ict_fields = [
# FVG proxy: 3-bar gap between bar[-2].high and bar[0].low
"Ref($low, 0) - Ref($high, 2)",
# Order block proxy: high-volume reversal bar 5-15 ago
"Ref($volume, 10) / Mean($volume, 20)",
# Liquidity sweep proxy: wick exceeding 1.5x body
"($high - $low) / Abs($close - $open + 1e-6)",
# Session indicator (hour 8-10 CT = NY AM)
"If(Hour($time) >= 8 & Hour($time) <= 10, 1, 0)",
]
ict_names = ["FVG_GAP", "OB_VOL_RATIO", "WICK_BODY", "NY_AM_FLAG"]
return fields + ict_fields, names + ict_names

### A.4 Recommended starting feature set per timeframe


| **Timeframe** | **Feature set** | **Rationale** |
| --- | --- | --- |
| **1m, 2m, 3m** | **Alpha158** | **Short-window features (5/10/20-bar) dominate; longer windows add noise at 1-3 min** |
| **5m** | **Alpha158_ICT (custom)** | **Sweet spot for ICT structure; add explicit FVG/OB/sweep proxies** |
| **1h** | **Alpha360** | **Longer windows (60-bar) are meaningful; richer Alpha360 helps capture intraday trend** |
| **1d, 1w** | **Alpha360** | **Macro context; Alpha360's extended rolling windows fit naturally** |


### A.5 Information Coefficient (IC) interpretation

Qlib reports IC for every feature and model. Use these thresholds to evaluate factor quality:
- IC > 0.05 — excellent signal (rare; treat with suspicion of overfitting)
- IC 0.03-0.05 — good signal worth keeping
- IC 0.02-0.03 — marginal; keep in ensemble but with low weight
- IC < 0.02 — noise; drop the factor
- Rank IC (rank correlation) is more robust than Pearson IC for non-linear models like LightGBM

## Appendix B — Qlib Model Registry (5 algos × 7 timeframes)

The 35-model registry that replaces the old ICT setup catalog. Each row is one Qlib workflow YAML in configs/qlib/. Trained weekly by vl-qlib-retrain.timer (Sun 02:00 CT). Predictions consumed by the NautilusTrader ensemble strategy.

### B.1 Algorithm survey


| **Algorithm** | **Qlib class** | **Train time** | **Strengths** |
| --- | --- | --- | --- |
| **LightGBM** | **qlib.contrib.model.gbdt.LGBModel** | **~30s/year** | **Fast baseline; handles tabular Alpha158 well; minimal hyperparameter tuning** |
| **XGBoost** | **qlib.contrib.model.xgboost.XGBModel** | **~45s/year** | **Different tree algorithm than LightGBM; adds diversity to ensemble** |
| **CatBoost** | **qlib.contrib.model.catboost.CatBoostModel** | **~60s/year** | **Better with categorical features (hour-of-day, session); robust against overfitting** |
| **LSTM** | **qlib.contrib.model.pytorch_lstm.LSTM** | **~20min/year (GPU)** | **Captures sequential dependencies in Alpha360; needs more data than trees** |
| **Transformer** | **qlib.contrib.model.pytorch_transformer.Transformer** | **~45min/year (GPU)** | **Attention-based; strongest on longer sequences (1h/1d/1w); hungry for data** |


### B.2 The 35-model registry

One row per (algorithm, timeframe) pair. Each gets its own Qlib YAML and trains independently.

| **#** | **Model ID** | **Algorithm** | **Timeframe** | **Features** | **Target** |
| --- | --- | --- | --- | --- | --- |
| **1** | **lightgbm_1m** | **LightGBM** | **1m** | **Alpha158** | **5-bar fwd return** |
| **2** | **lightgbm_2m** | **LightGBM** | **2m** | **Alpha158** | **5-bar fwd return** |
| **3** | **lightgbm_3m** | **LightGBM** | **3m** | **Alpha158** | **5-bar fwd return** |
| **4** | **lightgbm_5m** | **LightGBM** | **5m** | **Alpha158_ICT** | **5-bar fwd return** |
| **5** | **lightgbm_1h** | **LightGBM** | **1h** | **Alpha360** | **3-bar fwd return** |
| **6** | **lightgbm_1d** | **LightGBM** | **1d** | **Alpha360** | **2-bar fwd return** |
| **7** | **lightgbm_1w** | **LightGBM** | **1w** | **Alpha360** | **1-bar fwd return** |
| **8** | **xgboost_1m** | **XGBoost** | **1m** | **Alpha158** | **5-bar fwd return** |
| **9** | **xgboost_2m** | **XGBoost** | **2m** | **Alpha158** | **5-bar fwd return** |
| **10** | **xgboost_3m** | **XGBoost** | **3m** | **Alpha158** | **5-bar fwd return** |
| **11** | **xgboost_5m** | **XGBoost** | **5m** | **Alpha158_ICT** | **5-bar fwd return** |
| **12** | **xgboost_1h** | **XGBoost** | **1h** | **Alpha360** | **3-bar fwd return** |
| **13** | **xgboost_1d** | **XGBoost** | **1d** | **Alpha360** | **2-bar fwd return** |
| **14** | **xgboost_1w** | **XGBoost** | **1w** | **Alpha360** | **1-bar fwd return** |
| **15** | **catboost_1m** | **CatBoost** | **1m** | **Alpha158** | **5-bar fwd return** |
| **16** | **catboost_2m** | **CatBoost** | **2m** | **Alpha158** | **5-bar fwd return** |
| **17** | **catboost_3m** | **CatBoost** | **3m** | **Alpha158** | **5-bar fwd return** |
| **18** | **catboost_5m** | **CatBoost** | **5m** | **Alpha158_ICT** | **5-bar fwd return** |
| **19** | **catboost_1h** | **CatBoost** | **1h** | **Alpha360** | **3-bar fwd return** |
| **20** | **catboost_1d** | **CatBoost** | **1d** | **Alpha360** | **2-bar fwd return** |
| **21** | **catboost_1w** | **CatBoost** | **1w** | **Alpha360** | **1-bar fwd return** |
| **22** | **lstm_1m** | **LSTM** | **1m** | **Alpha158** | **5-bar fwd return** |
| **23** | **lstm_2m** | **LSTM** | **2m** | **Alpha158** | **5-bar fwd return** |
| **24** | **lstm_3m** | **LSTM** | **3m** | **Alpha158** | **5-bar fwd return** |
| **25** | **lstm_5m** | **LSTM** | **5m** | **Alpha360** | **5-bar fwd return** |
| **26** | **lstm_1h** | **LSTM** | **1h** | **Alpha360** | **3-bar fwd return** |
| **27** | **lstm_1d** | **LSTM** | **1d** | **Alpha360** | **2-bar fwd return** |
| **28** | **lstm_1w** | **LSTM** | **1w** | **Alpha360** | **1-bar fwd return** |
| **29** | **transformer_1m** | **Transformer** | **1m** | **Alpha158** | **5-bar fwd return** |
| **30** | **transformer_2m** | **Transformer** | **2m** | **Alpha158** | **5-bar fwd return** |
| **31** | **transformer_3m** | **Transformer** | **3m** | **Alpha158** | **5-bar fwd return** |
| **32** | **transformer_5m** | **Transformer** | **5m** | **Alpha360** | **5-bar fwd return** |
| **33** | **transformer_1h** | **Transformer** | **1h** | **Alpha360** | **3-bar fwd return** |
| **34** | **transformer_1d** | **Transformer** | **1d** | **Alpha360** | **2-bar fwd return** |
| **35** | **transformer_1w** | **Transformer** | **1w** | **Alpha360** | **1-bar fwd return** |


### B.3 Per-model storage layout

models/
├── lightgbm_5m/
│   ├── v2026.05.18/
│   │   ├── model.pkl          (LightGBM Booster)
│   │   ├── metadata.json      (train range, IC, Sharpe, features used)
│   │   ├── workflow_config.yaml  (snapshot of the YAML used)
│   │   └── feature_importance.csv
│   ├── v2026.05.11/
│   └── current → v2026.05.18/  (symlink for fast lookup)
├── transformer_1h/
│   ├── v2026.05.18/
│   │   └── model.pt           (PyTorch state_dict)
│   └── current → v2026.05.18/
└── ...

### B.4 Ensemble weights

Each model's contribution to the final signal is weighted by its rolling 30-day live Sharpe ratio. Recomputed every Sunday after retrain.
# src/qlib/update_ensemble_weights.py
import json, redis, glob
from datetime import datetime, timedelta
import numpy as np
import pandas as pd
REDIS = redis.Redis(host='localhost', port=6379, decode_responses=True)
def compute_weights():
weights = {}
for model_dir in glob.glob("models/*/current"):
model_id = model_dir.split("/")[-2]
# Load 30-day live predictions vs actuals
log_path = f"logs/predictions/{model_id}_30d.csv"
if not os.path.exists(log_path):
continue
df = pd.read_csv(log_path)
returns = df["actual"] * np.sign(df["pred"])  # signed returns
sharpe = returns.mean() / (returns.std() + 1e-9) * np.sqrt(252)
# Floor weight at 0 (no negative weights)
weights[model_id] = max(0.0, sharpe)
# Normalize to sum=1.0
total = sum(weights.values()) or 1.0
weights = {k: v / total for k, v in weights.items()}
# Publish
REDIS.set("vl:ensemble:weights", json.dumps(weights))
REDIS.publish("vl:alerts", json.dumps({
"level": "INFO",
"msg": f"Ensemble weights updated: {len(weights)} models active"
}))
return weights
if __name__ == "__main__":
compute_weights()

### B.5 Adding RD-Agent-discovered models

RD-Agent writes new model definitions to rdagent_workspace/discoveries/<date>/. The weekly retrain script picks up any new YAML files matching the pattern rdagent_*.yaml and adds them to the training queue. Discovered models become numbered #36, #37, ... in the registry.

## Appendix C — Risk Math


### C.1 Position sizing formula

# Account-based risk sizing for NQ / MNQ
def position_size(account_balance, risk_pct, stop_distance_points, instrument):
"""
Computes contract quantity that risks exactly risk_pct of account.
For NQ:  $20 per point per contract
For MNQ: $2 per point per contract
"""
point_value = {"NQ": 20.0, "MNQ": 2.0}[instrument]
dollar_risk_per_contract = stop_distance_points * point_value
dollar_budget = account_balance * (risk_pct / 100.0)
qty = int(dollar_budget // dollar_risk_per_contract)
# Enforce per-strategy max and per-account max
qty = min(qty, params.max_contracts_per_strategy, risk_rules.max_contracts)
# Day-1 live cap (Phase 6 acceptance criterion)
if first_live_day:
qty = min(qty, 1)
return max(qty, 0)

### C.2 Stop placement rules


| **Setup** | **Stop rule** |
| --- | --- |
| **FVG** | **Below FVG bottom (long) / above FVG top (short), wider of: 1 tick beyond FVG OR entry ± 2.5×ATR** |
| **LIQ_SWEEP_FVG** | **Below sweep candle low (long) / above sweep candle high (short) — deeper, higher conviction** |
| **OB_RETEST** | **1 tick beyond OB outer edge** |
| **OB_CONTINUATION** | **Below post-BOS swing low (long) / above swing high (short)** |
| **BOS_PULLBACK** | **Beyond 78.6% Fibonacci of BOS impulse** |
| **EQ_HL_LIQ** | **2-3 ticks beyond the sweep wick extreme** |


### C.3 Target and R:R logic

- Default target distance: stop_distance × params.target_rr (default 2.5×)
- Override for LIQ_SWEEP_FVG: 3.0× (premium setup, lower frequency, wider target)
- Override for BOS_PULLBACK: 2.0× (lowest conviction)
- Target clipped at nearest opposite liquidity pool if closer than R:R target
- If target_at_liquidity < 1.5× stop_distance → SKIP setup (R:R too poor)

### C.4 Trailing stop ladder

Applied uniformly across all setups unless setup-specific override:

| **Trigger** | **Action** |
| --- | --- |
| **Price reaches +1.0R** | **Move SL to break-even (entry price)** |
| **Price reaches +1.5R** | **Move SL to entry + 0.25R (lock minimum profit)** |
| **Price reaches +2.0R** | **Begin ATR-based trail: SL = max(SL, current_price - 1.5×ATR)** |
| **Price reaches +2.5R (target)** | **TP hit — close** |
| **Time-based exit** | **Hard close at 11:30 CT regardless of P&L (NY AM session ends)** |


### C.5 News blackout behavior

- HIGH-impact event: no new entries from T-5 min to T+5 min
- HIGH-impact event: existing positions STAY OPEN — stops protect them
- MED-impact event: no new entries from T-2 min to T+2 min; existing positions stay open
- LOW-impact event: no blackout
- FOMC / NFP override: existing positions FORCED CLOSED at T-2 min (flatten before event)

### C.6 Consecutive loss lockout state machine

# Tracking
state = {
"consecutive_losses": 0,
"lockout_until": None,
"lockout_duration_min": 30,
}
# After each closed trade:
def on_trade_closed(trade):
if trade.pnl < 0:
state["consecutive_losses"] += 1
if state["consecutive_losses"] >= 3:
state["lockout_until"] = now() + timedelta(minutes=30)
log_alert("LOCKOUT: 3 consecutive losses, halted for 30 min")
publish_event("strategy_locked")
else:
state["consecutive_losses"] = 0  # reset on any win
# Before each new entry:
def can_trade():
if state["lockout_until"] and now() < state["lockout_until"]:
return False
return True

### C.7 Daily loss / max DD enforcement


| **Threshold** | **Action** | **Recovery** |
| --- | --- | --- |
| **Daily loss at 75%** | **Alert only** | **Notification panel; reduce size on new entries (½ qty)** |
| **Daily loss at 90%** | **Halt new entries** | **Open positions can run; resume at next session** |
| **Daily loss at 100%** | **Flatten all** | **Auto-flatten, halt for remainder of day** |
| **Max DD at 75%** | **Alert only** | **Notification; reduce risk_pct to 50% of configured** |
| **Max DD at 90%** | **Auto-stop** | **Flatten + halt; require manual unlock via Emergency Stop screen** |


## Appendix D — Data Sources & API Keys


### D.1 News calendar — ForexFactory scraping

Decision: ForexFactory. Free, comprehensive US economic events, well-known XML feed. Requires scraping with care (rate limit + cache).

#### Implementation

# src/news/forexfactory_client.py
import httpx, json, asyncio
from datetime import datetime, timezone
from bs4 import BeautifulSoup
import redis.asyncio as redis
FF_URL = "https://nfs.faireconomy.media/ff_calendar_thisweek.json"  # public JSON feed
USER_AGENT = "VL/3.0 (personal-use)"
CACHE_TTL_SEC = 900  # refresh every 15 min
async def fetch_calendar():
async with httpx.AsyncClient(timeout=30, headers={"User-Agent": USER_AGENT}) as cli:
r = await cli.get(FF_URL)
r.raise_for_status()
return r.json()
# Filter to US-only HIGH/MED events for NQ trading
US_HIGH_EVENTS = {
"FOMC", "Non-Farm Employment Change", "CPI", "Core CPI", "PCE",
"Powell", "Yellen", "Unemployment Rate", "Retail Sales",
"GDP", "ISM Manufacturing", "ISM Services", "FOMC Minutes"
}
def filter_relevant(events):
return [
e for e in events
if e.get("country") == "USD"
and (e.get("impact") in ("High", "Medium"))
]
async def refresh_loop(r):
while True:
try:
events = filter_relevant(await fetch_calendar())
await r.set("vl:news:week", json.dumps(events), ex=CACHE_TTL_SEC)
await r.publish("vl:news:updated", str(len(events)))
except Exception as e:
logger.error(f"FF fetch failed: {e}")
await asyncio.sleep(900)  # 15 min

#### Fallback if ForexFactory feed breaks

- Scrape the HTML calendar page directly (selectors stable across years)
- Switch to Finnhub free tier (60 req/min) — already coded as alt
- Manual entry via Settings page (UI form to add ad-hoc events)

### D.2 Historical data — decision matrix

Deferred — pick one when starting Phase 3 backtest:

| **Source** | **Cost/mo** | **Format** | **Pros / Cons** |
| --- | --- | --- | --- |
| **Databento** | **~$125** | **Parquet, CSV, MBO** | **PRO: clean, well-documented, modern API, no embargo. CON: paid.** |
| **IQFeed** | **~$165** | **Native, CSV** | **PRO: depth + history bundle, deep history. CON: paid, Windows-friendly only, older API.** |
| **NT8 export** | **Free** | **CSV export** | **PRO: free, you already have NT8. CON: manual export per day, no tick-level history, limited backfill.** |
| **Polygon.io** | **$29-200** | **REST + WS** | **PRO: cheap tier exists, modern. CON: futures coverage thinner than equities.** |

Recommendation: start with NT8 export for Phase 3 (free, sufficient for first 1-month backtest). Upgrade to Databento before Phase 6 (paper → live) when you need clean, full-year, programmatic data for walk-forward CV.

### D.3 .env file template

# /home/hoang/vl_trading/.env
# ─── Core ─────────────────────────────────────
VL_API_TOKEN=<generate with: openssl rand -hex 32>
REDIS_URL=redis://localhost:6379
ENVIRONMENT=development  # development | paper | live
# ─── NinjaTrader 8 ───────────────────────────
NT8_SOCKET_HOST=host.docker.internal  # or Windows host IP
NT8_SOCKET_PORT=9001
NT8_SIGNALS_CSV=/mnt/c/trading/signals.csv
# ─── Data ────────────────────────────────────
DATA_PROVIDER=nt8_export  # nt8_export | databento | iqfeed
DATABENTO_API_KEY=  # fill if using Databento
IQFEED_USER=
IQFEED_PASS=
# ─── News ────────────────────────────────────
NEWS_PROVIDER=forexfactory
FOREXFACTORY_FEED_URL=https://nfs.faireconomy.media/ff_calendar_thisweek.json
# Optional fallback
FINNHUB_API_KEY=
# ─── ML (Phase 7 optional) ──────────────────
XGB_MODEL_DIR=/home/hoang/vl_trading/models
RETRAIN_SCHEDULE=0 3 * * MON  # 3 AM CT Mondays
# ─── Logging ────────────────────────────────
LOG_LEVEL=INFO
LOG_DIR=/home/hoang/vl_trading/logs
# ─── Frontend ───────────────────────────────
VITE_API_URL=http://localhost:8000
VITE_WS_URL=ws://localhost:8000/ws

## Appendix E — NinjaTrader Integration Resolution


### E.1 Claude-Trader license issue

J0shusmc/Claude-Trader-NinjaTrader has no LICENSE file in the repo, which means it's effectively all-rights-reserved by default. Three paths:
- Open a GitHub issue asking the author to add a permissive license (MIT/Apache 2.0). If they agree, use it as-is.
- Fork it as a clean-room reimplementation under your own MIT license (the CSV polling pattern is trivial — ~150 lines of NinjaScript).
- Replace entirely with a custom NinjaScript add-on (recommended for production).
Recommendation: option 3. Write your own. Below is the minimal spec.

### E.2 Custom NinjaScript order receiver (VLBridge.cs)

// VLBridge.cs - place in: Documents/NinjaTrader 8/bin/Custom/AddOns/
// Polls signals.csv every 2 seconds, places bracket orders, writes status back.
using System;
using System.IO;
using System.Threading.Tasks;
using NinjaTrader.Cbi;
using NinjaTrader.NinjaScript;
using NinjaTrader.NinjaScript.AddOns;
public class VLBridge : NTAddOn
{
private const string CSV_PATH = @"C:\trading\signals.csv";
private const int POLL_MS = 2000;
private Account account;
private System.Threading.Timer timer;
protected override void OnStateChange()
{
if (State == State.SetDefaults) { Name = "VL Bridge"; }
else if (State == State.Active)
{
account = Account.All.FirstOrDefault(a => a.Name == "Sim101");  // configurable
timer = new System.Threading.Timer(_ => Poll(), null, 0, POLL_MS);
}
else if (State == State.Terminated)
{
timer?.Dispose();
}
}
private void Poll()
{
if (!File.Exists(CSV_PATH)) return;
var lines = File.ReadAllLines(CSV_PATH).ToList();
if (lines.Count <= 1) return;  // header only
bool modified = false;
for (int i = 1; i < lines.Count; i++)
{
var parts = lines[i].Split(',');
if (parts.Length < 10 || parts[9] != "NEW") continue;
string action = parts[2];
string symbol = parts[3];
int qty = int.Parse(parts[4]);
double entry = double.Parse(parts[5]);
double stop = double.Parse(parts[6]);
double target = double.Parse(parts[7]);
try {
var instrument = Instrument.GetInstrument(symbol);
OrderAction orderAction = action == "BUY" ? OrderAction.Buy : OrderAction.Sell;
account.CreateOrder(instrument, orderAction, OrderType.Market,
OrderEntry.Manual, TimeInForce.Day,
qty, 0, 0, "", "vl_" + parts[1], account.SubmitOrder);
// TODO: attach OCO stop/target as separate orders
parts[9] = "FILLED";
modified = true;
} catch (Exception ex) {
parts[9] = "REJECTED:" + ex.Message;
modified = true;
}
lines[i] = string.Join(",", parts);
}
if (modified) File.WriteAllLines(CSV_PATH, lines);
}
}

### E.3 Test harness

Before wiring the real strategy, verify the bridge with a manual ping/pong:
# tools/test_bridge.py
import csv
from datetime import datetime, timezone
def write_test_signal():
row = [
datetime.now(timezone.utc).isoformat(),
"test_001",
"BUY", "NQ 06-26",
1, 21500.0, 21450.0, 21600.0,
"TEST", "NEW",
]
with open("/mnt/c/trading/signals.csv", "a", newline="") as f:
w = csv.writer(f)
w.writerow(row)
print("Wrote test signal. Watch NT8 for the order.")
if __name__ == "__main__":
write_test_signal()

### E.4 Pinned versions

- NinjaTrader 8: latest stable (8.1.x as of build date)
- ninja-socket: pin to commit hash, not a branch (the repo is small)
- Your VLBridge.cs: version it in src/nt8/ inside your repo
- Test bridge round-trip after every NT8 upgrade

## Appendix F — First Qlib Workflow Build Plan

Phase 2 deliverable. The first end-to-end Qlib workflow: convert NQ data, train LightGBM with Alpha158, evaluate, and verify the prediction is consumable by the NautilusTrader strategy. This proves the entire pipeline before scaling to 35 models.

### F.1 File / artifact build order


| **#** | **Artifact** | **Purpose** | **Acceptance test** |
| --- | --- | --- | --- |
| **1** | **scripts/install_qlib.sh** | **Install pyqlib + deps** | **uv pip install pyqlib lightgbm xgboost catboost; qrun --help works** |
| **2** | **src/qlib/convert_parquet_to_qlib.py** | **Convert NT8 Parquet → Qlib binary** | **Run on 1 year of NQ 5m Parquet → ~/.qlib/qlib_data/nq_5m/ populated; D.features() returns valid OHLCV** |
| **3** | **configs/qlib/lightgbm_5m.yaml** | **First workflow YAML** | **File parses with no errors; references valid Qlib classes** |
| **4** | **Run: qrun configs/qlib/lightgbm_5m.yaml** | **Train + backtest** | **Completes without error; mlflow run artifact saved; IC reported in stdout** |
| **5** | **Jupyter notebook: notebooks/inspect_lightgbm_5m.ipynb** | **Render report** | **Equity curve, IC distribution, top features all render via Qlib's analysis tooling** |
| **6** | **src/qlib/save_to_redis.py** | **Publish predictions** | **Latest model prediction appears at vl:predictions:lightgbm_5m:<ts> in Redis** |
| **7** | **src/strategy/qlib_consumer.py** | **Read prediction from Redis** | **latest_signal() returns float in [-1, +1]; non-zero on at least 10% of bars** |
| **8** | **src/strategy/nq_qlib_ensemble.py** | **Trade on prediction** | **NautilusTrader backtest with single model fires bracket orders on signals > threshold** |
| **9** | **tests/test_qlib_conversion.py** | **Unit tests** | **Parquet → Qlib round-trips identical OHLCV values** |
| **10** | **tests/test_ensemble_consumer.py** | **Unit tests** | **Mock Redis → consumer correctly weights and clips signal** |


### F.2 Step-by-step commands (copy-paste runnable)

# 1. Install Qlib environment
cd ~/vl_trading
uv venv .venvs/qlib_env --python 3.11
source .venvs/qlib_env/bin/activate
uv pip install "pyqlib>=0.9.7" lightgbm xgboost catboost \
pandas pyarrow torch redis mlflow
# 2. Convert NT8 Parquet to Qlib binary (1 year NQ 5m)
python -m src.qlib.convert_parquet_to_qlib \
--parquet data/parquet/nq_5m.parquet \
--out ~/.qlib/qlib_data/nq_5m \
--instrument NQ \
--start 2025-05-01 --end 2026-05-01
# 3. Run the first workflow
qrun configs/qlib/lightgbm_5m.yaml
# 4. Inspect results in Jupyter
jupyter notebook notebooks/inspect_lightgbm_5m.ipynb
# 5. Save predictions to Redis (for the live strategy to consume)
python -m src.qlib.save_to_redis \
--model_id lightgbm_5m \
--model_path models/lightgbm_5m/current/model.pkl
# 6. Run NautilusTrader backtest using these predictions
python -m src.strategy.run_backtest_qlib \
--strategy nq_qlib_ensemble \
--config configs/strategy_single_model.yaml

### F.3 Acceptance thresholds for Phase 2 → Phase 3 gate

Before scaling to all 35 models, the single-model workflow must meet:
- Out-of-sample IC ≥ 0.02 (Pearson) AND ≥ 0.025 (Rank IC)
- Backtest Sharpe ≥ 0.8 on 1-year OOS
- Profit factor ≥ 1.3
- Max drawdown ≤ $2,000 on $150K account
- Trade count ≥ 50 over the OOS year (proves the threshold isn't too tight)
- Redis publish round-trip latency < 200ms
- NautilusTrader backtest reproduces same Sharpe ±10% (parity check)

### F.4 Definition of done

Phase 2 is "done" when ALL of these are true:
- All 10 artifacts built and tested
- All 7 thresholds in F.3 met
- Documentation: notebooks/inspect_lightgbm_5m.ipynb has a written summary at the top explaining the result
- Reproducibility: a fresh clone + `make phase2` produces the same numbers
- Git tag: v5.0.0-qlib-phase2

## Appendix G — Default Configuration Files


### G.1 configs/strategy.yaml (NautilusTrader runtime)

# Live strategy parameters consumed by NQ_Qlib_Ensemble
name: NQ_Qlib_Ensemble
version: v5
instrument: NQ 06-26
# Risk
risk_pct: 0.5              # % of account per trade
stop_atr_mult: 2.5         # stop = N × ATR(14)
target_rr: 2.5             # target = R-multiple of stop
max_contracts_per_strategy: 3
trade_size_default: 1
# Ensemble signal threshold
signal_threshold: 0.3      # require |ensemble_signal| >= this to fire
signal_threshold_release: 0.15  # hysteresis release threshold
long_only: false
# Killzones (CT)
killzones:
- { name: London,  start: "02:00", end: "05:00" }
- { name: NY_AM,   start: "08:30", end: "11:30" }
- { name: NY_PM,   start: "13:00", end: "15:00" }
# Models (which timeframes contribute to the ensemble)
ensemble:
timeframes: [1m, 2m, 3m, 5m, 1h, 1d, 1w]
algorithms: [lightgbm, xgboost, catboost, lstm, transformer]
weights_source: redis    # vl:ensemble:weights (auto-tuned)
fallback: equal_weight   # if Redis empty, use 1/35 each
# Trailing
trail_be_at_r: 1.0
trail_lock_at_r: 1.5
trail_atr_at_r: 2.0
trail_atr_mult: 1.5
# Filters
news_blackout_minutes: 5   # HIGH events
session_close_flatten: "11:30"  # CT, NY AM end

### G.1b configs/qlib/lightgbm_5m.yaml (one of 35 workflow files)

# Qlib workflow — see Part III §20.2 for full template
qlib_init:
provider_uri: "~/.qlib/qlib_data/nq_5m"
region: us
market: &market all
benchmark: NQ
task:
model:
class: LGBModel
module_path: qlib.contrib.model.gbdt
kwargs:
loss: mse
num_leaves: 64
learning_rate: 0.05
n_estimators: 500
dataset:
class: DatasetH
module_path: qlib.data.dataset
kwargs:
handler:
class: Alpha158_ICT
module_path: src.qlib.handlers.alpha158_ict
kwargs:
start_time: 2022-01-01
end_time: 2026-05-01
fit_start_time: 2022-01-01
fit_end_time: 2025-05-01
instruments: *market
segments:
train: [2022-01-01, 2024-12-31]
valid: [2025-01-01, 2025-04-30]
test:  [2025-05-01, 2026-05-01]

### G.2 configs/risk.yaml

# Account-level risk rules (generic; tighten per prop firm later)
daily_loss_limit: -3000.00
max_drawdown: -5000.00
max_contracts: 5
consec_loss_lockout: 3
consec_loss_lockout_minutes: 30
trade_window:
start: "08:30"  # CT
end: "11:30"
news_blackout:
enabled: true
high_impact_minutes: 5
med_impact_minutes: 2
fomc_force_flatten_min: 2  # close before FOMC release
dd_alert_thresholds:
- { pct: 25, action: notify }
- { pct: 50, action: notify }
- { pct: 75, action: warn,      reduce_risk_pct_to: 0.5 }
- { pct: 90, action: auto_stop }

### G.3 configs/instruments/nq.yaml

# NQ E-mini Nasdaq-100 futures
symbol: NQ 06-26
exchange: CME_GLOBEX
tick_size: 0.25
point_value: 20.00         # $20 per point
margin_initial: 19800      # CME-set, varies; check broker
margin_maintenance: 18000
session_rth_start: "08:30" # CT
session_rth_end: "15:00"   # CT
session_eth_start: "17:00" # prior day CT
session_eth_end: "16:00"
# MNQ Micro
mnq:
symbol: MNQ 06-26
tick_size: 0.25
point_value: 2.00

## Appendix H — Glossary


| **Term** | **Definition** |
| --- | --- |
| **Alpha158** | **Qlib's prebuilt set of 158 engineered factors covering K-line shape, momentum, volume, volatility** |
| **Alpha360** | **Extended Qlib factor set with 360 factors including longer rolling windows** |
| **ATR** | **Average True Range — volatility indicator measuring typical bar range** |
| **BOS** | **Break of Structure — price breaks the prior swing high (bullish) or low (bearish). Captured implicitly by Alpha158 rank features.** |
| **Bracket order** | **Entry order paired with stop-loss and take-profit as a single OCO group** |
| **CatBoost** | **Yandex's gradient boosting library, one of the 5 ensemble algorithms** |
| **CHoCH** | **Change of Character — first BOS in opposite direction of prior trend; reversal signal** |
| **CT** | **Central Time (Chicago) — VL standardizes all times to CT** |
| **DD** | **Drawdown — peak-to-trough decline in account equity** |
| **Ensemble** | **Combined prediction from multiple models, weighted by recent performance** |
| **EMA** | **Exponential Moving Average** |
| **ETH** | **Electronic Trading Hours — futures session outside RTH** |
| **FOMC** | **Federal Open Market Committee — Fed interest rate decision events** |
| **FVG** | **Fair Value Gap — three-bar imbalance pattern. Captured by Alpha158 K-line features or custom Alpha158_ICT factor.** |
| **GPT-4o-mini** | **OpenAI's cost-efficient LLM, default for RD-Agent discovery loops** |
| **HTF** | **Higher Time Frame — context timeframes above the entry timeframe** |
| **IB** | **Initial Balance — first hour's high/low range** |
| **IC** | **Information Coefficient — correlation between model prediction and actual return; key Qlib metric** |
| **IS** | **In-Sample — backtest training data** |
| **Killzone** | **ICT term for high-probability trading session windows (London, NY AM, NY PM)** |
| **LightGBM** | **Microsoft's fast gradient boosting library, the default Qlib baseline model** |
| **LSTM** | **Long Short-Term Memory neural network for sequential data** |
| **MNQ** | **Micro E-mini Nasdaq futures — 1/10 the size of NQ** |
| **NQ** | **E-mini Nasdaq-100 futures** |
| **NT8** | **NinjaTrader 8 trading platform** |
| **OB** | **Order Block — last opposite-direction candle before a strong impulse; captured by volume-weighted Alpha158 features** |
| **OCO** | **One-Cancels-Other order pairing** |
| **OOS** | **Out-of-Sample — backtest validation data the model never saw during training** |
| **P&L** | **Profit and Loss** |
| **PDC / PDH / PDL** | **Previous Day Close / High / Low** |
| **PF** | **Profit Factor — gross wins / gross losses (>1.0 is profitable)** |
| **POC** | **Point of Control — price level with most volume traded in a volume profile** |
| **Qlib** | **Microsoft's AI-oriented quant platform; ships 23 prebuilt models and Alpha158/Alpha360 factor libraries** |
| **qrun** | **Qlib's CLI command for running a workflow YAML end-to-end** |
| **R / R:R** | **Risk multiple / Risk-to-Reward ratio (e.g., 2.5R = profit is 2.5x the stop distance)** |
| **Rank IC** | **Information Coefficient computed as rank correlation; more robust to outliers than Pearson IC** |
| **RD-Agent** | **Microsoft's LLM-powered research agent; autonomously discovers and tests new factors for Qlib** |
| **RTH** | **Regular Trading Hours — main futures session (08:30-15:00 CT for NQ)** |
| **Sharpe** | **Sharpe ratio — risk-adjusted return; higher is better** |
| **SL** | **Stop Loss** |
| **TP** | **Take Profit** |
| **Trailing stop** | **Stop-loss that moves in the trade's favor as price moves** |
| **Transformer** | **Attention-based neural network architecture; strongest on longer sequences** |
| **VWAP** | **Volume-Weighted Average Price — session anchor for institutional fair value** |
| **Walk-forward** | **Backtest method that repeatedly trains on past, tests on the next unseen period, then rolls forward** |
| **XGBoost** | **Distributed gradient boosting library, one of the 5 ensemble algorithms** |


## Appendix I — Qlib Quickstart (7 timeframes)

Step-by-step from zero to your first Qlib model running on NQ data.

### I.1 Prerequisites

- WSL2 Ubuntu 24.04 set up (Phase 1 complete)
- uv installed: `curl -LsSf https://astral.sh/uv/install.sh | sh`
- Python 3.11
- 1 year of NQ 5m OHLCV bars in Parquet at data/parquet/nq_5m.parquet
- Disk space: ~50 GB for all 7 timeframes of Qlib binary data
- RAM: 16 GB minimum (32 GB recommended for LSTM/Transformer training)

### I.2 Install

# Create Qlib environment
cd ~/vl_trading
uv venv .venvs/qlib_env --python 3.11
source .venvs/qlib_env/bin/activate
# Install Qlib + all model backends
uv pip install "pyqlib>=0.9.7" \
lightgbm xgboost catboost \
torch torchvision torchaudio \
pandas pyarrow numpy scikit-learn \
redis loguru mlflow
# Freeze
uv pip freeze > locks/qlib_env.txt
# Verify install
python -c "import qlib; print(qlib.__version__)"
qrun --help

### I.3 Convert NT8 Parquet to Qlib binary (per timeframe)

# src/qlib/convert_parquet_to_qlib.py — usage
python -m src.qlib.convert_parquet_to_qlib \
--parquet data/parquet/nq_5m.parquet \
--out ~/.qlib/qlib_data/nq_5m \
--instrument NQ \
--start 2022-01-01 \
--end 2026-05-01
# Repeat for all 7 timeframes
for tf in 1m 2m 3m 5m 1h 1d 1w; do
python -m src.qlib.convert_parquet_to_qlib \
--parquet data/parquet/nq_${tf}.parquet \
--out ~/.qlib/qlib_data/nq_${tf} \
--instrument NQ
done
# Verify
ls ~/.qlib/qlib_data/nq_5m/features/NQ/
# Should see: open.day.bin, high.day.bin, low.day.bin, close.day.bin, volume.day.bin, factor.day.bin

### I.4 First qrun (LightGBM + Alpha158 on 5m)

# Use the YAML from Part III §20.2
qrun configs/qlib/lightgbm_5m.yaml
# Expected output (truncated):
# [I 2026-05-20 14:23:01] Workflow lightgbm_5m starting...
# [I 2026-05-20 14:23:15] Loading Alpha158 features...
# [I 2026-05-20 14:24:02] Training LightGBM (500 estimators)...
# [I 2026-05-20 14:24:48] Done. IC: 0.0274 | Rank IC: 0.0312
# [I 2026-05-20 14:24:50] Backtest: Sharpe 1.42 | MaxDD -2.1% | PF 1.61
# [I 2026-05-20 14:24:51] MLflow run saved: mlruns/0/abc123def/
# Inspect the run
mlflow ui --port 5000
# Open http://localhost:5000 in browser

### I.5 Run all 35 models

# scripts/train_all_models.sh
#!/bin/bash
source ~/vl_trading/.venvs/qlib_env/bin/activate
cd ~/vl_trading
for algo in lightgbm xgboost catboost lstm transformer; do
for tf in 1m 2m 3m 5m 1h 1d 1w; do
echo "=== Training ${algo}_${tf} ==="
qrun configs/qlib/${algo}_${tf}.yaml \
--experiment_name vl_initial_$(date +%Y%m%d) \
2>&1 | tee logs/training/${algo}_${tf}.log
done
done
# Total time on a single workstation (no GPU):
#   - LightGBM/XGBoost/CatBoost × 7 tfs: ~10 min each = 3.5 hrs
#   - LSTM/Transformer × 7 tfs: 2-4 hrs each = 28+ hrs (NEEDS GPU)
# Without GPU: skip LSTM/Transformer initially; ensemble of 3 tree algos is fine for v1

### I.6 Common Qlib commands

# Re-run with different model
qrun configs/qlib/xgboost_5m.yaml
# Compare models in MLflow
mlflow ui --port 5000
# Load a trained model and inspect features
python << 'EOF'
import qlib, pickle
qlib.init(provider_uri="~/.qlib/qlib_data/nq_5m", region="us")
with open("models/lightgbm_5m/current/model.pkl", "rb") as f:
model = pickle.load(f)
print(model.feature_importance(importance_type="gain")[:20])
EOF

### I.7 Troubleshooting


| **Error** | **Fix** |
| --- | --- |
| **`provider_uri not found`** | **Run the conversion step first; verify ~/.qlib/qlib_data/nq_*/ exists** |
| **`KeyError: 'NQ' not in instruments`** | **Check instruments/all.txt is populated; pass --instrument NQ to converter** |
| **LSTM/Transformer hangs** | **PyTorch falling back to CPU. Install CUDA or skip LSTM/Transformer until GPU available** |
| **`Insufficient memory` during Alpha360** | **Reduce data range or use Alpha158 instead (158 features vs 360)** |
| **IC < 0.01 across all models** | **Data quality issue. Check for look-ahead bias, NaN handling, or incorrect timeframe alignment in converter** |


## Appendix J — RD-Agent Quickstart (GPT-4o-mini)

LLM-driven factor discovery for your Qlib workspace. Reads existing Qlib data, proposes new factors via LLM, tests them, merges winners back. Run nightly via systemd timer.

### J.1 Install

# Create RD-Agent environment (separate from qlib_env)
cd ~/vl_trading
uv venv .venvs/rdagent_env --python 3.11
source .venvs/rdagent_env/bin/activate
# Install RD-Agent + LLM SDKs + Qlib (it reads same data)
uv pip install "rdagent>=0.8.0" \
"pyqlib>=0.9.7" \
openai anthropic \
pydantic loguru
uv pip freeze > locks/rdagent_env.txt
# Verify
python -c "import rdagent; print(rdagent.__version__)"
rdagent --help

### J.2 Configure (GPT-4o-mini default)

# Save your OpenAI API key
mkdir -p ~/vl_trading/secrets
echo "sk-proj-..." > ~/vl_trading/secrets/openai.key
chmod 600 ~/vl_trading/secrets/openai.key
# Set environment via .env
cat >> ~/vl_trading/.env << 'EOF'
# RD-Agent config
OPENAI_API_KEY_FILE=/home/hoang/vl_trading/secrets/openai.key
RDAGENT_LLM_MODEL=gpt-4o-mini
RDAGENT_BUDGET_USD=5.00
RDAGENT_MAX_ITERATIONS=20
RDAGENT_QLIB_DATA=/home/hoang/.qlib/qlib_data/nq_5m
RDAGENT_LOG_DIR=/home/hoang/vl_trading/logs/rdagent
RDAGENT_OUTPUT_DIR=/home/hoang/vl_trading/rdagent_workspace/discoveries
EOF

### J.3 First discovery run (factor mining)

source ~/vl_trading/.venvs/rdagent_env/bin/activate
export OPENAI_API_KEY=$(cat ~/vl_trading/secrets/openai.key)
cd ~/vl_trading/rdagent_workspace
rdagent fin_factor \
--max_iter 20 \
--llm_model gpt-4o-mini \
--budget_usd 5.00 \
--output_dir ./discoveries/$(date +%Y%m%d)
# Expected output:
# [iter 1/20] Proposing factor: "volume-weighted RSI(14) scaled by ATR(14)"
# [iter 1/20] Implementing as Qlib expression...
# [iter 1/20] Backtesting on validation data...
# [iter 1/20] IC: 0.0341 ✓ ACCEPTED (above 0.02 threshold)
# [iter 2/20] Proposing factor: "..."
# ...
# Run complete. 3 factors accepted, 17 rejected.
# Cost: $1.42 / $5.00 budget
# Output: ./discoveries/20260520/factors_accepted.yaml

### J.4 Other RD-Agent commands


| **Command** | **Purpose** |
| --- | --- |
| **rdagent fin_factor** | **Propose new alpha factors (most useful for VL)** |
| **rdagent fin_model** | **Propose new model architectures and hyperparameters** |
| **rdagent fin_quant** | **Joint factor + model co-optimization (more expensive)** |
| **rdagent fin_factor_report** | **Extract factors from a research PDF and code them automatically** |
| **rdagent ui** | **Open local web UI at http://localhost:19899 to view trace** |


### J.5 Reading the trace

RD-Agent writes a detailed trace to logs/rdagent/<date>/trace.json. The Models page in VL dashboard reads this via the /api/rdagent/trace endpoint.
# Pretty-print latest trace
cat ~/vl_trading/logs/rdagent/$(ls -t ~/vl_trading/logs/rdagent | head -1)/trace.json | jq .
# Sample output:
{
"run_id": "20260520_230012",
"model": "gpt-4o-mini",
"iterations": [
{
"n": 1,
"factor_name": "VOL_WGT_RSI_ATR",
"expression": "Mean(($volume * RSI($close, 14)), 5) / ATR($close, 14)",
"ic_oos": 0.0341,
"rank_ic_oos": 0.0389,
"accepted": true,
"reasoning": "Combines momentum (RSI) with volume confirmation, normalized by volatility..."
},
...
],
"summary": {
"accepted": 3,
"rejected": 17,
"cost_usd": 1.42,
"duration_min": 18
}
}

### J.6 Applying discovered factors

Accepted factors land in rdagent_workspace/discoveries/<date>/factors_accepted.yaml. To use them in the next weekly retrain:
# Step 1: Review what was discovered
ls rdagent_workspace/discoveries/$(date +%Y%m%d)/
# Step 2: Apply via merge script
python -m src.rdagent.apply_discoveries \
--discovery_dir rdagent_workspace/discoveries/$(date +%Y%m%d) \
--target_handler src/qlib/handlers/alpha158_ict.py \
--dry_run  # remove to actually merge
# Step 3: Verify merged handler still imports
python -c "from src.qlib.handlers.alpha158_ict import Alpha158_ICT; print('OK')"
# Step 4: Trigger a quick re-test
qrun configs/qlib/lightgbm_5m.yaml
# Step 5: If improved, commit the new handler
git add src/qlib/handlers/alpha158_ict.py
git commit -m "feat: merge rdagent discoveries from $(date +%Y%m%d)"

### J.7 Cost management

- Default model gpt-4o-mini: ~$1-5 per run (20 iterations on ~/.qlib/qlib_data/nq_5m)
- Hard cap via --budget_usd 5.00; RD-Agent stops when exceeded
- Monthly: ~$150 if running nightly with $5 cap
- Upgrade trigger: if 5 consecutive nightly runs accept zero factors, switch to gpt-4o (4-8× cost, much better reasoning)
- Local Ollama fallback: free but produces lower-quality factor proposals; use only as emergency backup

## Appendix K — Ensemble Strategy Specification

How the 35 model predictions become one trade signal. The ensemble strategy is the brain that sits between Qlib's predictions and NautilusTrader's order submission.

### K.1 Data flow

Bar close (5m, 1h, 1d, etc.)
│
▼
Each of 35 models writes its prediction to Redis:
vl:predictions:lightgbm_1m:<ts>   → 0.42  (long-leaning)
vl:predictions:xgboost_1m:<ts>    → 0.35
vl:predictions:catboost_1m:<ts>   → 0.51
vl:predictions:lstm_1m:<ts>       → 0.28
vl:predictions:transformer_1m:<ts>→ 0.45
... (same pattern for 2m, 3m, 5m, 1h, 1d, 1w)
│
▼
Ensemble strategy reads all 35 predictions
│
▼
Apply per-model weights from vl:ensemble:weights:
weight = max(0, 30-day live Sharpe of that model) / sum(all weights)
│
▼
Apply per-timeframe weights (config: configs/strategy.yaml):
1m: 0.20, 2m: 0.15, 3m: 0.10, 5m: 0.20, 1h: 0.15, 1d: 0.15, 1w: 0.05
│
▼
Weighted sum → ensemble_signal in [-1, +1]
│
▼
Apply hysteresis (configs/strategy.yaml):
Enter long if  signal > +0.30
Exit long  if  signal < +0.15
Enter short if signal < -0.30
Exit short if  signal > -0.15
│
▼
NautilusTrader: submit bracket order

### K.2 Weight computation formula

# Per-model weight (recomputed weekly during retrain)
w_model = max(0, sharpe_30d_live_of_that_model)
# Normalize across models (within each timeframe)
w_model_norm = w_model / sum(all w_models in that timeframe)
# Per-timeframe weight (manually configured, can be auto-tuned)
w_tf = {
"1m":  0.20,  # short-term reactive
"2m":  0.15,
"3m":  0.10,
"5m":  0.20,  # primary trading timeframe
"1h":  0.15,
"1d":  0.15,  # trend filter
"1w":  0.05,  # macro context only
}
# Combined ensemble signal
ensemble_signal = sum(
w_model_norm[model][tf] * w_tf[tf] * prediction[model][tf]
for model in algorithms
for tf in timeframes
)
# Clip to [-1, +1]
ensemble_signal = max(-1, min(1, ensemble_signal))

### K.3 Conflict resolution between timeframes

When timeframes disagree (e.g. 1m says long but 1d says short), the weighted sum naturally resolves it. But for extreme conflicts, additional rules apply:
- If 1d AND 1w both have |signal| > 0.5 in OPPOSITE direction of 1m/5m signal → reduce position size by 50%
- If 1d signal is fresher than 24 hours old AND opposes the entry → veto the entry
- If all 5 algos on the 5m timeframe disagree (sign mismatch) → skip the bar (low confidence)

### K.4 Threshold tuning

The signal_threshold (default 0.30) controls trade frequency. Tuning guide:

| **Threshold** | **Trades/day** | **Win rate** | **Notes** |
| --- | --- | --- | --- |
| **0.15** | **~12-20** | **~45%** | **High frequency, marginal edges. Useful for high-volume strategies** |
| **0.20** | **~6-10** | **~50%** | **Active day-trading** |
| **0.30 (default)** | **~3-5** | **~55%** | **Balanced - the recommended starting point** |
| **0.40** | **~1-2** | **~60%** | **Conservative; only highest-conviction trades** |
| **0.50+** | **<1** | **~65%+** | **Very rare; only A+ setups** |

These are simulated estimates from backtest defaults. Real numbers depend on your data and model performance — verify empirically before going live.

### K.5 Failure modes and circuit breakers

- If >50% of models fail to publish a prediction within 30s of bar close → halt new entries (model serving issue)
- If ensemble signal is identical for 10+ consecutive bars → halt (stuck pipeline)
- If RD-Agent merges a factor that drops ensemble IC below 0.015 on next retrain → auto-revert
- If any individual model's live Sharpe drops below -0.5 → automatic zero weight (effectively disabled until manual review)

### K.6 Logging requirements

Every trade must log to journal/logs the following so it can be audited and replayed:
# Per-trade log entry (JSON)
{
"trade_id": "trd_20260520_143822",
"ts": "2026-05-20T14:38:22-05:00",
"side": "LONG",
"qty": 2,
"entry": 21487.50,
"ensemble_signal": 0.42,
"predictions": {
"lightgbm_1m": 0.38,
"lightgbm_5m": 0.51,
"xgboost_5m":  0.45,
"lstm_1h":     0.29,
"transformer_1d": -0.18,
...
},
"weights_used": "vl:ensemble:weights@20260518",
"threshold": 0.30,
"stop_loss": 21462.50,
"take_profit": 21547.50,
"reason": "ENSEMBLE_LONG"
}
This log is critical for the Trade Journal page (shows which models drove each trade) and for debugging when performance drifts.
*— End of Final Build Plan v5 + Appendices —*
*Build it.*
