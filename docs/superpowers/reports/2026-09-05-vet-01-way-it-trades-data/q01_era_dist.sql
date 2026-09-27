.mode column
.headers on
SELECT 'ALL-TIME' AS scope, COUNT(*) n, SUM(pnl_corrected IS NULL) pnl_null FROM trader_positions;
SELECT 'ERA>=08-15' AS scope, COUNT(*) n, SUM(pnl_corrected IS NULL) pnl_null, SUM(plan_id IS NOT NULL) has_plan, SUM(cited_scenario_id IS NOT NULL) has_scn, SUM(mae IS NOT NULL) has_mae, SUM(mfe IS NOT NULL) has_mfe FROM trader_positions WHERE entry_time >= 1755234000000;
SELECT source, COUNT(*) n, SUM(pnl_corrected IS NULL) pnl_null, ROUND(SUM(pnl_corrected),2) sum_pnl FROM trader_positions WHERE entry_time >= 1755234000000 GROUP BY source;
SELECT plan_session, COUNT(*) n, SUM(pnl_corrected IS NULL) pnl_null, ROUND(SUM(pnl_corrected),2) sum_pnl FROM trader_positions WHERE entry_time >= 1755234000000 GROUP BY plan_session;
SELECT side, COUNT(*) n, ROUND(SUM(pnl_corrected),2) sum_pnl FROM trader_positions WHERE entry_time >= 1755234000000 AND source<>'e7_farside_test' GROUP BY side;
SELECT close_reason, COUNT(*) n, ROUND(SUM(pnl_corrected),2) sum_pnl FROM trader_positions WHERE entry_time >= 1755234000000 AND source<>'e7_farside_test' GROUP BY close_reason ORDER BY n DESC;
SELECT trader_id, account, COUNT(*) n, MIN(id), MAX(id), ROUND(SUM(pnl_corrected),2) FROM trader_positions WHERE entry_time >= 1755234000000 GROUP BY trader_id, account;
SELECT date(entry_time/1000,'unixepoch','-5 hours') d, COUNT(*) n, ROUND(SUM(pnl_corrected),2) pnl FROM trader_positions WHERE entry_time >= 1755234000000 GROUP BY d ORDER BY d;
