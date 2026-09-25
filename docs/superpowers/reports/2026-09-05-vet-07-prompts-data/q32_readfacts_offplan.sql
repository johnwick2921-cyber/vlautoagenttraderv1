.mode list
.headers on
SELECT COUNT(*) n, SUM(tokens_in=0) tokens_in_zero, SUM(plan_id IS NULL OR plan_id='') plan_id_null, COUNT(DISTINCT prompt_hash) distinct_hash, SUM(bias_ai IS NULL OR bias_ai='') bias_ai_null FROM planner_read_facts;
SELECT '---opens 09-03: cited_scenario';
SELECT id, datetime(strftime('%s',timestamp),'unixepoch','-5 hours') ct, json_extract(decision_json,'$[0].action') act, json_extract(decision_json,'$[0].cited_scenario') cited, CASE WHEN execution_log LIKE '%refused: strict%' THEN 'strict' WHEN execution_log LIKE '%entry_gate%' THEN 'other_gate' ELSE 'other' END outcome FROM decision_records WHERE date(timestamp,'-5 hours')='2026-09-03' AND json_extract(decision_json,'$[0].action') LIKE 'open_%' ORDER BY id;
SELECT '---off-plan mentions 7d: waits vs opens';
SELECT json_extract(decision_json,'$[0].action') act, COUNT(*) n FROM decision_records WHERE date(timestamp,'-5 hours') BETWEEN '2026-08-29' AND '2026-09-04' AND decision_json LIKE '%off-plan%' GROUP BY 1;
SELECT '---09-04 decision_records by cycle_type';
SELECT COALESCE(cycle_type,'') ct, COALESCE(prompt_version,'') pv, COUNT(*) n, ROUND(AVG(length(system_prompt))) sys, ROUND(AVG(length(input_prompt))) inp FROM decision_records WHERE date(timestamp,'-5 hours')='2026-09-04' GROUP BY 1,2;
