SELECT id, plan_id, version, session, scenario, side, entry_px, stop_px, target_px, state, state_reason, entry_class, kind, condition, leg_index, leg_count, fill_price,
  datetime(strftime('%s',created_at),'unixepoch','-5 hours') created_ct,
  datetime(strftime('%s',updated_at),'unixepoch','-5 hours') updated_ct,
  boot_id, armed_under_version, placement_seq, created_at raw_created, updated_at raw_updated
FROM armed_orders
WHERE strftime('%s',created_at) >= strftime('%s','2026-09-01 21:00:00') 
ORDER BY id;
