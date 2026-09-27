-- q02: candidate_pool by level_kind: candidates, seated, seat rate, avg score, grade dist, cut reasons
SELECT level_kind, COUNT(*) n, SUM(seated) seated, ROUND(1.0*SUM(seated)/COUNT(*),3) seat_rate,
  ROUND(AVG(score),3) avg_score, ROUND(AVG(CASE WHEN seated=1 THEN score END),3) avg_score_seated,
  ROUND(AVG(CASE WHEN seated=0 THEN score END),3) avg_score_cut,
  SUM(grade='A') A, SUM(grade='B') B, SUM(grade='C') C, SUM(grade NOT IN ('A','B','C')) other,
  COUNT(DISTINCT plan_id||'/'||plan_version) plan_reads
FROM candidate_pool GROUP BY level_kind ORDER BY n DESC;
