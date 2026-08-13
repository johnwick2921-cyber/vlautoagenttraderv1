#!/usr/bin/env bash
#
# C3 installer — makes the systemd journal persistent and caps it at 2G.
#
# REQUIRES ROOT (writes /etc, restarts systemd-journald):  sudo bash deploy/install-journald.sh
# This is the ONE owner-run sudo step for C3; everything else in the hardening run
# is no-sudo. Idempotent.
set -euo pipefail

if [[ "${EUID}" -ne 0 ]]; then
  echo "install-journald: must run as root — try:  sudo bash $0" >&2
  exit 1
fi

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DROPIN_DIR=/etc/systemd/journald.conf.d

echo "== before =="
journalctl --disk-usage || true

mkdir -p "$DROPIN_DIR"
install -m 0644 "$REPO/deploy/journald-nofx.conf" "$DROPIN_DIR/nofx.conf"
# /var/log/journal already exists here, but create it to be safe on a fresh box.
mkdir -p /var/log/journal
systemd-tmpfiles --create --prefix /var/log/journal 2>/dev/null || true

systemctl restart systemd-journald
# Enforce the new cap immediately instead of waiting for the next rotation.
journalctl --vacuum-size=2G || true

echo "== after =="
journalctl --disk-usage
echo "effective config:"
systemd-analyze cat-config systemd/journald.conf 2>/dev/null | grep -iE "storage|systemmaxuse" || true
