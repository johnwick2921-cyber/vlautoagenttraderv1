SELECT id, datetime(timestamp,'-5 hours') ct, cycle_type, cycle_trigger, plan_version, cited_scenario_id, error_class, execution_status,
  substr(replace(execution_log, char(10), ' | '),1,400) execlog
FROM decision_records
WHERE date(timestamp,'-5 hours') BETWEEN '2026-09-02' AND '2026-09-04'
  AND (execution_log LIKE '%entry_gate%' OR execution_log LIKE '%REFUSED%' OR execution_log LIKE '%refused%' OR execution_log LIKE '%sl_too_tight%' OR execution_log LIKE '%rejected%' OR execution_log LIKE '%min_sl%')
ORDER BY timestamp;
