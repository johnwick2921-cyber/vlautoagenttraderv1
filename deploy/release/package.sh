#!/usr/bin/env bash
# W-ONE-BUTTON M4 — stage ONLY the allow-list, then prove what was staged.
#
# ALLOW-LIST, not deny-list, for what goes IN: a deny-list decides what to
# leave out and therefore ships anything nobody thought of. The secret scan is
# the deny-list, and it runs as a second, independent opinion over the result.
#
# What a release owns (design notes R-e/R-f): the binary, the built web assets,
# the RELEASE marker, the AddOn sources and their protocol doc, the updater
# binaries once they exist, and the licence. Nothing else. In particular the
# installation keeps its own data/ and .env — they are NEVER shipped and never
# overwritten (R-d).
set -uo pipefail
SRC="${1:-}"; OUT="${2:-}"
[ -d "${SRC:-}" ] && [ -n "${OUT:-}" ] || { echo "package: usage: package.sh <repo-root> <stage-dir>" >&2; exit 2; }
HERE="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

ALLOW=(
  "nofx-bin"
  "web/dist"
  "deploy/RELEASE"
  "ninjascript/vltrader_tcp_PROTOCOL.md"
  "LICENSE"
)
OPTIONAL=( "updater/nofx-updater" "updater/nofx-updater-bootstrap" "calendar_static_t1.json" )

mkdir -p "$OUT"
staged=()
copy_one() {
  local rel="$1" required="$2"
  if [ ! -e "$SRC/$rel" ]; then
    if [ "$required" = "required" ]; then echo "package: REFUSED — required artifact missing: $rel"; return 1; fi
    return 0
  fi
  bash "$HERE/check-archive-paths.sh" "$rel" >/dev/null || { echo "package: REFUSED — unsafe path: $rel"; return 1; }
  mkdir -p "$OUT/$(dirname "$rel")"
  cp -a "$SRC/$rel" "$OUT/$rel"
  staged+=("$rel")
}

rc=0
for rel in "${ALLOW[@]}"; do copy_one "$rel" required || rc=1; done
# the AddOn sources, by glob, each path checked
if compgen -G "$SRC/ninjascript/*.cs" >/dev/null; then
  for f in "$SRC"/ninjascript/*.cs; do copy_one "ninjascript/$(basename "$f")" required || rc=1; done
else
  echo "package: REFUSED — no ninjascript/*.cs found"; rc=1
fi
for rel in "${OPTIONAL[@]}"; do copy_one "$rel" optional || rc=1; done
[ "$rc" -eq 0 ] || { echo "package: REFUSED"; exit 1; }

# Every staged path re-checked as a set, so a glob cannot smuggle one through.
mapfile -t all < <(cd "$OUT" && find . -type f -printf '%P\n')
bash "$HERE/check-archive-paths.sh" "${all[@]}" >/dev/null || { echo "package: REFUSED — unsafe staged path"; exit 1; }

# A symlink in the stage would resolve on the OWNER's machine, not ours.
if find "$OUT" -type l | grep -q .; then
  echo "package: REFUSED — symlink in the staged tree:"; find "$OUT" -type l; exit 1
fi
echo "package: staged ${#all[@]} file(s) from ${#staged[@]} allow-list entr(ies)"
