**VL · Funded Future Trading**
**Strategy Studio — Make It Real**
The Complete Plan
*Universal (Crypto + CME Futures / NinjaTrader) · Wired · Verified*
*One merged plan — replaces the earlier two drafts (the Analysis plan + the Phased plan).*
Branch: feat/nt8-stage4-chart   ·   Prepared by Claude (CTO / engineering advisor)

# Contents


# 0. In plain words — what this is

**THE ONE-SENTENCE VERSION**
Your Strategy Studio page shows settings that aren't all connected to the bot — you edit boxes, but some of what you see isn't what the bot actually does. This plan fixes that, one section at a time, so the page tells the truth.
**The problem, concretely: **the page is a **disconnected shell**. It displays values, but the real trading behavior lives in backend code the page doesn't reflect. So: it doesn't show the real venue (NinjaTrader), the prompt preview differs from the prompt the bot actually used, and several risk boxes are editable-but-ignored. As you put it: “nothing displayed is real.”
**The fix: **go section by section. For each one, first understand exactly what's fake (read-only), then build the real version, wire it so your edits truly change the bot, and verify in a real browser that an edit changes behavior. One section fully done before the next.
**The order: **by financial risk — fix what affects real money first (risk limits that look set but don't bind), then visible truth, then polish.
**Note on this document: **this single plan replaces the earlier two files (an analysis draft and a phased draft — the same plan written twice, which was confusing). This is now the one source.

# 1. The 6 sections to fix (at a glance)


| **#** | **Section** | **What's fake → the fix (plain)** | **Type** |
| --- | --- | --- | --- |
| **1** | **Risk Control** | **Risk-limit boxes you can type into but the bot ignores → make the bot actually use them (or honestly relabel). HIGHEST PRIORITY — real money.** | **Wire** |
| **2** | **Prompt** | **The preview of the AI's instructions differs from what the bot really sends → make the preview match reality + honor the 4 prompt boxes.** | **Wire** |
| **3** | **Data Source** | **A futures strategy shows crypto wording, not “NinjaTrader” → show the real venue + neutral labels.** | **Label / UI** |
| **4** | **Indicators** | **A futures strategy shows crypto-only toggles (funding rate, crypto rankings) → show what fits futures, hide what doesn't.** | **UI** |
| **5** | **Strategy Type** | **A “Grid” option that physically can't run on futures → honestly gray it out with an explainer.** | **Parked** |
| **6** | **Defaults** | **The default strategy can't be edited (dead-ends you) → add a clear “make a copy to edit” hint.** | **UI** |

**WHY RISK CONTROL IS FIRST**
In a product that trades real money, a risk limit that looks set but doesn't actually bind the bot is the most dangerous kind of fake. So we fix that before anything cosmetic.

# 2. How each section gets fixed — the 5-step cycle

Big bundled builds went sideways before. The discipline that works: each section is its own small project, and every one runs the same five steps. We don't move on until the current section is real, wired, and verified.
- **DIG DEEP (read-only). **Understand that one section completely — what the page shows, what the bot actually does, and exactly where they're disconnected, pinned at the code location. STOP + report if something's changed.
- **BUILD THE REAL VERSION. **Show the truth + the section's own futures treatment (NinjaTrader venue, contracts/margin risk, the real prompt). Additive — crypto stays byte-identical.
- **WIRE IT. **Connect the controls to the engine so editing them changes what the bot does; make any preview match what actually trades.
- **VERIFY REAL. **In a real browser, prove an edit in this section provably changes the bot's behavior / what Recent Decisions shows. Screenshots + a behavior trace. Crypto unchanged.
- **COMMIT + NEXT. **Ship the section, record it in the plan, back up — then start the next section's deep-dive.
**THE NON-NEGOTIABLE RULES (every section)**
**Additive only **— keep every crypto capability byte-identical; add the futures path; delete nothing.
**Universal / neutral labels **— serve both markets; never gate or hide per-instrument (an earlier commit did this and was reverted — don't repeat it).
**Reuse what exists **— the symbol recognizer, tick sizes, the futures prompt, instrument data; no new hardcoded symbols or dollar values.
**Audit-first **— re-confirm every premise at the current code before any edit; STOP if it drifted.
**Real-browser verification **— prove futures-correct AND crypto-byte-identical in an actual browser, both modes.

# 3. The diagnosis — proof the page is a disconnected shell

Established by a full-page read-only pass (9 analyst maps + a live browser pass in both MNQ-futures and BTCUSDT-crypto modes + code re-verification of every headline). Four concrete proofs the page shows one thing while the bot does another:

| **What the page shows** | **What actually happens (hidden in code)** | **Verdict** |
| --- | --- | --- |
| **Data Source: “Static List” (a crypto source type)** | **The strategy trades through NinjaTrader (CME futures); the venue is never shown** | **WRONG / FAKE** |
| **Prompt Preview in Strategy Studio** | **Differs from the prompt in Recent Decisions — the bot uses a different one** | **FAKE** |
| **Risk boxes: Min R/R, Min Position Size, Max Margin** | **Editable, but the gate hardcodes the values — your numbers are ignored** | **EDITABLE-BUT-IGNORED** |
| **Indicators: crypto rankings + funding-rate toggle** | **None of these apply to a single futures instrument** | **WRONG MARKET** |

**The good news: **a futures engine already EXISTS underneath — symbol recognition, a quarterly contract resolver, a dedicated futures prompt, real NinjaTrader instrument data, and the execution bridge. What's missing is making the **page** reflect and drive it. So most of this work is connecting + showing the truth, not building a trading engine from scratch.

# 4. What “real CME futures” means (vs crypto)

Why futures can't just reuse the crypto controls — the things that are genuinely different (and already grounded in the code's sizing tables):

## 4.1 Sizing, leverage, margin

- **Sized in contracts, not dollar fractions. **Profit/loss = stop-distance-in-ticks × tick-value × number-of-contracts. Contracts = Risk$ ÷ (stop-ticks × tick-value), floored to a whole number; micro contracts give finer granularity.
- **No user-set leverage. **Leverage is emergent (notional ÷ margin posted). The crypto “set your leverage” control is meaningless for futures.
- **Margin is a performance bond, not a loan. **“Max Margin Usage” should mean “% of equity committed as bond,” not a leverage proxy.

## 4.2 Instrument, sessions, data

- **One instrument, front-month. **A futures strategy trades a single instrument on its lead contract (quarterly H/M/U/Z for index + treasuries; monthly for energy/metals) — not a basket. The quarterly resolver + pre-roll + expiry-block + Globex session/holiday check already exist.
- **No funding rate. **Futures have none. Open interest exists but is daily, with weekly-lagged COT positioning.
- **The honest gap. **COT / VWAP / volume-profile / DOM / order-flow do NOT exist anywhere in the code — that's the parked Layer-3 buildout, not a relabel.
- **Order types. **The NinjaTrader bridge is market-only — it can't place the resting limit orders a grid needs. So grid can't run on futures (parked); the futures path is market entries + protective stops.

# 5. The phases in detail


## Phase 1 — Risk Control (FIRST — highest financial stakes)

**The fake: **Risk Control shows crypto leverage + “BTC/ETH / Altcoin” position-value tiers; and Min R/R, Min Position Size, Max Margin Usage are editable-but-ignored — the gate hardcodes them. A customer sets risk limits that do not bind the bot.
**Show the truth: **Futures risk is contracts × point value, margin = performance bond (not user leverage), with a notional ceiling. Add a futures risk panel (contract sizing, $-risk, margin-as-bond) instead of crypto leverage/PVR tiers — additive; crypto tiers stay byte-identical.
**Wire it: **the core, money-critical: the gate must READ the config — min_risk_reward_ratio (not the hardcoded 3.0), min_position_size (not 12/60), and max_margin_usage (enforce or honestly relabel). Futures-scoped so crypto is byte-identical.
**Verify real: **set Min R/R on a futures strategy → the gate uses YOUR value (a trade that would breach it is now rejected, or one that respects it passes), proven via the gate path / a decision record / log. Crypto risk unchanged.
*Code anchors: *kernel/engine_position.go:133 / :73-84 · keep crypto PVR tier labels (do NOT rename to Primary/Secondary — regresses crypto clarity)

## Phase 2 — Prompt (Editor + Preview)

**The fake: **the Strategy Studio prompt preview differs from the prompt in Recent Decisions; the 4 Prompt-Editor boxes (Role / Frequency / Entry / Decision) are ignored by the futures builder.
**Show the truth: **the preview must render the SAME prompt the futures engine actually builds + sends (so Studio preview = what Recent Decisions shows).
**Wire it: **make the futures prompt builder HONOR the 4 boxes — mirror the crypto builder's override-or-default guards — and make the preview call the futures path.
**Verify real: **edit a box on a futures strategy → the Studio preview changes → and the next decision's prompt (Recent Decisions / log) reflects it. Crypto prompt unchanged.
*Code anchors: *engine_prompt_futures.go:111-113

## Phase 3 — Data Source (show NinjaTrader as the venue)

**The fake: **a futures strategy's Data Source shows the crypto “Static List” source type; there's no “NinjaTrader” label — the real venue is invisible.
**Show the truth: **surface NinjaTrader / CME as the venue (the backend infers it from the symbol). Decide the form at the deep-dive: a read-only venue badge (safest), or a source-type tile. Plus universal labels: “Coin Source”→“Data Source”, “Custom Coins”→“Symbols”, “Add Coin”→“Add”, neutral placeholders — both markets see the same neutral labels.
**Verify real: **a futures strategy clearly shows it runs on NinjaTrader; the resolved front-month contract + tick value (from instrument_info) are visible; crypto Data Source byte-identical.
*Code anchors: *CoinSourceEditor (not passed an isFutures prop today — the small enabler) · avoid the reverted gate/hide pattern

## Phase 4 — Indicators (futures-appropriate data)

**The fake: **a futures strategy shows the crypto data block (AI500 / OI-Ranking / NetFlow / Price-Ranking + a dead key) and a funding-rate toggle — none apply to futures.
**Show the truth: **keep the universal blocks (EMA/MACD/RSI/ATR/Bollinger, OHLCV, the 14 timeframes, Volume, OI). For futures, the funding-rate concept doesn't exist; the crypto ranking providers have no single-instrument analogue. Surface the truth (futures uses OHLCV + daily OI; no funding) without re-creating the reverted gate/hide pattern — confirm the approach at the deep-dive.
**Verify real: **a futures strategy shows futures-appropriate indicators; the funding toggle is gone; crypto byte-identical.
*Code anchors: *IndicatorEditor · parked Layer-3: real microstructure (VWAP / volume-profile / DOM / COT)

## Phase 5 — Strategy Type (honest about grid)

**The fake: **two modes; the AI-Grid option always renders. But grid needs resting limit orders, and the NT8 bridge is market-only — grid CANNOT execute on futures.
**Show the truth: **on a futures strategy, the AI-Grid option is honestly disabled with an explainer (“Grid needs resting limit orders; this futures path is market-only. Use AI Trading with protective stops.”). Don't fake grid on futures; don't hide it from crypto. Grid execution stays parked unless the bridge gains native limit/bracket orders.
**Verify real: **on a futures strategy the grid tile is clearly disabled + explained; crypto keeps both tiles byte-identical.
*Code anchors: *StrategyStudioPage.tsx:1081-1135

## Phase 6 — Defaults (visible + usable via Duplicate)

**The fake: **the “default” strategy can't be edited (the chip is frozen, inputs hidden) — it dead-ends you.
**Show the truth: **make the escape hatch obvious with a “Duplicate to edit” hint on the disabled default, so it isn't dead-ended. Keep the lock (it protects a shared fallback the engine relies on); fix the discoverability. No backend change.
**Verify real: **a default strategy shows a clear “Duplicate to edit” path; duplicating produces an editable copy that drives the bot.
*Code anchors: *the default lock is CORRECT (shared fallback) — only the discoverability is fixed

# 6. Parked, decided, and standing rules


## 6.1 Parked (later, separate work)

- **Layer-3 microstructure: **COT positioning, VWAP, volume profile, DOM / order-flow — genuinely new backend + data integration (not a relabel).
- **Prop-firm guardrails: **daily loss limit, trailing drawdown, contract cap, news blackout, consistency % — data-driven from the account, never hardcoded.
- **Grid-for-futures: **parked unless the NT8 bridge gains native limit / bracket orders.
- **Energy/metals tick sizes + the ZT tick-value convention **(code computes $15.625; some refs say $7.8125) — resolve before relying on those instruments.

## 6.2 Decided (settled this session)

- **Capstone gating reverted **— the page no longer hides leverage/USDT/funding per-instrument; we surface truth rather than hide.
- **Default edit-lock stays **— it protects a shared fallback; use Duplicate-to-edit (Phase 6 adds the hint).
- **No relabel that regresses crypto **— keep crypto-clear labels (e.g. BTC/ETH / Altcoin tiers) where renaming would obscure meaning.

## 6.3 Definition of done (per section)

- The section shows the truth for futures (real venue / contracts-margin risk / the real prompt), crypto stays byte-identical, an edit provably changes the bot's behavior in a real-browser test, and it's committed + recorded.

## 6.4 Standing note — the injected Google-Drive tools

Five Google-Drive tool definitions are injected into the system frame every turn (a leaked connector artifact). They are never used under any framing. The plan, the reports, all strategy config, the prompts, and the engine wiring live in the local repository and database, read with the repo tools (Read / grep / git) — never via Drive. There is no file to “find” in Drive.
**Recommended start: Phase 1 (Risk Control). ***Begin with its read-only deep-dive — understand exactly which risk boxes are fake before changing anything. This is a design + execution plan; no code changes until the Phase-1 deep-dive confirms the premises at the current code.*
