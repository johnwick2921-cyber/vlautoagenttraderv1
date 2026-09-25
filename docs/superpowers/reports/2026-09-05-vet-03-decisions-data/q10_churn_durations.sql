.mode list
.separator " | "
.headers on
SELECT '## q10a: plan versions per session-day since 09-01 + trigger mix' AS s;
SELECT trade_date, session, COUNT(*) versions, SUM(trigger_reason='level_event') level_event, SUM(trigger_reason LIKE '%scheduled_read') scheduled, SUM(trigger_reason='planner_fail_closed') fail_closed, SUM(trigger_reason='owner_reset') owner_reset, SUM(trigger_reason LIKE 'dormant%') dormant, SUM(trigger_reason LIKE 'rearmed%') rearmed, SUM(trigger_reason='replans_exhausted') exhausted FROM plans WHERE trade_date>='2026-09-01' AND session IN ('ASIA','LONDON','NY') GROUP BY 1,2 ORDER BY 1,2;
SELECT '## q10b: inter-version minutes since 09-01 (all sessions)' AS s;
WITH v AS (SELECT trade_date, session, version, strftime('%s', substr(created_at,1,19)) t FROM plans WHERE trade_date>='2026-09-01' AND session IN ('ASIA','LONDON','NY')),
d AS (SELECT a.trade_date, a.session, a.version, (b.t - a.t)/60.0 mins FROM v a JOIN v b ON a.trade_date=b.trade_date AND a.session=b.session AND b.version=a.version+1)
SELECT COUNT(*) n, ROUND(AVG(mins),1) mean_min, ROUND(MIN(mins),1) min_min, ROUND(MAX(mins),1) max_min, (SELECT ROUND(mins,1) FROM d ORDER BY mins LIMIT 1 OFFSET (SELECT COUNT(*)/2 FROM d)) median_min FROM d;
SELECT '## q10c: trigger_reason mix since 09-01' AS s;
SELECT CASE WHEN trigger_reason LIKE 'dormant%' THEN 'dormant:*' WHEN trigger_reason LIKE 'rearmed%' THEN 'rearmed:*' ELSE trigger_reason END tr, COUNT(*) n FROM plans WHERE trade_date>='2026-09-01' AND session IN ('ASIA','LONDON','NY') GROUP BY 1 ORDER BY n DESC;
SELECT '## q10d: executor ai_request_duration_ms since 09-01 (non-null)' AS s;
WITH x AS (SELECT ai_request_duration_ms d FROM decision_records WHERE date(timestamp,'-5 hours')>='2026-09-01' AND ai_request_duration_ms IS NOT NULL AND ai_request_duration_ms>0 ORDER BY 1)
SELECT COUNT(*) n, (SELECT d FROM x LIMIT 1 OFFSET (SELECT COUNT(*)/2 FROM x)) p50, (SELECT d FROM x LIMIT 1 OFFSET (SELECT COUNT(*)*9/10 FROM x)) p90, MAX(d) max FROM x;
SELECT '## q10e: cited_scenario_id distribution of executor cycles since 09-01' AS s;
SELECT COALESCE(cited_scenario_id,'(null)') c, COUNT(*) n FROM decision_records WHERE date(timestamp,'-5 hours')>='2026-09-01' GROUP BY 1 ORDER BY n DESC LIMIT 8;
