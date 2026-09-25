SELECT 'trader_positions', COUNT(*), MAX(created_at), MAX(datetime(entry_time/1000,'unixepoch','-5 hours')) FROM trader_positions
UNION ALL SELECT 'armed_orders', COUNT(*), MAX(datetime(strftime('%s',updated_at),'unixepoch','-5 hours')), NULL FROM armed_orders
UNION ALL SELECT 'plans', COUNT(*), MAX(created_at), NULL FROM plans
UNION ALL SELECT 'touch_outcomes', COUNT(*), MAX(created_at), NULL FROM touch_outcomes
UNION ALL SELECT 'candidate_pool', COUNT(*), MAX(created_at), NULL FROM candidate_pool
UNION ALL SELECT 'trade_excursions', COUNT(*), MAX(created_at), NULL FROM trade_excursions
UNION ALL SELECT 'decision_records', COUNT(*), MAX(timestamp), NULL FROM decision_records
UNION ALL SELECT 'ab_confirm_log', COUNT(*), MAX(created_at), NULL FROM ab_confirm_log
UNION ALL SELECT 'nt8_order_snapshots', COUNT(*), MAX(datetime(received_at_ms/1000,'unixepoch','-5 hours')), NULL FROM nt8_order_snapshots
UNION ALL SELECT 'bars', COUNT(*), MAX(datetime(open_time_ms/1000,'unixepoch','-5 hours')), NULL FROM bars
UNION ALL SELECT 'planner_read_facts', COUNT(*), MAX(created_at), NULL FROM planner_read_facts
UNION ALL SELECT 'plan_lifecycle_log', COUNT(*), MAX(at), NULL FROM plan_lifecycle_log
UNION ALL SELECT 'level_stats', COUNT(*), MAX(created_at), NULL FROM level_stats
UNION ALL SELECT 'touch_episodes', COUNT(*), MAX(created_at), NULL FROM touch_episodes;
