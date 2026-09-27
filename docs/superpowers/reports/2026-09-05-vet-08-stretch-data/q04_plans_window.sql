SELECT plan_id, version, trade_date, session, lifecycle, trigger_reason, degraded, dark_regime_count,
  datetime(strftime('%s',created_at),'unixepoch','-5 hours') created_ct,
  json_extract(doc,'$.bias') bias, json_extract(doc,'$.conviction') conviction, json_extract(doc,'$.day_type') day_type,
  json_array_length(json_extract(doc,'$.scenarios')) n_scen,
  (SELECT COUNT(*) FROM json_each(json_extract(doc,'$.scenarios')) s WHERE json_extract(s.value,'$.direction')='long') longs,
  (SELECT COUNT(*) FROM json_each(json_extract(doc,'$.scenarios')) s WHERE json_extract(s.value,'$.direction')='short') shorts,
  (SELECT COUNT(*) FROM json_each(json_extract(doc,'$.scenarios')) s WHERE json_extract(s.value,'$.arm.enabled')=1) arms_enabled
FROM plans WHERE trade_date BETWEEN '2026-09-01' AND '2026-09-04' ORDER BY created_at;
