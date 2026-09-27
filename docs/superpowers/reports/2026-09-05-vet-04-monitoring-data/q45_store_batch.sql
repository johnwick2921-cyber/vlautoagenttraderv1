.mode column
.headers on
SELECT '--- q45a: alert bell right now (unacked & not dismissed) ---' AS what;
SELECT level, COUNT(*) n FROM day_plan_alerts WHERE acked=0 AND dismissed=0 GROUP BY level;
SELECT COUNT(*) unacked_undismissed_total FROM day_plan_alerts WHERE acked=0 AND dismissed=0;
SELECT '--- q45b: RTH day range vs daily ATR(14) for 09-01..09-04 (1m MNQ 08:30-15:00 CT) ---' AS what;
WITH rth AS (
  SELECT date(open_time_ms/1000,'unixepoch','-5 hours') d, MAX(h) hi, MIN(l) lo, COUNT(*) n1m
  FROM bars WHERE symbol='MNQ' AND tf='1m' AND time(open_time_ms/1000,'unixepoch','-5 hours') BETWEEN '08:30:00' AND '14:59:00'
  AND date(open_time_ms/1000,'unixepoch','-5 hours') BETWEEN '2026-08-31' AND '2026-09-04' GROUP BY d),
d1 AS (
  SELECT date(open_time_ms/1000,'unixepoch','-5 hours') d, h, l, c, LAG(c) OVER (ORDER BY open_time_ms) pc FROM bars WHERE symbol='MNQ' AND tf='1d'),
tr AS (SELECT d, MAX(h-l, ABS(h-pc), ABS(l-pc)) tr FROM d1 WHERE pc IS NOT NULL),
atr AS (SELECT d, AVG(tr) OVER (ORDER BY d ROWS BETWEEN 13 PRECEDING AND CURRENT ROW) atr14 FROM tr)
SELECT rth.d, rth.n1m, rth.hi, rth.lo, ROUND(rth.hi-rth.lo,2) rth_range, ROUND(atr.atr14,2) atr14_daily, ROUND((rth.hi-rth.lo)/atr.atr14,2) range_over_atr
FROM rth LEFT JOIN atr ON atr.d = rth.d ORDER BY rth.d;
SELECT '--- q45c: arm working ages (created→updated, both normalised to CT) by terminal state ---' AS what;
SELECT state, COUNT(*) n,
 ROUND(AVG((strftime('%s',updated_at)-strftime('%s',created_at))/60.0),1) avg_min,
 ROUND(MIN((strftime('%s',updated_at)-strftime('%s',created_at))/60.0),1) min_min,
 ROUND(MAX((strftime('%s',updated_at)-strftime('%s',created_at))/60.0),1) max_min
FROM armed_orders WHERE session<>'TEST-E7' GROUP BY state;
SELECT '--- q45c2: the two arms still armed — age now (hours) ---' AS what;
SELECT id, session, scenario, state, datetime(strftime('%s',created_at),'unixepoch','-5 hours') created_ct, ROUND((strftime('%s','now')-strftime('%s',created_at))/3600.0,1) age_h FROM armed_orders WHERE state='armed';
SELECT '--- q45d: arms per CT day by state_reason (09-01..09-04) ---' AS what;
SELECT date(strftime('%s',created_at),'unixepoch','-5 hours') d, state, COALESCE(NULLIF(substr(state_reason,1,40),''),'(none)') reason, COUNT(*) n, GROUP_CONCAT(id) ids
FROM armed_orders WHERE date(strftime('%s',created_at),'unixepoch','-5 hours') BETWEEN '2026-09-01' AND '2026-09-04' AND session<>'TEST-E7' GROUP BY d, state, reason ORDER BY d, state;
SELECT '--- q45e: decision gaps > 10 min per CT day 09-01..09-04 ---' AS what;
WITH t AS (SELECT id, date(timestamp,'-5 hours') d, strftime('%s',timestamp) s, LAG(strftime('%s',timestamp)) OVER (ORDER BY timestamp) ps, LAG(id) OVER (ORDER BY timestamp) pid FROM decision_records WHERE date(timestamp,'-5 hours') BETWEEN '2026-09-01' AND '2026-09-04')
SELECT d, pid, id, datetime(ps,'unixepoch') prev_utc, ROUND((s-ps)/60.0,1) gap_min FROM t WHERE s-ps>600 ORDER BY d, s;
SELECT '--- q45f: cycle_type distribution 09-03 / 09-04 ---' AS what;
SELECT date(timestamp,'-5 hours') d, COALESCE(cycle_type,'(null)') cycle_type, COUNT(*) n FROM decision_records WHERE date(timestamp,'-5 hours') IN ('2026-09-03','2026-09-04') GROUP BY d, cycle_type ORDER BY d, n DESC;
SELECT '--- q45g: planner reads per session per day (planner_read_facts) ---' AS what;
SELECT trade_date, session, COUNT(*) reads, ROUND(AVG(atr5m),2) avg_atr5m, ROUND(AVG(stop_floor_pts),1) avg_floor FROM planner_read_facts WHERE trade_date BETWEEN '2026-09-01' AND '2026-09-04' GROUP BY trade_date, session ORDER BY trade_date, session;
SELECT '--- q45h: silent refusals 09-03 — execution_log samples with entry_gate ---' AS what;
SELECT id, datetime(timestamp,'-5 hours') ct, substr(execution_log,1,220) exec FROM decision_records WHERE date(timestamp,'-5 hours')='2026-09-03' AND execution_log LIKE '%entry_gate%' ORDER BY timestamp LIMIT 6;
SELECT '--- q45i: ab_confirm_log time_to_fill for real fills (last 10) ---' AS what;
SELECT id, session, scenario, condition, rule, fill_px, entry_px, ROUND((fill_px-entry_px),2) fill_minus_entry, time_to_fill_ms/1000 ttf_s, is_counterfactual, datetime(strftime('%s',created_at),'unixepoch','-5 hours') ct FROM ab_confirm_log WHERE is_counterfactual=0 AND fill_px>0 ORDER BY created_at DESC LIMIT 10;
SELECT '--- q45j: day_plan_alerts kinds on 09-04 ---' AS what;
SELECT id, datetime(created_at,'unixepoch','-5 hours') ct, level, kind, substr(title,1,60) title, acked, dismissed FROM day_plan_alerts WHERE date(created_at,'unixepoch','-5 hours')='2026-09-04' ORDER BY created_at;
