SELECT date(d.timestamp,'-5 hours') day, COUNT(*) cycles,
  SUM(CASE WHEN d.execution_log LIKE '%entry_gate%' THEN 1 ELSE 0 END) eg_in_execlog,
  SUM(CASE WHEN d.execution_log LIKE '%REFUSED%' OR d.execution_log LIKE '%refused%' THEN 1 ELSE 0 END) refused_word,
  SUM(CASE WHEN d.decision_json LIKE '%open_long%' THEN 1 ELSE 0 END) has_open_long,
  SUM(CASE WHEN d.decision_json LIKE '%open_short%' THEN 1 ELSE 0 END) has_open_short,
  SUM(CASE WHEN d.success=1 THEN 1 ELSE 0 END) ok,
  SUM(CASE WHEN d.error_message IS NOT NULL AND d.error_message<>'' THEN 1 ELSE 0 END) errs
FROM decision_records d
WHERE date(d.timestamp,'-5 hours') BETWEEN '2026-09-01' AND '2026-09-04'
GROUP BY day;
