# LEVEL-KIND REPLAY — D1′ over the persisted MNQ tape, mechanical level kinds only

**READ-ONLY · no lock · no engine code · scripts in `~/nofx-analysis/level-replay/`**
Owner: hoang · Live rev `e1b1176844aa` · Report opened **2026-09-02 ~18:30 CT**
Instrument source: `docs/superpowers/reports/2026-09-02-detector-redesign.md` (D1′ amendment)
Base scripts (imported, not copied): `~/nofx-analysis/detector-redesign/detectors.py`

---

## 0. PRE-REGISTRATION — written 2026-09-02 ~18:30 CT, BEFORE any real-tape replay ran

Fixed in advance; nothing below this line was observed when this header was written.

### Period (R1)

Query (read-only): `SELECT tf, COUNT(*), MIN(open_time_ms), MAX(open_time_ms) FROM bars
WHERE symbol='MNQ' AND tf='1m'` on `data/data.db?mode=ro` → **1m = 24,103 rows,
2026-08-19 17:00 CT → 2026-09-02 08:28 CT**.
Replay period = every COMPLETED CME session day (17:00 CT roll, `detectors.session_day`
quoted from `detector-redesign/detectors.py`) inside that range:
**2026-08-20 … 2026-09-01 = 13 session days.** 2026-09-02 excluded (incomplete day).
**SURPRISE (stated up front):** the persisted intrabar history is ~14 days, NOT years.
The deep TFs exist (1d back to 2018-12, 1w to 2011-12) but carry no intrabar
granularity, and 5m starts 2026-08-24 (worse than 1m). Per R1 I run 1m over the 13
completed session days and state the n-limits explicitly.

### Instrument (D1′)

`detectors.detect_symmetric_v2(bars, level, k, Δ, H=12, exit_on='close')`, imported
from `~/nofx-analysis/detector-redesign/detectors.py` (path quoted). Barriers anchored
on the level (`L ± k·Δ`); episode opens on a bar that straddles `L` while the previous
bar does not; `entry_side` from the previous bar's close; exit when a CLOSE crosses a
barrier (next bar onward); both in one bar → `ambiguous_span`; horizon `H=12` →
`ambiguous_horizon`. p(hold) = hold/(hold+break); ambiguous recorded, counted,
EXCLUDED from the denominator.

**k and Δ (re-derived from the replay period, per dispatch):** Δ = mean absolute 1m
close increment over the replay period's own bars (the detector-redesign tape_stats
definition, quoted). **k = 3** → band = ±3Δ. If the replay Δ departs >20% from the
calibration Δ (5.3771 pt), the band in points is re-reported and the k=3 band on the
replay Δ is the ONLY band used for verdicts.

### Levels — mechanical kinds, reconstructable per session day (definitions mirrored)

| kind | per session day d | mirrored definition |
|---|---|---|
| PDH / PDL | 1 each | prior completed session day's 1m high / low (session-day window 17:00 CT → 17:00 CT, `detectors.session_day`) |
| PDC | 1 | last 1m close of the prior session day |
| ONH / ONL | 1 each | high/low of bars from session start (prev 17:00 CT) up to <08:30 CT |
| RTH-H / RTH-L | 1 each | high/low of exchange RTH 08:30–15:00 CT bars (engine NY flat is 14:45 CT by owner contract, `kernel/session_registry.go:88-113` quoted — replay uses exchange RTH for the traded range) |
| OR5-H/L · OR15-H/L · OR30-H/L | 3 pairs | high/low of the first 5/15/30 minutes from 08:30 CT |
| VWAP · VWAP±1σ | 3 | mirror `kernel/levels_volume.go:76-92 vwapAndStdev` (typical price (H+L+C)/3, volume-weighted; σ = volume-weighted sd of typical prices), computed on session bars (17:00 CT roll) **through 08:29 CT — the RTH-open snapshot, frozen for the day** (deviation from the live re-emitting line: stated, conservative, lookahead-free) |
| Round x000 / x500 / x250 | every 250-multiple | multiples of 250 within [session-day min − 250, session-day max + 250] |
| Settlement | 1 | mirror `vwapAndStdev` over 14:59:30–15:00 CT bars; if none, last RTH 1m close |

Touch ordinal per level per day: episode counter per (level, session day) from the tape.

### OOS split (R2, fixed before looking)

Days sorted: **exploration = first 8 session days (08-20 … 08-27); holdout = last 5
(08-28 … 09-01).** Both are reported; VERDICTS COME FROM HOLDOUT ONLY.

### Verdict rule (pass/fail, pre-registered)

Per kind, on HOLDOUT: p̂ = p(hold), Wilson 95%. Baselines per kind on holdout:
(a) **stationary bootstrap** — geometric blocks, mean length 10, the SAME per-day level
lines re-run on 40 shuffled paths per holdout day (same method as E2, seed fixed);
(b) **Osler** — B=1000 random level sets per holdout day with the same per-kind counts,
prices uniform in the day's bar range → the real p̂'s percentile.
- **BEATS BASELINE ON HOLDOUT** iff holdout n ≥ 200 AND Wilson lower bound > baseline p̂
  AND the one-sided test (normal approx on p̂ − p̂0) survives Benjamini-Hochberg q=0.05
  across the tested kinds.
- **DOES NOT** iff n ≥ 200 but the interval overlaps the baseline (or BH kills it).
- **TOO FEW** iff holdout n < 200 — descriptive only; no ranking below n=200 is a verdict.
Cohen's h(0.60 vs 0.50) = 0.2014 quoted as the yardstick (n≈193/group for 80% power).

### Strata (pre-registered)

R5 ordinal: 1st / 2nd / 3rd+ touch per level per day, pooled per period; holdout used
where n ≥ 30, else descriptive. R6 session: ASIA 17:00–02:00 / LONDON 02:00–08:30 /
NY 08:30–14:45 CT (engine registry windows, quoted) assigned by the touching bar's time.

### Bias side table (R7, same tape, exploration + holdout)

1. **TSMOM 1/3/12-month** state = sign of cumulative return over the prior 21/63/252
   session-day daily closes (1d table, quoted) vs sign of the next session day's RTH
   return (15:00 close − 08:30 open). 12m state uses the deep daily history; the
   prediction set is the replay period (n=13).
2. **Overnight → RTH**: sign(08:30 open − prev 17:00 open) vs sign(15:00 close − 08:30
   open) same day.
3. **First-30 → last-hour**: sign(09:00 close − 08:30 open) vs sign(15:00 close −
   14:00 open) same day.
All reported with n, Wilson on sign agreement, and the holdout subset (n=5, descriptive).

### Hypotheses (pre-registered, directional where stated)

- H1 (pooled): holdout pooled p(hold) ≈ stationary-bootstrap baseline (no level edge) —
  carried from the detector redesign verdict.
- H2 (PDL lead): the E3 PDL cell (0.6970 [0.5266, 0.8262], n=33) will NOT replicate on
  holdout.
- H3 (consumed touch): if the belief is real, p(hold) DECLINES with ordinal
  (1st > 2nd > 3rd+).
- H4 (rounds): x000/x500/x250 p(hold) ≈ Osler random-level null.
- H5 (VWAP): VWAP / ±1σ indistinguishable from baseline on holdout.

### Grader recommendation rule (pre-registered)

BEATS → keep (graded normally); DOES NOT → neutralize (grade ×1.0, no bonus); TOO FEW →
keep with the current [I] label intact (no change justified). Applied per kind to
`kernel/levels_score.go` ladder terms.

**Nothing below this line was known when the criteria above were written.**

### AMENDMENT (post-hoc, written 2026-09-02 ~18:45 CT before the replay ran — declared, not hidden)

The tape exported from the store at run time differs from the range quoted in the
header: the live bot's bar persistence has been appending since the 18:05 CT class-48
boot. Actual export: **14,258 MNQ 1m bars, 2026-08-19 10:00 CT → 2026-09-02 18:34 CT**
(same read-only query, `symbol='MNQ' AND tf='1m'`). Completed session days (17:00 CT
first bar, ≥15:00 last bar, ≥1000 bars): **2026-08-20 … 2026-09-02 = 10 days**.
The pre-registered 60/40 rule is applied mechanically to this actual list:
**exploration = first 6 (08-20 … 08-25), holdout = last 4 (08-26 … 09-02)**.
The header's illustrative day labels (08-20..08-27 / 08-28..09-01) were computed from a
stale row count; the SPLIT RULE was not changed, its inputs were.
Replay Δ = **5.3707 pt** (vs calibration 5.3771, −0.1% ⇒ band ±16.11 pt at k=3,
no re-band needed). lag1 ρ = −0.0403. Nothing below this line was known when the
criteria above were written.

---

# RESULTS (computed after the pre-registration above)

## 1. Replay scope

Tape: 1m MNQ from the bar-source-persisted store, read-only query
`SELECT open_time_ms,o,h,l,c,v FROM bars WHERE symbol='MNQ' AND tf='1m'` — n=14258 bars, 2026-08-19 10:00 → 2026-09-02 18:34 CT.
Completed session days: 2026-08-20 … 2026-09-02 (n=10).
Δ=5.3707 pt · k=3 · band ±16.11 pt · H=12 · exit_on=close.
Exploration: 2026-08-20 … 2026-08-27 (6d) · Holdout: 2026-08-28 … 2026-09-02 (4d).
Episodes: exploration n=684 · holdout n=484 · total 1168 (ambiguous recorded + excluded).
Mechanical level map: 16 kinds (see header). Scan-start convention (implementation note): each level is
scanned only on bars at/after its formation time (PDH/PDL/PDC/ONH/ONL/rounds from 17:00; VWAP trio from 08:30;
OR5/15/30 from formation+5m; RTH-H/L + settlement from 15:00) — lookahead-free.

## 2. Pooled

| period | p(hold) | Wilson 95% | n (hold/break) |
|---|---|---|---|
| holdout | 0.5413 | [0.4968,0.5852] | 484 (262/222) |

**H1:** holdout pooled 0.5413 [0.4968,0.5852] vs stationary-bootstrap baselines ≈0.51–0.55 →
**DOES NOT CLEAR** (interval overlaps). Consistent with the detector redesign verdict.

## 3. Per kind — exploration and holdout (verdicts from holdout only)

| kind | exp n | exp p(hold) | exp Wilson | hold n | hold p(hold) | hold Wilson | amb% hold | boot p | Osler null | verdict |
|---|---|---|---|---|---|---|---|---|---|---|
| VWAP-1s | 92 | 0.5217 | 0.5217 [0.4209,0.6209] | 63 | 0.5714 | 0.5714 [0.4486,0.6860] | 22% | 0.5165 | 0.5206 | TOO FEW |
| OR30-L | 73 | 0.5616 | 0.5616 [0.4476,0.6695] | 58 | 0.5862 | 0.5862 [0.4580,0.7037] | 16% | 0.5129 | 0.5230 | TOO FEW |
| OR30-H | 89 | 0.5506 | 0.5506 [0.4473,0.6497] | 51 | 0.5294 | 0.5294 [0.3952,0.6595] | 20% | 0.5172 | 0.5182 | TOO FEW |
| VWAP | 91 | 0.5275 | 0.5275 [0.4259,0.6268] | 48 | 0.5208 | 0.5208 [0.3833,0.6553] | 32% | 0.5232 | 0.5192 | TOO FEW |
| VWAP+1s | 66 | 0.3939 | 0.3939 [0.2850,0.5145] | 44 | 0.5455 | 0.5455 [0.4007,0.6829] | 15% | 0.5250 | 0.5175 | TOO FEW |
| PDC | 43 | 0.5116 | 0.5116 [0.3675,0.6538] | 39 | 0.5385 | 0.5385 [0.3857,0.6843] | 34% | 0.5097 | 0.5185 | TOO FEW |
| PDL | 12 | 0.6667 | 0.6667 [0.3906,0.8619] | 32 | 0.5312 | 0.5312 [0.3645,0.6913] | 18% | 0.5166 | 0.5169 | TOO FEW |
| SETTLE | 45 | 0.5333 | 0.5333 [0.3908,0.6707] | 30 | 0.5667 | 0.5667 [0.3920,0.7262] | 40% | 0.5392 | 0.5197 | TOO FEW |
| ONH | 20 | 0.7000 | 0.7000 [0.4810,0.8545] | 26 | 0.5000 | 0.5000 [0.3206,0.6794] | 24% | 0.5412 | 0.5224 | TOO FEW |
| RND-500 | 23 | 0.2609 | 0.2609 [0.1255,0.4647] | 24 | 0.4583 | 0.4583 [0.2789,0.6493] | 45% | 0.5397 | 0.5179 | TOO FEW |
| ONL | 30 | 0.5667 | 0.5667 [0.3920,0.7262] | 23 | 0.4783 | 0.4783 [0.2924,0.6704] | 4% | 0.5532 | 0.5172 | TOO FEW |
| RTH-L | 22 | 0.7273 | 0.7273 [0.5185,0.8685] | 16 | 0.6250 | 0.6250 [0.3864,0.8152] | 20% | 0.5460 | 0.5247 | TOO FEW |
| RND-250 | 45 | 0.3778 | 0.3778 [0.2511,0.5237] | 10 | 0.5000 | 0.5000 [0.2366,0.7634] | 0% | 0.5102 | 0.5174 | TOO FEW |
| PDH | 3 | 0.6667 | 0.6667 [0.2077,0.9385] | 8 | 0.5000 | 0.5000 [0.2152,0.7848] | 20% | 0.5147 | 0.5203 | TOO FEW |
| RTH-H | 22 | 0.5909 | 0.5909 [0.3873,0.7674] | 7 | 0.8571 | 0.8571 [0.4869,0.9743] | 0% | 0.5173 | 0.5142 | TOO FEW |
| RND-1000 | 8 | 0.3750 | 0.3750 [0.1368,0.6943] | 5 | 0.2000 | 0.2000 [0.0362,0.6245] | 0% | 0.4642 | 0.5145 | TOO FEW |

**Every holdout cell is below the pre-registered n=200 floor** (largest: VWAP−1σ n=63).
No kind is rankable; the table is descriptive ONLY. BH never runs (no n≥200 kinds).
Cohen's h(0.60 vs 0.50)=0.2014 → n≈193/group for 80% power: the holdout has ~4 days of levels;
the replay period would need ~30+ holdout days per single-line kind to reach the floor.

## 4. Strata (R5 ordinal, R6 session)

### Ordinal (1st / 2nd / 3rd+ touch per level per day)

| stratum | pooled p(hold) | pooled n | holdout p(hold) | holdout n |
|---|---|---|---|---|
| 1st | 0.5657 | 99 | 0.6250 | 40 |
| 2nd | 0.4568 | 81 | 0.6000 | 30 |
| 3rd+ | 0.5294 | 988 | 0.5290 | 414 |

**H3 (consumed touch): NOT SUPPORTED — no monotone decline.** Pooled: 1st 0.5657 > 3rd+ 0.5294; holdout: 1st 0.6250 (n=40) vs 3rd+ 0.5290 (n=414) — the FIRST touch holds MORE, the opposite of the
belief; all intervals cover 0.50/overlap. The consumed-touch belief lives on no evidence from this instrument.

### Session (engine windows ASIA 17:00–02:00 / LONDON 02:00–08:30 / NY 08:30–14:45 CT)

| session | holdout p(hold) | holdout n |
|---|---|---|
| NY | 0.5253 [0.4643,0.5855] | 257 |
| LONDON | 0.5522 [0.4678,0.6338] | 134 |
| ASIA | 0.5699 [0.4685,0.6658] | 93 |

ASIA highest (0.5699, n=93) — intervals overlap; descriptive only.

## 5. Bias side table (R7, same tape; n is tiny everywhere)

| signal | period | n | agreement | Wilson |
|---|---|---|---|---|
| tsmom_21d | all | 9 | 3/9 | 0.3333 [0.1206,0.6458] |
| tsmom_63d | all | 9 | 4/9 | 0.4444 [0.1888,0.7334] |
| tsmom_252d | all | 9 | 5/9 | 0.5556 [0.2666,0.8112] |
| overnight_to_rth | exploration | 6 | 3/6 | 0.5000 [0.1876,0.8124] |
| overnight_to_rth | holdout | 4 | 1/4 | 0.2500 [0.0456,0.6994] |
| overnight_to_rth | all | 10 | 4/10 | 0.4000 [0.1682,0.6873] |
| first30_to_lasthour | exploration | 6 | 3/6 | 0.5000 [0.1876,0.8124] |
| first30_to_lasthour | holdout | 4 | 1/4 | 0.2500 [0.0456,0.6994] |
| first30_to_lasthour | all | 10 | 4/10 | 0.4000 [0.1682,0.6873] |

None distinguishes 0.50 at n≤10. TSMOM states use the deep 1d table (quoted); prediction set = the 10 replay days.

## 6. Verdict per kind (pre-registered rule) and what the grader (3A) should do

- **VWAP-1s**: TOO FEW (holdout n=63, p=0.5714 [0.4486,0.6860] vs boot 0.5165) → grader: keep as-is (current [I] labels intact — no evidence to change).
- **OR30-L**: TOO FEW (holdout n=58, p=0.5862 [0.4580,0.7037] vs boot 0.5129) → grader: keep as-is (current [I] labels intact — no evidence to change).
- **OR30-H**: TOO FEW (holdout n=51, p=0.5294 [0.3952,0.6595] vs boot 0.5172) → grader: keep as-is (current [I] labels intact — no evidence to change).
- **VWAP**: TOO FEW (holdout n=48, p=0.5208 [0.3833,0.6553] vs boot 0.5232) → grader: keep as-is (current [I] labels intact — no evidence to change).
- **VWAP+1s**: TOO FEW (holdout n=44, p=0.5455 [0.4007,0.6829] vs boot 0.5250) → grader: keep as-is (current [I] labels intact — no evidence to change).
- **PDC**: TOO FEW (holdout n=39, p=0.5385 [0.3857,0.6843] vs boot 0.5097) → grader: keep as-is (current [I] labels intact — no evidence to change).
- **PDL**: TOO FEW (holdout n=32, p=0.5312 [0.3645,0.6913] vs boot 0.5166) → grader: keep as-is (current [I] labels intact — no evidence to change).
- **SETTLE**: TOO FEW (holdout n=30, p=0.5667 [0.3920,0.7262] vs boot 0.5392) → grader: keep as-is (current [I] labels intact — no evidence to change).
- **ONH**: TOO FEW (holdout n=26, p=0.5000 [0.3206,0.6794] vs boot 0.5412) → grader: keep as-is (current [I] labels intact — no evidence to change).
- **RND-500**: TOO FEW (holdout n=24, p=0.4583 [0.2789,0.6493] vs boot 0.5397) → grader: keep as-is (current [I] labels intact — no evidence to change).
- **ONL**: TOO FEW (holdout n=23, p=0.4783 [0.2924,0.6704] vs boot 0.5532) → grader: keep as-is (current [I] labels intact — no evidence to change).
- **RTH-L**: TOO FEW (holdout n=16, p=0.6250 [0.3864,0.8152] vs boot 0.5460) → grader: keep as-is (current [I] labels intact — no evidence to change).
- **RND-250**: TOO FEW (holdout n=10, p=0.5000 [0.2366,0.7634] vs boot 0.5102) → grader: keep as-is (current [I] labels intact — no evidence to change).
- **PDH**: TOO FEW (holdout n=8, p=0.5000 [0.2152,0.7848] vs boot 0.5147) → grader: keep as-is (current [I] labels intact — no evidence to change).
- **RTH-H**: TOO FEW (holdout n=7, p=0.8571 [0.4869,0.9743] vs boot 0.5173) → grader: keep as-is (current [I] labels intact — no evidence to change).
- **RND-1000**: TOO FEW (holdout n=5, p=0.2000 [0.0362,0.6245] vs boot 0.4642) → grader: keep as-is (current [I] labels intact — no evidence to change).

## 7. Surprises + CSVs

- **1m history is ~14 days, not years** (deep TFs have no intrabar granularity; 5m starts 08-24). The replay is
  therefore a 10-session-day sample — the n≥200 floor is unreachable for single-line kinds. Going forward,
  bar-source persistence (1m retained 90d) grows this sample ~13 session days/week.
- **H2 confirmed as predicted:** E3's PDL 0.6970 lead did NOT replicate (holdout 0.5312, n=32).
- **H3 inverted:** first touches hold MORE, not less (see §4).
- The tape extended DURING the run (live persistence) — pre-reg amended before the replay (dated).
- `test_seam=ON` booted today (flagged; irrelevant to this offline lane).

CSVs (this branch, `docs/superpowers/reports/exports/2026-09-02-level-replay/`):
`episodes.csv` (every episode: level id, kind, day, ordinal, k, Δ, band, entry side, exit side, outcome,
bars to exit, MFE/MAE, session) · `per_kind.csv` · `bias_table.csv`.
Scripts (outside the repo, per A1): `~/nofx-analysis/level-replay/{build_inputs,replay,bias_table}.py`.

---

# PART 2 — the 1h variant (daily-level kinds on the coarse tape)

## 2.0 PRE-REGISTRATION — written 2026-09-02 ~19:20 CT, BEFORE any 1h replay ran

Owner-directed follow-up (quoted intent): the deep pantry is daily/weekly; 1h
goes back ~four-five months — a coarser replay of PDH/PDL/PDC/ONH/ONL at 1h is
the only route to a verdict before October. **It is a coarser instrument and is
stated as such everywhere.**

- **Tape:** MNQ 1h, read-only query `SELECT open_time_ms,o,h,l,c,v FROM bars WHERE
  symbol='MNQ' AND tf='1h'` → **2,000 bars, 2026-05-04 06:00 CT → 2026-09-02 17:00 CT**
  (~4 months, not five — quoted). Completed session days (17:00 CT first bar,
  ≥15:00 last bar, ≥20 bars): computed from the actual list; OOS split **first
  60% / last 40% of that ordered list, fixed before looking**.
- **Levels:** the five daily kinds ONLY — PDH / PDL / PDC / ONH / ONL — rebuilt
  from 1h bars (coarser than the 1m reconstruction: intra-hour extremes are
  hidden; stated). Definitions mirror Part 1. Scan-from 17:00 for all five.
- **Instrument:** D1′ (`detectors.detect_symmetric_v2`) on 1h bars, `exit_on='close'`.
  **H = 6 bars** (6 hours) primary; H=12 reported as sensitivity only.
- **k re-calibration (pre-registered, the §0 procedure):** IID-shuffle the 1h bar
  shapes (same `iid_shuffle`), run k ∈ {0.5, 1, 1.5, 2, 3, 4, 6} on the REAL 1h
  level map; **PASS iff some k yields p(hold) within 0.50 ± 0.03**; the chosen k
  is the one closest to 0.500, ties to larger k. If NO k passes, the instrument
  is declared structurally biased at 1h and the real replay reports the
  calibration curve with NO verdicts.
- **Ambiguity:** ambiguous_span / ambiguous_horizon recorded and EXCLUDED from
  the denominator; the ambiguous SHARE is quoted per kind and period (at 1h a
  single bar spans both barriers far more often — expected).
- **Verdict rule (identical to Part 1):** BEATS iff holdout n ≥ 200 AND Wilson
  lower bound > that kind's stationary-bootstrap holdout baseline AND survives
  BH q=0.05; DOES NOT if the interval overlaps; TOO FEW below n=200.
  Baselines: stationary bootstrap on 1h shapes (mean block 10, 40 reps/day,
  the real per-day level lines) + Osler random-level percentile (B=1000/day).
- **Strata:** ordinal 1st / 2nd / 3rd+ per level per day; session windows
  (ASIA 17:00–02:00 / LONDON 02:00–08:30 / NY 08:30–14:45 CT).
- **Hypotheses:** H6 pooled holdout does not clear the 1h baseline · H7 PDL again
  shows no hold edge · H8 ordinal again fails to decline (consumed touch) ·
  H9 ONH/ONL indistinguishable from PDH/PDL.
- **Grader:** the Part-1 ruling (nothing changes) stands UNLESS a kind BEATS on
  holdout here; then that kind alone is proposed KEEP-with-evidence, no other
  ladder term moves.

**Nothing below this line was known when the criteria above were written.**


---

# PART 2 RESULTS (computed after the §2.0 pre-registration)

## 2.1 Calibration (E1′ §0 rule on the 1h tape)

Tape: 2,001 1h bars, 2026-05-04 → 2026-09-02, **84 completed session days** (05-05 … 09-02) → exploration 50d (05-05 … 07-16) / holdout 34d (07-17 … 09-02).
Δ(1h) = **56.0084 pt**. IID-shuffle calibration (20 reps/day, real 1h level map):

| k | band (pt) | p(hold) | Wilson | n | amb% |
|---|---|---|---|---|---|
| 0.5 | 28.00 | 0.4989 | [0.4888,0.5090] | 9427 | 6.9% |
| 1 | 56.01 | 0.4917 | [0.4809,0.5025] | 8217 | 12.2% |
| 1.5 | 84.01 | 0.4743 | [0.4623,0.4863] | 6665 | 22.2% |
| 2 | 112.02 | 0.4919 | [0.4782,0.5056] | 5121 | 35.0% |
| 3 | 168.03 | 0.4690 | [0.4517,0.4863] | 3192 | 56.4% |
| 4 | 224.03 | 0.4684 | [0.4475,0.4894] | 2167 | 69.9% |
| 6 | 336.05 | 0.4847 | [0.4526,0.5171] | 918 | 87.0% |

**PASS set:** k=0.5 (0.4989), k=1 (0.4917), k=2 (0.4919). Chosen by the §0 rule: **k=0.5, band ±28.00 pt**
(closest to 0.500, ties → larger k). The coin-flip null at 1h is p_null=0.4989. Higher k collapses into ambiguity
(k=6 → 87% ambiguous). H=6 primary; sensitivity: H=12 identical (0.6389, n=216 — episodes never reach the horizon),
H=3 → 0.6280 [0.5604,0.6910] n=207 amb 8.8%. Stable.

## 2.2 Real replay — per kind (holdout verdicts only; floor n=200)

| kind | exp n | exp p | hold n | hold p(hold) | hold Wilson | amb% | boot p | Osler null | verdict |
|---|---|---|---|---|---|---|---|---|---|
| ONL | 85 | 0.6824 | 54 | 0.6852 | [0.5526,0.7932] | 8.5% | 0.5540 | 0.5531 | TOO FEW |
| ONH | 81 | 0.6667 | 53 | 0.7547 | [0.6243,0.8507] | 1.9% | 0.5449 | 0.5540 | TOO FEW |
| PDC | 69 | 0.4348 | 45 | 0.5333 | [0.3908,0.6707] | 6.2% | 0.5515 | 0.5536 | TOO FEW |
| PDL | 46 | 0.4565 | 36 | 0.4444 | [0.2954,0.6042] | 2.7% | 0.5258 | 0.5560 | TOO FEW |
| PDH | 54 | 0.4630 | 28 | 0.7500 | [0.5664,0.8732] | 3.4% | 0.5319 | 0.5539 | TOO FEW |

**Pooled holdout: p(hold) = 0.6389 [0.5729,0.7000], n=216 (amb 4.8%).**
Every per-kind cell is below the n=200 floor (largest ONL n=54) → **no kind ranks; TOO FEW everywhere.**

**The owner's n-expectation did not hold — measured, not estimated:** 582 episodes over 84 session days.
At 1h the D1′ touch-open events are rare (a 1h bar must straddle the level after a non-straddling bar)
and each episode consumes up to H bars. The n=200 floor needs ~10× more 1h history — the 1h retention
(4,000 rows ≈ 167 days live, growing) reaches it in months, not weeks.

## 2.3 Strata (descriptive only)

Ordinal (holdout): 1st **0.688** (n=125) · 2nd 0.571 (n=63) · 3rd+ 0.571 (n=28).
**H8: this time the ordinal DECLINES (1st > 2nd ≈ 3rd+)** — the opposite direction of the 1m replay, where 1st
was also highest but 3rd+ sat above 2nd. Both results are n-tiny; the belief has no stable direction on a
calibrated instrument. Session (holdout): ASIA 0.662 (n=68) · LONDON 0.692 (n=52) · NY 0.594 (n=96) — all above
0.50, all overlapping.

## 2.4 Nulls and the one number that stands out

- Per-kind stationary bootstrap (mean block 10, 40 reps/holdout day, the REAL level lines): 0.526–0.554.
- Osler random-level null (B=1000/holdout day): per-kind means 0.554–0.556; the POOLED null has mean 0.5453
  with sd 0.2216 (p05 0.167 / p95 0.889 — tiny per-day samples make the null extremely wide).
- The real pooled holdout 0.6389 sits at the **65.6th percentile of the Osler null — inside the bulk**.
  The pooled interval does not include 0.50 and clears the per-kind bootstrap means, but the Osler spread says
  that with n=216 the same number would arise from randomly-placed prices one day in three. **No verdict.**

## 2.5 Verdicts per kind and grader

- **PDH / PDL / PDC / ONH / ONL: all TOO FEW** (holdout n 28–54). ONH 0.7547 and PDH
  0.7500 are descriptively high vs their bootstrap nulls (~0.53–0.55) but are exactly the size of cell that
  multiplicity + tiny n produce by accident (Part 1 made this point at n=33; it stands at n=54).
- **H7:** PDL again shows no hold edge (0.4444, n=36).
- **H9:** ONH/ONL (0.69–0.75) descriptively above PDH/PDL/PDC (0.44–0.75, wide) — unrankable.
- **Grader ruling stands: nothing changes; every ladder term keeps its [I] label.** The instrument exists at
  two granularities now; the tape at both is still too short. The honest lever is time (1m retention 90d +
  1h retention both live), not a new threshold.

## 2.6 CSVs + scripts

`exports/2026-09-02-level-replay/episodes_1h.csv` (582 episodes) · `per_kind_1h.csv` · `calibration_1h.csv`.
Scripts: `~/nofx-analysis/level-replay/replay_1h.py` (+ `replay.py` Part 1).
