#!/usr/bin/env bash
#
# C1 installer — installs + enables the nofx auto-backup user timer. NO SUDO: uses
# `systemctl --user`, and the hoang user already has linger enabled, so the timer
# fires on schedule even with no interactive login. Idempotent; safe to re-run.
set -euo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
UNIT_DIR="$HOME/.config/systemd/user"

mkdir -p "$UNIT_DIR"
install -m 0644 "$REPO/deploy/systemd-user/nofx-backup.service" "$UNIT_DIR/nofx-backup.service"
install -m 0644 "$REPO/deploy/systemd-user/nofx-backup.timer" "$UNIT_DIR/nofx-backup.timer"
chmod +x "$REPO/deploy/nofx-db-backup.sh"

systemctl --user daemon-reload
systemctl --user enable --now nofx-backup.timer

echo "== installed nofx-backup.timer =="
systemctl --user list-timers nofx-backup.timer --all --no-pager || true
echo
echo "Run a backup immediately with:  systemctl --user start nofx-backup.service"
echo "Watch it with:                  journalctl --user -u nofx-backup.service -f"
