-- q32: what does exit_order_id carry on the 65 usable era rows? (close_reason is 'sync' on all)
.mode column
.headers on
SELECT exit_order_id, close_reason, COUNT(*) n, GROUP_CONCAT(id) ids
FROM trader_positions
WHERE entry_time >= 1786770000000 AND source <> 'e7_farside_test' AND pnl_corrected IS NOT NULL
GROUP BY exit_order_id, close_reason ORDER BY n DESC;
-- the 18 'neither target nor stop' exits
SELECT id, source, exit_order_id, close_reason, pnl_corrected
FROM trader_positions WHERE id IN (526,530,538,542,551,557,558,560,566,568,569,570,571,575,578,580,581,591) ORDER BY id;
