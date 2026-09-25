.mode column
.headers on
SELECT 'log_events gaps 09-03 CT (>10 min; ts_utc is EPOCH MS)' AS what;
WITH t AS (SELECT id, ts_utc s, LAG(ts_utc) OVER (ORDER BY ts_utc,id) ps, LAG(id) OVER (ORDER BY ts_utc,id) pid FROM log_events WHERE date(ts_utc/1000,'unixepoch','-5 hours')='2026-09-03')
SELECT pid, id, datetime(ps/1000,'unixepoch','-5 hours') prev_ct, datetime(s/1000,'unixepoch','-5 hours') ct, round((s-ps)/60000.0,2) gap_min FROM t WHERE s-ps>600000 ORDER BY s;
SELECT 'day_plan_alerts ack/dismiss rate all-time' AS what;
SELECT level, COUNT(*) n, SUM(acked) acked, SUM(dismissed) dismissed FROM day_plan_alerts GROUP BY level;
SELECT 'feed-down alert bodies' AS what;
SELECT id, datetime(created_at,'unixepoch','-5 hours') ct, title, substr(body,1,200) body, acked, dismissed FROM day_plan_alerts WHERE kind='feed-down' ORDER BY id;
SELECT 'alerts per CT day last 7 days' AS what;
SELECT date(CASE WHEN created_at>100000000000 THEN created_at/1000 ELSE created_at END,'unixepoch','-5 hours') d, COUNT(*) n, SUM(level='P0') p0, SUM(acked) acked FROM day_plan_alerts GROUP BY d ORDER BY d DESC LIMIT 8;
