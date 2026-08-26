# nofx DB backup & restore (C1)

Automatic SQLite backups of `data/data.db`, taken twice daily by a **user** systemd
timer (no root, survives logout via linger).

## What runs

- **Timer:** `~/.config/systemd/user/nofx-backup.timer` → fires **05:00 & 17:30 CT**
  (host is `America/Chicago`, so those calendar times are already CT). `Persistent=true`
  runs a missed backup after a sleep/off window.
- **Service:** `nofx-backup.service` (oneshot) → runs `deploy/nofx-db-backup.sh`.
- **Method:** SQLite **online backup API** via `python3` stdlib (no `sqlite3` CLI
  required; consistent even while the bot is writing), integrity-checked, then gzipped.
- **Location & retention** (under `~/nofx-backups/auto/`):
  - `daily/nofx-YYYY-MM-DD_HHMMSS.db.gz` — every run; newest **14** kept.
  - `weekly/nofx-...W##.db.gz` — one per ISO week; newest **8** kept.

## Install / manage (all no-sudo)

```bash
bash ~/nofx/deploy/install-db-backup.sh          # install + enable + show next run
systemctl --user start   nofx-backup.service      # back up right now
systemctl --user list-timers nofx-backup.timer    # when does it next run?
journalctl --user -u nofx-backup.service -n 50     # last run's log
systemctl --user disable --now nofx-backup.timer   # stop auto-backups
```

## Restore (TESTED read-back — 2026-08-13)

A backup is a complete, standalone SQLite database. To restore:

```bash
# 1. Pick a backup (newest daily shown here).
BK=$(ls -1 ~/nofx-backups/auto/daily/nofx-*.db.gz | sort -r | head -1)

# 2. Decompress to a scratch file and verify it BEFORE touching the live DB.
gunzip -c "$BK" > /tmp/restore.db
python3 -c "import sqlite3;print(sqlite3.connect('/tmp/restore.db').execute('PRAGMA quick_check').fetchone()[0])"
#   → must print: ok

# 3. Stop the bot so nothing is writing the live DB.
kill -9 "$(pgrep -f nofx-bin)"     # systemd Restart=on-failure relaunches it after step 5

# 4. Swap the file in (keep the current one aside first).
mv ~/nofx/data/data.db ~/nofx/data/data.db.pre-restore
cp /tmp/restore.db ~/nofx/data/data.db

# 5. Let systemd relaunch the bot (or start it), then confirm it came up.
journalctl -u nofx -n 20 --no-pager
```

**Verification performed 2026-08-13** on `nofx-2026-08-13_175507.db.gz`: decompressed,
`PRAGMA quick_check = ok`, 19 tables with a schema set **identical** to the live DB,
and core tables read back cleanly (`decision_records` 28042, `trader_positions` 516,
`strategies` 9, `exchanges` 1). The backup is fully restorable.

## Roll back the BINARY (and why you must re-arm `deploy/RELEASE`)

The DB restore above is only half a rollback. If you also go back to an earlier
binary, **`deploy/RELEASE` must be re-armed to the revision you are actually
running** — otherwise the boot assertion sees a mismatch and **REFUSES TRADING**
(`kernel/boot_integrity.go:135-138`: entries blocked, P0 alert, everything else
read-only). A rollback that skips this step comes up looking healthy and silently
takes no trades.

```bash
# 1. Pick the revision to go back to (e.g. the previous release).
cd ~/nofx && git log --oneline -5
TARGET=<sha>

# 2. Build that revision. (Checkout only if you intend to move the working tree;
#    otherwise build from a worktree so main stays where it is.)
git stash list && git status --porcelain      # know what you would disturb
git checkout "$TARGET" -- . 2>/dev/null || git checkout "$TARGET"
go build -o nofx-bin . && echo BUILD OK

# 3. RE-ARM the expected release to MATCH the binary you just built.  ← never skip
git rev-parse HEAD > /tmp/rel
{ grep '^#' deploy/RELEASE; cat /tmp/rel; } > deploy/RELEASE.new && mv deploy/RELEASE.new deploy/RELEASE

# 4. Relaunch and CONFIRM the assertion passed.
kill -9 "$(pgrep -f nofx-bin)"                # systemd Restart=on-failure relaunches
journalctl -u nofx --since '2 min ago' | grep 'BOOT INTEGRITY'
#   → must read "BOOT INTEGRITY OK — rev <X> · expected <X> · goldens PASS"
#   → "TRADING REFUSED" means step 3 was missed or the goldens drifted
```

To disable the assertion deliberately (e.g. while bisecting), leave the value in
`deploy/RELEASE` **blank** — it then logs the revision and never refuses.

Rolling the binary back across a schema migration also needs the matching DB
snapshot from above; restore the DB **first**, then the binary, then re-arm.

## Notes

- Backups are **gzip'd** (~402 MB DB → ~34 MB). Always `gunzip` before opening.
- The `.backup` API copies a transactionally-consistent snapshot, so a backup taken
  mid-trade is still valid — no torn writes.
- These automated backups are **separate** from the ad-hoc `~/nofx-backups/<name>/`
  guarded-write snapshots; both live under `~/nofx-backups/`.
