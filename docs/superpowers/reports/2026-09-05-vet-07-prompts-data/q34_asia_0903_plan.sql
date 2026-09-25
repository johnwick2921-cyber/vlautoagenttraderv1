.mode list
.headers on
SELECT version, lifecycle, datetime(strftime('%s',created_at),'unixepoch','-5 hours') ct, substr(trigger_reason,1,40) trig,
 json_extract(doc,'$.scenarios[0].id') s1, json_extract(doc,'$.scenarios[0].condition') s1_cond, json_extract(doc,'$.scenarios[0].direction') s1_dir,
 json_extract(doc,'$.scenarios[0].arm.enabled') s1_arm, json_extract(doc,'$.scenarios[0].arm.entry') s1_entry, json_extract(doc,'$.scenarios[0].confirm.rule') s1_confirm
FROM plans WHERE plan_id LIKE '2026-09-03:ASIA:%' ORDER BY version;
SELECT '--- armed_orders for that plan';
SELECT id, version, scenario, side, entry_px, state, substr(state_reason,1,60) reason, datetime(strftime('%s',created_at),'unixepoch','-5 hours') c, datetime(strftime('%s',updated_at),'unixepoch','-5 hours') u FROM armed_orders WHERE plan_id LIKE '2026-09-03:ASIA:%' ORDER BY id;
