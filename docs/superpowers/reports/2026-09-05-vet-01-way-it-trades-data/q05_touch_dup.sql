.mode column
.headers on
SELECT COUNT(*) rows_, COUNT(DISTINCT level_kind||'|'||level_price||'|'||opened_at_ms) distinct_touch, COUNT(DISTINCT plan_id||'|'||plan_version) plan_versions, COUNT(DISTINCT plan_id) plans, COUNT(DISTINCT session) sessions, COUNT(DISTINCT trader_id) traders FROM touch_outcomes;
SELECT level_kind, level_price, COUNT(*) n, COUNT(DISTINCT opened_at_ms) d_open, COUNT(DISTINCT plan_version) d_ver, COUNT(DISTINCT outcome) d_out, MIN(datetime(opened_at_ms/1000,'unixepoch','-5 hours')) first_open, MAX(datetime(opened_at_ms/1000,'unixepoch','-5 hours')) last_open FROM touch_outcomes GROUP BY 1,2 HAVING n>5 ORDER BY n DESC LIMIT 20;
-- dedup by (kind, price, opened_at_ms): pick one outcome
SELECT level_kind, SUM(o='hold') hold, SUM(o='break') brk, SUM(o='ambiguous_horizon') amb, COUNT(*) n FROM (SELECT level_kind, level_price, opened_at_ms, MIN(outcome) o FROM touch_outcomes GROUP BY 1,2,3) GROUP BY 1 ORDER BY n DESC;
SELECT plan_id, plan_version, session, COUNT(*) FROM touch_outcomes GROUP BY 1,2,3 ORDER BY 1,2;
SELECT candidate_seated, COUNT(*) FROM touch_outcomes GROUP BY 1;
