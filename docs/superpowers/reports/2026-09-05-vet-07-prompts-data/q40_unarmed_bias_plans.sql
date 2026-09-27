.mode list
.headers on
WITH p AS (
 SELECT plan_id, version, lifecycle, datetime(strftime('%s',created_at),'unixepoch','-5 hours') ct, json_extract(doc,'$.bias.direction') bias,
  (SELECT COUNT(*) FROM json_each(json_extract(doc,'$.scenarios')) s WHERE json_extract(s.value,'$.arm.enabled')=1) arms_total,
  (SELECT COUNT(*) FROM json_each(json_extract(doc,'$.scenarios')) s WHERE json_extract(s.value,'$.arm.enabled')=1 AND lower(json_extract(s.value,'$.direction'))=lower(json_extract(doc,'$.bias.direction'))) arms_in_bias,
  (SELECT COUNT(*) FROM json_each(json_extract(doc,'$.scenarios')) s) n_scen
 FROM plans WHERE created_at >= '2026-09-03 16:10:33' AND lifecycle IN ('active','dormant') AND json_valid(doc) AND session IN ('ASIA','LONDON','NY'))
SELECT COUNT(*) n_plans, SUM(bias IN ('long','short')) directional, SUM(bias IN ('long','short') AND arms_in_bias=0) directional_no_arm_in_bias, SUM(arms_total=0) zero_arms FROM p;
SELECT '--- the directional plans with no arm in bias';
SELECT substr(plan_id,1,17) pid, version, lifecycle, ct, bias, n_scen, arms_total, arms_in_bias FROM p WHERE bias IN ('long','short') AND arms_in_bias=0 ORDER BY ct;
