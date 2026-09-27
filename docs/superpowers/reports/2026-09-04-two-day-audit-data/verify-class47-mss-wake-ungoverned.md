# Adversarial verify — class 47 cadence does not reach the MSS wake

Verdict: **CONFIRMED** (independently reproduced; two corrections/additions).

## Code (worktree /home/hoang/nofx-2day04, base dfbfa660 = deployed 530009ff/boot 7)

`grep -rn 'WakeCadence' --include=*.go . | grep -v _test.go`
-> main.go:340 (boot line), trader/class47_wake_cadence.go (defs),
   trader/auto_trader_wake_levels.go:287,289,311 (the ONLY WakeCadenceDecision call site),
   comments only in trader/auto_trader_feedwatch.go:130 and store/strategy.go:1188,1304.
No hit in trader/auto_trader_transition.go. [A]

`grep -n 'cutoff\|cooldown\|minutesToSessionFlat\|anyPlannerStreamOpen' trader/auto_trader_planner.go`
-> ZERO hits: the shared read path runPlannerReadWithTriggerClaimedCtx (L860) applies no cadence either. [A]

maybeWakePlannerOnMSSAt = trader/auto_trader_transition.go:156-211 (verified by line-numbered awk).
Its only frequency limits: (a) dedupe key planID:version:mssTimeMs L175-178,
(b) shared wake_min_interval_min throttle L184-189 (live value 30, resolver default —
`json_extract(config,'$.day_plan.wake_min_interval_min')` is NULL for every strategy row,
and the 09-01 live line prints "(30m)"). [A]
Note (b) is additionally guarded by `StrategyConfig != nil && DayPlan != nil` — a nil DayPlan
removes even that throttle.

Boot line: trader/class47_wake_cadence.go:222, text verified. Live, from the running binary's
own log (not the default): /home/hoang/nofx/data/nofx_2026-09-03.log line 28999
`09-03 23:12:55 [INFO] cleanclone/main.go:340 ⏱ wakes: cutoff=25m(enforce) cooldown=30m(enforce,
fast-market≥1.5×ATR exempt) ... cutoffs govern LEVEL_EVENT/structure_mss wakes ONLY`.
No WAKE_CUTOFF_MIN / WAKE_COOLDOWN_MIN / FAST_MARKET_ATR key in /home/hoang/nofx/.env. [A]

NY flat is 14:45 CT in the LIVE registry (system_config key `session_registry`), not just the
default: `NY 08:30 -> 14:45 read 08:00 flat 14:45 enabled True`. The peer's 14:40/14:45 example
is right. The enclosing loop gate is `inSessionReadWindow(now, ReadCT 08:00, WindowEndCT 14:45)`
(auto_trader_planner.go:206), so the MSS wake IS reachable at 14:40. [A]

## The zero

grep -c 'structure MSS' (catches BOTH the fire line and the SKIPPED line):
  /home/hoang/nofx/data/nofx_2026-09-02.log -> 0
  /home/hoang/nofx/data/nofx_2026-09-03.log -> 0
Control that the grep would have found something: same files hold 489 and 55 'level wake' lines. [A]
sqlite3 mode=ro `SELECT COUNT(*) FROM log_events WHERE message LIKE '%structure MSS%'` -> 5, all time,
none on 09-02/09-03. So the zero is real, not a plausible-but-unverified zero. [A]

## ADDITIONS the peer did not have

1. The path is NOT dead. MSS wakes have FIRED 3x all-time and SKIPPED 2x (log_events ids
   4819/8971/8975/18234/18239): 08-24 08:45:01 NY (fire), 08-26 19:45:01 ASIA (skip) +
   19:46:46 (fire), 09-01 03:15:29 LONDON (skip, "30m elapsed < wake_min_interval_min (30m)")
   + 03:17:29 (fire). `plans` carries trigger_reason='structure_mss' on 2 rows vs 102 for
   level_event. Rare, live, and last exercised the day before the audit window. [A]

2. Class-53 pattern. trader/class47_wake_cadence_test.go:306-317
   (TestWakeCadenceEnforcementGovernsWakesOnly) asserts WakeCadenceGoverns("structure_mss")==true.
   It pins the PREDICATE, never the production call site, so it stays green while the MSS wake
   never calls it. Exactly "parity tests exercise production CALL SITES" (CLAUDE.md canon). [A]

3. Nuance on "no stream-defer": the MSS read does hold claimPlannerRead(planID) inside
   runPlannerReadWithTriggerClaimedCtx (auto_trader_planner.go:867), which serializes reads for
   the SAME (trader, trade_date, session). What is missing is class 47's anyPlannerStreamOpen()
   CROSS-session defer, plus the WakeStreamDeferKind counter. The peer's phrasing is right about
   class 47; it is not a total absence of mutual exclusion. [A]

## Missing on the MSS path vs the level_event path
25m-to-flat cutoff · 30m wake-authored cooldown (+ fast-market bypass) · cross-session stream
defer · the three store.IncWakeCounter records (WouldSkipCutoff / WouldSkipCooldown / StreamDefer).
