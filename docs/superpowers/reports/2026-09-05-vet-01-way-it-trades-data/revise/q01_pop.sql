.mode column
.headers on
SELECT 'all_era' k, count(*) n, round(sum(pnl_corrected),2) sm,
 sum(pnl_corrected>0) w, sum(pnl_corrected<0) l, sum(pnl_corrected=0) f
FROM trader_positions WHERE entry_time>=1786770000000 AND source<>'e7_farside_test' AND pnl_corrected IS NOT NULL;
SELECT 'compliant' k, count(*) n, round(sum(pnl_corrected),2) sm,
 sum(pnl_corrected>0) w, sum(pnl_corrected<0) l, sum(pnl_corrected=0) f,
 round(avg(pnl_corrected),4) mean
FROM trader_positions WHERE entry_time>=1786770000000 AND source<>'e7_farside_test' AND pnl_corrected IS NOT NULL AND plan_id<>'UNRESOLVABLE';
SELECT 'unresolvable' k, count(*) n, round(sum(pnl_corrected),2) sm, group_concat(id) ids
FROM trader_positions WHERE entry_time>=1786770000000 AND source<>'e7_farside_test' AND pnl_corrected IS NOT NULL AND plan_id='UNRESOLVABLE';
SELECT 'avgW' k, round(avg(pnl_corrected),4) v FROM trader_positions WHERE entry_time>=1786770000000 AND source<>'e7_farside_test' AND pnl_corrected>0 AND plan_id<>'UNRESOLVABLE';
SELECT 'avgL' k, round(avg(pnl_corrected),4) v FROM trader_positions WHERE entry_time>=1786770000000 AND source<>'e7_farside_test' AND pnl_corrected<0 AND plan_id<>'UNRESOLVABLE';
SELECT 'plan_id_distinct_nonnull' k, count(DISTINCT plan_id) v FROM trader_positions WHERE entry_time>=1786770000000 AND source<>'e7_farside_test';
