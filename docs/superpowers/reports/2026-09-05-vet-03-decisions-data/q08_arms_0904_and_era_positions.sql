.mode list
.separator " | "
.headers on
SELECT '## q08a: arms created 2026-09-04 CT (created_at normalised)' AS s;
SELECT id, scenario, side, kind, condition, entry_px, stop_px, target_px, state, substr(state_reason,1,60) reason, datetime(strftime('%s',created_at),'unixepoch','-5 hours') created_ct, datetime(strftime('%s',updated_at),'unixepoch','-5 hours') updated_ct, version, leg_index, leg_count, placement_seq, signal_id FROM armed_orders WHERE date(datetime(strftime('%s',created_at),'unixepoch','-5 hours'))='2026-09-04' ORDER BY id;
SELECT '## q08b: all filled arms' AS s;
SELECT id, session, scenario, side, kind, condition, entry_px, stop_px, target_px, fill_price, state_reason, datetime(strftime('%s',created_at),'unixepoch','-5 hours') created_ct, datetime(strftime('%s',updated_at),'unixepoch','-5 hours') updated_ct, plan_id FROM armed_orders WHERE state='filled' ORDER BY id;
SELECT '## q09: era positions (plan_id not null), excl e7 test seam' AS s;
SELECT id, datetime(entry_time/1000,'unixepoch','-5 hours') entry_ct, datetime(exit_time/1000,'unixepoch','-5 hours') exit_ct, side, entry_price, exit_price, pnl_corrected, realized_pnl, source, plan_session, cited_scenario_id, adherence_grade, plan_band, close_reason, mae, mfe, entry_confidence, plan_version FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' AND source<>'e7_farside_test' ORDER BY entry_time;
SELECT '## q09b: by cited slot (pnl_corrected, excl test seam, excl NULL)' AS s;
SELECT COALESCE(cited_scenario_id,'(none)') slot, COUNT(*) n, SUM(pnl_corrected IS NULL) null_excluded, ROUND(SUM(pnl_corrected),2) sum_pnl, ROUND(AVG(pnl_corrected),2) mean_pnl, SUM(pnl_corrected>0) wins, SUM(pnl_corrected<0) losses, SUM(pnl_corrected=0) scratch FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' AND source<>'e7_farside_test' GROUP BY 1 ORDER BY 1;
SELECT '## q09c: by source' AS s;
SELECT source, COUNT(*) n, SUM(pnl_corrected IS NULL) null_excluded, ROUND(SUM(pnl_corrected),2) sum_pnl, SUM(pnl_corrected>0) wins, SUM(pnl_corrected<0) losses FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' GROUP BY 1;
SELECT '## q09d: by plan_session' AS s;
SELECT plan_session, COUNT(*) n, SUM(pnl_corrected IS NULL) null_excluded, ROUND(SUM(pnl_corrected),2) sum_pnl, SUM(pnl_corrected>0) wins, SUM(pnl_corrected<0) losses FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' AND source<>'e7_farside_test' GROUP BY 1;
SELECT '## q09e: close_reason' AS s;
SELECT close_reason, COUNT(*) n, ROUND(SUM(pnl_corrected),2) sum_pnl FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' AND source<>'e7_farside_test' GROUP BY 1 ORDER BY n DESC;
SELECT '## q09f: overall era (excl test seam, excl NULL)' AS s;
SELECT COUNT(*) n_all, SUM(pnl_corrected IS NULL) null_excluded, COUNT(pnl_corrected) n_used, ROUND(SUM(pnl_corrected),2) sum_pnl, ROUND(AVG(pnl_corrected),2) mean, SUM(pnl_corrected>0) wins, SUM(pnl_corrected<0) losses, SUM(pnl_corrected=0) scratch, ROUND(AVG(CASE WHEN pnl_corrected>0 THEN pnl_corrected END),2) avg_win, ROUND(AVG(CASE WHEN pnl_corrected<0 THEN pnl_corrected END),2) avg_loss FROM trader_positions WHERE plan_id IS NOT NULL AND plan_id<>'' AND source<>'e7_farside_test';
