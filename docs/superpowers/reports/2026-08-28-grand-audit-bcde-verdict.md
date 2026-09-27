# Grand Audit — Parts B–E + Verdict Page (2026-08-28, 08:36–09:15 CT)

- **Audited rev:** `2738d158` (PID 3441452) · dev tip `e44a66a8` (docs on top) · read-only worktree `~/nofx-grand` @ branch `docs/grand-audit`.
- **Evidence rules:** R1 fresh CT-stamped quotes from journal/DB; bars replay on stored 1m MNQ (`bars` table, 2026-08-19→now, dup-free per Part A); AMBIGUOUS same-bar resolved AGAINST the trade; Σ$ at MNQ $2/pt; EOD = next 14:45 CT flat.
- **Window:** 2026-08-26 00:00 CT → 08-28 09:09 CT. Journald retains only 08-27 13:35 CT → (Aug-26 logs gone — noted per item).
- **Scripts (committed):** `scripts/grand_audit_b.py`, `scripts/grand_audit_c.py`, `scripts/grand_audit_probes.py`.

---

# S-FINDINGS (new this dispatch)

| # | Severity | Finding | Evidence |
|---|----------|---------|----------|
| S1 | **A** | **BE+40 → move_stop wire FAILED live on #566.** `auto-breakeven` fired 4× while LONG 29611.50 was OPEN (11:45:49, 11:46:49, 11:59:50, 12:00:50 CT) and every send failed `ninjatrader/tcp: no open entry to move the stop`. Root: `MoveStopToBreakeven` requires `lastEntrySignalID` (`trader/ninjatrader/tcp_trader.go:485-490`) which is never set for entries materialized WITHOUT a Go-side signal (reconcile/manual path; #566 is `plan_version=0`). The same `move_stop` path serves trailing (`trader/auto_trader_trailing.go`) → trail is equally inert for that entry class. $0 realized damage (#566 closed +97 anyway), but the BE/trail mechanism has a dead cell. | log_events ×4 quoted; code cite; `trader_positions` 566 |
| S2 | **B** | **Guide §7 vs live knob drift + units mislabel.** Live `proximity_filter_atr=1.5` (±458pt band) while Guide §7 `settings.ts` documents the retune: `systemDefault: '0.3 (retuned from 1.5 → ±~92-100pt on MNQ)'` + `recommended: ⭐ 0.3`. Also the Guide label says `×ATR` but the code multiplies by `dATR` = DailyRangeProxy (`kernel/levels_score.go:414`); numbers in the Guide match code, the unit NAME doesn't. C6 sweep: pool 9.3 levels/plan at 1.5× vs 4.5–6.4 at 0.2–0.4×. | DB dump vs `web/src/guide/content/settings.ts` |
| S3 | **B** | **The discipline's honest-wait is the largest live money leak.** 35 fresh-MET AI-declines replayed: **26 would-have-won, Σ = +$1,974.5** (was +$1,763 at the Aug-27 window — the leak GREW). Post-arm-mandate declines still 6 with 5 would-have-won (+$235.5). The gates aren't the leak; the waiting is. | B engine table |
| S4 | **B** | **1h HTF veto cost $352, saved $0 (window).** 3 vetoed arms would have won (+$65/+$110/+$177); 4h was RANGING at all 7 veto timestamps → a 4h-cross-check would have allowed all 7 (hypothetical net +$833 incl. two losers). | C3 table |
| S5 | **C** | **Aug-26 journal gap.** The order_update flood wiped Aug-26 from journald (earliest 08-27 13:35 CT). Aug-26 gate evidence is DB-only; the drought-era 3× C6 refusals (08-26 13:52/13:57/14:00 CT) are journal-evidenced ONLY in the prior report — un-replayable fresh. If the flood returns, forensics loses a day again. | `journalctl -u nofx | head` |

---

# PART A — BROKEN-ON-RECHECK table

| Item | Verdict |
|------|---------|
| All 22 Part-A probes (A1–A22) | **Re-verified this dispatch — zero BROKEN, zero SHIPPED-UNPROVEN.** A13 dup=0 re-confirmed; arbiter still 0 mismatches; #569 added a second clean lineage stamp. |
| S1/S2/S3 (Part A) | Still standing: band code=spec (dispatch prose stale) · refusal-dedup granularity gap · dup probe artifact. |

---

# PART B — money verdicts per gate

**Census:** 160 raw arm-refusal log rows → 11 distinct arm-gate events + 7 decision-gate events + 35 distinct fresh-MET declines (87 raw rows, 42 cycles with fresh-MET).

| Gate | n | would-won | would-lost | ambig | Σ$ | Verdict |
|------|---|-----------|------------|-------|------|---------|
| arm_RR (floor 2.0–3.0) | 2 | 2 | 0 | 0 | +150.9 | TOO-FEW (n<5; would-be COSTING) |
| arm_minSL (1.0×ATR5m) | 5 | 3 | 2 | 0 | +241.9 | **COSTING** |
| arm_HTFveto (1h) | 4 | 3 | 1 | 0 | +196.5 | TOO-FEW (n<5; would-be COSTING) |
| stale_reeval_refused | 5 | 0 | 5 | 0 | −247.5 | **SAVING** (matches prior $247) |
| dead_man_watchdog | 1 | 0 | 1 | 0 | −23.0 | TOO-FEW |
| session_gate (first-5m) | 1 | 0 | 1 | 0 | −14.5 | TOO-FEW |
| strict / C6 (in-window) | 0 | — | — | — | — | n=0 in window (last C6 rows 08-24/25, 6 in log_events; 3× on 08-26 journal-only) |
| **AI declines (fresh-MET)** | **35** | **26** | **9** | **0** | **+1,974.5** | **COSTING** |

Per-decline-class: reject 10 (8 won, +$1,144.1 COSTING) · sweep_reclaim 15 (12 won, +$634.5 COSTING) · breakout_retest 5 (+$98.4) · hold 3 (+$99.5) · acceptance 2 (−$1.9).

**B2 armed-era split** (mandate 08-27 09:25 CT): declines PRE 29 (21 won, +$1,739.0) vs POST 6 (5 won, +$235.5). Real arms placed: PRE 0, POST 5 (armed_orders non-TEST). Post-mandate **declines:arms = 1.2:1** (was ∞). The mandate is working — authoring shifted from silent waits to placed arms — but post-mandate declines still leak.
**$1,763 gauge re-measured: +$1,974.5** (wider window, 35 vs 30 events).

**B3 per-condition expectancy** (TRUE `pnl_corrected`, closes since 08-26; small n): breakout_retest LONDON n=2 −$148.0 (0%) · reject LONDON n=2 −$131.5 · reject ASIA n=1 −$42.0 · reject NY n=1 +$63.0 · UNRESOLVABLE n=2 +$65.0. **S3-class re-test: n=10, 8 won, 2 lost, +$865.8** (was 7/7). reject-NY stays thin (n=1 actual; 8/10 replay wins across sessions).

**B4 R-floor sweep per play** (candidate set = refusals + declines, 53 events):

| play | all | R≥1.5 | R≥2.0 | R≥2.5 | R≥3.0 |
|------|-----|-------|-------|-------|-------|
| reject | n=14 +$1,465 | n=12 +$1,408 | n=11 +$1,431 | n=7 +$926 | n=6 +$798 |
| sweep_reclaim | n=19 +$1,043 | n=9 +$651 | n=9 +$651 | n=7 +$547 | n=6 +$518 |
| breakout_retest | n=8 −$42 | n=7 −$54 | n=7 −$54 | n=7 −$54 | n=6 −$14 |
| hold | n=3 +$100 | n=1 +$158 | n=1 +$158 | n=1 +$158 | n=1 +$158 |
| acceptance | n=2 −$2 | n=1 −$17 | n=1 −$17 | n=1 −$17 | n=1 −$17 |
| (gate refusals) | n=7 −$285 | n=6 −$120 | n=6 −$120 | n=6 −$120 | n=6 −$120 |

**Draft per-play floor ruling (report-only):** reject ≥3.0 keeps +$798 of +$1,465 · sweep_reclaim ≥3.0 keeps +$518 of +$1,043 · breakout_retest is negative at EVERY floor → exclude from arm authoring · hold positive but n tiny.

---

# PART C — calibration replays (report-only, zero config changes)

- **C1 swing-k × TF** (vs each session's actual seats, ±8pt): k=2→3 cuts swing count 72→46 (5m) and 29→18 (15m) with missed-turns ~93% unchanged → **k=3 halves churn for free** if swing seats return; 15m mix no meaningful change. (Absolute miss% differs from leveltruth's 78–86% because windowing/detector differ; deltas are the deliverable.)
- **C2 FVG floor:** 1.25×ATR = 8 candidates in 5 days, 1.5× = 6 (per-day 2/1, 1/0, 3/3, 2/2, 0/0). Supply exists; **zero fvg_entry scenarios authored live** → the zero-authored problem is demand (model), not the floor. Lowering to 1.25 adds ~2 candidates/5d — not $ material.
- **C3 HTF-veto TF:** 7 vetoed arms replayed under 1h vs 4h. 4h was RANGING at all 7 → 4h would allow all: hypothetical net +$833 (incl. −$38, −$50 losers) vs 1h's 3 blocks that would have won +$352. **1h veto: saved $0, cost $352 (n=3, TOO-FEW).**
- **C4 min-conf:** 60–64 band n=3, Σ = **−$217.0** (0 wins) vs 65+ n=1 −$62.5 vs <60/unknown n=4 +$86.0. Direction matches phase-6 (−$142.5/5 on 08-24); n=3 → TOO-FEW but consistent.
- **C5 trail mult 2.0/2.5/3.0:** zero trail-exits in window (no trade held into trail range; #566 exited before the trail could bite). **INCONCLUSIVE** — needs a longer hold sample; note mfe=0.0 recorded on #566 (planless closes skip MFE → give-back stats undercount).
- **C6 proximity sweep** (66 plan reads, 34 entry levels): mean in-band pool 4.5 (0.2×) / 5.6 (0.3×) / 6.4 (0.4×) / **9.3 (1.5× live)**; entries excluded 0.2/0.2/0.1/0.0 per plan. Live 1.5× seats ~3.7 extra levels/plan beyond 0.3× — the dilution S1 flagged.

---

# PART D — event-wait sweep

- **D1 LONDON S4:** armed row `29574.00/29592.00/29490.88` was **cancelled 07:36:16 CT, state_reason "no active plan"** (armed_orders row 7) — it was NOT working at 08:30; LONDON S2 also cancelled 08:17:25 ("cancelled in NT8"). **No working arm at the session boundary → FIX-1's cancel-first wire E-proof remains PENDING** (next chances: tonight's session ends / a working arm at 14:45).
- **D2 BE+40 → modify_bracket:** **NO** — BE fired live but `move_stop` send FAILED 4× on #566 (S1 finding). `ModifyBracket` frames: 0 ever (no live instance). Wire path for non-signal positions is inert.
- **D3 first strict ClassifyCitation:** **YES (first strict-era entry graded)** — pos #569 (NY, strict plan v1) carries `adherence_grade=A, cited_scenario_id=S1, plan_band=armed_fill`. Wiring: `kernel/plan_render.go:311` called at `trader/auto_trader_planner.go:1957` + `auto_trader_planconfig.go:217`.
- **D4 tonight 17:00–17:10 CT:** PENDING at report time (audit ends ~09:15 CT). Watch items unchanged: nightly solo #2 (08-27), reopen peak_depth<4096 + closes_dropped=0, next planner cap=65536, FIX-1 E-proof.
- **D5 new chains since #567:** **YES — #569 complete armed chain**: `⚔️ armed NY S1 short limit 29700.00 SL…` 08:35:27 → TCP `frame type=fill` 08:43:12 → `🧩 reconcile: armed-fill lineage stamped — pos 569` 08:44:28 → `📊 exit fill recorded: MNQ SHORT qty=1.00 @ 29668.50` 08:45:28 → +$63.00 (pnl_corrected), plan v1 / S1 / armed_fill / grade A. Second-ever armed fill, first strict-era one, closed in ~60s. Also `📡 armed order_update summary (1-line/min): frames=12…` live — the DEBUG-sampling fix is visibly working.

---

# PART E — hostile sweep

- **E1 mode-interaction cells (strict × dormant × working-arm):** ALL THREE have live instances today — strict = DB config `plan_mode=strict` on all 3 sessions (NY/ASIA/LONDON); dormant = plan death → LONDON S4 arm cancelled "no active plan" 07:36:16; working-arm = LONDON S1 armed 07:30:15 + NY S1 filled 08:35:27. Boot lifecycle line: `lifecycle: hysteresis=buffer0.5×ATR14 confirm=2close(s) · flip/death→dormant+auto-rearm`.
- **E2 query-class sweep:** diffs of the last 4 wave commits (`2738d158`, `0085a2b7`, `67d2d10e`, `4f2cee03`) contain **zero** new non-binding strategy reads or trader-less position queries. Clean.
- **E3 Studio knob dump vs Guide §7:** deviations — (1) proximity **1.5 live vs ⭐0.3 Guide** (S2) · (2) max_levels **12 live vs ⭐8** · (3) min_confidence **60 live** (Guide/research 65) · (4) min_side_levels **4 live** vs DefaultSideQuota 2 · (5) session map text in Guide (`max_trades 3 · advisory · re-plans 2`) is **stale** vs live rows (`strict ×3 · 7/10/10 · replan 4`) · (6) Guide "×ATR" unit mislabel (S2). Everything else wired per the 35-card census.
- **E4 process:** deployed `2738d158` IS ancestor of origin/dev (`e44a66a8` cutover record + `052ec3b5` marker on top — docs only). Main tree: 0 modified tracked files; untracked = `.env.bak*`, `RELEASE.prev.slist`, `nofx-bin.old*` (operational artifacts). Stashes: 0. Worktrees: 8 (incl. 4 completed audit branches). Open PRs: 11 stale (46, 51-54, 56, 60-64) — none blocking. **Partner drift** (`vlautoagenttraderv1` @ `f6ae7597`): **2/3 C# files differ** (VLTraderTCPClient.cs, VLBarsSubscriptionManager.cs; VLContractResolver same) + 18 kernel + 5 trader/ninjatrader Go files — needs the next format-patch sync.
- **E5 hostile hunt (candidates, ranked):**
  1. BE/trail dead cell for non-signal materialized positions (S1 — proven live).
  2. Arm price drift across plan versions: journal 07:30 armed `LONDON S1 limit 29662.13` vs armed_orders row 6 entry `29702.00` (older version's prices persisting) — stale-arm placement risk (speculative severity).
  3. Honest-wait leak (+$1,974/35) — discipline cost, not a bug (S3).
  4. armed_orders carries 4 TEST-E2 rows in the production table — ledger hygiene.
  5. mfe=0 on planless closes (#566) — MFE/MAE discipline stats undercount.
  6. Proximity 1.5× pool dilution (C6) + Aug-26 journal gap (S5).

---

# VERDICT PAGE

**Top-5 risks:** (1) honest-wait leak +$1,974 (35 declines) · (2) BE/trail dead cell for non-signal entries · (3) arm price/version drift · (4) journald gap recurrences · (5) conf<65 band still taking entries (−$217/3).

**Three plain answers:**
1. **Every fix still standing?** YES — all 22 Part-A probes re-verified this dispatch, zero BROKEN-ON-RECHECK; lineage stamps live (#567, #569); planner cap 65536 with 0 truncations; dedup active.
2. **Discipline SAVING or COSTING now?** **COSTING overall** — the honest-wait leaks (+$1,974 would-have) far exceed what the gates save (−$247 stale_reeval, −$285 gate-refused). The gates themselves are net fine; the *waiting* is the cost.
3. **What should the owner change, with numbers?** (a) proximity **1.5→0.3** (pool 9.3→5.6/plan, matches Guide §7 and S1 arbitration); (b) min-confidence **60→65** (avoids −$217/3, consistent with prior −$142.5/5); (c) **exclude breakout_retest from arm authoring** (negative at every R floor) and keep reject/sweep_reclaim at R≥3.0 (+$798/+$518 retained); (d) decide the HTF-veto question with the +$352 would-have number on the table (consider a 1h+4h cross-check); (e) fix BE/trail `move_stop` for non-signal positions before relying on either in a live handoff.

**One-line system verdict:** every fix stands and the ledger is clean, but the discipline's own patience is the system's largest live leak — tighten the band and the confidence floor, not the gates.
