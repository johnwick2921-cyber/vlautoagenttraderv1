.mode column
.headers on
-- ERA = entry_time >= 2026-08-15 00:00 CT = 1786770000000 ms
SELECT 'ERA>=2026-08-15' AS scope, COUNT(*) n, SUM(pnl_corrected IS NULL) pnl_null, SUM(plan_id IS NOT NULL) has_plan, SUM(cited_scenario_id IS NOT NULL) has_scn, SUM(mae IS NOT NULL) has_mae, SUM(mfe IS NOT NULL) has_mfe, MIN(id), MAX(id) FROM trader_positions WHERE entry_time >= 1786770000000;
SELECT source, COUNT(*) n, SUM(pnl_corrected IS NULL) pnl_null, ROUND(SUM(pnl_corrected),2) sum_pnl, GROUP_CONCAT(id) ids FROM trader_positions WHERE entry_time >= 1786770000000 GROUP BY source;
SELECT plan_session, COUNT(*) n, SUM(pnl_corrected IS NULL) pnl_null, ROUND(SUM(pnl_corrected),2) sum_pnl, SUM(pnl_corrected>0) wins, SUM(pnl_corrected<0) losses, SUM(pnl_corrected=0) flat FROM trader_positions WHERE entry_time >= 1786770000000 AND source<>'e7_farside_test' GROUP BY plan_session;
SELECT side, COUNT(*) n, ROUND(SUM(pnl_corrected),2) sum_pnl, SUM(pnl_corrected>0) wins, SUM(pnl_corrected<0) losses FROM trader_positions WHERE entry_time >= 1786770000000 AND source<>'e7_farside_test' AND pnl_corrected IS NOT NULL GROUP BY side;
SELECT close_reason, COUNT(*) n, ROUND(SUM(pnl_corrected),2) sum_pnl FROM trader_positions WHERE entry_time >= 1786770000000 AND source<>'e7_farside_test' GROUP BY close_reason ORDER BY n DESC;
SELECT trader_id, account, COUNT(*) n, MIN(id), MAX(id), ROUND(SUM(pnl_corrected),2) FROM trader_positions WHERE entry_time >= 1786770000000 GROUP BY trader_id, account;
SELECT pnl_correction_note, COUNT(*) FROM trader_positions WHERE entry_time >= 1786770000000 GROUP BY 1;
SELECT id, source, plan_session, side, pnl_corrected, realized_pnl, close_reason, datetime(entry_time/1000,'unixepoch','-5 hours') FROM trader_positions WHERE entry_time >= 1786770000000 AND (pnl_corrected IS NULL OR source<>'system') ORDER BY id;
