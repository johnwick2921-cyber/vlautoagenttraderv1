.mode column
.headers on
SELECT 'log_events ts_utc sample' AS what;
SELECT id, ts_utc, level, component FROM log_events ORDER BY id DESC LIMIT 2;
SELECT 'log_events around 25754' AS what;
SELECT id, ts_utc, level, component, substr(message,1,80) msg FROM log_events WHERE id BETWEEN 25750 AND 25760 ORDER BY id;
SELECT 'log_events gaps 09-03 by epoch' AS what;
WITH t AS (SELECT id, ts_utc, CAST(strftime('%s',replace(substr(ts_utc,1,19),'T',' ')) AS INTEGER) s FROM log_events WHERE substr(ts_utc,1,10) IN ('2026-09-03')),
u AS (SELECT id, ts_utc, s, LAG(s) OVER (ORDER BY s,id) ps, LAG(id) OVER (ORDER BY s,id) pid FROM t)
SELECT pid, id, ts_utc, (s-ps)/60.0 gap_min FROM u WHERE s-ps>600 ORDER BY s;
SELECT 'day_plan_alerts 09-03 CT (epoch-mix normalised)' AS what;
SELECT id, datetime(CASE WHEN created_at>100000000000 THEN created_at/1000 ELSE created_at END,'unixepoch','-5 hours') ct, level, kind, substr(title,1,60) title, acked FROM day_plan_alerts
 WHERE date(CASE WHEN created_at>100000000000 THEN created_at/1000 ELSE created_at END,'unixepoch','-5 hours')='2026-09-03' ORDER BY 2;
SELECT 'day_plan_alerts epoch-mix count' AS what;
SELECT CASE WHEN created_at>100000000000 THEN 'ms' ELSE 's' END unit, COUNT(*) n, MIN(id), MAX(id) FROM day_plan_alerts GROUP BY 1;
