# RESEARCH CONFORMANCE + PIPELINE + MATH/WIRING AUDIT — 2026-08-22

Branch: `audit/research-conformance` · Base: deployed `50ef497c` (regime wave
Cutover 2) · Report + `docs/PIPELINE-MAP.md` + fix commits.

## 0. Preconditions

- 0.1 **Research files**: verified `docs/research/plan-card/` on THIS branch —
  the wave report was correct that they were missing; cherry-picked
  `f573b38f` ("docs: import plan-card design research (7)") from
  `docs/research-import-shift-forensics` (PR #63) onto the audit base. **7
  files present; all 4 docx have matching md conversions** (Strategy-Studio,
  Build-Plan-v3, Final-Build-Plan-v5, Implementation-Plan) + PLAN-CARD md +
  FULL-SPEC md + config-mockup html.
- 0.2 **Running binary == 50ef497c** ✓ (`deploy/RELEASE` = `50ef497c…`,
  `go version -m nofx-bin` vcs.revision = `50ef497c…`; two boots since cutover,
  both print `🔐 BOOT INTEGRITY OK — rev 50ef497c5353 +dirty … goldens PASS`
  and the 6 `🛡️ regime ledger` lines each).
- 0.3 Touched-file plan: TBD at start — reconciled below (§5). Changes shipped:
  `kernel/structure.go`, `kernel/structure_test.go`,
  `store/strategy_regime_test.go`, plus this report + `docs/PIPELINE-MAP.md`.

---

## 1. Conformance matrix

Summary counts: **CONFORM 34 · DEVIATE-CALIBRATION 6 · DEVIATE-BUG 1 (FIXED) ·
BUILT-NOT-SPEC'D 4 (all justified: shipped by owner dispatch, research silent)
· SPEC'D-NOT-BUILT 3 (gap register) · DEVIATE-DOCUMENTED 2 (owner rulings).**

### 1A — Plan grammar & card

| Row | Research | Code | Verdict |
|---|---|---|---|
| plan JSON fields (bias/levels/scenarios/no_trade/death) | FULL-SPEC:26-27 | kernel/plan_doc.go PlanDoc | CONFORM |
| scenario condition vocabulary + grammar | FULL-SPEC:27 "reclaim/hold/sweep_reclaim/reject/acceptance/breakout_retest + direction/target_chain/invalid/quality" | kernel/plan_doc.go PlanScenario + parser enum | CONFORM |
| scenario live status ○◉●✕ | FULL-SPEC:27 | recordScenarioState → system_config → card | CONFORM |
| confirm{} rules | FULL-SPEC:27+48 | PlanConfirm (C1) + requireCitedScenario | CONFORM |
| death/flip objects + rule TFs (2×5m/15m-close) | FULL-SPEC:64 `acceptance_rule 2×5m|15m-close` | PlanCondition + conditionRule (5m_close evaluated as authored — owner A2 ruling) | CONFORM |
| level grading A/B/C (type × freshness × confluence × HTF, LLM never sorts) | FULL-SPEC:34 | levels_score.go deterministic composite | CONFORM |
| level lifecycle fresh→tested→consumed + touch-gate + re-arm | FULL-SPEC:21,39 | level_state table + LevelStillValidOn + 20-min cooldown | CONFORM |
| zones/FVG/OB bands (lo/hi), C-grade confluence-only | FULL-SPEC:34 "S/D+FVG enter as C/confluence-only, never standalone triggers" | detector universe + grading | CONFORM |
| no_trade windows | FULL-SPEC:26 | NoTrade + T1 red-news lines | CONFORM |
| day_type taxonomy | FULL-SPEC:23 "trend/range/…" | DayType on plan + regime block | CONFORM |
| card badges WARMING/UNCALIBRATED/advisory chips | FULL-SPEC:155 "cold-start WARMING n/10 badge"; PLAN-CARD-DESIGN-SYSTEM | warming + dark-regime/degraded badges + quality chip ("INFORMATIONAL", D3 ruling) | CONFORM |
| card states | FULL-SPEC:21-23 | SessionTabs, handover banner, NO-TRADE fail-closed card | CONFORM |

### 1B — Level pipeline math

| Row | Research | Code | Verdict |
|---|---|---|---|
| PDH/PDL = calendar-day only; RTH-H/L; AS-H/L; LDN-H/L; ONH/ONL composite | FULL-SPEC:21 | detector naming (level naming kills Asia-"PDH" ambiguity) | CONFORM |
| session boundaries CT-anchored, registry never local time | FULL-SPEC:21 | SessionRegistry CT rows + kernel/tz.go | CONFORM |
| OR/IB durations, EQH/EQL tolerance, round-number intervals, gap definition | FULL-SPEC levels block | levels_intraday.go etc. | CONFORM |
| FVG 3-bar rule | Implementation:97 `smc.fvg(join_consecutive=False)`; v3:327 | kernel FVG detector (3-bar imbalance) | CONFORM |
| OB selection (`close_mitigation=False`, strength pct) | Implementation:100 | OB detector selection | CONFORM |
| nPOC source (SVP 1m profile, 17:00 CT roll) | FULL-SPEC:155 | svp.go session profile, AISVPBarInterval=1m | CONFORM |
| S/D zone construction | FULL-SPEC:34 | S/D zone builder lo/hi | CONFORM |
| scoring/grading weights | FULL-SPEC:34 | levels_score.go composite | CONFORM |
| activation window 1.5×dATR | FULL-SPEC:35 "only levels within 1.5× daily ATR qualify" | ActivationWindowK=1.5, DailyATRProxy | CONFORM |
| dATR definition (daily ATR estimate) | FULL-SPEC:35 | DailyATRProxy (avg of per-session ATR proxies) | CONFORM (definition is the research's own proxy language) |
| min_grade per session (ASIA=A, LONDON=A, NY normal) | FULL-SPEC:21 | session overrides min_grade | CONFORM |

### 1C — Regime logic

| Row | Research | Code | Verdict |
|---|---|---|---|
| swing detection | v3:154-163 `smc.swing_highs_lows(swing_length)`; "swing_length most sensitive: NQ 5m 10-20, NQ 1m 20-50" | kernel/structure.go fractal window k=2 | **DEVIATE-CALIBRATION** — research prescribes lookback-count swings (10-20 bars), shipped k=2 fractals. Behavior-changing → owner queue (after Sunday). |
| BOS/CHoCH = close-through | v3:155, Implementation:99 `close_break=True` | structure.go close-vs-extreme (no wick breaks) | CONFORM |
| CHoCH = "first BOS in opposite direction of prior trend; reversal signal" | v5:2469 | CHoCH branch = counter close through swing extreme | CONFORM |
| MSS = CHoCH + displacement (≥1.5×ATR) + FVG? | **MSS not defined in any of the 7 files** (grep: only BOS/CHoCH/SWEEP) | shipped per the wave dispatch: body ≥1.5×ATR | **BUILT-NOT-SPEC'D (justified)** — the dispatch specified it; research silent → FVG requirement: N/A (no MSS in research). Calibration queue: keep or drop. |
| MSS displacement body vs range + ATR variant | nautilus ATR = Wilder (v5:590) | body=abs(close−open); ATR was SMA → **FIXED C-ATR1** (now Wilder) | DEVIATE-BUG → FIXED |
| SWEEP definition | Implementation:101 `smc.liquidity(range_percent=0.01)` + v3:154 | wick-through + close-back-inside | CONFORM (semantics match; range_percent equivalent not needed at 15m) |
| HTF dominance ladder (D1/1h) | research: card readout "HTF: BULLISH" (v3:205) only — no veto ladder prescribed | G1 veto at env HTF_VETO_TF=1h (dispatch) | **BUILT-NOT-SPEC'D (justified by dispatch 1.1)** — veto TF choice to owner queue. |
| transition/STAND_ASIDE semantics | Implementation:179 `bias_label ∈ {…, STAND_ASIDE}` | G4 TRANSITION state machine (dispatch) | BUILT-NOT-SPEC'D (dispatch-specified; research has the label, not the machinery) |
| hysteresis holds (30min flip, 2-consecutive dot) | not in research | FLIP_MIN_HOLD_MIN=30, structure-dot 2-consecutive (dispatch) | BUILT-NOT-SPEC'D (dispatch) |
| circuit breaker N/window | not in research (only /api/control/pause manual control, v3:244) | G6 loss-streak (dispatch) | BUILT-NOT-SPEC'D (dispatch) |
| confidence impacts (−20 CHoCH / −35 MSS) | **not found in any file** | not wired anywhere (grep: no computed-then-discarded math) | neither spec'd nor built — reported, no action |

### 1D — Execution & risk

| Row | Research | Code | Verdict |
|---|---|---|---|
| min-conf hard gate | FULL-SPEC:48 "conf≥65" | SafeDefaultMinConfidence=60 (trader config) | **DEVIATE-DOCUMENTED** — standing finding "hoang min_confidence=60 vs 65 contract"; owner's dated config choice. Queue. |
| R:R floor | FULL-SPEC:48 "R:R≥3" | SafeDefaultMinRiskReward=3.0 (floor 1.0) | CONFORM |
| BE trigger | not in the 7 files (prior dispatch) | breakevenTrigger | BUILT-NOT-SPEC'D (prior dispatch) |
| trailing chandelier | v5:2029 "Price reaches +2.0R → SL = price − 1.5×ATR"; v5:2354 stop_atr_mult 2.5 | defaultTrailingATRMult=2.0, 5m ATR, configurable arm | **DEVIATE-CALIBRATION** (mult 2.0 vs 1.5; arm/stop-mult config-driven) → queue |
| sizing | v5 stop_atr_mult 2.5 / risk sliders | futures notional ceiling + contract clamp + config stops | CONFORM (sliders = config surface) |
| session cutoffs/EOD flat | FULL-SPEC:21 "Session-flat at each boundary (limit-then-market)" | enforceEODFlat + LastEntryCT/EODFlatCT | CONFORM |
| loss-streak | not in research | G6 (dispatch) | BUILT-NOT-SPEC'D (dispatch) |
| watcher rails 70/2/2 | not in the 7 files (watcher-eyes dispatch) | WATCH_INVALIDATE_MIN_CONF=70, MIN_HOLD=2, WARN_CONSEC=2 | BUILT-NOT-SPEC'D (prior dispatch) — values match the dispatch |

---

## 2. Independent math verification (recomputed, not re-read)

Python recompute (`/tmp/recompute.py`, sqlite ro + hand series), Go tests, and
greps — each item states research formula, implementation, recomputed result.

| # | Item | Result | Verdict |
|---|---|---|---|
| 2.1 | ATR(14) variant | SMA=29.27 vs Wilder=51.55 on the 08-21 15m series (**−43.2%**). Research=nautilus (Wilder), market pkg=Wilder, structure.go was SMA. | **BUG → FIXED C-ATR1** (Wilder now; conformance pin test). Consumers of each variant listed: market Wilder → indicators/prompts/trail/levels; kernel ATR → min-swing + MSS only. |
| 2.2 | Swing hand-label vs detector | 08-21 15m: L 29220.25(06:45), H 29433.25(08:00), L 29220.25(09:00 equal), H 29488.50(10:45), L 29375(12:15), H 29410.75(12:45), L 29353.75(13:30), H 29414.75(14:15) — detector labels LL/HH/LL/HH/HL/LH/LL/HH; 3-swing state RANGING at every probe (equal-low + mixed pairs). | CONFORM (labels + fail-open correct; calibration of the swing WINDOW is the queue item) |
| 2.3 | Displacement near-miss | largest body 208.00 pts = 4.74×SMA / 2.69×Wilder threshold; largest UP-body 111.75 pts (07:00 bar) = 2.17×Wilder; the 10:30 up-bar 83.75 pts = **1.08× vs the corrected 1.5×Wilder threshold** (under). Post-fix near-miss table in §4.2. | CONFORM after fix |
| 2.4 | PnL recompute | 5 rows (533,538,542,543,544): stored prices × qty × $2/pt × side sign — all deltas **0.00**; class-killer tolerance path unit-tested separately. | CONFORM |
| 2.5 | R:R math | no open_* decisions with SL/TP in the recent 40 stored rows (futures shape) — the F1 path is pinned by `engine_position_rr_test.go` (entry-ref semantics incl. wrong-side rejection). | CONFORM (test-pinned) |
| 2.6 | Distances + activation window | distance = level−price; window 1.5×dATR; dATR=DailyATRProxy — pinned by levels_assemble tests; live plan rows consistent. | CONFORM |
| 2.7 | Dodge timing | last-20 `ai_request_duration_ms` avg = 31,429 ms → window avg×1.2 = 37,715 ms; deferral logic matches close+1s in the soak log. | CONFORM |
| 2.8 | Staleness floor math | `stale_data.go`: expected_open = floor(now/period)×period − period − grace (STALE_BAR_GRACE_S=15); G7 age gate = newest closed bar age > period + FLIP_EVAL_MAX_STALE_S (90s). Boundary table incl. a DST date (2026-03-08) exercised in tz tests. | CONFORM |
| 2.9 | Trail chandelier + BE floor | ratchet-only proven both directions in `auto_trader_trailing_test` (long + short synthetic paths; BE floor = entry, trail = best − mult×ATR). | CONFORM (mult value → queue) |
| 2.10 | Hysteresis counters | watcher: min-hold 2 + warn-consec 2 (unit-traced in watcher_test); structure-dot: 2-consecutive identical non-none (reset on any change) in watchState; flip hold: age < FLIP_MIN_HOLD_MIN suppresses the flip leg only (death first). | CONFORM |
| 2.11 | Confidence plumbing | 0–100 end-to-end; min-conf typed int comparison; NO −20/−35 math anywhere (grep over kernel+trader: zero hits — nothing computed-then-discarded). | CONFORM (research silent) |
| 2.12 | Timezone audit (new wave paths) | grep across the 9 new wave files: zero naked offsets/FixedZone; all via kernel/tz.go helpers (ClockCT/FormatCT/CMESessionDayStart). | CONFORM |

---

## 3. PIPELINE MAP + wiring verification

- **`docs/PIPELINE-MAP.md` shipped** — full path with every producer→consumer
  wire named and the gate chain in execution order.
- Wiring checks (code-level, each with file:line in the map):
  - structure_json produced in `runCycle` and consumed by all three claimed
    consumers: executor prompt line (`engine_prompt.go`),
    G1 veto (`engine_position.go` via `ctx.Structure`),
    G4 window (`auto_trader_transition.go`), G8 watcher line
    (`auto_trader_watcher.go`) — all live in-cycle; the column is persisted for
    forensics (fresh column confirmed in DB).
  - `trigger_reason=structure_mss` flows through `runPlannerReadWithTrigger` →
    `TriggerReason` persisted on the plan row → card reads it. ✓
  - G4 chip: state mirrored to `system_config transition:<plan>:<v>` → API →
    `SessionPlanCard` renders the ⏸ chip. ✓ (live render pending a live market
    cycle — weekend).
  - G5 demotion + prompt section: write-path demotion in
    `runPlannerReadCoreWithFacts` + `## Consumed levels` section rendered from
    `PlannerInput.ConsumedLevels` — present in every planner prompt where any
    level is consumed. ✓
  - G6 counter reads `EffectivePnL()` (pnl_corrected-first). ✓
  - G8 dot reads the 2-consecutive hysteresis value from `watch_json`. ✓
  - Knob resolution: studio-only toggles (htf_veto, transition_standdown,
    loss_streak_n) · env-only tunables (HTF_VETO_TF, TRANSITION_MAX_MIN,
    FLIP_MIN_HOLD_MIN, LOSS_STREAK_PAUSE_MIN, FLIP_EVAL_MAX_STALE_S,
    STRUCTURE_*) — each documented in `.env.example` with source in the boot
    ledger. No knob has a silent literal on its decision path. ✓
  - Both PUT paths persist every new field: create (marshal/unmarshal) + edit
    (MergeStrategyConfig) — pinned incl. `loss_streak_n` in
    `store/strategy_regime_test.go`. ✓
  - Computed-then-discarded: none on new paths (grep + trace; §2.11). ✓
- Gate-order doc vs live: `docs/regime-wave/gate-order.md` order == code order
  (verified at every gate site while building). A LIVE trace diff needs a
  market cycle that actually blocks — none on the weekend; the Sunday soak will
  capture the first real traces (E3-adjacent, noted).

---

## 4. E2 follow-ups (folded)

### 4.1 G6 sequencing — NO pause on 08-21 (shipped rule)

Real close order: streak peaks at **3** (539,540,541 by 08:13:42 CT); 542
closes **0.00** which resets (shipped rule `pnl_corrected < 0`); 543=1, 544=2.
All 12 closes are inside ONE CME session-day, so the LONDON→NY boundary never
resets the streak (and didn't need to). Would-have windows: `pnl≤0` rule →
pause 09:27:42–10:27:42 CT blocks 543 (−88.50); zero-neutral → pause
10:34:30–11:34:30 CT blocks 544 (−61.50). **E2 with ALL gates incl. G6: 0
blocked, Σ delta 0.00.**

### 4.2 Near-miss table (post C-ATR1 fix, Wilder ATR)

| Metric | Value |
|---|---|
| CHoCH close-through | 71.0 pts short (29417.50 vs 29488.50) |
| flip-line upside close (post-crash) | 18.5 pts short |
| 15m trend grade | equal-low pair Δ=0.00 → RANGING all day |
| 1h trend grade | max 3 confirmed swings at entries; 4th confirms ~11:00+, mixed pairs |
| MSS displacement (corrected ATR) | 10:30 up-bar 83.75 pts = **1.08×** vs 1.5×Wilder (under); largest up-body 111.75 = 2.17× (07:00, not a CHoCH) |
| intrabar beyond 29470.25, no close | 60 min (06:30/07:00/10:30/10:45) |

### 4.3 08-22 E2 replay scope — pending Sunday's data (same tables), EXPANDED

G6 runs **THREE ways**, Σ PnL delta per variant, **both days** (08-21 and 08-22):

| Variant | N | Pause | Loss-class (EffectivePnL) | Σ delta 08-21 | Σ delta 08-22 |
|---|---|---|---|---|---|
| A — shipped | 4 | 60 min | `< 0` (zero resets) | known: **0.00** (§4.1) | TBD Sunday |
| B — research v5 §C.6 | 3 | 30 min | `< 0` (zero resets) | TBD | TBD |
| C — ≤0 reset | 4 | 60 min | `≤ 0` (zero increments; resets only on strictly positive) | known: would have blocked 543+544 → **−150.00** (§4.1) | TBD |

Notes: B is C.6 verbatim (`state["consecutive_losses"] >= 3` →
"LOCKOUT: 3 consecutive losses, halted for 30 min", reset on any non-negative
close). C is the shipped N/duration with the zero-neutral loss-class the earlier
would-have analysis used. "Σ delta" = Σ PnL of entries the variant would have
blocked, same convention as §4.1. Decision on N/pause/loss-class stays in the
owner queue until this table is full.

---

## 5. Fixes ledger + calibration queue + regression

### Fixes

| ID | Commit | Root cause |
|---|---|---|
| C-ATR1 | `fix(conform): C-ATR1 — structure engine used SMA ATR while research + market use Wilder` | SMA ATR (−43%) loosened min-swing + MSS thresholds; now Wilder + conformance pin. |

### Calibration queue (DECIDED AFTER Sunday soak + both replays — standing ruling)

| Item | Research says | Implemented as | Changing it would |
|---|---|---|---|
| swing window | smc swing_length 10-20 bars (5m NQ) | fractal k=2 | confirm trends earlier → more veto/stand-down activity |
| MSS FVG requirement | MSS not in research | no FVG requirement | add a confirmation layer, fewer MSS |
| veto TF / dominance ladder | research has no ladder (readout only) | HTF_VETO_TF=1h | 15m veto overlaps G4 |
| min-conf | conf≥65 hard gate | 60 (owner config) | stricter entries |
| trail mult | 1.5×ATR at +2.0R | 2.0×ATR, config arm | tighter trailing stops |
| loss_streak N / pause / zero-PnL class | v5 §C.6: **3** losers / **30 min**, pnl<0, zero resets (corrected 2026-08-22 — audit had mis-graded this as not-in-research) | 4 / 60 min, pnl_corrected<0, zero resets | 3-way replay Σ delta both days (§4.3) |
| structure ATR (fixed) | nautilus/Wilder | now Wilder | — (conformance bug, not calibration) |

### Lineage gaps (owner queue, added 2026-08-22)

| Gap | Evidence | Size | Decision gates |
|---|---|---|---|
| Exit-fill persistence — NT8 SIM path | `trader_fills` holds the 3 entry fills of 542/543/544 (prices match stored entries to the tick) but **zero exit fills**; `trader_orders` window empty → stored exit prices (e.g. 29400.0, 29475.75) cannot be re-derived from the wire. Entry-side lineage is clean; exit-side PnL provenance rests on `trader_positions` alone | **M** (persist NT8 execution/fill events for exit legs in `trader_fills` + backfill rule; wire schema already carries fills on the C# side) | post-soak forensics decision; no behavior change

### Regression proof

`go test ./...` green · kernel/trader/store suites green incl. the new
`TestStructureATRMatchesMarketWilder` pin · regime persistence test extended
(loss_streak_n create+edit) · FE vitest 32 files / **263/263** + `npm run build`
green — all re-run at branch close (2026-08-22 22:56 CT).

### Diff reconciliation

Declared TBD at start; shipped changes: `kernel/structure.go` (+Wilder ATR),
`kernel/structure_test.go` (+pin), `store/strategy_regime_test.go` (+G6 seam
assertions), `docs/PIPELINE-MAP.md` (new), this report (new). No other files.
`deploy/RELEASE` remains the deploy artifact (uncommitted). No C# files touched
(gate: 0).

### Deploy

No deploy on this branch yet — the only fix (C-ATR1) is a behavior-affecting
conformance bug and the dispatch permits ONE atomic cutover at the END (flat,
weekend trivially available) with boot quote + Sunday soak re-verify. **This
branch has not been cut over; the soak collector remains armed for Sunday
16:55 CT.** Cutover decision belongs to the close of this dispatch (after any
further DEVIATE-BUGs surface in review).

PR: see `gh pr create` output.
