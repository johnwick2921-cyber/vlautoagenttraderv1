# DECISION ANATOMY — census report (2026-08-19)

**Branch:** `docs/decision-anatomy` · read-only code-wise; artifacts = [docs/DECISION-ANATOMY.md](../../DECISION-ANATOMY.md) (the owner doc) + this report.
**Base documented:** deployed `1d67a675` (PR #55 final state).
**Method:** 3 parallel code-census agents (timeframe semantics · scenario grammar/enforcement · gate order), spot-verified by hand (the two findings against my own #55 code were re-confirmed [A] before inclusion), plus DB artifacts for the walkthroughs.
**Headline:** 60-row timeframe census · full scenario grammar with machine-vs-AI attribution · 87-step deployed gate order with 8 corrections to the assumed sequence · **21 unique FAIL rows** — the two most urgent are defects in TODAY's #55 code (watcher market-blindness, swallowable post-exit kick) → hotfix dispatch recommended before the next position.

---

## PART 1 — TIMEFRAME SEMANTICS CENSUS (60 rows)

Complete table with file:line in the census agent's terms; the owner-facing distillation is DECISION-ANATOMY.md §2. Key rows:

| site | TF | closed/forming | source | file:line | change |
|---|---|---|---|---|---|
| Scan ticker (the only decision clock) | 2m wall | — | `traders.scan_interval_minutes` | auto_trader.go:823 | Trader modal |
| Cadence mode | interval (default) / bar_close | — | `traders.cadence_mode` | auto_trader_clock.go:41-54,587,595 | Trader modal |
| bar_close gate | primary TF | CLOSED | derived | clock.go:169-177 | via primary TF |
| no-new-data dedup | primary TF ×1 | FORMING sig + FLAT | derived | clock.go:62-86 | — |
| primaryTimeframe() | **5m** default | — | `klines.primary_timeframe` | clock.go:115-124; store/strategy.go:1485 | Studio |
| Supersession watermark | primary TF ×5 | CLOSED | derived | loop.go:378; clock.go:146-162 | — |
| Dodge window | avg×1.2 → primary close | epoch math | const 1.2, ring 20; `STALE_DODGE` | discard_burn.go:33-39,109-131 | env |
| Dodge TF table | 1m/3m/5m/15m/30m/1h only | — | hardcoded | discard_burn.go:88-104 | code |
| Re-eval fresh bar | primary ×3 | CLOSED | derived | discard_burn.go:216-226 | — |
| Re-eval drift | **ATR(14,5m)** ×0.25 | provider | consts + `STALE_REEVAL_DRIFT_ATR` | discard_burn.go:36-58,228-229 | env |
| B4 stale ladder | 1m else 5m | forming-or-previous OK | snapshot + `STALE_BAR_GRACE_S`=15 | stale_data.go:46-56,96-109 | env |
| B4 clock | `ctx.SnapshotMs` (assembly instant) | — | engine_analysis.go:272-273 | stale_data.go:114-132 | — |
| C2 drift | 1m/5m vs POST-CALL now, tol 60s | snapshot vs wall | const | clock_drift.go:26,59-73 | — |
| Feed-down general / in-position | 1m age >10min / >120s | forming incl. | const / `INTRADE_FEED_ALERT_S` | feedwatch.go:23,29-39,52-85 | env |
| Monitor beat (feed+drawdown+BE+trail) | **60s** wall | — | hardcoded | auto_trader_risk.go:24,46-49 | code |
| acceptance_rule | `2x5m` default / `15m-close` | CLOSED only | day_plan + per-session | store/strategy.go:978-992; planconfig.go:51-56 | Studio |
| acceptance N / bucket | 2×5m or 1×15m, from **1m×2000** SVP cache | closed buckets | consts | scenario_facts.go:47-96,109-143; levelstate.go:40,101-117 | rule |
| sweep/reclaim/reject | **raw 1m**, 3-bar lookback (deliberately not rule-TF) | closed | hardcoded | scenario_facts.go:312-341 | code |
| death/flip rule | `PlanCondition.Rule` → 2×5m / 1×15m | closed, touch-gated | plan-authored | plan_lifecycle.go:186-228 | plan |
| Planner structure TFs | D/4h/1h/15m ×300 default | probe | `day_plan.planner_timeframes` | planner.go:744-788 | Studio |
| Level detectors / scorer ATR / dATR | 1m×2000 · ATR14 on 1m · session-day range | closed | consts | levels_assemble.go:35-94 | code |
| OR / IB | first 5min / 60min after 08:30 CT | closed | registry | levels_intraday.go:124-144 | registry |
| Activation window | 1.5×dATR + Studio proximity 0.5–3.0 | — | const + Studio | levels_score.go:135-143 | Studio |
| Snapshot TFs / counts | selected {5m,15m,1h}+ / primary 20 | newest may be FORMING | Studio klines.* | engine_analysis.go:622-657 | Studio |
| Forming/closed prompt label | only 1m/5m/15m labelable | vs `SetPromptSnapshotMs` | code | engine_prompt.go:575-591,672-679 | code |
| SVP | 1m×2000, session-scoped | closed | consts | svp.go:46-47 | pinned |
| Watch cadence | rides the same scan tick, no own timer | inherits | — | loop.go:321-327 | — |
| Watch hold | 2 **cycles** (not minutes) | — | `WATCH_MIN_HOLD_CYCLES` | watcher.go:29,38-41 | env |
| Trailing | ATR(period,5m), mult 2.0, 60s beat | provider | Studio trailing_* | trailing.go:24-59,164-171 | Studio |
| Post-exit delay | 2000ms | wall | `POST_EXIT_DELAY_MS` | postexit.go:28,38-45 | env |
| Sessions / last-entry / EOD-flat / half-day | CT wall, session-relative offsets (−15min), half-day pull-in | CT | registry + day_plan | clock.go:191-459; strategy.go:941-961 | Studio/registry |
| NT8 CloseTime convention | OpenTime+tf−1; **unknown TF → 60 000** | derived | hardcoded map | bars_market_bridge.go:36-88 | code |
| Forming-bar delivery | AddOn re-emits same-`t`; cache replaces | FORMING is real | — | VLBarsSubscriptionManager.cs:503-513; bar_cache.go:248-286 | code |

(Full 60-row table preserved in the census agent transcript; every row above spot-checks clean against code.)

---

## PART 2 — SCENARIO GRAMMAR SPEC

**Structs:** `kernel/plan_doc.go` — `PlanDoc{reasoning*, bias{direction*, conviction, flip_condition-prose}, levels[]{price*, label, grade A|B|C, instruction}, scenarios[]{id*, trigger-PROSE, condition∈reclaim|hold|sweep_reclaim|reject|acceptance|breakout_retest, direction∈long|short, target_chain[]>0, invalid-PROSE, quality∈A+|A|B}, no_trade[]-prose, death_condition*-prose, death{price,side∈below|above,rule∈2x5m|15m_close|5m_close}?, flip{…,flip_to}?}`. Caps 8 levels/3 scenarios (hard ceilings 12/5). Write-time validation: enums, counts, prices, both-side levels, duplicate-cluster reject, gap-continuation reachability (the ONE place trigger prose is machine-read — number mining), targets inside 1.5×dATR.

**Machine-enforced at runtime:** `death{}`/`flip{}` via `PlanConditionFiredSince` (plan_lifecycle.go:200-228 — touch-gated closes-beyond at the rule TF: 2×5m or 1×15m) → re-plan/NO-TRADE loop (planner.go:213-249) · acceptance counters (scenario_facts.go — consecutive CLOSED buckets aggregated from 1m) · sweep/reclaim/reject primitives (raw 1m, 3-bar) · level consumption + re-arm state machine (levelstate.go) · activation windows (1.5×dATR) · citation classification → adherence grade at close · `plan_mode` gate (direction/strict-citation only).

**AI-judgment only (rendered, never checked):** `scenario.trigger` prose · `scenario.invalid` prose · any confirmation requirement (no field exists) · `bias.flip_condition` and `death_condition` prose twins · `no_trade[]` (adherence penalty post-hoc, not a gate) · level `instruction`/`label` · `quality` (decorative) · `target_chain` at execution (only R:R constrains TP).

**Three real stored scenarios (plan `2026-08-19:NY` v3, the one that fed today's winner), annotated:**
```json
{"id":"S1","condition":"reject","direction":"short","quality":"B",
 "trigger":"Rally into 29648.25 OR-L stalls; 5m rejection confirms",   ← PROSE: AI judges (F1)
 "target_chain":[29514,29441],                                          ← reachability-checked at write, unenforced at exec (F9)
 "invalid":"15m close above 29648.25"}                                  ← PROSE (F2); the MACHINE twin lives in flip{} below
{"id":"S2","condition":"acceptance","direction":"short","quality":"A",
 "trigger":"5m close below 29505.75 PDC after losing 29514 RTH-L",      ← the 15m/5m-close-confirm example; still AI-judged as a trigger,
 "invalid":"15m close back above 29514.00"}                                but the LEVEL's acceptance (2x5m) is machine-counted in parallel
{"id":"S3","condition":"sweep_reclaim","direction":"long","quality":"B",
 "trigger":"Sweep of 29514/29505.75 then 5m close back above 29514",
 "invalid":"2x5m below 29441.00"}
death: {"price":29441,"side":"below","rule":"15m_close"}                ← MACHINE (PlanConditionFiredSince)
flip:  {"price":29648.25,"side":"above","rule":"15m_close","flip_to":"long"}  ← MACHINE
```
Executor render (verbatim from record 30123's prompt): `S1 [B] reject short: Rally into 29648.25 OR-L stalls; 5m rejection confirms → 29514.00,29441.00 · invalid 15m close above 29648.25` + `Cite rule: your decision JSON MUST include "cited_scenario"`.

**Citation flow:** prompt cite-rule (plan_render.go:140) → `Decision.CitedScenario` (engine.go:166) → `ClassifyCitation` → counters + `lastCitation` (planner.go:1163-1201) → optional strict gate (orders.go:286) → `decision_records.cited_scenario_id` (loop.go:434-442) → `SetPlanLink` on the position (decision.go:467-469) → adherence A/C/D at close (clock.go:504-515). Net: a label used for grading; a gate only in `strict`, and then direction-only.

---

## PART 3 — THE TWO WALKTHROUGHS

**(a) ENTERED — record #30123, cycle #20201, 11:18 CT → position #524 (+$273.50).**
Plan `2026-08-19:NY` v3 active (bias short/low; S1/S2/S3 above; FOMC-minutes hard blackout 13:00 CT listed in no_trade). Levels alive: OR-L 29648.25 [A] "fade on rejection", RTH-L 29514 [A], PDL 29441 [A]. Snapshot: multi-TF 1h/4h/1d/15m/3m/5m with forming labels; price rallying into OR-L. Prompt: the plan block quoted in Part 2 + market data + `Min confidence to open: 60`. AI: `open_short 29650.75 SL 29672.5 TP 29514 conf 65 cited=S1`, reasoning: *"Price rallied to 29653.75 just above OR-L 29648.25 and snapped back… lower high near 29642.50, matching S1 rejection short. Higher-timeframe bias remains short below 1h EMA50/200."* Gates in order: Stage A all pass (session NY open, account bound, no dodge that instant) · kernel: cap 0<3 ✓, master OFF (soft audit only), notional 1×MNQ ≈ $59k < 20×equity ✓, R:R = (29650.75−29514)/(29672.5−29650.75) = 136.75/21.75 = **6.3 ≥ 3.0** ✓, **conf 65 ≥ 60** ✓, B4 fresh ✓, B7 off · post-call: NOT superseded (the PRIOR cycle #20200 = record 30122 WAS superseded — `guardrail_skip: stale_bar_discarded`, the pre-#55 burn class, a perfect contrast) · executor: feed ✓ dead-man ✓ freeze ✓ boot ✓ pause none ✓ roll 30d out ✓ loss-halt off ✓ last-entry 11:18<14:30 ✓ session in-window ✓ plan-mode advisory ✓ approval off · size: 1 contract ≤2 ✓ → wire: Sim101 tradeable ✓ → fill 29650.75, bracket rested. `risk_check_passed=1`, execution_log `"✓ MNQ open_short succeeded"`. Aftermath: BE fired +51.2pts at 11:36:24 (PR #52's live proof), TP filled at 29514 exactly, close_reason=sync, adherence chain: cited S1, matched.

**(b) WAIT — record #30283, cycle #20361, 17:49 CT tonight (the NEW binary).**
Flat, ASIA session, same plan chain. Snapshot: price 29632.5 compressing under ONH 29638.25 / VAH 29636.25 / OR-L 29648.25. AI: `wait` — *"No 15m close above 29648.25 to trigger S1 reclaim long; no confirmed bearish reversal at 29648/29638 to trigger S2 reject…"* — the model explicitly names the unmet PROSE triggers (Part 2's F1 in action: the trigger judgment IS the AI). No gate fired; the cycle records success with a wait, ℹ️-class on the card. This is the canonical healthy "why no trade": the plan's conditions simply hadn't printed, and the record says so in its own words.

**Canonical gate order as deployed** — the full 87-row table (tick→wire) is preserved from the census agent and spot-verified; the owner-facing staged version is DECISION-ANATOMY.md §3. **Eight corrections to the naive order** (stopUntil→session→last-entry→roll→half-day→min_conf→R/R→size→B7→dodge/stale→wire): (1) the dodge is FIRST (tickOnce), supersession near-last — two distinct stale mechanisms; (2) R:R and min-conf fire INSIDE the AI call before every trader-side gate, and **R:R precedes min-conf**; (3) executor order is …stop_until → **roll** → loss-halt → **last-entry** → **session**…; (4) four integrity gates (feed, dead-man, freeze, boot) outrank stop_until; (5) TWO stopUntils exist — the legacy dormant field (loop.go:261, no producer) and the live `pauseUntilMs` (orders.go:210); (6) half-day is a modifier inside last-entry + EOD-flat, and EOD-flat runs very early; (7) sizing runs twice (kernel literals during parse; executor clamps last); (8) B7 is a kernel post-call rewrite, before all executor gates. Deployed-reality notes: C2 drift is log-only (its call-site comment still claims it refuses entries — stale doc); CME + roll each checked twice (loop + kernel); hold-lock lives in the loop, not the executor; the feed gate is the only entry gate that also blocks closes; env `RISK_MAX_CONTRACTS_PER_ORDER` loads and enforces nothing.

---

## PART 4 — consolidated FAIL register (21 unique; found-not-fixed, sized)

**Urgent — defects in TODAY's #55 code (hand-verified [A]):**

| # | FAIL | Size |
|---|---|---|
| U1 | **Watcher is market-blind**: `runWatchCycle` consumes the loop-built ctx; `MarketDataMap` + `SnapshotMs` are only populated inside `GetFullDecisionWithStrategy` → observer prompt has no bars/labels, price=0, PnL/MFE/MAE=0, no Snapshot clock. Watch-only authority bounds risk to junk assessments. Fix: fetch market data (+stamp SnapshotMs) in the watch path before building the prompt | **S/M — hotfix before the next position** |
| U2 | **Post-exit kick swallowable**: the kick enters `tickOnce` upstream of the cadence gates — `bar_close` mode drops it unconditionally; `interval` mode's `skipNoNewData` can eat it (flat + unchanged sig is exactly the post-close state in quiet tape); the dodge may also re-defer it. Fix: bypass cadence gates for `post_exit` kicks (mirror `skipDodgeOnce`) | **S — same hotfix** |
| U3 | **Watch calls pollute the dodge ring**: observer latencies feed `avgAICallMs`, mis-sizing the dodge window for decision cycles after long holds. Fix: separate ring or tag | S |

**Grammar (agent F-rows):** F1 `scenario.trigger` prose-only · F2 `scenario.invalid` prose-only (the invalidated dot derives from a DIFFERENT heuristic — anchor+danger-direction acceptance — and can contradict the written text) · F3 no confirmation field exists anywhere · F4 **`5m_close` phantom rule** (authored as 1 close, evaluated as 2×5m; the death reason string even prints the wrong rule next to the count) [M] · F5 death/flip objects optional while prose is required, and NO validator compares object↔prose despite the prompt asserting they match [S] · F6 `strict` plan-mode = citation-direction only, weaker than VL-DAYPLAN-FULL-SPEC.md:49 claims [S doc or M code] · F7 scenario state (armed/triggered/invalidated) is card-only, gates nothing · F8 anchor mining: no 4+-digit number within ±2pts of a level → scenario permanently unevaluated; sub-1000-priced instruments unevaluable by construction [S] · F9 target_chain unenforced at execution · F10 quality A+/A/B decorative · F11 id format unenforced · F12 **cite-rule absent from the futures output contract** → a contract-literal model omits it, every grade → D [S] · F13 dead `activePlanIsDead` path [S] · F14 overlay patches can rewrite scenarios freely, failed re-validate silently falls back to base [S].

**Timeframe (agent FAIL-rows, dedup'd):** T3 three disagreeing TF→ms tables — a legal `primary_timeframe=3m/30m` loses the forming label + B4 coverage; an unmapped TF makes the bridge fabricate `CloseTime=open+60s−1`, corrupting the bar-close gate and supersession watermark [M] · T5 **C2 drift measures snapshot bars against the post-call clock** — a legal 300s call logs as "drift" (the exact defect class B4 was cured of; log-only today) [S] · T6 feed-aliveness asymmetry: entry gate judges the snapshot's 5m (tolerates ~10min; with only {15m,1h} selected it never assesses) vs the 120s in-position 1m alert [M] · T7 bridge comment "NT8 bars are closed bars" contradicts the AddOn's forming-bar re-emits (doc) [S] · T8 level/scenario state writers claim bar-close cadence but run every interval tick; re-arm cooldown sampling uncalibrated for it [S] · plus the pre-existing dormant-stopUntil twin and the stale C2 call-site comment (doc rows).

---

## Deliverables

- **Owner doc:** [docs/DECISION-ANATOMY.md](../../DECISION-ANATOMY.md) — birth of a trade · the three timeframe meanings · staged gate map with live values · in-position lifecycle · the "why didn't it trade" table · knob table · caveats C1–C7.
- **Recommended immediately:** a `hotfix/watcher-eyes` dispatch for U1+U2+U3 (one small commit each), BEFORE the next entry; then Master Audit V2 as planned, with this report's FAIL register as seeded findings.
- **PR:** number parsed from the `gh pr create` output URL — stated in the chat delivery.
