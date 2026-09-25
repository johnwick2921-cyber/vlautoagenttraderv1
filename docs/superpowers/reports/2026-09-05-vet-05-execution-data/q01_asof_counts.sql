.mode column
.headers on
SELECT 'trader_positions' t, COUNT(*) n, datetime(MAX(created_at)) max_created FROM trader_positions
UNION ALL SELECT 'armed_orders', COUNT(*), MAX(updated_at) FROM armed_orders
UNION ALL SELECT 'plans', COUNT(*), MAX(created_at) FROM plans
UNION ALL SELECT 'plan_lifecycle_log', COUNT(*), MAX(at) FROM plan_lifecycle_log
UNION ALL SELECT 'touch_outcomes', COUNT(*), MAX(created_at) FROM touch_outcomes
UNION ALL SELECT 'candidate_pool', COUNT(*), MAX(created_at) FROM candidate_pool
UNION ALL SELECT 'trade_excursions', COUNT(*), MAX(created_at) FROM trade_excursions
UNION ALL SELECT 'decision_records', COUNT(*), MAX(timestamp) FROM decision_records
UNION ALL SELECT 'ab_confirm_log', COUNT(*), MAX(created_at) FROM ab_confirm_log
UNION ALL SELECT 'nt8_order_snapshots', COUNT(*), datetime(MAX(received_at_ms)/1000,'unixepoch','-5 hours') FROM nt8_order_snapshots
UNION ALL SELECT 'bars', COUNT(*), datetime(MAX(open_time_ms)/1000,'unixepoch','-5 hours') FROM bars
UNION ALL SELECT 'trader_fills', COUNT(*), MAX(created_at) FROM trader_fills
UNION ALL SELECT 'trader_orders', COUNT(*), MAX(created_at) FROM trader_orders
UNION ALL SELECT 'planner_read_facts', COUNT(*), MAX(created_at) FROM planner_read_facts;
