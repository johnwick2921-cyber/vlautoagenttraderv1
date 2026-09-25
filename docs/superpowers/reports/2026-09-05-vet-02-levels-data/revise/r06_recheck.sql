.mode list
.headers off
SELECT 'level_state', COUNT(*), SUM(consumed=1), SUM(consumed=1 AND times_tested=1) FROM level_state;
SELECT 'cited_all', COUNT(*) FROM trader_positions WHERE cited_scenario_id IS NOT NULL AND cited_scenario_id<>'' AND entry_time>=1755234000000;
SELECT 'cited_e7', COUNT(*) FROM trader_positions WHERE cited_scenario_id IS NOT NULL AND cited_scenario_id<>'' AND entry_time>=1755234000000 AND source='e7_farside_test';
SELECT 'cited_unres', GROUP_CONCAT(id) FROM trader_positions WHERE cited_scenario_id IS NOT NULL AND cited_scenario_id<>'' AND entry_time>=1755234000000 AND plan_id='UNRESOLVABLE';
SELECT 'cited_null', GROUP_CONCAT(id) FROM trader_positions WHERE cited_scenario_id IS NOT NULL AND cited_scenario_id<>'' AND entry_time>=1755234000000 AND source<>'e7_farside_test' AND plan_id<>'UNRESOLVABLE' AND pnl_corrected IS NULL;
SELECT 'unres_era_all', GROUP_CONCAT(id), COUNT(*), ROUND(SUM(COALESCE(pnl_corrected,0)),2) FROM trader_positions WHERE plan_id='UNRESOLVABLE' AND entry_time>=1755234000000 AND source<>'e7_farside_test';
SELECT 'pool_reads', COUNT(DISTINCT read_at_ms), COUNT(DISTINCT plan_id||'|'||plan_version) FROM candidate_pool;
SELECT 'pool_seatfull', seatn, COUNT(*) FROM (SELECT read_at_ms, SUM(seated) seatn FROM candidate_pool GROUP BY read_at_ms) GROUP BY seatn;
SELECT 'arm_split', substr(state_reason,1,28), COUNT(*), GROUP_CONCAT(id) FROM armed_orders WHERE id BETWEEN 62 AND 102 AND scenario='S2' AND condition='reclaim' AND entry_px=29591.02 GROUP BY 2;
SELECT 'bars_sessday', COUNT(*), n FROM (SELECT COUNT(*) n FROM bars WHERE symbol='MNQ' AND tf='1m' GROUP BY date(datetime(open_time_ms/1000,'unixepoch','-5 hours','+7 hours'))) GROUP BY n;
