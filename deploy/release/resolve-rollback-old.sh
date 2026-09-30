#!/usr/bin/env bash
# UPDATER-USABLE-V1 (item C) — resolve the OLD commit dbcompat must prove a
# release against.
#
# Usage: resolve-rollback-old.sh <new-sha> <current-tag> [rollback-from]
# Prints the OLD commit (40 hex) on stdout, or NOTHING when no old is
# provable (the pair is then ABSENT, never fabricated — canon 49/53).
#
# Rules:
#   1. rollback_from wins verbatim when it is 40 hex — it is the RUNNING
#      RELEASE the operator passed, the strongest possible answer.
#   2. Else the highest-version v* tag that is (a) not the current tag,
#      (b) REACHABLE from the new commit (an ancestor — a tag on a sideline
#      branch proves nothing about THIS release line), and (c) NOT from the
#      ancient v1.0 line (v1.0.* and v1.0-* — pre-release history; never
#      the rollback target).
#   3. None provable ⇒ nothing. An unprovable pair is absent; it is never
#      invented and never downgraded to a guess.
set -uo pipefail

NEW="${1:-}"; CUR="${2:-}"; FROM="${3:-}"
printf '%s' "$NEW" | grep -Eqx '[0-9a-f]{40}' || { echo "resolve-rollback-old: the new sha must be 40 hex" >&2; exit 2; }
[ -n "$CUR" ] || { echo "resolve-rollback-old: the current tag is required" >&2; exit 2; }

if printf '%s' "${FROM:-}" | grep -Eqx '[0-9a-f]{40}'; then
  printf '%s\n' "$FROM"
  exit 0
fi

# Iterate the v* tags from the highest version down. %(objectname) may be a
# tag object (annotated tag) — peel to the commit before the ancestor check.
while read -r name obj; do
  [ "$name" = "$CUR" ] && continue
  case "$name" in
    v1.0.*|v1.0-*) continue ;; # never the ancient v1.0 line
  esac
  commit="$(git rev-parse "${obj}^{commit}" 2>/dev/null || true)"
  printf '%s' "$commit" | grep -Eqx '[0-9a-f]{40}' || continue
  if git merge-base --is-ancestor "$commit" "$NEW" 2>/dev/null; then
    printf '%s\n' "$commit"
    exit 0
  fi
done < <(git for-each-ref --sort=-v:refname --format='%(refname:short) %(objectname)' 'refs/tags/v*')
exit 0
