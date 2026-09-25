.mode list
.headers on
SELECT 'total' k, COUNT(*) n FROM planner_rejected_prompts
UNION ALL SELECT 'full(len>=20000)', COUNT(*) FROM planner_rejected_prompts WHERE length(prompt_text)>=20000
UNION ALL SELECT 'repair(len<8000)', COUNT(*) FROM planner_rejected_prompts WHERE length(prompt_text)<8000
UNION ALL SELECT 'killzone_0730', COUNT(*) FROM planner_rejected_prompts WHERE prompt_text LIKE '%NY AM 07:30–10:00 CT%'
UNION ALL SELECT 'conviction_monday', COUNT(*) FROM planner_rejected_prompts WHERE prompt_text LIKE '%Conviction: down on Monday%'
UNION ALL SELECT 'week75', COUNT(*) FROM planner_rejected_prompts WHERE prompt_text LIKE '%75% win, +665 this week%'
UNION ALL SELECT 'accept0', COUNT(*) FROM planner_rejected_prompts WHERE prompt_text LIKE '%0% win evidence%'
UNION ALL SELECT 'fvg_none_fresh', COUNT(*) FROM planner_rejected_prompts WHERE prompt_text LIKE '%(none fresh right now)%'
UNION ALL SELECT 'attempt1_total', COUNT(*) FROM planner_rejected_prompts WHERE attempt=1
UNION ALL SELECT 'attempt1_with_corrections', COUNT(*) FROM planner_rejected_prompts WHERE attempt=1 AND prompt_text LIKE '%CORRECTIONS FROM%'
UNION ALL SELECT 'both_lists(valid7+schema9)', COUNT(*) FROM planner_rejected_prompts WHERE prompt_text LIKE '%Valid conditions: [acceptance, breakdown_continue, breakup_continue, hold, reclaim, reject, sweep_reclaim]%' AND prompt_text LIKE '%reclaim|hold|sweep_reclaim|reject|acceptance|breakout_retest|fvg_entry|breakdown_continue|breakup_continue%'
UNION ALL SELECT 'minsl10_literal', COUNT(*) FROM planner_rejected_prompts WHERE prompt_text LIKE '%min-SL ≥ 1.0×ATR5m%'
UNION ALL SELECT 'floor15_section', COUNT(*) FROM planner_rejected_prompts WHERE prompt_text LIKE '%(1.5×ATR5m%resolved)%'
UNION ALL SELECT 'errors_today_x2', COUNT(*) FROM planner_rejected_prompts WHERE (length(prompt_text)-length(replace(prompt_text,'errors today:','')))/length('errors today:')>=2;
SELECT '---' ;
SELECT id, trade_date, session, attempt, substr(reject_reason,1,230) r FROM planner_rejected_prompts WHERE id IN (116,123,120,122,125,128,131) ORDER BY id;
