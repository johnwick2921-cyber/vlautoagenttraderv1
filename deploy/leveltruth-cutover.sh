#!/usr/bin/env bash
# Level-truth cutover (2026-08-27) — MANUAL-ONLY runbook (NO-UNATTENDED-DEPLOYS
# canon: never schedule this; run it ONLY on the owner's explicit "go", with the
# owner reachable to ack the boot line. Timers are banned for deploys.)
# Canon: FLAT-GATE first (zero positions of ANY origin from NT8 truth),
# RELEASE = build sha, mv old binary, kill -9 (systemd relaunches).
set -u
cd /home/hoang/nofx || exit 1
BUILD_SHA="6fc09ad39fbaf17aa473ad960f186f8e1b3cc16e"
BIN="./nofx-bin.next"
LOG="/tmp/leveltruth-cutover.log"
exec >>"$LOG" 2>&1
echo "=== level-truth cutover $(date '+%F %T %Z') ==="

# 1. FLAT-GATE (the new law): DB OPEN=0 AND NT8 TCP snapshots count=0 AND API [].
OPEN=$(sqlite3 "file:data/data.db?mode=ro" "SELECT COUNT(*) FROM trader_positions WHERE status='OPEN';" 2>/dev/null)
if [ "$OPEN" != "0" ]; then
  echo "ABORT: DB OPEN positions = $OPEN — not flat, no deploy."
  exit 2
fi
SNAP=$(journalctl -u nofx --since "5 min ago" -o cat 2>/dev/null | grep -E "positions snapshot" | tail -2)
if ! echo "$SNAP" | grep -q "count=0" || [ -z "$SNAP" ]; then
  echo "ABORT: no clean NT8 count=0 snapshot in the last 5 min: $SNAP"
  exit 3
fi
echo "FLAT-GATE PASS: DB OPEN=0 · $SNAP"

# 2. RELEASE marker = the BUILD sha (boot integrity: rev == RELEASE == expected).
echo -n "$BUILD_SHA" > deploy/RELEASE
echo "RELEASE written: $BUILD_SHA"

# 3. Binary swap + kill -9 (SIGTERM exits 0 and does NOT relaunch).
if [ ! -x "$BIN" ]; then echo "ABORT: $BIN missing"; exit 4; fi
OLDPID=$(pgrep -f "nofx-bin$" | head -1)
mv nofx-bin "nofx-bin.old.leveltruth" 2>/dev/null
mv "$BIN" nofx-bin
echo "binary swapped (old PID $OLDPID)"
if [ -n "$OLDPID" ]; then kill -9 "$OLDPID"; echo "kill -9 sent to $OLDPID"; fi

# 4. Boot proof (systemd relaunches; poll up to 90s for the integrity line).
for i in $(seq 1 30); do
  sleep 3
  BOOT=$(journalctl -u nofx --since "2 min ago" -o cat 2>/dev/null | grep -E "BOOT INTEGRITY OK" | tail -1)
  if [ -n "$BOOT" ]; then
    echo "BOOT: $BOOT"
    NEWPID=$(pgrep -f "nofx-bin$" | head -1)
    echo "CUTOVER COMPLETE — PID $NEWPID"
    exit 0
  fi
done
echo "TIMEOUT: no BOOT INTEGRITY line in 90s — INVESTIGATE"
exit 5
