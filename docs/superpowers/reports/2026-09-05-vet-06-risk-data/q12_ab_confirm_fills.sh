#!/bin/bash
# q12 — ab_confirm_log net_pnl sanity (avg -18321 looked non-dollar) + broker-side commission in trader_fills
DB="file:/home/hoang/nofx/data/data.db?mode=ro"
S() { echo "--- $1"; sqlite3 -header "$DB" "$2"; }
S "ab_confirm_log net_pnl distribution" "SELECT COUNT(*) n, ROUND(MIN(net_pnl),2) mn, ROUND(MAX(net_pnl),2) mx, ROUND(AVG(net_pnl),2) avg, SUM(net_pnl=0) zeros, SUM(ABS(net_pnl)>1000) gt1000 FROM ab_confirm_log;"
S "ab_confirm_log: rows with |net_pnl|>1000 (ids, session, condition, rule, entry/exit, net_pnl, normalized)" "SELECT id, session, condition, rule, entry_px, stop_px, target_px, ROUND(net_pnl,2), normalized, recompute FROM ab_confirm_log WHERE ABS(net_pnl)>1000 ORDER BY ABS(net_pnl) DESC LIMIT 6;"
S "ab_confirm_log: rule x condition x n, usable (net_pnl<>0)" "SELECT rule, condition, COUNT(*) n, SUM(net_pnl<>0) nonzero, ROUND(AVG(CASE WHEN net_pnl<>0 THEN net_pnl END),2) avg_nonzero FROM ab_confirm_log GROUP BY 1,2 ORDER BY n DESC LIMIT 10;"
S "trader_fills columns" "PRAGMA table_info(trader_fills);"
S "trader_fills era: n, nonzero commission, sum" "SELECT COUNT(*), SUM(COALESCE(commission,0)<>0), ROUND(SUM(COALESCE(commission,0)),2) FROM trader_fills;"
