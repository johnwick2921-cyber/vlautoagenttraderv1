#!/bin/bash
# q02 — which cut yields the dispatch's "227 rows entry_time >= 2026-08-15"? and pin the CT epoch.
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
S() { echo "--- $1"; sqlite3 "$DB" "$2"; }
python3 - <<'PY'
import datetime, zoneinfo
ct = zoneinfo.ZoneInfo("America/Chicago")
for d in ("2026-08-15","2026-08-19","2026-09-02"):
    t = datetime.datetime.fromisoformat(d+"T00:00:00").replace(tzinfo=ct)
    print(d, "00:00 CT epoch_ms =", int(t.timestamp()*1000))
t = datetime.datetime(2026,9,2,7,49,tzinfo=ct); print("0B cut 2026-09-02 07:49 CT =", int(t.timestamp()*1000))
PY
S "count entry_time >= 2026-08-15 00:00 CT (1786856400000)" "SELECT COUNT(*) FROM trader_positions WHERE entry_time >= 1786856400000;"
S "count entry_time >= 2026-08-15 00:00 UTC (1786838400000)" "SELECT COUNT(*) FROM trader_positions WHERE entry_time >= 1786838400000;"
S "rows by CT month of entry_time" "SELECT strftime('%Y-%m', entry_time/1000,'unixepoch','-5 hours') m, COUNT(*) FROM trader_positions GROUP BY 1;"
S "what cut gives 227? created_at ranks" "SELECT COUNT(*) FROM trader_positions WHERE created_at >= 1786856400000;"
S "typeof created_at" "SELECT typeof(created_at), COUNT(*) FROM trader_positions GROUP BY 1;"
S "count id >= 361 (587-227+1)" "SELECT MIN(id), datetime(MIN(entry_time)/1000,'unixepoch','-5 hours') FROM (SELECT id, entry_time FROM trader_positions ORDER BY id DESC LIMIT 227);"
S "cited_scenario_id: null vs empty (era)" "SELECT CASE WHEN cited_scenario_id IS NULL THEN 'NULL' WHEN cited_scenario_id='' THEN 'EMPTY' ELSE 'SET' END, COUNT(*) FROM trader_positions WHERE entry_time >= 1786856400000 GROUP BY 1;"
S "plan_id: null vs empty (era)" "SELECT CASE WHEN plan_id IS NULL THEN 'NULL' WHEN plan_id='' THEN 'EMPTY' ELSE 'SET' END, COUNT(*) FROM trader_positions WHERE entry_time >= 1786856400000 GROUP BY 1;"
S "pnl_corrected null (era, entry_time cut)" "SELECT SUM(pnl_corrected IS NULL), SUM(pnl_corrected IS NOT NULL), COUNT(*) FROM trader_positions WHERE entry_time >= 1786856400000;"
S "entry_time vs created_at gap (era) seconds: min/median-ish/max" "SELECT MIN((created_at-entry_time)/1000.0), MAX((created_at-entry_time)/1000.0), AVG((created_at-entry_time)/1000.0) FROM trader_positions WHERE entry_time >= 1786856400000;"
S "reconcile rows: id, entry CT, created CT, pnl_corrected" "SELECT id, datetime(entry_time/1000,'unixepoch','-5 hours'), datetime(created_at/1000,'unixepoch','-5 hours'), pnl_corrected, plan_session FROM trader_positions WHERE entry_time >= 1786856400000 AND source='reconcile';"
