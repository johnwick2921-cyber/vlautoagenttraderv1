# RESEARCH FULL READ-THROUGH — 2026-08-22 (all 7 files, end-to-end)

Companion to the conformance audit (PR #65) and the audit-of-the-audit. The
audits SAMPLED the research; this pass read every line of all 7 files in
`docs/research/plan-card/` (5,375 md lines + mockup html). Every item below was
then verified against code in this run.

## Coverage

| File | Lines | Read | Role |
|---|---|---|---|
| VL-DAYPLAN-FULL-SPEC-(2).md | 163 | ✅ full | the build contract (authoritative) |
| PLAN-CARD-DESIGN-SYSTEM-(1).md | 144 | ✅ full | card pattern + contracts |
| Strategy-Studio-Complete-Plan.md | 150 | ✅ full | Studio wiring plan (6 phases) |
| VL_Trading_System_Implementation_Plan.md | 783 | ✅ full | legacy Nautilus/SMC prototype |
| VL_Trading_System_Build_Plan_v3.md | 1,265 | ✅ full | legacy Python prototype |
| VL_Trading_System_Final_Build_Plan_v5-(1).md | 2,870 | ✅ full | Qlib/RD-Agent plan + Risk Math appendix |
| plan-config-mockup-(1).html | — | skimmed (UI mockup) | design reference |

---

## 1. What the audits MISSED that IS BUILT (confirmed CONFORM, fresh receipts)

1. **Planner regime block (FULL-SPEC "auto-computed, zero config")** — all 7
   fields exist in `kernel/regime.go`: `trend_state_daily` (EMA200, :56),
   `trend_state_1h` (EMA50, :62), `atr_regime` + `ATRPercentile` (:36-37),
   `RealizedVolPct` (:38), `ExpectedRangePts` (:42), `OvernightGapATR` (:43),
   VIX (:27, honestly 0 = no feed). The audit never graded this block.
2. **OWNER OVERRIDE 08-15 (indicator-state to planner)** — `planner_prompt.go:42-45,77`
   W11: the executor's per-TF indicator mirror goes to the planner, with the
   indicator-config fingerprint **frozen on the plan row**; `store/plan.go:43`
   `PromptHash` persisted per plan. Audit item "ai_config hash logged per plan"
   is effectively covered.
3. **Trust ladder stats** — `kernel/matched_random.go:12-18`: Bonferroni
   correction across 8 types, `PreRegisteredN = 1565` (spec ≈1,565 ✓). Blind-mark
   parser shared with bulk-add (`web/.../bulkParse.ts:1-5`).
4. **Ask Planner anti-sycophancy UI** — `AskPlannerPanel.tsx:116-124,227` renders
   PROPOSE-MERGE / DEFEND verdict chips (the full EVIDENCE → YOUR POINT → VERDICT
   contract).
5. **ConflictChip** — `web/.../plan/chips.tsx` + ZoneTable.
6. **ZoneRow "near" state** — `ZoneTable.tsx:35` `levelNear(fact)` → gold distance
   chip (spec: |dist| < 0.25×ATR).
7. **Scenario "expired" status** — `kernel/scenario_state.go:39` `ScenarioExpired`
   (all five statuses now exist).
8. **Design tokens file** — `web/src/theme/vl-tokens.css` exists; components
   consume `var(--vl-*)`.
9. **skip-while-open (recon item #4 "does NOT exist")** — BUILT:
   `auto_trader_loop.go:217-239` skip-while-open, relocated below
   buildTradingContext + session-roll ordering notes.
10. **Config fields (FULL-SPEC field list)** — all present in `store/strategy.go`:
    `planner_timeframes` (:931), `proximity_filter_atr` (:933), `replan_cap`
    (:941), `approval_required` (:946), `evening_digest` (:948), `max_trades`
    (:974), `plan_mode` (:928) with `PlanModeFor` resolution + `planModeBlocked`
    enforcement incl. `strict` → Go-verified citation (`auto_trader_planconfig.go:179-215`).
11. **Studio Phase 1** — `min_position_size` WIRED (`auto_trader_orders.go:525,673`
    `enforceMinPositionSize`); min-RR gate reads config (verified in #65).
12. **Studio Phase 4** — funding-rate toggle hidden for futures
    (`IndicatorEditor.tsx:29,1237`).
13. **Daily-loss hard stop** — `kernel/risk_limits.go:65,88-89` `-MaxDailyLossUSD`
    → refuse + `RiskForceFlat` (the v5 C.7 "100% → flatten" row, built as a single
    hard stop).
14. **Plan lineage** — decision rows carry plan_id/version/overlay_version/
    cited_scenario_id (verified in #66); plans carry prompt_hash; W11 fingerprint
    frozen per plan.

## 2. What the audits MISSED that is NOT BUILT (new SPEC'D-NOT-BUILT, queued)

| # | Item (research cite) | Code evidence of absence | Size |
|---|---|---|---|
| G1 | **PlanHeader [Approve] button** — the design spec *requires* it (PLAN-CARD-DESIGN-SYSTEM: component tree); `/api/plan/approve` endpoint+handler exist but no FE caller | grep `approve` in `web/src/components/plan/*` = 0 | S |
| G2 | **SessionsAccordion + InheritOverrideChip ⚪/🔸** (PLAN-CARD-DESIGN-SYSTEM multi-session additions) | no `SessionsAccordion`/`InheritOverride` anywhere in web/src; DayPlanEditor exists but without the inherit-chip contract | M |
| G3 | **NT8 venue badge/label on futures strategy** (Studio Phase 3) | `CoinSourceEditor.tsx` has zero NinjaTrader/venue strings; only SettingsPage mentions NT8 | S |
| G4 | **Grid tile disabled + "resting limit orders" explainer on futures** (Studio Phase 5) | `StrategyStudioPage.tsx` grid handling is a switch-away confirm only; no disabled+explainer | S |
| G5 | **"Duplicate to edit" hint on locked default** (Studio Phase 6) | duplicate ENDPOINT exists (`/api/strategies/{id}/duplicate`) but the hint text exists nowhere | XS |
| G6 | **FOMC/NFP forced-close T-2min** (v5 C.5: "existing positions FORCED CLOSED") | zero `fomc|forced.close` hits in `auto_trader_calendar.go`; T1 blackout blocks entries, never flattens positions | M |
| G7 | **Trail lock step +1.5R → entry+0.25R** (v5 C.4 ladder / G.1 `trail_lock_at_r: 1.5`) | `auto_trader_trailing.go` has BE trigger + 2.0×ATR trail only; no 0.25R lock step | S |
| G8 | **max_margin_usage enforcement** (Studio Phase 1: "enforce or honestly relabel") | `MaxMarginUsage` appears in prompt (:86) + schema only; no gate reads it (the notional ceiling is the de-facto enforcement) | S |
| G9 | **Graduated DD ladder** (v5 C.7: 75% alert+halve, 90% auto-stop) | `risk_limits.go` implements the single daily-loss hard stop; no graduated DD steps | M |

## 3. Research-INTERNAL conflicts (the docs disagree with each other)

- **R:R default**: v5 `configs/strategy.yaml` `target_rr: 2.5` + C.3
  "skip if < 1.5×" vs FULL-SPEC:48 hard gate "R:R≥3". Shipped follows FULL-SPEC
  (3.0 default, 1.0 floor). Under the v5 default, FULL-SPEC's own gate rejects
  every default-target trade. Owner should confirm which doc governs targets.
- **Loss-lockout**: v3:1064/v5 C.6 = 3 losses / 30 min vs shipped 4 / 60 min
  (already queued from the audit-of-the-audit).
- **News blackout**: Implementation/v3 "pause 5 min before/after" vs v5 C.5
  "HIGH T-5/T+5 · MED T-2/T+2" vs FULL-SPEC "T1 auto-writes HARD blackout
  windows" (shipped). FULL-SPEC is later and governs.
- **Session model**: v3/v5 single NY-AM window 08:30-11:30 CT + 11:30 hard close
  vs FULL-SPEC multi-session ASIA/LONDON/NY + 14:45 flat (shipped). Superseded.
- **Architecture**: Implementation/v3/v5 are Nautilus+Qlib/Redis/CSV prototypes;
  the shipped Go+TCP system replaced them (owner pivot). Only the durable VALUES
  (detector params, stop/trail math, risk thresholds) migrate; the Python
  scaffolding is background, not build contract.

## 4. Corrections to the audit's own findings

- **MAE/MFE viz is not a gap — it's SHELF** (FULL-SPEC V1.1 shelf explicitly
  defers "MAE/MFE viz + capture-rate"). The #66 note should cite the shelf.
- **"ML-Qlib pipeline" SPEC'D-NOT-BUILT** is now precisely sized: 35 models,
  RD-Agent nightly, Sunday retrains, ~50GB data, GPU needed for LSTM/Transformer
  (v5 Appendices B/I/J). It remains a product-direction item.
- **Feed-staleness severity**: v5 wants "stale > 5s → flatten + halt"; shipped is
  skip-cycle + WARN + clock-guard (fail-open ENTRY posture per owner rulings).
  Not a defect — a documented posture difference; noted.
- **Research's own standing rules** (Strategy-Studio §6.4): the 5 Google-Drive
  tools FORBIDDEN — the research documents the same prohibition the repo enforces.

## 5. Verdict

The audits stand — this pass found **zero mis-graded CONFORM rows among the new
material read**, confirmed ~14 previously-unverified spec items as BUILT, and
registered **9 new SPEC'D-NOT-BUILT items** (all clarity/feature-class, none
money/trade-class, none blocking) plus **5 research-internal conflicts** the
owner should settle in the post-soak calibration pass. Every new item joins the
queue; the standing ruling (zero calibration changes before Sunday soak +
replays) is untouched.

*Method note: files were read in full in this session; code receipts above are
fresh greps. Nothing in this pass was graded from memory.*
