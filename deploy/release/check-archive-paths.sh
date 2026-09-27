#!/usr/bin/env bash
# W-ONE-BUTTON M4 — refuse an archive member that escapes its extraction root.
#
# A tar/zip entry may be absolute, may contain .., or may be a symlink that
# points outside the tree. Any of those lets an archive write wherever the
# extracting process can write. The updater will extract this archive on the
# owner's machine, so the check belongs at PACKAGING time, before a bad path
# exists to be extracted.
#
# Usage: check-archive-paths.sh <path> [<path> ...]
# Exit 0 = every path is a safe relative path. Non-zero = REFUSED.
set -uo pipefail

[ "$#" -gt 0 ] || { echo "check-archive-paths: usage: <path> [...]" >&2; exit 2; }

bad=0
for p in "$@"; do
  case "$p" in
    /*)        echo "check-archive-paths: REFUSED — absolute path: $p"; bad=1; continue ;;
    *'..'*)    echo "check-archive-paths: REFUSED — path escapes the root: $p"; bad=1; continue ;;
    '~'*)      echo "check-archive-paths: REFUSED — home-relative path: $p"; bad=1; continue ;;
  esac
  # A leading ./ is fine; anything with a NUL or newline is not (it breaks every
  # downstream tool that reads a path list).
  case "$p" in
    *$'\n'*)   echo "check-archive-paths: REFUSED — newline in path: ${p//$'\n'/\\n}"; bad=1; continue ;;
  esac
done

[ "$bad" -eq 0 ] || { echo "check-archive-paths: REFUSED"; exit 1; }
echo "check-archive-paths: ok ($# path(s))"
