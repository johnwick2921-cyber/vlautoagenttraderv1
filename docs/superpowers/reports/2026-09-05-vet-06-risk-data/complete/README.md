# Authoritative section 6 corrected evidence

This directory supersedes all numerical conclusions and recommendations in older section 6 evidence directories. Primary population: 58 eligible positions, 12 active CME session-days, corrected P&L -466.428572. Base origin/dev b4376246.

`population.sql`, `trade_sample.csv`, `days.csv`, `extraction.json`: read-only snapshot, exact rows, exclusion reasons and state allowlist. No credentials exported.

`recompute.py`, `results.json`, `evidence.txt`: primary day-block Monte Carlo, 100,000 paths, seed 2026090506. `--sample` reproduces without a database; use a dedicated scratch `--out`, never production directories. Offline results were verified byte-for-byte. NumPy 2.4.4, SciPy 1.17.1. `validate.py` is the independent stdlib validation used in scratch; its final assertion additionally expects the scratch offline-check/results.json, produced by the report's reproduction command.

`context_evidence.py`, `context.json`, `source-evidence.txt`: bounded code excerpts and read-only source context. The optional context recollection script names this worktree and store explicitly; static hashes are the default files, not proof of environment override values. `health.json`: sole unauthenticated HTTP read, status 200 recorded in report.

`legacy_mc_drawdown.py` is an unchanged copy of the original rig; `legacy_mc_drawdown.txt`, `drawdown_paths.csv`, `day_sim.csv` are CORRECTED-POPULATION compatibility outputs, secondary to day-block results. Its exact streak recursion excludes flats but its iid paths include them; do not use its sample-size/flat-inconsistent inference or trade-cap experiments as risk policy. Original ~/nofx-analysis/mc-drawdown remains untouched.

`build_report.py` generated the completed report from this lane's scratch results. Run only after adapting its explicit paths if reproducing elsewhere. `primary-sources.md` bounds external facts. `validation.txt` records completed checks. `SHA256SUMS` identifies this evidence snapshot.

CSV and source-excerpt line endings are normalized to LF for repository review; numerical values are unchanged. The legacy output is saved as .txt because repository ignores .out files.
