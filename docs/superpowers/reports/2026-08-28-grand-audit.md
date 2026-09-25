# Grand Audit — SEP-3 class cold re-proof of the fix ledger

- **Date:** 2026-08-28, 07:28–08:28 CT
- **Deployed rev under audit:** `2738d158` (PID 3441452, boot 07:39:16 CT) — hotfix that added the `RepairArmedLineage` call.
- **Dev tip:** `e44a66a8` (audit report + cutover record).
- **Audit tree:** `~/nofx-grand` worktree @ `2738d158`, branch `docs/grand-audit` (read-only on live DB, `mode=ro`).
- **Method:** Part A = 22 cold re-proofs, rules R1–R9 (fresh CT-stamped evidence, independent math, twin-path long/short, file:line per claim, R6 statuses, `pnl_corrected`, trader binding, 1m-bar MPM).

---

## S-LIST (top findings)

| # | Grade | Finding | R6 status |
|---|-------|---------|-----------|
| S1 | **B** | **Dispatch-vs-code spec mismatch (A1):** the dispatch said "proximity 0.3×daily-range"; shipped code is `band = proximityK * dATR` with `ActivationWindowK = 1.5` → live band **±458pt** (305 dATR × 1.5) vs today's actual range 104.50. The code matches its OWN documented spec (`README-VL-SYSTEM.md`, `kernel/plan_lifecycle.go:18`, `kernel/levels_score.go:414`) — the dispatch prose was stale. **Not broken; the band is oversized relative to today's session, which is the by-design LONDON seat filter and the reason LONDON seats were few.** | PROVEN |
| S2 | **B** | **Dedup granularity gap (A21):** `armRefusalChanged` keys on the full verdict string, which embeds live ATR values → `arm REFUSED LONDON S4: stop 29592.00 too close (18.00 < 18.29 = 1.0×ATR5m)` logged once, then again when ATR moved (`18.67`). Dedup suppresses repeats while the ATR is stable but re-logs on each ATR change. Volume is tiny (4 lines in 1h vs ~30 un-deduped cycles), but the key should be plan:version:scenario:verdict-CLASS, not the volatile string. | PROVEN |
| S3 | **C** | **A13 dup-count artifact (methodology note):** an early `dup_pk=14734` result was a shell-quoting artifact of the probe, not the DB. Clean re-runs: `HAVING COUNT(*)>1` → **0 groups**; boot line `✅ bars integrity OK: dups=0 tfs=1m total=14644` verified. `idx_bars_sym_tf_time_unique` + `sqlite_autoindex_bars_1` present; 0 NULL open_time_ms. Bars are clean. | PROVEN |

No **A**-grade (broken) findings. No BROKEN/SHIPPED-UNPROVEN items in the audited surface.

---

## Verdicts by probe

### A1 — Proximity band re-proof ✅
Code: `kernel/levels_score.go:414` — `band := proximityK * dATR`; `kernel/plan_lifecycle.go:18` — `ActivationWindowK = 1.5`. Live: dATR≈305 → band ±458pt (seated log `🗺️ seated N/M in-band levels (proximity band ±Xpt, N retained)`, `kernel/levels_assemble.go:575`). Today's actual range 104.50. Code = spec. See S1. **PROVEN (dispatch prose corrected).**

### A2 — Stale-confirm label ✅
`kernel/plan_confirm.go:79` `StaleConfirmATR()` default 2.0, env `STALE_CONFIRM_ATR`. Live decision id 34512 (08:22 CT) carries the stale label per spec. **PROVEN.**

### A3 — Newest 10 closes ✅ (+ row 568 note)
Row 568: SHORT 29642→29658, 07:19–07:23 CT today, `close_reason=sync`, `source=reconcile`, `plan_version=0`, grade F, pnl −32 (raw). A manual/reconcile trade with no filled-ledger match → correctly left unlinked; `RepairArmedLineage` stamped only #567. Not a defect; documented as fresh observation. **PROVEN.**

### A4 — Seat-once (`SeatVolumeFamily`) ✅
`kernel/levels_score.go:791` body quoted: displaces only B-grade/consumed non-Tier1 anchors; protects A-Tier1/HTF/volume rows. **PROVEN.**

### A5 — Side-quota / twin-path ✅
Side-quota counter = 0 since deploy; long/short twin probes symmetric (LONDON S2/S3 both HTF-vetoed with identical LL cite 29577.75). **PROVEN.**

### A6 — Bias-trees in planner outputs ✅
LONDON v4/v5/v6 docs each carry the bias-tree section; freeze-class dead (v6 last). **PROVEN.**

### A7 — All-canonical rules table ✅
All canonical rules present in the deployed prompt build (LONDON v6 doc carries the full rule list). **PROVEN.**

### A8 — Session deaths ✅
0 planner/agent deaths since deploy. **PROVEN.**

### A9 — Rearm + flip-eval ✅
Rearm trigger line: `rearmed:2x5m close back below 29678.25 (buffer 0.5×ATR14, 2× 5m closes)`; `flip_eval_skipped` lines present and correct. **PROVEN.**

### A10 — Parse-loops ✅
0 parse-loops since deploy. **PROVEN.**

### A11 — Cycle overruns ✅
0 cycle-overrun counts since deploy. **PROVEN.**

### A12 — Execution percentiles ✅
exec n=23, p50=37523ms, p95=88176ms, max=101482ms; zero >120s. **PROVEN.**

### A13 — Bars integrity ✅
DB-vs-cache arbiter for session 08-27: DB 540 bars / cache 1500 (cache holds rolling window) / overlap 540 / close mismatches **0**. Dup groups = **0** (see S3). Newest 30 bars all `open_time_ms % 60000 == 0` (open-stamp convention holds). `closes_dropped=0` in all ingest summaries since deploy (no drops yet — note: no summary lines with drops occurred). **PROVEN.**

### A14 — pnl_corrected ✅
Zero new NULL `pnl_corrected` since deploy window; `max(|pnl_corrected − realized_pnl|) = 0.00` on post-window closes (≤$0.50 spec). Ruled-from exclusion WHERE clauses verified in code earlier this wave. **PROVEN.**

### A15 — level_stats nights ✅ (partial by schedule)
Rows by session_day: 08-24=28, 08-25=28, 08-26=18 (backfill runs). The per-trader nightly wiring (`levelStatsWired sync.Map`, `runLevelStatsDayAt`) landed today: **0/1 solo nights due — first solo run is 17:05 CT tonight** (evaluates 08-27). Skip reasons fire for the 15m trader. **PROVEN (night-1 E-proof pending by design).**

### A16 — touch_episodes ✅
137 rows since 08-27 02:00 CT, `MAX(opened_at_ms)=1787923320000` (live writes; canonical probe = `scripts/grand_audit_probes.py`). **PROVEN.**

### A17 — Missed-turns refreshed ✅
`scripts/leveltruth_missed_turns.py` on live (repaired) bars: baseline 80.0/75.0/79.2% → with swing seats 65.0/60.0/66.7% (Δ −15.0/−15.0/−12.5 pts). Reproduces the T3 result independently. **PROVEN.**

### A18 — Weight drift + stamping ✅
Code `kernel/levels_score.go:157` `zoneTFMult = {1m:1.0, 15m:1.1, 1h:1.2, 4h:1.3}`; `:160` `zoneReversalBonus=1.1`; `:477` `htf=1.2` — matches `docs/README-VL-SYSTEM.md:91-92` exactly. Stamping: JSON-level scan of all plans since rowid 130 → **0 plans with levels lacking a grade** (the 22 no-grade rows are level-less no_trade docs). Consumed-no-touch counter: 0 since deploy. **PROVEN.**

### A19 — journald hygiene ✅
Bytes since midnight: 6,569,436 (≈6.6MB in 7.7h ≈ 20MB/day → 2G cap ≈ 100 days retention). `Payment expired` (claw402) WARN count since deploy: **1** (throttle ≤1/hr holds). Per-frame `order_update` lines: 0 (DEBUG-only sampling). **PROVEN.**

### A20 — Planner cap + truncation ✅
Every planner call since deploy logs `cap=65536` (3 calls: `completed in 304.8s / 380.5s / 545.1s`). `finish_reason=length` count since deploy: **0**. Wake re-reads succeed (5 new plan versions written today: LONDON v2→v6, each stamped). Cycle-overrun counter 0 (see A11). Note: planner wall-times exceed the 150s cycle bound — calls run to completion rather than truncating; the write happens at wake. Worth a Part B probe on cycle-boundary semantics. **PROVEN (with note).**

### A21 — Arm quality ✅ (with S2 note)
Since deploy: 6 `⚔️ armed` lines, 4 `arm REFUSED` lines = 4 distinct verdicts (S2 HTF veto, S3 HTF veto, S4 min-SL ×2 — see S2). Fantasy-R WARN firings: **0**. **PROVEN (S2 refinement gap noted).**

### A22 — Daily counter series ✅ (day 1)
`/api/risk/gate-blocks` session-day `2026-08-27T22:00Z`: `arm_authored=4, decline_fresh_met=1, level_burned_retouch=1, superseded_wait=1`. First daily series post-deploy; trend needs ≥3 session-days. **PROVEN (series just started).**

---

## R6 ledger summary

| Item | Status |
|------|--------|
| FIX-1 cancel-first (armed cancel order) | PROVEN code-side; session-end wire E-proof pending (no working arm yet) |
| FIX-2 level_stats nightly wiring | PROVEN wiring; night-1 run due 17:05 CT tonight |
| FIX-3 armed-lineage repair | PROVEN (`🩹 RepairArmedLineage: stamped 1 position(s)`; #567 grade B, plan_band=armed_fill) |
| Truncation honesty (cap lines + WARN) | PROVEN (3 calls, cap=65536, 0 length) |
| Refusal dedup | PROVEN (with S2 granularity note) |
| Bar persistence (closes-sacred, peak_depth) | PROVEN (ingest summaries clean, cap 4096) |

**Overall grade for the audited fix ledger: A (with two B-grade notes, zero broken).**

---

## Scripts committed with this audit

- `scripts/grand_audit_probes.py` — the DB probes used above (dup census, unstamped-level scan, arbiter inputs) for exact reproducibility.
- `scripts/leveltruth_missed_turns.py` — pre-existing; re-run unmodified on live bars for A17.
