.mode column
.headers on
SELECT 'snapshots per CT day' AS what;
SELECT date(received_at_ms/1000,'unixepoch','-5 hours') d, COUNT(*) n, datetime(MIN(received_at_ms)/1000,'unixepoch','-5 hours') first_ct, datetime(MAX(received_at_ms)/1000,'unixepoch','-5 hours') last_ct FROM nt8_order_snapshots GROUP BY d ORDER BY d;
SELECT 'log_events gaps 09-03 (>10 min, CT)' AS what;
WITH t AS (SELECT id, ts_utc, strftime('%s',ts_utc) s, LAG(strftime('%s',ts_utc)) OVER (ORDER BY ts_utc) ps, LAG(id) OVER (ORDER BY ts_utc) pid FROM log_events WHERE date(ts_utc,'-5 hours')='2026-09-03')
SELECT pid, id, datetime(ts_utc,'-5 hours') ct, (s-ps)/60.0 gap_min FROM t WHERE s-ps>600 ORDER BY ts_utc;
SELECT 'log_events last before / first after silence' AS what;
SELECT id, datetime(ts_utc,'-5 hours') ct, level, component, substr(message,1,90) msg FROM log_events WHERE date(ts_utc,'-5 hours')='2026-09-03' AND time(ts_utc,'-5 hours') BETWEEN '12:20:00' AND '14:25:00' ORDER BY ts_utc LIMIT 12;
SELECT 'equity snapshot gaps 09-03 (>10 min)' AS what;
WITH t AS (SELECT id, timestamp, strftime('%s',timestamp) s, LAG(strftime('%s',timestamp)) OVER (ORDER BY timestamp) ps, LAG(id) OVER (ORDER BY timestamp) pid, total_equity FROM trader_equity_snapshots WHERE date(timestamp,'-5 hours')='2026-09-03')
SELECT pid, id, datetime(timestamp,'-5 hours') ct, (s-ps)/60.0 gap_min, total_equity FROM t WHERE s-ps>600 ORDER BY timestamp;
SELECT 'equity snapshot timestamp sample (tz check)' AS what;
SELECT id, timestamp, created_at, total_equity FROM trader_equity_snapshots ORDER BY id DESC LIMIT 2;
SELECT 'day_plan_alerts 09-03 12:00-15:30 CT' AS what;
SELECT id, datetime(created_at,'-5 hours') ct_if_utc, created_at, level, kind, substr(title,1,70) title, acked FROM day_plan_alerts WHERE created_at LIKE '2026-09-03%' ORDER BY created_at;
SELECT 'day_plan_alerts kinds all-time' AS what;
SELECT kind, level, COUNT(*) n, MAX(created_at) last FROM day_plan_alerts GROUP BY kind, level ORDER BY n DESC;
SELECT 'telegram bound?' AS what;
SELECT id, length(token_col)>0 AS has_token, length(chat_id)>0 AS has_chat, bound_at, language FROM telegram_configs;
