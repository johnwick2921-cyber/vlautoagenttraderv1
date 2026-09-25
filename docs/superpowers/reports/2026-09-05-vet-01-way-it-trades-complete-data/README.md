# Section 1 complete evidence — 2026-09-05

This directory is the active evidence for the rewritten Section 1 report at base b4376246. The previous `2026-09-05-vet-01-way-it-trades-data` folder is historical and superseded; its 65-row primary population and inferred-risk outputs must not be used as current conclusions.

All current calculations use entry_time >=1786770000000 and exclude test source, UNRESOLVABLE plan ID, and NULL pnl_corrected. Exact eligible n=58; sum -466.428572; 18W/38L/2F; 12 occupied CME 17:00 CT days. Scripts fail if these invariants do not hold for the captured sample.

## Reproduce without production access

Copy this directory into the authorized scratch location `/home/hoang/nofx-analysis/vet-01-complete-0905` and run there:

```sh
python3 audit.py
python3 supplement.py
python3 legacy_check.py
python3 render_report.py
```

These commands use preserved local inputs. `inputs.json.gz` contains the exact selected SQLite and parsed broker rows used in this run. `legacy_floor_input.csv` and `legacy_plan_geometry.csv` preserve separately labeled historical claim inputs. No trading code is imported. `audit.py --extract` is a separate explicitly read-only live extraction using mode=ro, query_only=ON and a read transaction; it will capture a new snapshot rather than recreate this one. `collect_sources.py` and `binding.py` are read-only source/health and safe-field binding extraction scripts, not required for offline reproduction.

## Evidence identity and interpretation

- `trades.csv`: 58 positions, complete plan/scenario join, exact R provenance or missing reason, causal RV bar IDs.
- `trade_stats.csv`: all groups, W/L/F, n, Wilson intervals, descriptive mean intervals, separate R n/IDs, complete row membership. Each cell n<30 has NO VERDICT; larger n does not cure selection.
- `touch_keys.csv`: 423 price-time keys, all 677 raw members, conservative conflict handling, observed ordinal reconstruction and plan-asof sensitivity. Plan-asof is not level formation.
- `touch_stats.csv`: complete kind × observed ordinal matrix plus raw stored-ordinal sensitivity; hold/break/ambiguous n and Wilson bounds. No tradable edge verdict.
- `planned_arms.csv`, `target_obstacles.csv`: exact plan row/version/scenario geometry; same-document level matches and intervening directional levels, not machine-formation validation.
- `broker_R.csv`: nine initial broker stops and exact exit-child evidence where matched. Initial risk never comes from a mutable arm stop or exit loss.
- `floor_path_sensitivity.csv`: independent closed-bar ATR diagnostic, all contributing bar row IDs and explicit gaps. Not actual runtime floor or ordered target-hit probability.
- `zero_mae_sensitivity.csv`: raw winner/reject MAE quantiles and sensitivity excluding uncertain reconcile winner zeros 569/584; no zero-as-no-adverse claim, primary n58 preserved. The E4 NULL-on-uncomputed fix and excursion writer/hooks are already shipped, not new implementation proposals.
- `excursion_proxies.csv`: unordered, potentially boundary-contaminated/censored position MFE/MAE only. Actual trade_excursions has zero rows.
- `legacy_check.txt`: exact reproduction of old 17-scenario planned median and 3-of-36 proxy claim, eligible intersection 3/33. These legacy statistics are not the corrected primary experiment.
- `results.txt`, `summary.json`, `supplement.txt`: line-addressable results and methodology metadata; all probabilities include denominators and Wilson bounds.
- `source_evidence.txt`, `binding.txt`: exact source file:line, boot revision, safe settings and unauthenticated health receipt.

Broker log timestamps are interpreted as host CT. Source extraction retains file:line, order/signal IDs and numerical order states, without login/connection or balance messages. The frozen input contains no authentication tokens. All published P&L aggregates use pnl_corrected, including risk-normalized results. Fees are recorded as zero; no invented commission schedule is applied.

Scripts and outputs are Section 1 documentation evidence only. No strategy changes, database writes, production token helper, deployment, runtime restart or order changes occurred. Parent owns merging/integration; the author's worktree is retained until that integration.
