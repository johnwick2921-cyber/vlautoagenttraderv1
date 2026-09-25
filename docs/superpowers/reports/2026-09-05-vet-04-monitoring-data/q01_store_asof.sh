#!/usr/bin/env bash
# q01: row counts + max created_at per table used by section 04 (store as-of)
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
for t in trader_positions armed_orders plans plan_lifecycle_log touch_outcomes candidate_pool trade_excursions decision_records ab_confirm_log nt8_order_snapshots bars planner_rejected_prompts planner_read_facts trader_fills trader_orders system_config level_state level_stats touch_episodes; do
  col=created_at
  case $t in bars) col=open_time_ms;; nt8_order_snapshots) col=received_at_ms;; system_config) col='NULL';; plan_lifecycle_log) col=at;; decision_records) col=timestamp;; esac
  sqlite3 "$DB" "SELECT '$t', COUNT(*), MAX($col) FROM $t;" 2>&1
done
