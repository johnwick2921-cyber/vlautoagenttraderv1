.mode list
.headers on
SELECT version, json_extract(doc,'$.bias.direction') bias, json_extract(doc,'$.bias.conviction') conv,
 (SELECT group_concat(json_extract(s.value,'$.id')||':'||json_extract(s.value,'$.condition')||':'||json_extract(s.value,'$.direction')||':arm='||COALESCE(json_extract(s.value,'$.arm.enabled'),'none'), ' | ') FROM json_each(json_extract(doc,'$.scenarios')) s) scen
FROM plans WHERE plan_id LIKE '2026-09-03:ASIA:%' AND version IN (4,5,6) ORDER BY version;
SELECT '--- armed_orders plan_id sample + 09-03 rows';
SELECT id, substr(plan_id,1,22) pid, version, scenario, side, entry_px, state, datetime(strftime('%s',created_at),'unixepoch','-5 hours') c FROM armed_orders WHERE plan_id LIKE '2026-09-03%' ORDER BY id;
SELECT '--- 37304 decision';
SELECT substr(decision_json, instr(decision_json,'"action"'), 60) act, substr(decision_json, instr(decision_json,'"stop_loss"'), 70) sl, substr(decision_json, instr(decision_json,'"reasoning"'), 260) rsn FROM decision_records WHERE id=37304;
SELECT '--- 5m tape 09-03 20:30-22:00 CT';
SELECT datetime(open_time_ms/1000,'unixepoch','-5 hours') ct, o,h,l,c FROM bars WHERE symbol='MNQ' AND tf='5m' AND open_time_ms BETWEEN strftime('%s','2026-09-04 01:30:00')*1000 AND strftime('%s','2026-09-04 03:00:00')*1000 ORDER BY open_time_ms;
