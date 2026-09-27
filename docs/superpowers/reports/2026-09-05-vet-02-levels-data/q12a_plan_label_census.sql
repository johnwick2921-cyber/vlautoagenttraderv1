-- q12a: seated-level label census across ALL plan docs (254 plans, 08-15..09-04): label family, grade dist
WITH lv AS (
  SELECT p.trade_date, p.session, p.version, p.created_at,
         json_extract(j.value,'$.label') label, json_extract(j.value,'$.grade') grade, json_extract(j.value,'$.machine_grade') mgrade,
         json_extract(j.value,'$.price') price, json_extract(j.value,'$.instruction') instr
  FROM plans p, json_each(json_extract(p.doc,'$.levels')) j)
SELECT CASE
  WHEN label LIKE 'VWAP%' THEN 'VWAP*' WHEN label LIKE 'SWG-H%' THEN 'SWG-H' WHEN label LIKE 'SWG-L%' THEN 'SWG-L'
  WHEN label LIKE 'OB(%' THEN 'OB' WHEN label LIKE 'Demand%' THEN 'DEMAND' WHEN label LIKE 'Supply%' THEN 'SUPPLY'
  WHEN label LIKE 'FVG%' THEN 'FVG' WHEN label LIKE 'iFVG%' THEN 'IFVG' WHEN label LIKE 'EQH%' THEN 'EQH' WHEN label LIKE 'EQL%' THEN 'EQL'
  WHEN label LIKE 'nPOC%' THEN 'nPOC' WHEN label LIKE 'RN %' THEN 'RN' WHEN label LIKE 'GAP%' THEN 'GAP' WHEN label LIKE 'IB%' THEN 'IB*'
  ELSE label END fam,
  COUNT(*) n, COUNT(DISTINCT trade_date||session) plan_sessions, SUM(grade='A') A, SUM(grade='B') B, SUM(grade='C') C, SUM(mgrade IS NULL OR mgrade='') no_mgrade,
  SUM(grade<>mgrade AND mgrade IS NOT NULL AND mgrade<>'') grade_ne_machine
FROM lv GROUP BY fam ORDER BY n DESC;
SELECT COUNT(*) total_levels, COUNT(DISTINCT trade_date||session||version) plan_versions, ROUND(1.0*COUNT(*)/COUNT(DISTINCT trade_date||session||version),2) per_plan FROM (SELECT p.trade_date, p.session, p.version FROM plans p, json_each(json_extract(p.doc,'$.levels')) j);
SELECT grade, COUNT(*) FROM (SELECT json_extract(j.value,'$.grade') grade FROM plans p, json_each(json_extract(p.doc,'$.levels')) j) GROUP BY 1;
