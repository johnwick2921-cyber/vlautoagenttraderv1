# DAY-PLAN CAMPAIGN — IN PROGRESS

Owner-approved 2026-08-14. Build contract: [docs/VL-DAYPLAN-FULL-SPEC.md](docs/VL-DAYPLAN-FULL-SPEC.md)
(v1 FINAL, code-verified recon @ ca1f38c6). Multi-session, checkpointed (hardening pattern).

**Standing rules:** own commit per item · push-per-part · ADDITIVE · SIM-only · NO agent restarts
(owner restarts at ★ points only) · guardrails master untouched (OFF = owner learning mode).

**STEP 0 (this session, HEAD c051b975):** PASS — HEAD ≥ 3624a2a4 ✓ · tree clean ✓ ·
running rev 3624a2a4 (HEAD ancestor, FE-only since) ✓ · both traders cycling (hoang #440, 15m #84) ✓.

---

## Part ledger

Legend: ⬜ not started · 🔧 in progress · ✅ done+committed · 🚀 pushed

### P0 · FOUNDATIONS ✅ COMPLETE (pushed)
- ✅ **P0.1** `d1851dac` day_plan config JSON — both codecs, ROOT placement, round-trip golden FIRST + MergeStrategyConfig survival (`+config-truth`)
- ✅ **P0.2** `0a974d31` plans/plan_overlays append-only + decision FK + WAL + single-writer goroutine (25-way concurrent proof, -race clean)
- ✅ **P0.3** `6a0d233b` CT-anchored session registry — ASIA/LONDON/NY, NY-only enabled, killzones, half-day hook (dormant); NOT yet wired to live gate (P2)
- ✅ **P0.4** `b51ab5c2` scenario-fact evaluator (keystone) — distance/closes_beyond/sweep/acceptance/reclaim/reject/still_valid, 11 fixture tests
- ✅ **P0.5** `041e4450` level-state table — identity-keyed (times_tested/consumed/freshness A→B→C), cross-session persist; snapshot writer = P1 (RECON #2)
- ✅ **P0 EXIT BAR** — build/vet/`test ./...`/-race(store,kernel) green · goldens untouched · config-truth locked · zero FE touched (tsc/npm N/A)

**P0 zone flag for owner:** NY flat encoded 14:45 CT (= 15:45 ET, pre-close). Admin-editable, not yet live. Confirm at ★ OWNER RESTART 1.

### P1 · THE MAP — ✅ COMPLETE (8/8, pushed) · ★ OWNER RESTART 1 after P2
- ✅ **P1.1** `14adc47e` multi-day extractor (PDH/PDL/PDC · RTH/AS/LDN/ON · PW/PM)
- ✅ **P1.2** `e1cd5993` round numbers + gap tracker (fill-state) + OR/IB (+1.5×/2× ext)
- ✅ **P1.5** `3058e131` confluence scorer → graded TOP-8 + KEY LEVELS renderer
- ✅ **P1.6** `15e55520` regime block (7 fields, graceful VIX/RV degrade)
- ✅ **P1.4** `f138b7f3` EQH/EQL + S/D zones + FVG + OB (C/confluence-only)
- ✅ **P1.7** `c42a629c` KEY LEVELS block wired into the LIVE futures prompt (gated on day_plan; existing goldens byte-identical; NEW enabled golden)
- ✅ **P1.3** `a450adfc` durable session-profile store + 17:00-CT snapshot writer + nPOC detector (RECON #2); restart-safe, dormant
- ✅ **P1.8** `09579af3` calendar fetcher (ForexFactory + static T1 fallback, graceful outage) + replay-frozen per-day slice store
### P2 · THE CLOCK — ✅ COMPLETE (5/5, pushed) · ★ OWNER RESTART 1 procedure below
- ✅ **P2.1** `069500c9` true bar-close cadence (gated; barCloseGate + tickOnce; scan-timer default unchanged)
- ✅ **P2.2** `3bb7a730` skip-while-open gate (RECON #5 built; holding → skip AI cycle; bracket/breakeven independent)
- ✅ **P2.3** `21c57118` last_entry (13:00 CT) + eod_flat (14:45 CT, close-path flatten, half-day pull-in)
- ✅ **P2.4** `a43d006d` MAE/MFE + entry-confidence (additive columns; ComputeExcursion; close-time compute)
- ✅ **P2.5** `b0151d98` cmd/dayplan-arm (guarded/idempotent/dry-run arm tool)
- ✅ **EXIT BAR** build/vet/`test ./...`/-race green · NO golden touched · config-truth (persistence proven)
- **★ RESTART 1 handoff** in docs/superpowers/reports/2026-08-14-dayplan-p2-clock.md (stop→rebuild→backup→arm→start + 5-line VERIFY)

### P2 · (old placeholder) — superseded above
### P3 · THE PLANNER — ✅ COMPLETE (pushed) · dormant until ★ RESTART 2 (after P5)
- P3.6 FINISH: A digests `f9f5460f` · B scenario re-arm `29727291` · C sticky owner levels `9f000637` · D night mode `e19403fc`
- ✅ **P3.1** `389c0c9e` session gates — entries only inside an ENABLED session window (NY-only) + no-trade sub-windows (first-5m, lunch); gated/dormant
- ✅ **P3.2** `05703a08` planner model binding (RECON #12) — resolvePlannerClient, pinned ID, empty→primary
- ✅ **P3.3** `5d06b063` read jobs + planner call — plan-doc schema+validator, input-package assembler, call core (≤2 retries→fail-closed NO-TRADE), scheduler, plan rows; gated/dormant
- ✅ **P3.4** `d0d96a10` executor plan injection + RECON #4 reorder (PLAN BLOCK→prefix, SVP/KEY-LEVELS/STATUS→tail); NEW golden futures_mnq_plan.golden, existing byte-identical; sample ③ ✓
- ✅ **P3.5** `d2d4a975` advisory mode — decision cited_scenario + ClassifyCitation + match-rate counters via B6 (never gates)
- ✅ **P3.6** core `0a292148` (activation window + death→re-plan + restart recovery) + FINISH: A digests (`f9f5460f`), B scenario re-arm via level-state (`29727291`), C sticky owner levels (`9f000637`), D night mode (`e19403fc`)
### P4 · THE CARD — ⬜ (SessionTabs/timeline/HandoverBanner/PlanCard, chart overlay, alert center, Studio block, /api/plan/*)
### P5 · THE DOOR — ⬜ (overlay API, edit sheet + bulk-add, conflict chip, Ask-Planner, adherence grade, digest, stats gate, blind-mark prep) · ★ OWNER RESTART 2

---

## Checkpoint log
- **2026-08-15 ~03:00 CT** — **P3 · THE PLANNER COMPLETE ✅.** P3.6 finish (A digest writers f9f5460f · B scenario re-arm via level-state 29727291 · C sticky owner levels 9f000637 · D night mode e19403fc). EXIT BAR green (build/vet/test/-race); NO golden touched (A-D are storage/logic only). All gated/dormant until ★2. **CAMPAIGN: P0 ✅ P1 ✅ P2 ✅ (★1 live) P3 ✅.** NEXT: **P4 · THE CARD** (SessionTabs/timeline/HandoverBanner/PlanCard, chart overlay, alert center, Studio Day Plan block, /api/plan/*) → then P5, then ★ RESTART 2. Report: docs/superpowers/reports/2026-08-15-dayplan-p3d-finish.md.
- **2026-08-15 ~02:00 CT** — **P3 · PLANNER — executor injection + advisory + lifecycle-core.** P3.4 (d0d96a10) plan→executor reorder + golden + sample ③; P3.5 (d2d4a975) advisory cite/match-rate; P3.6 core (0a292148) activation window + death→re-plan + RESTART RECOVERY (mandatory fixture proves stateless recompute is identical pre/post restart). EXIT BAR green (build/vet/test/-race); goldens = ONLY the new futures_mnq_plan.golden (existing byte-identical). Samples ①②③ in the report. P3 is P3.1-P3.5 ✅ + P3.6 CORE ✅; P3.6 finishing sub-features (digests / scenario re-arm via level-state / sticky owner levels / night mode) remain. All gated/dormant until ★2. Report: docs/superpowers/reports/2026-08-15-dayplan-p3c-planner.md.
- **2026-08-15 ~01:00 CT** — **P3 · PLANNER CORE CHECKPOINT (3/6).** Built P3.3 (the planner core, 5d06b063): plan-doc schema + strict validator (`kernel/plan_doc.go`), input-package assembler (`kernel/planner_prompt.go` — regime/ranked-levels/calendar/digest/owner-note), the call core (`trader/auto_trader_planner.go` runPlannerReadCore: ≤2 retries → FAIL-CLOSED NO-TRADE plan + alert event; writes append-only plan rows with prompt_hash+model_id), the per-session scheduler (maybeRunSessionReads, wired into runCycle). Suite + -race green; NO golden touched (P3.4 reorder not yet done); all gated/dormant. Samples ① (assembled planner prompt) + ② (schema-valid plan JSON) captured in the report. REMAINING (next session): **P3.4 executor injection reorder + GOLDEN REGEN** (sample ③), **P3.5 advisory**, **P3.6 lifecycle + restart recovery**. Report: docs/superpowers/reports/2026-08-15-dayplan-p3b-planner.md.
- **2026-08-15 ~00:15 CT** — **P3 · THE PLANNER CHECKPOINT (2/6).** ★1 confirmed done (owner rebuilt+restarted 08-14 23:06, new binary PID 778475, day_plan armed, KEY LEVELS + cadence live). Built the two contained foundational items: session gates (P3.1, 389c0c9e) + planner model binding (P3.2, 05703a08). Suite + -race green; NO golden touched; all additive+dormant until ★ RESTART 2 (after P5). REMAINING (next session, budget-edge checkpoint): **P3.3 read jobs / AI planner core** (input package, schema-strict fail-closed, plan rows), **P3.4 prompt reorder + golden regen** (RECON #4), **P3.5 advisory**, **P3.6 lifecycle + restart recovery** — plus the exit-bar's 3 assembled samples (need the planner running). Report: docs/superpowers/reports/2026-08-15-dayplan-p3-planner.md.
- **2026-08-14 ~22:45 CT** — **P2 · THE CLOCK COMPLETE (5/5).** Bar-close cadence, skip-while-open (RECON #5 built), last_entry/eod_flat clock + enforcement, MAE/MFE + confidence, arm tool. 5 commits (069500c9→b0151d98). EXIT BAR green (build/vet/test/-race trader+kernel+store); NO golden touched (P2 has no prompt changes); all GATED on day_plan → running bot untouched. ★ RESTART 1 handoff (stop→rebuild→backup→arm→start + 5-line VERIFY) in the p2-clock report. NEXT after ★1: **P3 · THE PLANNER**. Report: docs/superpowers/reports/2026-08-14-dayplan-p2-clock.md.
- **2026-08-14 ~18:10 CT** — STEP 0 PASS. Spec + heartbeat committed. P0 recon workflow launched. Starting P0.1 (golden-first).
- **2026-08-14 ~18:40 CT** — **P0 · FOUNDATIONS COMPLETE.** All 5 items committed (d1851dac / 0a974d31 / 6a0d233b / b51ab5c2 / 041e4450) + config-truth test. EXIT BAR green (build/vet/test/-race). All ADDITIVE + dormant until wired (P1-P4) — the running bot (rev 3624a2a4, PID 363618) is untouched; NEW tables + WAL activate only at the next owner-driven rebuild+restart (★ RESTART 1). Report: docs/superpowers/reports/2026-08-14-dayplan-p0-foundations.md. NEXT: P1 · THE MAP (detectors + scorer + regime + key-levels prompt block + calendar).
- **2026-08-14 ~20:30 CT** — **P1 · THE MAP COMPLETE (8/8).** Finished the remaining 4: P1.4 zones (f138b7f3), P1.7 live-prompt wire (c42a629c, gated on day_plan → existing goldens byte-identical, ONE new enabled golden), P1.3 durable store + snapshot writer + nPOC (a450adfc, dormant + restart-safe), P1.8 calendar (09579af3, graceful outage + replay-frozen slice). EXIT BAR green: build/vet/`test ./...`/-race(kernel+store+trader+calendar). Goldens: only the NEW futures_mnq_keylevels.golden; all existing UNCHANGED. Report: docs/superpowers/reports/2026-08-14-dayplan-p1b-map.md. All additive+dormant; running bot untouched. NEXT: **P2 · THE CLOCK** (bar-close cadence, skip-while-open gate, last_entry/eod_flat, MAE/MFE) → then ★ OWNER RESTART 1.
- **2026-08-14 ~19:15 CT** — **P1 · THE MAP CHECKPOINT (4/8 items).** Built the pure/deterministic backbone: multi-day extractor (P1.1), round/gap/OR-IB detectors (P1.2), confluence scorer + KEY LEVELS renderer (P1.5), regime block (P1.6), + an end-to-end sample-block test. 5 commits (14adc47e→9436ea79), all pushed. Full suite + -race(kernel,store) green; goldens + FE UNTOUCHED (all new files); running bot still untouched. Sample KEY LEVELS block captured in the report. REMAINING (next session, budget-edge checkpoint — never start an item I can't finish): P1.3 durable store+snapshot writer (loop-coupled), P1.4 zones, P1.7 prompt wiring (goldens deliberate; scorer+renderer ready), P1.8 calendar. Report: docs/superpowers/reports/2026-08-14-dayplan-p1-map.md.
