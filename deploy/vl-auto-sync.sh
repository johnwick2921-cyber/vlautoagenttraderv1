#!/usr/bin/env bash
# vl-auto-sync.sh — owner→partner release sync (DS-105, 2026-10-17).
#
# When vl publishes a release tag (vYYYY.MM.DD.N), this script brings the
# partner tree to that tag: the machine-pinned carve-outs are kept, the
# result is byte-parity-checked against the vl tag, then gated on
# build + vet + goldens + deploy/updater tests + gitleaks. On green it commits the tree-sync + stamp
# + runbook and (unless DRY_RUN) pushes a branch + opens a PR. On any failure
# it prints "REFUSED — <reason>" and exits non-zero WITHOUT publishing.
#
# THE ONE RULE: nothing is deleted. The carve-out list is read from
# deploy/partner_sync_carveouts (one path per line, '#' comments, blanks ok).
#
# Env:
#   VL_REPO            vl repo URL   (default https://github.com/johnwick2921-cyber/vl.git)
#   NEW_TAG            the vl tag to sync (default: newest v* tag from VL_REPO)
#   NEW_SHA            override: sync to this exact vl sha (dry-run proof)
#   DRY_RUN=1          commit but do NOT push / open a PR / write the state file
#   GITLEAKS           path to gitleaks  (default ~/go/bin/gitleaks)
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

# A bare clone has no identity; the CI checkout sets one. Set a REPO-LOCAL
# identity (survives the HOME redirection below, which hides ~/.gitconfig) so
# the sync commits always succeed.
if [ -z "$(git config --local user.email 2>/dev/null || true)" ]; then
  git config --local user.email "vl-auto-sync@users.noreply.github.com"
  git config --local user.name "vl-auto-sync"
fi

VL_REPO="${VL_REPO:-https://github.com/johnwick2921-cyber/vl.git}"
GITLEAKS_BIN="${GITLEAKS:-$HOME/go/bin/gitleaks}"
CARVEOUTS="$ROOT/deploy/partner_sync_carveouts"
STATE="$ROOT/deploy/LAST_SYNCED_VL_TAG"
WORK="$(mktemp -d -t vl-autosync.XXXXXX)"
TH="$(mktemp -d -t vl-autosync-home.XXXXXX)"
trap 'rm -rf "${WORK:?}" "${TH:?}"' EXIT

say()  { printf '%s\n' "$*"; }
die()  { say "REFUSED — $*" >&2; exit 1; }

# --- 1. fetch vl refs into this clone (objects must be LOCAL for the parity check) ------
say "fetch: $VL_REPO"
git fetch --quiet "$VL_REPO" '+refs/tags/*:refs/vl/tags/*' '+refs/heads/*:refs/vl/heads/*'

# --- 2. resolve the target (new) vl sha -------------------------------------------------
NEW_TAG="${NEW_TAG:-}"
NEW_SHA_RESOLVED=""
if [ -n "${NEW_SHA:-}" ]; then
  NEW_SHA_RESOLVED="$NEW_SHA"
  NEW_TAG="${NEW_TAG:-manual-sha}"
else
  if [ -z "$NEW_TAG" ]; then
    NEW_TAG="$(git for-each-ref --format='%(refname:short)' refs/vl/tags \
                 | sed 's#vl/tags/##' | grep '^v' | sort -V | tail -1 || true)"
    [ -n "$NEW_TAG" ] || NEW_TAG="$(git ls-remote --tags "$VL_REPO" 'v*' \
                 | sed 's#.*refs/tags/##;s#\^{}##' | sort -V | tail -1 || true)"
    [ -n "$NEW_TAG" ] || die "no v* tag found in $VL_REPO"
  fi
  NEW_SHA_RESOLVED="$(git rev-parse "refs/vl/tags/${NEW_TAG}^{}" 2>/dev/null || true)"
  [ -n "$NEW_SHA_RESOLVED" ] || NEW_SHA_RESOLVED="$(git ls-remote "$VL_REPO" "refs/tags/${NEW_TAG}^{}" \
      | awk '{print $1}' | head -1 || true)"
  [ -n "$NEW_SHA_RESOLVED" ] || die "cannot resolve vl tag $NEW_TAG"
fi

# --- 3. resolve the last synced vl sha from the state file ------------------------------
[ -f "$STATE" ] || die "missing $STATE (record the last synced tag first)"
LAST_TAG="$(tr -d '[:space:]' < "$STATE")"
[ -n "$LAST_TAG" ] || die "empty $STATE"
LAST_SHA="$(git rev-parse "refs/vl/tags/${LAST_TAG}^{}" 2>/dev/null || true)"
[ -n "$LAST_SHA" ] || LAST_SHA="$(git ls-remote "$VL_REPO" "refs/tags/${LAST_TAG}^{}" | awk '{print $1}' | head -1 || true)"
[ -n "$LAST_SHA" ] || die "cannot resolve last synced tag $LAST_TAG"

if [ "$LAST_SHA" = "$NEW_SHA_RESOLVED" ]; then
  say "up to date: $LAST_TAG ($LAST_SHA)"; exit 0
fi
say "sync: $LAST_TAG ($LAST_SHA) .. ${NEW_TAG} ($NEW_SHA_RESOLVED)"

# --- 4. build the carve-out excludes from the machine-pinned list -----------------------
EXCLUDES=()
while IFS= read -r line; do
  case "$line" in ''|'#'*) continue;; esac
  EXCLUDES+=(":(exclude)$line")
done < "$CARVEOUTS"
[ "${#EXCLUDES[@]}" -gt 0 ] || die "empty carve-out list $CARVEOUTS"

# --- 5. net-diff apply (carve-outs kept) -------------------------------------------------
DIFF="$WORK/sync.diff"
# --binary: vl ships binary test fixtures (e.g. trader/testdata/*.json.gz, vl #45);
# a text-only diff cannot carry them and git apply refuses the WHOLE sync
# ("cannot apply binary patch ... without full index line", 10-22 runs).
git diff --binary "$LAST_SHA" "$NEW_SHA_RESOLVED" -- . "${EXCLUDES[@]}" > "$DIFF"
[ -s "$DIFF" ] || die "empty diff $LAST_TAG..${NEW_TAG} (carve-out list may be wrong)"

if ! git apply --3way --whitespace=nowarn "$DIFF" 2>"$WORK/apply.log"; then
  die "git apply failed; tail:\n$(tail -20 "$WORK/apply.log")"
fi
git add -A

# --- 6. byte parity: every non-carve-out file vl CHANGED must equal the vl tag ----------
# (Files vl did not touch in the range — e.g. partner-only files — are allowed to differ.)
CHANGED="$WORK/changed"
git diff --name-only "$LAST_SHA" "$NEW_SHA_RESOLVED" -- . "${EXCLUDES[@]}" > "$CHANGED"
PARITY_BAD="$WORK/paritybad"
: > "$PARITY_BAD"
while IFS= read -r f; do
  [ -z "$f" ] && continue
  A="$(git rev-parse ":$f" 2>/dev/null || echo none)"          # staged blob
  B="$(git rev-parse "$NEW_SHA_RESOLVED:$f" 2>/dev/null || echo none)"  # vl blob
  [ "$A" = "$B" ] || echo "$f (staged=$A vl=$B)" >> "$PARITY_BAD"
done < "$CHANGED"
if [ -s "$PARITY_BAD" ]; then
  say "parity failures:"
  cat "$PARITY_BAD"
  die "byte parity failed — a non-carve-out file changed by vl $NEW_TAG differs"
fi
say "parity: $(wc -l < "$CHANGED") files changed by vl $NEW_TAG are byte-identical"

# --- 7. commit the tree-sync (clean tree → the gates run at this sha) -------------------
git commit --quiet -m "tree-sync vl ${LAST_SHA}..${NEW_SHA_RESOLVED} (auto: $NEW_TAG)"
TREE_SYNC_SHA="$(git rev-parse HEAD)"

# --- 8. gates: build, vet, goldens, gitleaks (at the clean tree-sync commit) ------------
# Resolve the Go caches from the REAL environment BEFORE HOME is pointed at the
# throwaway dir (THE ONE RULE isolation), so go keeps its caches on any machine —
# a CI runner or a box. Never a hard-coded box path (auto-sync run 39056053329
# refused on /home/hoang: "mkdir /home/hoang: permission denied").
GOCACHE="${GOCACHE:-$(go env GOCACHE)}"
GOMODCACHE="${GOMODCACHE:-$(go env GOMODCACHE)}"
export GOCACHE GOMODCACHE
export HOME="$TH"

say "gate: go build"
go build -trimpath -o "$WORK/vl-bin" . >"$WORK/build.log" 2>&1 || die "go build failed: $(tail -5 "$WORK/build.log")"
VCS="$(go version -m "$WORK/vl-bin" 2>/dev/null | grep -E 'vcs.revision|vcs.modified' | tr '\n' ' ')"
case "$VCS" in
  *"vcs.revision=$TREE_SYNC_SHA"*"vcs.modified=false"*) : ;;
  *) die "build stamp wrong: $VCS (want revision=$TREE_SYNC_SHA modified=false)" ;;
esac
say "gate: build ok — $VCS"

say "gate: go vet"
go vet ./... >"$WORK/vet.log" 2>&1 || die "go vet failed: $(tail -10 "$WORK/vet.log")"

say "gate: goldens"
go test ./kernel/ -run 'Golden|Selfcheck' -count=1 >"$WORK/golden.log" 2>&1 || die "goldens failed: $(tail -10 "$WORK/golden.log")"

# The partner's own carve-out tests ride here (2026-10-17: the vl #35 sync
# merged with deploy/release_contract_test.go — a carve-out — missing vl's new
# fixture line, and partner main went red after the merge). deploy/ holds the
# release contract + carve-out pins; internal/updaterworker/ is the one-click
# Update path the partner box runs.
say "gate: deploy + updater tests"
go test ./deploy/... ./internal/updaterworker/... -count=1 >"$WORK/deploytest.log" 2>&1 \
  || die "deploy/updater tests failed: $(grep -E -- '--- FAIL|FAIL|panic' "$WORK/deploytest.log" | head -10)"

say "gate: gitleaks (delta)"
if "$GITLEAKS_BIN" detect --no-git --source . --report-format json --report-path "$WORK/gitleaks.json" >"$WORK/gitleaks.log" 2>&1; then
  say "gate: gitleaks ok — no findings"
else
  # findings present — the sync is a byte-copy of vl, so an in-range finding must be
  # byte-identical to vl (pre-existing dummy). A NEW in-range finding is a refusal.
  FIND_FILES="$(python3 -c "import json;d=json.load(open('$WORK/gitleaks.json'));print('\n'.join(sorted({f['File'] for f in d})))" 2>/dev/null || true)"
  IN_RANGE="$WORK/inrange"
  : > "$IN_RANGE"
  while IFS= read -r f; do
    [ -z "$f" ] && continue
    if git diff --quiet "$LAST_SHA" "$NEW_SHA_RESOLVED" -- "$f" 2>/dev/null; then
      : # untouched by the sync — pre-existing partner file
    else
      # in-range: must be blob-identical to vl at the tag (HEAD is the tree-sync)
      A="$(git rev-parse "HEAD:$f" 2>/dev/null || echo none)"
      B="$(git rev-parse "$NEW_SHA_RESOLVED:$f" 2>/dev/null || echo none)"
      [ "$A" = "$B" ] || echo "$f (head=$A vl=$B)" >> "$IN_RANGE"
    fi
  done <<< "$FIND_FILES"
  if [ -s "$IN_RANGE" ]; then
    die "gitleaks delta non-zero for in-range files: $(cat "$IN_RANGE")"
  fi
  say "gate: gitleaks ok — findings present but all pre-existing (delta 0)"
fi

# --- 9. stamp + runbook ----------------------------------------------------------------
printf '%s' "$TREE_SYNC_SHA" > deploy/RELEASE
git add deploy/RELEASE
git commit --quiet -m "stamp tip: deploy/RELEASE = $TREE_SYNC_SHA"

ADDON_BUILD="$(grep -m1 'VL_BUILD_ID *= *"' ninjascript/VLTraderTCPClient.cs 2>/dev/null | sed 's/.*"\([^"]*\)".*/\1/' || echo unknown)"
{
  printf '\n## AUTO — partner sync (vl %s, DS-105 auto-sync, %s)\n\n' "$NEW_TAG" "$(date -u +%Y-%m-%d)"
  printf 'Auto-synced to vl tag `%s` (`%s…`). Build commit `%s…` (see deploy/RELEASE).\n' "$NEW_TAG" "$NEW_SHA_RESOLVED" "$TREE_SYNC_SHA"
  printf 'AddOn VL_BUILD_ID: %s\n' "$ADDON_BUILD"
  printf 'Carve-outs: %s machine-pinned paths kept (deploy/partner_sync_carveouts).\n' "$(grep -vc '^#\|^$' "$CARVEOUTS")"
} >> docs/superpowers/runbooks/2026-09-22-vl-partner-update.md
git add docs/superpowers/runbooks/2026-09-22-vl-partner-update.md
git commit --quiet -m "runbook: auto section for $NEW_TAG"

say "SYNCED $LAST_TAG .. $NEW_TAG — tree-sync $TREE_SYNC_SHA"

# --- 10. publish (unless dry run) -------------------------------------------------------
if [ "${DRY_RUN:-0}" = "1" ]; then
  say "DRY_RUN — not pushed, no PR, state file untouched"
  exit 0
fi

# The state file rides IN the PR, committed before the one push: a second
# best-effort push could fail silently and merge a sync that never records
# its tag (the next run would re-sync it).
printf '%s\n' "$NEW_TAG" > "$STATE"
git add "$STATE"
git commit --quiet -m "record last synced vl tag: $NEW_TAG"

# THE ONE RULE: a branch left by an earlier failed run is never deleted or
# force-pushed over — a retry pushes to a fresh name instead.
BRANCH="auto-sync/${NEW_TAG}"
if git ls-remote --exit-code origin "refs/heads/$BRANCH" >/dev/null 2>&1; then
  BRANCH="auto-sync/${NEW_TAG}-${GITHUB_RUN_ID:-$(date +%s)}"
  say "auto-sync/${NEW_TAG} already exists (earlier failed run) — using $BRANCH"
fi
git push origin "HEAD:$BRANCH" >"$WORK/push.log" 2>&1 || die "push failed: $(tail -5 "$WORK/push.log")"
gh pr create --base main --head "$BRANCH" --title "partner auto-sync: vl $NEW_TAG" \
  --body "Auto-sync of vl tag \`$NEW_TAG\` (\`$NEW_SHA_RESOLVED\`). Gates: build/vet/goldens/deploy+updater tests/gitleaks green." \
  >"$WORK/pr.log" 2>&1 || die "gh pr create failed: $(tail -5 "$WORK/pr.log")"
say "PR opened: $(tail -1 "$WORK/pr.log")"

# Report the resolved tag/branch to the calling workflow (GitHub Actions
# reads these as step outputs). Absent outside CI; harmless when unset.
if [ -n "${GITHUB_OUTPUT:-}" ]; then
  {
    echo "tag=$NEW_TAG"
    echo "branch=$BRANCH"
    echo "sha=$NEW_SHA_RESOLVED"
  } >> "$GITHUB_OUTPUT"
fi
