.mode column
.headers on
SELECT is_counterfactual cf, condition, COUNT(*) n, SUM(outcome='win') w, ROUND(AVG(net_pnl),2) avg_pnl, ROUND(AVG(mfe_r),2) mfe_r, ROUND(AVG(mae_r),2) mae_r, ROUND(AVG(rr),2) rr, ROUND(AVG(atr5m),2) atr FROM ab_confirm_log GROUP BY 1,2 ORDER BY 1,2;
SELECT outcome, COUNT(*) FROM ab_confirm_log GROUP BY outcome;
SELECT rule, COUNT(*) FROM ab_confirm_log GROUP BY rule;
SELECT MIN(created_at), MAX(created_at) FROM ab_confirm_log;
SELECT level_kind, ordinal, SUM(outcome='hold') hold, SUM(outcome='break') brk, SUM(outcome='ambiguous_horizon') amb, COUNT(*) n FROM touch_outcomes GROUP BY 1,2 ORDER BY 1,2;
SELECT level_kind, SUM(outcome='hold') hold, SUM(outcome='break') brk, SUM(outcome='ambiguous_horizon') amb, COUNT(*) n, ROUND(1.0*SUM(outcome='hold')/NULLIF(SUM(outcome IN ('hold','break')),0),3) hold_rate FROM touch_outcomes GROUP BY 1 ORDER BY n DESC;
SELECT session, k, band_pts, horizon, COUNT(*) FROM touch_outcomes GROUP BY 1,2,3,4;
SELECT MIN(created_at), MAX(created_at), MIN(plan_id), MAX(plan_id) FROM touch_outcomes;
