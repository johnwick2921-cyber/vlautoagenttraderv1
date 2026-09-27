SELECT p.trade_date, p.session, p.version, datetime(strftime('%s',p.created_at),'unixepoch','-5 hours') created_ct, p.lifecycle,
  json_extract(p.doc,'$.bias.direction') bias,
  json_extract(s.value,'$.id') sid, json_extract(s.value,'$.direction') dir, json_extract(s.value,'$.condition') cond,
  json_extract(s.value,'$.quality') q,
  json_extract(s.value,'$.confirm.rule') c_rule, json_extract(s.value,'$.confirm.ref_price') c_ref, json_extract(s.value,'$.confirm.side') c_side,
  json_extract(s.value,'$.arm.enabled') arm_en, json_extract(s.value,'$.arm.entry') arm_entry, json_extract(s.value,'$.arm.stop') arm_stop, json_extract(s.value,'$.arm.target') arm_target, json_extract(s.value,'$.arm.kind') arm_kind,
  json_extract(s.value,'$.target_chain') targets,
  replace(json_extract(s.value,'$.trigger'),char(10),' ') trig,
  replace(json_extract(s.value,'$.invalid'),char(10),' ') invalid,
  json_extract(p.doc,'$.death_condition') death, json_extract(p.doc,'$.bias.flip_condition') flip
FROM plans p, json_each(json_extract(p.doc,'$.scenarios')) s
WHERE p.trade_date BETWEEN '2026-09-01' AND '2026-09-04' AND NOT (p.trade_date='2026-09-01' AND p.session IN ('LONDON','NY'))
ORDER BY p.created_at, sid;
