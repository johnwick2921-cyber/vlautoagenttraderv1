#!/bin/bash
# q01 — store census: verify the dispatch's GROUND TRUTH premises on trader_positions etc.
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
S() { echo "--- $1"; sqlite3 "$DB" "$2"; }
S "all-time rows / status" "SELECT COUNT(*), status FROM trader_positions GROUP BY status;"
S "entry_time typeof sample" "SELECT typeof(entry_time), COUNT(*) FROM trader_positions GROUP BY 1;"
S "rows entry_time >= 2026-08-15 CT (epoch ms 1786899600000 = 2026-08-15 00:00 CT)" "SELECT COUNT(*) FROM trader_positions WHERE entry_time >= 1786899600000;"
S "created_at >= 2026-08-15" "SELECT COUNT(*) FROM trader_positions WHERE created_at >= '2026-08-15';"
S "era rows: pnl_corrected NULL / non-NULL" "SELECT (pnl_corrected IS NULL) AS isnull, COUNT(*) FROM trader_positions WHERE entry_time >= 1786899600000 GROUP BY 1;"
S "era rows by source" "SELECT source, COUNT(*) FROM trader_positions WHERE entry_time >= 1786899600000 GROUP BY 1;"
S "era rows by plan_session" "SELECT COALESCE(plan_session,'NULL'), COUNT(*) FROM trader_positions WHERE entry_time >= 1786899600000 GROUP BY 1;"
S "all-time pnl_corrected NULL count" "SELECT COUNT(*) FROM trader_positions WHERE pnl_corrected IS NULL;"
S "era rows with plan_id / cited_scenario_id" "SELECT SUM(plan_id IS NOT NULL), SUM(cited_scenario_id IS NOT NULL) FROM trader_positions WHERE entry_time >= 1786899600000;"
S "era mae/mfe populated" "SELECT SUM(mae IS NOT NULL AND mae<>0), SUM(mfe IS NOT NULL AND mfe<>0), COUNT(*) FROM trader_positions WHERE entry_time >= 1786899600000;"
S "min/max id, entry_time CT of era" "SELECT MIN(id), MAX(id), datetime(MIN(entry_time)/1000,'unixepoch','-5 hours'), datetime(MAX(entry_time)/1000,'unixepoch','-5 hours') FROM trader_positions WHERE entry_time >= 1786899600000;"
S "table counts" "SELECT 'armed_orders',COUNT(*) FROM armed_orders UNION ALL SELECT 'plans',COUNT(*) FROM plans UNION ALL SELECT 'plan_lifecycle_log',COUNT(*) FROM plan_lifecycle_log UNION ALL SELECT 'touch_outcomes',COUNT(*) FROM touch_outcomes UNION ALL SELECT 'candidate_pool',COUNT(*) FROM candidate_pool UNION ALL SELECT 'decision_records',COUNT(*) FROM decision_records UNION ALL SELECT 'ab_confirm_log',COUNT(*) FROM ab_confirm_log UNION ALL SELECT 'nt8_order_snapshots',COUNT(*) FROM nt8_order_snapshots UNION ALL SELECT 'trade_excursions',COUNT(*) FROM trade_excursions UNION ALL SELECT 'bars',COUNT(*) FROM bars UNION ALL SELECT 'planner_rejected_prompts',COUNT(*) FROM planner_rejected_prompts UNION ALL SELECT 'planner_read_facts',COUNT(*) FROM planner_read_facts UNION ALL SELECT 'trader_positions',COUNT(*) FROM trader_positions;"
S "as-of: max created_at per table" "SELECT 'trader_positions',MAX(created_at),MAX(updated_at) FROM trader_positions UNION ALL SELECT 'armed_orders',MAX(created_at),MAX(updated_at) FROM armed_orders UNION ALL SELECT 'plans',MAX(created_at),'' FROM plans UNION ALL SELECT 'decision_records',MAX(timestamp),'' FROM decision_records UNION ALL SELECT 'ab_confirm_log',MAX(created_at),'' FROM ab_confirm_log UNION ALL SELECT 'bars',datetime(MAX(open_time_ms)/1000,'unixepoch','-5 hours'),'' FROM bars;"
S "pnl_correction_note UNRESOLVABLE count (era)" "SELECT COUNT(*), SUM(pnl_corrected IS NULL) FROM trader_positions WHERE entry_time >= 1786899600000 AND (pnl_correction_note LIKE '%UNRESOLV%');"
S "distinct pnl_correction_note prefixes (era)" "SELECT substr(COALESCE(pnl_correction_note,'<null>'),1,40), COUNT(*) FROM trader_positions WHERE entry_time >= 1786899600000 GROUP BY 1 ORDER BY 2 DESC LIMIT 15;"
