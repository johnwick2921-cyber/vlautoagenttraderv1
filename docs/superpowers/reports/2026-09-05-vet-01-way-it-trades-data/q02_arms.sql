.mode column
.headers on
SELECT id, plan_id, version v, session, scenario, side, entry_px, stop_px, target_px, ROUND(ABS(target_px-entry_px)/NULLIF(ABS(entry_px-stop_px),0),2) rr, state, substr(state_reason,1,60) reason, entry_class, fill_price, leg_index li, leg_count lc, kind, condition, datetime(strftime('%s',created_at),'unixepoch','-5 hours') created_ct, datetime(strftime('%s',updated_at),'unixepoch','-5 hours') updated_ct FROM armed_orders ORDER BY id;
SELECT state, COUNT(*) FROM armed_orders GROUP BY state;
SELECT scenario, COUNT(*) FROM armed_orders GROUP BY scenario;
SELECT session, COUNT(*), SUM(state='filled') filled FROM armed_orders GROUP BY session;
