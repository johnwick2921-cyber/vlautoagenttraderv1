.mode column
.headers on
SELECT is_counterfactual, COUNT(*) FROM ab_confirm_log GROUP BY 1;
SELECT condition, outcome, COUNT(*) n, ROUND(MIN(net_pnl),1) mn, ROUND(MAX(net_pnl),1) mx, ROUND(AVG(net_pnl),1) avg, ROUND(AVG(mfe),1) mfe, ROUND(AVG(mae),1) mae, ROUND(AVG(mfe_r),2) mfe_r, ROUND(AVG(mae_r),2) mae_r FROM ab_confirm_log GROUP BY 1,2 ORDER BY 1,2;
SELECT id, plan_id, version, session, scenario, rule, condition, direction, entry_px, stop_px, target_px, fill_px, rr, atr5m, mfe, mae, mfe_r, mae_r, time_to_mfe_bars, net_pnl, outcome, normalized, recompute, substr(created_at,1,19) c FROM ab_confirm_log WHERE outcome<>'open' ORDER BY id;
SELECT COUNT(DISTINCT plan_id||version||scenario) FROM ab_confirm_log;
SELECT date(created_at) d, COUNT(*) FROM ab_confirm_log GROUP BY 1;
