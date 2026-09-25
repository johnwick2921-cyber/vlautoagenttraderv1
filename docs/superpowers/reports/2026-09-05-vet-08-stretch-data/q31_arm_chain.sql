SELECT p.trade_date, p.session, p.version, json_extract(s.value,'$.id') sid,
  json_extract(s.value,'$.arm.enabled') arm_en, json_extract(s.value,'$.arm.wait_confirm') wait_confirm,
  json_array_length(json_extract(s.value,'$.arm.legs')) n_legs, json_extract(s.value,'$.arm.legs') legs,
  json_extract(s.value,'$.confirm2.rule') c2_rule, json_extract(s.value,'$.confirm2.ref_price') c2_ref, json_extract(s.value,'$.confirm2.side') c2_side
FROM plans p, json_each(json_extract(p.doc,'$.scenarios')) s
WHERE p.trade_date BETWEEN '2026-09-02' AND '2026-09-04' AND json_extract(s.value,'$.arm.enabled')=1
ORDER BY p.created_at, sid;
