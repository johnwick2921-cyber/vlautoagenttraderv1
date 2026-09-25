# F6 CLOCK-HOLD — chrony regression diagnosis + machine-side protection

2026-08-30 · branch `fix/clock-hold` · rides fix/move-seams · NO deploy (parked).

## 1. Diagnosis — why the chrony fix regressed within hours

Timeline: 0.12s drift at 09:xx → **−41s at 17:01** (`clock-health [session-roll:ASIA]
drift_ms=-41039 timesync{NTP=yes NTPSynchronized=no}`) → NTPSynchronized=yes again
by the 18:13 boot. Two mechanisms handcuffed the OS-side fix:

1. **chrony ran with `-x` (clock control disabled).** `chronyd-starter.sh`
   detects WSL2 as a container (`systemd-detect-virt --container`) AND finds no
   CAP_SYS_TIME, so it appends `-x`. The `makestep 1 -1` in
   `/etc/chrony/chrony.conf` was therefore inert from the start.
2. **The cron belt-and-suspenders called a binary that does not exist.**
   `fix-wsl2-clock.sh` v1 installed `*/10 * * * * /sbin/hwclock -s --utc` —
   `hwclock` is absent in this WSL2 rootfs (verified by nofx-clock-guard.sh),
   so every 10-min correction silently failed.

Net: nothing could correct the clock; drift accumulated between hypervisor
resyncs.

### chronyc BEFORE (handcuffed state — 21:24:43 CT capture)

```
$ chronyc tracking
Reference ID    : 50484330 (PHC0)
Stratum         : 1
System time     : 0.000060096 seconds fast of NTP time
Last offset     : +0.000039092 seconds
Frequency       : 124.878 ppm slow
Update interval : 8.0 seconds
```
Healthy-looking — but misleading: it tracks the hypervisor PHC0, while the
service journal proves it cannot STEP the system clock:
```
Aug 28 19:00:10 chronyd-starter.sh: Warning: Running in a container, likely impossible and unintended to sync system clock
Aug 28 19:00:10 chronyd-starter.sh: Adding -x as fallback disabling control of the system clock
Aug 28 19:00:10 chronyd[3754048]: Disabled control of system clock
Aug 28 19:00:13 chronyd[3754048]: Could not step system clock
```
And the step command is refused:
```
$ chronyc makestep
501 Not authorised
```

### chronyc AFTER (post-fix expectation — owner sudo runbook)

`sudo bash deploy/fix-wsl2-clock.sh` (v2) writes `/etc/default/chrony`
`SYNC_IN_CONTAINER=yes` (the documented override in
`/usr/share/doc/chrony/README.container`), restarts chrony, and the starter then
logs `Not falling back to disable control of the system clock`. The 10-min cron
now runs `chronyc makestep` (no hwclock). Expected `chronyc tracking` unchanged
in shape (PHC0, ~0 offset) — the difference is that `makestep 1 -1` now actually
fires when the guest clock drifts. The v2 script FAILS (exit 1) if the
"Disabled control of system clock" line still appears after restart — the fix
verifies itself.

## 2. Machine-side escalation (this PR — works WITHOUT root)

Because the OS path is owner-gated and previously regressed, the bot now
protects itself (kernel/clock_drift.go → `ClockHoldDecision`):

| Drift | Authoring | News windows |
|-------|-----------|--------------|
| no measurement | proceeds (fail-open, C2 contract) | unchanged |
| < 30s (warn) | proceeds | unchanged |
| 30–60s | proceeds | **widened by \|drift\|** |
| > 60s, drift NEGATIVE | **DEFERRED** | widened by \|drift\| |
| > 60s, drift POSITIVE | proceeds | widened by \|drift\| |

The negative-only deferral is deliberate: negative drift means the feed labeled
bars in the FUTURE — provably broken local clock (the −41s incident class).
Positive drift is ambiguous — a closed market's old bars look identical (the
16:55 Sunday ASIA read MUST fire with 10-min-old bars, P0B contract); C2 already
covers the positive-skew entry risk via feed-stamped signals.

Wire-up:
- `kernel/clock_health.go` — every clock-health read persists the measurement
  (`RecordClockDrift`); `ClockWarnMs()` exported.
- `kernel/calendar_blackout.go` — `WidenCTWindows` + `T1NoTradeLinesDrift`
  (windows shifted by ceil(|drift|/60s) minutes, wrap-aware, label-tagged).
- `trader/auto_trader_planner.go` — clock-hold gate in the claimed authoring
  path (scheduled, death, wake, and owner re-read all funnel through it):
  `🕰 clock-hold: planner authoring DEFERRED for <date> <session> (|drift| Xms >
  tolerance 60000ms) — no plan written, no budget consumed; exits and armed
  management unaffected (F6)`. T1 lines widened in the warn/critical bands.
- `trader/auto_trader_reread.go` — `ForceReread` refuses with a `clock-hold`
  reason when the clock is provably broken.
- `trader/auto_trader_calendar.go` — `currentT1Windows` widens the exec-side
  red-news blackout by the drift (warn line once per trade date).

Fixtures (all green): `kernel/clock_hold_f6_test.go` (decision table incl.
positive-vs-negative asymmetry + midnight-wrap widening + label marker),
`trader/clock_hold_test.go` (injected −61s drift → `authoring DEFERRED` line
carries trade-date/session/tolerance + NO plan row written; warn band does not
defer; positive drift never defers; no measurement fails open).

## 3. Verification

- `go build ./...` clean; `go test ./kernel/ ./trader/ -count=1` both `ok`
  (P0B 16:55-closed-market read still fires — positive drift never defers).
- `deploy/fix-wsl2-clock.sh` v2 self-verifies (fails on a still-handcuffed
  chrony).
- AUDIT-CHECKLIST class 20 appended ("OS-side fix that silently regresses").
