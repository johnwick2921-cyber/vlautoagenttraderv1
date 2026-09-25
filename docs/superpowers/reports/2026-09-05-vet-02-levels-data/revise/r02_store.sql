.mode list
.headers on
SELECT substr(state_reason,1,45) reason, COUNT(*) n, GROUP_CONCAT(id) ids FROM armed_orders WHERE id BETWEEN 62 AND 102 AND scenario='S2' AND condition='reclaim' AND entry_px=29591.02 GROUP BY reason;
SELECT 'all_rr' q, COUNT(*) FROM armed_orders WHERE state_reason LIKE 'gate changed: rr%';
SELECT 'unres_detail' q, id, source, cited_scenario_id, pnl_corrected, datetime(entry_time/1000,'unixepoch','-5 hours') et FROM trader_positions WHERE plan_id='UNRESOLVABLE' ORDER BY id;
SELECT 'rthl_conflict' q, datetime(opened_at_ms/1000,'unixepoch','-5 hours') opened_ct, outcome, COUNT(*) n FROM touch_outcomes WHERE level_kind='RTH-L' AND level_price=29199.25 GROUP BY opened_at_ms, outcome ORDER BY opened_at_ms;
SELECT 'rthl_formation_bar' q, datetime(open_time_ms/1000,'unixepoch','-5 hours') t, l FROM bars WHERE symbol='MNQ' AND tf='1m' AND l=29199.25 AND date(open_time_ms/1000,'unixepoch','-5 hours')='2026-09-03';
SELECT 'demand_conflict' q, level_price, datetime(opened_at_ms/1000,'unixepoch','-5 hours') opened_ct, outcome, COUNT(*) n FROM touch_outcomes WHERE level_kind='DEMAND' GROUP BY level_price, opened_at_ms, outcome HAVING level_price IN (SELECT level_price FROM touch_outcomes t2 WHERE t2.level_kind='DEMAND' AND t2.opened_at_ms=touch_outcomes.opened_at_ms GROUP BY level_price, opened_at_ms HAVING COUNT(DISTINCT outcome)>1) ORDER BY level_price, opened_at_ms;
SELECT 'fortnight' q, date(open_time_ms/1000,'unixepoch','-5 hours') d, MIN(l) lo, MAX(h) hi FROM bars WHERE symbol='MNQ' AND tf='1m' AND date(open_time_ms/1000,'unixepoch','-5 hours') BETWEEN '2026-08-21' AND '2026-09-03' GROUP BY d;
SELECT 'close_0821' q, c FROM bars WHERE symbol='MNQ' AND tf='1m' AND date(open_time_ms/1000,'unixepoch','-5 hours')='2026-08-21' ORDER BY open_time_ms DESC LIMIT 1;
SELECT 'close_0903' q, c FROM bars WHERE symbol='MNQ' AND tf='1m' AND date(open_time_ms/1000,'unixepoch','-5 hours')='2026-09-03' ORDER BY open_time_ms DESC LIMIT 1;
SELECT 'vwap2s_census' q, COUNT(DISTINCT trade_date||'|'||session) sessions, COUNT(*) rows_ FROM (SELECT p.trade_date, p.session, j.value AS lv FROM plans p, json_each(json_extract(p.doc,'$.levels')) j WHERE p.session<>'WEEKLY' AND json_extract(j.value,'$.label') LIKE 'VWAP%2σ%');
