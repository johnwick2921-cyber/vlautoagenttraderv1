# Adversarial REFUTE attempt — "class 47 cadence does not reach the structure_mss wake"

Verdict: **NOT REFUTED — CONFIRMED** (independently reproduced at the deployed rev). Two
corrections to the peer's phrasing, one downgrade of a caveat, one addition.

## Rev control (the peer's grounding was written at boot 7; I re-ran at boot 8)
- `git diff 70af663d -- trader/auto_trader_transition.go` → EMPTY. [A]
- `git diff --name-only 70af663d 492d2067` → only `deploy/RELEASE`, `web/src/guide/types.ts`. [A]
  The MSS wake path is byte-identical at dev tip, deployed rev 70af663d and the audit worktree.

## RESOLVED, from the running process (A11 — not a file default)
`/home/hoang/nofx/data/nofx_2026-09-04.log`, boot 08:30:11 CT, `🔐 BOOT INTEGRITY OK — rev
70af663dcb6f · built 2026-09-04T13:16:34Z · expected 70af663d · goldens PASS`:

    09-04 08:30:11 [INFO] nofx/main.go:340 ⏱ wakes: cutoff=25m(enforce) cooldown=30m(enforce,
    fast-market≥1.5×ATR exempt) cross-session=on stale-arm-expiry=on (class 47) — cutoffs govern
    LEVEL_EVENT/structure_mss wakes ONLY; …

So class 47 is ENFORCING (25m / 30m) and the binary ASSERTS structure_mss is governed. [A]

## The four greps I re-ran myself
1. `grep -rn "WakeCadenceDecision" --include=*.go .` → exactly ONE production construction:
   `trader/auto_trader_wake_levels.go:289`, inside `maybeWakePlannerOnLevelEventsAt`
   (auto_trader_wake_levels.go:251). Every other hit is `trader/class47_wake_cadence.go`
   (definitions) or `class47_wake_cadence_test.go`. [A]
2. `grep -n "cutoff|cooldown|minutesToSessionFlat|anyPlannerStreamOpen|WakeCadence|SkipFor|IncWakeCounter"
   trader/auto_trader_transition.go` → **zero output**. [A]
3. Same token set against `trader/auto_trader_planner.go` (the shared read path,
   `runPlannerReadWithTriggerClaimedCtx` L860) → **zero output**. No cadence downstream either. [A]
4. `grep -rn "IncWakeCounter" --include=*.go .` → 3 production sites, ALL in
   auto_trader_wake_levels.go (319 / 339 / 355). Zero on the MSS path. [A]

## Method-level caller search (the "naive func( grep" trap the dispatch warns about)
`grep -rn "maybeWakePlannerOnMSS" --include=*.go .` on the bare identifier (catches method values
and any interface satisfaction) returns 5 lines: the 2 definitions (transition.go:152, :156), one
comment (:146), the internal delegate (:153) and exactly ONE production call —
`trader/auto_trader_planner.go:341`. It is an unexported method on `*AutoTrader`; no interface in
the repo lists it. The peer's "1 production caller" survives a method-level search. [A]

## What actually limits the MSS wake (transition.go:156-211, verified by line-numbered awk)
- dedupe key `planID:version:mssTimeMs` — L175-178
- shared `wake_min_interval_min` throttle — L183-189
- L206 `runPlannerReadWithTriggerClaimedCtx(session, tradeDate, "structure_mss", …, failClosed=false)`

Missing versus the level_event path: 25m-to-flat cutoff · 30m wake-authored cooldown (+ its
fast-market bypass) · `anyPlannerStreamOpen()` cross-session defer · all three wake counters.

## Reachability — the divergence is live, not theoretical
`inSessionReadWindow(now, s.ReadCT, s.WindowEndCT)` (auto_trader_planner.go:206 →
auto_trader_clock.go:283) ends at the session's WindowEndCT, which the boot line resolves to the
FLAT (NY 14:45). So an MSS wake at 14:44 is reachable and fires; the identical level wake is
refused at 14:21+ by the enforcing cutoff. [A]

## Corrections / additions to the peer

**(1) Correction — "no cross-session defer" is right about class 47 but is not a total absence of
mutual exclusion.** The MSS read does take `claimPlannerRead(key)` at auto_trader_planner.go:865,
which serializes reads for the SAME (trader, trade_date, session). What is absent is class 47's
CROSS-session `anyPlannerStreamOpen()` defer and its `WakeStreamDeferKind` counter. [A]

**(2) Downgrade of the grounding file's caveat.** verify-class47-mss-wake-ungoverned.md:21-22 notes
the throttle is additionally guarded by `StrategyConfig != nil && DayPlan != nil`, implying a live
total loss of throttling. Measured: `SELECT json_extract(config,'$.day_plan') … FROM strategies` →
`dayplan_present` on all 9 rows. That caveat is THEORETICAL, not live. [A]

**(3) `wake_min_interval_min` live value.** `json_extract(config,'$.day_plan.wake_min_interval_min')`
is NULL on all 9 strategy rows → the resolution seam `DayPlanConfig.WakeMinIntervalMinutes()`
(store/strategy.go:1486-1491) returns `DefaultWakeMinIntervalMin = 30` (store/strategy.go:1447).
Resolver-derived, not a file default read off disk. [A]

**(4) The class-53 pattern holds.** `TestWakeCadenceEnforcementGovernsWakesOnly`
(trader/class47_wake_cadence_test.go:306-317) asserts `WakeCadenceGoverns("structure_mss")==true`.
It pins the PREDICATE, never the production call site, so the suite stays green while the MSS wake
never calls it. CLAUDE.md canon: "parity tests exercise production CALL SITES". [A]

**(5) The path is RARE but NOT dead (A29 does not apply).** All-time, mode=ro:
- `SELECT trigger_reason, COUNT(*) FROM plans GROUP BY 1` → `structure_mss` = **2**,
  `level_event` = **107**. The 2 MSS rows: `2026-08-24:NY…` v2 and `2026-08-26:ASIA…` v4. [A]
- `SELECT … FROM log_events WHERE message LIKE '%structure MSS%'` → **5** rows, ids
  4819 / 8971 / 8975 / 18234 / 18239 (08-24 08:45:01 NY fire · 08-26 19:45:01 ASIA skip +
  19:46:46 fire · 09-01 03:15:29 LONDON skip + 03:17:29 fire, CT). [A]
- Zero on 09-02, 09-03, 09-04 (`grep -c "structure MSS"` on each daily log = 0), against a control
  of 1,737 all-time `level wake` log_events rows and 17 today. [A]
So the divergence is **latent**: n=0 MSS wakes have yet landed inside a window the cutoff or
cooldown would have refused. It is a real ungoverned path, not a realised loss. Say so with the n.

## Bonus finding (outside the peer's claim)
`WakeCadenceBootLine()` (trader/class47_wake_cadence.go:220-225) is commented "every value READ
from its resolver (A12/A24: no literals in a boot line)", but `cross-session=%s stale-arm-expiry=%s`
are filled with `onOffWord(true), onOffWord(true)` — hard-coded literals (L224). The boot line
cannot report those two as off even if the code stopped doing them. [A]

## Commands
    cd /home/hoang/nofx-conform
    git diff 70af663d -- trader/auto_trader_transition.go
    git diff --name-only 70af663d 492d2067
    grep -rn "WakeCadenceDecision" --include=*.go .
    grep -n "cutoff\|cooldown\|minutesToSessionFlat\|anyPlannerStreamOpen\|WakeCadence\|SkipFor\|IncWakeCounter" trader/auto_trader_transition.go
    grep -n "cutoff\|cooldown\|minutesToSessionFlat\|anyPlannerStreamOpen\|WakeCadence\|SkipFor" trader/auto_trader_planner.go
    grep -rn "maybeWakePlannerOnMSS" --include=*.go .
    grep -rn "IncWakeCounter" --include=*.go .
    grep -h "⏱ wakes:" /home/hoang/nofx/data/nofx_2026-09-04.log | tail -1
    sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro" "SELECT trigger_reason, COUNT(*) FROM plans GROUP BY 1 ORDER BY 2 DESC;"
    sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro" "SELECT id, datetime(ts_utc/1000,'unixepoch','-5 hours'), substr(message,1,120) FROM log_events WHERE message LIKE '%structure MSS%' ORDER BY id;"
    sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro" "SELECT id, json_extract(config,'\$.day_plan.wake_min_interval_min') FROM strategies;"

## Report provenance (git log -1)
    docs/superpowers/reports/2026-09-04-two-day-audit.md
      f3c640c3f9799e6fa80ce124ae87ee915cad63ed 2026-09-04 07:26:52 -0500
    docs/superpowers/reports/2026-09-02-belief-census.md
      ee64a494c60eed32bb5e71f4a2b0c43d8b0c5574 2026-09-02 08:50:38 -0500
