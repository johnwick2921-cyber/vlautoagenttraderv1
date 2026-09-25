SELECT d.id, datetime(d.timestamp,'-5 hours') ct, d.plan_version, d.cited_scenario_id, json_extract(j.value,'$.action') action, json_extract(j.value,'$.entry') entry, json_extract(j.value,'$.stop_loss') sl, json_extract(j.value,'$.take_profit') tp, json_extract(j.value,'$.confidence') conf, d.execution_status, d.risk_check_passed, substr(replace(d.execution_log,char(10),' | '),1,300) execlog, substr(replace(d.risk_check_error,char(10),' '),1,200) rce
FROM decision_records d, json_each(d.decision_json) j
WHERE json_valid(d.decision_json) AND d.decision_json<>'' AND json_extract(j.value,'$.action') LIKE 'open%'
  AND date(d.timestamp,'-5 hours') BETWEEN '2026-09-02' AND '2026-09-04'
ORDER BY d.timestamp;
