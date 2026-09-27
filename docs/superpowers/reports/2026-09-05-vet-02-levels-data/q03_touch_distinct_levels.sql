-- q03: how many DISTINCT level prices underlie the touch corpus, per kind; ordinal distribution; per plan/session
SELECT level_kind, COUNT(*) n, COUNT(DISTINCT level_price) distinct_prices, COUNT(DISTINCT plan_id||'/'||plan_version) plan_reads,
  COUNT(DISTINCT session) sessions, MIN(ordinal) min_ord, MAX(ordinal) max_ord,
  SUM(ordinal=1) ord1, SUM(ordinal=2) ord2, SUM(ordinal>=3) ord3p,
  ROUND(AVG(bars_to_exit),1) avg_bars_to_exit, ROUND(AVG(mfe_pts),2) avg_mfe, ROUND(AVG(mae_pts),2) avg_mae
FROM touch_outcomes GROUP BY level_kind ORDER BY n DESC;
