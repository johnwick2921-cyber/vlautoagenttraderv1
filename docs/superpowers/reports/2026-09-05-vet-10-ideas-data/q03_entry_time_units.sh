#!/bin/bash
# q03: why 71 not 227 — entry_time unit / magnitude distribution
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
echo "--- length(entry_time) histogram"
sqlite3 "$DB" "select length(cast(entry_time as text)) L, typeof(entry_time) T, count(*), min(entry_time), max(entry_time) from trader_positions group by 1,2"
echo "--- rows by month (ms assumption)"
sqlite3 "$DB" "select strftime('%Y-%m', entry_time/1000,'unixepoch','-5 hours') m, count(*), sum(pnl_corrected is not null) pnl_nn, sum(plan_id is not null) plan_nn, sum(mae is not null) mae_nn from trader_positions group by 1 order by 1"
echo "--- created_at based era (created_at epoch ms?)"
sqlite3 "$DB" "select typeof(created_at), length(cast(created_at as text)), count(*) from trader_positions group by 1,2"
sqlite3 "$DB" "select strftime('%Y-%m', created_at/1000,'unixepoch','-5 hours') m, count(*) from trader_positions group by 1 order by 1"
echo "--- pnl_corrected non-null all-time by month"
sqlite3 "$DB" "select count(*) from trader_positions where pnl_corrected is not null"
echo "--- rows with plan_id by month"
sqlite3 "$DB" "select strftime('%Y-%m-%d', entry_time/1000,'unixepoch','-5 hours') d, count(*) from trader_positions where plan_id is not null group by 1 order by 1"
