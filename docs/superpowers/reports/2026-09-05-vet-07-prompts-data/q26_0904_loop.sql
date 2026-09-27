.mode list
.headers on
SELECT session, COUNT(*) full_n, SUM(prompt_text LIKE '%NY AM 07:30–10:00 CT%') kz FROM planner_rejected_prompts WHERE length(prompt_text)>=20000 GROUP BY session;
SELECT '---';
SELECT id, datetime(strftime('%s',created_at),'unixepoch','-5 hours') ct, session, attempt, length(prompt_text) len, substr(reject_reason,1,150) r FROM planner_rejected_prompts WHERE trade_date='2026-09-04' ORDER BY id;
