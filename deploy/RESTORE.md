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

**If the boot log's `🗄 bars` key line says `migrated`, run `deploy/bars-key-rollback.sh`
BEFORE starting an older binary** (W-BARS-CONTRACT-KEY, 2026-09-18 — an older binary
cannot write the migrated `bars` table; the section below has the detail).

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

## Roll back the bars CONTRACT-KEY migration (W-BARS-CONTRACT-KEY, 2026-09-18)

The first boot of a binary at/after this wave moves `bars` from
`PRIMARY KEY (symbol, tf, open_time_ms)` to `(symbol, tf, contract, open_time_ms)`.
It is a guarded write, done by code at boot (`store/bar_contract_key.go`):

1. a whole-database backup **before anything else** — `VACUUM INTO`
   `~/nofx-backups/pre-bars-key-<YYYYMMDD-HHMMSS>.db`, verified by `bars` row count;
   if it cannot be written the migration is **refused** and the bot runs on the old key
   (boot line `🗄 bars: migration FAILED — …; old table intact`);
2. `bars_v2` created on the new key, `INSERT … SELECT` of every row, then
   `bars` → `bars_pre_contract_key_<YYYY-MM-DD>` and `bars_v2` → `bars`, in **one
   transaction** (any error rolls back; the old table is never touched);
3. boot line `🗄 bars: key migrated to (symbol,tf,contract,open_time_ms) — rows=<n>
   backup=<path> old_table=<name>` (values read back). Later boots print
   `🗄 bars: key=(symbol,tf,contract,open_time_ms) (migrated <date>) rows=<n>` and do nothing.

Measured on a copy of the live DB (1,906,992 rows, 1.39 GB) 2026-09-18 01:00 CT:
migration 7.7 s including the backup; second boot 0.33 s.

**A PRE-MIGRATION BINARY MUST NOT BOOT ON THE MIGRATED TABLE** (tested,
`store/bar_contract_key_test.go TestBarsKeyOldBinaryStatementsOnTheMigratedTable`):
its bar upsert names `ON CONFLICT(symbol, tf, open_time_ms)`, which is no longer a
unique constraint, and SQLite refuses every write (`ON CONFLICT clause does not
match any PRIMARY KEY or UNIQUE constraint`) — reads work, **persistence dies
silently except for a `bars: persist … failed` WARN per batch**. Its Migrate does
NOT run its destructive 2026-08-27 dedupe block (the name-only unique-index check
is satisfied by the renamed table's index, which the migration keeps on purpose).

### Option A — rename back (keeps the DB, loses bars written after the migration)

Scripted: `deploy/bars-key-rollback.sh [--force] [--db PATH]` (rc 0 done · rc 4 nothing
to do · rc 2 bot running · rc 3 unsafe state) — discovers the old table, refuses while
`nofx-bin` runs unless `--force`, takes a `VACUUM INTO` backup, runs
`deploy/bars-key-rollback.sql` (the rename pair in one transaction, new-shape indexes
dropped from the parked copy so a later re-migration can recreate them), prints the counts.
The CTO's unattended cutover calls it on a binary rollback. By hand:

```bash
# 0. Stop the bot (nothing may write during the swap).
kill -9 "$(pgrep -f nofx-bin)"     # systemd relaunches it — do this only with the old binary installed AND step 2 done
# 1. Which tables exist?
sqlite3 ~/nofx/data/data.db "SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'bars%';"
# 2. Swap: the migrated table aside, the pre-migration table back under its name.
#    The old table still carries idx_bars_sym_tf_time_unique / idx_bars_contract / idx_bars_source,
#    so the old binary's Migrate finds its unique index and is a no-op.
OLD=$(sqlite3 ~/nofx/data/data.db "SELECT name FROM sqlite_master WHERE type='table' AND name LIKE 'bars_pre_contract_key_%' ORDER BY name DESC LIMIT 1")
sqlite3 ~/nofx/data/data.db "BEGIN; ALTER TABLE bars RENAME TO bars_contract_key_v2; ALTER TABLE \"$OLD\" RENAME TO bars; COMMIT;"
# 3. (optional) carry bars written AFTER the migration back onto the old key.
#    On the old key a minute held by two contracts collapses to ONE row (first wins) — this is the
#    exact loss the wave removed; accept it or skip this step.
sqlite3 ~/nofx/data/data.db "INSERT OR IGNORE INTO bars(symbol,tf,open_time_ms,o,h,l,c,v,convention,contract,source)
  SELECT symbol,tf,open_time_ms,o,h,l,c,v,convention,contract,source FROM bars_contract_key_v2;"
# 4. Verify, then boot the old binary (re-arm deploy/RELEASE — see the binary rollback above).
sqlite3 ~/nofx/data/data.db "SELECT name FROM pragma_table_info('bars') WHERE pk>0 ORDER BY pk;"   # → symbol tf open_time_ms
sqlite3 ~/nofx/data/data.db "SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND tbl_name='bars' AND name='idx_bars_sym_tf_time_unique';"  # → 1
```

To re-migrate later, the new binary refuses while a `bars_pre_contract_key_<today>`
table exists (it never overwrites a backup table): rename or drop
`bars_contract_key_v2` and the stale `bars_pre_contract_key_*` first, deliberately.

### Option B — restore the whole-database backup (loses EVERYTHING written after it)

```bash
BK=$(ls -1 ~/nofx-backups/pre-bars-key-*.db | sort -r | head -1)
python3 -c "import sqlite3,sys;print(sqlite3.connect(sys.argv[1]).execute('PRAGMA quick_check').fetchone()[0])" "$BK"   # → ok
kill -9 "$(pgrep -f nofx-bin)"
mv ~/nofx/data/data.db ~/nofx/data/data.db.pre-restore
cp "$BK" ~/nofx/data/data.db
# then the binary rollback + RELEASE re-arm above
```

Prefer A: it is bars-only. B rewinds decision_records, plans, positions — every table.

### Probes (also CLASS entry in docs/superpowers/AUDIT-CHECKLIST.md)

```sql
-- which key is live
SELECT name FROM pragma_table_info('bars') WHERE pk>0 ORDER BY pk;
-- roll overlaps: minutes held by two contracts (expected >0 after a roll on the new key; always 0 on the old key)
SELECT symbol,tf,open_time_ms,COUNT(DISTINCT contract) FROM bars GROUP BY 1,2,3 HAVING COUNT(*)>1;
-- the old-key loss, measured on the backup table: rows of the newer contract that the old key had NO slot for
SELECT COUNT(*) FROM bars b WHERE b.contract='MNQ 12-26' AND b.symbol='MNQ' AND b.tf='1m'
  AND EXISTS (SELECT 1 FROM bars a WHERE a.symbol=b.symbol AND a.tf=b.tf AND a.open_time_ms=b.open_time_ms AND a.contract='MNQ 09-26');
```

## Notes

- Backups are **gzip'd** (~402 MB DB → ~34 MB). Always `gunzip` before opening.
- The `.backup` API copies a transactionally-consistent snapshot, so a backup taken
  mid-trade is still valid — no torn writes.
- These automated backups are **separate** from the ad-hoc `~/nofx-backups/<name>/`
  guarded-write snapshots; both live under `~/nofx-backups/`.

## Manual boot after W-ONE-BUTTON M4 — the guide rev is a BUILD INPUT

**What changed and why the old preflight silently stops working.** Until this
wave `web/src/guide/types.ts` carried a literal:

```
export const GUIDE_BUILT_REV = '<40-hex sha>'
```

and the boot procedure grepped that line to check the guide matched the binary.
That literal **no longer exists**. The rev now arrives at build time, so the old
grep finds nothing — and "finds nothing" reads exactly like "nothing to check".
It is not a failure; it is a check that has quietly stopped checking.

**Every manual boot now does two things instead:**

```
# 1. BUILD with the rev of the commit you are booting
cd web && VITE_GUIDE_BUILT_REV=<40-hex merge sha> npm run build

# 2. VERIFY the served bundle actually carries it
grep -rq -- '<40-hex merge sha>' web/dist/assets/*.js \
  || { echo 'the dist does not carry the rev — do NOT boot'; exit 1; }
```

A production build with the variable missing or not 40-hex **fails at build
time** rather than shipping a guide whose claim about the running binary cannot
be checked. `deploy/cutover.sh` performs step 2 as a preflight and refuses the
cutover if it fails; `.github/workflows/release.yml` does exactly the same for a
tagged release, and additionally proves the negative — `VITE_GUIDE_BUILT_REV=
npm run build` must FAIL before the real build runs.

**Out-of-repo script:** the CTO's `~/nofx-backups/cutover-auto-rollback-v3.sh`
still greps the old literal. It must be edited the same way before the next
boot, or it will report a green preflight for a check that no longer exists.
`deploy/cutover.sh` is its in-git successor and already does this.

## Rollback copies

`deploy/cutover.sh` keeps both halves of the previous release, named so they
cannot collide (R-o — the old script reused `nofx-bin.old.<rev>`, so two
cutovers at the same rev overwrote the only way back):

```
nofx-bin.old.<rev12>.<YYYYmmdd-HHMMSS>
web/dist.old.<rev12>.<YYYYmmdd-HHMMSS>
```

A rollback restores **both**, and restores `web/dist` atomically by moving
directories rather than copying into a live one — a half-copied dist serves a
mix of old and new assets, which reads as a UI bug rather than a failed cutover.
