SELECT id, entry_time, exit_time, created_at, symbol, side, entry_quantity,
entry_price, exit_price, pnl_corrected, fee, plan_id, plan_session, source,
close_reason, mae, mfe FROM trader_positions
WHERE entry_time >= 1786770000000 AND status='CLOSED'
AND plan_id IS NOT NULL AND TRIM(plan_id)<>'' AND plan_id<>'UNRESOLVABLE'
AND COALESCE(source,'')<>'e7_farside_test' AND pnl_corrected IS NOT NULL
ORDER BY entry_time,id;
