.mode column
.headers on
SELECT 'created_at>=08-15' k, COUNT(*) FROM trader_positions WHERE created_at >= 1786770000000;
SELECT 'era cited non-empty' k, SUM(cited_scenario_id IS NOT NULL AND cited_scenario_id<>'') FROM trader_positions WHERE entry_time >= 1786770000000;
SELECT 'era mae/mfe nonnull' k, SUM(mae IS NOT NULL), SUM(mfe IS NOT NULL), SUM(mae<>0 OR mfe<>0) FROM trader_positions WHERE entry_time >= 1786770000000;
SELECT symbol, tf, convention, COUNT(*) n, COUNT(DISTINCT open_time_ms) d FROM bars WHERE tf IN ('1m','5m') GROUP BY 1,2,3;
SELECT 'nt8 snapshots 09-04' k, COUNT(*), SUM(working_count>0), MAX(working_count) FROM nt8_order_snapshots WHERE date(received_at_ms/1000,'unixepoch','-5 hours')='2026-09-04';
SELECT id, reason, order_count, working_count, datetime(received_at_ms/1000,'unixepoch','-5 hours') ct, substr(orders_json,1,400) FROM nt8_order_snapshots WHERE working_count>0 ORDER BY id DESC LIMIT 3;
SELECT 'latest snapshot' k, id, reason, order_count, working_count, datetime(received_at_ms/1000,'unixepoch','-5 hours') ct, substr(orders_json,1,300) FROM nt8_order_snapshots ORDER BY id DESC LIMIT 1;
