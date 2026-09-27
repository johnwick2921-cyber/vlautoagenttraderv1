# ADVERSARIAL VERIFY — "decision-path EntryGate refusal emits NO log line"

Verdict: **CONFIRMED** (one cited evidence command misquoted; finding unaffected).
Verifier ran everything independently. All times CT (UTC-5).

## 1. Count of decision-path EntryGate refusals — 19 [A]

NOTE: SQLite `LIKE` treats `_` as a single-char wildcard, so `LIKE 'entry_gate%'`
is loose. Re-run with GLOB (literal `_`) — same answer, 19:

    sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro" \
      "SELECT COUNT(*) FROM decision_records WHERE risk_check_error GLOB 'entry_gate:*';"
    -> 19          (all-time == in-window; nothing before 2026-09-02)

    sqlite3 ... "SELECT date(timestamp,'-5 hours') ct_day, COUNT(*) FROM decision_records
                 WHERE risk_check_error GLOB 'entry_gate:*' GROUP BY 1;"
    -> 2026-09-02 | 5
       2026-09-03 | 14

decision_records ids (CT, cycle, reason family):
36640 09-02 18:48:49 c26718 minSL | 36641 18:50:36 c26719 minSL | 36642 18:52:33 c26720 minSL
36645 09-02 19:00:23 c26723 R:R   | 36703 20:53:52 c26781 R:R
36864 09-03 02:12:56 c26942 R:R
37304 20:35:06 c27382 | 37305 20:37:51 | 37306 20:39:40 | 37307 20:42:27 | 37308 20:43:53 |
37310 20:49:04 | 37311 20:52:05 | 37313 20:54:28 | 37315 20:58:56 | 37318 21:03:22 |
37319 21:06:44 | 37320 21:09:03 | 37322 21:12:40   (13 x plan_mode=strict, ASIA S1)

Same 19 rows also carry it in `execution_log` (GLOB '*entry_gate:*' -> 19) and in the
child `decisions` JSON (GLOB -> 19). No hidden extras: the 30 rows a LIKE query returns
are `_`-wildcard artifacts.

## 2. File logs — 0 lines, and NOT a coverage gap [A]

    grep -F -c "entry-gate" /home/hoang/nofx/data/nofx_2026-09-0{2,3}.log   -> 0 / 0
    grep -F -c "entry_gate" ... -> 0 / 0
    grep -c "🚦" ...            -> 0 / 0
    grep -c "plan_mode=strict" -> 0 / 0 ; grep -c "below floor" -> 0 / 0

Coverage proof — every one of the 19 refusal SECONDS has log lines in the file
(14-45 lines each), so the zero is a real absence, not a dead logger:
    for t in "09-03 20:35:06" ...; do grep -c "^$t" nofx_2026-09-03.log; done
Files are contiguous: 09-02 log = 09-02 00:01:06 -> 09-03 10:28:08;
09-03 log = 09-03 10:28:29 -> 09-03 23:55:15.

The silent cycle, verbatim (decision 37304, cycle 27382):
    09-03 20:35:06 [INFO] auto_trader_loop.go:813 🔄 Execution order (optimized)...
    09-03 20:35:06 [INFO] auto_trader_loop.go:815   [1] MNQ open_short
    09-03 20:35:06 [INFO] auto_trader_loop.go:817
    09-03 20:35:06 [INFO] auto_trader_decision.go:62 📝 Decision record saved: ... cycle=27382
Nothing between the intent and the save.

## 3. log_events — peer's command output NOT reproducible; conclusion still 0 [A]

    sqlite3 ... "SELECT COUNT(*) FROM log_events WHERE message LIKE '%entry%gate%';"
    -> 18   (peer reported 0)
All 18 are 2026-08-30 22:55-23:04 CT "🚨 reconcile QTY DIVERGENCE ... later entry
(id=45->46 class) ... investigate." — `%entry%gate%` spans "entry ... investi-GATE-".
Outside the window, none is an entry-gate line.

Exact query:
    sqlite3 ... "SELECT COUNT(*) FROM log_events WHERE message GLOB '*entry_gate*';" -> 0
    sqlite3 ... "SELECT COUNT(*) FROM log_events WHERE message GLOB '*entry-gate*';" -> 0

Plumbing proof (kills "log_events was just broken"): a logger.Warnf line from the
SAME SECOND as refusal 36640 IS in log_events —
    sqlite3 ... "SELECT id,datetime(ts_utc/1000,'unixepoch','-5 hours'),level,substr(message,1,90)
                 FROM log_events WHERE ts_utc BETWEEN strftime('%s','2026-09-02 23:48:40')*1000
                 AND strftime('%s','2026-09-02 23:49:10')*1000;"
    -> 23492 | 2026-09-02 18:48:49 | warning | ⚠️ clock-drift DETECTED ...

## 4. Code — no logger call, in EVERY rev that actually ran [A]

`entryGateDecisionTelemetry` (trader/entry_gate.go:477-486 at dev tip dfbfa660,
file is 486 lines): IncGateBlock + Success=false + Error=reason. No logger call.
`recordEntryGateRefusal` (the "🚦 entry-gate REFUSED" logger) has exactly ONE
non-test call site: trader/armed_executor.go:511, path "arm".
    grep -rn "recordEntryGateRefusal" --include=*.go .   -> defn + armed_executor.go:511
    grep -rn "entryGateDecisionTelemetry" --include=*.go . -> defn + auto_trader_orders.go:334 + 2 tests

Refusal returns nil, so auto_trader_loop.go:883 takes the `else if actionRecord.Error != ""`
branch (l.895-906): appends "⛔ ... refused: ..." to record.ExecutionLog (a DB column)
and sets record.RiskCheckError. Still no logger call.
telemetry/gate_blocks.go:38 IncGateBlock — map increment only, no logging.

Revs live during the refusals (from BOOT INTEGRITY OK lines) and their
entryGateDecisionTelemetry body — all logger-free:
  09-02 18:48-19:00 -> boot 18:27:17 rev f34925c0   (body: Error = "entry_gate: "+reason)
  09-02 20:53       -> boot 20:42:28 rev 575e9c05   (same)
  09-03 02:12       -> rev 33de2bef lineage         (same)
  09-03 20:35-21:12 -> boot 19:06:06 rev 042ff360   (de-doubled prefix; still no log)
  dev tip / boot 7  -> 530009ff                     (same)
The doubled "entry_gate: entry_gate:" on the five 09-02 rows vs single on 09-03
independently corroborates which rev wrote which row.

## 5. Scope correction to the wording

"NO log line at all" is true of the LOGGER (file log, journald, log_events).
The refusal is NOT unrecorded: 19 decision_records rows carry it in
risk_check_error + execution_log, and it bumps the in-memory gate-block counter
(GET /api/risk/gate-blocks). Defect = invisible in the log stream, not unrecorded.

## 6. Asymmetry is real

The arm seam logs its refusals — "⚔️ arm REFUSED NY S1 leg 1: R:R 0.95 below arm
min 2.00" (armed_executor.go:1316 text), 09-02 09:55:01 / 10:25:01 / 13:06:29 —
so the arm path is observable and the decision path is not. The "🚦 entry-gate
REFUSED" line specifically fired 0 times in the window on either path.
