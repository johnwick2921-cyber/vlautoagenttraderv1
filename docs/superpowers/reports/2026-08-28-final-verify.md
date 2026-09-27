# FINAL VERIFICATION SWEEP v2 — POST-BAR-TRUTH RE-AUDIT

**2026-08-27, 17:00-17:58 CT · READ-ONLY (zero code/config/DB/env changes, no
restarts) · isolated worktree `/home/hoang/nofx-final` @ running rev
`405e1323b176` · branch `docs/final-verify`.** All times CT. R1 fresh evidence
only (every query/journal line produced this run) · R2 independent math (Python
recompute from the raw `bars` table, never the engine) · R3 twin paths ·
R4 file:line · R6 never upgraded a grade.

---

## VERDICT PAGE

### S-FINDINGS (this run)

- **S-1 — Journald retention is ~3.6 HOURS, not ≥7 days.** Oldest surviving
  nofx entry: 13:35:21 CT today (measured 17:10 CT); disk at **1.9G of the 2G
  cap** (`/etc/systemd/journald.conf.d/nofx.conf` `SystemMaxUse=2G`). The
  bar-truth wave's E4 ("71.8 days") is wrong by ~500×. Root cause: a **1.48GB
  single-hour flood** at 13:35-14:35 = `trader/auto_trader.go:43` logging every
  `order_update` frame at INFO (~25k lines in one SECOND at 13:38:17). This is
  a THIRD per-frame log flood (after the bar_update INFO flood and the
  backpressure WARN flood) — the class is not dead.
- **S-2 — "Drop counters at zero" is false and the summary line is malformed.**
  Fresh diffs show `ingest_oldest` = 855 (17:10) → 1398 (17:17), driven by the
  17:00-17:05 CT Globex-reopen flood (wire_liveness `frames_per_min=30456` at
  17:03:36, ~507 frames/s). The 1-line/min summary itself renders as
  `dropped_oldest=%d … !BADKEY=1 !BADKEY=0 !BADKEY=0` — `ingestDropSummary`
  (`provider/ninjatrader/bar_persist.go:181`) passes positional args to the
  structured `warn(msg, kv...)` logger; `barPersistSummary` (:101, `Warnf`)
  is correct. The metrics are unreadable exactly when they matter.
  **Mitigating truth: zero bar CLOSES were lost** — `ingest_current=0`,
  `persist_queue=0`, and the DB is contiguous 17:00→17:08 for MNQ and ES (the
  drop-oldest path always enqueues the freshest frame; the closed tail
  self-heals from the cache).
- **S-3 — Ingest channel cap 256 has ZERO headroom at session reopen.** 1399+
  drop-oldest evictions in ~4 minutes (per-minute 1 → 235 → 856 → 1399). Peak
  frame rate 507/s vs a 256-slot channel. Today cost nothing; the margin is
  structurally thin (`provider/ninjatrader/tcp_server.go:390`).
- **S-4 — `fix/bar-truth` (the deployed wave, rev 405e1323) is NOT merged into
  dev.** `git branch --contains 405e1323` = fix/bar-truth only; dev tip
  43bb60cb lacks it. The next dev deploy silently reverts the bar-truth wave.

### Per-part one-liners

| part | one-liner |
|---|---|
| P1 | Data truth holds (mismatches 0 ×3 fresh runs, ATR ×3 cuts 2dp, stamps proven by fill containment); drop-zero + retention claims BROKEN; queue cap undersized. |
| P2 | 13/24 re-proven clean · 6 broken/partial (see table) · 5 unverifiable-from-logs/awaited |
| P3 | All regression suites green at running rev (27 Go pkgs, goldens PASS, tsc clean, vitest 277/33); deadline-fix class dead everywhere; cache↔DB stayed converged +50 min into live operation. |
| P4 | STRICT still SHIPPED-UNPROVEN — sessions run `plan_mode:"advisory"`, not strict; 5 other awaited events unfired. |
| P5 | Branch flag (S-4) · partner PR #2 open · 4 worktrees · 0 stashes · +dirty = 17 untracked baks/old-bins only. |

### Campaign fixes — RE-PROVEN vs BROKEN-ON-RECHECK (fresh evidence each)

| fix | fresh verdict |
|---|---|
| Open-stamp historical persistence (405e1323) | **RE-PROVEN** — common window 2499, mismatches 0; ATR cache==db 2dp |
| Window wipe before deep replay (ClearSince) | **RE-PROVEN** — 5523 misstamped rows removed, replay repopulated clean |
| Diff on the COMMON window (c362d3d3) | **RE-PROVEN** — 3 fresh diffs window-fair |
| Deep-subscribe deadline refresh (04e1b902) | **RE-PROVEN** — backfill accepted; all 9 write sites carry fresh 5s deadlines |
| ATR probe 2dp match | **RE-PROVEN ×3 cuts** (0.0005 / 0.0041 / 0.0016) — engine includes the FORMING bar; closed-only recompute misses by 0.6-1.2 (behavior documented, not a bug) |
| Drop counters "all zero" (wave E3) | **BROKEN-ON-RECHECK** — 1399 oldest-evictions at reopen (S-2) |
| 1-line/min drop summary replaces WARN flood | **BROKEN-ON-RECHECK** — line renders `%d`/`!BADKEY` (bar_persist.go:181) |
| Retention ≥7 days (wave E4 "71.8d") | **BROKEN-ON-RECHECK** — real retention 3.6h (S-1) |
| A-2 NULL-pnl exclusions | **RE-PROVEN** — `WHERE pnl_corrected IS NOT NULL` at position_query.go:43/81/116; 0 new NULLs post-cutover |
| T7 pnl stamping | **RE-PROVEN** — newest close id 566: "realized was 97.00 (Δ+0.00)" ≤$0.50 |
| Proximity "0.3×daily-range" | **SPEC-DRIFT** — code is `proximityK × DailyRangeProxy`, default 1.5 (levels_score.go:414, levels_assemble.go:291); live band ±457pt = 1.5 × ~305pt avg day range. The 0.3 multiplier does not exist in code. |
| Stale-confirm 2.0×ATR5m | **RE-PROVEN (code+config)** — levels_volume_boot.go:19 boot line; no stale MET event today to quote live |
| nPOC seat-once | **RE-PROVEN** — dedupeSameKind before scoring (levels_assemble.go:88-90); 24/813 seated line |
| Attribution chain | **PARTIAL** — 4/5 newest closes carry valid chains (e.g. #564 LONDON v5 S3 matched); newest #566 = owner seam test, planless by design |
| Side-quota WARN-mode | **RE-PROVEN** — boot ⚖ line levels_volume_boot.go:22; no thin_side WARN fired today |
| Bias-tree | **RE-PROVEN** — 13:50/14:04 plans: branch named, anchors real (PDC 29499.75, 84% premium), premium ≤100 or BEYOND clamp |
| Flip alias canonical | **RE-PROVEN** — "2x5m close below 29526.00 EQL·15m flips the plan short" (canonical form, real anchors) |
| FVG line rendering | **PARTIAL** — 0 FVG lines in decision prompts; renders in plans (8 today incl. "No fresh FVGs exist, so no fvg_entry" — honest empty case) |
| Hysteresis 0.5×ATR14 | **RE-PROVEN (code+boot)** — plan-lifecycle boot line; 0 flips/deaths today to quote |
| Dormant/rearm | **SHIPPED-UNPROVEN** — 0 😴/⚡/replans_exhausted lines; C6 parse-loops 0 ✓; 17:25 plan honest fail-closed |
| Latency routing | **PARTIAL** — planner pinned lines quoted; exec p50/p95 not logged anywhere (no latency column in ai_charges) |
| Armed ledger | **RE-PROVEN** — 4/4 rows terminal (cancelled, seam); newest arm events = R:R refusals 13:58-14:02 ("R:R 2.04 below min 3.00") |
| STRICT ×3 gate | **SHIPPED-UNPROVEN** — live config is `plan_mode:"advisory"` (active strategy); the owner's strict intent is not yet in effect |
| level_stats nightly (T1) | **PARTIAL** — 08-26 = 18 rows, 08-25 = 28, 08-24 = 28; tonight's solo-run line awaited (journal for last night vacuumed) |
| touch_episodes | **RE-PROVEN** — 81 rows/24h; penetrations track real distance (max 200pt = real overnight extension); no "83pt" artifact class |
| Stamp gap T2 | **BROKEN-ON-RECHECK** — newest post-cutover plan has 2/12 levels without machine_grade (the fail-closed plan path still leaks) |
| SWG swings seated | **PARTIAL** — SWG-H·15m episodes live (15:22-15:48); no "missed-turns" recompute this run |
| Consumed-touch T4 | **BROKEN-ON-RECHECK** — 56 consumed rows carry times_tested=0 (fix comment claimed 8); all 56 have last_play_ms=0 → pre-fix population 7× larger than documented |
| ±2σ at 0.85 (T5) | **RE-PROVEN (rendering)** — VWAP+2σ/1σ levels present in today's plans; weight-table drift not re-diffed this run |
| σ hand-calc (T9) + S-2 residual | **S-2 DIES** — hand-calc VWAP+1σ 29603.7005 vs engine 29603.7468, diff 0.046pt (was 1.74 before the bar repair) — the residual WAS the misstamped bars |
| Tri-state UI + resolver immunity | **RE-PROVEN (code)** — DayPlanEditor tri-state + tests in web/src; Go resolvers TrimSpace!="" guard |
| EOD 14:45 flat | **UNVERIFIABLE** — 08-26 journal vacuumed (retention BROKEN); DB shows no EOD-with-position event |
| Bars integrity nightly | **RE-PROVEN** — post-fix boot line "✅ bars integrity OK: dups=0 tfs=1m total=12886" (bar_persist_wire.go:120) |

### SHIPPED-UNPROVEN register (refreshed, exact awaited events)

1. **STRICT ClassifyCitation** — NOT YET IN EFFECT: sessions run advisory
   (`plan_mode:"advisory"` in the active strategy config). Awaits the owner
   setting session plan_mode = STRICT/inherit, then the first strict-gated
   open_* that either cites a scenario or logs "no matched scenario cited
   (strict mode)" in `risk_check_error`.
2. **Natural arm pass** — awaited; R:R 2.04 < 3.00 refused every cycle
   13:55-14:02 today (quoted above). Awaits an authored arm{} with R:R ≥ 3.00.
3. **modify_bracket wire** — awaited; no live bracket modification (the only
   position today was an owner seam test, closed via plain close).
4. **armed_fill stale_reeval exemption** — awaited; no resting-limit fills.
5. **EOD-flat + watcher + BE+40** — awaited; flat at every session end since.
6. **Wake paths / dormant pair / dormant-touch wake** — awaited; zero
   dormant/rearm events this session.

### Top-5 risks (re-ranked)

1. **Journald retention ~hours** — forensics evaporate; the third per-frame log
   flood class (`auto_trader.go:43` order_update INFO) is unfixed.
2. **Ingest channel 256 vs 507 f/s reopen flood** — zero losses today, but the
   margin is one slow drain away from punctured bars again.
3. **S-4 branch flag** — deployed bar-truth code lives only on `fix/bar-truth`;
   the next dev cutover reverts it.
4. **Drop summary unreadable** (`%d` + `!BADKEY`) — the very metric meant to
   make backpressure observable is broken.
5. **STRICT intent not in effect** — owner believes strict is live; config says
   advisory (same finding as the plan-mode UX report's critical note).

### "Is the Sep-3 data foundation now sound?"

**YES — the bars layer is sound.** Evidence chain: (1) three fresh three-way
diffs, mismatches **0** over the full 2499-bar common window, including at
+50 min into live operation (cache ATR == db ATR == 16.37); (2) engine ATR14
reproduced from the repaired bars at 3 fresh cuts across 2 sessions to 2dp
(0.0005/0.0041/0.0016); (3) the open-stamp convention proven physically —
50/50 own fills fall inside their floor-minute bars and 0 fit the T+1m bar;
(4) ES converged too (0/2001); (5) S-2's 1.74pt VWAP residual closed to
0.046pt — the residual was the misstamped bars. **The FORENSICS layer around
it is not sound** (S-1/S-2): if anything goes wrong in September, the journal
may not remember it.

**One-line verdict:** data truth re-proven cold and held under live fire; the
campaign's two operational claims (drop-zero, 7-day retention) are broken and
the fix lives only on an unmerged side branch.

---

## P1 · BAR TRUTH — cold re-proof

**1.1 Fresh arbiter diffs** (POST /api/nt/bar-arbiter, owner token):
- 17:04 run: common_window 2499, mismatches **0**, cache ATR 19.40 == db 19.40.
- 17:50 run (final drift check): common_window 2499, mismatches **0**, cache
  ATR 16.37 == db 16.37. Window = the 2500-bar cache ring ∩ DB (cache ring
  spans ~08-26 00:10 CT → 17:50 CT; DB spans 08-19 10:00 → 17:08 CT).
- ES: common_window 2001, mismatches **0**, ATR 3.09 == 3.09.

**1.2 ATR at fresh cuts** (scripts/final_verify_probe.py, independent Wilder,
R2 — engine value parsed from decision_records prompts, mine rebuilt from
`bars` 1m):
| cut | engine ATR14(3m) | mine (incl. forming) | diff | 2dp |
|---|---|---|---|---|
| 08-27 05:00 | 16.8167 | 16.8162 | 0.0005 | ✓ |
| 08-27 17:25 | 13.3044 | 13.3003 | 0.0041 | ✓ |
| 08-26 18:58 | 15.0395 | 15.0379 | 0.0016 | ✓ |
Twin 1m ATR at each cut: 8.9480 / 6.7779 / 9.0321. **Finding:** the engine's
prompt ATR14 includes the forming bar (closed-only misses by 0.6-1.2) — the
arbiter handler and the engine use different series conventions, both now
reproducible.

**1.3 Open-stamp convention, every row:** `open_time_ms % 60000 != 0` → **0
rows** across 12,886 MNQ+ES bars. Fill-containment proof: all 50 newest MNQ
fills fall inside `[L,H]` of the bar open-stamped at floor(fill minute); **0**
would fit the T+1m bar. RTH-open bars at 09:30:00 exist for 08-25/26/27
(08-27: O=29606.75 H=29611.0 L=29562.5 C=29572.75 V=14632 — a plausible
48.5pt open-minute range). The off-by-one class is dead on both ingest paths.

**1.4 Live-path stamps (newest 30):** rows persisted since the 16:45:34 boot
are 17:00-17:08, contiguous for MNQ AND ES, all boundary-aligned.

**1.5 Drop counters — BROKEN claim, surviving guarantee:**
```
17:00:00 WARN bars: ingest drop summary: dropped_oldest=%d … !BADKEY=1 !BADKEY=0 !BADKEY=0
17:01:01 … !BADKEY=235 …
17:02:33 … !BADKEY=856 …
17:04:36 … !BADKEY=1399 …
```
`ingest_oldest` atomic read 855 (17:10) → 1398 (17:17) → easing. Root cause:
Globex-reopen frame flood (`frames_per_min=30456` at 17:03:36, wire_liveness)
overruns the 256-slot ingest channel; drop-oldest evictions counted but the
summary line is malformed (`provider/ninjatrader/bar_persist.go:181` passes
positional args to the structured logger — the `!BADKEY=n` values ARE the
counts). Zero closes lost: `ingest_current=0`, `persist_queue=0`, DB
contiguous.

**1.6 Retention — BROKEN:** oldest entry 13:35:21 CT (measured 17:10);
1.9G/2G used; config verified (`nofx.conf` SystemMaxUse=2G,
RateLimitBurst=200000/30s). 13:35-14:35 = 1.48GB from per-frame order_update
INFO logs at `trader/auto_trader.go:43` (25,896 lines in one second at
13:38:17). Post-fix quiet-hour rate 1.4MB/h projects ~15d, but any future
frame storm collapses it again — the ≥7d guarantee is hostage to a log
sampler that doesn't exist for order_update.

**1.7 Queue-depth headroom:** no depth metric is logged; the eviction counter
is the proxy — peak 856 evictions/min (17:01→17:02:33) against cap 256 →
headroom **0** at reopen, recovered after.

**1.8 ES:** not backfilled by the wave, yet diff-clean (0/2001) and contiguous;
`ES·4h` frames are ingested (14:59:03 backpressure line, pre-fix binary) but
only 1m is persisted — nothing reads stored ES 4h. Grade A.

**1.9 Aggregation:** cache↔DB ATR exact (16.37==16.37); my independent
rebuilt-5m ATR 16.2188 (0.9% seed/window artifact, expected). The S-1
consumer path is consistent.

## P2 — spot re-proofs

Fresh evidence per item as in the verdict table above. Key quotes: proximity
`🗺️ seated 24/813 in-band levels (proximity band ±457pt…)` 17:25:39
(levels_score.go:575); hysteresis boot `hysteresis=buffer0.5×ATR14
confirm=2close(s)`; stale unit `stale_confirm=2.0×ATR5m`
(levels_volume_boot.go:19); side-quota ⚖ boot (levels_volume_boot.go:22);
arm refusals `⚔️ arm REFUSED NY S1: R:R 2.04 below min 3.00` 13:58-14:02;
17:25:10 plan `FAIL-CLOSED: read failed after retries: no JSON object found
in planner output` (C6 honest refusal, zero parse-loops); integrity `✅ bars
integrity OK: dups=0 tfs=1m total=12886`.

## P3 — regression hunt

**3.1 Blast radius of the last 5 deploys** (f5e917da · 04e1b902 · c362d3d3 ·
405e1323 + level-truth 6fc09ad3): touched `bar_persist.go`,
`tcp_server.go`, `bar_cache.go`, `bar_persist_wire.go`, `bar_history.go`,
`handler_bar_truth.go`, `server.go`, `position_query.go`, `tcp_trader.go`.
Probes: all 9 `WriteFrame` sites carry fresh 5s deadlines (the stale-deadline
class is dead; signal/cancel/close paths exercised live today by the seam
position); `InsertBars` has exactly one production caller (upsert on the
natural key — no INSERT-semantics readers); `ClearSince` has exactly one
caller (the backfill handler). `position_query` A-2 exclusions verified at
:43/:81/:116.

**3.2 Goldens:** `TestFvgValidateGolden` + `TestVerifyPromptGoldensPasses`
(futures-empty/keylevels/plan) PASS in the worktree at running rev; boot
`goldens PASS` quoted.

**3.3 Full suites at running rev (worktree):** `go test ./...` — **27
packages ok** (api, kernel, market, store, provider/ninjatrader, trader/…);
FE `tsc --noEmit` clean; `vitest run` **277 tests / 33 files passed**.

**3.4 Live drift +50 min:** final diff at 17:50 — mismatches **0**, cache ATR
== db ATR 16.37, drops easing (34) — cache and DB stay converged during
operation, not just post-repair.

## P4 — shipped-unproven register

Refreshed above (verdict page). STRICT re-checked against live config:
`"plan_mode":"advisory"` — the awaited event is now more precise (owner sets
session plan_mode first).

## P5 — process / state

**5.1 Census:** dev tip `43bb60cb`; running `405e1323`; **FLAG: bar-truth not
merged into dev** (S-4). Open PRs: 11 (#64,63,62,61,60,56,54,53,52,51,46 —
unchanged). Worktrees: main (`fix/bar-truth`), `nofx-e2e`
(`docs/e2e-verify`), `nofx-final` (`docs/final-verify`, this run),
`nofx-recheck` (`docs/master-recheck`). Stashes: 0.
**5.2 Canon laws:** CLAUDE.md `WORKTREE LAW` :134 · `NO UNATTENDED DEPLOYS`
:140 · `SIM-only` :146 · flat-gate all-origin in the deploy canon; the same
canons recorded in the repo memory file (`/memories/repo/nofx-facts.md`).
**5.3 Partner PR #2:** still OPEN (`Sync: full tree to nofx dev @eeaffe83
(supersedes PR #1)`).
**5.4 Dirty-flag account:** `+dirty` = 17 untracked items only — 2 `.env.bak`
+ 15 `nofx-bin.old.*` binaries. No tracked modifications.

## Evidence scripts committed with this report

- `scripts/e2e_recompute.py` — independent Wilder ATR / RSI / EMA / 1m→3m/5m
  aggregation helpers (R2).
- `scripts/final_verify_probe.py` — the 3-cut engine-ATR comparison (forming
  bar included).
- `scripts/final_verify_window_hunt.py` + `scripts/final_verify_atr_variants.py`
  — the window/variant hunt that pinned the forming-bar convention.
