#!/usr/bin/env bash
# F5c v2 (2026-08-30) — WSL2 clock-drift remediation, PERSISTENT.
#
# v1 regression root cause (diagnosed 2026-08-30): chrony was installed but
# HANDCUFFED — chronyd-starter.sh detected WSL2 as a container (and no
# CAP_SYS_TIME) and appended `-x`, so `makestep 1 -1` never stepped
# ("Disabled control of system clock", "Could not step system clock"). The v1
# belt-and-suspenders cron used `hwclock`, which does not EXIST in this WSL2
# rootfs — every 10-min correction silently failed. Result: 0.12s at 09:xx →
# -41s by 17:01 (NTPSynchronized=no) → the F6 clock-hold layer had to protect
# the machine.
#
# v2 changes:
#   1. /etc/default/chrony `SYNC_IN_CONTAINER=yes` — overrides the -x guard
#      (chronyd-starter.sh: EFFECTIVE_SYNC_IN_CONTAINER).
#   2. Verify after restart: `chronyc tracking` + the absence of the
#      "Disabled control of system clock" line.
#   3. The 10-min cron fallback now uses `chronyc makestep` (no hwclock).
#
# Run ONCE with sudo (owner-gated):
#   sudo bash deploy/fix-wsl2-clock.sh
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "run with sudo: sudo bash $0" >&2
  exit 1
fi

echo "[1/4] installing chrony..."
if command -v apt-get >/dev/null 2>&1; then
  apt-get update -qq && DEBIAN_FRONTEND=noninteractive apt-get install -y -qq chrony
else
  echo "no apt-get found — install chrony manually, then re-run" >&2
  exit 1
fi

echo "[2/4] configuring chrony for WSL2..."
cat > /etc/chrony/chrony.conf <<'EOF'
# F5c v2 — WSL2 clock hardening.
pool time.windows.com iburst maxsources 4
# Step the clock freely if far off; never trust a drifting WSL clock.
makestep 1 -1
# Local RTC is maintained by the Windows host — use it as a second opinion.
refclock SHM 0 offset 0.0 refid RTC
driftfile /var/lib/chrony/chrony.drift
EOF

# v2: neutralize the container-guard -x. Without this, chronyd tracks but can
# never step (the 2026-08-30 regression).
if [ ! -f /etc/default/chrony ]; then
  printf 'SYNC_IN_CONTAINER=yes\n' > /etc/default/chrony
else
  grep -q '^SYNC_IN_CONTAINER=' /etc/default/chrony \
    && sed -i 's/^SYNC_IN_CONTAINER=.*/SYNC_IN_CONTAINER=yes/' /etc/default/chrony \
    || printf 'SYNC_IN_CONTAINER=yes\n' >> /etc/default/chrony
fi

echo "[3/4] restarting chrony + verifying it can actually step..."
systemctl enable --now chrony || service chrony restart || true
sleep 3
if journalctl -u chrony --since "-2 min" --no-pager 2>/dev/null | grep -q "Disabled control of system clock"; then
  echo "FAIL: chrony still runs with -x (clock control disabled) — inspect /etc/default/chrony and chronyd-starter.sh" >&2
  exit 1
fi
chronyc makestep || echo "makestep refused (expected if already in sync)"
chronyc tracking

echo "[4/4] 10-min cron fallback: chronyc makestep (hwclock absent in WSL2)..."
CRON_LINE="*/10 * * * * root /usr/bin/chronyc makestep >/dev/null 2>&1"
if [ -f /etc/cron.d/fix-wsl2-clock ]; then
  rm -f /etc/cron.d/fix-wsl2-clock
fi
printf '%s\n' "$CRON_LINE" > /etc/cron.d/fix-wsl2-clock
chmod 644 /etc/cron.d/fix-wsl2-clock

echo
echo "done — chrony steps the WSL clock again + makestep cron every 10 min."
echo "verify with: chronyc tracking   (System time ~0s, Reference ID not PHC0-only)"
