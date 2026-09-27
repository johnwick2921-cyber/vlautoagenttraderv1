#!/bin/bash
# q01 — re-measure every store premise in the dispatch GROUND TRUTH
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
S() { sqlite3 -header "$DB" "$1"; }
echo "## row counts"
for t in trader_positions armed_orders plans plan_lifecycle_log touch_outcomes candidate_pool trade_excursions decision_records ab_confirm_log nt8_order_snapshots bars planner_rejected_prompts planner_read_facts level_state level_stats touch_episodes trader_fills trader_orders; do
  printf "%-26s %s\n" "$t" "$(sqlite3 "$DB" "SELECT COUNT(*) FROM $t")"
done
echo "## trader_positions era (entry_time >= 2026-08-15 CT epoch ms)"
S "SELECT COUNT(*) all_rows, SUM(entry_time>=strftime('%s','2026-08-15 05:00:00')*1000) era_rows, SUM(entry_time>=strftime('%s','2026-08-15 05:00:00')*1000 AND pnl_corrected IS NOT NULL) era_pnlc, SUM(entry_time>=strftime('%s','2026-08-15 05:00:00')*1000 AND plan_id IS NOT NULL) era_plan, SUM(entry_time>=strftime('%s','2026-08-15 05:00:00')*1000 AND cited_scenario_id IS NOT NULL AND cited_scenario_id<>'') era_cited, SUM(pnl_corrected IS NULL) pnlc_null_all, SUM(entry_time>=strftime('%s','2026-08-15 05:00:00')*1000 AND mae IS NOT NULL) era_mae, SUM(entry_time>=strftime('%s','2026-08-15 05:00:00')*1000 AND mfe IS NOT NULL) era_mfe FROM trader_positions;"
echo "## typeof entry_time"
S "SELECT typeof(entry_time) t, COUNT(*) FROM trader_positions GROUP BY 1;"
S "SELECT source, COUNT(*) FROM trader_positions GROUP BY 1;"
S "SELECT plan_session, COUNT(*) FROM trader_positions GROUP BY 1;"
S "SELECT status, COUNT(*) FROM trader_positions GROUP BY 1;"
echo "## touch_outcomes outcome"
S "SELECT outcome, COUNT(*) FROM touch_outcomes GROUP BY 1;"
echo "## store as-of (max created_at)"
for t in trader_positions armed_orders plans plan_lifecycle_log touch_outcomes candidate_pool decision_records ab_confirm_log planner_rejected_prompts planner_read_facts; do
  col=created_at; [ "$t" = plan_lifecycle_log ] && col=at
  printf "%-26s %s\n" "$t" "$(sqlite3 "$DB" "SELECT MAX($col) FROM $t")"
done
echo "bars max open_time_ms CT: $(sqlite3 "$DB" "SELECT datetime(MAX(open_time_ms)/1000,'unixepoch','-5 hours') FROM bars")"
echo "decision_records max timestamp: $(sqlite3 "$DB" "SELECT MAX(timestamp) FROM decision_records")"
