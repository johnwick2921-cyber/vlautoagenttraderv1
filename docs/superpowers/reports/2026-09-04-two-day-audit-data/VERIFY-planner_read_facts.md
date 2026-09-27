# Adversarial verify — planner_read_facts (peer claim: 17 rows, all 09-03, n=0 for 09-02)

## Verdict: PLAUSIBLE — the NUMBERS reproduce exactly; the CLASSIFICATION ("A / DEFECT") is wrong.

### 1. Numbers reproduce exactly [A]
```
sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro" \
  "SELECT count(*), min(datetime(created_at,'-5 hours')), max(datetime(created_at,'-5 hours')) FROM planner_read_facts;"
-> 17|2026-09-03 00:00:56|2026-09-03 20:24:03
```
Full dump: ids 1..17 contiguous (no trim; PlannerReadFactsCap=500, store/planner_read_facts.go:66).
All 17 rows trade_date='2026-09-03'. Zero rows for 2026-09-02. No TZ error: raw
created_at carries an explicit '+00:00' suffix; row 17 raw = 2026-09-04
01:24:03Z -> 2026-09-03 20:24:03 CT, and the file log carries the SAME line at
`09-03 20:24:03 ... 📓 read facts` (nofx_2026-09-03.log:20190). CSV:
planner_read_facts_all_rows.csv

### 2. The zero is a DEPLOY DATE, not a write failure [A]
Writer `persistReadFacts` first appears in commit 4659874a (2026-09-02 21:52:07 CDT).
```
git show 1cee77a8:trader/auto_trader_planner.go | grep -c persistReadFacts  -> 0
git show 60f214d9:trader/auto_trader_planner.go | grep -c persistReadFacts  -> 3
```
- RELEASE 1cee77a8 booted 09-02 22:37:38 CT (nofx_2026-09-02.log:62066; marker cf8ed4f4) — writer ABSENT.
- RELEASE 60f214d9 booted 09-02 22:41:58 CT (nofx_2026-09-02.log:62424; marker 466ca82c) — writer PRESENT.
- LAST planner read on 09-02 CT: 22:04:51 (`🧠 planner model`), i.e. 37 min BEFORE the writer went live.
- FIRST planner read after it: 09-03 00:00:56 CT -> row id 1.
Zero rows on 09-02 is arithmetically required. Zero `read-facts write failed`
warnings in either log file and 0 matching log_events rows.

### 3. Coverage, with n [A]
`🧠 planner model` lines (one per planner read), both log files:
  09-02 CT: n=40 reads, 0 read-facts rows (writer not in any running binary)
  09-03 CT: n=17 reads, 17 read-facts rows -> 17/17 = 100%
CSV: planner_reads_from_logs_0902_0903.csv

### 4. "no bias_regime figure of ANY KIND for that day" is over-broad [A]
`plans.doc.bias_label` carries a regime word (kernel.RegimeCallWord = TrendDaily
+Trend1h -> up/down/neutral) and 2 of the 33 plans with trade_date='2026-09-02'
carry it (ASIA v14 written 09-03 00:08:45 CT "regime neutral", ASIA v15 09-03
00:34:14 CT "regime neutral"). NOTE this is a DIFFERENT function from
planner_read_facts.bias_regime (= TrendDaily + "/" + ATRRegime).

### 5. The real defect the peer walked past [A]
`persistReadFacts` (trader/auto_trader_planner.go:2419-2455) sets 13 of 18
fields. It NEVER sets BiasAI, BiasTree, PlanID, Version, TokensIn:
```
SELECT DISTINCT quote(bias_ai), quote(bias_tree), quote(bias_regime),
       quote(plan_id), quote(version), quote(tokens_in) FROM planner_read_facts;
-> ''|''|'up/NORMAL'|''|0|0     (ONE row -> every column is invariant, n=17)
```
- bias_ai / bias_tree are dead columns although the table's own doc comment
  (store/planner_read_facts.go:19) says it records "the bias labels".
  The AI-vs-tree disagreement the table exists to capture is 2/3 absent.
- plan_id='' and version=0 on ALL 17 -> a read-facts row cannot be joined to the
  plan it fed. (The struct comments call these "not yet written"/"unknown at
  render time", and nothing ever backfills them: SaveReadFact is the only
  writer, no update path.)
- bias_regime = 'up/NORMAL' on all 17 rows spanning 09-03 00:00:56-20:24:03 CT,
  a day whose MNQ range was 29075.00-29585.00 (510 pts) and which contained the
  only fill of the window (position 591, SHORT, -140.0).
  Whether that invariance is a stuck computation or a genuinely trend-up/normal-
  ATR day CANNOT BE DISTINGUISHED from the stored data — the inputs
  (Regime.TrendDaily, Regime.ATRRegime) are not persisted anywhere else.
