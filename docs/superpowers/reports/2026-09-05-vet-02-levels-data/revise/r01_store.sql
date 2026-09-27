.mode list
.headers on
SELECT 'level_state' AS q, COUNT(*) n, SUM(consumed=1) consumed, SUM(consumed=1 AND times_tested=1) consumed_tt1, SUM(consumed=1 AND level_type='other') consumed_other FROM level_state;
SELECT 'pos_ids' AS q, id, source, plan_id, cited_scenario_id, pnl_corrected FROM trader_positions WHERE id IN (567,587,578,576,577,579,530,539,572) ORDER BY id;
SELECT 'cited_pop' AS q, COUNT(*) all_cited, SUM(source='e7_farside_test') e7, SUM(plan_id='UNRESOLVABLE') unres, SUM(pnl_corrected IS NULL AND source<>'e7_farside_test' AND plan_id<>'UNRESOLVABLE') null_in_scope FROM trader_positions WHERE cited_scenario_id IS NOT NULL AND cited_scenario_id<>'' AND entry_time >= 1755234000000;
SELECT 'unres_ids' AS q, GROUP_CONCAT(id) ids, SUM(pnl_corrected) s FROM trader_positions WHERE plan_id='UNRESOLVABLE' AND entry_time >= 1755234000000;
SELECT 'unres_cited' AS q, id, cited_scenario_id, pnl_corrected FROM trader_positions WHERE plan_id='UNRESOLVABLE' AND cited_scenario_id IS NOT NULL AND cited_scenario_id<>'';
SELECT 'null_cited' AS q, GROUP_CONCAT(id) ids FROM trader_positions WHERE cited_scenario_id IS NOT NULL AND cited_scenario_id<>'' AND entry_time >= 1755234000000 AND pnl_corrected IS NULL AND source<>'e7_farside_test';
SELECT 'arm20' AS q, substr(state_reason,1,40) reason, COUNT(*) n, GROUP_CONCAT(id) ids FROM armed_orders WHERE id BETWEEN 62 AND 102 AND scenario='S2' AND condition='reclaim' AND entry_px=29591.02 GROUP BY 1;
SELECT 'arm_other_62_102' AS q, id, scenario, condition, side, entry_px, substr(state_reason,1,40) FROM armed_orders WHERE id BETWEEN 62 AND 102 AND NOT (scenario='S2' AND condition='reclaim' AND entry_px=29591.02);
SELECT 'arm38' AS q, id, scenario, condition, entry_px, substr(state_reason,1,40) FROM armed_orders WHERE id=38;
SELECT 'pool_reads' AS q, datetime(read_at_ms/1000,'unixepoch','-5 hours') read_ct, plan_id, plan_version, session, COUNT(*) n, SUM(seated) seated FROM candidate_pool GROUP BY read_at_ms ORDER BY read_at_ms;
SELECT 'pool_planver' AS q, COUNT(DISTINCT plan_id||'|'||plan_version) planvers, COUNT(DISTINCT read_at_ms) reads FROM candidate_pool;
SELECT 'pool_distinct' AS q, seated, COUNT(*) rows_, COUNT(DISTINCT level_kind||'|'||level_price) lv, COUNT(DISTINCT read_at_ms) reads, COUNT(DISTINCT date(read_at_ms/1000,'unixepoch','-5 hours')) days FROM candidate_pool GROUP BY seated;
SELECT 'strategies' AS q, id, name, json_extract(config,'$.day_plan.max_levels') ml, json_extract(config,'$.day_plan.proximity_filter_atr') prox FROM strategies;
SELECT 'bars_tf' AS q, tf, COUNT(*) n, datetime(MIN(open_time_ms)/1000,'unixepoch','-5 hours') mn, datetime(MAX(open_time_ms)/1000,'unixepoch','-5 hours') mx FROM bars WHERE symbol='MNQ' GROUP BY tf;
SELECT 'bars_1m_sessday' AS q, date((open_time_ms/1000)-17*3600+86400,'unixepoch','-5 hours') sd, COUNT(*) n FROM bars WHERE symbol='MNQ' AND tf='1m' GROUP BY sd ORDER BY sd;
SELECT 'bars_1m_days' AS q, COUNT(DISTINCT date(open_time_ms/1000,'unixepoch','-5 hours')) cal_days FROM bars WHERE symbol='MNQ' AND tf='1m';
