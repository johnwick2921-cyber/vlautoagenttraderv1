#!/usr/bin/env bash
#
# C1 — nofx SQLite online backup + retention prune.
#
# Uses the SQLite *online backup API* (via python3 stdlib — no sqlite3 CLI needed,
# and consistent even while the bot is writing) to snapshot data/data.db, verifies
# the copy's integrity, gzips it, and prunes to a daily+weekly retention window.
#
# Layout under $NOFX_BACKUP_DIR (default ~/nofx-backups/auto):
#   daily/nofx-YYYY-MM-DD_HHMMSS.db.gz    every run   → keep newest KEEP_DAILY
#   weekly/nofx-YYYY-MM-DD_HHMMSS.db.gz   1 per ISO wk → keep newest KEEP_WEEKLY
#
# Driven by the user systemd timer nofx-backup.timer (05:00 + 17:30 CT). Safe to
# run by hand any time. Exits non-zero (and keeps nothing partial) on any failure.
set -euo pipefail

DB="${NOFX_DB:-/home/hoang/nofx/data/data.db}"
ROOT="${NOFX_BACKUP_DIR:-$HOME/nofx-backups/auto}"
KEEP_DAILY="${NOFX_KEEP_DAILY:-14}"
KEEP_WEEKLY="${NOFX_KEEP_WEEKLY:-8}"

DAILY_DIR="$ROOT/daily"
WEEKLY_DIR="$ROOT/weekly"
mkdir -p "$DAILY_DIR" "$WEEKLY_DIR"

if [[ ! -f "$DB" ]]; then
  echo "nofx-backup: DB not found at $DB" >&2
  exit 1
fi

ts="$(date +%Y-%m-%d_%H%M%S)"
tmp="$DAILY_DIR/.nofx-${ts}.db.partial"
final="$DAILY_DIR/nofx-${ts}.db.gz"

# Online backup + integrity check on the COPY. Write to a .partial file first so a
# crash never leaves a truncated backup that looks complete.
python3 - "$DB" "$tmp" <<'PY'
import sqlite3, sys
src, dst = sys.argv[1], sys.argv[2]
s = sqlite3.connect(src)
d = sqlite3.connect(dst)
try:
    with d:
        s.backup(d)            # atomic, consistent snapshot via the backup API
    ok = d.execute("PRAGMA quick_check").fetchone()[0]
finally:
    d.close(); s.close()
if ok != "ok":
    sys.exit("backup integrity check FAILED: %s" % ok)
PY

gzip -f "$tmp"                  # -> $tmp.gz
mv -f "$tmp.gz" "$final"
echo "nofx-backup: wrote $final ($(du -h "$final" | cut -f1))"

# Weekly promotion: keep one snapshot per ISO week (the first run of the week wins).
week_tag="$(date +%G-W%V)"     # e.g. 2026-W33
if ! ls "$WEEKLY_DIR"/nofx-*."${week_tag}".db.gz >/dev/null 2>&1; then
  weekly="$WEEKLY_DIR/nofx-${ts}.${week_tag}.db.gz"
  cp -f "$final" "$weekly"
  echo "nofx-backup: promoted weekly $weekly"
fi

# Prune: keep the newest N in each tier (by name — timestamps sort lexically).
prune() {
  local dir="$1" keep="$2" f n=0
  # shellcheck disable=SC2012
  for f in $(ls -1 "$dir"/nofx-*.db.gz 2>/dev/null | sort -r); do
    n=$((n + 1))
    if [[ $n -gt $keep ]]; then
      rm -f "$f"
      echo "nofx-backup: pruned $(basename "$f")"
    fi
  done
}
prune "$DAILY_DIR" "$KEEP_DAILY"
prune "$WEEKLY_DIR" "$KEEP_WEEKLY"

echo "nofx-backup: done ($(ls -1 "$DAILY_DIR"/nofx-*.db.gz 2>/dev/null | wc -l) daily, $(ls -1 "$WEEKLY_DIR"/nofx-*.db.gz 2>/dev/null | wc -l) weekly)"
