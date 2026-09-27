# Assignment 18 — research and report-source review

[A] All 106 assigned files (8,052 source lines) were manually read in full at isolated worktree `/tmp/nofx-understanding-surfaces-20260913`, base `63968be62e44db2fb07a92883e02127b9064b0be`. `reads.json` records each full range and SHA-256; unread count is zero. `functions.json` records 252 named functions, including embedded shell/Python helpers; inline lambdas and callbacks are included in their enclosing function/file notes. This is source understanding of this slice, not a claim of whole-repository, runtime, or dataset validation.

[A] No reviewed research script was executed, no live DB/API/environment was read, no production/source edit was made, no orders/settings/accounts were changed, and no tests or simulations were run. Small review-only Python helpers parsed source and wrote these `/tmp` artifacts. The main instruction file was read for policy only. Initial isolated worktree status was clean. Statistical/data values appearing below are the values asserted or discussed by source, not newly measured sample results.

## Authority and freshness

[A] Read the canon and the audit checklist, including R1 fresh evidence, R2 independent arithmetic, R3 long/short symmetry, R4 exact source locations, R7 corrected PnL/NULL discipline, and isolation/publication requirements. Relevant SYSTEM-MAP PnL and RULEBOOK structural geometry/daily-loss sections were read as excerpts. For both relevant specification files, `git log -1` was `565e8fbe Sun Sep 13 00:39:02 2026 -0500 fix: use owner daily-loss controls without requiring a per-trade cap`. That owner correction supersedes old report proposals. A historical $150 cap, ATR ceiling, fixed-point cap, or 'risk_cap_missing' interpretation is not current authorization or policy. SIM-only and sacred account/config bindings remain controlling.

[A] Historical Understand Anything graph was parsed at `/home/hoang/nofx-untracked-stash-20260816/.understand-anything/knowledge-graph.json` (July10@7a8adce0). Zero nodes matched the 106 assigned paths, which are September artifacts. There are consequently no relevant historical graph edges to validate. `graph.json` preserves this negative result. No claim of current CGC coverage is made; root owns CGC export. Source takes precedence over those historical indexes.

## What this slice does and how it connects

[A] These files are evidence collection, analysis, report rendering, and offline research harnesses. They are not the NT8→BarCache→kernel→risk→SIM order runtime. The September5 families gather historical SQLite row IDs, selected logs, authored plan versions, decisions and snapshot witnesses; compute descriptive and bootstrap statistics; then render frozen reports. The `complete/`, `complete-data/`, and revision scripts frequently correct or explicitly retire earlier adjacent scripts. Reading the whole lineage matters: an older 65-row risk estimate is not the revised 58-row corrected sample, and neither is a current performance measurement.

[A] Typical historical flow is `trader_positions/plans/armed_orders/decision_records/bars` through read-only SQLite snapshots into CSV/JSON, then report summaries and literal prose. Most scripts use hardcoded historical worktree and production DB paths; some open a single query-only transaction, others use independent SQLite calls. A pathname beginning `file:...mode=ro` protects DB writes but does not make the collected multi-source output an atomic runtime snapshot. Reports may combine DB transactions, later log reads, source excerpts, API GETs and fixed text from distinct moments.

[A] The September12 path instead opens an explicit copied SQLite database, splits bars by contract, freezes detector inputs at each scheduled session read, constructs levels/zones, observes first touch and subsequent horizon outcomes, and writes event/geometry artifacts. `data.go:50–196` supplies read-only access, timeframe construction, contract selection, closed-bar slicing and Chicago session windows. `detect.go:62–146` supplies closed-window detector inputs and nPOCs. Structure-fade `main.go:92–292` dispatches reads and evaluates zone events; `out.go:13–233` formats results; surface files estimate families of alternatives. Current structural-stop `main.go:32` generates penetration events or dispatches its sweep; `summarize.py:22–112` summarizes complete registered geometry/fill cells. Important unassigned current sweep/production implementations were not fully reviewed here; root's other assignments own them.

[A] September8 runtime extractor is a sensitive operational reader: `read_runtime.py` reads `.env` and `/proc` environment, mints a short-lived JWT, and GETs local endpoints. It was source-read only. September10 `mutation_driver.py` actually changes source, builds/tests and restores in `finally`; its presence under reports does not make execution read-only. It was not run. `basis_receipts.py` provides pinned SHA/public-byte checks and does not establish fresh runtime state.

## Findings affecting interpretation of currently referenced research

### 1. nPOC historical availability needs an as-of proof

[A] `research/2026-09-12-backtest-structure-fade/harness/detect.go:125–146`, duplicated in the current structural-stop harness dependency, selects stored profile rows with `session_date <= readKey`. It does not require a profile completion timestamp or contract identity. Errors may become an empty list. Current production dependency `trader/auto_trader_dayplan.go:192–228` was read only to understand its stored-profile/touch boundary; no whole-function parity claim follows.

[B] A final same-day stored profile can therefore be available in an offline query before it could have been known at that session read. Contract mismatch is a separate possible provenance issue. This is a source-supported research leakage concern, not a reproduced incident: no live/backtest data was queried, no affected event IDs or count are known, and no claim is made that this changes the training-derived default. The remedy to interpreting the evidence is an immutable completion/contract as-of audit, not an assumed change to production.

### 2. Overshoot calibration is a conditional geometry measure

[A] Current structural-stop events freeze the read-side map and examine future bars as offline outcomes. The complete touch bar's high/low can precede the exact touch; penetration is explicitly an upper-bound observation. The separately read `summarize_overshoot.py` dependency computes C5 held-outcome conditional overshoot percentiles on training data, retains event IDs and checks the original artifact. It does not estimate unconditional loss, risk-of-ruin, or the profitability of applying that stop to every opportunity. H12 classification starts from later closed bars; the touch close is not independent future evidence.

[B] A provisional 4.50-point p95 value derived from the held cohort can be useful as that cohort's descriptive clearance, but cannot establish total loss protection or a profitable trading policy. Large break excursions are not represented by conditioning on holds. Minute OHLC does not recover intrabar touch ordering. No recommendation to reintroduce a mandatory per-trade cap follows.

### 3. Current summary admits prior exposure; stale cap labels remain

[A] Structural-stop `summarize.py:1–6,61–90` openly says the former held-out year has already been exposed. It uses common circular five-day block draws across cells, Bonferroni family-nine bounds and a finite-bootstrap correction. This is materially different from the older independent sign-flip surfaces below; do not transfer their exact implementation flaws to this file.

[A] `summarize.py:35` still counts `ConfiguredAdmission == 'risk_cap_missing'`, and line89 emits 'No monetary cap supplied'. Those labels reflect superseded policy at this base, not the owner's September13 requirement. They should be annotated when citing the artifact; the review did not edit them. `occupancy:38–56` correctly starts equity at zero and reserves an unfilled opportunity's touch minute, but expressly says this is not the live setup selector or attainable live equity curve. Insufficient/empty input groups can still fail in top-level resampling/printing; source assumes a populated frozen dataset.

### 4. Changed-stop composition receipts are selected evidence

[A] `audit_compositions.py` reads copied log lines for changed stops, associates the latest preceding plan and matching authored-stop price, and tolerates one second. The included records are not the denominator of all compositions. [B] Matching price/time is insufficient to prove immutable arm identity where versions or arms share geometry. Neither incidence rates nor accepted broker initial risk can be inferred without the missing denominator/identity. No affected live order is asserted here.

## Archived exploratory limitations — not current trading bugs

[A] Old structure-fade `main.go:170–292` marks FillA/FillC on zone overlap even if the entry anchor is not touched, and its minimum-R target selection can skip a closer opposing zone. It evaluates archived alternative geometry. Current production geometry is different; these are reasons not to cite that old fill surface as attainable execution. `stop_port.go:129` compares a raw embedded legacy function string to production source, rather than comparing the executable copied function directly. The present production signature differs, so source inspection predicts refusal on rerun; this was not executed.

[A] Structure-fade `surface.go:51–138` independently sign-flips flattened cell observations, uses fixed estimated standard errors and lacks the finite-permutation +1 correction. Zone-fade `surface.go:67–296` similarly breaks cross-cell or cross-horizon shared-observation structure in its null simulations. [B] Their maxT labels do not by themselves establish valid familywise inference for correlated event alternatives. Additional multiplicative Bonferroni handling is not a proof that the underlying null construction is right. These are archived statistical limitations, not the current common-day-bootstrap implementation.

[A] Zone-fade `stats.go:70–86` starts peak equity at the first cumulative return, so an initial loss is omitted from maximum drawdown from zero. `longestLosingStreak:88` counts nonpositive, including flats, despite its name. `seam.go:36` records boolean checks without necessarily aborting; some errors are ignored and its same-contract assertion largely verifies its own grouped construction. This is not independent validation of source contract labels. Current structural-stop occupancy avoids the initial-zero drawdown defect.

[A] September5 original `vet-02/.../q11_replay.py:136–142,193–196` uses final-day extrema to choose a round-number/null price universe despite causal wording; its volume-profile approximation spreads volume over bar ranges, differing from kernel close-bin behavior. Revised complete analysis explicitly quarantines unknown formation times and requires complete causal windows. Even revised readers have edge cases such as no-predecessor binary search becoming negative indexing and empty trail divisions. These are static conditions, not reproduced sample corruptions.

[A] Original execution `q15_decision_slip.py:24` has `and/or` precedence allowing an `entry_price` field to bypass the action test; loose nearest-time matching lacks complete trader/side/symbol identity. Original `q31_verified.py` misses the sentinel plan ID exclusion; revision `apply_edits.py` explicitly corrects it and `r01_compliant.py` compares the populations. Original ideas `q07_canonical_set.py:14–39` overwrites arms by plan/scenario, uses mutable stops, allows a nearest decision one minute after entry, and divides corrected cash by a fixed $2 without position quantity. Resulting R, full-stop and MFE/target figures are proxies, not immutable accepted risk. `q10_targets_mfe.py:25–34` calls its counterfactual an upper bound; intrabar order and actual limit fills remain unknown.

[A] Risk `legacy_mc_drawdown.py` combines a no-win Bernoulli recursion with empirical paths where flats exist; complete report explicitly retires that mismatch. Revised risk scripts sample historical trade/day blocks and include zero-baseline drawdown, but do not create new market regimes. `q10_sessions_conditions.py:11` labels a mean interval as t while using 1.96. Fixed UTC-5 is common and only suitable for the pinned summer period. Various summaries use mutable ledger state or zeros as proof of original excursions; later complete reports distinguish those limitations. Historical per-trade caps or live-pilot prose are proposals only and superseded by current daily-loss/SIM instructions.

[A] Forming-candle `analysis.py` explores associations using nearest same-day price/time pairing, not immutable formation identity. Post-open/closed-episode features can overlap the outcome; five-minute queries lack contract/source and continuity guarantees; alternative horizons pre-exclude originally ambiguous rows. Source's duration and mechanical checks help interpretation but do not establish a prospective edge. Literal final result counts are frozen historical assertions.

## Report integrity, tests and non-findings

[A] Prompt complete `measure.py:4–6` is a two-site textual replay, not a fresh provider render. Its manual line categorization is exhaustive byte provenance, not objective truth of every 'fact'. Same-tokenizer comparisons and whole/group BPE caveats are soundly disclosed. `validate_artifacts.py` reconstructs spans and verifies pinned totals/maps; 120 mapped units does not prove behavioral equivalence. Exact proposed policy cuts are explicitly never applied. These are useful artifact checks, not an LLM/validator parity test. Complete scripts explicitly retire earlier prompt evidence as primary.

[A] Stretch complete `verify.py:13–23` checks closed-minute checkpoints, copied source containment, lineage census and broker text witnesses; it explicitly declines complete current-rule broker replay, attainable PnL, immutable initial risk and post-strict profitability. Reaper `reaper.py:9–16` replays an observed cancellation time using the preceding snapshot; it states interval/cache assumptions. It does not synthesize missing broker inventory. Empty output and missing account scope are static limitations.

[A] Renderers such as risk `complete/build_report.py`, prompt `finalize.py`, and execution `revise/apply_edits.py` combine hardcoded prose with generated values. The execution patcher writes even if an exact-once replacement failed. Stretch `extract.py:10–15` returns on an empty CSV without clearing an old file. Ideas `complete/source_evidence.py:6` prints a fixed commit while reading worktree files, without proving HEAD matches. [B] Blindly rerunning these in a new tree/snapshot can yield misleading provenance or mixed stale/current artifacts; this review did not do so.

[A] No source review here demonstrated a new live-trading incident, credential exposure, order/account mutation, or corrupted current corrected-PnL cohort. Credential scans/redaction are heuristics: ideas `q03_eligible.py:29–30` prints raw snapshot JSON despite its following redaction comment; runtime reader accesses secrets if run. These are cautions about script execution/output, not claims that reviewed report outputs contain secrets. No exact sample-impact count is fabricated.

## Complete assigned-file coverage ledger

Every entry below is a full manual source read. Function start/end boundaries and purposes are recorded separately in `functions.json`; statements apply to the file's historical/current role described above. Additional dependency reads are recorded separately and do not inflate106-file coverage.

### 1. `docs/superpowers/reports/2026-09-04-research-conformance-data/d10dump/main.go`

[A] Lines 1–34. Argument DSN is trusted; comment says read-only but no DSN enforcement or argc check; production expectancy.LoadAndBuildAt supplies semantics, JSON errors ignored.

### 2. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-complete-data/collect_sources.py`

[A] Lines 1–19. Archived fixed source spans/base label and log line captures; reads isolated old checkout, selected main logs, immutable gate source and health GET; source base label not verified dynamically. No named functions.

### 3. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-complete-data/render_report.py`

[A] Lines 1–271. Full 271-line dense report renderer read in three bounded chunks. Explicitly supersedes old65-row/MFE-target/ATR-ceiling/session-ban/impossibility recommendations, keeps58 corrected cohort and9 immutable-risk subset. Mixture of dynamically rendered CSV and many fixed historical claims means rerun with changed inputs is not fresh verification. Does not run trading modules.

### 4. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-complete-data/supplement.py`

[A] Lines 1–64. Offline runpy audit dependency then ATR/path/provenance sensitivity. Complete consecutive closed-minute warmup, explicit no intra-minute ordering, separates raw zero-MAE uncertainty IDs569/584. No named functions; imported helpers and predicate lambdas.

### 5. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-data/q15_touch_episodes.py`

[A] Lines 1–45. Exploratory shape==rejection proxy mislabels itself hold; no formation/dedupe/row-ID population evidence; fixed UTC-5 and whole-hour NY split inaccurate outside narrow sample.

### 6. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-data/q16_decision_census.py`

[A] Lines 1–27. Exploratory decision action/refusal counters; timestamps grouped textual day, no trader scope/row IDs, malformed decisions skip risk/execution log counts. Execution-log strings iterate characters if parsed as string. No named functions.

### 7. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-data/q21_stats.py`

[A] Lines 1–155. Exploratory enriched CSV and live tape join; inherited pnl provenance not verified in this script. Five-minute ATR aggregates incomplete/gapped blocks and contracts; later complete supplement fixes closed-block continuity. MFE reach and stop tightness are unordered proxies, not executable counterfactual.

### 8. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-data/revise/lib.py`

[A] Lines 1–23. Shared archive helpers; explicit read-only DB URI; empty quantiles/means None, empty Wilson zeros.

### 9. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-data/revise/r01_headline.py`

[A] Lines 1–39. Archive CSV n65/n58 sensitivity with explicit unresolved ID exclusion, SD/normal CI/payoff/power calculations; requires nonempty wins/losses; no named functions. Headline labels pinned, counts printed dynamically.

### 10. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-data/revise/r07_replay.py`

[A] Lines 1–104. Exploratory counterfactual prototypes: unordered MFE/MAE threshold repricing, fixed $2/pt ignores quantity/costs, path omits partial entry minute and can use unfinished cutoff minute close; not trading validation.

### 11. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/complete/audit.py`

[A] Lines 1–98. Revised audit quarantines historical formation-unknown touch rates; uses read-frozen levels, complete 60m windows, trailing prior completed days, deterministic same-read distance matching. SQL is snapshot read-only, but no contract/source or trader filtering. Negative as-of index wraps future final row if no predecessor; empty trailing history divides by zero.

### 12. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/complete/summarize.py`

[A] Lines 1–55. Offline revised level summary explicitly says no clustered CI from one day. Many literal historical counts/source definitions; never current source validation. First exposure dedup by kind/price lacks formation identity.

### 13. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/complete/verify_details.py`

[A] Lines 1–59. Exact-version RTH-L provenance sensitivity, fixed Sep3 RTH bar bounds, IDs preserved; numeric-trigger regex heuristic and <=1pt price match not canonical identity.

### 14. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/q11_replay.py`

[A] Lines 1–205. Archived exploratory detector port; header lookahead-free overstates full pipeline: random null prices drawn from eventual day range and RN universe uses day high/low. Profiles approximate range-spread volume, not kernel close-bin implementation; UTC-5 fixed, no contract/source/gap guards.

### 15. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/q13_arms_positions_by_kind.py`

[A] Lines 1–83. Exploratory plan-level matching falls back to latest available version, risks retrospective misattribution; corrected PnL NULLs counted separately. Later verify_details uses exact versions.

### 16. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/q16_tod_gap_onrange.py`

[A] Lines 1–29. Exploratory tape descriptives exec prefix of external q11 file, inheriting fixed timezone/unfiltered tape; gap fill and drive are descriptive realized range associations, not causal results. No named functions in this file.

### 17. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/revise/r03_ep_analysis.py`

[A] Lines 1–56. Offline archive q11 episode sensitivity; dedup by day/price/open ignores disagreement by keeping first; demonstrates overlap and small independent level-day counts.

### 18. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/revise/r04_pool_dedup.py`

[A] Lines 1–40. Archive sensitivity exposes first-seated and ever-seated contrasts; ever-seated conditions on future reads, not causal seat efficacy. Imports q11 via exec and lacks complete-window censoring.

### 19. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/revise/r05_reach.py`

[A] Lines 1–41. Offline archived distance/level reach summaries; numerous literal Wilson examples separate from CSV computations; no zero guard in wilson.

### 20. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/revise/r08_pnl_cond.py`

[A] Lines 1–38. Corrected-PnL condition cells with exact plan/version matching, explicit null IDs and source/unresolvable sensitivity; no CLOSED/trader scope, repeated DB reads lack transaction.

### 21. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/revise/r09_labelcensus.py`

[A] Lines 1–18. Non-weekly archived plan-level label census; dedup sessions by date/session, not owner; parsing skips malformed docs and display prefix is not canonical kind. No named functions.

### 22. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/complete/audit.py`

[A] Lines 1–148. Revised complete decisions audit distinguishes persisted reject share, temporal reauthor association, bar reachability, broker execution; preserves UTC offsets and Chicago registry windows. Complete-grid uses count not spacing; latest lifecycle retrospectively excludes past-active versions; successor episode lookup omits trader key. No runtime outcome reproduction.

### 23. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/q04_decisions_parse.py`

[A] Lines 1–36. Parses action JSON including nested decisions and explicit parse/missing states; fixed UTC-5 SQL day, no trader filter; action sets count cycles rather than individual actions.

### 24. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/q05_plan_doc_peek.py`

[A] Lines 1–17. Read-only archived plan structure peek at fixed Sep4 NY highest version; truncated output intentionally exploratory, no ownership tie-break.

### 25. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/q06b_trigger_inforce.py`

[A] Lines 1–54. Archived FIRED means bar spans confirm price, not actual confirmation/execution. Session ends hardcoded ASIA01:30 and NY15:00 differ current registry; groups omit plan owner; inclusive end despite half-open documentation.

### 26. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/q13_coverage_leg3_broker_levels.py`

[A] Lines 1–74. Archive mixed coverage/book/geometry probe; counterfactual uses stop-first touch bar and truncated horizon last close, explicit truncation note; no cost/queue/source/contract checks.

### 27. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/q14_s1_episode_closes_durations.py`

[A] Lines 1–42. Archive source has concrete mislabeled 5-minute aggregation: HH:M prefix groups ten-minute buckets. Counterfactual merges flat and never-filled into zero; not present production execution.

### 28. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/revise/r02_era_positions.py`

[A] Lines 1–71. Exploratory plan-linked corrected-PnL groups retain row IDs/nulls but plan_id IS NOT NULL admits empty/unresolvable and no date guard. Reconcile is only lineage proxy; comparisons confounded by era/path.

### 29. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/revise/r10_population.py`

[A] Lines 1–90. Revised broad/compliant exact corrected-PnL exclusions; plan-linked population query no entry era condition, no CLOSED filter. Fixed offset and hardcoded report cohort labels.

### 30. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/revise/x01_population.py`

[A] Lines 1–64. Explicit entry-era query, corrected PnL NULL counts and IDs. Compliant only excludes UNRESOLVABLE and still includes empty plan IDs; no CLOSED filter; source lineage descriptive.

### 31. `docs/superpowers/reports/2026-09-05-vet-04-monitoring-data/q48_eod_verified.py`

[A] Lines 1–52. Historical EOD observation snapshot; retrospective arms updated after cutoff marked unknown; orders snapshot explicitly not position book; alerts ack current not historical. Raw account included in snapshot receipt export. Exact IDs/log line sources.

### 32. `docs/superpowers/reports/2026-09-05-vet-04-monitoring-data/q51_complete.py`

[A] Lines 1–69. Revised monitoring snapshot calls production cmd/arm-state-sql rather than retyping lifecycle; exact broker accepted/fill stop assertions distinguish ledger drift from slippage. Literal cohort/base assertions intentionally pin historical receipt; no live claim.

### 33. `docs/superpowers/reports/2026-09-05-vet-04-monitoring-data/q52_receipts.py`

[A] Lines 1–14. Selected NT8 and Go log lines at fixed historical times; immutable source diff and public health GET with explicit no readiness claim. No named functions.

### 34. `docs/superpowers/reports/2026-09-05-vet-04-monitoring-data/q53_validation.py`

[A] Lines 1–22. Historical artifact consistency validator: cohort/sample IDs, broker math, source receipts, syntax and changed-path scope. HTTP200 assertion establishes response only; no trading test. No named functions.

### 35. `docs/superpowers/reports/2026-09-05-vet-04-monitoring-data/r06_population.py`

[A] Lines 1–29. Population exclusion waterfall by source/era/unresolved/corrected NULL, row IDs retained except pre-era omitted literal count; no CLOSED check; supplemental fixed Wilson examples.

### 36. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/complete/q11_fill_vs_bar.py`

[A] Lines 1–48. Revised entry audit adds eligible corrected population, side match for arm and fill_time_ms, preserves missing columns; nearest price/time joins not unique execution proof; no contract/source filtering in bars. No named functions.

### 37. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/complete/q14_mae_mfe.py`

[A] Lines 1–33. Revised exit-label extraction explicitly nearest log side/price within five minutes, not execution-id proof. Unused statistics helpers remain; input log and cohort nonempty assumed.

### 38. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/complete/q32_sources.py`

[A] Lines 1–26. Archived source spans, selected NT8 receipts and strategy subset keyed historical bound strategy; captures base comparison and no new deployment assertion. No named functions.

### 39. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/complete/q35_complete.py`

[A] Lines 1–67. Revised execution funnel: IDs and whole/boundary-inclusive lifetime touch proxies explicitly not broker election; immutable initial risk unavailable cohort-wide; only arm35/position591 accepted geometry illustrated. filled_to_win is literal empty IDs, pinned losing fill rather than general calculation.

### 40. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/complete/q36_logs.py`

[A] Lines 1–17. Read-only archival log extraction removes ANSI, captures line provenance, selected arm35 times and cancellation guards. No named functions.

### 41. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/complete/q37_sources.py`

[A] Lines 1–35. Archived source comparison and selected bar/fill/strategy/arm35 evidence; snapshot states first distinct whole order JSON containing audited stop signals, not timeline proof. No named functions.

### 42. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/complete/validate_evidence.py`

[A] Lines 1–28. Historical evidence validator exact 58-ID whitelist, corrected PnL math, funnels/geometry, independent SQL and syntax. Does not test production call sites, and string presence checks do not prove universal read-only behavior. No named functions.

### 43. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/q11_fill_vs_bar.py`

[A] Lines 1–58. Original entry audit includes unresolved PnL population and maps source to market/limit without execution-type proof; same-price arm search lacks side filter, later complete version corrects this. No named functions.

### 44. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/q15_decision_slip.py`

[A] Lines 1–42. Exploratory decision price/slippage matching has operator precedence bug: entry_price truthy bypasses action check; also no trader/symbol/side/unique-execution match. Values are intention-to-fill proxy, not measured broker slippage.

### 45. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/q31_verified.py`

[A] Lines 1–92. Final archive execution audit improves explicit corrected PnL/ID receipts and immutable-risk caveats, but fixed cutoff and correction-note unresolved filter differ revised plan_id scope. Simple TR14 proxy is not Wilder actual entry ATR; all bars lack contract/source filter.

### 46. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/q32_sources.py`

[A] Lines 1–25. Original historical source/NT8/strategy subset receipts, pinned base2a66d91c; no named functions and no deployment claim.

### 47. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/q33_metrics.py`

[A] Lines 1–14. Fixed Wilson examples plus current DB selected arm IDs and open/EOD bar-bucket means; not inferential evidence of execution quality; fixed UTC-5.

### 48. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/revise/apply_edits.py`

[A] Lines 1–201. Archived report patcher exact-once replacement errors still permit partial write; historical 65-to-58 correction and policy prose, not executed.

### 49. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/revise/r01_compliant.py`

[A] Lines 1–46. Corrected 65 versus sentinel-excluded58 population with IDs, fixed UTC-5.

### 50. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/revise/r03_tape.py`

[A] Lines 1–50. Archived tape proxy: many-to-many approximate price/time join lacks trader/symbol.

### 51. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/complete/build_report.py`

[A] Lines 1–276. Frozen complete report renderer supersedes legacy risk artifacts; mixes literal prose with dynamic JSON. Per-trade-cap proposals superseded by daily-loss owner policy.

### 52. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/complete/context_evidence.py`

[A] Lines 1–30. Read-only snapshot calendar and all-account corrected cash sensitivity, not exact live replay.

### 53. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/complete/legacy_mc_drawdown.py`

[A] Lines 1–177. Archived conditional resampling: Bernoulli recursion handles flats differently than iid paths; complete report retires mismatch.

### 54. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/q03_daily_dist.sh`

[A] Lines 1–10. Read-only daily corrected inventory; raw PnL only unresolved diagnostics; fixed UTC-5.

### 55. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/q06_day_mc.py`

[A] Lines 1–76. Active-day block bootstrap truncates horizon; ad hoc t critical and ICC heuristics, finite sample only.

### 56. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/q07_stops_excursions.sh`

[A] Lines 1–16. Mutable arm distance cannot prove initial broker risk; zero extrema conflate missing and measured zero.

### 57. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/q09_strategy_knobs.sh`

[A] Lines 1–30. Explicit trader binding and recursive risk-keyword extraction, no robust redaction.

### 58. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/q10_sessions_conditions.py`

[A] Lines 1–32. Corrected subgroup descriptions; claimed t interval uses normal1.96; slot is condition proxy.

### 59. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/q12_ab_confirm_fills.sh`

[A] Lines 1–9. Commission diagnostics use all-time query despite era heading; NULL zero conflates missing.

### 60. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/r03_rig.py`

[A] Lines 1–173. Corrected population bootstrap and streak/drawdown diagnostics; flat distinctions and initial zero equity; conditional sample only.

### 61. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/r06_arms.py`

[A] Lines 1–50. Mutable arm distances and absolute reward/risk mask target side; ledger placements not broker acknowledgements; cap hypothesis retired.

### 62. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/r07_regime.py`

[A] Lines 1–68. Descriptive unfiltered tape regimes, mean14 TR not Wilder; no explicit complete-day check; cap hypothesis retired.

### 63. `docs/superpowers/reports/2026-09-05-vet-07-prompts-complete-data/audit_evidence.py`

[A] Lines 1–61. Pinned reject classification first matching regex; denominator64 fixed, assertion only no UNCLASSIFIED. Manual constraint mapping validates cardinality not semantic equivalence. Prompt shortage fields sourced exact stored bytes.

### 64. `docs/superpowers/reports/2026-09-05-vet-07-prompts-complete-data/finalize.py`

[A] Lines 1–77. Frozen prompt report publication, exact quote assertions, authored geometry expressly not realized risk. Original contradictions preserved; policy cuts never applied. Mixes literal historical totals and generated artifacts; safe only pinned snapshot.

### 65. `docs/superpowers/reports/2026-09-05-vet-07-prompts-complete-data/measure.py`

[A] Lines 1–53. Two textual historical prompt replacements, explicitly not runtime replay. Manual line-based fact/instruction/schema allocation; BPE category sums may differ whole. Same encodings for compression, not DeepSeek billing or model equivalence.

### 66. `docs/superpowers/reports/2026-09-05-vet-07-prompts-complete-data/validate_artifacts.py`

[A] Lines 1–39. Pinned artifact assertions validate58-row totals, spans reconstruct, maps count, docs scope and secret heuristics; no behavioral equivalence. Credential key scan does not prove all secrets absent.

### 67. `docs/superpowers/reports/2026-09-05-vet-07-prompts-data/q07_measure.py`

[A] Lines 1–33. Heuristic prompt shape and fallback tokenizer counts; any tokenizer exception becomes null. Empty documents divide by zero; unused split_hdr argument. Historical only.

### 68. `docs/superpowers/reports/2026-09-05-vet-07-prompts-data/q20_weekday.py`

[A] Lines 1–17. Weekday daily-bar up fraction excludes flats, assumes next calendar-day label and fixed UTC-5. No contract/source or closed-bar filter; descriptive association cannot justify weekday conviction.

### 69. `docs/superpowers/reports/2026-09-05-vet-08-stretch-data/complete-0905/extract.py`

[A] Lines 1–79. One read-only transaction, opportunity version/session windows, corrected eligible IDs. CSV writer silently leaves previous file if empty; strategy selected by fixed prefix first row. Logs/snapshots separate observations; no current runtime reconstruction.

### 70. `docs/superpowers/reports/2026-09-05-vet-08-stretch-data/complete-0905/logs.py`

[A] Lines 1–25. Regex log and broker excerpts with numbered provenance; lifecycle events parse Chicago time, broker account name redacted. Selected patterns and dates cannot prove full event coverage.

### 71. `docs/superpowers/reports/2026-09-05-vet-08-stretch-data/complete-0905/reaper.py`

[A] Lines 1–19. Observed cancellation-time reaper component approximation uses previous snapshot received age<=60s; assumptions explicitly30s interval/cache survival. No counterfactual inventory. Missing account/trader scoping; empty out indexing fails.

### 72. `docs/superpowers/reports/2026-09-05-vet-08-stretch-data/complete-0905/verify.py`

[A] Lines 1–23. Pinned complete artifact checks include exact source port containment and closed minute checkpoints. Explicitly excludes full replay/fills/PnL/initial-risk claims. Source-current containment expected to drift; no tests executed.

### 73. `docs/superpowers/reports/2026-09-05-vet-08-stretch-data/q34_wilson.py`

[A] Lines 1–34. Literal historical Wilson table mixes retired65-era proxies and ledger event units. Zero denominator represented zero triple instead of unavailable; not fresh measured rates.

### 74. `docs/superpowers/reports/2026-09-05-vet-09-complete-data/q01_context.py`

[A] Lines 1–13. Read-only transaction context plus health GET and source excerpts; origin/dev hash does not establish HEAD/runtime binding. Queries capture diagnostic row identities, not strategy eligibility.

### 75. `docs/superpowers/reports/2026-09-05-vet-09-complete-data/q03_proxy_sensitivity.py`

[A] Lines 1–14. Explicit sensitivity removes uncertain zero-MAE winners569/584 without changing primary58 population. Nonempty linear quantile; stored excursion/floor-age proxies remain not validated cohort.

### 76. `docs/superpowers/reports/2026-09-05-vet-09-top-ten-data/q03_eligible.py`

[A] Lines 1–46. Corrected-PnL canonical query differs exclusion diagnostic: sentinel plan_id exclusion missing from latter; blank plan IDs allowed. IDs and Chicago17h days present. Snapshot orders_json output not actually redacted despite comment. No execution.

### 77. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/complete/source_evidence.py`

[A] Lines 1–12. Source-only excerpt writer prints fixed rev488ce827 but reads working tree paths without verifying HEAD or cleanliness; provenance label can diverge on rerun.

### 78. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/q01_asof.sh`

[A] Lines 1–10. Independent read-only SQLite calls inventory freshness; no shared transaction, max-created errors collapsed to n/a. Fixed UTC-5 labels.

### 79. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/q02_era.sh`

[A] Lines 1–20. Old first-pass era aggregation lacks sentinel plan_id exclusion; UTC-midnight start differs CT era. Does not establish current corrected canonical cohort.

### 80. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/q04_store_summaries.sh`

[A] Lines 1–27. First-pass mutable arm/plan ledger and shadow/touch groups; raw net_pnl not corrected trade PnL, touch labels not strategy expectancy. No sample IDs for most counts.

### 81. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/q07_canonical_set.py`

[A] Lines 1–66. Archived risk proxy: note-based unresolved exclusion, timezone-naive epoch boundary; arm join overwrites by plan/scenario without version/side/trader, mutable stop; nearest decision includes one minute future and lacks side match. Fixed2 USD conversion ignores quantity. Not initial risk or causal R evidence.

### 82. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/q09_arms_kinds_sessions.sh`

[A] Lines 1–22. Ledger dedup plan/scenario/rounded price merges versions/legs; era funnel lacks era predicate. Decision text LIKE counts not executed trades. Touch group comparisons descriptive.

### 83. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/q10_targets_mfe.py`

[A] Lines 1–51. Consumes archived ambiguous risk joins and unordered stored MFE. Fixed-target counterfactual explicitly upper bound; cannot establish fill ordering/profitability. Median indexes len(S) rather than filtered RR population may fail; finite data assumptions.

### 84. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/q11_outage_executor_shadow.sh`

[A] Lines 1–18. Last/first decision times bound absence of decisions, not proved feed outage. Text intent counts and model-hours sum latency proxy, not provider billing or action execution.

### 85. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/r10_rest.sh`

[A] Lines 1–38. Historical read-only named-row diagnostics and weekday arithmetic; median fixed offset28 assumes sample cardinality, missing initial corrected eligibility consistency. No updates.

### 86. `docs/superpowers/reports/2026-09-08-the-strategy-data/extract.py`

[A] Lines 1–53. Historical single-transaction read-only live extraction; uniquely scopes trader and bound strategy, sanitizes identities/key prefixes. Plan/lifecycle/excursion/bars queries broader than trader scope; archived pre-contract/source projections not current replay-safe.

### 87. `docs/superpowers/reports/2026-09-08-the-strategy-data/read_runtime.py`

[A] Lines 1–32. Source only; DO NOT EXECUTE for this review. Reads .env and /proc environment, mints short-lived JWT then GETs four local endpoints and records loaded/disk hashes; first trader only. Sanitization is heuristic, stdout health is unsanitized.

### 88. `docs/superpowers/reports/2026-09-08-the-strategy-data/supplement.py`

[A] Lines 1–74. Offline exported cohort analysis retains IDs and effective explicit legs; checks historical 1.5 ATR floor, not current structural-stop law. Missing quantile populations/division geometry can abort.

### 89. `docs/superpowers/reports/2026-09-10-scenario-level-identity-data/basis_receipts.py`

[A] Lines 1–34. Immutable SHA public-basis receipt generator; compares HTTP bytes exactly to git blob; explicit pinned historical revisions and output argument. No named functions.

### 90. `docs/superpowers/reports/2026-09-10-scenario-level-identity-data/mutation_driver.py`

[A] Lines 1–37. Mutation driver EDITS SOURCE in derived repository, builds and tests, restores in finally. Not authorized to run here. Checks exact one replacement, expects build green/test nonzero, but arbitrary infrastructure test failure counts killed. No named functions.

### 91. `docs/superpowers/reports/2026-09-11-forming-candle-test-data/analysis.py`

[A] Lines 1–382. Archived forming-candle association study, not deployable prediction validation. Nearest ±10m same-price/day joins exclude multiply claimed outcomes but do not establish formation identity. Closed-episode features can overlap outcomes; diagnostics explicitly inspect this. No snapshot transaction; horizon SQL lacks contract/source filters and skips primary ambiguous rows. Footers hardcode historical counts.

### 92. `docs/superpowers/reports/2026-09-12-structural-stop/harness/audit_compositions.py`

[A] Lines 1–56. Copied changed-stop logs only, not denominator of all compositions. Reconstruct latest preceding plan with authored-stop price match; conditional selection and one-second tolerance cannot prove actual arm identity.

### 93. `docs/superpowers/reports/2026-09-12-structural-stop/harness/data.go`

[A] Lines 1–193. Read-only SQLite copy; groups MNQ bars by contract/timeframe; excludes mixed/off-scale and empty/spans-roll contracts. ContractAt nearest previous minute has no staleness ceiling or equal-time contract tie breaker. Calendar session windows hardcoded. No execution calls.

### 94. `docs/superpowers/reports/2026-09-12-structural-stop/harness/main.go`

[A] Lines 1–143. Measurement first zone touch per read; builds current kernel zone maps, complete shortlisted widths only, stores future bars intentionally for offline evaluation. H12 hold/break excludes touch close but penetration includes entire touch range; upper bound ordering caveat. Empty merged tape panics.

### 95. `docs/superpowers/reports/2026-09-12-structural-stop/harness/stop_port.go`

[A] Lines 1–122. Frozen legacy widest-wins comparator pinned to 6b3fddf7, explicitly not current production geometry.

### 96. `docs/superpowers/reports/2026-09-12-structural-stop/harness/summarize.py`

[A] Lines 1–112. Current structural-stop analysis acknowledges already-exposed held-out year; common five-day block draws, nine-cell family bounds; occupancy explicitly not live selector. risk_cap_missing output/text reflects superseded per-trade policy, must not restore it.

### 97. `docs/superpowers/research/2026-09-12-backtest-structure-fade/harness/detect.go`

[A] Lines 1–146. Production detector kernels called directly but no owner levels or level-state history. Latest 30 profile POCs queried by day without contract or completion timestamp; possible future same-day profile leakage requires population verification.

### 98. `docs/superpowers/research/2026-09-12-backtest-structure-fade/harness/main.go`

[A] Lines 1–440. Archived exploratory 16-cell geometry backtest; FillA/FillC always true at any zone overlap; target skips closer zones to meet minR; these differ from current production/updated geometry sweep. Emits all cells and held-out readouts; no actual trade calls.

### 99. `docs/superpowers/research/2026-09-12-backtest-structure-fade/harness/out.go`

[A] Lines 1–232. Archive reporting ignores unfilled returns, zeros unresolved statistics, uses event iteration order for drawdown (not occupancy equity); MNQ fixed $2/point.

### 100. `docs/superpowers/research/2026-09-12-backtest-structure-fade/harness/render_c1.py`

[A] Lines 1–37. Archive markdown renderer reads geometry/reference/surface JSON; index-zips era arrays without cell-identity checks; no named functions.

### 101. `docs/superpowers/research/2026-09-12-backtest-structure-fade/harness/stop_port.go`

[A] Lines 1–237. Archive legacy comparator plus text guard; verifyPort compares embedded string to production source, not executable copied function; production signature now intentionally differs, so archive startup expected to refuse.

### 102. `docs/superpowers/research/2026-09-12-backtest-structure-fade/harness/surface.go`

[A] Lines 1–135. Archive 16-cell significance surface; independent sign flips per flattened cell/event destroy pairing across same opportunities; fixed observed SE, maxT empirical p can equal zero, PBonf multiplies already maxT-adjusted p.

### 103. `docs/superpowers/research/2026-09-12-backtest-zone-fade/harness/data.go`

[A] Lines 1–193. Read-only SQLite copy; groups MNQ bars by contract/timeframe; excludes mixed/off-scale and empty/spans-roll contracts. ContractAt nearest previous minute has no staleness ceiling or equal-time contract tie breaker. Calendar session windows hardcoded. No execution calls.

### 104. `docs/superpowers/research/2026-09-12-backtest-zone-fade/harness/seam.go`

[A] Lines 1–134. Seam checks report booleans but do not stop replay; SQL/file/JSON errors may be ignored. Only named four timeframes checked; 06-22 delta waived but total fixed. Single-contract assertion checks grouping construction, not authoritative contract labels.

### 105. `docs/superpowers/research/2026-09-12-backtest-zone-fade/harness/stats.go`

[A] Lines 1–154. Local statistics despite comment claiming kernel Wilson; empty statistics become zero; maxDD starts peak at first cumulative return, excluding initial loss from zero.

### 106. `docs/superpowers/research/2026-09-12-backtest-zone-fade/harness/surface.go`

[A] Lines 1–295. Archive 81-cell surfaces; map events share tape but permutations treat map groups independently; hold outcomes globally shuffled per horizon, loses cross-horizon/tape dependence. No production gating.
