# DECISION ANATOMY — how this bot decides, in plain language

*The canonical owner's map of the machine as deployed 2026-08-19 (`1d67a675`, PR #55). For any cycle you can answer: why did it enter, why didn't it, what is it doing while holding — and where the knob is. Line references are clickable in the repo.*

---

## 1 · How a trade is born

Every **2 minutes** (your Studio scan interval) the bot wakes up and runs one cycle:

1. **Pre-checks** — is the CME open? is an NT8 account bound? is a bar close seconds away (then it *waits for the close* instead of wasting the AI call — the "dodge")?
2. **Snapshot** — one instant of truth: account equity, open positions, and bars on your selected timeframes (5m/15m/1h + 1m/3m), with the newest bar honestly labeled **FORMING**.
3. **The prompt** — the snapshot plus the **day plan**: bias, graded levels, scenarios S1/S2/S3, no-trade windows, and the death/flip rules. The AI is told: *cite the scenario you're trading, or say "off-plan"*.
4. **The AI decides** — `open_short @ 29650.75, SL 29672.5, TP 29514, confidence 65, cited S1` — or `wait` with its reasoning.
5. **The gate gauntlet** (§3) — ~20 independent checks, each of which can refuse the entry with a named reason in the log and the dashboard.
6. **The wire** — a market order to NT8/Sim101; the C# AddOn places the stop+target as a real OCO bracket at the exchange the moment the entry fills.

**A real example (2026-08-19, the trade that made +$273.50):** cycle #20201 at 11:18 CT read plan `NY v3` ("Bias: short · S1 [B] reject short: rally into 29648.25 OR-L stalls; 5m rejection confirms → 29514, 29441"). Price rallied to 29653.75, snapped back, the AI cited **S1**, entered short 29650.75 (confidence 65 ≥ your 60 gate), stop 29672.5, target 29514. At +51 pts the breakeven rail moved the stop to entry; the target filled at **29514 exactly**. Record `decision_records` #30123 → position #524.

**A real "why didn't it trade":** cycle #20361 at 17:49 CT — `wait`: *"No 15m close above 29648.25 to trigger S1 reclaim long; no confirmed bearish reversal at 29648/29638…"*. The plan's triggers simply hadn't fired, and the AI said so.

---

## 2 · The three places timeframes matter (don't mix them up)

| Concept | What it means | Timeframe | Where to change |
|---|---|---|---|
| **Cadence** — *how often it looks* | one full cycle per scan interval, forming bar included | **2 min** wall clock (`scan_interval_minutes`), mode `interval` | Trader modal |
| **Confirmation** — *when a scenario "fires"* | "5m rejection confirms", "15m close above" — written by the planner into the scenario text, **judged by the decision AI reading the chart** (see caveat C3) | whatever the scenario says (5m / 15m) | the plan itself (Ask-Planner / edit sheet) |
| **Acceptance** — *when a level is machine-counted as broken* | consecutive **closed** bars beyond a level: `2x5m` (default) or `15m-close`. Drives level CONSUMED state, plan **death**, and **flip** — pure Go, no AI | 5m ×2 or 15m ×1, built from 1m bars | Studio → Day Plan → Acceptance (+ per-session override) |

Supporting cast: the **dodge** watches the 5m close boundary (defers a cycle that would straddle it); **supersession** discards a decision computed on a bar the market already closed past (unless the entry re-validates on the fresh bar); breakeven/trailing ride a **60-second** monitor, not the 2-min cycle; trailing and re-eval use **ATR(14, 5m)**.

---

## 3 · Every gate, in firing order, with today's live value

*Order verified from code top-to-bottom. "Entry-only" = closes, stops and EOD-flat are never blocked by it.*

**Stage A — before the AI is even called** (whole-cycle skips)

| Gate | Live value today | Change it |
|---|---|---|
| CME session gate | open Sun 17:00 → Fri, daily break 16:00–17:00 CT | fixed |
| NT8 account bound | `Sim101` ✓ | Dashboard account picker |
| Bar-close cadence gate | inert (mode=interval) | Trader modal `cadence_mode` |
| No-new-data dedup | active, **flat-only** (never skips while holding) | fixed |
| **Stale dodge** | ON — defers a cycle starting within avg-call×1.2 of the 5m close to close+1s | `STALE_DODGE` env |
| EOD-flat | per session: NY flattens 14:30 CT (half-days pull it in — Labor Day 12:00) | Studio Day Plan `eod_flat` |
| No balance frame yet / trader stopped / no candidates | — | — |
| **In-position split** | `ai_watch` (default): a WATCH cycle runs instead (§4); `bracket_only`: silent skip | Trader modal "In-position AI mode" |

**Stage B — inside the AI call (kernel):** concurrent cap (max_positions **3**, cycle HOLDs at cap) → guardrails master (**OFF today** → daily-loss $450 / profit $900 / max-trades 3 / blackout / consistency all log "WOULD have tripped" and never block) → then each proposed entry must survive: **notional ≤ equity×20** (always on) · min size (12/60 USDT literals) · SL/TP sane · **R:R ≥ 3.0** · **confidence ≥ 60** (your stored value; unset strategies default 60 everywhere since #55) → then price-sanity, **B4 stale-feed** (entry→wait if the snapshot's freshest 1m/5m bar is a period behind), **B7 re-entry cooldown** (**OFF** — `reentry_cooldown_minutes=0`).

**Stage C — after the AI call:** supersession triage — a decision whose 5m bar closed mid-call is: discarded (contains closes) · quietly dropped (wait-only, free) · **re-validated for entries** (executes iff the fresh bar never touched its stop AND drift < 0.25×ATR14). Then safe-mode (3 AI failures → entries blocked) and **hold-lock** (ON — suppresses AI closes while holding; latent under ai_watch since the observer can't emit actions anyway).

**Stage D — the executor gauntlet (entry-only, in this exact order):** feed gate (also blocks closes) → dead-man watchdog → freeze → boot-integrity revision → **stop_until pause** (your ⏸ button, none armed) → **contract-roll** (blocks entries within 3 days of expiry; MNQ SEP26 expires 2026-09-18 → blocked Sep 15–18) → consecutive-loss halt (**OFF**) → **last-entry cutoff** (NY 14:30 CT) → **session gate** (windows, first-5m, lunch 12:00–13:30, red-news blackouts) → plan-mode (**advisory** — never blocks) → approval (**OFF**).

**Stage E — sizing + wire:** notional clamp (equity×20) → **max 2 contracts** (always on — the Studio toggle was fake and is gone) → SIM-only account check (hard) → duplicate-order guard → the order.

**Reading a refusal:** every gate writes one named line — `⛔ feed-gate`, `⏸ stop_until`, `📅 contract-roll`, `🛑 consecutive-loss halt`, `🕒 last-entry cutoff`, `🗓️ session gate`, `📐 plan-mode`, `⛔ stale_reeval outcome=refused`, `confidence too low (58), must be ≥60`… The dashboard decision card shows the same reason (ℹ️ grey = a rail did its job; ❌ red = something actually failed).

---

## 4 · In-position lifecycle (new since #55)

1. **Fill** → the C# AddOn rests your stop+target as a real OCO bracket at the exchange. The entry decision (direction, levels, reasoning) is saved as the **thesis**.
2. **Watch cycles** (mode `ai_watch`, default): every 2-min cycle while holding, the AI runs as a pure **observer** — it re-reads the market against the ORIGINAL thesis only and answers one question: *has the stated invalidation triggered?* → `intact / weakening / invalidated` + a note + confidence. **It cannot act. Nothing it says is ever sent to NT8** (enforced structurally, not just by prompt). *Caveat C1 below — fix pending before the first live watch.*
3. **Hysteresis** (why one nervous reading changes nothing): "invalidated" only counts if it cites the stated condition with confidence ≥ **70**; the status can worsen at most one step per cycle and never twice inside **2** cycles; recovery is instant.
4. **The WARN**: after **2 consecutive** accepted invalidated readings you get ONE dashboard warning — `⚠ THESIS INVALIDATED (2×): <the condition>`. It feeds **nothing else**: no gate, no exit, no future prompt. You decide.
5. **Breakeven** (ON, +50 pts): the 60-second monitor moves the stop to entry, once, via the proven `move_stop` amendment (fired 6× live, incl. today's winner).
6. **Trailing** (**OFF** — yours to arm in Studio → Risk Control): once armed (after breakeven, by default), each minute the stop ratchets to best-price ∓ 2.0×ATR(14,5m). It only ever tightens; it never goes back below entry once breakeven fired; the wire refuses any widening as a second lock.
7. **Exit** — target, stop, EOD-flat or manual. Real fill price + P&L recorded; a stop-loss exit arms B7's cooldown (if you enable it).
8. **Post-exit rescan** (ON): ~2 s after the close confirms, ONE fresh cycle fires — through every gate, with a normal prompt (no "revenge" context). *Caveat C2.*
9. **Scoring table**: every watch assessment is stored with what happened AFTER it (`watch_assessments`) — the evidence base for ever deciding whether the watcher earns authority. No decision on that yet, by design.

---

## 5 · "Why didn't it trade?" — the debugging guide

Look at the decision card / `data/nofx_YYYY-MM-DD.log` (journald drops INFO lines under load — the file log is complete). Find your line:

| You see | It means | The knob |
|---|---|---|
| `🌙 CME closed` | outside trading hours | none (calendar) |
| `⏳ stale_dodge deferred_ms=N` | cycle re-timed to just after the 5m close — a call started now would have died at the close | `STALE_DODGE=off` |
| `⏭ cycle_skip=no_new_data` | flat + literally nothing changed since last look | none (saves money) |
| `wait` with reasoning | the AI judged no scenario fired — read its note, then the plan | the plan (Ask-Planner) |
| `confidence too low (N), must be ≥60` | setup below your bar | Studio Min Confidence |
| `risk/reward ratio too low` | SL/TP geometry worse than 3:1 | Studio Min R:R |
| `⛔ stale_bar_discarded … verdict_hint=feed\|clock\|math` | feed was stale at snapshot time — the hint says which subsystem | fix feed; `STALE_BAR_GRACE_S` |
| `⏰ decision bar … DISCARDING` / `ℹ️ superseded_wait` / `⛔ stale_reeval outcome=refused` | the 5m bar closed mid-call; entry failed re-validation (stop touched / drift ≥ 0.25×ATR) | `STALE_REEVAL_DRIFT_ATR` |
| `🕒 last-entry cutoff` / `🗓️ session gate` | too late in the session / no-trade window | Studio Day Plan |
| `📅 contract-roll` | ≤3 days to expiry | `ROLL_BLOCK_DAYS_BEFORE_EXPIRY` |
| `⏸ stop_until` | your pause button | Dashboard ▶ Resume |
| `⛔ re-entry cooldown` | B7 after a stop-out (only if you armed it) | Studio Re-entry cooldown |
| `🛡️ Safe mode` | 3 AI failures in a row | recovers on next success |
| `🚨 A4 FROZEN` | an untracked-fill / reconcile divergence froze entries | `/api/risk/clear-freeze` after checking NT8 |
| `🔐 BOOT INTEGRITY REFUSAL` | running binary ≠ `deploy/RELEASE` | redeploy properly |
| `guardrail WOULD have tripped` | master OFF: the cage is narrating, not acting | Studio guardrails master |
| `🧘 skip-while-open` | bracket_only mode heartbeat | Trader modal position mode |
| `👁 watch: status=…` | ai_watch observer reading (no action possible) | Trader modal position mode |

---

## 6 · The knob table (who owns what)

- **Studio → Risk Control** (`ai_config.risk_control` — the nested path; top-level `risk_control` is a decoy): min_confidence **60** · min R:R **3.0** · max_positions 3 · guardrails master **OFF** + daily loss 450/profit 900/trades 3 (all individually off) · hold_discipline **ON** · breakeven **ON @ 50** · **trailing OFF** (mult 2.0 / ATR14 / arm after_breakeven) · consecutive-loss **OFF** · re-entry cooldown **OFF** · blackout OFF.
- **Studio → Day Plan**: plan on, mode advisory, acceptance `2x5m`, last-entry/EOD-flat offsets, per-session overrides, proximity 0.5–3.0 dATR, re-plan/re-align caps.
- **Trader modal** (per trader, DB): scan interval **2m** · cadence **interval** · position mode **ai_watch** · account **Sim101**.
- **Env** (`.env`, documented in `.env.example`): `STALE_DODGE` on · `STALE_REEVAL_DRIFT_ATR` 0.25 · `WATCH_INVALIDATE_MIN_CONF` 70 / `WATCH_MIN_HOLD_CYCLES` 2 / `WATCH_WARN_CONSECUTIVE` 2 · `POST_EXIT_RESCAN` on / delay 2000ms · `ROLL_BLOCK_DAYS_BEFORE_EXPIRY` 3 · `STALE_BAR_GRACE_S` 15 · `INTRADE_FEED_ALERT_S` 120 · `RISK_MAX_DAILY_LOSS_USD` 500 (only bites if master ON and Studio value empty) · `NT_ALLOWED_ACCOUNTS` (overrides the picker).
- **Plan-authored** (the planner writes, you edit via the door): bias/flip, levels+grades+instructions, scenarios + their trigger/invalid text, no-trade windows, death/flip objects.
- **Fixed in code** (deliberate): SIM-only · 2-contract clamp · equity×20 notional · 12/60 USDT minimums · 60s monitor · SVP 1m×2000 · dodge factor 1.2.

---

## 7 · Known caveats (honest edges, as of tonight)

- **C1 — the watcher is currently market-blind (top priority).** As shipped in #55, watch cycles reuse a context whose market block is only filled during *decision* calls — so the observer prompt has **no bars and price reads 0**. Zero trading risk (it cannot act), but its assessments would be junk. **Fix before the next position** — a one-commit hotfix dispatch is queued in the anatomy report.
- **C2 — the post-exit rescan can be swallowed** in quiet tape (the no-new-data dedup fires on the flat-and-unchanged state a close leaves behind) and is dropped outright in `bar_close` mode; the dodge can also re-defer it. Same hotfix dispatch.
- **C3 — scenario confirmations are STRUCTURED and machine-computed since the fail-register wave**: every scenario authors `confirm{rule: touch|1x5m_close|2x5m_close|15m_close, ref_price, side}`; the executor prompt shows a machine-computed "MET / NOT MET" line and the card a chip. The prose trigger/invalid remain AI-judged, and dots/chips stay **advisory by design (D1 ruling)** — a hard scenario-state gate would recreate the suppression class; AI judgment + plan discipline + the risk gates decide. `target_chain` stays guidance (the executor AI sets TP — D2); `quality` stays informational (D3). `strict` plan-mode now also band-grades the entry against the cited scenario (off-band or SL/TP-inconsistent = B, not A).
- **C4 — FIXED (fail-register wave):** `5m_close` now evaluates as authored — exactly one 5m close — and the death log names the rule that actually fired.
- **C5 — Studio Save button**: toggling a control does nothing until you press **Save** — the Aug-19 "toggles left no trace" mystery was exactly this.
- **C6 — journald still drops INFO lines** until you run the one root step: `sudo bash deploy/install-journald.sh`. WARN+ already reaches the dashboard regardless; the complete file log is `data/nofx_YYYY-MM-DD.log`.
- **C7 — two clocks disagree politely**: the C2 drift line can read a slow AI call as "drift" (log-only), and the entry-side feed staleness check tolerates ~10× more silence than the in-position 120s alert. Both documented in the report's FAIL table.
