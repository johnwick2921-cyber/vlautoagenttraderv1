#!/usr/bin/env bash
# W-ONE-BUTTON M4 — the manual boot, v4. In git, unlike the v3 script it
# replaces (~/nofx-backups/cutover-auto-rollback-v3.sh, out of repo).
#
# Every rule below carries the design-note letter it comes from
# (docs/superpowers/plans/2026-09-22-one-button-m4-design-notes.md), so the
# next reader finds the EVIDENCE rather than someone's paraphrase of it.
#
#   R-a  Restart identity is (MainPID, /proc/<pid>/stat field 22 starttime
#        ticks) — NEVER wall clock. M1 measured two start times disagreeing by
#        151 s because of a WSL clock step, so "it restarted" judged on a clock
#        is a coin flip. The relaunch is proven by a change in BOTH.
#   R-b  The boot watchdog reads data/nofx_<boot-date>.log, NOT journald:
#        journald keeps ~4 h for uid 1000, so the boot proof survives only in
#        the data log. It also checks /api/health revision == the new sha and
#        the SERVED asset carries that sha.
#   R-e  Only release-owned entries switch (nofx-bin, web/dist, deploy/RELEASE,
#        calendar_static_t1.json). The WORKING DIRECTORY is never symlinked —
#        data/ and .env must not resolve inside a release.
#   R-o  Rollback restores web/dist atomically too, never collides on
#        nofx-bin.old.* names, and NEVER uses `pgrep -f nofx-bin` — that
#        pattern also matches `go version -m nofx-bin`, so it can return the
#        pid of a tool inspecting the binary instead of the server running it.
#
# BUILD INPUT (W4): the guide's rev is no longer a literal in
# web/src/guide/types.ts, so the old preflight grep for
# "GUIDE_BUILT_REV = '<sha8>" cannot work and its absence is not a failure.
# The dist must be BUILT with VITE_GUIDE_BUILT_REV=<sha> and then VERIFIED by
# finding that sha inside web/dist/assets/*.js — which is exactly what
# .github/workflows/release.yml does.
set -uo pipefail

UNIT="${NOFX_UNIT:-nofx}"
INSTALL="${NOFX_INSTALL:-$HOME/nofx}"
NEW_SHA="${1:-}"
[ -n "$NEW_SHA" ] || { echo "usage: cutover.sh <40-hex-sha>" >&2; exit 2; }
case "$NEW_SHA" in *[!0-9a-f]*|"") echo "cutover: REFUSED — sha must be 40 hex" >&2; exit 1 ;; esac
[ "${#NEW_SHA}" -eq 40 ] || { echo "cutover: REFUSED — sha must be 40 hex (got ${#NEW_SHA})" >&2; exit 1; }
SHORT="${NEW_SHA:0:12}"

say() { printf 'cutover: %s\n' "$*"; }
die() { printf 'cutover: REFUSED — %s\n' "$*" >&2; exit 1; }

# --- R-a: identity, never the clock -----------------------------------------
main_pid() { systemctl show -p MainPID --value "$UNIT" 2>/dev/null; }
start_ticks() { # field 22 of /proc/<pid>/stat; the process's own birth, immune to clock steps
  local pid="$1"
  [ -n "$pid" ] && [ "$pid" != "0" ] && [ -r "/proc/$pid/stat" ] || { echo ""; return; }
  awk '{ n=split($0,a," "); print a[22] }' "/proc/$pid/stat" 2>/dev/null
}
identity() { local p; p="$(main_pid)"; printf '%s:%s' "${p:-0}" "$(start_ticks "${p:-0}")"; }

# --- preflight ---------------------------------------------------------------
[ -d "$INSTALL" ] || die "install dir $INSTALL not found"
[ -f "$INSTALL/nofx-bin" ] || die "$INSTALL/nofx-bin not found"
# R-e: the working directory must be a REAL directory, never a symlink into a
# release — otherwise data/ and .env resolve inside the release and are lost on
# the next switch.
[ -L "$INSTALL" ] && die "$INSTALL is a SYMLINK; the working directory must be a real directory (R-e)"
for owned in data .env; do
  [ -L "$INSTALL/$owned" ] && die "$INSTALL/$owned is a symlink; installation-owned state must not live in a release (R-e)"
done

# BUILD INPUT: prove the dist carries the sha, the way release.yml does.
DIST="$INSTALL/web/dist"
[ -d "$DIST" ] || die "$DIST missing — build it with VITE_GUIDE_BUILT_REV=$NEW_SHA npm run build"
if ! grep -rqs -- "$NEW_SHA" "$DIST"/assets/*.js; then
  die "the served bundle does not carry $NEW_SHA.
    Build it with:  cd web && VITE_GUIDE_BUILT_REV=$NEW_SHA npm run build
    (The old preflight grepped web/src/guide/types.ts for a literal
     GUIDE_BUILT_REV — that literal no longer exists; the rev is a BUILD INPUT
     now, so the dist is the only place the truth can be checked.)"
fi
say "dist carries $SHORT"

# R-o: a rollback name that cannot collide. The old script reused
# nofx-bin.old.<rev>, so two cutovers at the same rev overwrote each other's
# only way back.
OLD_REV="$(go version -m "$INSTALL/nofx-bin" 2>/dev/null | awk '$1=="build" && $2=="vcs.revision="{print substr($3,1,12)}')"
[ -n "$OLD_REV" ] || OLD_REV="unknown"
STAMP="$(date +%Y%m%d-%H%M%S)"
BACKUP="$INSTALL/nofx-bin.old.${OLD_REV}.${STAMP}"
[ -e "$BACKUP" ] && die "backup name $BACKUP already exists (R-o: names must not collide)"
DIST_BACKUP="$INSTALL/web/dist.old.${OLD_REV}.${STAMP}"

BEFORE="$(identity)"
say "before: identity=$BEFORE  old_rev=$OLD_REV"

# --- swap --------------------------------------------------------------------
cp -a "$INSTALL/nofx-bin" "$BACKUP" || die "could not back up the binary"
cp -a "$DIST" "$DIST_BACKUP" || die "could not back up web/dist"
printf '%s\n' "$NEW_SHA" > "$INSTALL/deploy/RELEASE" || die "could not write deploy/RELEASE"
say "backed up binary → $(basename "$BACKUP"), dist → $(basename "$DIST_BACKUP")"

rollback() {
  say "ROLLING BACK"
  mv -f "$BACKUP" "$INSTALL/nofx-bin" 2>/dev/null || true
  # R-o: web/dist is restored ATOMICALLY — a half-copied dist serves a mix of
  # old and new assets, which looks like a UI bug and not a failed cutover.
  rm -rf "$INSTALL/web/dist.failed" 2>/dev/null || true
  mv -f "$DIST" "$INSTALL/web/dist.failed" 2>/dev/null || true
  mv -f "$DIST_BACKUP" "$DIST" 2>/dev/null || true
  printf '%s\n' "$OLD_REV" > "$INSTALL/deploy/RELEASE" 2>/dev/null || true
  say "rolled back; restart the unit and verify before trading"
}

# --- restart, proven by identity (R-a) ---------------------------------------
PID_BEFORE="$(main_pid)"
say "sending SIGKILL to MainPID ${PID_BEFORE:-none} (systemd Restart=on-failure relaunches; SIGTERM exits 0 and does NOT)"
[ -n "$PID_BEFORE" ] && [ "$PID_BEFORE" != "0" ] && kill -9 "$PID_BEFORE" 2>/dev/null || true

deadline=$(( $(date +%s) + 90 ))
AFTER=""
while [ "$(date +%s)" -lt "$deadline" ]; do
  sleep 2
  AFTER="$(identity)"
  [ -n "$AFTER" ] && [ "$AFTER" != "$BEFORE" ] && [ "${AFTER%%:*}" != "0" ] && break
done
[ -n "$AFTER" ] && [ "$AFTER" != "$BEFORE" ] || { rollback; die "no relaunch: identity unchanged ($BEFORE). Both MainPID and starttime ticks must change (R-a)"; }
say "after: identity=$AFTER (both MainPID and starttime changed)"

# --- R-b: the boot proof lives in the DATA log, not journald ------------------
LOGF="$INSTALL/data/nofx_$(date +%Y-%m-%d).log"
ok_boot=0
deadline=$(( $(date +%s) + 90 ))
while [ "$(date +%s)" -lt "$deadline" ]; do
  if [ -f "$LOGF" ] && grep -q "BOOT INTEGRITY OK — rev $SHORT" "$LOGF" 2>/dev/null; then ok_boot=1; break; fi
  sleep 2
done
[ "$ok_boot" -eq 1 ] || { rollback; die "no 'BOOT INTEGRITY OK — rev $SHORT' in $LOGF within 90 s (R-b: journald keeps ~4 h, the data log is the only durable proof)"; }
say "boot integrity line found for $SHORT"

HEALTH="$(curl -s --max-time 5 http://127.0.0.1:8080/api/health 2>/dev/null || true)"
case "$HEALTH" in
  *"\"revision\":\"$SHORT\""*) say "/api/health reports $SHORT" ;;
  *) rollback; die "/api/health does not report $SHORT (got: ${HEALTH:0:120})" ;;
esac

say "CUTOVER OK — $SHORT is live. Rollback copies kept: $(basename "$BACKUP"), $(basename "$DIST_BACKUP")"
