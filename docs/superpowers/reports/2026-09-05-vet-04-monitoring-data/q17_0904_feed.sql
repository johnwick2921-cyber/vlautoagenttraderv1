.mode column
.headers on
SELECT 'decision_records 09-04 by CT hour' AS what;
SELECT strftime('%H', timestamp,'-5 hours') h, COUNT(*) n, MIN(id) min_id, MAX(id) max_id FROM decision_records WHERE date(timestamp,'-5 hours')='2026-09-04' GROUP BY h ORDER BY h;
SELECT 'bars 1m MNQ last 5 (CT)' AS what;
SELECT symbol, tf, datetime(open_time_ms/1000,'unixepoch','-5 hours') open_ct, c FROM bars WHERE symbol='MNQ' AND tf='1m' ORDER BY open_time_ms DESC LIMIT 5;
SELECT 'bars 1m MNQ count per CT hour 09-04' AS what;
SELECT strftime('%H', open_time_ms/1000,'unixepoch','-5 hours') h, COUNT(*) n FROM bars WHERE symbol='MNQ' AND tf='1m' AND date(open_time_ms/1000,'unixepoch','-5 hours')='2026-09-04' GROUP BY h ORDER BY h;
SELECT 'equity snapshots 09-04 last 3' AS what;
SELECT id, datetime(timestamp,'-5 hours') ct, total_equity, position_count FROM trader_equity_snapshots WHERE date(timestamp,'-5 hours')='2026-09-04' ORDER BY timestamp DESC LIMIT 3;
