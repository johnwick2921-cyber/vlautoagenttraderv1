.mode column
.headers on
-- q29: EOD page demo — closes per CT session-day, pnl_corrected only, test seam excluded, NULL counted not summed
SELECT date(entry_time/1000,'unixepoch','-5 hours') d, COUNT(*) n_all,
       SUM(source='e7_farside_test') test_seam_excl,
       SUM(pnl_corrected IS NULL AND source<>'e7_farside_test') unresolved_excl,
       COUNT(pnl_corrected) n_resolved,
       ROUND(SUM(CASE WHEN source<>'e7_farside_test' THEN pnl_corrected END),2) sum_pnl_corr,
       SUM(CASE WHEN source<>'e7_farside_test' AND pnl_corrected>0 THEN 1 END) wins,
       SUM(CASE WHEN source<>'e7_farside_test' AND pnl_corrected<0 THEN 1 END) losses,
       GROUP_CONCAT(id) ids
FROM trader_positions WHERE entry_time >= 1756875600000 GROUP BY d ORDER BY d DESC LIMIT 6;
SELECT '--- 09-03 rows ---' AS what;
SELECT id, side, datetime(entry_time/1000,'unixepoch','-5 hours') entry_ct, datetime(exit_time/1000,'unixepoch','-5 hours') exit_ct, entry_price, exit_price, pnl_corrected, realized_pnl, fee, close_reason, source, plan_session, cited_scenario_id, adherence_grade, mae, mfe FROM trader_positions WHERE date(entry_time/1000,'unixepoch','-5 hours') IN ('2026-09-03','2026-09-02') ORDER BY entry_time;
