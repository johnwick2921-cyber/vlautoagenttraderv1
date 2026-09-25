-- q03: planner rejects, plans, lifecycle, arms, gate refusals, bias facts, counters
.mode list
.separator " | "
.headers on
SELECT '## planner_rejected_prompts by reject_reason' AS s;
SELECT reject_reason, COUNT(*) n, MIN(date(created_at,'-5 hours')) first_ct, MAX(date(created_at,'-5 hours')) last_ct FROM planner_rejected_prompts GROUP BY 1 ORDER BY n DESC;
SELECT '## rejected by session/attempt' AS s;
SELECT session, attempt, COUNT(*) n FROM planner_rejected_prompts GROUP BY 1,2 ORDER BY 1,2;
SELECT '## rejected by CT date' AS s;
SELECT date(created_at,'-5 hours') d, COUNT(*) n, COUNT(DISTINCT trade_date||session) reads FROM planner_rejected_prompts GROUP BY 1 ORDER BY 1;
SELECT '## plans by session x lifecycle' AS s;
SELECT session, lifecycle, COUNT(*) n FROM plans GROUP BY 1,2 ORDER BY 1,2;
SELECT '## plans by trigger_reason' AS s;
SELECT trigger_reason, COUNT(*) n FROM plans GROUP BY 1 ORDER BY n DESC LIMIT 25;
SELECT '## plans by trade_date (versions per plan)' AS s;
SELECT trade_date, session, COUNT(*) versions, MAX(version) maxv, SUM(degraded) degraded, GROUP_CONCAT(DISTINCT lifecycle) lc FROM plans GROUP BY 1,2 ORDER BY 1,2;
SELECT '## plan_lifecycle_log all rows' AS s;
SELECT id, plan_id, version, event, substr(reason,1,120) reason, at FROM plan_lifecycle_log ORDER BY id;
SELECT '## armed_orders by state x state_reason' AS s;
SELECT state, substr(state_reason,1,70) reason, COUNT(*) n FROM armed_orders GROUP BY 1,2 ORDER BY n DESC;
SELECT '## armed_orders by kind x condition x side' AS s;
SELECT kind, condition, side, COUNT(*) n, SUM(state='filled') filled FROM armed_orders GROUP BY 1,2,3 ORDER BY n DESC;
SELECT '## armed_orders by session' AS s;
SELECT session, COUNT(*) n, SUM(state='filled') filled, SUM(state='cancelled') cancelled, SUM(state='working') working FROM armed_orders GROUP BY 1;
SELECT '## armed_orders by CT date (created_at normalised)' AS s;
SELECT date(datetime(strftime('%s',created_at),'unixepoch','-5 hours')) d, COUNT(*) n, SUM(state='filled') filled FROM armed_orders GROUP BY 1 ORDER BY 1;
SELECT '## decision_records entry_gate refusals by CT day since 09-02' AS s;
SELECT date(timestamp,'-5 hours') d, COUNT(*) n FROM decision_records WHERE date(timestamp,'-5 hours') >= '2026-09-02' AND execution_log LIKE '%entry_gate%' GROUP BY 1;
SELECT '## decision_records entry_gate refusal reasons since 09-02' AS s;
SELECT substr(risk_check_error,1,90) reason, COUNT(*) n, MIN(datetime(timestamp,'-5 hours')) first_ct, MAX(datetime(timestamp,'-5 hours')) last_ct FROM decision_records WHERE date(timestamp,'-5 hours') >= '2026-09-02' AND (execution_log LIKE '%entry_gate%' OR risk_check_error LIKE '%entry_gate%') GROUP BY 1 ORDER BY n DESC;
SELECT '## decision_records ALL risk_check_error classes since 09-02 (non-null)' AS s;
SELECT CASE WHEN risk_check_error LIKE '%entry_gate%strict%' THEN 'entry_gate strict' WHEN risk_check_error LIKE '%entry_gate%R:R%' THEN 'entry_gate rr' WHEN risk_check_error LIKE '%entry_gate%too close%' THEN 'entry_gate min_sl' WHEN risk_check_error LIKE '%entry_gate%' THEN 'entry_gate other' WHEN risk_check_error LIKE '%min%sl%' OR risk_check_error LIKE '%stop%too close%' OR risk_check_error LIKE '%ATR%' THEN 'min_sl (validateDecision)' WHEN risk_check_error LIKE '%confidence%' THEN 'confidence' WHEN risk_check_error LIKE '%plan%' THEN 'plan-mode/other-plan' ELSE substr(risk_check_error,1,50) END cls, COUNT(*) n FROM decision_records WHERE date(timestamp,'-5 hours') >= '2026-09-02' AND risk_check_error IS NOT NULL AND risk_check_error<>'' GROUP BY 1 ORDER BY n DESC;
SELECT '## decision intents by CT day since 08-27' AS s;
SELECT date(timestamp,'-5 hours') d, COUNT(*) cycles, SUM(decision_json LIKE '%open_long%') open_long, SUM(decision_json LIKE '%open_short%') open_short, SUM(risk_check_passed=0) risk_failed FROM decision_records WHERE date(timestamp,'-5 hours') >= '2026-08-27' GROUP BY 1 ORDER BY 1;
SELECT '## planner_read_facts bias columns' AS s;
SELECT trade_date, session, version, bias_ai, bias_tree, bias_regime, void_count, stop_floor_pts, atr5m, tokens_in FROM planner_read_facts ORDER BY created_at;
SELECT '## system_config counters' AS s;
SELECT key, value FROM system_config WHERE key LIKE 'arm_refusals%' OR key LIKE 'dayplan_replans%' OR key LIKE 'arms_boot%' OR key LIKE '%refus%' ORDER BY key;
SELECT '## ab_confirm_log rule x condition' AS s;
SELECT rule, condition, COUNT(*) n, SUM(is_counterfactual) cf, SUM(net_pnl IS NOT NULL AND net_pnl<>0) usable_pnl, ROUND(AVG(mfe_r),2) mfe_r, ROUND(AVG(mae_r),2) mae_r FROM ab_confirm_log GROUP BY 1,2 ORDER BY n DESC;
