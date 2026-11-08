#!/usr/bin/env bash
# vl-published-tag.sh — partner-only (m470). Sourced by deploy/vl-auto-sync.sh.
#
# A vl tag is not a vl release. Since vl release #63 a tag can stop at vl's
# quality gate (its test suite red on the tagged code), and a held release is
# marked pre-release on purpose. Either way the tag exists but nothing was
# published, and the partner must NOT build or ship it. Before m470 the sync
# took the newest v* tag by name, so a tag the vl gate refused still reached
# Binnie's Update button.
#
# vl_release_state TAG  -> prints one word:
#   published    a vl release exists, not a draft, not a pre-release
#   held         a vl release exists but is a draft or a pre-release
#   unpublished  no vl release for this tag (yet, or refused by the gate)
# Any other failure to read (network, auth, rate limit) is NOT "unpublished":
# it returns 2 so the caller refuses the whole sync instead of guessing.
#
# Env: VL_SLUG (owner/repo of vl), GH_BIN (default gh; tests stub it).

vl_release_state() {
  local tag="$1" out rc
  out="$("${GH_BIN:-gh}" api "repos/${VL_SLUG:?}/releases/tags/${tag}" \
          --jq '"\(.draft) \(.prerelease)"' 2>&1)" && rc=0 || rc=$?
  if [ "$rc" -eq 0 ]; then
    if [ "$out" = "false false" ]; then echo published; else echo held; fi
    return 0
  fi
  case "$out" in
    *"HTTP 404"*|*"Not Found"*) echo unpublished; return 0 ;;
  esac
  printf 'cannot read the vl release for %s: %s\n' "$tag" "$out" >&2
  return 2
}

# newest_published_vl_tag: reads candidate tag names on stdin (any order),
# prints the newest one (sort -V) that vl has PUBLISHED. Looks at most at
# VL_TAG_LOOKBACK (default 20) newest tags. Prints a skip line per newer tag
# on stderr so the run log says why it was not synced. Returns 1 when none of
# the candidates is published, 2 when a release could not be read.
newest_published_vl_tag() {
  local t st n=0
  while IFS= read -r t; do
    [ -n "$t" ] || continue
    n=$((n + 1))
    [ "$n" -le "${VL_TAG_LOOKBACK:-20}" ] || break
    st="$(vl_release_state "$t")" || return 2
    if [ "$st" = published ]; then
      printf '%s\n' "$t"
      return 0
    fi
    printf 'skip: vl tag %s is %s — not synced until vl publishes it\n' "$t" "$st" >&2
  done < <(grep '^v' | sort -rV)
  return 1
}
