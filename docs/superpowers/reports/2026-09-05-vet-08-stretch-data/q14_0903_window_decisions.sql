WITH d AS (SELECT * FROM decision_records WHERE timestamp BETWEEN '2026-09-03 14:15:00' AND '2026-09-03 17:30:00')
SELECT d.id, datetime(d.timestamp,'-5 hours') ct, d.cycle_type, d.cycle_trigger, d.plan_version, d.cited_scenario_id, d.error_class,
  json_extract(j.value,'$.action') action, json_extract(j.value,'$.confidence') conf,
  substr(replace(json_extract(j.value,'$.reasoning'),char(10),' '),1,500) reasoning,
  substr(replace(d.execution_log,char(10),' | '),1,160) execlog
FROM d LEFT JOIN json_each(CASE WHEN json_valid(d.decision_json) AND d.decision_json<>'' THEN d.decision_json ELSE '[]' END) j
ORDER BY d.timestamp;
