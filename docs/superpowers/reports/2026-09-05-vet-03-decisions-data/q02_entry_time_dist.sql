-- q02: entry_time distribution by CT date-month and typeof/magnitude, to reconcile 71 vs 227
SELECT MIN(entry_time), MAX(entry_time), MIN(LENGTH(entry_time)), MAX(LENGTH(entry_time)) FROM trader_positions;
SELECT strftime('%Y-%m', entry_time/1000, 'unixepoch', '-5 hours') ym, COUNT(*) n, SUM(pnl_corrected IS NOT NULL) pnlc, SUM(plan_id IS NOT NULL AND plan_id<>'') has_plan, SUM(cited_scenario_id IS NOT NULL AND cited_scenario_id<>'') cited FROM trader_positions GROUP BY 1 ORDER BY 1;
-- rows per CT date since 08-10
SELECT date(entry_time/1000, 'unixepoch', '-5 hours') d, COUNT(*) n, SUM(pnl_corrected IS NOT NULL) pnlc, SUM(plan_id IS NOT NULL AND plan_id<>'') has_plan, GROUP_CONCAT(DISTINCT source) src FROM trader_positions WHERE entry_time >= strftime('%s','2026-08-10')*1000 GROUP BY 1 ORDER BY 1;
-- created_at typeof
SELECT typeof(created_at), COUNT(*) FROM trader_positions GROUP BY 1;
SELECT MIN(created_at), MAX(created_at) FROM trader_positions;
-- how many have plan_id at all
SELECT COUNT(*) FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'';
SELECT COUNT(*) FROM trader_positions WHERE pnl_corrected IS NOT NULL;
