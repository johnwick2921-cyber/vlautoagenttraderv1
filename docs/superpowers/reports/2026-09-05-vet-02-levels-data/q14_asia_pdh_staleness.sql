-- q14: ASIA plans (read ~16:30 CT) — what PDH/PDL do they carry vs the calendar-day H/L of the day that just closed and the day before?
WITH lv AS (
  SELECT p.trade_date, p.session, p.version, datetime(p.created_at) created_utc, datetime(strftime('%s',p.created_at),'unixepoch','-5 hours') created_ct,
         json_extract(j.value,'$.label') label, json_extract(j.value,'$.price') price
  FROM plans p, json_each(json_extract(p.doc,'$.levels')) j WHERE p.session='ASIA' AND json_extract(j.value,'$.label') IN ('PDH','PDL')),
cal AS (
  SELECT date((open_time_ms/1000)-5*3600,'unixepoch') cd, MAX(h) hi, MIN(l) lo, COUNT(*) n FROM bars WHERE symbol='MNQ' AND tf='1m' GROUP BY 1)
SELECT lv.trade_date, lv.version, lv.created_ct, lv.label, lv.price,
  c1.cd d_minus1, c1.hi hi1, c1.lo lo1, c2.cd d_minus2, c2.hi hi2, c2.lo lo2,
  CASE WHEN lv.label='PDH' AND abs(lv.price-c1.hi)<=0.5 THEN 'D-1' WHEN lv.label='PDH' AND abs(lv.price-c2.hi)<=0.5 THEN 'D-2'
       WHEN lv.label='PDL' AND abs(lv.price-c1.lo)<=0.5 THEN 'D-1' WHEN lv.label='PDL' AND abs(lv.price-c2.lo)<=0.5 THEN 'D-2' ELSE 'neither' END which
FROM lv
LEFT JOIN cal c1 ON c1.cd = date(lv.created_ct)          -- the calendar day the ASIA read happens on (its RTH just closed)
LEFT JOIN cal c2 ON c2.cd = date(lv.created_ct,'-1 day')
WHERE lv.version = (SELECT MIN(version) FROM plans p2 WHERE p2.trade_date=lv.trade_date AND p2.session='ASIA')
ORDER BY lv.trade_date, lv.label;
