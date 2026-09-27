# BELIEF CENSUS — 2026-09-02

Every market belief encoded in the system: where it lives, what it claims,
where it came from, and what teeth it has. READ-ONLY dispatch — no engine code
touched, no lock. Companion to the 2026-08-30 knob census (numbers); this one
labels RULES.

## Label legend

| Label | Meaning |
|---|---|
| [R] | researched-supported (cite source) |
| [X] | researched-CONTRADICTED (cite) |
| [T] | measured on own tape (cite report, n) |
| [I] | invented / doctrine, untested |
| [O] | owner-ruled |
| `MUST` / `gate` / `weight` / `advisory` / `shadow` | live effect |

---

## A. Prompt beliefs (what the model is told)

| # | Belief | Where | Label | Effect |
|---|--------|-------|-------|--------|
| A1 | Bias tree: close>PDH→bull HIGH; PDH sweep+close back→bear MEDIUM; inside-day→direction of close vs PDC, LOW | kernel/planner_prompt.go:145-147 | [I] | advisory (reasoning MUST state branch) |
| A2 | "NY AM 08:30–11:00 ET primary; 10:00–11:00 premium FVG window" | planner_prompt.go:536 | [I] | advisory |
| A3 | "Conviction: down on Monday, up Thursday/Friday" | planner_prompt.go:537 | [I] — no own-tape test | advisory |
| A4 | STOP-DOING: "acceptance entries WITHOUT a prior sweep + displacement are 0% win evidence — skip them" | planner_prompt.go:541-542 | [T] week evidence (massive-move audit; validator reject class in entry_law) | advisory + REJECT at write (see B4) |
| A5 | T1 red-news ±15m = HARD no-trade; T2 caution | kernel/calendar_blackout.go:13-41 | [O] | gate |
| A6 | Lunch 12:00–13:30 CT no-trade · first-5m no-trade | auto_trader_session.go:118-123 | [O] | gate |
| A7 | HTF zone rows MUST be included as confluence, "never a standalone trigger" | planner_prompt.go:553-558 | [O] (1h wave) | advisory (prompt contract) |
| A8 | Scenario mix must follow regime+day_type; below PDL write continuation short / above PDH long | planner_prompt.go:594-596 | [I] | advisory |
| A9 | entry_mode=ce default; edge only for A-grade HTF-confluent origins | planner_prompt.go:522 | [R/O] fvg-entry-model wave | advisory |
| A10 | "resting limits fill at the authorized price; stale_reeval NOT applied" | armed_executor boot line | [O] contract | gate semantics |
| A11 | "never trade on a clock known broken" — defer authoring on provably-broken clock | kernel/clock_drift.go F6 | [R] 08-30 incident | gate |

## B. Validator beliefs (REJECT-at-write law)

| # | Belief | Where | Label | Effect |
|---|--------|-------|-------|--------|
| B1 | Breakdown/continuation needs BD_MIN_DISP 1.0×ATR5m displacement before immediate authoring | kernel/breakdown_continue.go | [T] waterfall replay (+$243 would-have) | REJECT at write |
| B2 | BD_MAX_PULLBACK 0.4×… / BD_CONFIRM_CLOSES 2 / BD_MAX_LEVEL_DIST 5.0×ATR | breakdown_continue.go | [I] | author-time refusal |
| B3 | A close "came back across" the breakdown level voids the breakdown | entry law (17:10:03 live reject quote) | [O] new entry law | REJECT at write |
| B4 | sweep_leg1_requires_touch — leg 1 needs a real sweep touch | kernel/entry_law.go | [O] | REJECT at write |
| B5 | FVG displacement floor ≥ max(2×tick, 2.0pt); gap sweet spot 20–80pt; CE width 20pt | kernel/fvg_entry.go:25-46 | [R] in-code citation | gate |
| B6 | min-SL ≥ 1.0×ATR5m + 2-tick clearance | kernel/min_sl.go:23-29 | [I/C] — no sweep; the 08-30 S1 cancel (stop 23.21 < 23.92 ATR) fired on it | **REJECT** (WARN-first violation candidate) |
| B7 | confirm staleness: a MET confirm farther than 2.0×ATR5m from ref_price is stale-MET, not the leak class | kernel/plan_confirm.go StaleConfirmATR | [I] | gate (arm/confirm evaluation) |
| B8 | Arm R:R ≥ 2.0 at arm time | trader/armed_executor.go:33 | [T] n=18 +$994 | gate |
| B9 | A resting limit on the marketable side fills instantly → never place | trader/armed_executor.go limitMarketableWrongSide | [R] 08-30 incident | gate (void) |
| B10 | Terminal arm rows re-authorize ONLY on plan-version change (manual-cancel-wins) | store/armed_orders.go UpsertArm | [R] 08-30 incident | gate |

## C. Computed signals

| # | Belief | Where | Label | Effect |
|---|--------|-------|-------|--------|
| C1 | HTF veto: 1h and 4h agreement blocks counter-trend entries (mode=cross) | kernel/htf_veto.go | [O] (P2 sweep KEEP) | gate |
| C2 | VIX regime bands <15 / 15-20 / 20-30 / >30 = context | kernel/regime.go:41 | [I] | advisory (dark-regime WARN) |
| C3 | ATR regime LOW/NORMAL/HIGH/EXTREME | kernel/regime.go:36 | [I] | advisory |
| C4 | Swing structure: k=2 fractals, min-move 0.25×ATR, MSS body 1.5×ATR | kernel/levels_swing.go | [T] swing-k (P2 keep) · min-move/MSS [I] | detector input |
| C5 | stale_reeval: superseded entry re-validated vs fresh bar; drift ≥0.25×ATR14 discards | trader/discard_burn.go:38 | [T] B1 −$372.5 saving | gate |
| C6 | stale-bar grace 15s; C2 tolerance 60s clock-drift | kernel/stale_data.go:46, clock_drift.go:26 | [R] C2 feed-stamp fix | gate |
| C7 | Fast-market wake when |Δprice| > 1.5×ATR5m since plan write | trader/auto_trader_loop.go:78 | [T] P2 sweep KEEP | wake trigger |
| C8 | Touch telemetry: band 16t, episode 12 bars, approach 5 | kernel/touch_telemetry.go | [I] "advisory, zero gates" | advisory |

## D. Level grader beliefs

| # | Belief | Where | Label | Effect |
|---|--------|-------|-------|--------|
| D1 | "Ground: freshness · departure speed · HTF alignment (SMC quality filters); RBD/DBR (reversal) > RBR/DBD (continuation)" | kernel/levels_score.go:133-134 | [R/O] v3 grading 08-24 | weight |
| D2 | zoneEvidenceByKind tiers + zoneTFMult 1.0/1.1/1.2/1.3 (effective 4h:1m ≈2.3×) | levels_score.go:149-157 | [I] values (R3 note: documented-not-changed) | weight |
| D3 | zoneReversalBonus ×1.1 | levels_score.go:159 | [I] | weight |
| D4 | zoneSizeMult ladder (≤0.3ATR ×1.25 … >2.5 ×0.50) | levels_score.go:205-222 | [I] | weight |
| D5 | Freshness ladder 1/.6/.3/.15 · anchor ladder 1/.8/.6/.5 | levels_score.go:437, levels_swing.go:22 | [I] | weight |
| D6 | Kind weights: swing .85, VWAP .85, VAH/VAL/SETT .80, MID-O .60, Round/Gap .55, zone-only .30 | levels_score.go:100-122 | [I] | weight |
| D7 | Proximity band ±0.3×dATR (owner) / default 1.5 | levels_score.go:396-412 | [O] (P2 keep) | filter |
| D8 | Cluster tolerance 12t (3pt) merges near-duplicates | levels_score.go:678 | [I] | filter |
| D9 | **Swing seats improve turn capture** | leveltruth wave (grand-audit) | **[X] own tape: missed-turns 80.0/75.0/79.2% → 65.0/60.0/66.7% with swing seats (Δ −15pts)** (grand-audit.md:74) | weight (still seated) |
| D10 | Consumed/3rd-touch/far-HTF → target_only, never entry | kernel/levels_role.go:28,107 | [I] doctrine | role demotion |

## E. Exits / lifecycle

| # | Belief | Where | Label | Effect |
|---|--------|-------|-------|--------|
| E1 | Breakeven at +40pt moves stop to entry | boot ledger (breakeven_trigger_points 40) | [O] | exit |
| E2 | Trailing 2.0×ATR14 after breakeven | boot ledger (trailing=2.0×ATR14, studio) | [O] | exit |
| E3 | EOD flat at session end (NY 14:45 CT) | kernel/session_registry.go (owner contract) | [O] | gate |
| E4 | Flip/death → dormant + auto re-arm, "replan budget untouched" | plan_lifecycle / dormant logs | [O] | lifecycle |
| E5 | Level-event wakes deserve a re-read | trader/auto_trader_wake_levels.go:279 | **[T] weak: 52 re-plans/7 days, 7 ever armed** (2026-09-02-level-event-wake-audit.md; WARN-first N=25 proposed) | REPLAN trigger (live) |

## F. Weekly reader

| # | Belief | Where | Label | Effect |
|---|--------|-------|-------|--------|
| F1 | NWOG (weekend gap: Friday ≤16:00 print → Sunday first print) is a weekly level | kernel/weekly_bias.go:47-58 | [I] | weight/advisory |
| F2 | IPDA range 20/40/60 trailing days | kernel/weekly_bias.go:58-61 | [I] | advisory |
| F3 | Weekly doc invalidated by 1h close beyond invalidation px | weekly reader (08-30 v1 bear invalidated 3s after write) | [O] | gate |
| F4 | Weekly DOA breach-at-write guard | 59dc9460 F5 wave | [O] | gate |
| F5 | Weekly confluence band 0.25×ATR / shadow mult 1.5 | kernel/weekly_knobs.go | [I] | weight |

## G. Doctrine flags (the named asks)

- **Fibonacci**: NOT FOUND anywhere live (grep across kernel/ + trader/ + prompts = 0). ✅
- **IPDA**: LIVE (weekly_bias.go:58, 20/40/60d) — advisory.
- **NWOG**: LIVE (weekly_bias.go:47) — advisory level.
- **Market-Profile 80% rule**: NOT FOUND as a rule; VAH/VAL/POC/nPOC exist as LEVEL KINDS (weights, not the 80% doctrine).
- **"Consumed touch" logic**: LIVE but demotive — `plan_lifecycle.go:107` (touched && consumed → level consumed) and `levels_role.go` (consumed/3rd-touch → target_only, never entry). Not an entry trigger anywhere.
- **Any [X] enforced as MUST/gate**: D9 (swing seats) is still a seated-weight input — [X] with live teeth. See demotion queue.
- **Any [I] enforced as REJECT (WARN-first law)**: **B6 min-SL 1.0×ATR5m** (REJECT at write, no sweep, caused the 08-30 S1 "gate changed: min_sl" cancel), **B2 BD_MAX_PULLBACK/BD_CONFIRM_CLOSES** (author-time refusals, waterfall family only partially taped), **B7 stale-confirm 2.0×ATR** (arm/MET gate). These are the law-violation candidates.

## Surprises (per dispatch rules)

1. **Class-35 (2026-09-01) already fixed the replan-budget accounting** this
   census would have flagged: recorded per-trigger spends replaced
   version−baseline arithmetic (`2026-09-01-class35-replan-budget.md:132` —
   `TestTriggerSpendsReplanClasses`, API now returns `replans_left == cap` for a
   6-row budget-free chain). Yesterday's "v1-v6 → 0" finding is resolved on dev.
2. The 09-01 wake-audit (origin/dev `a5a53bec`) pre-dates this census and
   already proposes the WARN-first demotion for level-event wakes (N=25) —
   listed in the demotion queue, not re-invented here.
3. The premium-FVG-window and Monday/Thursday conviction lines (A2/A3) carry
   no citation anywhere in docs/ — pure prompt doctrine.

---

## DEMOTION QUEUE — [X] or [I] beliefs with live teeth

| Rank | Belief | Label | Teeth | Demote to |
|---|---|---|---|---|
| 1 | **min-SL 1.0×ATR5m + 2t** | [I] | REJECT at write (cancels working arms mid-setup — the 08-30 S1 class) | WARN-first (author + WARN, place if owner-set otherwise); sweep the multiplier (knob census queue) |
| 2 | **Swing seats improve level turns** | [X] | seated weight; −15pt missed-turns on own tape | re-test vs baseline seats; likely unseat or reduce |
| 3 | **Level-event wakes deserve a re-read** | [T]-weak (52/7d, 7 armed) | full REPLAN trigger, budget-free | WARN-first N=25 proposal (already written); demote to advisory until n improves |
| 4 | **BD_MAX_PULLBACK 0.4 / BD_CONFIRM_CLOSES 2** | [I] | author-time refusal in the waterfall law | WARN-first; the rest of the BD family has tape, these two don't |
| 5 | **stale-confirm 2.0×ATR5m** | [I] | MET-confirm staleness gate (scenario dies on it) | WARN-first; measure stale-MET outcomes |
| 6 | **zoneTFMult + zoneEvidenceByKind + freshness/anchor ladders** | [I] | multiplicative weights on every level grade | sensitivity sweep (knob census §5 items 2-4) before any demotion of downstream gates |
| 7 | **Pre-NY sessions carry edge** | [X]-candidate (own tape 0/6 −$353.5 pre-NY vs NY 3/3 +$177, weekend P2 audit) | LONDON/ASIA sessions enabled + readable | owner decision: session enablement vs evidence |
| 8 | **Consumed/3rd-touch → never entry** | [I] | role demotion of levels | measure react rate of consumed levels vs first-touch (touch telemetry exists) |
| 9 | **Killzone/premium-window + Monday/Thursday conviction** | [I] | advisory prompt lines | keep advisory; add n counter or delete |
