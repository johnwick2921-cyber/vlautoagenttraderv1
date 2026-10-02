# VL rename boot — runbook (R2)

Wherever `<old>` appears in this runbook it stands for the pre-rename prefix:
the four-letter name this program removes. It is written out NOWHERE in the
migration script, its test or this file; the script assembles it at runtime.

This runbook covers the box move that `deploy/migrate-to-vl.sh` performs — the
one scripted, reversible step of the rename (the plan's R2 boot). The script is
the mechanism; this file is the operator's and owner's checklist.

## Before the run (the operator)

1. **Lock, acquired from inside the tree, 120 min**, at the START of pre-boot,
   before the on-box dry run:
   `cd ~/<old> && deploy/<old>-lock.sh acquire <session> "R2 vl rename boot" 120`.
   The script's step 0 (i) requires the meta to name exactly `--session` and
   `deploy/<old>-lock.sh check` to return 1 (held and fresh; 2/3/4 are refusals).
2. **`~/vl-main.lock.d` absent.** The script refuses when it exists.
3. **Stop the vite unit before the fast-forward:** `sudo systemctl stop <old>-web`
   (the production UI on :8080 is untouched).
4. **Fast-forward** the main tree to the R2 sha, then verify:
   `git -C ~/<old> rev-parse HEAD` is exactly `--sha`, and
   `git -C ~/<old> status --porcelain --untracked-files=no` is empty (the
   untracked `web/dist.old.*` rollback dists are expected and ignored).
5. **Build the release dir outside the tree**, with every binary built OUTSIDE
   its clone: the clone at `~/vl-release-<sha8>/src`, the artifacts at
   `~/vl-release-<sha8>/`. It must hold `vl-bin`, `web/dist` (built with
   `VITE_GUIDE_BUILT_REV=<sha>`), `vl-activate`, `updater/vl-updater` and
   `updater/vl-updater-bootstrap`, every binary stamped `vcs.revision=<sha>` and
   `vcs.modified=false`. The script refuses a stamp that is wrong or modified.
6. **The pre-boot db-compat proof** (the plan's R2 "before the run" step 5).
7. **NT8 OPEN and every account flat** until the script's step 0 passes. The
   script checks the installation gate itself; a stale `trader_cutover:*` or
   `working_orders` snapshot prints "NT8 looks closed — reopen NT8 (accounts
   flat), then re-run".
8. **A freshly minted `VL_CUTOVER_TOKEN`.** The script reads
   `VL_CUTOVER_TOKEN`, falling back to the old prefix's key; it is never in
   argv or a log (a 0600 header file is handed to curl).

The script also refuses any NON-TERMINAL updater job that is YOUNGER than 30
minutes (the new worker's start sweep would resume it against the new install).
A job OLDER than 30 minutes is stale — the worker's sweep marks such jobs
`recovery_needed` — so the script lists them and leaves them alone; existing
`recovery_needed` jobs are listed (ids only) and left alone.

## The run (the owner present)

```sh
cd ~
~/<old>/deploy/migrate-to-vl.sh \
  --session <session> --sha <40-hex R2 sha> --release-dir ~/vl-release-<sha8>
```

First do a dry run; a dry run that passes is the pre-boot readiness evidence:

```sh
~/<old>/deploy/migrate-to-vl.sh --session <session> --sha <sha> \
  --release-dir ~/vl-release-<sha8> --dry-run
```

The dry run executes every read-only step-0 check, prints every action, writes
nothing, and exits non-zero on any refusal.

The real run stops the units, backs up the DB to
`~/<old>-backups/pre-vl-rename-<ts>/data.db` with absolute `-db` paths, moves
the tree and the data dirs (each gets a symlink back for humans), installs the
new build with a WRITTEN (never copied) RELEASE marker, starts the vl units,
verifies the boot line and the health revision (a 7+ hex PREFIX of `--sha`,
never `==` against 40 hex), and finalizes. On any failure in steps 1–5, or on
the two auto verify legs, it rolls back automatically — the bot is never left
stopped. `--no-auto-rollback` exists only for the owner's explicit word at the
boot and prints a banner.

**Lock:** the script keeps the lock HELD by default. The deploy owner runs
`cd ~/vl && deploy/vl-lock.sh release <session>` only after the RELEASE marker
commit is pushed from the tree. `--release-lock` (for a machine with no marker
commit to push, e.g. a partner) releases at step 7, and only when the marker
names THIS boot's sha (a content check) AND the tree's HEAD equals its
upstream.

## Rollback

```sh
~/<old>/deploy/migrate-to-vl.sh --session <session> --rollback
```

It reads ONLY the state file (`~/.local/state/vl-migrate/<session>`) and never
globs parked binaries or old dists. The lock is checked FIRST, before anything
is stopped: the meta must name `--session` and `check` must return 1. A
foreign or stale lock refuses before anything is touched and prints the meta,
the state file and the exact recovery commands.

## After the boot

- The old units stay disabled as the rollback target until the R5 phase.
- The parked binaries (`vl-bin`'s old twin and the old updater) are kept for
  the rollback; R5 removes them.
- NT8: the owner copies the AddOn, presses F5 and fully restarts NT8 (the
  `VL_BUILD_ID` changed); see the plan's R3 section.
- **Updates re-enroll (boot sheet S1).** Run once after the boot:
  `<release-dir>/updater/vl-updater-bootstrap --install-dir /home/hoang/vl enroll --replace <email>`
  The MAC domain changed with the rename — until this runs, the Updates page
  answers 403.
- The whole run asks for sudo exactly ONCE: the step-0 `sudo -v`.
- Steps S2 and S5–S7 live in the CTO's boot sheet, not in this runbook.

## Accepted residue (stated, on purpose)

- `vl_*.log` files already written;
- updater backups and verdict or job files that carry `~/vl` paths (re-fetched
  on the next install);
- the main tree stays at the R2 sha — the compat readers understand either
  state, and resetting the tree is the deploy owner's call;
- `:3000` (vite, serving the R2 tree) sends the new update header to the
  restored old bot after a rollback, so its Updates page answers 403 there;
  `:8080` serves the restored old dist and works.
