# Adversarial verification — "decision-path EntryGate min-SL leg was fed the DAILY ATR"
Verdict: CONFIRMED (independently reproduced, not by inverting the peer's arithmetic).

## 1. The three refusals (store) [A]
sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro" -header -column "
SELECT id, datetime(created_at) AS utc,
       datetime(strftime('%s',substr(created_at,1,19)),'unixepoch','-5 hours') AS ct,
       plan_id, plan_version, cited_scenario_id, risk_check_error
FROM decision_records WHERE risk_check_error LIKE '%too close%' AND created_at >= '2026-09-01' ORDER BY id;"

36640  2026-09-02 23:48:49 UTC = 18:48:49 CT  ASIA v6 S2  entry_gate: entry_gate: stop 29231.00 too close (25.25 < 450.56 = 1.5xATR5m)
36641  2026-09-02 23:50:36 UTC = 18:50:36 CT  ASIA v6 S2  entry_gate: entry_gate: stop 29226.50 too close (21.00 < 450.56 = 1.5xATR5m)
36642  2026-09-02 23:52:33 UTC = 18:52:33 CT  ASIA v6 S2  entry_gate: entry_gate: stop 29230.00 too close (22.00 < 450.56 = 1.5xATR5m)

ALL-TIME count of '%too close%' in decision_records = 3 (n=3, complete population; the leg
has never fired on any other row). Double-prefix 'entry_gate: entry_gate:%' rows = 6,
spanning 09-02 18:48:49 CT .. 09-03 02:12:56 CT.

## 2. The arm seam, same setup, same window (file log) [A]
grep -n "arm stop ASIA S2" /home/hoang/nofx/data/nofx_2026-09-02.log
 18:45:17  atr_floor 29228.42 (1.5xATR5m 12.78)
 18:47:17  atr_floor 29230.43 (1.5xATR5m 14.12)
 18:51:17  atr_floor 29229.45 (1.5xATR5m 13.47)   <- interleaves record 36641
 18:53:17  atr_floor 29229.91 (1.5xATR5m 13.77)   <- interleaves record 36642
Same plan (ASIA S2), same side (short), same stop region (29228-29231).

## 3. Independent reproduction of 450.56 from raw bars (NOT 450.56/1.5) [A]
Both seams read the SAME series: market.FuturesBarsProvider(sym, AISVPBarInterval="1m",
AISVPBarCount=2000)  (kernel/svp.go:46-47; kernel/engine_analysis.go:297; trader/armed_executor.go:245).
Decision path fed kernel.PlanDATRFor <- SetPlanDATR(ctx.TraderID, dATR) (kernel/engine_analysis.go:469)
<- AssembleScoredLevels -> dATR = DailyRangeProxy(bars, now) (kernel/levels_assemble.go:56, 291-328):
mean high-low of each COMPLETED CME session-day (17:00 CT roll) in the window, developing day excluded.

Replayed over the last 2000 1m MNQ bars before 2026-09-02 18:45:17 CT:
  sess 2026-08-31  hi 29317.25  lo 29001.75  rng 315.50  completed
  sess 2026-09-01  hi 29212.50  lo 28927.25  rng 285.25  completed
  sess 2026-09-02  hi 29208.75  lo 29149.00  rng  59.75  DEVELOPING (excluded)
  dATR = (315.50+285.25)/2 = 300.3750     1.5 x dATR = 450.5625 -> "%.2f" -> 450.56   EXACT MATCH
  ATR5m(14) on the same window, 5m-aggregated (AcceptanceBars "2x5m" -> Wilder ATR14) = 13.42
    (inside the log's 12.78-14.12 band)
MIN_SL_ATR_MULT is NOT set in /home/hoang/nofx/.env; MinSLATRMultDefault = 1.5 (kernel/min_sl.go:34).

## 4. The buggy wiring was in the binary that wrote those rows [A]
Boot lines (data/nofx_2026-09-02.log): 18:27:17 rev f34925c0ae3a ... next boot 20:42:28 rev 575e9c05f2a3.
The three refusals (18:48-18:52 CT) fall inside f34925c0ae3a's run.
  git show f34925c0ae3a:trader/entry_gate.go | grep -n "ATR5m:"
    185:  ATR5m:  atr5m,                          <- Path:"arm"
    237:  ATR5m:  kernel.PlanDATRFor(at.id),      <- Path:"decision"
Two ATRs, one gate, in the exact running rev.

## 5. The fix [A]
git log --format='%H %ci %s' -- trader/entry_gate.go
  609067ec21505d27b66476d10ae89cd6cbda6e79  2026-09-03 08:23:22 -0500  fix(entry_gate): one gate, ONE ATR5m resolver
Diff: -ATR5m: kernel.PlanDATRFor(at.id)  +ATR5m: armSeamATR5m(d.Symbol); adds armSeamATR5mFromBars.
Ancestor of BOTH dfbfa660 (dev tip) and 530009ff (boot 7, PID 594377 since 09-03 23:12:54 CT) -> live.

## 6. Corrections / notes (do not change the verdict)
- 300.37 is a truncation; the value is 300.3750 exactly.
- The quantity is an average completed-session-day RANGE (DailyRangeProxy), not a Wilder daily
  ATR(14). The code names it dATR and the doc comment says "the daily-ATR used", so "DAILY ATR"
  is the repo's own loose name, not the peer's invention.
- "in the same minutes" is 1-4 minutes earlier for the two quoted lines; the 18:51/18:53 lines
  interleave the records and sit inside the quoted band, so the substance holds.
- NOT CLAIMED by the peer, found here: the same corrupted in.ATR5m also feeds the no-chase
  WARN leg (trader/entry_gate.go, EvaluateNoChase(NoChaseInputs{... ATR5m: in.ATR5m ...})) on the
  decision path, so no-chase measurements on that path over the same window are also off by ~22x. [B]
