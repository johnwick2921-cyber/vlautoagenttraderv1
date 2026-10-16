#!/usr/bin/env bash
# partner-first-time-setup.sh — ONE-TIME setup for the PARTNER machine's
# one-button update (UPDATER-USABLE-V1). Idempotent, attended, no sudo.
#
# usage: partner-first-time-setup.sh [--dry-run] [--install-dir <dir>]
#                                    [--release-dir <dir>] <owner-email>
#
# Steps (each printed; any failure REFUSES with a reason and writes nothing):
#   1. prerequisites: go, git, curl on PATH; systemctl --user + linger; the bot
#      install dir; <install>/deploy/release_allowed_signers = the PARTNER key
#      (never the owner repo's key).
#   2. enroll (attended): `vl-updater-bootstrap --install-dir <dir> enroll <email>`
#      — writes the enrollment files and mints the worker token into
#      ~/.config/vl-updater/env (the token is NEVER printed).
#   3. env file: ensures ~/.config/vl-updater/env (mode 0600) carries
#      VL_RELEASE_DIR (absolute, OUTSIDE the install) — the token line enroll
#      wrote is preserved byte-identically.
#   4. VL_UPDATER=1 in <install>/.env (backed up first; every other line kept).
#   5. deploy/install-updater-worker.sh at the CURRENT partner main sha (built
#      from that exact commit in a throwaway clone).
#   6. systemctl --user start vl-updater.
#   7. GET /api/updates must answer install_enabled=true + worker_listening=true
#      → PASS, else FAIL (auth is required: export VL_ADMIN_TOKEN with your
#      logged-in session Bearer token, or open Settings → Updates in a browser).
#
# --dry-run prints every action and writes NOTHING.
#
# Secrets: the worker token is never printed. The final API check's Bearer
# token is read from the VL_ADMIN_TOKEN environment and never echoed.
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"

# The bot install dir and the repo URL follow install-updater-worker.sh: the
# VL_ names win; the defaults are the partner repo and $HOME/vl (carve-out).
REPO_URL="${VL_UPDATER_BUILD_REPO:-https://github.com/johnwick2921-cyber/vlautoagenttraderv1}"
INSTALL_DIR="${VL_UPDATER_INSTALL_DIR:-$HOME/vl}"
API_PORT="${VL_API_PORT:-8080}"
API_HOST="${VL_API_HOST:-127.0.0.1}"

# The owner repo's (johnwick2921-cyber/vl) release public key — a partner box
# must never trust it (pinned by deploy/partner_signers_test.go).
VL_OWNER_RELEASE_KEY="AAAAC3NzaC1lZDI1NTE5AAAAIFAnymAm3evz3IP3DQPjgaBIGzMpobd55aLLhWhtwVqF"

DRY_RUN=false
RELEASE_DIR="${VL_RELEASE_DIR:-}"
EMAIL=""

usage() {
  echo "usage: partner-first-time-setup.sh [--dry-run] [--install-dir <dir>] [--release-dir <dir>] <owner-email>" >&2
  echo "  env: VL_RELEASE_DIR (outside the install), VL_ADMIN_TOKEN (the final GET /api/updates check)" >&2
  exit 2
}

while [ $# -gt 0 ]; do
  case "$1" in
    --dry-run) DRY_RUN=true ;;
    --install-dir) [ $# -ge 2 ] || usage; INSTALL_DIR="$2"; shift ;;
    --release-dir) [ $# -ge 2 ] || usage; RELEASE_DIR="$2"; shift ;;
    -h|--help) usage ;;
    --) shift; break ;;
    -*) echo "unknown flag: $1" >&2; usage ;;
    *) [ -n "$EMAIL" ] && usage; EMAIL="$1" ;;
  esac
  shift
done

[ -n "${HOME:-}" ] || { echo "REFUSED — no HOME" >&2; exit 2; }
[ -n "$EMAIL" ] || usage

# Email must look like an address (the same shape enroll's lookup uses; an
# empty or malformed one is refused before anything runs).
case "$EMAIL" in
  *@*.*) : ;;
  *) echo "REFUSED — <owner-email> must look like an address (got: $EMAIL)" >&2; exit 2 ;;
esac

say()  { echo "partner-first-time-setup: $*"; }
refuse() { echo "partner-first-time-setup: REFUSED — $*" >&2; exit 2; }

# step_run prints and executes a step; in --dry-run it prints only.
step_run() {
  if [ "$DRY_RUN" = true ]; then
    say "would run: $*"
    return 0
  fi
  say "run: $*"
  "$@"
}

# ── 1. prerequisites ───────────────────────────────────────────────────────
for tool in go git curl systemctl loginctl; do
  command -v "$tool" >/dev/null 2>&1 || refuse "$tool is not on PATH"
done

systemctl --user is-system-running >/dev/null 2>&1 \
  || refuse "systemctl --user is not usable (run: loginctl enable-linger $USER, then re-open the session)"
linger="$(loginctl show-user "$USER" -p Linger --value 2>/dev/null || true)"
[ "$linger" = "yes" ] || refuse "linger is not enabled for $USER (run: loginctl enable-linger $USER)"

[ -d "$INSTALL_DIR" ] || refuse "install dir does not exist: $INSTALL_DIR (set --install-dir or VL_UPDATER_INSTALL_DIR)"
[ -f "$INSTALL_DIR/.env" ] || refuse "$INSTALL_DIR/.env does not exist (the bot's env; back up first, then create it)"

# The installed systemd unit hardcodes %h/vl (deploy/systemd-user/vl-updater.service
# ExecStart=%h/bin/vl-updater --install-dir %h/vl). A worker built for any other
# install dir would never run under that unit — refuse it rather than install a
# worker that never starts.
real_default_install="$(realpath -m "$HOME/vl" 2>/dev/null || echo "$HOME/vl")"
real_cfg_install="$(realpath -m "$INSTALL_DIR" 2>/dev/null || echo "$INSTALL_DIR")"
if [ "$real_cfg_install" != "$real_default_install" ]; then
  refuse "the updater unit runs against $HOME/vl; a different install dir is not supported"
fi

# The trust anchor: exactly the partner key, never the owner's.
SIGNERS="$INSTALL_DIR/deploy/release_allowed_signers"
[ -f "$SIGNERS" ] || refuse "$SIGNERS does not exist (the worker verifies releases against it)"
if grep -q "$VL_OWNER_RELEASE_KEY" "$SIGNERS"; then
  refuse "$SIGNERS carries the OWNER repo's key — a partner sync overwrote the partner trust anchor; restore the partner key"
fi
if [ "$(grep -Ec '^release ssh-ed25519 ' "$SIGNERS" 2>/dev/null || echo 0)" != "1" ]; then
  refuse "$SIGNERS must hold exactly one 'release ssh-ed25519 …' line (got: $(grep -Ec '^release ssh-ed25519 ' "$SIGNERS" 2>/dev/null || echo 0))"
fi
say "prerequisites ok (go/git/curl/systemctl/linger, install dir, partner signers)"

# ── 2. release dir ─────────────────────────────────────────────────────────
if [ -z "$RELEASE_DIR" ]; then
  if [ "$DRY_RUN" = true ]; then
    RELEASE_DIR="$HOME/vl-releases"
    say "--dry-run: using placeholder release dir $RELEASE_DIR"
  else
    printf 'VL_RELEASE_DIR (absolute, OUTSIDE %s): ' "$INSTALL_DIR"
    IFS= read -r RELEASE_DIR || refuse "cannot read VL_RELEASE_DIR"
  fi
fi
case "$RELEASE_DIR" in
  /*) : ;;
  *) refuse "VL_RELEASE_DIR must be an absolute path (got: $RELEASE_DIR)" ;;
esac
real_root="$(realpath -m "$RELEASE_DIR" 2>/dev/null || echo "$RELEASE_DIR")"
real_install="$(realpath -m "$INSTALL_DIR" 2>/dev/null || echo "$INSTALL_DIR")"
case "$real_root" in
  "$real_install"|"$real_install"/*)
    refuse "VL_RELEASE_DIR ($RELEASE_DIR) must be OUTSIDE the install ($INSTALL_DIR)" ;;
esac
say "release dir ok: $RELEASE_DIR (outside the install)"

# ── 3. enroll (attended, idempotent) ───────────────────────────────────────
ENV_FILE="$HOME/.config/vl-updater/env"
already_enrolled=false
if [ -f "$INSTALL_DIR/data/updater/admin.json" ] || { [ -f "$ENV_FILE" ] && grep -Eq '^VL_CUTOVER_TOKEN=.+' "$ENV_FILE"; }; then
  already_enrolled=true
  say "already enrolled (admin.json and/or worker token present) — skipping enroll"
fi

if [ "$already_enrolled" = false ]; then
  if [ "$DRY_RUN" = true ]; then
    say "would run: go run ./cmd/vl-updater-bootstrap --install-dir $INSTALL_DIR enroll $EMAIL (attended)"
  else
    # enroll is INTERACTIVE (it prompts a typed confirmation), so its output
    # goes to the terminal AND a log (tee) — never captured in a way that
    # hides the prompt. It never prints the token, so tee leaks nothing.
    enroll_log="$(mktemp)"
    if ( cd "$REPO_ROOT" && go run ./cmd/vl-updater-bootstrap --install-dir "$INSTALL_DIR" enroll "$EMAIL" 2>&1 | tee "$enroll_log" ); then
      :
    else
      if grep -q "already enrolled" "$enroll_log"; then
        say "already enrolled — continuing"
      else
        rm -f "$enroll_log"
        refuse "enroll failed (see the terminal above)"
      fi
    fi
    rm -f "$enroll_log"
  fi
fi

# ── 4. env file: VL_RELEASE_DIR (token line preserved, never printed) ──────
ENV_DIR="$(dirname "$ENV_FILE")"
if [ "$DRY_RUN" = true ]; then
  say "would: ensure $ENV_FILE (mode 0600) has VL_RELEASE_DIR=$RELEASE_DIR (token line untouched)"
else
  mkdir -p "$ENV_DIR" || refuse "cannot create $ENV_DIR"
  chmod 700 "$ENV_DIR"
  if [ -f "$ENV_FILE" ]; then
    [ "$(stat -c '%a' "$ENV_FILE" 2>/dev/null)" = "600" ] || refuse "$ENV_FILE exists but is not mode 0600"
  fi
  # Replace only the VL_RELEASE_DIR line; every other line (including the
  # token enroll wrote) is preserved byte-identically.
  tmp_env="$(mktemp)"
  if [ -f "$ENV_FILE" ]; then
    grep -v '^VL_RELEASE_DIR=' "$ENV_FILE" > "$tmp_env" || true
  else
    : > "$tmp_env"
  fi
  printf 'VL_RELEASE_DIR=%s\n' "$RELEASE_DIR" >> "$tmp_env"
  install -m 0600 "$tmp_env" "$ENV_FILE" || refuse "cannot write $ENV_FILE"
  rm -f "$tmp_env"
  # The token must still be there (enroll wrote it), and never printed.
  grep -Eq '^VL_CUTOVER_TOKEN=.+' "$ENV_FILE" || refuse "$ENV_FILE is missing VL_CUTOVER_TOKEN after the write"
  say "env file ok: $ENV_FILE (VL_RELEASE_DIR set; token present, not shown)"
fi

# ── 5. VL_UPDATER=1 in the bot's .env (backed up; other lines kept) ────────
DOTENV="$INSTALL_DIR/.env"
if grep -Eq '^VL_UPDATER=1$' "$DOTENV"; then
  say "VL_UPDATER=1 already set in $DOTENV"
else
  if [ "$DRY_RUN" = true ]; then
    say "would: back up $DOTENV and set VL_UPDATER=1 (other lines kept)"
  else
    cp -a "$DOTENV" "$DOTENV.bak-$(date +%Y%m%d-%H%M%S)" || refuse "cannot back up $DOTENV"
    tmp_dotenv="$(mktemp)"
    grep -v '^VL_UPDATER=' "$DOTENV" > "$tmp_dotenv" || true
    printf 'VL_UPDATER=1\n' >> "$tmp_dotenv"
    cp "$tmp_dotenv" "$DOTENV" || refuse "cannot write $DOTENV"
    rm -f "$tmp_dotenv"
    say "VL_UPDATER=1 set in $DOTENV (backup written; restart the bot to read it)"
  fi
fi

# ── 6. install-updater-worker.sh at the CURRENT partner main sha ────────────
# The sha is the partner repo's origin/main tip; VL_UPDATER_PARTNER_SHA pins it
# (tests and re-runs), otherwise git ls-remote resolves it live.
main_sha="${VL_UPDATER_PARTNER_SHA:-}"
if [ -z "$main_sha" ]; then
  main_sha="$(git ls-remote "$REPO_URL" refs/heads/main 2>/dev/null | awk '{print $1}')"
fi
printf '%s' "$main_sha" | grep -Eqx '[0-9a-f]{40}' || refuse "cannot resolve the current partner main sha ($REPO_URL; set VL_UPDATER_PARTNER_SHA to pin one)"
say "current partner main sha: $main_sha"

INSTALLER="$SCRIPT_DIR/install-updater-worker.sh"
[ -f "$INSTALLER" ] || refuse "install-updater-worker.sh not found at $SCRIPT_DIR"
if [ "$DRY_RUN" = true ]; then
  say "would run: $INSTALLER $main_sha (builds ~/bin/vl-updater from that exact clean commit)"
else
  step_run "$INSTALLER" "$main_sha" || refuse "install-updater-worker.sh failed"
fi

# ── 7. start the worker attended ───────────────────────────────────────────
if [ "$DRY_RUN" = true ]; then
  say "would run: systemctl --user start vl-updater"
else
  step_run systemctl --user start vl-updater || refuse "systemctl --user start vl-updater failed"
fi

# ── 8. GET /api/updates → install_enabled + worker_listening ────────────────
check_updates() {
  local token="${VL_ADMIN_TOKEN:-}"
  if [ -z "$token" ]; then
    echo "FAIL — GET /api/updates needs auth: export VL_ADMIN_TOKEN (your logged-in session Bearer token) and re-run, or open Settings → Updates in a browser"
    return 1
  fi
  local resp
  resp="$(curl -sS -m 5 -H "Host: ${API_HOST}:${API_PORT}" -H "X-VL-Update: 1" -H "Authorization: Bearer ${token}" "http://${API_HOST}:${API_PORT}/api/updates" 2>/dev/null || true)"
  if printf '%s' "$resp" | grep -q '"install_enabled":true' && printf '%s' "$resp" | grep -q '"worker_listening":true'; then
    echo "PASS — GET /api/updates answers install_enabled=true and worker_listening=true"
    return 0
  fi
  echo "FAIL — GET /api/updates did not answer install_enabled=true + worker_listening=true"
  echo "  response: ${resp:-<no response>}"
  return 1
}

if [ "$DRY_RUN" = true ]; then
  say "would run: GET /api/updates and check install_enabled + worker_listening"
  say "--dry-run: nothing was written"
  exit 0
fi

if check_updates; then
  say "DONE — one-time setup complete; each release is now one button (plus F5 in NT8 when the job says nt8_updated)"
  exit 0
else
  refuse "final check failed (the setup itself is complete; the check needs a logged-in VL_ADMIN_TOKEN)"
fi
