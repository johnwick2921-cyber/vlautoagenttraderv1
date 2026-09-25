-- session OHLC from 1m MNQ bars; sessions per ledger: ASIA 17:00→02:00, LONDON 02:00→08:30, NY 08:30→14:45 CT (NY extended to 16:00 for whole-RTH view)
WITH b AS (
  SELECT open_time_ms, datetime(open_time_ms/1000,'unixepoch','-5 hours') ct, o,h,l,c,v,
    CASE
      WHEN time(open_time_ms/1000,'unixepoch','-5 hours') >= '17:00:00' THEN date(open_time_ms/1000,'unixepoch','-5 hours')||' ASIA'
      WHEN time(open_time_ms/1000,'unixepoch','-5 hours') < '02:00:00' THEN date(open_time_ms/1000,'unixepoch','-5 hours','-1 day')||' ASIA'
      WHEN time(open_time_ms/1000,'unixepoch','-5 hours') < '08:30:00' THEN date(open_time_ms/1000,'unixepoch','-5 hours')||' LONDON'
      WHEN time(open_time_ms/1000,'unixepoch','-5 hours') < '14:45:00' THEN date(open_time_ms/1000,'unixepoch','-5 hours')||' NY'
      ELSE date(open_time_ms/1000,'unixepoch','-5 hours')||' NY-late(14:45-17:00)'
    END sess
  FROM bars WHERE symbol='MNQ' AND tf='1m'
   AND open_time_ms BETWEEN strftime('%s','2026-09-01 22:00:00')*1000 AND strftime('%s','2026-09-05 00:00:00')*1000
)
SELECT sess, COUNT(*) n1m, MIN(ct) first, MAX(ct) last,
  (SELECT o FROM b b2 WHERE b2.sess=b.sess ORDER BY open_time_ms LIMIT 1) open,
  MAX(h) high, MIN(l) low,
  (SELECT c FROM b b2 WHERE b2.sess=b.sess ORDER BY open_time_ms DESC LIMIT 1) close,
  MAX(h)-MIN(l) range, SUM(v) vol
FROM b GROUP BY sess ORDER BY MIN(open_time_ms);
