#!/usr/bin/env bash
# new-install.sh — ONE installer for a brand-new person: a fresh Windows+WSL2
# machine with NinjaTrader 8 installed, to a running SIM bot + the one-button
# updater. Idempotent, attended, no sudo (it PRINTS the one sudo line it cannot
# run itself). --dry-run prints every action and writes NOTHING.
#
# usage: new-install.sh [--dry-run] [--install-dir <dir>]
#
# Steps (each printed; any failure REFUSES with a reason and writes nothing):
#   1. WSL check (mirrored networking is the human's Windows setting).
#   2. toolchain: go ≥ 1.26.9 (go.mod), node ≥ 18 (web/package.json vite 6).
#   3. build the bot (go build -o vl-bin .) and the web (npm install +
#      VITE_GUIDE_BUILT_REV=<checked-out sha> npm run build).
#   4. .env: copy .env.example, then ensure the required keys WITHOUT
#      overwriting existing values — JWT_SECRET generated (openssl rand -base64
#      64, NEVER printed), NT_TRANSPORT=tcp, TRADING_MODE=futures.
#   5. print the exact sudo autostart line (the human runs it).
#   6. NT8 AddOn: copy ninjascript/*.cs into the Windows AddOns folder, print
#      the F5 + restart instruction, and WAIT for the boot line
#      "nt8 addon: … match=yes" (bounded; Ctrl-C to re-run later — idempotent).
#   7. call deploy/partner-first-time-setup.sh (the one-button updater; pass
#      --email <you@example.com> to have it run automatically).
#   8. final health check: GET /api/health + the match=yes line → PASS/FAIL.
#
# Test seams (documented, not production knobs): VL_NEW_INSTALL_WSL (1/0 forces
# the WSL check), VL_NT8_ADDONS_DIR (the Windows AddOns folder — default: detect
# under /mnt/c/Users), VL_NEW_INSTALL_SKIP_BUILD (1 skips go/npm builds), and
# VL_NEW_INSTALL_FIRST_TIME (path to partner-first-time-setup.sh — default the
# sibling in this repo).
#
# SIM-only, by the laws: live/funded accounts stay blocked (IsAccountTradeable);
# this script never weakens that and never touches ~/vl (it runs from its own
# checkout).
set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(dirname "$SCRIPT_DIR")"
INSTALL_DIR="${VL_NEW_INSTALL_DIR:-$HOME/vl}"

DRY_RUN=false
EMAIL="${VL_NEW_INSTALL_EMAIL:-}"
while [ $# -gt 0 ]; do
  case "$1" in
    --dry-run) DRY_RUN=true ;;
    --install-dir) [ $# -ge 2 ] || { echo "usage: new-install.sh [--dry-run] [--install-dir <dir>] [--email <email>]" >&2; exit 2; }; INSTALL_DIR="$2"; shift ;;
    --email) [ $# -ge 2 ] || { echo "usage: new-install.sh [--dry-run] [--install-dir <dir>] [--email <email>]" >&2; exit 2; }; EMAIL="$2"; shift ;;
    -h|--help) echo "usage: new-install.sh [--dry-run] [--install-dir <dir>] [--email <email>]" >&2; exit 2 ;;
    -*) echo "unknown flag: $1" >&2; exit 2 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
  shift
done

[ -n "${HOME:-}" ] || { echo "new-install: REFUSED — no HOME" >&2; exit 2; }

say()    { echo "new-install: $*"; }
refuse() { echo "new-install: REFUSED — $*" >&2; exit 2; }
run() {
  if [ "$DRY_RUN" = true ]; then say "would run: $*"; return 0; fi
  say "run: $*"
  "$@"
}

# ── 1. WSL ─────────────────────────────────────────────────────────────────
is_wsl() {
  [ "${VL_NEW_INSTALL_WSL:-}" = "1" ] && return 0
  [ "${VL_NEW_INSTALL_WSL:-}" = "0" ] && return 1
  grep -qiE 'microsoft|wsl' /proc/version 2>/dev/null
}
is_wsl || refuse "this must run inside WSL2 (a Windows host with NT8); the AddOn connects over 127.0.0.1 which only crosses the boundary under WSL"
say "WSL ok"

# ── 2. toolchain ──────────────────────────────────────────────────────────
command -v go >/dev/null 2>&1 || refuse "go is not on PATH (go.mod needs go ≥ 1.26.9)"
command -v node >/dev/null 2>&1 || refuse "node is not on PATH (web needs node ≥ 18)"
command -v npm >/dev/null 2>&1 || refuse "npm is not on PATH"
command -v git >/dev/null 2>&1 || refuse "git is not on PATH"

# go version: "go version go1.26.9 linux/amd64" → major/minor/patch.
go_ver="$(go version 2>/dev/null | awk '{for(i=1;i<=NF;i++) if($i ~ /^go[0-9]/){print $i}}')"
go_ver="${go_ver#go}"
go_major="${go_ver%%.*}"; rest="${go_ver#*.}"; go_minor="${rest%%.*}"
case "$go_major" in ''|*[!0-9]*) refuse "cannot parse go version: $(go version 2>&1)" ;; esac
case "$go_minor" in ''|*[!0-9]*) refuse "cannot parse go version: $(go version 2>&1)" ;; esac
if [ "$go_major" -lt 1 ] || { [ "$go_major" -eq 1 ] && [ "$go_minor" -lt 26 ]; }; then
  refuse "go must be ≥ 1.26.9 (go.mod), got go${go_ver}"
fi
say "go ok: go${go_ver}"

node_ver="$(node --version 2>/dev/null | tr -d 'v')"
node_major="${node_ver%%.*}"
case "$node_major" in ''|*[!0-9]*) refuse "cannot parse node version: $(node --version 2>&1)" ;; esac
if [ "$node_major" -lt 18 ]; then
  refuse "node must be ≥ 18 (vite 6), got $(node --version 2>&1)"
fi
say "node ok: v${node_ver}"

# ── 3. build (from the checked-out sha) ────────────────────────────────────
if [ "${VL_NEW_INSTALL_SKIP_BUILD:-}" = "1" ]; then
  say "build skipped (VL_NEW_INSTALL_SKIP_BUILD=1)"
else
  run sh -c 'cd "$1" && go build -o vl-bin .' _ "$REPO_ROOT" || refuse "go build failed"
  run sh -c 'cd "$1/web" && npm install && VITE_GUIDE_BUILT_REV=$(git -C "$1" rev-parse HEAD) npm run build' _ "$REPO_ROOT" || refuse "web build failed"
fi

# ── 4. .env ────────────────────────────────────────────────────────────────
DOTENV="$INSTALL_DIR/.env"
TEMPLATE="$REPO_ROOT/.env.example"
[ -f "$TEMPLATE" ] || refuse ".env.example not found at $REPO_ROOT"
[ -d "$INSTALL_DIR" ] || refuse "install dir does not exist: $INSTALL_DIR"

# ensure_key: append KEY=VALUE only when KEY is not already present — an
# existing key is NEVER overwritten.
ensure_key() {
  local key="$1" value="$2" file="$3"
  if grep -Eq "^${key}=" "$file"; then
    say ".env: $key already present (left untouched)"
    return 0
  fi
  if [ "$DRY_RUN" = true ]; then
    say "would write to .env: $key (value not shown)"
    return 0
  fi
  printf '%s=%s\n' "$key" "$value" >> "$file"
  say ".env: $key set"
}

if [ "$DRY_RUN" = true ]; then
  say "would: create $DOTENV from .env.example, then ensure JWT_SECRET/NT_TRANSPORT/TRADING_MODE without overwriting existing keys"
else
  # Idempotent: when every required key is already present, nothing is backed
  # up and nothing is written.
  missing=""
  [ -f "$DOTENV" ] || missing="create"
  [ -n "$missing" ] || grep -Eq '^NT_TRANSPORT=' "$DOTENV" || missing="NT_TRANSPORT"
  [ -n "$missing" ] || grep -Eq '^TRADING_MODE=' "$DOTENV" || missing="TRADING_MODE"
  [ -n "$missing" ] || grep -Eq '^JWT_SECRET=.+' "$DOTENV" || missing="JWT_SECRET"

  if [ -z "$missing" ]; then
    say ".env ok: $DOTENV already complete (no change)"
  else
    if [ -f "$DOTENV" ]; then
      cp -a "$DOTENV" "$DOTENV.bak-$(date +%Y%m%d-%H%M%S)" || refuse "cannot back up $DOTENV"
    else
      cp "$TEMPLATE" "$DOTENV" || refuse "cannot create $DOTENV from $TEMPLATE"
    fi
    ensure_key "NT_TRANSPORT" "tcp" "$DOTENV"
    ensure_key "TRADING_MODE" "futures" "$DOTENV"
    if ! grep -Eq '^JWT_SECRET=.+' "$DOTENV"; then
      # Generate, never print. The value goes straight into the file.
      secret="$(openssl rand -base64 64 2>/dev/null | tr -d '\n')" || refuse "openssl rand failed (cannot generate JWT_SECRET)"
      ensure_key "JWT_SECRET" "$secret" "$DOTENV"
    fi
    say ".env ok: $DOTENV (required keys present; JWT_SECRET generated, never printed)"
  fi
fi

# ── 5. autostart (the ONE sudo step — printed, not run) ────────────────────
say "autostart: run this ONE sudo command yourself (it renders + installs the systemd units, nvm-aware):"
say "  sudo bash $REPO_ROOT/deploy/install-autostart.sh && sudo loginctl enable-linger $USER"

# ── 6. NT8 AddOn ───────────────────────────────────────────────────────────
ADDONS="${VL_NT8_ADDONS_DIR:-}"
if [ -z "$ADDONS" ]; then
  ADDONS="$(find /mnt/c/Users -maxdepth 4 -type d -path '*/Documents/NinjaTrader 8/bin/Custom/AddOns' 2>/dev/null | head -1 || true)"
fi
[ -n "$ADDONS" ] || refuse "NinjaTrader 8 AddOns folder not found (looked under /mnt/c/Users/*/Documents/NinjaTrader 8/bin/Custom/AddOns; set VL_NT8_ADDONS_DIR if it is elsewhere)"
[ -d "$ADDONS" ] || refuse "the NT8 AddOns folder does not exist: $ADDONS"
say "NT8 AddOns folder: $ADDONS"

if [ "$DRY_RUN" = true ]; then
  say "would copy ninjascript/*.cs → $ADDONS"
  say "would then print the F5 + full-restart instruction, WAIT for 'nt8 addon: … match=yes' in $INSTALL_DIR/data/vl_*.log, then call the one-button updater setup"
else
  run cp "$REPO_ROOT"/ninjascript/*.cs "$ADDONS/" || refuse "copy of ninjascript/*.cs failed"
  say "AddOn files copied. NOW: press F5 in NinjaTrader 8 (compile), then do ONE full NT8 restart."
  say "then start the bot with: cd $INSTALL_DIR && ./vl-bin"
  say "waiting for the boot line 'nt8 addon: … match=yes' in $INSTALL_DIR/data/vl_*.log (Ctrl-C to stop and re-run later — this script is idempotent)."
  wait_secs="${VL_NEW_INSTALL_WAIT_SECONDS:-900}"
  deadline=$(( $(date +%s) + wait_secs ))
  found=0
  while [ "$(date +%s)" -lt "$deadline" ]; do
    if grep -q 'match=yes' "$INSTALL_DIR"/data/vl_*.log 2>/dev/null; then
      found=1
      break
    fi
    sleep 5
  done
  if [ "$found" != 1 ]; then
    say "FAIL — no 'nt8 addon: … match=yes' line after ${wait_secs}s (did you press F5, fully restart NT8, and start ./vl-bin?). Re-run this script after the line appears — it is idempotent."
  else
    say "AddOn match confirmed: match=yes."
  fi
fi

# ── 7. one-button updater ─────────────────────────────────────────────────
FIRST_TIME="${VL_NEW_INSTALL_FIRST_TIME:-$SCRIPT_DIR/partner-first-time-setup.sh}"
if [ "$DRY_RUN" = true ]; then
  if [ -f "$FIRST_TIME" ]; then
    if [ -n "$EMAIL" ]; then
      say "would call: $FIRST_TIME $EMAIL (the one-button updater setup)"
    else
      say "would call: $FIRST_TIME <your email> (the one-button updater setup; pass --email <you@example.com> to have me run it)"
    fi
  else
    say "note — partner-first-time-setup.sh is not in this tree yet (PR #49); the one-button updater setup runs after that merges: bash $SCRIPT_DIR/partner-first-time-setup.sh <your email>"
  fi
else
  if [ -f "$FIRST_TIME" ]; then
    if [ -n "$EMAIL" ]; then
      run bash "$FIRST_TIME" "$EMAIL" || say "the one-button updater setup reported a problem (see its output above)"
    else
      say "one-button updater setup: run it yourself (I don't know your email): bash $FIRST_TIME <your account email>"
    fi
  else
    say "note — partner-first-time-setup.sh is not in this tree yet (PR #49); the one-button updater setup runs after that merges: bash $SCRIPT_DIR/partner-first-time-setup.sh <your email>"
  fi
fi

# ── 8. final health check ─────────────────────────────────────────────────
API_PORT="${VL_API_PORT:-8080}"
if [ "$DRY_RUN" = true ]; then
  say "would: GET http://127.0.0.1:$API_PORT/api/health and grep the log for 'nt8 addon: … match=yes'"
  say "--dry-run: nothing was written"
  exit 0
fi
health="$(curl -sS -m 5 "http://127.0.0.1:$API_PORT/api/health" 2>/dev/null || true)"
if [ -n "$health" ]; then
  say "PASS — /api/health answers (final check: confirm the bot log shows 'nt8 addon: … match=yes' and 'hello handshake OK')"
else
  say "FAIL — /api/health did not answer on :$API_PORT (start ./vl-bin, complete the first-account setup, then re-check)"
fi
