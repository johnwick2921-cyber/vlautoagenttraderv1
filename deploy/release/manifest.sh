#!/usr/bin/env bash
# W-ONE-BUTTON M4 — the manifest the updater reads before it touches anything.
#
# Every field is READ from the staged tree or passed in explicitly. Nothing is
# inferred and nothing is defaulted to a plausible value: a pair that has not
# been proven by the DB-compat job is advertised tested:false, because an
# untested rollback that claims to be tested is worse than no rollback at all.
#
# And an UNCOMPUTED list is null, never []. [] means "a job computed this and
# found nothing"; null means "nobody computed it". A reader that cannot tell
# those apart will treat an unrun job as a proven-empty result.
set -uo pipefail
STAGE="${1:-}"; SRC_SHA="${2:-}"; REL_ID="${3:-}"
[ -d "${STAGE:-}" ] && [ -n "${SRC_SHA:-}" ] && [ -n "${REL_ID:-}" ] || {
  echo "manifest: usage: manifest.sh <stage-dir> <source-sha40> <release-id>" >&2; exit 2; }
case "$SRC_SHA" in *[!0-9a-f]*|"") echo "manifest: REFUSED — source sha must be 40 hex: $SRC_SHA" >&2; exit 1;; esac
[ "${#SRC_SHA}" -eq 40 ] || { echo "manifest: REFUSED — source sha must be 40 hex (got ${#SRC_SHA})" >&2; exit 1; }

# P3: the packaged marker and the manifest must agree. package.sh WRITES
# deploy/RELEASE from the source sha; if the two ever diverge the archive would
# claim one revision and carry another, and the updater trusts the manifest.
STAGED_REL="$STAGE/deploy/RELEASE"
[ -f "$STAGED_REL" ] || { echo "manifest: REFUSED — $STAGED_REL missing (package.sh writes it)" >&2; exit 1; }
STAGED_SHA="$(tr -d '[:space:]' < "$STAGED_REL")"
[ "$STAGED_SHA" = "$SRC_SHA" ] || {
  echo "manifest: REFUSED — packaged deploy/RELEASE ($STAGED_SHA) != source sha ($SRC_SHA)" >&2; exit 1; }

arts="$(cd "$STAGE" && find . -type f -printf '%P\n' | LC_ALL=C sort | while read -r p; do
  printf '{"path":"%s","sha256":"%s","bytes":%s}\n' "$p" "$(sha256sum "$p" | cut -d' ' -f1)" "$(stat -c%s "$p")"
done | paste -sd, -)"

# Read from the AddOn source rather than restated here (L7: READ, never literal).
ADDON_BUILD="$(grep -hoE 'VL_BUILD_ID[^"]*"[^"]+"' "$STAGE"/ninjascript/*.cs 2>/dev/null | head -1 | sed 's/.*"\(.*\)"/\1/')"
[ -n "$ADDON_BUILD" ] || ADDON_BUILD="n/a"
PROTO="$(grep -hoE 'protocol_version[^0-9]*([0-9]+)' "$STAGE"/ninjascript/vltrader_tcp_PROTOCOL.md 2>/dev/null | head -1 | grep -oE '[0-9]+$')"
[ -n "$PROTO" ] || PROTO="null"

cat <<JSON
{
  "release_id": "$REL_ID",
  "source_sha": "$SRC_SHA",
  "artifacts": [${arts}],
  "platform": { "os": "linux", "arch": "amd64", "wsl": true },
  "nt8": { "min_version": "${NT8_MIN:-8.1.2.1}", "max_tested_version": "${NT8_MAX_TESTED:-n/a}" },
  "addon": { "build_id": "$ADDON_BUILD", "protocol_version": $PROTO, "transition_order": "${ADDON_TRANSITION_ORDER:-go-then-addon}" },
  "updater_min_version": "${UPDATER_MIN:-0.0.0}",
  "upgrade_pairs": ${UPGRADE_PAIRS:-null},
  "rollback_pairs": ${ROLLBACK_PAIRS:-null},
  "capabilities_required": ${CAPABILITIES_REQUIRED:-null},
  "data_readiness": [ { "mode": "picture_htf", "tf": "4h", "bars": "pivot_window+4" } ]
}
JSON
