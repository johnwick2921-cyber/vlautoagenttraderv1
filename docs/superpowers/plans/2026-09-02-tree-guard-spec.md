# SPEC — TREE GUARD: alarm when the deploy tree stops matching what shipped

Status: **SPEC ONLY — not built, not installed.** Owner approved speccing it 2026-09-02 as its own small wave.
Author: Fable (session fable-class45) · Scope: one wave, no cutover, no bot change.

## Why this exists

On 2026-09-02 at 08:46:33–34 the main tree `/home/hoang/nofx` lost the following shipped
code, silently, for ~3h20m until it was found by accident at 12:03:

| removed | shipped by |
|---|---|
| `CorrectedPnL`, `UnresolvedExcluded` | class 40 — the corrected-column law |
| `composeArmStop`, `IncArmRefusal`, `IncStopUnanchored` | 0B — the stop floor and both owner-ruled counters |
| `normalizeArmLegs`, `ArmNormalizations` | class 39 |
| the real `GetOpenOrders` (reverted to the `return []types.OpenOrder{}, nil` stub) | class 33 |

596 deletions across six files. **No agent did it and no git command did it.** Proof:

1. The six files came from THREE different points in history — `auto_trader_planner.go`
   matched `d4b38604` (09-01 08:38), `auto_trader_dayplan.go` matched `b7715a73`
   (09-01 18:02), the other four matched the 09-01 21:32–23:03 range. No commit in the
   last 120 matched more than 4 of 6, so no checkout, reset or rebase produced it.
2. VS Code's local history holds entries byte-identical to the lost content
   (`auto_trader_planner.go` md5 `e97c71cbdb4a` at 08:46:33; `armed_executor.go` md5
   `3235a50dd13f` at 08:46:34). VS Code writes those on **save**.
3. The entire day's main-tree write set is exactly those six files inside two seconds —
   a Save All over six stale tabs, independently enumerated by a second session.

The tree survived only because nobody built or committed from it during those hours. The
worktree law and `~/nofx-main.lock` both govern **agents**; an editor is not an agent, so
neither could have prevented this and neither will prevent the next one. The mitigation the
owner must do by hand (close/revert those tabs) is not enforceable from the repo. What IS
enforceable is **detection**: this guard.

## Contract

**Observe and alarm. Never repair.** The guard must never run `git checkout`, `restore`,
`reset`, `stash` or any writing command. An automatic repair here would be a second actor
mutating the deploy tree, which is the disease, not the cure. It reports; a human decides.

## Checks (each independently reported)

| # | check | ALARM when | rationale |
|---|---|---|---|
| 1 | `git -C <tree> status --porcelain` | non-empty | the 08:46 class, caught within one tick |
| 2 | shipped-symbol canary: `composeArmStop`, `CorrectedPnL`, `normalizeArmLegs`, `IncArmRefusal`, `IncStopUnanchored`, `ArmNormalizations` each grep-present | any missing | catches the same loss even if someone COMMITS it, which check 1 would then call clean |
| 3 | `deploy/RELEASE` vs `go version -m nofx-bin` `vcs.revision` | mismatch | the A19 class — twice today a marker carried a stale RELEASE while a different rev ran |
| 4 | HEAD vs `origin/dev` | behind by > N commits (suggest 20) or > 6 h | the tree sat 4 h behind on 09-02 with nobody noticing |

Check 2 is the one that would have caught today's incident *even after a commit*, and it is
the reason this guard is worth building rather than just trusting porcelain.

## Expected-dirty suppression

A cutover legitimately dirties the tree (the RELEASE write before the kill, per A19). So:

```
deploy/nofx-lock.sh check      # 0 free · 1 held-fresh · 2 held-stale

held-fresh (rc 1): checks 1 and 3 downgrade to INFO
                   ("dirty under lock <task> held by <session> — expected")
held-stale (rc 2): checks 1 and 3 WARN, naming the session and the heartbeat age.
                   STALE IS NOT DEAD — the guard reports, a human corroborates.
free       (rc 0): ALARM
```

This makes the lock earn a second job: it is the declaration that dirt is intentional. A
dirty tree with **no lock at all** is exactly the 08:46 signature.

**Updated 2026-09-03 (class 70).** This section originally read *"if ~/nofx-main.lock
exists AND its pid is alive (kill -0)"*. There is no pid any more. The lock is an atomic
`mkdir ~/nofx-main.lock.d` recording session · task · acquired · expiry · heartbeat, and
liveness is the heartbeat's age, because a pid answered "does some process exist" when the
question was "is the owner still working". A guard that asked `kill -0` would have
mis-answered in both directions on 2026-09-03: it would have called a working owner gone,
and it would have called a silently-replaced lock healthy.

## Shape

Follow the established no-sudo ops surface (`deploy/install-db-backup.sh`,
`deploy/install-clock-guard.sh`): a plain script plus a `systemd --user` timer, linger
already enabled. **No change to `nofx-bin`, therefore no cutover, no flat gate, no owner GO
for the deploy.**

```
deploy/nofx-tree-guard.sh                  the checks, read-only
deploy/systemd-user/nofx-tree-guard.service
deploy/systemd-user/nofx-tree-guard.timer  OnUnitActiveSec=60s (a git status is ~5 ms)
deploy/install-tree-guard.sh               install + enable --now, mirroring install-db-backup.sh
```

**Alarm channels**, in order of usefulness:

1. `journalctl --user -u nofx-tree-guard` — WARN/ERROR lines, always.
2. A state file (`~/nofx-backups/tree-guard/state`) holding the last verdict, so the next
   boot or any agent can read "was the tree ever dirty since the last deploy?" without
   scraping logs.
3. **Not** the bot's alert table: `data/data.db` is read-only to anything not explicitly
   authorised to write it, and a guard that writes to the DB it is guarding is a bad shape.

## Tests

- Fixture: a scratch clone made dirty → check 1 alarms, and quotes the file list.
- Fixture: a scratch clone with `composeArmStop` deleted **and committed** → check 1 says
  clean, check 2 ALARMS. This is the incident-specific pin.
- Fixture: RELEASE mismatched against a stamped binary → check 3 alarms.
- Fixture: dirty tree + a lock with a FRESH heartbeat → INFO, not ALARM.
- Fixture: dirty tree + a lock with a STALE heartbeat → WARN naming the session and the
  age, never the word "dead", and never a clear.
- Fixture: dirty tree + no lock → ALARM (the 08:46 signature).
- The guard must be proven to run zero writing git commands: a source pin asserting the
  script contains no `checkout|restore|reset|stash|clean|commit|push`.

## What this does NOT solve

The editor can still overwrite the tree at any moment; this guard shortens the discovery
window from hours to a minute, it does not close the hole. The only real fix is to stop
opening `/home/hoang/nofx` in an editor and open worktrees instead. That is an owner action
and no repo change substitutes for it. Say so in the Guide entry, plainly.

## Estimated size

~150 lines of shell, ~40 lines of unit files, ~120 lines of fixtures, one Guide entry, one
checklist class (the incident deserves its own: *"a non-agent writer mutating the deploy
tree — locks govern agents, not editors"*). No binary change, no cutover.
