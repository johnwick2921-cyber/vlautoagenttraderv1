.mode column
.headers on
SELECT '--- touch_outcomes 2026-09-03 (CT) by level_kind × outcome ---' AS what;
SELECT level_kind, outcome, COUNT(*) n FROM touch_outcomes WHERE date(created_at,'-5 hours')='2026-09-03' GROUP BY level_kind, outcome ORDER BY level_kind, outcome;
SELECT '--- planner_read_facts 2026-09-03 ---' AS what;
SELECT session, version, void_count, stop_floor_pts, atr5m, bias_ai, bias_tree, bias_regime, tokens_in, scope_bars, datetime(created_at,'-5 hours') ct FROM planner_read_facts WHERE trade_date='2026-09-03' ORDER BY created_at;
SELECT '--- plans 2026-09-03: versions per session + lifecycle ---' AS what;
SELECT session, COUNT(*) versions, MAX(version) max_v, GROUP_CONCAT(DISTINCT lifecycle) lifecycles, SUM(degraded) degraded, GROUP_CONCAT(DISTINCT trigger_reason) triggers FROM plans WHERE trade_date='2026-09-03' GROUP BY session;
SELECT '--- silent entry_gate refusals per CT day (decision_records.execution_log) ---' AS what;
SELECT date(timestamp,'-5 hours') d, COUNT(*) cycles, SUM(execution_log LIKE '%entry_gate%') entry_gate_mentions, SUM(execution_log LIKE '%REFUSED%') refused_mentions, SUM(execution_log LIKE '%no_balance%') no_balance FROM decision_records WHERE date(timestamp,'-5 hours') BETWEEN '2026-09-01' AND '2026-09-04' GROUP BY d;
SELECT '--- arm refusal counters (system_config) ---' AS what;
SELECT key, value FROM system_config WHERE key LIKE 'arm_refusals_0b:%' OR key LIKE 'arms_boot_swept%' OR key LIKE 'arm_superseded%' ORDER BY key DESC LIMIT 10;
SELECT '--- nt8_order_snapshots freshness now ---' AS what;
SELECT id, datetime(received_at_ms/1000,'unixepoch','-5 hours') received_ct, reason, order_count, working_count, build_id, (strftime('%s','now')*1000 - received_at_ms)/1000 age_s FROM nt8_order_snapshots ORDER BY received_at_ms DESC LIMIT 2;
