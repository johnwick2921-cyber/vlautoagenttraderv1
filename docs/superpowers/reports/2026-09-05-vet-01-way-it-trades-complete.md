# Section 1 — The way it trades: complete corrected audit

**One-page summary — 2026-09-05.** I judge the historical method **UNVERIFIED for a positive edge** and the enforced strict method **EVENT-WAIT: no realized entry after its first enforced boot**. I would retain one-contract SIM research. I do not claim professional trading experience: every [I] below is my analytical judgment, untested here. [T] denotes this tape; [R] denotes the specifically cited external study, with its transfer limits. [A] is directly inspected code or source evidence.

I see a book organized around price levels, predominantly short, whose largest executed family is range rejection. It is not a single stable “resting fade with a 1.5×ATR stop” experiment. The corrected population is **58 closed trades, 18 wins / 38 losses / 2 flats, −$466.428572 total, −$8.04 per trade**, across **12 occupied CME days beginning 17:00 CT**, or 4.83 trades per occupied day. Decided-trade wins are **18/56, 32.1%, Wilson 95% [21.4%, 45.2%]**. Average win is $125.47, average loss −$71.71, payoff **1.750**. Recorded fees are zero, so this is not measured net live expectancy. A day-block bootstrap gives **[−$35.33, +$21.33]** per trade; it does not demonstrate positive expectancy. Full IDs and exclusions: D/results.txt:2; day membership and bootstrap definitions: D/audit.py, D/trade_stats.csv. [T]

**My three biggest problems:**

1. **The history and current method are different books.** There are 55 eligible pre-0B trades, three after 0B but before strict, and zero after strict. A long continuation idea can exist in the narrative without an executable arm. The old report wrongly used this mixed history to pronounce the current shape impossible. [T/A; Q1, Q3]
2. **The reward thesis is less tested than its geometry.** 119 of 132 authored targets already coincide with a plan level, but 104 lie beyond a nearer plan level. A level-based target can still demand an untested journey through intervening structure. Nine trades have auditable broker initial risk; the other 49 have no defensible R denominator in this extraction. [T; Q4]
3. **The apparent edge evidence is selected and thin.** Reject is +$586 over 31 trades, with no demonstrated positive expectation; each session and volatility tercile is below 30. Touches are 677 stored rows but 423 price-time keys, selected retrospectively with formation leakage. None licenses a session ban, level flip, or fitted exit. [T; Q2]

**My three biggest opportunities:**

1. Define range rejection and continuation as separate executable theses, each with its own invalidation, target rationale and eligible opportunities. The result may be “no trade,” not a fabricated distant target. [I]
2. Test the target path against the first intervening level and remaining session time, using targets fixed before entry. Compare a map target with a fixed-R baseline; keep MFE as a diagnostic. [I]
3. Build a prospective, frozen-policy SIM sample with initial broker brackets, formation-time level identity and ordered excursions. Use it to test progress/time exits and regime selection before changing them. [I]

## Evidence contract and population repair

I own **Section 1 only**. Parent owns Section 9 integration. Branch `docs/vet-01-0905-complete` was claimed with `NOFX_SESSION=codex-vet-01-complete-0905` and `deploy/nofx-claim.sh new`; fresh worktree `/home/hoang/nofx-vet-01-complete` was detached from origin/dev **b4376246** before claiming. Scratch is exclusively `/home/hoang/nofx-analysis/vet-01-complete-0905`. The original full ten-section dispatch was read from the supplied attachment; this report answers every Section 1 question and subpart.

**This report replaces the old Section 1 prose and all of its recommendations.** Its old data folder remains a historical audit trail, not an active source. In particular, the old 65-row statistics, 14-calendar-day count, 370-touch primary identity, fitted MFE p60 target, ATR-as-ceiling proposal, impossibility claim and invented biography are superseded. I have not changed any trading code, configuration, database, runtime, prompt or order. No production token helper was run. After the parent update I also checked origin/dev **23b9f99e**: the relevant kernel/trader/store/provider/NinjaScript code is unchanged from b4376246. The instructed origin/dev base takes precedence over the checklist's running-revision worktree convention; code claims below name this base and do not imply deployed settings.

**D** in a citation expands to `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-complete-data/`. Every aggregate CSV row carries all of its position IDs or touch-key IDs; every touch key lists all contributing raw row IDs. Plans are identified by SQLite rowid plus `(plan_id, version, scenario)`. Bars retain rowid and `(symbol, tf, open_time_ms, convention)`. `inputs.json.gz` preserves the selected raw inputs and extraction SQL, including broker record file:line; `audit.py` and `supplement.py` reproduce the outputs offline with Python's standard library. Extraction as-of **2026-09-05T17:38:15.087919-05:00** used `mode=ro`, `PRAGMA query_only=ON`, and one read transaction. D/source_evidence.txt preserves code and boot lines; unauthenticated `/api/health` returned HTTP 200, rev `36648655cfe0`. It is behind the review base. [A]

The primary rule is entry time ≥ **1786770000000**, 2026-08-15 00:00 CT; exclude `source='e7_farside_test'`, `plan_id='UNRESOLVABLE'`, and NULL `pnl_corrected`. Of 71 era rows: test IDs **572–574**; non-test sentinel IDs **530, 539, 545, 546, 566, 571, 580**; non-test null-corrected IDs **576, 577, 579**. These reasons overlap: test row 572 has NULL P&L, and test rows 573–574 also carry the sentinel plan ID. The union removes 13 rows. The seven sentinel rows contribute −$97.50, explaining the difference between the old −$563.93 and corrected −$466.43. I never use `realized_pnl` as a replacement. D/excluded.csv:2; D/results.txt:2. [T]

I use W/(W+L), excluding flats only from win rates. Means retain flats. Wilson intervals describe proportions, not dollar means. They assume independent Bernoulli observations and do not cure within-day dependence, repeated levels, hindsight selection or multiple comparisons. Every cell below **n=30 receives NO VERDICT**, including all R subsets; cells above 30 still need valid sampling and an expectation interval. The whole-book naive normal interval is [−$35.04, +$18.95]; the primary sensitivity resamples the 12 occupied CME days 20,000 times, seed 90501, and recomputes sum P&L / trade count. These are only 12 observed day blocks, with no zero-trade days or out-of-sample regime coverage manufactured. D/summary.json; D/audit.py. [T]

## 1. The book I would brief to a risk manager

I would describe the **intended range trade** as buying a test of support or selling a test of resistance, expecting rejection back into the range. `reject` expresses this directly. `sweep_reclaim` requires a failed break and return through the level; I keep it separate from a plain first-touch fade. The **continuation candidate** expects acceptance/reclaim on the new side, or a pullback to a broken level followed by renewed movement. `reclaim` is context-dependent and is not automatically a pure breakout; I group reclaim/acceptance/breakout_retest only as a declared descriptive “continuation candidate” sensitivity. I do not pool it with sweep-reclaim just because the names overlap. [I; source condition rows D/trades.csv]

In the current code, reject is a limit at a level; sweep-reclaim is a confirmed/retrace limit; breakdown/breakup continuation arms are pullback limits chained on confirmation; reclaim maps to stop-entry beyond its trigger. Acceptance/hold have no arm kind, and breakout_retest/fvg_entry are shadowed by default. Under strict, the decision loop cannot open, so a scenario without a legal arm supplies a narrative, not exposure. This book fits a revisit/rejection or failed-pullback opportunity set. An uninterrupted move that never revisits its chosen price can leave it flat; that is a structural participation limit, not proof of lost achievable P&L. [A/I; `kernel/arm_kind.go:36`, `kernel/armed.go:17`, `kernel/plan_doc.go:130`, `kernel/condition_status.go:26`, `trader/entry_gate.go:184`]

**When/how often.** The bound trader's saved scan interval is two minutes and saved plan mode strict. Scheduled default reads are ASIA 16:30, LONDON 01:30 and NY 08:00 CT. Their default trading windows are 17:00–02:00, 02:00–08:30 and 08:30–14:45; corresponding flats are 02:00, 08:30 and 14:45. The system is not simply “flat at 14:45” for every session. Saved `sessions_enabled=[NY]` coexists with per-session ASIA/LONDON enables; I show actual plan-session attribution rather than inferring NY-only from one field. First/last eligible ledger entries are Aug 19 03:22:03 CT and Sep 3 09:05:14 CT. All 58 have entry quantity one and close reason `sync`; exit cause is not recoverable from that field. [T/A; D/binding.txt:3; `kernel/session_registry.go:83`; `trader/auto_trader_clock.go:476`; D/trades.csv]

**Owner-policy context.** The August 17 correction explicitly records `guardrails_enabled:false` as the owner's deliberate decision (`docs/superpowers/reports/2026-08-17-cto-final-verification.md:13`). I treat this as a dated SIM-learning policy choice, not a code defect or proof of today's resolved value. It is distinct from the separately captured daily-loss flag. I do not recommend reversing that policy as a bug fix. [A]

**Stop/target.** The composer widens to the farthest of **authored stop, nearest seated risk-side anchor plus clearance, and the ATR floor**; the dispatch omitted the authored-stop floor. Anchor search defaults to 3×ATR and the stop floor to 1.5×ATR5m. An “unanchored” counter alone cannot prove which other floor bound. The saved R:R minimum is 2.0; the shared gate tests entry/stop/target geometry at its execution-price input. `arm.target` is the executable bracket target; `target_chain` is narrative guidance. BE and trail are suspended by default and the inspected boots report both off. One contract gives one position-level exit path at a time; a scale-out would require a different size/exposure design. [A; `trader/arm_stop_anchor.go:20`, `trader/arm_stop_anchor.go:138`, `trader/entry_gate.go:259`, `kernel/plan_doc.go:104`, `kernel/planner_prompt.go:722`, `trader/exit_mechs_suspend.go:33`; D/binding.txt:4]

The exact epochs matter. First inspected 0B boot: **Sep 2 07:49:06 CT**, rev **4175e0b62de7**, epoch **1788353346000** (`data/nofx_2026-09-02.log:16710`, `:16755`). First enforced-strict boot: **Sep 3 11:10:33 CT**, rev **f478ed880dc9**, epoch **1788451833000** (`data/nofx_2026-09-03.log:3658`). The gate is independently present in that revision, lines 124–126, preserved in D/source_evidence.txt. The strict implementation's commit identity must not be substituted for the boot revision.

| Cell | n; W/L/F | Corrected $ sum | Mean $ | Win/decided; Wilson 95% | R n; mean | Evidence |
|---|---:|---:|---:|---|---|---|
| 0B_pre_strict | 3; 0/3/0 | -394.00 | -131.33 | 0/3; 0.0% [0.0, 56.2] | 1; -1.00 | D/trade_stats.csv:46 |
| pre_0B | 55; 18/35/2 | -72.43 | -1.32 | 18/53; 34.0% [22.7, 47.4] | 8; 0.40 | D/trade_stats.csv:47 |
| strict | 0; 0/0/0 | 0.00 | — | 0/0; — (0 decided) | 0; — | D/trade_stats.csv:72 |


Post-0B trades are **589, 590, 591**, all losses, −$394; two are historical decision-path breakout-retests, one an arm-associated reject. This is not a clean test of 0B arms. **No strict-era realized trades; NO VERDICT**, not a 0% win rate. [T]

| Cell | n; W/L/F | Corrected $ sum | Mean $ | Win/decided; Wilson 95% | R n; mean | Evidence |
|---|---:|---:|---:|---|---|---|
| acceptance | 6; 1/4/1 | 4.57 | 0.76 | 1/5; 20.0% [3.6, 62.4] | 0; — | D/trade_stats.csv:3 |
| breakout_retest | 9; 1/8/0 | -581.50 | -64.61 | 1/9; 11.1% [2.0, 43.5] | 0; — | D/trade_stats.csv:4 |
| hold | 1; 1/0/0 | 168.00 | 168.00 | 1/1; 100.0% [20.7, 100.0] | 0; — | D/trade_stats.csv:5 |
| reclaim | 5; 0/5/0 | -436.50 | -87.30 | 0/5; 0.0% [-0.0, 43.4] | 0; — | D/trade_stats.csv:6 |
| reject | 31; 14/16/1 | 586.00 | 18.90 | 14/30; 46.7% [30.2, 63.9] | 6; 0.55 | D/trade_stats.csv:7 |
| sweep_reclaim | 6; 1/5/0 | -207.00 | -34.50 | 1/6; 16.7% [3.0, 56.4] | 3; -0.36 | D/trade_stats.csv:8 |


No eligible fvg_entry, breakdown_continue or breakup_continue trades exist in this population: n=0, NO VERDICT. Reject supplies 31 of the 58 records and +$586, mean +$18.90 with a descriptive naive normal 95% mean interval **[−$18.26, +$56.06]** (D/trade_stats.csv condition=reject); its positive point estimate is not established expectancy. Acceptance and hold also have positive sums, so the old “everything except reject loses” sentence was false. The explicitly defined continuation-candidate set is 20 trades, 2W/17L/1F, −$1,013.43; it is below 30 and cannot establish that continuation fails as a method. All nine R-bearing rows are selected by broker evidence availability; their condition means are not complete-cell estimates. [T; D/trade_stats.csv thesis rows]

| Cell | n; W/L/F | Corrected $ sum | Mean $ | Win/decided; Wilson 95% | R n; mean | Evidence |
|---|---:|---:|---:|---|---|---|
| LONG | 19; 4/15/0 | -808.50 | -42.55 | 4/19; 21.1% [8.5, 43.3] | 3; 0.06 | D/trade_stats.csv:9 |
| SHORT | 39; 14/23/2 | 342.07 | 8.77 | 14/37; 37.8% [24.1, 53.9] | 6; 0.34 | D/trade_stats.csv:10 |


| Cell | n; W/L/F | Corrected $ sum | Mean $ | Win/decided; Wilson 95% | R n; mean | Evidence |
|---|---:|---:|---:|---|---|---|
| ASIA | 16; 2/13/1 | -552.43 | -34.53 | 2/15; 13.3% [3.7, 37.9] | 0; — | D/trade_stats.csv:11 |
| LONDON | 21; 7/14/0 | 24.00 | 1.14 | 7/21; 33.3% [17.2, 54.6] | 3; 1.23 | D/trade_stats.csv:12 |
| NY | 21; 9/11/1 | 62.00 | 2.95 | 9/20; 45.0% [25.8, 65.8] | 6; -0.24 | D/trade_stats.csv:13 |


| Cell | n; W/L/F | Corrected $ sum | Mean $ | Win/decided; Wilson 95% | R n; mean | Evidence |
|---|---:|---:|---:|---|---|---|
| arm-associated | 11; 6/5/0 | 94.50 | 8.59 | 6/11; 54.5% [28.0, 78.7] | 9; 0.25 | D/trade_stats.csv:40 |
| decision | 47; 12/33/2 | -560.93 | -11.93 | 12/45; 26.7% [16.0, 41.0] | 0; — | D/trade_stats.csv:41 |


The historical decision path accounts for 47 trades, not the previous 51. I cannot compare the 11 arm-associated trades with the decision trades as if execution mode had been randomized. `source=reconcile` alone is not proof of a particular entry mechanism; “arm-associated” is the historical source/lineage grouping, while nine exact broker signal chains are verified separately. [T]

Median hold is **25.64 min**, interquartile range **12.28–56.30**, maximum **219.71**. Winners' median is **43.04 min** (18 IDs in D/supplement.txt); losers' **19.74 min** (38 IDs). These are ledger entry-to-exit durations. For 591, broker fill precedes ledger materialization, so they are not exact market-exposure clocks. [T]

| Cell | n; W/L/F | Corrected $ sum | Mean $ | Win/decided; Wilson 95% | R n; mean | Evidence |
|---|---:|---:|---:|---|---|---|
| 0-15 | 19; 5/14/0 | -317.50 | -16.71 | 5/19; 26.3% [11.8, 48.8] | 4; -0.20 | D/trade_stats.csv:73 |
| 15-30 | 12; 1/10/1 | -872.00 | -72.67 | 1/11; 9.1% [1.6, 37.7] | 2; -0.31 | D/trade_stats.csv:74 |
| 30-60 | 13; 5/7/1 | 184.57 | 14.20 | 5/12; 41.7% [19.3, 68.0] | 2; 1.70 | D/trade_stats.csv:75 |
| 60+ | 14; 7/7/0 | 538.50 | 38.46 | 7/14; 50.0% [26.8, 73.2] | 1; 0.26 | D/trade_stats.csv:76 |


**Realized R: partially measurable, not a 58-row distribution.** I use `pnl_corrected / (abs(actual entry fill − first accepted protective stop) × $2 × entry_quantity)`, after matching exact signal identity, one-unit entry fill/price, and an accepted stop within ten seconds of that broker fill. I do not use a nearby decision, final arm stop, or loss/exit distance as initial risk. Nine broker-linked rows qualify: **568, 570, 575, 578, 582, 584, 585, 586, 591**. Mean **+0.248R**, median **+0.256R**, p25 **−1.000R**, p75 **+0.925R**, n=9: **NO VERDICT**. Their individual R values and source file:line are in D/broker_R.csv:2 and D/trades.csv. The other 49 IDs are explicitly listed under `R_missing_ids` in D/summary.json. Missing inputs: immutable entry signal/OCO identity and first accepted protective bracket, including any immediate modification; `trade_excursions` supplies zero records. I do not report the observed nine as representative. [T]

Plan attribution itself is partly retrospective: **568's cited version was created after its ledger entry**. I retain it because the mandated eligible population includes it, but its scenario cannot prove what was known at entry. Its broker stop still independently supports a dollar-risk denominator. Exact timestamps and plan row ID are in D/trades.csv. [T]

## 2. Where, if anywhere, is the edge?

### 2(a) Fades: hold by level kind × ordinal

I find **677 raw rows → 423 exact `(trader, symbol, kind, price, opened_at_ms)` keys**, not 370: the latter collapses different prices observed at the same kind/time. I retain disagreements as ambiguous rather than taking a majority vote. Conflicts are key representatives **167 and 180**, with all raw members in D/touch_keys.csv. Stored ordinals are not all one: 471 rows have ordinal one and the remaining rows span 2–15. [T; D/summary.json; D/touch_keys.csv]

The main key-level description is **170 HOLD / 125 BREAK / 128 ambiguous**, 295 decided keys. HOLD **57.6% [51.9%, 63.1%]**, BREAK **42.4% [36.9%, 48.1%]**. Ambiguous share is reported with its Wilson interval in D/touch_stats.csv:2. These are retrospective detector classifications, not executed fade wins or independent draws from a trade distribution. [T]

For ordinal sensitivity I sort exact kind/price keys chronologically, reset at the 17:00 CT day, and label first/second/third-or-later **observed in this corpus**. This is left-censored, particularly for a moving VWAP whose price identity changes. It does not recover lifetime first touch. The full kind × ordinal matrix (20 kinds × 3 buckets, including empty cells) and the separate raw kind × stored-ordinal matrix are in **D/touch_stats.csv**, with HOLD/BREAK Wilson bounds, ambiguous counts/intervals and every key ID. I do not recompute one ordinal across an entire kind as the old report did.

| Reconstructed observed ordinal | HOLD / BREAK / ambiguous | Decided n | HOLD; Wilson 95% | BREAK; Wilson 95% |
|---|---:|---:|---|---|
| 1 | 29 / 21 / 26 | 50 | 58.0% [44.2, 70.6] | 42.0% [29.4, 55.8] |
| 2 | 19 / 14 / 32 | 33 | 57.6% [40.8, 72.8] | 42.4% [27.2, 59.2] |
| 3+ | 122 / 90 / 70 | 212 | 57.5% [50.8, 64.0] | 42.5% [36.0, 49.2] |

All three ordinal descriptions are about 58% HOLD. I withdraw the old assertion that this corpus establishes a first-touch advantage. No ordinal receives a trading edge verdict: formation/availability and selection remain unresolved even where pooled n exceeds 30. [T/I]

### 2(b) Breaks: the same observations inverted

| Kind | HOLD / BREAK / ambiguous | Decided n | HOLD; Wilson 95% | BREAK; Wilson 95% |
|---|---:|---:|---|---|
| DEMAND | 8 / 8 / 2 | 16 | 50.0% [28.0, 72.0] | 50.0% [28.0, 72.0] |
| EQL | 2 / 0 / 0 | 2 | 100.0% [34.2, 100.0] | 0.0% [0.0, 65.8] |
| FVG | 0 / 1 / 0 | 1 | 0.0% [0.0, 79.3] | 100.0% [20.7, 100.0] |
| OB | 5 / 2 / 4 | 7 | 71.4% [35.9, 91.8] | 28.6% [8.2, 64.1] |
| ONH | 3 / 0 / 0 | 3 | 100.0% [43.8, 100.0] | 0.0% [0.0, 56.2] |
| ONL | 3 / 1 / 0 | 4 | 75.0% [30.1, 95.4] | 25.0% [4.6, 69.9] |
| OR-H | 9 / 9 / 4 | 18 | 50.0% [29.0, 71.0] | 50.0% [29.0, 71.0] |
| OR-L | 4 / 4 / 4 | 8 | 50.0% [21.5, 78.5] | 50.0% [21.5, 78.5] |
| PDC | 5 / 6 / 1 | 11 | 45.5% [21.3, 72.0] | 54.5% [28.0, 78.7] |
| PDH | 4 / 5 / 1 | 9 | 44.4% [18.9, 73.3] | 55.6% [26.7, 81.1] |
| PDL | 2 / 0 / 0 | 2 | 100.0% [34.2, 100.0] | 0.0% [0.0, 65.8] |
| POC | 9 / 5 / 6 | 14 | 64.3% [38.8, 83.7] | 35.7% [16.3, 61.2] |
| RTH-H | 7 / 3 / 2 | 10 | 70.0% [39.7, 89.2] | 30.0% [10.8, 60.3] |
| RTH-L | 4 / 8 / 2 | 12 | 33.3% [13.8, 60.9] | 66.7% [39.1, 86.2] |
| SUPPLY | 7 / 7 / 8 | 14 | 50.0% [26.8, 73.2] | 50.0% [26.8, 73.2] |
| SWG-H | 9 / 6 / 14 | 15 | 60.0% [35.7, 80.2] | 40.0% [19.8, 64.3] |
| SWG-L | 4 / 4 / 8 | 8 | 50.0% [21.5, 78.5] | 50.0% [21.5, 78.5] |
| VWAP | 82 / 50 / 67 | 132 | 62.1% [53.6, 69.9] | 37.9% [30.1, 46.4] |
| VWAP±2σ | 2 / 2 / 0 | 4 | 50.0% [15.0, 85.0] | 50.0% [15.0, 85.0] |
| eVWAP | 1 / 4 / 5 | 5 | 20.0% [3.6, 62.4] | 80.0% [37.6, 96.4] |

Every kind except VWAP has decided n<30: **NO VERDICT**. VWAP has n=132 decided but is retrospective and selected, so it also has **no actionable edge verdict**. A detector BREAK means price crossed its band according to the detector rule; it does not mean a stop-entry would fill and reach a profit target before a stop. It is not an executable complement strategy. [T/I; D/touch_stats.csv]

**RTH-L specifically:** 140 raw rows are **14 price-time keys**, all at 29199.25. Conservative conflict handling yields **4 HOLD / 8 BREAK / 2 ambiguous**, 12 decided; BREAK **8/12, 66.7%, Wilson [39.1%, 86.2%]**. The conflicting key 180 is not silently converted to BREAK by majority. The old 9/13 and raw 43/63 do not justify a flip. The raw RTH-L observations are attached to September 4 plan reads, yet **129 raw rows opened before September 3 08:30 CT**—before the prior NY session even began. All IDs are in D/supplement.txt `rth_formation`. That is direct evidence of looking backward with a subsequently identified level. `kernel/levels_multiday.go:92` derives the NY extrema; `:154` selects the prior calendar day; `trader/detector_record.go:57` runs the resulting seated price over the historical scope. Deduplication alone cannot repair this. [T/A]

Only **57 of 423 representative keys** even have their attached plan version timestamp at/before touch; some referenced versions are missing. This is a plan-availability sensitivity, **not** a formation-safe subset: prior-day levels may have existed earlier than their plan, whereas dynamic levels can be generated later. Exact formation/read availability, source-bar endpoint, stable level lineage, band at touch and candidate eligibility are missing. Thus a formation-safe hold/break edge is **UNMEASURABLE**, with those exact required inputs. I recommend no promotion, retirement or flipping of a kind from this table. [T/I]

### 2(c) Time of day: session and every occupied CT hour

Q1 gives plan-session distributions and separate R sample sizes. Actual entry-hour distributions follow; plan session and wall-clock session are different labels. All hour cells have n<30: **NO VERDICT**. Empty hours 14, 15 and 16 have n=0, also no verdict; they are not assumed losing or winning opportunities.

| Cell | n; W/L/F | Corrected $ sum | Mean $ | Win/decided; Wilson 95% | R n; mean | Evidence |
|---|---:|---:|---:|---|---|---|
| 0 | 2; 0/2/0 | -142.00 | -71.00 | 0/2; 0.0% [0.0, 65.8] | 0; — | D/trade_stats.csv:14 |
| 1 | 2; 1/1/0 | -11.50 | -5.75 | 1/2; 50.0% [9.5, 90.5] | 0; — | D/trade_stats.csv:15 |
| 10 | 3; 0/3/0 | -249.00 | -83.00 | 0/3; 0.0% [0.0, 56.2] | 0; — | D/trade_stats.csv:16 |
| 11 | 2; 2/0/0 | 443.00 | 221.50 | 2/2; 100.0% [34.2, 100.0] | 0; — | D/trade_stats.csv:17 |
| 12 | 2; 2/0/0 | 109.00 | 54.50 | 2/2; 100.0% [34.2, 100.0] | 2; 0.59 | D/trade_stats.csv:18 |
| 13 | 2; 1/1/0 | 27.00 | 13.50 | 1/2; 50.0% [9.5, 90.5] | 1; -1.00 | D/trade_stats.csv:19 |
| 17 | 1; 0/1/0 | -54.50 | -54.50 | 0/1; 0.0% [0.0, 79.3] | 0; — | D/trade_stats.csv:20 |
| 18 | 1; 0/1/0 | -43.00 | -43.00 | 0/1; 0.0% [0.0, 79.3] | 0; — | D/trade_stats.csv:21 |
| 19 | 2; 0/2/0 | -118.00 | -59.00 | 0/2; 0.0% [0.0, 65.8] | 0; — | D/trade_stats.csv:22 |
| 2 | 3; 1/2/0 | 30.00 | 10.00 | 1/3; 33.3% [6.1, 79.2] | 0; — | D/trade_stats.csv:23 |
| 20 | 2; 0/2/0 | -153.93 | -76.96 | 0/2; 0.0% [0.0, 65.8] | 0; — | D/trade_stats.csv:24 |
| 21 | 3; 1/1/1 | 93.00 | 31.00 | 1/2; 50.0% [9.5, 90.5] | 0; — | D/trade_stats.csv:25 |
| 22 | 2; 0/2/0 | -95.50 | -47.75 | 0/2; 0.0% [0.0, 65.8] | 0; — | D/trade_stats.csv:26 |
| 23 | 1; 0/1/0 | -27.00 | -27.00 | 0/1; 0.0% [0.0, 79.3] | 0; — | D/trade_stats.csv:27 |
| 3 | 2; 1/1/0 | 40.50 | 20.25 | 1/2; 50.0% [9.5, 90.5] | 0; — | D/trade_stats.csv:28 |
| 4 | 2; 1/1/0 | 99.00 | 49.50 | 1/2; 50.0% [9.5, 90.5] | 0; — | D/trade_stats.csv:29 |
| 5 | 4; 2/2/0 | 137.00 | 34.25 | 2/4; 50.0% [15.0, 85.0] | 1; 2.47 | D/trade_stats.csv:30 |
| 6 | 5; 1/4/0 | -208.50 | -41.70 | 1/5; 20.0% [3.6, 62.4] | 0; — | D/trade_stats.csv:31 |
| 7 | 4; 0/4/0 | -180.00 | -45.00 | 0/4; 0.0% [0.0, 49.0] | 1; -1.00 | D/trade_stats.csv:32 |
| 8 | 4; 2/1/1 | 21.50 | 5.38 | 2/3; 66.7% [20.8, 93.9] | 1; 2.21 | D/trade_stats.csv:33 |
| 9 | 9; 3/6/0 | -183.50 | -20.39 | 3/9; 33.3% [12.1, 64.6] | 3; -0.54 | D/trade_stats.csv:34 |


The four records in 11:00–12:59 CT are winners, but neither hourly cell has even three observations. I will not select a “profitable hour.” The dollar table is complete; hour/session R is **UNMEASURABLE for the full populations**, with observed broker-linked subsets shown explicitly. Missingness is concentrated by time and era, not randomly sampled. [T/I]

### 2(d) Machine regime and realized-volatility terciles

**Exact machine-regime join:** all 32 `planner_read_facts` rows have blank `plan_id`, version zero and `bias_regime=up/NORMAL`. No eligible trade has a valid `(plan_id, version)` machine-fact join. The as-of same-session sensitivity allows a preceding read no more than 30 minutes old: it finds only **position 591 / fact 6**, `up/NORMAL`, −$140, −1R, n=1: **NO VERDICT**. This is not exact plan attribution. The plan's `bias_label` is absent for 57 eligible records; the prose `day_type` is not a substitute for a machine regime. Its descriptive cells, including separate `trend-down` and `trend_down` spellings, remain in D/trade_stats.csv. **Regime-conditioned R edge: UNMEASURABLE** without an immutable entry-linked machine label and adequate observations in multiple regimes. [T]

**Realized volatility is not ATR.** I compute a causal RV60 proxy, `10,000 × sqrt(sum(log(c[t]/c[t−1])²))`, from exactly 60 one-minute returns using 61 consecutive, fully closed pre-entry MNQ bars. It is basis points over the preceding hour, not annualized. Identical overlapping bar conventions collapse to one timestamp; conflicting OHLC timestamps would be excluded. Every contributing bar rowid is in D/trades.csv. Four trades fail the complete-window rule: **521, 522, 523, 533**. The other **54** split into **18/18/18**; in-sample cutpoints are **16.1562 and 24.0159 bps**. These cutpoints are a descriptive sort, not causal production thresholds: the measurements are pre-entry, but the quantiles use the whole sample. [T]

| Cell | n; W/L/F | Corrected $ sum | Mean $ | Win/decided; Wilson 95% | R n; mean | Evidence |
|---|---:|---:|---:|---|---|---|
| UNMEASURABLE | 4; 2/2/0 | 141.00 | 35.25 | 2/4; 50.0% [15.0, 85.0] | 0; — | D/trade_stats.csv:68 |
| high | 18; 6/11/1 | -288.00 | -16.00 | 6/17; 35.3% [17.3, 58.7] | 4; -0.34 | D/trade_stats.csv:69 |
| low | 18; 5/13/0 | -356.50 | -19.81 | 5/18; 27.8% [12.5, 50.9] | 2; 0.73 | D/trade_stats.csv:70 |
| mid | 18; 5/12/1 | 37.07 | 2.06 | 5/17; 29.4% [13.3, 53.1] | 3; 0.71 | D/trade_stats.csv:71 |


Every tercile is n=18: **NO VERDICT**. Low RV is −$356.50, middle +$37.07, high −$288.00. That does not establish a monotonic low-volatility failure, and it replaces the old low-ATR −$795.50 claim. Session × tercile cells and row identities are preserved in D/trade_stats.csv to expose confounding, not to search for a preferred cell. Complete-case R per tercile is again only the small broker-linked subset, never a 54-row risk-normalized result. I would need entry ATR and risk, a label frozen before entry, and future observations across independent days to test a regime switch. [T/I]

## 3. Is the one-contract/limit/ATR-floor/2R/flat shape viable?

**My answer is “plausible shape, unproved here,” not “cannot make money.”** There is no arithmetic theorem that one contract with a single target cannot have positive expectancy. At the historical payoff 1.750, the algebraic decided-trade break-even win rate is **36.37% before costs**, versus observed 32.14%; uncertainty covers both favorable and unfavorable expectations. This calculation holds the estimated payoff fixed and is not a forecast. A higher target can lower its hit probability; a lower target can reduce winner size and interrupt large winners. Neither can be evaluated from a target distance alone. [T/I; D/summary.json]

The actual instrument is **MNQ**, not the larger NQ contract: CME specifies $2 per index point and a 0.25-point tick ($0.50). The price thesis may concern the same index, but order-book fills and costs do not transfer automatically to NQ size or live MNQ. I withdraw the old assertion that SIM fade win rate is a proven upper bound: this extraction has no live paired fill sample or queue model. [A/I; [CME contract page](https://www.cmegroup.com/markets/equities/nasdaq/micro-e-mini-nasdaq-100.html)]

I would change the **shape of the experiment**, subject to later approval, in three ways:

1. **Separate two trade theses.** For a range fade, require an observed rejection/failed break, explicit invalidation of the defended area and a reachable return destination. For continuation, require an executable acceptance/reclaim or failed-pullback route, with the same deterministic risk checks. A long bias alone must not be mistaken for a long opportunity. Define an explicit no-retest policy; remaining flat can be a valid outcome. This changes the participation contract, not an ATR constant. Existing reclaim stop-entry and waterfall pullback support must be tested, not re-proposed as missing features. [I/A; Q1 code]
2. **Make target choice falsifiable before entry.** Record why the intended path can pass each intervening level, the expected horizon and remaining time to session flat. Compare a map-target policy against a fixed-R baseline under identical eligible entries and costs. If the independently justified reward cannot meet the retained R:R floor, omit the trade. Do not remove the feasibility rule or invent a farther target to admit it. [I; Q4]
3. **Define thesis expiry separately from the protective stop.** An invalidated range rejection and a continuation pullback that remains valid are different events; any progress/time exit needs an explicit trigger and an ordered prospective replay. I would test it without tightening the current floor or re-enabling trailing. One-contract scale-outs are unavailable, but a single target or a single mechanically managed exit remains possible. This does not justify two contracts. [I]

**Where it fits / where it fails as a method.** I would expect a revisit-oriented range fade to require two-sided movement and room back into the range; I would expect a continuation method to need acceptance beyond the old boundary and room to the next destination. These are hypotheses [I], not validated regime filters. The code demonstrably struggles to express exposure when its authored idea has no legal arm, and its pullback entry cannot participate if there is no pullback. The tape shows loss overall and weak continuation-candidate history, but it cannot determine whether the economic thesis, entry selection, target path, era changes or execution caused that result. Historical long/short and session differences are not grounds to make it permanently short or NY-only.

**External research, used narrowly.** [R] Carol Osler's *Support for Resistance* (2000) tested support/resistance published by six firms during 1996–98 against one-minute indicative FX quotes for the dollar/mark, dollar/yen and dollar/pound; it found predictive trend interruptions with variation across firms and currencies. That supports investigating known-in-advance levels, not a MNQ fade's net expectancy, a first-touch ordinal rule, an ATR stop multiplier or a target percentile. [Primary paper](https://www.newyorkfed.org/medialibrary/media/research/epr/00v06n2/0007osle.pdf). Her separate 2001 currency-order study discusses reversals and acceleration after crosses; it supplies an FX microstructure mechanism to investigate, not a test of this MNQ implementation. [Primary source](https://www.newyorkfed.org/research/staff_reports/sr125.html). I withdraw the old report's stronger “first tests consume orders” citation and its unsupported research-based prescriptions about trail rankings and late-day participation.

## 4. Reward: verify the three claims, then choose a source

| Original claim | Corrected verification | Status and exact limit |
|---|---|---|
| Plans claim median 2.55:1 | **Verified as 2.553571 over the original 17-scenario export**, not the all-era book. The fresh all-era median is **2.3564 over 132** enabled arms; an explicitly defined Sep 1–2 **CT creation-date** slice gives **2.5357 over 57** | Original source: `2026-09-05-veteran-part-a.md:103` and `exports/2026-09-02-losses/plans.jsonl`. D/legacy_plan_geometry.csv preserves the 17 identities; D/legacy_check.txt:3 verifies arithmetic. D/planned_arms.csv and D/supplement.txt identify fresh samples. Revisions are repeated intent, not independent ideas. |
| Book realizes 1.66:1 | **1.74975** = average corrected winner $125.4722 / average corrected loser $71.7086, 18 wins and 38 losses | Realized dollar payoff, **not mean realized R or a planned R:R**. Old 1.66 and Section 1's 1.62 are superseded. D/results.txt:2. |
| Only 3 of 36 reached 2R off minimum stop | **The old proxy arithmetic reproduces: 3/36, 8.3%, Wilson [2.9%, 21.8%]**, hits **555, 557, 581**. Its denominator includes null-corrected **576, 577, 579**. The eligible intersection is **3/33, 9.1%, [3.1%, 23.6%]**. A fresh causal closed-bar diagnostic gives **5/54, 9.3%, [4.0%, 19.9%]**, adding hits **524, 529** | The 36-row historical CSV and all memberships are preserved in D/legacy_floor_input.csv and D/legacy_check.txt:1. It is a proxy from a limited bar-coverage era, not a current-population target-hit upper bound. Actual initial floor and ordered target-before-stop probability remain **UNMEASURABLE**. Fresh diagnostic: D/floor_path_sensitivity.csv, D/supplement.txt:1. |

I verified the original 36-row membership in `2026-09-04-research-conformance-data/E-d3-mae-mfe-per-trade.csv`; it is not missing. The missing evidence is its claimed actual entry-floor and ordered-path interpretation. Its rounded floor/ATR columns differ from exactly 1.5 by at most 0.00003049, so the old claim of zero arithmetic residual also overstates precision. The three historical proxy hits do not include the earlier winners 524 and 529 that the fresh coverage recovers.

For the fresh floor diagnostic I restart at gaps and require at least 15 complete 5m blocks. The engine's Wilder formula is visible at `market/data_indicators.go:86`; its exact live input cache, partial-bar policy and accepted bracket are not reconstructed by matching the formula alone. The old “14 of 62 reached actual initial 2R” and “8 of 59 reached their own target” are withdrawn as initial-risk/ordered-path claims. None of these counts establishes the alternative P&L that a different stop or target would have produced.

**The target-provenance result changes my diagnosis.** Authored arm targets lie within one MNQ tick of a level in their own plan in **119/132, 90.2%, Wilson [83.9%, 94.2%]**; **120/132, 90.9% [84.8%, 94.7%]** match their target chain. **104/132, 78.8% [71.1%, 84.9%]** lie beyond the nearest directional plan level. All comparison identities and distances are in D/target_obstacles.csv; all plan levels are retained in D/inputs.json.gz. This is same-document geometric provenance, not independent validation that the level was machine-seated, known early enough, or an effective obstacle. The nearer level is not automatically the right target either. [T]

I therefore replace “the model fabricates the reward” with **“the declared target path lacks validation.”** There is indeed clustering just above the floor, but it does not prove the author's intent: **47/132 in [2.0, 2.3), 35.6%, Wilson [28.0%, 44.1%]**. I would test whether targets beyond intervening structure earn enough additional payoff to offset their lower hit probability. The source of a number and its economic attainability are different questions. [T/I; D/supplement.txt; `kernel/planner_prompt.go:733`]

**Excursions asked for in the dispatch:** `trade_excursions` contains **zero rows**. Its condition-specific MFE p50/p80/p95 are **UNMEASURABLE**; the required inputs are populated entry/exit-linked excursion events, initial brackets, ordered timestamps, ambiguous-bar flags and entry ATR. For transparency I give the available `trader_positions.mfe/mae` **proxies**, in points, below. No MFE percentile becomes a production target.

| Condition | Position MFE n | MFE p50 / p80 / p95, pts | MAE p50 / p80 / p95, pts | Evidence |
|---|---:|---|---|---|
| acceptance | 6 | 17.62 / 89.75 / 140.00 | 47.12 / 57.50 / 60.50 | D/excursion_proxies.csv:2 |
| breakout_retest | 9 | 25.75 / 43.50 / 62.25 | 36.75 / 50.35 / 68.80 | D/excursion_proxies.csv:3 |
| hold | 1 | 92.00 / 92.00 / 92.00 | 22.50 / 22.50 / 22.50 | D/excursion_proxies.csv:4 |
| reclaim | 5 | 16.25 / 21.40 / 26.35 | 42.25 / 55.00 / 70.00 | D/excursion_proxies.csv:5 |
| reject | 31 | 41.50 / 69.75 / 88.25 | 17.75 / 42.00 / 66.00 | D/excursion_proxies.csv:6 |
| sweep_reclaim | 6 | 20.50 / 25.00 / 41.69 | 28.75 / 43.25 / 86.38 | D/excursion_proxies.csv:7 |

**Uncertain historical zero MAE.** Eligible winners **569 and 584** are reconcile rows with recorded MAE=0. I do not interpret these as measured absence of adverse movement. The historical `ComputeExcursion` starts at zero, can skip the fill-containing bar, and returns zero when no bars contribute (`kernel/mae_mfe.go:24`); before E4, `trader/auto_trader_clock.go` wrote that result without a computed-status check (parent of commit `d4aee04a`, lines 738–739). Current `trader/auto_trader_clock.go:752`–759 already uses `ComputePathExcursion` and leaves uncomputed values NULL. That fix is shipped. The raw proxies above remain unchanged; I separately exclude only these two uncertain MAE zeros:

| Group | Treatment | MAE n | MAE p50 / p80 / p95, pts | Evidence |
|---|---|---:|---|---|
| winners | raw_position_proxy | 18 | 11.38 / 20.40 / 46.41 | D/zero_mae_sensitivity.csv:2 |
| winners | exclude_uncertain_zero_MAE | 16 | 13.00 / 22.50 / 48.19 | D/zero_mae_sensitivity.csv:3 |
| reject | raw_position_proxy | 31 | 17.75 / 42.00 / 66.00 | D/zero_mae_sensitivity.csv:4 |
| reject | exclude_uncertain_zero_MAE | 29 | 18.00 / 42.20 / 67.45 | D/zero_mae_sensitivity.csv:5 |

The sensitivity does not repair either trade's unknown MAE or establish which other historical proxies are fully computed. Both winner samples and the adjusted reject MAE sample have n<30: **NO VERDICT**. The primary trade population remains 58; its P&L, held-time and RV results are unchanged. Neither raw nor sensitivity MAE justifies tightening stops, preserving every winner, or judging the floor from a presumed zero-adverse subset. The actual ordered/coverage-verified excursion distribution remains **UNMEASURABLE**. [T/A/I]

**Ordered-path limits.** These proxies may contain full entry/exit-bar extrema, including prices before broker entry or after exit. They stop at the historical exit, censoring the later path a changed stop would need. They do not tell whether target, stop, progress threshold and invalidation occurred first. Replaying close/high/low cannot disambiguate both-hit minutes. Complete fully interior held minutes also have missing bars for IDs 521–523. Even all other complete minute bars do not reveal within-minute order or queue position. A “no progress for 15 minutes” rule cannot be evaluated from final MFE; nor can final MAE tell which winners would survive a tighter stop. I withdraw both old claims of guaranteed winner preservation. [T/I; D/floor_path_sensitivity.csv; D/supplement.txt `path_coverage`]

| Target source | My analytical ruling [I] | Evidence needed before changing production |
|---|---|---|
| Model | Permit a proposed destination and rationale, tied to a frozen level identity; do not treat prose confidence or authored R:R as edge | Target source at author time; first obstacle; actual accepted bracket; independent future path and costs |
| Level map | Preferred **research candidate** because a range-return or continuation destination is explicit; nearest level is only a candidate, not a mandated exit | Machine level formation/read timestamp, role/direction, intervening obstacles, prospectively fixed map and ordered comparisons |
| MFE distribution | Diagnostic for feasibility and timing; **withdraw production p60/p80/p95 targeting from this sample** | Complete/censor-aware paths, training/test separation by day and policy era, no use of the evaluated trade to select its target |
| Fixed multiple | Keep a clearly specified baseline; 2R is a geometric filter/reference, not evidence of viability | Measured initial broker risk, consistent costs and comparison on the same eligible opportunities |

No recommendation here removes the 2.0 filter, converts the 1.5 floor into a ceiling, introduces partial exits or raises size. Such changes require an independently measured proposal and an owner ruling; none was made.

## 5. Three trader-method observations, each with its query

**1. A level touch is not a strategy.** The system can identify a level that sometimes holds yet fail to select a fade with a viable destination before session expiry. Conversely a break classification does not tell it how to enter continuation. My query is the joined **condition × path × era** census (`audit.py`, `trade_stats.csv`, exact IDs in each row): reject 31 / +$586; continuation-candidate 20 / −$1,013.43; post-strict n=0. Those are different theses and policies, not a single edge. I would require prospective opportunity denominators for each executable thesis before deciding that one should replace the other. [T/I]

**2. “Longer winners” does not mean “hold longer,” and “quick losers” does not prove a scratch.** Query: `hold_bucket` in D/trade_stats.csv and `winners`/`losers` in D/supplement.txt. Winners' median hold is 43.04 minutes, losers' 19.74; 15–30-minute exits total −$872 over 12 trades, while 60+ totals +$538.50 over 14. All those buckets are below 30. Conditioning on the final holding time selects the outcome of the exit process: a winner may have survived because it never hit its stop. My next measurement would be progress and adverse excursion at a predeclared elapsed time, evaluated on the same still-open population, with ordered replay and held-out days. I do not infer that keeping a losing trade longer repairs it. [T/I]

**3. A stop exit is not necessarily a losing trade; ledger stop distance is not initial risk.** Query: exact broker entry signal → accepted protective stop → filled OCO child in D/broker_R.csv and D/trades.csv. **570** starts with stop 29430, later exits at a broker stop 29471.75 for **+$17, +0.256 initial R**; it predates BE/trail suspension. **568** has accepted initial stop 29658, but its final arm row 6 says 29726.7. **591** has accepted stop **29355**, stop fill **29355**, hence **0.00 broker stop slippage** and **−1.000R**; arm row **35** says 29351.6284729, whose **3.3715271-point difference is ledger drift**, not slippage. Broker initial references are preserved at D/trades.csv; 591 fill source is NT8 `log.20260903.00000.txt:6260`. These examples change exit economics, not just bookkeeping. I cannot declare all `sync` losses stop-outs, or all stop-outs failed entries. [T]

## Recommendations, ordered: what / why / implementation category / metric

These are documentation proposals only. All [I] judgments are untested here; no n<30 cell is promoted into a trading rule.

| Priority; what | Why | What it takes | Metric I would watch |
|---|---|---|---|
| 1. Freeze a prospective strategy/era contract and evaluate range rejection separately from continuation | [T] 55 pre-0B, 3 0B-pre-strict, 0 strict; 31 rejects versus 20 continuation candidates. Same “strategy” label conceals materially different participation | **Data first; owner ruling** on the experiment; future **code/data** only for immutable revision/thesis/opportunity stamps | Per-policy n and occupied days; entry-eligible opportunities → fills; day-block net expectancy interval. No cell verdict below 30; 30 alone is insufficient |
| 2. Test map-target path against fixed-R on identical prospective entries; keep the current feasibility filter | [T] 119/132 already map to plan levels; 104/132 pass a nearer level. [I] Reward should have a path and horizon rationale | **Data first**, then **owner ruling / prompt / code** only after a reviewed result; freeze training choices before the test | Target-before-stop share with n/Wilson; net expectancy and day-block interval; payoff; MFE timing; ambiguous and missing-path counts; reject if interval cannot support improvement |
| 3. Recover initial broker risk and ordered excursion provenance | [T] 9 auditable R denominators, 49 missing; excursion table empty; 568/591 mutable-stop contradictions | **Data first:** verify the already-shipped writer/hooks (`44d4bbb7`) and NULL-on-uncomputed fix (`d4aee04a`) on future qualifying events; establish historical broker links offline. **Code only if a specific remaining failure is reproduced**; do not reimplement those shipped features | Uncertain-zero IDs and sensitivity; initial-risk missing IDs/count; linked bracket coverage; entry/exit boundary ambiguity; known exit causes; fees; complete risk-normalized sample |
| 4. Rebuild formation-safe level observations before testing first-touch fades or break-following | [T] 677 rows / 423 keys; RTH-L 140 / 14 with retrospective formation; reconstructed ordinal rates do not establish a first-touch advantage | **Data first; code** for formation/source availability and stable identity; no level promotion/retirement/flip | Formation-safe distinct episodes and independent days by kind×ordinal; HOLD/BREAK/ambiguous with n/Wilson; matched non-seated/control opportunities |
| 5. Test thesis expiry/progress exits as a separate prospective comparison, not a fitted scratch | [T] final hold/MFE data are selected and unordered. [I] A failed thesis and slow-but-valid pullback need different handling | **Data first; owner ruling** on predeclared trigger and evidence gate; **code** only after measured benefit | Paired net change versus existing exit on same entries; winners sacrificed; tail loss; same-minute ambiguity; day-block interval; no automatic 15-minute or 0.3R threshold |
| 6. Measure time/regime effects without installing a filter from these cells | [T] every session/hour/RV cell <30; exact machine-regime join unavailable | **Data first** for entry-time labels and future frozen-cutpoint sample; eventual **knob/owner ruling** only if justified | Per-cell eligible opportunity counts, independent days and net expectancy; transfer stability across later days; missing label/R counts |

**Already-fixed cross-check.** I read `AUDIT-CHECKLIST.md` classes 15–16 (fantasy R and small-n claims), excursion/era classes around lines 1010–1047, strict implementation around 1089, and class 79 at 1820. I do not re-recommend shipping strict, adding existing reclaim stop-entry, repairing the already-shared ATR resolver, adding an already-created excursion table or its already-shipped writer/hooks (`44d4bbb7`), repeating the NULL-on-uncomputed fix (`d4aee04a`; `trader/auto_trader_clock.go:752`), increasing the contract cap, re-enabling BE/trail, or rewriting the snapshot reaper. The outstanding Section 1 work is measured method selection and provenance completeness; source code being present does not prove outcomes. Cross-section broker/risk/prompt implementation and integration remain with their owners. [A]

## Explicit withdrawal of prior Section 1 recommendations

| Old item | Superseding ruling |
|---|---|
| R1: production target = nearer level/MFE p60; remove R:R prompt sentence | Withdraw fitted production target and feasibility-rule removal. Current recommendation 2 is an independently frozen comparison |
| R2: permanently forbid confirmation-close entries on the 29-trade class | Withdraw causal/permanent verdict at n=29; strict is already implemented. Study the exact executable thesis prospectively |
| R3: NY only and minimum ATR threshold | Withdraw; session/volatility cells are all below 30, with confounding and corrected RV results |
| R4: no-progress scratch would preserve winners | Withdraw guarantee and fitted threshold; ordered-path test required |
| R5: dedupe on kind/time and wait four weeks | Replace identity with price and formation-aware lineage. Calendar waiting alone does not meet an evidence bar |
| R6: classify all 65 exits; inferred stop/target proportions | Replace with corrected 58 cohort and exact broker cause where available; do not assign cause by price proximity |
| R7: use ATR floor as a ceiling and infer anchor never binds | Withdraw; unanchored count has no total/competing-floor denominator and winner MAE does not justify tighter risk |
| R8: confidence threshold to zero | No Section 1 causal evidence for this knob change; decision/prompt assessment belongs to its section owner |
| R9: delete stale prompt crowns | Unsupported numerical crowns remain incompatible with the sample law, but implementation is owned by Section 7/parent; no new performance claim supplied |
| R10: assume $2.50 round-trip cost and call expectancy net | Withdraw assumed actual fee. Recorded fees are zero; obtain the actual intended commission schedule and live fill evidence. For a hypothetical cost c per round trip, mean becomes −$8.041872−c |

I also withdraw the false professional biography, “one contract cannot work,” “only two lots or a lower MFE target can work,” “all losers die at the stop,” “model targets are fictional,” any claim that first touches here are proven superior, and any certainty about trail/no-progress counterfactuals from unordered extrema. Older Section 1 data artifacts are expressly superseded evidence, retained solely for reproducibility of what went wrong.

## Requirement coverage — every original Section 1 question and subpart

| Dispatch requirement | Answer/result location | Measurement status / exact missing inputs |
|---|---|---|
| 1: brief what it fades | Q1 range rejection and failed-break reversal; condition and thesis CSV | Answered; prospective selection edge unverified |
| 1: what it follows | Q1 continuation candidate and exact arm vocabulary | Answered; n=20 historical candidate grouping, no verdict |
| 1: when | Q1 read/window/flat schedule, saved binding and entry-hour table | Answered; default schedule distinguished from runtime binding |
| 1: with what stop | Q1 widest of authored/anchor/ATR; Q1 R, Q5 broker examples | Nine initial brackets recovered; missing immutable initial broker risk for 49 IDs |
| 1: with what target | Q1 arm.target versus target_chain; Q4 provenance | Authored and nine broker brackets answered; full initial executed targets missing with risk linkage |
| 1: how often | Q1 58 / 12 occupied CME days; D/trade_stats.csv day IDs | Answered; no inference about absent zero-trade days or opportunity frequency |
| 1: distribution by condition | Q1 condition table | Complete corrected cohort; exact plan-version join, 568 retrospective caveat |
| 1: distribution by side | Q1 side table | Complete corrected cohort |
| 1: distribution by session | Q1 session table | Complete plan-session attribution; not substituted for wall-clock hour |
| 1: distribution by hold time | Q1 quantiles/buckets, Q5 selection warning | Ledger holds complete; exact broker entry clocks incomplete |
| 1: realized R via positions/plans/excursions | Q1 R and D/trades.csv/broker_R.csv | **Partial: 9; full distribution UNMEASURABLE**, 49 exact initial brackets/entry identities missing; excursions empty |
| 2(a): fades, hold by kind×ordinal | Q2(a); full 20×3 matrix D/touch_stats.csv, raw stored-ordinal sensitivity | Descriptive test done; **formation-safe edge UNMEASURABLE**, requires formation/source endpoint/read availability and stable identity |
| 2(b): breaks, same inverted | Q2(b) full kind table and ordinal matrix | Descriptive inversion done; executed continuation payoff requires actual entry/stop/target path |
| 2(c): R by session | Q1 session table R n/mean; Q2(c) | **Partial R only**, all sessions below 30; full initial-risk denominators missing |
| 2(c): R by hour | Q2(c), all occupied hours and explicit empty hours | **Partial R only**, every hour below 30; full initial-risk denominators missing |
| 2(d): R by machine regime | Q2(d), exact join failure and fact-6 sensitivity | **UNMEASURABLE as a regime comparison**: blank plan IDs, one as-of trade, one label; need entry-linked labels and multi-regime sample |
| 2(d): R by realized-vol tercile | Q2(d) causal RV60 method, 18/18/18 table and R n | RV test done on 54; four pre-entry bar-window failures; full R incomplete and every cell <30 |
| 2: Wilson, n, no verdict below 30 | Every aggregate rate; full CSV intervals/IDs; evidence contract | Answered; Wilson never used as a mean interval or cure for selection |
| 3: one contract, resting limits, 1.5ATR, 2RR, flat14:45 viability | Q3 shape assessment and Q1 actual rules/eras | Plausible, unverified; no post-strict realized sample; session flats differ |
| 3: shape changes, not parameter changes | Q3 three design changes; recommendations 1/2/5 | Answered as [I] untested proposals; no runtime changes |
| 4: verify 2.55 planned RR | Q4 three-claim table, D/planned_arms.csv | Original export verified 2.553571 / n17; fresh 2.3564 / n132; declared CT slice 2.5357 / n57 |
| 4: verify 1.66 realized payoff | Q4 three-claim table | Corrected 1.74975, W18/L38; not mean R |
| 4: verify 3/36 reached 2R minimum-stop | Q4 historical reproduction, eligible intersection and fresh diagnostic | Original proxy 3/36 verified, including three null rows; eligible 3/33; fresh 5/54. **Actual initial-floor/ordered target claim UNMEASURABLE**, not the historical row list |
| 4: model versus map versus MFE versus fixed multiple | Q4 target-source rulings table | All four answered; model/map already overlap; no fitted MFE production target |
| 4: MFE p50/p80/p95 by condition from excursions | Q4 proxy table and D/excursion_proxies.csv | **Requested table UNMEASURABLE** because excursions=0; raw proxies separate; IDs569/584 zero MAE uncertain, exclusion sensitivity supplied |
| 4: target provenance and ordered path | Q4 obstacle analysis and limitations; D/target_obstacles.csv | Same-document geometry answered; causal machine-level provenance and ordered/censor-aware execution paths missing |
| 5: three trader observations, each with query | Q5 numbered observations and reproducible script/CSV identity references | Answered, without invented experience |
| Format: one-page verdict, three problems, three opportunities | Opening page | Answered |
| Format: recommendations what/why/category/metric and disagreement | Recommendations and withdrawal table | Answered; parent retains integration and other sections |
| Research law: [R]/[T]/[I], primary citations and transfer | Evidence contract; Q3 primary-source paragraph | Answered; FX findings not presented as MNQ execution/ordinal proof |
| Read-only, own scope, reproducible evidence, branch delivery | Evidence contract; data README and verification receipt | Docs-only; commit/push receipt produced at delivery; own worktree retained for parent |

The coverage table is not a claim that every requested statistic is measurable. The main open requirements are full-population initial R, formation-safe touch/ordinal performance, machine-regime comparison, populated ordered excursions and any realized assessment of strict. I have named the absent inputs rather than substituting a favorable estimate.
