# VL DAY-PLAN SYSTEM — FULL PLAN & CONFIG (v1 FINAL, 2 pages)
*The build contract. Consolidates: Professional Plan Card research · Integration Blueprint · Design System spec · Final design-closing research. The combined dispatch is written FROM this document.*

---

# PAGE 1 — THE SYSTEM

**What it is:** every morning a reasoner AI reads the market once, carefully (like a pro with coffee), writes a structured day plan — bias, ranked levels/zones, 2–3 if/then plays, no-trade windows, its own death condition — shown as the Plan Card on the dashboard. The executor AI then follows the plan at every 5m close instead of re-inventing its worldview 78×/day. The owner can watch, edit, question, or ignore it — fully automatic by default.

## Daily timeline
```
8:25 CT  planner reads → plan v1 → card appears (auto)
RTH      executor: "does this bar match a scenario?" — entries only
         per plan + all hard gates; card status updates live
event    plan's death condition hits → auto re-read (max 2/day →
         then no-trade rest of day)
15:45 ET flat (overrides all) · 17:00 plan expires
17:30    evening digest → 3-line "yesterday" note seeds tomorrow
FAIL     planner invalid JSON after ≤2 retries → NO-TRADE day + alert
```
**Cross-session continuity:** levels are RE-DERIVED each read (structure flows: Asia sees RTH-H/L, London sees AS-H/L, NY sees the night; nPOC/PDH persist across days) but LEVEL STATE persists in a level-state table keyed by level identity (times_tested, consumed, freshness) — a level burned in one session cannot return "fresh" in the next. **Owner levels are STICKY: they persist across sessions (with note + scenario tag, 👤) until consumed-by-acceptance or owner-deleted; every subsequent read receives them.** Session close writes a 3-line session digest; trade-date close rolls the day's sessions into ONE 3-line DAILY digest. **Each read receives (owner decision 08-14: ONE WEEK, tapered): all completed session digests of the CURRENT trade date (0–2 × 3 lines) + the last 3 daily digests FULL (3 lines each) + days 4–7 as ONE-LINERS + the owner standing note — ≈20 lines / ~350 tok, tapered so the week's character is known without old prose diluting the level table (LONG memory still lives in the level-state table — the map remembers structure, the digest remembers story).** Cage counters are CME-day global so remaining budget flows automatically. **Multi-session architecture (owner decision · research-settled 08-14):** plans are PER-SESSION — ASIA 17:00–02:00 CT (read 16:55) · LONDON 02:00–08:30 (read 01:55) · NY 08:30–14:45 CT = 09:30–15:45 ET (read 08:25; session end == EOD flat). **Registry:** global-admin, CT-anchored rows (window, read, flat, killzones, enabled — NT8 Trading-Hours pattern); never local time; London shows a DST-drift warning the ~3 weeks US/UK diverge. **Per-session overrides (minimal, inherit-from-strategy with visible ⚪/🔸 chips):** enable · replan_cap · plan_mode · acceptance_rule · min_grade (NEW) · optional max_trades/size-cap — all else inherits. **Risk defaults (evidence: Asia range exceeded 11.8% of days vs NY 37.8%; reopen spreads ~2.4×):** ASIA min_grade=A + max_trades=1, LONDON min_grade=A, NY normal — one shared daily cage stands. **Level naming (kills the Asia-"PDH" ambiguity):** PDH/PDL = calendar-day only · RTH-H/L · AS-H/L · LDN-H/L · ONH/ONL = Asia+London composite — every level session-prefixed. **Display:** SessionTabs + ONE active card, 24h timeline strip, session-shaded chart + killzone boxes, HandoverBanner (expired → reading → born, tab auto-advances); disabled session = night card; read failure = fail-closed NO-TRADE card (the 16:55 closed-market read is a first-class tested path — planner builds from stored data). **Session-flat** at each boundary (limit-then-market, slippage budgeted). **Storage:** additive — plan key becomes (trade_date, session, version) + session on decisions; legacy rows → NY. Sunday 17:00 = Asia of Monday's trade date (prior day = Friday RTH). Ship NY ✓; ASIA/LONDON earn enablement via replay + NY match-rate.

## The card (top → bottom)
1. **Bias** — direction + conviction + explicit flip condition ("flips short on 2×5m < 30148")
2. **Mini chart** — zones as shaded bands, levels as lines, live price, trigger markers (one data array shared with the table — cannot diverge)
3. **Zones & levels** (max 8): price/range · provenance chip (PDH, ONH, nPOC·Tue, D-4h, RN, EQH…) · grade A/B/C · fresh dot · instruction verb · distance
4. **Scenarios** (max 3, graded A+/A/B): formal grammar {trigger, condition ∈ reclaim/hold/sweep_reclaim/reject/acceptance/breakout_retest, direction, target_chain, invalid, quality} + live status ○waiting ◉armed ●triggered ✕invalid
5. **No-trade** — first 5m · 12:00–13:30 lunch · calendar blackouts
6. **Plan dies if** — the thesis-invalidation line + re-reads remaining
7. **Footer** — entries · plan-match% · day-type · gap · version chips

## Levels: sources, grading, day-trade lock
- **Detectors (Go, computed once/session):** PDH/PDL/PDC · ONH/ONL · prior wk/mo · naked POCs · opening range/IB · round numbers · equal highs/lows · S/D zones (base ≤6 candles body ≤0.5×ATR + departure ≥1.5–2×ATR; proximal/distal) · FVG/OB (3-candle imbalance; last opposing candle before displacement)
- **Grading (Go, deterministic — LLM never sorts):** type evidence grade × freshness (fresh > tested > consumed) × confluence count × HTF origin → A/B/C. S/D+FVG enter as C/confluence-only, never standalone triggers.
- **DAY-TRADE LOCK:** proximity filter — only levels within 1.5× daily ATR qualify; today's levels (OR/ONH/ONL/PDH/PDL/PDC) get first seats; owner levels always seated, lowest-grade AI rows drop at the cap.
- **Activation window (intraday drift):** Go hides levels currently >1.5×ATR from price from the executor's candidate set — no re-plan, no plan mutation, cache untouched; levels re-activate when price returns.

## Scenario lifecycle (final research verdicts)
Re-armable same day (pros re-trade levels) BUT: freshness decrements each play (A→B→C; below C = done for the day) · re-arm requires the setup to re-form (not a bare re-touch) · 20-min cooldown between plays · **permanently consumed once price ACCEPTS through the level (2×5m)** — the level flipped roles. An open position is never re-tagged by a re-plan; its (plan_id, version, scenario_id) is frozen at entry.

## Owner interaction (all optional; approval OFF by default)
- **Edit sheet:** tap a row → price/range, type, instruction verb (controlled vocab), grade, note (any language — goes to the AI) → Save = overlay patch (RFC 6902, origin-tagged 👤, `test`-op concurrency). Owner inputs pass the same B2 price armor.
- **Bulk add (confirmed):** one text box, natural multi-line entry → parsed rows → preview → one save. Long-press chart price also works. **Each staged row carries price/type/instruction/grade + NOTE (goes to the AI) + SCENARIO TAG (link S1/S2 or "＋ new play from this level" drafting a scenario via the grammar); one Save = one overlay version; card shows 👤+📝+play chip per owner level, scenarios list their levels.**
- **💬 Ask Planner:** plan-scoped Q&A. Anti-sycophancy contract: answer = EVIDENCE (restate provenance/score/facts) → OWNER'S POINT (new info vs bare disagreement) → VERDICT: DEFEND / CONCEDE-on-new-info / PROPOSE-MERGE patch. Bare disagreement never generates a patch; nothing changes without owner Apply. Every verdict logged → sycophancy rate + who-was-right stats.
- **🔄 Re-alignment on owner edit (owner decision 2026-08-16 — AUTO trigger):** saving an overlay edit (add/edit/delete a level, or a bulk-add batch) AUTOMATICALLY asks the planner to re-examine the **whole** plan in light of that change and return a **PROPOSAL** — never a silent mutation. Reuses the Ask-Planner path verbatim (same anti-sycophancy system prompt, same verdict vocabulary, same `plan_qa` KPI table, so agreement/sycophancy stats stay ONE comparable series); only the user prompt differs: overlay-resolved `plan_final` + live Go facts + the owner's change (price/type/instruction/grade/**note**/scenario tag) + four questions — (a) does this change bias or the flip line? (b) new scenario or fits an existing one? (c) do scenarios need re-targeting/re-grading/removal? (d) does it conflict with an AI level? — under an explicit *DEFEND the existing structure unless the owner's input is genuinely new information* rule. Output = the standard verdict; **NO-CHANGE** renders as a quiet auto-fading chip, **PROPOSE-MERGE** renders the patch card (target version named, every line a concrete −/+ row) with [Apply]/[Keep as-is]; Apply routes through the SAME `/plan/ask/apply` overlay door, so there is exactly one mutation path. **Guardrails:** ONE call per bulk batch (never per row), 20s debounce, per-plan auto cap `realign_cap` (default 5) → beyond it the card falls back to a manual **Re-align plan** button (which bypasses cap + debounce); fail-closed on API error/invalid JSON (no proposal, plan untouched, P1 alert row, never a partial patch); cost + latency logged per call.
- **⚡ Conflict chip:** overlapping owner/AI elements with opposing instructions → owner wins on execution, AI's kept ghosted, both logged.

## Enforcement + trust
- **plan_mode dial:** advisory (ship first: cite scenario_id, measure match-rate) → direction (side must match a live scenario) → strict (Go-verified trigger required). Promotion by demonstrated competence only. Plan restricts, never compels; hard gates (R:R≥3, conf≥60 [owner-dated 2026-08-18; was 65], guardrails, armor) always outrank it; plan governs ENTRIES ONLY.
- **Trust ladder:** (1) fixture unit tests + golden days, tick-exact · (2) owner blind-mark calibration (~10 historical days, owner marks FIRST) · (3) matched-random reaction stats in replay — each level type must beat random ~5pp and A>B>C or it's demoted/cut · (4) evening digest logs which levels reacted.

---

# PAGE 2 — CONFIG, DATA, BUILD

## Day Plan settings block (Strategy page, per-strategy — FINAL field list)
```
plan_enabled            bool      false      master switch
planner_model           enum      —          from multi-key registry (reasoner)
plan_mode               enum      advisory   advisory|direction|strict
planner_timeframes      multiset  D,4h,1h,15m   structure-summary TFs
proximity_filter_atr    float     1.5        0.5–3.0
max_levels              int       8          3–12
scenario_cap            int       3          1–5
acceptance_rule         enum      2×5m       2×5m | 15m-close
replan_cap              int       2          0–4 (per session)
sessions_enabled        set       [NY]       NY | ASIA | LONDON (each earns enable via evidence)
approval_required       bool      false      OFF = fully automatic
evening_digest          bool      true
realign_cap             int       5          0–9 auto re-aligns per plan (0=unlimited); manual button always available
─ GLOBAL (admin, not per-strategy): **calendar feed = ForexFactory weekly JSON (red→T1 blackout, orange→T2, yellow ignored; currency-filtered per session: USD always, +JPY/CNY Asia, +EUR/GBP London) with an owner-editable static T1 fallback file (FOMC/CPI/NFP dates) — feed outage → cache/static + warning, blackouts never silently vanish; each day's slice STORED with the plan (replay/audit uses the stored copy, never refetched)** · session times
  (open 08:30 CT / flat 14:45 CT = 15:45 ET / plan 8:25 CT) · instrument specs
─ INTERNAL CONSTANTS (not owner-facing): activation-window k=1.5 ·
  freshness floor=C · scenario re-arm cooldown=20m
─ NO indicator toggles. NO 12:30 check. (final research verdicts)
```

## Planner regime block (auto-computed, zero config, in the static block)
`trend_state_daily` (px vs EMA200) · `trend_state_1h` · `atr_regime` (ATR14 pctile: LOW<25/NORMAL/HIGH>75/EXTREME>90) · `realized_vol_pct` (5m RV vs 20d) · `vix_level+regime` (<15/15-20/20-30/>30) · `expected_range_pts` (VIX/√252×px, ATR cross-check) · `overnight_gap` (×ATR). Regime is planner INPUT context — never an output readout on the card (one-line annotation max).

## Planner input package (~1,500–2,500 tok, structured tables — never raw bars/images)
Session meta · regime block · structure summary 1 row/TF · **Go-ranked level table** (the decision-critical block, high-salience position) · overnight/prior-day auction story · multi-day context (≤3 days) · **tiered calendar, SESSION-SLICED — each read receives only events inside its window; T1 (red) events auto-write HARD no-trade blackout windows into that plan; T2 listed as caution** · yesterday digest + owner note · current price/distances. Schema-strict JSON out, reasoning-fields-before-answer-fields, ≤2 retries → fail-closed. **Owner may edit the plan's no-trade windows via the overlay (same flow as levels). Filter/behavior setting edits apply at the NEXT read — never mid-plan; enforcement-mode edits apply next cycle; all hot-read.**

## Storage (SQLite, append-only)
`plans(plan_id, version, strategy_id, trade_date, trigger_reason, lifecycle, model_id, prompt_hash, doc JSON CHECK(json_valid))` PK(plan_id,version) — never UPDATE · `plan_overlays(overlay_id, plan_id, plan_version, overlay_version, patch RFC6902, origin, created_at)` · `executor_decisions` gains (plan_id, plan_version, overlay_version, cited_scenario_id) FK → full replay + AI-vs-owner attribution by join.

## Prompt injection (executor, every cycle)
```
CACHED PREFIX (byte-stable all day ≈ free): system rules + PLAN
BLOCK (~400 tok: bias+flip, levels, scenarios, no-trade, cite-rule)
DYNAMIC TAIL (last, ~60–120 tok): market tables (existing) + PLAN
STATUS: price, active-level distances, per-scenario Go facts
(sweep=T/F, closes-beyond n, acceptance n/2), window, re-plans left
```
Facts = Go, judgment = AI. Re-state critical prices in the tail (no cross-reference reliance). Plan v2/overlay edit = one cache re-persist (pennies).

## Data sharing (single kernel)
Detectors computed ONCE → consumed by planner prompt · executor prompt · Plan Card · chart LevelOverlayPrimitive (one array, all renderers). Plans hot-read per cycle (D2 pattern). Shared plan per (symbol, plan-config); per-strategy override possible.

## Build stages (the combined dispatch)
```
STAGE 0  foundations: day_plan config JSON (additive, defaults off) ·
         plans/overlays schema + decision FK · SCENARIO EVALUATOR in
         Go (the keystone — primitives unit-tested on history first)
STAGE 1  detectors + scorer + regime block → 8:25 planner (schema-
         strict, fail-closed) → static/dynamic injection → ADVISORY
         mode + match-rate counters → card v1 (read-only) + chart
         overlay → BLIND-MARK CALIBRATION session (owner hour)
STAGE 2  overlay editing (sheet + bulk add + conflict chip) + Ask-
         Planner thread + evening digest → matched-random stats run
STAGE 3  direction mode (on match-rate evidence) → strict (on replay
         evidence) — promotion by data, never by mood
GAP-HUNT MUST-V1 (build with the stages above — the nervous system):
  🔔 NOTIFICATIONS (owner decisions 08-14: no Telegram, NO external
     push — IN-APP ONLY): alert center on the dashboard = bell +
     feed + pop-ups. P0 (halt, fill, close+P&L, read-fail/
     fail-closed, plan died→no-trade) renders as a pop-up toast +
     persistent banner until acknowledged · P1 (armed, triggered,
     flip, promotion) = bell feed · P2 = digest-only. Everything
     also lands in the feed history. Quiet hours n/a (nothing
     leaves the app). Dedupe/idempotent event bus.
  📊 STATS HONESTY GATE: matched-random verdicts show "WARMING —
     n/target" until the pre-registered sample (~1,565 touches/type
     for the 5pp bar); Bonferroni α≈0.006 across the 8 types; fixed
     evaluation cadence (weekly), never nightly optional-stopping.
     No green "beats random" on an underpowered sample, ever.
  📌 MODEL-ID PINNING: exact model strings only (never provider
     aliases); resolved model ID logged on every plan; model change =
     re-run goldens + blind-mark + RESET the stats window (no pooling
     across models).
  🚫 FALLBACK RULE: read failure → retries → FAIL-CLOSED no-trade +
     P0 alert. Never yesterday's plan, never an uncalibrated second
     provider. Stale > blind is false; blind sits out.
  🌱 COLD-START HONESTY: per-detector readiness badges (WARMING n/10
     sessions — nPOC etc.), UNCALIBRATED chip until blind-mark done,
     advisory-period banner; first week narrates machinery, not edge.
  ✅ ADHERENCE GRADE (A–F) per trade — did it follow the plan,
     computed from the existing FK links, separate from P&L — + tap a
     completed scenario → its trade review. The learning loop's v1.
  💾 SQLITE: WAL mode + single-writer goroutine for plans/decisions.
V1.1 SHELF (deliberately deferred, do not lose): Telegram bot
  variant of notifications (channel rejected for v1 — owner 08-14) ·
  MAE/MFE viz +
  capture-rate · per-scenario-type expectancy · plan-history browser ·
  actionable notification buttons (approve/veto) · /status + /halt
  Telegram commands · plan-diversity drift monitors · plan-sanity
  semantic checks · golden-plan prompt-regression harness · cost/
  latency telemetry dashboards · weekly report card.
DO-NOT-BUILD v1: 12:30 check · non-price triggers · planner
indicator toggles · chart images to AI · auto-learning memory
```

*Governance: this doc + the master spec govern the build; changes update the doc first. — v1 FINAL, 2026-08-14*

## RECON AMENDMENTS (code-verified at 3624a2a4 · report ca1f38c6 · 9 surprises, 0 blockers)
1. **day_plan config (P0):** StrategyConfig has HAND-ROLLED Marshal+Unmarshal (store/strategy.go:731-823) — the field must be added to BOTH codecs or it silently drops; place at ROOT (grid-switch nukes ai_config); write the round-trip golden FIRST.
2. **naked-POC (P1):** SVP engine is stateless + 1m cache holds ~1 prior session, wiped on restart → build a durable session-profile store + a 17:00-CT session-roll snapshot writer (MANDATORY). No deep backfill exists — the nPOC detector warms FORWARD from install (cold-start WARMING n/10 badge covers this honestly).
3. **prompt caching (P3):** transport is plain-string today — the cached-prefix win requires prompt REORDER (dynamic blocks incl. SVP move to prompt-end; plan block joins the stable prefix) + goldens updated deliberately.
4. **skip-while-open (P2):** does NOT exist at HEAD (only same-side refusal + MaxPositions; timer at auto_trader.go:715 Run) — the any-position→skip-cycle gate is a BUILD item, not an assumption.
5. **session-flat (P3):** route directly through the trader close path → bypasses hold-lock naturally, ZERO exemption code needed (simpler than designed).
6. **half-days (P3):** current session code treats half-days as full closures — the session registry + calendar feed own half-day truth; early-close pulls flat time.
7. **chart (P4):** level lines copy the SVP primitive verbatim; only session-shading is net-new.
8. **/api/plan/* (P4):** mirror the inline trader_id pattern of /api/risk/* (no owner-scoping exists).
9. **planner model (P3):** per-strategy planner binding is MISSING — build a 2nd binding + 2nd client via existing primitives (NewAIClientByProvider); empty → falls back to primary model.
Rollup: 7×S + 5×M + 0×L — campaign shape and stage order UNCHANGED.
