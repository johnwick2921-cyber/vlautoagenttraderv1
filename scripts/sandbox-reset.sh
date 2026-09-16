#!/usr/bin/env bash
# Wipe the sandbox database and re-seed from scratch, then restart the sandbox.
# Only ever touches data/sandbox.db — the live data/data.db is never opened.
set -uo pipefail
cd "$(dirname "$0")/.." || exit 1
"$PWD/scripts/sandbox-down.sh" >/dev/null 2>&1
echo "▸ deleting data/sandbox.db…"
rm -f data/sandbox.db data/sandbox.db-wal data/sandbox.db-shm
"$PWD/scripts/sandbox-up.sh"
