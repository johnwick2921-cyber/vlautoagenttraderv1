-- q06: DEDUPE touch_outcomes on (level_kind, level_price, opened_at_ms): the true distinct episode count per kind
WITH d AS (
  SELECT level_kind, level_price, opened_at_ms, MIN(outcome) outcome, COUNT(*) copies, COUNT(DISTINCT outcome) n_outcomes,
         MIN(ordinal) min_ord, MAX(ordinal) max_ord, MIN(session) s
  FROM touch_outcomes GROUP BY 1,2,3)
SELECT level_kind, COUNT(*) episodes, SUM(copies) raw_rows, COUNT(DISTINCT level_price) prices,
  SUM(outcome='hold') hold, SUM(outcome='break') brk, SUM(outcome='ambiguous_horizon') amb,
  ROUND(1.0*SUM(outcome='hold')/NULLIF(SUM(outcome IN ('hold','break')),0),3) p_hold,
  SUM(n_outcomes>1) conflicting, MAX(copies) max_copies
FROM d GROUP BY level_kind ORDER BY episodes DESC;
SELECT 'TOTAL' k, COUNT(*) episodes, SUM(copies) raw FROM (SELECT level_kind, level_price, opened_at_ms, COUNT(*) copies FROM touch_outcomes GROUP BY 1,2,3);
-- do conflicting outcomes exist for the same episode across reads?
SELECT level_kind, level_price, opened_at_ms, GROUP_CONCAT(DISTINCT outcome) outs, GROUP_CONCAT(DISTINCT ordinal) ords, COUNT(*) c FROM touch_outcomes GROUP BY 1,2,3 HAVING COUNT(DISTINCT outcome)>1 LIMIT 10;
-- RTH-L: the single price and its episodes
SELECT level_price, datetime(opened_at_ms/1000,'unixepoch','-5 hours') opened_ct, outcome, ordinal, session, plan_version, bars_to_exit, mfe_pts, mae_pts FROM touch_outcomes WHERE level_kind='RTH-L' ORDER BY opened_at_ms, plan_version LIMIT 40;
