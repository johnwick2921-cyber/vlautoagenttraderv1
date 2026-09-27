SELECT id,entry_time,exit_time,plan_id,plan_session,source,side,pnl_corrected,fee,mae,mfe FROM trader_positions
WHERE entry_time>=1786770000000 AND status='CLOSED'
AND plan_id IS NOT NULL AND TRIM(plan_id)<>'' AND plan_id<>'UNRESOLVABLE'
AND COALESCE(source,'')<>'e7_farside_test' AND pnl_corrected IS NOT NULL
ORDER BY entry_time,id;
