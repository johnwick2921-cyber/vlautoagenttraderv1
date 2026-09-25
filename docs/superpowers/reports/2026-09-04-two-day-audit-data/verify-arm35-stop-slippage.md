# Adversarial verification — arm 35 stop / "zero slippage" claim
Verifier run 2026-09-04. Source rev: worktree /home/hoang/nofx-2day04 (8579df0a, dev tip line for boot 7 / 530009ff).
DB opened read-only: sqlite3 "file:/home/hoang/nofx/data/data.db?mode=ro"

## Verdict: PLAUSIBLE core, REFUTED on "zero slippage" and on the evidence chain.

### 1. Stop composition is per-cycle, not a single value [A]
$ { awk '/^09-03 /' nofx_2026-09-02.log; cat nofx_2026-09-03.log; } | grep "🛑 arm stop"
NY S1 leg 1 short composed 24x on 09-03 from 08:30:54 to 09:15:01, range 29307.93 .. 29354.91.
The 6 values after the plan moved the authored stop to 29340.00:
  09:02:54  29354.91  (1.5xATR5m 46.61)   <- the one the peer quotes
  09:05:35  29351.63  (1.5xATR5m 44.42)   <- 21s AFTER the NT8 fill; this is the DB value
  09:06:54  29352.65  (45.10)
  09:08:54  29354.44  (46.29)
  09:10:54  29352.40  (44.93)
  09:12:54  29354.86  (46.58)
  09:15:01  29350.67  (43.78)

### 2. armed_orders row 35 [A]
stop_px = 29351.6284728996, created_at 2026-09-03 09:02:54.662-05:00, updated_at 2026-09-03 15:28:29.204+00:00 (=10:28:29 CT).
The VALUE is uniquely the 09:05:35 composition (no other cycle's ATR yields it), written by
store/armed_orders.go:190-195 UpsertArm (armed/working branch updates stop_px; does NOT touch state or signal_id).

### 3. bracket-modify branch is DEAD CODE [A]
trader/armed_executor.go:515-520  row := &store.ArmedOrderDB{ ... StopPx: leg.Stop, ... State: "armed", ... }
trader/armed_executor.go:525      row.ID = existing[i].ID          // ONLY the ID is copied
trader/armed_executor.go:563      if row.State == "working" && churnNeedsModify(row.StopPx, row.TargetPx, leg.Stop, leg.Target, tick)
trader/armed_executor.go:565      _ = nt.ModifyBracket(row.SignalID, leg.Stop, leg.Target)   // the ONLY ModifyBracket call site
Two independent always-false conditions:
  (a) row.State is the literal "armed"; the existing row's state is never copied -> == "working" never true.
  (b) row.StopPx was already set to leg.Stop by the literal -> churnNeedsModify(x,y,x,y,tick) = false.
Reproduced: churnNeedsModify(leg.Stop,leg.Target,leg.Stop,leg.Target,0.25) = false
            churnNeedsModify(29354.9141,29144.50,29351.6284728996,29144.50,0.25) = true (13.14 ticks)
$ grep -c "bracket modify" nofx_2026-*.log   -> 0 in ALL 18 files, 2026-08-17 .. 2026-09-03.
So the count 0 is a dead-code zero, true every day for every arm; it carries no information.

### 4. Rounding is on the live path [A]
RoundToTick is called on the TCP path: tcp_trader.go:379-381 (market), 448-450 (PlaceLimitEntry), 511-513 (stop entry), 624.
PlaceLimitEntry doc: "the AddOn submits OrderType.Limit and defers SL/TP to SubmitBracketOnEntryFill".
RoundToTick(29354.9141,0.25)=29355.00 (any float printing as 29354.91 rounds there); RoundToTick(29351.6284728996,0.25)=29351.75.

### 5. "ZERO slippage" is REFUTED [A]
Exit: 09-03 09:20:45 close_sync.go:196 "NT position closed: MNQ SHORT qty=1.00 exit=29355.00 reason=sl pnl=-140.00".
reason comes from p.ExitReason (NT8-reported), not Go-inferred.
1m MNQ bar 2026-09-03 09:20:00 CT: o 29343.50 h 29360.00 l 29342.25 c 29348.00 v 8048.
trader_positions.mae for id 591 = 75.00 pts = 29285.00+75.00 = 29360.00 (independent confirmation).
The market traded 5.00 pts / 20 ticks THROUGH the 29355.00 stop and the SIM still printed exactly 29355.00.
This is NT8 SIM; under a fill-at-stop-price engine exit == working stop identically for ANY stop value,
so "exit == stop" is not a slippage measurement. Boot line in force: "BE=off - trail=off - size=1".
