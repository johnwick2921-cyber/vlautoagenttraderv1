-- every closed position with entry 2026-09-01 16:30 CT .. 2026-09-04 23:59 CT (covers ASIA 09-01 read through 09-04)
SELECT id, source, plan_session, plan_id, plan_version, cited_scenario_id, side, entry_price, exit_price,
  datetime(entry_time/1000,'unixepoch','-5 hours') entry_ct,
  datetime(exit_time/1000,'unixepoch','-5 hours') exit_ct,
  realized_pnl, pnl_corrected, pnl_correction_note, close_reason, mae, mfe, entry_confidence, adherence_grade, plan_matched, account
FROM trader_positions
WHERE entry_time >= strftime('%s','2026-09-01 21:30:00')*1000 AND entry_time < strftime('%s','2026-09-05 05:00:00')*1000
ORDER BY entry_time;
