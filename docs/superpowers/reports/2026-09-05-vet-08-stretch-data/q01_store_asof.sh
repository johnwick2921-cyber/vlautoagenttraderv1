#!/bin/bash
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
for t in trader_positions armed_orders plans plan_lifecycle_log touch_outcomes candidate_pool trade_excursions decision_records ab_confirm_log nt8_order_snapshots planner_rejected_prompts planner_read_facts bars; do
  case $t in
    bars) q="SELECT '$t', COUNT(*), datetime(MAX(open_time_ms)/1000,'unixepoch','-5 hours')||' CT (max open_time)' FROM bars";;
    plan_lifecycle_log) q="SELECT '$t', COUNT(*), MAX(at) FROM $t";;
    *) q="SELECT '$t', COUNT(*), MAX(created_at) FROM $t";;
  esac
  sqlite3 "$DB" "$q"
done
