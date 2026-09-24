#!/usr/bin/env bash
# W-ONE-BUTTON M4 — prove a rollback pair before advertising it.
#
# A manifest that says rollback_pairs:[{to_release:X, tested:true}] is a promise
# that the owner can go back to X. The only way to keep that promise is to have
# actually done it, in this order:
#
#   1. the OLD binary creates a FRESH database   (its own schema)
#   2. the NEW binary boots against that database (forward migration)
#   3. the OLD binary boots against the migrated database (the rollback itself)
#
# Step 3 is the one that matters and the one nobody runs: a forward migration
# that drops a column the old binary still SELECTs makes the rollback fail at
# the moment it is needed most.
#
# CI has no NT8 and no network, so a binary cannot trade here and is not
# expected to stay up. Success is judged on THE DATABASE, not on the process:
# the file exists, carries its core tables, and no migration error was logged.
#
# Usage: db-compat.sh <old-ref> <new-ref> [work-dir]
# Exit 0 = the pair is proven. Non-zero = NOT proven; the caller must advertise
# tested:false (or omit the pair) rather than guess.
set -uo pipefail
OLD_REF="${1:-}"; NEW_REF="${2:-}"; WORK="${3:-$(mktemp -d)}"
[ -n "$OLD_REF" ] && [ -n "$NEW_REF" ] || { echo "db-compat: usage: db-compat.sh <old-ref> <new-ref> [work-dir]" >&2; exit 2; }
REPO="$(git rev-parse --show-toplevel)"
BOOT_SECS="${DB_COMPAT_BOOT_SECS:-45}"
mkdir -p "$WORK"

build_at() { # <ref> <out>
  local ref="$1" out="$2" wt="$WORK/src-$1"
  git -C "$REPO" worktree add --detach "$wt" "$ref" >/dev/null 2>&1 || { echo "db-compat: cannot check out $ref"; return 1; }
  ( cd "$wt" && go build -trimpath -o "$out" ./ ) || { echo "db-compat: build failed at $ref"; return 1; }
  git -C "$REPO" worktree remove --force "$wt" >/dev/null 2>&1 || true
  return 0
}

# Run a binary long enough to open and migrate the database, then stop it. The
# exit code is deliberately NOT the verdict: with no NT8 the process may exit on
# its own, and that says nothing about the schema.
migrate_with() { # <binary> <datadir> <label>
  local bin="$1" dir="$2" label="$3" log="$WORK/$label.log"
  mkdir -p "$dir"
  ( cd "$dir" && timeout "${BOOT_SECS}s" "$bin" >"$log" 2>&1 ) || true
  if [ ! -f "$dir/data/data.db" ]; then
    echo "db-compat: $label — no database was created at $dir/data/data.db"
    tail -20 "$log" | sed 's/^/    /'
    return 1
  fi
  # A migration error must fail the step even if the file exists.
  if grep -qiE 'migrat(e|ion).*(fail|error)|AutoMigrate.*error|no such column|table .* has no column' "$log"; then
    echo "db-compat: $label — migration error in the log:"
    grep -iE 'migrat(e|ion).*(fail|error)|AutoMigrate.*error|no such column|table .* has no column' "$log" | head -5 | sed 's/^/    /'
    return 1
  fi
  local n
  n="$(sqlite3 "$dir/data/data.db" "select count(*) from sqlite_master where type='table';" 2>/dev/null || echo 0)"
  if [ "${n:-0}" -lt 5 ]; then
    echo "db-compat: $label — only $n table(s); the schema did not migrate"
    return 1
  fi
  echo "db-compat: $label ok ($n tables)"
  return 0
}

OLD_BIN="$WORK/nofx-old"; NEW_BIN="$WORK/nofx-new"
build_at "$OLD_REF" "$OLD_BIN" || exit 1
build_at "$NEW_REF" "$NEW_BIN" || exit 1

INST="$WORK/install"
echo "db-compat: pair $OLD_REF -> $NEW_REF"
migrate_with "$OLD_BIN" "$INST" "1-old-creates-fresh"      || { echo "db-compat: NOT PROVEN"; exit 1; }
migrate_with "$NEW_BIN" "$INST" "2-new-migrates-forward"   || { echo "db-compat: NOT PROVEN"; exit 1; }
# THE ROLLBACK ITSELF — the step that is normally skipped.
migrate_with "$OLD_BIN" "$INST" "3-old-boots-migrated-db"  || { echo "db-compat: NOT PROVEN — the rollback is the failing step"; exit 1; }
echo "db-compat: PROVEN $OLD_REF <-> $NEW_REF"
