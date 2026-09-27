.mode list
.headers on
SELECT '--- A: bars coverage per era trade (first 6 by id)';
SELECT p.id, datetime(p.entry_time/1000,'unixepoch','-5 hours') entry_ct,
 (SELECT count(*) FROM bars b WHERE b.symbol='MNQ' AND b.tf='1m' AND b.open_time_ms BETWEEN p.entry_time-60000 AND p.exit_time+60000) nb
FROM trader_positions p WHERE p.entry_time>=1786770000000 AND p.source<>'e7_farside_test' AND p.pnl_corrected IS NOT NULL
ORDER BY p.id LIMIT 6;
SELECT '--- A2: how many era trades have zero 1m bars';
SELECT count(*) FROM (SELECT p.id FROM trader_positions p WHERE p.entry_time>=1786770000000 AND p.source<>'e7_farside_test' AND p.pnl_corrected IS NOT NULL
 AND (SELECT count(*) FROM bars b WHERE b.symbol='MNQ' AND b.tf='1m' AND b.open_time_ms BETWEEN p.entry_time-60000 AND p.exit_time+60000)=0);
SELECT '--- A3: MNQ 1m min/max in CT';
SELECT tf, coalesce(convention,'<NULL>'), count(*), count(DISTINCT open_time_ms),
 datetime(min(open_time_ms)/1000,'unixepoch','-5 hours'), datetime(max(open_time_ms)/1000,'unixepoch','-5 hours')
FROM bars WHERE symbol='MNQ' GROUP BY 1,2;
SELECT '--- A4: duplicate open_time_ms in MNQ 1m';
SELECT count(*) FROM (SELECT open_time_ms FROM bars WHERE symbol='MNQ' AND tf='1m' GROUP BY 1 HAVING count(*)>1);
SELECT '--- A5: ES bars by tf';
SELECT tf,count(*) FROM bars WHERE symbol='ES' GROUP BY 1;
SELECT '--- B: strict refusals by CT day and UTC day';
SELECT date(timestamp,'-5 hours') ct_day, date(timestamp) utc_day, count(*), min(timestamp), max(timestamp)
FROM decision_records WHERE risk_check_error LIKE '%strict%' GROUP BY 1,2;
SELECT '--- C: nt8 snapshots working_count>=8 on 09-04';
SELECT count(*), min(id), max(id), datetime(min(received_at_ms)/1000,'unixepoch','-5 hours'), datetime(max(received_at_ms)/1000,'unixepoch','-5 hours')
FROM nt8_order_snapshots WHERE working_count>=8;
SELECT '--- D: S2 arm rows 09-04 NY';
SELECT id, state_reason, datetime(strftime('%s',created_at),'unixepoch','-5 hours') FROM armed_orders WHERE scenario='S2' AND plan_id LIKE '2026-09-04:NY%' AND id>=38 ORDER BY id;
SELECT '--- E: plans count';
SELECT count(*) total, sum(trade_date>='2026-08-15') since, min(trade_date), max(trade_date) FROM plans;
SELECT '--- F: ab_confirm_log recompute classes';
SELECT coalesce(recompute,'<NULL>'), count(*), min(net_pnl), max(net_pnl), sum(net_pnl < -1000) FROM ab_confirm_log GROUP BY 1;
SELECT '--- G: armed_orders filled rows';
SELECT id, entry_px, fill_price, state, scenario, kind FROM armed_orders WHERE state='filled' ORDER BY id;
SELECT '--- H: touch_outcomes window';
SELECT count(*), datetime(min(opened_at_ms)/1000,'unixepoch','-5 hours'), datetime(max(opened_at_ms)/1000,'unixepoch','-5 hours') FROM touch_outcomes;
SELECT '--- H2: era trades entering inside touch_outcomes window';
SELECT count(*) FROM trader_positions p WHERE p.entry_time>=1786770000000 AND p.source<>'e7_farside_test' AND p.pnl_corrected IS NOT NULL
 AND p.entry_time BETWEEN (SELECT min(opened_at_ms) FROM touch_outcomes) AND (SELECT max(opened_at_ms) FROM touch_outcomes);
SELECT '--- I: touch_episodes window';
SELECT count(*), datetime(min(opened_at_ms)/1000,'unixepoch','-5 hours'), datetime(max(opened_at_ms)/1000,'unixepoch','-5 hours') FROM touch_episodes;
SELECT '--- I2: era trades inside touch_episodes window';
SELECT count(*) FROM trader_positions p WHERE p.entry_time>=1786770000000 AND p.source<>'e7_farside_test' AND p.pnl_corrected IS NOT NULL
 AND p.entry_time BETWEEN (SELECT min(opened_at_ms) FROM touch_episodes) AND (SELECT max(opened_at_ms) FROM touch_episodes);
SELECT '--- J: candidate_pool round numbers';
SELECT count(*) total, sum(CAST(level_price AS INT)=level_price AND CAST(level_price AS INT)%100=0) m100,
 sum(CAST(level_price AS INT)=level_price AND CAST(level_price AS INT)%50=0) m50,
 sum(CAST(level_price AS INT)=level_price AND CAST(level_price AS INT)%25=0) m25 FROM candidate_pool;
