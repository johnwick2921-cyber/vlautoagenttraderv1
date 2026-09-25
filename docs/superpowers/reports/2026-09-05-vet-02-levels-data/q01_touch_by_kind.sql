-- q01: touch_outcomes by level_kind (all rows) — hold/break/ambiguous + first/last
SELECT level_kind, COUNT(*) n,
  SUM(outcome='hold') hold, SUM(outcome='break') brk, SUM(outcome='ambiguous_horizon') amb,
  ROUND(1.0*SUM(outcome='hold')/NULLIF(SUM(outcome IN ('hold','break')),0),4) p_hold_resolved,
  ROUND(1.0*SUM(outcome='ambiguous_horizon')/COUNT(*),3) amb_share,
  MIN(created_at) first_at, MAX(created_at) last_at
FROM touch_outcomes GROUP BY level_kind ORDER BY n DESC;
