# Section 5 complete evidence — PRIMARY, 58 eligible positions

These files supersede all conflicting performance, fill, excursion and exit-reason outputs in the parent historical data directory. The source base is b4376246, the branch docs/vet-05-0905-complete. The report explicitly withdraws the old 65-row tables and recommendations derived from them. Preserved historical q01–q34 files outside `complete/` are provenance only and must not be used as primary population results.

Scratch: `/home/hoang/nofx-analysis/vet-05-complete-0905`. Worktree retained for parent: `/home/hoang/nofx-vet-05-complete`.

## Reproduce only in the authorized scratch directory

Run these Python files from scratch, in this order; every SQLite connection uses mode=ro AND query_only. No application imports/store initialization, authentication token creation or order commands occur.

```
python3 q36_logs.py > q36_logs.out
python3 q11_fill_vs_bar.py > q11_fill_vs_bar.out
python3 q14_mae_mfe.py > q14_mae_mfe.out
python3 q31_verified.py > q31_verified.out
python3 q32_sources.py > q32_sources.out
python3 q33_metrics.py > q33_metrics.out
python3 q34_integration.py > q34_integration.out
python3 q35_complete.py > q35_complete.out
python3 q37_sources.py > q37_sources.out
python3 validate_evidence.py > validation.out
```

Era start1786770000000; end1788584400000 exclusive. q31 enforces plan sentinel exclusion plus test/null/unresolved exclusions and asserts58 rows/−466.428572. q35 independently checks the exact13 excluded ids. q34 rolls CME days at17:00CT and checks records after the supplied09-03 11:10CT strict boundary. A future changed data store may fail frozen-cohort assertions; that is intentional, not permission to edit eligibility.

`eligible_fill_audit.csv` has116 rows, one entry and exit per eligible position, including missing bars and uncertain timestamps. This does not mean116 complete exchange execution records. Entry source time is preserved; nearest fill candidates are not execution-id proofs. q14 produces only exit-label provenance and no invented initial-risk columns. q31 floors are closed-bar simple-TR proxies, not the engine ATR or initial stop. MAE/MFE position fields are proxies; trade_excursions remains empty. q35 uses a consistent scenario-identity funnel while preserving attempts separately.

`raw_*sources.out` gives original raw-log paths/line numbers. `q32_sources.out` and `q37_sources.out` give code and NT8 source lines plus selected nonsecret config evidence. `stop_snapshot_states.json` records the first distinct snapshot state for audited stop signals; ongoing snapshot counts may increase during reproduction without changing the frozen trading cohort. q33 volume buckets are a separate bar-context sample, never a trade population.

No live fill distribution, queue estimate, initial-R population distribution or capped-entry/thesis-exit counterfactual is manufactured. The invalidity of a historical calculation is not repaired by merely labeling it a proxy; the current report removes unsupported return/savings conclusions entirely.
