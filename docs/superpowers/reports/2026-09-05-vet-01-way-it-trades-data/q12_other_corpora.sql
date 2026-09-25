.mode column
.headers on
SELECT kind, role, COUNT(*) n, SUM(touched) touched, SUM(reacted) reacted, SUM(broke_clean) broke, SUM(chopped) chopped, COUNT(DISTINCT session_day) days FROM level_stats GROUP BY 1,2 ORDER BY touched DESC;
SELECT MIN(session_day), MAX(session_day), COUNT(DISTINCT session_day) FROM level_stats;
SELECT label, COUNT(*) n, COUNT(DISTINCT session_day) days, ROUND(AVG(penetration_pts),1) pen, ROUND(AVG(wick_pen_pts),1) wick, ROUND(AVG(body_pen_pts),1) body, shape FROM touch_episodes GROUP BY label, shape ORDER BY n DESC LIMIT 40;
SELECT shape, COUNT(*) FROM touch_episodes GROUP BY 1;
SELECT MIN(session_day), MAX(session_day), COUNT(DISTINCT session_day), MAX(touch_number) FROM touch_episodes;
SELECT * FROM touch_episodes ORDER BY id DESC LIMIT 3;
SELECT trade_date, session, version, stop_floor_pts, atr5m, stop_floor_mlt, bias_ai, bias_tree, bias_regime, void_count, scope_bars FROM planner_read_facts ORDER BY created_at;
