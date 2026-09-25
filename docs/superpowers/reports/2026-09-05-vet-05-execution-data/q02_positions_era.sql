.mode column
.headers on
-- era = entry_time >= 2026-08-15 00:00 CT (epoch ms)
SELECT 'era_rows' k, COUNT(*) v FROM trader_positions WHERE entry_time >= 1786... ;
