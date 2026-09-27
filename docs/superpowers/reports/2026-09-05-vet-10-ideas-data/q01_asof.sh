#!/bin/bash
# q01: store as-of — row counts + max created_at for tables used in section 10
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
for t in trader_positions armed_orders plans plan_lifecycle_log touch_outcomes candidate_pool trade_excursions decision_records ab_confirm_log nt8_order_snapshots bars planner_rejected_prompts planner_read_facts level_stats touch_episodes trader_fills trader_orders; do
  c=$(sqlite3 "$DB" "select count(*) from $t")
  m=$(sqlite3 "$DB" "select max(created_at) from $t" 2>/dev/null || echo n/a)
  echo "$t|$c|$m"
done
echo "bars max open_time_ms CT: $(sqlite3 "$DB" "select datetime(max(open_time_ms)/1000,'unixepoch','-5 hours') from bars")"
echo "trader_positions max entry_time CT: $(sqlite3 "$DB" "select datetime(max(entry_time)/1000,'unixepoch','-5 hours') from trader_positions")"
