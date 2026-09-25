# Section 8 complete audit evidence

Source revision: b4376246c2c502ecedd119c6a44a27956ed2f616. These files supersede the old parent directory's replay/payoff conclusions. `report.md` is the same text as the report at the parent reports path. The retained CSV/JSON files permit offline reproduction without production access.

All analysis scripts were authored/run under `/home/hoang/nofx-analysis/vet-08-complete-0905`. SQLite extraction opens `file:/home/hoang/nofx/data/data.db?mode=ro`, sets `PRAGMA query_only=on` and uses one transaction. It never calls store.New. `extract.py` embeds each query beside the corresponding output. `logs.py` reads retained logs and preserves original path/line anchors; mirrored `.en` NT8 logs are excluded from counts. `rule_config.json` retains only nonsecret fields read recursively from the matching strategy's config; it is an as-of observation, not a history of resolved runtime settings.

Reproduce using a scratch copy of this directory and a checkout of the pinned source. The Go source imports only pure kernel/market functions; the two unexported arithmetic functions are copied verbatim and checked by verify.py. Run from the checkout so its source remains the module root; use the external modfile to avoid edits to repository module files. No package main for the application or gate-jwt is invoked.

```bash
# Optional fresh read-only extraction (changes the as-of data):
python3 /home/hoang/nofx-analysis/vet-08-complete-0905/extract.py
python3 /home/hoang/nofx-analysis/vet-08-complete-0905/logs.py

# Offline run from the pinned repository checkout:
GOTOOLCHAIN=local \
GOCACHE=/home/hoang/nofx-analysis/vet-08-complete-0905/go-cache \
GOPROXY=off \
/home/hoang/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.3.linux-amd64/bin/go run \
-mod=readonly \
-modfile=/home/hoang/nofx-analysis/vet-08-complete-0905/offline.mod \
/home/hoang/nofx-analysis/vet-08-complete-0905/replay.go \
/home/hoang/nofx-analysis/vet-08-complete-0905
python3 /home/hoang/nofx-analysis/vet-08-complete-0905/analyze.py
python3 /home/hoang/nofx-analysis/vet-08-complete-0905/bounds.py
python3 /home/hoang/nofx-analysis/vet-08-complete-0905/reaper.py
python3 /home/hoang/nofx-analysis/vet-08-complete-0905/verify.py /home/hoang/nofx-vet-08-complete
```

`replay_checkpoints.csv`: independent retained-minute necessary checks, not actual callbacks. `replay_static.csv`: disabled/validated scenarios and legs. `replay_opportunities.csv`: all enabled opportunities including those with no passing endpoint. `reach_bounds.csv`: independent price reach after a qualifying checkpoint; fill-indicator envelopes are conditional on that sampled information model and not an upper bound on a real full-book replay. No later P&L is computed. `reaper_observed.csv`: current three-valued predicate at the observed final stale-cancel timestamp; uses the preceding received snapshot and a 30-second configured interval, assumes cache continuity. It cannot reconstruct hypothetical broker inventory.

`sessions.csv` includes session fragments and missing data explicitly. Bar rowids need not increase with time because historical inserts occurred later: use open_time_ms and convention for durable joins. A present bar does not establish it was available live. `opportunity_ledger.csv` is the unique 188-key audit currency and includes all arm, intent, refusal and position IDs.

`q2_decisions_before_1000.csv` retains the actual pre-cutoff analytical text. `q2_news_at_time.json` is the result of a limited news-term search in decisions37097/37098, not proof that no event was known elsewhere. `q2_atr.json` is directly calculated by source. The report distinguishes these from final bar availability.

Primary reference pages consulted September 5, 2026:

- ISM release calendar: https://www.ismworld.org/supply-management-news-and-reports/reports/rob-report-calendar/ — September 3 Services date; generic Eastern-time label caveat preserved in report.
- BLS 2026 schedule: https://www.bls.gov/schedule/2026/ — September 4 Employment Situation 08:30 Eastern.
- NinjaTrader official CreateOrder signature: https://ninjatrader-staging.ninjatrader.com/support/helpguides/nt8/createorder.htm — limitPrice precedes stopPrice. No research study is used to validate an MNQ entry.

`manifest.sha256` covers retained files except itself and verification output. No cache, executable, production DB, credential, token or complete runtime configuration is included.
