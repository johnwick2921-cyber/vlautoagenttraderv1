# Section 2 complete audit evidence — 2026-09-05

Only Section 2 owns this directory. The parent integrates the docs branch. All scripts were created/run in `/home/hoang/nofx-analysis/vet-02-complete-0905`; detached worktree `/home/hoang/nofx-vet-02-complete` remains until integration. Source base: `488ce82748ca570804240630677c90d3055f128e`; claim branch `docs/vet-02-0905-complete`, NOFX_SESSION `codex-vet-02-complete-0905`.

This directory supersedes the **analytical use** of prior Section 2 data. Sibling legacy scripts/outputs remain untouched for provenance. In particular their wrong epoch, latest-plan fallback, future-range null sampling, differing RTH window, and uniform-across-bar volume allocation do not establish current-strategy outcomes. Read the correction ledger in the replacement report before reusing anything historical.

Reproduction (all output stays next to the script):

```bash
python3 /home/hoang/nofx-analysis/vet-02-complete-0905/audit.py
python3 /home/hoang/nofx-analysis/vet-02-complete-0905/summarize.py
python3 /home/hoang/nofx-analysis/vet-02-complete-0905/verify_details.py
```

`audit.py` opens `file:/home/hoang/nofx/data/data.db?mode=ro`, enables `PRAGMA query_only=ON`, and uses a read transaction. No store initialization, gate-jwt, writable database connection or runtime/order operation. It records raw inputs and outputs. `verify_details.py` uses the same read-only settings, exact plan versions and explicit reference-matching rules. **Run only from authorized scratch, not by writing outputs into a live application directory.** Neither script needs a network connection or API token.

`summarize.py` is fully offline against the saved JSON inputs; it recreates `evidence.txt` and `inventory.md`. The complete reference snapshot is preserved in `source_snapshot.txt` as `repo/path:line text` (trailing whitespace normalized). All code references in the report refer to the pinned source base, not future dev lines. JSON files are formatted to keep their record IDs inspectable. `SHA256SUMS` verifies the frozen bundle.

| Artifact | Purpose |
|---|---|
| `audit.py`, `audit-output.txt` | Query/read-only transaction, exact eligible P&L population, candidate and raw-touch census, fixed-horizon forward calculation, matching protocol. |
| `positions.json` | Included/excluded IDs, 58 usable trades, -466.428572 corrected P&L, 18/38/2, 12 CME days, no post-enforced-strict realized entries. |
| `candidate_rows.json`, `touch_rows_quarantined.json`, `plan_rows.json`, `bars_1m.json` | Frozen input records. Raw touches are contaminated and not tradable outcome evidence. Plans use SQLite rowid plus plan_id/version; bars use symbol/tf/open_time_ms. |
| `config_subset.json`, `details.json:trade_binding` | Minimal nonsensitive strategy settings and actual trader-strategy binding. Per-session min grade is nested in sessions; top-level NULL does not mean unset. |
| `forward_rows.json` | One row per recorded candidate/read, true missing cut score represented as NULL in analysis. Level price frozen at read. Read-minute excluded. Touch at next 60 complete bars; reaction measured on subsequent closes, never same touch bar. |
| `matched_pairs.json`, `summary.json` | Same-read, same-side, 50pt distance-bin match with 25pt caliper; 34 pairs on one day, repeated-price episodes explicitly counted. Not a strategy replay. |
| `inventory.json`, `inventory.md`, `evidence.txt` | Full inventory and per-kind raw/forward counts, all row IDs and ordinary Wilson95 intervals. E:n in the report refers to evidence.txt line n. |
| `verify_details.py`, `details-output.txt`, `details.json` | RTH-L exact-version authoring, arms and eligible-position reference matches; September 3 RTH low bar. Missing exact-version arms1–4 remain unclassified. |
| `grade_forward.json` | First-exposure seated grade A/B/C comparison; all 47 complete initially seated keys are A, B/C n=0. |
| `source_snapshot.txt` | Exact cited sources as path:line; does not alter source files. |
| `verification.txt` | Population, IDs, chronological/non-overlap checks and matching validation. |
| `health.json` | Public unauthenticated GET /api/health HTTP200; running revision36648655cfe0. No auth helper used. |

Method limits: first-exposure grouping is `(kind,price)`, not a reconstructed original formation ID. The first post-read touch can already belong to an ongoing price episode; it is not the production D1′ new-touch predicate. Dynamic lines are frozen and zones reduced to recorded midpoint. No initial risk, fill queue, costs, same-bar bracket ordering or stop/target scheme is inferred. Wilson intervals do not correct clustering; one day cannot support a day-block confidence interval. No full-universe ranking-versus-nearest policy return is claimed.

Primary sources checked through browsing on 2026-09-05:

- [Osler 2000, NY Fed full paper](https://www.newyorkfed.org/medialibrary/media/research/epr/00v06n2/0007osle.pdf): published FX levels and intraday trend interruptions; not MNQ weight validation.
- [Osler working paper](https://www.newyorkfed.org/medialibrary/media/research/staff_reports/sr125.pdf) and [publication record](https://www.newyorkfed.org/research/staff_reports/sr125.html): bank FX order clustering, later Journal of Finance2003; different instrument, data and geometry.
- [TradingView volume-profile definitions](https://www.tradingview.com/support/solutions/43000502040-volume-profile-indicators-basic-concepts/) and [TPO definitions](https://www.tradingview.com/support/solutions/43000713306-time-price-opportunity-tpo-indicator/): vendor definitions only, not outcome studies.
- [CME volume education](https://www.cmegroup.com/education/courses/introduction-to-futures/what-is-volume): activity and interpretation limits, not level-specific profitability.
