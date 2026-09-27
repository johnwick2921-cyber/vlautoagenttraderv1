-- q05: re-measure the 09-03 outage from store timestamps (all CT via -5h)
.mode column
.headers on
SELECT 'decision_records around gap' AS what;
SELECT id, datetime(timestamp,'-5 hours') AS ct, cycle_type, substr(execution_log,1,60) AS exec FROM decision_records
 WHERE date(timestamp,'-5 hours')='2026-09-03' AND time(timestamp,'-5 hours') BETWEEN '12:10:00' AND '15:15:00' ORDER BY timestamp;
SELECT 'largest decision gaps 09-03' AS what;
WITH t AS (SELECT id, datetime(timestamp,'-5 hours') ct, strftime('%s',timestamp) s, LAG(strftime('%s',timestamp)) OVER (ORDER BY timestamp) ps, LAG(id) OVER (ORDER BY timestamp) pid FROM decision_records WHERE date(timestamp,'-5 hours')='2026-09-03')
SELECT pid, id, ct, (s-ps)/60.0 AS gap_min FROM t WHERE s-ps > 600 ORDER BY gap_min DESC LIMIT 8;
SELECT 'nt8_order_snapshots gaps 09-03 (>5 min)' AS what;
WITH t AS (SELECT id, received_at_ms r, LAG(received_at_ms) OVER (ORDER BY received_at_ms) pr FROM nt8_order_snapshots WHERE date(received_at_ms/1000,'unixepoch','-5 hours')='2026-09-03')
SELECT id, datetime(pr/1000,'unixepoch','-5 hours') AS prev_ct, datetime(r/1000,'unixepoch','-5 hours') AS ct, (r-pr)/60000.0 AS gap_min FROM t WHERE r-pr > 300000 ORDER BY r;
SELECT 'log_events gaps 09-03 (>10 min)' AS what;
