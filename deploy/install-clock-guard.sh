#!/usr/bin/env bash
# install-clock-guard.sh — installs the P1 clock-guard user timer. NO SUDO
# (mirrors install-db-backup.sh; linger is already enabled for the deploying user so
# user timers fire without a login session).
#
#   bash deploy/install-clock-guard.sh
#
# Verify:  systemctl --user list-timers | grep clock-guard
# Logs:    journalctl --user -u nofx-clock-guard.service -f
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UNIT_DIR="$HOME/.config/systemd/user"

mkdir -p "$UNIT_DIR"
install -m 0644 "$REPO/deploy/systemd-user/nofx-clock-guard.service" "$UNIT_DIR/"
install -m 0644 "$REPO/deploy/systemd-user/nofx-clock-guard.timer" "$UNIT_DIR/"
chmod +x "$REPO/deploy/nofx-clock-guard.sh"

systemctl --user daemon-reload
systemctl --user enable --now nofx-clock-guard.timer
# Prime one run immediately so the state file + journal line exist right away.
systemctl --user start nofx-clock-guard.service

echo "clock-guard installed. Next runs:"
systemctl --user list-timers --no-pager | grep -E "NEXT|clock-guard" || true
