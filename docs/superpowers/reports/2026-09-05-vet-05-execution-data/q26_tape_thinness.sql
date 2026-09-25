.mode column
.headers on
-- 1m MNQ bars since 2026-08-19, weekdays (CT), by 10-min bucket: mean volume, mean range, n
WITH b AS (
  SELECT datetime(open_time_ms/1000,'unixepoch','-5 hours') ct, strftime('%w', open_time_ms/1000,'unixepoch','-5 hours') dow,
         strftime('%H:%M', open_time_ms/1000,'unixepoch','-5 hours') hm, h-l rng, v
  FROM bars WHERE tf='1m' AND symbol='MNQ' AND open_time_ms >= strftime('%s','2026-08-19 05:00:00')*1000
)
SELECT bucket, COUNT(*) n, ROUND(AVG(v),0) mean_vol, ROUND(AVG(rng),2) mean_range_pts, ROUND(MAX(rng),2) max_range
FROM (SELECT CASE WHEN hm BETWEEN '08:30' AND '08:39' THEN 'a 08:30-08:39 open'
                  WHEN hm BETWEEN '09:30' AND '09:39' THEN 'b 09:30-09:39'
                  WHEN hm BETWEEN '10:00' AND '10:09' THEN 'c 10:00-10:09'
                  WHEN hm BETWEEN '12:00' AND '12:09' THEN 'd 12:00-12:09 lunch'
                  WHEN hm BETWEEN '13:00' AND '13:09' THEN 'e 13:00-13:09'
                  WHEN hm BETWEEN '14:40' AND '14:49' THEN 'f 14:40-14:49 EOD flat'
                  WHEN hm BETWEEN '14:55' AND '15:04' THEN 'g 14:55-15:04 close/settle'
                  WHEN hm BETWEEN '01:30' AND '01:39' THEN 'h 01:30-01:39 LONDON'
                  WHEN hm BETWEEN '17:00' AND '17:09' THEN 'i 17:00-17:09 ASIA open'
                  WHEN hm BETWEEN '20:00' AND '20:09' THEN 'j 20:00-20:09 ASIA' END bucket, v, rng
      FROM b WHERE dow BETWEEN '1' AND '5')
WHERE bucket IS NOT NULL GROUP BY bucket ORDER BY bucket;
