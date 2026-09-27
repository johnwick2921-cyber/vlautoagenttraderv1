.mode column
.headers on
SELECT date(timestamp,'-5 hours') d, COUNT(*) cycles, SUM(decision_json LIKE '%open_long%') open_long, SUM(decision_json LIKE '%open_short%') open_short, SUM(decision_json LIKE '%"wait"%' OR decision_json LIKE '%"hold"%') wait_hold, SUM(decision_json LIKE '%close_%') closes, SUM(execution_log LIKE '%entry_gate%') eg_refusals, SUM(execution_log LIKE '%REFUSED%' OR execution_log LIKE '%refused%') refused_any, SUM(risk_check_passed=0) risk_fail, SUM(execution_status='filled' OR execution_status='executed') exec_ok FROM decision_records WHERE timestamp >= '2026-08-15' GROUP BY d ORDER BY d;
SELECT execution_status, COUNT(*) FROM decision_records WHERE timestamp >= '2026-08-15' GROUP BY 1;
SELECT cycle_type, cycle_trigger, COUNT(*) FROM decision_records WHERE timestamp >= '2026-08-15' GROUP BY 1,2 ORDER BY 3 DESC LIMIT 15;
SELECT error_class, COUNT(*) FROM decision_records WHERE timestamp >= '2026-08-15' GROUP BY 1 ORDER BY 2 DESC LIMIT 10;
