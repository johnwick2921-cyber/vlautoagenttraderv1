#!/bin/bash
# q03 — daily distribution of closed positions Aug-Sep 2026 (CT), to define the era and explain "227"
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
S() { echo "--- $1"; sqlite3 -header "$DB" "$2"; }
S "per CT day: rows, pnl_corrected non-null, sources, plan_id set, min id, max id" "SELECT date(entry_time/1000,'unixepoch','-5 hours') d, COUNT(*) n, SUM(pnl_corrected IS NOT NULL) pc, SUM(source='system') sys, SUM(source='reconcile') rec, SUM(source='armed_entry') arm, SUM(source='e7_farside_test') e7, SUM(plan_id IS NOT NULL AND plan_id<>'') pid, MIN(id) mn, MAX(id) mx, ROUND(SUM(pnl_corrected),2) sum_pc FROM trader_positions WHERE entry_time >= 1785906000000 GROUP BY 1 ORDER BY 1;"
S "id 361..520: source, plan_id, pnl_corrected null" "SELECT source, SUM(plan_id IS NOT NULL AND plan_id<>'') pid, SUM(pnl_corrected IS NOT NULL) pc, COUNT(*) FROM trader_positions WHERE id BETWEEN 361 AND 520 GROUP BY 1;"
S "all-time count by plan_id set" "SELECT (plan_id IS NOT NULL AND plan_id<>'') pid, COUNT(*), SUM(pnl_corrected IS NOT NULL) FROM trader_positions GROUP BY 1;"
S "first row with plan_id set" "SELECT id, datetime(entry_time/1000,'unixepoch','-5 hours'), plan_id, source FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' ORDER BY id LIMIT 3;"
S "the 4 era NULL pnl_corrected rows" "SELECT id, datetime(entry_time/1000,'unixepoch','-5 hours') et, source, side, entry_price, exit_price, realized_pnl, substr(pnl_correction_note,1,80) note, plan_session FROM trader_positions WHERE entry_time >= 1786770000000 AND pnl_corrected IS NULL;"
S "the 3 e7 rows" "SELECT id, datetime(entry_time/1000,'unixepoch','-5 hours') et, source, pnl_corrected, plan_session FROM trader_positions WHERE source='e7_farside_test';"
