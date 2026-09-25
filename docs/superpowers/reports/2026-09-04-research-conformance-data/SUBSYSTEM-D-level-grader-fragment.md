# SUBSYSTEM D — LEVEL GRADER (census D1–D10 + dispatch D5) — conformance fragment
Snapshot 2026-09-04 08:46 CT · deployed rev 70af663d (PID 878451, booted 08:30:11 CT) ·
worktree /home/hoang/nofx-conform @ fb50903f (base dev 492d2067) · READ-ONLY.

## Headline
1. kernel/levels_score.go is BYTE-IDENTICAL at the census commit ee64a494, at the deployed
   rev 70af663d, and in the worktree — md5 dfcda708e8cad3e2b03af3affe4df5d1. NO ladder moved.
   Last commit touching it: e86ae805 (2026-08-31 08:11:57 CT), BEFORE both the census
   (2026-09-02 08:50) and the level-kind replay (2026-09-02 19:03).
2. BOOT LINE LIES TWICE (A11): `seats=8` prints the const DefaultMaxLevels; resolved
   max_levels = 12. `proximity=cfg(...; retuned 0.3)` — "0.3" is a string literal;
   resolved proximity_filter_atr = 1.0 (band +/-510pt = 1.0 x proxy 510.00).
3. D9 [X] is a SIGN ERROR in the census: -15pt on a MISS rate is an improvement.
4. D1' is WRITE-ONLY: zero gates/weights read touch_outcomes or candidate_pool.

## Live rates (touch_outcomes, id<=424, n floor 30) — see D5b-*.csv
ALL: p(hold)=0.5062 (hold 162 / break 158, n=320; ambiguous 104 of 424 = 24.5%)
DEMAND 0.5176 (n=85) · VWAP 0.6143 (n=70) · RTH-L 0.3175 (n=63)
NY 0.5033 (n=153) · LONDON 0.5000 (n=120) · ordinal-1 0.4876 (n=242)
ASIA n=47 reported; every other kind/ordinal cell SUPPRESSED (n<30).
