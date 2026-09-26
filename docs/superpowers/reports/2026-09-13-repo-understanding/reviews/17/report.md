# Assignment 17 — archived research and replay source understanding

Source pin: `63968be62e44db2fb07a92883e02127b9064b0be`. Isolated read-only worktree: `/tmp/nofx-understanding-execution-20260913`. **106/106 assigned files, 8,052 lines, fully manually read; zero unread assigned files.** This slice comprises research/capture scripts and stand-alone Go harnesses, not the production trading service. 278 named functions/methods (including four shell helpers and nine named Python lambdas) are inventoried in `functions.json`; callbacks are covered under enclosing functions. Top-level scripts have file-level contracts in `reads.json` and the annex below.

Evidence: **[A]** directly read source and current source/hash/census inspection; **[B]** implications inferred from that source; no runtime incident, profitability result, live-account state, service health, or execution failure reproduced. Historical row counts below describe scripts' pinned assertions, not a new query of those rows. No assigned script was executed, no trading/network service was called, no live DB/environment was read, and no source/config/account was modified. Artifact-writing Python parsed source only. No financial/external fact claims require web research here.

## Boundaries and evidence precedence

[A] Read AGENTS instruction and CLAUDE canon, plus audit checklist R1–R10 (`AUDIT-CHECKLIST.md:2281–2308`), strict-corrected law (`:522–550`), system-map corrected analytics (`SYSTEM-MAP.md:301–309`) and research archive separation (`:404–425`). Relevant rulebook sections `VL-TRADING-RULEBOOK-v1.md:181–216,290–335` explicitly separate touch, confirmation, fill, historic populations and current policy. Freshness at base:

- AUDIT-CHECKLIST: `dfda15e1 2026-09-13T00:48:28-05:00 test: isolate session clock fixtures from weekly backfill workers`.
- SYSTEM-MAP and RULEBOOK: `565e8fbe 2026-09-13T00:39:02-05:00 fix: use owner daily-loss controls without requiring a per-trade cap`.

These are source-review prerequisites, not permission to rerun captures. Owner correction controls: loss limit is DAILY, no mandatory additional per-trade cap. Historical script references to a missing cap are archived design premises and must not be reinstated.

[A] July10 Understand Anything graph (`7a8adce0`, 3,121 nodes/9,588 edges) contains **zero assigned-path nodes/edges**. `graph.json` records that negative result. Current root Go AST inventory supplied function boundaries and syntax-only calls; Python AST and shell declarations supplied the rest. This does not make the historical CGC/UA index current. No indexing mutation was performed; no claim about type-resolved callers is made.

## End-to-end data and ownership flow

1. **Capture/census:** dated scripts read SQLite tables or previously exported CSV/JSON/logs. Completed audits generally open query-only snapshots and export row membership. Older shell invocations can open the configured source directly, one query at a time. Capture scripts may intentionally fetch HTTP and authentication context; they are not an offline-safe validation suite.
2. **Cohort construction:** strict `pnl_corrected`, resolved plan identity and test exclusions feed the completed58-trade/12-CME-day cohort. Earlier65-trade arithmetic, later65-trade strategy census and scenario/plan counts are distinct populations. They cannot be concatenated or presented as a single fresh result.
3. **Attribution:** plan/version/scenario keys, signal IDs and broker order rows provide the strongest available joins. Nearest-time/side/price heuristics and current mutable stop snapshots are weaker historical proxies. Repeated rejections, plan versions, touch episodes and actual positions have different denominators.
4. **Research transformation:** Monte Carlo, Wilson/mean intervals, excursion/ATR summaries, refusal replay, identity/style census and geometry studies generate artifacts. Complete replays distinguish necessary conditions from broker fills. Newer Go harnesses load concrete-contract/source bars, construct closed-prefix detector reads, then simulate touch policies independently over zones and sessions.
5. **Outputs:** local CSV/JSON/Markdown are report artifacts. Imports into `nofx/kernel` and the frozen geometry core reuse pure logic; these do not route orders or reproduce live receipt/account/queue constraints. Production `ComposeLevelFadeGeometry` resolves scenario identity; research `ComposeFrozenLevelFadeGeometry` is given a frozen zone index (`trader/structural_geometry.go:106–125`). Daily gates, one-position conflicts, resting-order cancellation, partial fills and broker receipts are outside the simulated fill model.

## Important research limitations and correction history

### Current reproducibility and production-contract separation

[A] Old zone harness `research/2026-09-12-backtest-zone-fade/harness/main.go:34` calls `verifyPort`; `stop_port.go:129–153` compares the old embedded `composeArmStop` signature/body with current source. Current `trader/arm_stop_anchor.go:76–88` adds structural context and forwards reject fades to `ComposeLevelFadeGeometry`. Thus the old benchmark guard detects drift at this base. **[B] It should refuse startup, not silently benchmark the old stop as current production.** No command was run to demonstrate its exit status. The guard itself compares the embedded string with production, not the actual local callable function text (`stop_port.go:45–124` versus `:158–237`); edits to only the callable copy could escape that particular guard. This is a static future-maintenance limitation, not observed divergence between those two old copies.

[A] `reports/2026-09-12-structural-stop/harness/sweep.go:84–100` directly calls the exported frozen-zone production geometry core. Its comments/branch at `:93–98` still discuss missing per-trade-cap permission. Rulebook `:195` and current owner correction supersede that premise. Do not treat clearing that historical diagnostic label as actual runtime risk authorization or restore the cap.

### Fill, cost, target and drawdown assumptions

[A] Original zone `main.go:259–265` sets touch-A filled at the anchor whenever the band is touched and assigns short C entry `anchor+tick`, long C `anchor-tick`; `evalLine:360–366` repeats the C rule. **[B] Band intersection does not establish an anchor fill; that tick direction is favorable, so it is not adverse slippage.** Original zone `eval.go:209,228,251` skips the fill bar, uses exact stop prices even across gaps and deducts fixed two-point friction. Structural-fade `eval.go:221,240,321` retains these original-model simplifications. Gross R uses gross outcome while net expectancy uses friction-adjusted points; they are different metrics, not inconsistent accounting by themselves.

[A] Later corrected structural sweep `sweep.go:123–202` includes fill-bar ambiguity, uses adverse long+tick/short−tick entry, handles stop gaps and target-through conditions. These are meaningful archived improvements; old optimistic model assumptions must not be re-reported as newly discovered production behavior. Even the corrected model is an OHLC scenario exercise, not queue/partial-fill/receipt-latency proof. Fixed friction and `$2/point`/tick assumptions apply to the declared MNQ one-contract research context; historical gaps can exceed that friction, and a collection of independently simulated touches is not one account equity path.

[A] `zone-fade/harness/out.go:111–117` builds cumulative values starting **after** trade1; its actual same-directory `stats.go:70–85` initializes peak at `cum[0]`. **[B] Initial losses are omitted from maximum drawdown (e.g. one losing trade yields zero), so this emitted research metric understates zero-capital-baseline drawdown until a higher peak establishes.** The same helper is in assigned structure-fade `stats.go:70`. This algebraic counterexample was reasoned from source, not run as a test. `out.go:108–110` leaves profit factor0 when lossSum0; that sentinel is not a finite observed no-loss PF. `longestLosingStreak` counts flats as losses, whereas old MC `max_streak` counts negative values only; compare definitions before comparing headline streaks.

### Causality, sample identity and uncertainty

[A] Old `vet-02-levels-data/q12_grade_vs_outcome.py:12–16` and `q12b_grade_vs_outcome_dedup.py:9–12` fall back to the whole current day when prior history is insufficient. **[B] A prospective feature interpretation would leak later tape.** Day/label/price first-seat dedup does not restore omitted contract/trader identity or statistical independence. `q10_live_dedup_wilson.py:12` and some recuts collapse contradictory outcomes lexically; completed vet01 `audit.py:144–151` explicitly preserves conflicts as ambiguous and reconstructs ordinals. These are successive research contracts, not interchangeable estimates.

[A] Old execution `q14_mae_mfe.py:29` selects5m bars with open before entry; completed `complete/q31_verified.py:39` uses close<=entry. Old `q16_funnel.py:33` invents a2minute life when stored duration is zero; completed `q31_verified.py:83–90` explicitly removes that invention. `r07_pnl_pop_SUPERSEDED.py:1` already identifies its wrong-year/latest-version defects. Preserve this correction history instead of describing the earliest drafts as current measurements.

[A] Completed `vet-01-way-it-trades-complete-data/audit.py:74–86` pins58 eligible rows and12days; `:89–104` joins signal→filled entry→Accepted stop within10seconds, improving on nearest-arm searches. Its closed-minute RV uses61 consecutive bars (`:113–117`), but cohort terciles use the whole cohort (`:118`): valid descriptive buckets, not independently trained live cutoffs. The cluster bootstrap (`:138–142`) resamples12 observed days; it does not create more independent days. Completed risk `complete/recompute.py:59–63,119–120` pins the same cohort, states cost sensitivity and omits absent days rather than fabricating zero days.

[A] Research profile lookup `structural-stop/harness/detect.go:125` / zone `detect.go:125` selects profile information by session key. **[B] Historical observation/receipt availability and concrete-contract lineage cannot be concluded from a key alone.** No actual profile rows were inspected here, so lookahead in that lookup is UNVERIFIED, not established. `structure-fade/data.go:65–111,133–150` improves contract/source selection and closes<=cutoff; it does not independently prove no duplicate same-time rows or sufficiently fresh last closed bar. Detector allfresh metadata is a modeling assumption, not a historical receipt claim.

[A] `zone-fade/main.go:28,141–147` fixes a heldout era; `main.go:405–474` emits parameter surfaces. Sample-specific Wilson thresholds are not a policy-validation guarantee. Independent-proportion intervals (`structure-fade/stats.go:26`, used in zone split logic `main.go:520–565`) do not account for paired/shared-session outcomes. Multiple zone touches share tape, and parameter selection/holdout reuse require experimental provenance. Surface dependency excerpt `zone-fade/surface.go:1–110` declares in-sample tuning and81-cell surfaces, but the full surface implementation is outside this assignment and not re-reviewed here.

### Execution safety of archived tools

[A] `vet-07-prompts-complete-data/capture.py:45–61` performs HTTP and, on401, reads `.env` and mints a token using a database user; `vet-09-top-ten-data/q01_verify.py:15` calls health; style `validate.py:26–27` reads live account/environment context for privacy scanning. They must not be casually executed as offline tests. No secret values were read or included by this review. `vet-05-execution-data/revise/apply_edits2.py:3–30` is a report mutator and can write after replacement-count failures; it is not a read-only verifier.

## Relevant validation and unresolved branches

[A] Assigned validators were fully read: risk `complete/validate.py`, vet09 `final_validate.py`, and style `validate.py`. They validate stored populations/arithmetic/artifact coverage, not full current production behavior; some read live context. None was executed. Port drift guard is a useful source consistency check with the limitation above. Seam report `structure-fade/seam.go:36–134` has unchecked SQL/JSON errors and reported booleans; zone `main.go:52` collects results without turning each false diagnostic into a fatal failure. Static syntax census and SHA256 recheck completed, covering all assigned files. No broad suites or broker tests were appropriate for this read-only archive assignment.

Unresolved: exact historical DB rows/current receipt timing, original report artifact hashes against original execution environments, external absolute-path helpers used by old scripts, current live account/day switches, actual broker outcomes, and a full type-resolved graph are not established by this slice. A claim of a new production incident or validated profitable strategy would exceed the evidence.

## Complete per-file coverage and purpose

Paths below are relative to repository root; all are full/manual reads. Function boundaries, syntax-only call expressions and per-file risk context are in `functions.json`; exact SHA256 and read ranges are in `reads.json`.

000. `docs/superpowers/reports/2026-09-03-mc-drawdown-data/mc_drawdown.py:1–177` — Seeded iid/block/day bootstrap of historical corrected trade sample; zero-baseline maxDD; policy examples and friction-free historical calibration only.

001. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-complete-data/audit.py:1–201` — Completed vet01 snapshot audit: exact plan/version and signal/order joins; excludes test/sentinel/NULL; pins58 trades/12days; day-cluster resampling; MFE proxies explicitly unordered.

002. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-complete-data/binding.py:1–17` — Recursively visits saved configuration with allowlisted fields; saved settings are not proof of runtime binding.

003. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-complete-data/legacy_check.py:1–22` — Recomputes archived MFE-floor arithmetic and labels limitations; not an ordered fill simulator.

004. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-data/q13_enrich.py:1–91` — Enriches corrected trades with approximate nearby arms/decisions/stop snapshots; fixed UTC-5; identity not authoritative.

005. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-data/revise/r04_close.py:1–36` — Approximate close-decision ±4minute joins and exit reason extraction; historical heuristics.

006. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-data/revise/r05_mfe.py:1–55` — MFE reach and ATR ratios from external helper; winners/threshold reach not ordered trade outcomes.

007. `docs/superpowers/reports/2026-09-05-vet-01-way-it-trades-data/revise/r12_tables.py:1–72` — Revised tables exclude named row IDs; whole-sample descriptive buckets, no prospective validation.

008. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/q10_live_dedup_wilson.py:1–29` — Deduplicates touches and Wilson intervals; MIN(outcome) can collapse contradictory outcomes lexically.

009. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/q12_grade_vs_outcome.py:1–88` — Grade/outcome exploration; delta fallback uses current entire day on insufficient prior days; external replay dependency.

010. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/q12b_grade_vs_outcome_dedup.py:1–69` — First-seat day/label/price dedup reduces repetition; retains current-day delta fallback and no trader/contract identity.

011. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/revise/r03_dedup.py:1–64` — Additional dedup and subgroup exploration; repeated related events and multiple comparisons remain.

012. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/revise/r04_q12b.py:1–29` — Touch/grade bucket tabulation with Wilson uncertainty; observation is not broker execution.

013. `docs/superpowers/reports/2026-09-05-vet-02-levels-data/revise/r07_pnl_pop_SUPERSEDED.py:1–50` — Explicitly SUPERSEDED wrong-year/latest-version population analysis; archived correction trail, not current bug.

014. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/complete/supplement.py:1–82` — Supplement reads snapshot and exports diagnostics; forward minute counterfactuals censor missing tape and ambiguous chronology.

015. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/q01_store_premises.sh:1–24` — Shell schema/population premises; each sqlite query separate, not one atomic snapshot.

016. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/q06_plan_corpus.py:1–99` — Plan corpus/latest version and price-touch FIRED proxies; fixed UTC-5 and no confirmation/fill guarantee.

017. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/q07_reauthor_cost.py:1–52` — Clusters reauthor attempts over20minutes; next publication may fail closed and is not acceptance latency.

018. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/q11_wilson_pnl_2r.py:1–75` — Historical Wilson/P&L/2R summaries; nearest ±3minute stop lookup lacks exact broker identity.

019. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/q12_refusals_by_leg.py:1–25` — Refusal-by-leg event counts; repeated rows are not deduplicated opportunities.

020. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/revise/r01_rejects_arms_intents.py:1–66` — Historical reject/arm/intent ledger classification and exact exported row IDs.

021. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/revise/r11_store_rechecks.py:1–200` — Store rechecks and author geometry; MFE target-first counterfactual ignores ordered path; nearest stop association.

022. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/revise/r16_minsl_repairs_rr_wilson.py:1–60` — Minimum-stop repairs/RR with nearby cycle rows; matching lacks exact trader/side identity at some joins.

023. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/revise/r18_flap_overlap.py:1–39` — Overlap from arm created/updated lifetime and short-name grouping; timestamps are not broker lifetime.

024. `docs/superpowers/reports/2026-09-05-vet-03-decisions-data/revise/x02_refusals.py:1–45` — Raw refusal totals and hardcoded reconciliation values; event and opportunity denominators differ.

025. `docs/superpowers/reports/2026-09-05-vet-04-monitoring-data/q01_store_asof.sh:1–8` — Shell as-of schema census; live path literals are archived instructions, never executed by this review.

026. `docs/superpowers/reports/2026-09-05-vet-04-monitoring-data/q50_eod_verified.py:1–53` — EOD diagnostic marks future-updated arm state UNKNOWN; decision silence and read availability do not prove flatness.

027. `docs/superpowers/reports/2026-09-05-vet-04-monitoring-data/r07_rth_vs_atr.py:1–42` — RTH counts permit389/390 as full; simple true-range averages differ from Wilder ATR.

028. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/complete/q31_verified.py:1–94` — Completed execution audit corrects closed5m availability and zero-lifetime invention; still distinguishes proxy stop joins from broker facts.

029. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/complete/q33_metrics.py:1–15` — Hardcoded historical rate/bar-bucket display; not a fresh query.

030. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/complete/q34_integration.py:1–17` — Completed integration detail reconstructs historical snapshots and literal broker stop; no current runtime assertion.

031. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/q14_mae_mfe.py:1–74` — Older excursion audit includes5m open before entry rather than close before entry; superseded by completed q31.

032. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/q15b_cycle_drift.py:1–32` — Cycle drift pairs nearby decision and bar close at open<=cycle; can use incomplete-minute close.

033. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/q16_funnel.py:1–69` — Old funnel lacks version identity and invents2minute life for zero duration; corrected q31 explicitly avoids this.

034. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/q19_snapshot_stop_price.py:1–14` — Snapshot stop-price inventory truncates names to8characters; grouping may alias identifiers.

035. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/q20_plan_scenarios.py:1–12` — Plan-scenario lookup uses ID prefix for exploration, not canonical full identity.

036. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/q27_guard_cancel_paths.py:1–27` — Cancel-path nearby windows and close at cancel minute; not ordered broker fill evidence.

037. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/q28_guard_counterfactual.py:1–40` — Guard counterfactual skips fillbar, uses exact execution prices and no friction; approximate research only.

038. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/q34_integration.py:1–16` — Integration diagnostics duplicate completed variant; runtime bindings not verified.

039. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/revise/apply_edits2.py:1–31` — Historical markdown replacement utility writes despite failed replacement counts; never run as an offline validator.

040. `docs/superpowers/reports/2026-09-05-vet-05-execution-data/revise/r02_legs_floor.py:1–77` — Revised floor/legs tables compare historical populations and excursion proxies, not causal policy improvements.

041. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/complete/recompute.py:1–123` — Completed risk recomputation snapshot pins58/12, exact IDs and strict corrected values; seeded day bootstrap and cost sensitivity.

042. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/complete/validate.py:1–36` — Committed-fixture risk validator checks exact58IDs/drawdown/streak and deterministic outputs; not run here.

043. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/q01_census.sh:1–19` — Initial risk census has documented epoch premise error; archived supersession.

044. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/q02_premise_227.sh:1–23` — Premise shell computes timestamps but queries conflicting literals; do not reuse as current cohort.

045. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/q04_build_sample.py:1–30` — Initial sample excludes correction notes rather than unresolved plan sentinel; compliant recut supersedes it.

046. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/q08_bars_regime.py:1–50` — Bar regime/true-range summaries lack explicit percontract continuity; retrospective descriptors.

047. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/q11_daily_regime.py:1–49` — Daily simple ATR/range classification is retrospective whole-day context, not a causal feature at entry.

048. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/q13_calendar.sh:1–34` — Shell discovers calendar tables; no live news-service validation.

049. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/q14_weekly_intraday.py:1–37` — Realized day/week summaries for historical corrected rows; not forward risk bounds.

050. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/q15_filled_arms_rate.sh:1–14` — Filled-arm shell ratios use historical literal cutoff; no present fill-rate claim.

051. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/q17_gonogo_math.py:1–33` — Hardcoded65-row go/no-go arithmetic and policy examples; no current permission or recommendation.

052. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/r01_population.sh:1–17` — Population reconciliation shell enumerates exclusions and counts.

053. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/r02_build_sample_compliant.py:1–38` — Compliant sample fixes unresolved plan sentinel and exports member IDs; field created_ms actually holds entry time.

054. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/r04_limits_buckets.py:1–106` — Limits/buckets and approximate mean intervals; posthoc day direction uses full-day close only as retrospective stratification.

055. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/r05_machine_counterfactual.py:1–51` — Machine counterfactual says account-scoped but SQL lacks account restriction and groups only by day; multiaccount limitation.

056. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/r08_rr_matched.py:1–25` — Matched entry signal improves identity but mutable ledger stop/assumed one lot do not freeze initial broker risk.

057. `docs/superpowers/reports/2026-09-05-vet-06-risk-data/r09_killswitch.py:1–42` — Kill-switch bootstrap on historical daily sequence; synthetic thresholds and tail estimates are not guarantees.

058. `docs/superpowers/reports/2026-09-05-vet-07-prompts-complete-data/capture.py:1–63` — Capture utility reads snapshots and HTTP endpoints;401 branch reads .env and mintsJWT; not offline safe, never executed.

059. `docs/superpowers/reports/2026-09-05-vet-07-prompts-data/q12_classify.py:1–55` — Priority regex prompt-refusal classifier counts rows, residualOTHER retained; not semantic complete attribution.

060. `docs/superpowers/reports/2026-09-05-vet-07-prompts-data/q14_tokens.py:1–59` — Tokenizer estimates use o200k/cl100k proxy encodings, not DeepSeek bill; runtime may retrieve encoding assets.

061. `docs/superpowers/reports/2026-09-05-vet-08-stretch-data/common.py:1–39` — Shared CT conversion,1m loads,5m aggregation and Wilder ATR; aggregation permits partial buckets.

062. `docs/superpowers/reports/2026-09-05-vet-08-stretch-data/complete-0905/analyze.py:1–56` — Completed replay analyzer preserves plan/version/scenario identities and unknown opportunities; no fabricated realized profits.

063. `docs/superpowers/reports/2026-09-05-vet-08-stretch-data/complete-0905/bounds.py:1–35` — Candidate fill bounds0..1 measure opportunity uncertainty, not queue/cancel/portfolio fill probability.

064. `docs/superpowers/reports/2026-09-05-vet-08-stretch-data/complete-0905/replay.go:1–161` — Go necessary-condition replay calls production kernel validators/confirm evaluators on closed prefixes; copied historical stop composition, not broker simulation.

065. `docs/superpowers/reports/2026-09-05-vet-08-stretch-data/q01_store_asof.sh:1–10` — Replay schema/as-of shell; no replay execution here.

066. `docs/superpowers/reports/2026-09-05-vet-08-stretch-data/q25_payoff.py:1–47` — Historical payoff stats use host-local naive era and nearest-stop association; no sentinel filter in this exploratory version.

067. `docs/superpowers/reports/2026-09-05-vet-08-stretch-data/q30_atr.py:1–29` — ATR enrichment uses host-local era and partial aggregation; older context helper.

068. `docs/superpowers/reports/2026-09-05-vet-08-stretch-data/replay.py:1–273` — Original Python replay advances120seconds but checks last1m only, uses mutable live stops, unsupported confirms false, no fees, open-end0pts; superseded qualification matters.

069. `docs/superpowers/reports/2026-09-05-vet-09-complete-data/final_validate.py:1–41` — Final archive validator checks six exports same58IDs, coverage/links/privacy patterns; not production parity.

070. `docs/superpowers/reports/2026-09-05-vet-09-top-ten-data/q01_verify.py:1–46` — Top-ten verification calls HTTPhealth and original cohort filter; not offline safe and not executed.

071. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/complete/recompute.py:1–77` — Completed idea recut pins58/12 and exports denominators, bootstrap/cost assumptions.

072. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/q03_entry_time_units.sh:1–14` — Shell compares timestamp units/entry observations; historical literal population.

073. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/q05_trades_deep.sh:1–18` — Shell deep-trade diagnostic queries historical rows; not an execution proof.

074. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/q06_planner_and_decisions.sh:1–22` — Shell planner/decision cadence uses LIKE and labels; multiple actions/empty labels can collapse.

075. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/q08_churn_silence.sh:1–20` — Shell churn/silence diagnostics count recorded events, not opportunities or actual broker cancels.

076. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/q12_exit_reasons.py:1–32` — Exit reasons inferred from nearby timestamp/price without side; native broker exit cause is separate.

077. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/r01_pop.sh:1–12` — Ideas population shell reconciles corrected exclusions.

078. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/r02_recut.py:1–66` — Ideas recut/mean intervals enumerate eligible historical groups; posthoc subgroups not validated edge.

079. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/r03_touch.sh:1–28` — Touch proxy joins and grouped diagnostics; no canonical immutable opportunity guarantee.

080. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/r04_more.sh:1–21` — Additional touch/snapshot diagnostic shell; interpretation remains descriptive.

081. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/r09_dedup.py:1–53` — Dedup uses lexical first outcome and narrow VWAP dynamic family; different dedup contracts yield different estimands.

082. `docs/superpowers/reports/2026-09-05-vet-10-ideas-data/wilson.py:1–10` — Wilson and approximate mean confidence interval helpers for static research.

083. `docs/superpowers/reports/2026-09-08-scenario-economics-data/measure.py:1–58` — Decimal frozen scenario geometry from pinned git cohort6095ca58 and DB snapshot; signed first target vs absolute arm R, no trade profitability claim.

084. `docs/superpowers/reports/2026-09-08-the-strategy-data/analyze.py:1–175` — Strategy style census preserves version/slot identities and ambiguous hold; publication close proxy and lifecycle cutoff; sample differs from Sep5 cohort.

085. `docs/superpowers/reports/2026-09-08-the-strategy-data/build_report.py:1–151` — Report generator mixes computed tables and hardcoded historical prose/runtime revision; not a self-updating current-system report.

086. `docs/superpowers/reports/2026-09-08-the-strategy-data/validate.py:1–34` — Style validator AST/fixture checks plus liveDBaccounts and.env privacy scans; not wholly offline despite header.

087. `docs/superpowers/reports/2026-09-10-scenario-level-identity-data/census.py:1–136` — Two read-only bounded DB snapshots with explicit non-atomic caveat; stage/schema and formation census.

088. `docs/superpowers/reports/2026-09-11-level-zones-evidence/overlap_replay.py:1–46` — Exploratory union-find overlap uses native overlap or anchor distance; not exact production non-transitive zone merge.

089. `docs/superpowers/reports/2026-09-11-one-setup-data/sectionC.py:1–118` — One-setup diagnostic uses host timezone, strategy_id=trader premise, fallback condition without version and nearby bar close; unresolved identity/causality.

090. `docs/superpowers/reports/2026-09-12-structural-stop/harness/audit_ledger.py:1–27` — Frozen200positive-price ledger geometry and plan inventory; copied database, not current account truth.

091. `docs/superpowers/reports/2026-09-12-structural-stop/harness/detect.go:1–146` — Closed-prefix level detection, production map helpers, allfresh/owner omissions; session profile availability/contract provenance not established by key alone.

092. `docs/superpowers/reports/2026-09-12-structural-stop/harness/summarize_overshoot.py:1–37` — Held-conditioned overshoot quantiles exclude breakouts; touchbar extremes upper bounds, training buffer not validated profitability.

093. `docs/superpowers/reports/2026-09-12-structural-stop/harness/sweep.go:1–209` — Corrected geometry sweep calls production frozen-zone core; fillbar included, adverse tick, stopgap and target-through handling; old missing per-trade-cap diagnostic superseded by daily-only owner rule.

094. `docs/superpowers/research/2026-09-12-backtest-structure-fade/harness/data.go:1–193` — Read-only DB opening and percontract/source bars; closed-prefix selectors and session windows; no explicit last-bar freshness or duplicate-time rejection.

095. `docs/superpowers/research/2026-09-12-backtest-structure-fade/harness/eval.go:1–334` — Band/line outcomes and trade model skipfillbar/exactstop;2pt friction, grossR; structural buffer grid and nearest-qualified target are research policy.

096. `docs/superpowers/research/2026-09-12-backtest-structure-fade/harness/extract.py:1–34` — JSON extraction/rendering from prior run outputs; not recomputation.

097. `docs/superpowers/research/2026-09-12-backtest-structure-fade/harness/seam.go:1–134` — Seam report counts and flags, ignores SQL/JSON errors; flags not asserted by caller; same-contract grouping check is not callsite parity.

098. `docs/superpowers/research/2026-09-12-backtest-structure-fade/harness/stats.go:1–154` — Wilson/independent difference/quantiles/RNG/streak helpers; maxDD starts at first cumulative observation; empty values often0.

099. `docs/superpowers/research/2026-09-12-backtest-zone-fade/harness/detect.go:1–146` — Same closed-prefix detector as structural-stop harness; allfresh inputs do not reproduce historical observation receipt.

100. `docs/superpowers/research/2026-09-12-backtest-zone-fade/harness/eval.go:1–264` — Original zone trade model skipsfillbar and exactstopgap;2point friction; touch and through variants remain hypothetical executions.

101. `docs/superpowers/research/2026-09-12-backtest-zone-fade/harness/extract.py:1–68` — Loads saved run and C4 rendering; output describes archived simulation only.

102. `docs/superpowers/research/2026-09-12-backtest-zone-fade/harness/main.go:1–653` — Zone main orchestrates sessions/maps/holdout and surfaces; bandtouchA alwaysfilled at anchor, C tick favorable; independent overlapping trades; startupport guard rejects changed production source.

103. `docs/superpowers/research/2026-09-12-backtest-zone-fade/harness/out.go:1–277` — C1/C4 grouping and rendering: net points*$2, grossR, PF0 for no losses; cumulative series omits initial0 so early drawdown missed.

104. `docs/superpowers/research/2026-09-12-backtest-zone-fade/harness/render_c1.py:1–19` — Markdown table renderer with n<30 decided flag; consume saved JSON, no computation of sample validity.

105. `docs/superpowers/research/2026-09-12-backtest-zone-fade/harness/stop_port.go:1–237` — Legacy stop widest-wins port and startup byteguard; current production variadic structural signature differs so replay aborts; guard compares embedded string, not callable copiedbody.
