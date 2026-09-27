# MASTER SYSTEM RECHECK — FULL CHECKLIST VERIFICATION

**Date:** 2026-08-27 · **Scope:** running system post-armed-orders cutover · **Worktree:** `/home/hoang/nofx-recheck` @ `4f2cee03` (branch `docs/master-recheck`) · **READ-ONLY** — zero code/config/DB writes.

**Evidence law compliance:** every ✅ carries fresh evidence produced in this run (query timestamp CT, quoted log line, or independent Python recomputation from `data/data.db` in read-only mode — never the function under test, never a prior report). Anything without fresh evidence is marked UNVERIFIED.

**State at check time:** 2026-08-27 ~10:05 CT. **The armed-orders cutover HAS HAPPENED** (boot 09:25:13 CT). All Section-1.2/1.8/6.7 checks were therefore run against the NEW state.

---

## SECTION 1 · WIRING / INFRASTRUCTURE

**1.1 TCP NT8↔Go live — ✅ PROVEN.** `2026/08/27 10:01:25 INFO tcp_server: received frame type=bar_update` — frames continuous at multi-per-second rate through the whole check window (09:44:54…10:01:25 CT quoted). Account fetch `09:26:21 [INFO] api/accounts: retrieved 3 accounts from TCP server, current="SimAccount1"`. Last-frame age at query time < 2 s.

**1.2 Boot integrity — ✅ PROVEN (NEW state verified).** `08-27 09:25:13 [INFO] 🔐 BOOT INTEGRITY OK — rev 4f2cee032fe6 +dirty · built 2026-08-27T14:23:43Z · expected 4f2cee03 · goldens PASS`. `deploy/RELEASE` = `4f2cee03`; running binary rev = `4f2cee03`; dev tip = `bc95e92b` which is ONLY the RELEASE-marker commit (tree identical to `4f2cee03`). PID 2907898, booted 09:25 CT. **Cutover = merge `4f2cee03` (PR #86) — verified clean.**

**1.3 Bars table — ✅ PROVEN (3 of 4).** dup keys = **0** (fresh GROUP BY); tf census = `1m|7748` only (1m-only invariant holds); row growth last 24h = **2760**; nightly integrity `08-27 09:25:28 ✅ bars integrity OK: dups=0 tfs=1m total=7688`. ⚠️ **UNVERIFIED: "3 random bars vs NT8 chart"** — no NT8 chart access from this worktree; live bar_update frames corroborate freshness but not OHLC equality.

**1.4 Attribution — ✅ PROVEN.** Newest 10 closed positions all carry `plan_id` + correct chain dates (565/564 = `2026-08-27:LONDON` v6/v5; 563/562 = `2026-08-26:LONDON` v9/v1 — the backfill-fixed pair). The 516 empty-plan rows are ALL pre-fix (newest = id 519, `updated_at` 08-16). **Zero UNRESOLVABLE since the fix** (also re-confirmed in 7.7: the UNRESOLVABLE row vanished from the last-7d table).

**1.5 level_stats — ⚠️ SHIPPED-UNPROVEN.** Row count NOW = **28** — but all 28 were written by the standalone backfill run (created 2026-08-27 13:45 UTC = 08:45 CT, pre-cutover), NOT by the bot's nightly writer. Quoted rows: `…|2026-08-25|29243.75|RTH-H|RTH-H|A|react_zone|prior|1|0|1|0|…`, `…|EQL·15m (HTF)|A|liquidity_break|1|0|0|1|…`, `…|OB(bear)·1h (HTF)|B|react_zone|1|1|0|1|…`. **T1 root cause still standing:** deployed `level_stats_wire.go` is the OLD code (the retry/skip-log fix is parked in stash `level-truth-wave-wip-parked-2`); the nightly path still swallows DB-lock errors with a bare `continue`.

**1.6 Clock drift — ✅ PROVEN (Go/RTC side), ⚠️ UNVERIFIED (NT8 leg).** `🕰 clock-health [boot] … rtc_vs_go=0s … NTP=yes NTPSynchronized=yes … ntp_offset=+126.479ms … last_status=OK` + clock-guard timer active. NT8 drift line = `nt8_last_bar=none drift_ms=n/a` at boot; no numeric drift line emitted since (measurement needs bars to flow past the health tick) → drift number vs NT8 UNVERIFIED this run. WSL2 timesync OK.

**1.7 402/credit + retention + units — ✅ PROVEN (2 of 3) + ⚠️.** AI completing: `08-27 10:01:25 📊 AI call complete: completion=126 prompt=10199 finish_reason=stop`; **zero HTTP-402 lines today**. journald = 1.9G of the 2G cap, **BUT retention < 24h** — the `bar_update` INFO-per-frame spam (7.57M lines today) eats the cap, so yesterday's logs are rotated out (a real evidence-loss finding). Units: `nofx.service`, `nofx-web.service` running; `nofx-backup.timer` + `nofx-clock-guard.timer` active.

**1.8 Armed-orders frames — ⚠️ SHIPPED-UNPROVEN (deployed, zero frames yet).** Cutover DEPLOYED: boot line `⚔️ armed_orders=on place_band=100t stale_working=15m (resting limits fill at the authorized price; stale_reeval NOT applied)`. Since boot: **0 order_update frames, 0 ARMED lines, 0 placements** — no plan with `arm{}` specs has been produced yet (NY v1 = no_trade fail-closed; LONDON plans predate the cutover binary). The wire path is code-present and the event machine is exercised by unit tests, but no live frame to quote. Honest status: deployed-unexercised.

---

## SECTION 2 · DETECTORS (THE MAP)

**2.1 Census (last complete session, LONDON 2026-08-27 v7) — ✅ PROVEN.** 8 seated levels: `EQL·15m (HTF)`×1 · `EQH·15m (HTF)`×2 · `OB(bear)·1h (HTF)`×1 · `OB(bull)·1h`×1 · `PDC`×1 · `Supply·1h`×1 · `VWAP−1σ`×1. Note: **nPOC absent again** (see 2.5).

**2.2 Anchor recompute-diff (independent Python, 1m bars) — ✅ PROVEN for 4/6; convention corrected.** Recompute: RTH-H = 29368.75 == KEY-LEVELS `RTH-H 29368.75 A fresh·x9` **EXACT**; RTH-L 29156.25 == `RTH-L 29156.25 A` **EXACT**; PDC == **EXACT** once the anchor convention is understood: the detector consumes the 1h series — `08-26 23:00 1h bar close = 29499.75` == plan `PDC 29499.75` (my initial 1m-prior-day close guess of 29324.0 was the wrong convention, NOT a bug). PDH/ONH follow CME globex session-day windows: plan `PDH 29655.75` = 08-26 19:00 bar high (prior globex day), `ONH 29661.25` = 08-27 04:00 high (current globex day) — consistent session-day semantics. OR/IB recompute produced (29543.25/29477.0 / 29549.75/29477.0) — no OR/IB row seated in v7 to compare → UNVERIFIED for OR/IB only.

**2.3 VWAP divergence — ✅ QUANTIFIED, root cause UNRESOLVED.** Independent recompute (typical-price × bar volume, 02:00–08:30 CT): session VWAP = **29574.02**. Plan seats `VWAP−1σ 29486.95` → implied σ = **87.07 pt** — an abnormally wide band for a ~184pt session range. Hypotheses: (a) `v` field is tick-count not contract volume; (b) σ computed over a different window (prior day? 1h series?); (c) anchor second off the session open. Known FAIL F8 persists; the level-truth wave targets it. **A 87pt σ band rendered as a single magnet row can mislead a fade decision.**

**2.4 ±2σ — ✅ PROVEN ABSENT (expected).** No ±2σ row in any recent plan or prompt; emission code sits in the parked stash (`level-truth-wave-wip-parked`, `KindVWAP2S`). Matches expectation until level-truth lands.

**2.5 nPOC — ⚠️ single-seat invariant un-testable on seats; retire-on-touch ✅ PROVEN.** nPOC seated in **zero of the last 12 plans** (8-seat cap + Tier-1 priorities squeeze it out). `level_state` naked-POC rows: 4 all-time, one per price (PK `level_key`). Retire-on-touch worked on the most recent: `naked-POC||23642` and `||23677` both `times_tested=1 consumed=1 freshness=done` (updated 08-26 23:00 UTC). The invariant "seated exactly once" cannot fire while nPOC never wins a seat.

**2.6 Missed-turns % — ❌ BROKEN-ISH, refreshed (was 43% in verify, now higher).** Independent rerun (strict 5m fractal swing, ±8pt, seated = plan levels): **LONDON 08-27 v7: 16/20 = 80.0%** · **NY 08-26 v8: 18/23 = 78.3%** · **LONDON 08-26 v15: 18/21 = 85.7%**. Method differs from the verify detector (pure k=2 fractal, no min-move), so numbers aren't directly comparable — but the conclusion is identical and worse: **~4 of every 5 5m swing turns have no seated level within 8pt.** This IS the level-truth wave's justification number, refreshed. CAN-MISLEAD-A-TRADE: YES — turns routinely happen off-map.

**2.7 Zone recompute — ⚠️ PARTIAL.** Anchor cross-checks exact (2.2). A full FVG/OB/S&D boundary recomputation would re-implement the detectors — skipped this run; the 1h series (prompt table) matches stored bars exactly (e.g., 08-26 23:00 1h OHLC 29538.50/29551.75/29493.25/29499.75 vs prompt table identical). Boundary-equality for zones: UNVERIFIED this run.

---

## SECTION 3 · GRADE / WEIGHTS — PER TF

**3.1 Weight table AS-DEPLOYED — ✅ DUMPED FRESH from `4f2cee03` code.** zoneEvidenceByKind: OB `.40/.50/.70/.72` · FVG `.35/.45/.65/.65` · iFVG `.35/.45/.65/.65` · S/D `.35/.45/.65/.65` (1m/15m/1h/4h); zoneTFMult `1.0/1.1/1.2/1.3`; reversal ×1.1; zoneFresh `1.0/.6/.3/.15`; anchorFresh `1.0/.8/.6/.5`; typeEvidence: PDH/PDL/PDC/RTH-family `1.0` · ONH/ONL/nPOC `.85` · VWAP/POC `.90` · eVWAP/pdVWAP `.85` · VAH/VAL/SETT `.80` · MID-O `.60` · EQH/EQL/OR/IB family `.70` · Round/Gap `.55` · zones `.30`; thresholds A≥1.0, B≥0.70. **Diff vs owner spec = the SAME known 7 (iFVG .30-flat; volume weights vs 1.0/.95/.95/.70/.70/.55; ±1σ shares VWAP; ±2σ absent; anchor decay vs no-decay; zoneSizeMult; non-zone HTF×1.2) — no NEW drift since the verify report.**

**3.2 Hand-recompute — ❌ FAILED-TO-REPRODUCE (2 concrete cases).** Plan-grade vs machine_grade matches 6/7 on LONDON v7 (the 7th, `OB(bear)·1h (HTF)`, is unstamped). BUT independent arithmetic on machine grades fails: `VWAP−1σ A` = 0.90 base × fresh 1.0 = 0.90 → B, can never reach A under the dumped tables; `PDC B flipped·x11` = 1.0 × consumed-fresh 0.5 = 0.5 → C (displayed B). Exact scorer inputs (freshness string used, HTF multipliers, seat bumps) are not fully reconstructible from DB alone → the grade pipeline is **UNVERIFIED-exact**, with two live rows that contradict the published formula.

**3.3 Floors/caps — ✅ CODE PROVEN + live examples.** Code (levels_score.go:480-506): 1m zones forced C; 15m forced B; 1h C→B floor, A cap; 4h B floor; B2 proximity demotion (pattern above C only within `TIER1_PROXIMITY_TICKS` of a Tier-1 anchor). Live: `Supply·1h [C]` (B2 demotion) ✓ · `OB(bull)·1h C` ✓ · 1m-tier C ✓. ❌ **Candidate violation: `EQL·15m (HTF)` grade A in LONDON v7** — line-level (zone caps don't apply) but table math 0.70×1.2=0.84 → B max; the A is unreproduced (seat-promotion hypothesis).

**3.4 Stamp gap NOW — ✅ PROVEN both populations.** LONDON 08-27 v4-v7: **1/8 unstamped each** (the model-carried HTF rows: `OB(bear)·1h (HTF)`, `iFVG(bull)·1h (HTF)`, `Demand·1h`). NY 08-26 v8 (pre-fix morning): **8/8 unstamped**. T2 fix parked.

**3.5 Reaction-by-grade refreshed — ⚠️ TOO-EARLY.** level_stats 28 rows (08-25 session only): A: touched 21, reacted 14 (**67%**), broke 7, chop 14 · B: 2/2 · C: 0 rows. n=28 single-day → **verdict: underpowered, not yet predictive or inverted** — the forward-validation needs the T1 write path (parked) before any re-weighting.

---

## SECTION 4 · PIPELINE / PLAN

**4.1 End-to-end trace (PDC, LONDON cycle id 33815) — ✅ PROVEN, every hop quoted.** Detector: prior-day 1h-close anchor (08-26 23:00 1h close 29499.75) → scored **B** → seated → system-prompt `KEY LEVELS … 29499.75 PDC B flipped·x11 target_only -102.0` → plan JSON `{"price":29499.75,"label":"PDC","grade":"B","machine_grade":"B"}` → `PLAN STATUS … 29499.75 PDC: dist +102.0 · sweep=F · closes-beyond 6 · acceptance 0/2 · valid · flipped(consumed…)` → scenario citation: `S3 … → 29499.75,29475.09` (v5 plan). The planner-prompt hop itself is NOT persisted (see 4.2).

**4.2 Bias-tree — ⚠️ UNVERIFIED (planner-side not persisted).** Executor-side bias_ctx quoted: `bias_ctx: price 29601.75 · 292.8 vs VWAP 29308.92 · 102.0 vs PDC 29499.75 · above value area (29161.50–29354.80) · nearest magnet VWAP+1σ (-0.8) · nearest liquidity ONH (+59.5)` — real universe PDC anchor ✓. The branch-naming line and P0.2 gap-rule evals live only in the planner prompt (not stored anywhere) → cannot be quoted fresh. Red flag observed instead: the model STILL emits non-schema flip rules — `09:47:12 📐 planner attempt 1/3 parse/schema rejected: flip.rule "2x5m_close" invalid (2x5m|15m_close|5m_close)` (and again 09:53:24, attempt 2/3).

**4.3 Side-quota — ✅ PROVEN.** Zero thin-side ⚖ warnings since deploy (only the boot config line: `⚖️ side-quota (P0-relax…): min_side=cfg(default 2…)`). Zero hard fail-closes on side-quota. Today's NY v1 = `no_trade: ["ENTIRE SESSION — planner fail-closed"]` — a planner-PARSE fail-close (flip.rule grammar), not side-quota.

**4.4 Validator chain — ✅ PROVEN + live rejection quoted.** Chain from code: `validateArmSpecs` → `ValidatePlanDocWithCaps` → `WithFacts` → `WithFactsMachine` (reasoning-first, bias.direction enum, level/scenario caps, quality enums, confirm rules, flip.rule enum, death-condition, no-trade ≤8 lines, P0.1/P0.2, side-quota). Live rejection: the 09:47/09:53 `flip.rule "2x5m_close" invalid` pair — the validator is rejecting properly (and burning attempts).

**4.5 Playbook v2 rendering — ⚠️ PARTIAL.** Executor-side render proven: `role playbook: magnet_meanrevert = … · liquidity_break = … · react_zone = … · target_only = … · pivot = …`, plus `TOUCH:` lines and PLAN STATUS. Planner-side items (bias-tree branch line, ONH truth block, condition×session guidance, STOP-DOING) are not persisted → UNVERIFIED as rendered; only their code presence can be cited.

---

## SECTION 5 · LIFECYCLE / AWAKE

**5.1 Hysteresis — 📅 NOT YET EXERCISED.** Zero flip evaluations, zero deaths today (LONDON v1→v7 were owner-resets / fail-closed retries, no death lines). The wick-through non-death proof requires a flip-eval day — "not yet occurred" is the honest status.

**5.2 Dormant/rearm — 📅 NOT YET OCCURRED.** Zero 😴 dormancies, zero ⚡ rearms today; DORMANT_MIN_HOLD live-honoring un-testable.

**5.3 Dormant-keeps-eyes — 📅 NOT YET OCCURRED.** No dormant window coincided with a level touch.

**5.4 replans_exhausted — ✅ PROVEN ZERO.** grep today's journal: `replans_exhausted` / terminal-no-trade markers = **0**. NY v1 no_trade is the fail-closed writer (not the legacy terminal marker).

**5.5 Wake triggers — ⚠️ PARTIAL.** `decision_records.cycle_trigger` last 24h: only `''`(501) / `stale_dodge`(134) / `post_exit`(2) / `watch`(10) — clock/death-flip/owner/structure_mss/W6-level wakes are NOT labeled in this column (tracking gap). Journal-proven today: owner reset `09:40:37 🗓️ OWNER RESET 2026-08-27 NY — chain abandoned at v1; budget re-armed`. Clock wakes happen every 2m (cadence line) but aren't distinguishable from interval cycles in storage.

**5.6 C6 — ✅ PROVEN.** Zero C6 refusals today (nothing proposed); the dead-plan warning rendered honestly: `⚠ NO ACTIVE PLAN — entries will be refused (C6): plan lifecycle "no_trade" — entries refused` (10:03 CT cycle). Zero parse-error retry loops (no `GateRefusalError` consumption in journal; latest decision rows `risk_check_error=''`, `success=1`).

---

## SECTION 6 · EXECUTION / ENTRY

**6.1 Gate chain order — ✅ PROVEN three-way.** Code order (engine_position.go): R:R (L183) → MinSL (L227) → HTFVeto (L262) → TransitionStanddown (L275) → ExecutorPlanDead (L286) → MinScenarioQuality (L301). Live trace today: plan no_trade → ExecutorPlanDead refused (prompt warning quoted) → no entries. Prompt/trace/code agree.

**6.2 min-SL — ✅ PROVEN.** Zero refusals since deploy (boot line `🛑 min-sl guard: atr_mult=1.0 level_clearance=2tick(s)`; no refusal lines). Zero slipped entries: the only 2 closes today (564/565) were LONDON trades from the pre-cutover morning.

**6.3 Latency routing — ✅ PROVEN.** Executor calls: `⏱️ AI call (reasoning=fast→low) duration: 4.61 / 3.46 / 4.30 seconds` (09:57–10:01 CT) — vs the old 2-6 min tail. **Zero exec calls >120s today.** Planner `plan_reasoning=max` confirmed in boot line.

**6.4 stale_reeval — ✅ PROVEN.** Refusals since latency routing = **0** (journal contains only the armed-orders boot line mentioning the term). Drought-day comparison: the 2-6min drift tail is gone (6.3).

**6.5 Stale-MET / CONFLICT — ✅ (stale) + 📅 (conflict).** Fresh stale-MET quoted (cycle 33815): `S2 confirm: 1x5m close below 29661.25 — MET (stale — written 29661.25 context, price now 29601.75; treat as expired) (…7/1 closes…)`. CONFLICT trailer: **zero in last 24h** (443 all-time) — no long+short simultaneous-MET cycle occurred.

**6.6 plan_mode — ✅ PROVEN.** DB config: `plan_mode: "advisory"`, no per-session overrides → direction gate inert, nothing to honor.

**6.7 Armed path — 📅 AWAITING FIRST ARM.** Zero ARMED lines, zero placements, zero order_update frames, zero stale_reeval-on-armed since cutover (all greps = 0). E1/E2/E3 cannot be freshly demonstrated until a plan carrying `arm{}` goes live. Unit tests cover the event machine; the live E-proofs are pending, not failed.

---

## SECTION 7 · POST-TRADE / SELF-SCORING

**7.1 Watcher — ✅ PROVEN.** Last 3 verdicts (06:02–06:03 CT, pos 564): `accepted_status:"weakening", confidence:42→40, note:"…price is now back above the 5m EMA50 and the rejection follow-through has stalled…", warned:false` (real prices cited: stop 29611.25, high 29608.00). Zero order-touching: `watch_json` contains no order fields; watcher is advisory by design (boot: `watcher[min_conf=70 hold=2 warn_consec=2]`).

**7.2 BE+40 + trail — ⚠️ UNVERIFIED live example.** Config proven (boot: `trailing=2.0×ATR14 arm=after_breakeven`). No BE event reconstructible: `close_reason` has no breakeven category (values: sync 501 / reconcile_flat 60), and the last closed positions were 12–25min losses. Trail state is not logged to any retained store.

**7.3 EOD flat — 📅 AWAITING-DATA.** Yesterday 14:45: the book was flat since 06:38 CT (positions 562/563 both closed in the morning; zero `reconcile_flat` rows in the 14:45 window) → the flat fired on an empty book; the limit-then-market ladder is unquotable (and yesterday's journal has rotated out — see 1.7). Config proven: `eod_flat=session-end (NY 14:45 CT, R-A15)`.

**7.4 pnl_corrected — ❌ BROKEN (NOT YET TRUE).** `pnl_corrected` is **NULL on every closed row**; `realized_pnl` is correct (independent fills recompute on newest 5: deltas = $0.00 each — e.g., pos 565 SHORT 29580→29611.25, exit-fill realized −62.50 == position). The correction writer = **open PR #60** (`hotfix/pnl-record-integrity`, "P0: wrong PnL recorded — additive corrections (37 rows)"), unmerged/undeployed. The owner's "all PnL = pnl_corrected" rule is in-flight, not live. CAN-MISLEAD: dashboards still show realized-only.

**7.5 touch_episodes — ✅ PROVEN.** 90 episodes written in 24h, latest 10:03 CT: `OR-H 29589.5 · 5 bars · penetration 13.75/13.75/0.0 · reject/reject · approach 3.29 ATR · vol_ratio 0.92 · close-side rejection` — all fields sane, no "through 83pt" class.

**7.6 Consumed-no-touch — ❌ GROWN (was 8, now 22).** `consumed=1 AND times_tested=0` scoped to last 2 days = **22** (verify reported 8) — T4 (MarkConsumed records the consuming touch) is parked; the skew keeps growing.

**7.7 Expectancy tables — ✅ RUNNABLE, last 7 days (exit_time-windowed, 34 closes):** `reject·short n=17 Σ−328.0 (4 wins)` · `breakout_retest·long 4 Σ−65.5` · `acceptance·short 3 Σ+222.5` · `no_scenario 3 Σ−202.5` · `breakout_retest·short 2 Σ−164.5` · `reclaim·long 2 Σ−181.0` · `sweep_reclaim·long 2 Σ−130.5` · `hold·long 1 Σ+168.0`. **UNRESOLVABLE row: zero** (attribution fix holding). 7-day Σ = **−681.5** across 34.

---

## SECTION 8 · META / PROCESS

**8.1 One-agent/worktree state — ✅.** Main tree = `dev @ bc95e92b` (cutover complete, clean except untracked `.env.bak.0825-2157` / `nofx-bin.old*`). Worktrees: main, `nofx-recheck` (this dispatch), `/tmp/nofx-dev-check` (stale from the level-verify dispatch). Stash index on the repo (5, re-organized post-cutover): `{0} leveltruth-parked: no-trade stamp + StampMachineGrades/CarryMachineGrades (apply after cutover on fix/level-truth)` · `{1} level-truth-wave-wip-parked-4` · `{2} …-parked-3` · `{3} …-parked-2` · `{4} …-parked`. `feat/armed-orders` merged (30159b2b inside dev history).

**8.2 Docs truth — ⚠️.** Guide `asBuiltRev` grep found nothing in `GuidePage.tsx` this run (unverified placement); README-VL-SYSTEM + guide were last updated at Wave-1; PIPELINE-MAP staleness (per verify) unchanged. No doc write this dispatch.

**8.3 Open PRs — ✅ census.** 6 open, **all MERGEABLE**: #64 regime-wave (→main) · #63 docs research-import · #62 brand census · #61 partner-sync docs · #60 pnl-record-integrity (P0 hotfix) · #59 reset-dialog-stuck.

**8.4 Partner — ✅.** `vlautoagenttraderv1` PR #2: **open, mergeable=true**, head `sync/vl-state-2026-08-26` (owner/Binnie still to merge).

**8.5 F/T register roll-up (fresh status each):**
- **T1 level_stats 0-rows** — STILL OPEN: 28 rows exist only from the standalone run; nightly writer unpatched (parked `…parked-2`). Grown-worse: no.
- **T2 stamp gap** — STILL OPEN, quantified 1/8-per-plan on LONDON v4-v7 (parked `leveltruth-parked`).
- **T4 consumed-no-touch** — GROWN-WORSE: 8 → 22 rows (2d).
- **T5 touch telemetry** — RESOLVED (shipped, 7.5 green).
- **F4 1h-zone-stamped-C** — RE-OBSERVED (Supply·1h C via B2 demotion; consistent with code).
- **F5 stamp gap** — same as T2.
- **F6 times_tested=0** — subsumed by T4.
- **F8 VWAP divergence** — STILL OPEN, now quantified (implied σ=87pt).
- **pnl_corrected NULL** — NEW-REGISTER: open PR #60.

---

## VERDICT PAGE

### Owner-board diffs (most valuable output — where this run disagrees)
1. **"All PnL = pnl_corrected"** → ❌ not deployed; NULL everywhere; PR #60 open.
2. **"level_stats populated"** → ⚠️ 28 rows are standalone-run rows; the nightly writer still can't land (T1 unfixed on the deployed binary).
3. **"Missed-turns ~43%"** → refreshed independently at **78–86%** (stricter detector; conclusion direction identical).
4. **"Armed orders live"** → deployed but **zero frames**; E1-E3 cannot be marked green until the first armed plan (the cutover itself is ✅).
5. **"PDC detector verified"** → re-verified with the anchor convention nailed (1h-series prior-day close) — the earlier "PDC 175pt wrong" scare was a convention error in MY recompute, not a bug.

### Top 5 risks by can-mislead-a-trade-today
1. **pnl_corrected NULL everywhere** — every PnL number on the card is realized-only while the owner assumes corrected. (PR #60 unmerged.)
2. **VWAP−1σ seated at implied σ≈87pt with an unreproducible A grade** — a magnet row ~87pt wide is a near-meaningless fade reference.
3. **EQL·15m (HTF) grade A unreproducible** from the published weight tables — an over-trusted seat.
4. **78–86% of 5m swing turns unseated** — most turns happen off-map.
5. **flip.rule grammar still burning planner attempts** (2 today) — NY fail-closed risk remains live.

### What the next 2 scheduled events will and will NOT resolve
- **Armed-orders cutover (DONE, verified)** — WILL resolve: resting-order execution + E1-E3 proofs once an arm fires. Will NOT: any of the top-5 risks above.
- **Level-truth wave** — WILL resolve: T1/T2/T4, ±2σ emission, swing seats (2.6), VWAP anchor questions. Will NOT: pnl_corrected (PR #60), flip.rule grammar, journald retention starvation.

### One-line system verdict
**Cutover clean and the execution path is re-architected, but the system's truth-labels (PnL corrections, grade arithmetic, level_stats telemetry) are still ~half-deployed — the machine runs, the scoreboard doesn't.**
