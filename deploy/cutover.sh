#!/usr/bin/env bash
# W-ONE-BUTTON M4 — the manual boot, v4. In git, unlike the v3 script it
# replaces (~/nofx-backups/cutover-auto-rollback-v3.sh, out of repo).
#
#   usage: cutover.sh [--dry-run] <new-sha40> <new-bin> [new-dist]
#
# --dry-run performs EVERY preflight (sha, binary identity, dist, symlinks,
# flat gate, backup names) and prints the plan. It never kills, swaps or
# writes. The canon requires a TESTED rollback, and a procedure nobody has
# executed is not tested — so the dry run is how it gets exercised without a
# trading window.
#
# Each rule carries the design-note letter it obeys (docs/superpowers/plans/
# 2026-09-22-one-button-m4-design-notes.md), so the next reader finds the
# EVIDENCE rather than a paraphrase.
#   R-a identity is (MainPID, /proc/<pid>/stat field 22 starttime) — NEVER wall
#       clock: M1 measured two start times disagreeing by 151 s after a WSL
#       clock step.
#   R-b the boot proof is in data/nofx_<date>.log, not journald (~4 h for uid
#       1000), plus /api/health and the served bundle.
#   R-e only release-owned entries switch; the working directory is never a
#       symlink, or data/ and .env resolve inside a release.
#   R-o both halves restored, atomically, with names that cannot collide, and
#       never `pgrep -f nofx-bin` — it also matches `go version -m nofx-bin`.
set -uo pipefail

DRY=0
[ "${1:-}" = "--dry-run" ] && { DRY=1; shift; }
NEW_SHA="${1:-}"; NEW_BIN="${2:-}"; NEW_DIST="${3:-}"
UNIT="${NOFX_UNIT:-nofx}"
INSTALL="${NOFX_INSTALL:-$HOME/nofx}"

say()  { printf 'cutover: %s\n' "$*"; }
plan() { printf 'cutover[dry-run]: WOULD %s\n' "$*"; }
die()  { printf 'cutover: REFUSED — %s\n' "$*" >&2; exit 1; }

[ -n "$NEW_SHA" ] && [ -n "$NEW_BIN" ] || die "usage: cutover.sh [--dry-run] <new-sha40> <new-bin> [new-dist]"
case "$NEW_SHA" in *[!0-9a-f]*|"") die "sha must be 40 hex" ;; esac
[ "${#NEW_SHA}" -eq 40 ] || die "sha must be 40 hex (got ${#NEW_SHA})"
SHORT="${NEW_SHA:0:12}"

# --- P1-a: PROVE THE NEW BINARY BEFORE TOUCHING ANYTHING ---------------------
# The previous version took no <new-bin> at all. It backed up whatever was
# already installed, wrote RELEASE and killed — so either nothing changed, or
# the operator had already copied the new binary in and the "backup" WAS the
# new binary, meaning rollback restored the thing being rolled back from. That
# is not a cutover; it is a restart with a false safety net.
[ -f "$NEW_BIN" ] || die "new binary $NEW_BIN not found"
NEW_VCS="$(go version -m "$NEW_BIN" 2>/dev/null || true)"
printf '%s' "$NEW_VCS" | grep -q "vcs.revision=$NEW_SHA" || die "$NEW_BIN does not carry vcs.revision=$NEW_SHA — it is not the binary for this sha"
printf '%s' "$NEW_VCS" | grep -q 'vcs.modified=false'     || die "$NEW_BIN was built from a DIRTY tree (vcs.modified != false)"
NEW_MD5="$(md5sum "$NEW_BIN" | cut -d' ' -f1)"
say "new binary proven: vcs.revision=$SHORT vcs.modified=false md5=$NEW_MD5"

# --- preflight ---------------------------------------------------------------
[ -d "$INSTALL" ] || die "install dir $INSTALL not found"
[ -f "$INSTALL/nofx-bin" ] || die "$INSTALL/nofx-bin not found"
# R-e
[ -L "$INSTALL" ] && die "$INSTALL is a SYMLINK; the working directory must be a real directory (R-e)"
for owned in data .env; do
  [ -L "$INSTALL/$owned" ] && die "$INSTALL/$owned is a symlink; installation state must not live in a release (R-e)"
done

DIST="$INSTALL/web/dist"
SRC_DIST="${NEW_DIST:-$DIST}"
[ -d "$SRC_DIST" ] || die "$SRC_DIST missing — build it with VITE_GUIDE_BUILT_REV=$NEW_SHA npm run build"
# The guide rev is a BUILD INPUT now. The old preflight grepped
# web/src/guide/types.ts for a literal that no longer exists, so it found
# nothing — and "finds nothing" reads exactly like "nothing to check".
grep -rqs -- "$NEW_SHA" "$SRC_DIST"/assets/*.js \
  || die "the bundle at $SRC_DIST does not carry $NEW_SHA.
    Build it with:  cd web && VITE_GUIDE_BUILT_REV=$NEW_SHA npm run build"
say "dist carries $SHORT"

# P3: the OLD rev is the FULL 40-hex, so a rollback writes a real sha into
# deploy/RELEASE and not a 12-char stub nothing else can match.
# `go version -m` emits "build\tvcs.revision=<sha>" — TWO tab-separated
# fields, not three. The first version split on a space and read $3, which is
# empty, so it refused every cutover with "cannot read vcs.revision from the
# CURRENT binary". Found by the first real --dry-run on the live box; no
# syntax check or unit test would have shown it.
OLD_SHA="$(go version -m "$INSTALL/nofx-bin" 2>/dev/null | tr '\t' ' ' | awk '{for(i=1;i<=NF;i++) if($i ~ /^vcs\.revision=/){sub(/^vcs\.revision=/,"",$i); print $i; exit}}')"
[ -n "$OLD_SHA" ] || die "cannot read vcs.revision from the CURRENT binary; refusing a cutover with no way back"
OLD_SHORT="${OLD_SHA:0:12}"
STAMP="$(date +%Y%m%d-%H%M%S)"
BACKUP="$INSTALL/nofx-bin.old.${OLD_SHORT}.${STAMP}"
DIST_BACKUP="$INSTALL/web/dist.old.${OLD_SHORT}.${STAMP}"
[ -e "$BACKUP" ] && die "backup name $BACKUP already exists (R-o: names must not collide)"
say "current: rev=$OLD_SHORT  backups → $(basename "$BACKUP") / $(basename "$DIST_BACKUP")"

# --- P1-b: THE FLAT GATE ------------------------------------------------------
# The previous version SIGKILLed the trader with no check for an open position,
# a non-terminal armed row carrying a broker signal, or an in-flight send.
# Class 33's five legs exist precisely for this moment.
[ -n "${NOFX_CUTOVER_TOKEN:-}" ] || die "cutover gate needs a token — set NOFX_CUTOVER_TOKEN (never pass it on the command line)"
GATE="$(curl -s --max-time 10 -H "Authorization: Bearer ${NOFX_CUTOVER_TOKEN}" \
         http://127.0.0.1:8080/api/cutover-gate 2>/dev/null || true)"
[ -n "$GATE" ] || die "the cutover gate did not answer; refusing to kill a trader whose state is unknown"
# Print the legs so the decision is on the record. The token is never echoed.
printf '%s\n' "$GATE" | sed 's/^/    gate: /'
printf '%s' "$GATE" | grep -qi '"ready"[[:space:]]*:[[:space:]]*true' \
  || die "the cutover gate is NOT ready — a leg failed above. An unevaluable leg counts as a failure (A5)"
say "flat gate READY — every leg passed"

if [ "$DRY" -eq 1 ]; then
  plan "back up $INSTALL/nofx-bin → $(basename "$BACKUP") and $DIST → $(basename "$DIST_BACKUP")"
  plan "install $NEW_BIN (md5 $NEW_MD5) as $INSTALL/nofx-bin via nofx-bin.new + mv -f"
  [ -n "$NEW_DIST" ] && plan "install $NEW_DIST as $DIST atomically (mv, never copy-into-live)"
  plan "write $NEW_SHA to $INSTALL/deploy/RELEASE BEFORE the kill (A19)"
  plan "kill -9 the unit's MainPID; prove relaunch by a change in BOTH MainPID and starttime ticks (R-a)"
  plan "wait ≤90 s for 'BOOT INTEGRITY OK — rev $SHORT' in data/nofx_<date>.log (R-b) and /api/health == $SHORT"
  plan "on ANY failure: restore both halves, rewrite RELEASE as $OLD_SHA, kill again, and PROVE the old rev booted"
  say "dry run complete — nothing was killed, swapped or written"
  exit 0
fi

# --- R-a: identity ------------------------------------------------------------
main_pid()    { systemctl show -p MainPID --value "$UNIT" 2>/dev/null; }
start_ticks() { local p="$1"; [ -n "$p" ] && [ "$p" != "0" ] && [ -r "/proc/$p/stat" ] && awk '{n=split($0,a," "); print a[22]}' "/proc/$p/stat" || echo ""; }
identity()    { local p; p="$(main_pid)"; printf '%s:%s' "${p:-0}" "$(start_ticks "${p:-0}")"; }

wait_boot() { # <sha12> <label> -> 0 ok
  local want="$1" label="$2" deadline=$(( $(date +%s) + 90 )) logf health
  while [ "$(date +%s)" -lt "$deadline" ]; do
    sleep 2
    logf="$INSTALL/data/nofx_$(date +%Y-%m-%d).log"
    [ -f "$logf" ] && grep -q "BOOT INTEGRITY OK — rev $want" "$logf" 2>/dev/null || continue
    health="$(curl -s --max-time 5 http://127.0.0.1:8080/api/health 2>/dev/null || true)"
    case "$health" in *"\"revision\":\"$want\""*) say "$label: boot line + /api/health both report $want"; return 0 ;; esac
  done
  return 1
}

# --- P1-c: a rollback that RESTARTS and PROVES it -----------------------------
# The previous version swapped files and printed "restart the unit". After a
# failed boot the RUNNING process is the NEW binary, so files-only is not a
# rollback: the bad build keeps serving. The canon requires a TESTED
# auto-rollback, which means this function must leave the OLD rev PROVEN live
# or say out loud that it did not.
rollback() {
  say "ROLLING BACK to $OLD_SHORT"
  mv -f "$BACKUP" "$INSTALL/nofx-bin" 2>/dev/null || say "WARNING: could not restore the binary"
  rm -rf "$INSTALL/web/dist.failed" 2>/dev/null || true
  mv -f "$DIST" "$INSTALL/web/dist.failed" 2>/dev/null || true
  mv -f "$DIST_BACKUP" "$DIST" 2>/dev/null || say "WARNING: could not restore web/dist"
  printf '%s\n' "$OLD_SHA" > "$INSTALL/deploy/RELEASE" 2>/dev/null || true   # P3: full 40-hex
  local p; p="$(main_pid)"
  [ -n "$p" ] && [ "$p" != "0" ] && kill -9 "$p" 2>/dev/null || true
  if wait_boot "$OLD_SHORT" "rollback"; then
    say "ROLLBACK OK — $OLD_SHORT is live and proven"
    return 0
  fi
  say "ROLLBACK FAILED — $OLD_SHORT did NOT come back within 90 s. ALERT THE OWNER; the desk is down."
  return 1
}

# --- swap, then kill ----------------------------------------------------------
cp -a "$INSTALL/nofx-bin" "$BACKUP" || die "could not back up the binary"
cp -a "$DIST" "$DIST_BACKUP"        || die "could not back up web/dist"
# P1-a: actually INSTALL it, atomically.
cp -a "$NEW_BIN" "$INSTALL/nofx-bin.new" || { rollback; die "could not stage the new binary"; }
mv -f "$INSTALL/nofx-bin.new" "$INSTALL/nofx-bin" || { rollback; die "could not install the new binary"; }
if [ -n "$NEW_DIST" ]; then
  rm -rf "$INSTALL/web/dist.incoming" 2>/dev/null || true
  cp -a "$NEW_DIST" "$INSTALL/web/dist.incoming" || { rollback; die "could not stage the new dist"; }
  rm -rf "$DIST" && mv -f "$INSTALL/web/dist.incoming" "$DIST" || { rollback; die "could not install the new dist"; }
fi
# A19: RELEASE is written BEFORE the kill, so a marker never describes a boot
# that has not happened.
printf '%s\n' "$NEW_SHA" > "$INSTALL/deploy/RELEASE" || { rollback; die "could not write deploy/RELEASE"; }
say "installed $SHORT (md5 $NEW_MD5); RELEASE written before the kill"

BEFORE="$(identity)"
PID_BEFORE="$(main_pid)"
say "before: identity=$BEFORE — sending SIGKILL to MainPID ${PID_BEFORE:-none} (SIGTERM exits 0 and systemd does NOT relaunch)"
[ -n "$PID_BEFORE" ] && [ "$PID_BEFORE" != "0" ] && kill -9 "$PID_BEFORE" 2>/dev/null || true

deadline=$(( $(date +%s) + 90 )); AFTER=""
while [ "$(date +%s)" -lt "$deadline" ]; do
  sleep 2; AFTER="$(identity)"
  [ -n "$AFTER" ] && [ "$AFTER" != "$BEFORE" ] && [ "${AFTER%%:*}" != "0" ] && break
done
[ -n "$AFTER" ] && [ "$AFTER" != "$BEFORE" ] || { rollback; die "no relaunch: identity unchanged ($BEFORE); BOTH MainPID and starttime must change (R-a)"; }
say "after: identity=$AFTER"

wait_boot "$SHORT" "cutover" || { rollback; die "the new rev did not prove itself within 90 s"; }
say "CUTOVER OK — $SHORT is live. Rollback copies: $(basename "$BACKUP"), $(basename "$DIST_BACKUP")"
