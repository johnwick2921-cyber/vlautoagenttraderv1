.mode column
.headers on
.width 4 34 4 7 20 5 9 9 9 12 26 16 6 19 19 9 9 5 3 3 5 7
SELECT id, plan_id, version v, session, scenario, side, entry_px, stop_px, target_px, state, substr(state_reason,1,26) reason, substr(entry_class,1,16) cls, substr(signal_id,1,6) sig, datetime(strftime('%s',created_at),'unixepoch','-5 hours') created_ct, datetime(strftime('%s',updated_at),'unixepoch','-5 hours') updated_ct, fill_price, fill_quantity, kind, leg_index li, leg_count lc, placement_seq seq, substr(condition,1,7) cond
FROM armed_orders WHERE datetime(strftime('%s',created_at),'unixepoch','-5 hours') >= '2026-09-02' ORDER BY id;
