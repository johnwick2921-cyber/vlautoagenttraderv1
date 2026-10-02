names rewritten to vl on 2026-09-30 (VL rename)
> Wait — I need to change the framing on this one before the table. The dispatch asks "what is ON right now". The measured answer is that **every prop-firm guardrail in this subsystem is off, and every size cap that "REMAINS enforced" is structurally unable to bind at size 1**. The only entry cap that has ever refused a trade in 20 days of logs is the day-plan per-session cap, which nobody's research names. That reframes D9 from "is the 3-trade cap inert?" to "the guardrail layer has no teeth at all, and the boot WARN's reassuring second clause is true-but-empty."

# SUBSYSTEM — GUARDRAILS / SIZING · D9 · read-only, 2026-09-04

**Resolved from:** running binary PID 878451 (`/home/hoang/vl/vl-bin`, rev `70af663d`), boot 8 at 09-04 08:30:11 CT; DB `file:/home/hoang/vl/data/data.db?mode=ro`; log files `/home/hoang/vl/data/vl_2026-08-16.log … vl_2026-09-04.log` (20 files).
**Source tree:** `/home/hoang/vl-conform` @ `fb50903f` (claim commit on dev tip `492d2067`).
**Auth-gated and NOT read this session:** `/api/config/resolved`, `/api/risk/gate-blocks`, `/api/risk/status` — all return `{"error":"Missing Authorization header"}`. Gate-block counters are **in-memory only** (`telemetry/gate_blocks.go:38-45`, a mutex-guarded map — no DB table), so there is no read-only substitute; the log census in M3 is what replaces them.

## Report provenance (`git log -1 -- <path>`)

```
docs/superpowers/reports/2026-09-03-mc-drawdown.md
  77e1cdfce0df36b091514f0eb2798545d9f8e898 Thu Sep 3 00:39:25 2026 -0500
  docs(1E): Monte Carlo drawdown results — n=64, expectancy indistinguishable from zero (CI -31 to +18), ~1810 trades needed
docs/superpowers/reports/2026-09-04-two-day-audit.md
  f3c640c3f9799e6fa80ce124ae87ee915cad63ed Fri Sep 4 07:26:52 2026 -0500
  docs(two-day audit D3): why the blindness went unalerted — a note, not a build
docs/superpowers/reports/2026-08-30-knob-census.md
  741bfc2a8c443feceaa0f31d30c015946b775633 Tue Sep 1 07:58:16 2026 -0500
  docs: archive 38 stranded research reports to dev + RESEARCH INDEX (docs-only …)
docs/superpowers/reports/2026-09-02-belief-census.md
  ee64a494c60eed32bb5e71f4a2b0c43d8b0c5574 Wed Sep 2 08:50:38 2026 -0500
  docs: belief census 2026-09-02 — every market belief labeled [R]/[X]/[T]/[I]/[O] …
docs/superpowers/reports/2026-09-03-expectancy-1d.md
  38a63a9bb2892beb91041bf7e551a8701df8cf9b Thu Sep 3 15:26:02 2026 -0500
  docs(1D): report — the model, RED/GREEN, the live table, and two surprises
```

Label legend per `2026-09-02-belief-census.md:10-19`. The belief census contains **no guardrail or sizing rows** (its sections are A prompt / B validator / C computed / D grader / E exits / F weekly / G doctrine) — so for most rows below the grounding is the **knob census** (values) or the **mc-drawdown / two-day audit** (state), and where neither names the rule I say **"none found"** rather than inventing a label source.

---

## 1. RESOLVED CONFIG — the bound strategy, never `LIMIT 1`

`traders.strategy_id` → `a5b7662e-7bf7-49bb-9f09-7efa48f95ac8` ("MNQ"), trader `hoang`, account `Sim101`, `is_running=1`. **[A]**

**Premise trap worth naming:** `risk_control` is **absent from the root** of `strategies.config`. It lives at `config.ai_config.risk_control`, resolved by `store/strategy.go:846-898` (`UnmarshalJSON` prefers `raw.AIConfig.RiskControl`). A probe reading `json_extract(config,'$.risk_control')` returns empty and would conclude "no risk config is set" — it is set.

```
guardrails_enabled       = false     daily_loss_limit_usd    = 450
daily_loss_enabled       = false     daily_profit_target_usd = 900
daily_profit_enabled     = false     max_daily_trades        = 3
max_daily_trades_enabled = false     max_contracts_per_order = 2
max_contracts_enabled    = false     notional_cap_enabled    = false
blackout_enabled         = false     max_positions           = 3
min_confidence           = 60        min_risk_reward_ratio   = 2
btc_eth_max_leverage = 5   altcoin_max_leverage = 5
btc_eth_max_position_value_ratio = 5   altcoin_max_position_value_ratio = 1
min_position_size = 12     max_margin_usage = 0.9
hold_discipline = true     breakeven_enabled = true (trigger 40)   trailing_enabled = true
ABSENT: consecutive_loss_halt · consistency_enabled/consistency_max_day_pct ·
        blackout_start_ct / blackout_end_ct · max_notional_leverage
day_plan.sessions max_trades = NY 10 / ASIA 7 / LONDON 10
```

Env (`/home/hoang/vl/.env`, variable **names** only — 27 keys): `STAGE_A_CONTRACT_CAP`, `RISK_MAX_DAILY_LOSS_USD`, `RISK_MAX_CONCURRENT_TRADES`, `RISK_MAX_NOTIONAL_USD`, `NT_ALLOWED_ACCOUNTS` are **all unset** → code defaults govern (`config/config.go:175,176,188` = 500 / 2 / 50 000; `kernel/risk_limits.go:305` StageA = 1). `/proc/878451/environ` carries none of them either (it is exec-time only; godotenv `os.Setenv` does not appear there, so this confirms the launch env, not the file). **[A]**

Live equity, latest snapshot `trader_equity_snapshots` id 37779, 2026-09-04 13:50:09Z: **$51,906.50**, `position_count=0`. **[A]**

## 2. BOOT-8 RESOLVED LINES (read from the running process, never a file default — A11)

```
08:30:11 trader/auto_trader.go:43  🧾 ledger boot: … · guardrails=master=OFF (soft-audit only) ·
                                    … trailing=2.0×ATR14 arm=after_breakeven (source: studio) …
08:30:11 vl/main.go:335          🛑 exits: stop=max(anchor+clr, 1.5×ATR5m) · anchor_max=3.0×ATR5m ·
                                    BE=off · trail=off · size=1 · re-arm-after-sweep=on (0B)
08:30:11 kernel/risk_limits.go:172 Plan 3 T21 / Strategy Studio: daily window reset to CME session-day 2026-09-03
08:30:11 → 08:46:10 kernel/engine_analysis.go:173  ⚠️ Strategy Studio: risk guardrails master OFF …
                                    — 9 lines, one per decision cycle at the 2m cadence
```

`guardrails=master=OFF (soft-audit only)` is genuinely resolved from the loaded strategy (`trader/auto_trader_pause.go:185-195` branches on `hlBool(rc.GuardrailsEnabled, true)`), not a literal. **[A]**

---

## 3. THE RULE TABLE

| rule | file:line | RESOLVED value now | label | report:line grounding | live effect | CONFORMS? | production callers |
|---|---|---|---|---|---|---|---|
| **guardrails master switch** (`guardrails_enabled`) | `kernel/engine_analysis.go:152` (read) / `:173` (WARN) / `kernel/risk_limits.go:241` (short-circuit) | **false — OFF**. Boot-8 ledger `guardrails=master=OFF (soft-audit only)`; 9/9 cycles printed the WARN | [O] | `2026-08-30-knob-census.md:86` (`guardrails_enabled false [O]`) | WARN + soft-audit only; bypasses all 5 daily guardrails | **yes** — research `false` = resolved `false` | 1 — `kernel/engine_analysis.go:152` |
| **daily loss limit** | `kernel/risk_limits.go:244` | **$450 configured, `daily_loss_enabled=false`, master OFF → NOT enforced** | [O] | `2026-09-03-mc-drawdown.md:32`; `2026-08-30-knob-census.md:81` | soft line only — 362 `daily loss would trip` lines in 20 days; **0** enforcement fires | **yes** — research "$450, inert, double-disabled" = resolved | 1 — `kernel/engine_analysis.go:183` |
| **daily profit target** | `kernel/risk_limits.go:247` | **$900 configured, `daily_profit_enabled=false`, master OFF → NOT enforced** | [O] | `2026-08-30-knob-census.md:82` | none — 0 soft lines, 0 fires | **yes** — 900 = 900 | 1 — `kernel/engine_analysis.go:183` |
| **max_daily_trades** | `kernel/risk_limits.go:250` | **3 configured, `max_daily_trades_enabled=false`, master OFF → NOT enforced** | [O] | `2026-09-04-two-day-audit.md:619` and `:925` (D38) | soft-audit only — 2 431 `max daily trades would trip` lines; **0** enforcement fires | **yes** — research "3 — NOT ENFORCED" = resolved "3 — NOT ENFORCED" | 1 — `kernel/engine_analysis.go:183` |
| **blackout window** | `kernel/engine_analysis.go:188`; window fn `kernel/cme_calendar.go:129` | **`blackout_enabled=false` AND `blackout_start_ct`/`end_ct` ABSENT → unconfigured; master OFF** | [I] | none found | cannot fire — three independent reasons; 0 lines | **yes** (no research value to contradict) | 1 — `kernel/engine_analysis.go:188` |
| **consistency rule** (`consistency_max_day_pct`) | `kernel/engine_analysis.go:196`; `kernel/risk_limits.go:358` | **`consistency_enabled` ABSENT → nil → default `false`; pct ABSENT → 0** | [I] | none found | none — 0 fires | **yes** (no research value) | 1 — `kernel/engine_analysis.go:196` |
| **consecutive_loss_halt** | `trader/auto_trader_orders.go:116` | **0 (key ABSENT from `risk_control`) → OFF**. NB: this breaker is **master-INDEPENDENT** (`auto_trader_orders.go:109-111` comment; no `GuardrailsEnabled` read) | [I] | none found | would REJECT new entries + P0 alert; currently OFF; 0 fires | **yes** (no research value) | 1 — `trader/auto_trader_orders.go:250` |
| **futures notional ceiling** (equity × N) | `kernel/risk_limits.go:343`; const `kernel/engine_position.go:25` = 20.0 | **`max_notional_leverage` ABSENT → venue default 20.0 → ceiling $51 906.50 × 20 = $1 038 130** | [I] | none found (the const's own comment is the only source) | gate at validate (`engine_position.go:104-107`) + cap at execute (`auto_trader_risk.go:255`); ALWAYS ON, master-independent; **INERT** — 1 MNQ ≈ $58 000 | **yes** (no research value) but see §5 | 2 — `kernel/engine_analysis.go:557`, `trader/auto_trader_risk.go:255` |
| **per-order contract clamp** | `kernel/risk_limits.go:284` | **`max_contracts_per_order=2` → `ResolveMaxContracts(2,2)` → `ClampStageAContracts` → `StageAContractCap()` = 1** | [O] | `2026-09-03-mc-drawdown.md:44-45` ("`max_contracts_per_order = 2` in config while 0B's `ClampStageAContracts` caps at 1 — the clamp governs") | clamp at execute; ALWAYS ON; can never reduce below the StageA 1 that `futuresOrderQuantity` already floors at | **yes** — research 1 = resolved 1 | 1 wrapper → 2 sites: `trader/auto_trader_orders.go:57` → `:532`, `:680` |
| **position size (StageA cap)** | `kernel/risk_limits.go:305-320` | **`STAGE_A_CONTRACT_CAP` unset → 1**; boot line `size=1`; tape: all 5 recent positions `quantity=1.0` | [O] | `2026-09-04-two-day-audit.md:609` ("position size · 1 contract (fixed by 0B) · boot line `size=1`") | hard size ceiling; arm path additionally **hardcodes** qty 1 at `trader/armed_executor.go:965` and `:1603` | **yes** — 1 = 1 | 2 — `trader/exit_mechs_suspend.go:100` (boot line), `kernel/risk_limits.go:289` (via `ResolveMaxContracts`) |
| **concurrent-position cap** (pre-prompt) | `kernel/risk_limits.go:331` + `kernel/engine_analysis.go:133` | **3**, source `strategy max_positions` (per-strategy 3 beats env fallback 2) | [O] | `2026-08-30-knob-census.md:76` | skip-cycle HOLD at ≥3 open; **0** fires; superseded in practice by `one_open_position` (cap 1) | **yes** — 3 = 3 | 1 — `kernel/engine_analysis.go:133` |
| **enforceMaxPositions** (executor) | `trader/auto_trader_risk.go:302` | **3** (same value, `<=0` fallback also 3) | [O] | `2026-08-30-knob-census.md:76` | REJECT at execute; **0** fires | **yes** — 3 = 3 | 2 — `trader/auto_trader_orders.go:483`, `:631` |
| **leverage caps** (btc_eth / altcoin) | `kernel/engine_position.go:82-85` | **5 / 5**. On futures the `isFutures` branch (`:62-70`) overrides only `maxPositionValue`, **not** `maxLeverage` → MNQ decisions are clamped to `altcoinLeverage=5` | **[X]** | `2026-08-30-knob-census.md:77` records them as live risk knobs | **label only on futures** — `decision.Leverage` is never read by the futures sizing path (`futuresOrderQuantity` = notional/(px×point_value), `auto_trader_orders.go:31-46`); NT8 reports `leverage: 1.0` (`tcp_trader.go:933`) | **NO** — research: a live 5× risk cap; resolved: a clamp on a field no futures code path consumes | 1 — `kernel/engine_position.go:82` |
| **isAccountTradeable (SIM-only block)** | `trader/ninjatrader/tcp_trader.go:292` | **ON, untoggleable.** `NT_ALLOWED_ACCOUNTS` unset → allow-list empty → any account the C# AddOn reports `IsSim` passes. Bound account `Sim101` | [O] | none found in reports — grounded in `CLAUDE.md` SIM-only canon (owner rule) | REJECT before the frame reaches the socket; **0** refusals in 20 days (only SIM has ever been bound) | **yes** — canon "SIM-only, do not weaken" holds; fail-safe on unknown account (`:305-307`) | 3 — `tcp_trader.go:333` (`placeEntry`), `:438` (`PlaceLimitEntry`), `:501` |
| **per-session trade cap** (`day_plan.sessions[].max_trades`) — *master-independent, and the only cap that has ever fired* | `trader/auto_trader_session.go:74` | **NY 10 / ASIA 7 / LONDON 10** | [O] | `2026-09-03-mc-drawdown.md:36` (`day_plan.sessions max_trades = 10 / 7 / 10`) | REJECT new entries; **6** refusals on tape (08-19 23:26 → 08-20 00:47 CT, "ASIA trade cap reached (3/3 this session)" — the cap was 3 then) | **yes** — 10/7/10 = 10/7/10 | 1 — `trader/auto_trader_session.go:43` → `trader/auto_trader_orders.go:281` |
| **drawdown auto-close** (profit >5 % & DD ≥40 %) | `trader/auto_trader_risk.go:138` | **hardcoded 5.0 / 40.0**, no knob. NT8 supplies `leverage=1.0` so the percentage is a **raw price %** | **[I]** | none found | would CLOSE a live position (`emergencyClosePosition`, `:143`); **DEAD IN PRACTICE** — max MFE on own tape = 156.75 pt = **0.533 %** of entry (n=251 CLOSED rows with `entry_price>0`; 65 carry MFE>0); **0** fires in 20 days | **NO** — no research supports a 5 % arming threshold on an instrument whose best-ever excursion is 0.53 % | 1 — `trader/auto_trader_risk.go:100` (`monitorTick`) → `checkPositionDrawdown` |
| **`RiskLimits.Classify`** (the "force-flat kill switch" classifier) | `kernel/risk_limits.go:87` | n/a | [M] | A29 | **DEAD — 0 callers anywhere, not even a test.** Worse: `RiskForceFlat`/`RiskBlockEntry` are *produced* by `DailyGuardrails.Check` but the sole call site discards them — `engine_analysis.go:183` is `_, gErr := g.Check()`. A daily-loss trip HOLDS the cycle; **nothing force-flats** | **NO** — the file header (`risk_limits.go:3`) advertises a "force-flat kill switch" that has no consumer | **0 — DEAD** |
| **`max_contracts_enabled` / `notional_cap_enabled`** | `store/strategy.go:1741`, `:1748` | both **false** in DB | [M] | `store/strategy.go:1747` — *"Deprecated (6.4 ruling B): same as MaxContractsEnabled — parse-only"* | none — parsed, persisted, rendered, read by nothing | **NO — dead knobs** | **0 — DEAD** (only `kernel/risk_config_truth_test.go:50`, `trader/caps_always_on_test.go:15-16`) |

---

## 4. THE ENFORCEMENT CENSUS — 20 days of logs, 2026-08-16 → 2026-09-04 **[A]**

| log line | count |
|---|---|
| `🔍 guardrail WOULD have tripped (master OFF, not enforced)` | **2 793** |
| — of which `max daily trades would trip` | 2 431 |
| — of which `daily loss would trip` | 362 |
| `⚠️ Strategy Studio daily guardrail tripped` (**enforcing**) | **0** |
| `⚠️ concurrent-position gate tripped` | **0** |
| `❌ [RISK CONTROL] Already at max positions` | **0** |
| `⚠️ [RISK CONTROL] … exceeds max … clamping` | **0** |
| `⚠️ [RISK CONTROL] Position … exceeds limit` | **0** |
| `🛑 consecutive-loss halt` | **0** |
| `⚠️ Strategy Studio blackout window active` | **0** |
| `⚠️ Strategy Studio consistency rule` | **0** |
| `not tradeable` (isAccountTradeable refusal) | **0** |
| `🚨 Drawdown close position condition triggered` | **0** |
| `🗓️ session gate: … trade cap reached` | **6** |

Per-day soft-audit lines (blank = log predates the feature): 08-19 301 · 08-20 756 · 08-21 76 · 08-24 202 · 08-25 271 · 08-26 253 · 08-27 68 · 08-28 168 · 08-30 5 · 08-31 256 · 09-01 263 · 09-02 174 · 09-03 0 · 09-04 0. These are **per-cycle** lines, not per-event — the same standing condition re-prints every 2 min, so 2 431 is not 2 431 distinct would-be blocks. 09-03/09-04 are 0 because the session-day trade count never reached 3.

**Every enforcing guardrail in this subsystem has fired exactly zero times in the entire retained log history.** The one entry cap that has ever refused a trade is the day-plan per-session cap — which no research report in the key list evaluates.

---

## 5. D9 — ANSWERS AGAINST THE MONTE-CARLO RIG

The dispatch's verdict line is a **paraphrase**; the report says it across four places. Exact text:

- `2026-09-03-mc-drawdown.md:175` — `95% CI [-31.268, +18.020]      t = -0.527`
- `:133` — `| 50 | IID | 866 | 1,477 | 1,677 | 2,065 | 3,030 |` → **maxDD@50 p95 = $1,677**
- `:162` — `| max_daily_trades 3 | 81.8% of days | −65.87 | +24.54 | −24.54 |`
- `:165` — *"The $450 loss limit forfeits nothing"*; `:223` — *"the $450 daily loss limit is close to inert"*
- `:230-231` — *"with expectancy statistically indistinguishable from zero (CI −$31 to +$18) and ~1,810 trades needed to resolve it"*

### Q: What is ON right now (resolved)?

| control | rig's finding | RESOLVED at boot 8 | verdict |
|---|---|---|---|
| guardrails master | `:33` `guardrails_enabled = False` | **OFF** (ledger boot line, 9/9 cycles) | conforms |
| daily loss $450 | `:32` 450; `:223` "close to inert" | **$450, `daily_loss_enabled=false`, master OFF** | conforms — and *doubly* off |
| daily profit $900 | `:33` 900, `daily_profit_enabled=False` | **$900, disabled, master OFF** | conforms |
| max_daily_trades 3 | `:34` 3, `max_daily_trades_enabled=False` | **3, disabled, master OFF** | conforms |
| max_contracts_per_order | `:35` 2, `:44-45` "clamp governs, so size is 1" | **config 2 → resolved 1** | conforms |
| size | `P3` `:19` "confirmed exactly on 4 rows" | **1** (boot `size=1`; tape `quantity=1.0`) | conforms |
| notional ceiling | not evaluated by the rig | **$1 038 130** (equity 51 906.50 × 20) | rig silent |
| session max_trades | `:36` 10/7/10 | **10/7/10** | conforms |

**Nothing in this subsystem is protecting the account today.** The rig said so at `:224-226` (*"Both are currently disabled — master off and each individually off — so neither is protecting anything today"*), and the state is unchanged 30 hours later at boot 8.

### Q: Is the 3-trade cap actually enforced? — **NO. It is inert in practice, and doubly so.**

Two independent locks, either of which alone would disable it:
1. `guardrails_enabled=false` → `DailyGuardrails.Check()` short-circuits at `kernel/risk_limits.go:241-243` and returns `RiskAllow` **before reading any limit**.
2. `max_daily_trades_enabled=false` → even with the master ON, `:250` requires `g.MaxDailyTradesEnabled`.

The evaluation that *does* run is `CheckSoft()` (`:212-232`), which deliberately **ignores every toggle** and evaluates configured values — that is the 2 431 `max daily trades would trip` lines. The cage speaks; it never closes.

**Re-verified at boot 8 (two-day audit `:925`, D38):** *"`max_daily_trades=3` is not enforced because the guardrails master switch is off. It would have blocked every entry from 09-02 09:59:29 CT onward; position **590** (−99.00) opened straight through it."* My independent count on `trader_positions`, CME session-day (17:00 CT roll):

| session-day | entries | window |
|---|---|---|
| 2026-08-30 | 10 | 08-30 17:38:33 → 08-31 13:37:08 CT |
| 2026-08-31 | 6 | 09-01 02:52:44 → 13:33:06 |
| **2026-09-01** | **4** | 09-02 00:17:44 → 10:37:17 |
| 2026-09-02 | 1 | 09-03 09:05:14 |
| **2026-09-03 (live at boot 8)** | **0** | — |

Session-day 2026-09-01, ids **587** 00:17:44 −62.50 · **588** 07:41:05 −65.00 · **589** 09:41:04 −155.00 · **590** 10:37:17 **−99.00 ← the 4th entry**, all `quantity=1.0`, `account=Sim101`. `TradesToday` (from `GetSessionDayActivity`, `store/position_query.go:145-153`, counting rows with `entry_time >= session start` on the active account) was **3** when 590 was decided → `TradesToday >= MaxDailyTrades` was TRUE → `Check()` would have returned `RiskBlockEntry`. **D38 confirmed independently.** At boot 8 the current session-day count is **0**, so the cap could not have bound today in any case.

### Q: Is the $450 limit live? — **NO.** $450 is persisted, `daily_loss_enabled=false`, master OFF. It has produced **362 soft lines and 0 blocks**. The rig's "close to inert" (`:223`) was a *counterfactual* about turning it on (9.1 % of days, $0.00 forfeited, because the one tripping day tripped on its final trade — `:165-167`). Resolved reality is stronger than inert: it is **absent**.

### The part of the boot WARN that overstates itself

> "futures SIZE caps — notional×N ceiling + per-order contract clamp — REMAIN enforced"

Literally true (both are in the call path, master-independent by hardening D3), but **neither can bind**:
- notional ceiling **$1 038 130** vs one MNQ ≈ 29 000 × $2 = **$58 000** → first binds at ~17 contracts;
- the clamp resolves to **1**, which is also the floor `futuresOrderQuantity` applies (`trader/auto_trader_orders.go:36-38` sets `contracts<1 → 1`), so `contracts > maxContracts` at `:42` can never be true;
- the **arm path consults neither** — `PlaceLimitEntry(..., 1, ...)` is hardcoded at `trader/armed_executor.go:965` and `:1603`.

Zero clamp log lines in 20 days corroborates this. The sentence reassures about caps that are geometrically incapable of acting at Stage-A size.

---

## 6. FINDINGS THAT ARE NOT MERE STATE

1. **[A] The force-flat class is produced and thrown away.** `kernel/risk_limits.go:3` calls this file a "force-flat kill switch". `DailyGuardrails.Check` returns `RiskForceFlat` on a daily-loss trip (`:245`), and the only production consumer is `engine_analysis.go:183`: `} else if _, gErr := g.Check(); gErr != nil {` — the decision is discarded, and the handler `return holdCycle("daily_guardrail")` merely **skips the cycle**. `RiskLimits.Classify` (`:87`), the other producer, has **zero callers anywhere including tests**. If the master were switched ON tomorrow, a daily-loss trip would hold new entries and leave open positions running — not flatten them. **A29: `Classify` is DEAD; `RiskForceFlat` has no consumer.**
2. **[A] Two persisted, UI-rendered knobs are read by nothing.** `max_contracts_enabled=false` and `notional_cap_enabled=false` are parsed into `store/strategy.go:1741,1748` and referenced only by `kernel/risk_config_truth_test.go:50` and `trader/caps_always_on_test.go:15-16`. The code even documents `NotionalCapEnabled` as *"Deprecated (6.4 ruling B) — parse-only"*. An owner toggling them sees nothing change.
3. **[A] The drawdown auto-close cannot arm on MNQ.** `positionPnLPct` multiplies by the position's `leverage`, and NT8 hardcodes `"leverage": 1.0` (`trader/ninjatrader/tcp_trader.go:933`). So `currentPnLPct > 5.0` demands a **5 % raw price move in your favour** — ~1 450 MNQ points, ~$2 900 at qty 1. Measured max MFE across n=251 closed rows (65 with MFE>0) is **156.75 pt = 0.533 %**, 9.4× short. 0 fires, 0 near-miss lines (`📊 Drawdown monitoring:` also 0). This is an [I] invented rule with real teeth on paper and none in fact.
4. **[A] `/api/risk/status` would report the wrong numbers.** `api/handler_risk.go:191-198` builds its response from `kernel.LoadRiskLimitsFromConfig()` — i.e. the **env** limits 500 / 2 / 50 000 — and sets `KillSwitchArmed: limits.MaxDailyLossUSD > 0` = **true**. A reader of that endpoint would see a $500 daily-loss kill switch "armed" while the enforced state is master-OFF and `daily_loss_enabled=false`. I could not call it (no Authorization header); this is read from code.
5. **[A] D44 re-verified at boot 8:** the same boot prints `🛑 exits: … BE=off · trail=off` (`vl/main.go:335`) and `🧾 ledger boot: … trailing=2.0×ATR14 arm=after_breakeven (source: studio)` (`trader/auto_trader_pause.go:201`, resolved from `trailingConfig(rc)` which sees `trailing_enabled=true`). Two boot lines, one boot, opposite claims — two-day audit `:930`.
6. **[B] Split-arm capacity is 2, not the boot line's "capacity=1".** `kernel/levels_volume_boot.go:26` prints the **literal** string *"split legs > capacity rejected (capacity=1 unless max_contracts_per_order raises)"*. `armLegCapacity` (`trader/armed_executor.go:678-683`) → `splitLegCapacity(2)` → **2**, so a 2-leg split arm is admissible on this strategy, and each leg places qty 1 → up to **2 contracts** on an account whose boot line says `size=1`. `oneLiveArmGuard` refuses an arm while a *position* is open, but two legs authored while flat are both resting orders, not positions. Not observed on tape (0 split arms in the window), which is why this is [B] and not [A].

---

## 7. COMMANDS USED (all read-only)

```bash
# bound strategy resolved config (never LIMIT 1)
python3 - <<'PY'
import json,sqlite3
c=sqlite3.connect('file:/home/hoang/vl/data/data.db?mode=ro',uri=True)
cfg=json.loads(list(c.execute("select config from strategies where id='a5b7662e-7bf7-49bb-9f09-7efa48f95ac8'"))[0][0])
print(json.dumps(cfg['ai_config']['risk_control'],indent=2))
PY

# entries per CME session-day (17:00 CT roll)
sqlite3 -header -column "file:/home/hoang/vl/data/data.db?mode=ro" "
select date(datetime(entry_time/1000,'unixepoch','-5 hours','-17 hours')) session_day,
       count(*) entries,
       min(datetime(entry_time/1000,'unixepoch','-5 hours')) first_ct,
       max(datetime(entry_time/1000,'unixepoch','-5 hours')) last_ct
from trader_positions
where trader_id='8d5c8af5_8ef641a7-815c-4bb5-9798-b070b67d7998_deepseek_1781246265'
  and entry_time >= strftime('%s','2026-08-29')*1000 group by 1 order by 1;"

# MFE ceiling vs the 5% drawdown-close threshold
sqlite3 -header -column "file:/home/hoang/vl/data/data.db?mode=ro" "
select count(*) n_closed, sum(case when mfe is not null and mfe>0 then 1 else 0 end) n_mfe_pos,
       round(max(mfe),2) max_mfe_pts, round(max(mfe*100.0/entry_price),4) max_mfe_pct
from trader_positions where trader_id='8d5c8af5_…' and status='CLOSED' and entry_price>0;"

# enforcement census
for p in "guardrail WOULD have tripped" "Strategy Studio daily guardrail tripped" \
         "concurrent-position gate tripped" "Already at max positions" "exceeds max" \
         "consecutive-loss halt" "blackout window active" "consistency rule" \
         "not tradeable" "Drawdown close position condition triggered" "trade cap reached"; do
  printf "%-45s %s\n" "$p" "$(grep -h "$p" /home/hoang/vl/data/vl_*.log | wc -l)"; done

# boot-8 resolved lines from the running process
awk '$1=="09-04" && $2>="08:29:00" && $2<="08:31:30"' /home/hoang/vl/data/vl_2026-09-04.log
```

## 8. Files written (worktree only)

- `/home/hoang/vl-conform/docs/superpowers/reports/2026-09-04-research-conformance-data/guardrails-sizing-rules.csv` — 18 rule rows in the dispatch's column shape
- `/home/hoang/vl-conform/docs/superpowers/reports/2026-09-04-research-conformance-data/guardrails-d9-measurements.md` — M1–M6 raw measurements behind every number above

No file in `/home/hoang/vl` was written, edited, or checked out.