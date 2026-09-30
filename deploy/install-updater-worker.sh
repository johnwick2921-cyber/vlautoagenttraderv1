#!/usr/bin/env bash
# UPDATER-USABLE-V1 — install the updater worker as a systemd --user unit.
#
# What this does, in order (every refusal is final and writes nothing):
#   1. The source must be a NAMED, CLEAN sha: 40 hex. The binary is built
#      from that exact commit in a THROWAWAY clone — the live install
#      (~/nofx) is never checked out, never modified, never built over.
#   2. %h/.config/nofx-updater/env (mode 0600) must exist and set
#      NOFX_RELEASE_DIR (absolute, OUTSIDE the install — the same containment
#      the worker enforces, internal/updaterworker/releaseroot.go) and
#      NOFX_CUTOVER_TOKEN (non-empty; never printed here).
#   3. go build -trimpath, then `go version -m` must report vcs.modified=false
#      and vcs.revision=<sha> — a dirty or mis-labelled build is refused.
#   4. The unit template at THAT sha is copied into ~/.config/systemd/user/
#      and systemctl --user daemon-reload + enable run. The worker is NOT
#      started here: start it attended (systemctl --user start nofx-updater).
#      serve refuses root, the bot's cgroup, and a set TZ — this unit avoids
#      all three by construction.
#
# The token: NOFX_CUTOVER_TOKEN is a gate-jwt whose lifetime is 24 HOURS
# (auth/auth.go:227). There is NO longer-lived token type — refresh it before
# each attended install window by re-minting and replacing the line in the env
# file. This script never prints it (pinned: the deploy test refuses a run
# whose output carries the token value).
#
# No privilege escalation anywhere in this script. Never run while the bot
# serves a hold — this script only writes ~/bin and ~/.config/systemd/user,
# but an install needs the worker and the bot quiet in the right order (see
# the runbook).
set -uo pipefail

SHA="${1:-}"
REPO_URL="${NOFX_UPDATER_BUILD_REPO:-https://github.com/johnwick2921-cyber/vlautoagenttraderv1}"  # PARTNER CARVE-OUT (PARTNER-SYNC-BOOT7): partner machines build from the partner repo
INSTALL_DIR="${NOFX_UPDATER_INSTALL_DIR:-$HOME/nofx}"

usage() {
  echo "usage: install-updater-worker.sh <40-hex sha>" >&2
  echo "  builds ~/bin/nofx-updater from that exact commit and installs the systemd --user unit" >&2
  echo "  env: NOFX_UPDATER_BUILD_REPO (default $REPO_URL), NOFX_UPDATER_INSTALL_DIR (default \$HOME/nofx)" >&2
}
[ -n "$SHA" ] || { usage; exit 2; }
printf '%s' "$SHA" | grep -Eqx '[0-9a-f]{40}' || {
  echo "install-updater-worker: REFUSED — the source must be a named 40-hex sha (got: $SHA)" >&2
  exit 2
}
[ -n "${HOME:-}" ] || { echo "install-updater-worker: REFUSED — no HOME" >&2; exit 2; }

ENV_FILE="$HOME/.config/nofx-updater/env"
[ -f "$ENV_FILE" ] || {
  echo "install-updater-worker: REFUSED — $ENV_FILE does not exist." >&2
  echo "  create it (mode 0600, owner you) with exactly two lines:" >&2
  echo "    NOFX_RELEASE_DIR=/absolute/path/outside/nofx" >&2
  echo "    NOFX_CUTOVER_TOKEN=<a fresh gate-jwt — 24h lifetime, refresh before each install window>" >&2
  exit 2
}
[ "$(stat -c '%a' "$ENV_FILE" 2>/dev/null)" = "600" ] || {
  echo "install-updater-worker: REFUSED — $ENV_FILE must be mode 0600" >&2
  exit 2
}

# Read the two required values WITHOUT printing them.
release_dir="$(awk -F= '$1=="NOFX_RELEASE_DIR"{print $2}' "$ENV_FILE" | tail -1)"
token_ok=no
grep -Eq '^NOFX_CUTOVER_TOKEN=.+' "$ENV_FILE" && token_ok=yes
{ [ -n "$release_dir" ] && [ "$token_ok" = "yes" ]; } || {
  echo "install-updater-worker: REFUSED — $ENV_FILE must set NOFX_RELEASE_DIR and NOFX_CUTOVER_TOKEN (both non-empty)" >&2
  exit 2
}
case "$release_dir" in
  /*) : ;;
  *) echo "install-updater-worker: REFUSED — NOFX_RELEASE_DIR must be an absolute path" >&2; exit 2 ;;
esac
# The containment check, on resolved paths, element-wise ("$install"/*) —
# never a string prefix (releaseroot.go's PathWithin is the worker's own one;
# this is the install-time mirror).
real_root="$(realpath -m "$release_dir" 2>/dev/null || echo "$release_dir")"
real_install="$(realpath -m "$INSTALL_DIR" 2>/dev/null || echo "$INSTALL_DIR")"
case "$real_root" in
  "$real_install"|"$real_install"/*)
    echo "install-updater-worker: REFUSED — NOFX_RELEASE_DIR ($release_dir) must be OUTSIDE the install ($INSTALL_DIR)" >&2
    exit 2 ;;
esac
echo "install-updater-worker: env ok (release dir outside the install; token present, not shown)"
echo "install-updater-worker: note — NOFX_CUTOVER_TOKEN is a 24-hour gate-jwt (auth/auth.go:227); refresh it before each attended install window — no longer-lived token type exists"

BUILD_DIR="$(mktemp -d /tmp/nofx-updater-build.XXXXXX)" || { echo "install-updater-worker: REFUSED — cannot make a build dir" >&2; exit 2; }
trap 'rm -rf "$BUILD_DIR"' EXIT
echo "install-updater-worker: cloning $REPO_URL (the live install is never touched)"
git clone -q "$REPO_URL" "$BUILD_DIR/src" 2>/dev/null || { echo "install-updater-worker: REFUSED — clone failed ($REPO_URL)" >&2; exit 2; }
git -C "$BUILD_DIR/src" checkout -q --detach "$SHA" 2>/dev/null || { echo "install-updater-worker: REFUSED — $SHA is not in $REPO_URL" >&2; exit 2; }
[ -z "$(git -C "$BUILD_DIR/src" status --porcelain)" ] || { echo "install-updater-worker: REFUSED — checkout not clean" >&2; exit 2; }
( cd "$BUILD_DIR/src" && go build -trimpath -o "$BUILD_DIR/nofx-updater" ./cmd/nofx-updater ) || {
  echo "install-updater-worker: REFUSED — build failed at $SHA" >&2
  exit 2
}
go version -m "$BUILD_DIR/nofx-updater" > "$BUILD_DIR/vcs.txt" 2>/dev/null || { echo "install-updater-worker: REFUSED — cannot read the build stamp" >&2; exit 2; }
grep -q 'vcs.modified=false' "$BUILD_DIR/vcs.txt" || { echo "install-updater-worker: REFUSED — vcs.modified is not false" >&2; exit 2; }
grep -q "vcs.revision=$SHA" "$BUILD_DIR/vcs.txt" || { echo "install-updater-worker: REFUSED — the binary does not carry the named sha" >&2; exit 2; }

mkdir -p "$HOME/bin"
install -m 0755 "$BUILD_DIR/nofx-updater" "$HOME/bin/nofx-updater" || { echo "install-updater-worker: REFUSED — cannot install $HOME/bin/nofx-updater" >&2; exit 2; }
mkdir -p "$HOME/.config/systemd/user"
cp "$BUILD_DIR/src/deploy/systemd-user/nofx-updater.service" "$HOME/.config/systemd/user/nofx-updater.service"
if command -v systemctl >/dev/null 2>&1 && systemctl --user daemon-reload 2>/dev/null; then
  if systemctl --user enable nofx-updater >/dev/null 2>&1; then
    echo "install-updater-worker: unit installed and enabled; start it attended: systemctl --user start nofx-updater"
  else
    echo "install-updater-worker: unit installed (enable skipped); start it attended: systemctl --user start nofx-updater"
  fi
else
  echo "install-updater-worker: note — systemctl unavailable; the unit file is installed at $HOME/.config/systemd/user/nofx-updater.service"
fi
echo "install-updater-worker: DONE — $HOME/bin/nofx-updater from $SHA (clean, stamped); unit installed"
