# Adversarial verification — arm 37 "cancelled in NT8" reason laundering
Verdict: PLAUSIBLE (core mechanism CONFIRMED; one clause of the reasoning is false)
All times CT (= stored UTC - 5h). Source rev: worktree /home/hoang/nofx-2day04 @ dfbfa660.

## Commands run (all read-only)

sqlite3 -header -column "file:/home/hoang/nofx/data/data.db?mode=ro" \
 "SELECT id,plan_id,version,session,scenario,side,entry_px,stop_px,target_px,state,state_reason,entry_class,signal_id,created_at,updated_at,leg_index,leg_count,kind,boot_id,armed_under_version FROM armed_orders WHERE id>=32 ORDER BY id;"

sqlite3 -header -column "file:/home/hoang/nofx/data/data.db?mode=ro" \
 "SELECT id, state, state_reason, created_at AS created_raw, updated_at AS updated_raw, datetime(strftime('%s',replace(substr(updated_at,1,19),' ','T')||'Z'),'unixepoch','-5 hours') AS updated_CT FROM armed_orders WHERE id IN (32,35,36,37);"

sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro" "SELECT id,'['||signal_id||']',state_reason FROM armed_orders WHERE id IN (32,33,34,37);"

sed -n '5240,5290p' /home/hoang/nofx/data/nofx_2026-09-03.log
grep -n "cancelled in NT8" /home/hoang/nofx/data/nofx_2026-09-03.log      # -> exactly 1 hit
grep -c "cancelled in NT8" /home/hoang/nofx/data/nofx_2026-09-02.log      # -> 0
grep -nE "🔒 armed cancel|📌|armed S3|43512e1d" /home/hoang/nofx/data/nofx_2026-09-03.log
grep -n "armed order_update summary" /home/hoang/nofx/data/nofx_2026-09-03.log
grep -n "🔒 armed cancel" /home/hoang/nofx/data/nofx_2026-09-02.log
grep -n "" /home/hoang/nofx-2day04/trader/armed_executor.go | sed -n '1145,1155p;1470,1495p'
sed -n '1447,1456p' /home/hoang/nofx-2day04/trader/armed_executor.go
sed -n '188,240p'   /home/hoang/nofx-2day04/trader/armed_executor.go
sed -n '2585,2615p' /home/hoang/nofx-2day04/trader/auto_trader_planner.go
grep -n "armedCancelAckTimeout" -A12 /home/hoang/nofx-2day04/trader/armed_executor.go
grep -n "ARMED_CANCEL_ACK_TIMEOUT_MS" /home/hoang/nofx/.env   # -> no hit (default 2000ms is the LIVE value)

## Reproduced exactly [A]

Log /home/hoang/nofx/data/nofx_2026-09-03.log, lines 5250 / 5262 / 5263, verbatim:
  09-03 12:15:00 [INFO] ... 😴 plan 2026-09-03 NY v7 DORMANT — death-condition: 2x5m close below 29502.25 ...
  09-03 12:15:01 [INFO] trader/armed_executor.go:1082 📡 armed order_update summary (1-line/min): frames=4 accepted=1 working=1 cancelpending=1 submitted=1
  09-03 12:15:01 [INFO] ... ✕ armed S3 cancelled in NT8
  09-03 12:15:01 [WARN] trader/auto_trader.go:48 ... 🔒 armed cancel: no active plan — 1 order(s) disarmed
Only ONE "cancelled in NT8" line exists on 09-03 (grep -c = 1) and ZERO on 09-02.
Arm 37 placed 09-03 11:58:33 (log 4939): "📌 armed S3 → WORKING limit 29543.75 signal=43512e1d-...".
DB row 37: state=cancelled, state_reason='cancelled in NT8', signal_id=43512e1d-..., updated_at 17:15:01.127+00:00 = 12:15:01 CT.

Line numbers cited by the peer are exact:
  1150-1151  case "cancelled": ledger.SetState(r.ID,"cancelled","cancelled in NT8")
  1479       at.onArmedOrderUpdate(u, ledger)      (inside the ack-drain loop)
  1480       acked = !at.armedRowStillActive(...)
  1488       SetState(..., reason+" (ack timeout — flatten proceeds)")

Chain proven: dormant plan -> installActivePlanProvider's ActivePlan returns nil
(auto_trader_planner.go:2612 `row.Lifecycle != "active"` -> nil) -> maybeManageArmedOrders
(armed_executor.go:210-216) sets reason="no active plan" -> cancelArmedOrdersSync ->
row 37 IS "working" WITH a signal id, so it takes the wire path -> onArmedOrderUpdate at
1479 stamps 'cancelled in NT8' -> armedRowStillActive false -> acked -> n=1 -> the 🔒 line.
The caller's reason never reaches the ledger for THIS row. [A]

## Correction 1 — the "only ever written on ack timeout" clause is FALSE [A]

armed_executor.go:1451-1454:
  if r.State != "working" || r.SignalID == "" || cancelFn == nil || src == nil {
      _ = ledger.SetState(r.ID, "cancelled", reason)   // line 1452 — PLAIN caller reason
      n++; continue }
Live counter-example: arm 32, signal_id EMPTY, state_reason 'session ended (EOD flat)',
updated 09-02 14:45:01 CT, matching log line 41110 of nofx_2026-09-02.log
("🔒 armed cancel: session ended (EOD flat) — 1 order(s) disarmed"). No ack timeout involved.
Correct statement: the caller's reason survives ONLY for rows that never reached WORKING or
have no signal id (line 1452), or whose ack times out (line 1488). For a WORKING row whose
cancel is acked — arm 37's case — it is pre-empted by 'cancelled in NT8'.

## Correction 2 — "the cancel was ours" is [B], not [A]

The ack loop treats ANY 'cancelled' frame for the signal id as its ack; onArmedOrderUpdate has
no notion of who initiated. A cancel frame NT8 raised on its own (and left buffered in the
shared subscription, which is drained only inside runArmedPlacement at armed_executor.go:941 —
below the dormancy early return at line 234) would be indistinguishable. So the code CANNOT
distinguish "NT8 acked our cancel" from "NT8 cancelled it for its own reason" — which is exactly
why the reason string is unsafe. Supporting evidence that it was ours: the order rested WORKING
from 11:58:33 with no ✕ line until 12:15:01; ARMED_CANCEL_ACK_TIMEOUT_MS is unset in
/home/hoang/nofx/.env so the LIVE ack window is the 2000ms default (≤2 attempts), meaning the
frame was consumed within ≤4s of our send; and the 12:15:01 summary batch carries cancelpending=1.

## Correction 3 — timezone of the handed premise

The audit brief states arm 37 was "cancelled in NT8 at 17:15". 17:15:01 is the UTC wall time as
stored (+00:00). In CT it is 12:15:01. Note armed_orders stores created_at with a -05:00 offset
and updated_at with a +00:00 offset — mixed offsets in one table.
