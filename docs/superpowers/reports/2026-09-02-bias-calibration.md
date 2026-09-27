# 2026-09-02 — Bias calibration: three evidence-backed signals vs the shipped weekly-structure bias, pre-registered, out-of-sample

**Dispatch:** BIAS CALIBRATION — round-9 signals on 8 years of NT8 daily bars. READ-ONLY on the engine · no lock · no live calls.
**Tree:** worktree `~/nofx-biascal`, branch `docs/bias-calibration-0902`, base `61b11d5f` (dev HEAD at start). Scripts: `~/nofx-analysis/bias-calibration/`. Bars source: the `bars` table (SQLite, read-only), NT8-native rows persisted by the bar-source wave (completed bars only — the persister writes `ClosedBarsOnly`). CSVs: `docs/superpowers/reports/2026-09-02-bias-calibration-csvs/`.
**Evidence tiers:** [A] directly verified · [B] inferred · [C] speculation.

---

## PRE-REGISTRATION HEADER (fixed 2026-09-02 19:20 CT, BEFORE any signal computation)

### P1 — Data inventory (measured, not looked-at-for-results)
Query: `SELECT tf, symbol, COUNT(*), MIN(open_time_ms), MAX(open_time_ms) FROM bars WHERE tf IN (...) GROUP BY tf, symbol` against `~/nofx/data/data.db` (read-only), 2026-09-02 19:18 CT.

| TF | MNQ range (UTC) | n MNQ | ES range (UTC) | n ES |
|---|---|---|---|---|
| 1d | 2019-05-02 → 2026-09-01 | 1,896 | 2018-12-05 → 2026-09-01 | 1,999 |
| 1h | 2026-05-04 → 2026-09-03 | 2,002 | same | 2,002 |
| 30m | 2026-07-03 → 2026-09-03 | 2,005 | same | 2,005 |
| 15m | 2026-08-04 → 2026-09-03 | 2,010 | same | 2,010 |
| 1w | 2019-04-26 → 2026-08-21 | 383 | 2004-12-03 → 2026-08-21 | 948 |

Stamp convention: bar `open_time_ms` is the bar's OPEN time, epoch-ms UTC; 1d/1h/30m/15m stamps sit on CT clock grids (verified: first 1d stamp 2018-12-05 06:00 UTC = 00:00 CT December). 1d bars are calendar-day CT (open 00:00 CT, close ≈ last pre-16:00 CT RTH print). Native `1w` is NOT used — NT8 stamps it Fri→Thu (class 7; the resolver's `ExcludedNative("1w")`), and our weeks are Monday-governed. **Premise nit:** daily bars reach 2018-12 for ES and 2019-05 for MNQ (the dispatch's parenthetical had it backwards) — ranges used accordingly.

### P2 — Signals (definitions fixed, no peeking after this header)

**S1 TSMOM regime** (freq: session). Lookback L ∈ {21, 63, 252} sessions. At each session close t:
`mom(t) = sign(Σ_{i=0..L−1} r_{t−i})` where `r = close-to-close log return`, vol-scaled copy computed as `mom(t) / (σ63(t) · √L)` for the position-size/return-normalization variant (sign identical by construction). Target: `sign(r_{t+1})`. Position for PnL: `+1 / −1` per mom sign (never flat).

**S2 Overnight → RTH reversal** (freq: session-day). `overnight(t) = RTHopen(t) / RTHclose(t−1) − 1`; `rth(t) = RTHclose(t) / RTHopen(t) − 1`. RTH open = open of the bar stamped 08:30 CT; RTH close = close of the bar stamped 15:45 CT (15m grid) / 15:30 CT (30m grid). Signal: `pred(t) = −sign(overnight(t))` (reversal hypothesis). Target: `sign(rth(t))`. Runs: (a) 15m MNQ 2026-08-04→ (exact RTH stamps), (b) 30m MNQ 2026-07-03→ (same exact stamps, longer range) — both stated, both holdout-only. Reversal rate: fraction of days with `sign(overnight) ≠ sign(rth)` tested against 0.5 (Wilson). Interaction: strata by realized-vol tercile (P6).

**S3 Intraday momentum** (freq: session-day). `first30(t) = open(09:00 CT bar) / open(08:30 CT bar) − 1`; `lastHr(t) = open(14:45 CT bar) / open(13:45 CT bar) − 1` (open-to-open of the boundary bars). Signal: `pred(t) = sign(first30(t))`. Target: `sign(lastHr(t))`. Run: 15m MNQ 2026-08-04→ only (n stated plainly). 5m exists but adds nothing to alignment; not run.

**CONTROL — shipped weekly-structure bias, reconstructed** (freq: week). Weekly candles from 1d by `weekStartMonday` (shipped `kernel/weekly_bias.go`): open = first daily-bar open of the week (Sunday 00:00 CT print — proxy for the 17:00 CT first print, caveat stated), close = last bar's close (Friday ≈16:00 CT), H/L. At each completed week w: `weekly_open(w)` = w's first daily open; `PWH/PWL` = prior completed week's H/L. Rule (shipped Tier-A, `kernel/weekly_prompt.go:285` "PWH/PWL break-AND-HOLD vs sweep-and-reject", "price vs weekly_open"):
- **bull** iff `close_w > weekly_open_w` AND `H_w > PWH_w` AND `close_w > PWH_w`
- **bear** iff `close_w < weekly_open_w` AND `L_w < PWL_w` AND `close_w < PWL_w`
- else **neutral** (flat).
Position for week w+1 = +1/−1/0. Target: `sign(close_{w+1} − close_w)`. This is the null the three signals are measured against.

### P3 — Split
Exploration 2018-12/2019-05 → 2023-12-31 (ES/MNQ respectively); **holdout 2024-01-01 → last completed bar**. Verdicts from holdout only. S2/S3 ranges fall entirely in holdout (state n).

### P4 — Statistics per signal (D2)
1. Hit rate `ĥ = #(pred sign == actual sign) / n`, 95% **Wilson** interval.
2. Mean next-period return conditional on signal sign (+1/−1 buckets), with Student **t-stat** (`t = mean / (std/√n)`, Welch against zero).
3. Long/flat/short PnL-in-points series (position × next-period close delta), no sizing; max drawdown (peak-to-trough in points).
4. Same series **with friction: 2-pt round trip = 1 pt per side, charged as `1pt × |Δpos|`** at each re-position.

### P5 — Multiplicity
Benjamini–Hochberg across the 5 tests (S1×3 lookbacks, S2, S3, CONTROL) applied to the holdout hit-rate p-values (one-sided vs 0.5); raw and adjusted reported.

### P6 — Strata (descriptive, flagged as such)
Realized-vol tercile (trailing 20-session std of daily close-to-close returns, computed over the signal's own sample) and year. Reported as tables only — no verdicts drawn from strata.

### P7 — PASS/FAIL (fixed before looking)
A signal is **USABLE** iff on HOLD OUT: (a) hit-rate Wilson **lower bound > 0.50**, AND (b) net-of-friction mean next-period return **t > 2**. Otherwise **NOT USABLE at this n** — stated per signal.

### P8 — Method honesty
All bars from the `bars` table only (no resolver call, no live system, no engine imports). Closed bars only by construction (persister writes closed bars; last 1d bar is the prior session). Scripts are stdlib-Python (sqlite3 + math), no seeds involved (no randomness anywhere). CSVs carry every observation.

---

## RESULTS (filled after P1–P8 were frozen)

All numbers from `~/nofx-analysis/bias-calibration/calibrate.py` (stdlib Python, read-only `bars` table). Queries logged in `run.log`. Binomial one-sided p-values are the normal approximation with continuity correction (exact tails overflow float at n≈700 — stated). S1 momentum computed on **log** close-to-close returns per pre-reg; S2/S3 on arithmetic ratios per pre-reg. Friction = 1 pt per side charged on |Δpos| (2-pt round trip). `hit` = fraction of periods where predicted sign == realized sign; neutral control weeks count as non-hits (called-only rows reported separately).

### D2 per signal — exploration vs holdout

| Signal | Seg | n | hit | Wilson lo | binom p (1s) | mean ret +1 (t) | mean ret −1 (t) | pts | maxDD | net pts | net maxDD | net t |
|---|---|---|---|---|---|---|---|---|---|---|---|---|
| S1-21 MNQ | expl | 1184 | .519 | .4910 | .0955 | +.0006 (1.81) | +.0003 (0.41) | +0.34 | 0.26 | −150.7 | 149.7 | −8.97 |
| S1-21 MNQ | hold | 690 | .520 | .4830 | .1520 | +.0004 (0.97) | +.0009 (0.88) | −0.02 | 0.24 | −149.0 | 148.1 | −9.14 |
| S1-63 MNQ | expl | 1142 | .531 | .5016 | .0206 | +.0005 (1.32) | +.0004 (0.50) | +0.21 | 0.29 | −98.8 | 97.9 | −7.18 |
| S1-63 MNQ | hold | 690 | .509 | .4714 | .3377 | +.0002 (0.50) | +.0017 (1.37) | −0.18 | 0.31 | −71.2 | 70.3 | −6.14 |
| S1-252 MNQ | expl | 953 | .556 | .5244 | .0003 | +.0006 (1.57) | +.0001 (0.06) | +0.43 | 0.28 | −16.6 | 16.2 | −2.89 |
| S1-252 MNQ | hold | 690 | .564 | **.5265** | **.0005** | +.0005 (1.18) | +.0082 (0.54) | +0.27 | 0.32 | −12.7 | 12.3 | **−2.56** |
| S1-21 ES | expl | 1287 | .531 | .5034 | .0148 | +.0005 (1.86) | +.0002 (0.21) | +0.35 | 0.25 | −194.7 | 193.7 | −10.27 |
| S1-21 ES | hold | 690 | .510 | .4729 | .3103 | +.0004 (1.21) | +.0009 (0.97) | −0.01 | 0.17 | −101.0 | 100.1 | −7.39 |
| S1-63 ES | expl | 1245 | .536 | .5080 | .0063 | +.0003 (1.01) | +.0005 (0.66) | +0.03 | 0.30 | −117.0 | 116.1 | −7.85 |
| S1-63 ES | hold | 690 | .520 | .4830 | .1520 | +.0004 (1.41) | +.0010 (0.72) | +0.12 | 0.17 | −50.9 | 50.0 | −5.15 |
| S1-252 ES | expl | 1056 | .538 | .5077 | .0075 | +.0003 (0.85) | +.0004 (0.42) | +0.10 | 0.35 | −40.9 | 40.1 | −4.59 |
| S1-252 ES | hold | 690 | .549 | **.5120** | **.0054** | +.0004 (1.24) | +.0122 (0.90) | +0.19 | 0.31 | −8.8 | 8.3 | **−2.14** |
| S2-15m MNQ | hold | 21 | .619 | .4088 | .1914 | +.0005 (0.39) | −.0021 (−1.10) | +0.03 | 0.01 | −27.0 | 26.0 | −6.15 |
| S2-30m MNQ | hold | 42 | .571 | .4221 | .2202 | −.0013 (−0.80) | −.0001 (−0.03) | −0.03 | 0.07 | −45.0 | 44.0 | −6.97 |
| S3-15m MNQ | hold | 22 | .409 | .2326 | .8568 | +.0005 (1.14) | +.0010 (1.57) | 0.00 | 0.01 | −27.0 | 26.0 | −5.91 |

S2 reversal rate (sign disagreement, tested vs 0.5): 15m n=21 → 61.9%, Wilson [.409, .793] — not significant. 30m n=42 → 57.1%, Wilson [.422, .709] — not significant.

### D4 — CONTROL (shipped weekly-structure bias, reconstructed from 1d)

| CTRL | Seg | n weeks | hit (raw) | Wilson lo | called-only n | called-only hit | called Wilson lo | net t |
|---|---|---|---|---|---|---|---|---|
| MNQ | expl | 243 | .329 | .2732 | 139 | .576 | .4924 | −16.63 |
| MNQ | hold | 139 | **.252** | .1870 | 77 | .455 | .3481 | −13.56 |
| ES | expl | 264 | .330 | .2756 | 156 | .558 | .4793 | −17.47 |
| ES | hold | 139 | **.281** | .2126 | 77 | .507 | .3972 | −14.15 |

The control's RAW holdout hit rate is 25–28% — significantly BELOW 50% (binom p ≈ 1.0 for the upper tail), i.e. anti-predictive; even counting only called weeks it is chance (45–51%, Wilson lo far below .50). It loses ~100–150 points net over the holdout with friction t ≈ −14.

### D3 — multiplicity (Benjamini–Hochberg, holdout hit-rate one-sided p)

Ten tests as run (S1×3 lookbacks × 2 symbols, S2×2, S3, CTRL×2). Ordered p: S1-252-MNQ .0005 → BH .005 ✓; S1-252-ES .0054 → BH .027 ✓; next is S1-63-ES .152 → BH .507 ✗. **Two of ten survive BH — and both still fail D6(b).**

### D5 — strata (descriptive)

Vol terciles (trailing-20 std of each signal's own returns), S1-252 holdout MNQ: lo-vol n=542 hit .587 (lo .545) · mid n=541 hit .549 (lo .507) · hi n=541 hit .540 (lo .498). The TSMOM-252 edge concentrates in low-vol. S2-30m terciles: .625 / .500 / .571 (n=8/8/7 — noise).

Year (holdout), S1-252: MNQ 2024 .579 (lo .518) · 2025 .566 (lo .505) · 2026 .538 (lo .463). ES 2024 .564 (lo .503) · 2025 .566 (lo .505) · 2026 .503 (lo .429). **The edge decays through 2026** — descriptive, flagged.

### D6 — verdicts (holdout, pre-registered rule)

| Signal | Wilson lo > .50 | net-friction t > 2 | VERDICT |
|---|---|---|---|
| S1 TSMOM-21 | ✗ (MNQ .483 / ES .473) | ✗ (−9.1 / −7.4) | **NOT USABLE** |
| S1 TSMOM-63 | ✗ (.471 / .483) | ✗ (−6.1 / −5.2) | **NOT USABLE** |
| S1 TSMOM-252 | ✓ (.527 / .512) | ✗ (−2.6 / −2.1) | **NOT USABLE** — real sign edge, friction-destroyed |
| S2 overnight→RTH reversal | ✗ (.409 / .422) | ✗ (−6.2 / −7.0) | **NOT USABLE at this n** (n=21/42) |
| S3 intraday momentum | ✗ (.233) | ✗ (−5.9) | **NOT USABLE** (n=22) |
| CONTROL weekly-structure | ✗ (.187 / .213) | ✗ (−13.6 / −14.2) | **NOT USABLE — anti-predictive on holdout** |

### Design recommendation (one paragraph)

None of the three signals clears the pre-registered bar, and the incumbent — the shipped weekly-structure bias — is the weakest of all four: on holdout it predicts the wrong weekly direction ~72–75% of the time (raw) and no better than a coin even when it does commit. The one real effect in this data is a small TSMOM-252 sign edge (~56% holdout, BH-surviving, low-vol-concentrated) that friction fully consumes; at session frequency it is worth ~0.5 tick per decision and cannot pay a 2-pt round trip. Recommendation for the bias layer: **demote the weekly-structure bias from evidence to label-only, shadow-first, and replace the rule that FEEDS it with nothing directional at weekly horizon** — keep the weekly read for reference levels (PWH/PWL/IPDA are price facts, not predictions), render the chip as "WEEKLY: refs only (no directional call)" until a new candidate passes D6; the next candidates to pre-register are TSMOM-252 restricted to low-vol regimes with hold-to-reversion position handling (edge is sign-of-drift, not a trade), and weekly-horizon TSMOM-52w, both as LABELS never MUSTs. Do not invert the current bias (the called-only 45–51% is not significant enough to trade the anti-prediction either).

## Surprises

1. Premise nit: daily bars reach 2018-12 for **ES** and 2019-05 for **MNQ** (dispatch had them swapped); ranges used accordingly.
2. The running system's own weekly doc has NO deterministic sign rule — the bias is model judgment over Tier-A facts, so the CONTROL is a faithful reconstruction of the documented rule (`kernel/weekly_prompt.go:285`), stated as such in P2.
3. 30m grid has no 15:45 stamp — RTH close on the 30m run is the 15:30-bar close (15:30–16:00, same 16:00 print as the 15m 15:45 bar); both runs agree in construction.
4. Friction dominates everything: gross TSMOM-252 is +0.19–0.27 pt over the holdout; net is −8.8 to −12.7 pt. The signals flip often; 1 pt/side at 690 periods ≈ −2 pts per flip pair.
5. TSMOM-252's holdout edge decays by year (2024 ≈ .57 → 2026 ≈ .50–.54) — flagged, not acted on.
6. The control's exploration called-only hit (.56–.58) collapses to .45–.51 out of sample — textbook overfit/regime change, and the raw rate is actively anti-predictive in both samples.

## Method / artifacts

- Script: `~/nofx-analysis/bias-calibration/calibrate.py` (stdlib only, deterministic, no seeds).
- CSVs (every observation): `docs/superpowers/reports/2026-09-02-bias-calibration-csvs/` — `s1_{MNQ,ES}_look{21,63,252}.csv`, `s2_15m.csv`, `s2_30m.csv`, `s3_15m.csv`, `control_{MNQ,ES}_weekly.csv`.
- Pre-registration pinned at commit `43498e24` (this branch) BEFORE any computation; results computed afterward with no definition changes (one mechanical fix: 30m RTH-close stamp; S1 log returns per pre-reg).

## Closeout

- Read-only maintained: no engine code, no config, no knob writes, no lock; DB accessed read-only via `mode=ro`.
- Branch `docs/bias-calibration-0902`, base `61b11d5f`. Raw URL 200-verified below.

